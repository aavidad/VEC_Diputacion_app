package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type casoOriginalC8Prueba struct {
	recibo   ports.ReciboOriginalC8
	llamadas int
}

func (c *casoOriginalC8Prueba) Aportar(_ context.Context, _ ports.OrdenOriginalC8, _ ports.PeticionAnexoOriginalC8) (ports.ReciboOriginalC8, error) {
	c.llamadas++
	return c.recibo, nil
}
func (c *casoOriginalC8Prueba) Revisar(_ context.Context, _ ports.OrdenOriginalC8, _ ports.PeticionRevisionOriginalC8V2) (ports.ReciboOriginalC8, error) {
	c.llamadas++
	return c.recibo, nil
}

type resolverOriginalC8Prueba struct{ llamadas int }

func (r *resolverOriginalC8Prueba) ResolverOriginal(*http.Request) (ports.OrdenOriginalC8, error) {
	r.llamadas++
	return ports.OrdenOriginalC8{}, nil
}

func TestOriginalC8HTTPReciboConfirmadoYReintento(t *testing.T) {
	ref := "permiso:cronos:solicitud:abcdefgh"
	doc := "ref:" + strings.Repeat("a", 64)
	sha := strings.Repeat("b", 64)
	c := &casoOriginalC8Prueba{recibo: ports.ReciboOriginalC8{
		SolicitudRef: ref, DocumentoID: doc, DocumentoVersion: 1, SHA256: sha,
		Estado: domain.JustificacionPendiente, Version: 1,
		ReciboRef:   "recibo:cronos:00000000-0000-4000-8000-000000000001",
		InstanteUTC: time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC),
	}}
	r := &resolverOriginalC8Prueba{}
	m, err := NuevoManejadorOriginalC8(c, r)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"solicitud_ref":"` + ref + `","clave_operacion":"ref:` + strings.Repeat("c", 64) +
		`","version_esperada":"0","documento_id":"` + doc + `","documento_version":"1"}`
	for _, replay := range []bool{false, true} {
		c.recibo.Replay = replay
		req := httptest.NewRequest(http.MethodPost, RutaAportarOriginalC8, strings.NewReader(cuerpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)
		esperado := http.StatusCreated
		if replay {
			esperado = http.StatusOK
		}
		if w.Code != esperado {
			t.Fatalf("replay=%v: HTTP %d: %s", replay, w.Code, w.Body.String())
		}
		var body struct {
			Recibo struct {
				ReciboRef string `json:"recibo_ref"`
				Replay    bool   `json:"replay"`
			} `json:"recibo"`
		}
		if json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Recibo.ReciboRef != c.recibo.ReciboRef || body.Recibo.Replay != replay {
			t.Fatalf("recibo HTTP divergente: %s", w.Body.String())
		}
	}
	if c.llamadas != 2 || r.llamadas != 2 {
		t.Fatalf("llamadas caso=%d resolver=%d", c.llamadas, r.llamadas)
	}
}

func TestOriginalC8HTTPNoAceptaIdentidadNiVersionAmbigua(t *testing.T) {
	c := &casoOriginalC8Prueba{}
	r := &resolverOriginalC8Prueba{}
	m, err := NuevoManejadorOriginalC8(c, r)
	if err != nil {
		t.Fatal(err)
	}
	for _, cuerpo := range []string{
		`{"solicitud_ref":"x","empleado_ref":"emp_ajeno"}`,
		`{"solicitud_ref":"x","solicitud_ref":"y"}`,
		`{"solicitud_ref":"x","clave_operacion":"y","version_esperada":"01","decision":"aceptada","motivo_ref":"m"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, RutaRevisarOriginalC8, strings.NewReader(cuerpo))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest || c.llamadas != 0 || r.llamadas != 0 {
			t.Fatalf("aceptó cuerpo ajeno/ambiguo: %d", w.Code)
		}
	}
}
