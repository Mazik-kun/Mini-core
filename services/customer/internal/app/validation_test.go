package app


import (
	"testing"

)

func TestValidatePhoneNumber(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid +995", "+995 555 123 456", false},
		{"valid digits", "555123456", false},
		{"empty", "", true},
		{"letters", "abc123", true},
		{"too short", "12345", true},
		{"too long", "1234567890123456789", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePhoneNumber(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("got err=%v, want err=%v", err, tc.wantErr)
			}
		})
	}
}