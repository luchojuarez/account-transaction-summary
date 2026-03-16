package cli

import (
	"os"
	"strings"
	"testing"
)

func TestNewAccountProcessor_S3Only(t *testing.T) {
	// Setup required env vars for email.NewSMTPNotifierFromEnv()
	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "test")
	os.Setenv("SMTP_PASSWORD", "test")
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
	}()

	tests := []struct {
		name        string
		csvPath     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "Valid S3 URI",
			csvPath:     "s3://bucket/key.csv",
			wantErr:     false,
		},
		{
			name:        "Empty path",
			csvPath:     "",
			wantErr:     true,
			errContains: "csvPath must be an S3 URI",
		},
		{
			name:        "Local file path",
			csvPath:     "./data/txns.csv",
			wantErr:     true,
			errContains: "csvPath must be an S3 URI",
		},
		{
			name:        "Invalid S3 URI (missing bucket)",
			csvPath:     "s3:///key.csv",
			wantErr:     true,
			errContains: "URI must have a bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAccountProcessor(tt.csvPath, "test@example.com", "Test User")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAccountProcessor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("NewAccountProcessor() error = %v, wantErrContains %v", err, tt.errContains)
			}
		})
	}
}
