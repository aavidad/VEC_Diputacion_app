package inscripcion

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaterialPresentacionUnicoParaDecisionYEfecto(t *testing.T) {
	p := Presentacion{
		ConvocatoriaRef: "cv1_YXV4aWxpYXI_v1", CategoriaRef: "categoria:rpt:auxiliar",
		CatalogoVersion: 1, ClaveIdempotencia: "inscripcion-00000001",
		Declaraciones: []Declaracion{{RequisitoCodigo: "titulacion"}, {RequisitoCodigo: "identidad_certificada"}},
	}
	primero, huella, err := MaterialPresentacion(p)
	if err != nil {
		t.Fatal(err)
	}
	p.Declaraciones[0], p.Declaraciones[1] = p.Declaraciones[1], p.Declaraciones[0]
	segundo, otra, err := MaterialPresentacion(p)
	if err != nil || !bytes.Equal(primero, segundo) || huella != otra {
		t.Fatal("declaraciones equivalentes cambiaron el material")
	}
	if !bytes.Contains(primero, []byte(`"declaraciones":[{"requisito_codigo":"identidad_certificada"},{"requisito_codigo":"titulacion"}]`)) {
		t.Fatal("orden canonico de declaraciones")
	}
	recurso, err := RecursoPresentacion(p, huella)
	if err != nil || string(recurso) != `{"ambitos":{"categoria_ref":"categoria:rpt:auxiliar","convocatoria_ref":"cv1_YXV4aWxpYXI_v1"},"atributos":{"material_sha256":"`+huella+`"}}` {
		t.Fatalf("recurso canonico: %s error=%v", recurso, err)
	}
	ref, err := ReferenciaSolicitud("per_sintetica_001", p)
	if err != nil || !strings.HasPrefix(ref, "solicitud_inscripcion_") || len(ref) != len("solicitud_inscripcion_")+64 {
		t.Fatalf("referencia=%q error=%v", ref, err)
	}
}
