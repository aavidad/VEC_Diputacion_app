package postgres

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSesionConsultaRRHHConAmbitoNoAceptaPoolLegacyNiSQLAntiguo(t *testing.T) {
	if _, err := NuevaSesionConsultaRRHHConAmbitoPostgreSQL(nil); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("pool ausente aceptado: %v", err)
	}
	casos := []struct {
		sql, nuevo, antiguo string
		ultimo              int
	}{
		{consultaCuadroRRHHAmbitoPostgreSQL, "consultar_cuadro_rrhh_ambito_v1(", "consultar_cuadro_rrhh_atestado_v2(", 19},
		{consultaDetalleRRHHAmbitoPostgreSQL, "consultar_detalle_rrhh_ambito_v1(", "consultar_detalle_rrhh_atestado_v1(", 16},
		{consultaOriginalPropuestaRRHHAmbitoPostgreSQL, "consultar_original_propuesta_rrhh_ambito_v1(", "consultar_original_propuesta_rrhh_atestado_v1(", 16},
	}
	for _, c := range casos {
		if !strings.Contains(c.sql, c.nuevo) || strings.Contains(c.sql, c.antiguo) ||
			strings.Count(c.sql, "$"+strconv.Itoa(c.ultimo)+"::jsonb") != 1 {
			t.Fatalf("fachada producto no cerrada: %s", c.nuevo)
		}
	}
}
