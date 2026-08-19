package migrations

import "gorm.io/gorm"

func addExpenseAdvanceStatus(db *gorm.DB) error {
	return db.Exec(`
		ALTER TABLE expenses
			ADD COLUMN is_advanced BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN advanced_at TIMESTAMPTZ;
		CREATE INDEX idx_expenses_user_advanced_at ON expenses (user_id, advanced_at);
		ALTER TABLE expenses
			ADD CONSTRAINT chk_expenses_advanced_at
			CHECK ((is_advanced = FALSE AND advanced_at IS NULL) OR (is_advanced = TRUE AND advanced_at IS NOT NULL)),
			ADD CONSTRAINT chk_expenses_advance_type
			CHECK (is_advanced = FALSE OR LOWER(type) IN ('única', 'unica', 'parcelada'));
	`).Error
}
