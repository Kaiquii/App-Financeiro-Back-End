package expenses

import "testing"

func TestParsePaymentStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		isPaid  bool
		enabled bool
		wantErr bool
	}{
		{name: "without filter", input: "", enabled: false},
		{name: "paid", input: "paid", isPaid: true, enabled: true},
		{name: "Portuguese paid", input: " Paga ", isPaid: true, enabled: true},
		{name: "pending", input: "pending", enabled: true},
		{name: "invalid", input: "later", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isPaid, enabled, err := parsePaymentStatus(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if isPaid != test.isPaid || enabled != test.enabled {
				t.Fatalf("got isPaid=%v enabled=%v", isPaid, enabled)
			}
		})
	}
}
