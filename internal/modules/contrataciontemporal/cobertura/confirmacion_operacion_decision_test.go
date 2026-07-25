package cobertura_test

import (
	"bytes"
	"context"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestConfirmacionOrdenDecisionLigaExactamenteAmbasRamasYCopia(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	casos := []struct {
		nombre    string
		orden     cobertura.OrdenOperacionDecisionCobertura
		recibo    cobertura.ReciboOperacionDecisionCobertura
		resultado cobertura.ResultadoConfirmacionOperacionDecisionCobertura
	}{
		{"concedida", e.ordenConcedida, e.reciboConcedido, e.resultadoConcedido},
		{"denegada", e.ordenDenegada, e.reciboDenegado, e.resultadoDenegado},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			recibo, err := caso.resultado.ReciboPara(caso.orden)
			if err != nil || recibo.DecisionVECRef != caso.recibo.DecisionVECRef {
				t.Fatalf("recibo exacto rechazado: %#v, %v", recibo, err)
			}
			recibo.DecisionVECRef = "decision_ajena"
			segunda, err := caso.resultado.ReciboPara(caso.orden)
			if err != nil ||
				segunda.DecisionVECRef != caso.recibo.DecisionVECRef {
				t.Fatal("el resultado compartió la copia entregada")
			}
		})
	}
	if _, err := e.resultadoConcedido.ReciboPara(e.ordenDenegada); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("resultado concedido aceptado para orden denegada: %v", err)
	}
	if _, err := cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
		e.ordenConcedida,
		e.reciboDenegado,
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("recibo de otra rama aceptado: %v", err)
	}
	entrada := clonarReciboConfirmacionOrdenC3(e.reciboConcedido)
	resultado, err :=
		cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
			e.ordenConcedida,
			entrada,
		)
	if err != nil {
		t.Fatal(err)
	}
	entrada.DecisionVECRef = "decision_mutada_despues"
	conservado, err := resultado.ReciboPara(e.ordenConcedida)
	if err != nil || conservado.DecisionVECRef !=
		e.reciboConcedido.DecisionVECRef {
		t.Fatal("el constructor retuvo alias del recibo de entrada")
	}
}

func TestConfirmacionOrdenDecisionRechazaAdulteracionRecibo(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	casos := []struct {
		nombre string
		mutar  func(*cobertura.ReciboOperacionDecisionCobertura)
	}{
		{"recibo", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.ReciboRef = "recibo_ajeno"
		}},
		{"reserva", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.ReservaRef = "reserva_ajena"
		}},
		{"auditoría", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.AuditoriaRef = "auditoria_ajena"
		}},
		{"correlación", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.CorrelacionVECRef = "correlacion_ajena"
		}},
		{"decisión VEC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.DecisionVECRef = "decision_ajena"
		}},
		{"huella VEC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.DecisionVECHuellaSHA256 = strings.Repeat("8", 64)
		}},
		{"código VEC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = "codigo_ajeno"
		}},
		{"rama VEC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.ConcedidaVEC = false
		}},
		{"cercado", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.RevisionCercado++
		}},
		{"ámbito HMAC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.AmbitoIdempotenciaHMAC += "a"
		}},
		{"semántica HMAC", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.HuellaSemanticaHMAC += "a"
		}},
		{"decisión C2", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.DecisionCoberturaRef = "decision_cobertura_ajena"
		}},
		{"huella C2", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.DecisionCoberturaHuella = strings.Repeat("7", 64)
		}},
		{"versión", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.VersionResultante++
		}},
		{"evento", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.EventoRef = "evento_ajeno"
		}},
		{"actuación", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.ActuacionRef = "actuacion_ajena"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			recibo := clonarReciboConfirmacionOrdenC3(e.reciboConcedido)
			caso.mutar(&recibo)
			if _, err := cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
				e.ordenConcedida,
				recibo,
			); !errors.Is(
				err,
				cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
			) {
				t.Fatalf("adulteración aceptada: %v", err)
			}
		})
	}
	recibo := clonarReciboConfirmacionOrdenC3(e.reciboDenegado)
	recibo.DenegadaVEC = nil
	recibo.Aplicada = &cobertura.ResultadoAplicadoOperacionDecisionCobertura{}
	if _, err := cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
		e.ordenDenegada,
		recibo,
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("efecto C2 aceptado en denegación: %v", err)
	}
}

