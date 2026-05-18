package utils

import (
	"fmt"
	"strings"
	"unicode"
)

// GenerateSSOUsername генерирует безопасный username из данных SSO
// приоритет в трансляции Фамилия - ExternalID - fallback
func GenerateSSOUsername(fir, sir, externalID string) string {
	base := sir
	if base == "" {
		base = fir
	}

	if base != "" {
		username := transliterate(base)
		username = strings.ToLower(username)
		// Убираем всё кроме букв и цифр
		var clean strings.Builder
		for _, r := range username {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				clean.WriteRune(r)
			}
		}
		if clean.Len() > 0 {
			return clean.String()[:32]
		}
	}

	return fmt.Sprintf("user_%s", externalID)
}

// Простая транслитерация (кириллица - латиница)
func transliterate(s string) string {
	mapping := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo", 'ж': "zh",
		'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o",
		'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts",
		'ч': "ch", 'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo", 'Ж': "Zh",
		'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O",
		'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U", 'Ф': "F", 'Х': "Kh", 'Ц': "Ts",
		'Ч': "Ch", 'Ш': "Sh", 'Щ': "Shch", 'Ъ': "", 'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}
	var res strings.Builder
	for _, r := range s {
		if v, ok := mapping[r]; ok {
			res.WriteString(v)
		} else {
			res.WriteRune(r)
		}
	}
	return res.String()
}
