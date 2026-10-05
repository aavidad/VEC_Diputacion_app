package main

import (
	"encoding/json"
)

func clavesDocumentoMantenimientoFijo(o map[string]json.RawMessage) error {
	var m map[string]json.RawMessage
	var rs []map[string]json.RawMessage
	if !clavesExactas(o, "esquema", "manifiesto", "registros") || json.Unmarshal(o["manifiesto"], &m) != nil || !clavesCoberturaExactas(m) || json.Unmarshal(o["registros"], &rs) != nil {
		return errJSONInvalido
	}
	for _, r := range rs {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		if tipo != "mantenimiento_perfil_fijo_admin" && tipo != "intento_mantenimiento_perfil_fijo_admin" {
			b, _ := json.Marshal([]map[string]json.RawMessage{r})
			if clavesDocumentoBootstrapCentral(map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}) != nil {
				return errJSONInvalido
			}
			continue
		}
		obj := "mantenimiento_fijo"
		extras := []string{"plan_sha256", "preimagen_sha256", "catalogo_sha256", "rol_origen_ref", "rol_origen_sha256", "rol_destino_ref", "rol_destino_sha256", "asignacion_1_origen_ref", "asignacion_1_origen_sha256", "asignacion_1_destino_ref", "asignacion_1_destino_sha256", "asignacion_2_origen_ref", "asignacion_2_origen_sha256", "asignacion_2_destino_ref", "asignacion_2_destino_sha256"}
		if tipo == "intento_mantenimiento_perfil_fijo_admin" {
			obj = "intento_mantenimiento_fijo"
			extras = []string{"solicitud_sha256"}
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", obj) || json.Unmarshal(r[obj], &campos) != nil {
			return errJSONInvalido
		}
		claves := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256", "operador_login", "accion", "modulo_id", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref"}
		if !clavesExactas(campos, append(claves, extras...)...) {
			return errJSONInvalido
		}
	}
	return nil
}
