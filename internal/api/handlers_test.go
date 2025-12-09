package api

import (
	"os"
	"testing"
)

func TestGetPathParams(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedParts int
		wantErr       bool
		wantParts     []string
	}{
		{
			name:          "valid path with 2 parts",
			path:          "/search/se",
			expectedParts: 2,
			wantErr:       false,
			wantParts:     []string{"search", "se"},
		},
		{
			name:          "valid path with 3 parts",
			path:          "/search/se/example",
			expectedParts: 3,
			wantErr:       false,
			wantParts:     []string{"search", "se", "example"},
		},
		{
			name:          "path with trailing slash",
			path:          "/search/se/",
			expectedParts: 2,
			wantErr:       false,
			wantParts:     []string{"search", "se"}, // Trailing slash is trimmed
		},
		{
			name:          "path with too many parts",
			path:          "/search/se/extra/part",
			expectedParts: 2,
			wantErr:       true,
			wantParts:     nil,
		},
		{
			name:          "path with too few parts",
			path:          "/search",
			expectedParts: 3,
			wantErr:       true,
			wantParts:     nil,
		},
		{
			name:          "empty path",
			path:          "/",
			expectedParts: 1,
			wantErr:       false,
			wantParts:     []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := getPathParams(tt.path, tt.expectedParts)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPathParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(parts) != len(tt.wantParts) {
					t.Errorf("getPathParams() returned %d parts, want %d", len(parts), len(tt.wantParts))
					return
				}
				for i := range parts {
					if parts[i] != tt.wantParts[i] {
						t.Errorf("getPathParams() parts[%d] = %v, want %v", i, parts[i], tt.wantParts[i])
					}
				}
			}
		})
	}
}

func TestGetTLDEnvVars(t *testing.T) {
	// Save original environment
	originalSeDb := os.Getenv("MYSQL_SEDUMP_DATABASE")
	originalSeUser := os.Getenv("MYSQL_SEDUMP_USERNAME")
	originalSePass := os.Getenv("MYSQL_SEDUMP_PASSWORD")

	// Clean up after test
	defer func() {
		if originalSeDb != "" {
			os.Setenv("MYSQL_SEDUMP_DATABASE", originalSeDb)
		} else {
			os.Unsetenv("MYSQL_SEDUMP_DATABASE")
		}
		if originalSeUser != "" {
			os.Setenv("MYSQL_SEDUMP_USERNAME", originalSeUser)
		} else {
			os.Unsetenv("MYSQL_SEDUMP_USERNAME")
		}
		if originalSePass != "" {
			os.Setenv("MYSQL_SEDUMP_PASSWORD", originalSePass)
		} else {
			os.Unsetenv("MYSQL_SEDUMP_PASSWORD")
		}
	}()

	tests := []struct {
		name     string
		tld      string
		setEnv   bool
		wantErr  bool
		wantDb   string
		wantUser string
		wantPass string
	}{
		{
			name:     "valid TLD se",
			tld:      "se",
			setEnv:   true,
			wantErr:  false,
			wantDb:   "test_se_db",
			wantUser: "test_se_user",
			wantPass: "test_se_pass",
		},
		{
			name:     "valid TLD nu",
			tld:      "nu",
			setEnv:   true,
			wantErr:  false,
			wantDb:   "test_nu_db",
			wantUser: "test_nu_user",
			wantPass: "test_nu_pass",
		},
		{
			name:     "invalid TLD",
			tld:      "invalid",
			setEnv:   false,
			wantErr:  true,
			wantDb:   "",
			wantUser: "",
			wantPass: "",
		},
		{
			name:     "se_diff TLD",
			tld:      "se_diff",
			setEnv:   true,
			wantErr:  false,
			wantDb:   "test_se_diff_db",
			wantUser: "test_se_diff_user",
			wantPass: "test_se_diff_pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				// Set environment variables based on TLD config
				config, ok := tldConfigs[tt.tld]
				if ok {
					os.Setenv(config.Database, tt.wantDb)
					os.Setenv(config.Username, tt.wantUser)
					os.Setenv(config.Password, tt.wantPass)
				}
			}

			db, user, pass, err := getTLDEnvVars(tt.tld)
			if (err != nil) != tt.wantErr {
				t.Errorf("getTLDEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if db != tt.wantDb {
					t.Errorf("getTLDEnvVars() db = %v, want %v", db, tt.wantDb)
				}
				if user != tt.wantUser {
					t.Errorf("getTLDEnvVars() user = %v, want %v", user, tt.wantUser)
				}
				if pass != tt.wantPass {
					t.Errorf("getTLDEnvVars() pass = %v, want %v", pass, tt.wantPass)
				}
			}

			// Clean up environment variables after each test
			if tt.setEnv {
				config, ok := tldConfigs[tt.tld]
				if ok {
					os.Unsetenv(config.Database)
					os.Unsetenv(config.Username)
					os.Unsetenv(config.Password)
				}
			}
		})
	}
}
