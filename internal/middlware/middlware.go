package middlware

import (
	"net/http"
	"strings"

	"github.com/andynikk/advancedmetrics/internal/constants"
	"github.com/andynikk/advancedmetrics/internal/networks"
)

func CheckIP(endpoint func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		xRealIP := r.Header.Get("X-Real-IP")

		ok := networks.AddressAllowed(strings.Split(xRealIP, constants.SepIPAddress))
		if ok {
			w.WriteHeader(http.StatusOK)
			endpoint(w, r)
			return
		}

		w.WriteHeader(http.StatusForbidden)
		_, err := w.Write([]byte("Not IP address allowed"))
		if err != nil {
			constants.Logger.ErrorLog(err)
		}
	})
}
