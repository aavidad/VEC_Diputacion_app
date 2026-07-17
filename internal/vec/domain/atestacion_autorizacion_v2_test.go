package domain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestMensajeAtestacionAutorizacionV2VectorDeterministaYEsquemaCerrado(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	original := clonarDecisionAtestacionAutorizacionV1Prueba(decision)

	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
	if err != nil {
		t.Fatalf("serializar mensaje VEC-AD-2: %v", err)
	}
	if !reflect.DeepEqual(decision, original) {
		t.Fatal("la serializacion VEC-AD-2 modifico la decision recibida")
	}
	prefijo := append([]byte(EsquemaMensajeAtestacionAutorizacionV2), 0)
	if !bytes.HasPrefix(mensaje, prefijo) ||
		bytes.HasPrefix(mensaje, append([]byte(EsquemaMensajeAtestacionAutorizacionV1), 0)) {
		t.Fatalf("dominio criptografico VEC-AD-2 ausente o cruzado: %x", mensaje[:min(len(mensaje), len(prefijo))])
	}
	posicionVersion := len(prefijo)
	if version := binary.BigEndian.Uint16(mensaje[posicionVersion : posicionVersion+2]); version != VersionFormatoAtestacionAutorizacionV2 {
		t.Fatalf("version binaria = %d", version)
	}
	if longitud := binary.BigEndian.Uint64(mensaje[len(mensaje)-8:]); longitud != uint64(len(mensaje)) {
		t.Fatalf("longitud final = %d; bytes completos = %d", longitud, len(mensaje))
	}

	huella, err := HuellaSHA256MensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
	if err != nil {
		t.Fatalf("calcular huella VEC-AD-2: %v", err)
	}
	const longitudEsperada = 2326
	const huellaEsperada = "b095845f68d24df46361f110fa3dbfce82202d8021a87749ad054ef398289eab"
	if len(mensaje) != longitudEsperada || huella != huellaEsperada {
		t.Fatalf("cambio incompatible del vector VEC-AD-2: longitud=%d huella=%s", len(mensaje), huella)
	}

	// El orden fisico de un mapa Go no forma parte de su valor canonico.
	otra := clonarDecisionAtestacionAutorizacionV1Prueba(decision)
	otra.PoliticasEvaluadasHuellasSHA256 = map[string]string{
		otra.PoliticasEvaluadasRefs[1]: decision.PoliticasEvaluadasHuellasSHA256[otra.PoliticasEvaluadasRefs[1]],
		otra.PoliticasEvaluadasRefs[0]: decision.PoliticasEvaluadasHuellasSHA256[otra.PoliticasEvaluadasRefs[0]],
	}
	otra.PoliticasHuellasSHA256 = map[string]string{
		otra.PoliticasRefs[0]: decision.PoliticasHuellasSHA256[otra.PoliticasRefs[0]],
	}
	otroMensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, otra, referencia)
	if err != nil || !bytes.Equal(mensaje, otroMensaje) {
		t.Fatalf("el orden fisico de mapas altero VEC-AD-2: err=%v", err)
	}
}

func TestEsquemaDecisionAtestacionAutorizacionV2EsExhaustivoYFallaCerrado(t *testing.T) {
	const numeroCamposCongelados = 35
	tipo := reflect.TypeOf(DecisionAutorizacion{})
	if tipo.NumField() != numeroCamposCongelados ||
		len(camposDecisionAtestacionAutorizacionV2) != numeroCamposCongelados ||
		!comprobarEsquemaDecisionAtestacionAutorizacionV2() {
		t.Fatalf(
			"DecisionAutorizacion cambio sin ampliar VEC-AD-2: campos=%d contrato=%d",
			tipo.NumField(), len(camposDecisionAtestacionAutorizacionV2),
		)
	}
	for indice, esperado := range camposDecisionAtestacionAutorizacionV2 {
		campo := tipo.Field(indice)
		if campo.Name != esperado.nombreGo ||
			strings.Split(campo.Tag.Get("json"), ",")[0] != esperado.etiqueta {
			t.Fatalf("campo %d no congelado: %s/%q", indice, campo.Name, campo.Tag.Get("json"))
		}
	}
}

