package postgres

import (
	"os"
	"strings"
	"testing"
)

// Desde CT181 el ejecutor CT sólo puede llamar a las v3, que calculan la
// huella interior con los ámbitos de la asignación de quien firma (AD206).
// El adaptador debe llamar exactamente a las funciones que crea CT181.
func TestRegistroFirmaUsaLasV3DeCT181(t *testing.T) {
	b, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000181_firma_interior_ambitos_asignacion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ct181 := string(b)
	for sql, funcion := range map[string]string{
		registrarFirmaSQL172:        "vec_contratacion_temporal.registrar_firma_verificada_v3(",
		registrarFirmaConPlanSQL176: "vec_contratacion_temporal.registrar_firma_con_plan_v3(",
	} {
		if !strings.Contains(sql, "SELECT "+funcion) || !strings.Contains(ct181, "GRANT EXECUTE ON FUNCTION "+funcion) {
			t.Fatalf("el adaptador no llama a la v3 que concede CT181: %s", sql)
		}
	}
}
