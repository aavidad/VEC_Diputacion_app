package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func solicitudLoteOrdinarioPrueba(t *testing.T) (domain.SolicitudLoteAdministracionPerfiles, *poolCatalogoPrueba, time.Time) {
	t.Helper()
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
		OperacionRef: acto.OperacionRef, Actor: acto.Actor, Evidencia: acto.Evidencia,
		InstantaneaAutorizacion: acto.InstantaneaAutorizacion, Motivo: acto.Motivo,
		ReferenciaActo: acto.ReferenciaActo, CorrelacionRef: acto.CorrelacionRef,
		Cambios: []domain.CambioPerfilAdministracion{{Operacion: domain.OperacionOtorgarPerfil,
			RolVersionRef: acto.RolVersionRef, Objetivo: acto.Objetivo}},
	}
	_, h, err := s.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	s.HuellaSolicitudSHA256 = h
	return s, pool, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
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
	a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(ahora)}
	recibo, err := a.AplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 {
		t.Fatal("rol_sensible_entro_en_lote_ordinario")
	}
}

func TestLoteOrdinarioNoAdmiteAutoaltaAntesDeBD(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	s.Cambios[0].Objetivo.PersonaRef = s.Actor.PersonaRef
	a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(ahora)}
	recibo, err := a.AplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 {
		t.Fatal("autoalta_entro_en_bd")
	}
}
