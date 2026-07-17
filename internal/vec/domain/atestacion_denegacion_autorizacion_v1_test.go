package domain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestMensajeAtestacionDenegacionAutorizacionV1VectorDeterministaYSeparado(t *testing.T) {
	cabecera := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	original := clonarDecisionAtestacionAutorizacionV1Prueba(decision)

	mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabecera,
		decision,
		referencia,
	)
	if err != nil {
		t.Fatalf("serializar VEC-AD-D-1: %v", err)
	}
	if !reflect.DeepEqual(decision, original) {
		t.Fatal("la serializacion VEC-AD-D-1 modifico la decision")
	}
	prefijo := append([]byte(EsquemaMensajeAtestacionDenegacionAutorizacionV1), 0)
	if !bytes.HasPrefix(mensaje, prefijo) ||
		bytes.HasPrefix(mensaje, append([]byte(EsquemaMensajeAtestacionAutorizacionV1), 0)) ||
		bytes.HasPrefix(mensaje, append([]byte(EsquemaMensajeAtestacionAutorizacionV2), 0)) {
		t.Fatal("dominio criptografico de denegacion ausente o cruzado")
	}
	posicionVersion := len(prefijo)
	if version := binary.BigEndian.Uint16(mensaje[posicionVersion : posicionVersion+2]); version != VersionFormatoAtestacionDenegacionAutorizacionV1 {
		t.Fatalf("version VEC-AD-D-1 = %d", version)
	}
	if longitud := binary.BigEndian.Uint64(mensaje[len(mensaje)-8:]); longitud != uint64(len(mensaje)) {
		t.Fatalf("longitud final = %d; bytes = %d", longitud, len(mensaje))
	}

	huella, err := HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1(
		cabecera,
		decision,
		referencia,
	)
	if err != nil {
		t.Fatalf("calcular huella VEC-AD-D-1: %v", err)
	}
	const longitudEsperada = 2371
	const huellaEsperada = "ff44e2eeab73f9c9e1c8563d006880bf63224396b545ab94bf184da186ef0380"
	if len(mensaje) != longitudEsperada || huella != huellaEsperada {
		t.Fatalf(
			"cambio incompatible del vector VEC-AD-D-1: longitud=%d huella=%s",
			len(mensaje),
			huella,
		)
	}
}

