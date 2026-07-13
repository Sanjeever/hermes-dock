package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostFileHandlersWriteReadListAndMove(t *testing.T) {
	app := NewApp()
	root := t.TempDir()
	source := filepath.Join(root, "nested", "hello.txt")

	writeResponse := callHostJSONHandler(t, app.handleHostFileWrite, map[string]interface{}{
		"path":           source,
		"content_base64": base64.StdEncoding.EncodeToString([]byte("你好，Hermes")),
		"create_parents": true,
		"overwrite":      true,
	})
	if writeResponse.Code != http.StatusOK {
		t.Fatalf("write status = %d: %s", writeResponse.Code, writeResponse.Body.String())
	}

	readResponse := callHostJSONHandler(t, app.handleHostFileRead, map[string]interface{}{"path": source})
	if readResponse.Code != http.StatusOK {
		t.Fatalf("read status = %d: %s", readResponse.Code, readResponse.Body.String())
	}
	var readResult struct {
		Content string `json:"content_base64"`
	}
	if err := json.Unmarshal(readResponse.Body.Bytes(), &readResult); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(readResult.Content)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "你好，Hermes" {
		t.Fatalf("read content = %q", decoded)
	}

	listResponse := callHostJSONHandler(t, app.handleHostFileList, map[string]interface{}{"path": filepath.Dir(source)})
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "hello.txt") {
		t.Fatalf("list response = %d: %s", listResponse.Code, listResponse.Body.String())
	}

	target := filepath.Join(root, "moved", "hello.txt")
	moveResponse := callHostJSONHandler(t, app.handleHostFileMove, map[string]interface{}{
		"source":         source,
		"target":         target,
		"create_parents": true,
		"overwrite":      false,
	})
	if moveResponse.Code != http.StatusOK {
		t.Fatalf("move status = %d: %s", moveResponse.Code, moveResponse.Body.String())
	}

	statResponse := callHostJSONHandler(t, app.handleHostFileStat, map[string]interface{}{"path": target})
	if statResponse.Code != http.StatusOK || !strings.Contains(statResponse.Body.String(), `"size":15`) {
		t.Fatalf("stat response = %d: %s", statResponse.Code, statResponse.Body.String())
	}
}

func TestHostFileWriteRejectsExistingFileWithoutOverwrite(t *testing.T) {
	app := NewApp()
	path := filepath.Join(t.TempDir(), "existing.txt")
	first := map[string]interface{}{
		"path":           path,
		"content_base64": base64.StdEncoding.EncodeToString([]byte("first")),
		"overwrite":      true,
	}
	if response := callHostJSONHandler(t, app.handleHostFileWrite, first); response.Code != http.StatusOK {
		t.Fatalf("first write status = %d: %s", response.Code, response.Body.String())
	}
	second := map[string]interface{}{
		"path":           path,
		"content_base64": base64.StdEncoding.EncodeToString([]byte("second")),
		"overwrite":      false,
	}
	if response := callHostJSONHandler(t, app.handleHostFileWrite, second); response.Code != http.StatusConflict {
		t.Fatalf("second write status = %d, want %d: %s", response.Code, http.StatusConflict, response.Body.String())
	}
}

func TestAbsoluteHostPathRejectsRelativePath(t *testing.T) {
	if _, err := absoluteHostPath("relative/file.txt"); err == nil {
		t.Fatal("expected relative host path to be rejected")
	}
}

func callHostJSONHandler(t *testing.T, handler http.HandlerFunc, value interface{}) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "http://host.test", strings.NewReader(string(data)))
	response := httptest.NewRecorder()
	handler(response, request)
	return response
}
