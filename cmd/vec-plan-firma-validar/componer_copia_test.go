package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	vd "vec-diputacion-granada/internal/vec/domain"
)

func configuracionCopiaPrueba(t *testing.T) (string, string) {
	t.Helper()
	c := composicionPrueba(t)
	claves := []string{"informe_definitivo.p1", "informe_definitivo.p2", "resolucion.p1", "resolucion.p2", "resolucion.p3"}
	var circuito struct {
		Catalogo vd.CatalogoConfigurable `json:"catalogo"`
	}
	bCircuito, err := os.ReadFile(c.CircuitoArchivo)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bCircuito, &circuito); err != nil {
		t.Fatal(err)
	}
	var entradas []vd.EntradaCatalogoConfigurable
	for i, clave := range claves {
		e := c.Catalogo.Entradas[0]
		e.Clave, e.Orden, e.Atributos = clave, i+1, maps.Clone(e.Atributos)
		documento, paso, _ := strings.Cut(clave, ".p")
		e.Atributos["documento"], e.Atributos["paso_orden"] = documento, paso
		e.Atributos["perfil_esperado_ref"] = circuito.Catalogo.Entradas[i].Atributos["perfil_ref"]
		e.Atributos["cargo_ref"] = "cargo:copia:" + strconv.Itoa(i+1)
		entradas = append(entradas, e)
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(entradas)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "entradas.json")
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	cfg := configuracionPlanCopia{Version: 1, CopiaRef: "l-plan-copia", CircuitoArchivo: c.CircuitoArchivo,
		CircuitoSHA: c.CircuitoArchivoSHA256, EntradasArchivo: ruta, EntradasSHA: huella(b),
		FuenteRef: c.Catalogo.FuenteRef, CreadoPor: c.Catalogo.CreadoPor, PerfilCreador: c.PerfilCreador,
		PublicadoPor: c.Publicador, PerfilPublicador: c.PerfilPublicador, AprobacionRef: c.AprobacionRef,
		PreparacionRef: "preparacion:plan:copia:001", ClaveCrear: c.ClaveCrear, ClavePublicar: c.ClavePublicar,
		CorrelacionCrear: c.CorrelacionCrear, CorrelacionPublicar: c.CorrelacionPublicar}
	b, err = json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, b, 0600); err != nil {
		t.Fatal(err)
	}
	return config, filepath.Join(dir, "material")
}

func TestPreparadorCompatibleConConsumidorHYCincoPasos(t *testing.T) {
	config, dir := configuracionCopiaPrueba(t)
	var salida bytes.Buffer
	args := []string{"--config", config, "--salida", dir}
	if codigo := ejecutar(args, &salida); codigo != 0 {
		t.Fatalf("preparar=%d", codigo)
	}
	if !strings.Contains(salida.String(), "preparado_sin_autorizacion") || strings.Contains(salida.String(), "actor:") || strings.Contains(salida.String(), config) {
		t.Fatal("resumen incorrecto")
	}
	for _, op := range []string{"crear", "publicar"} {
		material := filepath.Join(dir, op+"-material.json")
		b, err := os.ReadFile(material)
		if err != nil {
			t.Fatal(err)
		}
		h, err := os.ReadFile(filepath.Join(dir, op+"-material.sha256"))
		if err != nil || huella(b) != strings.TrimSpace(string(h)) {
			t.Fatal("huella incorrecta")
		}
		if ejecutar([]string{material, strings.TrimSpace(string(h))}, &bytes.Buffer{}) != 0 {
			t.Fatal("validador histórico rechazó salida")
		}
		var c vd.CatalogoConfigurable
		catalogo, err := os.ReadFile(filepath.Join(dir, op+"-catalogo.json"))
		if err != nil || json.Unmarshal(catalogo, &c) != nil || len(c.Entradas) != 5 {
			t.Fatal("faltan pasos")
		}
		for _, e := range c.Entradas {
			if e.Atributos["paso_ref"] != "vec.contratacion_temporal.circuito_firma:2:"+e.Clave {
				t.Fatal("paso ajeno al circuito")
			}
		}
		info, err := os.Stat(material)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("salida no privada")
		}
	}
	if ejecutar(args, &bytes.Buffer{}) == 0 {
		t.Fatal("reemplazó carpeta conservada")
	}
}

func TestPreparadorRechazaConfiguracionAmbiguaSinEscribir(t *testing.T) {
	for _, cambio := range []func([]byte) []byte{
		func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1)
		},
		func(b []byte) []byte { return bytes.Replace(b, []byte(`"version":`), []byte(`"Version":`), 1) },
	} {
		config, dir := configuracionCopiaPrueba(t)
		b, err := os.ReadFile(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(config, cambio(b), 0600); err != nil {
			t.Fatal(err)
		}
		var salida bytes.Buffer
		if ejecutar([]string{"--config", config, "--salida", dir}, &salida) == 0 || salida.Len() != 0 {
			t.Fatal("entrada ambigua aceptada")
		}
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatal("creó salida con configuración inválida")
		}
	}
}

func TestPreparadorExigePadrePrivadoAntesDeCrearSalida(t *testing.T) {
	config, dir := configuracionCopiaPrueba(t)
	padre := filepath.Dir(dir)
	if err := os.Chmod(padre, 0777); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(padre, 0700); err != nil {
			t.Error(err)
		}
	})
	var salida bytes.Buffer
	if ejecutar([]string{"--config", config, "--salida", dir}, &salida) == 0 || salida.Len() != 0 {
		t.Fatal("padre compartido aceptado")
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatal("creó salida bajo padre compartido")
	}
}
