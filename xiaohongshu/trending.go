package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

// TrendingAction 热点获取
type TrendingAction struct {
	page *rod.Page
}

// NewTrendingAction 创建热点获取实例
func NewTrendingAction(page *rod.Page) *TrendingAction {
	return &TrendingAction{page: page}
}

// GetHotSearches 获取热搜榜
func (a *TrendingAction) GetHotSearches(ctx context.Context) (*TrendingResponse, error) {
	logrus.Info("开始获取小红书热搜榜")

	// 绑定context，确保超时能正确传递
	a.page = a.page.Context(ctx)

	// 访问小红书首页
	if err := a.page.Navigate("https://www.xiaohongshu.com"); err != nil {
		return nil, fmt.Errorf("navigate to home failed: %w", err)
	}

	if err := a.page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load failed: %w", err)
	}

	// 等待页面加载
	time.Sleep(2 * time.Second)

	// 尝试点击搜索框触发热搜列表
	searchInput, _ := a.page.Timeout(5 * time.Second).Element(".search-input")
	if searchInput == nil {
		searchInput, _ = a.page.Timeout(5 * time.Second).Element("input[placeholder*='搜索']")
	}

	if searchInput != nil {
		searchInput.MustClick()
		time.Sleep(1 * time.Second)
	}

	// 获取热搜数据
	hotSearches, err := a.parseHotSearches()
	if err != nil {
		return nil, fmt.Errorf("parse hot searches failed: %w", err)
	}

	return &TrendingResponse{
		HotSearches: hotSearches,
		UpdatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// parseHotSearches 解析热搜数据
func (a *TrendingAction) parseHotSearches() ([]HotSearch, error) {
	var hotSearches []HotSearch

	// 尝试多个可能的选择器
	selectors := []string{
		".hot-list .hot-item",
		".search-hot-list .hot-item",
		".trending-list .trending-item",
		"[class*='hot'] [class*='item']",
	}

	var items rod.Elements
	for _, selector := range selectors {
		elements, _ := a.page.Timeout(3 * time.Second).Elements(selector)
		if elements != nil && len(elements) > 0 {
			items = elements
			logrus.Infof("找到热搜列表，选择器: %s, 数量: %d", selector, len(items))
			break
		}
	}

	if len(items) == 0 {
		logrus.Warn("未找到热搜列表，尝试从页面脚本中提取")
		return a.parseHotSearchesFromScript()
	}

	// 解析每个热搜项
	for i, item := range items {
		if i >= 20 { // 只取前20条
			break
		}

		keyword := ""
		label := ""
		heat := ""
		url := ""

		// 获取关键词
		keywordEl, _ := item.Timeout(1 * time.Second).Element(".keyword")
		if keywordEl == nil {
			keywordEl, _ = item.Timeout(1 * time.Second).Element(".title")
		}
		if keywordEl == nil {
			keywordEl, _ = item.Timeout(1 * time.Second).Element("span")
		}
		if keywordEl != nil {
			keyword = keywordEl.MustText()
		}

		// 获取标签
		labelEl, _ := item.Timeout(1 * time.Second).Element(".label")
		if labelEl == nil {
			labelEl, _ = item.Timeout(1 * time.Second).Element(".tag")
		}
		if labelEl != nil {
			label = labelEl.MustText()
		}

		// 获取热度
		heatEl, _ := item.Timeout(1 * time.Second).Element(".heat")
		if heatEl == nil {
			heatEl, _ = item.Timeout(1 * time.Second).Element(".count")
		}
		if heatEl != nil {
			heat = heatEl.MustText()
		}

		// 获取链接
		linkEl, _ := item.Timeout(1 * time.Second).Element("a")
		if linkEl != nil {
			href, _ := linkEl.Property("href")
			url = href.String()
		}

		if keyword != "" {
			hotSearches = append(hotSearches, HotSearch{
				Rank:    i + 1,
				Keyword: keyword,
				Heat:    heat,
				Label:   label,
				URL:     url,
			})
		}
	}

	logrus.Infof("成功解析 %d 条热搜", len(hotSearches))
	return hotSearches, nil
}

// parseHotSearchesFromScript 从页面脚本中提取热搜数据（备用方案）
func (a *TrendingAction) parseHotSearchesFromScript() ([]HotSearch, error) {
	// 尝试从 window.__INITIAL_STATE__ 中获取数据
	js := `
		(() => {
			try {
				const state = window.__INITIAL_STATE__;
				if (state && state.search && state.search.hotSearches) {
					return state.search.hotSearches;
				}
				// 备用方案：尝试其他可能的路径
				if (state && state.trending) {
					return state.trending.list || state.trending.items;
				}
				return [];
			} catch (e) {
				return [];
			}
		})()
	`

	result, err := a.page.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("eval script failed: %w", err)
	}

	// 将结果转换为热搜列表
	// 这里需要根据实际返回的数据结构进行调整
	logrus.Debug("从脚本获取到的热搜数据:", result.Value)

	// 简化处理：如果无法从脚本获取，返回空列表
	return []HotSearch{}, nil
}