func TestMensajeAtestacionAutorizacionV2OrdenBinarioIncluyeDecisionYMotivo(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
	if err != nil {
		t.Fatalf("serializar VEC-AD-2: %v", err)
	}

	lector := nuevoLectorAtestacionAutorizacionV1Prueba(t, mensaje)
	lector.exigirBytes(append([]byte(EsquemaMensajeAtestacionAutorizacionV2), 0))
	lector.exigirUint16(cabecera.FormatoVersion)
	lector.exigirTexto(cabecera.Suite)
	lector.exigirTexto(cabecera.ClaveID)
	lector.exigirTexto(cabecera.Audiencia)
	lector.exigirTexto(decision.DecisionRef)
	lector.exigirBooleano(decision.Concedida)
	lector.exigirTexto(decision.Codigo)
	lector.exigirTexto(decision.PrincipalID)
	lector.exigirTexto(decision.PerfilActivoRef)
	lector.exigirTexto(decision.Accion)
	lector.exigirTexto(decision.RecursoRef)
	lector.exigirTexto(decision.ModuloID)
	lector.exigirTexto(decision.TipoRecurso)
	lector.exigirTexto(decision.ContextoRecursoHuellaSHA256)
	lector.exigirTexto(decision.Finalidad)
	lector.exigirTexto(decision.CorrelacionRef)
	lector.exigirTexto(decision.EsquemaHuellaSolicitud)
	lector.exigirTexto(decision.SolicitudHuellaSHA256)
	lector.exigirTexto(decision.EsquemaHuellaMotivo)
	lector.exigirTexto(decision.MotivoHuellaSHA256)
	exigirVinculoAtestacionAutorizacionV2Prueba(t, lector, decision.VinculoAutenticacionActor)
	lector.exigirTexto(decision.AsignacionRef)
	lector.exigirTexto(decision.AsignacionHuellaSHA256)
	lector.exigirTexto(decision.VersionRolRef)
	lector.exigirTexto(decision.VersionRolHuellaSHA256)
	lector.exigirTexto(decision.ControlVigenciaVersionRolRef)
	lector.exigirUint64(decision.ControlVigenciaVersionRolRevision)
	lector.exigirTexto(decision.ControlVigenciaVersionRolHuellaSHA256)
	lector.exigirUint64(decision.RevisionCatalogoPoliticas)
	lector.exigirTexto(decision.CatalogoPoliticasHuellaSHA256)
	lector.exigirLista(decision.PoliticasEvaluadasRefs)
	lector.exigirMapa(decision.PoliticasEvaluadasHuellasSHA256)
	lector.exigirLista(decision.PoliticasRefs)
	lector.exigirMapa(decision.PoliticasHuellasSHA256)
	lector.exigirTexto(string(decision.GarantiaMinima))
	lector.exigirLista(decision.CamposPermitidos)
	lector.exigirLista(decision.Obligaciones)
	lector.exigirInstante(decision.EmitidaEn)
	lector.exigirInstante(decision.ValidaHasta)
	lector.exigirTexto(referencia.CatalogoID)
	if version := lector.leerUint64(); version != uint64(referencia.CatalogoVersion) {
		t.Fatalf("version de catalogo = %d; esperada %d", version, referencia.CatalogoVersion)
	}
	lector.exigirTexto(referencia.CatalogoHuellaSHA256)
	lector.exigirTexto(referencia.EntradaClave)
	lector.exigirLongitudFinal()
}

