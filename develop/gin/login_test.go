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
