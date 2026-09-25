package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func materialEventoPlazoPrueba() MaterialEventoPlazoLlamamiento {
	ref := func(e string) ports.ReferenciaGobernadaComunicacionLlamamiento {
		return ports.ReferenciaGobernadaComunicacionLlamamiento{Referencia: "vec.bolsa.reglas:1:" + e, Version: 1, HuellaSHA256: strings.Repeat("e", 64)}
	}
	return MaterialEventoPlazoLlamamiento{
		Solicitud: ports.SolicitudRegistrarEventoPlazoLlamamiento{
			ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "organizacion:plazo",
			ExpedienteRef: "expediente:plazo", LlamamientoRef: "llamamiento:plazo", ComunicacionRef: "comunicacion:plazo",
			VersionComunicacionEsperada: 2, Tipo: ports.EventoPlazoContactoEfectivo,
			InstanteEn: time.Date(2026, 9, 28, 9, 30, 0, 123000, time.UTC), PruebaRef: "prueba:llamada",
		},
		Plazo: &ports.PlazoRespuestaGobernado{RespuestaHasta: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC), UltimoDia: "2026-09-29",
			Politica: ref("b05.plazo_respuesta"), TratamientoFueraDePlazo: domain.TratamientoFueraDePlazoNoAdmitir,
			ConfirmacionExpiracion: domain.ConfirmacionExpiracionRRHH, CriterioRespuesta: ref("b07.fuera_de_plazo"),
			CriterioExpiracion: ref("b08.sin_respuesta_baja"), ReglaEjemplo: true},
	}
}

// El JSON es el contrato exacto de CT111: nueve campos de solicitud y ocho de
// plazo con sus nombres; la huella del recurso liga todos ellos.
func TestMaterialEventoPlazoConservaElContratoSQL(t *testing.T) {
	m := materialEventoPlazoPrueba()
	b, err := codificarMaterialEventoPlazo(m)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 2 || len(campos["Solicitud"]) != 9 || len(campos["Plazo"]) != 8 ||
		string(campos["Solicitud"]["InstanteEn"]) != `"2026-09-28T09:30:00.000123Z"` ||
		string(campos["Plazo"]["TratamientoFueraDePlazo"]) != `"no_admitir"` {
		t.Fatalf("material inesperado: %s", b)
	}
	r, err := RecursoEventoPlazoLlamamiento(m)
	if err != nil || r.Tipo != TipoRecursoResolucionManualLlamamiento || r.Referencia != "expediente:plazo" {
		t.Fatalf("recurso inesperado: %+v %v", r, err)
	}
	h, _ := r.HuellaContextoAutorizacionSHA256()
	otro := materialEventoPlazoPrueba()
	otro.Plazo.RespuestaHasta = otro.Plazo.RespuestaHasta.Add(time.Hour)
	r2, _ := RecursoEventoPlazoLlamamiento(otro)
	if h2, _ := r2.HuellaContextoAutorizacionSHA256(); h2 == h {
		t.Fatal("el vencimiento no queda ligado a la autorización")
	}
	causa := materialEventoPlazoPrueba()
	causa.Solicitud.Tipo = ports.EventoPlazoCausaJustificada
	if causa.Validar() == nil {
		t.Fatal("causa con plazo aceptada")
	}
	causa.Plazo = nil
	if b, err := codificarMaterialEventoPlazo(causa); err != nil || strings.Contains(string(b), "Plazo") {
		t.Fatalf("causa sin plazo: %s %v", b, err)
	}
}

func TestEventoPlazoErroresNoExponenDetallesSQL(t *testing.T) {
	for codigo, esperado := range map[string]error{"P0590": ports.ErrSolicitudEventoPlazoInvalida, "P0591": ports.ErrClaveEventoPlazoUsada,
		"P0592": ports.ErrEventoPlazoEnConflicto, "P0593": ports.ErrOperacionEventoPlazoDenegada, "42501": ports.ErrOperacionEventoPlazoDenegada,
		"P0594": ports.ErrEventoPlazoNoDisponible, "40001": ports.ErrEventoPlazoNoDisponible} {
		if err := normalizarErrorEventoPlazo(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle privado"}); !errors.Is(err, esperado) || strings.Contains(err.Error(), "privado") {
			t.Fatal(codigo, err)
		}
	}
	for codigo, esperado := range map[string]error{"P0585": ports.ErrRespuestaFueraDePlazoLlamamiento, "P0586": ports.ErrPlazoRespuestaNoVencido} {
		if err := normalizarErrorResolucionManualLlamamiento(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle privado"}); !errors.Is(err, esperado) {
			t.Fatal(codigo, err)
		}
	}
}
