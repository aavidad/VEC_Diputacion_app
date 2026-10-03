package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestCLIEmisorArchivoYAlertasResultados(t *testing.T) {
	directorio := t.TempDir()
	cfg := observabilidad.ConfiguracionRecolector{
		Directorio: filepath.Join(directorio, "registros"), MaxLineaBytes: 1024,
		MaxArchivoBytes: 4096, MaxArchivos: 4, RetencionSegundos: 86400,
		VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 1},
		UmbralesResultado: map[domain.CodigoResultadoTecnico]uint64{
			domain.ResultadoTecnicoDenegado: 2, domain.ResultadoTecnicoNoDisponible: 1,
		},
	}
	carga, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(directorio, "config.json")
	if err := os.WriteFile(ruta, carga, 0600); err != nil {
		t.Fatal(err)
	}
	var entrada bytes.Buffer
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: &entrada, Entorno: "pruebas"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, resultado := range []domain.CodigoResultadoTecnico{
		domain.ResultadoTecnicoDenegado, domain.ResultadoTecnicoNoDisponible,
		domain.ResultadoTecnicoDenegado, domain.ResultadoTecnicoDenegado,
		domain.ResultadoTecnicoNoDisponible, domain.ResultadoTecnicoCorrecto,
	} {
		emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{
			Resultado: resultado, Componente: domain.ComponenteIncidenciaHTTP, Etapa: domain.EtapaIncidenciaPeticion,
		})
	}
	if err := emisor.Cerrar(t.Context()); err != nil {
		t.Fatal(err)
	}
	canonicas := append([]byte(nil), entrada.Bytes()...)
	// Una denegación con campo ajeno no suma para alcanzar el umbral.
	primera, _, _ := bytes.Cut(canonicas, []byte{'\n'})
	invalida := bytes.Replace(primera, []byte(`"resultado":`), []byte(`"secreto":"dato-personal-sintetico","resultado":`), 1)
	// Con una línea inválida y una válida, el umbral de dos no se alcanza.
	cfgBajoUmbral := cfg
	cfgBajoUmbral.Directorio = filepath.Join(directorio, "bajo-umbral")
	var bajoUmbral bytes.Buffer
	bajoUmbral.Write(append(invalida, '\n'))
	bajoUmbral.Write(append(primera, '\n'))
	var sinAviso bytes.Buffer
	metricas, err := observabilidad.RecolectarIncidencias(&bajoUmbral, &sinAviso, cfgBajoUmbral)
	if err != nil || metricas.Alertas != 0 || metricas.Rechazadas != 1 || metricas.Escritas != 1 || sinAviso.Len() != 0 {
		t.Fatal("entrada rechazada cuenta para el umbral", err, metricas)
	}
	entrada.Reset()
	entrada.Write(append(invalida, '\n'))
	entrada.Write(canonicas)
	var salida, avisos bytes.Buffer
	if codigo := ejecutar([]string{"--config", ruta, "--textos", "../../web/static/textos/es/registros-tecnicos.json"}, &entrada, &salida, &avisos); codigo != 0 {
		t.Fatal("CLI no completada", codigo)
	}
	var resumen struct {
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}
	if json.Unmarshal(salida.Bytes(), &resumen) != nil || resumen.Metricas.Escritas != 6 || resumen.Metricas.Rechazadas != 1 ||
		resumen.Metricas.Alertas != 2 || resumen.Metricas.PorResultado[domain.ResultadoTecnicoDenegado] != 3 ||
		resumen.Metricas.PorResultado[domain.ResultadoTecnicoNoDisponible] != 2 {
		t.Fatal("resumen de resultados incorrecto", salida.String())
	}
	guardadas, err := os.ReadFile(filepath.Join(cfg.Directorio, "incidencias-activo.jsonl"))
	if err != nil || !bytes.Equal(guardadas, canonicas) {
		t.Fatal("registros perdidos o entrada ajena guardada", err)
	}
	lineas := bytes.Split(bytes.TrimSpace(avisos.Bytes()), []byte{'\n'})
	if len(lineas) != 2 {
		t.Fatal("avisos repetidos o ausentes")
	}
	for _, linea := range lineas {
		var aviso map[string]any
		if json.Unmarshal(linea, &aviso) != nil || len(aviso) != 6 || aviso["esquema"] != "vec.alerta_resultado_tecnico.v1" {
			t.Fatal("alerta fuera del contrato cerrado")
		}
		for _, clave := range []string{"esquema", "instante", "resultado", "nivel", "recuento", "ventana_segundos"} {
			if _, existe := aviso[clave]; !existe {
				t.Fatal("campos ajenos en alerta")
			}
		}
	}
	if bytes.Contains(append(salida.Bytes(), avisos.Bytes()...), []byte("dato-personal-sintetico")) {
		t.Fatal("datos ajenos en salida")
	}
}

func TestCLIRechazaUmbralesResultadoAmbiguos(t *testing.T) {
	for _, configuracion := range []string{
		`{"umbrales_resultado":{"denegado":1,"denegado":2}}`,
		`{"umbrales_resultado":{"DENEGADO":1}}`,
		`{"umbrales_resultado":{"correcto":1}}`,
		`{"umbrales_resultado":{"no_disponible":1,"secreto":1}}`,
		`{"umbrales_resultado":{},"umbrales_resultado":{}}`,
	} {
		f := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(f, []byte(configuracion), 0600); err != nil {
			t.Fatal(err)
		}
		var cfg observabilidad.ConfiguracionRecolector
		if leerArchivoRecolector(f, &cfg) == nil {
			t.Fatal("configuracion de resultados ambigua aceptada")
		}
	}
}
