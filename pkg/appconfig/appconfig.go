// Package appconfig provides functionality for parsing, validating, and managing
// application configuration for the Gredentures CLI tool. It supports reading
// configuration from command-line arguments, environment variables, and YAML files.
package appconfig

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	y "gopkg.in/yaml.v3" // Alias this import to avoid conflicts

	"github.com/docopt/docopt-go"
)

// Usage defines the command-line usage instructions for the Gredentures CLI tool.
const Usage = `Usage:
  gredentures -t <token> [-c <config>] [-o <org>] [-d <device>] [-p <profile>] [--timeout <seconds>] [--verbose]
  gredentures --token <token> [--config <config>] [--org <org>] [--device <device>] [--profile <profile>] [--timeout <seconds>] [--verbose]
  gredentures --help

Options:
  -t <token>, --token <token>       MFA token (required)
  -c <config>, --config <config>    Path to gredentures config file [default: $HOME/.gredentures.yml]
  -o <org>, --org <org>             Organization (optional if set in config)
  -d <device>, --device <device>    MFA device ARN (optional if set in config)
  -p <profile>, --profile <profile> Name to use for the session creds profile [default: default-mfa]
  --timeout <seconds>               Token timeout in seconds [default: 86400]
  --verbose                         Enable verbose output
  --help                            Show this help message`

// AppConfig represents the configuration options for the Gredentures CLI tool.
// It includes fields for command-line arguments and configuration file values.
type AppConfig struct {
	Token   string `docopt:"--token"`   // MFA token (required).
	Config  string `docopt:"--config"`  // Path to the configuration file.
	Org     string `docopt:"--org"`     // Organization name.
	Device  string `docopt:"--device"`  // MFA device ARN.
	Verbose bool   `docopt:"--verbose"` // Enable verbose output.
	Timeout int32  `docopt:"--timeout"` // Token timeout in seconds.
	Profile string `docopt:"--profile"` // Profile name for session credentials.
}

// setLogger configures the logging level for the application based on the verbose flag.
// If verbose is true, debug-level logging is enabled; otherwise, info-level logging is used.
func setLogger(verbose bool) error {
	level := slog.LevelInfo

	if verbose {
		level = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	return nil
}

// Parse parses the command-line arguments and populates the AppConfig struct.
// It also sets default values for fields like Profile and configures logging.
func (config *AppConfig) Parse(args []string) error {
	opts, err := docopt.ParseArgs(Usage, args, "Gredentures 0.1")
	if err != nil {
		return fmt.Errorf("error parsing options: %v", err)
	}

	// Bind command-line arguments to AppConfig
	if err := opts.Bind(&config); err != nil {
		return fmt.Errorf("error binding options: %v", err)
	}

	// Set default value for Profile if not provided
	if config.Profile == "" {
		config.Profile = "default-mfa"
	}

	// Setup logging
	if err := setLogger(config.Verbose); err != nil {
		fmt.Printf("Error setting logger: %v\n", err)
	}

	return nil
}

// WriteGredenturesConfig writes the current AppConfig values to a YAML configuration file.
// If the file does not exist, it creates a new one.
func (conf *AppConfig) WriteGredenturesConfig() error {
	k := koanf.New(".") // Initialize koanf with a delimiter

	// Load the current AppConfig values into koanf
	configMap := map[string]interface{}{
		"gredentures.Org":     conf.Org,
		"gredentures.Device":  conf.Device,
		"gredentures.Timeout": conf.Timeout,
	}
	if err := k.Load(confmap.Provider(configMap, "."), nil); err != nil {
		return fmt.Errorf("failed to load AppConfig values into koanf: %w", err)
	}

	// Marshal the configuration into YAML
	yamlData, err := y.Marshal(k.All())
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to YAML: %w", err)
	}

	// Write the YAML data to the specified file
	if err := os.WriteFile(conf.Config, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}

	return nil
}

// GetGredenturesConfig ensures that the configuration file exists and loads its values
// into the AppConfig struct. If the file does not exist, it creates a new one.
func (conf *AppConfig) GetGredenturesConfig() error {
	if conf.Config == "" {
		conf.Config = fmt.Sprintf("%s/.gredentures.yml", os.Getenv("HOME"))
	}

	// Check if the gredentures config file exists
	slog.Debug("Checking for gredentures config file", "path", conf.Config)
	if _, err := os.Stat(conf.Config); err == nil {
		return conf.LoadGredenturesConfig()
	} else if os.IsNotExist(err) {
		slog.Debug("Gredentures config file does not exist", "path", conf.Config)
		// Create a new gredentures config if it doesn't exist
		return conf.WriteGredenturesConfig()
	} else {
		return fmt.Errorf("error checking config file: %w", err)
	}
}

// LoadGredenturesConfig loads the configuration values from the YAML file into the AppConfig struct.
// It updates fields only if they are not already set.
func (conf *AppConfig) LoadGredenturesConfig() error {
	k := koanf.New(".") // Initialize koanf with a delimiter

	// Load the existing AppConfig values into koanf
	existingConfig := map[string]interface{}{
		"gredentures.Org":     conf.Org,
		"gredentures.Device":  conf.Device,
		"gredentures.Timeout": conf.Timeout,
	}
	if err := k.Load(confmap.Provider(existingConfig, "."), nil); err != nil {
		return fmt.Errorf("failed to load existing AppConfig values into koanf: %w", err)
	}

	// Load the YAML file into koanf
	if err := k.Load(file.Provider(conf.Config), yaml.Parser()); err != nil {
		return fmt.Errorf("failed to load YAML file into koanf: %w", err)
	}

	// Update AppConfig fields only if they are not already set
	if conf.Org == "" {
		conf.Org = k.String("gredentures.Org")
	}
	if conf.Device == "" {
		conf.Device = k.String("gredentures.Device")
	}
	if conf.Timeout == 0 {
		conf.Timeout = int32(k.Int("gredentures.Timeout"))
	}

	return nil
}

// ValidateOptions validates the AppConfig fields to ensure all required options are set.
// It checks for the presence of a token, organization, and device, and returns an error if any are missing.
func (config *AppConfig) ValidateOptions() error {
	slog.Debug("Validating options")
	if err := config.GetGredenturesConfig(); err != nil {
		return fmt.Errorf("error getting gredentures config: %w", err)
	}

	// Confirm required values have been found
	switch {
	case config.Token == "":
		slog.Debug("Checking for token")
		return fmt.Errorf("token must be supplied for MFA")
	case config.Org == "" || config.Device == "":
		slog.Debug("Checking for org and device")
		return fmt.Errorf("the Token must be set with a commandline arg. Org, and Device must be set in a config file or as commandline options")
	}

	return nil
}
