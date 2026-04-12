package api

import (
	"bytes"
	"errors"
	"img_process/model"
	"img_process/tools"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	oldScanAndSave := scanAndSaveFunc
	done := make(chan struct{}, 1)
	scanAndSaveFunc = func(arg model.DoScanImgArg) (string, error) {
		done <- struct{}{}
		return `{"ok":true}`, nil
	}
	t.Cleanup(func() {
		scanAndSaveFunc = oldScanAndSave
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/img/scan", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(ImgRecordOwnApi).DoScanImg(c)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scan goroutine did not complete")
	}
}

func TestDoScanImgConflict(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)
	scanMu.Lock()
	defer scanMu.Unlock()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/img/scan", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	new(ImgRecordOwnApi).DoScanImg(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
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
	c.Request = httptest.NewRequest(http.MethodDelete, "/img/delete?scanUuid=test-id", nil)
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

func TestDoScanImgAsyncErrorStillUnlocks(t *testing.T) {
	ensureLogger()
	gin.SetMode(gin.TestMode)
	oldScanAndSave := scanAndSaveFunc
	scanAndSaveFunc = func(arg model.DoScanImgArg) (string, error) {
		return "", errors.New("scan failed")
	}
	t.Cleanup(func() {
		scanAndSaveFunc = oldScanAndSave
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/img/scan", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	new(ImgRecordOwnApi).DoScanImg(c)

	if scanMu.TryLock() {
		scanMu.Unlock()
		return
	}

	// give goroutine a chance in slower environments
	for i := 0; i < 20; i++ {
		if scanMu.TryLock() {
			scanMu.Unlock()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("scanMu should be unlocked after async error")
}
