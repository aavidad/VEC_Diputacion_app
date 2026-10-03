package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func archivosPrueba(t *testing.T) (string, string, string) {
	t.Helper()
	fecha := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	concesion := domain.ConcesionRol{Accion: "sintetico.leer", ModuloID: "sintetico", TipoRecurso: "expediente", Finalidades: []string{"revision"}, GarantiaMinima: domain.AuthAssuranceHigh, CamposPermitidos: []string{"estado"}, Obligaciones: []string{"auditar"}}
	c := domain.CatalogoAccionesAdministracionV1{Referencia: "catalogo:sintetico", Version: 1, FuenteRef: "fuente:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), VigenteDesde: fecha,
		Entradas: []domain.EntradaAccionAdministracionV1{{Referencia: "entrada:sintetica", Version: 1, FuenteRef: "fuente:modulo", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64), Concesion: concesion, DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada", VigenteDesde: fecha}}, Perfiles: []domain.PerfilPublicadoAdministracionV1{}}
	hc, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	he, err := c.Entradas[0].HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p := domain.PropuestaPerfilAdministracionV1{CatalogoRef: c.Referencia, CatalogoVersion: 1, CatalogoHuellaSHA256: hc,
		RolPropuesto: domain.VersionRol{RolID: "revision_sintetica", Version: 1, Nombre: "revision_sintetica", Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{concesion}, PublicadaPor: "actor:sintetico", PublicadaEn: fecha},
		Selecciones:  []domain.SeleccionAccionAdministracionV1{{EntradaRef: c.Entradas[0].Referencia, EntradaVersion: 1, EntradaHuellaSHA256: he}}}
	dir := t.TempDir()
	rc, rp := filepath.Join(dir, "catalogo.json"), filepath.Join(dir, "propuesta.json")
	for ruta, valor := range map[string]any{rc: c, rp: p} {
		datos, err := json.Marshal(valor)
		if err != nil {
			t.Fatal(err)
		}
		if os.WriteFile(ruta, datos, 0600) != nil {
			t.Fatal("write fixture")
		}
	}
	return rc, rp, fecha.Format(time.RFC3339Nano)
}

func TestCLIComprobacionRealUsaCatalogosESEN(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		t.Run(idioma, func(t *testing.T) {
			c, p, fecha := archivosPrueba(t)
			var salida bytes.Buffer
			textos := filepath.Join("../../web/static/textos", idioma, "admin-comprobar-perfil.json")
			if codigo := ejecutar([]string{textos, c, p, fecha}, &salida); codigo != 0 {
				t.Fatalf("exit=%d salida=%s", codigo, salida.String())
			}
			var resultado salidaComprobador
			if json.Unmarshal(salida.Bytes(), &resultado) != nil {
				t.Fatal("JSON")
			}
			if !resultado.Comprobado || resultado.Publicado || resultado.Dictamen == nil || resultado.Mensaje == resultado.Codigo || resultado.Limite == "" {
				t.Fatalf("resultado=%+v", resultado)
			}
			// Una alteración conserva la selección y debe producir dictamen negativo localizado.
			datos, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			datos = bytes.ReplaceAll(datos, []byte("sintetico.leer"), []byte("sintetico.modificar"))
			if os.WriteFile(p, datos, 0600) != nil {
				t.Fatal("mutation")
			}
			salida.Reset()
			if codigo := ejecutar([]string{textos, c, p, fecha}, &salida); codigo != 1 {
				t.Fatalf("exit=%d salida=%s", codigo, salida.String())
			}
			var rechazo salidaComprobador
			if json.Unmarshal(salida.Bytes(), &rechazo) != nil {
				t.Fatal("JSON")
			}
			if rechazo.Comprobado || rechazo.Publicado || rechazo.Dictamen != nil || rechazo.Codigo != domain.ErrPermisoPerfilAdministracionNoCoincide.Error() {
				t.Fatalf("rechazo=%+v", rechazo)
			}
		})
	}
}

func TestCLIRechazaJSONAmbiguoYNoAcotado(t *testing.T) {
	casos := []string{`{"catalogo_version":1,"catalogo_version":2}`, `{"catalogo_version":1,"Catalogo_version":2}`, `{"catalogo_version":1,"catalogo_\u0076ersion":2}`, `{"desconocido":true}`, `{} {}`, "{\"version_rol_base_ref\":\"\xff\"}", strings.Repeat("[", 18) + "0" + strings.Repeat("]", 18), "[" + strings.Repeat("0,", 512) + "0]"}
	for _, datos := range casos {
		t.Run(datos[:min(30, len(datos))], func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "entrada.json")
			if os.WriteFile(ruta, []byte(datos), 0600) != nil {
				t.Fatal("fixture")
			}
			var destino domain.PropuestaPerfilAdministracionV1
			if leerJSON(ruta, &destino, 65536) == nil {
				t.Fatal("accepted ambiguous input")
			}
		})
	}
}

func TestCLIRechazaEntradasSinEmitirDictamen(t *testing.T) {
	c, p, fecha := archivosPrueba(t)
	textos := filepath.Join("../../web/static/textos/es", "admin-comprobar-perfil.json")
	for nombre, args := range map[string][]string{
		"catalogo ausente": {textos, filepath.Join(t.TempDir(), "ausente.json"), p, fecha},
		"fecha invalida":   {textos, c, p, "fecha_invalida"},
	} {
		t.Run(nombre, func(t *testing.T) {
			var salida bytes.Buffer
			if codigo := ejecutar(args, &salida); codigo != 1 {
				t.Fatalf("exit=%d salida=%s", codigo, salida.String())
			}
			var resultado salidaComprobador
			if json.Unmarshal(salida.Bytes(), &resultado) != nil || resultado.Comprobado || resultado.Publicado ||
				resultado.Dictamen != nil || resultado.Codigo != errEntrada.Error() || resultado.Mensaje == resultado.Codigo {
				t.Fatalf("resultado=%+v", resultado)
			}
		})
	}
	for nombre, args := range map[string][]string{
		"argumentos incompletos": {textos, c, p},
		"textos ausentes":        {filepath.Join(t.TempDir(), "ausente.json"), c, p, fecha},
	} {
		t.Run(nombre, func(t *testing.T) {
			var salida bytes.Buffer
			if codigo := ejecutar(args, &salida); codigo != 2 || salida.Len() != 0 {
				t.Fatalf("exit=%d salida=%s", codigo, salida.String())
			}
		})
	}
}

func TestLecturaJSONRechazaTamanioYDirectorio(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "entrada.json")
	if err := os.WriteFile(ruta, []byte(`{"catalogo_version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta   string
		maximo int64
	}{{ruta, 4}, {dir, 65536}} {
		var destino domain.PropuestaPerfilAdministracionV1
		if leerJSON(caso.ruta, &destino, caso.maximo) == nil {
			t.Fatal("accepted oversized or non-regular input")
		}
	}
}

func TestCLIRechazaIdiomaDeCatalogoSinInicializar(t *testing.T) {
	catalogo, propuesta, fecha := archivosPrueba(t)
	textos := filepath.Join(t.TempDir(), "textos.json")
	if err := os.WriteFile(textos, []byte(`{"idioma":"","mensajes":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{textos, catalogo, propuesta, fecha}, &salida); codigo != 2 || salida.Len() != 0 {
		t.Fatalf("codigo=%d salida=%q", codigo, salida.String())
	}
}
