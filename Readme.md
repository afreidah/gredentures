---

 Gredentures

**Gredentures** is a CLI tool designed to simplify the management of AWS credentials and configurations, particularly for workflows involving multi-factor authentication (MFA). It provides functionality for parsing configuration files, managing session tokens, and updating AWS credentials files.

---

## Features

- **AWS Credential Management**:
  - Retrieve default AWS credentials.
  - Generate and manage session credentials using MFA.
  - Update AWS credentials files with default and session credentials.

- **Configuration Management**:
  - Parse command-line arguments and YAML configuration files.
  - Validate required options for MFA workflows.
  - Dynamically write and load configuration files.

- **Logging**:
  - Configurable logging levels (info and debug) for better visibility.

- **Testing**:
  - Comprehensive unit tests for configuration and AWS credential management.
  - Mocked AWS STS client for testing session token generation.

---

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/alexfreidah/gredentures.git
   cd gredentures
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the CLI binary:
   ```bash
   go build -o build/gredentures ./cmd/gredentures
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

---

## Usage

### Command-Line Options

The CLI supports the following options:

```text
Usage:
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
  --help                            Show this help message
```

### Example Commands

1. Generate session credentials with MFA:
   ```bash
   gredentures -t 123456 -o my-org -d arn:aws:iam::123456789012:mfa/my-device
   ```

2. Use a custom configuration file:
   ```bash
   gredentures --config /path/to/config.yml --token 123456
   ```

3. Enable verbose logging:
   ```bash
   gredentures --verbose -t 123456
   ```

---

## Configuration

### YAML Configuration File

The CLI supports a YAML configuration file for storing default values. Example:

```yaml
gredentures:
  Org: my-org
  Device: arn:aws:iam::123456789012:mfa/my-device
  Timeout: 3600
```

The default configuration file path is `$HOME/.gredentures.yml`. You can specify a custom path using the `--config` flag.

---

## Development

### Directory Structure

```plaintext
.
├── build/                 # Output directory for the compiled binary
├── cmd/
│   └── gredentures/       # Main entry point for the CLI
│       └── main.go
├── pkg/
│   ├── appconfig/         # Configuration management logic
│   │   ├── appconfig.go
│   │   ├── appconfig_test.go
│   │   └── mocks/
│   └── awsconfig/         # AWS credential management logic
│       ├── awsconfig.go
│       ├── awsconfig_test.go
│       └── mocks/
│           └── mock_sts.go
└── taskfile.yaml          # Taskfile for automating builds and tests
```

### Running Tasks

This project uses `Taskfile` for automation. Install `Task` and run the following commands:

- Build the binary:
  ```bash
  task build
  ```

- Run tests:
  ```bash
  task test
  ```

- Watch for changes and rebuild:
  ```bash
  task watch-build
  ```

- Watch for changes and re-run tests:
  ```bash
  task watch-test
  ```

---

## Testing

The project includes unit tests for both `appconfig` and `awsconfig` packages. Tests are written using the `testing` package and `testify` for assertions.

### Running Tests

Run all tests with:
```bash
go test ./...
```

### Mocking

The `awsconfig` package includes a mocked AWS STS client (`mock_sts.go`) for testing session token generation without making actual AWS API calls.

---

## Contributing

1. Fork the repository.
2. Create a new branch for your feature or bugfix.
3. Submit a pull request with a detailed description of your changes.

---

## License

This project is licensed under the MIT License. See the `LICENSE` file for details.

---

## Acknowledgments

- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2)
- [Koanf](https://github.com/knadh/koanf) for configuration management
- [Docopt](https://github.com/docopt/docopt.go) for command-line argument parsing
- [Testify](https://github.com/stretchr/testify) for testing utilities

---
