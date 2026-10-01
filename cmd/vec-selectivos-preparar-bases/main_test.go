package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func ejemplo(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + nombre + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPreparacionCompletaConservaLimitesYLocaliza(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := ejecutar(context.Background(), []string{"--catalogos-dir", "../../web/static/textos", "--idioma", idioma}, bytes.NewReader(ejemplo(t, "material-completo")), &out, &errOut)
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("code=%d error=%s", code, errOut.String())
			}
			var r struct {
				Preparacion ports.PreparacionBases `json:"preparacion"`
				Limite      string                 `json:"limite"`
				Mensajes    []pendienteVisible     `json:"mensajes"`
			}
			if json.Unmarshal(out.Bytes(), &r) != nil {
				t.Fatal("JSON")
			}
			if r.Preparacion.Estado != "pendiente" || r.Preparacion.ContenidoCanonicoBolsa == nil || len(r.Mensajes) != 14 {
				t.Fatalf("preparacion=%+v", r.Preparacion)
			}
			for _, p := range r.Preparacion.Pendientes {
				if p.Codigo != "referencia_no_verificada" && p.Codigo != "circuito_pendiente" {
					t.Fatalf("pendiente=%+v", p)
				}
			}
			catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
			if err != nil || r.Limite != catalogo.T(idioma, "limite") {
				t.Fatal("catalogo")
			}
			if r.Preparacion.MaterialPropuesto.VersionMaterial != 1 || r.Preparacion.ContenidoCanonicoBolsa.Categorias[0] != "administrativo" {
				t.Fatal("material")
			}
			for _, prohibido := range []string{"firma_validada_ref", "recibo_custodia_ref", "estado_gobierno", "aprobacion_publicacion", "huella_version_sha256"} {
				if bytes.Contains(out.Bytes(), []byte(`"`+prohibido+`"`)) {
					t.Fatal(prohibido)
				}
			}
		})
	}
}

func TestMaterialIncompletoDevuelveFaltantesSinRellenarlos(t *testing.T) {
	var material ports.MaterialBasesPropuesto
	if json.Unmarshal(ejemplo(t, "material-incompleto"), &material) != nil {
		t.Fatal("fixture")
	}
	r, err := application.PrepararMaterialBases(context.Background(), material, bolsa.CanonizadorBases{})
	if err != nil || r.ContenidoCanonicoBolsa != nil || len(r.MaterialPropuesto.Contenido.Documentos) != 0 || len(r.MaterialPropuesto.Contenido.Plazos) != 0 {
		t.Fatalf("resultado=%+v error=%v", r, err)
	}
	pendientes := map[string]string{}
	for _, p := range r.Pendientes {
		pendientes[p.Campo] = p.Codigo
	}
	for _, campo := range []string{"fuente_bases", "plaza", "oep", "rpt"} {
		if pendientes[campo] != "referencia_ausente" {
			t.Fatal(campo, pendientes)
		}
	}
	if pendientes["documentos_propuestos"] != "material_ausente" || pendientes["plazos"] != "material_ausente" || pendientes["contenido"] != "contenido_no_validado" {
		t.Fatal(pendientes)
	}
	material.Contenido.Categorias[0] = "cambiada"
	if r.MaterialPropuesto.Contenido.Categorias[0] == "cambiada" {
		t.Fatal("material compartido")
	}
}

func TestEntradaAdversariaNoEmitePreparacion(t *testing.T) {
	base := string(ejemplo(t, "material-completo"))
	for nombre, entrada := range map[string]string{
		"duplicada":    strings.Replace(base, `"version_material": 1`, `"version_material": 1, "version_material": 2`, 1),
		"alias":        strings.Replace(base, `"version_material": 1`, `"version_material": 1, "Version_material": 2`, 1),
		"desconocida":  strings.Replace(base, `"version_material": 1`, `"version_material": 1, "aprobada": true`, 1),
		"no sintetica": strings.Replace(base, "preparacion_sintetica", "produccion", 1),
		"extra":        base + `{}`,
		"excesiva":     strings.Repeat(" ", maximoEntradaJSON+1),
		"profundidad":  strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34),
	} {
		t.Run(nombre, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if ejecutar(context.Background(), []string{"--catalogos-dir", "../../web/static/textos"}, strings.NewReader(entrada), &out, &errOut) == 0 || out.Len() != 0 || errOut.Len() == 0 {
				t.Fatalf("out=%s err=%s", out.String(), errOut.String())
			}
		})
	}
}

func TestCanonizadorRechazaURLExternaYCopiasNoCompartenDatos(t *testing.T) {
	var m ports.MaterialBasesPropuesto
	if json.Unmarshal(ejemplo(t, "material-completo"), &m) != nil {
		t.Fatal("fixture")
	}
	r, err := application.PrepararMaterialBases(context.Background(), m, bolsa.CanonizadorBases{})
	if err != nil || r.ContenidoCanonicoBolsa == nil {
		t.Fatal(err)
	}
	r.ContenidoCanonicoBolsa.Categorias[0] = "otra"
	if m.Contenido.Categorias[0] == "otra" || r.MaterialPropuesto.Contenido.Categorias[0] == "otra" {
		t.Fatal("canon compartido")
	}
	m.Contenido.Documentos[0].URL = "https://example.invalid/bases.pdf"
	r, err = application.PrepararMaterialBases(context.Background(), m, bolsa.CanonizadorBases{})
	if err != nil || r.ContenidoCanonicoBolsa != nil || r.MaterialPropuesto.Contenido.Documentos[0].URL != m.Contenido.Documentos[0].URL {
		t.Fatal("URL propuesta no conservada o falsamente validada")
	}
}

func TestCatalogoFueraDeRaizYContextoCanceladoNoEmitenDatos(t *testing.T) {
	for _, args := range [][]string{
		{"--catalogos-dir", "../../web/static/textos", "--idioma", "../es"},
		{"--catalogos-dir", "testdata"},
	} {
		var out, errOut bytes.Buffer
		if ejecutar(context.Background(), args, bytes.NewReader(ejemplo(t, "material-completo")), &out, &errOut) == 0 || out.Len() != 0 {
			t.Fatal("catalogo")
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	var out, errOut bytes.Buffer
	if ejecutar(ctx, []string{"--catalogos-dir", "../../web/static/textos"}, bytes.NewReader(ejemplo(t, "material-completo")), &out, &errOut) == 0 || out.Len() != 0 {
		t.Fatal("cancelacion")
	}
}
