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

func TestMaterialDecisionLigaVersionMotivoYClave(t *testing.T) {
	d := Decision{SolicitudRef: "solicitud_inscripcion_" + strings.Repeat("a", 64), Tipo: "rechazar", MotivoCodigo: "requisito.no.acreditado", VersionEsperada: 2, ClaveIdempotencia: "inscripcion-00000002"}
	material, huella, err := MaterialDecision(d)
	if err != nil || !bytes.Contains(material, []byte(`"version_esperada":2`)) || !bytes.Contains(material, []byte(`"motivo_codigo":"requisito.no.acreditado"`)) {
		t.Fatalf("material=%s err=%v", material, err)
	}
	recurso, err := RecursoDecision(d, huella)
	if err != nil || !bytes.Contains(recurso, []byte(d.SolicitudRef)) || !bytes.Contains(recurso, []byte(huella)) {
		t.Fatalf("recurso=%s err=%v", recurso, err)
	}
	d.VersionEsperada = 3
	_, otra, err := MaterialDecision(d)
	if err != nil || otra == huella {
		t.Fatal("una version distinta conserva la misma huella")
	}
}

func TestMaterialIncorporacionLigaEvidenciaSinAceptarParticipacion(t *testing.T) {
	i := Incorporacion{SolicitudRef: "solicitud_inscripcion_" + strings.Repeat("a", 64), EvidenciaRef: "acta:resolucion:001", VersionEsperada: 2, ClaveIdempotencia: "inscripcion-00000003"}
	material, huella, err := MaterialIncorporacion(i)
	if err != nil || !bytes.Contains(material, []byte(`"evidencia_ref":"acta:resolucion:001"`)) || bytes.Contains(material, []byte("participacion_ref")) {
		t.Fatalf("material=%s err=%v", material, err)
	}
	recurso, err := RecursoIncorporacion(i, huella)
	if err != nil || !bytes.Contains(recurso, []byte(i.SolicitudRef)) || !bytes.Contains(recurso, []byte(huella)) {
		t.Fatalf("recurso=%s err=%v", recurso, err)
	}
	i.EvidenciaRef = "acta:resolucion:002"
	_, otra, err := MaterialIncorporacion(i)
	if err != nil || otra == huella {
		t.Fatal("evidencia distinta conserva la huella")
	}
}
