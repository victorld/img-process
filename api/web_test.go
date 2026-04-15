package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func modelErr(message string) error {
	return &stubErr{message: message}
}

type stubErr struct {
	message string
}

func (e *stubErr) Error() string {
	return e.message
}
