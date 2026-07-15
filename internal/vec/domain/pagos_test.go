package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var instanteBaseCobro = time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)

const (
	personaTecnicaCobroPrueba = "per_0123456789abcdefghijkl"
	personaSujetoCobroPrueba  = "per_abcdefghijkl0123456789"
	perfilCobroPrueba         = "prf_0123456789abcdefghijkl"
	representacionCobroPrueba = "rep_0123456789abcdefghijkl"
)

func huellaCobro(caracter byte) string { return strings.Repeat(string(caracter), 64) }

func recursoAutorizableCobroPrueba(referencia string) RecursoAutorizable {
	return RecursoAutorizable{Referencia: referencia, ModuloID: "pagos", Tipo: "orden_cobro"}
}

func altaOrdenCobroValida() AltaOrdenCobro {
	return AltaOrdenCobro{
		ID:                     "cob_0123456789abcdefghijkl",
		IndiceIdempotenciaHMAC: "hmac-sha256:pagos-v1:" + huellaCobro('a'),
		ExpedienteRef:          "expediente:seleccion:uno",
		SolicitudRef:           "solicitud:seleccion:uno",
		LiquidacionRef:         "liquidacion:seleccion:uno",
		Tarifa: ReferenciaTarifaCobro{
			TarifaID: "tasa_inscripcion", Version: 3, HuellaSHA256: huellaCobro('b'),
			ReglaCalculoRef: "regla:tasa-inscripcion:v3",
		},
		SujetoRef:             personaSujetoCobroPrueba,
		RepresentacionRef:     representacionCobroPrueba,
		Importe:               DineroCobro{UnidadesMenores: 2_735, Moneda: "EUR"},
		Concepto:              "Tasa de inscripcion en proceso selectivo",
		Finalidad:             "tramitar_solicitud_seleccion",
		CorrelacionRef:        "correlacion:cobro:uno",
		CreadaEn:              instanteBaseCobro,
		CaducaEn:              instanteBaseCobro.Add(30 * time.Minute),
		EvidenciaCreacionRef:  "evidencia:liquidacion:uno",
		HuellaEvidenciaSHA256: huellaCobro('c'),
		Motivo:                "Tasa determinada por la tarifa publicada",
	}
}

func decisionCobro(accion AccionCobro, recurso, finalidad, correlacion string, instante time.Time) DecisionAutorizacion {
	especificacion := especificacionesAutorizacionCobro[accion]
	huellaCatalogo, err := HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		panic(err)
	}
	huellaContexto, err := recursoAutorizableCobroPrueba(recurso).HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	vinculo, err := vinculoAutenticacionActorV1PruebaSinT(instante)
	if err != nil {
		panic(err)
	}
	return DecisionAutorizacion{
		DecisionRef: "decision:" + strings.ReplaceAll(string(accion), ".", ":"),
		Concedida:   true, Codigo: "concedida", PrincipalID: personaTecnicaCobroPrueba,
		PerfilActivoRef: perfilCobroPrueba, Accion: string(accion), RecursoRef: recurso,
		ModuloID: "pagos", TipoRecurso: "orden_cobro", ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: finalidad, CorrelacionRef: correlacion,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:tesoreria:uno", AsignacionHuellaSHA256: huellaCobro('d'),
		VersionRolRef: "rol:tesoreria:v3", VersionRolHuellaSHA256: huellaCobro('e'),
		ControlVigenciaVersionRolRef:      "rol:tesoreria:v3",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: huellaCobro('f'),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  especificacion.garantiaMinima,
		CamposPermitidos:                append([]string(nil), especificacion.campos...),
		EmitidaEn:                       instante.Add(-30 * time.Second), ValidaHasta: instante.Add(4 * time.Minute),
	}
}

type verificadorAutenticacionCobroPrueba struct {
	resultado ResultadoVerificacionAutenticacionCobro
	err       error
}

func (v verificadorAutenticacionCobroPrueba) VerificarAutenticacionCobro(ctx context.Context, _ SolicitudVerificacionAutenticacionCobro) (ResultadoVerificacionAutenticacionCobro, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoVerificacionAutenticacionCobro{}, err
	}
	return v.resultado, v.err
}

const sesionCobroPrueba = "ses_0123456789abcdefghijkl"

func huellaSesionCobroPrueba() string { return "hmac-sha256:sesion-v1:" + huellaCobro('9') }

func atestacionAutenticacionCobroPrueba(t *testing.T, instante time.Time) AtestacionAutenticacionCobro {
	t.Helper()
	atestacion, err := NuevaAtestacionAutenticacionCobro(context.Background(), verificadorAutenticacionCobroPrueba{
		resultado: ResultadoVerificacionAutenticacionCobro{
			PrincipalRef: personaTecnicaCobroPrueba, Metodo: AuthMethodCertificate,
			Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_0123456789abcdefghijkl",
			SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(),
			EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(10 * time.Minute),
		},
	}, sesionCobroPrueba, huellaSesionCobroPrueba(), instante)
	if err != nil {
		t.Fatalf("NuevaAtestacionAutenticacionCobro() error = %v", err)
	}
	return atestacion
}

func contextoCobro(t *testing.T, accion AccionCobro, recurso, finalidad, correlacion string, instante time.Time) ContextoAutorizacionCobro {
	t.Helper()
	contexto, err := NuevoContextoAutorizacionCobro(
		decisionCobro(accion, recurso, finalidad, correlacion, instante),
		atestacionAutenticacionCobroPrueba(t, instante), recursoAutorizableCobroPrueba(recurso), instante,
	)
	if err != nil {
		t.Fatalf("NuevoContextoAutorizacionCobro() error = %v", err)
	}
	return contexto
}

func contextoOrdenCobro(t *testing.T, orden OrdenCobro, accion AccionCobro, instante time.Time) ContextoAutorizacionCobro {
	t.Helper()
	return contextoCobro(t, accion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
}

func nuevaOrdenCobroPrueba(t *testing.T) OrdenCobro {
	t.Helper()
	alta := altaOrdenCobroValida()
	autorizacion := contextoCobro(t, AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
	orden, err := NuevaOrdenCobro(alta, autorizacion)
	if err != nil {
		t.Fatalf("NuevaOrdenCobro() error = %v", err)
	}
	return orden
}

func datosServidorCobro(orden OrdenCobro, sufijo string, caracter byte, instante time.Time) DatosEvidenciaServidorCobro {
	return DatosEvidenciaServidorCobro{
		EvidenciaRef:             "evidencia:pasarela:" + sufijo,
		HuellaSHA256:             huellaCobro(caracter),
		ConectorID:               "pasarela_corporativa",
		VersionConector:          2,
		OrdenRef:                 orden.ID,
		LiquidacionRef:           orden.LiquidacionRef,
		OperacionProveedorRef:    "operacion:opaca:uno",
		Importe:                  orden.Importe,
		Concepto:                 orden.Concepto,
		Codigo:                   "resultado_" + sufijo,
		MetodoAutenticacion:      MetodoAutenticacionCobroFirmaYTLSMutuo,
		Audiencia:                "vec.cobros",
		VerificacionRef:          "verificacion:pasarela:" + sufijo,
		HuellaVerificacionSHA256: huellaCobro('8'),
		EmitidaEn:                instante.Add(-time.Second),
		RecibidaEn:               instante,
		VerificadaEn:             instante,
	}
}

func enviarOrdenCobroPrueba(t *testing.T, orden OrdenCobro, instante time.Time) (OrdenCobro, EvidenciaInicioOperacionCobro) {
	t.Helper()
	evidencia, err := NuevaEvidenciaInicioOperacionCobroVerificada(datosServidorCobro(orden, "inicio", 'd', instante))
	if err != nil {
		t.Fatalf("crear evidencia de inicio: %v", err)
	}
	autorizacion := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instante)
	enviada, repetida, err := orden.RegistrarEnvio(evidencia, instante, autorizacion, "Inicio de operacion alojada")
	if err != nil || repetida {
		t.Fatalf("RegistrarEnvio() = repetida %v, error %v", repetida, err)
	}
	return enviada, evidencia
}

func aplicarResultadoCobroPrueba(t *testing.T, orden OrdenCobro, resultado ResultadoOperacionCobro, sufijo string, caracter byte, instante time.Time) (OrdenCobro, EvidenciaResultadoCobro) {
	t.Helper()
	evidencia, err := NuevaEvidenciaResultadoCobroVerificada(datosServidorCobro(orden, sufijo, caracter, instante), resultado)
	if err != nil {
		t.Fatalf("crear evidencia de resultado: %v", err)
	}
	autorizacion := contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instante)
	nueva, repetida, err := orden.AplicarResultadoServidor(evidencia, instante, autorizacion, "Resultado autenticado del servidor")
	if err != nil || repetida {
		t.Fatalf("AplicarResultadoServidor() = repetida %v, error %v", repetida, err)
	}
	return nueva, evidencia
}

func confirmarOrdenCobroPrueba(t *testing.T) OrdenCobro {
	t.Helper()
	orden := nuevaOrdenCobroPrueba(t)
	instanteCaducada := orden.CaducaEn.Add(time.Second)
	contextoInicioTardio := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instanteCaducada)
	if _, err := orden.PrepararInicioOperacion("retorno:uno", "notificacion:uno", instanteCaducada, contextoInicioTardio); !errors.Is(err, ErrTransicionCobroInvalida) {
		t.Fatalf("se preparo un inicio tras caducar: %v", err)
	}
	evidenciaInicioTardia, _ := NuevaEvidenciaInicioOperacionCobroVerificada(
		datosServidorCobro(orden, "inicio-tardio", 'd', instanteCaducada),
	)
	if _, _, err := orden.RegistrarEnvio(evidenciaInicioTardia, instanteCaducada, contextoInicioTardio, "Inicio tardio"); !errors.Is(err, ErrTransicionCobroInvalida) {
		t.Fatalf("se registro un inicio tras caducar: %v", err)
	}

	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	orden, _ = aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroConfirmado, "confirmado", 'e', instanteBaseCobro.Add(2*time.Minute))
	return orden
}

