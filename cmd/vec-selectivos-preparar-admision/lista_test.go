package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func argumentosLista(idioma string) []string {
	return append(argumentos(idioma), "--salida", "lista-provisional", "--catalogo-admision-dir", "../../data/catalogos/seleccion")
}

func materialLista(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/lista-material.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestListaProvisionalProduceElMismoContratoEnCadaIdioma(t *testing.T) {
	esperado, err := os.ReadFile("testdata/lista-resultado.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), argumentosLista(idioma), strings.NewReader(materialLista(t)), &salida, &errores); codigo != 0 || errores.Len() != 0 {
			t.Fatalf("%d %s", codigo, errores.String())
		}
		if !bytes.Equal(esperado, salida.Bytes()) {
			t.Fatalf("%s: el resultado difiere del contrato conservado", idioma)
		}
		var lista domain.ListaAdmisionProvisional
		if json.Unmarshal(salida.Bytes(), &lista) != nil || lista.Aprobada || lista.Publicada || lista.Persistida {
			t.Fatal(salida.String())
		}
		catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
		if err != nil {
			t.Fatal(err)
		}
		for _, clave := range lista.Pendientes {
			if _, ok := mensajeCatalogo(catalogo, idioma, clave); !ok {
				t.Fatalf("%s: falta el texto de %s", idioma, clave)
			}
		}
	}
}

func TestListaProvisionalSinSalidaAnteErrores(t *testing.T) {
	raw := materialLista(t)
	casos := map[string]struct {
		args    []string
		entrada string
		clave   string
	}{
		"catalogo_ausente": {append(argumentosLista("es"), "--catalogo-admision", "no-existe.json"), raw, "seleccion.lista_admision.catalogo_no_disponible"},
		"version_ausente":  {argumentosLista("es"), strings.Replace(raw, `"ejemplo-1"`, `"ejemplo-2"`, 1), "seleccion.lista_admision.catalogo_no_disponible"},
		"motivo_ajeno":     {argumentosLista("es"), strings.Replace(raw, `"solicitud_fuera_de_plazo"`, `"inventado"`, 1), "seleccion.lista_admision.entrada_invalida"},
		"clave_duplicada":  {argumentosLista("es"), strings.Replace(raw, `"revision": 1,`, `"revision": 1, "revision": 2,`, 1), "seleccion.lista_admision.entrada_invalida"},
		"json_adicional":   {argumentosLista("en"), raw + "{}", "seleccion.lista_admision.entrada_invalida"},
	}
	for nombre, c := range casos {
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), c.args, strings.NewReader(c.entrada), &salida, &errores); codigo != 1 || salida.Len() != 0 ||
			!strings.Contains(errores.String(), `"error_clave":"`+c.clave+`"`) || !strings.Contains(errores.String(), `"mensaje":`) {
			t.Errorf("%s: %d %q %q", nombre, codigo, salida.String(), errores.String())
		}
	}
}

func TestListaDefinitivaProduceElMismoContratoYTextos(t *testing.T) {
	material, err := os.ReadFile("testdata/lista-definitiva-material.json")
	esperado, err2 := os.ReadFile("testdata/lista-definitiva-resultado.json")
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	for _, idioma := range []string{"es", "en"} {
		var salida, errores bytes.Buffer
		args := append(argumentos(idioma), "--salida", "lista-definitiva", "--catalogo-admision-dir", "../../data/catalogos/seleccion")
		if codigo := ejecutar(context.Background(), args, bytes.NewReader(material), &salida, &errores); codigo != 0 || errores.Len() != 0 {
			t.Fatalf("%d %s", codigo, errores.String())
		}
		if !bytes.Equal(esperado, salida.Bytes()) {
			t.Fatalf("%s: el resultado difiere del contrato conservado", idioma)
		}
		var lista domain.ListaAdmisionDefinitiva
		if json.Unmarshal(salida.Bytes(), &lista) != nil || lista.Aprobada || lista.Publicada || lista.Persistida {
			t.Fatal(salida.String())
		}
		catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
		if err != nil {
			t.Fatal(err)
		}
		for _, clave := range lista.Pendientes {
			if _, ok := mensajeCatalogo(catalogo, idioma, clave); !ok {
				t.Fatalf("%s: falta el texto de %s", idioma, clave)
			}
		}
	}
}

