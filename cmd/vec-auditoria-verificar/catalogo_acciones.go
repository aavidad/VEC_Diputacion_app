package main

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// El esquema AD219 conserva todas las familias anteriores y admite únicamente
// los dos objetos nuevos con sus columnas proyectadas exactas.
func clavesDocumentoCatalogoAcciones(o map[string]json.RawMessage) error {
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
		var objeto string
		var extras []string
		switch tipo {
		case "catalogo_acciones_admin":
			objeto = "catalogo_acciones"
			extras = []string{"operacion_ref", "plan_sha256", "catalogo_ref", "catalogo_version", "catalogo_sha256",
				"paquete_ref", "paquete_version", "paquete_sha256", "censo_sha256", "entradas_numero",
				"perfiles_numero", "aprobacion_ref", "aprobacion_sha256"}
		case "intento_catalogo_acciones_admin":
			objeto, extras = "intento_catalogo_acciones", []string{"solicitud_sha256"}
		default:
			b, err := json.Marshal([]map[string]json.RawMessage{registro})
			if err != nil {
				return errJSONInvalido
			}
			previo := map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}
			if err := clavesDocumentoIdentidadInterna(previo); err != nil {
				return err
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(registro, "tipo_registro", objeto) {
			return errJSONInvalido
		}
		if err := json.Unmarshal(registro[objeto], &campos); err != nil {
			return errJSONInvalido
		}
		if !clavesExactas(campos, append(append([]string(nil), comunes...), extras...)...) {
			return errJSONInvalido
		}
	}
	return nil
}

func decodificarJSONCatalogoAcciones(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
	t := json.NewDecoder(bytes.NewReader(b))
	t.UseNumber()
	if err := valorUnicoContextoPreV2(t, nil); err != nil {
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
	return clavesDocumentoCatalogoAcciones(o)
}
