# v2n-coremesh

`v2n-coremesh` orchestrates multiple cores first, then starts xray, and generates a runtime xray config automatically.

## Build

```bash
cd create_exe
go build ./cmd/v2n-coremesh
```

## Usage

### 1) Auto-load from v2rayN (recommended)

```bash
./v2n-coremesh -v2rayn-home /path/to/v2rayN -dry-run
./v2n-coremesh -v2rayn-home /path/to/v2rayN
```

`-v2rayn-home` should point to your v2rayN root directory (contains `guiConfigs/` and `bin/`).

### 2) Manual config file mode

```bash
./v2n-coremesh -config ./examples/config.yaml -dry-run
./v2n-coremesh -config ./examples/config.yaml
```

### 3) Custom core routing example (multi-core)

```bash
./v2n-coremesh -config ./examples/custom-cores.config.yaml -dry-run
./v2n-coremesh -config ./examples/custom-cores.config.yaml
```

Example files:
- `examples/custom-cores.config.yaml`
- `examples/custom-cores.rules.yaml`

Rule behavior in this example:
- `baidu.com` -> `core-naive-a`
- `qq.com` -> `core-naive-b`
- `geosite:cn` and all unmatched traffic -> `direct`

## Flags

- `-v2rayn-home`: auto-parse v2rayN config + database (preferred)
- `-config`: main config file path in manual mode (default: `./config.yaml`)
- `-dry-run`: validate + generate xray config only, do not start processes

## What `-dry-run` does

It will:
- load configuration / parse v2rayN
- validate paths and port conflicts
- generate `xray.generated.json`

It will not:
- start core processes
- start xray process

## Auto Mode Requirements (`-v2rayn-home`)

- `/path/to/v2rayN/guiConfigs/guiNConfig.json`
- `/path/to/v2rayN/guiConfigs/guiNDB.db`
- `/path/to/v2rayN/bin/xray/xray` (or `.exe` on Windows)
- if rules include `geosite:*`, `/path/to/v2rayN/bin/geosite.dat` must exist
- recommended: also provide `/path/to/v2rayN/bin/geoip.dat` (for geoip-based rules)
- custom core configs must be JSON and include listen host/port (used for outbound mapping)

## Output

- Auto mode default output: `/path/to/v2rayN/guiTemps/create_exe/xray.generated.json`
- Manual mode output path: `app.generated_xray_config`

## Common Errors

- `v2rayN gui config not found`:
  the `-v2rayn-home` path is incorrect, or `guiConfigs/guiNConfig.json` is missing
- `v2rayN db not found`:
  `guiConfigs/guiNDB.db` is missing
- `cannot infer listen host:port`:
  listen address is not detectable in the custom core config; update config format or use manual mode
- `duplicate listen endpoint`:
  multiple cores are configured with the same `host:port`

## Notes

- `xray.base_config` must be valid JSON.
- `routing_rules_file` must reference existing `cores[].outbound_tag`.
- `{{config}}` placeholder in args will be replaced with the generated config path.
