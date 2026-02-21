# Aggregate Package

Go-native implementation of Tracee behavior aggregation with baseline deduplication for supply chain security analysis.

## Features

- **Per-process aggregation**: Detailed stats for each process
- **Baseline deduplication**: Subtract known-safe behavior to find anomalies
- **Streaming JSONL parser**: Memory-efficient processing of large files
- **node_modules filtering**: Automatically filters out npm cache noise
- **Network activity tracking**: DNS queries and IP connections
- **Risk flag detection**: Suspicious patterns (shells, sensitive files, etc.)

## Usage

### CLI

```bash
# Build
go build -o aggregate-cli ./cmd/aggregate

# Create safe baseline (run on known-good npm install)
./aggregate-cli -input safe-sample.jsonl -collection safe -output safe.json

# Analyze target with deduplication
./aggregate-cli -input behavior.jsonl -collection suspicious \
  -dedup-source safe.json -output diff.json

# Send diff.json to LLM for security analysis
```

## Workflow

1. **Create Baseline**: Run aggregation on a known-safe npm install
2. **Analyze Target**: Run with `-dedup-source` to subtract baseline
3. **Review Delta**: The output contains only suspicious/anomalous behavior
4. **LLM Analysis**: Send the diff to an LLM for security assessment

## Output Format

### DedupedProcessStats

```json
{
  "collection": "suspicious",
  "baseline_source": "safe",
  "count_processes": 9,
  "removed_processes": 2,
  "removed_files": 53,
  "removed_commands": 6,
  "removed_syscalls": 7,
  "per_process": {
    "trufflehog": {
      "file_access": {
        "/etc/passwd": 3,
        "/root/.truffler-cache/trufflehog": 8
      },
      "executed_commands": {
        "/tmp/overseer-ed11f0accafecab4": 1
      },
      "network_activity": {
        "ips": {...},
        "dns_records": {
          "github.com": 2,
          "oss.trufflehog.org": 1
        }
      }
    }
  }
}
```

## Dedup Logic

For each process present in both target and baseline:
- **Files**: Only keep files not accessed in baseline
- **Commands**: Only keep commands not executed in baseline
- **Syscalls**: Keep only additional syscalls (count - baseline count)
- **Network**: Only new IPs/DNS queries
- **Processes**: Remove entirely if all behavior matches baseline

## Example Analysis

```bash
# Safe baseline
./aggregate-cli -input safe-sample.jsonl -collection safe -output safe.json
# Output: 6 processes (npm, node, sh, mkdir, etc.)

# Suspicious package
./aggregate-cli -input malicious.jsonl -collection evil \
  -dedup-source safe.json -output diff.json
# Output: 9 processes after dedup
# Shows: curl downloading from GitHub, trufflehog scanning, shell chains
```

## Risk Indicators in Output

Look for these in the deduped output:

- **New processes**: curl, wget, trufflehog (not in baseline)
- **Sensitive files**: /etc/passwd, /etc/shadow, .ssh/*
- **Network activity**: External IPs, suspicious DNS
- **Shell execution**: Command chains spawning shells
- **Download tools**: curl, wget fetching external resources

## Performance

- Safe sample (6 processes): ~15ms
- Suspicious sample (11 processes): ~30ms
- Deduplication: ~20µs
- Streaming: handles millions of events efficiently
