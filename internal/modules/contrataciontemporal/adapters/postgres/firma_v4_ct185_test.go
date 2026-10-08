package postgres

import (
	"os"
	"strings"
	"testing"
)

// El ejecutor CT sólo llama a la v4 del plan (CT185), que calcula las dos
// huellas con los ámbitos de la asignación; la v3 del plan (CT181) se le
// retira. La v3 directa de CT172 queda para el propietario (la llama la v4).
func TestRegistroFirmaUsaLaV4DeCT185(t *testing.T) {
	leer := func(f string) string {
		b, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/" + f)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	ct181, ct185 := leer("000181_firma_interior_ambitos_asignacion.up.sql"), leer("000185_firma_plan_exterior_ambitos_asignacion.up.sql")
	plan := "vec_contratacion_temporal.registrar_firma_con_plan_v4("
	directa := "vec_contratacion_temporal.registrar_firma_verificada_v3("
	if !strings.Contains(registrarFirmaConPlanSQL176, "SELECT "+plan) || !strings.Contains(ct185, "GRANT EXECUTE ON FUNCTION "+plan) ||
		!strings.Contains(ct185, "REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.registrar_firma_con_plan_v3(") {
		t.Fatalf("el adaptador no llama a la v4 del plan que concede CT185: %s", registrarFirmaConPlanSQL176)
	}
	if !strings.Contains(registrarFirmaSQL172, "SELECT "+directa) || strings.Contains(ct181, "GRANT EXECUTE ON FUNCTION "+directa) {
		t.Fatal("la v3 directa de CT172 no debe concederse al ejecutor")
	}
}
