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

type LangCode = language.Tag

type Region struct {
	Code RegionCode
	Name string
}

func ParseRegionCode(s string) (RegionCode, error) {
	return language.ParseRegion(s)
}

type RegionCode = language.Region
