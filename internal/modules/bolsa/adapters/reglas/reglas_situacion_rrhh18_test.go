package reglas

import (
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

type relojRRHH18 struct{}

func (relojRRHH18) Ahora() time.Time { return time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC) }

func TestCatalogoRRHH18PublicaRevisionSinSuspension(t *testing.T) {
	consulta, err := fichero.NuevaConsultaCatalogos("../../../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v4.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa,
		Reloj: relojRRHH18{}, MunicipioSede: vecreglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	publicacion, hay, err := NuevasReglasSituacion(resolutor).PoliticaTransiciones(t.Context())
	if err != nil || !hay {
		t.Fatalf("política nueva: hay=%v error=%v", hay, err)
	}
	pares := publicacion.Politica.Pares()
	for _, par := range []string{"renuncia>en_revision", "no_disponible>en_revision", "en_revision>disponible", "en_revision>excluido"} {
		if !slices.Contains(pares, par) {
			t.Fatalf("falta transición %s: %v", par, pares)
		}
	}
	for _, par := range []string{"disponible>no_disponible", "disponible>disponible_desde", "renuncia>disponible", "no_disponible>disponible"} {
		if slices.Contains(pares, par) {
			t.Fatalf("transición suspendida %s: %v", par, pares)
		}
	}
}
