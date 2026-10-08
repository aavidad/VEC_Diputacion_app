package bootstrap

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
)

func clavesManifiestos(t *testing.T, cfg config.Config) []string {
	t.Helper()
	visibles, err := cfg.PortalModulosVisibles()
	if err != nil {
		t.Fatalf("lista visible: %v", err)
	}
	filtrados, err := filtrarManifiestosPortal(manifiestosShellVEC(cfg), visibles)
	if err != nil {
		t.Fatalf("filtrar: %v", err)
	}
	ids := make([]string, 0, len(filtrados))
	for _, m := range filtrados {
		ids = append(ids, m.ID)
	}
	return ids
}

func TestPortalModulosVisiblesSinListaConservaTodos(t *testing.T) {
	cfg := config.Config{}
	if got, want := len(clavesManifiestos(t, cfg)), len(manifiestosShellVEC(cfg)); got != want {
		t.Fatalf("sin lista: %d módulos, esperados %d", got, want)
	}
}

func TestPortalModulosVisiblesOcultaLosNoListados(t *testing.T) {
	cfg := config.Config{PortalModulosVisiblesLista: " bolsa, contratacion_temporal ,administracion,usuarios"}
	ids := clavesManifiestos(t, cfg)
	permitidos := map[string]bool{
		"vec.module.bolsa": true, "vec.module.contratacion_temporal": true,
		"vec.module.administracion": true, "vec.module.usuarios": true,
	}
	if len(ids) != len(permitidos) {
		t.Fatalf("visibles=%v", ids)
	}
	for _, id := range ids {
		if !permitidos[id] {
			t.Fatalf("módulo no listado visible: %s", id)
		}
	}
}

func TestPortalModulosVisiblesFallaCerradoAnteClaveDesconocida(t *testing.T) {
	cfg := config.Config{PortalModulosVisiblesLista: "bolsa,nominas"}
	visibles, err := cfg.PortalModulosVisibles()
	if err != nil {
		t.Fatalf("formato válido rechazado: %v", err)
	}
	if _, err := filtrarManifiestosPortal(manifiestosShellVEC(cfg), visibles); !errors.Is(err, config.ErrConfiguracionPortalModulos) {
		t.Fatalf("clave desconocida aceptada: %v", err)
	}
}

func TestPortalModulosVisiblesRechazaFormatoInvalido(t *testing.T) {
	for _, valor := range []string{"bolsa,,dietas", "bolsa,bolsa", "vec.module.bolsa", "Bolsa", "bolsa;dietas"} {
		if _, err := (config.Config{PortalModulosVisiblesLista: valor}).PortalModulosVisibles(); !errors.Is(err, config.ErrConfiguracionPortalModulos) {
			t.Fatalf("%q aceptado: %v", valor, err)
		}
	}
}
