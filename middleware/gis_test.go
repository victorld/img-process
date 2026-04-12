package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"img_process/cons"
)

func TestGetGisDataFromJson(t *testing.T) {
	locJSON := `{"regeocode":{"formatted_address":"湖北省武汉市洪山区关山街道","addressComponent":{"province":"湖北省","district":"洪山区","township":"关山街道","streetNumber":{"street":"软件园中路"}}}}`

	gisData, err := GetGisDataFromJson(locJSON)
	if err != nil {
		t.Fatalf("GetGisDataFromJson returned error: %v", err)
	}
	if gisData.LocAddr == "" || gisData.LocStreet == "" {
		t.Fatalf("unexpected gisData: %#v", gisData)
	}
}

func TestGetGisDataFromJsonInvalidPayload(t *testing.T) {
	if _, err := GetGisDataFromJson(`{"foo":"bar"}`); err == nil {
		t.Fatal("expected error for invalid payload")
	}
}

func TestGetLocationAddressOnlineStatusError(t *testing.T) {
	oldBaseURL := amapBaseURL
	oldClient := amapClient
	amapBaseURL = "http://example.invalid"
	amapClient = &http.Client{Timeout: time.Second}
	t.Cleanup(func() {
		amapBaseURL = oldBaseURL
		amapClient = oldClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	amapBaseURL = server.URL
	cons.GisKey = "test"

	if _, err := GetLocationAddressOnline("114.279656,30.559343"); err == nil {
		t.Fatal("expected status error")
	}
}
