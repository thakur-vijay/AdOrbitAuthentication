package emailotp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"math/big"

	"github.com/thakur-vijay/adorbit-go/mailservice"
)

const verificationTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="color-scheme" content="light">
    <title>Verify your email - AdOrbit</title>
</head>

<body style="margin:0;padding:0;background-color:#f3f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Arial,sans-serif;">

<table width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f3f5f9;padding:40px 12px;">
<tr>
<td align="center">

<table width="100%" cellpadding="0" cellspacing="0" border="0"
style="max-width:520px;background-color:#ffffff;border-radius:16px;overflow:hidden;">

    <!-- Header -->
    <tr>
        <td align="center" style="background-color:#111827;padding:32px 24px;">
            <h1 style="margin:0;color:#ffffff;font-size:27px;font-weight:750;letter-spacing:-1px;">
                AdOrbit<span style="color:#6366f1;">.</span>
            </h1>
            <p style="margin:9px 0 0;color:#9ca3af;font-size:12px;letter-spacing:2px;text-transform:uppercase;">
                Advertising Intelligence Platform
            </p>
        </td>
    </tr>

    <!-- Main Content -->
    <tr>
        <td style="padding:38px 32px 20px;">
            <h2 style="margin:0 0 14px;color:#111827;font-size:24px;font-weight:700;text-align:center;">
                Verify your email
            </h2>

            <p style="margin:0;color:#6b7280;font-size:15px;line-height:1.7;text-align:center;">
                Welcome to AdOrbit! To secure your account, please use the verification code below.
            </p>
        </td>
    </tr>

    <!-- OTP -->
    <tr>
        <td align="center" style="padding:20px 32px 30px;">

            <p style="margin:0 0 14px;color:#6b7280;font-size:12px;font-weight:600;letter-spacing:1.5px;text-transform:uppercase;">
                Your verification code
            </p>

            <table cellpadding="0" cellspacing="0" border="0">
                <tr>
                    <td align="center" style="background-color:#f3f4ff;border:1px solid #e0e3ff;border-radius:12px;padding:20px 35px;">
                        <span style="color:#4f46e5;font-size:36px;font-weight:750;letter-spacing:12px;font-family:monospace;">
                            {{.OTP}}
                        </span>
                    </td>
                </tr>
            </table>

            <p style="margin:18px 0 0;color:#9ca3af;font-size:12px;">
                Select the code above to copy it.
            </p>

        </td>
    </tr>

    <!-- Expiry -->
    <tr>
        <td style="padding:0 32px 28px;">
            <table width="100%" cellpadding="0" cellspacing="0" border="0"
            style="background-color:#fffbeb;border:1px solid #fde68a;border-radius:10px;">
                <tr>
                    <td style="padding:15px 16px;">
                        <p style="margin:0;color:#92400e;font-size:13px;line-height:1.6;">
                            <strong>Security notice</strong><br>
                            This verification code expires in <strong>5 minutes</strong>.
                        </p>
                    </td>
                </tr>
            </table>
        </td>
    </tr>

    <!-- Security -->
    <tr>
        <td style="padding:0 32px 35px;border-bottom:1px solid #f1f2f5;">
            <p style="margin:0;color:#6b7280;font-size:13px;line-height:1.7;text-align:center;">
                If you didn't request this verification code, you can safely ignore this email.
                Never share your OTP with anyone.
            </p>
        </td>
    </tr>

    <!-- Footer -->
    <tr>
        <td align="center" style="padding:25px 24px 30px;">
            <p style="margin:0 0 8px;color:#111827;font-size:13px;font-weight:600;">
                AdOrbit
            </p>

            <p style="margin:0;color:#9ca3af;font-size:11px;line-height:1.6;">
                © 2026 AdOrbit. All rights reserved.<br>
                This is an automated email. Please do not reply.
            </p>
        </td>
    </tr>

</table>

</td>
</tr>
</table>

</body>
</html>
`

type Service struct {
	mailService   mailservice.Service
	repository    *Repository
	otpHMACSecret []byte
}

func NewService(mailService mailservice.Service, repository *Repository, otpHMACSecret []byte) *Service {
	return &Service{
		mailService:   mailService,
		repository:    repository,
		otpHMACSecret: otpHMACSecret,
	}
}

func (s *Service) Send(ctx context.Context, userID, email string) error {
	otp, err := generateOTP()
	if err != nil {
		return err
	}

	otpHash := s.generateOTPHMAC(userID, otp)

	emailOTP := EmailOTP{
		UserID:  userID,
		OTPHash: otpHash,
	}
	err = s.repository.Create(ctx, emailOTP)
	if err != nil {
		return err
	}
	err = s.sendMail(ctx, otp, email)

	return err
}

func (s *Service) sendMail(ctx context.Context, otp, email string) error {
	tmpl, err := template.New("verification").Parse(verificationTemplate)
	if err != nil {
		return fmt.Errorf("parse verification template: %w", err)
	}

	var htmlBody bytes.Buffer

	if err := tmpl.Execute(&htmlBody, struct {
		OTP string
	}{
		OTP: otp,
	}); err != nil {
		return fmt.Errorf("render verification template: %w", err)
	}

	return s.mailService.Send(ctx, mailservice.Message{
		To:       []string{email},
		Subject:  "Verify your email - AdOrbit",
		HTMLBody: htmlBody.String(),
		TextBody: fmt.Sprintf(
			"Your AdOrbit verification OTP is %s. It expires in 5 minutes.",
			otp,
		),
	})
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

func (s *Service) generateOTPHMAC(userID, otp string) string {
	mac := hmac.New(sha256.New, s.otpHMACSecret)
	mac.Write([]byte(userID + ":" + otp))

	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) verifyOTPHMAC(userID, otp, storedHash string) bool {
	expectedHash := s.generateOTPHMAC(userID, otp)

	return hmac.Equal([]byte(expectedHash), []byte(storedHash))
}
