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
		resultado.Recibo.ConfirmadaEn.Location() != time.UTC ||
		resultado.Recibo.ConfirmadaEn.Nanosecond() != 123456000 ||
		resultado.ObservadaEnPrimario.Location() != time.UTC ||
		resultado.Coordenadas.ReciboRef != recibo.ReciboRef ||
		resultado.HuellaOrdenSHA256 != consulta.HuellaOrdenSHA256 {
		t.Fatalf("resultado primario incompleto: %+v", resultado)
	}
}

func TestReciboDecisionCoberturaO404ENormalizaInstantePostgreSQL(
	t *testing.T,
) {
	t.Parallel()
	dto := reciboDenegadoDecisionCoberturaO404EPrueba()
	contenido, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := decodificarReciboDecisionCoberturaO404E(contenido)
	if err != nil || recibo.ConfirmadaEn.Location() != time.UTC ||
		recibo.ConfirmadaEn.Nanosecond() != 123456000 {
		t.Fatalf("recibo de cobertura no normalizado: %#v, %v", recibo, err)
	}
}

func TestCargaTerminalDecisionCoberturaDurableNormalizaInstantesPostgreSQL(
	t *testing.T,
) {
	t.Parallel()
	zona := time.FixedZone("postgresql-utc", 0)
	carga := normalizarCargaTerminalDecisionCoberturaDurablePostgreSQL(
		cargaTerminalDecisionCoberturaDurableV1{
			ReservaTerminal: reservaTerminalDecisionCoberturaDurableV1{
				ObservadaEnDB: time.Date(2026, 9, 4, 13, 50, 52, 123456789, zona),
			},
			Recibo: reciboDecisionCoberturaDurableV1{
				ConfirmadaEn: time.Date(2026, 9, 4, 13, 51, 52, 654321987, zona),
			},
		},
	)
	if carga.ReservaTerminal.ObservadaEnDB.Location() != time.UTC ||
		carga.ReservaTerminal.ObservadaEnDB.Nanosecond() != 123456000 ||
		carga.Recibo.ConfirmadaEn.Location() != time.UTC ||
		carga.Recibo.ConfirmadaEn.Nanosecond() != 654321000 {
		t.Fatalf("terminal durable no normalizado: %#v", carga)
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
		ConfirmadaEn: time.Date(
			2026, 7, 26, 10, 0, 0, 123456789,
			time.FixedZone("postgresql-utc", 0),
		),
		Aplicada: false, DenegadaVEC: true,
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
			RecursoRef: recurso.Referencia, RecursoModulo: recurso.ModuloID,
			RecursoTipo:                 recurso.Tipo,
			Ambitos:                     clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
			Atributos:                   clonarMapaDecisionCoberturaO404E(recurso.Atributos),
			ContextoRecursoHuellaSHA256: huella,
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

func TestDecisionVECDecisionCoberturaO404ERecursoInmutableYFormaCerrada(
	t *testing.T,
) {
	t.Parallel()
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica := []byte{0xaa}
	motivoCanonico := []byte{0xbb}
	dto := decisionVECDecisionCoberturaO404E{
		DecisionCanonica:            decisionCanonica,
		MotivoCanonico:              motivoCanonico,
		ContextoRecursoHuellaSHA256: huella,
	}
	copiarRecursoDecisionVECDecisionCoberturaO404E(&dto, recurso)

	recurso.Ambitos["organizacion_ref"] = "otra-organizacion"
	recurso.Atributos["clasificacion"] = "secreto"
	if dto.Ambitos["organizacion_ref"] != "diputacion-granada" ||
		dto.Atributos["clasificacion"] != "interno" ||
		!validarRecursoDecisionVECDecisionCoberturaO404E(dto) {
		t.Fatal("la decisión VEC no conservó una copia defensiva válida")
	}

	var salida bytes.Buffer
	if err := escribirDecisionVECDecisionCoberturaO404E(
		&salida,
		dto,
	); err != nil {
		t.Fatal(err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(salida.Bytes(), &objeto); err != nil {
		t.Fatal(err)
	}
	claves := []string{
		"decision_canonica_hex", "motivo_canonico_hex",
		"persona_version", "perfil_version", "decision_ref",
		"decision_huella_sha256", "codigo_probatorio", "concedida",
		"emitida_en", "valida_hasta", "principal_id", "perfil_activo_ref",
		"accion", "recurso_ref", "recurso_modulo", "recurso_tipo",
		"ambitos", "atributos", "contexto_recurso_huella_sha256",
		"finalidad", "correlacion_ref",
	}
	if len(objeto) != len(claves) {
		t.Fatalf("forma decision_vec abierta o incompleta: %s", salida.Bytes())
	}
	for _, clave := range claves {
		if _, existe := objeto[clave]; !existe {
			t.Fatalf("falta la clave cerrada %q: %s", clave, salida.Bytes())
		}
	}

	dto.Atributos["clasificacion"] = "alterado"
	if validarRecursoDecisionVECDecisionCoberturaO404E(dto) {
		t.Fatal("una mutación de atributos conservó la huella VEC")
	}
	dto.Atributos["clasificacion"] = "interno"
	ambitosDTO := dto.Ambitos
	atributosDTO := dto.Atributos
	limpiarDecisionVECDecisionCoberturaO404E(&dto)
	if len(ambitosDTO) != 0 || len(atributosDTO) != 0 ||
		!bytes.Equal(decisionCanonica, []byte{0}) ||
		!bytes.Equal(motivoCanonico, []byte{0}) ||
		dto.Ambitos != nil || dto.Atributos != nil {
		t.Fatal("la limpieza no borró la identidad técnica VEC")
	}
}

func TestDecisionVECYDenegacionDecisionCoberturaO404EExigenMismoRecurso(
	t *testing.T,
) {
	t.Parallel()
	carga := cargaDenegadaMinimaDecisionCoberturaO404EPrueba(t)
	contenido, err := codificarCargaConfirmarDecisionCoberturaO404E(carga)
	if err != nil {
		t.Fatalf("recurso repetido nominal rechazado: %v", err)
	}
	borrarBytes(contenido)

	otro := recursoDenegacionDecisionCoberturaO404EPrueba()
	otro.Ambitos["organizacion_ref"] = "otra-organizacion"
	otraHuella, err := otro.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	carga.Denegacion.RecursoRef = otro.Referencia
	carga.Denegacion.RecursoModulo = otro.ModuloID
	carga.Denegacion.RecursoTipo = otro.Tipo
	carga.Denegacion.Ambitos = clonarMapaDecisionCoberturaO404E(otro.Ambitos)
	carga.Denegacion.Atributos = clonarMapaDecisionCoberturaO404E(
		otro.Atributos,
	)
	carga.Denegacion.RecursoHuellaSHA256 = otraHuella
	if !validarRecursoDenegacionDecisionCoberturaO404E(*carga.Denegacion) {
		t.Fatal("el segundo recurso válido fue rechazado aisladamente")
	}
	if recursosDecisionVECYDenegacionDecisionCoberturaO404EIguales(
		carga.DecisionVEC,
		*carga.Denegacion,
	) {
		t.Fatal("dos recursos válidos distintos se consideraron iguales")
	}
	if _, err := codificarCargaConfirmarDecisionCoberturaO404E(
		carga,
	); err == nil {
		t.Fatal("la frontera aceptó identidades de recurso divergentes")
	}
}
