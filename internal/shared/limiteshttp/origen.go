package limiteshttp

import (
	"net/http"
	"net/url"
	"strings"
)

// MismoOrigenEstricto protege los POST multipart, cuyo tipo de contenido
// permite una petición simple desde otro sitio. El certificado identifica
// a la persona; estas señales impiden el envío cruzado desde un navegador.
func MismoOrigenEstricto(r *http.Request) bool {
	if r == nil || r.TLS == nil || len(r.Header.Values("Sec-Fetch-Site")) != 1 ||
		r.Header.Get("Sec-Fetch-Site") != "same-origin" || len(r.Header.Values("Origin")) != 1 {
		return false
	}
	origen := r.Header.Get("Origin")
	if len(origen) > 2048 {
		return false
	}
	u, err := url.Parse(origen)
	return err == nil && u.Scheme == "https" && u.Opaque == "" && u.User == nil &&
		u.Host != "" && strings.EqualFold(u.Host, r.Host) && u.Path == "" && u.RawQuery == "" &&
		!u.ForceQuery && u.Fragment == ""
}
