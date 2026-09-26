package reglas

import (
	"errors"
	"testing"
	"time"
)

// Un asunto urgente usa la cantidad urgente de la regla (c03: cinco días en
// lugar de diez) y, si la regla no la tiene, la ordinaria.
func TestVencimientoUrgenteUsaLaCantidadUrgenteDelCatalogo(t *testing.T) {
	fin := time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)
	calculadora := &calculadoraFalsa{resultado: Vencimiento{UltimoDia: "2026-10-09", VenceAntesDe: fin}}
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, calculadora)
	inicio := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	if _, _, err := resolutor.Vencimiento(t.Context(), CTPlazoFiscalizacion, inicio, ""); err != nil || calculadora.recibida.Cantidad != 10 {
		t.Fatalf("ordinario: cantidad %d, %v", calculadora.recibida.Cantidad, err)
	}
	regla, _, err := resolutor.VencimientoUrgente(t.Context(), CTPlazoFiscalizacion, inicio, "")
	if err != nil || calculadora.recibida.Cantidad != 5 || regla.Clave != CTPlazoFiscalizacion {
		t.Fatalf("urgente: cantidad %d, %v", calculadora.recibida.Cantidad, err)
	}
	if _, _, err := resolutor.VencimientoUrgente(t.Context(), CTPlazoSubsanacion, inicio, ""); err != nil || calculadora.recibida.Cantidad != 10 {
		t.Fatalf("sin cantidad urgente rige la ordinaria: %d, %v", calculadora.recibida.Cantidad, err)
	}
}

func TestVencimientoUrgenteRechazaCantidadUrgenteMalFormada(t *testing.T) {
	regla := Regla{Unidad: UnidadDiasHabiles, Cantidad: 10, Computo: ComputoAdministrativo, Inicio: "x"}
	for _, texto := range []string{"0", "-1", "05", "cinco", "100001"} {
		regla.Atributos = map[string]string{AtributoCantidadUrgente: texto}
		if _, err := cantidadPlazo(regla, true); !errors.Is(err, ErrReglaInvalida) {
			t.Errorf("%q admitida: %v", texto, err)
		}
		if cantidad, err := cantidadPlazo(regla, false); err != nil || cantidad != 10 {
			t.Errorf("el cálculo ordinario no depende de %q: %d, %v", texto, cantidad, err)
		}
	}
}
