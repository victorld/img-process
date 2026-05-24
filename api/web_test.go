package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"

	"github.com/gin-gonic/gin"
)

func escapeJSONString(value string) string {
	encoded, _ := json.Marshal(value)
	return strings.Trim(string(encoded), `"`)
}

func TestListJobsReturnsActionAndFolderCounts(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldListJobs := listJobsFunc
	oldCountGrouped := countGroupedActionItemsByJobsFunc
	listJobsFunc = func(search model.ScanJobSearch) ([]model.ScanJobDB, int64, error) {
		startAt := time.Date(2026, 4, 15, 18, 40, 8, 0, time.Local)
		return []model.ScanJobDB{
			{
				CommonModel: model.CommonModel{ID: 11},
				JobUUID:     "job-uuid-11",
				Status:      model.JobStatusSucceeded,
				Source:      model.JobSourceManual,
				TotalCount:  463,
				StartAt:     &startAt,
				SummaryJSON: `{"dirTotal":28,"dirTotalBak":19,"fileTotalBak":431}`,
			},
		}, 1, nil
	}
	countGroupedActionItemsByJobsFunc = func(jobIDs []uint) (map[uint]model.ScanActionGroupedCounts, error) {
		if len(jobIDs) != 1 || jobIDs[0] != 11 {
			t.Fatalf("jobIDs = %v, want [11]", jobIDs)
		}
		return map[uint]model.ScanActionGroupedCounts{
			11: {
				Pending:  model.ScanActionCounts{Total: 6},
				Executed: model.ScanActionCounts{Total: 2},
			},
		}, nil
	}
	t.Cleanup(func() {
		listJobsFunc = oldListJobs
		countGroupedActionItemsByJobsFunc = oldCountGrouped
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs?page=1&pageSize=20", nil)

	new(WebAPI).ListJobs(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			Total int64 `json:"total"`
			List  []struct {
				ID                     uint  `json:"id"`
				TotalCount             int64 `json:"totalCount"`
				TotalFolderCount       int64 `json:"totalFolderCount"`
				TotalBackupFileCount   int64 `json:"totalBackupFileCount"`
				TotalBackupFolderCount int64 `json:"totalBackupFolderCount"`
				PendingActionCount     int64 `json:"pendingActionCount"`
				ExecutedActionCount    int64 `json:"executedActionCount"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Total != 1 {
		t.Fatalf("total = %d, want 1", resp.Data.Total)
	}
	if len(resp.Data.List) != 1 {
		t.Fatalf("list len = %d, want 1", len(resp.Data.List))
	}
	item := resp.Data.List[0]
	if item.ID != 11 {
		t.Fatalf("id = %d, want 11", item.ID)
	}
	if item.TotalFolderCount != 28 {
		t.Fatalf("totalFolderCount = %d, want 28", item.TotalFolderCount)
	}
	if item.TotalCount != 463 {
		t.Fatalf("totalCount = %d, want 463", item.TotalCount)
	}
	if item.TotalBackupFolderCount != 19 {
		t.Fatalf("totalBackupFolderCount = %d, want 19", item.TotalBackupFolderCount)
	}
	if item.TotalBackupFileCount != 431 {
		t.Fatalf("totalBackupFileCount = %d, want 431", item.TotalBackupFileCount)
	}
	if item.PendingActionCount != 6 {
		t.Fatalf("pendingActionCount = %d, want 6", item.PendingActionCount)
	}
	if item.ExecutedActionCount != 2 {
		t.Fatalf("executedActionCount = %d, want 2", item.ExecutedActionCount)
	}
}

func TestListJobsDoesNotFallbackPendingActionCountToDumpFileCnt(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldListJobs := listJobsFunc
	oldCountGrouped := countGroupedActionItemsByJobsFunc
	listJobsFunc = func(search model.ScanJobSearch) ([]model.ScanJobDB, int64, error) {
		return []model.ScanJobDB{
			{
				CommonModel: model.CommonModel{ID: 12},
				JobUUID:     "job-uuid-12",
				Status:      model.JobStatusSucceeded,
				Source:      model.JobSourceManual,
				SummaryJSON: `{"DumpFileCnt":3}`,
			},
		}, 1, nil
	}
	countGroupedActionItemsByJobsFunc = func(jobIDs []uint) (map[uint]model.ScanActionGroupedCounts, error) {
		return map[uint]model.ScanActionGroupedCounts{
			12: {},
		}, nil
	}
	t.Cleanup(func() {
		listJobsFunc = oldListJobs
		countGroupedActionItemsByJobsFunc = oldCountGrouped
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs?page=1&pageSize=20", nil)

	new(WebAPI).ListJobs(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			List []struct {
				PendingActionCount int64 `json:"pendingActionCount"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data.List) != 1 {
		t.Fatalf("list len = %d, want 1", len(resp.Data.List))
	}
	if resp.Data.List[0].PendingActionCount != 0 {
		t.Fatalf("pendingActionCount = %d, want 0", resp.Data.List[0].PendingActionCount)
	}
}

func TestDeleteJobSuccess(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldDeleteJob := deleteJobFunc
	deleteJobFunc = func(jobID uint) (dao.ScanJobDeleteCounts, error) {
		if jobID != 42 {
			t.Fatalf("jobID = %d, want 42", jobID)
		}
		return dao.ScanJobDeleteCounts{
			ActionItems: 3,
			Events:      2,
			Logs:        1,
			Schedules:   1,
			Jobs:        1,
		}, nil
	}
	t.Cleanup(func() {
		deleteJobFunc = oldDeleteJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/jobs/42", nil)

	new(WebAPI).DeleteJob(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			ID          uint  `json:"id"`
			ActionItems int64 `json:"actionItems"`
			Events      int64 `json:"events"`
			Logs        int64 `json:"logs"`
			Schedules   int64 `json:"schedules"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.ID != 42 || resp.Data.ActionItems != 3 || resp.Data.Events != 2 || resp.Data.Logs != 1 || resp.Data.Schedules != 1 {
		t.Fatalf("response data = %+v", resp.Data)
	}
}

func TestDeleteJobRejectsActiveJob(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldDeleteJob := deleteJobFunc
	deleteJobFunc = func(jobID uint) (dao.ScanJobDeleteCounts, error) {
		return dao.ScanJobDeleteCounts{}, modelErr("pending or running jobs cannot be deleted")
	}
	t.Cleanup(func() {
		deleteJobFunc = oldDeleteJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/api/jobs/42", nil)

	new(WebAPI).DeleteJob(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListJobActionItemsReturnsGroupedCounts(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldListActionItems := listActionItemsFunc
	listActionItemsFunc = func(search model.ScanActionItemSearch) ([]model.ScanActionItemView, model.ScanActionCounts, model.ScanActionGroupedCounts, int64, error) {
		if search.Tab != "executed" {
			t.Fatalf("search.Tab = %q, want executed", search.Tab)
		}
		if search.ActionType != model.ActionTypeRename {
			t.Fatalf("search.ActionType = %q, want rename", search.ActionType)
		}
		return []model.ScanActionItemView{{ID: 11, ActionType: model.ActionTypeRename}}, model.ScanActionCounts{Rename: 1, Total: 1}, model.ScanActionGroupedCounts{
			Pending:  model.ScanActionCounts{Rename: 1, Total: 1},
			Executed: model.ScanActionCounts{Rename: 2, Total: 2},
			Error:    model.ScanActionCounts{Rename: 1, Total: 1},
		}, 2, nil
	}
	t.Cleanup(func() {
		listActionItemsFunc = oldListActionItems
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs/42/action-items?tab=executed&type=rename&page=1&pageSize=20", nil)

	new(WebAPI).ListJobActionItems(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			Total         int64                         `json:"total"`
			Counts        model.ScanActionCounts        `json:"counts"`
			GroupedCounts model.ScanActionGroupedCounts `json:"groupedCounts"`
			List          []model.ScanActionItemView    `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data.List) != 1 || resp.Data.List[0].ActionType != model.ActionTypeRename {
		t.Fatalf("list = %+v", resp.Data.List)
	}
	if resp.Data.GroupedCounts.Executed.Rename != 2 {
		t.Fatalf("executed rename = %d, want 2", resp.Data.GroupedCounts.Executed.Rename)
	}
	if resp.Data.GroupedCounts.Error.Rename != 1 {
		t.Fatalf("error rename = %d, want 1", resp.Data.GroupedCounts.Error.Rename)
	}
}

func TestGetJobBackupDiffReturnsArtifactItems(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldWorkDir := cons.WorkDir
	cons.WorkDir = t.TempDir()
	t.Cleanup(func() {
		cons.WorkDir = oldWorkDir
	})
	scanUUID := "2026-05-24-09-00-23_testscan"
	artifactDir := filepath.Join(cons.WorkDir, "log", "dump_delete_file", scanUUID)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	newPath := filepath.Join(artifactDir, "bak_new_file_list")
	deletePath := filepath.Join(artifactDir, "bak_delete_file_list")
	if err := os.WriteFile(newPath, []byte("2024-02-02|IMG_0001.JPG\n2024-02-02-trip|IMG_0002.JPG\nbad-key\n"), 0o644); err != nil {
		t.Fatalf("write new artifact: %v", err)
	}
	if err := os.WriteFile(deletePath, []byte("2020-08-09|VID_0001.MP4\n"), 0o644); err != nil {
		t.Fatalf("write delete artifact: %v", err)
	}

	oldGetJob := getJobFunc
	getJobFunc = func(jobID uint) (model.ScanJobDB, error) {
		if jobID != 42 {
			t.Fatalf("jobID = %d, want 42", jobID)
		}
		return model.ScanJobDB{
			CommonModel: model.CommonModel{ID: 42},
			ScanUUID:    scanUUID,
			SummaryJSON: `{"BakNewFile":{"count":3,"sample":["sample-only|IMG.JPG"],"artifactPath":"` + escapeJSONString(newPath) + `"},"BakDeleteFile":{"count":1,"sample":[],"artifactPath":"` + escapeJSONString(deletePath) + `"}}`,
		}, nil
	}
	t.Cleanup(func() {
		getJobFunc = oldGetJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs/42/backup-diff", nil)

	new(WebAPI).GetJobBackupDiff(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Data struct {
			NewFiles struct {
				Count    int  `json:"count"`
				Complete bool `json:"complete"`
				Items    []struct {
					Key            string `json:"key"`
					DirectoryLabel string `json:"directoryLabel"`
					FileName       string `json:"fileName"`
					Date           string `json:"date"`
				} `json:"items"`
			} `json:"newFiles"`
			DeletedFiles struct {
				Count int `json:"count"`
			} `json:"deletedFiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Data.NewFiles.Complete {
		t.Fatal("newFiles should be complete")
	}
	if resp.Data.NewFiles.Count != 3 || len(resp.Data.NewFiles.Items) != 3 {
		t.Fatalf("newFiles count/items = %d/%d, want 3/3", resp.Data.NewFiles.Count, len(resp.Data.NewFiles.Items))
	}
	first := resp.Data.NewFiles.Items[0]
	if first.DirectoryLabel != "2024-02-02" || first.FileName != "IMG_0001.JPG" || first.Date != "2024-02-02" {
		t.Fatalf("first item = %+v", first)
	}
	if resp.Data.NewFiles.Items[2].Date != "" {
		t.Fatalf("invalid date item date = %q, want empty", resp.Data.NewFiles.Items[2].Date)
	}
	if resp.Data.DeletedFiles.Count != 1 {
		t.Fatalf("deleted count = %d, want 1", resp.Data.DeletedFiles.Count)
	}
}

func TestGetJobBackupDiffFallsBackToSampleForUnsafeArtifact(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldWorkDir := cons.WorkDir
	cons.WorkDir = t.TempDir()
	t.Cleanup(func() {
		cons.WorkDir = oldWorkDir
	})

	oldGetJob := getJobFunc
	getJobFunc = func(jobID uint) (model.ScanJobDB, error) {
		return model.ScanJobDB{
			CommonModel: model.CommonModel{ID: jobID},
			ScanUUID:    "2026-05-24-09-00-23_testscan",
			SummaryJSON: `{"bakNewFile":{"count":2,"sample":["2025-05-01|IMG_1.JPG"],"artifactPath":"` + escapeJSONString(filepath.Join(cons.WorkDir, "log", "dump_delete_file", "other", "bak_new_file_list")) + `"}}`,
		}, nil
	}
	t.Cleanup(func() {
		getJobFunc = oldGetJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs/42/backup-diff", nil)

	new(WebAPI).GetJobBackupDiff(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Data struct {
			NewFiles struct {
				Complete bool `json:"complete"`
				Items    []struct {
					Key string `json:"key"`
				} `json:"items"`
			} `json:"newFiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.NewFiles.Complete {
		t.Fatal("newFiles should be incomplete when artifact path is unsafe")
	}
	if len(resp.Data.NewFiles.Items) != 1 || resp.Data.NewFiles.Items[0].Key != "2025-05-01|IMG_1.JPG" {
		t.Fatalf("items = %+v", resp.Data.NewFiles.Items)
	}
}

func TestGetJobBackupDiffReturnsEmptyGroups(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldGetJob := getJobFunc
	getJobFunc = func(jobID uint) (model.ScanJobDB, error) {
		return model.ScanJobDB{CommonModel: model.CommonModel{ID: jobID}, ScanUUID: "scan-empty", SummaryJSON: `{}`}, nil
	}
	t.Cleanup(func() {
		getJobFunc = oldGetJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/jobs/42/backup-diff", nil)

	new(WebAPI).GetJobBackupDiff(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp struct {
		Data struct {
			NewFiles struct {
				Count int      `json:"count"`
				Items []string `json:"items"`
			} `json:"newFiles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.NewFiles.Count != 0 || len(resp.Data.NewFiles.Items) != 0 {
		t.Fatalf("newFiles = %+v", resp.Data.NewFiles)
	}
}

func TestGetFileAnalysisBindsFiltersAndPreviewURL(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldListFileAnalysis := listFileAnalysisFunc
	listFileAnalysisFunc = func(search model.FileAnalysisSearch) (model.FileAnalysisResult, error) {
		if search.Page != 2 || search.PageSize != 5 {
			t.Fatalf("page = %d/%d, want 2/5", search.Page, search.PageSize)
		}
		if search.FileKey != "IMG" || search.ShootDateStatus != "present" || search.GeoStatus != "full" {
			t.Fatalf("search basic filters = %+v", search)
		}
		if search.LocAddrKeyword != "上海" || search.ShootDateStart != "2024-01-01" || search.ShootDateEnd != "2024-12-31" {
			t.Fatalf("search range filters = %+v", search)
		}
		return model.FileAnalysisResult{
			Summary: model.FileAnalysisSummary{TotalCount: 1, WithShootDateCount: 1, WithLocNumCount: 1, WithLocAddrCount: 1},
			List: []model.FileAnalysisItem{{
				ID:       9,
				ImgKey:   "2024-01-02|IMG_0001.JPG",
				DirDate:  "2024-01-02",
				FileName: "IMG_0001.JPG",
			}},
			Total: 1,
		}, nil
	}
	t.Cleanup(func() {
		listFileAnalysisFunc = oldListFileAnalysis
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/files/analysis?page=2&pageSize=5&fileKey=IMG&shootDateStatus=present&geoStatus=full&locAddrKeyword=%E4%B8%8A%E6%B5%B7&shootDateStart=2024-01-01&shootDateEnd=2024-12-31", nil)

	new(WebAPI).GetFileAnalysis(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var resp struct {
		Data struct {
			List []struct {
				PreviewURL string `json:"previewUrl"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Data.List) != 1 {
		t.Fatalf("list len = %d, want 1", len(resp.Data.List))
	}
	if !strings.Contains(resp.Data.List[0].PreviewURL, "/api/files/analysis/preview?") || !strings.Contains(resp.Data.List[0].PreviewURL, "quality=thumb") {
		t.Fatalf("previewUrl = %q", resp.Data.List[0].PreviewURL)
	}
}

func TestResolveFileAnalysisPreviewPath(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "2024", "2024-01", "2024-01-02", "IMG_0001.JPG")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	oldStartPath := cons.StartPath
	oldStartPathBak := cons.StartPathBak
	cons.StartPath = root
	cons.StartPathBak = ""
	t.Cleanup(func() {
		cons.StartPath = oldStartPath
		cons.StartPathBak = oldStartPathBak
	})

	got, err := resolveFileAnalysisPreviewPath("2024-01-02|IMG_0001.JPG")
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}
	if got != filePath {
		t.Fatalf("path = %q, want %q", got, filePath)
	}
	fallbackPath := filepath.Join(root, "2024", "imported-album", "IMG_0002.JPG")
	if err := os.MkdirAll(filepath.Dir(fallbackPath), 0o755); err != nil {
		t.Fatalf("mkdir fallback: %v", err)
	}
	if err := os.WriteFile(fallbackPath, []byte("fallback"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}
	got, err = resolveFileAnalysisPreviewPath("2024-01-02|IMG_0002.JPG")
	if err != nil {
		t.Fatalf("resolve fallback path: %v", err)
	}
	if got != fallbackPath {
		t.Fatalf("fallback path = %q, want %q", got, fallbackPath)
	}
	if _, err := resolveFileAnalysisPreviewPath("2024-01-02|../IMG_0001.JPG"); err == nil {
		t.Fatalf("expected invalid img key to fail")
	}
	if _, err := resolveFileAnalysisPreviewPath("2024-01-02|MISSING.JPG"); err == nil {
		t.Fatalf("expected missing file to fail")
	}
}

func TestModifyJobShootTimeActionItemSuccess(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeModifyShootTimeActionItemFunc
	executeModifyShootTimeActionItemFunc = func(jobID uint, itemID uint) error {
		if jobID != 42 {
			t.Fatalf("jobID = %d, want 42", jobID)
		}
		if itemID != 9 {
			t.Fatalf("itemID = %d, want 9", itemID)
		}
		return nil
	}
	t.Cleanup(func() {
		executeModifyShootTimeActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/modify-shoot-time", nil)

	new(WebAPI).ModifyJobShootTimeActionItem(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestModifyJobShootTimeActionItemBadRequest(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeModifyShootTimeActionItemFunc
	executeModifyShootTimeActionItemFunc = func(jobID uint, itemID uint) error {
		return modelErr("target date is empty")
	}
	t.Cleanup(func() {
		executeModifyShootTimeActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/modify-shoot-time", nil)

	new(WebAPI).ModifyJobShootTimeActionItem(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestMoveJobActionItemSuccess(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeMoveActionItemFunc
	executeMoveActionItemFunc = func(jobID uint, itemID uint) error {
		if jobID != 42 {
			t.Fatalf("jobID = %d, want 42", jobID)
		}
		if itemID != 9 {
			t.Fatalf("itemID = %d, want 9", itemID)
		}
		return nil
	}
	t.Cleanup(func() {
		executeMoveActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/move", nil)

	new(WebAPI).MoveJobActionItem(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMoveJobActionItemBadRequest(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeMoveActionItemFunc
	executeMoveActionItemFunc = func(jobID uint, itemID uint) error {
		return modelErr("move target path is empty")
	}
	t.Cleanup(func() {
		executeMoveActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/move", nil)

	new(WebAPI).MoveJobActionItem(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRenameJobActionItemSuccess(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeRenameActionItemFunc
	executeRenameActionItemFunc = func(jobID uint, itemID uint) error {
		if jobID != 42 {
			t.Fatalf("jobID = %d, want 42", jobID)
		}
		if itemID != 9 {
			t.Fatalf("itemID = %d, want 9", itemID)
		}
		return nil
	}
	t.Cleanup(func() {
		executeRenameActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/rename", nil)

	new(WebAPI).RenameJobActionItem(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRenameJobActionItemBadRequest(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldExecute := executeRenameActionItemFunc
	executeRenameActionItemFunc = func(jobID uint, itemID uint) error {
		return modelErr("rename target path is empty")
	}
	t.Cleanup(func() {
		executeRenameActionItemFunc = oldExecute
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "42"}, {Key: "itemId", Value: "9"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/api/jobs/42/action-items/9/rename", nil)

	new(WebAPI).RenameJobActionItem(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetSystemStatusReturnsGroupedMaskedConfig(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldDbUsername := cons.DbUsername
	oldDbPassword := cons.DbPassword
	oldDbHost := cons.DbHost
	oldDbPort := cons.DbPort
	oldDbName := cons.DbName
	oldDbConfig := cons.DbConfig
	oldHttpPort := cons.HttpPort
	oldHttpUsername := cons.HttpUsername
	oldHttpPassword := cons.HttpPassword
	oldStartPath := cons.StartPath
	oldStartPathBak := cons.StartPathBak
	oldDeleteShow := cons.DeleteShow
	oldMoveFileShow := cons.MoveFileShow
	oldModifyDateShow := cons.ModifyDateShow
	oldRenameFileShow := cons.RenameFileShow
	oldMd5Show := cons.Md5Show
	oldDeleteAction := cons.DeleteAction
	oldMoveFileAction := cons.MoveFileAction
	oldModifyDateAction := cons.ModifyDateAction
	oldRenameFileAction := cons.RenameFileAction
	oldImgCache := cons.ImgCache
	oldSyncTable := cons.SyncTable
	oldTruncateTable := cons.TruncateTable
	oldSqlDebug := cons.SqlDebug
	oldPoolSize := cons.PoolSize
	oldMd5Retry := cons.Md5Retry
	oldMd5CountLength := cons.Md5CountLength
	oldGisKey := cons.GisKey
	oldIDInsertBatchSize := cons.IDInsertBatchSize
	oldIDDeleteBatchSize := cons.IDDeleteBatchSize
	oldGDUpdateBatchSize := cons.GDUpdateBatchSize
	oldAppConfig := cons.AppConfig
	t.Cleanup(func() {
		cons.DbUsername = oldDbUsername
		cons.DbPassword = oldDbPassword
		cons.DbHost = oldDbHost
		cons.DbPort = oldDbPort
		cons.DbName = oldDbName
		cons.DbConfig = oldDbConfig
		cons.HttpPort = oldHttpPort
		cons.HttpUsername = oldHttpUsername
		cons.HttpPassword = oldHttpPassword
		cons.StartPath = oldStartPath
		cons.StartPathBak = oldStartPathBak
		cons.DeleteShow = oldDeleteShow
		cons.MoveFileShow = oldMoveFileShow
		cons.ModifyDateShow = oldModifyDateShow
		cons.RenameFileShow = oldRenameFileShow
		cons.Md5Show = oldMd5Show
		cons.DeleteAction = oldDeleteAction
		cons.MoveFileAction = oldMoveFileAction
		cons.ModifyDateAction = oldModifyDateAction
		cons.RenameFileAction = oldRenameFileAction
		cons.ImgCache = oldImgCache
		cons.SyncTable = oldSyncTable
		cons.TruncateTable = oldTruncateTable
		cons.SqlDebug = oldSqlDebug
		cons.PoolSize = oldPoolSize
		cons.Md5Retry = oldMd5Retry
		cons.Md5CountLength = oldMd5CountLength
		cons.GisKey = oldGisKey
		cons.IDInsertBatchSize = oldIDInsertBatchSize
		cons.IDDeleteBatchSize = oldIDDeleteBatchSize
		cons.GDUpdateBatchSize = oldGDUpdateBatchSize
		cons.AppConfig = oldAppConfig
	})

	cons.DbUsername = "root"
	cons.DbPassword = "secret"
	cons.DbHost = "db-host"
	cons.DbPort = "3306"
	cons.DbName = "img"
	cons.DbConfig = "charset=utf8"
	cons.HttpPort = "8081"
	cons.HttpUsername = "admin"
	cons.HttpPassword = "admin"
	cons.StartPath = "/data/pic-lab"
	cons.StartPathBak = "/data/bak"
	cons.DeleteShow = true
	cons.MoveFileShow = true
	cons.ModifyDateShow = false
	cons.RenameFileShow = true
	cons.Md5Show = true
	cons.DeleteAction = false
	cons.MoveFileAction = true
	cons.ModifyDateAction = false
	cons.RenameFileAction = true
	cons.ImgCache = false
	cons.SyncTable = true
	cons.TruncateTable = false
	cons.SqlDebug = true
	cons.PoolSize = 8
	cons.Md5Retry = 3
	cons.Md5CountLength = 65536
	cons.GisKey = "abc"
	cons.IDInsertBatchSize = 1000
	cons.IDDeleteBatchSize = 300
	cons.GDUpdateBatchSize = 2000
	cons.AppConfig.Basic.ColorOutput = true

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/system/status", nil)

	new(WebAPI).GetSystemStatus(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		Data struct {
			ConfigFile string `json:"configFile"`
			Config     struct {
				Database map[string]any `json:"database"`
				Server   map[string]any `json:"server"`
				ScanArgs map[string]any `json:"scanArgs"`
				Basic    map[string]any `json:"basic"`
				Cache    map[string]any `json:"cache"`
				Dump     map[string]any `json:"dump"`
				Bak      map[string]any `json:"bak"`
				Gis      map[string]any `json:"gis"`
				Batch    map[string]any `json:"batch"`
			} `json:"config"`
			ReadonlySections []string `json:"readonlySections"`
			Server           struct {
				HttpPort  string `json:"httpPort"`
				StartPath string `json:"startPath"`
				PoolSize  int    `json:"poolSize"`
				SqlDebug  bool   `json:"sqlDebug"`
			} `json:"server"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Config.Database["DbPassword"] != "s*****" {
		t.Fatalf("DbPassword = %v, want s*****", resp.Data.Config.Database["DbPassword"])
	}
	if resp.Data.Config.Server["HttpPassword"] != "a****" {
		t.Fatalf("HttpPassword = %v, want a****", resp.Data.Config.Server["HttpPassword"])
	}
	if resp.Data.Config.Gis["key"] != "a**" {
		t.Fatalf("gis.key = %v, want a**", resp.Data.Config.Gis["key"])
	}
	if resp.Data.Config.Database["DbHost"] != "db-host" {
		t.Fatalf("DbHost = %v, want db-host", resp.Data.Config.Database["DbHost"])
	}
	if resp.Data.Config.ScanArgs["StartPath"] != "/data/pic-lab" {
		t.Fatalf("StartPath = %v, want /data/pic-lab", resp.Data.Config.ScanArgs["StartPath"])
	}
	if resp.Data.Config.Basic["ColorOutput"] != true {
		t.Fatalf("ColorOutput = %v, want true", resp.Data.Config.Basic["ColorOutput"])
	}
	if resp.Data.Config.Cache["SyncTable"] != true {
		t.Fatalf("SyncTable = %v, want true", resp.Data.Config.Cache["SyncTable"])
	}
	if resp.Data.Config.Dump["PoolSize"] != float64(8) {
		t.Fatalf("PoolSize = %v, want 8", resp.Data.Config.Dump["PoolSize"])
	}
	if resp.Data.Config.Bak["StartPathBak"] != "/data/bak" {
		t.Fatalf("StartPathBak = %v, want /data/bak", resp.Data.Config.Bak["StartPathBak"])
	}
	if resp.Data.Config.Batch["GDUpdateBatchSize"] != float64(2000) {
		t.Fatalf("GDUpdateBatchSize = %v, want 2000", resp.Data.Config.Batch["GDUpdateBatchSize"])
	}
	if !containsString(resp.Data.ReadonlySections, "database") || !containsString(resp.Data.ReadonlySections, "server") {
		t.Fatalf("readonlySections = %v, want database and server", resp.Data.ReadonlySections)
	}
	if resp.Data.Server.HttpPort != "8081" || resp.Data.Server.StartPath != "/data/pic-lab" || resp.Data.Server.PoolSize != 8 || !resp.Data.Server.SqlDebug {
		t.Fatalf("legacy server fields mismatch: %+v", resp.Data.Server)
	}
}

func TestUpdateSystemSettingsRejectsReadonlySections(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/system/settings", strings.NewReader(`{"config":{"server":{"HttpPort":"9090"}}}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(WebAPI).UpdateSystemSettings(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSelectSystemDirectoryReturnsSelectedPath(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	selected := filepath.Join(root, "selected")
	if err := os.Mkdir(selected, 0755); err != nil {
		t.Fatalf("mkdir selected: %v", err)
	}
	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		if path != realPathForTest(root) {
			t.Fatalf("path = %q, want %q", path, realPathForTest(root))
		}
		if title != "选择扫描目录" {
			t.Fatalf("title = %q, want 选择扫描目录", title)
		}
		return selected + string(filepath.Separator), nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/system/select-directory", strings.NewReader(`{"path":"`+root+`","title":"选择扫描目录"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(WebAPI).SelectSystemDirectory(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Data struct {
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Path != realPathForTest(selected) {
		t.Fatalf("path = %q, want %q", resp.Data.Path, realPathForTest(selected))
	}
}

func TestSelectSystemDirectoryReportsCancel(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		return "", errors.New("已取消选择目录")
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/system/select-directory", strings.NewReader(`{"path":"/tmp"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(WebAPI).SelectSystemDirectory(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "已取消选择目录") {
		t.Fatalf("body = %s, want cancel message", w.Body.String())
	}
}

func TestSelectSystemDirectoryReportsScriptFailure(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		return "", errors.New("打开系统目录选择窗口失败：boom")
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/system/select-directory", strings.NewReader(`{"path":"/tmp"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(WebAPI).SelectSystemDirectory(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "打开系统目录选择窗口失败") {
		t.Fatalf("body = %s, want script failure message", w.Body.String())
	}
}

func TestRunSystemDirectoryPickerIgnoresStderrNoise(t *testing.T) {
	root := t.TempDir()
	oldRunner := runSystemDirectoryPickerScriptFunc
	runSystemDirectoryPickerScriptFunc = func(script string) ([]byte, []byte, error) {
		return []byte(root + "\n"), []byte("2026-05-23 22:14:11.981 osascript[72154:136702606] +[IMKClient subclass]: chose IMKClient_Modern\n"), nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerScriptFunc = oldRunner
	})

	got, err := runSystemDirectoryPicker("/tmp", "选择目录")
	if err != nil {
		t.Fatalf("run picker: %v", err)
	}
	if got != root {
		t.Fatalf("selected path = %q, want %q", got, root)
	}
}

func TestRunSystemDirectoryPickerCancelIgnoresStderrNoise(t *testing.T) {
	oldRunner := runSystemDirectoryPickerScriptFunc
	runSystemDirectoryPickerScriptFunc = func(script string) ([]byte, []byte, error) {
		return []byte("__CANCELLED__\n"), []byte("2026-05-23 22:14:11.981 osascript[72154:136702606] +[IMKClient subclass]: chose IMKClient_Modern\n"), nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerScriptFunc = oldRunner
	})

	_, err := runSystemDirectoryPicker("/tmp", "选择目录")
	if err == nil || !strings.Contains(err.Error(), "已取消选择目录") {
		t.Fatalf("err = %v, want cancel error", err)
	}
	if strings.Contains(err.Error(), "请选择有效目录") || strings.Contains(err.Error(), "IMKClient") {
		t.Fatalf("err = %v, want clean cancel error", err)
	}
}

func TestRunSystemDirectoryPickerReportsStderrOnFailure(t *testing.T) {
	oldRunner := runSystemDirectoryPickerScriptFunc
	runSystemDirectoryPickerScriptFunc = func(script string) ([]byte, []byte, error) {
		return nil, []byte("permission denied\n"), errors.New("exit status 1")
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerScriptFunc = oldRunner
	})

	_, err := runSystemDirectoryPicker("/tmp", "选择目录")
	if err == nil || !strings.Contains(err.Error(), "打开系统目录选择窗口失败：permission denied") {
		t.Fatalf("err = %v, want stderr failure detail", err)
	}
}

func TestSelectSystemDirectoryFallsBackFromMissingDefaultToParent(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "missing", "leaf")
	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		if path != realPathForTest(root) {
			t.Fatalf("default path = %q, want existing parent %q", path, realPathForTest(root))
		}
		return root, nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	got, err := selectSystemDirectory(child, "选择目录")
	if err != nil {
		t.Fatalf("select directory: %v", err)
	}
	if got != realPathForTest(root) {
		t.Fatalf("selected path = %q, want %q", got, realPathForTest(root))
	}
}

func TestSelectSystemDirectoryNormalizesRelativeSelection(t *testing.T) {
	root := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	dir := "relative-dir"
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("mkdir relative dir: %v", err)
	}
	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		return dir, nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	got, err := selectSystemDirectory("", "选择目录")
	if err != nil {
		t.Fatalf("select directory: %v", err)
	}
	want := realPathForTest(filepath.Join(root, dir))
	if got != want {
		t.Fatalf("selected path = %q, want %q", got, want)
	}
}

func TestSelectSystemDirectoryRejectsSelectedFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("not dir"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	oldPicker := runSystemDirectoryPickerFunc
	runSystemDirectoryPickerFunc = func(path string, title string) (string, error) {
		return filePath, nil
	}
	t.Cleanup(func() {
		runSystemDirectoryPickerFunc = oldPicker
	})

	_, err := selectSystemDirectory(root, "选择目录")
	if err == nil || !strings.Contains(err.Error(), "请选择有效目录") {
		t.Fatalf("err = %v, want valid directory error", err)
	}
}

func realPathForTest(path string) string {
	if abs, err := filepath.Abs(filepath.Clean(path)); err == nil {
		path = abs
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	return path
}

func TestListSystemDirectoriesReturnsOnlyDirectories(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Beta"), 0755); err != nil {
		t.Fatalf("mkdir Beta: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "alpha"), 0755); err != nil {
		t.Fatalf("mkdir alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("skip"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/system/directories?path="+root, nil)

	new(WebAPI).ListSystemDirectories(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Data struct {
			Path    string `json:"path"`
			Parent  string `json:"parent"`
			Entries []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"entries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Data.Path != root {
		t.Fatalf("path = %q, want %q", resp.Data.Path, root)
	}
	if resp.Data.Parent != filepath.Dir(root) {
		t.Fatalf("parent = %q, want %q", resp.Data.Parent, filepath.Dir(root))
	}
	if len(resp.Data.Entries) != 2 {
		t.Fatalf("entries = %+v, want two directories", resp.Data.Entries)
	}
	if resp.Data.Entries[0].Name != "alpha" || resp.Data.Entries[1].Name != "Beta" {
		t.Fatalf("entries order = %+v, want alpha then Beta", resp.Data.Entries)
	}
}

func TestListSystemDirectoriesUsesHomeWhenPathEmpty(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/system/directories", nil)

	new(WebAPI).ListSystemDirectories(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var resp struct {
		Data struct {
			Path string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home dir: %v", err)
	}
	if resp.Data.Path != filepath.Clean(home) {
		t.Fatalf("path = %q, want home %q", resp.Data.Path, filepath.Clean(home))
	}
}

func TestListSystemDirectoriesRejectsFilePath(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	filePath := filepath.Join(root, "file.txt")
	if err := os.WriteFile(filePath, []byte("not a dir"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/system/directories?path="+filePath, nil)

	new(WebAPI).ListSystemDirectories(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func modelErr(message string) error {
	return &stubErr{message: message}
}

type stubErr struct {
	message string
}

func (e *stubErr) Error() string {
	return e.message
}
