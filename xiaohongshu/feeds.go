package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/xpzouying/xiaohongshu-mcp/errors"
)

type FeedsListAction struct {
	page *rod.Page
}

func NewFeedsListAction(page *rod.Page) *FeedsListAction {
	pp := page.Timeout(60 * time.Second)

	pp.MustNavigate("https://www.xiaohongshu.com")
	pp.MustWaitDOMStable()

	return &FeedsListAction{page: pp}
}

// GetFeedsList 获取页面的 Feed 列表数据
func (f *FeedsListAction) GetFeedsList(ctx context.Context) ([]Feed, error) {
	page := f.page.Context(ctx)

	time.Sleep(3 * time.Second) // 增加等待时间

	// 先检查 feeds 对象的结构
	feedsInfo := page.MustEval(`() => {
		if (!window.__INITIAL_STATE__) return "no __INITIAL_STATE__";
		if (!window.__INITIAL_STATE__.feed) return "no feed";
		if (!window.__INITIAL_STATE__.feed.feeds) return "no feeds";

		const feeds = window.__INITIAL_STATE__.feed.feeds;
		// 尝试直接返回 feeds（如果它本身就是数组）
		if (Array.isArray(feeds)) {
			return JSON.stringify(feeds);
		}
		// 尝试 value 和 _value
		const feedsData = feeds.value !== undefined ? feeds.value : feeds._value;
		if (feedsData) {
			return JSON.stringify(feedsData);
		}
		return "feeds exists but no data: " + JSON.stringify(Object.keys(feeds));
	}`).String()

	if feedsInfo == "" || feedsInfo == "no __INITIAL_STATE__" ||
		feedsInfo == "no feed" || feedsInfo == "no feeds" ||
		strings.HasPrefix(feedsInfo, "feeds exists but no data") {
		return nil, fmt.Errorf("%w: %s", errors.ErrNoFeeds, feedsInfo)
	}

	var feeds []Feed
	if err := json.Unmarshal([]byte(feedsInfo), &feeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal feeds: %w (data: %s)", err, feedsInfo[:min(100, len(feedsInfo))])
	}

	return feeds, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
