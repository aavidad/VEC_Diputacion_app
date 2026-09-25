package reglas

import (
	"errors"
	"testing"
)

const rutaRetribucionesCTPrueba = "../../../data/demo/reglas/ct_retribuciones.demo.json"

func TestCatalogoVigenteDevuelveCatalogosDeFormatoPropio(t *testing.T) {
	resolutor := resolutorReal(t, rutaRetribucionesCTPrueba, CatalogoRetribucionesCT, ModuloContratacionTemporal, nil)
	catalogo, huella, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if catalogo.ID != CatalogoRetribucionesCT || catalogo.FuenteRef != MarcaPaqueteEjemplo ||
		len(huella) != 64 || !instante.Equal(diaPresentacion.Ahora()) || len(catalogo.Entradas) < 6 {
		t.Fatalf("catálogo inesperado: %s %s %q %v %d", catalogo.ID, catalogo.FuenteRef, huella, instante, len(catalogo.Entradas))
	}
	// Sus entradas no siguen el contrato de las reglas: Reglas las rechaza.
	if _, err := resolutor.Reglas(t.Context()); !errors.Is(err, ErrReglaInvalida) {
		t.Fatalf("las retribuciones no son reglas: %v", err)
	}
	var nulo *Resolutor
	if _, _, _, err := nulo.CatalogoVigente(t.Context()); !errors.Is(err, ErrReglasNoConfiguradas) {
		t.Fatalf("sin resolutor: %v", err)
	}
	antes := resolutorReal(t, rutaRetribucionesCTPrueba, CatalogoRetribucionesCT, ModuloContratacionTemporal, nil)
	antes.cfg.Reloj = relojFijo(catalogo.PublicadoEn.Add(-1))
	if _, _, _, err := antes.CatalogoVigente(t.Context()); !errors.Is(err, ErrReglasNoDisponibles) {
		t.Fatalf("antes de publicarse no hay catálogo vigente: %v", err)
	}
}
