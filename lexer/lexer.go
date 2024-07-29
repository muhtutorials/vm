package lexer

import "vm/token"

// Lexer is a lexer for VM
type Lexer struct {
	pos        int    // current character position
	nextPos    int    // next character position
	char       rune   // current character
	characters []rune // rune slice of input string
}

// New creates a Lexer instance from string input
func New(input string) *Lexer {
	l := &Lexer{characters: []rune(input)}
	// prime the pump
	l.readChar()
	return l
}

// readChar reads next character
func (l *Lexer) readChar() {
	if l.nextPos >= len(l.characters) {
		l.char = rune(0)
	} else {
		l.char = l.characters[l.nextPos]
	}
	l.pos = l.nextPos
	l.nextPos++
}

// NextToken reads the next token, skipping the white space
func (l *Lexer) NextToken() token.Token {
	var tok token.Token
	l.skipWhitespace()

	// skip single-line comments unless they are immediately followed by a number,
	// because the registers are "#N"
	if l.char == '#' {
		if !isDigit(l.peekChar()) {
			l.skipComment()
			return l.NextToken()
		}
	}

	switch l.char {
	case ',':
		tok = newToken(token.COMMA, l.char)
	case '"':
		tok.Type = token.STR
		tok.Literal = l.readStr()
	case ':':
		tok.Type = token.LABEL
		tok.Literal = l.readLabel()
	case rune(0):
		tok.Type = token.EOF
		tok.Literal = ""
	default:
		if isDigit(l.char) {
			return l.readDecimal()
		}

		tok.Literal = l.readIdentifier()
		tok.Type = token.LookupIdentifier(tok.Literal)
		return tok
	}

	l.readChar()
	return tok
}

func newToken(typ token.Type, char rune) token.Token {
	return token.Token{
		Type:    typ,
		Literal: string(char),
	}
}

func (l *Lexer) skipWhitespace() {
	for isWhiteSpace(l.char) {
		l.readChar()
	}
}

func (l *Lexer) skipComment() {
	for l.char != '\n' && l.char != rune(0) {
		l.readChar()
	}
}

func (l *Lexer) peekChar() rune {
	if l.nextPos >= len(l.characters) {
		return rune(0)
	}
	return l.characters[l.nextPos]
}

func (l *Lexer) readStr() string {
	var str string

	for {
		l.readChar()
		if l.char == '"' {
			break
		}

		// handle \n, \r, \t, \", etc.
		// todo: why is it double backslash?
		if l.char == '\\' {
			l.readChar()

			if l.char == 'n' {
				l.char = '\n'
			}
			if l.char == 't' {
				l.char = '\t'
			}
			if l.char == 'r' {
				l.char = '\r'
			}
			if l.char == '"' {
				l.char = '"'
			}
			if l.char == '\\' {
				l.char = '\\'
			}
		}
		str += string(l.char)
	}
	return str
}

func (l *Lexer) readLabel() string {
	return l.readUntilWhitespace()
}

func (l *Lexer) readUntilWhitespace() string {
	pos := l.pos
	// 7 8 9 10 11
	// : f o r  sp
	for !isWhiteSpace(l.char) && l.char != rune(0) {
		l.readChar()
	}
	return string(l.characters[pos:l.pos])
}

func (l *Lexer) readDecimal() token.Token {
	integer := l.readNumber()
	if isWhiteSpace(l.char) || isEmpty(l.char) || l.char == ',' {
		return token.Token{Type: token.INT, Literal: integer}
	}

	illegalPart := l.readUntilWhitespace()

	return token.Token{Type: token.ILLEGAL, Literal: integer + illegalPart}
}

func (l *Lexer) readNumber() string {
	pos := l.pos
	for isHexDigit(l.char) {
		l.readChar()
	}
	return string(l.characters[pos:l.pos])
}

func (l *Lexer) readIdentifier() string {
	pos := l.pos
	for isIdentifier(l.char) {
		l.readChar()
	}
	return string(l.characters[pos:l.pos])
}

// isWhiteSpace checks if a character is a whitespace
func isWhiteSpace(char rune) bool {
	return char == ' ' || char == '\n' || char == '\t' || char == '\r'
}

func isEmpty(char rune) bool {
	return char == rune(0)
}

func isIdentifier(char rune) bool {
	return char != ',' && !isWhiteSpace(char) && !isEmpty(char)
}

// isDigit checks if a character is a digit
func isDigit(char rune) bool {
	return '0' <= char && char <= '9'
}

func isHexDigit(char rune) bool {
	if isDigit(char) {
		return true
	}
	if 'a' <= char && char <= 'f' {
		return true
	}
	if 'A' <= char && char <= 'F' {
		return true
	}
	if 'x' == char || 'X' == char {
		return true
	}
	return false
}
