package gredentures

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/docopt/docopt-go"
	"gopkg.in/ini.v1"
)

const Usage = `Usage:
  gredentures [-t <token>] [-c <file>] [-o <org>] [-d <device>]
  gredentures [-t <token>] [-o <org>] [-d <device>]
  gredentures [-t <token>] [-d <device>]
  gredentures [-t <token>]

Options:
  -h, --help                        Show this screen.
  -t <token>, --token <token>       MFA token.
  -c <file>, --config <file>        Config file.
  -o <org>, --org <org>             Organization.
  -d <device>, --device <device>    MFA device arn.
  --verbose			    Show verbose output [default: false].`

type appConfig struct {
	Token   string `docopt:"--token"`
	Config  string `docopt:"--config"`
	Org     string `docopt:"--org"`
	Device  string `docopt:"--device"`
	Verbose bool   `docopt:"--verbose"`
}

type awsConfig struct {
	defaultCreds aws.Credentials
	sessionCreds *sts.GetSessionTokenOutput
}

func (conf *awsConfig) createUpdatedConfig() error {
	inidata := ini.Empty()

	// Helper function to create a section and add keys
	addKeysToSection := func(sectionName string, keys map[string]string) error {
		slog.Debug("Creating section", "section", sectionName)
		sec, err := inidata.NewSection(sectionName)
		if err != nil {
			return fmt.Errorf("failed to create section '%s': %w", sectionName, err)
		}
		for key, value := range keys {
			slog.Debug("Creating key", "key", key, "value", value)
			if _, err := sec.NewKey(key, value); err != nil {
				return fmt.Errorf("failed to create key '%s' in section '%s': %w", key, sectionName, err)
			}
		}
		return nil
	}

	// Add keys to the "default" section
	defaultKeys := map[string]string{
		"aws_access_key_id":     conf.defaultCreds.AccessKeyID,
		"aws_secret_access_key": conf.defaultCreds.SecretAccessKey,
	}
	if err := addKeysToSection("default", defaultKeys); err != nil {
		return err
	}

	// Add keys to the "default-mfa" section
	defaultMfaKeys := map[string]string{
		"aws_session_token":     *conf.sessionCreds.Credentials.SessionToken,
		"aws_access_key_id":     *conf.sessionCreds.Credentials.AccessKeyId,
		"aws_secret_access_key": *conf.sessionCreds.Credentials.SecretAccessKey,
	}
	if err := addKeysToSection("default-mfa", defaultMfaKeys); err != nil {
		return err
	}

	// Save the new ~/.aws/credentials file
	credentialsPath := fmt.Sprintf("%s/.aws/credentials", os.Getenv("HOME"))
	slog.Debug("Saving credentials file", "path", credentialsPath)
	if err := inidata.SaveTo(credentialsPath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (conf *awsConfig) getSessionCreds(appconfig appConfig) error {
	config, err := getDefaultAccount()
	if err != nil {
		return fmt.Errorf("failed to get default account: %w", err)
	}

	client := sts.NewFromConfig(config)

	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(3600),
		SerialNumber:    aws.String(appconfig.Device),
		TokenCode:       aws.String(appconfig.Token),
	}

	slog.Debug("Getting session token", "serial_number", appconfig.Device, "token_code", appconfig.Token)
	creds, err := client.GetSessionToken(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to get session token: %w", err)
	}

	conf.sessionCreds = creds

	return nil
}

func (conf *awsConfig) getDefaultCreds() error {
	config, err := getDefaultAccount()
	if err != nil {
		return fmt.Errorf("failed to get default account: %w", err)
	}

	slog.Debug("Getting default credentials")
	creds, err := config.Credentials.Retrieve(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to retrieve default credentials: %w", err)
	}

	conf.defaultCreds = creds

	return nil
}

func (conf *appConfig) writeGredenturesConfig() error {
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

	slog.Debug("Writing gredentures config file", "path", conf.Config)
	if err := inidata.SaveTo(conf.Config); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (conf *appConfig) getGredenturesConfig() error {
	if conf.Config == "" {
		conf.Config = fmt.Sprintf("%s/.gredentures", os.Getenv("HOME"))
	}

	// Check if the gredentures config file exists
	slog.Debug("Checking for gredentures config file", "path", conf.Config)
	if _, err := os.Stat(conf.Config); err == nil {
		return conf.loadGredenturesConfig()
	} else if os.IsNotExist(err) {
		slog.Debug("Gredentures config file does not exist", "path", conf.Config)
		// Create a new gredentures config if it doesn't exist
		return conf.writeGredenturesConfig()
	} else {
		return fmt.Errorf("error checking config file: %w", err)
	}
}

func (conf *appConfig) loadGredenturesConfig() error {
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
	if conf.Org == "" {
		slog.Debug("Setting Org from config")
		conf.Org = section.Key("Org").String()
	}
	if conf.Device == "" {
		slog.Debug("Setting Device from config")
		conf.Device = section.Key("Device").String()
	}

	return nil
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

func (config *appConfig) validateOptions() error {
	// get any values pulled from the gredentures config file
	// and use them to fill any unsupplied values from the command line
	slog.Debug("Validating options")
	if err := config.getGredenturesConfig(); err != nil {
		return fmt.Errorf("error getting gredentures config: %w", err)
	}

	// check that all required values are set
	slog.Debug("Checking for token")
	if config.Token == "" {
		return fmt.Errorf("token must be supplied for MFA")
	}

	slog.Debug("Checking for org and device")
	if config.Org == "" || config.Device == "" {
		return fmt.Errorf("the Token must be set with a commandline arg. Org, and Device must be set in a config file or as commandline options")
	}

	return nil
}

func getDefaultAccount() (aws.Config, error) {
	slog.Debug("Loading default AWS config")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"),
		config.WithSharedConfigProfile("default"))
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return cfg, nil
}

func (config *appConfig) Parse(args []string) error {
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

func CLI(args []string) int {
	var appconfig appConfig
	var awsconfig awsConfig

	// PARSE COMMAND LINE ARGS
	slog.Debug("Parsing command line arguments")
	if err := appconfig.Parse(args); err != nil {
		fmt.Printf("Error parsing command line arguments: %v\n", err)
	}

	// VALIDATE GREDENTURES CONFIG AND OPTIONS
	slog.Info("Validating gredentures options and config...")
	if err := appconfig.validateOptions(); err != nil {
		fmt.Printf("Error validating options: %v\n", err)
		return 1
	}

	// LOAD DEFAULT CREDS
	slog.Info("Getting default aws credentials...")
	if err := awsconfig.getDefaultCreds(); err != nil {
		fmt.Printf("Error getting default credentials: %v\n", err)
		return 1
	}

	// ACQUIRE SESSION CREDS
	slog.Info("Getting aws session credentials...")
	if err := awsconfig.getSessionCreds(appconfig); err != nil {
		fmt.Printf("Error getting session credentials: %v\n", err)
		return 1
	}

	// REWRITE ~/.aws/credentials FILE
	slog.Info("Writing updated aws credentials file...")
	if err := awsconfig.createUpdatedConfig(); err != nil {
		fmt.Printf("Error creating updated config: %v\n", err)
		return 1
	}

	return 0
}
