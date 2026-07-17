package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
)

func TestServicioRechazaCadaDesvinculacionDelReciboConsumoFuente(t *testing.T) {
	otraReferencia := func(referencia oficial.ReferenciaExactaV1) oficial.ReferenciaExactaV1 {
		referencia.Referencia += ":otra"
		return referencia
	}
	otraVersion := func(referencia oficial.ReferenciaExactaV1) oficial.ReferenciaExactaV1 {
		referencia.Version++
		return referencia
	}
	otraHuella := func(referencia oficial.ReferenciaExactaV1) oficial.ReferenciaExactaV1 {
		referencia.HuellaSHA256 = strings.Repeat("6", 64)
		return referencia
	}
	casos := []struct {
		nombre string
		mutar  func(*datosReciboConsumoFuenteDoble)
	}{
		{"decision_ref", func(d *datosReciboConsumoFuenteDoble) { d.decisionRef += ":otra" }},
		{"esquema_decision", func(d *datosReciboConsumoFuenteDoble) { d.esquemaDecision += ".otro" }},
		{"huella_decision", func(d *datosReciboConsumoFuenteDoble) { d.huellaDecision = strings.Repeat("1", 64) }},
		{"recurso_ref", func(d *datosReciboConsumoFuenteDoble) { d.recursoRef += ":otro" }},
		{"huella_contexto", func(d *datosReciboConsumoFuenteDoble) { d.huellaContexto = strings.Repeat("2", 64) }},
		{"correlacion_ref", func(d *datosReciboConsumoFuenteDoble) { d.correlacionRef += ":otra" }},
		{"huella_selector", func(d *datosReciboConsumoFuenteDoble) { d.huellaSelector = strings.Repeat("3", 64) }},
		{"fuente_referencia", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraReferencia(d.fuenteExacta) }},
		{"fuente_version", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraVersion(d.fuenteExacta) }},
		{"fuente_huella", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraHuella(d.fuenteExacta) }},
		{"prueba_referencia", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraReferencia(d.consumoPrueba) }},
		{"prueba_version", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraVersion(d.consumoPrueba) }},
		{"prueba_huella", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraHuella(d.consumoPrueba) }},
		{"auditoria_referencia", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraReferencia(d.auditoria) }},
		{"auditoria_version", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraVersion(d.auditoria) }},
		{"auditoria_huella", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraHuella(d.auditoria) }},
		{"consumo_antes_solicitud", func(d *datosReciboConsumoFuenteDoble) { d.consumidaEn = d.consumidaEn.Add(-time.Microsecond) }},
		{"consumo_despues_fuente", func(d *datosReciboConsumoFuenteDoble) { d.consumidaEn = d.consumidaEn.Add(time.Microsecond) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.fuente.alterarConsumo = caso.mutar
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, ErrFuenteNoConfiable) || escenario.fuente.llamadas != 1 ||
				len(escenario.exigidor.llamadas) != 1 || escenario.confirmador.llamadas != 0 {
				t.Fatalf("recibo desvinculado alcanzo confirmacion: %v", err)
			}
		})
	}
}

func TestServicioRechazaReciboAusenteOIdentidadConsumoMalformada(t *testing.T) {
	casos := []struct {
		nombre  string
		omitir  bool
		alterar func(*datosReciboConsumoFuenteDoble)
	}{
		{nombre: "ausente", omitir: true},
		{nombre: "referencia_consumo", alterar: func(d *datosReciboConsumoFuenteDoble) {
			d.consumo.Referencia = "?"
		}},
		{nombre: "version_consumo", alterar: func(d *datosReciboConsumoFuenteDoble) {
			d.consumo.Version = 0
		}},
		{nombre: "huella_consumo", alterar: func(d *datosReciboConsumoFuenteDoble) {
			d.consumo.HuellaSHA256 = "no-es-huella"
		}},
		{nombre: "confusion_consumo_fuente", alterar: func(d *datosReciboConsumoFuenteDoble) {
			d.consumo = d.fuenteExacta
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.fuente.omitirConsumo = caso.omitir
			escenario.fuente.alterarConsumo = caso.alterar
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, ErrFuenteNoConfiable) || escenario.confirmador.llamadas != 0 {
				t.Fatalf("identidad de consumo invalida aceptada: %v", err)
			}
		})
	}
}
