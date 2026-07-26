package postgres

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func confirmadoRecuperacionResultadoCoberturaO405Prueba() resultadoConfirmadoRecuperacionResultadoCoberturaO405 {
	ambito := "hmac-sha256:vec.contratacion-temporal." +
		"cobertura-decision.ambito/v2:" + strings.Repeat("a", 64)
	semantica := "hmac-sha256:vec.contratacion-temporal." +
		"cobertura-decision.semantica/v2:" + strings.Repeat("b", 64)
	observadaDB := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	confirmada := observadaDB.Add(time.Minute)
	return resultadoConfirmadoRecuperacionResultadoCoberturaO405{
		Esquema:                esquemaResultadoRecuperacionResultadoCoberturaO405,
		Estado:                 estadoConfirmadoRecuperacionResultadoCoberturaO405,
		OrganizacionRef:        "organizacion_diputacion",
		ExpedienteRef:          "expediente_temporal_2026",
		VersionExpediente:      2,
		ReservaRef:             "reserva_2026",
		ReciboRef:              "recibo_2026",
		ActuacionRef:           "actuacion_preasignada_2026",
		AuditoriaRef:           "auditoria_2026",
		EventoRef:              "evento_preasignado_2026",
		CorrelacionVECRef:      "correlacion_vec_2026",
		DecisionVECRef:         "decision_vec_2026",
		AmbitoIdempotenciaHMAC: ambito,
		HuellaSemanticaHMAC:    semantica,
		RevisionCercado:        7,
		ObservadaEnDB:          observadaDB,
		Recibo: reciboDecisionCoberturaO404E{
			Esquema:                 esquemaReciboDecisionCoberturaO404E,
			ReciboRef:               "recibo_2026",
			ReservaRef:              "reserva_2026",
			AuditoriaRef:            "auditoria_2026",
			CorrelacionVECRef:       "correlacion_vec_2026",
			DecisionVECRef:          "decision_vec_2026",
			DecisionVECHuellaSHA256: strings.Repeat("c", 64),
			CodigoProbatorioVEC:     "denegada_por_politica",
			ConcedidaVEC:            false,
			RevisionCercado:         7,
			AmbitoIdempotenciaHMAC:  ambito,
			HuellaSemanticaHMAC:     semantica,
			ConfirmadaEn:            confirmada,
			Aplicada:                false,
			DenegadaVEC:             true,
			DecisionCoberturaRef:    "",
			DecisionCoberturaHuella: "",
			VersionResultante:       0,
			EventoRef:               "",
			ActuacionRef:            "",
		},
		ObservadaEn: confirmada.Add(time.Minute),
	}
}

