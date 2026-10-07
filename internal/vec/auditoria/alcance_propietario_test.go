package auditoria

import (
	"errors"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func asignacionAlcancePrueba(p Peticion, ahora time.Time, dimension, alcance string) vecdomain.AsignacionPerfil {
	return vecdomain.AsignacionPerfil{
		AsignacionID: "asig-auditoria-prueba", Version: 1,
		PerfilActivoRef: p.Contexto.Resultado.Contexto.PerfilActivoRef,
		PrincipalID:     p.Contexto.Resultado.Contexto.Principal.ID,
		VersionRolRef:   "rol:auditoria_prueba:v1",
		Estado:          vecdomain.EstadoAsignacionPerfilActiva,
		Ambitos: []vecdomain.AmbitoPerfil{
			{Clave: "fuente", Valores: []string{p.Filtro.Fuente}},
			{Clave: dimension, Valores: []string{alcance}},
		},
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		EmitidaPor: "administracion:prueba", EmitidaEn: ahora.Add(-2 * time.Hour),
	}
}

func TestRecursoDinamicoUsaSoloAmbitoNominalDeLaAsignacion(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for _, caso := range []struct {
		fuente, dimension, alcance, recurso string
	}{
		{"ct", "organizacion_ref", "organizacion:rrhh:001", "expediente:ct:001"},
		{"bolsa", "bolsa_ref", "bolsa:constituida:001", "participacion:bolsa:001"},
	} {
		t.Run(caso.fuente, func(t *testing.T) {
			p := peticionAuditoriaIdentidadPrueba(t, ahora)
			p.Filtro.Fuente, p.Filtro.ExpedienteRef = caso.fuente, caso.recurso
			asignacion := asignacionAlcancePrueba(p, ahora, caso.dimension, caso.alcance)
			if err := asignacion.Validar(); err != nil {
				t.Fatal(err)
			}
			recurso, err := RecursoFiltroConAsignacionNominal(p.Filtro, p.Contexto, asignacion, ahora)
			huella, errHuella := HuellaFiltro(p.Filtro)
			if err != nil || errHuella != nil || recurso.Referencia != caso.recurso ||
				recurso.Ambitos["fuente"] != caso.fuente || recurso.Ambitos[caso.dimension] != caso.alcance ||
				len(recurso.Ambitos) != 2 || recurso.Atributos["filtro_sha256"] != huella {
				t.Fatalf("recurso fuera del alcance nominal: %+v %v", recurso, err)
			}
			p.Filtro.ExpedienteRef = caso.recurso + ":otro"
			otro, err := RecursoFiltroConAsignacionNominal(p.Filtro, p.Contexto, asignacion, ahora)
			if err != nil || otro.Referencia != p.Filtro.ExpedienteRef ||
				otro.Ambitos[caso.dimension] != caso.alcance ||
				otro.Atributos["filtro_sha256"] == recurso.Atributos["filtro_sha256"] {
				t.Fatalf("filtro distinto no quedó ligado: %+v %v", otro, err)
			}
		})
	}
}

func TestRecursoDinamicoDeniegaAmbitoAjenoYAsignacionAmbigua(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := peticionAuditoriaIdentidadPrueba(t, ahora)
	a := asignacionAlcancePrueba(p, ahora, "organizacion_ref", "organizacion:rrhh:001")
	casos := []struct {
		nombre  string
		cambiar func(*vecdomain.AsignacionPerfil, *Filtro)
	}{
		{"fuente ajena", func(a *vecdomain.AsignacionPerfil, _ *Filtro) { a.Ambitos[0].Valores[0] = "bolsa" }},
		{"dimensión ajena", func(a *vecdomain.AsignacionPerfil, _ *Filtro) { a.Ambitos[1].Clave = "bolsa_ref" }},
		{"varios ámbitos", func(a *vecdomain.AsignacionPerfil, _ *Filtro) {
			a.Ambitos[1].Valores = []string{"organizacion:rrhh:001", "organizacion:rrhh:002"}
		}},
		{"tercera dimensión", func(a *vecdomain.AsignacionPerfil, _ *Filtro) {
			a.Ambitos = append(a.Ambitos, vecdomain.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:001"}})
		}},
		{"perfil ajeno", func(a *vecdomain.AsignacionPerfil, _ *Filtro) { a.PerfilActivoRef = "prf_otro_perfil_0123456789" }},
		{"recurso inválido", func(_ *vecdomain.AsignacionPerfil, f *Filtro) { f.ExpedienteRef = "*" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := a
			actual.Ambitos = []vecdomain.AmbitoPerfil{
				{Clave: a.Ambitos[0].Clave, Valores: append([]string(nil), a.Ambitos[0].Valores...)},
				{Clave: a.Ambitos[1].Clave, Valores: append([]string(nil), a.Ambitos[1].Valores...)},
			}
			f := p.Filtro
			caso.cambiar(&actual, &f)
			if r, err := RecursoFiltroConAsignacionNominal(f, p.Contexto, actual, ahora); r.Referencia != "" || !errors.Is(err, ErrDenegada) {
				t.Fatalf("ámbito ajeno creó recurso: %+v %v", r, err)
			}
		})
	}
	if r, err := RecursoFiltroConAsignacionNominal(p.Filtro, p.Contexto, a, ahora.Add(31*time.Minute)); r.Referencia != "" || !errors.Is(err, ErrDenegada) {
		t.Fatal("vínculo caducado permitió construir el recurso")
	}
}
