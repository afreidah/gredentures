package appconfig

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/docopt/docopt-go"
	"gopkg.in/ini.v1"
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
	inidata := ini.Empty()

	slog.Debug("Creating gredentures config file", "path", conf.Config)
	sec, err := inidata.NewSection("gredentures")
	if err != nil {
		return fmt.Errorf("failed to create section 'gredentures': %w", err)
	}
	slog.Debug("Creating section", "section", "gredentures")
	if _, err := sec.NewKey("Org", conf.Org); err != nil {
		return fmt.Errorf("failed to create key 'Org' in section 'gredentures': %w", err)
	}

	slog.Debug("Creating key", "key", "Org", "value", conf.Org)
	if _, err := sec.NewKey("Device", conf.Device); err != nil {
		return fmt.Errorf("failed to create key 'Device' in section 'gredentures': %w", err)
	}

	slog.Debug("Creating key", "key", "Timeout", "value", conf.Timeout)
	if _, err := sec.NewKey("Timeout", strconv.Itoa(int(conf.Timeout))); err != nil {
		return fmt.Errorf("failed to create key 'Timeout' in section 'gredentures': %w", err)
	}

	slog.Debug("Writing gredentures config file", "path", conf.Config)
	if err := inidata.SaveTo(conf.Config); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
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
	slog.Debug("Loading gredentures config file", "path", conf.Config)
	cfg, err := ini.Load(conf.Config)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	slog.Debug("Getting gredentures section")
	section, err := cfg.GetSection("gredentures")
	if err != nil {
		return fmt.Errorf("failed to get gredentures section: %w", err)
	}

	// Populate configuration fields if not already set
	slog.Debug("Populating configuration fields")
	if conf.Org == "" && section.HasKey("Org") {
		slog.Debug("Setting Org from config")
		conf.Org = section.Key("Org").String()
	}
	if conf.Device == "" && section.HasKey("Device") {
		slog.Debug("Setting Device from config")
		conf.Device = section.Key("Device").String()
	}
	if section.HasKey("Timeout") {
		slog.Debug("Setting Timeout from config")
		timeoutStr := section.Key("Timeout").String()

		timeoutInt, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return fmt.Errorf("failed to convert Timeout to int: %v", err)
		}

		conf.Timeout = int32(timeoutInt)
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
