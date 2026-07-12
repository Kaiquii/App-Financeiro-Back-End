package appversion

import "gorm.io/gorm"

func PrepareSchema(db *gorm.DB) {
	if db == nil {
		return
	}

	if db.Migrator().HasIndex(&AppVersion{}, "idx_app_versions_platform") {
		_ = db.Migrator().DropIndex(&AppVersion{}, "idx_app_versions_platform")
	}
}
