package mailer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func RenderText(tmpl string, data map[string]any) (string, error) {
	t, err := template.New("t").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// WrapHTML wraps a plain body in a minimal responsive HTML shell.
func WrapHTML(body string) string {
	body = strings.TrimSpace(body)
	return `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
</head>
<body style="margin:0;padding:0;background:#f6f7f9;font-family:system-ui,-apple-system,'Segoe UI',Roboto,sans-serif;color:#1a1a1a;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#f6f7f9;padding:32px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:600px;background:#ffffff;border-radius:12px;padding:32px;">
          <tr><td>` + body + `</td></tr>
        </table>
        <div style="max-width:600px;margin-top:16px;font-size:12px;color:#8a8f99;text-align:center;">
          You received this email from Rogele. If you didn't expect it, you can safely ignore it.
        </div>
      </td>
    </tr>
  </table>
</body>
</html>`
}
