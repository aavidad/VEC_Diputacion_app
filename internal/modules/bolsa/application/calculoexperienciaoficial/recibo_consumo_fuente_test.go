package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
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
		{"recurso_ref", func(d *datosReciboConsumoFuenteDoble) { d.recursoRef = "fuente:" + strings.Repeat("4", 64) }},
		{"huella_contexto", func(d *datosReciboConsumoFuenteDoble) { d.huellaContexto = strings.Repeat("2", 64) }},
		{"correlacion_ref", func(d *datosReciboConsumoFuenteDoble) {
			d.correlacionRef = "correlacion_abcdef0123456789abcdef0123456789"
		}},
		{"huella_selector", func(d *datosReciboConsumoFuenteDoble) { d.huellaSelector = strings.Repeat("3", 64) }},
		{"huella_entrada", func(d *datosReciboConsumoFuenteDoble) { d.huellaEntrada = strings.Repeat("4", 64) }},
		{"fuente_referencia", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraReferencia(d.fuenteExacta) }},
		{"fuente_version", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraVersion(d.fuenteExacta) }},
		{"fuente_huella", func(d *datosReciboConsumoFuenteDoble) { d.fuenteExacta = otraHuella(d.fuenteExacta) }},
		{"verificador_referencia", func(d *datosReciboConsumoFuenteDoble) { d.verificador = otraReferencia(d.verificador) }},
		{"verificador_version", func(d *datosReciboConsumoFuenteDoble) { d.verificador = otraVersion(d.verificador) }},
		{"verificador_huella", func(d *datosReciboConsumoFuenteDoble) { d.verificador = otraHuella(d.verificador) }},
		{"prueba_referencia", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraReferencia(d.consumoPrueba) }},
		{"prueba_version", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraVersion(d.consumoPrueba) }},
		{"prueba_huella", func(d *datosReciboConsumoFuenteDoble) { d.consumoPrueba = otraHuella(d.consumoPrueba) }},
		{"auditoria_referencia", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraReferencia(d.auditoria) }},
		{"auditoria_version", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraVersion(d.auditoria) }},
		{"auditoria_huella", func(d *datosReciboConsumoFuenteDoble) { d.auditoria = otraHuella(d.auditoria) }},
		{"prueba_emitida", func(d *datosReciboConsumoFuenteDoble) { d.pruebaEmitidaEn = d.pruebaEmitidaEn.Add(-time.Microsecond) }},
		{"prueba_validez", func(d *datosReciboConsumoFuenteDoble) {
			d.pruebaValidaHasta = d.pruebaValidaHasta.Add(time.Microsecond)
		}},
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

func TestServicioRechazaFuenteMutadaDespuesDeEmitirRecibo(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*puertosbolsa.FuenteExactaCalculoReglasBaremo)
	}{
		{"tramos_y_huella_declarada", func(f *puertosbolsa.FuenteExactaCalculoReglasBaremo) {
			entrada := entradaPrueba(t, true)
			f.Entrada = entrada
			f.Prueba.HuellaEntradaSHA256 = debePrueba(entrada.HuellaSHA256())
		}},
		{"verificador", func(f *puertosbolsa.FuenteExactaCalculoReglasBaremo) {
			f.Prueba.Verificador = referenciaPrueba(t, "verificador:fuente:alternativo", 2)
		}},
		{"emitida_en", func(f *puertosbolsa.FuenteExactaCalculoReglasBaremo) {
			f.Prueba.EmitidaEn = f.Prueba.EmitidaEn.Add(-time.Minute)
		}},
		{"valida_hasta", func(f *puertosbolsa.FuenteExactaCalculoReglasBaremo) {
			f.Prueba.ValidaHasta = f.Prueba.ValidaHasta.Add(-time.Minute)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.fuente.alterarFuenteDespuesConsumo = func(
				fuente *puertosbolsa.FuenteExactaCalculoReglasBaremo,
			) {
				huellaAntes := debePrueba(fuente.ConsumoAutorizacion.HuellaSHA256V1())
				caso.mutar(fuente)
				huellaDespues := debePrueba(fuente.ConsumoAutorizacion.HuellaSHA256V1())
				if huellaAntes != huellaDespues {
					t.Fatal("la mutacion de la fuente sustituyo el recibo de control")
				}
			}
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, ErrFuenteNoConfiable) ||
				len(escenario.exigidor.llamadas) != 1 || escenario.confirmador.llamadas != 0 {
				t.Fatalf("fuente mutada con el mismo recibo alcanzo confirmacion: %v", err)
			}
		})
	}
}

func TestUnionCompletaDeRolesDeFuenteNoAdmiteReferenciasReutilizadas(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	autorizacion, solicitadaEn, err := escenario.servicio.autorizarLectura(
		context.Background(), escenario.datosOrden, escenario.ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	fuente, err := escenario.fuente.ObtenerFuenteExacta(
		context.Background(), puertosbolsa.SolicitudFuenteExactaCalculoReglasBaremo{
			Selector: escenario.datosOrden.Selector, Autorizacion: autorizacion,
			SolicitadaEn: solicitadaEn,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosAutorizacion := debePrueba(autorizacion.Datos())
	consumo := debePrueba(fuente.ConsumoAutorizacion.Consumo())
	objetivos := []string{
		consumo.Referencia, datosAutorizacion.Decision.DecisionRef,
		datosAutorizacion.Decision.RecursoRef, datosAutorizacion.Decision.CorrelacionRef,
		fuente.Prueba.Verificador.Referencia(), fuente.ConsumoPrueba.Referencia(),
		fuente.Auditoria.Referencia(), fuente.Prueba.EstadoReglas.Contenido().Referencia(),
		fuente.Prueba.InstantaneaEntrada.Referencia(), fuente.Prueba.SujetoPseudonimo.Referencia(),
		fuente.Prueba.Convocatoria.Referencia(),
	}
	for indice, objetivo := range objetivos {
		alterada := fuente
		alterada.Prueba.Evidencia = referenciaPrueba(t, objetivo, uint64(indice+20))
		if referenciasFuenteDistintas(alterada, autorizacion) {
			t.Fatalf("la union completa acepto confusion con el rol %d", indice)
		}
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