func TestDineroCobroYEstadosSonListasCerradas(t *testing.T) {
	for _, dinero := range []DineroCobro{{1, "EUR"}, {2_735, "EUR"}, {9_223_372_036_854_775_807, "JPY"}} {
		if err := dinero.Validar(); err != nil {
			t.Errorf("%+v deberia ser valido: %v", dinero, err)
		}
	}
	for _, dinero := range []DineroCobro{{0, "EUR"}, {-1, "EUR"}, {1, "eur"}, {1, "EU"}, {1, "EURO"}, {1, "12A"}} {
		if !errors.Is(dinero.Validar(), ErrDineroCobroInvalido) {
			t.Errorf("%+v deberia fallar cerrado", dinero)
		}
	}
	if !(DineroCobro{2_735, "EUR"}).Igual(DineroCobro{2_735, "EUR"}) ||
		(DineroCobro{2_735, "EUR"}).Igual(DineroCobro{2_736, "EUR"}) {
		t.Fatal("Igual() no compara importe y moneda exactamente")
	}
	for _, estado := range []EstadoCobro{
		EstadoCobroCreada, EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente,
		EstadoCobroConfirmada, EstadoCobroConciliada, EstadoCobroRechazada,
		EstadoCobroCancelada, EstadoCobroCaducada, EstadoCobroResultadoDesconocido,
		EstadoCobroDevolucionSolicitada, EstadoCobroDevolucionRechazada, EstadoCobroDevuelta,
		EstadoCobroDevolucionConciliada, EstadoCobroIncidenciaBloqueada,
	} {
		if !estado.Valido() {
			t.Errorf("estado declarado no reconocido: %q", estado)
		}
	}
	if EstadoCobro("nuevo_por_accidente").Valido() || AccionCobro("cobros.todo").Valida() ||
		AccionCobro(" cobros.orden.crear").Valida() || AccionCobro("cobros.orden.crear ").Valida() {
		t.Fatal("un estado o permiso no enumerado fue aceptado")
	}
	if TuplaHechoCobroValida(HechoCobroIncidenciaDetectada, EstadoCobroIncidenciaBloqueada, AccionCobroCrearOrden) ||
		TuplaHechoCobroValida(HechoCobroEvidenciaAdicional, EstadoCobro("estado_futuro"), AccionCobroIniciarOperacion) ||
		TuplaHechoCobroValida(HechoCobroEvidenciaAdicional, EstadoCobroConfirmada, AccionCobro("cobros.todo")) {
		t.Fatal("una tupla no enumerada de hecho, estado y accion fue aceptada")
	}
}

func TestCicloCompletoDerivaComandosDelAgregado(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	contextoInicio := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instanteBaseCobro.Add(30*time.Second))
	comandoInicio, err := orden.PrepararInicioOperacion("retorno:vec:uno", "notificacion:servidor:uno", instanteBaseCobro.Add(30*time.Second), contextoInicio)
	if err != nil || comandoInicio.Validar() != nil {
		t.Fatalf("comando de inicio invalido: %v", err)
	}
	datosInicio, _ := comandoInicio.Datos()
	if datosInicio.OrdenRef != orden.ID || !datosInicio.Importe.Igual(orden.Importe) || datosInicio.HuellaOrdenSHA256 != orden.HuellaEstadoSHA256 {
		t.Fatal("el comando de inicio no deriva exactamente del agregado")
	}

	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	orden, _ = aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroDesconocido, "desconocido", 'e', instanteBaseCobro.Add(2*time.Minute))
	orden, _ = aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroPendiente, "pendiente", 'f', instanteBaseCobro.Add(3*time.Minute))
	orden, _ = aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroConfirmado, "confirmado", '1', instanteBaseCobro.Add(4*time.Minute))

	autorizacionConciliar := contextoOrdenCobro(t, orden, AccionCobroConciliar, instanteBaseCobro.Add(5*time.Minute))
	comandoConciliar, err := orden.PrepararConciliacion(TipoConciliacionCobroIngreso, "cierre:recaudacion:uno", instanteBaseCobro.Add(5*time.Minute), autorizacionConciliar)
	if err != nil || comandoConciliar.Validar() != nil {
		t.Fatalf("comando de conciliacion invalido: %v", err)
	}
	evidenciaConciliar, err := NuevaEvidenciaConciliacionCobroVerificada(
		datosServidorCobro(orden, "conciliado", '2', instanteBaseCobro.Add(5*time.Minute)),
		TipoConciliacionCobroIngreso, "cierre:recaudacion:uno", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err := orden.AplicarConciliacionServidor(evidenciaConciliar, instanteBaseCobro.Add(5*time.Minute), autorizacionConciliar, "Coincidencia en cierre")
	if err != nil || repetida || orden.Estado != EstadoCobroConciliada {
		t.Fatalf("conciliacion = estado %q, repetida %v, error %v", orden.Estado, repetida, err)
	}

	solicitud := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:solicitud",
		HuellaEvidenciaSHA256:  huellaCobro('3'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('4'),
		Motivo:                 "Resolucion administrativa de devolucion", SolicitadaEn: instanteBaseCobro.Add(6 * time.Minute),
	}
	autorizacionDevolver := contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, solicitud.SolicitadaEn)
	orden, comandoDevolver, repetida, err := orden.SolicitarDevolucion(solicitud, autorizacionDevolver)
	if err != nil || repetida || comandoDevolver.Validar() != nil || orden.Estado != EstadoCobroDevolucionSolicitada {
		t.Fatalf("devolucion solicitada = estado %q, repetida %v, error %v", orden.Estado, repetida, err)
	}
	datosDevolver, _ := comandoDevolver.Datos()
	if datosDevolver.DevolucionRef != solicitud.DevolucionRef || !datosDevolver.Importe.Igual(orden.Importe) ||
		datosDevolver.OperacionProveedorRef != orden.OperacionProveedorRef {
		t.Fatal("el comando de devolucion no deriva exactamente del agregado")
	}

	for indice, caso := range []struct {
		resultado ResultadoDevolucionCobro
		sufijo    string
		estado    EstadoCobro
	}{
		{ResultadoDevolucionCobroDesconocido, "devolucion-desconocida", EstadoCobroDevolucionSolicitada},
		{ResultadoDevolucionCobroConfirmada, "devuelta", EstadoCobroDevuelta},
	} {
		instante := instanteBaseCobro.Add(time.Duration(7+indice) * time.Minute)
		datos := datosServidorCobro(orden, caso.sufijo, byte('5'+indice), instante)
		evidencia, errorEvidencia := NuevaEvidenciaResultadoDevolucionCobroVerificada(datos, solicitud.DevolucionRef, caso.resultado)
		if errorEvidencia != nil {
			t.Fatal(errorEvidencia)
		}
		autorizacion := contextoOrdenCobro(t, orden, AccionCobroProcesarDevolucion, instante)
		orden, repetida, err = orden.AplicarResultadoDevolucionServidor(evidencia, instante, autorizacion, "Resultado de devolucion autenticado")
		if err != nil || repetida || orden.Estado != caso.estado {
			t.Fatalf("resultado devolucion = estado %q, repetida %v, error %v", orden.Estado, repetida, err)
		}
	}

	instanteConciliacion := instanteBaseCobro.Add(9 * time.Minute)
	autorizacionConciliar = contextoOrdenCobro(t, orden, AccionCobroConciliar, instanteConciliacion)
	comandoConciliar, err = orden.PrepararConciliacion(TipoConciliacionCobroDevolucion, "cierre:devolucion:uno", instanteConciliacion, autorizacionConciliar)
	if err != nil || comandoConciliar.Validar() != nil {
		t.Fatalf("preparar conciliacion de devolucion: %v", err)
	}
	evidenciaConciliarDevolucion, err := NuevaEvidenciaConciliacionCobroVerificada(
		datosServidorCobro(orden, "devolucion-conciliada", '7', instanteConciliacion),
		TipoConciliacionCobroDevolucion, "cierre:devolucion:uno", solicitud.DevolucionRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err = orden.AplicarConciliacionServidor(evidenciaConciliarDevolucion, instanteConciliacion, autorizacionConciliar, "Devolucion conciliada")
	if err != nil || repetida || orden.Estado != EstadoCobroDevolucionConciliada || orden.Version != len(orden.Historial) {
		t.Fatalf("final = estado %q, version %d, hechos %d, repetida %v, error %v", orden.Estado, orden.Version, len(orden.Historial), repetida, err)
	}
	if err := orden.Validar(); err != nil {
		t.Fatalf("orden final invalida: %v", err)
	}
}

func TestCadenaHuellasYControlConcurrencia(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	anterior := huellaNulaCobro
	for indice, hecho := range orden.Historial {
		if hecho.HuellaEstadoAnteriorSHA256 != anterior {
			t.Fatalf("hecho %d no enlaza con la huella anterior", indice+1)
		}
		calculada, err := calcularHuellaHechoCobro(hecho)
		if err != nil || calculada != hecho.HuellaEstadoPosteriorSHA256 {
			t.Fatalf("hecho %d tiene huella no canonica", indice+1)
		}
		anterior = hecho.HuellaEstadoPosteriorSHA256
	}
	version, huella, err := orden.ControlConcurrencia()
	if err != nil || version != len(orden.Historial) || huella != anterior {
		t.Fatalf("control OCC inesperado: version=%d huella=%q err=%v", version, huella, err)
	}
}

