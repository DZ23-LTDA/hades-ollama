//go:build !cgo && (windows || darwin)

package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ollama/ollama/app/store"
	"github.com/ollama/ollama/app/webview"
)

type Webview struct {
	port       int
	token      string
	webview    webview.WebView
	mutex      sync.Mutex
	onboarding atomic.Bool
	Store      *store.Store
}

func (w *Webview) Run(string) unsafe.Pointer { return nil }
func (w *Webview) Terminate() {
	w.onboarding.Store(false)
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.webview != nil {
		w.webview.Terminate()
		w.webview.Destroy()
		w.webview = nil
	}
}
func (w *Webview) OnboardingActive() bool { return w.onboarding.Load() }
func (w *Webview) IsRunning() bool        { w.mutex.Lock(); defer w.mutex.Unlock(); return w.webview != nil }

var _ = errors.New
