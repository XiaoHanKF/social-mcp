package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/xiaohongshu-mcp/configs"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

var (
	// 全局参数
	accountID string
	headless  bool
	timeout   int

	// 发布参数
	title      string
	content    string
	images     []string
	tags       []string
	isOriginal bool
	visibility string

	// 搜索参数
	keyword string
	sortBy  string

	// 话题参数
	category string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "xhs",
		Short: "小红书 CLI 工具 - 支持多账号管理",
		Long:  "通过命令行操作小红书：登录、发布、搜索、获取热点等",
	}

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&accountID, "account", "default", "账号ID，用于多账号管理")
	rootCmd.PersistentFlags().BoolVar(&headless, "headless", true, "是否使用无头模式")
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 60, "操作超时时间（秒）")

	// 添加子命令
	rootCmd.AddCommand(loginCmd())
	rootCmd.AddCommand(checkLoginCmd())
	rootCmd.AddCommand(publishCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(hotSearchesCmd())
	rootCmd.AddCommand(trendingTopicsCmd())
	rootCmd.AddCommand(listFeedsCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newBrowser 创建浏览器实例（支持账号隔离）
func newBrowser() *headless_browser.Browser {
	configs.InitHeadless(headless)

	opts := []headless_browser.Option{
		headless_browser.WithHeadless(configs.IsHeadless()),
	}

	// 加载对应账号的 Cookie
	cookiePath := getCookiePath(accountID)
	if _, err := os.Stat(cookiePath); err == nil {
		cookieLoader := cookies.NewLoadCookie(cookiePath)
		if data, err := cookieLoader.LoadCookies(); err == nil {
			opts = append(opts, headless_browser.WithCookies(string(data)))
			logrus.Debugf("已加载账号 %s 的 Cookie", accountID)
		}
	}

	return headless_browser.New(opts...)
}

// loginCmd 登录命令
func loginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "登录小红书账号",
		Run: func(cmd *cobra.Command, args []string) {
			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			loginAction := xiaohongshu.NewLogin(page)

			// 获取二维码
			ctx := context.Background()
			img, loggedIn, err := loginAction.FetchQrcodeImage(ctx)
			if err != nil {
				logrus.Fatalf("获取二维码失败: %v", err)
			}

			if loggedIn {
				fmt.Println("✅ 已经登录")
				return
			}

			// 保存二维码到文件
			qrcodeFile := fmt.Sprintf("qrcode_%s.txt", accountID)
			if err := os.WriteFile(qrcodeFile, []byte(img), 0644); err != nil {
				logrus.Fatalf("保存二维码失败: %v", err)
			}

			fmt.Printf("📱 请用小红书 App 扫码登录\n")
			fmt.Printf("二维码已保存到: %s\n", qrcodeFile)
			fmt.Println("等待扫码...")

			// 等待登录
			if loginAction.WaitForLogin(ctx) {
				// 保存 Cookie
				cks, err := page.Browser().GetCookies()
				if err != nil {
					logrus.Fatalf("获取 Cookie 失败: %v", err)
				}

				data, _ := json.Marshal(cks)
				cookieLoader := cookies.NewLoadCookie(getCookiePath(accountID))
				if err := cookieLoader.SaveCookies(data); err != nil {
					logrus.Fatalf("保存 Cookie 失败: %v", err)
				}

				fmt.Printf("✅ 登录成功！账号: %s\n", accountID)
			} else {
				fmt.Println("❌ 登录超时")
			}
		},
	}
}

// checkLoginCmd 检查登录状态
func checkLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-login",
		Short: "检查登录状态",
		Run: func(cmd *cobra.Command, args []string) {
			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			loginAction := xiaohongshu.NewLogin(page)
			ctx := context.Background()

			isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
			if err != nil {
				logrus.Fatalf("检查登录状态失败: %v", err)
			}

			if isLoggedIn {
				fmt.Printf("✅ 账号 %s 已登录\n", accountID)
			} else {
				fmt.Printf("❌ 账号 %s 未登录，请先执行: xhs login --account %s\n", accountID, accountID)
				os.Exit(1)
			}
		},
	}
}

