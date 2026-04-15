import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Image,
  Popconfirm,
  Row,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Timeline,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { api } from "../api";
import type {
  Job,
  ScanActionCounts,
  ScanActionGroupedCounts,
  ScanActionItem,
  ScanEvent,
  ScanJobLog,
} from "../types";
import { formatDateTime } from "../utils/dateTime";
import {
  formatActionStatus,
  formatJobSource,
  formatJobStatus,
  formatLogLevel,
  formatPhase,
} from "../utils/displayText";
import { formatParentDirectory } from "../utils/path";
import { NO_SCROLL_TABLE_CLASS } from "../utils/table";

type ActionTabKey = "pending" | "executed" | "error";

type DuplicatePairLine = {
  key: string;
  pairId: number;
  side: "A" | "B";
  fileName: string;
  path: string;
  slot: "pair_a" | "pair_b";
  recommendedDelete: boolean;
  deleteTargetSide: "A" | "B";
  status: string;
  executedAt?: string;
  errorMessage?: string;
};

const actionTypeOptions = [
  { key: "delete", label: "删除" },
  { key: "move", label: "移动" },
  { key: "modify_time", label: "修改时间" },
  { key: "delete_duplicate", label: "重复项" },
  { key: "rename", label: "重命名" },
] as const;

const EMPTY_COUNTS: ScanActionCounts = {
  delete: 0,
  move: 0,
  modifyTime: 0,
  deleteDuplicate: 0,
  rename: 0,
  total: 0,
};

const actionTabs: ActionTabKey[] = ["pending", "executed", "error"];

