package postgres

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestJSONEstrictoDecisionCoberturaO404ERechazaVariantesAdversarias(
	t *testing.T,
) {
	t.Parallel()
	valido := reciboDenegadoDecisionCoberturaO404EPrueba()
	contenido, err := json.Marshal(valido)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string][]byte{
		"clave_duplicada": bytes.Replace(
			contenido,
			[]byte(`"recibo_ref":`),
			[]byte(`"recibo_ref":"duplicada","recibo_ref":`),
			1,
		),
		"mayuscula": bytes.Replace(
			contenido,
			[]byte(`"recibo_ref"`),
			[]byte(`"Recibo_ref"`),
			1,
		),
		"contenido_posterior": append(
			append([]byte(nil), contenido...),
			[]byte(` {}`)...,
		),
		"huella_mayuscula": bytes.Replace(
			contenido,
			[]byte(strings.Repeat("a", 64)),
			[]byte(strings.Repeat("A", 64)),
			1,
		),
		"hmac_invalido": bytes.Replace(
			contenido,
			[]byte("hmac-sha256:vec.ct.ambito/v1:"),
			[]byte("hmac-no-admitido:"),
			1,
		),
	}
	for nombre, adversario := range casos {
		adversario := adversario
		t.Run(nombre, func(t *testing.T) {
			if _, err := decodificarReciboDecisionCoberturaO404E(
				adversario,
			); err == nil {
				t.Fatal("contenido adversario aceptado")
			}
		})
	}
	profundo := []byte(`{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":1}}}}}}}}}`)
	if err := validarJSONSinDuplicadosDecisionCoberturaO404E(
		profundo,
		maximaProfundidadJSONDecisionCoberturaO404E,
	); err == nil {
		t.Fatal("profundidad excesiva aceptada")
	}
	sobredimensionado := bytes.Repeat(
		[]byte{'x'},
		maximoBytesReciboDecisionCoberturaO404E+1,
	)
	if _, err := decodificarReciboDecisionCoberturaO404E(
		sobredimensionado,
	); err == nil {
		t.Fatal("recibo sobredimensionado aceptado")
	}
}

func TestResultadoPrimarioDecisionCoberturaO404EEncontradoNominal(t *testing.T) {
	t.Parallel()
	recibo := reciboDenegadoDecisionCoberturaO404EPrueba()
	consulta := consultaPrimariaDecisionCoberturaO404E{
		Esquema:         esquemaConsultaPrimariaDecisionCoberturaO404E,
		OrganizacionRef: "org", ExpedienteRef: "exp",
		VersionExpediente: 1, ReservaRef: recibo.ReservaRef,
		ReciboRef: recibo.ReciboRef, CorrelacionVECRef: recibo.CorrelacionVECRef,
		DecisionVECRef:    recibo.DecisionVECRef,
		RevisionCercado:   recibo.RevisionCercado,
		HuellaOrdenSHA256: strings.Repeat("d", 64),
	}
	dto := resultadoPrimarioDecisionCoberturaO404E{
		Esquema:    esquemaResultadoPrimarioDecisionCoberturaO404E,
		Encontrado: true, Consulta: &consulta, Recibo: &recibo,
		ObservadaEnPrimario: recibo.ConfirmadaEn.Add(time.Second),
	}
	contenido, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err :=
		decodificarResultadoPrimarioDecisionCoberturaO404E(contenido)
	if err != nil {
		t.Fatal(err)
	}
	if !resultado.Encontrado || resultado.Recibo.DenegadaVEC == nil ||
		resultado.Recibo.Aplicada != nil ||
		resultado.Coordenadas.ReciboRef != recibo.ReciboRef ||
		resultado.HuellaOrdenSHA256 != consulta.HuellaOrdenSHA256 {
		t.Fatalf("resultado primario incompleto: %+v", resultado)
	}
}

func reciboDenegadoDecisionCoberturaO404EPrueba() reciboDecisionCoberturaO404E {
	return reciboDecisionCoberturaO404E{
		Esquema:   esquemaReciboDecisionCoberturaO404E,
		ReciboRef: "rec", ReservaRef: "res", AuditoriaRef: "aud",
		CorrelacionVECRef: "cor", DecisionVECRef: "dec",
		DecisionVECHuellaSHA256: strings.Repeat("a", 64),
		CodigoProbatorioVEC:     "denegada_por_politica",
		ConcedidaVEC:            false, RevisionCercado: 2,
		AmbitoIdempotenciaHMAC: "hmac-sha256:vec.ct.ambito/v1:" +
			strings.Repeat("b", 64),
		HuellaSemanticaHMAC: "hmac-sha256:vec.ct.semantica/v1:" +
			strings.Repeat("c", 64),
		ConfirmadaEn: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
		Aplicada:     false, DenegadaVEC: true,
	}
}

func cargaDenegadaMinimaDecisionCoberturaO404EPrueba(
	t *testing.T,
) cargaConfirmarDecisionCoberturaO404E {
	t.Helper()
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return cargaConfirmarDecisionCoberturaO404E{
		Esquema: esquemaCargaDecisionCoberturaO404E,
		Rama:    cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada,
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: []byte{1}, MotivoCanonico: []byte{2},
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{},
		Denegacion: &denegacionDecisionCoberturaO404E{
			RecursoRef: recurso.Referencia, RecursoModulo: recurso.ModuloID,
			RecursoTipo:         recurso.Tipo,
			Ambitos:             clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:           clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			RecursoHuellaSHA256: huella,
		},
	}
}
