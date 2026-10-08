package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
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

// fuenteLoteValidaPrueba devuelve descriptores válidos para que las pruebas
// lleguen hasta el emisor cuando no hay otra causa de rechazo.
type fuenteLoteValidaPrueba struct{}

func (fuenteLoteValidaPrueba) ResolverUnidadLote(_ context.Context, org, unidad string) (AmbitosFuenteLote, error) {
	return AmbitosFuenteLote{OrganizacionRef: org, UnidadRef: unidad, Descriptores: []DimensionFuenteLote{
		{Dimension: "organizacion_ref", Valores: []string{org}, Fuente: FuenteDescriptorLote{Referencia: "prc_fuente_organizacion", Version: 1, HuellaSHA256: strings.Repeat("a", 64)}},
		{Dimension: "unidad_ref", Valores: []string{unidad}, Fuente: FuenteDescriptorLote{Referencia: "fuente:unidad:prueba", Version: 1, HuellaSHA256: strings.Repeat("b", 64)}}}}, nil
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
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLoteValidaPrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	recibo, err := a.aplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 || emisor.llamadas != 0 {
		t.Fatal("rol_sensible_entro_en_lote_ordinario")
	}
}

func TestLoteOrdinarioNoAdmiteAutoaltaAntesDeBD(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	s.Cambios[0].Objetivo.PersonaRef = s.Actor.PersonaRef
	emisor := &emisorLotePrueba{}
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLoteValidaPrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	recibo, err := a.aplicarLoteOrdinario(context.Background(), s)
	if err == nil || recibo.OperacionRef != "" || len(recibo.Cambios) != 0 || pool.comienzos != 0 || emisor.llamadas != 0 {
		t.Fatal("autoalta_entro_en_bd")
	}
}

// Control positivo: sin causa de rechazo, la orden llega al emisor. Así las
// pruebas negativas anteriores fallarían si se quitara su comprobación.
func TestLoteOrdinarioSinRechazoLlegaAlEmisor(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	categoria, unidad := "", true
	rol := rolJSON{VersionRef: s.Cambios[0].RolVersionRef, Clase: domain.ClaseControlPerfilOrdinario,
		CategoriaAdmin: &categoria, HuellaSHA256: strings.Repeat("f", 64),
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(24 * time.Hour), UnidadRequerida: &unidad}
	b, err := json.Marshal(rol)
	if err != nil {
		t.Fatal(err)
	}
	pool.roles[rol.VersionRef] = b
	emisor := &emisorLotePrueba{}
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLoteValidaPrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	if _, err := a.aplicarLoteOrdinario(context.Background(), s); err == nil || emisor.llamadas != 1 || pool.comienzos != 0 {
		t.Fatalf("control positivo no llega al emisor: llamadas=%d err=%v", emisor.llamadas, err)
	}
	emisor.llamadas = 0
	a.proveedor = fuenteLotePrueba{}
	if _, err := a.aplicarLoteOrdinario(context.Background(), s); err == nil || emisor.llamadas != 0 {
		t.Fatal("sin fuentes se emitio una decision")
	}
}

type emisorLoteDenegadoPrueba struct{}

func (emisorLoteDenegadoPrueba) EmitirLoteOrdinario(context.Context, domain.ContextoActor,
	domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
	domain.RecursoAutorizable, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, domain.ErrAutorizacionDenegada
}

// La denegación explícita del PDP llega como denegación, no como indisponible.
func TestLoteOrdinarioConservaDenegacionDelEmisor(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	categoria, unidad := "", true
	b, err := json.Marshal(rolJSON{VersionRef: s.Cambios[0].RolVersionRef, Clase: domain.ClaseControlPerfilOrdinario,
		CategoriaAdmin: &categoria, HuellaSHA256: strings.Repeat("f", 64),
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(24 * time.Hour), UnidadRequerida: &unidad})
	if err != nil {
		t.Fatal(err)
	}
	pool.roles[s.Cambios[0].RolVersionRef] = b
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisorLoteDenegadoPrueba{}, proveedor: fuenteLoteValidaPrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	if _, err := a.aplicarLoteOrdinario(context.Background(), s); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("denegacion convertida: %v", err)
	}
}

func TestTraducirErrorLoteSQL(t *testing.T) {
	casos := map[string]error{"42501": domain.ErrAutorizacionDenegada, "40001": api.ErrConflictoEstado,
		"55000": api.ErrConflictoEstado, "P0002": api.ErrConflictoEstado,
		"23505": api.ErrConflictoEstado, "55P03": api.ErrConflictoEstado, "22023": domain.ErrActoAdministracionPerfilesInvalido,
		"22P02": domain.ErrActoAdministracionPerfilesInvalido, "XX000": ports.ErrAutoridadAdministracionPerfilesNoDisponible}
	for codigo, esperado := range casos {
		if err := traducirErrorLoteSQL(context.Background(), &pgconn.PgError{Code: codigo, Message: "SECRETO"}); !errors.Is(err, esperado) {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
	if err := traducirErrorLoteSQL(context.Background(), errors.New("SECRETO")); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
		t.Fatal("error ajeno no cerrado")
	}
}

// Una baja de un perfil que ya no se ofrece llega al emisor: la clase la
// comprueba AUT44 sobre el registro, no la vigencia del perfil.
func TestLoteOrdinarioBajaDePerfilNoVigenteLlegaAlEmisor(t *testing.T) {
	s, pool, ahora := solicitudLoteOrdinarioPrueba(t)
	s.Cambios[0].Operacion, s.Cambios[0].InicioVigencia = domain.OperacionRevocarPerfil, ""
	s.Cambios[0].Objetivo.PerfilVersion, s.Cambios[0].Objetivo.VinculoVersion = 1, 1
	s.Cambios[0].Objetivo.VigenteDesde, s.Cambios[0].Objetivo.VigenteHasta = time.Time{}, time.Time{}
	_, h, err := s.CanonicoYHuella()
	if err != nil {
		t.Fatal(err)
	}
	s.HuellaSolicitudSHA256 = h
	delete(pool.roles, s.Cambios[0].RolVersionRef)
	emisor := &emisorLotePrueba{}
	a := &AutoridadLoteOrdinario{pool: pool, emisor: emisor, proveedor: fuenteLoteValidaPrueba{},
		reloj: relojFijo(ahora), organizacion: "org_prueba"}
	if _, err := a.aplicarLoteOrdinario(context.Background(), s); err == nil || emisor.llamadas != 1 {
		t.Fatalf("baja no llega al emisor: llamadas=%d err=%v", emisor.llamadas, err)
	}
}
