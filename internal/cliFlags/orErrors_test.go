package cliflags

import "testing"

func TestOnErrorsFlag_SetAndString(t *testing.T) {
	tests := []struct {
		input    string
		expected OnErrorsFlag
		str      string
		wantErr  bool
	}{
		{"stop", OnErrorsStop, "stop", false},
		{"continue", OnErrorsNeverStop, "continue", false},
		{"retry", OnErrorsRetry, "retry", false},
		{"5", OnErrorsFlag(5), "5", false},
		{"invalid", 0, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var f OnErrorsFlag
			err := f.Set(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != tt.expected {
				t.Errorf("got %v, want %v", f, tt.expected)
			}
			if f.String() != tt.str {
				t.Errorf("String() = %q, want %q", f.String(), tt.str)
			}
		})
	}
}
