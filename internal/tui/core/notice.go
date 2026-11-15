package core

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

type NoticeKind uint8

const (
	NoticeInfo NoticeKind = iota
	NoticeSuccess
	NoticeWarning
	NoticeError
)

const (
	noticePaddingColumns    = 1
	noticeTopBannerRows     = 1
	noticeBorderColumns     = 1
	noticeRoundedBorderRows = 3
)

// RenderNotice fills a fixed banner without moving surrounding controls. Text
// wraps inside its padding, with contrasting text on the severity color.
func (t Theme) RenderNotice(width, height int, message string, kind NoticeKind) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if message == "" {
		return CropView(tea.NewView(""), width, height, 0).Content
	}
	style := t.noticeStyle(kind)
	sideBorder := width > noticeBorderColumns*2
	roundedBorder := sideBorder && height >= noticeRoundedBorderRows
	innerWidth, innerHeight := width, height
	if sideBorder {
		innerWidth -= noticeBorderColumns * 2
	}
	if roundedBorder {
		innerHeight -= BorderOverhead
	}
	padding := min(noticePaddingColumns, innerWidth/2)
	contentWidth := max(0, innerWidth-padding*2)
	message = strings.ReplaceAll(message, "\r", "")
	if innerHeight == 1 {
		message = ansi.Truncate(strings.ReplaceAll(message, "\n", " "), contentWidth, "…")
	}
	content := ansi.Wrap(message, contentWidth, " ")
	content = strings.TrimRight(CropView(tea.NewView(content), contentWidth, innerHeight, 0).Content, " \n")
	return style.Bold(true).
		Border(lipgloss.RoundedBorder(), roundedBorder, sideBorder, roundedBorder, sideBorder).
		BorderForeground(style.GetBackground()).
		Width(width).Height(height).Padding(0, padding).AlignVertical(lipgloss.Center).
		Render(content)
}

// RenderTopNotice replaces the top row without shifting the screen or its cursor.
// Dialogs and scenes retain their geometry while a global notice is visible.
func (t Theme) RenderTopNotice(view tea.View, width, height int, message string, kind NoticeKind) tea.View {
	if width <= 0 || height <= 0 {
		return view
	}
	view = CropView(view, width, height, 0)
	lines := strings.Split(view.Content, "\n")
	lines[0] = t.RenderNotice(width, noticeTopBannerRows, message, kind)
	view.Content = strings.Join(lines, "\n")
	if view.Cursor != nil && view.Cursor.Y < noticeTopBannerRows {
		view.Cursor = nil
	}
	return view
}
