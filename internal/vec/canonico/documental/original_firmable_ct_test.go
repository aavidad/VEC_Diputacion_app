package documental

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"
)

func datosReservaOriginalPrueba() DatosReservaOriginal {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	return DatosReservaOriginal{
		ID: ref("8"), ClaveIdempotencia: ref("9"), ModuloID: "contratacion_temporal",
		ExpedienteRef: ref("1"), TipoRef: ref("2"), Version: 7,
		MIME: "application/pdf", HuellaSHA256: strings.Repeat("a", 64), Tamano: 17,
		PoliticaRef: ref("3"), VersionPolitica: 1, HuellaPoliticaSHA256: strings.Repeat("b", 64),
		Proteccion: "conservacion", EstadoPolitica: "aprobada",
		ConservacionHasta: time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestOriginalFirmablePreimagenFijaBytesTipoYPolitica(t *testing.T) {
	r := datosReservaOriginalPrueba()
	base, err := r.Preimagen()
	if err != nil || !bytes.Contains(base, []byte(`"mime":"application/pdf"`)) ||
		!bytes.Contains(base, []byte(`"huella_sha256":"`+r.HuellaSHA256+`"`)) ||
		!bytes.Contains(base, []byte(`"politica_ref":"`)) {
		t.Fatalf("preimagen incompleta: %s %v", base, err)
	}
	alterada := r
	alterada.HuellaSHA256 = strings.Repeat("c", 64)
	otra, _ := alterada.Preimagen()
	if sha256.Sum256(base) == sha256.Sum256(otra) {
		t.Fatal("bytes distintos conservan el mismo efecto")
	}
	alterada = r
	alterada.MIME = "text/plain"
	if _, err := alterada.Preimagen(); !errors.Is(err, ErrOriginalFirmableInvalido) {
		t.Fatalf("otro MIME: %v", err)
	}
}

func TestIntentoOriginalFirmableRechazaOtraClaveYHuella(t *testing.T) {
	r := datosReservaOriginalPrueba()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	i := IntentoOriginalFirmable{ReservaRef: ref("d"), Estado: "pendiente", DocumentoID: r.ID,
		HuellaSHA256: r.HuellaSHA256, Numero: 1, ClaveAlmacenRef: ref("e")}
	if !i.ValidoContra(r) {
		t.Fatal("intento correcto rechazado")
	}
	i.HuellaSHA256 = strings.Repeat("f", 64)
	if i.ValidoContra(r) {
		t.Fatal("intento de otros bytes aceptado")
	}
	i.HuellaSHA256 = r.HuellaSHA256
	i.ClaveAlmacenRef = ""
	if i.ValidoContra(r) {
		t.Fatal("intento sin clave durable aceptado")
	}
}

func TestConfirmacionOriginalFirmableLigaClaveNuevaTrasFalloSQL(t *testing.T) {
	r := datosReservaOriginalPrueba()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	primero := IntentoOriginalFirmable{ReservaRef: ref("d"), Estado: "pendiente", DocumentoID: r.ID,
		HuellaSHA256: r.HuellaSHA256, Numero: 1, ClaveAlmacenRef: ref("e")}
	segundo := primero
	segundo.Numero, segundo.ClaveAlmacenRef = 2, ref("f")
	objeto := ObjetoOriginalFirmable{ClaveAlmacenRef: primero.ClaveAlmacenRef, HuellaSHA256: r.HuellaSHA256,
		ObjetoRef: "objeto:primero", ObjetoVersion: "1", ConectorRef: "ensayo.local",
		ReciboObjetoRef: "recibo:primero", ReciboObjetoHuellaSHA256: strings.Repeat("d", 64),
		MIME: "application/pdf", Tamano: r.Tamano}
	preimagenPrimera, err := (DatosConfirmacionOriginal{Intento: primero, Objeto: objeto}).Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (DatosConfirmacionOriginal{Intento: segundo, Objeto: objeto}).Preimagen(); !errors.Is(err, ErrOriginalFirmableInvalido) {
		t.Fatalf("objeto del intento anterior confirmado por el siguiente: %v", err)
	}
	objeto.ClaveAlmacenRef = segundo.ClaveAlmacenRef
	preimagenSegunda, err := (DatosConfirmacionOriginal{Intento: segundo, Objeto: objeto}).Preimagen()
	if err != nil || sha256.Sum256(preimagenPrimera) == sha256.Sum256(preimagenSegunda) {
		t.Fatalf("intentos distintos conservan efecto V3: %v", err)
	}
}
