# v2n-coremesh

`v2n-coremesh` has two subcommands:

- `parse`: parse v2rayN and generate `xray.generated.json` + `coremesh.state.json`
- `run`: ensure geo assets, start all cores first, then start xray

Default config directory: `$HOME/.v2n_coremesh`

## Build

```bash
cd create_exe
go build ./cmd/v2n-coremesh
```

## Usage

### 1) parse

```bash
./v2n-coremesh parse -v2rayn-home /path/to/v2rayN
./v2n-coremesh parse -v2rayn-home /path/to/v2rayN -conf-dir /custom/conf/dir
```

What `parse` does:

- Parses all custom cores from v2rayN (with active flag)
- Reads xray base config
- Appends non-active cores into xray `outbounds`
- Prepends rules from `custom_rules.yaml` (if present) to `routing.rules`
- Writes:
  - `<conf-dir>/xray.generated.json`
  - `<conf-dir>/coremesh.state.json`

### 2) run

```bash
./v2n-coremesh run
./v2n-coremesh run -conf-dir /custom/conf/dir
```

What `run` does:

- Checks `<conf-dir>/geosite.dat` and `<conf-dir>/geoip.dat`
- Downloads missing/stale files (older than 30 days)
- Starts all cores in order, then starts xray
- Sets xray environment variables:
  - `XRAY_LOCATION_ASSET=<conf-dir>`
  - `XRAY_LOCATION_CERT=<conf-dir>`

On Windows, it also manages system proxy:

- Tries to set system proxy to an inbound endpoint from `xray.generated.json`
- If system proxy is already enabled (including PAC), it does not modify anything
- If proxy was set by this program, it restores previous settings on exit
- `ProxyOverride` keeps existing entries and merges required bypass entries

## parse Input Requirements

- `/path/to/v2rayN/guiConfigs/guiNConfig.json`
- `/path/to/v2rayN/guiConfigs/guiNDB.db`
- `/path/to/v2rayN/bin/xray/xray` (or `.exe` on Windows)
- Custom core configs must expose a detectable listen address (for socks outbound injection)

## custom_rules.yaml

Location: `<conf-dir>/custom_rules.yaml`

Format: YAML array, each item must match one xray `routing.rules[]` object.
