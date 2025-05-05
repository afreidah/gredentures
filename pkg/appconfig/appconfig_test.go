package appconfig

import (
	"bytes"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/stretchr/testify/assert"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
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
		name            string
		args            []string
		wantErr         bool
		expectedProfile string
	}{
		{
			name:            "Valid arguments with default profile",
			args:            []string{"--token", "test-token", "--config", "/tmp/config", "--org", "test-org", "--device", "test-device", "--timeout", "3600", "--verbose"},
			wantErr:         false,
			expectedProfile: "default-mfa",
		},
		{
			name:            "Valid arguments with custom profile",
			args:            []string{"--token", "test-token", "--profile", "custom-profile", "--config", "/tmp/config", "--org", "test-org", "--device", "test-device", "--timeout", "3600", "--verbose"},
			wantErr:         false,
			expectedProfile: "custom-profile",
		},
		{
			name:            "Valid arguments with defaults omitted",
			args:            []string{"--token", "test-token", "--org", "test-org", "--device", "test-device"},
			wantErr:         false,
			expectedProfile: "default-mfa",
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
			} else {
				if tt.wantErr {
					t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
				}
			}
			if config.Profile != tt.expectedProfile {
				t.Errorf("Parse() Profile = %v, expected %v", config.Profile, tt.expectedProfile)
			}
		})
	}
}

func TestParseDefaults(t *testing.T) {
	resetLogging()

	tests := []struct {
		name            string
		args            []string
		wantErr         bool
		expectedProfile string
		expectedTimeout int32
	}{
		{
			name:            "Valid arguments with defaults",
			args:            []string{"--token", "test-token", "--org", "test-org", "--device", "test-device"},
			wantErr:         false,
			expectedProfile: "default-mfa",
			expectedTimeout: 86400,
		},
		{
			name:            "Valid arguments with overrides",
			args:            []string{"--token", "test-token", "--org", "test-org", "--device", "test-device", "--timeout", "6000", "-p", "custom-profile"},
			wantErr:         false,
			expectedProfile: "custom-profile",
			expectedTimeout: int32(6000),
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
			} else {
				if tt.wantErr {
					t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
				}
			}
			if config.Profile != tt.expectedProfile {
				t.Errorf("Parse() Profile = %v, expected %v", config.Profile, tt.expectedProfile)
			}
			if config.Timeout != tt.expectedTimeout {
				t.Errorf("Parse() Timeout = %v, expected %v", config.Timeout, tt.expectedTimeout)
			}
		})
	}
}

func TestWriteGredenturesConfig(t *testing.T) {
	t.Run("Write all values to YAML file", func(t *testing.T) {
		// Create a temporary file for the YAML config
		tempFile, err := os.CreateTemp("", "gredentures_config_*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())

		// Initialize AppConfig with values
		conf := &AppConfig{
			Config:  tempFile.Name(),
			Org:     "test-org",
			Device:  "test-device",
			Timeout: 3600,
		}

		// Call WriteGredenturesConfig
		err = conf.WriteGredenturesConfig()
		assert.NoError(t, err)

		// Load the written YAML file to verify its contents
		k := koanf.New(".")
		err = k.Load(file.Provider(tempFile.Name()), yaml.Parser())
		assert.NoError(t, err)

		// Verify the values in the YAML file
		assert.Equal(t, "test-org", k.String("gredentures.Org"))
		assert.Equal(t, "test-device", k.String("gredentures.Device"))
		assert.Equal(t, "3600", k.String("gredentures.Timeout"))
	})

	t.Run("Write empty values to YAML file", func(t *testing.T) {
		// Create a temporary file for the YAML config
		tempFile, err := os.CreateTemp("", "gredentures_config_*.yaml")
		assert.NoError(t, err)
		defer os.Remove(tempFile.Name())

		// Initialize AppConfig with empty values
		conf := &AppConfig{
			Config: tempFile.Name(),
		}

		// Call WriteGredenturesConfig
		err = conf.WriteGredenturesConfig()
		assert.NoError(t, err)

		// Load the written YAML file to verify its contents
		k := koanf.New(".")
		err = k.Load(file.Provider(tempFile.Name()), yaml.Parser())
		assert.NoError(t, err)

		// Verify the values in the YAML file
		assert.Equal(t, "", k.String("gredentures.Org"))
		assert.Equal(t, "", k.String("gredentures.Device"))
		assert.Equal(t, "0", k.String("gredentures.Timeout"))
	})
}

func TestLoadGredenturesConfig(t *testing.T) {
	// Create a temporary YAML config file
	tempFile, err := os.CreateTemp("", "gredentures_config_*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write test data to the YAML file
	_, err = tempFile.WriteString(`
gredentures:
  Org: file-org
  Device: file-device
  Timeout: 300
`)
	assert.NoError(t, err)
	assert.NoError(t, tempFile.Close())

	t.Run("Command-line values take precedence", func(t *testing.T) {
		// Initialize AppConfig with command-line values
		conf := &AppConfig{
			Config:  tempFile.Name(),
			Org:     "cmdline-org",
			Device:  "cmdline-device",
			Timeout: 600,
		}

		// Call LoadGredenturesConfig
		err = conf.LoadGredenturesConfig()
		assert.NoError(t, err)

		// Verify that the merged values prioritize command-line values
		assert.Equal(t, "cmdline-org", conf.Org)       // Command-line value takes precedence
		assert.Equal(t, "cmdline-device", conf.Device) // Command-line value takes precedence
		assert.Equal(t, int32(600), conf.Timeout)      // Command-line value takes precedence
	})

	t.Run("Config file values are used when command-line values are missing", func(t *testing.T) {
		// Initialize AppConfig without command-line values
		conf := &AppConfig{
			Config: tempFile.Name(),
		}

		// Call LoadGredenturesConfig
		err = conf.LoadGredenturesConfig()
		assert.NoError(t, err)

		// Verify that the values from the config file are used
		assert.Equal(t, "file-org", conf.Org)       // Config file value is used
		assert.Equal(t, "file-device", conf.Device) // Config file value is used
		assert.Equal(t, int32(300), conf.Timeout)   // Config file value is used
	})
}
