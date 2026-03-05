package mailer

import (
	"bytes"
	"fmt"
	htmpl "html/template"
	ttmpl "text/template"
)

// RenderText renders a text/template to a string.
func RenderText(t *ttmpl.Template, data any) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil text template")
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderHTML renders a html/template to a string.
func RenderHTML(t *htmpl.Template, data any) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil html template")
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}