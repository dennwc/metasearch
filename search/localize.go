package search

import (
	"golang.org/x/text/language"
)

type Language struct {
	Code LangCode
	Name string
}

func ParseLangCode(s string) (LangCode, error) {
	return language.Parse(s)
}

func MustParseLangCode(s string) LangCode {
	code, err := language.Parse(s)
	if err != nil {
		panic(err)
	}
	return code
}

type LangCode = language.Tag

type Region struct {
	Code RegionCode
	Name string
}

func ParseRegionCode(s string) (RegionCode, error) {
	return language.ParseRegion(s)
}

func MustParseRegionCode(s string) RegionCode {
	code, err := language.ParseRegion(s)
	if err != nil {
		panic(err)
	}
	return code
}

type RegionCode = language.Region
