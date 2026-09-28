package validator

import (
	"reflect"
	"strings"

	frameworkErrors "github.com/Gongaji-Apps/GONGAJI-FRAMEWORK/errors"
	"github.com/go-playground/validator/v10"
)

type Translator struct {
	Lang string
}

func NewTranslator(lang string) *Translator {
	return &Translator{Lang: lang}
}

// isEnglish: header Accept-Language diawali "en" (mis. "en", "en-US,en;q=0.9").
func isEnglish(lang string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "en")
}

func (t *Translator) getMessages() map[string]string {
	if isEnglish(t.Lang) {
		return messagesEN
	}
	return messagesID
}

func (t *Translator) Translate(e validator.FieldError) string {
	messages := t.getMessages()

	tag := e.Tag()
	msg, ok := "", false
	if e.Kind() == reflect.String {
		msg, ok = messages[tag+".str"]
	}
	if !ok {
		msg, ok = messages[tag]
	}
	if !ok {
		msg = messages["_default"]
	}

	msg = strings.ReplaceAll(msg, "{field}", frameworkErrors.FieldLabel(e.Field()))
	msg = strings.ReplaceAll(msg, "{param}", strings.ReplaceAll(e.Param(), " ", ", "))

	return msg
}
