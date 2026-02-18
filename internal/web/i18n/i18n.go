package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gaetanlhf/ZIMServer/internal/web/utils"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

type Translations map[string]interface{}

type I18n struct {
	translations map[string]Translations
	matcher      language.Matcher
	mu           sync.RWMutex
	tags         []language.Tag
}

type LanguageInfo struct {
	Code string
	Name string
}

func New() (*I18n, error) {
	i := &I18n{
		translations: make(map[string]Translations),
	}

	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			langCode := strings.TrimSuffix(entry.Name(), ".json")
			content, err := localesFS.ReadFile("locales/" + entry.Name())
			if err != nil {
				return nil, err
			}

			var t Translations
			if err := json.Unmarshal(content, &t); err != nil {
				return nil, fmt.Errorf("failed to parse locale %s: %w", langCode, err)
			}

			i.translations[langCode] = t
			i.tags = append(i.tags, language.Make(langCode))
		}
	}

	i.matcher = language.NewMatcher(i.tags)
	return i, nil
}

func (i *I18n) GetLanguage(r *http.Request) string {
	if cookie, err := r.Cookie("zimserver_language"); err == nil {
		if cookie.Value != "auto" && cookie.Value != "" {
			for _, tag := range i.tags {
				base, _ := tag.Base()
				if base.String() == cookie.Value {
					return cookie.Value
				}
			}
		}
	}

	accept := r.Header.Get("Accept-Language")
	tags, _, _ := language.ParseAcceptLanguage(accept)
	tag, _, _ := i.matcher.Match(tags...)
	base, _ := tag.Base()
	return base.String()
}

func (i *I18n) Translate(lang, key string) interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	parts := strings.Split(key, ".")

	if t, ok := i.translations[lang]; ok {
		if val := i.findValue(t, parts); val != nil {
			return val
		}
	}

	if lang != "en" {
		if t, ok := i.translations["en"]; ok {
			if val := i.findValue(t, parts); val != nil {
				return val
			}
		}
	}

	return key
}

func (i *I18n) findValue(t Translations, parts []string) interface{} {
	var current interface{} = t
	for _, part := range parts {
		if m, ok := current.(Translations); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else if m, ok := current.(map[string]interface{}); ok {
			if val, exists := m[part]; exists {
				current = val
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func (i *I18n) GetAvailableLanguages() []LanguageInfo {
	i.mu.RLock()
	defer i.mu.RUnlock()

	languages := make([]LanguageInfo, 0, len(i.tags))
	for _, tag := range i.tags {
		base, _ := tag.Base()
		code := base.String()
		name := utils.GetLanguageName(code)
		languages = append(languages, LanguageInfo{
			Code: code,
			Name: name,
		})
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Name < languages[j].Name
	})

	return languages
}
