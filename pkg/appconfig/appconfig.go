package appconfig

import (
	"fmt"
	"log/slog"
	"os"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/parsers/yaml"
	y "gopkg.in/yaml.v3" // Alias this import to avoid conflicts

	"github.com/docopt/docopt-go"
)

const Usage = `Usage:
  gredentures -t <token> [-c <config>] [-o <org>] [-d <device>] [--timeout <seconds>] [--verbose]
  gredentures --token <token> [--config <config>] [--org <org>] [--device <device>] [--timeout <seconds>] [--verbose]
  gredentures --help

Options:
  -t <token>, --token <token>       MFA token (required)
  -c <config>, --config <config>    Path to gredentures config file [default: $HOME/.gredentures]
  -o <org>, --org <org>             Organization (optional if set in config)
  -d <device>, --device <device>    MFA device ARN (optional if set in config)
  --timeout <seconds>               Token timeout in seconds [default: 86400]
  --verbose                         Enable verbose output
  --help                            Show this help message`

type AppConfig struct {
	Token   string `docopt:"--token"`
	Config  string `docopt:"--config"`
	Org     string `docopt:"--org"`
	Device  string `docopt:"--device"`
	Verbose bool   `docopt:"--verbose"`
	Timeout int32  `docopt:"--timeout"`
}

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

func (config *AppConfig) Parse(args []string) error {
	opts, err := docopt.ParseArgs(Usage, args, "Gredentures 0.1")
	if err != nil {
		return fmt.Errorf("error parsing options: %v", err)
	}

	// BIND COMMAND LINE ARGS TO APP CONFIG
	if err := opts.Bind(&config); err != nil {
		return fmt.Errorf("error binding options: %v", err)
	}

	// SETUP LOGGING
	if err := setLogger(config.Verbose); err != nil {
		fmt.Printf("Error setting logger: %v\n", err)
	}

	return nil
}

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
	if err := os.WriteFile(conf.Config, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}

	return nil
}

func (conf *AppConfig) GetGredenturesConfig() error {
	if conf.Config == "" {
		conf.Config = fmt.Sprintf("%s/.gredentures", os.Getenv("HOME"))
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

func (config *AppConfig) ValidateOptions() error {
	slog.Debug("Validating options")
	if err := config.GetGredenturesConfig(); err != nil {
		return fmt.Errorf("error getting gredentures config: %w", err)
	}

	// confirm required values have been found
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
