package main

import (
	"bytes"
	"encoding/json"
	"io"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func clavesDocumentoContextoAdminPreV2(o map[string]json.RawMessage) error {
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
		if tipo != "contexto_admin_pre_v2" {
			b, err := json.Marshal([]map[string]json.RawMessage{r})
			if err != nil {
				return errJSONInvalido
			}
			if clavesDocumentoFronteraAdminTecnica(map[string]json.RawMessage{"esquema": o["esquema"], "manifiesto": o["manifiesto"], "registros": b}) != nil {
				return errJSONInvalido
			}
			continue
		}
		var campos map[string]json.RawMessage
		if !clavesExactas(r, "tipo_registro", "contexto_admin_pre_v2") || json.Unmarshal(r["contexto_admin_pre_v2"], &campos) != nil {
			return errJSONInvalido
		}
		copia := make(map[string]json.RawMessage, len(campos))
		for k, v := range campos {
			copia[k] = v
		}
		for _, k := range []string{"actor_ref", "perfil_activo_ref", "fuente_ref", "fuente_sha256"} {
			if bytes.Equal(bytes.TrimSpace(copia[k]), []byte("null")) {
				copia[k] = json.RawMessage(`"marcador_nullable"`)
			}
		}
		if !clavesExactas(copia, "auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256", "registrada_en", "tipo_registro", "evento_ref", "evento_material_sha256", "operador_login", "actor_ref", "perfil_activo_ref", "accion", "modulo_id", "recurso_ref", "resultado", "motivo_ref", "proceso", "canal", "finalidad_ref", "correlacion_ref", "fuente_ref", "fuente_sha256") {
			return errJSONInvalido
		}
	}
	return nil
}

// El parser histórico rechaza todo null. Esta entrada propia sólo los permite
// en los cuatro slots acreditativos del payload AD192, sin relajar otros tipos.
func decodificarJSONContextoAdminPreV2(b []byte, d *auditoria.DocumentoVerificacionMixta) error {
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
	return clavesDocumentoContextoAdminPreV2(o)
}
func valorUnicoContextoPreV2(d *json.Decoder, ruta []string) error {
	if len(ruta) > 16 {
		return errJSONInvalido
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t == nil {
		if len(ruta) == 4 && ruta[0] == "registros" && ruta[2] == "contexto_admin_pre_v2" {
			switch ruta[3] {
			case "actor_ref", "perfil_activo_ref", "fuente_ref", "fuente_sha256":
				return nil
			}
		}
		return errJSONInvalido
	}
	delim, ok := t.(json.Delim)
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
			token, err := d.Token()
			if err != nil {
				return err
			}
			var valido bool
			clave, valido = token.(string)
			if !valido || vistas[clave] {
				return errJSONInvalido
			}
			vistas[clave] = true
		}
		siguiente := append(append([]string(nil), ruta...), clave)
		if err := valorUnicoContextoPreV2(d, siguiente); err != nil {
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
