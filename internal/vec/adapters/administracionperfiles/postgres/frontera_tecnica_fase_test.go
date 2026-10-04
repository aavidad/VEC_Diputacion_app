package postgres

import (
	"context"
	"testing"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

func TestFronteraSesionResueltaIncompatibleNuncaEsTecnica(t *testing.T) {
	p := &poolTecnicoPrueba{}
	nominal := &nominalTecnicoPrueba{}
	a, err := NuevoAuditorFronteraCompuesto(nominal, registradorTecnicoPrueba(p))
	if err != nil {
		t.Fatal(err)
	}
	err = a.RegistrarDenegacionADMIN(context.Background(), api.DenegacionADMIN{SesionResuelta: true, Codigo: "respuesta_incompatible"})
	if err == nil || len(p.eventos) != 0 || nominal.llamadas != 0 {
		t.Fatal("sesion_resuelta_downgrade")
	}
}