func TestTodaMutacionDelAgregadoOHistoricoSeDetecta(t *testing.T) {
	base := confirmarOrdenCobroPrueba(t)
	mutaciones := []struct {
		nombre string
		muta   func(*OrdenCobro)
	}{
		{"id", func(o *OrdenCobro) { o.ID = "cob_abcdefghijkl0123456789" }},
		{"version esquema", func(o *OrdenCobro) { o.VersionEsquemaIntegridad++ }},
		{"version", func(o *OrdenCobro) { o.Version++ }},
		{"idempotencia", func(o *OrdenCobro) { o.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaCobro('f') }},
		{"expediente", func(o *OrdenCobro) { o.ExpedienteRef = "expediente:seleccion:dos" }},
		{"solicitud", func(o *OrdenCobro) { o.SolicitudRef = "solicitud:seleccion:dos" }},
		{"liquidacion", func(o *OrdenCobro) { o.LiquidacionRef = "liquidacion:seleccion:dos" }},
		{"tarifa id", func(o *OrdenCobro) { o.Tarifa.TarifaID = "tasa_alternativa" }},
		{"tarifa version", func(o *OrdenCobro) { o.Tarifa.Version++ }},
		{"tarifa huella", func(o *OrdenCobro) { o.Tarifa.HuellaSHA256 = huellaCobro('f') }},
		{"regla tarifa", func(o *OrdenCobro) { o.Tarifa.ReglaCalculoRef = "regla:tasa:v4" }},
		{"sujeto", func(o *OrdenCobro) { o.SujetoRef = "per_otra23456789abcdefghijk" }},
		{"representacion", func(o *OrdenCobro) { o.RepresentacionRef = "rep_otra23456789abcdefghijk" }},
		{"importe", func(o *OrdenCobro) { o.Importe.UnidadesMenores++ }},
		{"concepto", func(o *OrdenCobro) { o.Concepto = "Tasa distinta" }},
		{"finalidad", func(o *OrdenCobro) { o.Finalidad = "otra_finalidad" }},
		{"correlacion", func(o *OrdenCobro) { o.CorrelacionRef = "correlacion:cobro:dos" }},
		{"creacion", func(o *OrdenCobro) { o.CreadaEn = o.CreadaEn.Add(time.Second) }},
		{"caducidad", func(o *OrdenCobro) { o.CaducaEn = o.CaducaEn.Add(time.Minute) }},
		{"estado", func(o *OrdenCobro) { o.Estado = EstadoCobroRechazada }},
		{"actualizacion", func(o *OrdenCobro) { o.UltimaActualizacionEn = o.UltimaActualizacionEn.Add(time.Second) }},
		{"huella final", func(o *OrdenCobro) { o.HuellaEstadoSHA256 = huellaCobro('f') }},
		{"actor historico", func(o *OrdenCobro) { o.Historial[0].ActorRef = "persona:otra" }},
		{"version esquema historica", func(o *OrdenCobro) { o.Historial[0].VersionEsquemaIntegridad++ }},
		{"perfil historico", func(o *OrdenCobro) { o.Historial[0].PerfilActivoRef = "perfil:otro" }},
		{"accion historica", func(o *OrdenCobro) { o.Historial[0].AccionAutorizada = AccionCobroCancelar }},
		{"autorizacion historica", func(o *OrdenCobro) { o.Historial[0].AutorizacionRef = "decision:otra" }},
		{"atestacion historica", func(o *OrdenCobro) { o.Historial[0].AtestacionAutenticacionRef = "aut_otra23456789abcdefghijkl" }},
		{"sesion historica", func(o *OrdenCobro) { o.Historial[0].SesionRef = "ses_abcdefghijkl0123456789" }},
		{"huella sesion historica", func(o *OrdenCobro) { o.Historial[0].HuellaSesionHMAC = "hmac-sha256:sesion-v1:" + huellaCobro('8') }},
		{"metodo historico", func(o *OrdenCobro) { o.Historial[0].MetodoAutenticacion = AuthMethodDNIe }},
		{"garantia historica", func(o *OrdenCobro) { o.Historial[0].GarantiaAutenticacion = AuthAssuranceSubstantial }},
		{"motivo historico", func(o *OrdenCobro) { o.Historial[0].Motivo = "Otro motivo valido" }},
		{"evidencia historica", func(o *OrdenCobro) { o.Historial[1].HuellaEvidenciaSHA256 = huellaCobro('f') }},
		{"codigo historico", func(o *OrdenCobro) { o.Historial[1].CodigoResultado = "otro_codigo" }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := base.Clonar()
			caso.muta(&alterada)
			if !errors.Is(alterada.Validar(), ErrOrdenCobroInvalida) {
				t.Fatalf("la mutacion %q no fue detectada", caso.nombre)
			}
		})
	}
}

func TestIdempotenciaDeDevolucionTambienFormaParteDeLaCadena(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(3 * time.Minute)
	solicitud := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:cadena",
		HuellaEvidenciaSHA256:  huellaCobro('1'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('2'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instante,
	}
	orden, _, _, err := orden.SolicitarDevolucion(
		solicitud, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
	)
	if err != nil {
		t.Fatal(err)
	}
	alterada := orden.Clonar()
	nuevaIdempotencia := "hmac-sha256:devoluciones-v1:" + huellaCobro('3')
	alterada.IndiceIdempotenciaDevolucionHMAC = nuevaIdempotencia
	alterada.Historial[len(alterada.Historial)-1].IndiceIdempotenciaHMAC = nuevaIdempotencia
	if !errors.Is(alterada.Validar(), ErrOrdenCobroInvalida) {
		t.Fatal("se alteraron conjuntamente la idempotencia proyectada e historica sin romper la cadena")
	}
}

func TestReplayExactoEvidenciaAdicionalYConflictoBloqueante(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	orden, confirmacion := aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroConfirmado, "confirmacion-original", 'e', instanteBaseCobro.Add(2*time.Minute))
	versionConfirmada := orden.Version
	autorizacion := contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteBaseCobro.Add(2*time.Minute))
	repetidaOrden, repetida, err := orden.AplicarResultadoServidor(confirmacion, instanteBaseCobro.Add(2*time.Minute), autorizacion, "Replay exacto")
	if err != nil || !repetida || repetidaOrden.Version != versionConfirmada {
		t.Fatalf("replay exacto: version=%d repetida=%v error=%v", repetidaOrden.Version, repetida, err)
	}

	instanteAdicional := instanteBaseCobro.Add(3 * time.Minute)
	evidenciaAdicional, err := NuevaEvidenciaResultadoCobroVerificada(
		datosServidorCobro(orden, "confirmacion-adicional", 'f', instanteAdicional), ResultadoOperacionCobroConfirmado,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err = orden.AplicarResultadoServidor(
		evidenciaAdicional, instanteAdicional, contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteAdicional), "Consulta posterior",
	)
	if err != nil || repetida || orden.Version != versionConfirmada+1 || orden.Historial[len(orden.Historial)-1].Tipo != HechoCobroEvidenciaAdicional {
		t.Fatalf("evidencia adicional: version=%d repetida=%v error=%v", orden.Version, repetida, err)
	}

	instanteConflicto := instanteBaseCobro.Add(4 * time.Minute)
	evidenciaConflicto, err := NuevaEvidenciaResultadoCobroVerificada(
		datosServidorCobro(orden, "rechazo-conflictivo", '1', instanteConflicto), ResultadoOperacionCobroRechazado,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err = orden.AplicarResultadoServidor(
		evidenciaConflicto, instanteConflicto, contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteConflicto), "Resultado contradictorio",
	)
	ultimo := orden.Historial[len(orden.Historial)-1]
	if err != nil || repetida || orden.Estado != EstadoCobroIncidenciaBloqueada || ultimo.Tipo != HechoCobroIncidenciaDetectada ||
		ultimo.EvidenciaRelacionadaRef != "evidencia:pasarela:rechazo-conflictivo" {
		t.Fatalf("conflicto no bloqueo con trazabilidad: estado=%q hecho=%+v err=%v", orden.Estado, ultimo, err)
	}
}

func TestReferenciaReutilizadaConContenidoDistintoTambienBloquea(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	orden, original := aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroPendiente, "misma-referencia", 'e', instanteBaseCobro.Add(2*time.Minute))
	datosAlterados := datosServidorCobro(orden, "misma-referencia", 'f', instanteBaseCobro.Add(3*time.Minute))
	datosAlterados.Codigo = "contenido_alterado"
	alterada, err := NuevaEvidenciaResultadoCobroVerificada(datosAlterados, ResultadoOperacionCobroConfirmado)
	if err != nil || original.Validar() != nil {
		t.Fatal(err)
	}
	orden, repetida, err := orden.AplicarResultadoServidor(
		alterada, instanteBaseCobro.Add(3*time.Minute), contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteBaseCobro.Add(3*time.Minute)), "Colision de referencia",
	)
	if err != nil || repetida || orden.Estado != EstadoCobroIncidenciaBloqueada {
		t.Fatalf("la colision alterada no genero incidencia: estado=%q repetida=%v err=%v", orden.Estado, repetida, err)
	}
	ultimo := orden.Historial[len(orden.Historial)-1]
	if ultimo.EvidenciaRef == datosAlterados.EvidenciaRef || ultimo.EvidenciaRelacionadaRef != datosAlterados.EvidenciaRef {
		t.Fatalf("la incidencia no separa referencia propia y relacionada: %+v", ultimo)
	}
}

