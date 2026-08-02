package appversion

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizePlatformAcceptsOnlyAndroid(t *testing.T) {
	if got := normalizePlatform(" Android "); got != "android" {
		t.Fatalf("expected android, got %q", got)
	}

	if got := normalizePlatform("ios"); got != "" {
		t.Fatalf("expected unsupported platform to be empty, got %q", got)
	}
}

func TestParseAppVersionFilters(t *testing.T) {
	filters, err := parseAppVersionFilters(url.Values{
		"search":       {" 1.1 "},
		"force_update": {" FALSE "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filters.Search != "1.1" {
		t.Fatalf("expected trimmed search, got %q", filters.Search)
	}
	if !filters.ForceUpdateEnabled || filters.ForceUpdate {
		t.Fatalf("expected optional update filter set to false, got %+v", filters)
	}
}

func TestListAppVersionsRejectsInvalidFilterBeforeQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "platform", Value: "android"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/app-version/android/history?force_update=required", nil)

	listAppVersions(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAppVersionResponse(t *testing.T) {
	version := AppVersion{
		Platform:               "android",
		LatestVersionName:      "1.0.2",
		LatestVersionCode:      102,
		MinRequiredVersionCode: 101,
		ForceUpdate:            false,
		PlayStoreURL:           "https://play.google.com/store/apps/details?id=com.example",
		Message:                "Nova versao disponivel.",
	}

	response := appVersionResponse(version)
	if response.Platform != "android" {
		t.Fatalf("expected android platform, got %q", response.Platform)
	}
	if response.ID != version.ID {
		t.Fatalf("expected id %d, got %d", version.ID, response.ID)
	}
	if response.LatestVersionCode != 102 {
		t.Fatalf("expected latest version code 102, got %d", response.LatestVersionCode)
	}
	if response.MinRequiredVersionCode != 101 {
		t.Fatalf("expected min required version code 101, got %d", response.MinRequiredVersionCode)
	}
}

func TestListAppVersionsRejectsInvalidPaginationBeforeQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "platform", Value: "android"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/app-version/android/history?page=0", nil)

	listAppVersions(context)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
