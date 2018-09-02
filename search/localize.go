package search

import (
	"strings"
)

type Language struct {
	Code LangCode
	Name string
}

func ParseLangCode(s string) (LangCode, error) {
	i := strings.Index(s, "-")
	if i < 0 {
		return LangCode{Code: s}, nil
	}
	return LangCode{Code: s[:i], Dia: s[i+1:]}, nil
}

type LangCode struct {
	Code string
	Dia  string
}

func (l LangCode) Country() CountryCode {
	if l.Zero() {
		return CountryCode{}
	}
	var c CountryCode
	copy(c[:], l.Code)
	return c
}
func (l LangCode) Zero() bool {
	return l == (LangCode{})
}
func (l LangCode) String() string {
	if l.Dia == "" {
		return l.Code
	}
	return l.Code + "-" + l.Dia
}

type CountryCode [2]byte

func (c CountryCode) Zero() bool {
	return c == (CountryCode{})
}
func (c CountryCode) String() string {
	if c.Zero() {
		return ""
	}
	return string(c[:])
}
