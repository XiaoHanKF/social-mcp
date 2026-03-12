package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// MCP 工具处理函数（全部增加 TenantContext 参数）

// handleCheckLoginStatus 处理检查登录状态
func (s *AppServer) handleCheckLoginStatus(ctx context.Context, tc TenantContext) *MCPToolResult {
	logrus.Infof("MCP: 检查登录状态 [tenant=%s, app=%s]", tc.TenantID, tc.AppID)

	status, err := s.xiaohongshuService.CheckLoginStatus(ctx, tc)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "检查登录状态失败: " + err.Error()}},
			IsError: true,
		}
	}

	var resultText string
	if status.IsLoggedIn {
		resultText = fmt.Sprintf("✅ 已登录\n用户名: %s\n\n你可以使用其他功能了。", status.Username)
	} else {
		resultText = fmt.Sprintf("❌ 未登录\n\n请使用 get_login_qrcode 工具获取二维码进行登录。")
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText}},
	}
}

// handleGetLoginQrcode 处理获取登录二维码请求
func (s *AppServer) handleGetLoginQrcode(ctx context.Context, tc TenantContext) *MCPToolResult {
	logrus.Infof("MCP: 获取登录扫码图片 [tenant=%s, app=%s]", tc.TenantID, tc.AppID)

	result, err := s.xiaohongshuService.GetLoginQrcode(ctx, tc)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取登录扫码图片失败: " + err.Error()}},
			IsError: true,
		}
	}

	if result.IsLoggedIn {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "你当前已处于登录状态"}},
		}
	}

	now := time.Now()
	deadline := func() string {
		d, err := time.ParseDuration(result.Timeout)
		if err != nil {
			return now.Format("2006-01-02 15:04:05")
		}
		return now.Add(d).Format("2006-01-02 15:04:05")
	}()

	contents := []MCPContent{
		{Type: "text", Text: "请用小红书 App 在 " + deadline + " 前扫码登录 👇"},
		{
			Type:     "image",
			MimeType: "image/png",
			Data:     strings.TrimPrefix(result.Img, "data:image/png;base64,"),
		},
	}
	return &MCPToolResult{Content: contents}
}

// handleDeleteCookies 处理删除 cookies 请求
func (s *AppServer) handleDeleteCookies(ctx context.Context, tc TenantContext) *MCPToolResult {
	logrus.Infof("MCP: 删除 cookies [tenant=%s, app=%s]", tc.TenantID, tc.AppID)

	err := s.xiaohongshuService.DeleteCookies(ctx, tc)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "删除 cookies 失败: " + err.Error()}},
			IsError: true,
		}
	}

	cookiePath := tc.CookiesFilePath()
	resultText := fmt.Sprintf("Cookies 已成功删除，登录状态已重置。\n\n删除的文件路径: %s\n\n下次操作时，需要重新登录。", cookiePath)
	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: resultText}},
	}
}

// handlePublishContent 处理发布内容
func (s *AppServer) handlePublishContent(ctx context.Context, args PublishContentArgs) *MCPToolResult {
	logrus.Infof("MCP: 发布内容 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	req := &PublishRequest{
		Title:      args.Title,
		Content:    args.Content,
		Images:     args.Images,
		Tags:       args.Tags,
		ScheduleAt: args.ScheduleAt,
		IsOriginal: args.IsOriginal,
		Visibility: args.Visibility,
		Products:   args.Products,
	}

	logrus.Infof("MCP: 发布内容 - 标题: %s, 图片数量: %d, 标签数量: %d, 定时: %s, 原创: %v, visibility: %s, 商品: %v",
		req.Title, len(req.Images), len(req.Tags), req.ScheduleAt, req.IsOriginal, req.Visibility, req.Products)

	result, err := s.xiaohongshuService.PublishContent(ctx, args.TenantContext, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发布失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("内容发布成功: %+v", result)}},
	}
}

// handlePublishVideo 处理发布视频内容
func (s *AppServer) handlePublishVideo(ctx context.Context, args PublishVideoArgs) *MCPToolResult {
	logrus.Infof("MCP: 发布视频内容 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.Video == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发布失败: 缺少本地视频文件路径"}},
			IsError: true,
		}
	}

	req := &PublishVideoRequest{
		Title:      args.Title,
		Content:    args.Content,
		Video:      args.Video,
		Tags:       args.Tags,
		ScheduleAt: args.ScheduleAt,
		Visibility: args.Visibility,
		Products:   args.Products,
	}

	logrus.Infof("MCP: 发布视频 - 标题: %s, 标签数量: %d, 定时: %s, visibility: %s, 商品: %v",
		req.Title, len(req.Tags), req.ScheduleAt, req.Visibility, req.Products)

	result, err := s.xiaohongshuService.PublishVideo(ctx, args.TenantContext, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发布失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("视频发布成功: %+v", result)}},
	}
}

