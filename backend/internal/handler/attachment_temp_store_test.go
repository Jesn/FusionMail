package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestStore(t *testing.T) *AttachmentTempStore {
	t.Helper()
	s, err := NewAttachmentTempStore(filepath.Join(t.TempDir(), "att"), 1024)
	if err != nil {
		t.Fatalf("创建 store 失败: %v", err)
	}
	return s
}

func TestAttachmentTempStoreSaveAndRead(t *testing.T) {
	s := newTestStore(t)

	tempPath, err := s.Save([]byte("hello attachment"))
	if err != nil {
		t.Fatalf("Save 失败: %v", err)
	}
	if tempPath == "" {
		t.Fatal("temp_path 不应为空")
	}

	got, err := s.Read(tempPath)
	if err != nil {
		t.Fatalf("Read 失败: %v", err)
	}
	if string(got) != "hello attachment" {
		t.Fatalf("内容不一致: %q", got)
	}

	// 移除后不可再读
	s.Remove(tempPath)
	if _, err := s.Read(tempPath); err == nil {
		t.Fatal("Remove 后 Read 应失败")
	}
}

func TestAttachmentTempStoreRejectsTraversal(t *testing.T) {
	s := newTestStore(t)

	// 放一个受害文件在 baseDir 之外
	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	attacks := []string{
		secret,                             // 绝对路径逃逸
		filepath.Join(secret, "..", "att"), // .. 逃逸后再进
		"../../" + filepath.Base(outsideDir) + "/secret.txt", // 相对路径逃逸
		"", // 空
	}
	for _, p := range attacks {
		if _, err := s.Read(p); err == nil {
			t.Errorf("Read(%q) 应被拒绝（路径穿越防护）", p)
		}
		if _, err := s.resolveWithinBase(p); err == nil {
			// Remove 的静默路径也依赖同一守卫
			t.Errorf("resolveWithinBase(%q) 应拒绝", p)
		}
	}
	// baseDir 自身：不算逃逸，但 Read 必须失败（是目录）
	if _, err := s.Read(s.baseDir); err == nil {
		t.Error("Read(目录) 应失败")
	}
}

func TestAttachmentTempStoreSizeLimit(t *testing.T) {
	s := newTestStore(t) // maxFileSize = 1024

	if _, err := s.Save(bytes.Repeat([]byte("x"), 1025)); err == nil {
		t.Fatal("Save 超限应失败")
	}
	// 落盘一个超限文件再 Read，也要被拒
	big := filepath.Join(t.TempDir(), "big.att")
	if err := os.WriteFile(big, bytes.Repeat([]byte("x"), 2048), 0o600); err != nil {
		t.Fatal(err)
	}
	// 直接放在 baseDir 内绕过 Save 的前置检查
	inside := filepath.Join(s.baseDir, "big.att")
	if err := os.WriteFile(inside, bytes.Repeat([]byte("x"), 2048), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(inside); err == nil {
		t.Fatal("Read 超限文件应失败")
	}
	_ = big
}

func TestAttachmentTempStoreRemoveExpired(t *testing.T) {
	s := newTestStore(t)

	oldPath, err := s.Save([]byte("old"))
	if err != nil {
		t.Fatal(err)
	}
	// 把修改时间拨回 2 小时前
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldPath, past, past); err != nil {
		t.Fatal(err)
	}
	freshPath, err := s.Save([]byte("fresh"))
	if err != nil {
		t.Fatal(err)
	}

	s.RemoveExpired(1 * time.Hour)

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("过期文件应被清理，stat err=%v", err)
	}
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("新鲜文件不应被清理: %v", err)
	}
}

func TestUploadAttachmentReturnsTempPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := newTestStore(t)
	h := &SendHandler{tempStore: s}

	// 构造 multipart 上传
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("attachment content")); err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/emails/attachments", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	h.UploadAttachment(c)

	if w.Code != http.StatusOK {
		t.Fatalf("上传应成功，got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"temp_path"`) {
		t.Fatalf("响应应含 temp_path: %s", w.Body.String())
	}
}
