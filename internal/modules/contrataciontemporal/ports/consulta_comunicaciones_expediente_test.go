package ports

import (
	"strings"
	"testing"
	"time"
)

func TestPaginaComunicacionesExpedienteOrdenYCampos(t *testing.T) {
	c := ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:ct140-uno", Limite: 2}
	fecha := time.Date(2026, 9, 28, 12, 0, 0, 123456000, time.UTC)
	a := ComunicacionExpediente{
		OrganizacionRef: "organizacion:ct140-uno", ExpedienteRef: c.ExpedienteRef,
		LlamamientoRef: "llamamiento:ct140-uno", ComunicacionRef: "comunicacion:ct140-a",
		Version: 2, Estado: "registrada_localmente", RegistradaEn: fecha,
		ReciboComunicacionRef: "recibo:ct140-a", AntecedenteTipo: "seleccion_confirmada",
		ReciboAntecedenteRef: "recibo:seleccion-a",
	}
	b := a
	b.LlamamientoRef = "llamamiento:ct140-dos"
	b.ComunicacionRef = "comunicacion:ct140-b"
	b.ReciboComunicacionRef = "recibo:ct140-b"
	b.AntecedenteTipo = "continuacion_confirmada"
	b.ReciboAntecedenteRef = "recibo:continuacion-b"
	p := PaginaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef, Comunicaciones: []ComunicacionExpediente{a, b}, SiguienteCursor: b.ComunicacionRef + "#" + strings.Repeat("a", 64)}
	if err := p.ValidarPara(c); err != nil {
		t.Fatalf("pagina valida: %v", err)
	}
	p.Comunicaciones[1].ReciboAntecedenteRef = ""
	if err := p.ValidarPara(c); err == nil {
		t.Fatal("aceptó antecedente ausente")
	}
	p.Comunicaciones[1] = b
	p.Comunicaciones[1].ComunicacionRef = a.ComunicacionRef
	if err := p.ValidarPara(c); err == nil {
		t.Fatal("aceptó orden no estricto")
	}
}
