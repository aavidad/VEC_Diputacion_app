package postgrespublico

import (
	"testing"
	"time"

	httpbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/httpapi"
)

func TestBolsaB10ValidaSoloLaProyeccionMinimizada(t *testing.T) {
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	bolsa := httpbolsa.BolsaPublica{
		BolsaRef: "bolsa:administrativo:2026-09-18", Categoria: "Administrativo",
		Grupos: []string{"C1"}, TipoLista: "definitiva", VigenteDesde: ahora,
	}
	if !bolsaB10Valida(bolsa) {
		t.Fatal("bolsa publica valida rechazada")
	}
	bolsa.BolsaRef = "bolsa con espacios"
	if bolsaB10Valida(bolsa) {
		t.Fatal("referencia no publica aceptada")
	}
}

func TestEstadoB10SoloAdmiteCatalogoPublicado(t *testing.T) {
	for _, estado := range []string{"disponible", "ocupado", "no_disponible", "excluido", "renuncia_pendiente"} {
		if !estadoB10Valido(estado) {
			t.Fatalf("estado valido rechazado: %s", estado)
		}
	}
	if estadoB10Valido("nombre_completo") || estadoB10Valido("Disponible") {
		t.Fatal("estado no publicado aceptado")
	}
}
