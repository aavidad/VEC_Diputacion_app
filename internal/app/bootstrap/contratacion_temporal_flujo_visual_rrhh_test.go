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
	if resultado.Huella != "4c0c91dad747199168610639d0840be5ed3e3df19a831583e32c408ebda23650" ||
		len(resultado.Fases) != 8 || resultado.FaseActual != "analisis_rrhh" {
		t.Fatalf("presentación RRHH inesperada: %#v", resultado)
	}
	resultado, err = lector.Resolver(context.Background(), origen, "fiscalizacion")
	if err != nil || resultado.FaseActual != "" {
		t.Fatalf("una fase no acreditada quedó resaltada: %#v, %v", resultado, err)
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
		"mapeo_alterado":        `"fase_presentacion": "gestion_bolsa"`,
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
