package administracionperfiles

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadatosRechazaRutasAuxiliaresYBusquedaNombreAntesDeFuente(t *testing.T) {
	for _, ruta := range []string{PrefijoV1 + "/capacidades", PrefijoV1 + "/roles", PrefijoV1 + "/propuestas", PrefijoV1 + "/personas?q=Elena"} {
		t.Run(ruta, func(t *testing.T) {
			s := sesionADMINPrueba(t)
			f := &fuenteUsuariosPrueba{}
			a := &auditorPrueba{}
			h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", &sesionPrueba{resultado: s}, f, a)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodGet, ruta, ""))
			if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest || f.llamadas != 0 {
				t.Fatal("ruta_fuera_alcance")
			}
		})
	}
}
