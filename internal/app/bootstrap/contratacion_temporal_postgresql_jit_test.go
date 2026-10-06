package bootstrap

import "testing"

// Las lecturas de Bolsa y Contratación temporal no deben pagar compilación
// JIT por llamada; el resto de parámetros de seguridad se conservan.
func TestParametrosSesionPostgreSQLContratacionTemporalApaganJIT(t *testing.T) {
	parametros := map[string]string{}
	aplicarParametrosSesionPostgreSQLContratacionTemporalDesarrollo(parametros, "vec-prueba")
	esperados := map[string]string{
		"application_name":                    "vec-prueba",
		"timezone":                            "UTC",
		"search_path":                         "pg_catalog,pg_temp",
		"default_transaction_isolation":       "serializable",
		"default_transaction_read_only":       "off",
		"statement_timeout":                   "15s",
		"lock_timeout":                        "3s",
		"idle_in_transaction_session_timeout": "20s",
		"jit":                                 "off",
	}
	if len(parametros) != len(esperados) {
		t.Fatalf("parámetros inesperados: %v", parametros)
	}
	for clave, valor := range esperados {
		if parametros[clave] != valor {
			t.Fatalf("%s=%q; se esperaba %q", clave, parametros[clave], valor)
		}
	}
}