// publishCmd 发布内容
func publishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "发布图文内容到小红书",
		Example: `  xhs publish --account alice --title "美食推荐" --content "今天吃到的超好吃的美食" --images /path/to/img1.jpg,/path/to/img2.jpg --tags 美食,推荐`,
		Run: func(cmd *cobra.Command, args []string) {
			if title == "" || content == "" || len(images) == 0 {
				fmt.Println("❌ 必须提供 --title, --content 和 --images")
				os.Exit(1)
			}

			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			ctx := context.Background()
			action, err := xiaohongshu.NewPublishImageAction(page)
			if err != nil {
				logrus.Fatalf("创建发布 Action 失败: %v", err)
			}

			publishContent := xiaohongshu.PublishImageContent{
				Title:      title,
				Content:    content,
				ImagePaths: images,
				Tags:       tags,
				IsOriginal: isOriginal,
				Visibility: visibility,
			}

			if err := action.Publish(ctx, publishContent); err != nil {
				logrus.Fatalf("发布失败: %v", err)
			}

			fmt.Printf("✅ 内容发布成功！\n")
			fmt.Printf("   账号: %s\n", accountID)
			fmt.Printf("   标题: %s\n", title)
			fmt.Printf("   图片数: %d\n", len(images))
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "标题（必填，最多20字）")
	cmd.Flags().StringVar(&content, "content", "", "内容（必填，最多1000字）")
	cmd.Flags().StringSliceVar(&images, "images", []string{}, "图片路径列表，逗号分隔（必填）")
	cmd.Flags().StringSliceVar(&tags, "tags", []string{}, "标签列表，逗号分隔")
	cmd.Flags().BoolVar(&isOriginal, "original", false, "是否声明原创")
	cmd.Flags().StringVar(&visibility, "visibility", "公开可见", "可见范围")

	return cmd
}

// searchCmd 搜索内容
func searchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search",
		Short: "搜索小红书内容",
		Example: `  xhs search --keyword "美食" --sort 最新`,
		Run: func(cmd *cobra.Command, args []string) {
			if keyword == "" {
				fmt.Println("❌ 必须提供 --keyword")
				os.Exit(1)
			}

			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			ctx := context.Background()
			action := xiaohongshu.NewSearchAction(page)

			filters := xiaohongshu.FilterOption{
				SortBy: sortBy,
			}

			feeds, err := action.Search(ctx, keyword, filters)
			if err != nil {
				logrus.Fatalf("搜索失败: %v", err)
			}

			// 输出 JSON 结果
			result, _ := json.MarshalIndent(map[string]interface{}{
				"keyword": keyword,
				"count":   len(feeds),
				"feeds":   feeds,
			}, "", "  ")

			fmt.Println(string(result))
		},
	}

	cmd.Flags().StringVar(&keyword, "keyword", "", "搜索关键词（必填）")
	cmd.Flags().StringVar(&sortBy, "sort", "综合", "排序方式：综合|最新|最多点赞")

	return cmd
}

// hotSearchesCmd 获取热搜榜
func hotSearchesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hot-searches",
		Short: "获取小红书热搜榜",
		Run: func(cmd *cobra.Command, args []string) {
			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			ctx := context.Background()
			action := xiaohongshu.NewTrendingAction(page)

			result, err := action.GetHotSearches(ctx)
			if err != nil {
				logrus.Fatalf("获取热搜失败: %v", err)
			}

			// 输出 JSON 结果
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
		},
	}
}

// trendingTopicsCmd 获取热门话题
func trendingTopicsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trending-topics",
		Short: "获取小红书热门话题",
		Example: `  xhs trending-topics --category 美食`,
		Run: func(cmd *cobra.Command, args []string) {
			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			ctx := context.Background()
			action := xiaohongshu.NewTrendingAction(page)

			result, err := action.GetTrendingTopics(ctx, category)
			if err != nil {
				logrus.Fatalf("获取热门话题失败: %v", err)
			}

			// 输出 JSON 结果
			output, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(output))
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "话题分类：美食|旅行|时尚等")

	return cmd
}

// listFeedsCmd 获取 Feed 列表
func listFeedsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-feeds",
		Short: "获取首页推荐列表",
		Run: func(cmd *cobra.Command, args []string) {
			b := newBrowser()
			defer b.Close()

			page := b.NewPage()
			defer page.Close()

			ctx := context.Background()
			action := xiaohongshu.NewFeedsListAction(page)

			feeds, err := action.GetFeedsList(ctx)
			if err != nil {
				logrus.Fatalf("获取 Feed 列表失败: %v", err)
			}

			// 输出 JSON 结果
			result, _ := json.MarshalIndent(map[string]interface{}{
				"count": len(feeds),
				"feeds": feeds,
			}, "", "  ")

			fmt.Println(string(result))
		},
	}
}

// 辅助函数

func getCookiePath(accountID string) string {
	// 确保目录存在
	dir := "./data/cookies"
	os.MkdirAll(dir, 0755)
	return fmt.Sprintf("%s/cookies_%s.json", dir, accountID)
}
