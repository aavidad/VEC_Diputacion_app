package ports

import "testing"

func TestAuditoriaFronteraPersonalB2AceptaSoloSuperficieYRutasNominales(t *testing.T) {
	base := OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_0123456789abcdef0123456789abcdef",
		Motivo:         MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     SuperficieAuditoriaFronteraRutaExactaPersonal,
	}
	for _, ruta := range []string{
		"/api/vec/personal/empleados", "/api/vec/personal/hechos",
		"/api/vec/personal/vacantes", "/api/vec/personal/catalogos-registro-empleado",
		"/api/vec/personal/empleados/{emp_ref}",
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl",
	} {
		orden := base
		orden.Ruta = ruta
		if err := orden.Validar(); err != nil {
			t.Fatalf("ruta %q rechazada: %v", ruta, err)
		}
		orden.Superficie = SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
		if orden.Validar() == nil {
			t.Fatalf("ruta %q aceptada sobre superficie CT", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/personal/empleados/", "/api/vec/personal/empleados/emp_corta",
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl/relaciones",
		"/api/vec/personal/empleados/emp_0123456789abcdefghijkl?dato=privado",
		"/api/vec/personal/catalogos-registro-empleado/otra",
		"/api/vec/personal/otra", "/api/vec/contratacion-temporal/solicitudes",
	} {
		orden := base
		orden.Ruta = ruta
		if orden.Validar() == nil {
			t.Fatalf("ruta %q aceptada", ruta)
		}
	}
}
