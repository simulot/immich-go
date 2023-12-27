package myflag

import "testing"

func Test_BoolFn(t *testing.T) {
	tc := []struct {
		name         string
		defaultValue bool
		want         bool
		wantErr      bool
	}{
		{
			name:         "",
			want:         true,
			defaultValue: true,
		},
		{
			name:         "",
			want:         true,
			defaultValue: false,
		},
		{
			name: "true",
			want: true,
		},
		{
			name: "false",
			want: false,
		},
		{
			name: "1",
			want: true,
		},
		{
			name: "T",
			want: true,
		},
		{
			name: "F",
			want: false,
		},
		{
			name: "0",
			want: false,
		},
		{
			name:    "let's be affirmative",
			want:    false,
			wantErr: true,
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			var b bool
			fn := BoolFlagFn(&b, c.defaultValue)

			err := fn(c.name)
			if (err == nil && c.wantErr) || (err != nil && !c.wantErr) {
				t.Errorf("fn(%q)=%v, expecting error: %v", c.name, err, c.wantErr)
				return
			}
			if b != c.want {
				t.Errorf("fn(%q) set b to %v, expecting: %v", c.name, b, c.want)
			}
		})
	}
}
