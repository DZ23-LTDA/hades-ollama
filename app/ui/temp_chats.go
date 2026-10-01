//go:build windows || darwin

package ui

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/ollama/ollama/app/store"
	"github.com/ollama/ollama/app/types/not"
)

// chatPersistence is the subset of the store used while streaming a chat.
// Anonymous chats use an in-memory implementation so nothing reaches disk.
type chatPersistence interface {
	ChatWithOptions(id string, loadAttachmentData bool) (*store.Chat, error)
	SetChat(chat store.Chat) error
	UpdateLastMessage(chatID string, message store.Message) error
	AppendMessage(chatID string, message store.Message) error
	UpdateChatBrowserState(chatID string, state json.RawMessage) error
}

// tempChatStore keeps anonymous chats in memory for the life of the app.
// They are never listed in history and disappear when the app exits.
type tempChatStore struct {
	mu    sync.Mutex
	chats map[string]store.Chat
}

// anonymousChats is shared by the single UI server in the desktop app.
var anonymousChats = newTempChatStore()

func newTempChatStore() *tempChatStore {
	return &tempChatStore{chats: map[string]store.Chat{}}
}

func cloneChat(chat store.Chat) store.Chat {
	chat.Messages = slices.Clone(chat.Messages)
	chat.BrowserState = slices.Clone(chat.BrowserState)
	return chat
}

func (t *tempChatStore) has(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.chats[id]
	return ok
}

func (t *tempChatStore) ChatWithOptions(id string, _ bool) (*store.Chat, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	chat, ok := t.chats[id]
	if !ok {
		return nil, fmt.Errorf("%w: chat %s", not.Found, id)
	}
	clone := cloneChat(chat)
	return &clone, nil
}

func (t *tempChatStore) SetChat(chat store.Chat) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.chats[chat.ID] = cloneChat(chat)
	return nil
}

func (t *tempChatStore) UpdateLastMessage(chatID string, message store.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	chat, ok := t.chats[chatID]
	if !ok || len(chat.Messages) == 0 {
		return fmt.Errorf("%w: chat %s", not.Found, chatID)
	}
	chat.Messages = slices.Clone(chat.Messages)
	chat.Messages[len(chat.Messages)-1] = message
	t.chats[chatID] = chat
	return nil
}

func (t *tempChatStore) AppendMessage(chatID string, message store.Message) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	chat, ok := t.chats[chatID]
	if !ok {
		return fmt.Errorf("%w: chat %s", not.Found, chatID)
	}
	chat.Messages = append(slices.Clone(chat.Messages), message)
	t.chats[chatID] = chat
	return nil
}

func (t *tempChatStore) UpdateChatBrowserState(chatID string, state json.RawMessage) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	chat, ok := t.chats[chatID]
	if !ok {
		return fmt.Errorf("%w: chat %s", not.Found, chatID)
	}
	chat.BrowserState = slices.Clone(state)
	t.chats[chatID] = chat
	return nil
}

func (t *tempChatStore) DeleteChat(id string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.chats[id]
	delete(t.chats, id)
	return ok
}
