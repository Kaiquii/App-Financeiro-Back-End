package uploads

import (
	"os"
	"path/filepath"
	"strings"
)

const PublicPath = "/uploads"

func Dir() string {
	dir := strings.TrimSpace(os.Getenv("UPLOADS_DIR"))
	if dir == "" {
		return "uploads"
	}

	return dir
}

func PublicURL(parts ...string) string {
	pathParts := append([]string{PublicPath}, parts...)
	return filepath.ToSlash(filepath.Join(pathParts...))
}

func UserAvatarKey(userID string) string {
	return filepath.ToSlash(filepath.Join("users", userID, "avatar.jpg"))
}
