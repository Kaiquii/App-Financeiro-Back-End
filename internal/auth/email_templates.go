package auth

import (
	"fmt"
	"html"
)

func passwordResetEmailHTML(name string, code string) string {
	escapedName := html.EscapeString(name)

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="pt-BR">
<head>
	<meta charset="UTF-8">
	<title>Redefinição de senha</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fb;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f7fb;padding:32px 0;">
		<tr>
			<td align="center">
				<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;background-color:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e5e7eb;">
					<tr>
						<td style="background-color:#2563eb;padding:24px 32px;text-align:center;">
							<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">SobraAi</h1>
						</td>
					</tr>
					<tr>
						<td style="padding:32px;">
							<h2 style="margin:0 0 16px;color:#111827;font-size:20px;">Redefinição de senha</h2>
							<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">Olá, %s.</p>
							<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">Recebemos uma solicitação para redefinir a senha da sua conta.</p>
							<p style="margin:0 0 24px;font-size:15px;line-height:1.6;color:#374151;">Use o código abaixo para confirmar sua identidade:</p>
							<div style="text-align:center;margin:28px 0;">
								<div style="display:inline-block;background-color:#eff6ff;border:1px solid #bfdbfe;border-radius:10px;padding:16px 28px;">
									<span style="font-size:32px;font-weight:700;letter-spacing:6px;color:#1d4ed8;">%s</span>
								</div>
							</div>
							<p style="margin:24px 0 0;font-size:14px;line-height:1.6;color:#4b5563;">Este código expira em <strong>10 minutos</strong>.</p>
							<p style="margin:12px 0 0;font-size:14px;line-height:1.6;color:#4b5563;">Se você não solicitou essa alteração, pode ignorar este e-mail com segurança.</p>
						</td>
					</tr>
					<tr>
						<td style="background-color:#f9fafb;padding:20px 32px;text-align:center;border-top:1px solid #e5e7eb;">
							<p style="margin:0;font-size:12px;color:#6b7280;">Este e-mail foi enviado automaticamente. Não responda a esta mensagem.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>
`, escapedName, code)
}

func registrationCodeEmailHTML(code string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="pt-BR">
<head>
	<meta charset="UTF-8">
	<title>Confirmação de e-mail</title>
</head>
<body style="margin:0;padding:0;background-color:#f4f7fb;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f7fb;padding:32px 0;">
		<tr>
			<td align="center">
				<table width="100%%" cellpadding="0" cellspacing="0" style="max-width:560px;background-color:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e5e7eb;">
					<tr>
						<td style="background-color:#2563eb;padding:24px 32px;text-align:center;">
							<h1 style="margin:0;color:#ffffff;font-size:22px;font-weight:700;">SobraAi</h1>
						</td>
					</tr>
					<tr>
						<td style="padding:32px;">
							<h2 style="margin:0 0 16px;color:#111827;font-size:20px;">Confirme seu e-mail</h2>
							<p style="margin:0 0 16px;font-size:15px;line-height:1.6;color:#374151;">Use o código abaixo para confirmar seu e-mail e concluir o cadastro no SobraAi.</p>
							<div style="text-align:center;margin:28px 0;">
								<div style="display:inline-block;background-color:#eff6ff;border:1px solid #bfdbfe;border-radius:10px;padding:16px 28px;">
									<span style="font-size:32px;font-weight:700;letter-spacing:6px;color:#1d4ed8;">%s</span>
								</div>
							</div>
							<p style="margin:24px 0 0;font-size:14px;line-height:1.6;color:#4b5563;">Este código expira em <strong>10 minutos</strong>.</p>
							<p style="margin:12px 0 0;font-size:14px;line-height:1.6;color:#4b5563;">Se você não tentou criar uma conta, pode ignorar este e-mail com segurança.</p>
						</td>
					</tr>
					<tr>
						<td style="background-color:#f9fafb;padding:20px 32px;text-align:center;border-top:1px solid #e5e7eb;">
							<p style="margin:0;font-size:12px;color:#6b7280;">Este e-mail foi enviado automaticamente. Não responda a esta mensagem.</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>
`, code)
}
