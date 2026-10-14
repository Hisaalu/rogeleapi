package mailer

type DefaultTemplate struct {
	Name    string
	Subject string
	Body    string // HTML body, may use {{.Var}} placeholders
}

var DefaultTemplates = []DefaultTemplate{
	{
		Name:    "password_reset",
		Subject: "Reset your Rogele password",
		Body: `<h2 style="margin:0 0 16px;">Reset your password</h2>
<p>Hi {{.FirstName}},</p>
<p>Someone (hopefully you) requested a password reset for your Rogele account.</p>
<p>Click the button below to choose a new password. This link expires in 30 minutes.</p>
<p style="margin:24px 0;">
  <a href="{{.ResetLink}}" style="display:inline-block;padding:12px 20px;background:#1a73e8;color:#fff;text-decoration:none;border-radius:8px;">Reset Password</a>
</p>
<p>If the button doesn't work, paste this link into your browser:</p>
<p style="word-break:break-all;color:#555;">{{.ResetLink}}</p>
<p>If you didn't request this, you can ignore this email.</p>`,
	},
	{
		Name:    "payment_confirmation",
		Subject: "Payment received — {{.PlanName}}",
		Body: `<h2 style="margin:0 0 16px;">Payment received</h2>
<p>Hi {{.FirstName}},</p>
<p>We've received your payment for <strong>{{.PlanName}}</strong>.</p>
<table style="border-collapse:collapse;margin:16px 0;">
  <tr><td style="padding:6px 12px 6px 0;color:#666;">Amount</td><td style="padding:6px 0;"><strong>{{.Amount}} {{.Currency}}</strong></td></tr>
  <tr><td style="padding:6px 12px 6px 0;color:#666;">Method</td><td style="padding:6px 0;">{{.PaymentMethod}}</td></tr>
  <tr><td style="padding:6px 12px 6px 0;color:#666;">Transaction ID</td><td style="padding:6px 0;">{{.TransactionID}}</td></tr>
</table>
<p>Your subscription is active until <strong>{{.EndDate}}</strong>.</p>
<p>Thank you for using Rogele.</p>`,
	},
	{
		Name:    "subscription_expiring",
		Subject: "Your Rogele subscription expires in {{.DaysLeft}} days",
		Body: `<h2 style="margin:0 0 16px;">Heads up — your subscription is ending soon</h2>
<p>Hi {{.FirstName}},</p>
<p>Your Rogele subscription expires on <strong>{{.EndDate}}</strong> ({{.DaysLeft}} days from now).</p>
<p>Renew now to keep uninterrupted access to your lessons, homework, and quizzes.</p>
<p style="margin:24px 0;">
  <a href="{{.RenewLink}}" style="display:inline-block;padding:12px 20px;background:#1a73e8;color:#fff;text-decoration:none;border-radius:8px;">Renew Now</a>
</p>`,
	},
	{
		Name:    "trial_ending",
		Subject: "Your free trial ends in {{.DaysLeft}} days",
		Body: `<h2 style="margin:0 0 16px;">Your trial is almost over</h2>
<p>Hi {{.FirstName}},</p>
<p>Your Rogele free trial ends on <strong>{{.EndDate}}</strong> ({{.DaysLeft}} days from now).</p>
<p>Subscribe now to keep using everything you've already started.</p>
<p style="margin:24px 0;">
  <a href="{{.SubscribeLink}}" style="display:inline-block;padding:12px 20px;background:#1a73e8;color:#fff;text-decoration:none;border-radius:8px;">Choose a Plan</a>
</p>`,
	},
}
