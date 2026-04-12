package api

import (
	"bytes"
	"img_process/model"
	"img_process/tools"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ensureLogger() {
	if tools.Logger == nil {
		tools.Logger = zap.NewNop().Sugar()
	}
}

func TestDoScanImgAccepted(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)
	oldCreateJob := createJobFunc
	createJobFunc = func(source string, scheduleID *uint, arg model.DoScanImgArg) (model.ScanJobDB, error) {
		return model.ScanJobDB{CommonModel: model.CommonModel{ID: 123}, JobUUID: "job-123", Status: model.JobStatusPending}, nil
	}
	t.Cleanup(func() {
		createJobFunc = oldCreateJob
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/img/scan", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(ImgRecordOwnApi).DoScanImg(c)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestDoScanImgBadJSON(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/img/scan", bytes.NewBufferString(`{`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(ImgRecordOwnApi).DoScanImg(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteMD5DupFilesRequiresScanUUID(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/img/delete", nil)

	new(ImgRecordOwnApi).DeleteMD5DupFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteMD5DupFilesSuccess(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)
	oldDelete := deleteDumpFileFunc
	called := false
	deleteDumpFileFunc = func(path string) {
		called = true
	}
	t.Cleanup(func() {
		deleteDumpFileFunc = oldDelete
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/img/delete?scanUuid=2025-01-25-20-07-24_f0530738db1411ef97c02656", nil)

	new(ImgRecordOwnApi).DeleteMD5DupFiles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !called {
		t.Fatal("deleteDumpFileFunc should be called")
	}
}

func TestDeleteMD5DupFilesRejectsInvalidScanUUID(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodDelete, "/img/delete?scanUuid=../bad", nil)

	new(ImgRecordOwnApi).DeleteMD5DupFiles(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
