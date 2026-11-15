package widgets

import (
	"bytes"
	"encoding/json"
)

// PayloadFormatter lets hosts add formats without changing the payload editor.
type PayloadFormatter interface {
	Name() string
	Format(string) (string, error)
}

type JSONFormatter struct{}

func (JSONFormatter) Name() string { return "JSON" }

func (JSONFormatter) Format(value string) (string, error) {
	var out bytes.Buffer
	// Indent preserves number precision and object key order.
	if err := json.Indent(&out, []byte(value), "", "  "); err != nil {
		return value, err
	}
	return out.String(), nil
}
