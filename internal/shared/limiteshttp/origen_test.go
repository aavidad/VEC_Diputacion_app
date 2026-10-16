package limiteshttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMismoOrigenEstrictoNoAceptaOrigenAjenoNiCredenciales(t *testing.T) {
	for _, q := range []struct {
		origen, sitio string
		valida        bool
	}{
		{"https://localhost:18445", "same-origin", true},
		{"https://otro.invalid", "cross-site", false},
		{"https://localhost:18445", "same-site", false},
		{"https://localhost:18445", "", false},
		{"https://usuario@localhost:18445", "same-origin", false},
		{"https://localhost:18445/", "same-origin", false},
		{"http://localhost:18445", "same-origin", false},
	} {
		r := httptest.NewRequest(http.MethodPost, "https://localhost:18445/api/vec/bolsa/mi-bolsa/justificantes", nil)
		r.Header.Set("Origin", q.origen)
		r.Header.Set("Sec-Fetch-Site", q.sitio)
		if MismoOrigenEstricto(r) != q.valida {
			t.Errorf("origen=%q sitio=%q", q.origen, q.sitio)
		}
	}
}
