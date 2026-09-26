package reglas

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

func resolutorAvisosPrueba(t *testing.T, cambios ...[2]string) *vecreglas.Resolutor {
	t.Helper()
	ruta := rutaReglasBolsaPrueba
	if len(cambios) > 0 {
		original, err := os.ReadFile(rutaReglasBolsaPrueba)
		if err != nil {
			t.Fatal(err)
		}
		texto := string(original)
		for _, c := range cambios {
			if !strings.Contains(texto, c[0]) {
				t.Fatalf("el catálogo de ejemplo no contiene %q", c[0])
			}
			texto = strings.Replace(texto, c[0], c[1], 1)
		}
		ruta = filepath.Join(t.TempDir(), "reglas.demo.json")
		if err := os.WriteFile(ruta, []byte(texto), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := vecreglas.NuevoResolutor(vecreglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: vecreglas.CatalogoBolsa, ModuloID: vecreglas.ModuloBolsa,
		Reloj: relojPrueba{}, MunicipioSede: vecreglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestPoliticaAvisosDelCatalogoDeEjemplo(t *testing.T) {
	publicacion, hay, err := PoliticaAvisos(t.Context(), resolutorAvisosPrueba(t))
	if err != nil || !hay {
		t.Fatalf("política: %v %v", hay, err)
	}
	p := publicacion.Politica
	if !p.ContinuadoConfigurado || p.ContinuadoMeses != 36 || p.ContinuadoAntelacionDias != 30 ||
		p.EncadenamientoUmbralMeses != 18 || p.EncadenamientoVentanaMeses != 24 ||
		p.PrestaServiciosModo != "aviso" || !slices.Equal(p.PrestaServiciosSituaciones, []string{"trabajando", "pendiente_incorporacion"}) {
		t.Fatalf("parámetros inesperados: %+v", p)
	}
	if publicacion.CatalogoRef != "vec.bolsa.reglas:1:avisos" || len(publicacion.CatalogoSHA256) != 64 {
		t.Fatalf("referencia: %+v", publicacion)
	}
}

func TestPoliticaAvisosCambiaConElCatalogoSinTocarCodigo(t *testing.T) {
	publicacion, _, err := PoliticaAvisos(t.Context(), resolutorAvisosPrueba(t,
		[2]string{`"antelacion_dias": "30"`, `"antelacion_dias": "60"`},
		[2]string{`"cantidad": "18",`, `"cantidad": "20",`},
		[2]string{`"modo": "aviso"`, `"modo": "excluir"`}))
	p := publicacion.Politica
	if err != nil || p.ContinuadoAntelacionDias != 60 || p.EncadenamientoUmbralMeses != 20 || p.PrestaServiciosModo != "excluir" {
		t.Fatalf("catálogo modificado: %+v %v", p, err)
	}
}

func TestPoliticaAvisosSinAtributosNoConfiguraYMalFormadaFalla(t *testing.T) {
	if _, hay, err := PoliticaAvisos(t.Context(), nil); hay || err != nil {
		t.Fatalf("sin catálogo: %v %v", hay, err)
	}
	_, hay, err := PoliticaAvisos(t.Context(), resolutorAvisosPrueba(t,
		[2]string{`"antelacion_dias": "30",`, ``},
		[2]string{`"ventana_meses": "24",`, ``},
		[2]string{`"modo": "aviso",`, ``},
		[2]string{`"situaciones": "trabajando,pendiente_incorporacion",`, ``}))
	if hay || err != nil {
		t.Fatalf("sin atributos rige la base: %v %v", hay, err)
	}
	for nombre, cambio := range map[string][2]string{
		"antelación no numérica": {`"antelacion_dias": "30"`, `"antelacion_dias": "treinta"`},
		"ventana menor":          {`"ventana_meses": "24"`, `"ventana_meses": "12"`},
		"modo desconocido":       {`"modo": "aviso"`, `"modo": "bloquear"`},
		"situación disponible":   {`"situaciones": "trabajando,pendiente_incorporacion"`, `"situaciones": "disponible"`},
		"modo sin situaciones":   {`"situaciones": "trabajando,pendiente_incorporacion",`, ``},
	} {
		if _, _, err := PoliticaAvisos(t.Context(), resolutorAvisosPrueba(t, cambio)); !errors.Is(err, errReglasAvisos) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
}
