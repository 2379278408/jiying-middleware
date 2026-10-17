package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDecodeUploadBase64(t *testing.T) {
	raw := []byte("relay upload")
	std := base64.StdEncoding.EncodeToString(raw)
	rawNoPad := base64.RawStdEncoding.EncodeToString(raw)
	for _, in := range []string{std, rawNoPad, std[:8] + "\r\n" + std[8:]} {
		got, err := decodeUploadBase64(in)
		if err != nil || !bytes.Equal(got, raw) {
			t.Fatalf("decode %q failed: %v", in, err)
		}
	}
	if _, err := decodeUploadBase64(""); err == nil {
		t.Fatal("empty content should fail")
	}
	if _, err := decodeUploadBase64("不是 base64"); err == nil {
		t.Fatal("invalid base64 should fail")
	}
}

func TestCleanUploadContentType(t *testing.T) {
	cases := map[string]string{
		"":                          "application/octet-stream",
		"text/plain":                "text/plain",
		"text/plain; charset=utf-8": "text/plain; charset=utf-8",
		"image/png\r\nX-Evil: 1":    "application/octet-stream",
		"not a type":                "application/octet-stream",
	}
	for in, want := range cases {
		if got := cleanUploadContentType(in); got != want {
			t.Fatalf("clean %q = %q, want %q", in, got, want)
		}
	}
}

func TestJiyingAPIUploadBase64(t *testing.T) {
	setupMediaBucket(t, "upload-handler")
	var gotName, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("multipart reader: %v", err)
		}
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("next part: %v", err)
			}
			if p.FormName() == "file" {
				gotName = p.FileName()
				b, _ := io.ReadAll(p)
				gotBody = string(b)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"file":{"id":"fid-upload","size_bytes":5}}}`))
	}))
	defer srv.Close()
	saveRelayConn(RelayConn{Account: AccountConn{BaseURL: srv.URL, Email: "e", Secret: "pw"}})
	SetUserSession("e", "tok-upload")

	form := url.Values{
		"name":            {"报告.txt"},
		"content_type":    {"text/plain"},
		"data_base64":     {"aGVsbG8="},
		"conversation_id": {"c1"},
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/jiying/api/upload", strings.NewReader(form.Encode()))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	jiyingAPIUpload(c)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			FileID string `json:"file_id"`
			Name   string `json:"name"`
			Size   int    `json:"size"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 200 || resp.Data.FileID != "fid-upload" || resp.Data.Name != "报告.txt" || resp.Data.Size != 5 {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
	if gotName != "报告.txt" || gotBody != "hello" {
		t.Fatalf("upstream fields mismatch: name=%q body=%q", gotName, gotBody)
	}
}
