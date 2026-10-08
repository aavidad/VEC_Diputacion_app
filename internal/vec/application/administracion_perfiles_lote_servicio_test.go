package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type preparadorLotePrueba struct {
	llamadas int
	mutar    func(*domain.PreparacionLoteAdministracionPerfiles)
}

func (p *preparadorLotePrueba) PrepararLoteOrdinario(_ context.Context, s domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error) {
	p.llamadas++
	r := domain.PreparacionLoteAdministracionPerfiles{OperacionRef: s.OperacionRef, AuditoriaRef: "aud_v3_" + strings.Repeat("a", 32),
		PreparadaEn: s.Actor.ResueltoEn.UTC().Truncate(time.Microsecond), PersonaRef: s.PersonaRef, PersonaVersion: 1,
		CuentaRef: "cta_" + strings.Repeat("c", 32), CuentaVersion: 1, ProcedenciaRef: "procedencia:prueba:1", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("e", 64), OrganizacionRef: s.OrganizacionRef, UnidadRef: s.UnidadRef,
		Altas: []domain.AltaPosibleLoteAdministracion{}, Bajas: []domain.BajaPosibleLoteAdministracion{}}
	if p.mutar != nil {
		p.mutar(&r)
	}
	return r, nil
}

// servicioLotesPrueba usa el entorno de las pruebas del lote y añade a la
// versión del rol la concesión exacta del lote ordinario.
func servicioLotesPrueba(t *testing.T) (*ServicioLotesAdministracionPerfiles, domain.SolicitudLoteAdministracionPerfiles,
	*autoridadPerfilesLotePrueba, *preparadorLotePrueba, catalogoPerfilesLotePrueba) {
	t.Helper()
	_, s, a, c := lotePerfilesAplicacionPrueba(t)
	s.InstantaneaAutorizacion.VersionRol.Concesiones = append(s.InstantaneaAutorizacion.VersionRol.Concesiones,
		domain.ConcesionRol{Accion: accionLoteOrdinario, ModuloID: "administracion", TipoRecurso: "persona",
			Finalidades: []string{"gestion_perfiles"}, GarantiaMinima: domain.AuthAssuranceHigh, Obligaciones: []string{"auditar"}})
	sellarLotePerfilesPrueba(t, &s)
	p := &preparadorLotePrueba{}
	servicio, err := NuevoServicioLotesAdministracionPerfiles(c, a, p, &relojAutorizacionServicioPrueba{ahora: s.Actor.ResueltoEn})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, s, a, p, c
}

func preparacionDesdeLotePrueba(s domain.SolicitudLoteAdministracionPerfiles) domain.SolicitudPreparacionLoteAdministracionPerfiles {
	return domain.SolicitudPreparacionLoteAdministracionPerfiles{OperacionRef: "prep_admin:" + strings.Repeat("b", 32),
		OrganizacionRef: s.OrganizacionRef, UnidadRef: "unidad:prueba", PersonaRef: s.Cambios[0].Objetivo.PersonaRef,
		Actor: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion, CorrelacionRef: s.CorrelacionRef}
}

func TestServicioLotesAplicaConConcesionDelLoteSinCategoriaDelCatalogo(t *testing.T) {
	servicio, s, a, _, c := servicioLotesPrueba(t)
	// El catálogo del lote no expone la categoría del administrador: no se exige.
	delete(c, s.InstantaneaAutorizacion.VersionRol.Referencia())
	r, err := servicio.AplicarLoteOrdinario(context.Background(), s)
	if err != nil || a.lotes != 1 || r.ValidarPara(s) != nil {
		t.Fatalf("lote no aplicado: %v", err)
	}
}

func TestServicioLotesSinConcesionDelLoteNoLlegaALaAutoridad(t *testing.T) {
	servicio, s, a, p, _ := servicioLotesPrueba(t)
	s.InstantaneaAutorizacion.VersionRol.Concesiones = s.InstantaneaAutorizacion.VersionRol.Concesiones[:len(s.InstantaneaAutorizacion.VersionRol.Concesiones)-1]
	sellarLotePerfilesPrueba(t, &s)
	if _, err := servicio.AplicarLoteOrdinario(context.Background(), s); !errors.Is(err, domain.ErrControlAdministracionPerfilesInvalido) || a.lotes != 0 {
		t.Fatal("lote sin concesión")
	}
	if _, err := servicio.PrepararLoteOrdinario(context.Background(), preparacionDesdeLotePrueba(s)); !errors.Is(err, domain.ErrControlAdministracionPerfilesInvalido) || p.llamadas != 0 {
		t.Fatal("preparación sin concesión")
	}
}

func TestServicioLotesAltaDePerfilNoOrdinarioNoLlegaALaAutoridad(t *testing.T) {
	servicio, s, a, _, c := servicioLotesPrueba(t)
	rol := c[s.Cambios[0].RolVersionRef]
	rol.Clase = domain.ClaseControlPerfilAdministrador
	c[s.Cambios[0].RolVersionRef] = rol
	if _, err := servicio.AplicarLoteOrdinario(context.Background(), s); err == nil || a.lotes != 0 {
		t.Fatal("alta de perfil no ordinario")
	}
}

func TestServicioLotesPreparaYCotejaLaRespuesta(t *testing.T) {
	servicio, s, _, p, _ := servicioLotesPrueba(t)
	solicitud := preparacionDesdeLotePrueba(s)
	if r, err := servicio.PrepararLoteOrdinario(context.Background(), solicitud); err != nil || p.llamadas != 1 || r.PersonaRef != solicitud.PersonaRef {
		t.Fatalf("preparación no entregada: %v", err)
	}
	p.mutar = func(r *domain.PreparacionLoteAdministracionPerfiles) { r.UnidadRef = "unidad:otra" }
	if _, err := servicio.PrepararLoteOrdinario(context.Background(), solicitud); err == nil {
		t.Fatal("preparación ajena aceptada")
	}
}

func TestServicioLotesExigeTodasLasDependencias(t *testing.T) {
	_, _, a, p, c := servicioLotesPrueba(t)
	reloj := &relojAutorizacionServicioPrueba{}
	if _, err := NuevoServicioLotesAdministracionPerfiles(c, a, nil, reloj); err == nil {
		t.Fatal("sin preparador")
	}
	if _, err := NuevoServicioLotesAdministracionPerfiles(nil, a, p, reloj); err == nil {
		t.Fatal("sin catálogo")
	}
}
