package main

import "encoding/json"

func clavesDocumentoBootstrapCentral(objeto map[string]json.RawMessage) error {
	var registros []map[string]json.RawMessage
	var manifiesto map[string]json.RawMessage
	if !clavesExactas(objeto, "esquema", "manifiesto", "registros") ||
		json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil || !clavesCoberturaExactas(manifiesto) ||
		json.Unmarshal(objeto["registros"], &registros) != nil {
		return errJSONInvalido
	}
	for _, r := range registros {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		if tipo == "intento_bootstrap_central_admin" {
			var campos map[string]json.RawMessage
			if !clavesExactas(r, "tipo_registro", "intento_bootstrap_central") || json.Unmarshal(r["intento_bootstrap_central"], &campos) != nil ||
				!clavesExactas(campos, "auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256",
					"modulo_id", "operador_login", "solicitud_sha256", "accion", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref") {
				return errJSONInvalido
			}
			continue
		}
		b, _ := json.Marshal([]map[string]json.RawMessage{r})
		anterior := map[string]json.RawMessage{"esquema": objeto["esquema"], "manifiesto": objeto["manifiesto"], "registros": b}
		if clavesDocumentoUnidadInicial(anterior) != nil {
			return errJSONInvalido
		}
	}
	return nil
}
