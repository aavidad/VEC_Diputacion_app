package main

import "encoding/json"

func clavesDocumentoFuentesIniciales(objeto map[string]json.RawMessage) error {
	var registros []map[string]json.RawMessage
	if !clavesExactas(objeto, "esquema", "manifiesto", "registros") || json.Unmarshal(objeto["registros"], &registros) != nil {
		return errJSONInvalido
	}
	for _, r := range registros {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		if tipo != "provision_fuentes_iniciales_admin" {
			b, _ := json.Marshal([]map[string]json.RawMessage{r})
			anterior := map[string]json.RawMessage{"esquema": objeto["esquema"], "manifiesto": objeto["manifiesto"], "registros": b}
			if clavesDocumentoPreperfil(anterior) != nil {
				return errJSONInvalido
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", "fuentes_iniciales") || json.Unmarshal(r["fuentes_iniciales"], &campos) != nil ||
			!clavesEventoExactas(campos, []string{"operador_login", "plan_ref", "plan_sha256", "preimagen_sha256", "configuracion_sha256", "aprobacion_ref", "alcance_fuente"}) {
			return errJSONInvalido
		}
	}
	var manifiesto map[string]json.RawMessage
	if json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil || !clavesCoberturaExactas(manifiesto) {
		return errJSONInvalido
	}
	return nil
}