func TestListaDefinitivaSinSalidaAnteErrores(t *testing.T) {
	raw, err := os.ReadFile("testdata/lista-definitiva-material.json")
	if err != nil {
		t.Fatal(err)
	}
	argsDefinitiva := []string{"--catalogos-dir", "../../web/static/textos", "--idioma", "es", "--salida", "lista-definitiva", "--catalogo-admision-dir", "../../data/catalogos/seleccion"}
	casos := map[string]struct{ entrada, clave string }{
		"huella_ajena": {strings.Replace(string(raw), `"a0de31453bd8555d34f2573aae10b1ff6097a4e07dee4e4adff0a4b8443f42f0"`, `"`+strings.Repeat("0", 64)+`"`, 1), "seleccion.lista_definitiva.entrada_invalida"},
		"motivo_nuevo": {strings.Replace(string(raw), `"motivos_persistentes": [
        "documento_identidad_no_aportado"`, `"motivos_persistentes": [
        "titulacion_no_acreditada"`, 1), "seleccion.lista_definitiva.entrada_invalida"},
		"subsanar_fuera_plazo":  {strings.Replace(string(raw), `"via": "reclamacion"`, `"via": "subsanacion"`, 1), "seleccion.lista_definitiva.entrada_invalida"},
		"catalogo_otra_version": {strings.Replace(string(raw), `"ejemplo-1"`, `"ejemplo-2"`, 1), "seleccion.lista_admision.catalogo_no_disponible"},
	}
	for nombre, c := range casos {
		if c.entrada == string(raw) {
			t.Fatalf("%s: el caso no cambia el material", nombre)
		}
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), argsDefinitiva, strings.NewReader(c.entrada), &salida, &errores); codigo != 1 || salida.Len() != 0 ||
			!strings.Contains(errores.String(), `"error_clave":"`+c.clave+`"`) {
			t.Errorf("%s: %d %q %q", nombre, codigo, salida.String(), errores.String())
		}
	}
}

func TestRevisionProvisionalContratoYErrores(t *testing.T) {
	material, err := os.ReadFile("testdata/revision-material.json")
	esperado, err2 := os.ReadFile("testdata/revision-resultado.json")
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	args := func(idioma string) []string {
		return append(argumentos(idioma), "--salida", "revision-provisional", "--catalogo-admision-dir", "../../data/catalogos/seleccion")
	}
	for _, idioma := range []string{"es", "en"} {
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), args(idioma), bytes.NewReader(material), &salida, &errores); codigo != 0 || !bytes.Equal(esperado, salida.Bytes()) {
			t.Fatalf("%s: %d %s", idioma, codigo, errores.String())
		}
		var r domain.RevisionListaProvisional
		catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
		if json.Unmarshal(salida.Bytes(), &r) != nil || err != nil || r.Lista.Aprobada || r.Lista.Publicada {
			t.Fatal(salida.String())
		}
		for _, clave := range r.Pendientes {
			if _, ok := mensajeCatalogo(catalogo, idioma, clave); !ok {
				t.Fatalf("%s: falta %s", idioma, clave)
			}
		}
	}
	casos := map[string]string{
		"misma_revision":    strings.Replace(string(material), `"revision": 2,`, `"revision": 1,`, 1),
		"anterior_alterada": strings.Replace(string(material), `"excluidas_subsanables": 1`, `"excluidas_subsanables": 2`, 1),
		"campo_de_mas":      strings.Replace(string(material), `"anterior": {`, `"anterior": {"nota": "x",`, 1),
		"sin_huella":        strings.Replace(string(material), `"antecedente_anterior"`, `"antecedente_otro"`, 1),
		"huella_ajena":      strings.Replace(string(material), `"a0de31453bd8555d34f2573aae10b1ff6097a4e07dee4e4adff0a4b8443f42f0"`, `"`+strings.Repeat("0", 64)+`"`, 1),
	}
	for nombre, entrada := range casos {
		if entrada == string(material) {
			t.Fatalf("%s: el caso no cambia el material", nombre)
		}
		var salida, errores bytes.Buffer
		if codigo := ejecutar(context.Background(), args("es"), strings.NewReader(entrada), &salida, &errores); codigo != 1 || salida.Len() != 0 ||
			!strings.Contains(errores.String(), `"error_clave":"seleccion.revision_lista.entrada_invalida"`) {
			t.Errorf("%s: %d %q", nombre, codigo, errores.String())
		}
	}
}
