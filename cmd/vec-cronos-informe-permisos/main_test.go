package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestCLIPermisosSoloSintetica(t *testing.T) {
	args := []string{"../../web/static/textos/es/cronos-informe-permisos.json", "testdata/ejemplo.json"}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-")) || !bytes.Contains(salida.Bytes(), []byte("/Lang (es-ES)")) {
		t.Fatal(err, salida.Len())
	}
	datos, err := os.ReadFile(args[1])
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ nombre, desde, hasta string }{{"sin_demo", `"demo": true`, `"demo": false`}, {"tipo_no_permitido", "vacaciones_ejemplo", "tipo_ajeno"}, {"campo_ajeno", `"nombre":`, `"solicitud_ref": "dato_ajeno", "nombre":`}} {
		t.Run(c.nombre, func(t *testing.T) {
			ruta := t.TempDir() + "/ejemplo.json"
			if err := os.WriteFile(ruta, bytes.Replace(datos, []byte(c.desde), []byte(c.hasta), 1), 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{args[0], ruta}, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("CLI aceptó fixture ajeno", err)
			}
		})
	}
	salida.Reset()
	args[0] = "../../web/static/textos/en/cronos-informe-permisos.json"
	if err := ejecutar(context.Background(), args, &salida); err != nil || !bytes.Contains(salida.Bytes(), []byte("/Lang (en-GB)")) {
		t.Fatal("catálogo inglés no produjo el idioma pedido", err)
	}
}

func TestCLIPermisosExigeCamposExplicitosYEmiteSubconjunto(t *testing.T) {
	base, err := os.ReadFile("testdata/ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	catalogo := "../../web/static/textos/es/cronos-informe-permisos.json"
	completa := []byte(`"campos_permitidos": ["etiqueta", "unidad", "computo", "pendiente_resolver", "concedido", "restante", "conciliacion"]`)
	for _, caso := range []struct {
		nombre    string
		reemplazo []byte
		valido    bool
	}{
		{"subconjunto", []byte(`"campos_permitidos": ["etiqueta", "unidad", "concedido"]`), true},
		{"sin_campos", []byte(`"campos_permitidos": []`), false},
		{"cantidad_sin_unidad", []byte(`"campos_permitidos": ["etiqueta", "concedido"]`), false},
		{"campo_ajeno", []byte(`"campos_permitidos": ["motivo"]`), false},
		{"campo_duplicado", []byte(`"campos_permitidos": ["unidad", "unidad"]`), false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ruta := t.TempDir() + "/ejemplo.json"
			if err := os.WriteFile(ruta, bytes.Replace(base, completa, caso.reemplazo, 1), 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			err := ejecutar(context.Background(), []string{catalogo, ruta}, &salida)
			if caso.valido && (err != nil || !bytes.HasPrefix(salida.Bytes(), []byte("%PDF-"))) {
				t.Fatal(err)
			}
			if !caso.valido && (err == nil || salida.Len() != 0) {
				t.Fatal("salida con campos inválidos", err)
			}
		})
	}
	// Ausencia del campo también debe fallar; no se adopta todos por defecto.
	ruta := t.TempDir() + "/sin-lista.json"
	sinLista := bytes.Replace(base, append(append([]byte(nil), completa...), ',', '\n'), nil, 1)
	if err := os.WriteFile(ruta, sinLista, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if err := ejecutar(context.Background(), []string{catalogo, ruta}, &salida); err == nil || salida.Len() != 0 {
		t.Fatal("lista implícita admitida", err)
	}
}
func TestCLIPermisosErrorSinDatos(t *testing.T) {
	var salida bytes.Buffer
	if err := informarError(&salida, errors.New("Carmen Molina /ruta/privada")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(salida.String(), "Carmen") || !strings.Contains(salida.String(), ports.ErrExportacionPermisosNoDisponible.Error()) {
		t.Fatal(salida.String())
	}
}

func TestCLICatalogoIdiomaInvalidoSinBytes(t *testing.T) {
	catalogo, err := os.ReadFile("../../web/static/textos/en/cronos-informe-permisos.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"en_GB", "en-gb", "", "en-GB) /OpenAction ("} {
		t.Run(idioma, func(t *testing.T) {
			ruta := t.TempDir() + "/catalogo.json"
			datos := bytes.Replace(catalogo, []byte(`"en-GB"`), []byte(`"`+idioma+`"`), 1)
			if err := os.WriteFile(ruta, datos, 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if err := ejecutar(context.Background(), []string{ruta, "testdata/ejemplo.json"}, &salida); err == nil || salida.Len() != 0 {
				t.Fatal("catálogo inválido produjo bytes", err)
			}
		})
	}
}
