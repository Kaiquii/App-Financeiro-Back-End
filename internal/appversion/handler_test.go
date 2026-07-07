package appversion

import "testing"

func TestNormalizePlatformAcceptsOnlyAndroid(t *testing.T) {
	if got := normalizePlatform(" Android "); got != "android" {
		t.Fatalf("expected android, got %q", got)
	}

	if got := normalizePlatform("ios"); got != "" {
		t.Fatalf("expected unsupported platform to be empty, got %q", got)
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
