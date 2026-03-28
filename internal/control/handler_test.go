package control

import (
	"encoding/base64"
	"testing"
)

func TestValidatePublicKey(t *testing.T) {
	validKey := base64.StdEncoding.EncodeToString(make([]byte, 32))

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid 32-byte key", validKey, false},
		{"not base64", "not-valid!!!", true},
		{"too short: 16 bytes", base64.StdEncoding.EncodeToString(make([]byte, 16)), true},
		{"too long: 64 bytes", base64.StdEncoding.EncodeToString(make([]byte, 64)), true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePublicKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePublicKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}
