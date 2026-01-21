# gotest

Automated Go testing: runs vet, stdlib tests, race detection, coverage, and WASM tests.

## Installation

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Usage

```bash
gotest
```

## What it does

1. Runs `go vet ./...`
2. Runs `go test -race -cover ./...` (stdlib tests only)
3. Calculates coverage
4. Auto-detects and runs WASM tests if found (`*Wasm*_test.go`)
5. Updates README badges

## Test Caching

`gotest` includes an intelligent caching mechanism to avoid re-running tests when the code hasn't changed.

- **How it works**: It generates a unique key for the current module based on its git state (last commit hash + hash of uncommitted changes).
- **Behavior**: If a match is found in the cache, `gotest` returns the previous successful result immediately without executing any tests.
- **Persistence**: Caches are stored in `/tmp/gotest-cache/` and are automatically invalidated if any `.go` file or the git state changes.

## Output

```
✅ vet ok, ✅ tests stdlib ok, ✅ race detection ok, ✅ coverage: 71%, ✅ tests wasm ok
```

**Cached run:**
The message is identical to the original run, but it executes instantly.

**On failure:**
Shows only failed tests with error details, filters out passing tests. Failed runs are never cached.

## Exit codes

- `0` - All tests passed
- `1` - Tests failed, vet issues, or race conditions detected

## Notes

- No flags required - auto-detects test types
- Filters verbose output automatically
- Badge updates in `README.md` under `BADGES_SECTION`
