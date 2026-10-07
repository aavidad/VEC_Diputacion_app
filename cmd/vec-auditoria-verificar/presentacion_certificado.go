package main

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/auditoria"
)

// Los seis valores actuales son NULL únicamente al revocar. AD192 conserva
// sus cuatro valores históricamente anulables en su propio objeto.
func nuloPresentacionCertificadoAdmitido(ruta []string) bool {
	if len(ruta) != 4 || ruta[0] != "registros" {
		return false
	}
	if ruta[2] == "contexto_admin_pre_v2" {
		switch ruta[3] {
		case "actor_ref", "perfil_activo_ref", "fuente_ref", "fuente_sha256":
			return true
		}
	}
	if ruta[2] == "presentacion_certificado" {
		switch ruta[3] {
		case "actor_cuenta_ref", "actor_autenticacion_ref", "actor_sesion_ref",
			"acr_actual", "politica_actual_ref", "politica_actual_sha256", "canal_sha256",
			"asercion_actual_sha256", "presentacion_valida_hasta",
			"control_sesion_ref", "control_sesion_revision", "control_origen_operacion_ref":
			return true
		}
	}
	return false
}

func valorUnicoPresentacionCertificado(d *json.Decoder, ruta []string) error {
	if len(ruta) > 16 {
		return errJSONInvalido
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token == nil {
		if nuloPresentacionCertificadoAdmitido(ruta) {
			return nil
		}
		return errJSONInvalido
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errJSONInvalido
	}
	vistas := map[string]bool{}
	for d.More() {
		clave := "indice"
		if delim == '{' {
			valor, err := d.Token()
			if err != nil {
				return err
			}
			clave, ok = valor.(string)
			if !ok || vistas[clave] {
				return errJSONInvalido
			}
			vistas[clave] = true
		}
		if err := valorUnicoPresentacionCertificado(d, append(append([]string(nil), ruta...), clave)); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil {
		return err
	}
	if delim == '{' && cierre != json.Delim('}') || delim == '[' && cierre != json.Delim(']') {
		return errJSONInvalido
	}
	return nil
}

func clavesDocumentoPresentacionCertificado(o map[string]json.RawMessage) error {
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
	extras := []string{"fase", "operacion_ref", "presentacion_ref", "autenticacion_original_ref",
		"sesion_original_ref", "cuenta_ref", "actor_cuenta_ref", "actor_autenticacion_ref",
		"actor_sesion_ref", "sujeto_id_hmac", "superficie", "tipo", "recibo_sha256",
		"acr_original", "acr_actual", "politica_original_ref", "politica_original_sha256",
		"politica_actual_ref", "politica_actual_sha256", "canal_sha256",
		"asercion_actual_sha256", "presentacion_valida_hasta", "control_sesion_ref",
		"control_sesion_revision", "control_origen_operacion_ref"}
	for _, registro := range registros {
		var tipo string
		if err := json.Unmarshal(registro["tipo_registro"], &tipo); err != nil {
			return errJSONInvalido
		}
		if tipo != "presentacion_certificado_v1" {
			b, err := json.Marshal([]map[string]json.RawMessage{registro})
			if err != nil {
				return errJSONInvalido
			}
			previo := map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}
			if err := clavesDocumentoCatalogoAcciones(previo); err != nil {
				return err
			}
			continue
		}
		if !clavesExactas(registro, "tipo_registro", "presentacion_certificado") {
			return errJSONInvalido
		}
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(registro["presentacion_certificado"], &campos); err != nil {
			return errJSONInvalido
		}
		if !clavesExactas(campos, append(append([]string(nil), comunes...), extras...)...) {
			return errJSONInvalido
		}
		var subtipo string
		if err := json.Unmarshal(campos["tipo"], &subtipo); err != nil {
			return errJSONInvalido
		}
		tecnica := subtipo == "revocacion_control_is2_sin_actor_humano_nominal"
		for _, clave := range []string{"actor_cuenta_ref", "actor_autenticacion_ref", "actor_sesion_ref"} {
			esNulo := bytes.Equal(bytes.TrimSpace(campos[clave]), []byte("null"))
			if esNulo != tecnica {
				return errJSONInvalido
			}
		}
		for _, clave := range []string{"acr_actual", "politica_actual_ref", "politica_actual_sha256",
			"canal_sha256", "asercion_actual_sha256", "presentacion_valida_hasta"} {
			esNulo := bytes.Equal(bytes.TrimSpace(campos[clave]), []byte("null"))
			if esNulo != (subtipo == "revocacion" || tecnica) {
				return errJSONInvalido
			}
		}
		for _, clave := range []string{"control_sesion_ref", "control_sesion_revision", "control_origen_operacion_ref"} {
			esNulo := bytes.Equal(bytes.TrimSpace(campos[clave]), []byte("null"))
			if esNulo == tecnica {
				return errJSONInvalido
			}
		}
	}
	return nil
}

func decodificarJSONPresentacionCertificado(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
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
	return clavesDocumentoPresentacionCertificado(o)
}
