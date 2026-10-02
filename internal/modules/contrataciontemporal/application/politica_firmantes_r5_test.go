package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuentePoliticaFirmantesPrueba struct {
	p   ports.PoliticaMismaPersonaEnPasos
	err error
}

func (f fuentePoliticaFirmantesPrueba) PoliticaMismaPersonaEnPasos(context.Context, string, string) (ports.PoliticaMismaPersonaEnPasos, error) {
	return f.p, f.err
}

func TestPoliticaMismaPersonaEnPasosVersionadaFallaCerrada(t *testing.T) {
	ref, huella := "catalogo:firma:rrhh:v2", strings.Repeat("a", 64)
	if permite, err := ResolverPoliticaMismaPersonaEnPasos(context.Background(), nil, ref, huella); err != nil || permite {
		t.Fatalf("sin fuente no debe conceder: %v, %v", permite, err)
	}
	for _, permite := range []bool{false, true} {
		p := ports.PoliticaMismaPersonaEnPasos{CatalogoRef: ref, CatalogoHuella: huella, Permite: permite}
		obtenido, err := ResolverPoliticaMismaPersonaEnPasos(context.Background(), fuentePoliticaFirmantesPrueba{p: p}, ref, huella)
		if err != nil || obtenido != permite {
			t.Fatalf("valor publicado %v: %v, %v", permite, obtenido, err)
		}
	}
	ajena := ports.PoliticaMismaPersonaEnPasos{CatalogoRef: ref, CatalogoHuella: strings.Repeat("b", 64), Permite: true}
	if permite, err := ResolverPoliticaMismaPersonaEnPasos(context.Background(), fuentePoliticaFirmantesPrueba{p: ajena}, ref, huella); !errors.Is(err, ErrCircuitoFirmaNoDisponible) || permite {
		t.Fatalf("otra version no debe conceder: %v, %v", permite, err)
	}
	if permite, err := ResolverPoliticaMismaPersonaEnPasos(context.Background(), fuentePoliticaFirmantesPrueba{err: errors.New("caida")}, ref, huella); !errors.Is(err, ErrCircuitoFirmaNoDisponible) || permite {
		t.Fatalf("fuente caida no debe conceder: %v, %v", permite, err)
	}
}
