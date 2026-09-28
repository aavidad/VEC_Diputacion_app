package postgres

import (
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestDecodificarHistorialMiBolsaRechazaCamposInternos(t *testing.T) {
	consulta := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	base := `{"consultada_en":"2026-09-27T12:00:00.000000Z","pagina":1,"tamano":20,"hay_mas":false,"items":[{"clase":"contrato_bolsa","bolsa":"bolsa:auxiliar","categoria":"Auxiliar","ocurrido_en":"2026-09-27T11:00:00.000000Z","tipo":"incorporacion","inicio":"2026-09-29T12:00:00.000000Z","fin_previsto":null,"modalidad_clave":null,"procedencia":"evento_ct_recibido"}]}`
	p, err := decodificarHistorialMiBolsa([]byte(base), consulta, 1)
	if err != nil || len(p.Items) != 1 || p.Items[0].Inicio == nil || !p.Items[0].Inicio.After(consulta) {
		t.Fatalf("contrato futuro informativo: %+v, %v", p, err)
	}
	malicioso := `{"consultada_en":"2026-09-27T12:00:00.000000Z","pagina":1,"tamano":20,"hay_mas":false,"items":[{"clase":"renuncia","bolsa":"bolsa:auxiliar","categoria":"Auxiliar","ocurrido_en":"2026-09-27T11:00:00.000000Z","respuesta":"renuncia","modo":"firme","estado":"respuesta_registrada","justificante_ref":"privado"}]}`
	if _, err := decodificarHistorialMiBolsa([]byte(malicioso), consulta, 1); !errors.Is(err, puertosbolsa.ErrResultadoHistorialMiBolsaInvalido) {
		t.Fatalf("no se rechazó campo personal inesperado: %v", err)
	}
}
