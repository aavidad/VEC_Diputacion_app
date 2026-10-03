package main

import (
	"encoding/json"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func decodificarJSONEstrictoPreperfil(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
	return decodificarJSONEstricto(b, d)
}

func clavesDocumentoPreperfil(objeto map[string]json.RawMessage) error {
	var manifiesto map[string]json.RawMessage
	var registros []map[string]json.RawMessage
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
		var campos map[string]json.RawMessage
		switch tipo {
		case "consumo_confirmado":
			if !clavesExactas(r, "tipo_registro", "consumo") || json.Unmarshal(r["consumo"], &campos) != nil ||
				!clavesExactas(campos, "auditoria_ref", "secuencia", "decision_ref", "efecto_ref", "huella_efecto_sha256", "anterior_sha256", "huella_sha256", "consumo_huella_sha256") {
				return errJSONInvalido
			}
		case "intento_nominal":
			if !clavesExactas(r, "tipo_registro", "intento") || json.Unmarshal(r["intento"], &campos) != nil || !clavesIntentoExactas(campos) {
				return errJSONInvalido
			}
		case "preperfil_autenticado", "bootstrap_operador":
			objeto := "preperfil"
			extra := []string{"actor_ref"}
			if tipo == "bootstrap_operador" {
				objeto = "bootstrap"
				extra = []string{"operador_login", "plan_sha256", "aprobacion_ref"}
			}
			if !clavesExactas(r, "tipo_registro", objeto) || json.Unmarshal(r[objeto], &campos) != nil || !clavesEventoExactas(campos, extra) {
				return errJSONInvalido
			}
		default:
			return errJSONInvalido
		}
	}
	return nil
}

func clavesEventoExactas(campos map[string]json.RawMessage, extra []string) bool {
	claves := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256",
		"modulo_id", "accion", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref", "fuente_ref", "fuente_sha256"}
	return clavesExactas(campos, append(claves, extra...)...)
}