func TestConfirmacionOrdenDecisionExigeVentanaExclusivaYEnterosSeguros(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	casos := []struct {
		nombre string
		mutar  func(*cobertura.ReciboOperacionDecisionCobertura)
	}{
		{"límite exacto", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.ConfirmadaEn = e.limite
		}},
		{"antes del efecto", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.ConfirmadaEn = e.base.Add(3999 * time.Microsecond)
		}},
		{"cercado 2^53", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.RevisionCercado =
				cobertura.MaximoEnteroSeguroOperacionDecisionCobertura + 1
		}},
		{"versión resultante 2^53", func(r *cobertura.ReciboOperacionDecisionCobertura) {
			r.Aplicada.VersionResultante =
				cobertura.MaximoEnteroSeguroOperacionDecisionCobertura + 1
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			recibo := clonarReciboConfirmacionOrdenC3(e.reciboConcedido)
			caso.mutar(&recibo)
			if _, err := cobertura.NuevaResultadoConfirmacionOperacionDecisionCobertura(
				e.ordenConcedida,
				recibo,
			); !errors.Is(
				err,
				cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
			) {
				t.Fatalf("límite inválido aceptado: %v", err)
			}
		})
	}
}

func TestIntentoConfirmacionOrdenDecisionPriorizaReciboTrasCancelacion(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &transaccionConfirmacionOrdenC3{
		resultado: e.resultadoConcedido,
		err:       context.Canceled,
		despues:   cancelar,
	}
	intento, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		ctx,
		tx,
		e.ordenConcedida,
	)
	confirmacion, confirmada := intento.ConfirmacionPara(e.ordenConcedida)
	if err != nil || !confirmada || tx.llamadas.Load() != 1 {
		t.Fatalf("recibo válido quedó ambiguo: %v", err)
	}
	if _, err = confirmacion.ReciboPara(e.ordenConcedida); err != nil {
		t.Fatal(err)
	}
	if _, ambigua := intento.ReconciliacionPara(e.ordenConcedida); ambigua {
		t.Fatal("un recibo válido generó reconciliación")
	}
}

func TestIntentoConfirmacionOrdenDecisionAmbiguoNuncaReintenta(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	tx := &transaccionConfirmacionOrdenC3{err: errTransporteConfirmacionOrdenC3}
	intento, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		tx,
		e.ordenDenegada,
	)
	if !errors.Is(
		err,
		cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo,
	) || tx.llamadas.Load() != 1 {
		t.Fatalf("resultado ambiguo mal clasificado: %v", err)
	}
	if _, confirmada := intento.ConfirmacionPara(e.ordenDenegada); confirmada {
		t.Fatal("la ambigüedad fabricó una confirmación")
	}
	solicitud, requiere := intento.ReconciliacionPara(e.ordenDenegada)
	if !requiere {
		t.Fatal("la ambigüedad no produjo consulta primaria")
	}
	coordenadas, err := solicitud.CoordenadasPrimarias()
	if err != nil || coordenadas.RevisionCercado != 1 ||
		coordenadas.ReservaRef != e.reciboDenegado.ReservaRef {
		t.Fatalf("coordenadas primarias inesperadas: %#v, %v", coordenadas, err)
	}
	if tx.llamadas.Load() != 1 {
		t.Fatal("se produjo un retry ciego")
	}
}

func TestIntentoConfirmacionOrdenDecisionRespetaCancelacionPreviaYTypedNil(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	tx := &transaccionConfirmacionOrdenC3{}
	if _, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		ctx,
		tx,
		e.ordenConcedida,
	); !errors.Is(err, context.Canceled) || tx.llamadas.Load() != 0 {
		t.Fatalf("cancelación previa invocó la transacción: %v", err)
	}
	var nula *transaccionConfirmacionOrdenC3
	if _, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		nula,
		e.ordenConcedida,
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("dependencia typed-nil aceptada: %v", err)
	}
}

