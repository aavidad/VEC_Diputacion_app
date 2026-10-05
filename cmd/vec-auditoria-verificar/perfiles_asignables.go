package main

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// Familia AD196 con claves exactas; el resto de tipos se delega en el parser
// del esquema de contexto ADMIN previo a V2, que conserva los históricos.
func clavesDocumentoPerfilesAsignables(o map[string]json.RawMessage) error {
	var m map[string]json.RawMessage
	var rs []map[string]json.RawMessage
	if !clavesExactas(o, "esquema", "manifiesto", "registros") || json.Unmarshal(o["manifiesto"], &m) != nil || !clavesCoberturaExactas(m) || json.Unmarshal(o["registros"], &rs) != nil {
		return errJSONInvalido
	}
	comunes := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref", "evento_material_sha256", "operador_login", "accion", "modulo_id", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref"}
	for _, r := range rs {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		var objeto string
		var extras []string
		switch tipo {
		case "perfiles_asignables_admin":
			objeto, extras = "perfiles_asignables", []string{"plan_sha256", "operacion_ref", "perfiles_sha256", "perfiles_numero", "aprobacion_sha256"}
		case "intento_perfiles_asignables_admin":
			objeto, extras = "intento_perfiles_asignables", []string{"solicitud_sha256"}
		default:
			b, err := json.Marshal([]map[string]json.RawMessage{r})
			if err != nil {
				return errJSONInvalido
			}
			if clavesDocumentoContextoAdminPreV2(map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}) != nil {
				return errJSONInvalido
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", objeto) || json.Unmarshal(r[objeto], &campos) != nil || !clavesExactas(campos, append(append([]string(nil), comunes...), extras...)...) {
			return errJSONInvalido
		}
	}
	return nil
}

// Admite null sólo donde lo admite el esquema previo a V2 (contexto AD192).
func decodificarJSONPerfilesAsignables(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
	t := json.NewDecoder(bytes.NewReader(b))
	t.UseNumber()
	if err := valorUnicoContextoPreV2(t, nil); err != nil {
		return err
	}
	if _, err := t.Token(); err != io.EOF {
		return errJSONInvalido
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(d) != nil || decoder.Decode(new(any)) != io.EOF {
		return errJSONInvalido
	}
	var o map[string]json.RawMessage
	if json.Unmarshal(b, &o) != nil {
		return errJSONInvalido
	}
	return clavesDocumentoPerfilesAsignables(o)
}
