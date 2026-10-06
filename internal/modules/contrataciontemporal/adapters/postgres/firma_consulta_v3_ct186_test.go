package postgres

import (
	"os"
	"strings"
	"testing"
)

// Desde CT186 el adaptador llama a las v3 de la consulta y la recuperación
// R5 V2, que CT186 concede al ejecutor, y a las que retira las v2.
func TestConsultaYRecuperacionUsanLasV3DeCT186(t *testing.T) {
	b, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/migraciones/000186_consulta_firmas_r5_ambitos_asignacion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ct186 := string(b)
	i := strings.Index(ct186, "GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v3(")
	j := strings.Index(ct186, "TO vec_contratacion_temporal_ejecutor;")
	if i < 0 || j < i {
		t.Fatal("CT186 no concede las v3 al ejecutor")
	}
	concesion := ct186[i:j]
	for sql, nombre := range map[string]string{consultarFirmasSQL172: "consultar_firmas_r5_atestadas", recuperarFirmasSQL175: "recuperar_firmas_r5_atestadas"} {
		if !strings.Contains(sql, "SELECT vec_contratacion_temporal."+nombre+"_v3(") || !strings.Contains(concesion, nombre+"_v3(") {
			t.Fatalf("el adaptador no llama a la v3 de %s que concede CT186: %s", nombre, sql)
		}
	}
	if !strings.Contains(ct186, "REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_firmas_r5_atestadas_v2(") {
		t.Fatal("CT186 no retira las v2 al ejecutor")
	}
}
