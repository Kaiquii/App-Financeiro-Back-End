package appversion

import "time"

type AppVersion struct {
	ID                     uint      `json:"id" gorm:"primaryKey"`
	Platform               string    `json:"platform" gorm:"type:varchar(30);index;uniqueIndex:idx_app_versions_platform_code"`
	LatestVersionName      string    `json:"latest_version_name"`
	LatestVersionCode      int       `json:"latest_version_code" gorm:"uniqueIndex:idx_app_versions_platform_code"`
	MinRequiredVersionCode int       `json:"min_required_version_code"`
	ForceUpdate            bool      `json:"force_update"`
	PlayStoreURL           string    `json:"play_store_url"`
	Message                string    `json:"message" gorm:"type:text"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SaveAppVersionRequest struct {
	LatestVersionName      string `json:"latest_version_name" binding:"required"`
	LatestVersionCode      int    `json:"latest_version_code" binding:"required,gt=0"`
	MinRequiredVersionCode int    `json:"min_required_version_code" binding:"gte=0"`
	ForceUpdate            bool   `json:"force_update"`
	PlayStoreURL           string `json:"play_store_url" binding:"required"`
	Message                string `json:"message"`
}

type AppVersionResponse struct {
	ID                     uint      `json:"id"`
	Platform               string    `json:"platform"`
	LatestVersionName      string    `json:"latest_version_name"`
	LatestVersionCode      int       `json:"latest_version_code"`
	MinRequiredVersionCode int       `json:"min_required_version_code"`
	ForceUpdate            bool      `json:"force_update"`
	PlayStoreURL           string    `json:"play_store_url"`
	Message                string    `json:"message"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}
