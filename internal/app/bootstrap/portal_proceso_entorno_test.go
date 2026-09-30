package bootstrap

import (
	"errors"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

func TestSeparacionDeniegaSiNoPuedeResolverDirectorioPersonalPostgreSQL(t *testing.T) {
	fallo := errors.New("usuario del sistema no disponible")
	for _, portal := range []string{"interno", "externo", "errata"} {
		t.Run(portal, func(t *testing.T) {
			llamadas := 0
			resolver := func() (separacionportales.Entorno, error) { llamadas++; return separacionportales.Entorno{}, fallo }
			_, err := comprobarSeparacionPortalConEntorno(config.Config{PortalProceso: portal}, resolver)
			if !errors.Is(err, fallo) || llamadas != 1 {
				t.Fatalf("entorno no comprobable aceptado: %v llamadas=%d", err, llamadas)
			}
		})
	}
}

func TestSeparacionCombinadaConservaPoliticaAnteErrorDeDirectorioPersonal(t *testing.T) {
	resolver := func() (separacionportales.Entorno, error) {
		return separacionportales.Entorno{}, errors.New("usuario no disponible")
	}
	portal, err := comprobarSeparacionPortalConEntorno(config.Config{}, resolver)
	if err != nil || portal != separacionportales.PortalCombinado {
		t.Fatalf("politica combinada alterada: %v %v", portal, err)
	}
}