export function JobDetailPage() {
  const { id = "" } = useParams();
  const [messageApi, contextHolder] = message.useMessage();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState("pending");
  const [selectedActionType, setSelectedActionType] =
    useState<string>("delete");
  const [actionPage, setActionPage] = useState({ current: 1, pageSize: 20 });

  const jobQuery = useQuery({
    queryKey: ["job", id],
    queryFn: () => api.getJob(id),
    refetchInterval: (query) => {
      const status = (query.state.data?.job.status ?? "") as string;
      return status === "running" || status === "pending" ? 2000 : false;
    },
  });

  const actionQuery = useQuery({
    queryKey: ["job-actions", id, activeTab, selectedActionType, actionPage],
    enabled: isActionTab(activeTab),
    queryFn: () => {
      const params = new URLSearchParams({
        tab: activeTab,
        page: String(actionPage.current),
        pageSize: String(actionPage.pageSize),
      });
      params.set("type", selectedActionType);
      return api.getJobActionItems(id, params);
    },
  });

  const eventQuery = useQuery({
    queryKey: ["job-events", id],
    queryFn: () =>
      api.getJobEvents(id, new URLSearchParams({ page: "1", pageSize: "50" })),
    refetchInterval: 3000,
  });

  const logQuery = useQuery({
    queryKey: ["job-logs", id],
    queryFn: () =>
      api.getJobLogs(id, new URLSearchParams({ page: "1", pageSize: "200" })),
    refetchInterval: () => {
      const status = (queryClient.getQueryData<{ job: Job }>(["job", id])?.job
        .status ?? "") as string;
      return status === "running" || status === "pending" ? 2000 : false;
    },
  });

  useEffect(() => {
    if (!id) return;
    const source = new EventSource(`/api/jobs/${id}/stream`, {
      withCredentials: true,
    });
    source.addEventListener("job", () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    });
    source.addEventListener("event", () => {
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-logs", id] });
    });
    return () => source.close();
  }, [id, queryClient]);

  const duplicateMutation = useMutation({
    mutationFn: () => api.executeDuplicateDelete(id),
    onSuccess: () => {
      messageApi.success("重复文件删除已执行");
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const duplicateDeleteActionMutation = useMutation({
    mutationFn: ({ itemId, side }: { itemId: number; side: "A" | "B" }) =>
      api.executeDuplicateDeleteActionItem(id, itemId, side),
    onSuccess: (_, variables) => {
      messageApi.success(`重复项 ${variables.side} 图已删除`);
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const deleteActionMutation = useMutation({
    mutationFn: (itemId: number) => api.executeDeleteActionItem(id, itemId),
    onSuccess: () => {
      messageApi.success("删除动作已执行");
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const modifyShootTimeMutation = useMutation({
    mutationFn: (itemId: number) =>
      api.executeModifyShootTimeActionItem(id, itemId),
    onSuccess: () => {
      messageApi.success("拍摄时间已变更");
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const moveActionMutation = useMutation({
    mutationFn: (itemId: number) => api.executeMoveActionItem(id, itemId),
    onSuccess: () => {
      messageApi.success("位置已变更");
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const renameActionMutation = useMutation({
    mutationFn: (itemId: number) => api.executeRenameActionItem(id, itemId),
    onSuccess: () => {
      messageApi.success("文件名已变更");
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const deleteAllMutation = useMutation({
    mutationFn: () => api.executeDeleteAllActionItems(id),
    onSuccess: (data) => {
      messageApi.success(`已执行 ${data.count} 条删除动作`);
      queryClient.invalidateQueries({ queryKey: ["job-actions", id] });
      queryClient.invalidateQueries({ queryKey: ["job-events", id] });
      queryClient.invalidateQueries({ queryKey: ["job", id] });
    },
    onError: (error) => {
      messageApi.error(error instanceof Error ? error.message : "执行失败");
    },
  });

  const job = jobQuery.data?.job;
  const summary = job?.summary as Record<string, number | string> | undefined;
  const totalDirectoryCount = numberFromSummaryKey(
    summary,
    "dirTotal",
    "DirTotal",
  );
  const events = eventQuery.data?.list ?? [];
  const logs = logQuery.data?.list ?? [];
  const groupedCounts =
    actionQuery.data?.groupedCounts ?? summaryToGroupedCounts(summary);
  const pendingActionTotal = groupedCounts.pending.total;
  const executedActionTotal = groupedCounts.executed.total;
  const tabItems = useMemo(
    () => [
      { key: "pending", label: `待执行动作 (${pendingActionTotal})` },
      { key: "executed", label: `已执行动作 (${executedActionTotal})` },
      { key: "events", label: "实时事件" },
      { key: "logs", label: "运行日志" },
      { key: "stats", label: "统计" },
      { key: "raw", label: "原始参数" },
    ],
    [executedActionTotal, pendingActionTotal],
  );

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      {contextHolder}
      <Typography.Title level={3}>扫描详情 #{id}</Typography.Title>

      {job?.errorMessage ? (
        <Alert
          type="error"
          showIcon
          message="任务失败"
          description={job.errorMessage}
        />
      ) : null}

      <Card>
        <Descriptions column={3}>
          <Descriptions.Item label="状态">
            <Tag>{formatJobStatus(job?.status)}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="来源">
            {formatJobSource(job?.source)}
          </Descriptions.Item>
          <Descriptions.Item label="阶段">
            {formatPhase(job?.currentPhase)}
          </Descriptions.Item>
          <Descriptions.Item label="任务UUID">{job?.jobUuid}</Descriptions.Item>
          <Descriptions.Item label="扫描UUID">
            {job?.scanUuid}
          </Descriptions.Item>
          <Descriptions.Item label="是否执行动作">
            {job?.hasAction ? "是" : "否"}
          </Descriptions.Item>
          <Descriptions.Item label="开始时间">
            {formatDateTime(job?.startAt)}
          </Descriptions.Item>
          <Descriptions.Item label="结束时间">
            {formatDateTime(job?.endAt)}
          </Descriptions.Item>
          <Descriptions.Item label="最近心跳">
            {formatDateTime(job?.lastHeartbeatAt)}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <Row gutter={16}>
        <Col span={12}>
          <Card>
            <Statistic title="总文件夹数" value={totalDirectoryCount} />
          </Card>
        </Col>
        <Col span={12}>
          <Card>
            <Statistic title="总文件数" value={job?.totalCount ?? 0} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={(key) => {
            setActiveTab(key);
            setActionPage({ current: 1, pageSize: 20 });
          }}
          items={tabItems.map((item) => ({
            ...item,
            children: renderTab(item.key, {
              id,
              job,
              summary,
              events,
              logs,
              actions: actionQuery.data?.list ?? [],
              groupedCounts,
              actionTotal: actionQuery.data?.total ?? 0,
              actionPage,
              actionType: selectedActionType,
              duplicateDetectionEnabled: isScanArgEnabled(
                job?.scanArgs,
                "md5Show",
              ),
              isDeletingDuplicateAction:
                duplicateDeleteActionMutation.isPending,
              isDeletingDuplicateBulk: duplicateMutation.isPending,
              isModifyingShootTime: modifyShootTimeMutation.isPending,
              isMovingAction: moveActionMutation.isPending,
              isRenamingAction: renameActionMutation.isPending,
              isDeletingAction: deleteActionMutation.isPending,
              isDeletingAll: deleteAllMutation.isPending,
              onDeleteDuplicateAction: (itemId, side) =>
                duplicateDeleteActionMutation.mutate({ itemId, side }),
              onDeleteRecommendedDuplicates: () => duplicateMutation.mutate(),
              onModifyShootTime: (itemId) =>
                modifyShootTimeMutation.mutate(itemId),
              onMoveAction: (itemId) => moveActionMutation.mutate(itemId),
              onRenameAction: (itemId) => renameActionMutation.mutate(itemId),
              onDeleteAction: (itemId) => deleteActionMutation.mutate(itemId),
              onDeleteAll: () => deleteAllMutation.mutate(),
              setActionType: (value) => {
                setSelectedActionType(value);
                setActionPage({ current: 1, pageSize: actionPage.pageSize });
              },
              setActionPage,
              setActiveTab,
            }),
          }))}
        />
      </Card>
    </Space>
  );
}

function renderTab(
  key: string,
  context: {
    id: string;
    job?: Job;
    summary?: Record<string, number | string>;
    events: ScanEvent[];
    logs: ScanJobLog[];
    actions: ScanActionItem[];
    groupedCounts: ScanActionGroupedCounts;
    actionTotal: number;
    actionPage: { current: number; pageSize: number };
    actionType: string;
    duplicateDetectionEnabled: boolean;
    isDeletingDuplicateAction: boolean;
    isDeletingDuplicateBulk: boolean;
    isModifyingShootTime: boolean;
    isMovingAction: boolean;
    isRenamingAction: boolean;
    isDeletingAction: boolean;
    isDeletingAll: boolean;
    onDeleteDuplicateAction: (itemId: number, side: "A" | "B") => void;
    onDeleteRecommendedDuplicates: () => void;
    onModifyShootTime: (itemId: number) => void;
    onMoveAction: (itemId: number) => void;
    onRenameAction: (itemId: number) => void;
    onDeleteAction: (itemId: number) => void;
    onDeleteAll: () => void;
    setActionType: (value: string) => void;
    setActionPage: (page: { current: number; pageSize: number }) => void;
    setActiveTab: (key: string) => void;
  },
) {
  if (key === "events") {
    return (
      <Timeline
        items={context.events.map((event) => ({
          children: `${formatDateTime(event.createdAt)} | ${event.title} | ${event.message}${event.relatedPath ? ` | ${event.relatedPath}` : ""}`,
        }))}
      />
    );
  }

  if (key === "stats") {
    return (
      <pre className="json-block">
        {JSON.stringify(context.summary ?? {}, null, 2)}
      </pre>
    );
  }

  if (key === "logs") {
    return (
      <pre className="json-block log-block">
        {context.logs.length === 0
          ? "暂无日志"
          : context.logs
              .map((log) => {
                const payload = log.payloadJson ? ` ${log.payloadJson}` : "";
                return `[${formatDateTime(log.createdAt)}] [${formatLogLevel(log.level)}] [${formatPhase(log.phase || "runtime")}] ${log.message}${payload}`;
              })
              .join("\n")}
      </pre>
    );
  }

  if (key === "raw") {
    return (
      <pre className="json-block">
        {JSON.stringify(context.job?.scanArgs ?? {}, null, 2)}
      </pre>
    );
  }

  if (isActionTab(key)) {
    return renderActionTab(key, context);
  }

  return null;
}

function renderActionTab(
  tab: ActionTabKey,
  context: {
    id: string;
    actions: ScanActionItem[];
    groupedCounts: ScanActionGroupedCounts;
    actionTotal: number;
    actionPage: { current: number; pageSize: number };
    actionType: string;
    duplicateDetectionEnabled: boolean;
    isDeletingDuplicateAction: boolean;
    isDeletingDuplicateBulk: boolean;
    isModifyingShootTime: boolean;
    isMovingAction: boolean;
    isRenamingAction: boolean;
    isDeletingAction: boolean;
    isDeletingAll: boolean;
    onDeleteDuplicateAction: (itemId: number, side: "A" | "B") => void;
    onDeleteRecommendedDuplicates: () => void;
    onModifyShootTime: (itemId: number) => void;
    onMoveAction: (itemId: number) => void;
    onRenameAction: (itemId: number) => void;
    onDeleteAction: (itemId: number) => void;
    onDeleteAll: () => void;
    setActionType: (value: string) => void;
    setActionPage: (page: { current: number; pageSize: number }) => void;
    setActiveTab: (key: string) => void;
  },
) {
  const tabCounts = context.groupedCounts[tab];
  const isDuplicateType = context.actionType === "delete_duplicate";
  const duplicateRows = isDuplicateType
    ? buildDuplicatePairLines(context.actions)
    : [];
  const showRenameRedirect =
    tab === "pending" &&
    context.actionType === "rename" &&
    context.actions.length === 0 &&
    context.groupedCounts.executed.rename > 0;

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <Tabs
        activeKey={context.actionType}
        onChange={context.setActionType}
        items={actionTypeOptions.map((item) => ({
          key: item.key,
          label: `${item.label} (${countForType(tabCounts, item.key)})`,
          children: (
            <Space direction="vertical" size={12} style={{ width: "100%" }}>
              {tab === "pending" &&
              item.key === "delete_duplicate" &&
              !context.duplicateDetectionEnabled ? (
                <Alert
                  type="info"
                  showIcon
                  message="当前任务未开启“计算重复文件”"
                  description="这次扫描不会产出重复项数据。如需查看重复项，请在新建扫描时勾选“计算重复文件（md5Show）”。"
                />
              ) : null}
              {showRenameRedirect ? (
                <Alert
                  type="info"
                  showIcon
                  message="当前没有待执行的重命名动作"
                  description="这个任务已经产生过重命名结果。可以直接切到“已执行动作”里的重命名页签查看。"
                  action={
                    <Button
                      size="small"
                      disabled={context.groupedCounts.executed.rename === 0}
                      onClick={() => context.setActiveTab("executed")}
                    >
                      查看已执行 ({context.groupedCounts.executed.rename})
                    </Button>
                  }
                />
              ) : null}
              {tab === "pending" &&
              item.key === "delete_duplicate" &&
              context.duplicateDetectionEnabled ? (
                <Row justify="end">
                  <Col>
                    <Popconfirm
                      title="确认按建议删除所有照片"
                      description="将批量删除当前重复项中所有“建议删除=是”的照片，是否继续？"
                      okText="确认"
                      cancelText="取消"
                      onConfirm={context.onDeleteRecommendedDuplicates}
                    >
                      <Button
                        danger
                        loading={context.isDeletingDuplicateBulk}
                        disabled={context.actionTotal === 0}
                      >
                        按建议删除所有照片
                      </Button>
                    </Popconfirm>
                  </Col>
                </Row>
              ) : null}
              {tab === "pending" && item.key === "delete" ? (
                <Row justify="end">
                  <Col>
                    <Popconfirm
                      title="确认全部删除"
                      description="将立即执行当前任务下全部待删除项，是否继续？"
                      okText="确认"
                      cancelText="取消"
                      onConfirm={context.onDeleteAll}
                    >
                      <Button
                        danger
                        loading={context.isDeletingAll}
                        disabled={context.actionTotal === 0}
                      >
                        全部删除
                      </Button>
                    </Popconfirm>
                  </Col>
                </Row>
              ) : null}
              <div className={NO_SCROLL_TABLE_CLASS}>
                {isDuplicateType ? (
                  <Table
                    rowKey="key"
                    tableLayout="fixed"
                    columns={getDuplicatePairColumns(
                      context.id,
                      tab,
                      context.isDeletingDuplicateAction,
                      context.onDeleteDuplicateAction,
                    )}
                    dataSource={duplicateRows}
                    rowClassName={(record) =>
                      `duplicate-line-row duplicate-line-row-${record.side.toLowerCase()}`
                    }
                    locale={{
                      emptyText: getActionEmptyState(
                        tab,
                        item.key,
                        context.duplicateDetectionEnabled,
                      ),
                    }}
                    pagination={{
                      total: context.actionTotal,
                      current: context.actionPage.current,
                      pageSize: context.actionPage.pageSize,
                      onChange: (current, pageSize) =>
                        context.setActionPage({ current, pageSize }),
                    }}
                  />
                ) : (
                  <Table
                    rowKey="id"
                    tableLayout="fixed"
                    columns={getActionColumns(
                      context.id,
                      item.key,
                      tab,
                      context.isDeletingAction,
                      context.onDeleteAction,
                      context.isModifyingShootTime,
                      context.onModifyShootTime,
                      context.isMovingAction,
                      context.onMoveAction,
                      context.isRenamingAction,
                      context.onRenameAction,
                    )}
                    dataSource={context.actions}
                    locale={{
                      emptyText: getActionEmptyState(
                        tab,
                        item.key,
                        context.duplicateDetectionEnabled,
                      ),
                    }}
                    pagination={{
                      total: context.actionTotal,
                      current: context.actionPage.current,
                      pageSize: context.actionPage.pageSize,
                      onChange: (current, pageSize) =>
                        context.setActionPage({ current, pageSize }),
                    }}
                  />
                )}
              </div>
            </Space>
          ),
        }))}
      />
    </Space>
  );
}

function getActionColumns(
  id: string,
  actionType: string,
  tab: ActionTabKey,
  isDeletingAction: boolean,
  onDeleteAction: (itemId: number) => void,
  isModifyingShootTime: boolean,
  onModifyShootTime: (itemId: number) => void,
  isMovingAction: boolean,
  onMoveAction: (itemId: number) => void,
  isRenamingAction: boolean,
  onRenameAction: (itemId: number) => void,
): ColumnsType<ScanActionItem> {
  const historyColumns = getHistoryColumns();

  switch (actionType) {
    case "move":
      return appendHistoryColumns(
        [
          photoColumn(id, "照片"),
          textColumn("文件名", (record) => record.detail?.fileName),
          textColumn("当前位置", (record) =>
            formatParentDirectory(record.detail?.currentPath),
          ),
          textColumn("移动位置", (record) =>
            formatParentDirectory(record.detail?.targetPath),
          ),
          textColumn("目录日期", (record) => record.detail?.dirDate),
          textColumn("文件名提取日期", (record) => record.detail?.fileNameDate),
          textColumn("照片拍摄日期", (record) => record.detail?.shootDate),
          textColumn("文件修改时间", (record) => record.detail?.modifyDate),
          textColumn("最小日期", (record) => record.detail?.minDate),
          ...(tab === "pending"
            ? [moveActionColumn(isMovingAction, onMoveAction)]
            : []),
        ],
        tab,
        historyColumns,
      );
    case "modify_time":
      return appendHistoryColumns(
        [
          photoColumn(id, "照片"),
          textColumn("文件名", (record) => record.detail?.fileName),
          textColumn("目录日期", (record) => record.detail?.dirDate),
          textColumn("文件名提取日期", (record) => record.detail?.fileNameDate),
          textColumn("文件修改时间", (record) => record.detail?.modifyDate),
          textColumn("照片拍摄日期", (record) => record.detail?.shootDate),
          textColumn("最小日期", (record) => record.detail?.minDate),
          ...(tab === "pending"
            ? [
                modifyShootTimeActionColumn(
                  isModifyingShootTime,
                  onModifyShootTime,
                ),
              ]
            : []),
        ],
        tab,
        historyColumns,
      );
    case "rename":
      return appendHistoryColumns(
        [
          photoColumn(id, "照片"),
          textColumn("原文件名", (record) => record.detail?.fileName),
          textColumn("新文件名", (record) => record.detail?.targetFileName),
          textColumn("当前位置", (record) =>
            formatParentDirectory(record.detail?.currentPath),
          ),
          ...(tab === "pending"
            ? [renameActionColumn(isRenamingAction, onRenameAction)]
            : []),
        ],
        tab,
        historyColumns,
      );
    case "delete":
    default: {
      const baseColumns: ColumnsType<ScanActionItem> = [
        textColumn(
          "文件名",
          (record) =>
            record.detail?.fileName ?? record.sourcePath.split("/").pop(),
        ),
        textColumn(
          "位置",
          (record) => record.detail?.currentPath ?? record.sourcePath,
        ),
        textColumn("原因", (record) => getDeleteReasonText(record)),
      ];
      if (tab === "pending") {
        baseColumns.push(deleteActionColumn(isDeletingAction, onDeleteAction));
      }
      return appendHistoryColumns(baseColumns, tab, historyColumns);
    }
  }
}

function appendHistoryColumns(
  columns: ColumnsType<ScanActionItem>,
  tab: ActionTabKey,
  historyColumns: ColumnsType<ScanActionItem>,
) {
  if (tab === "pending") {
    return columns;
  }
  return [...columns, ...historyColumns];
}

function getHistoryColumns(): ColumnsType<ScanActionItem> {
  return [
    {
      title: "状态",
      dataIndex: "status",
      width: 88,
      render: (value) => <Tag>{formatActionStatus(String(value ?? ""))}</Tag>,
    },
    {
      title: "执行时间",
      dataIndex: "executedAt",
      width: 120,
      render: (value) => formatDateTime(value),
    },
    {
      title: "错误",
      dataIndex: "errorMessage",
      width: 180,
      render: (value) => value || "-",
    },
  ];
}

function getDuplicatePairColumns(
  id: string,
  tab: ActionTabKey,
  isDeletingDuplicateAction: boolean,
  onDeleteDuplicateAction: (itemId: number, side: "A" | "B") => void,
): ColumnsType<DuplicatePairLine> {
  const columns: ColumnsType<DuplicatePairLine> = [
    {
      title: "AB",
      dataIndex: "side",
      width: 72,
      render: (value: DuplicatePairLine["side"]) => (
        <span className="duplicate-side-cell">{value}</span>
      ),
    },
    {
      title: "照片",
      dataIndex: "slot",
      width: 104,
      render: (_, record) => (
        <PreviewImage
          id={id}
          itemId={record.pairId}
          slot={record.slot}
          alt={record.fileName}
        />
      ),
    },
    {
      title: "照片文件名",
      dataIndex: "fileName",
      render: (value: string) => value || "-",
    },
    {
      title: "照片位置",
      dataIndex: "path",
      render: (value: string) => value || "-",
    },
    {
      title: "建议删除",
      dataIndex: "recommendedDelete",
      width: 108,
      render: (value: boolean) =>
        value ? <Typography.Text type="danger">是</Typography.Text> : "-",
    },
  ];

  if (tab === "pending") {
    columns.push({
      title: "操作",
      width: 96,
      render: (_, record) => (
        <Popconfirm
          title="确认删除"
          description={`将立即删除当前 ${record.side} 图，是否继续？`}
          okText="确认"
          cancelText="取消"
          onConfirm={() =>
            onDeleteDuplicateAction(record.pairId, record.deleteTargetSide)
          }
        >
          <Button danger type="link" loading={isDeletingDuplicateAction}>
            删除
          </Button>
        </Popconfirm>
      ),
    });
  } else {
    columns.push(
      {
        title: "状态",
        dataIndex: "status",
        width: 88,
        render: (value: string) => <Tag>{formatActionStatus(value)}</Tag>,
      },
      {
        title: "执行时间",
        dataIndex: "executedAt",
        width: 120,
        render: (value: string | undefined) => formatDateTime(value),
      },
      {
        title: "错误",
        dataIndex: "errorMessage",
        width: 180,
        render: (value: string | undefined) => value || "-",
      },
    );
  }

  return columns;
}

function deleteActionColumn(
  isDeletingAction: boolean,
  onDeleteAction: (itemId: number) => void,
): ColumnsType<ScanActionItem>[number] {
  return {
    title: "操作",
    width: 96,
    render: (_, record) => (
      <Popconfirm
        title="确认删除"
        description="将立即执行当前删除动作，是否继续？"
        okText="确认"
        cancelText="取消"
        onConfirm={() => onDeleteAction(record.id)}
      >
        <Button danger type="link" loading={isDeletingAction}>
          删除
        </Button>
      </Popconfirm>
    ),
  };
}

function modifyShootTimeActionColumn(
  isModifyingShootTime: boolean,
  onModifyShootTime: (itemId: number) => void,
): ColumnsType<ScanActionItem>[number] {
  return {
    title: "操作",
    width: 96,
    render: (_, record) => (
      <Popconfirm
        title="确认变更"
        description="将立即把当前文件的拍摄时间写为最小日期 00:00:00，是否继续？"
        okText="确认"
        cancelText="取消"
        onConfirm={() => onModifyShootTime(record.id)}
      >
        <Button type="link" loading={isModifyingShootTime}>
          变更
        </Button>
      </Popconfirm>
    ),
  };
}

function moveActionColumn(
  isMovingAction: boolean,
  onMoveAction: (itemId: number) => void,
): ColumnsType<ScanActionItem>[number] {
  return {
    title: "操作",
    width: 96,
    render: (_, record) => (
      <Popconfirm
        title="确认变更"
        description="将立即按当前建议目录执行移动，并重算当前文件候选动作，是否继续？"
        okText="确认"
        cancelText="取消"
        onConfirm={() => onMoveAction(record.id)}
      >
        <Button type="link" loading={isMovingAction}>
          变更
        </Button>
      </Popconfirm>
    ),
  };
}

function renameActionColumn(
  isRenamingAction: boolean,
  onRenameAction: (itemId: number) => void,
): ColumnsType<ScanActionItem>[number] {
  return {
    title: "操作",
    width: 96,
    render: (_, record) => (
      <Popconfirm
        title="确认变更"
        description="将立即按当前建议文件名执行重命名，并重算当前文件候选动作，是否继续？"
        okText="确认"
        cancelText="取消"
        onConfirm={() => onRenameAction(record.id)}
      >
        <Button type="link" loading={isRenamingAction}>
          变更
        </Button>
      </Popconfirm>
    ),
  };
}

function photoColumn(
  id: string,
  title: string,
): ColumnsType<ScanActionItem>[number] {
  return {
    title,
    width: 92,
    render: (_, record) => (
      <PreviewImage
        id={id}
        itemId={record.id}
        slot={record.detail?.previewSlot ?? "source"}
        alt={record.detail?.fileName ?? "照片"}
      />
    ),
  };
}

function buildDuplicatePairLines(
  actions: ScanActionItem[],
): DuplicatePairLine[] {
  return actions.flatMap((record) => {
    if (!record.pair) return [];
    return [
      {
        key: `${record.id}-A`,
        pairId: record.id,
        side: "A",
        fileName: record.pair.photoA.fileName,
        path: formatParentDirectory(record.pair.photoA.path) || "",
        slot: "pair_a",
        recommendedDelete: true,
        deleteTargetSide: "A",
        status: record.status,
        executedAt: record.executedAt,
        errorMessage: record.errorMessage,
      },
      {
        key: `${record.id}-B`,
        pairId: record.id,
        side: "B",
        fileName: record.pair.photoB.fileName,
        path: formatParentDirectory(record.pair.photoB.path) || "",
        slot: "pair_b",
        recommendedDelete: false,
        deleteTargetSide: "B",
        status: record.status,
        executedAt: record.executedAt,
        errorMessage: record.errorMessage,
      },
    ];
  });
}

function textColumn(
  title: string,
  getter: (record: ScanActionItem) => string | undefined,
): ColumnsType<ScanActionItem>[number] {
  return {
    title,
    render: (_, record) => getter(record) || "-",
  };
}

function getDeleteReasonText(record: ScanActionItem): string {
  if (
    record.actionType === "delete_empty_dir" ||
    record.reasonCode === "empty_dir"
  ) {
    return "空目录";
  }

  const explicitReason = record.reasonText?.trim();
  if (explicitReason && explicitReason !== "文件名命中删除规则") {
    return explicitReason;
  }

  const fileName = (
    record.detail?.fileName ??
    record.sourcePath.split("/").pop() ??
    ""
  ).trim();
  if (fileName.startsWith(".")) {
    return "文件名以 . 开头";
  }
  if (fileName.startsWith("IMG_E")) {
    return "文件名以 IMG_E 开头";
  }
  if (fileName.endsWith("_.pic.jpg")) {
    return "文件名以 _.pic.jpg 结尾";
  }
  if (fileName.endsWith("nas_downloading")) {
    return "文件名以 nas_downloading 结尾";
  }

  if (explicitReason) {
    return explicitReason;
  }
  return "文件大小为 0";
}

function PreviewImage({
  id,
  itemId,
  slot,
  alt,
}: {
  id: string;
  itemId: number;
  slot: string;
  alt: string;
}) {
  return (
    <Image
      className="action-preview"
      width={72}
      height={72}
      src={api.getJobActionPreviewUrl(id, itemId, slot)}
      alt={alt}
      fallback="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='72' height='72'%3E%3Crect width='72' height='72' fill='%23eef2ef'/%3E%3Ctext x='36' y='40' text-anchor='middle' fill='%23839590' font-size='12'%3E%E6%97%A0%E5%9B%BE%3C/text%3E%3C/svg%3E"
    />
  );
}

function countForType(counts: ScanActionCounts, type: string) {
  switch (type) {
    case "delete":
      return counts.delete;
    case "move":
      return counts.move;
    case "modify_time":
      return counts.modifyTime;
    case "delete_duplicate":
      return counts.deleteDuplicate;
    case "rename":
      return counts.rename;
    default:
      return 0;
  }
}

function getActionEmptyState(
  tab: ActionTabKey,
  actionType: string,
  duplicateDetectionEnabled: boolean,
) {
  if (actionType === "delete_duplicate" && !duplicateDetectionEnabled) {
    return <Empty description="当前任务未开启重复检测" />;
  }
  return (
    <Empty
      description={
        tab === "pending"
          ? "暂无待执行动作"
          : tab === "executed"
            ? "暂无已执行动作"
            : "暂无失败动作"
      }
    />
  );
}

function summaryToCounts(
  summary?: Record<string, number | string>,
): ScanActionCounts {
  const deleteCount = numberFromSummaryKey(
    summary,
    "deleteFileCnt",
    "DeleteFileCnt",
  );
  const moveCount = numberFromSummaryKey(summary, "moveFileCnt", "MoveFileCnt");
  const modifyCount = numberFromSummaryKey(
    summary,
    "modifyDateFileCnt",
    "ModifyDateFileCnt",
  );
  const duplicateCount = numberFromSummaryKey(
    summary,
    "dumpFileCnt",
    "DumpFileCnt",
  );
  const renameCount = numberFromSummaryKey(
    summary,
    "renameFileCnt",
    "RenameFileCnt",
  );
  return {
    delete: deleteCount,
    move: moveCount,
    modifyTime: modifyCount,
    deleteDuplicate: duplicateCount,
    rename: renameCount,
    total: deleteCount + moveCount + modifyCount + duplicateCount + renameCount,
  };
}

function summaryToGroupedCounts(
  summary?: Record<string, number | string>,
): ScanActionGroupedCounts {
  return {
    pending: summaryToCounts(summary),
    executed: EMPTY_COUNTS,
    error: EMPTY_COUNTS,
  };
}

function numberFromSummaryKey(
  summary: Record<string, number | string> | undefined,
  ...keys: string[]
) {
  for (const key of keys) {
    const value = summary?.[key];
    const count = numberFromSummary(value);
    if (count > 0 || value === 0 || value === "0") {
      return count;
    }
  }
  return 0;
}

function numberFromSummary(value: number | string | undefined) {
  if (typeof value === "number") return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function isScanArgEnabled(
  scanArgs: Record<string, unknown> | undefined,
  key: string,
) {
  return scanArgs?.[key] === true;
}

function isActionTab(key: string): key is ActionTabKey {
  return actionTabs.includes(key as ActionTabKey);
}
