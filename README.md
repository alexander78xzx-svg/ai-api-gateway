# FinOps & Security AI API Gateway

A high-throughput , zero dependency L7 reverse proxy built in Go. Designed to sit between developper AI coding agents (Claude Code) and LLM providers (Anthropic).

This gateway drastically reduces __LLM API costs__, intercepts __sensitive data__ before it even leaves the local network and provides zero-latency __token streaming__ with multi-tier __fault tolerance__.

## Business value & Impact
- **Data Loss Prevention** (Security): Automatically strips AWS keys, SSH private keys, and environment variables from developer prompts before they reach external AI providers.

- __FinOps & Cost Reduction__:  Truncates massive terminal crash logs (saving up to 80% of input tokens) and dynamically downgrades simple tasks to cheaper models (e.g., Sonnet → Haiku).

- __High Availability__ (Reliability): handles upstream 500, 529, and 429 errors thanks to a 3-tier fallback chain, ensuring developers never experience an AI outage.

## System Architecture

The gateway executes a strict, low-latency pipeline to normalize and route payloads.

![Gateway Architecture Diagram](assets/architecture.png)

- __Authorization__: Validates incoming requests via API keys.

- __Deterministic Caching__: Checks an in-memory hash map for an __exact cache match__. If found, returns the answer directly to the client with __0 API cost__.

- __Payload Inspection & Normalization__:
Analyzes the last message content to determine if it is code, JSON, or a log/shell trace.
If it is a log/trace, the gateway performs __character clean-up, head/tail truncation, and hides private data__ (Pattern-Based Secret Redaction).

- __Dynamic Model Router__: Analyzes the normalized prompt complexity and chooses whether to downgrade to a cheaper, faster model.

- __Fault-Tolerant Upstream Call__: Sends the request to the upstream API. If the API does not return a Status 200, the gateway repeats the attempt __up to 3 times__, actively changing to fallback models if necessary.

- __Stream__: Upon a successful 200 OK, the gateway concurrently streams the Server-Sent Events (SSE) response back to the client while normalizing the response and saving it to memory.

## Core Technical Features
1. __Dual-Write Stream Teeing__ (Zero-Latency) : 
Streaming AI responses requires zero buffering. The gateway leverages Go's http.Flusher to push \n delimited tokens to the client instantly, while simultaneously writing the byte stream to an asynchronous memory buffer for future cache hits.

2. __Deterministic Cache Hashing__ :
Unlike standard proxies that cache raw HTTP requests, this gateway hashes the payload __after__ the DLP and Truncator modules have run. By stripping variable entropy (like dynamic timestamps and secrets) from the logs, the exact-match cache hit rate increases significantly.

3. __Multi-Stage Docker Build__ : 
Deployed via a highly optimized scratch __Docker__ image. By disabling CGO and compiling a static binary, the final production image is __extremely lightweight (~9.3MB)__ and inherently __secure against shell-based exploits.__

## Benchmarks & Testing

The critical mutation path (Security + Truncator) is fully benchmarked using Go's standard `testing` package to ensure it does not bottleneck the network stream. 

You can find the core tests located directly alongside their respective packages:
* **DLP Engine:** `pkg/security/security_test.go`
* **Log Truncation & Token Savings:** `pkg/truncator/truncator_test.go`
* **Dynamic Fallback Routing:** `pkg/router/router_test.go`

**Performance Metrics (Apple M5):**
Processing a massive 12,000+ token crash log through the regex engine and string allocation pipeline yields the following results:
* **Latency Overhead:** Adds only **~0.61ms** of processing time per request.
* **FinOps Savings:** Safely truncates to ~1,700 tokens, saving an average of **10,400+ input tokens** per massive payload.

To run the unit tests and benchmarks yourself:
```bash
go test -v -bench=. ./...
```

## Quick Start
You can run the gateway locally with a single command.

Prerequisites: __Docker__ and __Docker Compose__.

Clone the repository and navigate to the root directory.
Create a __.env__ file and add your __Anthropic API key__ :
```Bash
ANTHROPIC_API_KEY=sk-ant-your-real-api-key-here
```
Start the gateway:

```Bash
docker-compose up -d --build
```
The proxy is now listening on http://localhost:8080/v1/messages.

