// Package i18n is a tiny message catalog for API error texts. The English
// message string used at the error site doubles as the lookup key, so adding
// a new error in error_code/handlers doesn't need a separate "key constant"
// — just adding translations to the catalog is enough.
package i18n

import (
	"strings"
)

// Locale is a supported UI/message language tag.
type Locale string

// Supported locales; everything else falls back to EN.
const (
	EN Locale = "en"
	RU Locale = "ru"
)

// supported is the source-of-truth set; everything else falls back to EN.
var supported = map[Locale]bool{EN: true, RU: true}

// ParseAcceptLanguage picks the highest-priority supported locale from an
// `Accept-Language` header, falling back to EN. We don't do full RFC 4647
// negotiation — quality factors are honoured by relative order only.
func ParseAcceptLanguage(h string) Locale {
	if h == "" {
		return EN
	}

	for _, part := range strings.Split(h, ",") {
		tag := strings.ToLower(strings.TrimSpace(strings.SplitN(part, ";", 2)[0]))
		// strip region: "ru-RU" → "ru"
		if i := strings.Index(tag, "-"); i > 0 {
			tag = tag[:i]
		}

		if supported[Locale(tag)] {
			return Locale(tag)
		}
	}

	return EN
}

// T returns the localized message for key in locale, falling back to the key
// itself (which is the English message used at the error site).
func T(locale Locale, key string) string {
	cat, ok := catalogs[locale]
	if !ok {
		return key
	}

	if v, ok := cat[key]; ok {
		return v
	}

	return key
}

// catalogs: locale → (English-key → translation). Keep keys in sync with
// the messages used at error sites (validation.go, handlers, error_code.Map).
var catalogs = map[Locale]map[string]string{
	RU: {
		// shape / validation
		"invalid body":                           "некорректное тело запроса",
		"invalid params":                         "некорректные параметры",
		"invalid email":                          "неверный email",
		"password must be at least 8 characters": "пароль должен быть не короче 8 символов",
		"first name is required":                 "укажите имя",
		"init_data is required":                  "требуется init_data",
		// auth
		"invalid credentials":      "неверный email или пароль",
		"invalid init data":        "не удалось подтвердить вход через Telegram",
		"email already registered": "email уже зарегистрирован",
		// generic
		"unauthorized":   "не авторизовано",
		"not found":      "не найдено",
		"conflict":       "конфликт",
		"internal error": "внутренняя ошибка",
	},
}
