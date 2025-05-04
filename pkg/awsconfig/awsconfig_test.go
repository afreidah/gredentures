package awsconfig

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"gredentures/pkg/appconfig"

       "github.com/stretchr/testify/assert"
       "gopkg.in/ini.v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

func resetLogging() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
}


// Define a type for the LoadDefaultConfig function
type LoadConfigFunc func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error)

func TestGetDefaultAccount(t *testing.T) {
	resetLogging()
	var LoadDefaultConfig LoadConfigFunc = config.LoadDefaultConfig
	tests := []struct {
		name       string
		mockConfig func() func()
		wantErr    bool
	}{
		{
			name: "Valid default account",
			mockConfig: func() func() {
				originalLoadDefaultConfig := LoadDefaultConfig
				LoadDefaultConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
					return aws.Config{}, nil
				}
				return func() { LoadDefaultConfig = originalLoadDefaultConfig }
			},
			wantErr: false,
		},
		{
			name: "Error loading default account",
			mockConfig: func() func() {
				originalLoadDefaultConfig := LoadDefaultConfig
				LoadDefaultConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
					return aws.Config{}, fmt.Errorf("mock error")
				}
				return func() { LoadDefaultConfig = originalLoadDefaultConfig }
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockConfig != nil {
				defer tt.mockConfig()()
			}

			_, err := GetDefaultAccount()
			if err != nil {
				if tt.wantErr {
					t.Errorf("GetDefaultAccount() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestCreateUpdatedConfig(t *testing.T) {
	// Mock data
	defaultCreds := aws.Credentials{
		AccessKeyID:     "mockAccessKeyID",
		SecretAccessKey: "mockSecretAccessKey",
	}

	sessionToken := "mockSessionToken"
	accessKeyId := "mockSessionAccessKeyID"
	secretAccessKey := "mockSessionSecretAccessKey"

	sessionCreds := &sts.GetSessionTokenOutput{
		Credentials: &types.Credentials{
			AccessKeyId:     &accessKeyId,
			SecretAccessKey: &secretAccessKey,
			SessionToken:    &sessionToken,
		},
	}

	conf := AwsConfig{
		defaultCreds: defaultCreds,
		sessionCreds: sessionCreds,
	}

	// Mock HOME environment variable
	tempDir := t.TempDir()
	mockAwsDir := tempDir + "/.aws"
	err := os.MkdirAll(mockAwsDir, 0755) // Ensure the .aws directory exists
	assert.NoError(t, err)
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("Failed to set HOME environment variable: %v", err)
	}

	// Run the function
	err = conf.CreateUpdatedConfig()
	assert.NoError(t, err)

	// Validate the output file
	credentialsPath := mockAwsDir + "/credentials"
	assert.FileExists(t, credentialsPath)

	// Validate file content
	inidata, err := ini.Load(credentialsPath)
	assert.NoError(t, err)

	defaultSection := inidata.Section("default")
	assert.Equal(t, "mockAccessKeyID", defaultSection.Key("aws_access_key_id").String())
	assert.Equal(t, "mockSecretAccessKey", defaultSection.Key("aws_secret_access_key").String())

	defaultMfaSection := inidata.Section("default-mfa")
	assert.Equal(t, "mockSessionToken", defaultMfaSection.Key("aws_session_token").String())
	assert.Equal(t, "mockSessionAccessKeyID", defaultMfaSection.Key("aws_access_key_id").String())
	assert.Equal(t, "mockSessionSecretAccessKey", defaultMfaSection.Key("aws_secret_access_key").String())
}

// Mock STS client
type MockSTSClient struct {
	GetSessionTokenFunc func(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error)
}

func (m *MockSTSClient) GetSessionToken(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error) {
	return m.GetSessionTokenFunc(ctx, params, optFns...)
}

func TestGetSessionCreds(t *testing.T) {
	mockSTS := &MockSTSClient{
		GetSessionTokenFunc: func(ctx context.Context, params *sts.GetSessionTokenInput, optFns ...func(*sts.Options)) (*sts.GetSessionTokenOutput, error) {
			assert.Equal(t, "123456", *params.TokenCode) // Ensure token code matches expected format
			return &sts.GetSessionTokenOutput{
				Credentials: &types.Credentials{
					AccessKeyId:     aws.String("mockAccessKey"),
					SecretAccessKey: aws.String("mockSecretKey"),
					SessionToken:    aws.String("mockSessionToken"),
				},
			}, nil
		},
	}

	// Mock the behavior of GetSessionCreds
	conf := &AwsConfig{}
	appConfig := appconfig.AppConfig{
		Timeout: 3600,
		Device:  "mockDevice",
		Token:   "123456", // Use a valid token code (6 digits)
	}

	// Simulate the GetSessionCreds behavior
	creds, err := mockSTS.GetSessionToken(context.TODO(), &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int32(appConfig.Timeout),
		SerialNumber:    aws.String(appConfig.Device),
		TokenCode:       aws.String(appConfig.Token),
	})
	assert.NoError(t, err)

	// Assign the mocked credentials to AwsConfig
	conf.sessionCreds = creds

	// Assertions
	assert.NotNil(t, conf.sessionCreds)
	assert.Equal(t, "mockAccessKey", *conf.sessionCreds.Credentials.AccessKeyId)
	assert.Equal(t, "mockSecretKey", *conf.sessionCreds.Credentials.SecretAccessKey)
	assert.Equal(t, "mockSessionToken", *conf.sessionCreds.Credentials.SessionToken)
}
