package reglas

import (
	"errors"
	"testing"
	"time"
)

func TestProyeccionUsaCabezaAutorizadaSinVolverALeerAjustes(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 1, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	proyectadas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	for _, regla := range proyectadas {
		if regla.Clave != CTPlazoFiscalizacion {
			continue
		}
		if regla.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != 1 ||
			regla.AjusteNoAplicable || regla.ReferenciaEntrada.CatalogoID != vigente.CatalogoID {
			t.Fatalf("proyección no aplicó cabeza CT148: %+v", regla)
		}
		return
	}
	t.Fatal("falta c03")
}

func TestProyeccionAisladaMarcaAjusteIncompatible(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 2, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "61"},
	})
	proyectadas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	marcada, otra := false, false
	for _, regla := range proyectadas {
		switch regla.Clave {
		case CTPlazoFiscalizacion:
			marcada = regla.AjusteNoAplicable && regla.Ajuste == nil
		case CTPlazoSubsanacion:
			otra = !regla.AjusteNoAplicable && regla.Cantidad == 10
		}
	}
	if !marcada || !otra {
		t.Fatalf("ajuste inválido contaminó catálogo: marcada=%v otra=%v", marcada, otra)
	}
}

func TestProyeccionRechazaCabezaFuturaYUsaVersionEfectiva(t *testing.T) {
	resolutor := resolutorReal(t, rutaReglasCTPrueba, CatalogoContratacionTemporal, ModuloContratacionTemporal, nil)
	base, _, instante, err := resolutor.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vigente := versionAjustes(t, 1, instante.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	futura := versionAjustes(t, 2, instante.Add(time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "9"},
	})
	if _, err := ProyectarReglasConAjustes(base, instante, &futura); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("cabeza futura aplicada antes del efecto: %v", err)
	}
	reglas, err := ProyectarReglasConAjustes(base, instante, &vigente)
	if err != nil {
		t.Fatal(err)
	}
	for _, regla := range reglas {
		if regla.Clave == CTPlazoFiscalizacion {
			if regla.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != 1 {
				t.Fatalf("versión efectiva omitida: %+v", regla)
			}
			return
		}
	}
	t.Fatal("falta la regla proyectada")
}
