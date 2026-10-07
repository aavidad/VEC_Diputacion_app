package observabilidad

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestRecolectorConservaResultadoYRechazaCorrelacionOTextoAjeno(t *testing.T) {
	var entrada bytes.Buffer
	emisor := nuevoEmisor(t, OpcionesEmisor{
		Destino: &entrada, Entorno: "pruebas", VersionBinario: "936aac665",
	})
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	emisor.EmitirResultadoConContexto(ctx, solicitudResultadoPrueba(domain.ResultadoTecnicoCorrecto))
	cerrar(t, emisor)
	valida := append([]byte(nil), entrada.Bytes()...)
	if len(valida) == 0 {
		t.Fatal("resultado técnico no emitido")
	}
	manipulada := bytes.Replace(valida, []byte(`"correlacion_ref":"correlacion_`), []byte(`"correlacion_ref":"correlacion_a`), 1)
	libre := bytes.Replace(valida, []byte(`"resultado":`), []byte(`"mensaje":"dato-personal-sintetico","resultado":`), 1)
	entrada.Reset()
	entrada.Write(manipulada)
	entrada.Write(libre)
	entrada.Write(valida)
	cfg := configRecolectorPrueba(t)
	metricas, err := RecolectarIncidencias(&entrada, new(bytes.Buffer), cfg)
	if err != nil || metricas.Recibidas != 3 || metricas.Rechazadas != 2 ||
		metricas.Escritas != 1 || metricas.PorResultado[domain.ResultadoTecnicoCorrecto] != 1 ||
		len(metricas.PorCodigo) != 0 {
		t.Fatalf("resultado y rechazos: métricas=%+v error=%v", metricas, err)
	}
	guardada, err := os.ReadFile(filepath.Join(cfg.Directorio, archivoActivoRecolector))
	if err != nil || !bytes.Equal(guardada, valida) || strings.Contains(string(guardada), "dato-personal-sintetico") {
		t.Fatal("salida técnica alterada o texto libre conservado")
	}
}
