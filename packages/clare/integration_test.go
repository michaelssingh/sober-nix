package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClareTUIResolutionAndMpvDryRun(t *testing.T) {
	t.Log("Building the clare binary...")
	buildCmd := exec.Command("go", "build", "-o", "clare-bin", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to compile clare binary: %v", err)
	}
	defer os.Remove("clare-bin")

	// Create temp directory for mock mpv
	tmpDir, err := os.MkdirTemp("", "clare-mock-mpv-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write mock mpv source code
	mockSource := `package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("MOCK_MPV_CALLED", os.Args)
	if len(os.Args) < 2 {
		os.Exit(0)
	}

	// The last argument is the stream URL
	streamURL := os.Args[len(os.Args)-1]

	// Find the referer and user-agent from headers arg
	referer := ""
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0"
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--http-header-fields=") {
			fields := strings.TrimPrefix(arg, "--http-header-fields=")
			parts := strings.Split(fields, ",")
			for _, p := range parts {
				kv := strings.SplitN(p, ": ", 2)
				if len(kv) == 2 {
					if strings.ToLower(kv[0]) == "referer" {
						referer = kv[1]
					} else if strings.ToLower(kv[0]) == "user-agent" {
						userAgent = kv[1]
					}
				}
			}
		}
	}

	// Validate the stream by downloading first 2MB
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", streamURL, nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		fmt.Printf("Bad status code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	buf := make([]byte, 1024)
	n, _ := io.ReadFull(resp.Body, buf)
	if n < 100 {
		fmt.Printf("Stream payload too small: %d bytes\n", n)
		os.Exit(1)
	}

	// Validate container headers
	isValid := false
	if buf[0] == 0x47 {
		isValid = true
	} else {
		inspectStr := string(buf[:n])
		if strings.Contains(inspectStr, "ftyp") || strings.Contains(inspectStr, "FLV") || strings.Contains(inspectStr, "matroska") || strings.Contains(inspectStr, "RIFF") || strings.Contains(inspectStr, "#EXTM3U") {
			isValid = true
		}
	}

	if !isValid {
		fmt.Printf("Invalid video container headers: %x\n", buf[:20])
		os.Exit(1)
	}

	fmt.Println("MOCK_MPV_VALIDATION_SUCCESS")
	os.Exit(0)
}
`

	mockGoPath := filepath.Join(tmpDir, "mock_mpv.go")
	if err := os.WriteFile(mockGoPath, []byte(mockSource), 0644); err != nil {
		t.Fatalf("Failed to write mock mpv source: %v", err)
	}

	// Compile mock mpv binary
	t.Log("Compiling the mock mpv helper...")
	compileCmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "mpv"), mockGoPath)
	if err := compileCmd.Run(); err != nil {
		t.Fatalf("Failed to compile mock mpv: %v", err)
	}

	t.Log("Fetching top 20 currently airing anime from AniList...")
	shows, err := fetchAiringAnime()
	if err != nil {
		t.Fatalf("Failed to fetch airing anime: %v", err)
	}

	t.Logf("Testing clare-bin with %d airing shows...", len(shows))
	successCount := 0

	for idx, show := range shows {
		title := show.TitleRomaji
		if title == "" {
			title = show.TitleEnglish
		}

		t.Logf("[%d/20] Testing: %q", idx+1, title)

		// Run clare-bin with mock mpv in PATH
		cmd := exec.Command("./clare-bin", "-s", title, "-e", "1", "-m", "sub")
		cmd.Env = append(os.Environ(), "PATH="+tmpDir+":"+os.Getenv("PATH"))
		
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Logf("  [SKIP] clare-bin execution returned error (e.g. no episodes/sources or bad stream): %v", err)
			continue
		}

		outStr := stdout.String()
		errStr := stderr.String()

		if strings.Contains(outStr, "MOCK_MPV_VALIDATION_SUCCESS") {
			t.Logf("  [SUCCESS] clare-bin resolved stream, invoked mpv, and validated stream download!")
			successCount++
		} else {
			t.Logf("  [INFO] clare-bin did not call mpv or validation failed. Out: %q, Err: %q", outStr, errStr)
		}
	}

	t.Logf("Clare verification complete: %d / %d shows successfully launched and validated the stream.", successCount, len(shows))
	if successCount < 5 {
		t.Fatalf("Only %d shows validated successfully (expected >= 5)", successCount)
	}
}
