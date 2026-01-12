package utils

import (
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

func GetLanguageName(code string) string {
	tag, err := language.Parse(code)
	if err != nil {
		return code
	}

	name := display.Self.Name(tag)
	return CapitalizeFirst(name)
}

func GetLanguageCode(code string) string {
	if len(code) == 2 {
		return strings.ToUpper(code)
	}

	tag, err := language.Parse(code)
	if err != nil {
		return strings.ToUpper(code)
	}

	base, _ := tag.Base()
	return strings.ToUpper(base.String())
}
