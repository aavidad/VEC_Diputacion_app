package bootstrap

import (
	"errors"
	"io"
	"strings"
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
	rutas, autoridad, cerrar, err := nuevasRutasContratacionTemporalDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, composicion.emisorKMS, io.Discard)
	if !errors.Is(err, config.ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) || rutas != nil || autoridad != nil || cerrar != nil {
		t.Fatalf("composición sin PostgreSQL: rutas=%v autoridad=%v cerrar_presente=%t err=%v", rutas, autoridad, cerrar != nil, err)
	}
	(&DependenciasCT{}).Cerrar()
}

func TestRutasCTConPostgreSQLMinimoExigenContextoRegistradoAntesDeArrancar(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	pg, err := config.NuevaConfiguracionPostgreSQLContratacionTemporal("ejecutor", "gobierno", "registro", "confirmador", "lector")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ContratacionTemporalPostgreSQL = pg
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	resolvedor, err := composicion.ResolvedorIdentidad()
	if err != nil {
		t.Fatal(err)
	}
	rutas, autoridad, cerrar, err := nuevasRutasContratacionTemporalDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia, composicion.emisorKMS, io.Discard)
	if !errors.Is(err, config.ErrConfiguracionPostgreSQLContratacionTemporalIncompleta) ||
		!strings.Contains(err.Error(), "contexto actor registrado") || rutas != nil || autoridad != nil || cerrar != nil {
		t.Fatalf("CT mínimo abrió rutas sin identidad operativa: rutas=%v autoridad=%v cierre=%t err=%v", rutas, autoridad, cerrar != nil, err)
	}
}
