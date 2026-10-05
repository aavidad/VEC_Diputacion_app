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
