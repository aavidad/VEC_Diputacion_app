package ports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestContextoSeCompruebaTrasFuenteYVerificadores(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultado := resultadoRCFirmadoPrueba(
		t, solicitud, validacion, MotivoFuenteAnalisis{}, metadatos,
	)
	t.Run("fuente", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		verificadorLlamado := false
		_, err := ValidarRCConFuente(
			ctx,
			fuentePresupuestariaDoble(func(
				context.Context,
				SolicitudValidarRC,
			) (ResultadoValidacionRC, error) {
				cancelar()
				return resultado, nil
			}),
			verificadorRespuestaDoble(func(
				context.Context,
				SolicitudVerificarRespuestaFuenteAnalisis,
			) (ConfirmacionRespuestaFuenteAnalisis, error) {
				verificadorLlamado = true
				return ConfirmacionRespuestaFuenteAnalisis{}, nil
			}),
			verificadorPublicacionNoInvocablePrueba(t),
			consumidorRespuestaPrueba(metadatos.EmitidaEn.Add(time.Second)),
			relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
			solicitud,
		)
		if !errors.Is(err, context.Canceled) || verificadorLlamado {
			t.Fatalf("no cortó tras fuente: %v", err)
		}
	})

	t.Run("verificador de respuesta", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		consumidorLlamado := false
		_, err := ValidarRCConFuente(
			ctx,
			fuentePresupuestariaDoble(func(
				context.Context,
				SolicitudValidarRC,
			) (ResultadoValidacionRC, error) {
				return resultado, nil
			}),
			verificadorRespuestaDoble(func(
				context.Context,
				SolicitudVerificarRespuestaFuenteAnalisis,
			) (ConfirmacionRespuestaFuenteAnalisis, error) {
				cancelar()
				return ConfirmacionRespuestaFuenteAnalisis{}, nil
			}),
			verificadorPublicacionNoInvocablePrueba(t),
			consumidorRespuestaDoble(func(
				context.Context,
				OrdenConsumoRespuestaFuenteAnalisis,
			) (ReciboConsumoRespuestaFuenteAnalisis, error) {
				consumidorLlamado = true
				return ReciboConsumoRespuestaFuenteAnalisis{}, nil
			}),
			relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
			solicitud,
		)
		if !errors.Is(err, context.Canceled) || consumidorLlamado {
			t.Fatalf("no cortó tras verificador: %v", err)
		}
	})
}

func TestCancelacionTrasConsumoDurableConfirmadoNoCreaExitoAmbiguo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	ctx, cancelar := context.WithCancel(context.Background())
	consumidor := consumidorRespuestaDoble(func(
		_ context.Context,
		orden OrdenConsumoRespuestaFuenteAnalisis,
	) (ReciboConsumoRespuestaFuenteAnalisis, error) {
		recibo, err := NuevoReciboConsumoRespuestaFuenteAnalisis(
			orden,
			"consumo_durable_respuesta_0123456789",
			metadatos.EmitidaEn.Add(time.Second),
		)
		cancelar()
		return recibo, err
	})
	if _, err := CalcularCosteConFuente(
		ctx,
		calculadorCosteDoble(func(
			context.Context,
			SolicitudCalcularCoste,
		) (ResultadoCalculoCoste, error) {
			return resultado, nil
		}),
		verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
		consumidor,
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
		solicitud,
	); err != nil {
		t.Fatalf("un consumo confirmado se volvió ambiguo: %v", err)
	}
}

func TestFuenteRecibeTimeoutMaximoPropio(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	calculador := calculadorCosteDoble(func(
		ctx context.Context,
		_ SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		limite, existe := ctx.Deadline()
		restante := time.Until(limite)
		if !existe || restante <= 0 ||
			restante > TiempoMaximoFuenteAnalisis+time.Second {
			t.Fatalf("timeout propio ausente: %s", restante)
		}
		return resultado, nil
	})
	if _, err := CalcularCosteConFuente(
		context.Background(),
		calculador,
		verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
		consumidorRespuestaPrueba(metadatos.EmitidaEn.Add(time.Second)),
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(time.Second)),
		solicitud,
	); err != nil {
		t.Fatal(err)
	}
}

func TestSolicitudesDetectanAdulteracionInterna(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	copiaRC := solicitudRC
	datosRC := *solicitudRC.datos
	copiaRC.datos = &datosRC
	copiaRC.datos.VersionExpediente++
	if copiaRC.Validar() == nil {
		t.Fatal("solicitud RC alterada aceptada")
	}
	copiaRC = solicitudRC
	datosRC = *solicitudRC.datos
	copiaRC.datos = &datosRC
	copiaRC.datos.HuellaPeticionHMAC =
		dominioSelloPeticionAnalisis + strings.Repeat("c", 64)
	if copiaRC.Validar() == nil {
		t.Fatal("solicitud RC con sello sustituido aceptada")
	}
}

func TestDependenciasNulasTipadasFallanCerrado(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	var fuente *fuentePresupuestariaNulaPrueba
	if _, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		verificadorRespuestaHMACPrueba(inicio),
		verificadorPublicacionNoInvocablePrueba(t),
		consumidorRespuestaPrueba(inicio),
		relojFijoFuenteAnalisis(inicio),
		solicitudValidarRCPrueba(t, inicio),
	); !errors.Is(err, ErrPeticionFuenteAnalisisInvalida) {
		t.Fatalf("fuente nula tipada aceptada: %v", err)
	}
}

type fuentePresupuestariaNulaPrueba struct{}

func (*fuentePresupuestariaNulaPrueba) ValidarRC(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	panic("no debe invocarse")
}
