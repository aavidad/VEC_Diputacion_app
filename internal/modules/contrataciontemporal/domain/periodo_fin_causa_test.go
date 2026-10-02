package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPeriodoPrevistoFinPorCausaConservaAlternativaExclusiva(t *testing.T) {
	inicio := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	fin := inicio.AddDate(0, 1, 0)
	legado := PeriodoPrevisto{Inicio: inicio, Fin: fin}
	if legado.Validar() != nil {
		t.Fatal("periodo fechado legado inválido")
	}
	contenido, err := json.Marshal(legado)
	if err != nil || !strings.Contains(string(contenido), `"fin":"2026-11-02T00:00:00Z"`) || strings.Contains(string(contenido), "causa_fin") || strings.Contains(string(contenido), "politica_fin") {
		t.Fatalf("canon legado cambiado: %s, %v", contenido, err)
	}
	abierto := PeriodoPrevisto{Inicio: inicio, CausaFin: "reincorporacion_titular", PoliticaFin: PoliticaFin{
		ReglaRef: "catalogo:3:c12.modalidad.sustitucion", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), FechaFin: "no_aplica", CausaFin: "reincorporacion_titular",
	}}
	if abierto.Validar() != nil {
		t.Fatal("fin por causa inválido")
	}
	contenido, err = json.Marshal(abierto)
	if err != nil || strings.Contains(string(contenido), `"fin"`) || !strings.Contains(string(contenido), `"causa_fin":"reincorporacion_titular"`) || !strings.Contains(string(contenido), `"politica_fin"`) {
		t.Fatalf("fin ficticio o snapshot ausente: %s, %v", contenido, err)
	}
	abierto.Fin = fin
	if abierto.Validar() == nil {
		t.Fatal("fecha y causa aceptadas juntas")
	}
	abierto.Fin = time.Time{}
	abierto.PoliticaFin.CausaFin = "cobertura_reglamentaria"
	if abierto.Validar() == nil {
		t.Fatal("causa distinta de política aceptada")
	}
}