func TestReconciliacionOrdenDecisionConsultaPrimarioSinAutorizarRetry(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, err :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			e.ordenDenegada,
		)
	if err != nil {
		t.Fatal(err)
	}
	confirmada, err :=
		cobertura.NuevaResultadoReconciliacionConfirmadaOperacionDecisionCobertura(
			solicitud,
			e.reciboDenegado,
			e.reciboDenegado.ConfirmadaEn.Add(time.Microsecond),
		)
	if err != nil {
		t.Fatal(err)
	}
	if candidata, valida := confirmada.ConfirmacionPara(
		e.ordenDenegada,
	); !valida {
		t.Fatal("la candidata primaria exacta no pudo elevarse con la orden")
	} else if _, err = candidata.ReciboPara(e.ordenDenegada); err != nil {
		t.Fatal(err)
	}
	if _, valida := confirmada.ConfirmacionPara(e.ordenConcedida); valida {
		t.Fatal("la candidata primaria se elevó con una orden ajena")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	reconciliador := &reconciliadorConfirmacionOrdenC3{
		resultado: confirmada, err: context.Canceled, despues: cancelar,
	}
	resultado, err := cobertura.ReconciliarConfirmacionOperacionDecisionCobertura(
		ctx,
		reconciliador,
		solicitud,
		e.ordenDenegada,
	)
	if err != nil || reconciliador.llamadas.Load() != 1 {
		t.Fatalf("recibo primario válido quedó ambiguo: %v", err)
	}
	if _, err = resultado.ReciboPara(e.ordenDenegada); err != nil {
		t.Fatal(err)
	}
	noConcluyente, err :=
		cobertura.NuevaResultadoReconciliacionNoConcluyenteOperacionDecisionCobertura(
			solicitud,
			e.reciboDenegado.ConfirmadaEn.Add(time.Microsecond),
		)
	if err != nil {
		t.Fatal(err)
	}
	reconciliador = &reconciliadorConfirmacionOrdenC3{
		resultado: noConcluyente,
	}
	if _, err = cobertura.ReconciliarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		reconciliador,
		solicitud,
		e.ordenDenegada,
	); !errors.Is(
		err,
		cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo,
	) || reconciliador.llamadas.Load() != 1 {
		t.Fatalf("ausencia no concluyente autorizó retry: %v", err)
	}
	if _, err = cobertura.NuevaResultadoReconciliacionConfirmadaOperacionDecisionCobertura(
		solicitud,
		e.reciboDenegado,
		e.reciboDenegado.ConfirmadaEn.Add(-time.Microsecond),
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("observación primaria anterior al recibo aceptada: %v", err)
	}
	solicitudAjena, err :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			e.ordenConcedida,
		)
	if err != nil {
		t.Fatal(err)
	}
	reconciliador = &reconciliadorConfirmacionOrdenC3{
		resultado: noConcluyente,
	}
	if _, err = cobertura.ReconciliarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		reconciliador,
		solicitudAjena,
		e.ordenDenegada,
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) || reconciliador.llamadas.Load() != 0 {
		t.Fatalf("solicitud A/B alcanzó el reconciliador: %v", err)
	}
}

func TestContratosConfirmacionOrdenDecisionSonOpacos(t *testing.T) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	tx := &transaccionConfirmacionOrdenC3{err: errTransporteConfirmacionOrdenC3}
	intento, _ := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		tx,
		e.ordenDenegada,
	)
	solicitud, _ := intento.ReconciliacionPara(e.ordenDenegada)
	datos, err := solicitud.CoordenadasPrimarias()
	if err != nil {
		t.Fatal(err)
	}
	reconciliacion, err :=
		cobertura.NuevaResultadoReconciliacionNoConcluyenteOperacionDecisionCobertura(
			solicitud,
			e.base.Add(time.Second),
		)
	if err != nil {
		t.Fatal(err)
	}
	valores := []any{
		e.resultadoConcedido,
		datos,
		solicitud,
		intento,
		reconciliacion,
	}
	for _, valor := range valores {
		comprobarOpacidadConfirmacionOrdenC3(t, valor)
	}
}

