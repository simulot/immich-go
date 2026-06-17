package app

import "testing"

func TestRedactFlagValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		value    string
		want     string
	}{
		{
			name:     "nextcloud password is redacted",
			flagName: "nextcloud-password",
			value:    "abcdef123456",
			want:     "********3456",
		},
		{
			name:     "api key keeps suffix",
			flagName: "api-key",
			value:    "abcdef123456",
			want:     "********3456",
		},
		{
			name:     "short secret fully masked",
			flagName: "client-secret",
			value:    "abcd",
			want:     "****",
		},
		{
			name:     "non sensitive flag unchanged",
			flagName: "nextcloud-user",
			value:    "myuseraccount",
			want:     "myuseraccount",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := redactFlagValue(tt.flagName, tt.value); got != tt.want {
				t.Fatalf("redactFlagValue(%q, %q) = %q, want %q", tt.flagName, tt.value, got, tt.want)
			}
		})
	}
}
