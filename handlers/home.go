package handlers

import (
	"net/http"

	"github.com/patrickdappollonio/pdbotapp/utils"
)

func Home(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, utils.MustResolvePath("templates/home.tmpl"))
}
