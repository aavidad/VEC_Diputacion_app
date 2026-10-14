package postgres

import (
	"strings"
	"testing"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestSubidaLocalRepositorioRechazaPostimagenIncompleta(t *testing.T) {
	base := docports.InstantaneaSubidaLocal{
		Resultado: docports.ResultadoSubidaLocal{CargaID: "ref:" + strings.Repeat("1", 64),
			Estado: docports.EstadoSubidaLocalReservada},
		RecursoRef: "ref:" + strings.Repeat("2", 64), PersonaRef: "per_0123456789abcdefghijkl",
		CorrelacionRef: "correlacion_" + strings.Repeat("3", 32),
		Clasificacion: "confidencial", HuellaSolicitudHMAC: "hmac-sha256:local_v1:" + strings.Repeat("4", 64),
		HuellaPDFSHA256: strings.Repeat("5", 64), MIME: "application/pdf", Tamano: 1234, Version: 1,
	}
	if !instantaneaSubidaLocalValida(base) {
		t.Fatal("reserva íntegra rechazada")
	}
	conComprobante := base
	conComprobante.Resultado.ComprobanteRef = "ref:" + strings.Repeat("6", 64)
	if instantaneaSubidaLocalValida(conComprobante) {
		t.Fatal("se publicó comprobante antes del análisis")
	}
	versionAlterada := base
	versionAlterada.Version = 4
	if instantaneaSubidaLocalValida(versionAlterada) {
		t.Fatal("se aceptó versión terminal con estado reservado")
	}
	rechazoSinRecibo := base
	rechazoSinRecibo.Version = 4
	rechazoSinRecibo.Resultado.Estado = docports.EstadoSubidaLocalRechazada
	rechazoSinRecibo.Resultado.CodigoRechazo = "contenido_no_permitido"
	if instantaneaSubidaLocalValida(rechazoSinRecibo) {
		t.Fatal("rechazo sin recibo ni análisis aceptado")
	}
}
