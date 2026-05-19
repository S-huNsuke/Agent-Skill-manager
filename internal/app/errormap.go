package app

import (
	"os"
	"strings"
)

/** 将底层 Go 错误翻译为用户可读的中文提示，根据错误关键词匹配常见失败场景 */
func translateSkillError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "permission denied"):
		return "权限不足，无法操作技能目录"
	case strings.Contains(lower, "unmanaged skill"):
		return "该技能非本应用管理，无法操作"
	case strings.Contains(lower, "health") && strings.Contains(lower, "install"):
		return "代理状态异常，请先修复代理"
	case strings.Contains(lower, "does not exist") || strings.Contains(lower, "no such file"):
		return "技能目录不存在，可能已被手动删除"
	case strings.Contains(lower, "not a directory"):
		return "技能路径不是目录，请检查路径是否正确"
	case strings.Contains(lower, "skill dir") && strings.Contains(lower, "empty"):
		return "技能目录为空，没有有效文件"
	case strings.Contains(lower, "cannot read") || strings.Contains(lower, "unreadable"):
		return "无法读取技能目录，可能权限不足"
	case strings.Contains(lower, "no cached source"):
		return "未找到技能的缓存源，请先同步商店"
	case strings.Contains(lower, "remove") && (strings.Contains(lower, "dir") || strings.Contains(lower, "file")):
		return "删除文件失败，请检查目录权限"
	case strings.Contains(lower, "copy") && strings.Contains(lower, "file"):
		return "复制技能文件失败，请检查磁盘空间和权限"
	case strings.Contains(lower, "write") && strings.Contains(lower, "marker"):
		return "写入管理标记失败，请检查目录写入权限"
	}

	if os.IsTimeout(err) {
		return "操作超时，请检查网络连接后重试"
	}

	return msg
}

/** 将底层商店同步错误翻译为用户可读的中文提示 */
func translateSyncError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "no such host") || strings.Contains(lower, "connection refused") || strings.Contains(lower, "timeout") || strings.Contains(lower, "i/o timeout"):
		return "网络连接失败，请检查网络设置"
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "403"):
		return "GitHub API 请求频率超限，请稍后重试"
	case strings.Contains(lower, "404"):
		return "仓库不存在或无法访问，请检查 URL 是否正确"
	case strings.Contains(lower, "invalid github url"):
		return "GitHub URL 格式不正确，请检查商店源地址"
	case strings.Contains(lower, "no skills") && strings.Contains(lower, "directory"):
		return "该仓库中没有 skills 目录，请确认仓库结构正确"
	case strings.Contains(lower, "no readme"):
		return "技能目录中没有 README 文件"
	}

	return msg
}
