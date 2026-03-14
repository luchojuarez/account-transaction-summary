package email

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/luchojuarez/account-transaction-summary/internal/domain"
)

// Stori-inspired palette: purple primary, white, light gray (brandfetch-style).
const emailTemplate = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Account Summary</title>
</head>
<body style="margin:0; padding:0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f0fdf4; color: #000000;">
  <div style="max-width: 560px; margin: 0 auto; padding: 24px 16px;">
    
    <div style="text-align: center; padding: 80px 0 32px;">
      <img src="https://www.storicard.com/_next/static/media/storis_savvi_color.7e286ddd.svg" alt="Stori Logo" style="height: 100px; width: auto; display: block; margin: 0 auto;">
    </div>

    <div style="background: #ffffff; border-radius: 12px; padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.06);">
      <h1 style="margin: 0 0 8px; font-size: 20px; font-weight: 600; color: #000000;">
        {{ if .ToName }}Hello, {{ .ToName }}!{{ else }}Hello!{{ end }}
      </h1>
      <p style="margin: 0; font-size: 15px; color: #6b7280; line-height: 1.5;">
        Here is your account transaction summary.
      </p>
    </div>

    <div style="background: #ffffff; border-radius: 12px; padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.06);">
      <h2 style="margin: 0 0 20px; font-size: 16px; font-weight: 600; color: #004d40; text-transform: uppercase; letter-spacing: 0.5px;">Summary</h2>
      <table style="width: 100%; border-collapse: collapse;">
        <tr>
          <td style="padding: 10px 0; font-size: 14px; color: #6b7280;">Total balance</td>
          <td style="padding: 10px 0; font-size: 14px; font-weight: 600; color: #000000; text-align: right;">{{ formatMoney .Summary.TotalBalance }}</td>
        </tr>
        <tr>
          <td style="padding: 10px 0; font-size: 14px; color: #6b7280;">Total transactions</td>
          <td style="padding: 10px 0; font-size: 14px; font-weight: 600; color: #000000; text-align: right;">{{ .Summary.TotalTransactions }}</td>
        </tr>
        <tr>
          <td style="padding: 10px 0; font-size: 14px; color: #6b7280;">Average credit</td>
          <td style="padding: 10px 0; font-size: 14px; font-weight: 600; color: #a3e635; text-align: right;">{{ formatMoney .Summary.OverallAvgCredit }}</td>
        </tr>
        <tr>
          <td style="padding: 10px 0; font-size: 14px; color: #6b7280;">Average debit</td>
          <td style="padding: 10px 0; font-size: 14px; font-weight: 600; color: #004d40; text-align: right;">{{ formatMoney .Summary.OverallAvgDebit }}</td>
        </tr>
      </table>
    </div>

    {{ if .Summary.Monthly }}
    <div style="background: #ffffff; border-radius: 12px; padding: 24px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.06);">
      <h2 style="margin: 0 0 20px; font-size: 16px; font-weight: 600; color: #004d40; text-transform: uppercase; letter-spacing: 0.5px;">Monthly breakdown</h2>
      <table style="width: 100%; border-collapse: collapse; font-size: 14px;">
        <thead>
          <tr style="border-bottom: 2px solid #e5e7eb;">
            <th style="text-align: left; padding: 12px 8px; color: #6b7280; font-weight: 600;">Month</th>
            <th style="text-align: right; padding: 12px 8px; color: #6b7280; font-weight: 600;">Transactions</th>
            <th style="text-align: right; padding: 12px 8px; color: #6b7280; font-weight: 600;">Avg credit</th>
            <th style="text-align: right; padding: 12px 8px; color: #6b7280; font-weight: 600;">Avg debit</th>
          </tr>
        </thead>
        <tbody>
          {{ range .Summary.Monthly }}
          <tr style="border-bottom: 1px solid #f3f4f6;">
            <td style="padding: 12px 8px; color: #000000;">{{ .MonthName }} {{ .Year }}</td>
            <td style="padding: 12px 8px; color: #000000; text-align: right;">{{ .Count }}</td>
            <td style="padding: 12px 8px; color: #a3e635; text-align: right;">{{ if .AvgCredit }}{{ formatMoneyPtr .AvgCredit }}{{ else }}—{{ end }}</td>
            <td style="padding: 12px 8px; color: #004d40; text-align: right;">{{ if .AvgDebit }}{{ formatMoneyPtr .AvgDebit }}{{ else }}—{{ end }}</td>
          </tr>
          {{ end }}
        </tbody>
      </table>
    </div>
    {{ end }}

    <p style="margin: 24px 0 0; font-size: 12px; color: #9ca3af; text-align: center;">
      This email was generated automatically by the account-transaction-summary service.
    </p>
  </div>
</body>
</html>
`

// templateData is the root object passed to the email template.
type templateData struct {
	Summary domain.UserSummary
	ToName  string
}

// formatMoney formats a float as currency (e.g. $1,234.56).
func formatMoney(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}

// formatMoneyPtr formats a *float64 for template; returns "$0.00" if nil.
func formatMoneyPtr(p *float64) string {
	if p == nil {
		return "—"
	}
	return formatMoney(*p)
}

var emailTmpl *template.Template

func init() {
	emailTmpl = template.Must(template.New("email").Funcs(template.FuncMap{
		"formatMoney":    formatMoney,
		"formatMoneyPtr": formatMoneyPtr,
	}).Parse(emailTemplate))
}

// GenerateHTMLSummary renders the user summary as Stori-branded HTML.
func GenerateHTMLSummary(summary domain.UserSummary, toName string) (string, error) {
	data := templateData{Summary: summary, ToName: toName}
	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// GeneratePlainTextSummary returns a plain-text version of the summary for multipart emails.
func GeneratePlainTextSummary(summary domain.UserSummary, toName string) string {
	var buf bytes.Buffer
	if toName != "" {
		fmt.Fprintf(&buf, "Hello %s,\n\n", toName)
	} else {
		buf.WriteString("Hello,\n\n")
	}
	buf.WriteString("Here is your account transaction summary.\n\n")
	fmt.Fprintf(&buf, "Total balance: %s\n", formatMoney(summary.TotalBalance))
	fmt.Fprintf(&buf, "Total transactions: %d\n", summary.TotalTransactions)
	fmt.Fprintf(&buf, "Average credit: %s\n", formatMoney(summary.OverallAvgCredit))
	fmt.Fprintf(&buf, "Average debit: %s\n\n", formatMoney(summary.OverallAvgDebit))
	if len(summary.Monthly) > 0 {
		buf.WriteString("Monthly breakdown:\n")
		for _, m := range summary.Monthly {
			fmt.Fprintf(&buf, "- %s %d: %d transaction(s)", m.MonthName, m.Year, m.Count)
			if m.AvgCredit != nil {
				fmt.Fprintf(&buf, ", avg credit %s", formatMoney(*m.AvgCredit))
			}
			if m.AvgDebit != nil {
				fmt.Fprintf(&buf, ", avg debit %s", formatMoney(*m.AvgDebit))
			}
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("This email was generated automatically by the account-transaction-summary service.\n")
	return buf.String()
}
