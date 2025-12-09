package database

import (
	"os"
	"testing"
)

func TestConnect(t *testing.T) {
	// Save original environment
	originalHostname := os.Getenv("MYSQL_HOSTNAME")

	// Clean up after test
	defer func() {
		if originalHostname != "" {
			os.Setenv("MYSQL_HOSTNAME", originalHostname)
		} else {
			os.Unsetenv("MYSQL_HOSTNAME")
		}
	}()

	tests := []struct {
		name     string
		dbName   string
		dbUser   string
		dbPass   string
		hostname string
		wantErr  bool
	}{
		{
			name:     "missing hostname",
			dbName:   "testdb",
			dbUser:   "testuser",
			dbPass:   "testpass",
			hostname: "",
			wantErr:  true,
		},
		{
			name:     "invalid credentials",
			dbName:   "nonexistent_db",
			dbUser:   "invalid_user",
			dbPass:   "invalid_pass",
			hostname: "localhost",
			wantErr:  true,
		},
		{
			name:     "empty database name",
			dbName:   "",
			dbUser:   "testuser",
			dbPass:   "testpass",
			hostname: "localhost",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hostname != "" {
				os.Setenv("MYSQL_HOSTNAME", tt.hostname)
			} else {
				os.Unsetenv("MYSQL_HOSTNAME")
			}

			db, err := Connect(tt.dbName, tt.dbUser, tt.dbPass)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
			if db != nil {
				db.Close()
			}
		})
	}
}
