package mailer

type DefaultTemplate struct {
	Name    string
	Subject string
	Body    string
}

var DefaultTemplates = []DefaultTemplate{
	{
		Name:    "password_reset",
		Subject: "Reset your password",
		Body: `<p style="font-size:18px;font-weight:600;margin:0 0 16px;">Hi {{.FirstName}},</p>
<p style="margin:0 0 16px;">A password reset for your account was requested.</p>
<p style="margin:0 0 16px;">Click the button below to choose a new password. This link is valid for 30 minutes.</p>
<p style="text-align:center;margin:28px 0;">
  <a href="{{.ResetLink}}" style="display:inline-block;padding:14px 32px;background-color:{{._brand_accent}};color:#ffffff;text-decoration:none;border-radius:50px;font-weight:600;font-size:15px;">Reset Password</a>
</p>
<p style="margin:0;color:#888;font-size:13px;">If you did not make this request, you can ignore this email.</p>`,
	},
	{
		Name:    "payment_confirmation",
		Subject: "Payment received - {{.PlanName}}",
		Body: `<p style="font-size:18px;font-weight:600;margin:0 0 16px;">Hi {{.FirstName}},</p>
<p style="margin:0 0 16px;">We've received your payment for the <strong>{{.PlanName}}</strong> plan.</p>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:16px 0;font-size:14px;">
  <tr><td style="padding:6px 14px 6px 0;color:#666;">Amount</td><td><strong>UGX {{.Amount}}</strong></td></tr>
  <tr><td style="padding:6px 14px 6px 0;color:#666;">Method</td><td>{{.PaymentMethod}}</td></tr>
  <tr><td style="padding:6px 14px 6px 0;color:#666;">Transaction</td><td>{{.TransactionID}}</td></tr>
</table>
<p style="margin:0 0 16px;">Your subscription is active until <strong>{{.EndDate}}</strong>.</p>
<p style="margin:0;color:#888;font-size:13px;">Thank you for supporting the platform.</p>`,
	},
	{
		Name:    "subscription_expiring",
		Subject: "Your subscription expires in {{.DaysLeft}} days",
		Body: `<p style="font-size:18px;font-weight:600;margin:0 0 16px;">Hi {{.FirstName}},</p>
<p style="margin:0 0 16px;">Your subscription expires on <strong>{{.EndDate}}</strong> ({{.DaysLeft}} days from now).</p>
<p style="margin:0 0 16px;">Renew now to keep uninterrupted access to your lessons, homework, and quizzes.</p>
<p style="text-align:center;margin:28px 0;">
  <a href="{{.RenewLink}}" style="display:inline-block;padding:14px 32px;background-color:{{._brand_accent}};color:#ffffff;text-decoration:none;border-radius:50px;font-weight:600;font-size:15px;">Renew Now</a>
</p>`,
	},
	{
		Name:    "trial_ending",
		Subject: "Your free trial ends in {{.DaysLeft}} days",
		Body: `<p style="font-size:18px;font-weight:600;margin:0 0 16px;">Hi {{.FirstName}},</p>
<p style="margin:0 0 16px;">Your free trial ends on <strong>{{.EndDate}}</strong> ({{.DaysLeft}} days from now).</p>
<p style="margin:0 0 16px;">Subscribe now to keep using everything you've already started.</p>
<p style="text-align:center;margin:28px 0;">
  <a href="{{.SubscribeLink}}" style="display:inline-block;padding:14px 32px;background-color:{{._brand_accent}};color:#ffffff;text-decoration:none;border-radius:50px;font-weight:600;font-size:15px;">Choose a Plan</a>
</p>`,
	},
}
