package httpinterno

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestProyeccionCuadroRRHHPlazoFaseEsOpcional(t *testing.T) {
	pagina := paginaRRHHPrueba()
	sinPlazo, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sinPlazo), "plazo_fase") {
		t.Fatalf("sin catálogo no debe aparecer el campo: %s", sinPlazo)
	}
	pagina.Plazos = make([]*ports.PlazoFaseRRHH, len(pagina.Expedientes))
	pagina.Plazos[0] = &ports.PlazoFaseRRHH{
		UltimoDia: "2026-09-29", VenceAntesDe: time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC),
		Estado: ports.PlazoFaseVenceHoy, ReglaRef: "vec.contratacion_temporal.reglas:1:c03.plazo_fiscalizacion",
		ReglaEjemplo: true,
	}
	conPlazo, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil {
		t.Fatal(err)
	}
	var salida struct {
		Expedientes []struct {
			PlazoFase map[string]any `json:"plazo_fase"`
		} `json:"expedientes"`
	}
	if err := json.Unmarshal(conPlazo, &salida); err != nil {
		t.Fatal(err)
	}
	plazo := salida.Expedientes[0].PlazoFase
	if len(plazo) != 5 || plazo["ultimo_dia"] != "2026-09-29" ||
		plazo["vence_antes_de"] != "2026-09-29T22:00:00Z" || plazo["estado"] != "vence_hoy" ||
		plazo["regla_ref"] != pagina.Plazos[0].ReglaRef || plazo["regla_ejemplo"] != true {
		t.Fatalf("plazo proyectado inesperado: %s", conPlazo)
	}
	pagina.Plazos[0] = &ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}
	noCalculado, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil || !strings.Contains(string(noCalculado), `"plazo_fase":{"estado":"no_calculado"}`) {
		t.Fatalf("plazo no calculado mal proyectado: %s %v", noCalculado, err)
	}
	// Un plazo mal formado no se publica.
	pagina.Plazos[0] = &ports.PlazoFaseRRHH{UltimoDia: "2026-09-29", VenceAntesDe: time.Now(), ReglaRef: "r"}
	pagina.Plazos[0].Estado = "casi"
	invalido, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil || strings.Contains(string(invalido), "plazo_fase") {
		t.Fatalf("plazo no válido publicado: %s %v", invalido, err)
	}
}

func TestProyeccionCuadroRRHHUrgenciaSoloSiSeDeclaro(t *testing.T) {
	pagina := paginaRRHHPrueba()
	sin, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil || strings.Contains(string(sin), "urgente") {
		t.Fatalf("sin urgencias no aparece el campo: %s %v", sin, err)
	}
	pagina.Urgentes = make([]bool, len(pagina.Expedientes))
	pagina.Urgentes[0] = true
	con, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil || strings.Count(string(con), `"urgente":true`) != 1 {
		t.Fatalf("la urgencia declarada debe viajar una vez: %s %v", con, err)
	}
	pagina.Urgentes = []bool{true, true, true, true, true, true, true, true, true}[:len(pagina.Expedientes)+1]
	desalineada, err := json.Marshal(proyectarPaginaCuadroRRHH(pagina))
	if err != nil || strings.Contains(string(desalineada), "urgente") {
		t.Fatalf("una urgencia desalineada no se publica: %s %v", desalineada, err)
	}
}
