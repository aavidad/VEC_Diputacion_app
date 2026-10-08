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

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
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
		`{"catalogo_incidencias":null}`,
		`{"catalogo_incidencias":""}`,
		`{"catalogo_incidencias":"primero","catalogo_incidencias":"segundo"}`,
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

func TestCLICatalogoConfiguradoTreceCodigosYFalloAntesDeEscribir(t *testing.T) {
	dir := t.TempDir()
	catalogo, err := catalogoincidencias.PorIdioma("en")
	if err != nil {
		t.Fatal(err)
	}
	var entrada bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &entrada, Catalogo: catalogo, Entorno: "pruebas"})
	if err != nil {
		t.Fatal(err)
	}
	for _, codigo := range domain.CodigosIncidenciaTecnica() {
		def, _ := domain.DefinicionIncidenciaTecnicaDe(codigo)
		emisor.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: codigo, Componente: def.Componentes[0], Etapa: def.Etapas[0]})
	}
	if err := emisor.Cerrar(t.Context()); err != nil {
		t.Fatal(err)
	}
	valida := bytes.Clone(entrada.Bytes())
	cfg := observabilidad.ConfiguracionRecolector{Directorio: filepath.Join(dir, "registros"), MaxLineaBytes: 1024, MaxArchivoBytes: 16384, MaxArchivos: 4, RetencionSegundos: 3600, VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 1}, CatalogoIncidencias: "../../web/static/textos/en/incidencias_tecnicas.json"}
	rutaConfig := filepath.Join(dir, "config.json")
	b, err := json.Marshal(cfg)
	if err != nil || os.WriteFile(rutaConfig, b, 0600) != nil {
		t.Fatal("configuración no creada")
	}
	args := []string{"--config", rutaConfig, "--textos", "../../web/static/textos/en/registros-tecnicos.json"}
	var salida, diagnostico bytes.Buffer
	if codigo := ejecutar(args, &entrada, &salida, &diagnostico); codigo != 0 {
		t.Fatal("consumidor real falló", codigo)
	}
	var resumen struct {
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}
	if json.Unmarshal(salida.Bytes(), &resumen) != nil || resumen.Metricas.Escritas != 13 || resumen.Metricas.Rechazadas != 0 || len(resumen.Metricas.PorCodigo) != 13 {
		t.Fatal("catálogo completo no recogido")
	}
	guardada, err := os.ReadFile(filepath.Join(cfg.Directorio, "incidencias-activo.jsonl"))
	if err != nil || !bytes.Equal(valida, guardada) {
		t.Fatal("estructura JSONL o mensajes modificados")
	}
	// La selección de un catálogo inválido detiene la CLI antes de crear
	// archivos. El diagnóstico no copia el catálogo ni su ruta.
	cfg.Directorio = filepath.Join(dir, "no_crear")
	cfg.CatalogoIncidencias = filepath.Join(dir, "dato_privado_sintetico.json")
	if os.WriteFile(cfg.CatalogoIncidencias, []byte(`{"esquema":"dato_privado_sintetico"}`), 0600) != nil {
		t.Fatal("catálogo inválido no creado")
	}
	b, _ = json.Marshal(cfg)
	if os.WriteFile(rutaConfig, b, 0600) != nil {
		t.Fatal("configuración inválida no creada")
	}
	salida.Reset()
	diagnostico.Reset()
	if codigo := ejecutar(args, bytes.NewReader(valida), &salida, &diagnostico); codigo != 2 || strings.Contains(diagnostico.String(), "dato_privado_sintetico") || salida.Len() != 0 {
		t.Fatal("fallo abierto o datos libres en CLI")
	}
	if _, err := os.Stat(cfg.Directorio); !os.IsNotExist(err) {
		t.Fatal("almacén creado con catálogo inválido")
	}
}

