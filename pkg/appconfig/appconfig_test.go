package appconfig

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

func resetLogging() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}

func TestSetLogger(t *testing.T) {
	resetLogging()
	tests := []struct {
		name           string
		verbose        bool
		expected_debug bool
	}{
		{"Verbose", true, false},
		{"Non-Verbose", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origStderr := os.Stderr

			r, w, _ := os.Pipe()
			os.Stderr = w

			if err := setLogger(tt.verbose); err != nil {
				t.Errorf("setLogger() error = %v", err)
			}
			slog.Debug("test message") // test writing to DEBUG level

			if err := w.Close(); err != nil {
				t.Errorf("setLogger() error = %v", err)
			}
			os.Stderr = origStderr

			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Errorf("setLogger() error = %v", err)
			}

			output := buf.String()

			if strings.Contains(output, "Debug") != tt.expected_debug {
				t.Errorf("setLogger() output mismatch: got %v, want %v", strings.Contains(output, "Debug"), tt.expected_debug)
			}
		})
	}
}

func TestParse(t *testing.T) {
	resetLogging()
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "Valid arguments",
			args:    []string{"--token", "test-token", "--config", "/tmp/config", "--org", "test-org", "--device", "test-device", "--timeout", "3600", "--verbose"},
			wantErr: false,
		},
		{
			name:    "Invalid timeout value",
			args:    []string{"--token", "test-token", "--timeout", "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AppConfig{}
			err := config.Parse(tt.args)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestWriteGredenturesConfig(t *testing.T) {
	resetLogging()
	tests := []struct {
		name       string
		config     *AppConfig
		wantErr    bool
		verifyFile func(t *testing.T, path string)
	}{
		{
			name: "Valid configuration",
			config: &AppConfig{
				Config:  "/tmp/test_gredentures.ini",
				Org:     "test-org",
				Device:  "test-device",
				Timeout: 3600,
			},
			wantErr: false,
			verifyFile: func(t *testing.T, path string) {
				cfg, err := ini.Load(path)
				if err != nil {
					t.Fatalf("Failed to load config file: %v", err)
				}

				section, err := cfg.GetSection("gredentures")
				if err != nil {
					t.Fatalf("Failed to get section: %v", err)
				}

				if section.Key("Org").String() != "test-org" {
					t.Errorf("Org key mismatch: got %v, want %v", section.Key("Org").String(), "test-org")
				}
				if section.Key("Device").String() != "test-device" {
					t.Errorf("Device key mismatch: got %v, want %v", section.Key("Device").String(), "test-device")
				}
				if section.Key("Timeout").String() != "3600" {
					t.Errorf("Timeout key mismatch: got %v, want %v", section.Key("Timeout").String(), "3600")
				}
			},
		},
		{
			name: "Missing config path",
			config: &AppConfig{
				Config:  "",
				Org:     "test-org",
				Device:  "test-device",
				Timeout: 3600,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.WriteGredenturesConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteGredenturesConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.verifyFile != nil {
				tt.verifyFile(t, tt.config.Config)
				_ = os.Remove(tt.config.Config) // Clean up test file
			}
		})
	}
}

func TestGetGredenturesConfig(t *testing.T) {
	resetLogging()
	tests := []struct {
		name       string
		config     *AppConfig
		setupFile  func(t *testing.T, path string)
		wantErr    bool
		verifyFile func(t *testing.T, path string)
	}{
		{
			name: "Config file exists",
			config: &AppConfig{
				Config: "/tmp/existing_gredentures.ini",
			},
			setupFile: func(t *testing.T, path string) {
				cfg := ini.Empty()
				section, _ := cfg.NewSection("gredentures")
				if _, err := section.NewKey("Org", "test-org"); err != nil {
					t.Fatalf("Failed to create Org key: %v", err)
				}
				if _, err := section.NewKey("Device", "test-device"); err != nil {
					t.Fatalf("Failed to create device key: %v", err)
				}
				if _, err := section.NewKey("Timeout", "3600"); err != nil {
					t.Fatalf("Failed to create timeout key: %v", err)
				}
				if err := cfg.SaveTo(path); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Config file does not exist",
			config: &AppConfig{
				Config:  "/tmp/nonexistent_gredentures.ini",
				Org:     "test-org",
				Device:  "test-device",
				Timeout: 3600,
			},
			setupFile: nil,
			wantErr:   false,
			verifyFile: func(t *testing.T, path string) {
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("Expected config file to be created, but it does not exist")
				}
			},
		},
		{
			name: "Error checking config file",
			config: &AppConfig{
				Config: "/invalid/path/gredentures.ini",
			},
			setupFile: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile != nil {
				tt.setupFile(t, tt.config.Config)
				defer func() {
					if err := os.Remove(tt.config.Config); err != nil {
						t.Fatalf("Error removing file %s: %v", tt.config.Config, err)
					}
				}()
			}

			err := tt.config.GetGredenturesConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGredenturesConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.verifyFile != nil {
				tt.verifyFile(t, tt.config.Config)
			}
		})
	}
}

func TestLoadGredenturesConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *AppConfig
		setupFile func(t *testing.T, path string)
		wantErr   bool
		verify    func(t *testing.T, config *AppConfig)
	}{
		{
			name: "Valid config file",
			config: &AppConfig{
				Config: "/tmp/valid_gredentures.ini",
			},
			setupFile: func(t *testing.T, path string) {
				cfg := ini.Empty()
				section, _ := cfg.NewSection("gredentures")
				if _, err := section.NewKey("Org", "test-org"); err != nil {
					t.Fatalf("Failed to create Org key: %v", err)
				}
				if _, err := section.NewKey("Device", "test-device"); err != nil {
					t.Fatalf("Failed to create device key: %v", err)
				}
				if _, err := section.NewKey("Timeout", "3600"); err != nil {
					t.Fatalf("Failed to create device key: %v", err)
				}
				if err := cfg.SaveTo(path); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
			},
			wantErr: false,
			verify: func(t *testing.T, config *AppConfig) {
				if config.Org != "test-org" {
					t.Errorf("Org mismatch: got %v, want %v", config.Org, "test-org")
				}
				if config.Device != "test-device" {
					t.Errorf("Device mismatch: got %v, want %v", config.Device, "test-device")
				}
				if config.Timeout != 3600 {
					t.Errorf("Timeout mismatch: got %v, want %v", config.Timeout, 3600)
				}
			},
		},
		{
			name: "Invalid Timeout value",
			config: &AppConfig{
				Config: "/tmp/invalid_timeout.ini",
			},
			setupFile: func(t *testing.T, path string) {
				cfg := ini.Empty()
				section, _ := cfg.NewSection("gredentures")
				if _, err := section.NewKey("Timeout", "invalid"); err != nil {
					t.Fatalf("Failed to create Timeout key: %v", err)
				}
				if err := cfg.SaveTo(path); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
			},
			wantErr: true,
		},
		{
			name: "Missing config file",
			config: &AppConfig{
				Config: "/tmp/missing_gredentures.ini",
			},
			setupFile: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFile != nil {
				tt.setupFile(t, tt.config.Config)
				// cleanup test file
				defer func() {
					if err := os.Remove(tt.config.Config); err != nil {
						t.Fatalf("Error removing file %s: %v", tt.config.Config, err)
					}
				}()
			}

			err := tt.config.LoadGredenturesConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadGredenturesConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && tt.verify != nil {
				tt.verify(t, tt.config)
			}
		})
	}
}

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		name    string
		config  *AppConfig
		setup   func(t *testing.T, config *AppConfig)
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &AppConfig{
				Token:  "test-token",
				Org:    "test-org",
				Device: "test-device",
			},
			setup:   nil,
			wantErr: false,
		},
		{
			name: "Missing token",
			config: &AppConfig{
				Org:    "test-org",
				Device: "test-device",
			},
			setup:   nil,
			wantErr: true,
		},
		{
			name: "Missing org and device",
			config: &AppConfig{
				Config: "/tmp/test_validate_config_empty.ini",
				Token:  "test-token",
			},
			setup:   nil,
			wantErr: true,
		},
		{
			name: "Load config from file",
			config: &AppConfig{
				Config: "/tmp/test_validate_config.ini",
				Token:  "test-token",
			},
			setup: func(t *testing.T, config *AppConfig) {
				cfg := ini.Empty()
				section, _ := cfg.NewSection("gredentures")
				if _, err := section.NewKey("Org", "test-org"); err != nil {
					t.Fatalf("Failed to create Org key: %v", err)
				}
				if _, err := section.NewKey("Device", "test-device"); err != nil {
					t.Fatalf("Failed to create device key: %v", err)
				}
				if err := cfg.SaveTo(config.Config); err != nil {
					t.Fatalf("Failed to create test config file: %v", err)
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, tt.config)
				defer func() {
					if err := os.Remove(tt.config.Config); err != nil {
						t.Fatalf("Error removing file %s: %v", tt.config.Config, err)
					}
				}()
			}

			err := tt.config.ValidateOptions()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