func TestMensajeAtestacionAutorizacionV2LigaCadaCompromiso(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	base := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	mensajeBase, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, base, referencia)
	if err != nil {
		t.Fatalf("crear mensaje base: %v", err)
	}

	cambios := []struct {
		nombre       string
		debeRechazar bool
		mutar        func(*DecisionAutorizacion)
	}{
		{"esquema_solicitud", true, func(d *DecisionAutorizacion) { d.EsquemaHuellaSolicitud = EsquemaMensajeAtestacionAutorizacionV1 }},
		{"huella_solicitud", false, func(d *DecisionAutorizacion) { d.SolicitudHuellaSHA256 = strings.Repeat("8", 64) }},
		{"esquema_motivo", true, func(d *DecisionAutorizacion) { d.EsquemaHuellaMotivo = EsquemaMensajeAtestacionAutorizacionV1 }},
		{"huella_motivo", true, func(d *DecisionAutorizacion) { d.MotivoHuellaSHA256 = strings.Repeat("9", 64) }},
	}
	for _, cambio := range cambios {
		t.Run(cambio.nombre, func(t *testing.T) {
			decision := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			cambio.mutar(&decision)
			mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
			if cambio.debeRechazar {
				if !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
					t.Fatalf("compromiso incoherente aceptado: %v", err)
				}
				return
			}
			if err != nil || bytes.Equal(mensaje, mensajeBase) {
				t.Fatalf("compromiso valido no quedo ligado: err=%v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV2LigaCadaCoordenadaMotivo(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	baseReferencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	baseDecision := decisionAtestacionAutorizacionV2Prueba(t, baseReferencia)
	mensajeBase, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, baseDecision, baseReferencia)
	if err != nil {
		t.Fatalf("crear mensaje base: %v", err)
	}

	cambios := []struct {
		nombre string
		mutar  func(*ReferenciaEntradaCatalogo)
	}{
		{"catalogo_id", func(r *ReferenciaEntradaCatalogo) { r.CatalogoID = "motivos_autorizacion_rrhh_2" }},
		{"catalogo_version", func(r *ReferenciaEntradaCatalogo) { r.CatalogoVersion++ }},
		{"catalogo_huella_publicada", func(r *ReferenciaEntradaCatalogo) { r.CatalogoHuellaSHA256 = strings.Repeat("8", 64) }},
		{"entrada_clave", func(r *ReferenciaEntradaCatalogo) { r.EntradaClave = "motivo_33333333333333333333333333333333" }},
	}
	for _, cambio := range cambios {
		t.Run(cambio.nombre, func(t *testing.T) {
			referencia := baseReferencia
			cambio.mutar(&referencia)
			if _, err := SerializarMensajeAtestacionAutorizacionV2(
				cabecera, baseDecision, referencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("cruce de motivo aceptado: %v", err)
			}

			decision := clonarDecisionAtestacionAutorizacionV1Prueba(baseDecision)
			decision.MotivoHuellaSHA256, err = HuellaSHA256MotivoAutorizacionV2(referencia)
			if err != nil {
				t.Fatalf("recalcular compromiso: %v", err)
			}
			mensaje, err := SerializarMensajeAtestacionAutorizacionV2(cabecera, decision, referencia)
			if err != nil || bytes.Equal(mensaje, mensajeBase) {
				t.Fatalf("coordenada valida no quedo ligada: err=%v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV2RechazaV1DenegacionesCrucesYCeros(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	referenciaA := referenciaMotivoAtestacionAutorizacionV2Prueba()
	referenciaB := referenciaA
	referenciaB.EntradaClave = "motivo_44444444444444444444444444444444"
	decision := decisionAtestacionAutorizacionV2Prueba(t, referenciaA)

	casos := []struct {
		nombre     string
		cabecera   CabeceraAtestacionAutorizacionV2
		decision   DecisionAutorizacion
		referencia ReferenciaEntradaCatalogo
	}{
		{"decision_V1", cabecera, decisionAtestacionAutorizacionV1Prueba(t), referenciaA},
		{"cabecera_V1", CabeceraAtestacionAutorizacionV2{
			FormatoVersion: VersionFormatoAtestacionAutorizacionV1,
			Suite:          "VEC-AD-PRUEBA-1",
			ClaveID:        "clave:prueba:2026-01",
			Audiencia:      "vec-diputacion/pruebas/vec/autorizacion",
		}, decision, referenciaA},
		{"cruce_motivo", cabecera, decision, referenciaB},
		{"cabecera_cero", CabeceraAtestacionAutorizacionV2{}, decision, referenciaA},
		{"decision_cero", cabecera, DecisionAutorizacion{}, referenciaA},
		{"motivo_cero", cabecera, decision, ReferenciaEntradaCatalogo{}},
	}
	denegada := clonarDecisionAtestacionAutorizacionV1Prueba(decision)
	denegada.Concedida = false
	denegada.Codigo = "denegada"
	casos = append(casos, struct {
		nombre     string
		cabecera   CabeceraAtestacionAutorizacionV2
		decision   DecisionAutorizacion
		referencia ReferenciaEntradaCatalogo
	}{"denegacion", cabecera, denegada, referenciaA})

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := SerializarMensajeAtestacionAutorizacionV2(
				caso.cabecera, caso.decision, caso.referencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("entrada incompatible aceptada: %v", err)
			}
		})
	}
}

func TestCabeceraAtestacionAutorizacionV2EsNominalYLigaConfiguracion(t *testing.T) {
	if reflect.TypeOf(CabeceraAtestacionAutorizacionV1{}) == reflect.TypeOf(CabeceraAtestacionAutorizacionV2{}) {
		t.Fatal("V1 y V2 comparten tipo de cabecera")
	}
	base := cabeceraAtestacionAutorizacionV2Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	huellaBase, err := HuellaSHA256MensajeAtestacionAutorizacionV2(base, decision, referencia)
	if err != nil {
		t.Fatalf("huella base: %v", err)
	}
	cambiosValidos := []struct {
		nombre string
		mutar  func(*CabeceraAtestacionAutorizacionV2)
	}{
		{"suite", func(c *CabeceraAtestacionAutorizacionV2) { c.Suite = "VEC-AD-OTRA-SUITE-2" }},
		{"clave", func(c *CabeceraAtestacionAutorizacionV2) { c.ClaveID = "clave:prueba:2026-03" }},
		{"audiencia", func(c *CabeceraAtestacionAutorizacionV2) { c.Audiencia = "vec-diputacion/otro/vec/autorizacion-v2" }},
	}
	for _, cambio := range cambiosValidos {
		t.Run(cambio.nombre, func(t *testing.T) {
			candidata := base
			cambio.mutar(&candidata)
			huella, err := HuellaSHA256MensajeAtestacionAutorizacionV2(candidata, decision, referencia)
			if err != nil || huella == huellaBase {
				t.Fatalf("cabecera no ligada: huella=%q err=%v", huella, err)
			}
		})
	}

	invalidas := []struct {
		nombre string
		mutar  func(*CabeceraAtestacionAutorizacionV2)
	}{
		{"version_cero", func(c *CabeceraAtestacionAutorizacionV2) { c.FormatoVersion = 0 }},
		{"version_V1", func(c *CabeceraAtestacionAutorizacionV2) { c.FormatoVersion = VersionFormatoAtestacionAutorizacionV1 }},
		{"suite_ausente", func(c *CabeceraAtestacionAutorizacionV2) { c.Suite = "" }},
		{"suite_comodin", func(c *CabeceraAtestacionAutorizacionV2) { c.Suite = "VEC-AD-*" }},
		{"clave_unicode", func(c *CabeceraAtestacionAutorizacionV2) { c.ClaveID = "clave:ñ" }},
		{"audiencia_control", func(c *CabeceraAtestacionAutorizacionV2) { c.Audiencia = "vec\nproduccion" }},
		{"audiencia_grande", func(c *CabeceraAtestacionAutorizacionV2) { c.Audiencia = strings.Repeat("a", 513) }},
	}
	for _, caso := range invalidas {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := base
			caso.mutar(&candidata)
			if _, err := SerializarMensajeAtestacionAutorizacionV2(
				candidata, decision, referencia,
			); !errors.Is(err, ErrConfiguracionAccesoInvalida) ||
				!errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("cabecera ambigua aceptada: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV2RechazaListasYMapasNoCanonicos(t *testing.T) {
	cabecera := cabeceraAtestacionAutorizacionV2Prueba()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	base := decisionAtestacionAutorizacionV2Prueba(t, referencia)

	politicasAplicadasDesordenables := func(d *DecisionAutorizacion) {
		d.PoliticasRefs = append([]string(nil), d.PoliticasEvaluadasRefs...)
		d.PoliticasHuellasSHA256 = map[string]string{
			d.PoliticasRefs[0]: d.PoliticasEvaluadasHuellasSHA256[d.PoliticasRefs[0]],
			d.PoliticasRefs[1]: d.PoliticasEvaluadasHuellasSHA256[d.PoliticasRefs[1]],
		}
		d.PoliticasRefs[0], d.PoliticasRefs[1] = d.PoliticasRefs[1], d.PoliticasRefs[0]
	}
	cambios := []struct {
		nombre string
		mutar  func(*DecisionAutorizacion)
	}{
		{"comodin", func(d *DecisionAutorizacion) { d.Accion = "bolsa.*" }},
		{"politicas_evaluadas_desordenadas", func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasRefs[0], d.PoliticasEvaluadasRefs[1] = d.PoliticasEvaluadasRefs[1], d.PoliticasEvaluadasRefs[0]
		}},
		{"politicas_aplicadas_desordenadas", politicasAplicadasDesordenables},
		{"campos_desordenados", func(d *DecisionAutorizacion) {
			d.CamposPermitidos[0], d.CamposPermitidos[1] = d.CamposPermitidos[1], d.CamposPermitidos[0]
		}},
		{"obligaciones_desordenadas", func(d *DecisionAutorizacion) {
			d.Obligaciones[0], d.Obligaciones[1] = d.Obligaciones[1], d.Obligaciones[0]
		}},
		{"lista_UTF8_invalida", func(d *DecisionAutorizacion) { d.CamposPermitidos[0] = string([]byte{0xff}) }},
		{"lista_duplicada", func(d *DecisionAutorizacion) { d.Obligaciones[1] = d.Obligaciones[0] }},
		{"mapa_evaluado_sin_clave", func(d *DecisionAutorizacion) { delete(d.PoliticasEvaluadasHuellasSHA256, d.PoliticasEvaluadasRefs[0]) }},
		{"mapa_aplicado_huella_invalida", func(d *DecisionAutorizacion) { d.PoliticasHuellasSHA256[d.PoliticasRefs[0]] = "no_sha256" }},
		{"mapa_adicional", func(d *DecisionAutorizacion) { d.PoliticasHuellasSHA256["politica:ajena:v1"] = strings.Repeat("a", 64) }},
		{"lista_desbordada", func(d *DecisionAutorizacion) {
			d.CamposPermitidos = make([]string, maximoElementosAutorizacion+1)
		}},
		{"mapa_desbordado", func(d *DecisionAutorizacion) {
			d.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, maximoElementosAutorizacion+1)
			for indice := 0; indice <= maximoElementosAutorizacion; indice++ {
				d.PoliticasEvaluadasHuellasSHA256[string(rune(0x1000+indice))] = strings.Repeat("a", 64)
			}
		}},
	}
	for _, cambio := range cambios {
		t.Run(cambio.nombre, func(t *testing.T) {
			decision := clonarDecisionAtestacionAutorizacionV1Prueba(base)
			cambio.mutar(&decision)
			if _, err := SerializarMensajeAtestacionAutorizacionV2(
				cabecera, decision, referencia,
			); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
				t.Fatalf("coleccion no canonica aceptada: %v", err)
			}
		})
	}
}

func TestMensajeAtestacionAutorizacionV2AplicaLimiteExacto512KiB(t *testing.T) {
	if !limitesEscritorAtestacionAutorizacionV2Compatibles(
		TamanoMaximoMensajeAtestacionAutorizacionV2,
		TamanoMaximoMensajeAtestacionAutorizacionV1,
	) {
		t.Fatalf(
			"limites incompatibles: V2=%d escritorV1=%d",
			TamanoMaximoMensajeAtestacionAutorizacionV2,
			TamanoMaximoMensajeAtestacionAutorizacionV1,
		)
	}
	for _, limites := range [][2]int{
		{TamanoMaximoMensajeAtestacionAutorizacionV2 - 1, TamanoMaximoMensajeAtestacionAutorizacionV1},
		{TamanoMaximoMensajeAtestacionAutorizacionV2, TamanoMaximoMensajeAtestacionAutorizacionV1 - 1},
		{TamanoMaximoMensajeAtestacionAutorizacionV2 + 1, TamanoMaximoMensajeAtestacionAutorizacionV1 + 1},
	} {
		if limitesEscritorAtestacionAutorizacionV2Compatibles(limites[0], limites[1]) {
			t.Fatalf("acoplamiento de limites divergente aceptado: %v", limites)
		}
	}

	for _, objetivo := range []int{
		TamanoMaximoMensajeAtestacionAutorizacionV2 - 1,
		TamanoMaximoMensajeAtestacionAutorizacionV2,
	} {
		t.Run(strconv.Itoa(objetivo)+"_bytes", func(t *testing.T) {
			decision, referencia := decisionAtestacionAutorizacionV2ConTamanoObjetivo(t, objetivo)
			mensaje, err := SerializarMensajeAtestacionAutorizacionV2(
				cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
			)
			if err != nil || len(mensaje) != objetivo {
				t.Fatalf("borde %d: longitud=%d err=%v", objetivo, len(mensaje), err)
			}
		})
	}

	objetivo := TamanoMaximoMensajeAtestacionAutorizacionV2 + 1
	decision, referencia := decisionAtestacionAutorizacionV2ConTamanoObjetivo(t, objetivo)
	if _, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("borde %d aceptado: %v", objetivo, err)
	}
}

func TestMensajeAtestacionAutorizacionV2NoTruncaVersionCatalogoEn64Bits(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("el valor mayor que uint32 no es representable por ReferenciaEntradaCatalogo en 386")
	}
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	versionFueraDeUint32 := uint64(math.MaxUint32)
	versionFueraDeUint32++
	referencia.CatalogoVersion = int(versionFueraDeUint32)
	decision := decisionAtestacionAutorizacionV2Prueba(t, referenciaMotivoAtestacionAutorizacionV2Prueba())
	if _, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
	); !errors.Is(err, ErrMensajeAtestacionAutorizacionInvalido) {
		t.Fatalf("version fuera del perfil portable fue truncada o aceptada: %v", err)
	}
}

