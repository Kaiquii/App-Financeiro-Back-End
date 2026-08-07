package migrations

import "gorm.io/gorm"

func addExpensePaymentSplits(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE expense_payment_splits (
			id BIGSERIAL PRIMARY KEY,
			expense_id BIGINT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
			payment_source TEXT NOT NULL,
			amount NUMERIC NOT NULL CHECK (amount > 0),
			CONSTRAINT uq_expense_payment_splits_source UNIQUE (expense_id, payment_source)
		);
		CREATE INDEX idx_expense_payment_splits_expense_id ON expense_payment_splits (expense_id);
	`).Error
}
