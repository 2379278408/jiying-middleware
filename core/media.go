package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// SendMessage 发送任意合法消息体，返回服务端消息 ID。
func (c *JiyingClient) SendMessage(ctx context.Context, conversationID, replyTo string, message map[string]any) (string, error) {
	if conversationID == "" {
		return "", fmt.Errorf("缺少会话 ID")
	}
	return c.SendTo(ctx, convTarget("conversation", conversationID), replyTo, message)
}

// SendMediaToConv 按会话归属应用通道发送媒体消息。
func SendMediaToConv(ctx context.Context, conversationID, replyTo string, message map[string]any) (string, error) {
	c := clientForConv(conversationID)
	if c == nil {
		return "", errJiyingNotConnected
	}
	return c.SendMessage(ctx, conversationID, replyTo, message)
}

// looksLikeURL 判断媒体来源是否已是可直接拉取的绝对 URL。
func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// resolveMediaContent 将媒体来源统一解析为服务端可拉取的 URL。
// 传入绝对 URL 时原样返回；传入 file_id 时通过临时文件读取接口换取签名 URL。
func resolveMediaContent(ctx context.Context, conversationID, messageID, src string) (string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return "", fmt.Errorf("缺少媒体来源")
	}
	if looksLikeURL(src) {
		return src, nil
	}
	return appReadTemporaryURL(ctx, conversationID, messageID, src)
}

// buildImageMessage 解析图片来源并构造图片消息体。
func buildImageMessage(ctx context.Context, conversationID, src, caption, captionType string) (map[string]any, error) {
	content, err := resolveMediaContent(ctx, conversationID, "", src)
	if err != nil {
		return nil, err
	}
	return ImageMessage(content, caption, captionType), nil
}

// buildFileMessage 解析文件来源并构造文件消息体。
func buildFileMessage(ctx context.Context, conversationID, src, name string, sizeBytes int64) (map[string]any, error) {
	url, err := resolveMediaContent(ctx, conversationID, "", src)
	if err != nil {
		return nil, err
	}
	return FileMessage(url, name, sizeBytes), nil
}

// ImageMessage 构造图片消息体，content 需为服务端可拉取的 URL。
func ImageMessage(content, caption, captionType string) map[string]any {
	m := map[string]any{"type": "image", "content": content}
	if caption != "" {
		if captionType != "markdown" {
			captionType = "text"
		}
		m["caption"] = caption
		m["caption_type"] = captionType
	}
	return m
}

// FileMessage 构造文件消息体；url 需为服务端可拉取的远程 URL。
// 注意：服务端对 file 的远程拉取字段是 url，content 仅用于内联纯文本内容。
func FileMessage(url, name string, sizeBytes int64) map[string]any {
	return map[string]any{"type": "file", "url": url, "name": name, "size_bytes": sizeBytes}
}

// MarkdownMessage 构造 Markdown 富文本消息体。
func MarkdownMessage(content string) map[string]any {
	return map[string]any{"type": "markdown", "content": content}
}

// CardMessage 构造卡片消息体。
func CardMessage(title, description, url string) map[string]any {
	m := map[string]any{"type": "card", "title": title, "description": description}
	if url != "" {
		m["url"] = url
	}
	return m
}

// LinkMessage 构造链接消息体。
func LinkMessage(url, title string) map[string]any {
	return map[string]any{"type": "link", "url": url, "title": title}
}

// ChartMessage 构造图表消息体；data 需符合协议中的 labels/series 结构。
func ChartMessage(chartType, title, description string, data map[string]any) map[string]any {
	if chartType == "" {
		chartType = "bar"
	}
	return map[string]any{"type": "chart", "chart_type": chartType, "title": title, "description": description, "data": data}
}

// 便捷发送入口，供控制台接口与脚本调用。

func SendImageToConv(ctx context.Context, conversationID, src, caption, replyTo string) error {
	message, err := buildImageMessage(ctx, conversationID, src, caption, "text")
	if err != nil {
		return err
	}
	_, err = SendMediaToConv(ctx, conversationID, replyTo, message)
	return err
}

