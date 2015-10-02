package sqlbind

import "unicode"

const (
	typeSQL = iota
	typePlaceholder
	typeVariable
	typeNames
	typeValues
	typeNameValue
	typeSeparator
)

type part struct {
	t    int
	data string
}

type decoded struct {
	parts []part
}

type decodeState struct {
	step func(*decodeState, string) int
	str  []byte
	err  error
}

func decode(str string) *decoded {
	c := &decoded{parts: []part{}}
	d := newDecodeState()
	cur := typeSQL
	start := 0
	for i := range str {
		next := d.step(d, str[i:])
		if next != cur && cur != typeSeparator && i > 0 {
			c.parts = append(c.parts, part{t: cur, data: str[start:i]})
			start = i
		}
		if cur == typeSeparator {
			start = i
		}
		cur = next
	}
	if len(str) > start && cur != typeSeparator {
		c.parts = append(c.parts, part{t: cur, data: str[start:]})
	}
	return c
}

func newDecodeState() *decodeState {
	return &decodeState{step: scanSQL}
}

func scanSQL(d *decodeState, str string) int {
	switch str[0] {
	case ':':
		d.step = scanColon
		return typeSeparator
	case '{':
		d.step = scanVariable
		return typeSeparator
	case '"':
		d.step = scanString
		return typeSQL
	}
	return typeSQL
}

func scanString(d *decodeState, str string) int {
	switch str[0] {
	case '"':
		d.step = scanSQL
		return typeSQL
	}
	return typeSQL
}

func scanColon(d *decodeState, str string) int {
	if str[0] == ':' {
		return scanDoubleColon(d, str)
	}
	d.step = scanPlaceholder
	return d.step(d, str)
}

var allowedPlaceholderRunes = []*unicode.RangeTable{unicode.Letter, unicode.Digit}

func scanPlaceholder(d *decodeState, str string) int {
	// TODO : try DecodeRuneInString to enable UTF8 placeholder
	if !unicode.IsOneOf(allowedPlaceholderRunes, rune(str[0])) && str[0] != '_' {
		d.step = scanSQL
		return typeSQL

	}
	return typePlaceholder
}

func scanDoubleColon(d *decodeState, str string) int {
	switch {
	case len(str) >= 13 && str[:13] == ":name=::value":
		d.step = skipN(13, typeNameValue)
	case len(str) >= 6 && str[:6] == ":names":
		d.step = skipN(6, typeNames)
	case len(str) >= 7 && str[:7] == ":values":
		d.step = skipN(7, typeValues)
	default:
		d.step = scanSQL
		return typeSQL
	}
	return d.step(d, str)
}

func scanVariable(d *decodeState, str string) int {
	switch str[0] {
	case '}':
		d.step = scanSQL
		return typeSeparator
	}
	return typeVariable
}

func skipN(n int, t int) func(d *decodeState, str string) int {
	return func(d *decodeState, str string) int {
		n--
		if n >= 0 {
			return t
		} else {
			d.step = scanSQL
			return typeSQL
		}
	}
}
