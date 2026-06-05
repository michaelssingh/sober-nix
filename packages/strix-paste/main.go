package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func printHelp() {
	fmt.Fprintf(os.Stderr, `strix-paste: A CLI helper to upload content to the sober-strix pastebin.

Usage:
  strix-paste [options] [file]
  cat file.txt | strix-paste [options]

Options:
  -c, --clipboard    Read content from system clipboard (using wl-paste)
  -u, --url <url>    Override server URL (default: STRIX_SERVER_URL or https://sober-strix.fly.dev)
  -t, --token <tok>  Override auth token (default: STRIX_AUTH_TOKEN or ~/.config/strix/token)
  -n, --no-copy      Do not copy the resulting URL back to system clipboard
  -h, --help         Show this help message

Examples:
  strix-paste log.txt
  cat error.log | strix-paste
  strix-paste -c
`)
}

func getAuthToken(flagToken string) (string, error) {
	if flagToken != "" {
		return flagToken, nil
	}
	if tok := os.Getenv("STRIX_AUTH_TOKEN"); tok != "" {
		return tok, nil
	}
	// Try ~/.config/strix/token
	home, err := os.UserHomeDir()
	if err == nil {
		tokenPath := filepath.Join(home, ".config", "strix", "token")
		if data, err := os.ReadFile(tokenPath); err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}
	return "", fmt.Errorf("auth token not found (set STRIX_AUTH_TOKEN or write it to ~/.config/strix/token)")
}

func getServerURL(flagURL string) string {
	if flagURL != "" {
		return flagURL
	}
	if url := os.Getenv("STRIX_SERVER_URL"); url != "" {
		return url
	}
	return "https://sober-strix.fly.dev"
}

func readClipboard() ([]byte, error) {
	cmd := exec.Command("wl-paste", "--no-newline")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run wl-paste: %w (make sure you are in a Wayland session)", err)
	}
	return out.Bytes(), nil
}

func copyToClipboard(content string) error {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func main() {
	// Custom flag parsing to support both single and double dash
	var (
		clipboard bool
		noCopy    bool
		urlStr    string
		tokenStr  string
		help      bool
	)

	flag.BoolVar(&clipboard, "c", false, "Read from clipboard")
	flag.BoolVar(&clipboard, "clipboard", false, "Read from clipboard")
	flag.BoolVar(&noCopy, "n", false, "Do not copy resulting URL")
	flag.BoolVar(&noCopy, "no-copy", false, "Do not copy resulting URL")
	flag.StringVar(&urlStr, "u", "", "Server URL")
	flag.StringVar(&urlStr, "url", "", "Server URL")
	flag.StringVar(&tokenStr, "t", "", "Auth token")
	flag.StringVar(&tokenStr, "token", "", "Auth token")
	flag.BoolVar(&help, "h", false, "Show help")
	flag.BoolVar(&help, "help", false, "Show help")

	flag.Usage = printHelp
	flag.Parse()

	if help {
		printHelp()
		os.Exit(0)
	}

	var contentReader io.Reader
	var uploadFilename string

	// 1. Resolve content source
	args := flag.Args()
	if clipboard {
		data, err := readClipboard()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		contentReader = bytes.NewReader(data)
		uploadFilename = "clipboard.txt"
	} else if len(args) > 0 {
		// Read file path
		filePath := args[0]
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		contentReader = file
		uploadFilename = filepath.Base(filePath)
	} else {
		// Check if stdin has data
		stat, err := os.Stdin.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error stating stdin: %v\n", err)
			os.Exit(1)
		}
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			contentReader = os.Stdin
			uploadFilename = "stdin.txt"
		} else {
			printHelp()
			os.Exit(1)
		}
	}

	// 2. Resolve credentials & server
	token, err := getAuthToken(tokenStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	serverURL := getServerURL(urlStr)
	if !strings.HasSuffix(serverURL, "/") {
		serverURL += "/"
	}

	// 3. Build multipart request body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", uploadFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating multipart form: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.Copy(part, contentReader); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading content: %v\n", err)
		os.Exit(1)
	}
	if err := writer.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing multipart writer: %v\n", err)
		os.Exit(1)
	}

	// 4. Send HTTP request
	req, err := http.NewRequest("POST", serverURL, &body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request to %s: %v\n", serverURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading server response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Upload failed (HTTP %d): %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}

	resultingURL := strings.TrimSpace(string(respBody))
	fmt.Println(resultingURL)

	// 5. Copy back to clipboard
	if !noCopy {
		if err := copyToClipboard(resultingURL); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not copy URL to clipboard: %v\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "(URL copied to clipboard)")
		}
	}
}
