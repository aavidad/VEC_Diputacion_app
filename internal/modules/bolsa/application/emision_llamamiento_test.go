package application

import (
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestEmitirLlamamientoRecuperaSoloLaMismaSolicitud(t *testing.T) {
	configuracion := puertosbolsa.ConfiguracionLlamamiento{
		Referencia: "NEC-1", Descripcion: "Cobertura", Categoria: "Auxiliar", Centro: "Centro", Modalidad: "Sustitución",
		FechaInicio: "2026-10-01", Plazo: "48 horas", PlantillaVersion: "bolsa-llamamiento-v1", Asunto: "Aviso", Cuerpo: "Contenido",
	}
	previa := puertosbolsa.EmisionLlamamiento{BolsaRef: "bolsa:1", Participaciones: []string{"participacion:1"}, Configuracion: configuracion}
	misma := puertosbolsa.SolicitudEmitirLlamamiento{BolsaRef: "bolsa:1", Participaciones: []string{"participacion:1"}, Configuracion: configuracion, ClaveIdempotencia: "clave"}
	if !mismaSolicitudEmision(previa, misma) {
		t.Fatal("el replay exacto debe reconocerse como la misma solicitud")
	}

	distinta := misma
	distinta.Configuracion.Centro = "Otro centro"
	if mismaSolicitudEmision(previa, distinta) {
		t.Fatal("una clave reutilizada con otro centro debe ser divergente")
	}
	distinta = misma
	distinta.Participaciones = []string{"participacion:2"}
	if mismaSolicitudEmision(previa, distinta) {
		t.Fatal("una clave reutilizada con otros candidatos debe ser divergente")
	}
}
