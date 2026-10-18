package core

import (
	"sort"
	"strings"
)

// redactSecretText 把可能被脚本或工具输出带出的凭据替换为 ***。
// 只替换长度足够的精确值，避免把普通短词误当成密钥。
func redactSecretText(text string) string {
	if text == "" {
		return text
	}
	values := []string{CurrentAccountConfig().Password, scriptConsoleToken()}
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	for _, v := range values {
		if len(v) < 6 {
			continue
		}
		text = strings.ReplaceAll(text, v, "***")
	}
	return text
}