func cabeceraAtestacionAutorizacionV2Prueba() CabeceraAtestacionAutorizacionV2 {
	return CabeceraAtestacionAutorizacionV2{
		FormatoVersion: VersionFormatoAtestacionAutorizacionV2,
		Suite:          "VEC-AD-PRUEBA-2",
		ClaveID:        "clave:prueba:2026-02",
		Audiencia:      "vec-diputacion/pruebas/vec/autorizacion-v2",
	}
}

func referenciaMotivoAtestacionAutorizacionV2Prueba() ReferenciaEntradaCatalogo {
	return ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_rrhh",
		CatalogoVersion:      3,
		CatalogoHuellaSHA256: strings.Repeat("6", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
}

func decisionAtestacionAutorizacionV2Prueba(
	t *testing.T,
	referencia ReferenciaEntradaCatalogo,
) DecisionAutorizacion {
	t.Helper()
	decision := decisionAtestacionAutorizacionV1Prueba(t)
	decision.CorrelacionRef = "correlacion_22222222222222222222222222222222"
	decision.EsquemaHuellaSolicitud = EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = strings.Repeat("7", 64)
	decision.EsquemaHuellaMotivo = EsquemaHuellaMotivoAutorizacionV2
	var err error
	decision.MotivoHuellaSHA256, err = HuellaSHA256MotivoAutorizacionV2(referencia)
	if err != nil {
		t.Fatalf("crear compromiso de motivo: %v", err)
	}
	if err := decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision V2 de prueba invalida: %v", err)
	}
	return decision
}

