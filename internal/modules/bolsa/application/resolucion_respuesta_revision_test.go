package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type registroRespuestaRevisionPrueba struct{ llamadas int }

func (r *registroRespuestaRevisionPrueba) ResolverRespuestaRevision(context.Context,
	ports.OrdenResolucionRespuestaRevision,
) (ports.ReciboResolucionRespuestaRevision, error) {
	r.llamadas++
	return ports.ReciboResolucionRespuestaRevision{}, nil
}

type lectorRespuestaRevisionPrueba struct{ llamadas int }

func (l *lectorRespuestaRevisionPrueba) ConsultarRespuestasRevision(context.Context,
	ports.OrdenConsultaRespuestasRevision,
) (ports.PaginaRespuestasRevision, error) {
	l.llamadas++
	return ports.PaginaRespuestasRevision{}, nil
}

func solicitudRespuestaRevisionPrueba(t *testing.T) ports.SolicitudResolverRespuestaRevision {
	t.Helper()
	ahora := time.Date(2026, 10, 16, 8, 30, 0, 123456000, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		core.AuthMethodCertificate, core.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.SolicitudResolverRespuestaRevision{
		RespuestaRef: "respuesta-portal:" + strings.Repeat("a", 64),
		Resultado:    "confirmar_aceptacion", ClaveIdempotencia: "resolucion_0001",
		SituacionEsperadaDesde: ahora.Add(-time.Hour), BolsaRef: "bolsa:administrativo:1",
		UnidadRef: "unidad:rrhh:1", AmbitoRef: "ambito:bolsa:1",
		MotivoRef: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_revision_bolsa",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_resolver"},
		Motivo:             "Se confirma la respuesta recibida",
		MotivoAutorizacion: motivoBorradorPrueba(), Vinculo: vinculo,
		ResultadoContexto: resultado, Correlacion: correlacionBorradorPrueba(t),
	}
	if err := q.Validar(); err != nil {
		t.Fatalf("solicitud de prueba inválida: %v", err)
	}
	return q
}

func TestResolverRespuestaRevisionNoLlegaARepositorioConResultadoOAutoridadIncorrectos(t *testing.T) {
	q := solicitudRespuestaRevisionPrueba(t)
	registro, lector := &registroRespuestaRevisionPrueba{}, &lectorRespuestaRevisionPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: q.SituacionEsperadaDesde.Add(time.Hour)}
	s, err := NuevoServicioResolucionRespuestaRevision(autorizador, registro, lector)
	if err != nil {
		t.Fatal(err)
	}
	invalida := q
	invalida.Resultado = "rechazar_aceptacion"
	if _, err := s.Resolver(context.Background(), invalida); !errors.Is(err, ports.ErrResolucionRespuestaRevisionInvalida) ||
		autorizador.llamadas != 0 || registro.llamadas != 0 {
		t.Fatalf("resultado ajeno alcanzó autoridad/repositorio: %v auth=%d sql=%d", err, autorizador.llamadas, registro.llamadas)
	}
	// El emisor de Borrador firma una audiencia distinta; debe rechazarse
	// aunque la acción y la decisión V3 se hayan construido correctamente.
	if _, err := s.Resolver(context.Background(), q); !errors.Is(err, ports.ErrResolucionRespuestaRevisionNoDisponible) ||
		autorizador.llamadas != 1 || registro.llamadas != 0 {
		t.Fatalf("material de otra audiencia alcanzó SQL: %v auth=%d sql=%d", err, autorizador.llamadas, registro.llamadas)
	}
}

func TestConsultaRevisionNoLeeAntesDeV3(t *testing.T) {
	q := solicitudRespuestaRevisionPrueba(t)
	registro, lector := &registroRespuestaRevisionPrueba{}, &lectorRespuestaRevisionPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: q.SituacionEsperadaDesde.Add(time.Hour), err: core.ErrAutorizacionDenegada}
	s, err := NuevoServicioResolucionRespuestaRevision(autorizador, registro, lector)
	if err != nil {
		t.Fatal(err)
	}
	consulta := ports.SolicitudConsultarRespuestasRevision{BolsaRef: q.BolsaRef, Limite: 20,
		UnidadRef: q.UnidadRef, AmbitoRef: q.AmbitoRef, MotivoAutorizacion: q.MotivoAutorizacion,
		Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, Correlacion: q.Correlacion}
	if _, err := s.Consultar(context.Background(), consulta); !errors.Is(err, core.ErrAutorizacionDenegada) ||
		lector.llamadas != 0 || registro.llamadas != 0 {
		t.Fatalf("consulta denegada tocó repositorio: %v lecturas=%d escrituras=%d", err, lector.llamadas, registro.llamadas)
	}
}
