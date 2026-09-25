package httpinterno

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Los rechazos de entrada conservan su causa técnica (M2a) sin cambiar la
// respuesta pública: 400 con «peticion_invalida» y sin detalle interno.
func TestRechazosDeEntradaConservanCausaYRespuesta(t *testing.T) {
	var sintaxis *json.SyntaxError
	var excedido *http.MaxBytesError
	var escape url.EscapeError
	for _, c := range []struct {
		nombre string
		cuerpo string
		causa  func(error) bool
	}{
		{"json roto", `{"clave_operacion":`, func(err error) bool {
			return errors.Is(err, errPeticionResolucionInvalida) && err != errPeticionResolucionInvalida
		}},
		{"sintaxis", `{"clave_operacion" 1}`, func(err error) bool { return errors.As(err, &sintaxis) }},
		{"excede el límite", `{"motivo":"` + strings.Repeat("a", maximoCuerpoResolver) + `"}`, func(err error) bool { return errors.As(err, &excedido) }},
		{"sin causa técnica", `{"otro":"x"}`, func(err error) bool { return err == errPeticionResolucionInvalida }},
	} {
		w := httptest.NewRecorder()
		_, err := decodificarCadenasAcotadas(w, peticionJSON(http.MethodPost, RutaResolverPermiso, c.cuerpo), nil, map[string]int{"clave_operacion": 128, "motivo": 16})
		if !errors.Is(err, errPeticionResolucionInvalida) || !c.causa(err) {
			t.Fatalf("%s: causa no conservada: %v", c.nombre, err)
		}
		h, _ := NuevoManejadorResolucionPermisos(&casoResolucionPrueba{}, &resolverResolucionPrueba{})
		w = httptest.NewRecorder()
		h.ServeHTTP(w, peticionJSON(http.MethodPost, RutaResolverPermiso, c.cuerpo))
		if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != `{"error":"peticion_invalida"}` {
			t.Fatalf("%s: respuesta pública cambiada: %d %s", c.nombre, w.Code, w.Body.String())
		}
	}
	if _, err := parametroPaso("paso=%zz"); !errors.Is(err, errPeticionResolucionInvalida) || !errors.As(err, &escape) {
		t.Fatalf("consulta mal codificada sin causa: %v", err)
	}
	if _, err := parametroPaso("paso=jefatura"); err != errPeticionResolucionInvalida {
		t.Fatalf("paso no admitido: %v", err)
	}
	h, _ := NuevoManejadorResolucionPermisos(&casoResolucionPrueba{}, &resolverResolucionPrueba{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaBandejaPermisos+"?paso=%zz", nil))
	if w.Code != http.StatusBadRequest || strings.TrimSpace(w.Body.String()) != `{"error":"peticion_invalida"}` {
		t.Fatalf("bandeja con consulta mal codificada: %d %s", w.Code, w.Body.String())
	}
}