func decisionAtestacionAutorizacionV2ConTamanoObjetivo(
	t *testing.T,
	objetivo int,
) (DecisionAutorizacion, ReferenciaEntradaCatalogo) {
	t.Helper()
	referencia := referenciaMotivoAtestacionAutorizacionV2Prueba()
	decision := decisionAtestacionAutorizacionV2Prueba(t, referencia)
	decision.CamposPermitidos = listaAjustableAtestacionAutorizacionV1Prueba("c")
	decision.Obligaciones = listaAjustableAtestacionAutorizacionV1Prueba("o")
	mensajeBase, err := SerializarMensajeAtestacionAutorizacionV2(
		cabeceraAtestacionAutorizacionV2Prueba(), decision, referencia,
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

func exigirVinculoAtestacionAutorizacionV2Prueba(
	t *testing.T,
	lector *lectorAtestacionAutorizacionV1Prueba,
	vinculo VinculoAutenticacionActorV1,
) {
	t.Helper()
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatalf("obtener vinculo: %v", err)
	}
	lector.exigirUint16(datos.BloqueVersion)
	lector.exigirTexto(datos.AutenticacionRef)
	lector.exigirTexto(datos.AutenticacionHuellaSHA256)
	lector.exigirTexto(datos.AsercionRef)
	lector.exigirTexto(datos.SesionRef)
	lector.exigirTexto(datos.ControlSesionRef)
	lector.exigirUint64(datos.ControlSesionRevision)
	lector.exigirTexto(datos.ControlSesionHuellaSHA256)
	lector.exigirTexto(datos.CuentaRef)
	lector.exigirTexto(datos.CuentaOrdinariaRef)
	lector.exigirTexto(datos.PrincipalID)
	lector.exigirTexto(datos.PerfilActivoRef)
	lector.exigirBooleano(datos.CuentaPrivilegiada)
	lector.exigirTexto(string(datos.Superficie))
	lector.exigirTexto(string(datos.MetodoObservado))
	lector.exigirTexto(string(datos.GarantiaObservada))
	lector.exigirTexto(datos.PoliticaGarantiaRef)
	lector.exigirTexto(datos.PoliticaGarantiaHuellaSHA256)
	lector.exigirInstante(datos.AutenticacionVerificadaEn)
	lector.exigirInstante(datos.SesionEmitidaEn)
	lector.exigirInstante(datos.SesionValidaHasta)
	lector.exigirInstante(datos.SesionRevalidadaEn)
	lector.exigirTexto(datos.ContextoActorRef)
	lector.exigirUint64(datos.ContextoActorVersion)
	lector.exigirTexto(datos.ContextoActorHuellaSHA256)
}