func TestCLIGrxFirmaConHistoriaJSONLV1IntactaYCatalogoAnteriorRechazado(t *testing.T) {
	dir := t.TempDir()
	// Línea histórica del formato original, previa a la ampliación del
	// catálogo. Nunca llevó un campo con la revisión del catálogo.
	historica := []byte(`{"esquema":"vec.incidencia_tecnica.v1","instante":"2026-10-03T00:00:00.000Z","codigo":"ARRANQUE_FALLIDO","severidad":"critica","componente":"servidor","etapa":"escucha","entorno":"pruebas","version_binario":"936aac665","correlacion":"0123456789abcdef0123456789abcdef","recuento":1,"mensaje":"El servidor no ha podido arrancar."}` + "\n")
	var nueva bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &nueva, Entorno: "pruebas"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{
		Resultado: domain.ResultadoTecnicoNoDisponible, Componente: domain.ComponenteIncidenciaGrxFirma, Etapa: domain.EtapaIncidenciaPeticion,
	})
	if err := emisor.Cerrar(t.Context()); err != nil {
		t.Fatal(err)
	}
	if nueva.Len() == 0 {
		t.Fatal("resultado GrxFirma no emitido")
	}
	mixta := append(bytes.Clone(historica), nueva.Bytes()...)
	cfg := observabilidad.ConfiguracionRecolector{Directorio: filepath.Join(dir, "registros"), MaxLineaBytes: 1024, MaxArchivoBytes: 8192, MaxArchivos: 4, RetencionSegundos: 3600, VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 5}}
	rutaConfig := filepath.Join(dir, "config.json")
	b, _ := json.Marshal(cfg)
	if os.WriteFile(rutaConfig, b, 0600) != nil {
		t.Fatal("configuración no creada")
	}
	args := []string{"--config", rutaConfig, "--textos", "../../web/static/textos/es/registros-tecnicos.json"}
	var salida, diagnostico bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(mixta), &salida, &diagnostico); codigo != 0 {
		t.Fatal("historia mixta rechazada", codigo)
	}
	conservada, err := os.ReadFile(filepath.Join(cfg.Directorio, "incidencias-activo.jsonl"))
	if err != nil || !bytes.Equal(mixta, conservada) {
		t.Fatal("historia antigua o resultado nuevo reescritos")
	}
	// El esquema v1 no admite atribuirle una revisión por un campo nuevo.
	conRevision := bytes.Replace(nueva.Bytes(), []byte(`"esquema":`), []byte(`"version_catalogo":1,"esquema":`), 1)
	salida.Reset()
	diagnostico.Reset()
	if codigo := ejecutar(args, bytes.NewReader(conRevision), &salida, &diagnostico); codigo != 0 {
		t.Fatal("recogida no completada")
	}
	var resumen struct {
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}
	if json.Unmarshal(salida.Bytes(), &resumen) != nil || resumen.Metricas.Rechazadas != 1 || resumen.Metricas.Escritas != 0 {
		t.Fatal("atribución de revisión no conforme aceptada")
	}
	// Un archivo externo de revisión 1 no se convierte a revisión 2.
	b, err = os.ReadFile("../../web/static/textos/es/incidencias_tecnicas.json")
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.Replace(b, []byte(`"version_catalogo": "2"`), []byte(`"version_catalogo": "1"`), 1)
	cfg.CatalogoIncidencias = filepath.Join(dir, "catalogo_anterior.json")
	cfg.Directorio = filepath.Join(dir, "no_crear")
	if os.WriteFile(cfg.CatalogoIncidencias, b, 0600) != nil {
		t.Fatal("catálogo anterior no creado")
	}
	b, _ = json.Marshal(cfg)
	if os.WriteFile(rutaConfig, b, 0600) != nil {
		t.Fatal("configuración anterior no creada")
	}
	salida.Reset()
	diagnostico.Reset()
	if codigo := ejecutar(args, bytes.NewReader(nueva.Bytes()), &salida, &diagnostico); codigo != 2 {
		t.Fatal("catálogo anterior admitido")
	}
	if _, err := os.Stat(cfg.Directorio); !os.IsNotExist(err) {
		t.Fatal("almacén creado con catálogo anterior")
	}
}
