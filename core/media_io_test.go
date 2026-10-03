package core

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

func setupMediaBucket(t *testing.T, name string) {
	t.Helper()
	setupAgentBucket(t, name)
}

func TestUserUploadFile(t *testing.T) {
	setupMediaBucket(t, "media-upload")
	var gotName, gotBody, gotCT, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/temporary-files" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		if !strings.HasPrefix(gotCT, "multipart/form-data") {
			t.Errorf("want multipart content type, got %q", gotCT)
		}
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
		_, _ = w.Write([]byte(`{"data":{"file":{"id":"fid-123","size_bytes":5}}}`))
	}))
	defer srv.Close()
	saveRelayConn(RelayConn{Account: AccountConn{BaseURL: srv.URL, Email: "e", Secret: "pw"}})
	SetUserSession("e", "tok-1")
	id, err := UserUploadFile(context.Background(), "a.txt", []byte("hello"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if id != "fid-123" {
		t.Fatalf("want fid-123, got %q", id)
	}
	if gotName != "a.txt" || gotBody != "hello" || gotCT == "" || gotAuth != "Bearer tok-1" {
		t.Fatalf("upload fields mismatch: name=%q body=%q ct=%q auth=%q", gotName, gotBody, gotCT, gotAuth)
	}
}

func TestUserUploadFileRejectsEmpty(t *testing.T) {
	setupMediaBucket(t, "media-upload-empty")
	if _, err := UserUploadFile(context.Background(), "a.txt", nil, ""); err == nil {
		t.Fatal("empty content should be rejected")
	}
}

func TestAccountReadTemporaryURL(t *testing.T) {
	setupMediaBucket(t, "media-read")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/temporary-files/read-urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok-2" {
			t.Errorf("want bearer tok-2, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"urls":[{"file_id":"f1","url":"https://cdn.example/x/f1","expires_at":"2026-09-25T00:00:00Z"}]}}`))
	}))
	defer srv.Close()
	saveRelayConn(RelayConn{Account: AccountConn{BaseURL: srv.URL, Email: "e", Secret: "pw"}})
	SetUserSession("e", "tok-2")
	url, err := accountReadTemporaryURL(context.Background(), "f1")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://cdn.example/x/f1" {
		t.Fatalf("unexpected url %q", url)
	}
}

func TestReadFileContent(t *testing.T) {
	setupMediaBucket(t, "media-content")
	var contentSrv *httptest.Server
	readSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/temporary-files/read-urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"urls":[{"file_id":"f1","url":"` + contentSrv.URL + `/dl","expires_at":"2026-09-25T00:00:00Z"}]}}`))
	}))
	defer readSrv.Close()
	contentSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dl" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("file-body"))
	}))
	defer contentSrv.Close()
	saveRelayConn(RelayConn{Account: AccountConn{BaseURL: readSrv.URL, Email: "e", Secret: "pw"}})
	SetUserSession("e", "tok-3")
	data, err := ReadFileContent(context.Background(), "", "", "f1")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "file-body" {
		t.Fatalf("unexpected content %q", string(data))
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"a.txt":         "a.txt",
		"../etc/passwd": "_etc_passwd",
		`win\evil`:      "win_evil",
		"":              "file",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMediaCtxRecordAndEvict(t *testing.T) {
	setupMediaBucket(t, "media-ctx")
	recordMediaCtx("c1", "m1", []string{"f1"})
	if ref, ok := mediaRefFor("c1"); !ok || ref.MessageID != "m1" || len(ref.FileIDs) != 1 || ref.FileIDs[0] != "f1" {
		t.Fatalf("media ctx not recorded: %+v", ref)
	}
	recordMediaCtx("c1", "m2", []string{"f2"})
	if ref, _ := mediaRefFor("c1"); ref.MessageID != "m2" {
		t.Fatalf("media ctx should be replaced by latest message")
	}
	mediaCtx.Lock()
	mediaCtx.items = map[string]mediaRef{}
	mediaCtx.order = nil
	mediaCtx.Unlock()
	for i := 0; i < mediaCtxCap+4; i++ {
		recordMediaCtx("mc-"+string(rune('a'+i%26)), "m", []string{"f"})
	}
	mediaCtx.Lock()
	order := len(mediaCtx.order)
	mediaCtx.Unlock()
	if order > mediaCtxCap {
		t.Fatalf("media ctx should cap at %d, got %d", mediaCtxCap, order)
	}
}

func TestGojaArgString(t *testing.T) {
	rt := goja.New()
	_ = rt.Set("x", 1)
	call := goja.FunctionCall{
		Arguments: []goja.Value{
			rt.ToValue("hello"),
			goja.Undefined(),
			goja.Null(),
		},
	}
	if got := gojaArgString(call, 0); got != "hello" {
		t.Fatalf("arg 0 = %q", got)
	}
	if got := gojaArgString(call, 1); got != "" {
		t.Fatalf("undefined arg should be empty, got %q", got)
	}
	if got := gojaArgString(call, 2); got != "" {
		t.Fatalf("null arg should be empty, got %q", got)
	}
	if got := gojaArgString(call, 9); got != "" {
		t.Fatalf("missing arg should be empty, got %q", got)
	}
}

func TestRegisterSendApisUploadBinds(t *testing.T) {
	setupMediaBucket(t, "media-js-upload")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/temporary-files" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"file":{"id":"fid-js","size_bytes":3}}}`))
	}))
	defer srv.Close()
	saveRelayConn(RelayConn{Account: AccountConn{BaseURL: srv.URL, Email: "e", Secret: "pw"}})
	SetUserSession("e", "tok-js")
	out, err := evalScriptCtx(`
		result = uploadFile("n.txt", "abc", "text/plain");
	`, "", "conv-js", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "fid-js" {
		t.Fatalf("uploadFile should return file id, got %q", out)
	}
	out, err = evalScriptCtx(`
		result = uploadFileBase64("n.txt", "`+base64.StdEncoding.EncodeToString([]byte("abc"))+`", "text/plain");
	`, "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "fid-js" {
		t.Fatalf("uploadFileBase64 should return file id, got %q", out)
	}
}
