package observabilidad

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func solicitudResultadoPrueba(codigo domain.CodigoResultadoTecnico) domain.SolicitudResultadoTecnico {
	return domain.SolicitudResultadoTecnico{
		Resultado: codigo, Componente: domain.ComponenteIncidenciaPostgreSQL,
		Etapa: domain.EtapaIncidenciaConsulta,
	}
}

func TestResultadosTecnicosCompartenColaYCorrelacionConIncidencias(t *testing.T) {
	destino := &destinoSeguro{}
	instante := time.Date(2026, 10, 3, 12, 0, 0, 123456789, time.UTC)
	e := nuevoEmisor(t, OpcionesEmisor{
		Destino: destino, Entorno: "pruebas", VersionBinario: "936aac665",
		Reloj: func() time.Time { return instante },
	})
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		t.Fatal("correlacion ausente")
	}
	ctx, cancelar := context.WithCancel(ctx)
	cancelar()
	e.EmitirConContexto(ctx, solicitudValida())
	e.EmitirResultadoConContexto(ctx, solicitudResultadoPrueba(domain.ResultadoTecnicoCorrecto))
	e.EmitirResultadoConContexto(ctx, solicitudResultadoPrueba(domain.ResultadoTecnicoDenegado))
	cerrar(t, e)
	lineas := destino.lineas(t)
	if len(lineas) != 3 || lineas[0]["esquema"] != domain.EsquemaIncidenciaTecnica ||
		lineas[1]["esquema"] != domain.EsquemaResultadoTecnico || lineas[2]["esquema"] != domain.EsquemaResultadoTecnico {
		t.Fatalf("familias técnicas mezcladas: %d líneas", len(lineas))
	}
	for _, linea := range lineas {
		if linea["correlacion"] != correlacion {
			t.Fatal("correlacion cambiada por el trabajador")
		}
	}
	for indice, nivel := range []string{"info", "warn"} {
		linea := lineas[indice+1]
		if linea["correlacion_ref"] != "correlacion_"+correlacion || linea["nivel"] != nivel ||
			linea["mensaje"] != nil || linea["codigo"] != nil {
			t.Fatal("resultado incorporó incidencia o texto libre")
		}
		claves := make([]string, 0, len(linea))
		for clave := range linea {
			claves = append(claves, clave)
		}
		sort.Strings(claves)
		if strings.Join(claves, ",") != "componente,correlacion,correlacion_ref,entorno,esquema,etapa,instante,nivel,resultado,version_binario" {
			t.Fatal("campos técnicos fuera de lista cerrada")
		}
	}
	if m := e.MetricasEmision(); m.Escritas != 1 {
		t.Fatalf("resultado contado como incidencia: %+v", m)
	}
	if m := e.MetricasResultadosTecnicos(); m.Aceptados != 2 || m.Escritos != 2 || m.SinCorrelacion != 0 {
		t.Fatalf("métricas de resultados: %+v", m)
	}
}

type destinoCortoResultado struct{}

func (destinoCortoResultado) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestResultadosTecnicosCuentanPerdidaSinCorrelacionYEscrituraCorta(t *testing.T) {
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destinoCortoResultado{}})
	e.EmitirResultadoConContexto(context.Background(), solicitudResultadoPrueba(domain.ResultadoTecnicoCorrecto))
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e.EmitirResultadoConContexto(ctx, solicitudResultadoPrueba(domain.ResultadoTecnicoCorrecto))
	e.Emitir(solicitudValida())
	cerrar(t, e)
	if m := e.MetricasResultadosTecnicos(); m.SinCorrelacion != 1 || m.Aceptados != 1 ||
		m.Escritos != 0 || m.FallosEscritura != 1 {
		t.Fatalf("pérdida de resultado no contada: %+v", m)
	}
	if m := e.MetricasEmision(); m.Escritas != 0 || m.FallosEscritura != 1 {
		t.Fatalf("escritura corta de incidencia contada como éxito: %+v", m)
	}
}