func TestReintentoDevolucionConMismaEvidenciaYOtraSemanticaBloquea(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(3 * time.Minute)
	solicitud := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:colision",
		HuellaEvidenciaSHA256:  huellaCobro('1'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('2'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instante,
	}
	orden, _, _, err := orden.SolicitarDevolucion(
		solicitud, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.IndiceIdempotenciaHMAC = "hmac-sha256:devoluciones-v1:" + huellaCobro('3')
	orden, comando, repetida, err := orden.SolicitarDevolucion(
		solicitud, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
	)
	if err != nil || repetida || comando.Validar() == nil || orden.Estado != EstadoCobroIncidenciaBloqueada {
		t.Fatalf("colision local no bloqueada: estado=%q repetida=%v err=%v", orden.Estado, repetida, err)
	}
}

func TestAutorizacionEsConcedidaExactaBreveYValorCeroDeniega(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(time.Minute)
	valida := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instante)
	if _, err := orden.PrepararInicioOperacion("retorno:uno", "notificacion:uno", instante, valida); err != nil {
		t.Fatalf("autorizacion valida rechazada: %v", err)
	}

	casos := []struct {
		nombre string
		muta   func(*ContextoAutorizacionCobro)
	}{
		{"valor cero", func(c *ContextoAutorizacionCobro) { *c = ContextoAutorizacionCobro{} }},
		{"otra accion", func(c *ContextoAutorizacionCobro) { c.datos.Accion = AccionCobroCancelar }},
		{"otro recurso", func(c *ContextoAutorizacionCobro) { c.datos.RecursoRef = "cob_abcdefghijkl0123456789" }},
		{"otra finalidad", func(c *ContextoAutorizacionCobro) { c.datos.Finalidad = "otra_finalidad" }},
		{"otra correlacion", func(c *ContextoAutorizacionCobro) { c.datos.CorrelacionRef = "correlacion:otra" }},
		{"garantia baja", func(c *ContextoAutorizacionCobro) { c.datos.Garantia = AuthAssuranceLow }},
		{"otro metodo", func(c *ContextoAutorizacionCobro) { c.datos.Metodo = AuthMethodClave }},
		{"otra autenticacion", func(c *ContextoAutorizacionCobro) { c.datos.AutenticacionRef = "sesion:otra" }},
		{"caducada", func(c *ContextoAutorizacionCobro) { c.datos.EvaluadaEn = c.datos.VigenteHasta }},
		{"demasiado larga", func(c *ContextoAutorizacionCobro) { c.datos.VigenteHasta = c.datos.VigenteDesde.Add(6 * time.Minute) }},
		{"sin decision", func(c *ContextoAutorizacionCobro) { c.datos.DecisionRef = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instante)
			caso.muta(&alterada)
			if _, err := orden.PrepararInicioOperacion("retorno:uno", "notificacion:uno", instante, alterada); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
				t.Fatalf("la autorizacion alterada fue aceptada: %v", err)
			}
		})
	}

	denegada := decisionCobro(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
	denegada.Concedida = false
	if _, err := NuevoContextoAutorizacionCobro(denegada, atestacionAutenticacionCobroPrueba(t, instante), recursoAutorizableCobroPrueba(orden.ID), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("una decision denegada produjo contexto: %v", err)
	}
	conObligacionNoImplementada := decisionCobro(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
	conObligacionNoImplementada.Obligaciones = []string{"doble_control"}
	if _, err := NuevoContextoAutorizacionCobro(conObligacionNoImplementada, atestacionAutenticacionCobroPrueba(t, instante), recursoAutorizableCobroPrueba(orden.ID), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("se ignoro una obligacion de autorizacion no implementada: %v", err)
	}
	if _, err := json.Marshal(valida); !errors.Is(err, ErrSerializacionAutorizacionCobro) {
		t.Fatalf("se serializo directamente un contexto interno: %v", err)
	}
}

func TestAutorizacionCobroExigeCamposExactosYAtestacionRealVinculada(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(time.Minute)
	decisionBase := decisionCobro(AccionCobroProcesarResultado, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)

	crearAtestacion := func(t *testing.T, resultado ResultadoVerificacionAutenticacionCobro) AtestacionAutenticacionCobro {
		t.Helper()
		atestacion, err := NuevaAtestacionAutenticacionCobro(
			context.Background(),
			verificadorAutenticacionCobroPrueba{resultado: resultado}, resultado.SesionRef, resultado.HuellaSesionHMAC, instante,
		)
		if err != nil {
			t.Fatalf("atestacion valida rechazada: %v", err)
		}
		return atestacion
	}
	resultadoValido := ResultadoVerificacionAutenticacionCobro{
		PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodCertificate,
		Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_0123456789abcdefghijkl",
		SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(),
		EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(10 * time.Minute),
	}
	atestacionValida := crearAtestacion(t, resultadoValido)
	if _, err := NuevoContextoAutorizacionCobro(decisionBase, atestacionValida, recursoAutorizableCobroPrueba(orden.ID), instante); err != nil {
		t.Fatalf("decision y atestacion validas rechazadas: %v", err)
	}

	casosDecision := []struct {
		nombre string
		muta   func(*DecisionAutorizacion)
	}{
		{"campos vacios", func(d *DecisionAutorizacion) { d.CamposPermitidos = nil }},
		{"campo faltante", func(d *DecisionAutorizacion) { d.CamposPermitidos = d.CamposPermitidos[1:] }},
		{"campo adicional", func(d *DecisionAutorizacion) { d.CamposPermitidos = append(d.CamposPermitidos, "orden.otro") }},
		{"campo sustituido", func(d *DecisionAutorizacion) { d.CamposPermitidos[0] = "orden.otro" }},
		{"obligacion desconocida", func(d *DecisionAutorizacion) { d.Obligaciones = []string{"doble_control"} }},
		{"garantia declarada insuficiente", func(d *DecisionAutorizacion) { d.GarantiaMinima = AuthAssuranceSubstantial }},
		{"huella de contexto ajena", func(d *DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = huellaCobro('8') }},
	}
	for _, caso := range casosDecision {
		t.Run(caso.nombre, func(t *testing.T) {
			decision := clonarDecisionAutorizacionCobro(decisionBase)
			caso.muta(&decision)
			if _, err := NuevoContextoAutorizacionCobro(decision, atestacionValida, recursoAutorizableCobroPrueba(orden.ID), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
				t.Fatalf("decision sin concesion positiva exacta aceptada: %v", err)
			}
		})
	}
	recursoDivergente := recursoAutorizableCobroPrueba(orden.ID)
	recursoDivergente.Atributos = map[string]string{"estado": "conciliado"}
	if _, err := NuevoContextoAutorizacionCobro(decisionBase, atestacionValida, recursoDivergente, instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("contexto de recurso distinto aceptado: %v", err)
	}

	atestacionesInvalidas := []struct {
		nombre    string
		resultado ResultadoVerificacionAutenticacionCobro
	}{
		{"otro principal", ResultadoVerificacionAutenticacionCobro{PrincipalRef: "per_otra23456789abcdefghijk", Metodo: AuthMethodCertificate, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_otra23456789abcdefghijkl", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"otra autenticacion del mismo principal", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodCertificate, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_otra23456789abcdefghijkl", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"otra sesion del mismo principal", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodCertificate, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_0123456789abcdefghijkl", SesionRef: "ses_abcdefghijkl0123456789", HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"otro metodo del mismo principal", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodDNIe, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_0123456789abcdefghijkl", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"garantia real insuficiente", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodCertificate, Garantia: AuthAssuranceSubstantial, AutenticacionRef: "aut_baja23456789abcdefghijkl", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"metodo de demostracion", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodDemo, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_demo23456789abcdefghijkl", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(time.Minute)}},
		{"sesion caducada", ResultadoVerificacionAutenticacionCobro{PrincipalRef: decisionBase.PrincipalID, Metodo: AuthMethodCertificate, Garantia: AuthAssuranceHigh, AutenticacionRef: "aut_caducada56789abcdefghijk", SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-2 * time.Minute), ValidaHasta: instante}},
	}
	for _, caso := range atestacionesInvalidas {
		t.Run(caso.nombre, func(t *testing.T) {
			atestacion, err := NuevaAtestacionAutenticacionCobro(context.Background(), verificadorAutenticacionCobroPrueba{resultado: caso.resultado}, caso.resultado.SesionRef, caso.resultado.HuellaSesionHMAC, instante)
			if err != nil {
				if caso.nombre == "metodo de demostracion" || caso.nombre == "sesion caducada" {
					return
				}
				t.Fatalf("la atestacion estructural debia formarse para probar su cruce: %v", err)
			}
			if _, err = NuevoContextoAutorizacionCobro(decisionBase, atestacion, recursoAutorizableCobroPrueba(orden.ID), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
				t.Fatalf("atestacion no coincidente aceptada: %v", err)
			}
		})
	}
	resultadoOtraSesion := resultadoValido
	resultadoOtraSesion.SesionRef = "ses_abcdefghijkl0123456789"
	if _, err := NuevaAtestacionAutenticacionCobro(
		context.Background(),
		verificadorAutenticacionCobroPrueba{resultado: resultadoOtraSesion},
		sesionCobroPrueba, huellaSesionCobroPrueba(), instante,
	); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("el verificador pudo responder por otra sesion: %v", err)
	}
	resultadoOtraHuella := resultadoValido
	resultadoOtraHuella.HuellaSesionHMAC = "hmac-sha256:sesion-v1:" + huellaCobro('8')
	if _, err := NuevaAtestacionAutenticacionCobro(
		context.Background(),
		verificadorAutenticacionCobroPrueba{resultado: resultadoOtraHuella},
		sesionCobroPrueba, huellaSesionCobroPrueba(), instante,
	); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("el verificador pudo responder con otra huella de sesion: %v", err)
	}
	resultadoOtroDominio := resultadoValido
	resultadoOtroDominio.HuellaSesionHMAC = "hmac-sha256:otro-dominio:" + huellaCobro('9')
	if _, err := NuevaAtestacionAutenticacionCobro(
		context.Background(),
		verificadorAutenticacionCobroPrueba{resultado: resultadoOtroDominio},
		sesionCobroPrueba, resultadoOtroDominio.HuellaSesionHMAC, instante,
	); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("una HMAC de otro dominio se acepto como vinculo de sesion: %v", err)
	}
	if _, err := NuevaAtestacionAutenticacionCobro(context.Background(), nil, sesionCobroPrueba, huellaSesionCobroPrueba(), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("se acepto un verificador ausente: %v", err)
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := NuevaAtestacionAutenticacionCobro(
		ctxCancelado,
		verificadorAutenticacionCobroPrueba{resultado: resultadoValido},
		sesionCobroPrueba,
		huellaSesionCobroPrueba(),
		instante,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("un contexto cancelado no se propago: %v", err)
	}
	if _, err := json.Marshal(atestacionValida); !errors.Is(err, ErrSerializacionAutorizacionCobro) {
		t.Fatalf("se serializo la atestacion opaca: %v", err)
	}
	if _, err := json.Marshal(resultadoValido); !errors.Is(err, ErrSerializacionAutorizacionCobro) ||
		strings.Contains(fmt.Sprintf("%+v", resultadoValido), resultadoValido.HuellaSesionHMAC) {
		t.Fatal("el resultado interno de identidad filtro el vinculo de sesion")
	}
	solicitudInterna := SolicitudVerificacionAutenticacionCobro{
		SesionRef: sesionCobroPrueba, HuellaSesionHMAC: huellaSesionCobroPrueba(), Instante: instante,
	}
	if _, err := json.Marshal(solicitudInterna); !errors.Is(err, ErrSerializacionAutorizacionCobro) ||
		strings.Contains(fmt.Sprintf("%+v", solicitudInterna), solicitudInterna.HuellaSesionHMAC) {
		t.Fatal("la solicitud interna de identidad filtro el vinculo de sesion")
	}
}

