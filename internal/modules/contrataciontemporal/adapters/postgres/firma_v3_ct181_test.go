package postgres

import (
	"os"
	"strings"
	"testing"
)

// Desde CT181 el ejecutor CT sólo llama a CT176 v3; la v3 directa de CT172
// queda para el propietario (la llama CT176 v3). El adaptador llama a las
// funciones que crea CT181 y sólo la del plan se concede al ejecutor.
func TestRegistroFirmaUsaLasV3DeCT181(t *testing.T) {
	b, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000181_firma_interior_ambitos_asignacion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ct181 := string(b)
	plan := "vec_contratacion_temporal.registrar_firma_con_plan_v3("
	directa := "vec_contratacion_temporal.registrar_firma_verificada_v3("
	if !strings.Contains(registrarFirmaConPlanSQL176, "SELECT "+plan) || !strings.Contains(ct181, "GRANT EXECUTE ON FUNCTION "+plan) {
		t.Fatalf("el adaptador no llama a la v3 del plan que concede CT181: %s", registrarFirmaConPlanSQL176)
	}
	if !strings.Contains(registrarFirmaSQL172, "SELECT "+directa) || strings.Contains(ct181, "GRANT EXECUTE ON FUNCTION "+directa) {
		t.Fatal("la v3 directa de CT172 no debe concederse al ejecutor")
	}
}
