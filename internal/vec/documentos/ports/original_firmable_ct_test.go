package ports

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

func reservaOriginalPrueba(t *testing.T) ReservaOriginalFirmable {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	s, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(ref("1"), ref("2"), ref("3"), ref("4"), ref("5"), 1,
		bytes.Repeat([]byte{0x6a}, 32), ref("6"), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	p, err := vecports.NuevaPoliticaConservacionDocumental(s, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, "", vecports.EstadoPoliticaConservacionDocumentalAprobada, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := vecports.NuevoResultadoPoliticaConservacionDocumental(p, s, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return ReservaOriginalFirmable{ID: ref("8"), ClaveIdempotencia: ref("9"), ModuloID: "contrataciontemporal",
		ExpedienteRef: s.ExpedienteRef(), TipoRef: s.TipoDocumentalRef(), Version: 1,
		MIME: "application/pdf", HuellaSHA256: strings.Repeat("a", 64), Tamano: 17, Politica: resultado}
}

func TestOriginalFirmablePreimagenFijaBytesTipoYPolitica(t *testing.T) {
	r := reservaOriginalPrueba(t)
	base, err := r.Preimagen()
	if err != nil || !bytes.Contains(base, []byte(`"mime":"application/pdf"`)) ||
		!bytes.Contains(base, []byte(`"huella_sha256":"`+r.HuellaSHA256+`"`)) ||
		!bytes.Contains(base, []byte(`"politica_ref":"`)) {
		t.Fatalf("preimagen incompleta: %s %v", base, err)
	}
	alterada := r
	alterada.HuellaSHA256 = strings.Repeat("b", 64)
	otra, _ := alterada.Preimagen()
	if HuellaEfectoV3(base) == HuellaEfectoV3(otra) {
		t.Fatal("bytes distintos conservan el mismo efecto")
	}
	alterada = r
	alterada.MIME = "text/plain"
	if _, err := alterada.Preimagen(); !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("otro MIME: %v", err)
	}
}

func TestIntentoOriginalFirmableRechazaOtraClaveYHuella(t *testing.T) {
	r := reservaOriginalPrueba(t)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	i := IntentoOriginalFirmable{ReservaRef: ref("a"), Estado: "pendiente", DocumentoID: r.ID,
		HuellaSHA256: r.HuellaSHA256, Numero: 1, ClaveAlmacenRef: ref("b")}
	if err := i.ValidarContra(r); err != nil {
		t.Fatal(err)
	}
	i.HuellaSHA256 = strings.Repeat("c", 64)
	if !errors.Is(i.ValidarContra(r), ErrCapacidadNoDisponible) {
		t.Fatal("intento de otros bytes aceptado")
	}
	i.HuellaSHA256 = r.HuellaSHA256
	i.ClaveAlmacenRef = ""
	if !errors.Is(i.ValidarContra(r), ErrCapacidadNoDisponible) {
		t.Fatal("intento sin clave durable aceptado")
	}
}

// Tras un fallo de confirmación SQL, el repositorio puede entregar otro
// intento pendiente para la misma reserva. Cada confirmación compromete su
// número y clave: el objeto del primer intento no confirma el segundo.
func TestConfirmacionOriginalFirmableLigaClaveNuevaTrasFalloSQL(t *testing.T) {
	r := reservaOriginalPrueba(t)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	primero := IntentoOriginalFirmable{ReservaRef: ref("a"), Estado: "pendiente", DocumentoID: r.ID,
		HuellaSHA256: r.HuellaSHA256, Numero: 1, ClaveAlmacenRef: ref("b")}
	segundo := primero
	segundo.Numero, segundo.ClaveAlmacenRef = 2, ref("c")
	objeto := ObjetoOriginalFirmable{ClaveAlmacenRef: primero.ClaveAlmacenRef, HuellaSHA256: r.HuellaSHA256,
		ObjetoRef: "objeto:primero", ObjetoVersion: "1", ConectorRef: "ensayo.local",
		ReciboObjetoRef: "recibo:primero", ReciboObjetoHuellaSHA256: strings.Repeat("d", 64),
		MIME: "application/pdf", Tamano: r.Tamano}
	preimagenPrimera, err := (ConfirmacionOriginalFirmable{Intento: primero, Objeto: objeto}).Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (ConfirmacionOriginalFirmable{Intento: segundo, Objeto: objeto}).Preimagen(); !errors.Is(err, ErrSolicitudInvalida) {
		t.Fatalf("objeto del intento anterior confirmado por el siguiente: %v", err)
	}
	objeto.ClaveAlmacenRef = segundo.ClaveAlmacenRef
	preimagenSegunda, err := (ConfirmacionOriginalFirmable{Intento: segundo, Objeto: objeto}).Preimagen()
	if err != nil || HuellaEfectoV3(preimagenPrimera) == HuellaEfectoV3(preimagenSegunda) {
		t.Fatalf("intentos distintos conservan efecto V3: %v", err)
	}
}
