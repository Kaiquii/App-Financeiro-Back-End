package migrations

import "gorm.io/gorm"

func addExpensePaymentStatus(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE expenses
			ADD COLUMN IF NOT EXISTS is_paid BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ
	`).Error
}
