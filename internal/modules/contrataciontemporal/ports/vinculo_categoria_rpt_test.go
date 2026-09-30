package ports

import (
	"strings"
	"testing"
)

func registroVinculoRPTPrueba() RegistroVinculoCategoriaRPT {
	return RegistroVinculoCategoriaRPT{Esquema: EsquemaRegistroVinculoCategoriaRPT,
		OrganizacionRef: "organizacion:uno", ExpedienteRef: "expediente:uno", VersionExpedienteEsperada: 7,
		AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisHuellaSHA256: strings.Repeat("a", 64),
		CategoriaRef: "categoria:tecnica", CatalogoID: "rpt-categorias", ModuloID: "personal", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "categoria:tecnica", FuenteRef: "fuente:expediente",
		MotivoRef: "motivo:uno", AprobacionRef: "aprobacion:uno", RevisionEsperada: 0, ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
}

func TestRegistroVinculoRPTCanonicoFijoYConflictoLiteral(t *testing.T) {
	m := registroVinculoRPTPrueba()
	b, e := m.Canonico()
	if e != nil {
		t.Fatal(e)
	}
	esperado := `{"esquema":"vec.ct.vinculo-categoria-rpt.registro.v1","organizacion_ref":"organizacion:uno","expediente_ref":"expediente:uno","version_expediente_esperada":7,"analisis_version":2,"analisis_recibo_ref":"recibo:analisis","analisis_huella_sha256":"` + strings.Repeat("a", 64) + `","categoria_ref":"categoria:tecnica","catalogo_id":"rpt-categorias","modulo_id":"personal","catalogo_version":1,"catalogo_huella_sha256":"` + strings.Repeat("b", 64) + `","categoria_id":"categoria:tecnica","fuente_ref":"fuente:expediente","motivo_ref":"motivo:uno","aprobacion_ref":"aprobacion:uno","revision_esperada":0,"anterior_recibo_ref":null,"clave_idempotencia":"11111111-1111-4111-8111-111111111111"}`
	if string(b) != esperado {
		t.Fatalf("material alterado: %s", b)
	}
	m.CategoriaID = "categoria:auxiliar"
	if _, e = m.Canonico(); e == nil {
		t.Fatal("categoria diferente admitida")
	}
	m = registroVinculoRPTPrueba()
	m.RevisionEsperada = 1
	if _, e = m.Canonico(); e == nil {
		t.Fatal("revision sin recibo anterior admitida")
	}
	anterior := "recibo:vinculo-anterior"
	m.AnteriorReciboRef = &anterior
	if _, e = m.Canonico(); e != nil {
		t.Fatalf("revision con recibo anterior rechazada: %v", e)
	}
	m.ClaveIdempotencia = "clave:libre"
	if _, e = m.Canonico(); e == nil {
		t.Fatal("clave no UUID admitida")
	}
}
