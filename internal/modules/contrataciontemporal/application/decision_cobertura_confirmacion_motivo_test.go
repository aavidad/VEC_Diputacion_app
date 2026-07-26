package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestDecidirCoberturaAlternativaExigeYRevalidaMotivoGobernado(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCoberturaConVias(t, false, 2)
	resolutor := nuevoResolutorMotivoConfirmacionPrueba(
		t,
		"motivos_confirmacion_cobertura",
		escenario.base.global.entorno.reloj.Ahora(),
	)
	motivos := &resolutorSecuenciaMotivoConfirmacionPrueba{
		resolutores: []*cobertura.ResolutorMotivoDecisionCobertura{
			resolutor,
		},
	}
	escenario.servicio.motivos = motivos
	escenario.solicitud.MotivoClave = claveMotivoConfirmacionPrueba
	escenario.solicitud.ViaElegida = "via_global_02"

	recibo, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, denegada := recibo.ResultadoDenegadoVEC(); !denegada ||
		motivos.total() != 2 ||
		escenario.vec.total() != 1 ||
		escenario.transaccion.total() != 1 {
		t.Fatalf("motivo alternativo no llegó ligado a Tx: %+v", recibo)
	}
}

func TestDecidirCoberturaClasificaFalloYSalidaInvalidaDelMotivo(
	t *testing.T,
) {
	t.Run("dependencia", func(t *testing.T) {
		escenario := nuevoEscenarioConfirmacionCobertura(t, false)
		escenario.servicio.motivos =
			&resolutorSecuenciaMotivoConfirmacionPrueba{
				err: errors.New("catálogo de motivos no disponible"),
			}
		escenario.solicitud.MotivoClave = claveMotivoConfirmacionPrueba
		_, err := escenario.servicio.Decidir(
			context.Background(),
			escenario.solicitud,
		)
		if !errors.Is(
			err,
			ErrConfirmacionDecisionCoberturaNoDisponible,
		) || escenario.sellador.total() != 0 {
			t.Fatalf("fallo del catálogo mal clasificado: %v", err)
		}
	})
	t.Run("salida inválida", func(t *testing.T) {
		escenario := nuevoEscenarioConfirmacionCobertura(t, false)
		escenario.servicio.motivos =
			&resolutorSecuenciaMotivoConfirmacionPrueba{invalida: true}
		escenario.solicitud.MotivoClave = claveMotivoConfirmacionPrueba
		_, err := escenario.servicio.Decidir(
			context.Background(),
			escenario.solicitud,
		)
		if !errors.Is(
			err,
			ErrConfirmacionDecisionCoberturaNoConfiable,
		) || escenario.sellador.total() != 0 {
			t.Fatalf("motivo no confiable se aceptó: %v", err)
		}
	})
}

func TestDecidirCoberturaSegundaLecturaMotivoNoSeAplastaAConflicto(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCoberturaConVias(t, false, 2)
	resolutor := nuevoResolutorMotivoConfirmacionPrueba(
		t,
		"motivos_confirmacion_segunda_lectura",
		escenario.base.global.entorno.reloj.Ahora(),
	)
	motivos := &resolutorSecuenciaMotivoConfirmacionPrueba{
		resolutores: []*cobertura.ResolutorMotivoDecisionCobertura{resolutor},
		errores: map[int]error{
			2: errors.New("catálogo no disponible en segunda lectura"),
		},
	}
	escenario.servicio.motivos = motivos
	escenario.solicitud.MotivoClave = claveMotivoConfirmacionPrueba
	escenario.solicitud.ViaElegida = "via_global_02"
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaNoDisponible) ||
		motivos.total() != 2 ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("fallo tardío del catálogo perdió su clase: %v", err)
	}
}

func TestDecidirCoberturaDetectaDerivaValidaEntreLecturasDelMotivo(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCoberturaConVias(t, false, 2)
	instante := escenario.base.global.entorno.reloj.Ahora()
	primero := nuevoResolutorMotivoConfirmacionPrueba(
		t,
		"motivos_confirmacion_primarios",
		instante,
	)
	segundo := nuevoResolutorMotivoConfirmacionPrueba(
		t,
		"motivos_confirmacion_sustitutos",
		instante,
	)
	motivos := &resolutorSecuenciaMotivoConfirmacionPrueba{
		resolutores: []*cobertura.ResolutorMotivoDecisionCobertura{
			primero,
			segundo,
		},
	}
	escenario.servicio.motivos = motivos
	escenario.solicitud.MotivoClave = claveMotivoConfirmacionPrueba
	escenario.solicitud.ViaElegida = "via_global_02"
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(err, ErrConfirmacionDecisionCoberturaEnConflicto) ||
		motivos.total() != 2 ||
		escenario.vec.total() != 0 ||
		escenario.transaccion.total() != 0 {
		t.Fatalf("deriva válida del motivo no cerró antes de VEC: %v", err)
	}
}
