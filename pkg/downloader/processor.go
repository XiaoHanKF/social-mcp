package downloader

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/configs"
)

// ImageProcessor 图片处理器
type ImageProcessor struct {
	downloader *ImageDownloader
	savePath   string
}

// NewImageProcessor 创建图片处理器（向后兼容）
func NewImageProcessor() *ImageProcessor {
	savePath := configs.GetImagesPath()
	return &ImageProcessor{
		downloader: NewImageDownloader(savePath),
		savePath:   savePath,
	}
}

// NewImageProcessorWithPath 创建指定保存路径的图片处理器
func NewImageProcessorWithPath(savePath string) *ImageProcessor {
	return &ImageProcessor{
		downloader: NewImageDownloader(savePath),
		savePath:   savePath,
	}
}

// IsBase64Image 检测是否为 Base64 图片（data:image/ 前缀）
func IsBase64Image(s string) bool {
	return strings.HasPrefix(s, "data:image/")
}

// ProcessImages 处理图片列表，返回本地文件路径
// 支持三种输入格式：
// 1. URL格式 (http/https开头) - 自动下载到本地
// 2. Base64格式 (data:image/开头) - 解码保存到本地
// 3. 本地文件路径 - 直接使用
func (p *ImageProcessor) ProcessImages(images []string) ([]string, error) {
	localPaths := make([]string, 0, len(images))

	for _, image := range images {
		switch {
		case IsImageURL(image):
			localPath, err := p.downloader.DownloadImage(image)
			if err != nil {
				return nil, fmt.Errorf("下载图片失败 %s: %w", image, err)
			}
			localPaths = append(localPaths, localPath)

		case IsBase64Image(image):
			localPath, err := p.saveBase64Image(image)
			if err != nil {
				return nil, fmt.Errorf("保存Base64图片失败: %w", err)
			}
			localPaths = append(localPaths, localPath)

		default:
			localPaths = append(localPaths, image)
		}
	}

	if len(localPaths) == 0 {
		return nil, fmt.Errorf("no valid images found")
	}

	return localPaths, nil
}

// saveBase64Image 解析 data URI 并保存为文件
func (p *ImageProcessor) saveBase64Image(dataURI string) (string, error) {
	// 格式: data:image/png;base64,iVBOR...
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("无效的 data URI 格式")
	}

	// 解析 MIME 类型获取扩展名
	ext := "png" // 默认
	header := parts[0]
	if strings.Contains(header, "image/jpeg") {
		ext = "jpg"
	} else if strings.Contains(header, "image/gif") {
		ext = "gif"
	} else if strings.Contains(header, "image/webp") {
		ext = "webp"
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(p.savePath, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成文件名
	hash := sha256.Sum256(data)
	shortHash := fmt.Sprintf("%x", hash)[:16]
	fileName := fmt.Sprintf("b64_%s_%d.%s", shortHash, time.Now().UnixNano(), ext)
	filePath := filepath.Join(p.savePath, fileName)

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存图片文件失败: %w", err)
	}

	return filePath, nil
}
