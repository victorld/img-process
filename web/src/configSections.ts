export type ConfigSectionKey =
  | "database"
  | "server"
  | "scanArgs"
  | "basic"
  | "cache"
  | "dump"
  | "bak"
  | "gis"
  | "batch";

export type ConfigField = {
  key: string;
  label: string;
  valueType?: "string" | "boolean" | "number";
  secret?: boolean;
};

export type ConfigSection = {
  key: ConfigSectionKey;
  title: string;
  fields: ConfigField[];
};

export const configSections: ConfigSection[] = [
  {
    key: "database",
    title: "database 数据库参数",
    fields: [
      { key: "DbUsername", label: "数据库用户名" },
      { key: "DbPassword", label: "数据库密码", secret: true },
      { key: "DbHost", label: "数据库主机" },
      { key: "DbPort", label: "数据库端口" },
      { key: "DbName", label: "数据库名称" },
      { key: "DbConfig", label: "数据库连接参数" },
    ],
  },
  {
    key: "server",
    title: "server Web 服务参数",
    fields: [
      { key: "HttpPort", label: "HTTP 端口" },
      { key: "HttpUsername", label: "HTTP 用户名" },
      { key: "HttpPassword", label: "HTTP 密码", secret: true },
    ],
  },
  {
    key: "scanArgs",
    title: "scanArgs 扫描参数",
    fields: [
      { key: "StartPath", label: "扫描目录" },
      { key: "DeleteShow", label: "显示重复删除项", valueType: "boolean" },
      { key: "MoveFileShow", label: "显示移动项", valueType: "boolean" },
      { key: "ModifyDateShow", label: "显示修改时间项", valueType: "boolean" },
      { key: "RenameFileShow", label: "显示重命名项", valueType: "boolean" },
      { key: "Md5Show", label: "计算 MD5", valueType: "boolean" },
      { key: "DeleteAction", label: "执行重复删除", valueType: "boolean" },
      { key: "MoveFileAction", label: "执行移动", valueType: "boolean" },
      { key: "ModifyDateAction", label: "执行修改时间", valueType: "boolean" },
      { key: "RenameFileAction", label: "执行重命名", valueType: "boolean" },
    ],
  },
  {
    key: "basic",
    title: "basic 基础配置",
    fields: [
      { key: "ColorOutput", label: "彩色日志", valueType: "boolean" },
      { key: "SqlDebug", label: "SQL 调试", valueType: "boolean" },
    ],
  },
  {
    key: "cache",
    title: "cache 缓存配置",
    fields: [
      { key: "ImgCache", label: "图片缓存", valueType: "boolean" },
      { key: "SyncTable", label: "同步表数据", valueType: "boolean" },
      { key: "TruncateTable", label: "清空表数据", valueType: "boolean" },
    ],
  },
  {
    key: "dump",
    title: "dump 扫描执行配置",
    fields: [
      { key: "PoolSize", label: "并发数", valueType: "number" },
      { key: "Md5Retry", label: "MD5 重试次数", valueType: "number" },
      { key: "Md5CountLength", label: "MD5 采样长度", valueType: "number" },
    ],
  },
  {
    key: "bak",
    title: "bak 备份配置",
    fields: [{ key: "StartPathBak", label: "备份目录" }],
  },
  {
    key: "gis",
    title: "gis 地理编码配置",
    fields: [{ key: "key", label: "高德 Key", secret: true }],
  },
  {
    key: "batch",
    title: "batch 批处理配置",
    fields: [
      { key: "IDInsertBatchSize", label: "图片记录插入批次", valueType: "number" },
      { key: "IDDeleteBatchSize", label: "图片记录删除批次", valueType: "number" },
      { key: "GDUpdateBatchSize", label: "地理数据更新批次", valueType: "number" },
    ],
  },
];
