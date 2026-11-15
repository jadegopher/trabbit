package widgets

import (
	"github.com/jadegopher/trabbit/internal/tui/core"
	"image"
)

const (
	// Keep the form readable on wide terminals while leaving most space for text.
	publishFieldsMaxWidth     = 48
	publishFieldsWidthDivisor = 3

	publishHeaderRows          = 1
	publishHelpRows            = 1
	publishPayloadTitleRows    = 1
	publishPanelBorderCells    = 1
	publishPanelLeadingPadding = 1
	publishWheelScrollRows     = 3
)

// All rectangles use screen coordinates. Content includes the viewport's own
// scrollbar column; only the editor text width excludes that column.
type publishPanelGeometry struct {
	bounds  image.Rectangle
	content image.Rectangle
}

type publishGeometry struct {
	header, panels, hints            image.Rectangle
	fields, payload                  publishPanelGeometry
	payloadTitle, editor, editorText image.Rectangle
}

func publishPanelBounds(bounds image.Rectangle) publishPanelGeometry {
	content := bounds.Inset(publishPanelBorderCells)
	content.Min.X = min(content.Max.X, content.Min.X+publishPanelLeadingPadding)
	return publishPanelGeometry{bounds: bounds, content: content}
}

// geometry is the single source for widget sizes, composition and mouse offsets.
func (p *PublishForm) geometry() publishGeometry {
	headerEnd := min(p.height, publishHeaderRows)
	panelsEnd := max(headerEnd, p.height-publishHelpRows)
	fieldsWidth := min(publishFieldsMaxWidth, p.width/publishFieldsWidthDivisor)
	layout := publishGeometry{
		header:  image.Rect(0, 0, p.width, headerEnd),
		panels:  image.Rect(0, headerEnd, p.width, panelsEnd),
		hints:   image.Rect(0, panelsEnd, p.width, p.height),
		fields:  publishPanelBounds(image.Rect(0, headerEnd, fieldsWidth, panelsEnd)),
		payload: publishPanelBounds(image.Rect(fieldsWidth, headerEnd, p.width, panelsEnd)),
	}
	layout.payloadTitle = layout.payload.content
	layout.payloadTitle.Max.Y = min(layout.payloadTitle.Max.Y, layout.payloadTitle.Min.Y+publishPayloadTitleRows)
	layout.editor = layout.payload.content
	layout.editor.Min.Y = layout.payloadTitle.Max.Y
	layout.editorText = layout.editor
	layout.editorText.Max.X = max(layout.editorText.Min.X, layout.editorText.Max.X-core.ScrollbarColumns)
	return layout
}
