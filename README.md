# Secure Package Registry

A preventative security layer for open-source software supply chains. Instead of detecting compromised packages after they have already caused damage, we intercept them before they ever reach your project.

## Inspiration

One week. That was how long I was blocked with nothing to do. Reason? Security review of a critical dependency we needed for a feature. While interning at Huawei on the cloud reliability team, anything that made it to production had to go through a compliance process to ensure dependencies were safe to use. Only after packages made it into Cloudartifact (their private registry) could things move forward. I was naively mildly annoyed back then.

Fast forward to September 2025, and all my API keys and secrets were now in a public GitHub repo. The [Shai-Hulud worm](https://socket.dev/blog/ongoing-supply-chain-attack-targets-crowdstrike-npm-packages) spread through the NPM ecosystem like wildfire, and a simple `npm install` was all it took to get pwned.

So surely there is an in-between where small to medium-sized companies (regulated or not) did not need to hire expensive security teams to do all that duplicate work of reviewing supply chains and moving at a snail's pace in development, while also not simply praying not to get hacked.

The answer was a disappointing no. Existing tools like Snyk and Socket.dev are after-the-fact detectors that rely on indicators of compromise (IoCs). If you are an early victim developing fast, there is nothing they can do.

## What It Does

We place a layer between you and the large attack surface of package registries (NPM, PyPi, Maven), and this layer scales to millions of packages while maintaining reliability.

**From the user's perspective:**

1. Upload your list of dependencies (e.g. `package.json`)
2. We run each one through a series of automated tests, collecting behavioral data using eBPF (file accesses, network connections, DNS lookups, spawned processes)
3. We maximize the signal-to-noise ratio by filtering out known common behavior
4. Results are aggregated into a summary for LLM agents with MCP tooling, allowing deeper investigation of interesting behavior
5. Suspicious packages are flagged for human review and blocked from the registry
6. Safe packages are promoted to a private registry
7. Configure npm to use the private registry with a single command (`npm config set`)
8. Install dependencies with peace of mind

When tested against real malware samples from September 2025, the system correctly detected credential exfiltration attempts via trufflehog and blocked the malicious packages from the registry.

## How We Built It

We planned the implementation in detail and experimented with various libraries and tools before the hackathon, but all code was written during the hackathon itself. The scope was large, covering ground that would normally take multiple weeks.

We relied heavily on open-source to minimize effort, but still worked for 22 hours with only food and bathroom breaks.

To maximize efficiency, we split the team into front-end and back-end groups, mostly mocking the front-end until we could plug in actual data. For technology choices, we went with Golang for the back-end because its ecosystem fits the use case, and React for the front-end due to team experience. GitHub Actions was used as the runner, as it is simpler than setting up a custom container environment securely.

### Architecture

The system works as a pipeline:

1. **Dependency Resolution** -- Parse `package.json` into a dependency tree
2. **Behavioral Analysis** -- For each package, spawn an isolated container and monitor it with Tracee (eBPF-based runtime security), capturing all syscalls: `execve`, `open`, `connect`, `read`, `write`, etc.
3. **Data Aggregation** -- Reduce high-volume behavioral data into structured summaries: domain contacts, file reads/writes, command executions, and network activity
4. **LLM Reasoning** -- An AI agent with MCP tooling analyzes the aggregated data, reasons about anomalies (honeypot token access, suspicious domains, unexpected file writes), and makes approval/rejection decisions
5. **Registry Promotion** -- Approved packages are promoted to a secured private registry; flagged packages block their entire dependency chain

### Key Technical Decisions

- **Tracee over eCapture**: Tracee has native container scoping, captures all syscalls (not just network), and outputs JSON ready for LLM consumption
- **Two-registry strategy**: Separate "unsafe" (immediate mirror) and "safe" (post-analysis) registries in Gitea
- **GitHub Actions as runner**: Simpler and more secure than managing our own container infrastructure
- **Top-down bisection**: When a top-level package is flagged, we traverse down the dependency tree to isolate the specific cause, reducing work on the most common happy paths

## What's Next

This is only a small part of a much larger system we plan to build over the coming months and eventually years. We have the core done: automated behavioral checks and a secure package registry that we can use today.

From the technical side, we want reproducible builds that further enhance reliability, handling of edge cases like non-deterministic git/https dependencies, and automatic polling from NPM to have secured packages ready before companies need them. We can also capture market share from existing players by expanding CVE detection and risk management.

However, the most important part of our vision is the proper funding of open source. By providing security for enterprises looking to utilize open source, we serve as a legitimate means to distribute the profits derived from societal good back to the original contributors. For companies, they can simplify complex financial relationships with numerous individual contributors, while volunteers no longer have to beg for funding.
