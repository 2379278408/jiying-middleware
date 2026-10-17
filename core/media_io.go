package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

// 媒体闭环：以账号身份上传临时文件、以账号或应用身份申请读取地址、下载附件内容。

const maxMediaDownloadBytes = 20 << 20

// sanitizeFilename 去除文件名中的路径与敏感字符，防止 multipart 部分注入。
func sanitizeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r == '/' || r == '\\' || r == '"':
			return '_'
		case r == '\r' || r == '\n' || r == '\t' || r == 0x7f || r < 0x20 || r == 0x2028 || r == 0x2029:
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	name = strings.Trim(name, ". ")
	if name == "" {
		return "file"
	}
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}

// safeContentType 只接受语法合法的媒体类型，丢弃参数并回退到二进制流。
func safeContentType(raw string) string {
	mt, _, err := mime.ParseMediaType(strings.TrimSpace(raw))
	if err != nil || mt == "" || strings.ContainsAny(mt, "\r\n") {
		return "application/octet-stream"
	}
	return mt
}

// UserUploadFile 以账号身份把字节上传为临时文件，返回 file_id。
func UserUploadFile(ctx context.Context, name string, data []byte, contentType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("上传内容为空")
	}
	contentType = safeContentType(contentType)
	if !userLoggedIn() {
		cfg := CurrentAccountConfig()
		if err := loginUser(ctx, cfg); err != nil {
			return "", err
		}
	}
	base := CurrentAccountConfig().BaseURL
	if base == "" {
		return "", fmt.Errorf("账号通道未配置")
	}
	userSession.Lock()
	token := userSession.token
	client := userSession.client
	userSession.Unlock()
	if client == nil {
		client = http.DefaultClient
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, sanitizeFilename(name)))
	hdr.Set("Content-Type", contentType)
	part, err := mw.CreatePart(hdr)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(data); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/client/temporary-files", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("上传失败 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}
	var parsed struct {
		Data struct {
			File struct {
				ID string `json:"id"`
			} `json:"file"`
		} `json:"data"`
	}
	if json.Unmarshal(out, &parsed) != nil || parsed.Data.File.ID == "" {
		return "", fmt.Errorf("上传响应解析失败")
	}
	return parsed.Data.File.ID, nil
}

// accountReadTemporaryURL 以账号身份申请临时文件的访问地址。
func accountReadTemporaryURL(ctx context.Context, fileID string) (string, error) {
	if fileID == "" {
		return "", fmt.Errorf("缺少临时文件 ID")
	}
	if !userLoggedIn() {
		cfg := CurrentAccountConfig()
		if err := loginUser(ctx, cfg); err != nil {
			return "", err
		}
	}
	base := CurrentAccountConfig().BaseURL
	if base == "" {
		return "", fmt.Errorf("账号通道未配置")
	}
	raw, err := UserCall(ctx, http.MethodPost, "/api/client/temporary-files/read-urls", map[string]any{"file_ids": []string{fileID}})
	if err != nil {
		return "", err
	}
	var parsed struct {
		Data struct {
			URLs []struct {
				FileID string `json:"file_id"`
				URL    string `json:"url"`
			} `json:"urls"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	for _, u := range parsed.Data.URLs {
		if u.FileID == fileID && u.URL != "" {
			return u.URL, nil
		}
	}
	return "", fmt.Errorf("没有可用的读取地址")
}

// appReadTemporaryURL 优先用应用通道申请临时文件读取地址；应用通道需要会话与消息上下文授权，
// 无法提供会话/消息上下文或调用失败时回退账号身份。
func appReadTemporaryURL(ctx context.Context, convID, msgID, fileID string) (string, error) {
	if convID != "" && msgID != "" {
		c := clientForConv(convID)
		if c != nil {
			raw, err := c.Call(ctx, "temporary_files.read_urls", map[string]any{
				"conversation_id": convID,
				"message_id":      msgID,
				"file_ids":        []string{fileID},
			})
			if err == nil {
				var parsed struct {
					URLs []struct {
						FileID string `json:"file_id"`
						URL    string `json:"url"`
					} `json:"urls"`
				}
				if json.Unmarshal(raw, &parsed) == nil {
					for _, u := range parsed.URLs {
						if u.FileID == fileID && u.URL != "" {
							return u.URL, nil
						}
					}
				}
			}
			GetMyLog().Warn("应用通道读取临时文件失败，回退账号身份: %v", err)
		}
	}
	return accountReadTemporaryURL(ctx, fileID)
}

// ReadFileContent 下载临时文件内容，最大 20MiB。
func ReadFileContent(ctx context.Context, convID, msgID, fileID string) ([]byte, error) {
	url, err := appReadTemporaryURL(ctx, convID, msgID, fileID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("读取失败 HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxMediaDownloadBytes))
}

// mediaRef 记录某会话最近一条入站媒体的消息上下文，供应用通道二次鉴权读取附件。
type mediaRef struct {
	MessageID string
	FileIDs   []string
}

var mediaCtx = struct {
	sync.Mutex
	items map[string]mediaRef
	order []string
}{items: map[string]mediaRef{}}

const mediaCtxCap = 256

// recordMediaCtx 在收到含附件的消息时记录会话上下文。
func recordMediaCtx(convID, msgID string, fileIDs []string) {
	if convID == "" || msgID == "" || len(fileIDs) == 0 {
		return
	}
	mediaCtx.Lock()
	if _, ok := mediaCtx.items[convID]; !ok {
		mediaCtx.order = append(mediaCtx.order, convID)
	}
	mediaCtx.items[convID] = mediaRef{MessageID: msgID, FileIDs: append([]string(nil), fileIDs...)}
	for len(mediaCtx.order) > mediaCtxCap {
		oldest := mediaCtx.order[0]
		mediaCtx.order = mediaCtx.order[1:]
		delete(mediaCtx.items, oldest)
	}
	mediaCtx.Unlock()
}

func mediaRefFor(convID string) (mediaRef, bool) {
	mediaCtx.Lock()
	defer mediaCtx.Unlock()
	r, ok := mediaCtx.items[convID]
	return r, ok
}
