package m4

import (
	"bytes"
	"io"
	"unicode"
)

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenQuoted
	TokenOther
)

type Token struct {
	Type  TokenType
	Value string
}

type Lexer struct {
	stack       []*inputSource
	leftQuote   string
	rightQuote  string
	leftComment string
	rightComment string
}

type inputSource struct {
	buf []rune
	pos int
}

func (s *inputSource) readRune() (rune, bool) {
	if s.pos >= len(s.buf) {
		return 0, false
	}
	r := s.buf[s.pos]
	s.pos++
	return r, true
}

func (s *inputSource) unreadRune() {
	if s.pos > 0 {
		s.pos--
	}
}

func (s *inputSource) peekRune() (rune, bool) {
	if s.pos >= len(s.buf) {
		return 0, false
	}
	return s.buf[s.pos], true
}

func NewLexer() *Lexer {
	return &Lexer{
		stack:        make([]*inputSource, 0),
		leftQuote:    "`",
		rightQuote:   "'",
		leftComment:  "#",
		rightComment: "\n",
	}
}

// PushInput は評価対象のテキストを入力スタックに積みます（後入れ先出しで先頭から評価）
func (l *Lexer) PushInput(input string) {
	l.stack = append(l.stack, &inputSource{
		buf: []rune(input),
		pos: 0,
	})
}

func (l *Lexer) currentSource() *inputSource {
	if len(l.stack) == 0 {
		return nil
	}
	return l.stack[len(l.stack)-1]
}

func (l *Lexer) popSource() {
	if len(l.stack) > 0 {
		l.stack = l.stack[:len(l.stack)-1]
	}
}

func (l *Lexer) readRune() (rune, bool) {
	for len(l.stack) > 0 {
		src := l.currentSource()
		if r, ok := src.readRune(); ok {
			return r, true
		}
		l.popSource()
	}
	return 0, false
}

func (l *Lexer) unreadRune() {
	if len(l.stack) > 0 {
		l.currentSource().unreadRune()
	}
}

func (l *Lexer) matchPrefix(prefix string) bool {
	if prefix == "" {
		return false
	}
	runes := []rune(prefix)
	matched := 0
	for _, expected := range runes {
		r, ok := l.readRune()
		if !ok {
			break
		}
		if r != expected {
			l.unreadRune()
			break
		}
		matched++
	}
	if matched == len(runes) {
		return true
	}
	// 不一致なら読んだ分を巻き戻す
	for i := 0; i < matched; i++ {
		l.unreadRune()
	}
	return false
}

// NextToken は次のトークンを取得します
func (l *Lexer) NextToken() (Token, error) {
	for {
		r, ok := l.readRune()
		if !ok {
			return Token{Type: TokenEOF}, io.EOF
		}

		// クォート開始判定
		l.unreadRune()
		if l.matchPrefix(l.leftQuote) {
			quoted, err := l.readQuoted()
			return Token{Type: TokenQuoted, Value: quoted}, err
		}

		r, _ = l.readRune()

		// 識別子（アルファベットまたはアンダースコアで始まる単語）
		if isIdentStart(r) {
			var buf bytes.Buffer
			buf.WriteRune(r)
			for {
				nr, nok := l.readRune()
				if !nok {
					break
				}
				if isIdentPart(nr) {
					buf.WriteRune(nr)
				} else {
					l.unreadRune()
					break
				}
			}
			return Token{Type: TokenIdent, Value: buf.String()}, nil
		}

		return Token{Type: TokenOther, Value: string(r)}, nil
	}
}

func (l *Lexer) readQuoted() (string, error) {
	var buf bytes.Buffer
	depth := 1

	for {
		if l.matchPrefix(l.leftQuote) {
			depth++
			buf.WriteString(l.leftQuote)
			continue
		}
		if l.matchPrefix(l.rightQuote) {
			depth--
			if depth == 0 {
				return buf.String(), nil
			}
			buf.WriteString(l.rightQuote)
			continue
		}

		r, ok := l.readRune()
		if !ok {
			// クォートが閉じずに EOF に達した場合はそのまま返す
			return buf.String(), nil
		}
		buf.WriteRune(r)
	}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
