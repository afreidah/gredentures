// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/aws/aws-sdk-go-v2/config (interfaces: Config)
//
// Generated by this command:
//
//	mockgen -destination=pkg/awsconfig/mocks/mock_sts.go -package=mocks github.com/aws/aws-sdk-go-v2/config Config
//

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "go.uber.org/mock/gomock"
)

// MockConfig is a mock of Config interface.
type MockConfig struct {
	ctrl     *gomock.Controller
	recorder *MockConfigMockRecorder
	isgomock struct{}
}

// MockConfigMockRecorder is the mock recorder for MockConfig.
type MockConfigMockRecorder struct {
	mock *MockConfig
}

// NewMockConfig creates a new mock instance.
func NewMockConfig(ctrl *gomock.Controller) *MockConfig {
	mock := &MockConfig{ctrl: ctrl}
	mock.recorder = &MockConfigMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConfig) EXPECT() *MockConfigMockRecorder {
	return m.recorder
}
