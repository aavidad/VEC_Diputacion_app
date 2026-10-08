package observabilidad

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestResultadosGrxFirmaConCorrelacionYRecolectorSinAlias(t *testing.T) {
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		t.Fatal("correlación ausente")
	}
	destino := &destinoSeguro{}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Entorno: "pruebas"})
	for _, resultado := range []domain.CodigoResultadoTecnico{domain.ResultadoTecnicoCorrecto, domain.ResultadoTecnicoNoDisponible} {
		e.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{
			Resultado: resultado, Componente: domain.ComponenteIncidenciaGrxFirma, Etapa: domain.EtapaIncidenciaPeticion,
		})
	}
	cerrar(t, e)
	lineas := destino.lineas(t)
	if len(lineas) != 2 {
		t.Fatal("resultados no emitidos")
	}
	for _, linea := range lineas {
		if linea["esquema"] != domain.EsquemaResultadoTecnico || linea["componente"] != string(domain.ComponenteIncidenciaGrxFirma) ||
			linea["correlacion"] != correlacion || linea["correlacion_ref"] != "correlacion_"+correlacion {
			t.Fatal("proyección o correlación modificadas")
		}
		if _, presente := linea["mensaje"]; presente {
			t.Fatal("texto añadido al resultado")
		}
	}
	var entrada bytes.Buffer
	for _, alias := range []string{"GrxFirma", "validador_firma", "remoto"} {
		lineas[0]["componente"] = alias
		b, err := json.Marshal(lineas[0])
		if err != nil {
			t.Fatal(err)
		}
		entrada.Write(b)
		entrada.WriteByte('\n')
	}
	entrada.WriteString(destino.texto())
	cfg := configRecolectorPrueba(t)
	m, err := RecolectarIncidencias(&entrada, new(bytes.Buffer), cfg)
	if err != nil || m.Escritas != 2 || m.Rechazadas != 3 || m.PorResultado[domain.ResultadoTecnicoCorrecto] != 1 || m.PorResultado[domain.ResultadoTecnicoNoDisponible] != 1 {
		t.Fatalf("resultado o alias recogidos incorrectamente: %+v %v", m, err)
	}
	b, err := os.ReadFile(filepath.Join(cfg.Directorio, archivoActivoRecolector))
	if err != nil || bytes.Contains(b, []byte(`"remoto"`)) || bytes.Contains(b, []byte(`"GrxFirma"`)) {
		t.Fatal("alias no conforme conservado")
	}
}
