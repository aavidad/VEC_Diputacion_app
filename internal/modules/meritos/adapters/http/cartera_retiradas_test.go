package meritoshttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type proveedorRetiradaCarteraPrueba struct{}

func (proveedorRetiradaCarteraPrueba) PrepararSolicitudRetiradaCartera(*http.Request) (application.SolicitudRetiradaCartera, error) {
	return application.SolicitudRetiradaCartera{}, nil
}

type retiradorCarteraPrueba struct {
	consultas []application.SolicitudRetiradaCartera
	repetida  bool
}

func (s *retiradorCarteraPrueba) Retirar(_ context.Context,
	q application.SolicitudRetiradaCartera) (ports.ConfirmacionRetiradaCartera, error) {
	s.consultas = append(s.consultas, q)
	if q.HechoRef == "" {
		return ports.ConfirmacionRetiradaCartera{}, ports.ErrCarteraEntradaInvalida
	}
	return ports.ConfirmacionRetiradaCartera{Recibo: ports.ReciboRetiradaCartera{
		Referencia: "recibo:1", HechoRef: q.HechoRef, VersionHecho: q.HechoVersionEsperada,
		Disponibilidad: ports.DisponibilidadCartera{Estado: "retirado", Version: 1},
		RetiradoEn:     time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)},
		Repetida: s.repetida, AuditoriaAccesoRef: "auditoria:1"}, nil
}

func peticionRetiradaCarteraPrueba(cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaRetiradasCartera, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestRetiradaCarteraHTTPConservaReciboYReintento(t *testing.T) {
	s := &retiradorCarteraPrueba{}
	h, err := NuevoHandlerRetiradasCartera(proveedorRetiradaCarteraPrueba{}, s)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"hecho_ref":"hecho:1","hecho_version_esperada":2,"disponibilidad_version_esperada":0,"motivo":{"catalogo_id":"motivos_meritos_cartera","catalogo_version":1,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","entrada_clave":"motivo_0123456789abcdef0123456789abcdef"},"clave_idempotencia":"clave:retirada:1"}`
	for _, tc := range []struct {
		repetida bool
		estado   int
	}{{false, http.StatusCreated}, {true, http.StatusOK}} {
		s.repetida = tc.repetida
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRetiradaCarteraPrueba(cuerpo))
		if w.Code != tc.estado || w.Header().Get("X-Audit-Ref") != "auditoria:1" ||
			!strings.Contains(w.Body.String(), `"referencia":"recibo:1"`) ||
			len(s.consultas) == 0 || s.consultas[len(s.consultas)-1].HechoRef != "hecho:1" ||
			s.consultas[len(s.consultas)-1].HechoVersionEsperada != 2 ||
			s.consultas[len(s.consultas)-1].MotivoRetirada != (vec.ReferenciaEntradaCatalogo{
				CatalogoID: "motivos_meritos_cartera", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("a", 64),
				EntradaClave:         "motivo_0123456789abcdef0123456789abcdef"}) {
			t.Fatalf("recibo/replay: repetida=%v estado=%d cuerpo=%s", tc.repetida, w.Code, w.Body.String())
		}
	}
}

func TestRetiradaCarteraHTTPRechazaDuplicadosYNoActuaConEntradaInvalida(t *testing.T) {
	s := &retiradorCarteraPrueba{}
	h, err := NuevoHandlerRetiradasCartera(proveedorRetiradaCarteraPrueba{}, s)
	if err != nil {
		t.Fatal(err)
	}
	for _, cuerpo := range []string{
		`{"hecho_ref":"hecho:1","hecho_ref":"hecho:2"}`,
		`{"hecho_ref":"hecho:1","hecho_version_esperada":1,"disponibilidad_version_esperada":0,"motivo":{"catalogo_id":"a","catalogo_id":"b"},"clave_idempotencia":"clave:1"}`,
		`{"hecho_ref":"hecho:1","persona_ref":"persona:ajena"}`,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionRetiradaCarteraPrueba(cuerpo))
		if w.Code != http.StatusBadRequest || len(s.consultas) == 0 ||
			s.consultas[len(s.consultas)-1].HechoRef != "" ||
			strings.Contains(w.Body.String(), "hecho:1") {
			t.Fatalf("entrada ambigua alcanzó efecto: estado=%d consulta=%+v", w.Code, s.consultas)
		}
	}
}
