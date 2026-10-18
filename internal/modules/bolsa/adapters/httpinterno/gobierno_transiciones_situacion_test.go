package httpinterno

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type preparadorGobiernoTransicionesPrueba struct{ llamadas, auditorias int }

func (p *preparadorGobiernoTransicionesPrueba) PrepararConsultaGobiernoTransiciones(context.Context) (bool, error) {
	p.llamadas++
	return true, nil
}
func (p *preparadorGobiernoTransicionesPrueba) AuditarConsultaGobiernoTransiciones(context.Context, bool) error {
	p.auditorias++
	return nil
}
func (p *preparadorGobiernoTransicionesPrueba) PrepararPublicacionGobiernoTransiciones(_ context.Context,
	e EntradaGobiernoTransicionesSituacion) (ports.ComandoGobiernoTransicionesSituacion, error) {
	p.llamadas++
	return ports.ComandoGobiernoTransicionesSituacion{VersionEsperada: e.VersionEsperada,
		Transiciones: e.Transiciones, Motivo: e.Motivo, ClaveIdempotencia: e.ClaveIdempotencia}, nil
}

type operadorGobiernoTransicionesPrueba struct {
	llamadas    int
	reutilizada bool
}

func (o *operadorGobiernoTransicionesPrueba) Consultar(context.Context) (ports.GobiernoTransicionesSituacion, error) {
	o.llamadas++
	return ports.GobiernoTransicionesSituacion{Version: 1}, nil
}
func (o *operadorGobiernoTransicionesPrueba) Publicar(_ context.Context,
	c ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	o.llamadas++
	return ports.GobiernoTransicionesSituacion{Version: c.VersionEsperada + 1,
		TransicionesGuardadas: c.Transiciones, ReciboRef: "recibo:politica-transiciones:prueba",
		Reutilizada: o.reutilizada}, nil
}

func TestGobiernoTransicionesRechazaIdentidadYCamposDelCliente(t *testing.T) {
	p := &preparadorGobiernoTransicionesPrueba{}
	o := &operadorGobiernoTransicionesPrueba{}
	h, err := NuevoHandlerGobiernoTransicionesSituacion(p, o)
	if err != nil {
		t.Fatal(err)
	}
	for _, cuerpo := range []string{
		`{"version_esperada":1,"transiciones":["disponible>excluido"],"motivo":"ajuste","clave_idempotencia":"clave-12345","actor_ref":"otro"}`,
		`{"version_esperada":1,"transiciones":["disponible>excluido"],"motivo":"ajuste","clave_idempotencia":"clave-12345","catalogo_sha256":"otro"}`,
	} {
		r := httptest.NewRequest(http.MethodPost, RutaGobiernoTransicionesSituacion, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnprocessableEntity || p.llamadas != 0 || o.llamadas != 0 {
			t.Fatalf("la entrada ajena llegó a la autoridad: %d %d %d", w.Code, p.llamadas, o.llamadas)
		}
	}
}

func TestGobiernoTransicionesPublicaYRecuperaRecibo(t *testing.T) {
	p := &preparadorGobiernoTransicionesPrueba{}
	o := &operadorGobiernoTransicionesPrueba{}
	h, err := NuevoHandlerGobiernoTransicionesSituacion(p, o)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"version_esperada":1,"transiciones":["disponible>excluido"],"motivo":"ajuste","clave_idempotencia":"clave-12345"}`
	for _, caso := range []struct {
		reutilizada bool
		estado      int
	}{{false, http.StatusCreated}, {true, http.StatusOK}} {
		o.reutilizada = caso.reutilizada
		r := httptest.NewRequest(http.MethodPost, RutaGobiernoTransicionesSituacion, strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var salida struct {
			Data struct {
				ReciboRef   string `json:"recibo_ref"`
				Reutilizada bool   `json:"reutilizada"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil || w.Code != caso.estado ||
			salida.Data.ReciboRef == "" || salida.Data.Reutilizada != caso.reutilizada {
			t.Fatalf("recibo: %d %s %v", w.Code, w.Body.String(), err)
		}
	}
}

func TestGobiernoTransicionesConsultaAuditaTrasLeer(t *testing.T) {
	p := &preparadorGobiernoTransicionesPrueba{}
	o := &operadorGobiernoTransicionesPrueba{}
	h, err := NuevoHandlerGobiernoTransicionesSituacion(p, o)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, RutaGobiernoTransicionesSituacion, nil))
	if w.Code != http.StatusOK || p.llamadas != 1 || p.auditorias != 1 || o.llamadas != 1 {
		t.Fatalf("lectura sin auditoría confirmada: %d %+v %+v", w.Code, p, o)
	}
}
