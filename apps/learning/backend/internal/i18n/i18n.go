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
		// learning agent
		"unknown agent or level":              "неизвестный агент или уровень",
		"API key is required":                 "укажите API-ключ",
		"API key must be 8 to 500 characters": "API-ключ должен быть от 8 до 500 символов",
		"the provider rejected this API key":  "провайдер не принял этот API-ключ",
		// learning agent completion
		"prompt is required": "укажите запрос",
		"prompt is too long": "запрос слишком длинный",
		"no AI agent chosen": "агент не выбран",
		"the stored API key cannot be read; enter it again": "сохранённый API-ключ не читается; введите его заново",
		"the provider rejected the AI agent's API key":      "провайдер не принял API-ключ агента",
		"the AI agent is temporarily unavailable":           "агент временно недоступен",
		"the AI agent did not return an answer":             "агент не вернул ответ",
		// learning picture and topic chat
		"a picture is required":                                                 "нужна картинка",
		"the picture must be a JPEG, PNG or GIF up to 6 MB":                     "картинка должна быть JPEG, PNG или GIF до 6 МБ",
		"the picture is too large: send a smaller one":                          "картинка слишком большая: отправьте поменьше",
		"the picture could not be read: try another one":                        "не удалось прочитать картинку: попробуйте другую",
		"reading a picture is not available for the language of this topic yet": "для языка этой темы чтение фото пока недоступно",
		"reading a picture is not available on this server":                     "чтение фото на этом сервере недоступно",
		"invalid topic id": "некорректный id темы",
		"topic not found":  "тема не найдена",
		"dialog not found": "диалог не найден",
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
