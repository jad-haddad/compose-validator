# compose-validator

A Docker Compose YAML field order validator and auto-fixer with support for pre-commit hooks.

## Features

- **Field Order Validation**: Enforces consistent ordering of Docker Compose service fields
- **Alphabetization Checks**: Validates that environment variables, volumes, and labels are alphabetized
- **Auto-Fix**: Automatically reorders and alphabetizes files in-place
- **Configurable**: Per-project configuration via `.compose-validator.yaml`
- **Multi-document Support**: Handles YAML files with multiple documents
- **Pre-commit Integration**: Works with both `pre-commit` and `prek` tools

> **Note**: Comment and anchor preservation is currently limited. See [Known Limitations](#known-limitations) below.

## Default Field Order

```yaml
services:
  example:
    container_name: xxx     # 1. Identity
    image: xxx              #    (or build:)
    user: xxx               # 2. User/Permissions (optional)
    environment:            # 3. Environment (A-Z sorted)
      - AAA=xxx
      - ZZZ=xxx
    env_file:               #    (if needed)
      - xxx
    networks:               # 4. Runtime
      - proxy_net
    network_mode: service:xxx #    (mutually exclusive)
    ports:
      - "xxx:xxx"
    devices:                # 5. Devices (if needed)
      - xxx
    healthcheck:            # 6. Health checks (if needed)
      test: xxx
    restart: always         # 7. Lifecycle
    cap_add:                # 8. Capabilities (if needed)
      - xxx
    privileged: true        #    (if needed)
    extra_hosts:            #    (if needed)
    volumes:                # 9. Storage (A-Z sorted)
      - /aaa:/aaa
      - /zzz:/zzz
    labels:                 # 10. Labels (A-Z sorted, always last)
      - "traefik..."
```

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap yourusername/tap
brew install compose-validator
```

### Go Install

```bash
go install github.com/yourusername/compose-validator/cmd/compose-validator@latest
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/yourusername/compose-validator/releases).

### Pre-commit Hook

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/yourusername/compose-validator
    rev: v1.0.0
    hooks:
      - id: compose-validator
        files: ^docker-compose.*\.ya?ml$
```

Install hooks:

```bash
pip install pre-commit
pre-commit install
```

## Usage

### Check Files

```bash
compose-validator docker-compose.yml
compose-validator docker-compose.yml docker-compose.prod.yml
compose-validator *.yml
```

### Auto-Fix Files

```bash
compose-validator --fix docker-compose.yml
```

### Configuration

Create `.compose-validator.yaml` in your project root:

```yaml
# Custom field order
field_order:
  - container_name
  - image
  - build
  - user
  - environment
  - networks
  - ports
  - restart
  - volumes
  - labels

# Alphabetization rules
alphabetization:
  environment: true
  volumes: true
  labels: true

# Strict mode (no extra fields allowed)
strict: false

# Exclude patterns
exclude:
  - "**/docker-compose.override.yml"
  - "**/test/**"

# Custom field order for specific services
service_overrides:
  database:
    field_order:
      - container_name
      - image
      - environment
      - volumes
```

### CLI Options

```
compose-validator [flags] [files...]

Flags:
  -h, --help                    Help for compose-validator
  -v, --verbose                 Enable verbose output
      --fix                     Automatically fix violations
      --config string          Path to configuration file
      --check-order-only        Only check field order
      --check-alphabetization-only  Only check alphabetization
      --version                 Print version information
```

## Configuration File Locations

The tool searches for configuration in the following order (first found wins):

1. Path specified with `--config`
2. `.compose-validator.yaml` (current directory)
3. `.compose-validator.yml` (current directory)
4. `compose-validator.yaml` (current directory)
5. `compose-validator.yml` (current directory)
6. Same files in parent directories (walking up)

If no configuration is found, default values are used.

## Examples

### Example: Invalid File

```yaml
services:
  app:
    image: nginx:latest      # Wrong order! container_name should come first
    container_name: my-app
    environment:
      - ZZZ=value3           # Not alphabetized
      - AAA=value1
      - MMM=value2
```

### After Auto-Fix

```yaml
services:
  app:
    container_name: my-app   # Correct order
    image: nginx:latest
    environment:
      - AAA=value1           # Now alphabetized
      - MMM=value2
      - ZZZ=value3
```

## How It Works

1. **Parsing**: Uses `gopkg.in/yaml.v3` to parse YAML files into Go data structures
2. **Validation**: 
   - Field order is validated against the configured sequence
   - Environment variables, volumes, and labels are checked for alphabetization
3. **Auto-Fix**: 
   - Reorders fields according to the configuration
   - Sorts alphabetizable fields case-insensitively
   - Regenerates YAML output with proper formatting

## Known Limitations

- **Comment Preservation**: Comments are not currently preserved during auto-fix operations. The YAML is parsed into Go data structures, modified, and regenerated. This is a known limitation that may be addressed in future versions by implementing AST-based manipulation.
- **YAML Anchors**: Anchors (`&`) and aliases (`*`) are expanded during parsing and may not be preserved in the exact original format.
- **Multi-document Files**: While parsing supports multi-document YAML files, the document separator (`---`) may not be preserved in the exact original format during auto-fix.

## Development

### Build

```bash
go build -o compose-validator ./cmd/compose-validator
```

### Test

```bash
go test ./...
```

### Run Locally

```bash
go run ./cmd/compose-validator --fix docker-compose.yml
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- Built with [goccy/go-yaml](https://github.com/goccy/go-yaml) for YAML parsing with comment preservation
- Inspired by the need for consistent Docker Compose configurations in homelab environments
