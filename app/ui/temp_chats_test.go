//go:build windows || darwin

package ui

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/app/store"
)

func TestTempChatStoreIsolatesCopies(t *testing.T) {
	temp := newTempChatStore()
	chat := store.NewChat("c1")
	chat.Messages = append(chat.Messages, store.NewMessage("user", "oi", nil))
	if err := temp.SetChat(*chat); err != nil {
		t.Fatal(err)
	}
	chat.Messages[0].Content = "mutated after save"

	got, err := temp.ChatWithOptions("c1", true)
	if err != nil || got.Messages[0].Content != "oi" {
		t.Fatalf("stored chat = %+v err=%v", got, err)
	}
	if err := temp.AppendMessage("c1", store.NewMessage("assistant", "a", nil)); err != nil {
		t.Fatal(err)
	}
	if err := temp.UpdateLastMessage("c1", store.NewMessage("assistant", "b", nil)); err != nil {
		t.Fatal(err)
	}
	got, _ = temp.ChatWithOptions("c1", true)
	if len(got.Messages) != 2 || got.Messages[1].Content != "b" {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if !temp.DeleteChat("c1") || temp.has("c1") {
		t.Fatal("delete should remove the chat")
	}
	if _, err := temp.ChatWithOptions("missing", false); err == nil {
		t.Fatal("expected not found")
	}
}

func TestAnonymousChatIsNeverPersisted(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"capabilities":["completion"]}`)
		case "/api/chat":
			w.Header().Set("Content-Type", "application/x-ndjson")
			io.WriteString(w, `{"model":"m","message":{"role":"assistant","content":"olá"},"done":false}`+"\n")
			io.WriteString(w, `{"model":"m","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop"}`+"\n")
		default:
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{}`)
		}
	}))
	defer upstream.Close()
	t.Setenv("OLLAMA_HOST", upstream.URL)

	st := &store.Store{DBPath: filepath.Join(t.TempDir(), "db.sqlite")}
	defer st.Close()
	if _, err := st.Chats(); err != nil {
		t.Skipf("store unavailable in this build: %v", err)
	}

	handler := (&Server{Token: "t", Store: st}).Handler()
	send := func(id, body string) []string {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/"+id, strings.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "token", Value: "t"})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d: %s", rr.Code, rr.Body.String())
		}
		var lines []string
		scanner := bufio.NewScanner(rr.Body)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return lines
	}

	lines := send("new", `{"model":"m","prompt":"oi","temporary":true}`)
	var created struct {
		ChatID string `json:"chatId"`
	}
	for _, line := range lines {
		if strings.Contains(line, "chat_created") {
			if err := json.Unmarshal([]byte(line), &created); err != nil {
				t.Fatal(err)
			}
		}
	}
	if created.ChatID == "" {
		t.Fatalf("no chat_created event in %v", lines)
	}

	// A follow-up turn stays anonymous without repeating the flag.
	send(created.ChatID, `{"model":"m","prompt":"de novo"}`)

	if _, err := st.ChatWithOptions(created.ChatID, false); err == nil {
		t.Fatal("anonymous chat was written to the store")
	}
	chats, err := st.Chats()
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 0 {
		t.Fatalf("history should be empty, got %d chats", len(chats))
	}
	kept, err := anonymousChats.ChatWithOptions(created.ChatID, false)
	if err != nil || len(kept.Messages) < 3 {
		t.Fatalf("anonymous chat = %+v err=%v", kept, err)
	}
	anonymousChats.DeleteChat(created.ChatID)
}
