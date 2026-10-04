package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type emisorLotePrueba struct{ llamadas int }

func (e *emisorLotePrueba) EmitirLoteOrdinario(context.Context, domain.ContextoActor,
	domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
	domain.RecursoAutorizable, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("emisor_no_conectado")
}

type fuenteLotePrueba struct{}

func (fuenteLotePrueba) ResolverUnidadLote(context.Context, string, string) (AmbitosFuenteLote, error) {
	return AmbitosFuenteLote{}, errors.New("fuente_no_conectada")
}

func solicitudLoteOrdinarioPrueba(t *testing.T) (domain.SolicitudLoteAdministracionPerfiles, *poolCatalogoPrueba, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	acto, _, pool, _ := contratoV2Prueba(t)
	version := acto.InstantaneaAutorizacion.VersionRol
	version.Version = 6
	version.Concesiones = append(version.Concesiones, domain.ConcesionRol{
		Accion: accionLoteOrdinario, ModuloID: "administracion", TipoRecurso: "persona",
		Finalidades: []string{"gestion_perfiles"}, GarantiaMinima: domain.AuthAssuranceHigh,
		Obligaciones: []string{"auditar"},
	})
	acto.InstantaneaAutorizacion.VersionRol = version
	acto.InstantaneaAutorizacion.AsignacionPerfil.VersionRolRef = version.Referencia()
	acto.InstantaneaAutorizacion.ControlVigenciaVersionRol.VersionRolRef = version.Referencia()
	s := domain.SolicitudLoteAdministracionPerfiles{
		OperacionRef: acto.OperacionRef, OrganizacionRef: "org_prueba", Actor: acto.Actor, Evidencia: acto.Evidencia,
		InstantaneaAutorizacion: acto.InstantaneaAutorizacion, Motivo: acto.Motivo,
		ReferenciaActo: acto.ReferenciaActo, CorrelacionRef: acto.CorrelacionRef,
		Cambios: []domain.CambioPerfilAdministracion{{Operacion: domain.OperacionOtorgarPerfil,
			InicioVigencia: domain.InicioVigenciaLoteProgramado,
			RolVersionRef:  acto.RolVersionRef, Objetivo: acto.Objetivo}},
	}
	s.Cambios[0].Objetivo.VigenteDesde = ahora.Add(time.Minute)
	s.Cambios[0].Objetivo.CentroRef = ""
	_, h, err := s.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	s.HuellaSolicitudSHA256 = h
	return s, pool, ahora
}

func TestLoteOrdinarioRechazaRolSensibleAntesDeEmitirYEscribir(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	categoria, unidad := "aplicacion", true
	rol := rolJSON{VersionRef: s.Cambios[0].RolVersionRef, Clase: domain.ClaseControlPerfilAdministrador,
		CategoriaAdmin: &categoria, HuellaSHA256: strings.Repeat("f", 64),
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), UnidadRequerida: &unidad}
	b, err := json.Marshal(rol)
	if err != nil {
		t.Fatal(err)
	}
	pool.roles[rol.VersionRef] = b
	emisor := &emisorLotePrueba{}
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLotePrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	recibo, err := a.AplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 || emisor.llamadas != 0 {
		t.Fatal("rol_sensible_entro_en_lote_ordinario")
	}
}

func TestLoteOrdinarioNoAdmiteAutoaltaAntesDeBD(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	s.Cambios[0].Objetivo.PersonaRef = s.Actor.PersonaRef
	emisor := &emisorLotePrueba{}
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLotePrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	recibo, err := a.AplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 || emisor.llamadas != 0 {
		t.Fatal("autoalta_entro_en_bd")
	}
}
