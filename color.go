package shell

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type StyleCode int

const (
	// Text Colors (30-37)
	TextBlack StyleCode = 30 + iota
	TextRed
	TextGreen
	TextYellow
	TextBlue
	TextMagenta
	TextCyan
	TextWhite
)

const (
	// Background Colors (40-47)
	BgBlack StyleCode = 40 + iota
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

const (
	// Attributes
	Reset     StyleCode = 0
	Bold      StyleCode = 1
	Dim       StyleCode = 2
	Italic    StyleCode = 3
	Underline StyleCode = 4
	Blink     StyleCode = 5
	Reverse   StyleCode = 7
	Hidden    StyleCode = 8
)

// 预定义颜色组合
func Black(a ...any) string   { return Style(append([]any{TextBlack}, a...)...) }
func Red(a ...any) string     { return Style(append([]any{TextRed}, a...)...) }
func Green(a ...any) string   { return Style(append([]any{TextGreen}, a...)...) }
func Yellow(a ...any) string  { return Style(append([]any{TextYellow}, a...)...) }
func Blue(a ...any) string    { return Style(append([]any{TextBlue}, a...)...) }
func Magenta(a ...any) string { return Style(append([]any{TextMagenta}, a...)...) }
func Cyan(a ...any) string    { return Style(append([]any{TextCyan}, a...)...) }
func White(a ...any) string   { return Style(append([]any{TextWhite}, a...)...) }

func BBlack(a ...any) string   { return Style(append([]any{BgBlack, TextWhite}, a...)...) }
func BRed(a ...any) string     { return Style(append([]any{BgRed, TextWhite}, a...)...) }
func BGreen(a ...any) string   { return Style(append([]any{BgGreen, TextBlack}, a...)...) }
func BYellow(a ...any) string  { return Style(append([]any{BgYellow, TextBlack}, a...)...) }
func BBlue(a ...any) string    { return Style(append([]any{BgBlue, TextWhite}, a...)...) }
func BMagenta(a ...any) string { return Style(append([]any{BgMagenta, TextWhite}, a...)...) }
func BCyan(a ...any) string    { return Style(append([]any{BgCyan, TextBlack}, a...)...) }
func BWhite(a ...any) string   { return Style(append([]any{BgWhite, TextBlack}, a...)...) }

func Style(a ...any) string {
	if runtime.GOOS == "windows" {
		var out []any
		for _, v := range a {
			if _, ok := v.(StyleCode); !ok {
				out = append(out, v)
			}
		}
		return fmt.Sprint(out...)
	}

	var codes []string
	var content []any

	for _, v := range a {
		if code, ok := v.(StyleCode); ok {
			codes = append(codes, strconv.Itoa(int(code)))
		} else {
			content = append(content, v)
		}
	}

	if len(codes) == 0 {
		return fmt.Sprint(content...)
	}

	return fmt.Sprintf("\033[%sm%s\033[0m", strings.Join(codes, ";"), fmt.Sprint(content...))
}

// 常用重绘功能
func ClearCurrentLine() string { return "\r\033[2K" }

// StripANSI 移除字符串中的 ANSI 色码
func StripANSI(str string) string {
	const pattern = `\x1b\[[0-9;]*[a-zA-Z]`
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(str, "")
}
