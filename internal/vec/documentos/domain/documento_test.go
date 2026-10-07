package domain

import (
	"strings"
	"testing"
	"time"
)

func TestReferenciaExpedienteTipadaSoloEnCampoExpediente(t *testing.T) {
	id := strings.Repeat("a", 64)
	legacy := "ref:" + id
	tipada := "expediente:ct:" + id
	if !ReferenciaExpedienteValida(tipada) || !ReferenciaExpedienteModuloValida("contratacion_temporal", tipada) {
		t.Fatal("rechazo expediente CT del catálogo")
	}
	if !ReferenciaExpedienteValida(legacy) || !ReferenciaExpedienteModuloValida("dietas", legacy) {
		t.Fatal("rechazo referencia histórica")
	}
	if ReferenciaOpacaValida(tipada) {
		t.Fatal("el identificador opaco se ensanchó")
	}
	if ReferenciaExpedienteModuloValida("bolsa", tipada) {
		t.Fatal("aceptó módulo productor distinto")
	}
	for _, s := range []string{
		"expediente:bolsa:" + id,
		"expediente:CT:" + id,
		"Expediente:ct:" + id,
		"expediente:ct:" + strings.ToUpper(id),
		"expediente:ct:" + id[:63],
		"expediente:ct:" + id + "0",
		"expediente:ct:" + id + ":otro",
		"expediente:ct:../" + id,
	} {
		if ReferenciaExpedienteValida(s) {
			t.Errorf("aceptó referencia fuera del contrato %q", s)
		}
	}
}

func BenchmarkReferenciaExpedienteValida(b *testing.B) {
	s := "expediente:ct:" + strings.Repeat("a", 64)
	for i := 0; i < b.N; i++ {
		if !ReferenciaExpedienteValida(s) {
			b.Fatal("referencia válida rechazada")
		}
	}
}

func TestDocumentoSoloAceptaReferenciasOpacasYFirmaPendiente(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	d := Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-1", ModuloID: "dietas",
		ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 1,
		MIME: "application/pdf", HuellaSHA256: strings.Repeat("4", 64), Tamano: 3,
		ObjetoRef: "objeto:1", ObjetoVersion: "version:1", PoliticaRef: ref("5"),
		VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: time.Now().UTC().Add(time.Hour), Proteccion: "conservacion", EstadoPolitica: EstadoPoliticaAprobada,
		EstadoFirma: EstadoFirmaPendienteProveedor, CreadoEn: time.Now().UTC(), Custodia: CustodiaVEC,
	}
	if err := d.Validar(); err != nil {
		t.Fatalf("documento valido: %v", err)
	}
	d.ExpedienteRef = "DNI:12345678Z"
	if d.Validar() == nil {
		t.Fatal("admitio identificador personal en expediente")
	}
	d.ExpedienteRef = ref("2")
	d.EstadoFirma = "firmado_verificado"
	if d.Validar() == nil {
		t.Fatal("admitio firma sin transicion acreditada")
	}
	d.EstadoFirma = EstadoFirmaPendienteProveedor
	d.EstadoPolitica = EstadoPoliticaProvisional
	if d.Validar() != nil {
		t.Fatal("denego politica provisional ordinaria")
	}
	for _, estado := range []string{"", "retirada", "definitiva"} {
		d.EstadoPolitica = estado
		if d.Validar() == nil {
			t.Fatalf("admitio estado de politica fuera de catalogo %q", estado)
		}
	}
	d.EstadoPolitica, d.Proteccion = EstadoPoliticaProvisional, "bloqueo"
	if d.Validar() == nil {
		t.Fatal("admitio bloqueo con politica provisional")
	}
}

func TestDocumentoConCustodiaExternaNoLlevaObjetoYExigeReferenciaYHuella(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	huella := strings.Repeat("4", 64)
	d := Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-2", ModuloID: "dietas",
		ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 1,
		HuellaSHA256: huella, PoliticaRef: ref("5"),
		VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: time.Now().UTC().Add(time.Hour), Proteccion: "conservacion", EstadoPolitica: EstadoPoliticaAprobada,
		EstadoFirma: EstadoFirmaPendienteProveedor, CreadoEn: time.Now().UTC(), Custodia: CustodiaExterna,
		CustodiaExternaRef: ReferenciaCustodiaExterna{CustodioID: "dietas.justificantes", Referencia: "justificante:0001", HuellaSHA256: huella},
	}
	if err := d.Validar(); err != nil || d.Descargable() {
		t.Fatalf("referencia externa valida sin MIME ni tamano: err=%v descargable=%v", err, d.Descargable())
	}
	casos := map[string]func(*Documento){
		"objeto en custodia externa":   func(d *Documento) { d.ObjetoRef, d.ObjetoVersion = "objeto:1", "version:1" },
		"huella distinta del custodio": func(d *Documento) { d.CustodiaExternaRef.HuellaSHA256 = strings.Repeat("7", 64) },
		"referencia con ruta":          func(d *Documento) { d.CustodiaExternaRef.Referencia = "../etc/passwd" },
		"custodio sin identificador":   func(d *Documento) { d.CustodiaExternaRef.CustodioID = "" },
		"mime con parametros":          func(d *Documento) { d.MIME = "application/pdf; charset=x" },
		"tamano negativo":              func(d *Documento) { d.Tamano = -1 },
		"custodia desconocida":         func(d *Documento) { d.Custodia = "otra" },
		"custodia vacia":               func(d *Documento) { d.Custodia = "" },
	}
	for nombre, alterar := range casos {
		copia := d
		alterar(&copia)
		if copia.Validar() == nil {
			t.Errorf("%s: aceptado", nombre)
		}
	}
	v := d
	v.Custodia = CustodiaVEC
	if v.Validar() == nil {
		t.Fatal("custodia VEC sin objeto aceptada")
	}
}

// La versión del objeto la fija el conector: «1» en el de ficheros, el
// VersionId en S3. Antes se exigía la longitud mínima de una referencia y todo
// documento del almacén de ficheros resultaba inválido.
func TestVersionObjetoSigueElContratoDelAlmacen(t *testing.T) {
	for _, v := range []string{"1", "ov1", "3HL4kqtJlcpXroDTDmJ.rmSpXd3dIbrHY", strings.Repeat("v", 256)} {
		if !VersionObjetoValida(v) {
			t.Errorf("versión válida rechazada: %q", v)
		}
	}
	for _, v := range []string{"", " ", "1 2", "../1", "a/b", `a\b`, "v*", "v%", "ñ", strings.Repeat("v", 257)} {
		if VersionObjetoValida(v) {
			t.Errorf("versión inválida aceptada: %q", v)
		}
	}
}
