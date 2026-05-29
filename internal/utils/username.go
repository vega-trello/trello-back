package utils

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

var adjectives = []string{
	"swift", "bright", "silent", "bold", "calm",
	"fierce", "gentle", "wild", "brave", "clever",
	"dark", "eager", "fancy", "glad", "jolly",
}

var nouns = []string{
	"fox", "hawk", "wolf", "bear", "lynx",
	"crow", "deer", "otter", "eagle", "tiger",
	"shark", "raven", "panda", "cobra", "falcon",
}

func GenerateRandomUsername() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	noun := nouns[rand.Intn(len(nouns))]
	num := rand.Intn(1000)
	return adj + "_" + noun + "_" + strconv.Itoa(num)
}

func GenerateSSOUsername(fir, sir, uai string) string {
	base := strings.ToLower(strings.TrimSpace(sir))

	base = transliterateCyrillic(base)

	re := regexp.MustCompile(`[^a-z0-9\s_-]`)
	clean := re.ReplaceAllString(base, "")
	clean = strings.ReplaceAll(clean, " ", "_")

	reMulti := regexp.MustCompile(`[_-]{2,}`)
	clean = reMulti.ReplaceAllString(clean, "_")
	clean = strings.Trim(clean, "_-")

	if clean == "" {
		clean = "user_" + uai
	}

	maxLen := 32
	if len(clean) < maxLen {
		maxLen = len(clean)
	}
	return clean[:maxLen]
}

func transliterateCyrillic(s string) string {
	replacements := map[rune]string{
		'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
		'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
		'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
		'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "",
		'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
		'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "Yo",
		'Ж': "Zh", 'З': "Z", 'И': "I", 'Й': "Y", 'К': "K", 'Л': "L", 'М': "M",
		'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T", 'У': "U",
		'Ф': "F", 'Х': "H", 'Ц': "Ts", 'Ч': "Ch", 'Ш': "Sh", 'Щ': "Sch", 'Ъ': "",
		'Ы': "Y", 'Ь': "", 'Э': "E", 'Ю': "Yu", 'Я': "Ya",
	}

	var result strings.Builder
	result.Grow(len(s) * 2)

	for _, r := range s {
		if repl, ok := replacements[r]; ok {
			result.WriteString(repl)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