func TestContextoCobroSeValidaEnInstanteActualYNoDesdeCache(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(time.Minute)
	contexto := contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instante)
	if err := contexto.ValidarEn(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante); err != nil {
		t.Fatalf("contexto fresco rechazado: %v", err)
	}
	instanteCaducado := instante.Add(vigenciaMaximaUsoContextoCobro)
	if err := contexto.ValidarEn(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instanteCaducado); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("contexto cacheado aceptado al vencer: %v", err)
	}
	if _, err := orden.PrepararInicioOperacion("retorno:uno", "notificacion:uno", instanteCaducado, contexto); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("la preparacion reutilizo un contexto caducado: %v", err)
	}
}

func TestHechoCobroConservaVinculoForenseDeSesionSinCredencial(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	hecho := orden.Historial[0]
	if hecho.SesionRef != sesionCobroPrueba || hecho.HuellaSesionHMAC != huellaSesionCobroPrueba() ||
		hecho.AtestacionAutenticacionRef != "aut_0123456789abcdefghijkl" ||
		hecho.MetodoAutenticacion != AuthMethodCertificate || hecho.GarantiaAutenticacion != AuthAssuranceHigh {
		t.Fatalf("vinculo forense incompleto: %+v", hecho)
	}
	for _, valor := range []string{hecho.SesionRef, hecho.AtestacionAutenticacionRef} {
		if strings.Contains(strings.ToLower(valor), "token") || strings.Contains(strings.ToLower(valor), "cookie") {
			t.Fatalf("se persistio una credencial en vez de una referencia opaca: %q", valor)
		}
	}
}

func TestContextoCobroLigaYCopiaDecisionCompleta(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(time.Minute)
	decision := decisionCobro(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
	decision.PoliticasRefs = []string{"politica:cobro:uno"}
	decision.PoliticasHuellasSHA256 = map[string]string{"politica:cobro:uno": huellaCobro('f')}
	decision.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasRefs...)
	decision.PoliticasEvaluadasHuellasSHA256 = map[string]string{"politica:cobro:uno": huellaCobro('f')}
	decision.CatalogoPoliticasHuellaSHA256, _ = HuellaEvidenciasCatalogoPoliticasAutorizacion(
		decision.PoliticasEvaluadasRefs, decision.PoliticasEvaluadasHuellasSHA256,
	)
	contexto, err := NuevoContextoAutorizacionCobro(decision, atestacionAutenticacionCobroPrueba(t, instante), recursoAutorizableCobroPrueba(orden.ID), instante)
	if err != nil {
		t.Fatal(err)
	}
	decision.PoliticasRefs[0] = "politica:alterada"
	decision.PoliticasEvaluadasRefs[0] = "politica:alterada"
	decision.PoliticasHuellasSHA256["politica:cobro:uno"] = huellaCobro('a')
	decision.PoliticasEvaluadasHuellasSHA256["politica:cobro:uno"] = huellaCobro('a')
	decision.CamposPermitidos[0] = "campo.alterado"
	if err := contexto.ValidarEn(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante); err != nil {
		t.Fatalf("el llamador pudo alterar el contexto tras construirlo: %v", err)
	}

	proyeccion, err := contexto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	proyeccion.CamposPermitidos[0] = "campo.alterado"
	if err := contexto.ValidarEn(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante); err != nil {
		t.Fatalf("la proyeccion compartia memoria con la capacidad: %v", err)
	}

	contexto.datos.decision.Codigo = "denegada_por_politica"
	if err := contexto.ValidarEn(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("se acepto una alteracion de la decision completa: %v", err)
	}
}

func TestCamposRequeridosAccionCobroSonCerradosYDefensivos(t *testing.T) {
	campos, existe := CamposRequeridosAccionCobro(AccionCobroCrearOrden)
	if !existe || len(campos) == 0 {
		t.Fatal("la accion conocida carece de campos positivos")
	}
	campos[0] = "alterado"
	otraCopia, _ := CamposRequeridosAccionCobro(AccionCobroCrearOrden)
	if otraCopia[0] == "alterado" {
		t.Fatal("el catalogo devolvio memoria mutable compartida")
	}
	if campos, existe := CamposRequeridosAccionCobro(AccionCobro("cobros.todo")); existe || campos != nil {
		t.Fatal("una accion no enumerada obtuvo campos")
	}
}

func TestSerializacionSoloPermiteVistaMinima(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	if _, err := json.Marshal(orden); !errors.Is(err, ErrSerializacionOrdenCobroProhibida) {
		t.Fatalf("se serializo el agregado: %v", err)
	}
	if _, err := json.Marshal(orden.Historial[0]); !errors.Is(err, ErrSerializacionEvidenciaCobroProhibida) {
		t.Fatalf("se serializo aisladamente un hecho interno: %v", err)
	}
	var destino OrdenCobro
	if err := json.Unmarshal([]byte(`{"id":"cob_0123456789abcdefghijkl"}`), &destino); !errors.Is(err, ErrSerializacionOrdenCobroProhibida) {
		t.Fatalf("se deserializo directamente el agregado: %v", err)
	}
	comando, _ := orden.PrepararConciliacion(
		TipoConciliacionCobroIngreso, "cierre:uno", instanteBaseCobro.Add(3*time.Minute),
		contextoOrdenCobro(t, orden, AccionCobroConciliar, instanteBaseCobro.Add(3*time.Minute)),
	)
	if _, err := json.Marshal(comando); !errors.Is(err, ErrSerializacionEvidenciaCobroProhibida) {
		t.Fatalf("se serializo un comando interno: %v", err)
	}
	evidencia, _ := NuevaEvidenciaResultadoCobroVerificada(
		datosServidorCobro(orden, "otra-confirmacion", 'f', instanteBaseCobro.Add(3*time.Minute)), ResultadoOperacionCobroConfirmado,
	)
	if _, err := json.Marshal(evidencia); !errors.Is(err, ErrSerializacionEvidenciaCobroProhibida) {
		t.Fatalf("se serializo evidencia interna: %v", err)
	}
	vista, err := orden.VistaTitular()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(vista)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, secreto := range []string{"expediente", "solicitud", "liquidacion", "sujeto", "historial", "idempotencia", "operacion_proveedor"} {
		if strings.Contains(texto, secreto) {
			t.Fatalf("la vista minima filtro %q: %s", secreto, texto)
		}
	}
	if strings.Contains(fmt.Sprintf("%+v", orden), orden.ID) || strings.Contains(fmt.Sprintf("%+v", evidencia), orden.ID) {
		t.Fatal("el formateo de depuracion filtro datos internos")
	}
}

func TestDetectorTarjetaCubreSeparadoresEtiquetasYUnicode(t *testing.T) {
	prohibidos := []string{
		"4111111111111111",
		"4111 1111 1111 1111",
		"4111-1111-1111-1111",
		"4111\u200b1111\u200b1111\u200b1111",
		"４１１１ １１１１ １１１１ １１１１",
		"card_number aportado",
		"c\u200bv\u200bv aportado",
		"Incluye CVV",
	}
	for _, valor := range prohibidos {
		if !contieneDatoTarjetaCobro(valor) {
			t.Errorf("no se detecto dato de tarjeta en %q", valor)
		}
	}
	if contieneDatoTarjetaCobro("Campaña de inscripción y pago de tasa") {
		t.Fatal("se produjo un falso positivo por una subcadena")
	}
	alta := altaOrdenCobroValida()
	alta.Concepto = "Pago con 4111 1111 1111 1111"
	contexto := contextoCobro(t, AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
	if _, err := NuevaOrdenCobro(alta, contexto); !errors.Is(err, ErrDatoTarjetaProhibido) {
		t.Fatalf("la orden acepto PAN separado: %v", err)
	}
}

func TestVigenciasYRelojesFallanCerrados(t *testing.T) {
	alta := altaOrdenCobroValida()
	alta.CaducaEn = alta.CreadaEn.Add(vigenciaMaximaOrdenCobro + time.Second)
	contexto := contextoCobro(t, AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
	if _, err := NuevaOrdenCobro(alta, contexto); !errors.Is(err, ErrOrdenCobroInvalida) {
		t.Fatalf("se acepto una orden con vigencia excesiva: %v", err)
	}

	orden := nuevaOrdenCobroPrueba(t)
	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	datos := datosServidorCobro(orden, "stale", 'e', instanteBaseCobro.Add(2*time.Minute))
	evidencia, _ := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado)
	contextoTardio := contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteBaseCobro.Add(18*time.Minute))
	if _, _, err := orden.AplicarResultadoServidor(evidencia, instanteBaseCobro.Add(18*time.Minute), contextoTardio, "Evidencia antigua"); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("se acepto evidencia antigua: %v", err)
	}

	datosFuturos := datosServidorCobro(orden, "futura", 'f', instanteBaseCobro.Add(5*time.Minute))
	evidenciaFutura, _ := NuevaEvidenciaResultadoCobroVerificada(datosFuturos, ResultadoOperacionCobroConfirmado)
	contextoAnterior := contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteBaseCobro.Add(2*time.Minute))
	if _, _, err := orden.AplicarResultadoServidor(evidenciaFutura, instanteBaseCobro.Add(2*time.Minute), contextoAnterior, "Evidencia futura"); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("se acepto evidencia futura: %v", err)
	}

	autorizacionCaducar := contextoOrdenCobro(t, orden, AccionCobroCaducar, instanteBaseCobro.Add(2*time.Minute))
	if _, _, err := orden.Caducar("evidencia:caducidad", huellaCobro('1'), "Caducidad automatica", instanteBaseCobro.Add(2*time.Minute), autorizacionCaducar); !errors.Is(err, ErrTransicionCobroInvalida) {
		t.Fatalf("se caduco antes del plazo: %v", err)
	}
	ordenDesconocida, _ := aplicarResultadoCobroPrueba(t, orden, ResultadoOperacionCobroDesconocido, "indeterminado", '2', instanteBaseCobro.Add(3*time.Minute))
	if ordenDesconocida.Estado != EstadoCobroResultadoDesconocido {
		t.Fatalf("un resultado desconocido se convirtio en rechazo: %q", ordenDesconocida.Estado)
	}

	confirmada := confirmarOrdenCobroPrueba(t)
	solicitudFutura := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:futura",
		HuellaEvidenciaSHA256:  huellaCobro('3'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('4'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instanteBaseCobro.Add(6 * time.Minute),
	}
	contextoAnteriorDevolucion := contextoOrdenCobro(t, confirmada, AccionCobroSolicitarDevolucion, instanteBaseCobro.Add(3*time.Minute))
	if _, _, _, err := confirmada.SolicitarDevolucion(solicitudFutura, contextoAnteriorDevolucion); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("se uso una autorizacion antigua para fechar una devolucion futura: %v", err)
	}
}

