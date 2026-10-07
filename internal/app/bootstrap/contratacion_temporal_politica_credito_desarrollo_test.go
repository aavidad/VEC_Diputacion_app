package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaReglasCTEjemploV4Prueba = "../../../data/demo/reglas/ct_reglas.ejemplo.demo.v4.json"

func resolutorCreditoPrueba(t *testing.T, ruta string) *reglas.Resolutor {
	t.Helper()
	resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoContratacionTemporal,
		reglas.ModuloContratacionTemporal, nil, relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

// catalogoCreditoPrueba copia el catálogo v4 con otro valor de c25.
func catalogoCreditoPrueba(t *testing.T, valor string) string {
	t.Helper()
	contenido, err := os.ReadFile(rutaReglasCTEjemploV4Prueba)
	if err != nil {
		t.Fatal(err)
	}
	var datos map[string]any
	if err := json.Unmarshal(contenido, &datos); err != nil {
		t.Fatal(err)
	}
	for _, entrada := range datos["catalogo"].(map[string]any)["entradas"].([]any) {
		e := entrada.(map[string]any)
		if e["clave"] == reglas.CTCreditoOferta {
			e["atributos"].(map[string]any)[reglas.AtributoCosteConPartidas] = valor
		}
	}
	contenido, err = json.Marshal(datos)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "ct_reglas.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestPoliticaCreditoOfertaSigueLaReglaC25(t *testing.T) {
	ctx := context.Background()
	casos := []struct {
		nombre  string
		reglas  *reglas.Resolutor
		exige   bool
		fallido bool
	}{
		{"sin catálogo rige la decisión de RRHH", nil, true, false},
		{"catálogo v3 sin la regla", resolutorCreditoPrueba(t, "../../../data/demo/reglas/ct_reglas.ejemplo.demo.v3.json"), true, false},
		{"catálogo v4 la exige", resolutorCreditoPrueba(t, rutaReglasCTEjemploV4Prueba), true, false},
		{"RRHH la desactiva", resolutorCreditoPrueba(t, catalogoCreditoPrueba(t, "no_exigido")), false, false},
		{"valor desconocido falla cerrado", resolutorCreditoPrueba(t, catalogoCreditoPrueba(t, "quizas")), false, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			politica, err := politicaCreditoOfertaDesarrollo{reglas: caso.reglas}.PoliticaCreditoOferta(ctx)
			if caso.fallido {
				if !errors.Is(err, ports.ErrPoliticaCreditoOfertaNoDisponible) {
					t.Fatalf("debía fallar cerrado: %v", err)
				}
				return
			}
			if err != nil || politica.ExigeCosteConPartidas != caso.exige {
				t.Fatalf("política = %+v, %v; se esperaba exige=%v", politica, err, caso.exige)
			}
		})
	}
}