func TestDecodificarRecuperacionResultadoCoberturaO405AceptaRamasExactas(
	t *testing.T,
) {
	t.Parallel()
	noObservable := respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba()
	resultado, err :=
		decodificarResultadoRecuperacionResultadoCoberturaO405(noObservable)
	if err != nil || resultado.Encontrado || resultado.ObservadaEn.IsZero() {
		t.Fatalf("no observable inválido: resultado=%+v err=%v", resultado, err)
	}
	confirmado := confirmadoRecuperacionResultadoCoberturaO405Prueba()
	contenido, err := json.Marshal(confirmado)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err =
		decodificarResultadoRecuperacionResultadoCoberturaO405(contenido)
	if err != nil || !resultado.Encontrado ||
		resultado.Reserva.ReciboRef != confirmado.ReciboRef ||
		resultado.Recibo.ReciboRef != confirmado.ReciboRef ||
		resultado.Recibo.DenegadaVEC == nil ||
		resultado.Recibo.Aplicada != nil {
		t.Fatalf("confirmado inválido: resultado=%+v err=%v", resultado, err)
	}
	aplicado := confirmadoRecuperacionResultadoCoberturaO405Prueba()
	aplicado.Recibo.CodigoProbatorioVEC = "concedida"
	aplicado.Recibo.ConcedidaVEC = true
	aplicado.Recibo.Aplicada = true
	aplicado.Recibo.DenegadaVEC = false
	aplicado.Recibo.DecisionCoberturaHuella = strings.Repeat("d", 64)
	aplicado.Recibo.DecisionCoberturaRef = "decision-cobertura:sha256:" +
		aplicado.Recibo.DecisionCoberturaHuella
	aplicado.Recibo.VersionResultante = aplicado.VersionExpediente + 1
	aplicado.Recibo.EventoRef = aplicado.EventoRef
	aplicado.Recibo.ActuacionRef = aplicado.ActuacionRef
	contenidoAplicado, err := json.Marshal(aplicado)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err =
		decodificarResultadoRecuperacionResultadoCoberturaO405(
			contenidoAplicado,
		)
	if err != nil || resultado.Recibo.Aplicada == nil ||
		resultado.Recibo.DenegadaVEC != nil ||
		resultado.Recibo.Aplicada.EventoRef != aplicado.EventoRef {
		t.Fatalf("aplicado inválido: resultado=%+v err=%v", resultado, err)
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(contenido, &objeto) != nil ||
		len(objeto) != 18 {
		t.Fatalf("rama confirmada no tiene 18 claves: %d", len(objeto))
	}
	objeto = nil
	if json.Unmarshal(noObservable, &objeto) != nil ||
		len(objeto) != 3 {
		t.Fatalf("rama no observable no tiene 3 claves: %d", len(objeto))
	}
}

func TestDecodificarRecuperacionResultadoCoberturaO405RechazaJSONAdversario(
	t *testing.T,
) {
	t.Parallel()
	confirmado, err := json.Marshal(
		confirmadoRecuperacionResultadoCoberturaO405Prueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	noObservable := respuestaNoObservableRecuperacionResultadoCoberturaO405Prueba()
	var sinClave map[string]any
	if err := json.Unmarshal(confirmado, &sinClave); err != nil {
		t.Fatal(err)
	}
	delete(sinClave, "recibo_ref")
	confirmadoSinClave, _ := json.Marshal(sinClave)
	var conExtra map[string]any
	if err := json.Unmarshal(noObservable, &conExtra); err != nil {
		t.Fatal(err)
	}
	conExtra["recibo"] = map[string]any{}
	noObservableConExtra, _ := json.Marshal(conExtra)

	casos := map[string][]byte{
		"vacio": nil,
		"sobredimensionado": []byte(
			`{"estado":"no_observable","relleno":"` +
				strings.Repeat(
					"x",
					maximoBytesResultadoRecuperacionResultadoCoberturaO405,
				) + `"}`,
		),
		"no_observable_con_terminal": noObservableConExtra,
		"confirmado_sin_clave":       confirmadoSinClave,
		"duplicada_superior": []byte(
			strings.Replace(
				string(confirmado),
				`{"esquema":`,
				`{"estado":"confirmado","esquema":`,
				1,
			),
		),
		"duplicada_recibo": []byte(
			strings.Replace(
				string(confirmado),
				`"recibo":{"esquema":`,
				`"recibo":{"recibo_ref":"otro","esquema":`,
				1,
			),
		),
		"estado_desconocido": []byte(
			strings.Replace(
				string(noObservable),
				`"no_observable"`,
				`"ausente"`,
				1,
			),
		),
		"instante_no_canonico": []byte(
			strings.Replace(
				string(noObservable),
				`2026-07-26T10:02:00Z`,
				`2026-07-26T12:02:00+02:00`,
				1,
			),
		),
	}
	for nombre, contenido := range casos {
		nombre, contenido := nombre, contenido
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			resultado, errDecodificar :=
				decodificarResultadoRecuperacionResultadoCoberturaO405(
					contenido,
				)
			if !errors.Is(
				errDecodificar,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
			) || resultado.Encontrado ||
				!resultado.ObservadaEn.IsZero() {
				t.Fatalf(
					"adversario aceptado: resultado=%+v err=%v",
					resultado,
					errDecodificar,
				)
			}
		})
	}
}

func TestDecodificarRecuperacionResultadoCoberturaO405RechazaProyeccionCorrupta(
	t *testing.T,
) {
	t.Parallel()
	base := confirmadoRecuperacionResultadoCoberturaO405Prueba()
	casos := map[string]func(*resultadoConfirmadoRecuperacionResultadoCoberturaO405){
		"referencia_cruzada": func(d *resultadoConfirmadoRecuperacionResultadoCoberturaO405) {
			d.Recibo.ReciboRef = "recibo_cruzado_2026"
		},
		"sello_cruzado": func(d *resultadoConfirmadoRecuperacionResultadoCoberturaO405) {
			d.Recibo.HuellaSemanticaHMAC = "hmac-sha256:vec." +
				"contratacion-temporal.cobertura-decision.semantica/v2:" +
				strings.Repeat("d", 64)
		},
		"revision_cruzada": func(d *resultadoConfirmadoRecuperacionResultadoCoberturaO405) {
			d.Recibo.RevisionCercado++
		},
		"instante_invertido": func(d *resultadoConfirmadoRecuperacionResultadoCoberturaO405) {
			d.ObservadaEn = d.Recibo.ConfirmadaEn.Add(-time.Second)
		},
		"rama_contradictoria": func(d *resultadoConfirmadoRecuperacionResultadoCoberturaO405) {
			d.Recibo.Aplicada = true
		},
	}
	for nombre, mutar := range casos {
		nombre, mutar := nombre, mutar
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			dto := base
			mutar(&dto)
			contenido, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			_, err = decodificarResultadoRecuperacionResultadoCoberturaO405(
				contenido,
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
			) {
				t.Fatalf("proyección corrupta aceptada: %v", err)
			}
		})
	}
}