func TestConfirmacionPosteriorACancelacionGeneraIncidencia(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	instanteCancelacion := instanteBaseCobro.Add(2 * time.Minute)
	orden, repetida, err := orden.Cancelar(
		"evidencia:cancelacion:uno", huellaCobro('1'), "Cancelacion administrativa",
		instanteCancelacion, contextoOrdenCobro(t, orden, AccionCobroCancelar, instanteCancelacion),
	)
	if err != nil || repetida || orden.Estado != EstadoCobroCancelada {
		t.Fatalf("cancelar: estado=%q repetida=%v err=%v", orden.Estado, repetida, err)
	}
	instanteConfirmacion := instanteBaseCobro.Add(3 * time.Minute)
	evidencia, err := NuevaEvidenciaResultadoCobroVerificada(
		datosServidorCobro(orden, "confirmacion-tardia", '2', instanteConfirmacion), ResultadoOperacionCobroConfirmado,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err = orden.AplicarResultadoServidor(
		evidencia, instanteConfirmacion, contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instanteConfirmacion), "Confirmacion tardia",
	)
	if err != nil || repetida || orden.Estado != EstadoCobroIncidenciaBloqueada ||
		orden.Historial[len(orden.Historial)-1].Tipo != HechoCobroIncidenciaDetectada {
		t.Fatalf("confirmacion tardia no bloqueo: estado=%q repetida=%v err=%v", orden.Estado, repetida, err)
	}
}

func TestIdempotenciaCanonicaIgnoraGeneradosPeroFijaSemantica(t *testing.T) {
	base := altaOrdenCobroValida()
	canonica, err := BytesCanonicosIdempotenciaAltaCobro(base)
	if err != nil {
		t.Fatal(err)
	}
	generada := base
	generada.ID = "cob_abcdefghijkl0123456789"
	generada.CreadaEn = generada.CreadaEn.Add(time.Hour)
	generada.CaducaEn = generada.CaducaEn.Add(time.Hour)
	generada.CorrelacionRef = "correlacion:generada:dos"
	generada.EvidenciaCreacionRef = "evidencia:generada:dos"
	generada.HuellaEvidenciaSHA256 = huellaCobro('f')
	generada.Motivo = "Otro texto de auditoria"
	otraCanonica, err := BytesCanonicosIdempotenciaAltaCobro(generada)
	if err != nil || !bytes.Equal(canonica, otraCanonica) {
		t.Fatal("los valores generados alteraron la clave semantica")
	}
	mutaciones := []func(*AltaOrdenCobro){
		func(a *AltaOrdenCobro) { a.ExpedienteRef = "expediente:dos" },
		func(a *AltaOrdenCobro) { a.SolicitudRef = "solicitud:dos" },
		func(a *AltaOrdenCobro) { a.LiquidacionRef = "liquidacion:dos" },
		func(a *AltaOrdenCobro) { a.SujetoRef = "per_otra23456789abcdefghijk" },
		func(a *AltaOrdenCobro) { a.RepresentacionRef = "rep_otra23456789abcdefghijk" },
		func(a *AltaOrdenCobro) { a.Tarifa.Version++ },
		func(a *AltaOrdenCobro) { a.Importe.UnidadesMenores++ },
		func(a *AltaOrdenCobro) { a.Concepto = "Otro concepto" },
		func(a *AltaOrdenCobro) { a.Finalidad = "otra_finalidad" },
	}
	for indice, muta := range mutaciones {
		alterada := base
		muta(&alterada)
		bytesAlterados, errorAlterada := BytesCanonicosIdempotenciaAltaCobro(alterada)
		if errorAlterada != nil || bytes.Equal(canonica, bytesAlterados) {
			t.Errorf("mutacion semantica %d no cambio la forma canonica", indice)
		}
	}
}

func TestAltaCobroRechazaEntradasQueSoloSerianValidasTrasRecortarEspacios(t *testing.T) {
	base := altaOrdenCobroValida()
	autorizacion := contextoCobro(
		t, AccionCobroCrearOrden, base.LiquidacionRef, base.Finalidad, base.CorrelacionRef, base.CreadaEn,
	)
	casos := []struct {
		nombre            string
		muta              func(*AltaOrdenCobro)
		compruebaCanonica bool
	}{
		{"concepto con prefijo", func(a *AltaOrdenCobro) { a.Concepto = " " + a.Concepto }, true},
		{"concepto con sufijo", func(a *AltaOrdenCobro) { a.Concepto += " " }, true},
		{"finalidad con prefijo", func(a *AltaOrdenCobro) { a.Finalidad = " " + a.Finalidad }, true},
		{"finalidad con sufijo", func(a *AltaOrdenCobro) { a.Finalidad += " " }, true},
		{"motivo con prefijo", func(a *AltaOrdenCobro) { a.Motivo = " " + a.Motivo }, false},
		{"motivo con sufijo", func(a *AltaOrdenCobro) { a.Motivo += " " }, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := base
			caso.muta(&alterada)
			if caso.compruebaCanonica {
				if _, err := BytesCanonicosIdempotenciaAltaCobro(alterada); !errors.Is(err, ErrOrdenCobroInvalida) {
					t.Fatalf("la forma canonica normalizo la entrada: %v", err)
				}
			}
			if _, err := NuevaOrdenCobro(alterada, autorizacion); !errors.Is(err, ErrOrdenCobroInvalida) {
				t.Fatalf("el alta normalizo la entrada: %v", err)
			}
		})
	}
}

func TestEvidenciaServidorCobroNoNormalizaCamposAntesDeValidarlos(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	base := datosServidorCobro(orden, "exactitud", '7', instanteBaseCobro.Add(time.Minute))
	casos := []struct {
		nombre string
		muta   func(*DatosEvidenciaServidorCobro)
	}{
		{"codigo con prefijo", func(d *DatosEvidenciaServidorCobro) { d.Codigo = " " + d.Codigo }},
		{"codigo con sufijo", func(d *DatosEvidenciaServidorCobro) { d.Codigo += " " }},
		{"concepto con prefijo", func(d *DatosEvidenciaServidorCobro) { d.Concepto = " " + d.Concepto }},
		{"concepto con sufijo", func(d *DatosEvidenciaServidorCobro) { d.Concepto += " " }},
		{"audiencia con prefijo", func(d *DatosEvidenciaServidorCobro) { d.Audiencia = " " + d.Audiencia }},
		{"audiencia con sufijo", func(d *DatosEvidenciaServidorCobro) { d.Audiencia += " " }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterados := base
			caso.muta(&alterados)
			if _, err := NuevaEvidenciaResultadoCobroVerificada(
				alterados, ResultadoOperacionCobroConfirmado,
			); !errors.Is(err, ErrEvidenciaCobroInvalida) {
				t.Fatalf("la evidencia normalizo la entrada: %v", err)
			}
		})
	}
}

func TestComandosSonSelladosYDefensivos(t *testing.T) {
	if (ComandoInicioOperacionCobro{}).Validar() == nil || (ComandoDevolucionCobro{}).Validar() == nil ||
		(ComandoConciliacionCobro{}).Validar() == nil {
		t.Fatal("un comando cero fue aceptado")
	}
	orden := nuevaOrdenCobroPrueba(t)
	comando, err := orden.PrepararInicioOperacion(
		"retorno:uno", "notificacion:uno", instanteBaseCobro.Add(time.Minute),
		contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instanteBaseCobro.Add(time.Minute)),
	)
	if err != nil {
		t.Fatal(err)
	}
	primera, _ := comando.Datos()
	primera.Concepto = "Intento de cambio"
	segunda, _ := comando.Datos()
	if segunda.Concepto != orden.Concepto || segunda.VersionOrden != orden.Version || segunda.HuellaOrdenSHA256 != orden.HuellaEstadoSHA256 {
		t.Fatal("Datos() permitio alterar el comando sellado")
	}
	comandoInicioDominioIncorrecto := comando
	comandoInicioDominioIncorrecto.datos.IndiceIdempotenciaHMAC = "hmac-sha256:devoluciones-v1:" + huellaCobro('1')
	if !errors.Is(comandoInicioDominioIncorrecto.Validar(), ErrComandoCobroInvalido) {
		t.Fatal("el comando de inicio acepto el dominio HMAC de devoluciones")
	}

	confirmada := confirmarOrdenCobroPrueba(t)
	instanteDevolucion := instanteBaseCobro.Add(3 * time.Minute)
	_, comandoDevolucion, repetida, err := confirmada.SolicitarDevolucion(
		SolicitudDevolucionOrdenCobro{
			DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:comando",
			HuellaEvidenciaSHA256:  huellaCobro('2'),
			IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('3'),
			Motivo:                 "Resolucion administrativa", SolicitadaEn: instanteDevolucion,
		},
		contextoOrdenCobro(t, confirmada, AccionCobroSolicitarDevolucion, instanteDevolucion),
	)
	if err != nil || repetida {
		t.Fatalf("crear comando de devolucion: repetida=%v err=%v", repetida, err)
	}
	comandoDevolucionDominioIncorrecto := comandoDevolucion
	comandoDevolucionDominioIncorrecto.datos.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaCobro('3')
	if !errors.Is(comandoDevolucionDominioIncorrecto.Validar(), ErrComandoCobroInvalido) {
		t.Fatal("el comando de devolucion acepto el dominio HMAC de altas")
	}
}

