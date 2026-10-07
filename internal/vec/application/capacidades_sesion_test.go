package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type selectorCapacidadesSesionPrueba struct {
	seleccion ports.SeleccionSesion
	err       error
}

type relojSecuenciaCapacidadesSesionPrueba struct {
	instantes []time.Time
	indice    int
}

func (r *relojSecuenciaCapacidadesSesionPrueba) Ahora() time.Time {
	if r.indice < len(r.instantes)-1 {
		r.indice++
		return r.instantes[r.indice-1]
	}
	return r.instantes[len(r.instantes)-1]
}

func (s selectorCapacidadesSesionPrueba) SeleccionarSesion(context.Context, domain.Principal) (ports.SeleccionSesion, error) {
	return s.seleccion, s.err
}

func TestCapacidadesSesionCruzaPerfilVigentePoliticasYMontaje(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.VersionRol.Concesiones[0].CamposPermitidos = []string{"nombre", "dni"}
	instantanea.VersionRol.Concesiones[0].Obligaciones = []string{"auditar"}
	instantanea.VersionRol.Concesiones = append(instantanea.VersionRol.Concesiones, domain.ConcesionRol{
		Accion: "personal.consultar", ModuloID: "personal", TipoRecurso: "ficha",
		Finalidades: []string{"gestion_personal"}, GarantiaMinima: domain.AuthAssuranceSubstantial,
	})
	instantanea.Politicas = []domain.PoliticaRestrictiva{{
		PoliticaID: "campos_bolsa", Version: 1, Nombre: "Campos bolsa", Estado: domain.EstadoPoliticaRestrictivaPublicada,
		Efecto: domain.EfectoPoliticaRestringir, Acciones: []string{"bolsa.expediente.leer"},
		Modulos: []string{"bolsa"}, TiposRecurso: []string{"expediente"},
		FinalidadesPermitidas: []string{"gestion_bolsa"}, RestringeCampos: true,
		CamposPermitidos: []string{"nombre"}, Obligaciones: []string{"registrar_consulta"},
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(30 * time.Minute),
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-2 * time.Hour),
	}}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(instantanea.Politicas)
	if err != nil {
		t.Fatal(err)
	}
	instantanea.CatalogoPoliticasHuellaSHA256 = huella
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("fixture invalida: %v", err)
	}
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	selector := selectorCapacidadesSesionPrueba{seleccion: ports.SeleccionSesion{
		PerfilActivoRef: instantanea.AsignacionPerfil.PerfilActivoRef,
		Superficie:      "interna", VigenteHasta: ahora.Add(2 * time.Hour),
		Exposiciones: []ports.ExposicionSesion{
			{Superficie: "interna", ModuloID: "bolsa", TipoRecurso: "expediente", Accion: "bolsa.expediente.leer"},
			{Superficie: "externa", ModuloID: "personal", TipoRecurso: "ficha", Accion: "personal.consultar"},
		},
	}}
	proyector, err := NuevoProyectorCapacidadesSesion(fuente, selector, &relojAutorizacionServicioPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := proyector.Proyectar(context.Background(), solicitudAutorizacionServicioPrueba().Principal)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultado.Capacidades) != 1 || resultado.Capacidades[0].ModuloID != "bolsa" ||
		len(resultado.Capacidades[0].CamposPermitidos) != 1 || resultado.Capacidades[0].CamposPermitidos[0] != "nombre" ||
		len(resultado.Capacidades[0].Obligaciones) != 2 ||
		resultado.PerfilActivoRef != instantanea.AsignacionPerfil.PerfilActivoRef ||
		resultado.VersionRolRef != instantanea.VersionRol.Referencia() ||
		resultado.Revisiones.Asignacion != instantanea.AsignacionPerfil.Version ||
		resultado.Revisiones.ControlRol != instantanea.ControlVigenciaVersionRol.Revision ||
		resultado.Revisiones.CatalogoPoliticas != instantanea.RevisionCatalogoPoliticas ||
		!resultado.VigenteHasta.Equal(ahora.Add(30*time.Minute)) {
		t.Fatalf("proyeccion sin interseccion exacta: %+v", resultado)
	}
	if len(resultado.Capacidades[0].Ambitos) != 1 || resultado.Capacidades[0].Ambitos[0].Valores[0] != "seleccion" {
		t.Fatalf("ambito perdido: %+v", resultado.Capacidades[0].Ambitos)
	}
}

