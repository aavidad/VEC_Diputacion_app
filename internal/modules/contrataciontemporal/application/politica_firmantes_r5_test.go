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

func TestPasoLogicoFirmaR5SoloEquivaleEnMismaPosicionYCatalogo(t *testing.T) {
	base := PasoLogicoFirmaR5{Documento: "informe_definitivo", PasoOrden: 1, CatalogoHuella: strings.Repeat("a", 64)}
	if !base.MismoPaso(base) {
		t.Fatal("la misma posicion publicada dejo de ser equivalente")
	}
	for nombre, otro := range map[string]PasoLogicoFirmaR5{
		"otro documento":    {Documento: "resolucion", PasoOrden: 1, CatalogoHuella: base.CatalogoHuella},
		"otro orden":        {Documento: base.Documento, PasoOrden: 2, CatalogoHuella: base.CatalogoHuella},
		"catalogo cambiado": {Documento: base.Documento, PasoOrden: 1, CatalogoHuella: strings.Repeat("b", 64)},
		"invalido":          {Documento: base.Documento, PasoOrden: 1},
	} {
		t.Run(nombre, func(t *testing.T) {
			if base.MismoPaso(otro) {
				t.Fatal("se equipararon pasos de identidad distinta o ambigua")
			}
		})
	}
}

func TestCoincidenciaPersonaR5DefaultNoYPermisoPublicado(t *testing.T) {
	for _, caso := range []struct {
		nombre                         string
		permite, coincide, desconocido bool
		denegado                       bool
	}{
		{"sin coincidencia", false, false, false, false},
		{"misma persona otro paso", false, true, false, true},
		{"legacy sin identidad otro paso", false, false, true, true},
		{"permiso publicado", true, true, false, false},
		{"permiso publicado con legacy", true, false, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			err := EvaluarCoincidenciaPersonaR5(caso.permite, caso.coincide, caso.desconocido)
			if errors.Is(err, ports.ErrMismaPersonaEnOtroPasoR5) != caso.denegado {
				t.Fatalf("decision inesperada: %v", err)
			}
		})
	}
}