func TestClonarNoComparteHistorial(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	clon := orden.Clonar()
	clon.Historial[0].Motivo = "Alterado solo en clon"
	if orden.Historial[0].Motivo == clon.Historial[0].Motivo || orden.Validar() != nil {
		t.Fatal("Clonar() compartio el historial mutable")
	}
}

func TestEvidenciasCeroYDeNavegadorSeRechazan(t *testing.T) {
	if (EvidenciaInicioOperacionCobro{}).Validar() == nil ||
		(EvidenciaResultadoCobro{}).Validar() == nil ||
		(EvidenciaResultadoDevolucionCobro{}).Validar() == nil ||
		(EvidenciaConciliacionCobro{}).Validar() == nil {
		t.Fatal("una evidencia cero fue aceptada")
	}
	datos := datosServidorCobro(nuevaOrdenCobroPrueba(t), "navegador", 'a', instanteBaseCobro.Add(time.Minute))
	datos.MetodoAutenticacion = MetodoAutenticacionEvidenciaCobro("parametro_navegador")
	if _, err := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("se acepto evidencia declarada por navegador: %v", err)
	}
	datos.MetodoAutenticacion = MetodoAutenticacionEvidenciaCobro("decision_interna")
	if _, err := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("un conector suplanto una decision interna: %v", err)
	}
	datos.MetodoAutenticacion = MetodoAutenticacionCobroFirmaYTLSMutuo
	datos.Audiencia = "otro.servicio"
	if _, err := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("se acepto evidencia destinada a otra audiencia: %v", err)
	}
	datos.Audiencia = audienciaEvidenciaCobro
	datos.EvidenciaRef = "incidencia:" + huellaCobro('a')
	if _, err := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado); !errors.Is(err, ErrEvidenciaCobroInvalida) {
		t.Fatalf("un conector uso el espacio reservado de incidencias: %v", err)
	}
}