func comprobarOpacidadConfirmacionOrdenC3(t *testing.T, valor any) {
	t.Helper()
	esperado := cobertura.ErrSerializacionOperacionDecisionCoberturaProhibida
	if _, err := json.Marshal(valor); !errors.Is(err, esperado) {
		t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
	}
	if _, err := xml.Marshal(valor); !errors.Is(err, esperado) {
		t.Fatalf("XML no bloqueado para %T: %v", valor, err)
	}
	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(valor); !errors.Is(err, esperado) {
		t.Fatalf("gob no bloqueado para %T: %v", valor, err)
	}
	if _, err := valor.(encoding.TextMarshaler).MarshalText(); !errors.Is(
		err,
		esperado,
	) {
		t.Fatalf("texto no bloqueado para %T: %v", valor, err)
	}
	if _, err := valor.(encoding.BinaryMarshaler).MarshalBinary(); !errors.Is(
		err,
		esperado,
	) {
		t.Fatalf("binario no bloqueado para %T: %v", valor, err)
	}
	if _, err := valor.(interface {
		MarshalYAML() (any, error)
	}).MarshalYAML(); !errors.Is(err, esperado) {
		t.Fatalf("YAML no bloqueado para %T: %v", valor, err)
	}
	if _, err := valor.(interface {
		MarshalCBOR() ([]byte, error)
	}).MarshalCBOR(); !errors.Is(err, esperado) {
		t.Fatalf("CBOR no bloqueado para %T: %v", valor, err)
	}
	if texto := strings.ToUpper(fmt.Sprintf("%+v", valor)); !strings.Contains(
		texto,
		"OPACA",
	) {
		t.Fatalf("formato no redactado para %T: %q", valor, texto)
	}
	if texto := strings.ToUpper(slog.AnyValue(valor).String()); !strings.Contains(
		texto,
		"OPACA",
	) {
		t.Fatalf("log no redactado para %T: %q", valor, texto)
	}
}

func TestConsultaPrimariaOrdenDecisionEsMinimizada(t *testing.T) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, err :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			e.ordenConcedida,
		)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := solicitud.CoordenadasPrimarias()
	if err != nil {
		t.Fatal(err)
	}
	tipo := reflect.TypeOf(datos)
	prohibidos := []string{
		"token", "actor", "perfil", "motivo", "recurso", "agregado",
		"propuesta", "evidencia", "huellaorden",
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("consulta primaria expone %s", tipo.Field(indice).Name)
			}
		}
	}
	if datos.VersionExpediente >=
		cobertura.MaximoEnteroSeguroOperacionDecisionCobertura {
		t.Fatal("versión no portable en consulta primaria")
	}
}

func TestConfirmacionOrdenDecisionEsConcurrenteYNoComparteEstado(
	t *testing.T,
) {
	e := nuevoEscenarioConfirmacionOrdenC3(t)
	tx := &transaccionConfirmacionOrdenC3{resultado: e.resultadoConcedido}
	const trabajadores = 32
	var grupo sync.WaitGroup
	errores := make(chan error, trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			intento, err :=
				cobertura.IntentarConfirmacionOperacionDecisionCobertura(
					context.Background(),
					tx,
					e.ordenConcedida,
				)
			if err == nil {
				confirmacion, ok := intento.ConfirmacionPara(e.ordenConcedida)
				if !ok {
					err = errors.New("confirmación ausente")
				} else {
					recibo, errRecibo := confirmacion.ReciboPara(
						e.ordenConcedida,
					)
					if errRecibo != nil {
						err = errRecibo
					} else {
						recibo.DecisionVECRef = "mutacion_local"
					}
				}
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
	if tx.llamadas.Load() != trabajadores {
		t.Fatalf("invocaciones inesperadas: %d", tx.llamadas.Load())
	}
	recibo, err := e.resultadoConcedido.ReciboPara(e.ordenConcedida)
	if err != nil || recibo.DecisionVECRef == "mutacion_local" {
		t.Fatal("una mutación concurrente alcanzó el resultado compartido")
	}
}
