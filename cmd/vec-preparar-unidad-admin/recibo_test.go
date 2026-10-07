package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestUnidadReciboCerradoYVinculado(t *testing.T) {
	p := planFixture(t)
	base := envelopeFixture(t, p, "permitido", false)
	casos := map[string]func([]byte) []byte{
		"nodo_ajeno": func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte(p.Plan.Unidad.NodoRef), []byte("bbbbbbbb-2222-4333-8444-555555555555"))
		},
		"campo_extra": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"estado":`), []byte(`"perfil":"admin","estado":`), 1)
		},
		"dup": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"estado":`), []byte(`"estado":"permitido","estado":`), 1)
		},
		"seq_overflow": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"secuencia":3`), []byte(`"secuencia":9007199254740992`), 1)
		},
		"audit_ajena": func(b []byte) []byte { return bytes.Replace(b, []byte("aud_v3_ui_"), []byte("aud_v3_fi_"), 1) },
		"receipt_null": func(b []byte) []byte {
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			m["recibo"] = nil
			b, _ = json.Marshal(m)
			return b
		},
		"padre_no_null": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"centro_padre_ref":null`), []byte(`"centro_padre_ref":"bbbbbbbb-2222-4333-8444-555555555555"`), 1)
		},
		"retirada": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"retirado":false`), []byte(`"retirado":true`), 1)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, e := validarEnvoltura(cambiar(base), p.Plan, p.HuellaPlanSHA256); e == nil {
				t.Fatal("recibo falso aceptado")
			}
		})
	}
}
