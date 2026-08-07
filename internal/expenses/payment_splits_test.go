package expenses

import "testing"

func TestValidatePaymentSplits(t *testing.T) {
	tests := []struct {
		name    string
		total   float64
		legacy  string
		splits  []PaymentSplitInput
		wantErr bool
	}{
		{name: "legacy source", total: 1200, legacy: "Salario"},
		{name: "split payment", total: 1200, splits: []PaymentSplitInput{{PaymentSource: "Salário", Amount: 1000}, {PaymentSource: "Renda Extra", Amount: 200}}},
		{name: "incorrect total", total: 1200, splits: []PaymentSplitInput{{PaymentSource: "Salário", Amount: 1000}}, wantErr: true},
		{name: "repeated source", total: 1200, splits: []PaymentSplitInput{{PaymentSource: "Salário", Amount: 1000}, {PaymentSource: "salario", Amount: 200}}, wantErr: true},
		{name: "conflicting contracts", total: 1200, legacy: "Salário", splits: []PaymentSplitInput{{PaymentSource: "Renda Extra", Amount: 1200}}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := ValidatePaymentSplits(test.total, test.legacy, test.splits)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestHydratePaymentSplitsKeepsLegacyDataInResponseShape(t *testing.T) {
	expense := Expense{ID: 12, Amount: 1200, PaymentSource: "Salario"}
	HydratePaymentSplits(&expense)

	if len(expense.PaymentSplits) != 1 {
		t.Fatalf("payment splits = %d, want 1", len(expense.PaymentSplits))
	}
	split := expense.PaymentSplits[0]
	if split.ExpenseID != expense.ID || split.Amount != expense.Amount || split.PaymentSource != "Salário" {
		t.Fatalf("unexpected virtual split: %+v", split)
	}
}
