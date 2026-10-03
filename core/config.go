package core

import (
	"os"
	"path/filepath"
	"strings"

	"jiying/middleware/utils"
)

const confFileName = "config.yaml"

func initYamlConf() {
	ReadYaml(filepath.Join(utils.GetResHome(), confFileName))
}

func ReadYaml(path string) string {
	data, err := os.ReadFile(path)
	if err == nil {
		text := string(data)
		if len(strings.TrimSpace(text)) == 0 {
			GetMyLog().Warn("配置文件 %s 内容为空", path)
		}
		return text
	}
	dir := filepath.Dir(path)
	if mdErr := os.MkdirAll(dir, 0o777); mdErr != nil {
		GetMyLog().Error("创建配置目录失败 %s: %v", dir, mdErr)
		return ""
	}
	content := defaultConfYaml()
	if f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o777); openErr == nil {
		_, _ = f.WriteString(content)
		_ = f.Close()
		GetMyLog().Info("已生成默认配置文件 %s", path)
		return content
	}
	return ""
}

func defaultConfYaml() string {
	return `# 即应中间件配置文件
# 注意：修改后需重启服务生效

http:
  # 授权配置
  port: 8080               # Http 服务端口（也可用环境变量 JIYING_PORT 覆盖）
  certFile: ""             # HTTPS 证书文件（配置后自动启用 TLS）
  keyFile: ""              # HTTPS 私钥文件
  team_publicip_api: ""    # 公网 IP 查询接口
  local_host: ""           # 内网地址

system:
  # 系统配置
  UpdateSource: ""         # 更新源
  AutoCheck: true          # 自动检查更新
`
}
