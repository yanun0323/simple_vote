package homepage

import (
	"net/http"

	"main/internal/keyword"
	"main/internal/utils"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func HomePage() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			return
		}

		w.Write(utils.ReadFile("./internal/resource/homepage.html", map[string]string{
			keyword.UID:  uuid.NewString(),
			keyword.Host: viper.GetString("host"),
		}))
	}
}
