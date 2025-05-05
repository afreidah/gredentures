// Package awsconfig provides functionality for managing AWS configurations and credentials.
// It includes methods for retrieving default AWS credentials, creating updated configurations,
// and managing session tokens for MFA authentication.
package awsconfig

import (
	"context"
	"fmt"
	"gredentures/pkg/appconfig"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"gopkg.in/ini.v1"
)

// AwsConfig represents the AWS configuration and credentials.
// It includes default credentials and session credentials for MFA authentication.
type AwsConfig struct {
	defaultCreds aws.Credentials            // Default AWS credentials.
	sessionCreds *sts.GetSessionTokenOutput // Session credentials for MFA authentication.
}

// GetDefaultAccount loads the default AWS configuration using the "default" profile.
// It returns the AWS configuration or an error if the configuration cannot be loaded.
func GetDefaultAccount() (aws.Config, error) {
	slog.Debug("Loading default AWS config")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"),
		config.WithSharedConfigProfile("default"))
	if err != nil {
		return aws.Config{}, fmt.Errorf("unable to load SDK config, %v", err)
	}

	return cfg, nil
}

// CreateUpdatedConfig creates an updated AWS credentials file with default and session credentials.
// It writes the credentials to the ~/.aws/credentials file and returns an error if the operation fails.
func (conf *AwsConfig) CreateUpdatedConfig() error {
	inidata := ini.Empty()

	// Helper function to create a section and add keys.
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

	// Add keys to the "default" section.
	defaultKeys := map[string]string{
		"aws_access_key_id":     conf.defaultCreds.AccessKeyID,
		"aws_secret_access_key": conf.defaultCreds.SecretAccessKey,
	}
	if err := addKeysToSection("default", defaultKeys); err != nil {
		return err
	}

	// Add keys to the "default-mfa" section.
	defaultMfaKeys := map[string]string{
		"aws_session_token":     *conf.sessionCreds.Credentials.SessionToken,
		"aws_access_key_id":     *conf.sessionCreds.Credentials.AccessKeyId,
		"aws_secret_access_key": *conf.sessionCreds.Credentials.SecretAccessKey,
	}
	if err := addKeysToSection("default-mfa", defaultMfaKeys); err != nil {
		return err
	}

	// Save the new ~/.aws/credentials file.
	credentialsPath := fmt.Sprintf("%s/.aws/credentials", os.Getenv("HOME"))
	slog.Debug("Saving credentials file", "path", credentialsPath)
	if err := inidata.SaveTo(credentialsPath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// GetSessionCreds retrieves session credentials using MFA authentication.
// It uses the provided AppConfig to generate a session token and stores the credentials in AwsConfig.
func (conf *AwsConfig) GetSessionCreds(appconfig appconfig.AppConfig) error {
	config, err := GetDefaultAccount()
	if err != nil {
		return fmt.Errorf("failed to get default account: %w", err)
	}

	client := sts.NewFromConfig(config)

	slog.Debug("Getting session token", "device", appconfig.Device, "org", appconfig.Org)
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(appconfig.Timeout),
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

// GetDefaultCreds retrieves the default AWS credentials and stores them in AwsConfig.
// It uses the default AWS configuration to retrieve the credentials.
func (conf *AwsConfig) GetDefaultCreds() error {
	config, err := GetDefaultAccount()
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
