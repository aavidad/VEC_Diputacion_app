package main

import "encoding/json"

func clavesDocumentoUnidadInicial(objeto map[string]json.RawMessage) error {
	var registros []map[string]json.RawMessage
	if !clavesExactas(objeto, "esquema", "manifiesto", "registros") || json.Unmarshal(objeto["registros"], &registros) != nil {
		return errJSONInvalido
	}
	for _, r := range registros {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		if tipo == "intento_unidad_inicial_personal" {
			if !clavesIntentoUnidadInicialExactas(r) {
				return errJSONInvalido
			}
			continue
		}
		if tipo != "unidad_inicial_personal" {
			b, _ := json.Marshal([]map[string]json.RawMessage{r})
			anterior := map[string]json.RawMessage{"esquema": objeto["esquema"], "manifiesto": objeto["manifiesto"], "registros": b}
			if clavesDocumentoFuentesIniciales(anterior) != nil {
				return errJSONInvalido
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", "unidad_inicial") || json.Unmarshal(r["unidad_inicial"], &campos) != nil ||
			!clavesEventoExactas(campos, []string{"operador_login", "plan_ref", "plan_sha256", "preimagen_sha256", "configuracion_sha256", "aprobacion_ref", "alcance_fuente", "recibo_ref", "recibo_sha256"}) {
			return errJSONInvalido
		}
	}
	var manifiesto map[string]json.RawMessage
	if json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil || !clavesCoberturaExactas(manifiesto) {
		return errJSONInvalido
	}
	return nil
}

func clavesIntentoUnidadInicialExactas(r map[string]json.RawMessage) bool {
	var campos map[string]json.RawMessage
	return clavesExactas(r, "tipo_registro", "intento_unidad_inicial") && json.Unmarshal(r["intento_unidad_inicial"], &campos) == nil &&
		clavesExactas(campos, "auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256",
			"modulo_id", "operador_login", "solicitud_sha256", "accion", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref")
}
