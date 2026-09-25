package domain

import (
	"strings"
	"testing"
	"time"
)

func TestDocumentoSoloAceptaReferenciasOpacasYFirmaPendiente(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	d := Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-1", ModuloID: "dietas",
		ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 1,
		MIME: "application/pdf", HuellaSHA256: strings.Repeat("4", 64), Tamano: 3,
		ObjetoRef: "objeto:1", ObjetoVersion: "version:1", PoliticaRef: ref("5"),
		VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: time.Now().UTC().Add(time.Hour), Proteccion: "conservacion",
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
}

func TestDocumentoConCustodiaExternaNoLlevaObjetoYExigeReferenciaYHuella(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	huella := strings.Repeat("4", 64)
	d := Documento{
		ID: ref("1"), NumeroVEC: "VEC-2026-2", ModuloID: "dietas",
		ExpedienteRef: ref("2"), TipoRef: ref("3"), Version: 1,
		HuellaSHA256: huella, PoliticaRef: ref("5"),
		VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("6", 64),
		ConservacionHasta: time.Now().UTC().Add(time.Hour), Proteccion: "conservacion",
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
