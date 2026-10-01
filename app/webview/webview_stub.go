//go:build !cgo

package webview

import (
	"errors"
	"unsafe"
)

type Hint int

const (
	HintNone Hint = iota
	HintFixed
	HintMin
	HintMax
)

type WebView interface {
	Run()
	Terminate()
	Dispatch(func())
	Destroy()
	Window() unsafe.Pointer
	Navigate(string)
	SetHtml(string)
	SetTitle(string)
	SetSize(int, int, Hint)
	Init(string)
	Eval(string)
	Bind(string, interface{}) error
	Unbind(string) error
	SetZoom(float64)
	GetZoom() float64
}
type unavailableWebView struct{}

func New(bool) WebView                       { return &unavailableWebView{} }
func NewWindow(bool, unsafe.Pointer) WebView { return &unavailableWebView{} }
func (*unavailableWebView) Run()             {}
func (*unavailableWebView) Terminate()       {}
func (w *unavailableWebView) Dispatch(f func()) {
	if f != nil {
		f()
	}
}
func (*unavailableWebView) Destroy()               {}
func (*unavailableWebView) Window() unsafe.Pointer { return nil }
func (*unavailableWebView) Navigate(string)        {}
func (*unavailableWebView) SetHtml(string)         {}
func (*unavailableWebView) SetTitle(string)        {}
func (*unavailableWebView) SetSize(int, int, Hint) {}
func (*unavailableWebView) Init(string)            {}
func (*unavailableWebView) Eval(string)            {}
func (*unavailableWebView) Bind(string, interface{}) error {
	return errors.New("webview indisponível sem CGO")
}

func (*unavailableWebView) Unbind(string) error { return errors.New("webview indisponível sem CGO") }
func (*unavailableWebView) SetZoom(float64)     {}
func (*unavailableWebView) GetZoom() float64    { return 1 }
