package domain

import (
	"strings"
	"testing"
)

func TestAuditoriaFronteraBolsaSoloRutasYActorCandidatoVerificado(t *testing.T) {
	rutas := []string{
		"/api/vec/bolsa/mi-bolsa",
		"/api/vec/bolsa/mi-bolsa/historial",
		"/api/vec/bolsa/mi-bolsa/solicitudes",
		"/api/vec/bolsa/mi-bolsa/solicitudes-documentales",
		"/api/vec/bolsa/mi-bolsa/respuestas",
		"/api/vec/bolsa/mi-bolsa/disposiciones",
		"/api/vec/bolsa/mi-bolsa/contacto",
	}
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaBolsaCandidato,
	}
	for _, ruta := range rutas {
		orden := base
		orden.Ruta = ruta
		if err := orden.Validar(); err != nil {
			t.Fatalf("401 de %q: %v", ruta, err)
		}
		orden.Motivo = MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		orden.ActorRef = "per_" + strings.Repeat("a", 22)
		if err := orden.Validar(); err != nil {
			t.Fatalf("403 de %q: %v", ruta, err)
		}
		orden.Superficie = SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
		if orden.Validar() == nil {
			t.Fatalf("ruta Bolsa %q aceptada como CT", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/bolsa/mi-bolsa/", "/api/vec/bolsa/mi-bolsa/otra",
		"/api/vec/bolsa/mi-bolsa/historial/otra", "/api/vec/bolsa/mi-bolsa?persona=privada",
		"/api/vec/bolsa/mi-bolsa/solicitudes-documentales/otra",
		"/api/vec/bolsa/mi-bolsa%2Fhistorial", "/api/vec/bolsa/convocatorias",
	} {
		orden := base
		orden.Ruta = ruta
		if orden.Validar() == nil {
			t.Fatalf("ruta no nominal aceptada: %q", ruta)
		}
	}
	for _, actor := range []string{"per_" + strings.Repeat("a", 22), "", "can_" + strings.Repeat("a", 22)} {
		orden := base
		orden.Ruta = rutas[0]
		orden.ActorRef = actor
		if actor != "" && orden.Validar() == nil {
			t.Fatalf("401 con actor %q aceptado", actor)
		}
		orden.Motivo = MotivoAuditoriaFronteraRutaExactaAccesoDenegado
		if actor != "per_"+strings.Repeat("a", 22) && orden.Validar() == nil {
			t.Fatalf("403 con actor no canonico %q aceptado", actor)
		}
	}
}
