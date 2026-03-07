package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"idea_arena/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// compile-time check: Pojie52Plugin implements biz.CrawlerPluginInterface
var _ biz.CrawlerPluginInterface = (*Pojie52Plugin)(nil)

// Pojie52Plugin 52pojie.cn 热门帖子爬虫
type Pojie52Plugin struct {
	client *http.Client
	log    *log.Helper
}

// NewPojie52Plugin 创建 52pojie 爬虫插件
func NewPojie52Plugin(logger log.Logger) *Pojie52Plugin {
	return &Pojie52Plugin{
		client: &http.Client{Timeout: 20 * time.Second},
		log:    log.NewHelper(logger),
	}
}

func (p *Pojie52Plugin) Name() string  { return "52pojie" }
func (p *Pojie52Plugin) Label() string { return "吾爱破解 - 热门帖子" }

func (p *Pojie52Plugin) Fetch(ctx context.Context, limit int) ([]*biz.RawTopic, error) {
	url := "https://www.52pojie.cn/forum.php?mod=guide&view=hot"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("52pojie request build error: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("52pojie fetch error: %w", err)
	}
	defer resp.Body.Close()

	// 52pojie 使用 GBK 编码，需要转为 UTF-8
	utf8Reader := transform.NewReader(resp.Body, simplifiedchinese.GBK.NewDecoder())
	body, err := io.ReadAll(utf8Reader)
	if err != nil {
		return nil, fmt.Errorf("52pojie read body error: %w", err)
	}

	return p.parseHotThreads(string(body), limit), nil
}

// parseHotThreads 解析 52pojie 热门帖子列表
// 实际 HTML 结构（Discuz! 热门导读）：
//   <tbody id="normalthread_2093204">
//   <tr>
//     <th class="common">
//       <a href="thread-2093204-1-1.html" ... class="xst" >标题</a>
//     </th>
//     <td class="num"><a ...>回复数</a><em>浏览数</em></td>
//   </tr>
//   </tbody>
func (p *Pojie52Plugin) parseHotThreads(html string, limit int) []*biz.RawTopic {
	var topics []*biz.RawTopic

	// 按 tbody 分割，每个 normalthread 是一个帖子块
	rowRe := regexp.MustCompile(`(?s)<tbody[^>]*id="normalthread_(\d+)"[^>]*>(.*?)</tbody>`)
	// 标题链接：<a href="thread-XXX-1-1.html" ... class="xst" ...>标题</a>
	titleRe := regexp.MustCompile(`(?s)<a[^>]*href="([^"]*thread-\d+-[^"]*)"[^>]*class="xst"[^>]*>(.*?)</a>`)
	// 回复数和浏览数：<td class="num"><a ...>回复</a><em>浏览</em></td>
	numRe := regexp.MustCompile(`(?s)<td[^>]*class="num"[^>]*>.*?<a[^>]*>(\d+)</a>.*?<em>(\d+)</em>`)

	rows := rowRe.FindAllStringSubmatch(html, limit*3)
	for _, row := range rows {
		if len(topics) >= limit {
			break
		}

		block := row[2]
		titleMatch := titleRe.FindStringSubmatch(block)
		if titleMatch == nil {
			continue
		}
		href := titleMatch[1]
		title := stripHTML(titleMatch[2])
		if title == "" {
			continue
		}

		if !strings.HasPrefix(href, "http") {
			href = "https://www.52pojie.cn/" + href
		}

		replies := 0
		views := 0
		numMatch := numRe.FindStringSubmatch(block)
		if numMatch != nil {
			replies, _ = strconv.Atoi(numMatch[1])
			views, _ = strconv.Atoi(numMatch[2])
		}

		topics = append(topics, &biz.RawTopic{
			Title:      title,
			URL:        href,
			Source:     "52pojie",
			Popularity: views,
			Replies:    replies,
			Snippet:    fmt.Sprintf("浏览: %d, 回复: %d", views, replies),
		})
	}

	return topics
}

// stripHTML 移除 HTML 标签
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}
