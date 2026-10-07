package main

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func decodificarJSONIdentidadInterna(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
	t := json.NewDecoder(bytes.NewReader(b))
	t.UseNumber()
	if valorUnicoContextoPreV2(t, nil) != nil {
		return errJSONInvalido
	}
	if _, err := t.Token(); err != io.EOF {
		return errJSONInvalido
	}
	t = json.NewDecoder(bytes.NewReader(b))
	t.DisallowUnknownFields()
	if t.Decode(d) != nil || t.Decode(new(any)) != io.EOF {
		return errJSONInvalido
	}
	var o map[string]json.RawMessage
	if json.Unmarshal(b, &o) != nil || separarEslabonesV5(o) != nil {
		return errJSONInvalido
	}
	return clavesDocumentoIdentidadInterna(o)
}

func clavesDocumentoIdentidadInterna(o map[string]json.RawMessage) error {
	var m map[string]json.RawMessage
	var rs []map[string]json.RawMessage
	if !clavesExactas(o, "esquema", "manifiesto", "registros") || json.Unmarshal(o["manifiesto"], &m) != nil ||
		!clavesCoberturaExactas(m) || json.Unmarshal(o["registros"], &rs) != nil {
		return errJSONInvalido
	}
	comunes := []string{"auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "evento_ref",
		"evento_material_sha256", "operador_login", "accion", "modulo_id", "recurso_ref", "resultado", "motivo_ref",
		"proceso", "canal", "finalidad_ref", "correlacion_ref"}
	for _, r := range rs {
		var tipo string
		if json.Unmarshal(r["tipo_registro"], &tipo) != nil {
			return errJSONInvalido
		}
		objeto := ""
		var extras []string
		switch tipo {
		case "provision_identidad_interna_sintetica":
			objeto = "provision_identidad_interna"
			extras = []string{"operacion_ref", "plan_ref", "plan_sha256", "preimagen_sha256", "configuracion_sha256",
				"aprobacion_ref", "alcance_fuente", "fuente_ref", "fuente_sha256"}
		case "intento_identidad_interna_sintetica":
			objeto = "intento_identidad_interna"
			extras = []string{"solicitud_sha256"}
		default:
			b, err := json.Marshal([]map[string]json.RawMessage{r})
			if err != nil {
				return errJSONInvalido
			}
			anterior := map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}
			if clavesDocumentoPerfilesAsignables(anterior) != nil && clavesDocumentoMantenimientoFijo(anterior) != nil &&
				clavesDocumentoFronteraAdminTecnica(anterior) != nil {
				return errJSONInvalido
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", objeto) || json.Unmarshal(r[objeto], &campos) != nil ||
			!clavesExactas(campos, append(append([]string(nil), comunes...), extras...)...) {
			return errJSONInvalido
		}
	}
	return nil
}
