package ports

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventosRelacionConservanAusenciaHistoricaYListaNueva(t *testing.T) {
	antigua := EntradaOrigenLlamamientoCT{OrdenActa: 1}
	raw, err := json.Marshal(antigua)
	if err != nil || strings.Contains(string(raw), "eventos_relacion") {
		t.Fatalf("replay histórico cambió bytes: %s %v", raw, err)
	}
	lista := []string{}
	nueva := EntradaOrigenLlamamientoCT{OrdenActa: 1, EventosRelacion: &lista}
	raw, err = json.Marshal(nueva)
	if err != nil || !strings.Contains(string(raw), `"eventos_relacion":[]`) {
		t.Fatalf("origen nuevo omitió lista vacía: %s %v", raw, err)
	}
}
