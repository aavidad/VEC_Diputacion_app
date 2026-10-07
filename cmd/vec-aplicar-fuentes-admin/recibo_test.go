package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvolturaCerradaYVinculada(t *testing.T) {
	p := planFixture(t)
	base := envelopeFixture(t, p, "permitido", false)
	casos := map[string]func([]byte) []byte{
		"campo_extra": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"estado":`), []byte(`"secreto":"no mostrar","estado":`), 1)
		},
		"repetido": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"estado":`), []byte(`"estado":"permitido","estado":`), 1)
		},
		"audit_null": func(b []byte) []byte {
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			m["auditoria_intento"] = nil
			b, _ = json.Marshal(m)
			return b
		},
		"seq_overflow": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"secuencia":3`), []byte(`"secuencia":9007199254740992`), 1)
		},
		"receipt_ajeno": func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte(p.Plan.OperacionRef), []byte("pfi_"+strings.Repeat("j", 22)))
		},
		"permitido_sin_receipt": func(b []byte) []byte {
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			m["recibo"] = nil
			b, _ = json.Marshal(m)
			return b
		},
		"denegado_con_receipt": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"estado":"permitido"`), []byte(`"estado":"denegado"`), 1)
		},
		"fuente_excesiva": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"organizacion_version":1`), []byte(`"organizacion_version":1,"clave_hmac":"no mostrar"`), 1)
		},
		"receipt_incompleto": func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"configuracion_sha256":"`+strings.Repeat("e", 64)+`",`), nil, 1)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, e := validarEnvoltura(cambiar(base), p.Plan, p.HuellaPlanSHA256); e == nil {
				t.Fatal("acepta respuesta inválida")
			}
		})
	}
}
func TestReplayReciboOriginalIntentoDistinto(t *testing.T) {
	p := planFixture(t)
	primero := envelopeFixture(t, p, "permitido", false)
	segundo := envelopeFixture(t, p, "permitido", true)
	var a, b envoltura
	if decodificarEstricto(primero, &a) != nil || decodificarEstricto(segundo, &b) != nil {
		t.Fatal("fixture")
	}
	b.AuditoriaIntento.AuditoriaRef = "aud_v3_fi_" + strings.Repeat("b", 32)
	b.AuditoriaIntento.Secuencia++
	raw, e := json.Marshal(b)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := validarEnvoltura(raw, p.Plan, p.HuellaPlanSHA256); e != nil {
		t.Fatal(e)
	}
	original, _ := json.Marshal(a.Recibo)
	recuperado, _ := json.Marshal(b.Recibo)
	if !bytes.Equal(original, recuperado) || a.AuditoriaIntento.AuditoriaRef == b.AuditoriaIntento.AuditoriaRef {
		t.Fatal("replay cambia recibo o reutiliza intento")
	}
}
