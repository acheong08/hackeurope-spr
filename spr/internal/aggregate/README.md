# Aggregate Package

Go-native implementation of Tracee behavior aggregation, replacing the Python/MongoDB service.

## Features

- **Streaming JSONL parser**: Memory-efficient processing of large files
- **Aggregations**: syscall profile, file access, executed commands, network connections, DNS queries
- **Risk flag detection**: Automatically detects suspicious patterns
- **Per-process breakdown**: Detailed stats for each process

## Usage

### Library

```go
import "github.com/acheong08/hackeurope-spr/internal/aggregate"

// Overall statistics
agg := aggregate.NewAggregator()
stats, err := agg.ProcessFile("behavior.jsonl", "collection_name")

// Per-process statistics
procAgg := aggregate.NewProcessAggregator()
procStats, err := procAgg.ProcessFile("behavior.jsonl", "collection_name")
```

### CLI

```bash
# Build
go build -o aggregate-cli ./cmd/aggregate

# Overall stats
./aggregate-cli -input behavior.jsonl -collection module2 -output stats.json

# Per-process stats
./aggregate-cli -input behavior.jsonl -collection module2 -per-process -output per_process.json
```

## Output Format

### Stats

```json
{
  "collection": "module2",
  "total_events": 1640,
  "syscall_profile": {
    "openat": 1545,
    "connect": 48,
    "execve": 27,
    "net_packet_dns_request": 16,
    "open": 4
  },
  "file_access": {
    "/etc/passwd": 7,
    "/proc": 32,
    ...
  },
  "executed_commands": {
    "/bin/sh": 5,
    "/bin/bash": 3,
    ...
  },
  "network_activity": {
    "ips": {"1.2.3.4:443": 12},
    "dns_records": {"git.duti.dev": 4}
  },
  "risk_flags": ["sensitive_file_access", "shell_spawned", "network_activity", "procfs_access"]
}
```

### PerProcessStats

```json
{
  "collection": "module2",
  "per_process": {
    "trufflehog": {
      "syscall_profile": {"openat": 150, "execve": 2},
      "file_access": {...},
      "executed_commands": {...},
      "network_activity": {...}
    }
  },
  "count_processes": 11
}
```

## Risk Flags

- `sensitive_file_access`: Access to /etc/passwd, /etc/shadow, /root, .ssh
- `shell_spawned`: Execution of /bin/sh, /bin/bash
- `network_activity`: Any network connections
- `procfs_access`: Access to /proc filesystem

## Performance

Processing 1640 events takes ~20ms on a typical development machine.
