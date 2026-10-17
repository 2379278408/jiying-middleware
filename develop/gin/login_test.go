package gin

import (
	"strconv"
	"testing"
)

// 找回验证码必须是 6 位数字且熵充分；旧实现只生成末三位为 0 的值（如 010000），
// 有效空间仅 1000，配合无重试限制可被在线枚举。
func TestGenVerifyCodeEntropy(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过随机性检查")
	}
	distinct := map[string]bool{}
	multiplesOfThousand := 0
	const n = 200
	for i := 0; i < n; i++ {
		code := genVerifyCode()
		if len(code) != 6 {
			t.Fatalf("验证码应为 6 位: %q", code)
		}
		v, err := strconv.Atoi(code)
		if err != nil || v < 0 || v > 999999 {
			t.Fatalf("验证码应为 6 位数字: %q", code)
		}
		if v%1000 == 0 {
			multiplesOfThousand++
		}
		distinct[code] = true
	}
	if len(distinct) < n*3/4 {
		t.Fatalf("验证码重复过多，疑似熵不足: %d unique / %d", len(distinct), n)
	}
	if multiplesOfThousand > 3 {
		t.Fatalf("验证码疑似退化为旧的 3 位实现: %d/%d 为整千", multiplesOfThousand, n)
	}
}

func TestResetTicketOneTimeAndBoundToUser(t *testing.T) {
	token, code := issueResetTicket("admin")
	if token == "" || len(code) != 6 {
		t.Fatalf("票据或验证码生成失败: %q %q", token, code)
	}
	if _, ok := verifyResetTicket(token, "000000"); ok && code != "000000" {
		t.Fatal("错误验证码不应通过")
	}
	userID, ok := verifyResetTicket(token, code)
	if !ok || userID != "admin" {
		t.Fatalf("正确验证码应通过: %q %v", userID, ok)
	}
	if _, ok := verifyResetTicket(token, code); ok {
		t.Fatal("票据必须一次性使用")
	}
}

func TestResetTicketAttemptLimit(t *testing.T) {
	token, code := issueResetTicket("ops")
	if token == "" {
		t.Fatal("票据生成失败")
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	for i := 0; i < resetTicketAttempts; i++ {
		if _, ok := verifyResetTicket(token, wrong); ok {
			t.Fatal("错误验证码不应通过")
		}
	}
	if _, ok := verifyResetTicket(token, code); ok {
		t.Fatal("超过尝试次数后票据必须失效")
	}
}

func TestSameRequestHost(t *testing.T) {
	if !sameRequestHost("https://console.example/login", "console.example") {
		t.Fatal("同源地址应放行")
	}
	if !sameRequestHost("https://console.example:8443/login", "console.example:8443") {
		t.Fatal("同端口地址应放行")
	}
	if sameRequestHost("https://attacker.example/login", "console.example") {
		t.Fatal("跨站地址必须拒绝")
	}
	if sameRequestHost("null", "console.example") {
		t.Fatal("null 来源必须拒绝")
	}
}
