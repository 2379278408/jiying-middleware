package core

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"jiying/middleware/utils"
)

const confFileName = "config.yaml"

func initYamlConf() {
	text := ReadYaml(filepath.Join(utils.GetResHome(), confFileName))
	applyYamlConf(text)
}

// yamlConf 对应 config.yaml；字段名与桶中配置键保持一致，启动时同步。
type yamlConf struct {
	HTTP struct {
		Port   int    `yaml:"port"`
		Cert   string `yaml:"certFile"`
		Key    string `yaml:"keyFile"`
		Public string `yaml:"team_publicip_api"`
		Local  string `yaml:"local_host"`
	} `yaml:"http"`
	System struct {
		UpdateSource string `yaml:"UpdateSource"`
		AutoCheck    *bool  `yaml:"AutoCheck"`
	} `yaml:"system"`
}

// applyYamlConf 把配置文件同步进应用桶，保证编辑 config.yaml 后重启即可生效。
func applyYamlConf(text string) {
	if BucketApp == nil || strings.TrimSpace(text) == "" {
		return
	}
	var conf yamlConf
	if err := yaml.Unmarshal([]byte(text), &conf); err != nil {
		GetMyLog().Warn("配置文件解析失败: %v", err)
		return
	}
	if conf.HTTP.Port > 0 {
		BucketApp.Set("port", conf.HTTP.Port)
	}
	BucketApp.Set("certFile", strings.TrimSpace(conf.HTTP.Cert))
	BucketApp.Set("keyFile", strings.TrimSpace(conf.HTTP.Key))
	BucketApp.Set("team_publicip_api", strings.TrimSpace(conf.HTTP.Public))
	BucketApp.Set("local_host", strings.TrimSpace(conf.HTTP.Local))
	if strings.TrimSpace(conf.System.UpdateSource) != "" {
		BucketApp.Set("UpdateSource", strings.TrimSpace(conf.System.UpdateSource))
	}
	if conf.System.AutoCheck != nil {
		BucketApp.Set("AutoCheck", *conf.System.AutoCheck)
	}
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
	if mdErr := os.MkdirAll(dir, 0o700); mdErr != nil {
		GetMyLog().Error("创建配置目录失败 %s: %v", dir, mdErr)
		return ""
	}
	content := defaultConfYaml()
	if f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600); openErr == nil {
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
