package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestIdentidadInternaJSONExacto(t *testing.T) {
	d := auditoria.DocumentoVerificacionMixta{
		Esquema: auditoria.EsquemaVerificacionIdentidadInternaSintetica,
		Registros: []auditoria.RegistroMixtoV2{{
			TipoRegistro:              "provision_identidad_interna_sintetica",
			ProvisionIdentidadInterna: &auditoria.RegistroProvisionIdentidadInternaSinteticaV1{},
		}},
	}
	b, err := json.Marshal(d)
	if err != nil || decodificarJSONIdentidadInterna(b, &auditoria.DocumentoVerificacionMixta{}) != nil {
		t.Fatal("estructura AD215 válida rechazada")
	}
	for _, caso := range []struct {
		nombre string
		a, b   []byte
	}{
		{"campo_desconocido", []byte(`"operacion_ref":""`), []byte(`"operacion_ref":"","campo_inyectado":"x"`)},
		{"campo_retirado", []byte(`"operacion_ref":"",`), nil},
		{"tipo_duplicado", []byte(`"tipo_registro":"provision_identidad_interna_sintetica"`), []byte(`"tipo_registro":"provision_identidad_interna_sintetica","tipo_registro":"provision_identidad_interna_sintetica"`)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := bytes.Replace(b, caso.a, caso.b, 1)
			if bytes.Equal(mutado, b) || decodificarJSONIdentidadInterna(mutado, &auditoria.DocumentoVerificacionMixta{}) == nil {
				t.Fatal("documento ambiguo o incompleto aceptado")
			}
		})
	}
}
