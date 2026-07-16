package postgres

import (
	"context"
	"testing"
	"time"

	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type verificadorFinalidadPostgreSQLPrueba struct {
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion
}

func (v *verificadorFinalidadPostgreSQLPrueba) VerificarSelloBaremacion(
	_ context.Context, solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	v.solicitud = solicitud
	return nil
}

func TestConfirmacionPostgreSQLUsaFinalidadHMACV2(t *testing.T) {
	token, err := transaccionbolsa.GenerarTokenReserva()
	if err != nil {
		t.Fatal(err)
	}
	baremacion := baremacionPostgreSQLPrueba(t)
	solicitud := puertosbolsa.SolicitudConfirmarCambioBaremacion{
		Contexto: contextoPostgreSQLBaremacionPrueba(
			t, puertosbolsa.AccionConfirmarAltaBaremacion, baremacion.ID,
		),
		Token: token, Clase: puertosbolsa.ClaseCambioAltaBaremacion,
		HuellaSolicitudHMAC: hmacPostgreSQLBaremacionPrueba("8"),
		Agregado:            baremacion,
		Trazabilidad: puertosbolsa.TrazabilidadCambioBaremacion{
			MotivoClave: "alta_autobaremacion",
			Motivo:      "Alta de la autobaremacion calculada oficialmente.",
		},
		ConfirmadaEn: instantePostgreSQLPrueba.Add(-30 * time.Second),
	}
	verificador := &verificadorFinalidadPostgreSQLPrueba{}
	repositorio := &RepositorioBaremaciones{verificador: verificador}
	if err = repositorio.verificarSelloConfirmacion(context.Background(), solicitud); err != nil {
		t.Fatal(err)
	}
	if verificador.solicitud.Finalidad != puertosbolsa.FinalidadSelloConfirmacionBaremacionV2 {
		t.Fatalf("finalidad HMAC inesperada: %s", verificador.solicitud.Finalidad)
	}
}
