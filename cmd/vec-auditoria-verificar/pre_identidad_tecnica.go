package main

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// AD222 añade cuatro campos técnicos a la proyección común. Conserva los
// nulos históricos acreditados de AD192/AD221 sin admitir ninguno en AD222.
func clavesDocumentoPreIdentidadTecnica(o map[string]json.RawMessage) error {
	var manifiesto map[string]json.RawMessage
	var registros []map[string]json.RawMessage
	if !clavesExactas(o, "esquema", "manifiesto", "registros") {
		return errJSONInvalido
	}
	if err := json.Unmarshal(o["manifiesto"], &manifiesto); err != nil {
		return errJSONInvalido
	}
	if !clavesCoberturaExactas(manifiesto) {
		return errJSONInvalido
	}
	if err := json.Unmarshal(o["registros"], &registros); err != nil {
		return errJSONInvalido
	}
	comunes := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en",
		"evento_ref", "evento_material_sha256", "operador_login", "accion", "modulo_id", "recurso_ref",
		"resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref"}
	for _, registro := range registros {
		var tipo string
		if err := json.Unmarshal(registro["tipo_registro"], &tipo); err != nil {
			return errJSONInvalido
		}
		if tipo != "pre_identidad_tecnica_v1" {
			b, err := json.Marshal([]map[string]json.RawMessage{registro})
			if err != nil {
				return errJSONInvalido
			}
			previo := map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}
			if err := clavesDocumentoPresentacionCertificado(previo); err != nil {
				return err
			}
			continue
		}
		if !clavesExactas(registro, "tipo_registro", "pre_identidad_tecnica") {
			return errJSONInvalido
		}
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(registro["pre_identidad_tecnica"], &campos); err != nil {
			return errJSONInvalido
		}
		if !clavesExactas(campos, append(append([]string(nil), comunes...),
			"fase", "metodo_esperado", "ruta", "superficie")...) {
			return errJSONInvalido
		}
	}
	return nil
}

func decodificarJSONPreIdentidadTecnica(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
	t := json.NewDecoder(bytes.NewReader(b))
	t.UseNumber()
	if err := valorUnicoPresentacionCertificado(t, nil); err != nil {
		return errJSONInvalido
	}
	if _, err := t.Token(); err != io.EOF {
		return errJSONInvalido
	}
	t = json.NewDecoder(bytes.NewReader(b))
	t.DisallowUnknownFields()
	if err := t.Decode(d); err != nil {
		return errJSONInvalido
	}
	if err := t.Decode(new(any)); err != io.EOF {
		return errJSONInvalido
	}
	var o map[string]json.RawMessage
	if err := json.Unmarshal(b, &o); err != nil {
		return errJSONInvalido
	}
	if err := separarEslabonesV5(o); err != nil {
		return err
	}
	return clavesDocumentoPreIdentidadTecnica(o)
}
