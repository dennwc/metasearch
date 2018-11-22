package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dennwc/metasearch/search"
)

const (
	defaultCountry  = "US"
	defaultLanguage = "en"
	languagesURL    = "https://" + defaultHostname + "/preferences?#languages"
)

var (
	countryHostname = map[string]string{
		"BG": "www.google.bg",     // Bulgaria
		"CZ": "www.google.cz",     // Czech Republic
		"DE": "www.google.de",     // Germany
		"DK": "www.google.dk",     // Denmark
		"AT": "www.google.at",     // Austria
		"CH": "www.google.ch",     // Switzerland
		"GR": "www.google.gr",     // Greece
		"AU": "www.google.com.au", // Australia
		"CA": "www.google.ca",     // Canada
		"GB": "www.google.co.uk",  // United Kingdom
		"ID": "www.google.co.id",  // Indonesia
		"IE": "www.google.ie",     // Ireland
		"IN": "www.google.co.in",  // India
		"MY": "www.google.com.my", // Malaysia
		"NZ": "www.google.co.nz",  // New Zealand
		"PH": "www.google.com.ph", // Philippines
		"SG": "www.google.com.sg", // Singapore
		// "US": "www.google.us",  // United States, redirect to .com
		"ZA": "www.google.co.za",  // South Africa
		"AR": "www.google.com.ar", // Argentina
		"CL": "www.google.cl",     // Chile
		"ES": "www.google.es",     // Spain
		"MX": "www.google.com.mx", // Mexico
		"EE": "www.google.ee",     // Estonia
		"FI": "www.google.fi",     // Finland
		"BE": "www.google.be",     // Belgium
		"FR": "www.google.fr",     // France
		"IL": "www.google.co.il",  // Israel
		"HR": "www.google.hr",     // Croatia
		"HU": "www.google.hu",     // Hungary
		"IT": "www.google.it",     // Italy
		"JP": "www.google.co.jp",  // Japan
		"KR": "www.google.co.kr",  // South Korea
		"LT": "www.google.lt",     // Lithuania
		"LV": "www.google.lv",     // Latvia
		"NO": "www.google.no",     // Norway
		"NL": "www.google.nl",     // Netherlands
		"PL": "www.google.pl",     // Poland
		"BR": "www.google.com.br", // Brazil
		"PT": "www.google.pt",     // Portugal
		"RO": "www.google.ro",     // Romania
		"RU": "www.google.ru",     // Russia
		"SK": "www.google.sk",     // Slovakia
		"SI": "www.google.si",     // Slovenia
		"SE": "www.google.se",     // Sweden
		"TH": "www.google.co.th",  // Thailand
		"TR": "www.google.com.tr", // Turkey
		"UA": "www.google.com.ua", // Ukraine
		// "CN": "www.google.cn",  // China, only from China ?
		"HK": "www.google.com.hk", // Hong Kong
		"TW": "www.google.com.tw", // Taiwan
	}
)

func (s *Service) Languages(ctx context.Context) ([]search.Language, error) {
	doc, err := s.GetHTML(ctx, languagesURL, nil)
	if err != nil {
		return nil, err
	}
	var out []search.Language
	doc.Find(`input[name="lang"]`).Each(func(_ int, sel *goquery.Selection) {
		code := sel.AttrOr("id", "")
		if code == "" {
			return
		}
		code = code[1:]
		if strings.HasPrefix(code, "xx-") {
			return // fake languages
		}
		title := sel.AttrOr("data-name", "")
		if title == "" {
			return
		}
		l, er := search.ParseLangCode(code)
		if er != nil {
			err = fmt.Errorf("cannot parse language %q %q: %v", title, code, er)
			return
		}
		out = append(out, search.Language{
			Code: l, Name: title,
		})
	})
	if err != nil {
		return nil, err
	} else if len(out) == 0 {
		return nil, fmt.Errorf("cannot parse languages list")
	}
	return out, nil
}

func (s *Service) Regions(ctx context.Context) ([]search.Region, error) {
	return nil, nil
}
