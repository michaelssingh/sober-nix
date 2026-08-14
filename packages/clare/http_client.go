package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	UserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"
	StreamReferer = "https://vidsrc.net/"
)

type loggingRoundTripper struct {
	next http.RoundTripper
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	debugLog("[API-REQ] %s %s", req.Method, req.URL.String())
	resp, err := l.next.RoundTrip(req)
	if err != nil {
		debugLog("[ERROR] %s %s -> %v", req.Method, req.URL.String(), err)
		return nil, err
	}

	debugLog("[API-RESP] %s %s -> %d", req.Method, req.URL.String(), resp.StatusCode)
	return resp, nil
}

func newLoggingHttpClient(timeout time.Duration) *http.Client {
	baseTransport := http.DefaultTransport
	if baseTransport == nil {
		baseTransport = &http.Transport{}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: &loggingRoundTripper{next: baseTransport},
	}
}

func doHTTPReqWithRetry(method, reqURL string, payload []byte, headers map[string]string) ([]byte, error) {
	var body []byte
	var err error
	client := newLoggingHttpClient(10 * time.Second)

	_ = os.Getenv("CLARE_COOKIE")
	maxAttempts := 4
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var req *http.Request
		if len(payload) > 0 {
			req, err = http.NewRequest(method, reqURL, bytes.NewReader(payload))
		} else {
			req, err = http.NewRequest(method, reqURL, nil)
		}
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", UserAgent)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		var resp *http.Response
		resp, err = client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			body, err = io.ReadAll(resp.Body)
			if err == nil {
				if resp.StatusCode != http.StatusOK {
					bodySnippet := string(body)
					if len(bodySnippet) > 300 {
						bodySnippet = bodySnippet[:300] + "..."
					}
					debugLog("[API-RESP-ERR] Status %d on %s | Payload: %s", resp.StatusCode, reqURL, bodySnippet)
				}

				isTransientError := (resp.StatusCode >= 500 && resp.StatusCode < 600) || resp.StatusCode == 429
				bodyStr := string(body)
				if !isTransientError && (strings.Contains(bodyStr, "error code: 5") || strings.Contains(bodyStr, "Too many requests")) {
					isTransientError = true
					sleepSec := 3
					reSec := regexp.MustCompile(`try again in (\d+) second`)
					if m := reSec.FindStringSubmatch(bodyStr); len(m) >= 2 {
						if parsedSec, err := strconv.Atoi(m[1]); err == nil && parsedSec > 0 {
							sleepSec = parsedSec + 1
						}
					}
					debugLog("[API-RATE-LIMIT] Rate limit on %s. Sleeping %d seconds before retry (attempt %d/%d)...", reqURL, sleepSec, attempt, maxAttempts)
					time.Sleep(time.Duration(sleepSec) * time.Second)
				}

				if !isTransientError {
					return body, nil
				}
				err = fmt.Errorf("transient HTTP error (status %d): %s", resp.StatusCode, strings.TrimSpace(bodyStr))
			}
		}

		debugLog("HTTP request attempt %d/%d failed for %s: %v", attempt, maxAttempts, reqURL, err)
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	return nil, fmt.Errorf("after %d attempts, request failed: %w", maxAttempts, err)
}
