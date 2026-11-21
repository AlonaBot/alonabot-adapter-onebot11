package client

import (
	"strings"

	msg "github.com/AlonaBot/AlonaBot/sdk/message"
)

// ButtonsText 显式将按钮标签与 URL 转成文本，回调按钮标注无法交互。
func ButtonsText(buttons msg.ButtonsSegment) msg.TextSegment {
	var rows []string
	for _, row := range buttons.Rows {
		var labels []string
		for _, button := range row {
			label := button.Label
			if button.URL != "" {
				label += " (" + button.URL + ")"
			} else {
				label += "（回调按钮，无法交互）"
			}
			labels = append(labels, label)
		}
		rows = append(rows, strings.Join(labels, "  "))
	}
	return msg.TextSegment{Text: strings.Join(rows, "\n")}
}

// SplitSentences 一次性按句末标点或换行切分，连接结果与原文逐字符相同。
func SplitSentences(text string) []string {
	var out []string
	start := 0
	for index, char := range text {
		if strings.ContainsRune("。！？!?；;\n", char) {
			end := index + len(string(char))
			out = append(out, text[start:end])
			start = end
		}
	}
	if start < len(text) {
		out = append(out, text[start:])
	}
	return out
}
