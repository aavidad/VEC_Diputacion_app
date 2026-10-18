package postgres

import (
	"reflect"
	"strings"
	"testing"
)

// columnasSelectConsultaRRHH devuelve, en orden, los nombres de columna entre
// SELECT y FROM, y la función de la que lee la consulta.
func columnasSelectConsultaRRHH(t *testing.T, consulta string) ([]string, string) {
	t.Helper()
	cuerpo := strings.TrimSpace(consulta)
	inicio, fin := strings.Index(cuerpo, "SELECT "), strings.Index(cuerpo, "\n  FROM ")
	if inicio != 0 || fin < 0 {
		t.Fatalf("consulta sin SELECT ... FROM reconocible")
	}
	var columnas []string
	for _, c := range strings.Split(cuerpo[len("SELECT "):fin], ",") {
		c = strings.TrimSpace(c)
		c = strings.TrimSuffix(c, "::bigint")
		columnas = append(columnas, c)
	}
	desde := strings.TrimSpace(cuerpo[fin+len("\n  FROM "):])
	return columnas, desde[:strings.Index(desde, "(")]
}

// El detalle ordinario lee la cifra de personas en sus cuatro variantes; con
// la concesión de firma histórica (selectores) ya no queda «sin cantidad».
// El orden de columnas debe coincidir con el de destinos de consultarDetalle:
// cierre, resultado de Bolsa, cifra y anexo de selectores.
func TestDetalleRRHHLeeNumeroPersonasTambienConSelectores(t *testing.T) {
	base := []string{"contenido_canonico", "esquema", "acceso_ref", "secuencia",
		"anterior_sha256", "huella_sha256", "vinculo_identidad_huella_sha256",
		"alcance_huella_sha256", "registrada_en", "auditoria_vec_ref",
		"auditoria_vec_huella_sha256", "consumo_vec_huella_sha256",
		"contenido_huella_sha256", "resultado_huella_sha256", "cursor_huella_sha256",
		"generada_en", "expediente_ref", "version_expediente", "total", "recibo_sello_sha256"}
	if len(destinosDetalleConsultaRRHH(&salidaDetalleConsultaRRHH{})) != len(base) {
		t.Fatal("destinos del cierre no coinciden con las columnas base")
	}
	selectores := []string{"selectores_canon", "selectores_huella_sha256",
		"sello_selectores_sha256", "recibo_selectores_canon"}
	casos := []struct {
		nombre, consulta, funcion string
		bolsa, conSelectores      bool
	}{
		{"detalle", consultaDetalleRRHHPostgreSQL,
			"vec_contratacion_temporal.consultar_detalle_rrhh_con_numero_ct225", false, false},
		{"detalle_bolsa", consultaDetalleConBolsaRRHHPostgreSQL,
			"vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_con_numero_ct225", true, false},
		{"detalle_selectores", consultaDetalleConSelectoresRRHHPostgreSQL,
			"vec_contratacion_temporal.consultar_detalle_rrhh_selectores_con_numero_ct225", false, true},
		{"detalle_bolsa_selectores", consultaDetalleConBolsaYSelectoresRRHHPostgreSQL,
			"vec_contratacion_temporal.consultar_detalle_rrhh_bolsa_selectores_con_numero_ct225", true, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if !consultaDetalleRRHHConNumeroPersonas(caso.consulta) {
				t.Fatal("la consulta no declara la cifra de personas")
			}
			esperadas := append([]string(nil), base...)
			if caso.bolsa {
				esperadas = append(esperadas, "resultado_bolsa")
			}
			esperadas = append(esperadas, "numero_personas")
			if caso.conSelectores {
				esperadas = append(esperadas, selectores...)
			}
			columnas, funcion := columnasSelectConsultaRRHH(t, caso.consulta)
			if !reflect.DeepEqual(columnas, esperadas) {
				t.Fatalf("columnas = %v, esperadas %v", columnas, esperadas)
			}
			if funcion != caso.funcion {
				t.Fatalf("lee de %q, se esperaba %q", funcion, caso.funcion)
			}
		})
	}
	for _, consulta := range []string{consultaOriginalPropuestaRRHHPostgreSQL,
		consultaOriginalPropuestaConSelectoresRRHHPostgreSQL} {
		columnas, _ := columnasSelectConsultaRRHH(t, consulta)
		if consultaDetalleRRHHConNumeroPersonas(consulta) ||
			strings.Contains(strings.Join(columnas, ","), "numero_personas") {
			t.Fatal("el original de la propuesta no debe leer la cifra")
		}
	}
}