// handleListFeeds 处理获取Feeds列表
func (s *AppServer) handleListFeeds(ctx context.Context, tc TenantContext) *MCPToolResult {
	logrus.Infof("MCP: 获取Feeds列表 [tenant=%s, app=%s]", tc.TenantID, tc.AppID)

	result, err := s.xiaohongshuService.ListFeeds(ctx, tc)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取Feeds列表失败: " + err.Error()}},
			IsError: true,
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("获取Feeds列表成功，但序列化失败: %v", err)}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(jsonData)}},
	}
}

// handleSearchFeeds 处理搜索Feeds
func (s *AppServer) handleSearchFeeds(ctx context.Context, args SearchFeedsArgs) *MCPToolResult {
	logrus.Infof("MCP: 搜索Feeds [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.Keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "搜索Feeds失败: 缺少关键词参数"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 搜索Feeds - 关键词: %s", args.Keyword)

	filter := xiaohongshu.FilterOption{
		SortBy:      args.Filters.SortBy,
		NoteType:    args.Filters.NoteType,
		PublishTime: args.Filters.PublishTime,
		SearchScope: args.Filters.SearchScope,
		Location:    args.Filters.Location,
	}

	result, err := s.xiaohongshuService.SearchFeeds(ctx, args.TenantContext, args.Keyword, filter)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "搜索Feeds失败: " + err.Error()}},
			IsError: true,
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("搜索Feeds成功，但序列化失败: %v", err)}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(jsonData)}},
	}
}

// handleGetFeedDetail 处理获取Feed详情
func (s *AppServer) handleGetFeedDetail(ctx context.Context, args FeedDetailArgs) *MCPToolResult {
	logrus.Infof("MCP: 获取Feed详情 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.FeedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取Feed详情失败: 缺少feed_id参数"}},
			IsError: true,
		}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取Feed详情失败: 缺少xsec_token参数"}},
			IsError: true,
		}
	}

	config := xiaohongshu.DefaultCommentLoadConfig()

	if args.LoadAllComments {
		config.ClickMoreReplies = args.ClickMoreReplies

		limit := args.Limit
		if limit <= 0 {
			limit = 20
		}
		config.MaxCommentItems = limit

		replyLimit := args.ReplyLimit
		if replyLimit <= 0 {
			replyLimit = 10
		}
		config.MaxRepliesThreshold = replyLimit

		if args.ScrollSpeed != "" {
			config.ScrollSpeed = args.ScrollSpeed
		}
	}

	logrus.Infof("MCP: 获取Feed详情 - Feed ID: %s, loadAllComments=%v, config=%+v", args.FeedID, args.LoadAllComments, config)

	result, err := s.xiaohongshuService.GetFeedDetailWithConfig(ctx, args.TenantContext, args.FeedID, args.XsecToken, args.LoadAllComments, config)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取Feed详情失败: " + err.Error()}},
			IsError: true,
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("获取Feed详情成功，但序列化失败: %v", err)}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(jsonData)}},
	}
}

// handleUserProfile 获取用户主页
func (s *AppServer) handleUserProfile(ctx context.Context, args UserProfileArgs) *MCPToolResult {
	logrus.Infof("MCP: 获取用户主页 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.UserID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取用户主页失败: 缺少user_id参数"}},
			IsError: true,
		}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取用户主页失败: 缺少xsec_token参数"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 获取用户主页 - User ID: %s", args.UserID)

	result, err := s.xiaohongshuService.UserProfile(ctx, args.TenantContext, args.UserID, args.XsecToken)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取用户主页失败: " + err.Error()}},
			IsError: true,
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("获取用户主页，但序列化失败: %v", err)}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(jsonData)}},
	}
}

// handleLikeFeed 处理点赞/取消点赞
func (s *AppServer) handleLikeFeed(ctx context.Context, args LikeFeedArgs) *MCPToolResult {
	logrus.Infof("MCP: 点赞操作 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.FeedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}

	var res *ActionResult
	var err error

	if args.Unlike {
		res, err = s.xiaohongshuService.UnlikeFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken)
	} else {
		res, err = s.xiaohongshuService.LikeFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken)
	}

	if err != nil {
		action := "点赞"
		if args.Unlike {
			action = "取消点赞"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}

	action := "点赞"
	if args.Unlike {
		action = "取消点赞"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handleFavoriteFeed 处理收藏/取消收藏
func (s *AppServer) handleFavoriteFeed(ctx context.Context, args FavoriteFeedArgs) *MCPToolResult {
	logrus.Infof("MCP: 收藏操作 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.FeedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}

	var res *ActionResult
	var err error

	if args.Unfavorite {
		res, err = s.xiaohongshuService.UnfavoriteFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken)
	} else {
		res, err = s.xiaohongshuService.FavoriteFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken)
	}

	if err != nil {
		action := "收藏"
		if args.Unfavorite {
			action = "取消收藏"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}

	action := "收藏"
	if args.Unfavorite {
		action = "取消收藏"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handlePostComment 处理发表评论到Feed
func (s *AppServer) handlePostComment(ctx context.Context, args PostCommentArgs) *MCPToolResult {
	logrus.Infof("MCP: 发表评论 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.FeedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发表评论失败: 缺少feed_id参数"}},
			IsError: true,
		}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发表评论失败: 缺少xsec_token参数"}},
			IsError: true,
		}
	}
	if args.Content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发表评论失败: 缺少content参数"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 发表评论 - Feed ID: %s, 内容长度: %d", args.FeedID, len(args.Content))

	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken, args.Content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "发表评论失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("评论发表成功 - Feed ID: %s", result.FeedID)}},
	}
}

