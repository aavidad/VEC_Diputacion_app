package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRespuestaRecibidaMaterialLigaDeclaracionCompleta(t *testing.T) {
	s := ports.SolicitudRegistrarRespuestaRecibida{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "org:sintetica", ExpedienteRef: "exp:sintetico", LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica", VersionComunicacionEsperada: 2, Respuesta: ports.RespuestaLlamamientoAceptada, CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64), RecibidaEn: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)}
	r, err := RecursoRegistroRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := r.HuellaContextoAutorizacionSHA256()
	for nombre, mutar := range map[string]func(*ports.SolicitudRegistrarRespuestaRecibida){
		"clave": func(s *ports.SolicitudRegistrarRespuestaRecibida) {
			s.ClaveIdempotencia = "21111111-1111-4111-8111-111111111111"
		},
		"organizacion": func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.OrganizacionRef += "b" },
		"expediente":   func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.ExpedienteRef += "b" },
		"llamamiento":  func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.LlamamientoRef += "b" },
		"comunicacion": func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.ComunicacionRef += "b" },
		"respuesta":    func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.Respuesta = ports.RespuestaLlamamientoRenunciada },
		"correo":       func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.CorreoRef += "b" },
		"huella":       func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = strings.Repeat("b", 64) },
		"fecha":        func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.RecibidaEn = s.RecibidaEn.Add(time.Microsecond) },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := s
			mutar(&otra)
			r, err := RecursoRegistroRespuestaRecibida(otra)
			if err != nil {
				t.Fatal(err)
			}
			nueva, _ := r.HuellaContextoAutorizacionSHA256()
			if nueva == h {
				t.Fatal("campo no ligado")
			}
		})
	}
	s.VersionComunicacionEsperada = 3
	if _, err := RecursoRegistroRespuestaRecibida(s); err == nil {
		t.Fatal("otra versión admitida")
	}
}

func TestRespuestaRecibidaErroresNoExponenDetallesSQL(t *testing.T) {
	for codigo, esperado := range map[string]error{"P0560": ports.ErrSolicitudRespuestaRecibidaInvalida, "P0561": ports.ErrClaveRespuestaRecibidaUsada, "P0562": ports.ErrVersionRespuestaRecibidaEnConflicto, "P0563": ports.ErrOperacionRespuestaRecibidaDenegada, "42501": ports.ErrOperacionRespuestaRecibidaDenegada, "40001": ports.ErrRespuestaRecibidaNoDisponible, "08006": ports.ErrRespuestaRecibidaNoDisponible} {
		if err := normalizarErrorRespuestaRecibida(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle privado"}); !errors.Is(err, esperado) || strings.Contains(err.Error(), "privado") {
			t.Fatal(codigo, err)
		}
	}
}
