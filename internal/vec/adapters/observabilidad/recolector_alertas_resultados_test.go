package observabilidad

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestAlertasResultadosCompartenVentanaConIncidencias(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	cfg.UmbralesResultado = map[domain.CodigoResultadoTecnico]uint64{
		domain.ResultadoTecnicoDenegado: 2, domain.ResultadoTecnicoNoDisponible: 1,
	}
	inicio := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	c := contadorAlertas{cfg: cfg, inicio: inicio,
		cantidades: make(map[domain.CodigoIncidenciaTecnica]uint64), avisados: make(map[domain.CodigoIncidenciaTecnica]bool)}
	var salida bytes.Buffer
	var metricas MetricasRecolector
	denegado := lineaResultado{Resultado: string(domain.ResultadoTecnicoDenegado), Nivel: "warn"}
	noDisponible := lineaResultado{Resultado: string(domain.ResultadoTecnicoNoDisponible), Nivel: "error"}
	registrar := func(l lineaResultado, ahora time.Time) {
		t.Helper()
		if err := c.registrarResultado(l, ahora, &salida, &metricas); err != nil {
			t.Fatal(err)
		}
	}
	registrar(denegado, inicio)
	if metricas.Alertas != 0 {
		t.Fatal("aviso antes del umbral")
	}
	registrar(noDisponible, inicio)
	registrar(denegado, inicio.Add(59*time.Second))
	registrar(denegado, inicio.Add(59*time.Second))
	registrar(noDisponible, inicio.Add(59*time.Second))
	if metricas.Alertas != 2 {
		t.Fatal("resultados mezclados o aviso repetido", metricas)
	}
	// Una incidencia en el límite de la ventana inicia la siguiente para
	// ambos tipos de registro; el primer denegado vuelve a quedar bajo umbral.
	incidencia, err := validarLineaRecolector(lineaRecolectorPrueba(t))
	if err != nil || c.registrar(incidencia, inicio.Add(time.Minute), &salida, &metricas) != nil {
		t.Fatal("incidencia no aceptada", err)
	}
	registrar(denegado, inicio.Add(time.Minute))
	if metricas.Alertas != 2 {
		t.Fatal("ventana de resultado no reiniciada")
	}
	registrar(denegado, inicio.Add(time.Minute))
	registrar(noDisponible, inicio.Add(time.Minute))
	registrar(lineaResultado{Resultado: string(domain.ResultadoTecnicoCorrecto), Nivel: "info"}, inicio.Add(time.Minute))
	if c.registrar(incidencia, inicio.Add(time.Minute), &salida, &metricas) != nil || metricas.Alertas != 5 {
		t.Fatal("ventana de incidencia o resultado no compartida", metricas)
	}
	decodificador := json.NewDecoder(&salida)
	var avisosResultado int
	for {
		var aviso map[string]any
		if err := decodificador.Decode(&aviso); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if aviso["esquema"] != "vec.alerta_resultado_tecnico.v1" {
			continue
		}
		avisosResultado++
		if len(aviso) != 6 || aviso["instante"] == nil || aviso["ventana_segundos"] != float64(60) {
			t.Fatal("aviso fuera del contrato cerrado", aviso)
		}
		switch aviso["resultado"] {
		case "denegado":
			if aviso["nivel"] != "warn" || aviso["recuento"] != float64(2) {
				t.Fatal("aviso de denegacion alterado", aviso)
			}
		case "no_disponible":
			if aviso["nivel"] != "error" || aviso["recuento"] != float64(1) {
				t.Fatal("aviso de indisponibilidad alterado", aviso)
			}
		default:
			t.Fatal("resultado no admitido en avisos", aviso)
		}
	}
	if avisosResultado != 4 {
		t.Fatal("faltan avisos de resultados", avisosResultado)
	}
}

func TestConfiguracionUmbralesResultadoEsOpcionalYRestringida(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	if !configuracionRecolectorValida(cfg) {
		t.Fatal("configuracion anterior rechazada")
	}
	inicio := time.Now()
	c := contadorAlertas{cfg: cfg, inicio: inicio}
	var metricas MetricasRecolector
	if err := c.registrarResultado(lineaResultado{Resultado: "denegado", Nivel: "warn"}, inicio.Add(time.Hour), new(bytes.Buffer), &metricas); err != nil ||
		c.inicio != inicio || metricas.Alertas != 0 {
		t.Fatal("resultado cambia la ventana con configuracion anterior")
	}
	for _, umbrales := range []map[domain.CodigoResultadoTecnico]uint64{
		{domain.ResultadoTecnicoDenegado: 0},
		{domain.ResultadoTecnicoNoDisponible: 0},
		{domain.ResultadoTecnicoCorrecto: 1},
		{domain.ResultadoTecnicoCancelado: 1},
		{domain.ResultadoTecnicoEntradaInvalida: 1},
		{"dato-personal-sintetico": 1},
	} {
		cfg.UmbralesResultado = umbrales
		if configuracionRecolectorValida(cfg) {
			t.Fatal("configuracion de resultados no admitidos aceptada")
		}
	}
}

func TestFalloSalidaAlertaResultadoNoCuentaEntrega(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	cfg.UmbralesResultado = map[domain.CodigoResultadoTecnico]uint64{domain.ResultadoTecnicoNoDisponible: 1}
	inicio := time.Now()
	c := contadorAlertas{cfg: cfg, inicio: inicio}
	var metricas MetricasRecolector
	err := c.registrarResultado(lineaResultado{Resultado: "no_disponible", Nivel: "error"}, inicio, escritorAlertaResultadoFallido{}, &metricas)
	if !errors.Is(err, io.ErrClosedPipe) || metricas.Alertas != 0 || c.avisadosResultado[1] {
		t.Fatal("fallo de entrega interpretado como aviso", err, metricas)
	}
}

type escritorAlertaResultadoFallido struct{}

func (escritorAlertaResultadoFallido) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestAlertasRechazanEscrituraParcialSinError(t *testing.T) {
	for _, tipo := range []string{"resultado", "incidencia"} {
		t.Run(tipo, func(t *testing.T) {
			cfg := configRecolectorPrueba(t)
			cfg.Umbrales[domain.IncidenciaArranqueFallido] = 1
			cfg.UmbralesResultado = map[domain.CodigoResultadoTecnico]uint64{domain.ResultadoTecnicoNoDisponible: 1}
			inicio := time.Now()
			c := contadorAlertas{cfg: cfg, inicio: inicio,
				cantidades: make(map[domain.CodigoIncidenciaTecnica]uint64), avisados: make(map[domain.CodigoIncidenciaTecnica]bool)}
			var metricas MetricasRecolector
			var err error
			if tipo == "resultado" {
				err = c.registrarResultado(lineaResultado{Resultado: "no_disponible", Nivel: "error"}, inicio, escritorAlertaParcial{}, &metricas)
			} else {
				incidencia, errValidar := validarLineaRecolector(lineaRecolectorPrueba(t))
				if errValidar != nil {
					t.Fatal(errValidar)
				}
				err = c.registrar(incidencia, inicio, escritorAlertaParcial{}, &metricas)
			}
			if !errors.Is(err, io.ErrShortWrite) || metricas.Alertas != 0 ||
				c.avisadosResultado[1] || c.avisados[domain.IncidenciaArranqueFallido] {
				t.Fatal("aviso parcial contado como entregado", err, metricas)
			}
		})
	}
}

type escritorAlertaParcial struct{}

func (escritorAlertaParcial) Write(datos []byte) (int, error) { return len(datos) - 1, nil }
