package mailer

import (
	"bytes"
	"fmt"
	"html"
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

func WrapHTML(body string, b Branding) string {
	body = strings.TrimSpace(body)

	primary := b.ColorPrimary
	if primary == "" {
		primary = defaultBranding.ColorPrimary
	}
	accent := b.ColorAccent
	if accent == "" {
		accent = defaultBranding.ColorAccent
	}
	schoolName := b.SchoolName
	if schoolName == "" {
		schoolName = defaultBranding.SchoolName
	}

	var header string
	if b.LogoURL != "" {
		header = fmt.Sprintf(`
			<img src="%s" alt="%s" style="max-height:60px;max-width:220px;display:block;margin:0 auto 12px;" />
		`, html.EscapeString(b.LogoURL), html.EscapeString(schoolName))
	}

	footer := ""
	if b.WebsiteURL != "" {
		footer = fmt.Sprintf(`
			<a href="%s" style="color:%s;text-decoration:none;">%s</a>
		`, html.EscapeString(b.WebsiteURL), primary, html.EscapeString(b.WebsiteURL))
	}

	return `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:40px 20px;background:#f8fafc;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',sans-serif;line-height:1.6;color:#000;">
	<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="max-width:600px;margin:0 auto;background:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 4px 6px rgba(0,0,0,0.06);">
		<tr>
			<td style="background-color:` + primary + `;padding:36px 30px;text-align:center;">
				` + header + `
			</td>
		</tr>
		<tr>
			<td style="padding:36px 30px;">
				` + body + `
			</td>
		</tr>
		<tr>
			<td style="background:#f8fafc;padding:20px 30px;text-align:center;font-size:12px;color:#666;border-top:1px solid #e2e8f0;">
				<p style="margin:0 0 6px;">&copy; ` + fmt.Sprint(currentYear()) + ` ` + html.EscapeString(schoolName) + `. All rights reserved.</p>
				` + footer + `
			</td>
		</tr>
	</table>
</body>
</html>`
}

func currentYear() int {
	return timeNow().Year()
}

func Button(label, href string, b Branding) string {
	accent := b.ColorAccent
	if accent == "" {
		accent = defaultBranding.ColorAccent
	}
	return fmt.Sprintf(`
		<p style="text-align:center;margin:28px 0;">
			<a href="%s" style="display:inline-block;padding:14px 32px;background-color:%s;color:#ffffff;text-decoration:none;border-radius:50px;font-weight:600;font-size:15px;">%s</a>
		</p>
	`, html.EscapeString(href), accent, html.EscapeString(label))
}
