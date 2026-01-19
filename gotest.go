package devflow

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Test executes the test suite for the project
func (g *Go) Test() (string, error) {
	// Detect Module Name
	moduleName, err := getModuleName(".")
	if err != nil {
		return "", fmt.Errorf("error: %v", err)
	}

	// Initialize Status
	testStatus := "Failed"
	coveragePercent := "0"
	raceStatus := "Detected"
	vetStatus := "Issues"

	var msgs []string
	addMsg := func(ok bool, msg string) {
		symbol := "✅"
		if !ok {
			symbol = "❌"
		}
		msgs = append(msgs, fmt.Sprintf("%s %s", symbol, msg))
	}

	// Parallel Phase 1: Vet + WASM detection
	var wg1 sync.WaitGroup
	var vetOutput string
	var vetErr error
	var enableWasmTests bool

	wg1.Add(2)

	// Go Vet (async)
	go func() {
		defer wg1.Done()
		vetOutput, vetErr = RunCommand("go", "vet", "./...")
	}()

	// Check for WASM test files by build tags ONLY (async)
	go func() {
		defer wg1.Done()
		// Use go list to detect packages with test files for the WASM architecture
		// This is cross-platform and more accurate than grep.
		// We check for both internal (.TestGoFiles) and external (.XTestGoFiles) tests.
		cmd := exec.Command("go", "list", "-f", "{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}", "./...")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "GOOS=js", "GOARCH=wasm")
		// We use CombinedOutput or a buffer because go list might fail (exit 1)
		// if some subpackages have import errors under WASM, but it still prints the list.
		out, _ := cmd.CombinedOutput()
		if strings.TrimSpace(string(out)) != "" {
			// Filter out actual error messages to see if there's any package path
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.Contains(line, ":") && !strings.Contains(line, " ") {
					enableWasmTests = true
					break
				}
			}
		}
	}()

	wg1.Wait()

	// Process vet results
	if vetErr != nil {
		// Check if it's just "no packages" error (WASM-only projects)
		if strings.Contains(vetOutput, "matched no packages") ||
			strings.Contains(vetOutput, "no packages to vet") ||
			strings.Contains(vetOutput, "build constraints exclude all Go files") {
			vetStatus = "OK"
			addMsg(true, "vet ok")
		} else {
			vetStatus = "Issues"
			// Filter unsafe.Pointer warnings
			lines := strings.Split(vetOutput, "\n")
			var filteredLines []string
			for _, line := range lines {
				if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") { // Ignore comments/empty
					continue
				}
				if !strings.Contains(line, "possible misuse of unsafe.Pointer") {
					filteredLines = append(filteredLines, line)
				}
			}

			if len(filteredLines) > 0 {
				addMsg(false, "vet issues found")
			} else {
				vetStatus = "OK"
				addMsg(true, "vet ok")
			}
		}
	} else {
		vetStatus = "OK"
		addMsg(true, "vet ok")
	}

	// Run tests with race detection AND coverage in a single command
	// go test ./... automatically discovers all packages with tests
	var testErr error
	var testOutput string

	testCmd := exec.Command("go", "test", "-race", "-cover", "-count=1", "./...")

	testBuffer := &bytes.Buffer{}

	testFilter := NewConsoleFilter(nil)

	testPipe := &paramWriter{
		write: func(p []byte) (n int, err error) {
			s := string(p)
			testBuffer.Write(p)
			testFilter.Add(s)
			return len(p), nil
		},
	}

	testCmd.Stdout = testPipe
	testCmd.Stderr = testPipe
	testErr = testCmd.Run()
	testFilter.Flush()

	testOutput = testBuffer.String()

	// Process test results
	// Determine if any stdlib tests actually ran by looking for ok/FAIL markers in output
	hasStdOk := strings.Contains(testOutput, "\tok\t") || strings.Contains(testOutput, "ok  \t")
	hasStdFail := strings.Contains(testOutput, "\tFAIL\t") || strings.Contains(testOutput, "FAIL  \t")
	stdTestsRan := hasStdOk || hasStdFail

	if testErr != nil {
		// Check if it's strictly a WASM-only module (no std tests ran AND we see exclusions)
		isExclusionError := strings.Contains(testOutput, "matched no packages") ||
			strings.Contains(testOutput, "build constraints exclude all Go files")

		if !stdTestsRan && isExclusionError {
			testStatus = "Passing"
			raceStatus = "Clean"
			// Ensure WASM tests are enabled if we literally found no std tests due to tags
			enableWasmTests = true
			g.log("No stdlib tests matched/run (possibly WASM-only module), skipping stdlib tests...")
		} else {
			// Real test failure or partial failure with some tests actually running
			addMsg(false, fmt.Sprintf("Test errors found in %s", moduleName))
			testStatus = "Failed"
			raceStatus = "Detected"
			// Even if it failed, if some tests ran, coverage is still valid
		}
	} else {
		testStatus = "Passing"
		raceStatus = "Clean"
		addMsg(true, "tests stdlib ok")
		addMsg(true, "race detection ok")
		stdTestsRan = true
	}

	// Process coverage results (from the same test run)
	if stdTestsRan {
		coveragePercent = calculateAverageCoverage(testOutput)
		if coveragePercent != "0" {
			addMsg(true, "coverage: "+coveragePercent+"%")
		}
	}

	// WASM Tests
	if enableWasmTests {

		if err := g.installWasmBrowserTest(); err != nil {

			addMsg(false, "WASM tests skipped (setup failed)")
		} else {
			execArg := "wasmbrowsertest -quiet"
			testArgs := []string{"test", "-exec", execArg, "-cover", "./..."}
			execArg = "wasmbrowsertest"
			testArgs = []string{"test", "-exec", execArg, "-v", "-cover", "./..."}

			wasmCmd := exec.Command("go", testArgs...)
			wasmCmd.Env = os.Environ()
			wasmCmd.Env = append(wasmCmd.Env, "GOOS=js", "GOARCH=wasm")

			var wasmOut bytes.Buffer

			var wasmFilterCallback func(string)

			wasmFilter := NewConsoleFilter(wasmFilterCallback)
			wasmPipe := &paramWriter{
				write: func(p []byte) (n int, err error) {
					s := string(p)
					wasmOut.Write(p)
					wasmFilter.Add(s)
					return len(p), nil
				},
			}

			wasmCmd.Stdout = wasmPipe
			wasmCmd.Stderr = wasmPipe

			err := wasmCmd.Run()
			wasmFilter.Flush()

			wOutput := wasmOut.String()

			if err != nil {
				// WASM test failure - ConsoleFilter already filtered the output in quiet mode
				addMsg(false, "tests wasm failed")
				testStatus = "Failed"
			} else {
				addMsg(true, "tests wasm ok")
				if testStatus != "Failed" {
					testStatus = "Passing"
				}
				wCov := calculateAverageCoverage(wOutput)
				if wCov != "0" {
					// Prefer WASM coverage if stdlib had 0% (common in WASM-only packages)
					if coveragePercent == "0" {
						coveragePercent = wCov
						addMsg(true, "coverage: "+coveragePercent+"%")
					}
				}
			}
		}
	}

	// Badges

	licenseType := "MIT"
	if checkFileExists("LICENSE") {
		// naive check
	}
	goVer := getGoVersion()

	bh := NewBadges()
	bh.SetLog(g.log)
	if err := bh.updateBadges("README.md", licenseType, goVer, testStatus, coveragePercent, raceStatus, vetStatus, true); err != nil {

	}

	// Return error if tests or vet failed
	summary := strings.Join(msgs, ", ")
	if testStatus == "Failed" || vetStatus == "Issues" {
		return summary, fmt.Errorf("%s", summary)
	}

	return summary, nil
}

type paramWriter struct {
	write func(p []byte) (n int, err error)
}

func (p *paramWriter) Write(b []byte) (n int, err error) {
	return p.write(b)
}

func calculateAverageCoverage(output string) string {
	lines := strings.Split(output, "\n")
	var total float64
	var count int

	re := regexp.MustCompile(`coverage:\s+(\d+(\.\d+)?)%`)

	for _, line := range lines {
		if strings.Contains(line, "[no test files]") {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			val, err := strconv.ParseFloat(matches[1], 64)
			if err == nil && val > 0 {
				total += val
				count++
			}
		}
	}

	if count == 0 {
		return "0"
	}
	return fmt.Sprintf("%.0f", total/float64(count))
}

func (g *Go) installWasmBrowserTest() error {
	if _, err := RunCommandSilent("which", "wasmbrowsertest"); err == nil {
		return nil
	}

	_, err := RunCommand("go", "install", "github.com/tinywasm/wasmbrowsertest@latest")
	if err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	return nil
}