// GetTrendingTopics 获取热门话题（发现页）
func (a *TrendingAction) GetTrendingTopics(ctx context.Context, category string) (*TopicsResponse, error) {
	logrus.Infof("开始获取热门话题，分类: %s", category)

	// 访问发现页
	url := "https://www.xiaohongshu.com/explore"
	if category != "" {
		url = fmt.Sprintf("%s?category=%s", url, category)
	}

	if err := a.page.Navigate(url); err != nil {
		return nil, fmt.Errorf("navigate to explore failed: %w", err)
	}

	if err := a.page.WaitLoad(); err != nil {
		return nil, fmt.Errorf("wait load failed: %w", err)
	}

	time.Sleep(2 * time.Second)

	// 解析话题列表
	topics, err := a.parseTopics()
	if err != nil {
		return nil, fmt.Errorf("parse topics failed: %w", err)
	}

	return &TopicsResponse{
		Topics:    topics,
		Category:  category,
		UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// parseTopics 解析话题数据
func (a *TrendingAction) parseTopics() ([]Topic, error) {
	var topics []Topic

	// 尝试多个可能的选择器
	selectors := []string{
		".topic-card",
		".channel-item",
		".explore-item",
		"[class*='topic']",
	}

	var cards rod.Elements
	for _, selector := range selectors {
		elements, _ := a.page.Timeout(3 * time.Second).Elements(selector)
		if elements != nil && len(elements) > 0 {
			cards = elements
			logrus.Infof("找到话题列表，选择器: %s, 数量: %d", selector, len(cards))
			break
		}
	}

	if len(cards) == 0 {
		logrus.Warn("未找到话题列表")
		return topics, nil
	}

	// 解析每个话题卡片
	for _, card := range cards {
		name := ""
		desc := ""
		viewCount := ""
		postCount := ""
		coverURL := ""
		isHot := false
		topicID := ""

		// 话题名称
		nameEl, _ := card.Timeout(1 * time.Second).Element(".topic-name")
		if nameEl == nil {
			nameEl, _ = card.Timeout(1 * time.Second).Element(".title")
		}
		if nameEl != nil {
			name = nameEl.MustText()
		}

		// 话题描述
		descEl, _ := card.Timeout(1 * time.Second).Element(".desc")
		if descEl == nil {
			descEl, _ = card.Timeout(1 * time.Second).Element(".description")
		}
		if descEl != nil {
			desc = descEl.MustText()
		}

		// 浏览量
		viewEl, _ := card.Timeout(1 * time.Second).Element(".view-count")
		if viewEl == nil {
			viewEl, _ = card.Timeout(1 * time.Second).Element(".views")
		}
		if viewEl != nil {
			viewCount = viewEl.MustText()
		}

		// 参与人数
		postEl, _ := card.Timeout(1 * time.Second).Element(".post-count")
		if postEl == nil {
			postEl, _ = card.Timeout(1 * time.Second).Element(".posts")
		}
		if postEl != nil {
			postCount = postEl.MustText()
		}

		// 封面图
		imgEl, _ := card.Timeout(1 * time.Second).Element("img")
		if imgEl != nil {
			src, _ := imgEl.Property("src")
			coverURL = src.String()
		}

		// 是否热门
		hotTag, _ := card.Timeout(1 * time.Second).Element(".hot-tag")
		if hotTag == nil {
			hotTag, _ = card.Timeout(1 * time.Second).Element(".icon-hot")
		}
		isHot = hotTag != nil

		// 话题ID（从链接提取）
		linkEl, _ := card.Timeout(1 * time.Second).Element("a")
		if linkEl != nil {
			href, _ := linkEl.Property("href")
			// 从 URL 中提取 topic ID
			url := href.String()
			// 简单提取最后一段
			if len(url) > 0 {
				parts := splitURL(url)
				if len(parts) > 0 {
					topicID = parts[len(parts)-1]
				}
			}
		}

		if name != "" {
			topics = append(topics, Topic{
				TopicID:     topicID,
				Name:        name,
				Description: desc,
				ViewCount:   viewCount,
				PostCount:   postCount,
				CoverURL:    coverURL,
				IsHot:       isHot,
			})
		}
	}

	logrus.Infof("成功解析 %d 个话题", len(topics))
	return topics, nil
}

// splitURL 简单的 URL 分割函数
func splitURL(url string) []string {
	var parts []string
	current := ""
	for _, ch := range url {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
