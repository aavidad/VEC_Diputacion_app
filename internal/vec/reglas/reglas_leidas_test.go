package reglas

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Los errores de ReglasLeidas son los de Vencimiento ante las mismas entradas.
func TestReglasLeidasRechazaComoVencimiento(t *testing.T) {
	var vacia ReglasLeidas
	if _, _, err := vacia.Vencimiento(t.Context(), "x", time.Time{}, "", false); !errors.Is(err, ErrReglaSinPlazo) {
		t.Fatalf("inicio cero: %v", err)
	}
	if _, _, err := vacia.Vencimiento(t.Context(), "x", time.Now(), "", false); !errors.Is(err, ErrReglasNoConfiguradas) {
		t.Fatalf("sin resolutor: %v", err)
	}
	conAjustes := ReglasLeidas{resolutor: &Resolutor{cfg: Configuracion{Ajustes: ajustesNulosPrueba{}}}}
	if _, _, err := conAjustes.Vencimiento(t.Context(), "x", time.Now(), "", false); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("con ajustes: %v", err)
	}
	leidas := ReglasLeidas{resolutor: &Resolutor{}, reglas: []Regla{{Clave: "a", AjusteNoAplicable: true}, {Clave: "b"}}}
	if _, _, err := leidas.Vencimiento(t.Context(), "a", time.Now(), "", false); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("ajuste no aplicable: %v", err)
	}
	if _, _, err := leidas.Vencimiento(t.Context(), "z", time.Now(), "", false); !errors.Is(err, ErrReglaNoEncontrada) {
		t.Fatalf("clave ausente: %v", err)
	}
	if _, _, err := leidas.Vencimiento(t.Context(), "b", time.Now(), "", false); !errors.Is(err, ErrReglaSinPlazo) {
		t.Fatalf("regla sin plazo: %v", err)
	}
	// Reglas devuelve copias: cambiarlas no altera la lectura.
	copia := leidas.Reglas()
	copia[1].Clave = "cambiada"
	if leidas.reglas[1].Clave != "b" {
		t.Fatal("Reglas devolvió un alias de la lectura")
	}
}

type ajustesNulosPrueba struct{}

func (ajustesNulosPrueba) AjustesVigentesEn(context.Context, string, time.Time) (VersionAjustes, bool, error) {
	return VersionAjustes{}, false, nil
}
