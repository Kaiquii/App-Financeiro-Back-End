package migrations

import "gorm.io/gorm"

func addUserLastActive(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE users
			ADD COLUMN last_active_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
	`).Error
}
