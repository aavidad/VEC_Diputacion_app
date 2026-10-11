package documental

import (
	"bytes"
	"testing"
)

func TestPreimagenReservaCargaV3FijaOrdenYCambioDeVersion(t *testing.T) {
	d := ReservaCargaV3{Accion: "documentos.carga.preparar_reserva", CargaID: "carga:1",
		RecursoRef: "ref:1", PersonaRef: "per_1", ClaveReserva: "clave:1",
		HuellaReservaSHA256: "hr", VersionReserva: 2, IndiceCargaHMAC: "i",
		HuellaCargaHMAC: "hc", MIME: "application/pdf", Tamano: 7,
		HuellaContenido: "sha", HuellaToken: "ht"}
	primera, err := d.Preimagen()
	if err != nil {
		t.Fatal(err)
	}
	const esperado = `{"accion":"documentos.carga.preparar_reserva","carga_id":"carga:1","recurso_ref":"ref:1","persona_ref":"per_1","clave_reserva":"clave:1","huella_reserva_sha256":"hr","version_reserva":2,"indice_carga_hmac":"i","huella_carga_hmac":"hc","mime":"application/pdf","tamano":7,"huella_contenido_sha256":"sha","huella_token_sha256":"ht"}`
	if string(primera) != esperado {
		t.Fatalf("bytes distintos: %s", primera)
	}
	segunda, err := d.Preimagen()
	if err != nil || !bytes.Equal(primera, segunda) {
		t.Fatal("canon no determinista")
	}
	d.VersionReserva++
	otra, err := d.Preimagen()
	if err != nil || bytes.Equal(primera, otra) || HuellaEfectoCargaV3(primera) == HuellaEfectoCargaV3(otra) {
		t.Fatal("version D16 no cambia la identidad del efecto")
	}
}
