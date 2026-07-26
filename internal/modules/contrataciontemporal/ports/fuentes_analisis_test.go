package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

type generadorPeticionAnalisisDoble func(
	context.Context,
	TipoPeticionFuenteAnalisis,
) (string, error)

func (g generadorPeticionAnalisisDoble) NuevaReferenciaPeticionFuenteAnalisis(
	ctx context.Context,
	tipo TipoPeticionFuenteAnalisis,
) (string, error) {
	return g(ctx, tipo)
}

type selladorPeticionAnalisisDoble func(
	context.Context,
	PreimagenPeticionFuenteAnalisis,
) (string, error)

func (s selladorPeticionAnalisisDoble) SellarPeticionFuenteAnalisis(
	ctx context.Context,
	preimagen PreimagenPeticionFuenteAnalisis,
) (string, error) {
	return s(ctx, preimagen)
}

type relojFuenteAnalisisDoble func() time.Time

func (r relojFuenteAnalisisDoble) Ahora() time.Time { return r() }

type fuentePresupuestariaDoble func(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error)

func (f fuentePresupuestariaDoble) ValidarRC(
	ctx context.Context,
	solicitud SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	return f(ctx, solicitud)
}

func (f fuentePresupuestariaDoble) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionAutoridadPrueba(
		RolFuentePresupuestaria,
		"fuente_presupuesto_0123456789",
		"backend_presupuesto_0123456789",
		desafio,
	)
}

type calculadorCosteDoble func(
	context.Context,
	SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error)

func (c calculadorCosteDoble) CalcularCoste(
	ctx context.Context,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	return c(ctx, solicitud)
}

func (c calculadorCosteDoble) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionAutoridadPrueba(
		RolCalculadorCoste,
		"tabla_retributiva_2026_v3",
		"backend_calculo_coste_0123456789",
		desafio,
	)
}

type verificadorRespuestaDoble func(
	context.Context,
	SolicitudVerificarRespuestaFuenteAnalisis,
) (ConfirmacionRespuestaFuenteAnalisis, error)

func (v verificadorRespuestaDoble) VerificarRespuestaFuenteAnalisis(
	ctx context.Context,
	solicitud SolicitudVerificarRespuestaFuenteAnalisis,
) (ConfirmacionRespuestaFuenteAnalisis, error) {
	return v(ctx, solicitud)
}

func (v verificadorRespuestaDoble) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionAutoridadPrueba(
		RolVerificadorRespuesta,
		"verificador_tcb_presupuestario_012345",
		"backend_verificador_tcb_0123456789",
		desafio,
	)
}

type verificadorPublicacionMotivoDoble func(
	context.Context,
	SolicitudVerificarPublicacionMotivoFuenteAnalisis,
) (ConfirmacionPublicacionMotivoFuenteAnalisis, error)

func (v verificadorPublicacionMotivoDoble) VerificarPublicacionMotivoFuenteAnalisis(
	ctx context.Context,
	solicitud SolicitudVerificarPublicacionMotivoFuenteAnalisis,
) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	return v(ctx, solicitud)
}

func (v verificadorPublicacionMotivoDoble) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return presentacionAutoridadPrueba(
		RolPublicadorCatalogo,
		"publicador_catalogo_motivos_012345",
		"backend_publicador_catalogo_012345",
		desafio,
	)
}

type consumidorRespuestaDoble func(
	context.Context,
	OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error)

func (c consumidorRespuestaDoble) ConsumirRespuestaFuenteAnalisis(
	ctx context.Context,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	return c(ctx, orden)
}

func TestAyudaLegacyValidarRCExigeAtestacionVerificadaYConsumo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultado := resultadoRCFirmadoPrueba(
		t,
		solicitud,
		validacion,
		MotivoFuenteAnalisis{},
		metadatos,
	)
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultado, nil
	})
	obtenida, err := validarRCConFuenteOrquestadaPrueba(
		context.Background(),
		fuente,
		verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
		verificadorPublicacionNoInvocablePrueba(t),
		consumidorRespuestaPrueba(metadatos.EmitidaEn.Add(time.Second)),
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(1500*time.Millisecond)),
		solicitud,
	)
	if err != nil || obtenida.Validar() != nil ||
		obtenida.Importe == validacion.Importe ||
		obtenida.FechaRC == validacion.FechaRC {
		t.Fatalf("validación RC no confiable: %#v, %v", obtenida, err)
	}
}

