package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TenantContext 租户上下文，用于多租户隔离
type TenantContext struct {
	TenantID string `json:"tenant_id" jsonschema:"required,租户ID"`
	AppID    string `json:"app_id" jsonschema:"required,应用ID"`
}

// Validate 校验租户参数非空且无路径穿越
func (tc TenantContext) Validate() error {
	if tc.TenantID == "" {
		return fmt.Errorf("tenant_id 不能为空")
	}
	if tc.AppID == "" {
		return fmt.Errorf("app_id 不能为空")
	}
	if containsPathTraversal(tc.TenantID) {
		return fmt.Errorf("tenant_id 包含非法字符")
	}
	if containsPathTraversal(tc.AppID) {
		return fmt.Errorf("app_id 包含非法字符")
	}
	return nil
}

// CookiesFilePath 返回租户级别的 cookies 文件路径
func (tc TenantContext) CookiesFilePath() string {
	return filepath.Join(getDataRoot(), tc.TenantID, "cookies.json")
}

// ImagesPath 返回应用级别的图片存储路径
func (tc TenantContext) ImagesPath() string {
	return filepath.Join(getDataRoot(), tc.TenantID, tc.AppID, "images")
}

// containsPathTraversal 检测路径穿越字符
func containsPathTraversal(s string) bool {
	return strings.Contains(s, "..") ||
		strings.Contains(s, "/") ||
		strings.Contains(s, "\\")
}

// getDataRoot 获取数据根目录，优先使用环境变量 DATA_ROOT
func getDataRoot() string {
	if root := os.Getenv("DATA_ROOT"); root != "" {
		return root
	}
	return "/app/data"
}
