package locales

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"

	"github.com/egoist/mygo/i18n"
)

//go:embed messages/*.json
var files embed.FS

//go:embed aliases.json
var aliasData []byte
var aliases map[string]string
var Catalog *i18n.Catalog

func init() {
	sub, err := fs.Sub(files, "messages")
	if err != nil {
		panic(err)
	}
	Catalog, err = i18n.NewCatalog(sub, "en")
	if err != nil {
		panic(err)
	}
	i18n.Register(Catalog)
	if err = json.Unmarshal(aliasData, &aliases); err != nil {
		panic(err)
	}
}

func Text(locale, text string, parameters ...string) string {
	values := map[string]any{}
	for i := 0; i+1 < len(parameters); i += 2 {
		values[parameters[i]] = parameters[i+1]
	}
	if strings.HasPrefix(strings.ToLower(locale), "ru") {
		if key := aliases[text]; key != "" {
			return Catalog.Format("ru", key, values)
		}
	}
	for k, v := range values {
		text = strings.ReplaceAll(text, "{"+k+"}", v.(string))
	}
	return text
}
