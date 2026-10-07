package domain

import "testing"

func TestOrganizacionHistoricaOrdenAuditoriaSoloRutaNominal(t *testing.T) {
	o := OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: "corr_no_disponible", Motivo: MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida, Superficie: SuperficieAuditoriaFronteraRutaExactaOrganizacionHistoricaPersonal, Ruta: "/api/vec/personal/organizacion-historica"}
	if err := o.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{o.Ruta + "/", o.Ruta + "?vigente_en=2026-01-01", o.Ruta + "/importaciones/publicar", "/api/vec/personal/vacantes"} {
		x := o
		x.Ruta = ruta
		if x.Validar() == nil {
			t.Fatalf("ruta ampliada aceptada %s", ruta)
		}
	}
	o.Superficie = SuperficieAuditoriaFronteraRutaExactaPersonal
	if o.Validar() == nil {
		t.Fatal("consulta histórica no debe heredar superficie B2")
	}
}
