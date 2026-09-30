//go:build !cgo

package xgrammar

import "errors"

var ErrUnavailable = errors.New("xgrammar indisponível: este build não inclui CGO/native runtime")

type Compiler struct{}

func New(string, []string, int, []int32, int, int64) (*Compiler, error) { return nil, ErrUnavailable }
func (*Compiler) Path() string                                          { return "" }
func (*Compiler) Version() string                                       { return "unavailable" }
func (*Compiler) Compile(string) (*Matcher, error)                      { return nil, ErrUnavailable }
func (*Compiler) Close()                                                {}

type Matcher struct{}

func (*Matcher) Terminated() bool           { return true }
func (*Matcher) Fill([]int32) (bool, error) { return false, ErrUnavailable }
func (*Matcher) Accept(int32) error         { return ErrUnavailable }
func (*Matcher) Rollback(int) error         { return ErrUnavailable }
func (*Matcher) Close()                     {}