func TestAyudaLegacyValidarRCConservaMotivoPublicadoCompletoEnConsumo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCNegativaPrueba(t, solicitud, inicio.Add(time.Second))
	motivo := motivoFuenteAnalisisPrueba(t)
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultado := resultadoRCFirmadoPrueba(t, solicitud, validacion, motivo, metadatos)
	vinculoEsperado, _ := motivo.Datos()
	consumidor := consumidorRespuestaDoble(func(
		_ context.Context,
		orden OrdenConsumoRespuestaFuenteAnalisis,
	) (ReciboConsumoRespuestaFuenteAnalisis, error) {
		datos, err := orden.Datos()
		vinculo, errMotivo := datos.Motivo.Datos()
		if err != nil || errMotivo != nil ||
			datos.ConfirmacionPublicacion == nil ||
			vinculo.CatalogoRef != vinculoEsperado.CatalogoRef ||
			vinculo.CatalogoVersion != vinculoEsperado.CatalogoVersion ||
			vinculo.CatalogoHuella != vinculoEsperado.CatalogoHuella ||
			vinculo.EntradaClave != vinculoEsperado.EntradaClave ||
			vinculo.ClaveMensajeI18N != vinculoEsperado.ClaveMensajeI18N {
			t.Fatal("el consumo perdió el vínculo publicado del motivo")
		}
		return NuevoReciboConsumoRespuestaFuenteAnalisis(
			orden,
			"consumo_respuesta_rc_0123456789",
			metadatos.EmitidaEn.Add(time.Second),
		)
	})
	obtenida, err := validarRCConFuenteOrquestadaPrueba(
		context.Background(),
		fuentePresupuestariaDoble(func(
			context.Context,
			SolicitudValidarRC,
		) (ResultadoValidacionRC, error) {
			return resultado, nil
		}),
		verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
		verificadorPublicacionPrueba(metadatos.EmitidaEn.Add(250*time.Millisecond)),
		consumidor,
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(1500*time.Millisecond)),
		solicitud,
	)
	if err != nil ||
		obtenida.Motivo != "contratacion_temporal.rc.no_requerida" ||
		!obtenida.HabilitaAvance() {
		t.Fatalf("motivo gobernado no materializado: %#v, %v", obtenida, err)
	}
}

func TestAyudaLegacyCalcularCosteVerificaYConsumeRespuesta(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	fuenteRef := "tabla_retributiva_2026_v3"
	metadatos := metadatosRespuestaPrueba(
		fuenteRef,
		"recibo_coste_0123456789",
		inicio,
	)
	resultado := resultadoCosteFirmadoPrueba(t, solicitud, metadatos)
	obtenido, err := calcularCosteConFuenteOrquestadoPrueba(
		context.Background(),
		calculadorCosteDoble(func(
			context.Context,
			SolicitudCalcularCoste,
		) (ResultadoCalculoCoste, error) {
			return resultado, nil
		}),
		verificadorRespuestaHMACPrueba(metadatos.EmitidaEn.Add(500*time.Millisecond)),
		consumidorRespuestaPrueba(metadatos.EmitidaEn.Add(time.Second)),
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(metadatos.EmitidaEn.Add(1500*time.Millisecond)),
		solicitud,
	)
	datos, errDatos := obtenido.Datos()
	if err != nil || errDatos != nil || datos.Importe.Centimos != 3_148_025 {
		t.Fatalf("coste no verificado: %#v, %v, %v", datos, err, errDatos)
	}
}

func TestErroresDeProveedorNoExponenLaCausa(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	privado := errors.New("DNI 12345678Z en backend presupuestario")
	_, err := validarRCConFuenteOrquestadaPrueba(
		context.Background(),
		fuentePresupuestariaDoble(func(
			context.Context,
			SolicitudValidarRC,
		) (ResultadoValidacionRC, error) {
			return ResultadoValidacionRC{}, privado
		}),
		verificadorRespuestaHMACPrueba(inicio),
		verificadorPublicacionNoInvocablePrueba(t),
		consumidorRespuestaPrueba(inicio),
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(inicio),
		solicitud,
	)
	if !errors.Is(err, ErrFuentePresupuestariaNoDisponible) ||
		errors.Is(err, privado) ||
		err.Error() != ErrFuentePresupuestariaNoDisponible.Error() ||
		strings.Contains(err.Error(), "12345678") {
		t.Fatalf("causa privada expuesta: %v", err)
	}
}

func solicitudValidarRCPrueba(t *testing.T, instante time.Time) SolicitudValidarRC {
	t.Helper()
	solicitud, err := nuevaSolicitudValidarRCOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacionValidarRCPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func solicitudCalcularCostePrueba(
	t *testing.T,
	instante time.Time,
) SolicitudCalcularCoste {
	t.Helper()
	solicitud, err := nuevaSolicitudCalcularCosteOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_abcdefghij0123456789klmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacionCalcularCostePrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func generadorFijoFuenteAnalisis(referencia string) generadorPeticionAnalisisDoble {
	return func(context.Context, TipoPeticionFuenteAnalisis) (string, error) {
		return referencia, nil
	}
}

func selladorHMACFuenteAnalisisPrueba() selladorPeticionAnalisisDoble {
	return func(
		_ context.Context,
		preimagen PreimagenPeticionFuenteAnalisis,
	) (string, error) {
		contenido, err := preimagen.Bytes()
		if err != nil {
			return "", err
		}
		mac := hmac.New(sha256.New, []byte("clave-prueba-fuente-analisis"))
		_, _ = mac.Write(contenido)
		return dominioSelloPeticionAnalisis + hex.EncodeToString(mac.Sum(nil)), nil
	}
}

func relojFijoFuenteAnalisis(instante time.Time) relojFuenteAnalisisDoble {
	return func() time.Time { return instante }
}

func instanteFuenteAnalisisPrueba() time.Time {
	return time.Date(2026, 7, 23, 9, 0, 0, 123_456_000, time.UTC)
}