func TestIdentidadesCobroSoloAdmitenReferenciasOpacas(t *testing.T) {
	for _, sujeto := range []string{"00000000A", "persona@example.org", "persona:opaca:uno"} {
		alta := altaOrdenCobroValida()
		alta.SujetoRef = sujeto
		contexto := contextoCobro(t, AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
		if _, err := NuevaOrdenCobro(alta, contexto); !errors.Is(err, ErrOrdenCobroInvalida) {
			t.Errorf("se acepto sujeto no opaco %q: %v", sujeto, err)
		}
	}
	alta := altaOrdenCobroValida()
	alta.RepresentacionRef = "representante@example.org"
	contexto := contextoCobro(t, AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
	if _, err := NuevaOrdenCobro(alta, contexto); !errors.Is(err, ErrOrdenCobroInvalida) {
		t.Fatalf("se acepto una representacion no opaca: %v", err)
	}
	instante := instanteBaseCobro.Add(time.Minute)
	resultado := ResultadoVerificacionAutenticacionCobro{
		PrincipalRef: "00000000A", Metodo: AuthMethodCertificate, Garantia: AuthAssuranceHigh,
		AutenticacionRef: "aut_0123456789abcdefghijkl", SesionRef: sesionCobroPrueba,
		HuellaSesionHMAC: huellaSesionCobroPrueba(), EmitidaEn: instante.Add(-time.Minute),
		ValidaHasta: instante.Add(time.Minute),
	}
	if _, err := NuevaAtestacionAutenticacionCobro(
		context.Background(),
		verificadorAutenticacionCobroPrueba{resultado: resultado}, sesionCobroPrueba,
		huellaSesionCobroPrueba(), instante,
	); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("una identidad DNI produjo atestacion: %v", err)
	}
	orden := nuevaOrdenCobroPrueba(t)
	decision := decisionCobro(AccionCobroIniciarOperacion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
	decision.PerfilActivoRef = "tesoreria@example.org"
	if _, err := NuevoContextoAutorizacionCobro(decision, atestacionAutenticacionCobroPrueba(t, instante), recursoAutorizableCobroPrueba(orden.ID), instante); !errors.Is(err, ErrContextoAutorizacionCobroInvalido) {
		t.Fatalf("un perfil no opaco produjo autorizacion: %v", err)
	}
}

func TestIdempotenciaDevolucionEsCanonicaDeDominioExactoYNoReutilizable(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(3 * time.Minute)
	solicitud := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:canonica",
		HuellaEvidenciaSHA256:  huellaCobro('4'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('5'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instante,
	}
	canonica, err := BytesCanonicosIdempotenciaDevolucionCobro(orden, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	reintento := solicitud
	reintento.SolicitadaEn = instante.Add(10 * time.Second)
	otraCanonica, err := BytesCanonicosIdempotenciaDevolucionCobro(orden, reintento)
	if err != nil || !bytes.Equal(canonica, otraCanonica) {
		t.Fatal("el instante del servidor altero la idempotencia semantica")
	}
	alterada := solicitud
	alterada.Motivo = "Otra resolucion administrativa"
	bytesAlterados, err := BytesCanonicosIdempotenciaDevolucionCobro(orden, alterada)
	if err != nil || bytes.Equal(canonica, bytesAlterados) {
		t.Fatal("un cambio semantico no altero la forma canonica")
	}
	dominioIncorrecto := solicitud
	dominioIncorrecto.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaCobro('5')
	if _, _, _, err := orden.SolicitarDevolucion(
		dominioIncorrecto,
		contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
	); !errors.Is(err, ErrDevolucionCobroInvalida) {
		t.Fatalf("se acepto el dominio HMAC de altas en una devolucion: %v", err)
	}
	nueva, _, repetida, err := orden.SolicitarDevolucion(
		solicitud, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
	)
	if err != nil || repetida {
		t.Fatalf("primera devolucion: repetida=%v err=%v", repetida, err)
	}
	reintento.SolicitadaEn = instante.Add(10 * time.Second)
	repetidaOrden, comando, repetida, err := nueva.SolicitarDevolucion(
		reintento, contextoOrdenCobro(t, nueva, AccionCobroSolicitarDevolucion, reintento.SolicitadaEn),
	)
	if err != nil || !repetida || repetidaOrden.Version != nueva.Version || comando.Validar() != nil {
		t.Fatalf("reintento exacto no idempotente: version=%d repetida=%v err=%v", repetidaOrden.Version, repetida, err)
	}
	conflictiva := reintento
	conflictiva.EvidenciaRef = "evidencia:devolucion:otra"
	conflictiva.SolicitadaEn = instante.Add(20 * time.Second)
	bloqueada, _, repetida, err := nueva.SolicitarDevolucion(
		conflictiva, contextoOrdenCobro(t, nueva, AccionCobroSolicitarDevolucion, conflictiva.SolicitadaEn),
	)
	if err != nil || repetida || bloqueada.Estado != EstadoCobroIncidenciaBloqueada {
		t.Fatalf("reutilizacion semantica no bloqueo: estado=%q repetida=%v err=%v", bloqueada.Estado, repetida, err)
	}
}

func TestDevolucionCobroNoNormalizaMotivoAntesDeValidar(t *testing.T) {
	orden := confirmarOrdenCobroPrueba(t)
	instante := instanteBaseCobro.Add(3 * time.Minute)
	base := SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:exacta",
		HuellaEvidenciaSHA256:  huellaCobro('4'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('5'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instante,
	}
	for nombre, motivo := range map[string]string{
		"prefijo": " " + base.Motivo,
		"sufijo":  base.Motivo + " ",
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := base
			alterada.Motivo = motivo
			if _, err := BytesCanonicosIdempotenciaDevolucionCobro(orden, alterada); !errors.Is(err, ErrDevolucionCobroInvalida) {
				t.Fatalf("la forma canonica normalizo el motivo: %v", err)
			}
			if _, _, _, err := orden.SolicitarDevolucion(
				alterada, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instante),
			); !errors.Is(err, ErrDevolucionCobroInvalida) {
				t.Fatalf("la solicitud normalizo el motivo: %v", err)
			}
		})
	}
}

func TestTransicionesCobroNoNormalizanMotivosAntesDeAutorizar(t *testing.T) {
	t.Run("registrar envio", func(t *testing.T) {
		orden := nuevaOrdenCobroPrueba(t)
		instante := instanteBaseCobro.Add(time.Minute)
		evidencia, err := NuevaEvidenciaInicioOperacionCobroVerificada(
			datosServidorCobro(orden, "motivo-inicio", '1', instante),
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = orden.RegistrarEnvio(
			evidencia, instante, contextoOrdenCobro(t, orden, AccionCobroIniciarOperacion, instante), " Inicio alojado",
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("RegistrarEnvio normalizo el motivo: %v", err)
		}
	})

	t.Run("resultado de cobro", func(t *testing.T) {
		orden, _ := enviarOrdenCobroPrueba(t, nuevaOrdenCobroPrueba(t), instanteBaseCobro.Add(time.Minute))
		instante := instanteBaseCobro.Add(2 * time.Minute)
		evidencia, err := NuevaEvidenciaResultadoCobroVerificada(
			datosServidorCobro(orden, "motivo-resultado", '2', instante), ResultadoOperacionCobroConfirmado,
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = orden.AplicarResultadoServidor(
			evidencia, instante, contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, instante), "Resultado autenticado ",
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("AplicarResultadoServidor normalizo el motivo: %v", err)
		}
	})

	t.Run("resultado de devolucion", func(t *testing.T) {
		orden := confirmarOrdenCobroPrueba(t)
		instanteSolicitud := instanteBaseCobro.Add(3 * time.Minute)
		solicitud := SolicitudDevolucionOrdenCobro{
			DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:motivo",
			HuellaEvidenciaSHA256:  huellaCobro('3'),
			IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaCobro('4'),
			Motivo:                 "Resolucion administrativa", SolicitadaEn: instanteSolicitud,
		}
		var err error
		orden, _, _, err = orden.SolicitarDevolucion(
			solicitud, contextoOrdenCobro(t, orden, AccionCobroSolicitarDevolucion, instanteSolicitud),
		)
		if err != nil {
			t.Fatal(err)
		}
		instante := instanteBaseCobro.Add(4 * time.Minute)
		evidencia, err := NuevaEvidenciaResultadoDevolucionCobroVerificada(
			datosServidorCobro(orden, "motivo-devolucion", '5', instante), solicitud.DevolucionRef,
			ResultadoDevolucionCobroConfirmada,
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = orden.AplicarResultadoDevolucionServidor(
			evidencia, instante, contextoOrdenCobro(t, orden, AccionCobroProcesarDevolucion, instante), " Resultado de devolucion",
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("AplicarResultadoDevolucionServidor normalizo el motivo: %v", err)
		}
	})

	t.Run("conciliacion", func(t *testing.T) {
		orden := confirmarOrdenCobroPrueba(t)
		instante := instanteBaseCobro.Add(3 * time.Minute)
		evidencia, err := NuevaEvidenciaConciliacionCobroVerificada(
			datosServidorCobro(orden, "motivo-conciliacion", '6', instante),
			TipoConciliacionCobroIngreso, "cierre:motivo:uno", "",
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = orden.AplicarConciliacionServidor(
			evidencia, instante, contextoOrdenCobro(t, orden, AccionCobroConciliar, instante), "Conciliacion exacta ",
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("AplicarConciliacionServidor normalizo el motivo: %v", err)
		}
	})

	t.Run("cancelacion local", func(t *testing.T) {
		orden := nuevaOrdenCobroPrueba(t)
		instante := instanteBaseCobro.Add(time.Minute)
		_, _, err := orden.Cancelar(
			"evidencia:cancelacion:no-canonica", huellaCobro('7'), " Cancelacion administrativa", instante,
			contextoOrdenCobro(t, orden, AccionCobroCancelar, instante),
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("Cancelar normalizo el motivo: %v", err)
		}
	})

	t.Run("caducidad local", func(t *testing.T) {
		orden := nuevaOrdenCobroPrueba(t)
		instante := orden.CaducaEn.Add(time.Second)
		_, _, err := orden.Caducar(
			"evidencia:caducidad:no-canonica", huellaCobro('8'), "Caducidad administrativa ", instante,
			contextoOrdenCobro(t, orden, AccionCobroCaducar, instante),
		)
		if !errors.Is(err, ErrEvidenciaCobroInvalida) {
			t.Fatalf("Caducar normalizo el motivo: %v", err)
		}
	})
}

func TestMatrizHechosDistingueEvidenciaLocalYRemota(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	localAlterado := orden.Historial[0]
	localAlterado.ConectorID = "pasarela_corporativa"
	localAlterado.VersionConector = 2
	localAlterado.OperacionProveedorRef = "operacion:inventada"
	if !errors.Is(localAlterado.Validar(), ErrEvidenciaCobroInvalida) {
		t.Fatal("un hecho local suplanto procedencia de pasarela")
	}
	enviada, _ := enviarOrdenCobroPrueba(t, orden, instanteBaseCobro.Add(time.Minute))
	remotoAlterado := enviada.Historial[len(enviada.Historial)-1]
	remotoAlterado.VerificacionEvidenciaRef = ""
	remotoAlterado.HuellaVerificacionSHA256 = ""
	remotoAlterado.MetodoVerificacionEvidencia = ""
	remotoAlterado.AudienciaEvidencia = ""
	remotoAlterado.EvidenciaEmitidaEn = time.Time{}
	remotoAlterado.EvidenciaRecibidaEn = time.Time{}
	remotoAlterado.EvidenciaVerificadaEn = time.Time{}
	if !errors.Is(remotoAlterado.Validar(), ErrEvidenciaCobroInvalida) {
		t.Fatal("un hecho remoto sin atestacion de verificacion fue aceptado")
	}
	instanteCancelacion := instanteBaseCobro.Add(2 * time.Minute)
	cancelada, _, err := enviada.Cancelar(
		"evidencia:cancelacion:matriz", huellaCobro('6'), "Cancelacion administrativa", instanteCancelacion,
		contextoOrdenCobro(t, enviada, AccionCobroCancelar, instanteCancelacion),
	)
	if err != nil {
		t.Fatal(err)
	}
	hechoLocal := cancelada.Historial[len(cancelada.Historial)-1]
	if hechoCobroEsRemoto(hechoLocal) || hechoLocal.ConectorID != "" || hechoLocal.HuellaMensajeOriginalSHA256 != "" ||
		hechoLocal.VerificacionEvidenciaRef != "" || cancelada.Validar() != nil {
		t.Fatal("la decision local heredo datos de procedencia remota")
	}
	hechoRemoto := enviada.Historial[len(enviada.Historial)-1]
	if hechoRemoto.HuellaDecisionSHA256 == "" || hechoRemoto.VerificacionEvidenciaRef == "" ||
		hechoRemoto.HuellaVerificacionSHA256 == "" || hechoRemoto.AutorizacionEvaluadaEn.IsZero() ||
		hechoRemoto.AutenticacionVerificadaEn.IsZero() {
		t.Fatal("el hecho remoto perdio la decision o la atestacion verificadora")
	}
	incidenciaLocalFalsificada := hechoLocal
	incidenciaLocalFalsificada.Tipo = HechoCobroIncidenciaDetectada
	incidenciaLocalFalsificada.AccionAutorizada = AccionCobroCancelar
	incidenciaLocalFalsificada.HuellaMensajeOriginalSHA256 = hechoRemoto.HuellaMensajeOriginalSHA256
	incidenciaLocalFalsificada.ConectorID = hechoRemoto.ConectorID
	incidenciaLocalFalsificada.VersionConector = hechoRemoto.VersionConector
	incidenciaLocalFalsificada.OperacionProveedorRef = hechoRemoto.OperacionProveedorRef
	incidenciaLocalFalsificada.VerificacionEvidenciaRef = hechoRemoto.VerificacionEvidenciaRef
	incidenciaLocalFalsificada.HuellaVerificacionSHA256 = hechoRemoto.HuellaVerificacionSHA256
	incidenciaLocalFalsificada.MetodoVerificacionEvidencia = hechoRemoto.MetodoVerificacionEvidencia
	incidenciaLocalFalsificada.AudienciaEvidencia = hechoRemoto.AudienciaEvidencia
	incidenciaLocalFalsificada.EvidenciaEmitidaEn = hechoRemoto.EvidenciaEmitidaEn
	incidenciaLocalFalsificada.EvidenciaRecibidaEn = hechoRemoto.EvidenciaRecibidaEn
	incidenciaLocalFalsificada.EvidenciaVerificadaEn = hechoRemoto.EvidenciaVerificadaEn
	if matrizCamposHechoCobroValida(incidenciaLocalFalsificada) {
		t.Fatal("una incidencia local pudo autodeclararse como evidencia remota")
	}
	incidenciaRemotaSinPrueba := hechoRemoto
	incidenciaRemotaSinPrueba.Tipo = HechoCobroIncidenciaDetectada
	incidenciaRemotaSinPrueba.AccionAutorizada = AccionCobroProcesarResultado
	incidenciaRemotaSinPrueba.HuellaMensajeOriginalSHA256 = ""
	incidenciaRemotaSinPrueba.ConectorID = ""
	incidenciaRemotaSinPrueba.VersionConector = 0
	incidenciaRemotaSinPrueba.OperacionProveedorRef = ""
	incidenciaRemotaSinPrueba.VerificacionEvidenciaRef = ""
	incidenciaRemotaSinPrueba.HuellaVerificacionSHA256 = ""
	incidenciaRemotaSinPrueba.MetodoVerificacionEvidencia = ""
	incidenciaRemotaSinPrueba.AudienciaEvidencia = ""
	incidenciaRemotaSinPrueba.EvidenciaEmitidaEn = time.Time{}
	incidenciaRemotaSinPrueba.EvidenciaRecibidaEn = time.Time{}
	incidenciaRemotaSinPrueba.EvidenciaVerificadaEn = time.Time{}
	if matrizCamposHechoCobroValida(incidenciaRemotaSinPrueba) {
		t.Fatal("una incidencia remota pudo omitir la prueba de verificacion")
	}
}

func TestIncidenciaConEvidenciaAntiguaUsaHoraConfiableDeDeteccion(t *testing.T) {
	orden := nuevaOrdenCobroPrueba(t)
	instanteInicio := instanteBaseCobro.Add(time.Minute)
	orden, _ = enviarOrdenCobroPrueba(t, orden, instanteInicio)
	orden, _ = aplicarResultadoCobroPrueba(
		t, orden, ResultadoOperacionCobroConfirmado, "confirmada-incidencia", '7', instanteBaseCobro.Add(2*time.Minute),
	)
	datos := datosServidorCobro(orden, "conflicto-antiguo", '8', instanteInicio)
	datos.EvidenciaRef = orden.Historial[1].EvidenciaRef
	evidencia, err := NuevaEvidenciaResultadoCobroVerificada(datos, ResultadoOperacionCobroConfirmado)
	if err != nil {
		t.Fatal(err)
	}
	detectadaEn := instanteBaseCobro.Add(3 * time.Minute)
	bloqueada, repetida, err := orden.AplicarResultadoServidor(
		evidencia, detectadaEn,
		contextoOrdenCobro(t, orden, AccionCobroProcesarResultado, detectadaEn),
		"Replay conflictivo detectado",
	)
	if err != nil || repetida || bloqueada.Estado != EstadoCobroIncidenciaBloqueada {
		t.Fatalf("la evidencia antigua impidio bloquear: estado=%q repetida=%v err=%v", bloqueada.Estado, repetida, err)
	}
	hecho := bloqueada.Historial[len(bloqueada.Historial)-1]
	if !hecho.OcurridoEn.Equal(detectadaEn) || !hecho.EvidenciaRecibidaEn.Equal(instanteInicio) ||
		hecho.OcurridoEn.Equal(hecho.EvidenciaRecibidaEn) {
		t.Fatalf("no se separaron recepcion y deteccion: recibido=%s detectado=%s", hecho.EvidenciaRecibidaEn, hecho.OcurridoEn)
	}
}