func TestMensajeAtestacionDenegacionAutorizacionV1LigaResultadoYMotivo(t *testing.T) {
	cabecera := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	base := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	mensajeBase, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabecera,
		base,
		referencia,
	)
	if err != nil {
		t.Fatalf("crear mensaje base: %v", err)
	}

	otroCodigo := clonarDecisionAtestacionAutorizacionV1Prueba(base)
	otroCodigo.Codigo = "denegacion_por_garantia"
	mensajeCodigo, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabecera,
		otroCodigo,
		referencia,
	)
	if err != nil || bytes.Equal(mensajeBase, mensajeCodigo) {
		t.Fatalf("el codigo de denegacion no quedo ligado: %v", err)
	}

	cambiosMotivo := []struct {
		nombre string
		mutar  func(*ReferenciaEntradaCatalogo)
	}{
		{"catalogo", func(r *ReferenciaEntradaCatalogo) { r.CatalogoID = "motivos_autorizacion_otro" }},
		{"version", func(r *ReferenciaEntradaCatalogo) { r.CatalogoVersion++ }},
		{"huella", func(r *ReferenciaEntradaCatalogo) { r.CatalogoHuellaSHA256 = strings.Repeat("8", 64) }},
		{"clave", func(r *ReferenciaEntradaCatalogo) { r.EntradaClave = "motivo_33333333333333333333333333333333" }},
	}
	for _, cambio := range cambiosMotivo {
		t.Run(cambio.nombre, func(t *testing.T) {
			otraReferencia := referencia
			cambio.mutar(&otraReferencia)
			if _, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
				cabecera,
				base,
				otraReferencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("cruce de motivo aceptado: %v", err)
			}

			otraDecision := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			otraDecision.MotivoHuellaSHA256, err = HuellaSHA256MotivoAutorizacionV2(
				otraReferencia,
			)
			if err != nil {
				t.Fatalf("recalcular motivo: %v", err)
			}
			mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
				cabecera,
				otraDecision,
				otraReferencia,
			)
			if err != nil || bytes.Equal(mensajeBase, mensaje) {
				t.Fatalf("coordenada de motivo no ligada: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionDenegacionAutorizacionV1RechazaConcesionesYCruces(t *testing.T) {
	cabecera := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	denegacion := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	concesion := decisionAtestacionAutorizacionV2Prueba(t, referencia)

	cabeceraCero := CabeceraAtestacionDenegacionAutorizacionV1{}
	cabeceraVersionConcesion := cabecera
	cabeceraVersionConcesion.FormatoVersion = VersionFormatoAtestacionAutorizacionV2
	codigoConcesion := clonarDecisionAtestacionAutorizacionV1Prueba(denegacion)
	codigoConcesion.Codigo = "concedida"

	casos := []struct {
		nombre     string
		cabecera   CabeceraAtestacionDenegacionAutorizacionV1
		decision   DecisionAutorizacion
		referencia ReferenciaEntradaCatalogo
	}{
		{"concesion", cabecera, concesion, referencia},
		{"codigo_concesion", cabecera, codigoConcesion, referencia},
		{"decision_V1", cabecera, decisionAtestacionAutorizacionV1Prueba(t), referencia},
		{"cabecera_cero", cabeceraCero, denegacion, referencia},
		{"version_concesion", cabeceraVersionConcesion, denegacion, referencia},
		{"decision_cero", cabecera, DecisionAutorizacion{}, referencia},
		{"motivo_cero", cabecera, denegacion, ReferenciaEntradaCatalogo{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
				caso.cabecera,
				caso.decision,
				caso.referencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("entrada incompatible aceptada: %v", err)
			}
		})
	}

	if reflect.TypeOf(CabeceraAtestacionDenegacionAutorizacionV1{}) ==
		reflect.TypeOf(CabeceraAtestacionAutorizacionV2{}) {
		t.Fatal("concesion y denegacion comparten tipo de cabecera")
	}
}

func TestCabeceraAtestacionDenegacionAutorizacionV1LigaConfiguracion(t *testing.T) {
	base := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	huellaBase, err := HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1(
		base,
		decision,
		referencia,
	)
	if err != nil {
		t.Fatalf("huella base: %v", err)
	}
	for _, cambio := range []func(*CabeceraAtestacionDenegacionAutorizacionV1){
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.Suite = "VEC-AD-D-OTRA-SUITE-1" },
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.ClaveID = "clave:denegacion:2026-08" },
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) {
			c.Audiencia = "vec-diputacion/otro/vec/autorizacion-denegacion"
		},
	} {
		candidata := base
		cambio(&candidata)
		huella, err := HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1(
			candidata,
			decision,
			referencia,
		)
		if err != nil || huella == huellaBase {
			t.Fatalf("cabecera no ligada: huella=%q err=%v", huella, err)
		}
	}

	for _, cambio := range []func(*CabeceraAtestacionDenegacionAutorizacionV1){
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.FormatoVersion = 0 },
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.Suite = "VEC-AD-D-*" },
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.ClaveID = "clave:ñ" },
		func(c *CabeceraAtestacionDenegacionAutorizacionV1) { c.Audiencia = "vec\nproduccion" },
	} {
		candidata := base
		cambio(&candidata)
		if _, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
			candidata,
			decision,
			referencia,
		); !errors.Is(err, ErrConfiguracionAccesoInvalida) ||
			!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
			t.Fatalf("cabecera ambigua aceptada: %v", err)
		}
	}
}