func SendFileToConv(ctx context.Context, conversationID, src, name string, size int64, replyTo string) error {
	message, err := buildFileMessage(ctx, conversationID, src, name, size)
	if err != nil {
		return err
	}
	_, err = SendMediaToConv(ctx, conversationID, replyTo, message)
	return err
}

func SendMarkdownToConv(ctx context.Context, conversationID, content, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, MarkdownMessage(content))
	return err
}

func SendCardToConv(ctx context.Context, conversationID, title, description, url, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, CardMessage(title, description, url))
	return err
}

func SendLinkToConv(ctx context.Context, conversationID, url, title, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, LinkMessage(url, title))
	return err
}

func SendChartToConv(ctx context.Context, conversationID, chartType, title, description string, data map[string]any, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, ChartMessage(chartType, title, description, data))
	return err
}

// newTraceID 为一次功能调用生成可关联的追踪 ID。
func newTraceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("trace-%d", time.Now().UnixNano())
	}
	return "trace-" + hex.EncodeToString(b[:])
}

// gojaArgString 安全读取函数调用的第 i 个参数，缺失或为空返回空串。
func gojaArgString(call goja.FunctionCall, i int) string {
	if i < 0 || i >= len(call.Arguments) {
		return ""
	}
	v := call.Argument(i)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return ""
	}
	return v.String()
}

