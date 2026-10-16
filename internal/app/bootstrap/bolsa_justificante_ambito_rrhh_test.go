package bootstrap

import (
	"errors"
	"testing"
	"time"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestJustificanteRRHHUsaSoloAmbitoNominalPropioYUnivoco(t *testing.T) {
	sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, time.Now().UTC().Truncate(time.Microsecond))
	unidad, ambito, err := ambitoPropioJustificanteRRHH(t.Context(), fuente, sesion.Resultado, sesion.Vinculo)
	if err != nil || unidad != "unidad:prueba" || ambito != "ambito:prueba" || fuente.principal != sesion.Resultado.Contexto.PersonaRef ||
		fuente.perfil != sesion.Resultado.Contexto.PerfilActivoRef {
		t.Fatalf("ámbito nominal propio perdido: %v", err)
	}
	for _, mutar := range []func(*fuenteLecturaActualInscripcionPrueba){
		func(f *fuenteLecturaActualInscripcionPrueba) {
			f.i.AsignacionPerfil.Ambitos[0].Valores = []string{"unidad:uno", "unidad:dos"}
		},
		func(f *fuenteLecturaActualInscripcionPrueba) {
			f.i.AsignacionPerfil.Ambitos[0].Clave = "otra_dimension"
		},
		func(f *fuenteLecturaActualInscripcionPrueba) {
			f.i.AsignacionPerfil.PrincipalID = "per_abcdefghijkl0123456789"
		},
	} {
		sesion, fuente := escenarioLecturaRespuestasLlamamiento(t, time.Now().UTC().Truncate(time.Microsecond))
		mutar(fuente)
		if _, _, err := ambitoPropioJustificanteRRHH(t.Context(), fuente, sesion.Resultado, sesion.Vinculo); !errors.Is(err, vec.ErrAutorizacionDenegada) {
			t.Fatal("se aceptó una asignación ajena, ambigua o con dimensión desconocida")
		}
	}
}
