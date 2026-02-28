# Docker Compose Field Order Validator

A custom prek/pre-commit hook for enforcing standardized Docker Compose YAML field ordering and alphabetization.

## Project Overview

### Purpose
Create a standalone validation tool that enforces consistent Docker Compose service definitions across multiple repositories. The validator ensures:

- **Field Order**: Services follow a strict, predefined field order
- **Alphabetization**: Environment variables, volumes, and labels are sorted alphabetically
- **Auto-Fix**: Can automatically reformat non-compliant files
- **Blocking**: Prevents commits with improperly formatted compose files

### Target Stack
This validator was originally designed for a media server homelab with the following standardization:

```yaml
services:
  example:
    container_name: xxx        # 1. Identity
    image: xxx                 #    (or build:)
    user: xxx                  # 2. User/Permissions (optional)
    environment:               # 3. Environment (A-Z sorted)
      - AAA=xxx
      - ZZZ=xxx
    env_file:                  #    (if needed)
      - xxx
    networks:                  # 4. Runtime
      - proxy_net
    network_mode: service:xxx  #    (mutually exclusive)
    ports:
      - "xxx:xxx"
    devices:                   # 5. Devices (if needed)
      - xxx
    healthcheck:               # 6. Health checks (if needed)
      test: xxx
    restart: always            # 7. Lifecycle
    cap_add:                   # 8. Capabilities (if needed)
      - xxx
    privileged: true           #    (if needed)
    extra_hosts:               #    (if needed)
    volumes:                   # 9. Storage (A-Z sorted)
      - /aaa:/aaa
      - /zzz:/zzz
    labels:                    # 10. Labels (A-Z sorted, always last)
      - "traefik..."
      - "wud..."
```

## Technical Specifications

### Field Order Rules

The validator enforces this exact field sequence:

```python
FIELD_ORDER = [
    # Identity
    'container_name',
    'image',
    'build',

    # Permissions
    'user',

    # Environment (MUST be alphabetized)
    'environment',
    'env_file',

    # Runtime
    'networks',
    'network_mode',
    'ports',
    'devices',

    # Health checks
    'healthcheck',

    # Lifecycle
    'restart',
    'cap_add',
    'privileged',
    'extra_hosts',

    # Storage (MUST be alphabetized)
    'volumes',

    # Proxy integration (MUST be alphabetized, always last)
    'labels'
]
```

### Alphabetization Rules

#### Environment Variables
```yaml
# CORRECT (alphabetized)
environment:
  - DOZZLE_ENABLE_ACTIONS=true
  - DOZZLE_REMOTE_AGENT=mlrig:7007
  - TZ=Europe/Paris

# INCORRECT
environment:
  - TZ=Europe/Paris
  - DOZZLE_ENABLE_ACTIONS=true
  - DOZZLE_REMOTE_AGENT=mlrig:7007
```

#### Volumes
```yaml
# CORRECT (alphabetized by source path)
volumes:
  - /media/hdd/filesharing/:/app/upload
  - ../config/rustypaste/config.toml:/app/config.toml

# INCORRECT
volumes:
  - ../config/rustypaste/config.toml:/app/config.toml
  - /media/hdd/filesharing/:/app/upload
```

#### Labels
```yaml
# CORRECT (alphabetized by key)
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.share.entrypoints=websecure"
  - "wud.watch=false"

# INCORRECT
labels:
  - "wud.watch=false"
  - "traefik.enable=true"
```

## Repository Structure

```
docker-compose-field-validator/
├── .github/
│   └── workflows/
│       └── test.yml              # CI tests
├── .pre-commit-hooks.yaml        # Pre-commit hook definition
├── README.md                     # Main documentation
├── pyproject.toml               # Python package config
├── setup.py                     # Package setup
├── src/
│   └── compose_validator/
│       ├── __init__.py
│       ├── validator.py         # Core validation logic
│       ├── fixer.py             # Auto-fix implementation
│       └── cli.py               # Command-line interface
├── tests/
│   ├── test_validator.py
│   ├── test_fixer.py
│   └── fixtures/
│       ├── valid-compose.yml
│       ├── invalid-order.yml
│       └── invalid-alpha.yml
└── examples/
    ├── .pre-commit-config.yaml  # Example pre-commit config
    └── stacks/                   # Example compose files
        └── example.yml
```

## Implementation Plan

### Phase 1: Core Validator (MVP)

