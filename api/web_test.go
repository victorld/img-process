package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"img_process/cons"
	"img_process/dao"
	"img_process/model"

	"github.com/gin-gonic/gin"
)

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
				SummaryJSON: `{"dirTotal":28}`,
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
				ID                  uint  `json:"id"`
				TotalCount          int64 `json:"totalCount"`
				TotalFolderCount    int64 `json:"totalFolderCount"`
				PendingActionCount  int64 `json:"pendingActionCount"`
				ExecutedActionCount int64 `json:"executedActionCount"`
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
	if item.PendingActionCount != 6 {
		t.Fatalf("pendingActionCount = %d, want 6", item.PendingActionCount)
	}
	if item.ExecutedActionCount != 2 {
		t.Fatalf("executedActionCount = %d, want 2", item.ExecutedActionCount)
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
			Server struct {
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
	if resp.Data.Server.HttpPort != "8081" || resp.Data.Server.StartPath != "/data/pic-lab" || resp.Data.Server.PoolSize != 8 || !resp.Data.Server.SqlDebug {
		t.Fatalf("legacy server fields mismatch: %+v", resp.Data.Server)
	}
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
