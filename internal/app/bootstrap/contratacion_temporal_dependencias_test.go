package bootstrap

import (
	"errors"
	"io"
	"testing"

	"vec-diputacion-granada/config"
)

func TestDependenciasCTCerrarEsIdempotente(t *testing.T) {
	var cierres int
	dependencias := &DependenciasCT{cerrar: func() { cierres++ }}
	dependencias.Cerrar()
	dependencias.Cerrar()
	if cierres != 1 {
		t.Fatalf("cierres=%d; se esperaba un único cierre", cierres)
	}
}

func TestRutasCTSinPostgreSQLFallaCerradoYSinCierre(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	resolvedor, err := composicion.ResolvedorIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	rutas, autoridad, cerrar, err := nuevasRutasContratacionTemporalDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, io.Discard)
	if !errors.Is(err, config.ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) || rutas != nil || autoridad != nil || cerrar != nil {
		t.Fatalf("composición sin PostgreSQL: rutas=%v autoridad=%v cerrar_presente=%t err=%v", rutas, autoridad, cerrar != nil, err)
	}
	(&DependenciasCT{}).Cerrar()
}
