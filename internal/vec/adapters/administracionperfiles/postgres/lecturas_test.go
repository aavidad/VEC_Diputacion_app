package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestLecturasNoPierdenFiltrosDelContratoHTTP(t *testing.T) {
	if err := consultaLecturaHeredada(api.ConsultaPersonas{Texto: "Ana", Cursor: "cursor-anterior"}); err != nil {
		t.Fatal(err)
	}
	for _, consulta := range []api.ConsultaPersonas{
		{}, {Texto: "Ana", PerfilRef: "rol:rrhh:v1"}, {Texto: "Ana", UnidadRef: "unidad:rrhh"},
		{Texto: "Ana", Estado: "vigente"}, {Texto: strings.Repeat("a", 81)},
	} {
		if !errors.Is(consultaLecturaHeredada(consulta), ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
			t.Fatal("un filtro no soportado se elimina para usar la consulta antigua")
		}
	}
	for _, consulta := range []api.ConsultaPersonas{
		{Texto: "a"}, {Texto: " Ana"}, {Texto: "Ana\n"}, {Texto: "Ana", Estado: "activo"},
		{Texto: "Ana", PerfilRef: "rol:*:v1"}, {Texto: "Ana", UnidadRef: "unidad:*"},
		{Texto: "Ana", Cursor: strings.Repeat("a", 257)},
	} {
		if !errors.Is(consultaLecturaHeredada(consulta), domain.ErrActoAdministracionPerfilesInvalido) {
			t.Fatal("consulta HTTP malformada aceptada")
		}
	}
}

func TestLecturasClasificacionFijoDebeProcederDeLaFuente(t *testing.T) {
	for _, dato := range []string{`{"roles":[{"fijo":true}]}`, `{"roles":[{"fijo":false}]}`, `{"roles":[]}`} {
		if !fijosLecturaPresentes([]byte(dato)) {
			t.Fatal("clasificación explícita rechazada")
		}
	}
	for _, dato := range []string{`{"roles":[{}]}`, `{"roles":[{"fijo":null}]}`, `{"roles":null}`, `{}`} {
		if fijosLecturaPresentes([]byte(dato)) {
			t.Fatal("clasificación ausente convertida en gestionable")
		}
	}
}

func TestLecturasPreimagenConservaCentroYFechaInicial(t *testing.T) {
	ahora := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	o := api.Objetivo{CentroRef: "centro:nominal", VigenteDesde: ahora, VigenteHasta: ahora.Add(time.Hour),
		CuentaRef: "cta_" + strings.Repeat("a", 24), CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("b", 24), PersonaVersion: 1,
		PerfilRef: "prf_" + strings.Repeat("c", 24), VinculoRef: "vca_" + strings.Repeat("d", 24), HuellaSHA256: strings.Repeat("e", 64),
		ProcedenciaRef: "procedencia:nominal", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("f", 64)}
	p := preimagenLectura(o)
	if p.CentroRef != o.CentroRef || !p.VigenteDesde.Equal(o.VigenteDesde) || p.HuellaSHA256 != o.HuellaSHA256 || p.ValidarPara(domain.OperacionOtorgarPerfil, domain.ClaseControlPerfilOrdinario) != nil {
		t.Fatal("la proyección recorta el ámbito o la vigencia central")
	}
	o.PerfilVersion, o.VinculoVersion = 1, 1
	o.VigenteDesde, o.VigenteHasta = time.Time{}, time.Time{}
	if preimagenLectura(o).ValidarPara(domain.OperacionRevocarPerfil, domain.ClaseControlPerfilOrdinario) != nil {
		t.Fatal("la proyección altera las fechas cero de revocación")
	}
}

func TestLecturasPropuestaNoHabilitaCierreSinMaterialNominal(t *testing.T) {
	ahora := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	a := domain.ContextoActor{PersonaRef: "per_" + strings.Repeat("a", 24)}
	m := api.MotivoLectura{Motivo: api.Motivo{CatalogoID: "motivos_administracion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "motivo_" + strings.Repeat("f", 32)}, Etiqueta: "Cambio de funciones"}
	p := api.Propuesta{PropuestaRef: "propuesta_admin:" + strings.Repeat("b", 32), ProponentePersonaRef: "per_" + strings.Repeat("c", 24),
		ObjetivoPersonaRef: "per_" + strings.Repeat("d", 24), ObjetivoNombre: "Persona sintética", RolVersionRef: "rol:rrhh:v1", Operacion: "otorgar", HuellaSHA256: strings.Repeat("e", 64),
		CaducaEn: ahora.Add(time.Hour), PuedeCerrar: true, MotivosCierre: []api.MotivoLectura{m}, ProponenteNombre: "Proponente sintética", Motivo: &m,
		Ambitos: []api.AmbitoPerfil{{Dimension: "unidad", Referencia: "unidad:rrhh", Nombre: "Recursos Humanos"}}, VigenteDesde: ahora, VigenteHasta: ahora.Add(2 * time.Hour)}
	if !validarPropuestaLectura(p, a, ahora) {
		t.Fatal("material nominal completo rechazado")
	}
	for _, cambiar := range []func(*api.Propuesta){
		func(p *api.Propuesta) { p.ProponenteNombre = "" }, func(p *api.Propuesta) { p.Ambitos = nil }, func(p *api.Propuesta) { p.Motivo = nil },
		func(p *api.Propuesta) { p.VigenteDesde = time.Time{} }, func(p *api.Propuesta) { p.ProponentePersonaRef = a.PersonaRef },
		func(p *api.Propuesta) {
			p.Ambitos = []api.AmbitoPerfil{{Dimension: "unidad", Referencia: "*", Nombre: "Recursos Humanos"}}
		},
	} {
		copia := p
		cambiar(&copia)
		if validarPropuestaLectura(copia, a, ahora) {
			t.Fatal("cierre habilitado sin material o con la misma persona")
		}
	}
	p.PuedeCerrar = false
	p.ProponenteNombre = ""
	p.Ambitos = nil
	p.Motivo = nil
	p.VigenteDesde = time.Time{}
	if !validarPropuestaLectura(p, a, ahora) {
		t.Fatal("la propuesta de sólo lectura se confunde con una autorización de cierre")
	}
}

func TestLecturasHistoriaMinimizadaExigeIdentidadNominal(t *testing.T) {
	i := api.IdentidadHistoria{PersonaRef: "per_" + strings.Repeat("a", 24), PerfilActivoRef: "prf_" + strings.Repeat("b", 24), AsignacionRef: "asignacion:rrhh:v1", Nombre: "Persona sintética", PerfilActivoNombre: "Gestor de personal"}
	if !validarIdentidadHistoria(i) {
		t.Fatal("identidad nominal completa rechazada")
	}
	i.AsignacionRef = ""
	if validarIdentidadHistoria(i) {
		t.Fatal("historia sin asignación nominal aceptada")
	}
	for _, accion := range []string{"*", "superusuario", "administracion.perfiles.consultar"} {
		if accionCapacidadLectura(accion) {
			t.Fatal("la fuente amplía las capacidades del contrato HTTP")
		}
	}
}
