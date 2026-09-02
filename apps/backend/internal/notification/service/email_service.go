package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	userService "backend/internal/user/service"
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type EmailService struct {
	mailer Mailer
}

func NewEmailService(mailer Mailer) *EmailService {
	return &EmailService{mailer: mailer}
}

func (s *EmailService) SendVerificationCode(
	ctx context.Context,
	email string,
	purpose string,
	code string,
	expiresAt time.Time,
) error {
	var purposeName string
	switch purpose {
	case userService.VerificationPurposeRegister:
		purposeName = "注册"
	case userService.VerificationPurposePasswordReset:
		purposeName = "重置密码"
	default:
		return errors.New("unsupported verification purpose")
	}

	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return errors.New("verification code expired")
	}
	minutes := int((remaining + time.Minute - 1) / time.Minute)
	subject := fmt.Sprintf("Koyomi Gal | %s验证码", purposeName)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<body style="
  margin:0;
  padding:0;
  background:#f5f5f7;
  font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Microsoft YaHei',sans-serif;
">

<table width="100%%" cellpadding="0" cellspacing="0" style="padding:40px 16px;">
<tr>
<td align="center">

<table width="100%%" cellpadding="0" cellspacing="0"
  style="
    max-width:560px;
    background:#fff;
    border-radius:18px;
    overflow:hidden;
    box-shadow:0 8px 32px rgba(0,0,0,.08);
  "
>

  <!-- Banner -->
  <tr>
    <td>
      <img
        src="https://img.example.com/email/banner.webp"
        width="560"
        alt="Koyomi Gal"
        style="display:block;width:100%%;border:0;"
      >
    </td>
  </tr>

  <!-- Welcome -->
  <tr>
    <td style="padding:36px 40px 12px;">

      <div style="
        font-size:13px;
        color:#9b8ec4;
        text-align:center;
        margin-bottom:12px;
      ">
        Koyomi Gal
      </div>

      <h1 style="
        margin:0;
        font-size:24px;
        color:#292638;
        text-align:center;
      ">
        欢迎回来 🌸
      </h1>

      <p style="
        margin:14px 0 0;
        text-align:center;
        color:#8b8497;
        font-size:14px;
        line-height:1.8;
      ">
        「愿你今天也能遇见喜欢的故事。」
      </p>

    </td>
  </tr>

  <!-- Content -->
  <tr>
    <td style="padding:20px 40px 38px;">

      <p style="
        margin:0 0 10px;
        color:#4b4657;
        font-size:15px;
        line-height:1.8;
      ">
        你好呀，
      </p>

      <p style="
        margin:0 0 26px;
        color:#6f687a;
        font-size:14px;
        line-height:1.8;
      ">
        你正在进行 Koyomi Gal 的
        <strong style="color:#554d65;">%s</strong>
        操作，请使用下面的验证码完成验证。
      </p>

      <!-- Verification Code -->
      <div style="
        padding:22px;
        margin:0 0 26px;
        text-align:center;
        background:linear-gradient(135deg,#faf8ff,#f5f1ff);
        border:1px solid #eee8ff;
        border-radius:14px;
      ">

        <div style="
          margin-bottom:8px;
          font-size:12px;
          color:#9d93ae;
        ">
          VERIFICATION CODE
        </div>

        <span style="
          font-size:38px;
          font-weight:700;
          letter-spacing:10px;
          color:#7557d3;
        ">
          %s
        </span>

      </div>

      <p style="
        margin:0;
        color:#777080;
        font-size:14px;
        line-height:1.8;
        text-align:center;
      ">
        验证码将在
        <strong style="color:#554d65;">%d 分钟</strong>
        后失效。
      </p>

      <!-- Security -->
      <div style="
        margin-top:28px;
        padding:16px 18px;
        background:#fafafa;
        border-radius:10px;
        color:#99929f;
        font-size:12px;
        line-height:1.8;
      ">
        如果这不是你的操作，请直接忽略这封邮件。
        为了账户安全，请不要将验证码告诉任何人。
      </div>

      <!-- Goodbye -->
      <div style="
        margin-top:32px;
        color:#827a8c;
        font-size:13px;
        line-height:1.8;
      ">
        愿故事与你再次相遇。<br>
        <strong style="color:#635c70;">Koyomi Gal</strong>
      </div>

    </td>
  </tr>

  <!-- Footer -->
  <tr>
    <td style="
      padding:22px;
      background:#faf9fc;
      color:#aaa3b1;
      text-align:center;
      font-size:11px;
      line-height:1.8;
    ">
      此邮件由 Koyomi Gal 自动发送，请勿直接回复。<br>
      © Koyomi Gal
    </td>
  </tr>

</table>

</td>
</tr>
</table>

</body>
</html>
`,
		purposeName,
		code,
		minutes,
	)
	return s.mailer.Send(ctx, email, subject, body)
}
