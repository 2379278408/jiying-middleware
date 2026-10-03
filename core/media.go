package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// SendMessage 发送任意合法消息体，返回服务端消息 ID。
func (c *JiyingClient) SendMessage(ctx context.Context, conversationID, replyTo string, message map[string]any) (string, error) {
	if conversationID == "" || len(message) == 0 {
		return "", fmt.Errorf("缺少会话或消息内容")
	}
	body := map[string]any{
		"target":  map[string]any{"type": "conversation", "conversation_id": conversationID},
		"message": message,
	}
	if replyTo != "" {
		body["reply_to_message_id"] = replyTo
	}
	raw, err := c.Call(ctx, "message.send", body)
	if err != nil {
		return "", err
	}
	var resp struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	_ = json.Unmarshal(raw, &resp)
	return resp.Message.ID, nil
}

// SendMediaToConv 按会话归属应用通道发送媒体消息。
func SendMediaToConv(ctx context.Context, conversationID, replyTo string, message map[string]any) (string, error) {
	c := clientForConv(conversationID)
	if c == nil {
		return "", errJiyingNotConnected
	}
	return c.SendMessage(ctx, conversationID, replyTo, message)
}

// ImageMessage 构造图片消息体。
func ImageMessage(fileID, caption string) map[string]any {
	m := map[string]any{"type": "image", "file_id": fileID}
	if caption != "" {
		m["caption"] = caption
		m["caption_type"] = "text"
	}
	return m
}

// FileMessage 构造文件消息体。
func FileMessage(fileID, name string, sizeBytes int64) map[string]any {
	return map[string]any{"type": "file", "file_id": fileID, "name": name, "size_bytes": sizeBytes}
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

func SendImageToConv(ctx context.Context, conversationID, fileID, caption, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, ImageMessage(fileID, caption))
	return err
}

func SendFileToConv(ctx context.Context, conversationID, fileID, name string, size int64, replyTo string) error {
	_, err := SendMediaToConv(ctx, conversationID, replyTo, FileMessage(fileID, name, size))
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
	_ = rt.Set("sendText", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(map[string]any{"type": "text", "content": call.Argument(0).String()}))
	})
	_ = rt.Set("sendImage", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(ImageMessage(call.Argument(0).String(), call.Argument(1).String())))
	})
	_ = rt.Set("sendFile", func(call goja.FunctionCall) goja.Value {
		return rt.ToValue(send(FileMessage(call.Argument(0).String(), call.Argument(1).String(), call.Argument(2).ToInteger())))
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
}