// gojaArgStrings 把第 i 个参数（JS 数组或单值）转为去空后的字符串切片。
func gojaArgStrings(call goja.FunctionCall, i int) []string {
	if i < 0 || i >= len(call.Arguments) {
		return nil
	}
	v := call.Argument(i)
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	if v.String() == "" {
		return nil
	}
	switch arr := v.Export().(type) {
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, e := range arr {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return arr
	case string:
		if strings.TrimSpace(arr) == "" {
			return nil
		}
		return []string{arr}
	}
	return nil
}

// registerSendApis 向脚本运行时注入会话级发送与媒体能力；无会话上下文时发送返回 false。
func registerSendApis(rt *goja.Runtime, convID, replyTo string) {
	send := func(message map[string]any) bool {
		if convID == "" {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := SendMediaToConv(ctx, convID, replyTo, message); err != nil {
			GetMyLog().Warn("脚本发送消息失败：%v", err)
			return false
		}
		return true
	}
	resolveAndSend := func(build func(ctx context.Context) (map[string]any, error)) bool {
		if convID == "" {
			return false
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		message, err := build(ctx)
		if err != nil {
			GetMyLog().Warn("脚本解析媒体来源失败：%v", err)
			return false
		}
		return send(message)
	}
	_ = rt.Set("sendText", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(map[string]any{"type": "text", "content": call.Argument(0).String()}))
	})
	_ = rt.Set("sendMarkdown", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(MarkdownMessage(call.Argument(0).String())))
	})
	_ = rt.Set("sendImage", func(call goja.FunctionCall) goja.Value {
		src := call.Argument(0).String()
		caption := call.Argument(1).String()
		captionType := call.Argument(2).String()
		return rt.ToValue(resolveAndSend(func(ctx context.Context) (map[string]any, error) {
			return buildImageMessage(ctx, convID, src, caption, captionType)
		}))
	})
	_ = rt.Set("sendFile", func(call goja.FunctionCall) goja.Value {
		src := call.Argument(0).String()
		name := call.Argument(1).String()
		size := call.Argument(2).ToInteger()
		return rt.ToValue(resolveAndSend(func(ctx context.Context) (map[string]any, error) {
			return buildFileMessage(ctx, convID, src, name, size)
		}))
	})
	_ = rt.Set("sendCard", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(CardMessage(call.Argument(0).String(), call.Argument(1).String(), call.Argument(2).String())))
	})
	_ = rt.Set("sendLink", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(LinkMessage(call.Argument(0).String(), call.Argument(1).String())))
	})
	_ = rt.Set("sendChart", func(call goja.FunctionCall) goja.Value {
		data := map[string]any{}
		if v := call.Argument(3).Export(); v != nil {
			if m, ok := v.(map[string]any); ok {
				data = m
			}
		}
		return rt.ToValue(send(ChartMessage(call.Argument(0).String(), call.Argument(1).String(), call.Argument(2).String(), data)))
	})
	// uploadFile(name, content[, contentType]) -> file_id；失败返回空串。
	_ = rt.Set("uploadFile", func(call goja.FunctionCall) goja.Value {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		id, err := UserUploadFile(ctx, gojaArgString(call, 0), []byte(gojaArgString(call, 1)), gojaArgString(call, 2))
		if err != nil {
			GetMyLog().Warn("脚本上传文件失败：%v", err)
			return rt.ToValue("")
		}
		return rt.ToValue(id)
	})
	// uploadFileBase64(name, b64[, contentType]) -> file_id；失败返回空串。
	_ = rt.Set("uploadFileBase64", func(call goja.FunctionCall) goja.Value {
		data, err := base64.StdEncoding.DecodeString(gojaArgString(call, 1))
		if err != nil {
			GetMyLog().Warn("脚本上传文件 base64 解码失败：%v", err)
			return rt.ToValue("")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		id, err := UserUploadFile(ctx, gojaArgString(call, 0), data, gojaArgString(call, 2))
		if err != nil {
			GetMyLog().Warn("脚本上传文件失败：%v", err)
			return rt.ToValue("")
		}
		return rt.ToValue(id)
	})
	// lastMediaFileId() -> 该会话最近一条入站媒体的 file_id；无则空串。
	_ = rt.Set("lastMediaFileId", func(call goja.FunctionCall) goja.Value {
		if ref, ok := mediaRefFor(convID); ok && len(ref.FileIDs) > 0 {
			return rt.ToValue(ref.FileIDs[0])
		}
		return rt.ToValue("")
	})
	// readFileUrl(fileId) -> 临时文件访问地址；失败返回空串。
	_ = rt.Set("readFileUrl", func(call goja.FunctionCall) goja.Value {
		fileID := gojaArgString(call, 0)
		if fileID == "" {
			return rt.ToValue("")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		url := ""
		if ref, ok := mediaRefFor(convID); ok {
			url, _ = appReadTemporaryURL(ctx, convID, ref.MessageID, fileID)
		} else {
			url, _ = accountReadTemporaryURL(ctx, fileID)
		}
		if url == "" {
			GetMyLog().Warn("脚本读取临时文件地址失败 file=%s", fileID)
		}
		return rt.ToValue(url)
	})
	// readFileBase64(fileId) -> 附件内容 base64；失败返回空串。
	_ = rt.Set("readFileBase64", func(call goja.FunctionCall) goja.Value {
		fileID := gojaArgString(call, 0)
		if fileID == "" {
			return rt.ToValue("")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		msgID := ""
		if ref, ok := mediaRefFor(convID); ok {
			msgID = ref.MessageID
		}
		data, err := ReadFileContent(ctx, convID, msgID, fileID)
		if err != nil {
			GetMyLog().Warn("脚本读取附件失败 file=%s: %v", fileID, err)
			return rt.ToValue("")
		}
		return rt.ToValue(base64.StdEncoding.EncodeToString(data))
	})
	// readFile(fileId) -> 附件内容 UTF-8 字符串；失败返回空串。
	_ = rt.Set("readFile", func(call goja.FunctionCall) goja.Value {
		fileID := gojaArgString(call, 0)
		if fileID == "" {
			return rt.ToValue("")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		msgID := ""
		if ref, ok := mediaRefFor(convID); ok {
			msgID = ref.MessageID
		}
		data, err := ReadFileContent(ctx, convID, msgID, fileID)
		if err != nil {
			GetMyLog().Warn("脚本读取附件失败 file=%s: %v", fileID, err)
			return rt.ToValue("")
		}
		return rt.ToValue(string(data))
	})
	registerAppApis(rt, convID)
}
