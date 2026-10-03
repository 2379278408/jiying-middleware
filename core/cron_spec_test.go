package core

import "testing"

func TestIsValidCronSpec(t *testing.T) {
	valid := []string{
		"*/30 * * * * *", // 6 段含秒
		"0 0 12 * * *",   // 每天 12:00
		"*/5 * * * *",    // 5 段标准
		"@daily",         // 描述符
	}
	for _, spec := range valid {
		if !isValidCronSpec(spec) {
			t.Fatalf("isValidCronSpec(%q)=false want true", spec)
		}
	}
	invalid := []string{
		"",
		"   ",
		"every 5 minutes",
		"* * *",
		"* * * * * * *",
		"61 * * * * *",
	}
	for _, spec := range invalid {
		if isValidCronSpec(spec) {
			t.Fatalf("isValidCronSpec(%q)=true want false", spec)
		}
	}
}

func TestCronSchedulerAcceptsValidatedSpec(t *testing.T) {
	// 校验通过且与调度器使用同一解析器，挂载不应报错。
	if _, err := cronSpecParser.Parse("*/30 * * * * *"); err != nil {
		t.Fatalf("6 段表达式应可解析: %v", err)
	}
	if _, err := cronSpecParser.Parse("*/5 * * * *"); err != nil {
		t.Fatalf("5 段表达式应可解析: %v", err)
	}
}
