package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestCLIRecogeIncidenciaRealDelEmisorYNoDatosLibres(t *testing.T) {
	dir := t.TempDir()
	cfg := observabilidad.ConfiguracionRecolector{Directorio: filepath.Join(dir, "registros"), MaxLineaBytes: 1024, MaxArchivoBytes: 4096, MaxArchivos: 4, RetencionSegundos: 86400, VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 1}}
	b, _ := json.Marshal(cfg)
	rutaConfig := filepath.Join(dir, "config.json")
	if err := os.WriteFile(rutaConfig, b, 0600); err != nil {
		t.Fatal(err)
	}
	var entrada bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &entrada, Entorno: "pruebas", VersionBinario: "936aac665"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		t.Fatal("correlacion no disponible")
	}
	emisor.EmitirConContexto(ctx, domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaArranqueFallido, Componente: domain.ComponenteIncidenciaServidor, Etapa: domain.EtapaIncidenciaEscucha})
	if err := emisor.Cerrar(t.Context()); err != nil {
		t.Fatal(err)
	}
	entrada.WriteString("dato-personal-sintetico\n")
	var salida, diagnostico bytes.Buffer
	codigo := ejecutar([]string{"--config", rutaConfig, "--textos", "../../web/static/textos/es/registros-tecnicos.json"}, &entrada, &salida, &diagnostico)
	if codigo != 0 {
		t.Fatal("CLI no completada", codigo, diagnostico.String())
	}
	var resumen struct {
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}
	if err := json.Unmarshal(salida.Bytes(), &resumen); err != nil || resumen.Metricas.Escritas != 1 || resumen.Metricas.Rechazadas != 1 || resumen.Metricas.Alertas != 1 {
		t.Fatal("resumen incorrecto", err, salida.String())
	}
	if strings.Contains(salida.String()+diagnostico.String(), "dato-personal-sintetico") || strings.Contains(diagnostico.String(), rutaConfig) {
		t.Fatal("datos en salida")
	}
	archivo, err := os.ReadFile(filepath.Join(cfg.Directorio, "incidencias-activo.jsonl"))
	if err != nil {
		t.Fatal("registro técnico no conservado")
	}
	var conservada struct {
		Correlacion string `json:"correlacion"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(archivo), &conservada); err != nil || conservada.Correlacion != correlacion {
		t.Fatal("correlación de la petición no conservada")
	}
}

func TestCLIRecogeResultadoTecnicoYConservaReferenciaV3(t *testing.T) {
	directorio := t.TempDir()
	cfg := observabilidad.ConfiguracionRecolector{
		Directorio: filepath.Join(directorio, "registros"), MaxLineaBytes: 1024,
		MaxArchivoBytes: 4096, MaxArchivos: 4, RetencionSegundos: 86400,
		VentanaSegundos: 60,
		Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{
			domain.IncidenciaArranqueFallido: 1,
		},
	}
	carga, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rutaConfig := filepath.Join(directorio, "config.json")
	if err := os.WriteFile(rutaConfig, carga, 0600); err != nil {
		t.Fatal(err)
	}
	var entrada bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{
		Destino: &entrada, Entorno: "pruebas", VersionBinario: "936aac665",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		t.Fatal("correlacion ausente")
	}
	emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{
		Resultado:  domain.ResultadoTecnicoCorrecto,
		Componente: domain.ComponenteIncidenciaPostgreSQL,
		Etapa:      domain.EtapaIncidenciaConsulta,
	})
	if err := emisor.Cerrar(t.Context()); err != nil {
		t.Fatal(err)
	}
	var salida, diagnostico bytes.Buffer
	if codigo := ejecutar([]string{
		"--config", rutaConfig, "--textos", "../../web/static/textos/es/registros-tecnicos.json",
	}, &entrada, &salida, &diagnostico); codigo != 0 {
		t.Fatalf("CLI no recogio resultado: codigo=%d", codigo)
	}
	var resumen struct {
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}
	if err := json.Unmarshal(salida.Bytes(), &resumen); err != nil ||
		resumen.Metricas.Escritas != 1 || resumen.Metricas.PorResultado[domain.ResultadoTecnicoCorrecto] != 1 ||
		len(resumen.Metricas.PorCodigo) != 0 {
		t.Fatal("resumen no distingue resultado de incidencia")
	}
	guardada, err := os.ReadFile(filepath.Join(cfg.Directorio, "incidencias-activo.jsonl"))
	if err != nil {
		t.Fatal("resultado no conservado")
	}
	var registro struct {
		Esquema        string `json:"esquema"`
		CorrelacionRef string `json:"correlacion_ref"`
	}
	if json.Unmarshal(bytes.TrimSpace(guardada), &registro) != nil ||
		registro.Esquema != domain.EsquemaResultadoTecnico ||
		registro.CorrelacionRef != "correlacion_"+correlacion {
		t.Fatal("resultado tecnico perdio referencia de la peticion")
	}
}

func TestCLIUsaCatalogoYNoCopiaErrores(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		var salida, diagnostico bytes.Buffer
		textos := "../../web/static/textos/" + idioma + "/registros-tecnicos.json"
		if ejecutar([]string{"--textos", textos, "--ayuda"}, strings.NewReader(""), &salida, &diagnostico) != 0 || salida.Len() == 0 {
			t.Fatal("ayuda sin catalogo")
		}
		salida.Reset()
		if ejecutar([]string{"--textos", textos, "--config", "/ruta-sintetica-no-existente/clave"}, strings.NewReader(""), &salida, &diagnostico) != 2 || strings.Contains(diagnostico.String(), "ruta-sintetica") {
			t.Fatal("error libre en diagnostico")
		}
	}
	// La configuración no puede reabrir permisos del directorio existente.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err := observabilidad.RecolectarIncidencias(strings.NewReader(""), new(bytes.Buffer), observabilidad.ConfiguracionRecolector{Directorio: dir, MaxLineaBytes: 1024, MaxArchivoBytes: 4096, MaxArchivos: 4, RetencionSegundos: int64(time.Hour / time.Second), VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 1}})
	if err == nil {
		t.Fatal("directorio no privado aceptado")
	}
}

func TestCLIRechazaConfiguracionAmbigua(t *testing.T) {
	for _, configuracion := range []string{
		`{"directorio":"primero","directorio":"segundo"}`,
		`{"DIRECTORIO":"segundo"}`,
		`{"umbrales_alerta":{"ARRANQUE_FALLIDO":1,"ARRANQUE_FALLIDO":2}}`,
	} {
		f := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(f, []byte(configuracion), 0600); err != nil {
			t.Fatal(err)
		}
		var cfg observabilidad.ConfiguracionRecolector
		if leerArchivoRecolector(f, &cfg) == nil {
			t.Fatal("configuracion ambigua aceptada")
		}
	}
}