**Tasks:**
1. Create Python package structure
2. Implement YAML parsing with ruamel.yaml (preserves comments/formatting)
3. Implement field order validation
4. Implement alphabetization checks
5. Create CLI with `--check` mode
6. Add comprehensive error reporting with line numbers

**Deliverables:**
- `src/compose_validator/validator.py`
- `src/compose_validator/cli.py`
- Basic test suite

**Success Criteria:**
- Can detect field order violations
- Can detect alphabetization violations
- Provides clear error messages with line numbers
- Returns non-zero exit code on violations

### Phase 2: Auto-Fix Capability

**Tasks:**
1. Implement field reordering without losing comments
2. Implement alphabetization auto-sorting
3. Add `--fix` CLI flag
4. Handle edge cases (multi-line strings, anchors, etc.)
5. Preserve YAML formatting and comments

**Deliverables:**
- `src/compose_validator/fixer.py`
- Updated CLI with `--fix` flag
- Integration tests

**Success Criteria:**
- Can auto-fix all violations
- Preserves comments and formatting
- Creates valid YAML output
- Idempotent (running twice produces same result)

### Phase 3: Prek/Pre-commit Integration

**Tasks:**
1. Create `.pre-commit-hooks.yaml` definition
2. Support both `pre-commit` and `prek` tools
3. Add staged file detection
4. Support `--fix` in pre-commit mode (with instructions)
5. Create example configurations

**Deliverables:**
- `.pre-commit-hooks.yaml`
- Example `.pre-commit-config.yaml`
- Installation documentation

**Success Criteria:**
- Works as pre-commit hook
- Works with prek
- Blocks commits on violations
- Can be bypassed with `--no-verify` (standard practice)

### Phase 4: CI/CD and Distribution

**Tasks:**
1. Set up GitHub Actions for testing
2. Create PyPI package
3. Add to pre-commit-hooks repository listing
4. Create comprehensive documentation
5. Add support for additional field order presets

**Deliverables:**
- Published PyPI package
- GitHub releases
- Full documentation site

## Implementation Details

### Core Algorithm

```python
def validate_service(service_name: str, config: dict) -> List[str]:
    """Validate a single service configuration."""
    errors = []

    # 1. Check field order
    actual_fields = [k for k in config.keys() if k in FIELD_ORDER]
    expected_fields = [f for f in FIELD_ORDER if f in config.keys()]

    for i, (actual, expected) in enumerate(zip(actual_fields, expected_fields)):
        if actual != expected:
            errors.append(
                f"{service_name}.{actual}: should be at position {i}, "
                f"but '{expected}' is expected there"
            )

    # 2. Check environment alphabetization
    if 'environment' in config:
        env_vars = config['environment']
        if isinstance(env_vars, list):
            keys = [parse_env_key(e) for e in env_vars]
            if keys != sorted(keys, key=str.lower):
                errors.append(f"{service_name}.environment: not alphabetized")

    # 3. Check volumes alphabetization
    if 'volumes' in config:
        vols = config['volumes']
        if isinstance(volumes, list):
            sources = [v.split(':')[0] for v in vols if ':' in v]
            if sources != sorted(sources, key=str.lower):
                errors.append(f"{service_name}.volumes: not alphabetized")

    # 4. Check labels alphabetization
    if 'labels' in config:
        labels = config['labels']
        if isinstance(labels, list):
            keys = [parse_label_key(l) for l in labels]
            if keys != sorted(keys, key=str.lower):
                errors.append(f"{service_name}.labels: not alphabetized")

    return errors
```

### Auto-Fix Algorithm

```python
def fix_service(service_config: dict) -> dict:
    """Reorder and alphabetize a service configuration."""
    fixed = {}

    # Reorder fields
    for field in FIELD_ORDER:
        if field in service_config:
            value = service_config[field]

            # Alphabetize if needed
            if field == 'environment' and isinstance(value, list):
                value = alphabetize_environment(value)
            elif field == 'volumes' and isinstance(value, list):
                value = alphabetize_volumes(value)
            elif field == 'labels' and isinstance(value, list):
                value = alphabetize_labels(value)

            fixed[field] = value

    # Preserve any extra fields not in FIELD_ORDER
    for field, value in service_config.items():
        if field not in FIELD_ORDER:
            fixed[field] = value

    return fixed
```

### CLI Interface

