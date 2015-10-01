package rdf

import (
	"errors"
	"log"
)

type Token struct {
	value        string
	isUri        bool
	isLiteral    bool
	isNamespaced bool
}

type TurtleParser struct {
	index   int
	text    string
	chars   []rune
	length  int
	triples []Triple
	err     error
}

func NewTurtleParser(text string) TurtleParser {
	// Convert the original string to an array of unicode runes.
	// This allows us to iterate on it as if it was an array
	// of ASCII chars even if there are Unicode characters on it
	// that use 2-4 bytes.
	chars := stringToRunes(text)
	parser := TurtleParser{text: text, chars: chars}
	return parser
}

func (parser *TurtleParser) Parse() error {
	parser.err = nil
	parser.index = 0
	parser.length = len(parser.chars)
	for parser.canRead() {
		triple, err := parser.GetNextTriple()
		if err != nil {
			parser.err = err
			break
		}
		parser.triples = append(parser.triples, triple)
		parser.advanceWhiteSpace()
	}
	return parser.err
}

func (parser TurtleParser) Triples() []Triple {
	return parser.triples
}

func (parser *TurtleParser) GetNextTriple() (Triple, error) {
	var subject, predicate, object Token
	var err error
	var triple Triple

	log.Printf("GetNextTriple %d", parser.index)
	subject, err = parser.GetNextToken()
	if err == nil {
		predicate, err = parser.GetNextToken()
		if err == nil {
			object, err = parser.GetNextToken()
			if err == nil {
				err = parser.AdvanceTriple()
				if err == nil {
					triple = NewTriple(subject.value, predicate.value, object.value, object.isLiteral)
				}
			}
		}
	}
	return triple, err
}

func (parser *TurtleParser) GetNextToken() (Token, error) {
	var err error
	var isLiteral, isUri, isNamespaced bool
	var value string

	parser.advanceWhiteSpace()
	if !parser.canRead() {
		return Token{}, errors.New("No token found")
	}

	firstChar := parser.char()
	switch {
	case firstChar == '<':
		isUri = true
		value, err = parser.parseUri()
	case firstChar == '"':
		isLiteral = true
		value, err = parser.parseString()
	case parser.isNamespacedChar():
		isNamespaced = true
		value = parser.parseNamespacedValue()
	default:
		return Token{}, errors.New("Invalid first character")
	}

	if err != nil {
		return Token{}, err
	}

	parser.advance()
	token := Token{value: value, isUri: isUri, isLiteral: isLiteral, isNamespaced: isNamespaced}
	return token, nil
}

// Advances the index to the beginning of the next triple.
func (parser *TurtleParser) AdvanceTriple() error {
	for parser.canRead() {
		if parser.char() == '.' {
			break
		}
		if parser.isWhiteSpaceChar() {
			parser.advance()
			continue
		}
		return errors.New("Triple did not end with a period.")
	}
	parser.advance()
	return nil
}

// Advances the index to the next character.
func (parser *TurtleParser) advance() {
	if parser.canRead() {
		parser.index++
	}
}

func (parser *TurtleParser) advanceWhiteSpace() {
	for parser.canRead() {
		if parser.atLastChar() || !parser.isWhiteSpaceChar() {
			break
		}
		parser.advance()
	}
}

func (parser TurtleParser) atLastChar() bool {
	return parser.index == (parser.length - 1)
}

func (parser *TurtleParser) canRead() bool {
	if len(parser.chars) == 0 {
		return false
	}
	return parser.index < len(parser.chars)
}

func (parser TurtleParser) char() rune {
	return parser.chars[parser.index]
}

func (parser TurtleParser) isNamespacedChar() bool {
	char := parser.char()
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		(char == ':')
}

func (parser TurtleParser) isUriChar() bool {
	char := parser.char()
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		(char == ':') || (char == '/') ||
		(char == '%') || (char == '#') ||
		(char == '+')
}

func (parser TurtleParser) isWhiteSpaceChar() bool {
	char := parser.char()
	return char == ' ' || char == '\t' || char == '\n' || char == '\r'
}

// Extracts a value in the form xx:yy or xx
func (parser *TurtleParser) parseNamespacedValue() string {
	start := parser.index
	parser.advance()
	for parser.canRead() {
		if parser.isNamespacedChar() {
			parser.advance()
			continue
		} else {
			break
		}
	}
	return string(parser.chars[start:parser.index])
}

// Extracts a value in quotes, e.g. "hello"
func (parser *TurtleParser) parseString() (string, error) {
	// TODO: Move the advance outside of here.
	// We should already be inside the URI.
	start := parser.index
	parser.advance()
	for parser.canRead() {
		if parser.char() == '"' {
			uri := string(parser.chars[start : parser.index+1])
			return uri, nil
		}
		parser.advance()
	}
	return "", errors.New("String did not end with \"")
}

// Extracts an URI in the form <hello>
func (parser *TurtleParser) parseUri() (string, error) {
	// TODO: Move the advance outside of here.
	// We should already be inside the URI.
	start := parser.index
	parser.advance()
	for parser.canRead() {
		if parser.char() == '>' {
			uri := string(parser.chars[start : parser.index+1])
			return uri, nil
		}
		if !parser.isUriChar() {
			return "", errors.New("Invalid character in URI")
		}
		parser.advance()
	}
	return "", errors.New("URI did not end with >")
}

func stringToRunes(text string) []rune {
	var chars []rune
	for _, c := range text {
		chars = append(chars, c)
	}
	return chars
}
