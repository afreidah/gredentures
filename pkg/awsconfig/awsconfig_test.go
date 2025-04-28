package awsconfig

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
