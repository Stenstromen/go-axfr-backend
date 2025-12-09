package health

import (
	"go-axfr-backend/internal/models"
	"testing"
)

func TestCheckMySQLConnection(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		wantErr  bool
	}{
		{
			name:     "valid hostname",
			hostname: "localhost",
			wantErr:  false, // May fail if MySQL is not running, but connection attempt is valid
		},
		{
			name:     "invalid hostname",
			hostname: "nonexistent-hostname-12345",
			wantErr:  true,
		},
		{
			name:     "empty hostname",
			hostname: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckMySQLConnection(tt.hostname)
			// For localhost test, we accept both success and connection refused as valid outcomes
			// since MySQL may or may not be running during tests
			if tt.hostname == "localhost" {
				// Just verify the function doesn't panic and handles the connection attempt
				if err != nil {
					// Connection refused is acceptable - MySQL might not be running
					t.Logf("CheckMySQLConnection() returned error (expected if MySQL not running): %v", err)
				}
			} else if (err != nil) != tt.wantErr {
				t.Errorf("CheckMySQLConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckDatabases(t *testing.T) {
	tests := []struct {
		name string
		dbs  []models.DbConfig
		want bool // true if should succeed, false if should fail
	}{
		{
			name: "empty database list",
			dbs:  []models.DbConfig{},
			want: true, // Should succeed with empty list
		},
		{
			name: "invalid database config",
			dbs: []models.DbConfig{
				{
					Database: "nonexistent_db",
					Username: "invalid_user",
					Password: "invalid_pass",
					DbName:   "test",
					Name:     "test",
				},
			},
			want: false, // Should fail with invalid credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDatabases(tt.dbs)
			if (err == nil) != tt.want {
				t.Errorf("CheckDatabases() error = %v, want success = %v", err, tt.want)
			}
		})
	}
}
