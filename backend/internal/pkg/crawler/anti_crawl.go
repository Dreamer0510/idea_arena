package crawler

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type crawlerFetchOptions struct {
	AcceptLanguage string
	Referer        string
	MaxRetries     int
	MinBackoff     time.Duration
	MaxBackoff     time.Duration
	DetectAntiBot  bool
}

var crawlerUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0",
}

var crawlerAcceptLanguages = []string{
	"zh-CN,zh;q=0.9,en;q=0.8",
	"zh-CN,zh;q=0.8,en-US;q=0.7,en;q=0.6",
	"en-US,en;q=0.9,zh-CN;q=0.5",
}

func newCrawlerHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 8 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func fetchHTMLWithRetry(
	ctx context.Context,
	client *http.Client,
	targetURL string,
	options crawlerFetchOptions,
) (string, error) {
	maxRetries := options.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	minBackoff := options.MinBackoff
	if minBackoff <= 0 {
		minBackoff = 250 * time.Millisecond
	}
	maxBackoff := options.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 1200 * time.Millisecond
	}

	var lastError error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			waitDuration := randomBackoff(minBackoff, maxBackoff)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(waitDuration):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", randomUserAgent())
		if options.AcceptLanguage != "" {
			req.Header.Set("Accept-Language", options.AcceptLanguage)
		} else {
			req.Header.Set("Accept-Language", randomAcceptLanguage())
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Pragma", "no-cache")
		req.Header.Set("DNT", "1")
		req.Header.Set("Upgrade-Insecure-Requests", "1")
		if options.Referer != "" {
			req.Header.Set("Referer", options.Referer)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastError = err
			continue
		}

		bodyBytes, readError := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readError != nil {
			lastError = readError
			continue
		}

		body := string(bodyBytes)
		if shouldRetryStatusCode(resp.StatusCode) {
			lastError = fmt.Errorf("temporary status %d", resp.StatusCode)
			continue
		}
		if options.DetectAntiBot && looksLikeAntiBotPage(body, targetURL) {
			lastError = fmt.Errorf("anti-bot page detected")
			continue
		}
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("http status %d", resp.StatusCode)
		}
		return body, nil
	}

	if lastError != nil {
		return "", lastError
	}
	return "", fmt.Errorf("request failed")
}

func randomUserAgent() string {
	return crawlerUserAgents[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(crawlerUserAgents))]
}

func randomAcceptLanguage() string {
	return crawlerAcceptLanguages[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(crawlerAcceptLanguages))]
}

func randomBackoff(minimum time.Duration, maximum time.Duration) time.Duration {
	if maximum <= minimum {
		return minimum
	}
	delta := maximum - minimum
	randomDelta := rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(int64(delta))
	return minimum + time.Duration(randomDelta)
}

func shouldRetryStatusCode(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout {
		return true
	}
	return statusCode >= 500
}

func looksLikeAntiBotPage(body string, targetURL string) bool {
	content := strings.ToLower(body)
	indicators := []string{
		"captcha",
		"verify you are human",
		"unusual traffic",
		"access denied",
		"robot check",
		"wappass.baidu.com/static/captcha",
		"请完成安全验证",
		"访问受限",
		"异常流量",
	}
	for _, indicator := range indicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	// Bing 在某些环境会循环重定向到 Object moved 页面
	if strings.Contains(content, "<title>object moved</title>") &&
		(strings.Contains(targetURL, "bing.com") || strings.Contains(targetURL, "cn.bing.com")) {
		return true
	}
	return false
}
