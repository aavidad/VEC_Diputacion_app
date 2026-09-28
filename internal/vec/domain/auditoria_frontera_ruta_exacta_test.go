package domain

import "testing"

func TestAuditoriaFronteraAuditoriaAceptaSoloSuperficieYRutasNominales(t *testing.T) {
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaAuditoria,
	}
	for _, ruta := range []string{
		"/api/vec/auditoria/opciones",
		"/api/vec/auditoria/consultas",
	} {
		orden := base
		orden.Ruta = ruta
		if err := orden.Validar(); err != nil {
			t.Fatalf("ruta nominal %q rechazada: %v", ruta, err)
		}
		orden.Motivo = MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
		if err := orden.Validar(); err != nil {
			t.Fatalf("denegacion 401 de %q rechazada: %v", ruta, err)
		}
		orden.Superficie = SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
		if orden.Validar() == nil {
			t.Fatalf("ruta %q aceptada sobre superficie CT", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/auditoria", "/api/vec/auditoria/",
		"/api/vec/auditoria/consultas/otra",
		"/api/vec/auditoria/consultas?dato=privado",
		"/api/vec/auditoria/opciones%2Fotra",
		"/api/vec/contratacion-temporal/solicitudes",
		"/api/vec/personal/empleados",
	} {
		orden := base
		orden.Ruta = ruta
		if orden.Validar() == nil {
			t.Fatalf("ruta ajena o ampliada %q aceptada", ruta)
		}
	}
}
