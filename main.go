// gredentures
package main

import (
	"fmt"
	"log/slog"
	"os"

	appc "gredentures/pkg/appconfig"
	appa "gredentures/pkg/awsconfig"
)

const EnvVarMessage = `***********************************************************************************
* To use your session creds by default please add the following to your profile:  *
*                                                                                 *
* export AWS_PROFILE=default-mfa                                                  *
* then source your profile                                                        *
***********************************************************************************
`

func main() {
	var g_app appc.AppConfig
	var g_aws appa.AwsConfig

	// PARSE COMMAND LINE ARGS
	if err := g_app.Parse(os.Args[1:]); err != nil {
		fmt.Printf("Error parsing command line arguments: %v\n", err)
	}

	// VALIDATE GREDENTURES CONFIG AND OPTIONS
	slog.Info("Validating gredentures options and config...")
	if err := g_app.ValidateOptions(); err != nil {
		fmt.Printf("Error validating options: %v\n", err)
	}

	// LOAD DEFAULT CREDS
	slog.Info("Getting default aws credentials...")
	if err := g_aws.GetDefaultCreds(); err != nil {
		fmt.Printf("Error getting default credentials: %v\n", err)
	}

	// ACQUIRE SESSION CREDS
	slog.Info("Getting aws session credentials...")
	if err := g_aws.GetSessionCreds(g_app); err != nil {
		fmt.Printf("Error getting session credentials: %v\n", err)
	}

	// REWRITE ~/.aws/credentials FILE
	slog.Info("Writing updated aws credentials file...")
	if err := g_aws.CreateUpdatedConfig(); err != nil {
		fmt.Printf("Error creating updated config: %v\n", err)
	}

	// PRINT ENV VAR MESSAGE IF NOT DEFAULT-MFA
	if os.Getenv("AWS_PROFILE") != "default-mfa" {
		fmt.Print(EnvVarMessage)
	}
}