func TestMensajeAtestacionDenegacionAutorizacionV1RechazaColeccionesNoCanonicas(t *testing.T) {
	cabecera := cabeceraAtestacionDenegacionAutorizacionV1Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	base := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	cambios := []func(*DecisionAutorizacion){
		func(d *DecisionAutorizacion) { d.Accion = "bolsa.*" },
		func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasRefs[0], d.PoliticasEvaluadasRefs[1] =
				d.PoliticasEvaluadasRefs[1], d.PoliticasEvaluadasRefs[0]
		},
		func(d *DecisionAutorizacion) {
			d.CamposPermitidos[0], d.CamposPermitidos[1] =
				d.CamposPermitidos[1], d.CamposPermitidos[0]
		},
		func(d *DecisionAutorizacion) {
			delete(d.PoliticasEvaluadasHuellasSHA256, d.PoliticasEvaluadasRefs[0])
		},
		func(d *DecisionAutorizacion) {
			d.Obligaciones = make([]string, maximoElementosAutorizacion+1)
		},
	}
	for indice, cambio := range cambios {
		t.Run(strconv.Itoa(indice), func(t *testing.T) {
			decision := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			cambio(&decision)
			if _, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
				cabecera,
				decision,
				referencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("coleccion no canonica aceptada: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionDenegacionAutorizacionV1AplicaLimiteExacto512KiB(t *testing.T) {
	if !limitesEscritorAtestacionDenegacionAutorizacionV1Compatibles(
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
		TamanoMaximoMensajeAtestacionAutorizacionV1,
	) {
		t.Fatal("el limite VEC-AD-D-1 diverge del escritor binario")
	}
	for _, limites := range [][2]int{
		{TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 - 1, TamanoMaximoMensajeAtestacionAutorizacionV1},
		{TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1, TamanoMaximoMensajeAtestacionAutorizacionV1 - 1},
		{TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 + 1, TamanoMaximoMensajeAtestacionAutorizacionV1 + 1},
	} {
		if limitesEscritorAtestacionDenegacionAutorizacionV1Compatibles(
			limites[0],
			limites[1],
		) {
			t.Fatalf("limites divergentes aceptados: %v", limites)
		}
	}

	for _, objetivo := range []int{
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 - 1,
		TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1,
	} {
		t.Run(strconv.Itoa(objetivo)+"_bytes", func(t *testing.T) {
			decision, referencia := decisionAtestacionDenegacionAutorizacionV1ConTamanoObjetivo(
				t,
				objetivo,
			)
			mensaje, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
				cabeceraAtestacionDenegacionAutorizacionV1Prueba(),
				decision,
				referencia,
			)
			if err != nil || len(mensaje) != objetivo {
				t.Fatalf("borde %d: longitud=%d err=%v", objetivo, len(mensaje), err)
			}
		})
	}

	objetivo := TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 + 1
	decision, referencia := decisionAtestacionDenegacionAutorizacionV1ConTamanoObjetivo(
		t,
		objetivo,
	)
	if _, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(),
		decision,
		referencia,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("borde %d aceptado: %v", objetivo, err)
	}
}

func cabeceraAtestacionDenegacionAutorizacionV1Prueba() CabeceraAtestacionDenegacionAutorizacionV1 {
	return CabeceraAtestacionDenegacionAutorizacionV1{
		FormatoVersion: VersionFormatoAtestacionDenegacionAutorizacionV1,
		Suite:          "VEC-AD-D-PRUEBA-1",
		ClaveID:        "clave:denegacion:prueba:2026-07",
		Audiencia:      "vec-diputacion/pruebas/vec/autorizacion-denegacion",
	}
}

func decisionAtestacionDenegacionAutorizacionV1Prueba(
	t *testing.T,
	referencia ReferenciaEntradaCatalogo,
) DecisionAutorizacion {
	t.Helper()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	decision.Concedida = false
	decision.Codigo = "denegacion_por_defecto"
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("denegacion V2 de prueba invalida: %v", err)
	}
	return decision
}

func decisionAtestacionDenegacionAutorizacionV1ConTamanoObjetivo(
	t *testing.T,
	objetivo int,
) (DecisionAutorizacion, ReferenciaEntradaCatalogo) {
	t.Helper()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionDenegacionAutorizacionV1Prueba(t, referencia)
	decision.CamposPermitidos = listaAjustableAtestacionAutorizacionV1Prueba("c")
	decision.Obligaciones = listaAjustableAtestacionAutorizacionV1Prueba("o")
	mensajeBase, err := SerializarMensajeAtestacionDenegacionAutorizacionV1(
		cabeceraAtestacionDenegacionAutorizacionV1Prueba(),
		decision,
		referencia,
	)
	if err != nil || len(mensajeBase) > objetivo {
		t.Fatalf("fixture base de %d para objetivo %d: %v", len(mensajeBase), objetivo, err)
	}

	restante := objetivo - len(mensajeBase)
	for _, lista := range []*[]string{&decision.CamposPermitidos, &decision.Obligaciones} {
		for indice := range *lista {
			capacidad := 512 - len((*lista)[indice])
			aumento := min(restante, capacidad)
			(*lista)[indice] += strings.Repeat("x", aumento)
			restante -= aumento
			if restante == 0 {
				if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
					t.Fatalf("fixture ajustada invalida para %d: %v", objetivo, err)
				}
				return decision, referencia
			}
		}
	}
	t.Fatalf("fixture sin capacidad para objetivo %d: faltan %d bytes", objetivo, restante)
	return DecisionAutorizacion{}, ReferenciaEntradaCatalogo{}
}