func TestCapacidadesSesionNoConfundeAusenciaConFalloYDeniegaAsignacionAjena(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	principal := solicitudAutorizacionServicioPrueba().Principal
	selector := selectorCapacidadesSesionPrueba{seleccion: ports.SeleccionSesion{
		PerfilActivoRef: instantanea.AsignacionPerfil.PerfilActivoRef,
		Superficie:      "interna", VigenteHasta: ahora.Add(time.Hour),
	}}
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	proyector, err := NuevoProyectorCapacidadesSesion(fuente, selector, &relojAutorizacionServicioPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	vacia, err := proyector.Proyectar(context.Background(), principal)
	if err != nil || vacia.Capacidades == nil || len(vacia.Capacidades) != 0 {
		t.Fatalf("sin exposiciones montadas: %+v, %v", vacia, err)
	}
	fuente.err = ports.ErrFuenteAutorizacionNoDisponible
	if _, err := proyector.Proyectar(context.Background(), principal); !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("indisponibilidad confundida con cero capacidades: %v", err)
	}
	fuente.err = nil
	fuente.instantanea.AsignacionPerfil.PrincipalID = "otra-persona"
	if _, err := proyector.Proyectar(context.Background(), principal); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("asignacion ajena aceptada: %v", err)
	}
}

func TestCapacidadesSesionDescartaCargaCanceladaYPoliticaCondicionada(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	selector := selectorCapacidadesSesionPrueba{seleccion: ports.SeleccionSesion{
		PerfilActivoRef: instantanea.AsignacionPerfil.PerfilActivoRef,
		Superficie:      "interna", VigenteHasta: ahora.Add(time.Hour),
		Exposiciones: []ports.ExposicionSesion{{Superficie: "interna", ModuloID: "bolsa", TipoRecurso: "expediente", Accion: "bolsa.expediente.leer"}},
	}}
	ctx, cancelar := context.WithCancel(context.Background())
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea, despues: cancelar}
	proyector, err := NuevoProyectorCapacidadesSesion(fuente, selector, &relojAutorizacionServicioPrueba{ahora: ahora})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := proyector.Proyectar(ctx, solicitudAutorizacionServicioPrueba().Principal); !errors.Is(err, context.Canceled) {
		t.Fatalf("carga tardia publicada: %v", err)
	}

	fuente.despues = nil
	instantanea.Politicas = []domain.PoliticaRestrictiva{{
		PoliticaID: "condicion_recurso", Version: 1, Nombre: "Condicion recurso",
		Estado: domain.EstadoPoliticaRestrictivaPublicada, Efecto: domain.EfectoPoliticaRestringir,
		Acciones: []string{"bolsa.expediente.leer"}, Modulos: []string{"bolsa"}, TiposRecurso: []string{"expediente"},
		Restricciones: []domain.RestriccionAtributoRecurso{{Clave: "estado", ValoresPermitidos: []string{"abierto"}}},
		VigenteDesde:  ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		PublicadaPor: "seguridad", PublicadaEn: ahora.Add(-2 * time.Hour),
	}}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(instantanea.Politicas)
	if err != nil {
		t.Fatal(err)
	}
	instantanea.CatalogoPoliticasHuellaSHA256 = huella
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("fixture invalida: %v", err)
	}
	fuente.instantanea = instantanea
	resultado, err := proyector.Proyectar(context.Background(), solicitudAutorizacionServicioPrueba().Principal)
	if err != nil || len(resultado.Capacidades) != 0 {
		t.Fatalf("restriccion de recurso presentada como permiso global: %+v, %v", resultado, err)
	}
}

func TestCapacidadesSesionDeniegaSiCaducaMientrasLeeLaFuente(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	selector := selectorCapacidadesSesionPrueba{seleccion: ports.SeleccionSesion{
		PerfilActivoRef: instantanea.AsignacionPerfil.PerfilActivoRef,
		Superficie:      "interna", VigenteHasta: ahora.Add(time.Minute),
		Exposiciones: []ports.ExposicionSesion{{Superficie: "interna", ModuloID: "bolsa", TipoRecurso: "expediente", Accion: "bolsa.expediente.leer"}},
	}}
	reloj := &relojSecuenciaCapacidadesSesionPrueba{instantes: []time.Time{ahora, ahora.Add(2 * time.Minute)}}
	proyector, err := NuevoProyectorCapacidadesSesion(&fuenteAutorizacionServicioPrueba{instantanea: instantanea}, selector, reloj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := proyector.Proyectar(context.Background(), solicitudAutorizacionServicioPrueba().Principal); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("caducidad durante lectura aceptada: %v", err)
	}
}