```bash
# Check mode (default)
docker-compose-validator stacks/*.yml
# Exit code: 0 if valid, 1 if errors

# Fix mode
docker-compose-validator --fix stacks/*.yml
# Rewrites files in place

# Verbose mode
docker-compose-validator -v stacks/*.yml

# Specific checks only
docker-compose-validator --check-order-only stacks/*.yml
docker-compose-validator --check-alphabetization-only stacks/*.yml

# Configuration file
docker-compose-validator --config .compose-validator.yaml stacks/*.yml
```

## Configuration Options

Users can customize behavior via `.compose-validator.yaml`:

```yaml
# Field order (customizable)
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

## Installation Methods

### Method 1: Pre-commit Hook (Recommended)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/yourusername/docker-compose-field-validator
    rev: v1.0.0
    hooks:
      - id: docker-compose-field-validator
        files: ^stacks/.*\.yml$
```

Install:
```bash
pip install pre-commit
pre-commit install
```

### Method 2: Prek (Rust-based, faster)

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/yourusername/docker-compose-field-validator
    rev: v1.0.0
    hooks:
      - id: docker-compose-field-validator
        files: ^stacks/.*\.yml$
```

Install:
```bash
curl --proto '=https' --tlsv1.2 -LsSf https://github.com/j178/prek/releases/latest/download/prek-installer.sh | sh
prek install
```

### Method 3: Standalone CLI

```bash
pip install docker-compose-field-validator
docker-compose-validator stacks/*.yml
```

### Method 4: Docker

```bash
docker run --rm -v $(pwd):/workspace \
  docker-compose-field-validator:latest \
  /workspace/stacks/*.yml
```

## Error Output Examples

### Field Order Violation
```
ERROR: stacks/media.yml

Service 'bazarr': Field order violation
  Line 15: 'ports' should come after 'networks'

Expected order:
  1. container_name
  2. image
  3. environment
  4. networks
  5. ports        <-- here
  6. restart
  7. volumes
  8. labels

Actual order:
  ...
  4. ports        <-- wrong position (should be #5)
  5. networks
  ...

Fix: Move 'ports' after 'networks'
Auto-fix available: run with --fix
```

### Alphabetization Violation
```
ERROR: stacks/downloads.yml

Service 'gluetun': environment variables not alphabetized
  Line 12-16:
    - OPENVPN_PASSWORD=${OPENVPN_PASSWORD}
    - OPENVPN_USER=${OPENVPN_USER}
    - SERVER_COUNTRIES=France            <-- wrong position
    - VPNSP=fastestvpn                   <-- wrong position
    - VPN_TYPE=openvpn

Expected order (alphabetical):
    - OPENVPN_PASSWORD=${OPENVPN_PASSWORD}
    - OPENVPN_USER=${OPENVPN_USER}
    - SERVER_COUNTRIES=France
    - VPN_TYPE=openvpn
    - VPNSP=fastestvpn

Auto-fix available: run with --fix
```

## Testing Strategy

### Unit Tests
- Field order detection
- Alphabetization detection
- Edge cases (empty services, missing fields)
- YAML parsing edge cases

### Integration Tests
- Full file validation
- Auto-fix verification
- Comment preservation
- Multi-document YAML

### Test Fixtures
Create sample files:
- `valid-compose.yml` - Perfectly formatted
- `invalid-order.yml` - Wrong field order
- `invalid-alpha.yml` - Wrong alphabetization
- `complex.yml` - With comments, anchors, etc.

## Release Checklist

- [ ] Core validator implemented
- [ ] Auto-fix working
- [ ] Pre-commit hook tested
- [ ] Prek compatibility verified
- [ ] PyPI package published
- [ ] Documentation complete
- [ ] CI/CD pipeline passing
- [ ] MIT license applied
- [ ] README with examples
- [ ] CHANGELOG.md created

## Resources

### Similar Projects
- `prettier` - Code formatter (doesn't handle Docker Compose well)
- `yamllint` - YAML linting (no field order validation)
- `docker-compose-linter` - Syntax only, no ordering

### Documentation References
- [Docker Compose Specification](https://github.com/compose-spec/compose-spec/blob/master/spec.md)
- [ruamel.yaml Documentation](https://yaml.readthedocs.io/)
- [Pre-commit Hooks Documentation](https://pre-commit.com/hooks.html)
- [Prek Documentation](https://prek.j178.dev/)

## Maintenance

### Versioning
- Semantic versioning (semver)
- Changelog maintenance
- Deprecation policy for breaking changes

### Support
- GitHub Issues for bugs
- Discussions for questions
- Security policy for vulnerabilities

