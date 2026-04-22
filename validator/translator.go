package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type Translator struct {
	Lang string
}

func NewTranslator(lang string) *Translator {
	return &Translator{Lang: lang}
}

func (t *Translator) getMessages() map[string]string {
	switch t.Lang {
	case "en":
		return messagesEN
	default:
		return messagesID
	}
}

func (t *Translator) Translate(e validator.FieldError) string {
	messages := t.getMessages()

	msg, ok := messages[e.Tag()]
	if !ok {
		return "invalid value"
	}

	field := strings.ToLower(e.Field())
	param := e.Param()

	msg = strings.ReplaceAll(msg, "{field}", field)
	msg = strings.ReplaceAll(msg, "{param}", param)

	return msg
}
