package httpinterno

import (
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCoberturaSinCreditoExplicaElMotivoConCodigoPropio(t *testing.T) {
	pruebas := map[domain.MotivoSinCredito]string{
		domain.SinCreditoAnalisisPendiente:     "sin_credito_analisis_pendiente",
		domain.SinCreditoRetencionRechazada:    "sin_credito_retencion_rechazada",
		domain.SinCreditoPartidasSinCoste:      "sin_credito_partidas_sin_coste",
		domain.SinCreditoPartidasNoAcreditadas: "sin_credito_partidas_no_acreditadas",
	}
	for motivo, codigo := range pruebas {
		for _, publico := range []error{
			application.ErrPresentacionPropuestaCoberturaEstadoNoAdmite,
			application.ErrConfirmacionDecisionCoberturaEnConflicto,
		} {
			obtenido := clasificarErrorCobertura(errors.Join(publico, domain.NuevoErrorSinCredito(motivo)))
			if obtenido.estado != http.StatusConflict || obtenido.codigo != codigo ||
				obtenido.claveI18n != "api.contratacion_temporal.cobertura.error."+codigo {
				t.Fatalf("%s con %v: %+v", motivo, publico, obtenido)
			}
		}
	}
	// Sin motivo de crédito, el conflicto sigue siendo el genérico de siempre.
	if obtenido := clasificarErrorCobertura(application.ErrConfirmacionDecisionCoberturaEnConflicto); obtenido.codigo != "conflicto" {
		t.Fatalf("conflicto genérico: %+v", obtenido)
	}
	// Un motivo desconocido no inventa un código.
	if obtenido := clasificarErrorCobertura(errors.Join(application.ErrConfirmacionDecisionCoberturaEnConflicto,
		domain.NuevoErrorSinCredito("inventado"))); obtenido.codigo != "conflicto" {
		t.Fatalf("motivo desconocido: %+v", obtenido)
	}
}
