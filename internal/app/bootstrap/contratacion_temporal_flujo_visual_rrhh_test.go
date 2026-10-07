package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestLectorFlujoVisualRRHHResuelveSoloMapeosAcreditados(t *testing.T) {
	contenido, err := os.ReadFile("../../../config/contratacion_temporal_flujo_visual_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	lector, err := LeerLectorFlujoVisualRRHH(strings.NewReader(string(contenido)))
	if err != nil {
		t.Fatal(err)
	}
	origen := domain.ReferenciaFlujo{
		DefinicionRef: "flujo:ct:desarrollo", Version: 1,
		HuellaSHA256: "d0b1d05c902fab11f2ffa482eda8bfd0598ffb77ef3824e93fa301cdd06cd47a",
	}
	resultado, err := lector.Resolver(context.Background(), origen, "analisis")
	if err != nil {
		t.Fatal(err)
	}
	if len(resultado.Fases) != 8 || resultado.FaseActual != "analisis_rrhh" {
		t.Fatalf("presentación RRHH inesperada: %#v", resultado)
	}
	resultado, err = lector.Resolver(context.Background(), origen, "fiscalizacion")
	if err != nil || resultado.FaseActual != "fiscalizacion" {
		t.Fatalf("la fiscalización no quedó resaltada: %#v, %v", resultado, err)
	}
}

func TestLectorFlujoVisualRRHHRechazaOrigenYManifestAlterados(t *testing.T) {
	contenido, err := os.ReadFile("../../../config/contratacion_temporal_flujo_visual_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	lector, err := LeerLectorFlujoVisualRRHH(strings.NewReader(string(contenido)))
	if err != nil {
		t.Fatal(err)
	}
	origenAjeno := domain.ReferenciaFlujo{
		DefinicionRef: "flujo:ct:desarrollo", Version: 1,
		HuellaSHA256: strings.Repeat("a", 64),
	}
	if _, err := lector.Resolver(context.Background(), origenAjeno, "analisis"); !errors.Is(err, ErrManifestFlujoVisualRRHHInvalido) {
		t.Fatalf("origen alterado aceptado: %v", err)
	}
	alterado := strings.Replace(
		string(contenido), `"fases": [`, `"desconocido": true, "fases": [`, 1,
	)
	if lector, err := LeerLectorFlujoVisualRRHH(strings.NewReader(alterado)); lector != nil || !errors.Is(err, ErrManifestFlujoVisualRRHHInvalido) {
		t.Fatalf("campo ajeno aceptado: %#v, %v", lector, err)
	}
	for nombre, reemplazo := range map[string]string{
		"mapeo_alterado":        `"fase_presentacion": "fase_inexistente"`,
		"fase_actual_publicada": `"fase_actual": "analisis_rrhh", "fases": [`,
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := string(contenido)
			if nombre == "mapeo_alterado" {
				alterado = strings.Replace(alterado, `"fase_presentacion": "analisis_rrhh"`, reemplazo, 1)
			} else {
				alterado = strings.Replace(alterado, `"fases": [`, reemplazo, 1)
			}
			if lector, err := LeerLectorFlujoVisualRRHH(strings.NewReader(alterado)); lector != nil || !errors.Is(err, ErrManifestFlujoVisualRRHHInvalido) {
				t.Fatalf("manifest alterado aceptado: %#v, %v", lector, err)
			}
		})
	}
}

func TestFlujoVisualRRHHV2SigueHuellaAdministrativaPublicada(t *testing.T) {
	d, err := cargarDefinicionCircuitoRRHH("../../../config/contratacion_temporal_circuito_rrhh_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := os.ReadFile("../../../config/contratacion_temporal_flujo_visual_rrhh_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	lector, err := LeerLectorFlujoVisualRRHH(strings.NewReader(string(contenido)))
	if err != nil {
		t.Fatal(err)
	}
	if resultado, err := lector.Resolver(t.Context(), d.Flujo, "resolucion"); err != nil || len(resultado.Fases) != 12 {
		t.Fatalf("la presentación debe seguir la misma definición administrativa: fases=%d err=%v", len(resultado.Fases), err)
	}
	alterado := d.Flujo
	alterado.HuellaSHA256 = strings.Repeat("a", 64)
	if _, err := lector.Resolver(t.Context(), alterado, "resolucion"); !errors.Is(err, ErrManifestFlujoVisualRRHHInvalido) {
		t.Fatalf("la presentación aceptó otra huella: %v", err)
	}
}
