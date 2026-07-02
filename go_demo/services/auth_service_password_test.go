package services

import (
	"testing"
)

// TestValidatePasswordComplexity 覆盖 validatePasswordComplexity 的所有分支：
// - 空字符串 / 过短 / 过长 → 拒绝
// - 缺少大写 / 小写 / 数字 / 特殊字符 → 拒绝
// - 合法密码（满足全部条件）→ 接受
func TestValidatePasswordComplexity(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		if err := validatePasswordComplexity(""); err == nil {
			t.Fatal("expected error for empty password")
		}
	})
	t.Run("too short", func(t *testing.T) {
		if err := validatePasswordComplexity("Ab1!"); err == nil {
			t.Fatal("expected error for length < 8")
		}
	})
	t.Run("too long", func(t *testing.T) {
		long := "Ab1!x" + repeatString("x", 68) // 73 chars, exceeds max 72
		if err := validatePasswordComplexity(long); err == nil {
			t.Fatal("expected error for length > 72")
		}
	})
	t.Run("missing uppercase", func(t *testing.T) {
		if err := validatePasswordComplexity("abcd1234!"); err == nil {
			t.Fatal("expected error missing uppercase")
		}
	})
	t.Run("missing lowercase", func(t *testing.T) {
		if err := validatePasswordComplexity("ABCD1234!"); err == nil {
			t.Fatal("expected error missing lowercase")
		}
	})
	t.Run("missing digit", func(t *testing.T) {
		if err := validatePasswordComplexity("Abcdefgh!"); err == nil {
			t.Fatal("expected error missing digit")
		}
	})
	t.Run("missing special char", func(t *testing.T) {
		if err := validatePasswordComplexity("Abcdefg1234"); err == nil {
			t.Fatal("expected error missing special character")
		}
	})
	t.Run("valid passwords", func(t *testing.T) {
		valid := []string{
			"Admin@123",
			"Passw0rd!",
			"aB3$5678",
			"P@ssw0rd",
			"Abcdef1!",
		}
		for _, pwd := range valid {
			if err := validatePasswordComplexity(pwd); err != nil {
				t.Errorf("password %q should be valid but got: %v", pwd, err)
			}
		}
	})
	t.Run("special characters set coverage", func(t *testing.T) {
		specials := "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"
		for _, r := range specials {
			pwd := "Abcdef1" + string(r)
			if err := validatePasswordComplexity(pwd); err != nil {
				t.Errorf("password with special char %q should be valid: %v", r, err)
			}
		}
	})
}

func repeatString(s string, n int) string {
	b := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		b = append(b, s...)
	}
	return string(b)
}
