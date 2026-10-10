package inscripcion

import (
	"bytes"
	"strings"
	"testing"
)

func fuentesGestionPrueba() (AmbitoGestionInscripcion, AmbitoGestionInscripcion) {
	conjunto := AmbitoGestionInscripcion{ConjuntoRef: "conjunto:gestion:rrhh", UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:bolsa",
		FuenteRef: "catalogo:conjunto:2", FuenteVersion: 2, FuenteSHA256: strings.Repeat("b", 64)}
	solicitud := AmbitoGestionInscripcion{UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:bolsa",
		FuenteRef: "catalogo:convocatoria:1", FuenteVersion: 1, FuenteSHA256: strings.Repeat("a", 64)}
	return conjunto, solicitud
}

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
	recurso, err := RecursoPresentacion(p, huella, map[string]string{"candidato_ref": "can_candidata_0000000001"})
	if err != nil || string(recurso) != `{"ambitos":{"candidato_ref":"can_candidata_0000000001"},"atributos":{"catalogo_version":"1","categoria_ref":"categoria:rpt:auxiliar","convocatoria_ref":"cv1_YXV4aWxpYXI_v1","material_sha256":"`+huella+`"}}` {
		t.Fatalf("recurso canonico: %s error=%v", recurso, err)
	}
	if _, err := RecursoPresentacion(p, huella, map[string]string{"categoria_ref": p.CategoriaRef}); err == nil {
		t.Fatal("la categoría del navegador no puede convertirse en ámbito")
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
	conjunto, solicitud := fuentesGestionPrueba()
	recurso, err := RecursoDecision(d, huella, map[string]string{"unidad_ref": "unidad:rrhh", "ambito_ref": "ambito:bolsa"}, conjunto, solicitud)
	if err != nil || !bytes.Contains(recurso, []byte(d.SolicitudRef)) || !bytes.Contains(recurso, []byte(huella)) {
		t.Fatalf("recurso=%s err=%v", recurso, err)
	}
	if _, err := RecursoDecision(d, huella, map[string]string{"solicitud_ref": d.SolicitudRef}, conjunto, solicitud); err == nil {
		t.Fatal("la solicitud del navegador no puede convertirse en ámbito RRHH")
	}
	conjunto.FuenteVersion = 3
	recursoActual, err := RecursoDecision(d, huella, map[string]string{"unidad_ref": "unidad:rrhh", "ambito_ref": "ambito:bolsa"}, conjunto, solicitud)
	if err != nil || bytes.Equal(recursoActual, recurso) {
		t.Fatal("la fuente actual no queda ligada a la nueva decisión V3")
	}
	materialRepetido, mismaHuella, err := MaterialDecision(d)
	if err != nil || !bytes.Equal(materialRepetido, material) || mismaHuella != huella {
		t.Fatal("la fuente de autorización alteró la clave del comando idempotente")
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
	conjunto, solicitud := fuentesGestionPrueba()
	recurso, err := RecursoIncorporacion(i, huella, map[string]string{"unidad_ref": "unidad:rrhh", "ambito_ref": "ambito:bolsa"}, conjunto, solicitud)
	if err != nil || !bytes.Contains(recurso, []byte(i.SolicitudRef)) || !bytes.Contains(recurso, []byte(huella)) {
		t.Fatalf("recurso=%s err=%v", recurso, err)
	}
	i.EvidenciaRef = "acta:resolucion:002"
	_, otra, err := MaterialIncorporacion(i)
	if err != nil || otra == huella {
		t.Fatal("evidencia distinta conserva la huella")
	}
}
