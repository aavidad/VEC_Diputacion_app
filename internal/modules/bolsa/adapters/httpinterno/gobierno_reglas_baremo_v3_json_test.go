package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestGobiernoReglasHTTPV3RechazaJSONAmbiguoAntesDelNegocio(t *testing.T) {
	for _, ruta := range []string{RutaAltaGobiernoReglasBaremoV3, RutaConsultaGobiernoReglasBaremoV3, RutaRecuperarGobiernoReglasBaremoV3} {
		_, _, o, base := escenarioGobiernoHTTPPrueba(t)
		if ruta != RutaAltaGobiernoReglasBaremoV3 {
			v, err := reglas.RestaurarVersionGobernadaReglasBaremo(o.recibo.VersionCanonica)
			if err != nil {
				t.Fatal(err)
			}
			selector, err := selectorSalidaGobiernoHTTPV3(v)
			if err != nil {
				t.Fatal(err)
			}
			base = map[string]any{"selector": selector, "motivo": base["motivo"]}
			if ruta == RutaRecuperarGobiernoReglasBaremoV3 {
				base["clave_operacion"], base["huella_solicitud_sha256"] = o.recibo.ClaveOperacion, o.recibo.HuellaSolicitudSHA256
			}
		}
		canon, err := json.Marshal(base)
		if err != nil {
			t.Fatal(err)
		}
		original := string(canon)
		variantes := []string{
			strings.Replace(original, `"motivo":`, `"motivo":{},"motivo":`, 1),
			strings.Replace(original, `"motivo":`, `"Motivo":`, 1),
			strings.Replace(original, `"entrada_clave":`, `"entrada_clave":"otro","entrada_clave":`, 1),
			strings.Replace(original, `"entrada_clave":`, `"Entrada_Clave":`, 1),
			strings.Replace(original, `"entrada_clave":`, `"entrada_\u0063lave":"otro","entrada_clave":`, 1),
		}
		if ruta != RutaAltaGobiernoReglasBaremoV3 {
			variantes = append(variantes,
				strings.Replace(original, `"version":`, `"version":2,"version":`, 1),
				strings.Replace(original, `"version":`, `"Version":`, 1))
		}
		for _, cuerpo := range variantes {
			h, _, operador, _ := escenarioGobiernoHTTPPrueba(t)
			r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != 400 || operador.llamadas != 0 {
				t.Fatalf("JSON ambiguo llegó al negocio: %d", w.Code)
			}
		}
	}
}

func TestGobiernoReglasHTTPV3Conserva413AlLeerTodoElCuerpo(t *testing.T) {
	h, _, operador, body := escenarioGobiernoHTTPPrueba(t)
	canon, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	canon = append(canon, bytes.Repeat([]byte(" "), maximoEntradaGobiernoReglasV3)...)
	r := httptest.NewRequest(http.MethodPost, RutaAltaGobiernoReglasBaremoV3, bytes.NewReader(canon))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 413 || operador.llamadas != 0 {
		t.Fatalf("límite de cuerpo perdido: %d", w.Code)
	}
}

func TestGobiernoReglasHTTPV3AuditaErrorEntradaYExigeAcuse(t *testing.T) {
	for _, fallo := range []error{nil, errors.New("auditor no disponible")} {
		h, _, o, _ := escenarioGobiernoHTTPPrueba(t)
		llamadas := 0
		h.auditarError = func(_ context.Context, err error) error {
			llamadas++
			if !errors.Is(err, app.ErrGobiernoV3PeticionInvalida) {
				t.Fatal("categoría de entrada perdida")
			}
			return fallo
		}
		r := httptest.NewRequest(http.MethodPost, RutaAltaGobiernoReglasBaremoV3, strings.NewReader(`{"motivo":{},"motivo":{}}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		esperado := 400
		if fallo != nil {
			esperado = 503
		}
		if llamadas != 1 || w.Code != esperado || o.llamadas != 0 {
			t.Fatalf("error de entrada sin auditoría: %d llamadas=%d", w.Code, llamadas)
		}
	}
}