// handleReplyComment 处理回复评论
func (s *AppServer) handleReplyComment(ctx context.Context, args ReplyCommentArgs) *MCPToolResult {
	logrus.Infof("MCP: 回复评论 [tenant=%s, app=%s]", args.TenantID, args.AppID)

	if args.FeedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "回复评论失败: 缺少feed_id参数"}},
			IsError: true,
		}
	}
	if args.XsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "回复评论失败: 缺少xsec_token参数"}},
			IsError: true,
		}
	}
	if args.CommentID == "" && args.UserID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "回复评论失败: 缺少comment_id或user_id参数"}},
			IsError: true,
		}
	}
	if args.Content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "回复评论失败: 缺少content参数"}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 回复评论 - Feed ID: %s, Comment ID: %s, User ID: %s, 内容长度: %d",
		args.FeedID, args.CommentID, args.UserID, len(args.Content))

	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, args.TenantContext, args.FeedID, args.XsecToken, args.CommentID, args.UserID, args.Content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "回复评论失败: " + err.Error()}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("评论回复成功 - Feed ID: %s, Comment ID: %s, User ID: %s",
			result.FeedID, result.TargetCommentID, result.TargetUserID)}},
	}
}

// handleGetHotSearches 获取热搜榜
func (s *AppServer) handleGetHotSearches(ctx context.Context, tc TenantContext) *MCPToolResult {
	logrus.Infof("MCP: 获取热搜榜 [tenant=%s, app=%s]", tc.TenantID, tc.AppID)

	result, err := s.xiaohongshuService.GetHotSearches(ctx, tc)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取热搜失败: " + err.Error()}},
			IsError: true,
		}
	}

	var textOutput strings.Builder
	textOutput.WriteString(fmt.Sprintf("📈 小红书热搜榜 (更新时间: %s)\n\n", result.UpdatedAt))

	if len(result.HotSearches) == 0 {
		textOutput.WriteString("暂无热搜数据\n")
	} else {
		for _, hs := range result.HotSearches {
			label := ""
			if hs.Label != "" {
				label = fmt.Sprintf(" [%s]", hs.Label)
			}
			heat := ""
			if hs.Heat != "" {
				heat = fmt.Sprintf(" 🔥 %s", hs.Heat)
			}
			textOutput.WriteString(fmt.Sprintf("%d. %s%s%s\n", hs.Rank, hs.Keyword, label, heat))
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: textOutput.String() + "\n详细数据序列化失败"}},
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: textOutput.String()},
			{Type: "text", Text: "\n📊 详细数据(JSON):\n" + string(jsonData)},
		},
	}
}

// handleGetTrendingTopics 获取热门话题
func (s *AppServer) handleGetTrendingTopics(ctx context.Context, tc TenantContext, category string) *MCPToolResult {
	logrus.Infof("MCP: 获取热门话题 [tenant=%s, app=%s] - 分类: %s", tc.TenantID, tc.AppID, category)

	result, err := s.xiaohongshuService.GetTrendingTopics(ctx, tc, category)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取热门话题失败: " + err.Error()}},
			IsError: true,
		}
	}

	var textOutput strings.Builder
	textOutput.WriteString("🔥 小红书热门话题")
	if category != "" {
		textOutput.WriteString(fmt.Sprintf(" - %s", category))
	}
	textOutput.WriteString(fmt.Sprintf(" (更新时间: %s)\n\n", result.UpdatedAt))

	if len(result.Topics) == 0 {
		textOutput.WriteString("暂无话题数据\n")
	} else {
		for i, topic := range result.Topics {
			hotMark := ""
			if topic.IsHot {
				hotMark = " 🔥"
			}
			textOutput.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, topic.Name, hotMark))
			if topic.Description != "" {
				textOutput.WriteString(fmt.Sprintf("   📝 %s\n", topic.Description))
			}
			if topic.ViewCount != "" {
				textOutput.WriteString(fmt.Sprintf("   👀 浏览: %s\n", topic.ViewCount))
			}
			if topic.PostCount != "" {
				textOutput.WriteString(fmt.Sprintf("   📮 参与: %s\n", topic.PostCount))
			}
			textOutput.WriteString("\n")
		}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: textOutput.String() + "\n详细数据序列化失败"}},
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{
			{Type: "text", Text: textOutput.String()},
			{Type: "text", Text: "📊 详细数据(JSON):\n" + string(jsonData)},
		},
	}
}
