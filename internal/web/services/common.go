package services

import (
	"strings"
)

func detectIllustrationMimeType(content []byte, path string) string {
	if len(content) == 0 {
		return ""
	}

	if len(content) >= 4 {
		if content[0] == 0x89 && content[1] == 0x50 && content[2] == 0x4E && content[3] == 0x47 {
			return "image/png"
		}
		if content[0] == 0x00 && content[1] == 0x00 && content[2] == 0x01 && content[3] == 0x00 {
			return "image/x-icon"
		}
		if content[0] == 0xFF && content[1] == 0xD8 && content[2] == 0xFF {
			return "image/jpeg"
		}
		if content[0] == 0x47 && content[1] == 0x49 && content[2] == 0x46 && content[3] == 0x38 {
			return "image/gif"
		}
	}

	if len(content) >= 12 {
		if content[0] == 0x52 && content[1] == 0x49 && content[2] == 0x46 && content[3] == 0x46 &&
			content[8] == 0x57 && content[9] == 0x45 && content[10] == 0x42 && content[11] == 0x50 {
			return "image/webp"
		}
	}

	if len(content) >= 4 {
		contentStr := string(content[:minInt(100, len(content))])
		if strings.Contains(contentStr, "<svg") || strings.Contains(contentStr, "<?xml") {
			return "image/svg+xml"
		}
	}

	if strings.HasSuffix(strings.ToLower(path), ".png") {
		return "image/png"
	}
	if strings.HasSuffix(strings.ToLower(path), ".ico") {
		return "image/x-icon"
	}
	if strings.HasSuffix(strings.ToLower(path), ".jpg") || strings.HasSuffix(strings.ToLower(path), ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(strings.ToLower(path), ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(strings.ToLower(path), ".webp") {
		return "image/webp"
	}
	if strings.HasSuffix(strings.ToLower(path), ".svg") {
		return "image/svg+xml"
	}

	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
