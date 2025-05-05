// Package main is the entry point for the Gredentures CLI tool.
// It orchestrates the parsing of command-line arguments, validation of configurations,
// and management of AWS credentials for MFA authentication.
package main

import (
	"fmt"
	"log/slog"
	"os"

	appc "gredentures/pkg/appconfig"
	appa "gredentures/pkg/awsconfig"
)

var version = "dev" // Overwritten during build

// EnvVarMessageTemplate provides instructions for setting the AWS_PROFILE environment variable
// to use the session credentials by default.
const EnvVarMessageTemplate = `
***********************************************************************************
* To use your session creds by default please add the following to your profile:  *
*                                                                                 *
* export AWS_PROFILE=%s                                                           *
* then source your profile                                                        *
***********************************************************************************
`

// main is the entry point for the Gredentures CLI tool.
// It handles the parsing of command-line arguments, validation of configurations,
// and management of AWS credentials for MFA authentication.
func main() {
	fmt.Printf("Gredentures CLI version: %s\n", version)

	var g_app appc.AppConfig
	var g_aws appa.AwsConfig

	// Parse command-line arguments.
	if err := g_app.Parse(os.Args[1:]); err != nil {
		fmt.Printf("Error parsing command line arguments: %v\n", err)
	}

	// Validate Gredentures configuration and options.
	slog.Info("Validating gredentures options and config...")
	if err := g_app.ValidateOptions(); err != nil {
		fmt.Printf("Error validating options: %v\n", err)
	}

	// Load default AWS credentials.
	slog.Info("Getting default aws credentials...")
	if err := g_aws.GetDefaultCreds(); err != nil {
		fmt.Printf("Error getting default credentials: %v\n", err)
	}

	// Acquire session credentials.
	slog.Info("Getting aws session credentials...")
	if err := g_aws.GetSessionCreds(g_app); err != nil {
		fmt.Printf("Error getting session credentials: %v\n", err)
	}

	// Rewrite ~/.aws/credentials file.
	slog.Info("Writing updated aws credentials file...")
	if err := g_aws.CreateUpdatedConfig(); err != nil {
		fmt.Printf("Error creating updated config: %v\n", err)
	}

	// Print environment variable message if not the selected profile.
	if os.Getenv("AWS_PROFILE") != g_app.Profile {
		fmt.Printf(EnvVarMessageTemplate, g_app.Profile)
	}
}
