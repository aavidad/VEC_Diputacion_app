package postgres

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestDecodificarConfirmacionDirectaConservaReciboOriginal(t *testing.T) {
	fecha := "2026-10-16T00:00:00.123456Z"
	ref := "respuesta-portal:" + strings.Repeat("a", 64)
	b, err := json.Marshal(map[string]any{
		"bolsa_ref": "bolsa:1", "participacion_ref": "participacion:1", "llamamiento_ref": "llamamiento:1",
		"situacion_desde_esperada": fecha, "motivo": "Aceptación confirmada",
		"recibo_ref": "recibo:respuesta-revision:" + strings.Repeat("b", 64),
		"situacion":  "pendiente_incorporacion", "desde": fecha, "fecha_disponible": nil,
		"resuelta_en": fecha, "respuesta_ref": ref,
		"resolucion_ref": "resolucion:respuesta-revision:" + strings.Repeat("c", 64),
		"canon_sha256":   strings.Repeat("d", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := decodificarConfirmacionDirecta(b, true)
	if err != nil || !v.Reutilizada || v.RespuestaRef != ref || v.ResueltaEn == nil ||
		!v.ResueltaEn.Equal(time.Date(2026, 10, 16, 0, 0, 0, 123456000, time.UTC)) ||
		v.Situacion != "pendiente_incorporacion" || v.FechaDisponible != nil {
		t.Fatalf("recibo original incompleto: %v", err)
	}
	comando := ports.ComandoConfirmarAceptacionDirecta{
		ComandoCambiarSituacionParticipacion: ports.ComandoCambiarSituacionParticipacion{
			BolsaRef: "bolsa:1", ReciboRef: v.ReciboRef,
			Cambio: dominiobolsa.CambioSituacionParticipacion{
				ParticipacionRef: "participacion:1", Motivo: "Aceptación confirmada"},
		}, LlamamientoRef: "llamamiento:1", SituacionEsperadaDesde: v.SituacionEsperadaDesde,
		CanonSHA256: v.CanonSHA256,
	}
	if v.ValidarPara(comando) != nil {
		t.Fatal("el recibo válido no pasó la comprobación previa al COMMIT")
	}
}
