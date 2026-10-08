package catalogoincidencias

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/web"
)

func TestCatalogoTrecesCodigosEIdiomasDesdeDatos(t *testing.T) {
	for _, idioma := range []string{"", "es", "en"} {
		c, err := PorIdioma(idioma)
		if err != nil || !c.Valido() {
			t.Fatalf("catálogo no válido: %v", err)
		}
		for _, codigo := range domain.CodigosIncidenciaTecnica() {
			texto, ok := c.Plantilla(codigo)
			if !ok || texto == "" || texto == string(codigo) {
				t.Fatal("clave sin traducción", codigo)
			}
		}
	}
	if _, err := PorIdioma("../../privado"); err == nil {
		t.Fatal("idioma no catalogado admitido")
	}
	if (&Catalogo{}).Valido() || (*Catalogo)(nil).Valido() {
		t.Fatal("catálogo vacío válido")
	}
}

func TestCatalogoRechazaJSONAmbiguoVersionIncompletaYTextoVariable(t *testing.T) {
	b, err := web.TextosIncidenciasTecnicas("")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(map[string]any){
		func(d map[string]any) { d["version_catalogo"] = domain.VersionCatalogoIncidenciasTecnicas + 1 },
		func(d map[string]any) { d["version_catalogo"] = domain.VersionCatalogoIncidenciasTecnicas },
		func(d map[string]any) {
			d["version_catalogo"] = strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas + 1)
		},
		func(d map[string]any) {
			d["version_catalogo"] = "0" + strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas)
		},
		func(d map[string]any) {
			d["version_catalogo"] = "+" + strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas)
		},
		func(d map[string]any) {
			d["version_catalogo"] = strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas) + ".0"
		},
		func(d map[string]any) {
			d["version_catalogo"] = " " + strconv.Itoa(domain.VersionCatalogoIncidenciasTecnicas)
		},
		func(d map[string]any) { d["version_catalogo"] = nil },
		func(d map[string]any) { d["esquema"] = "2" },
		func(d map[string]any) { delete(d, "esquema") },
		func(d map[string]any) { d["campo_ajeno"] = "dato_sintetico_privado" },
		func(d map[string]any) { delete(d["plantillas"].(map[string]any), "ARRANQUE_FALLIDO") },
		func(d map[string]any) { d["plantillas"].(map[string]any)["ARRANQUE_FALLIDO"] = nil },
		func(d map[string]any) { d["plantillas"].(map[string]any)["ARRANQUE_FALLIDO"] = "%s" },
		func(d map[string]any) { d["plantillas"].(map[string]any)["ARRANQUE_FALLIDO"] = "{error}" },
		func(d map[string]any) { d["plantillas"].(map[string]any)["ARRANQUE_FALLIDO"] = "dato\nlibre" },
		func(d map[string]any) { d["plantillas"].(map[string]any)["ARRANQUE_FALLIDO"] = "" },
		func(d map[string]any) { d["plantillas"].(map[string]any)["arranque_fallido"] = "Duplicado" },
	} {
		var d map[string]any
		if json.Unmarshal(b, &d) != nil {
			t.Fatal("fixture no válido")
		}
		mutar(d)
		alterado, _ := json.Marshal(d)
		if c, err := Cargar(bytes.NewReader(alterado)); err != os.ErrInvalid || c != nil {
			t.Fatalf("catálogo inválido admitido: %v", err)
		}
	}
	for _, alterado := range [][]byte{
		append(bytes.Clone(b), []byte(`{}`)...),
		[]byte(`null`),
		[]byte(strings.Repeat(" ", MaxBytes+1)),
		bytes.Replace(b, []byte(`"esquema":`), []byte(`"esquema":"1","esquema":`), 1),
		bytes.Replace(b, []byte(`"ARRANQUE_FALLIDO":`), []byte(`"ARRANQUE_FALLIDO":"Otro","ARRANQUE_FALLIDO":`), 1),
		bytes.Replace(b, []byte(`"ARRANQUE_FALLIDO":`), []byte(`"ARRANQUE_FALLIDO":"\ud800","otro":`), 1),
	} {
		if c, err := Cargar(bytes.NewReader(alterado)); err != os.ErrInvalid || c != nil {
			t.Fatal("JSON ambiguo o excesivo admitido")
		}
	}
}

func TestCatalogoArchivoRegularAcotadoSinEnlaces(t *testing.T) {
	b, err := web.TextosIncidenciasTecnicas("")
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "catalogo.json")
	if os.WriteFile(f, b, 0600) != nil {
		t.Fatal("archivo no creado")
	}
	if c, err := DesdeArchivo(f); err != nil || !c.Valido() {
		t.Fatal("catálogo regular rechazado")
	}
	enlace := f + ".enlace"
	if err := os.Symlink(f, enlace); err != nil {
		t.Fatal(err)
	}
	if c, err := DesdeArchivo(enlace); c != nil || err != os.ErrInvalid {
		t.Fatal("enlace aceptado")
	}
	if os.WriteFile(f, bytes.Repeat([]byte(" "), MaxBytes+1), 0600) != nil {
		t.Fatal("archivo excesivo no creado")
	}
	if c, err := DesdeArchivo(f); c != nil || err != os.ErrInvalid {
		t.Fatal("archivo excesivo aceptado")
	}
}
