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
	ES Locale = "es"
)

// supported is the source-of-truth set; everything else falls back to EN.
var supported = map[Locale]bool{EN: true, RU: true, ES: true}

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
	ES: {
		"invalid body":                                      "cuerpo de la solicitud no válido",
		"invalid params":                                    "parámetros no válidos",
		"invalid email":                                     "correo no válido",
		"password must be at least 8 characters":            "la contraseña debe tener al menos 8 caracteres",
		"first name is required":                            "indica tu nombre",
		"init_data is required":                             "se requiere init_data",
		"unknown agent or level":                            "agente o nivel desconocido",
		"API key is required":                               "indica la clave de API",
		"API key must be 8 to 500 characters":               "la clave de API debe tener de 8 a 500 caracteres",
		"the provider rejected this API key":                "el proveedor rechazó esta clave de API",
		"prompt is required":                                "indica la consulta",
		"prompt is too long":                                "la consulta es demasiado larga",
		"no AI agent chosen":                                "no se ha elegido ningún agente",
		"the stored API key cannot be read; enter it again": "la clave de API guardada no se puede leer; introdúcela de nuevo",
		"the provider rejected the AI agent's API key":      "el proveedor rechazó la clave de API del agente",
		"the AI agent is temporarily unavailable":           "el agente no está disponible temporalmente",
		"the AI agent did not return an answer":             "el agente no devolvió respuesta",
		"a picture is required":                             "se necesita una imagen",
		"the picture must be a JPEG, PNG or GIF up to 6 MB": "la imagen debe ser JPEG, PNG o GIF de hasta 6 MB",
		"the picture is too large: send a smaller one":      "la imagen es demasiado grande: envía una más pequeña",
		"the picture could not be read: try another one":    "no se pudo leer la imagen: prueba con otra",
		"reading a picture is not available for the language of this topic yet": "la lectura de fotos aún no está disponible para el idioma de este tema",
		"reading a picture is not available on this server":                     "la lectura de fotos no está disponible en este servidor",
		"invalid topic id":         "id de tema no válido",
		"topic not found":          "tema no encontrado",
		"dialog not found":         "diálogo no encontrado",
		"invalid credentials":      "correo o contraseña incorrectos",
		"invalid init data":        "no se pudo confirmar el inicio de sesión con Telegram",
		"email already registered": "el correo ya está registrado",
		"unauthorized":             "no autorizado",
		"not found":                "no encontrado",
		"conflict":                 "conflicto",
		"internal error":           "error interno",
	},
}
