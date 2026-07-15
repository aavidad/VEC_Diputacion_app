package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type pasarelaCobroContratoPrueba struct{}

type selladorSolicitudCobroContratoPrueba struct{}

func (selladorSolicitudCobroContratoPrueba) SellarIndiceAltaCobro(context.Context, []byte) (string, error) {
	return "hmac-sha256:pagos-v1:" + huellaPuertoCobro('1'), nil
}
func (selladorSolicitudCobroContratoPrueba) SellarHuellaPeticionCobro(context.Context, []byte) (string, error) {
	return "hmac-sha256:peticion-v1:" + huellaPuertoCobro('2'), nil
}
func (selladorSolicitudCobroContratoPrueba) SellarIndiceDevolucionCobro(context.Context, []byte) (string, error) {
	return "hmac-sha256:devoluciones-v1:" + huellaPuertoCobro('3'), nil
}

func (pasarelaCobroContratoPrueba) Capacidades(context.Context) (CapacidadesPasarelaCobro, error) {
	return CapacidadesPasarelaCobro{}, nil
}
func (pasarelaCobroContratoPrueba) CrearOperacion(context.Context, SolicitudOperacionCobro) (InicioOperacionCobro, error) {
	return InicioOperacionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) ConsultarOperacion(context.Context, ReferenciaOperacionCobro) (ResultadoOperacionCobro, error) {
	return ResultadoOperacionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) VerificarNotificacionCobro(context.Context, NotificacionCobro) (ResultadoOperacionCobro, error) {
	return ResultadoOperacionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) VerificarNotificacionDevolucion(context.Context, NotificacionCobro, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error) {
	return ResultadoDevolucionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) SolicitarDevolucion(context.Context, SolicitudDevolucionCobro) (ResultadoDevolucionCobro, error) {
	return ResultadoDevolucionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) ConsultarDevolucion(context.Context, ReferenciaDevolucionCobro) (ResultadoDevolucionCobro, error) {
	return ResultadoDevolucionCobro{}, nil
}
func (pasarelaCobroContratoPrueba) Conciliar(context.Context, SolicitudConciliacionCobro) (ResultadoConciliacionCobro, error) {
	return ResultadoConciliacionCobro{}, nil
}

var _ PasarelaCobro = pasarelaCobroContratoPrueba{}
var _ SelladorSolicitudCobro = selladorSolicitudCobroContratoPrueba{}

var instantePuertoCobro = time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

const (
	sesionPuertoCobro      = "ses_0123456789abcdefghijkl"
	personaPuertoCobro     = "per_0123456789abcdefghijkl"
	sujetoPuertoCobro      = "per_abcdefghijkl0123456789"
	perfilPuertoCobro      = "prf_0123456789abcdefghijkl"
	tokenReservaCobro      = "res_cob_0123456789abcdefghijkl"
	tokenReservaDevolucion = "res_dev_0123456789abcdefghijkl"
)

func huellaPuertoCobro(caracter byte) string { return strings.Repeat(string(caracter), 64) }

func altaPuertoCobro() domain.AltaOrdenCobro {
	return domain.AltaOrdenCobro{
		ID:                     "cob_0123456789abcdefghijkl",
		IndiceIdempotenciaHMAC: "hmac-sha256:pagos-v1:" + huellaPuertoCobro('a'),
		ExpedienteRef:          "expediente:uno",
		SolicitudRef:           "solicitud:uno",
		LiquidacionRef:         "liquidacion:uno",
		Tarifa: domain.ReferenciaTarifaCobro{
			TarifaID: "tasa_inscripcion", Version: 2, HuellaSHA256: huellaPuertoCobro('b'),
			ReglaCalculoRef: "regla:tarifa:dos",
		},
		SujetoRef: sujetoPuertoCobro, Importe: domain.DineroCobro{UnidadesMenores: 2_735, Moneda: "EUR"},
		Concepto: "Tasa de inscripcion", Finalidad: "tramitar_solicitud_seleccion",
		CorrelacionRef: "correlacion:uno", CreadaEn: instantePuertoCobro,
		CaducaEn: instantePuertoCobro.Add(30 * time.Minute), EvidenciaCreacionRef: "liquidacion:uno",
		HuellaEvidenciaSHA256: huellaPuertoCobro('c'), Motivo: "Tarifa publicada aplicable",
	}
}

func decisionPuertoCobro(
	t *testing.T,
	accion domain.AccionCobro,
	recurso, finalidad, correlacion string,
	instante time.Time,
) (domain.DecisionAutorizacion, domain.RecursoAutorizable) {
	t.Helper()
	campos, existe := domain.CamposRequeridosAccionCobro(accion)
	if !existe {
		t.Fatalf("accion de cobro desconocida: %q", accion)
	}
	recursoAutorizable := domain.RecursoAutorizable{Referencia: recurso, ModuloID: "pagos", Tipo: "orden_cobro"}
	huellaContexto, err := recursoAutorizable.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella de contexto: %v", err)
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatalf("huella de catalogo: %v", err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:" + strings.ReplaceAll(string(accion), ".", ":"),
		Concedida:   true, Codigo: "concedida", PrincipalID: personaPuertoCobro,
		PerfilActivoRef: perfilPuertoCobro, Accion: string(accion), RecursoRef: recurso,
		ModuloID: "pagos", TipoRecurso: "orden_cobro", ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: finalidad, CorrelacionRef: correlacion,
		VinculoAutenticacionActor: vinculoAutenticacionActorPuertoPrueba(t, instante),
		AsignacionRef:             "asignacion:tesoreria:uno", AsignacionHuellaSHA256: huellaPuertoCobro('d'),
		VersionRolRef: "rol:tesoreria:v2", VersionRolHuellaSHA256: huellaPuertoCobro('e'),
		ControlVigenciaVersionRolRef:      "rol:tesoreria:v2",
		ControlVigenciaVersionRolRevision: 1, ControlVigenciaVersionRolHuellaSHA256: huellaPuertoCobro('f'),
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  domain.AuthAssuranceHigh,
		CamposPermitidos:                campos,
		EmitidaEn:                       instante.Add(-30 * time.Second), ValidaHasta: instante.Add(4 * time.Minute),
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("crear decision reforzada de cobro: %v", err)
	}
	return decision, recursoAutorizable
}

func contextoPuertoCobro(t *testing.T, accion domain.AccionCobro, recurso, finalidad, correlacion string, instante time.Time) domain.ContextoAutorizacionCobro {
	t.Helper()
	decision, recursoAutorizable := decisionPuertoCobro(
		t, accion, recurso, finalidad, correlacion, instante,
	)
	atestacion, err := domain.NuevaAtestacionAutenticacionCobro(context.Background(), verificadorAutenticacionPuertoCobro{
		resultado: domain.ResultadoVerificacionAutenticacionCobro{
			PrincipalRef: personaPuertoCobro, Metodo: domain.AuthMethodCertificate,
			Garantia: domain.AuthAssuranceHigh, AutenticacionRef: "aut_0123456789abcdefghijkl",
			SesionRef: sesionPuertoCobro, HuellaSesionHMAC: "hmac-sha256:sesion-v1:" + huellaPuertoCobro('9'),
			EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(10 * time.Minute),
		},
	}, sesionPuertoCobro, "hmac-sha256:sesion-v1:"+huellaPuertoCobro('9'), instante)
	if err != nil {
		t.Fatalf("crear atestacion: %v", err)
	}
	resultado, err := domain.NuevoContextoAutorizacionCobro(decision, atestacion, recursoAutorizable, instante)
	if err != nil {
		t.Fatalf("crear contexto: %v", err)
	}
	return resultado
}

type verificadorAutenticacionPuertoCobro struct {
	resultado domain.ResultadoVerificacionAutenticacionCobro
}

func (v verificadorAutenticacionPuertoCobro) VerificarAutenticacionCobro(
	context.Context,
	domain.SolicitudVerificacionAutenticacionCobro,
) (domain.ResultadoVerificacionAutenticacionCobro, error) {
	return v.resultado, nil
}

func nuevaOrdenPuertoCobro(t *testing.T) domain.OrdenCobro {
	t.Helper()
	alta := altaPuertoCobro()
	autorizacion := contextoPuertoCobro(t, domain.AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn)
	orden, err := domain.NuevaOrdenCobro(alta, autorizacion)
	if err != nil {
		t.Fatalf("crear orden: %v", err)
	}
	return orden
}

func contextoOrdenPuertoCobro(t *testing.T, orden domain.OrdenCobro, accion domain.AccionCobro, instante time.Time) domain.ContextoAutorizacionCobro {
	t.Helper()
	return contextoPuertoCobro(t, accion, orden.ID, orden.Finalidad, orden.CorrelacionRef, instante)
}

func datosEvidenciaPuertoCobro(orden domain.OrdenCobro, sufijo string, caracter byte, instante time.Time) domain.DatosEvidenciaServidorCobro {
	return domain.DatosEvidenciaServidorCobro{
		EvidenciaRef: "evidencia:pasarela:" + sufijo, HuellaSHA256: huellaPuertoCobro(caracter),
		ConectorID: "pasarela_corporativa", VersionConector: 2, OrdenRef: orden.ID,
		LiquidacionRef: orden.LiquidacionRef, OperacionProveedorRef: "operacion:opaca:uno",
		Importe: orden.Importe, Concepto: orden.Concepto, Codigo: "resultado_" + sufijo,
		MetodoAutenticacion: domain.MetodoAutenticacionCobroFirmaYTLSMutuo, Audiencia: "vec.cobros",
		VerificacionRef: "verificacion:pasarela:" + sufijo, HuellaVerificacionSHA256: huellaPuertoCobro('8'),
		EmitidaEn: instante.Add(-time.Second), RecibidaEn: instante, VerificadaEn: instante,
	}
}

func ordenEnviadaPuertoCobro(t *testing.T) domain.OrdenCobro {
	t.Helper()
	orden := nuevaOrdenPuertoCobro(t)
	instante := instantePuertoCobro.Add(time.Minute)
	evidencia, err := domain.NuevaEvidenciaInicioOperacionCobroVerificada(datosEvidenciaPuertoCobro(orden, "inicio", 'd', instante))
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err := orden.RegistrarEnvio(evidencia, instante, contextoOrdenPuertoCobro(t, orden, domain.AccionCobroIniciarOperacion, instante), "Inicio alojado")
	if err != nil || repetida {
		t.Fatalf("registrar envio: repetida=%v err=%v", repetida, err)
	}
	return orden
}

func ordenConfirmadaPuertoCobro(t *testing.T) domain.OrdenCobro {
	t.Helper()
	orden := ordenEnviadaPuertoCobro(t)
	instante := instantePuertoCobro.Add(2 * time.Minute)
	evidencia, err := domain.NuevaEvidenciaResultadoCobroVerificada(
		datosEvidenciaPuertoCobro(orden, "confirmada", 'e', instante), domain.ResultadoOperacionCobroConfirmado,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, repetida, err := orden.AplicarResultadoServidor(evidencia, instante, contextoOrdenPuertoCobro(t, orden, domain.AccionCobroProcesarResultado, instante), "Confirmacion autenticada")
	if err != nil || repetida {
		t.Fatalf("confirmar: repetida=%v err=%v", repetida, err)
	}
	return orden
}

func TestCapacidadesPasarelaCobroExigenCanalServidor(t *testing.T) {
	valida := CapacidadesPasarelaCobro{
		ConectorID: "pasarela_corporativa", VersionConector: 2, RedireccionAlojada: true,
		NotificacionAutenticada: true, ConsultaOperacion: true, Devolucion: true, Conciliacion: true,
		IdempotenciaProveedor: true,
	}
	if err := valida.Validar(); err != nil {
		t.Fatalf("capacidades validas: %v", err)
	}
	sinIdempotencia := valida
	sinIdempotencia.IdempotenciaProveedor = false
	if !errors.Is(sinIdempotencia.Validar(), ErrCapacidadPasarelaCobroNoDisponible) {
		t.Fatal("se habilito una pasarela sin idempotencia contractual")
	}
	casos := []CapacidadesPasarelaCobro{
		{},
		{ConectorID: "pasarela_corporativa", VersionConector: 2, NotificacionAutenticada: true},
		{ConectorID: "pasarela_corporativa", VersionConector: 2, RedireccionAlojada: true},
		{ConectorID: "Pasarela", VersionConector: 2, RedireccionAlojada: true, ConsultaOperacion: true},
	}
	for indice, capacidades := range casos {
		if !errors.Is(capacidades.Validar(), ErrCapacidadPasarelaCobroNoDisponible) {
			t.Errorf("caso %d debio fallar cerrado", indice)
		}
	}
}

func TestSolicitudesRemotasSoloAceptanComandosDerivados(t *testing.T) {
	orden := nuevaOrdenPuertoCobro(t)
	comandoInicio, err := orden.PrepararInicioOperacion(
		"retorno:publico:uno", "notificacion:interna:uno", instantePuertoCobro.Add(time.Minute),
		contextoOrdenPuertoCobro(t, orden, domain.AccionCobroIniciarOperacion, instantePuertoCobro.Add(time.Minute)),
	)
	if err != nil || (SolicitudOperacionCobro{Comando: comandoInicio}).Validar() != nil {
		t.Fatalf("solicitud de inicio valida rechazada: %v", err)
	}
	if !errors.Is((SolicitudOperacionCobro{}).Validar(), ErrSolicitudOperacionCobroInvalida) {
		t.Fatal("una solicitud de inicio sin comando fue aceptada")
	}

	confirmada := ordenConfirmadaPuertoCobro(t)
	comandoConciliar, err := confirmada.PrepararConciliacion(
		domain.TipoConciliacionCobroIngreso, "cierre:uno", instantePuertoCobro.Add(3*time.Minute),
		contextoOrdenPuertoCobro(t, confirmada, domain.AccionCobroConciliar, instantePuertoCobro.Add(3*time.Minute)),
	)
	if err != nil || (SolicitudConciliacionCobro{Comando: comandoConciliar}).Validar() != nil {
		t.Fatalf("solicitud de conciliacion valida rechazada: %v", err)
	}
	solicitudDevolucion := domain.SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:uno",
		HuellaEvidenciaSHA256:  huellaPuertoCobro('f'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaPuertoCobro('1'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instantePuertoCobro.Add(3 * time.Minute),
	}
	_, comandoDevolucion, repetida, err := confirmada.SolicitarDevolucion(
		solicitudDevolucion,
		contextoOrdenPuertoCobro(t, confirmada, domain.AccionCobroSolicitarDevolucion, solicitudDevolucion.SolicitadaEn),
	)
	if err != nil || repetida || (SolicitudDevolucionCobro{Comando: comandoDevolucion}).Validar() != nil {
		t.Fatalf("solicitud de devolucion valida rechazada: repetida=%v err=%v", repetida, err)
	}
	if !errors.Is((SolicitudDevolucionCobro{}).Validar(), ErrSolicitudDevolucionCobroInvalida) ||
		!errors.Is((SolicitudConciliacionCobro{}).Validar(), ErrSolicitudConciliacionCobroInvalida) {
		t.Fatal("un comando remoto cero fue aceptado")
	}
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(SolicitudOperacionCobro{}), reflect.TypeOf(SolicitudDevolucionCobro{}), reflect.TypeOf(SolicitudConciliacionCobro{}),
	} {
		if tipo.NumField() != 1 || tipo.Field(0).Name != "Comando" {
			t.Fatalf("%s permite parametros libres fuera del comando sellado", tipo.Name())
		}
	}
}

func origenPuertoCobroValido() OrigenPasarelaCobroPublicado {
	origen := OrigenPasarelaCobroPublicado{
		ID: "pasarela_corporativa", Version: 2, BaseHTTPS: "https://pagos.example.test",
		RutasPermitidas: []string{"/operaciones/iniciar"}, CamposHandoffPermitidos: []string{"firma", "operacion"},
		PublicadaEn: instantePuertoCobro.Add(-time.Hour),
	}
	huella, err := CalcularHuellaConfiguracionOrigenPasarelaCobro(origen)
	if err != nil {
		panic(err)
	}
	origen.HuellaConfiguracionSHA256 = huella
	return origen
}

func inicioPuertoCobroValido(t *testing.T) (InicioOperacionCobro, domain.ComandoInicioOperacionCobro, OrigenPasarelaCobroPublicado) {
	t.Helper()
	orden := nuevaOrdenPuertoCobro(t)
	comando, err := orden.PrepararInicioOperacion(
		"retorno:publico:uno", "notificacion:interna:uno", instantePuertoCobro,
		contextoOrdenPuertoCobro(t, orden, domain.AccionCobroIniciarOperacion, instantePuertoCobro),
	)
	if err != nil {
		t.Fatal(err)
	}
	datosComando, _ := comando.Datos()
	evidencia, err := domain.NuevaEvidenciaInicioOperacionCobroVerificada(
		datosEvidenciaPuertoCobro(orden, "handoff", '3', instantePuertoCobro),
	)
	if err != nil {
		t.Fatal(err)
	}
	origen := origenPuertoCobroValido()
	carga, err := NuevaCargaHandoffCobro(
		[]CampoHandoffCobro{{Nombre: "firma", Valor: "firma-opaca"}, {Nombre: "operacion", Valor: "operacion-opaca"}},
		origen.CamposHandoffPermitidos,
	)
	if err != nil {
		t.Fatal(err)
	}
	inicio := InicioOperacionCobro{
		Evidencia: evidencia, Origen: origen, Ruta: "/operaciones/iniciar",
		VersionOrden: datosComando.VersionOrden, HuellaOrdenSHA256: datosComando.HuellaOrdenSHA256,
		HuellaConfiguracionSHA256: origen.HuellaConfiguracionSHA256,
		Metodo:                    MetodoHandoffCobroPOSTFormulario, Carga: carga,
		GeneradaEn: instantePuertoCobro, ExpiraEn: instantePuertoCobro.Add(10 * time.Minute),
	}
	return inicio, comando, origen
}

func TestHandoffUsaOrigenPublicadoPOSTListaCerradaYTTL(t *testing.T) {
	valido, comando, origen := inicioPuertoCobroValido(t)
	if err := valido.Validar(); err != nil {
		t.Fatalf("inicio valido: %v", err)
	}
	if err := valido.ValidarContra(comando, origen, instantePuertoCobro); err != nil {
		t.Fatalf("inicio no quedo ligado a comando y origen: %v", err)
	}
	basesInvalidas := []string{
		"http://pagos.example.test", "https://usuario:clave@pagos.example.test",
		"https://pagos.example.test?token=secreto", "https://pagos.example.test?",
		"https://pagos.example.test/#fragmento", "https://pagos.example.test/ruta", "https://:443",
		" https://pagos.example.test", "https://pagos.example.test ",
	}
	for _, base := range basesInvalidas {
		alterado := valido
		alterado.Origen.BaseHTTPS = base
		if !errors.Is(alterado.Validar(), ErrInicioOperacionCobroInvalido) {
			t.Errorf("origen %q debio rechazarse", base)
		}
	}
	for _, ruta := range []string{
		"//evil.test", "/operaciones/iniciar?secreto=uno", "/no-publicada", "https://evil.test",
		" /operaciones/iniciar", "/operaciones/iniciar ", "/operaciones//iniciar", "/operaciones/../iniciar",
		"/operaciones/%2e%2e/iniciar", "/operaciones\\iniciar",
	} {
		alterado := valido
		alterado.Ruta = ruta
		if !errors.Is(alterado.Validar(), ErrInicioOperacionCobroInvalido) {
			t.Errorf("ruta %q debio rechazarse", ruta)
		}
	}
	for _, rutaPublicada := range []string{
		" /operaciones/iniciar", "/operaciones/iniciar ", "/operaciones//iniciar",
		"/operaciones/../iniciar", "/operaciones/%2e%2e/iniciar", "/operaciones\\iniciar",
	} {
		origenInseguro := origen
		origenInseguro.RutasPermitidas = []string{rutaPublicada}
		origenInseguro.HuellaConfiguracionSHA256 = ""
		if _, err := CalcularHuellaConfiguracionOrigenPasarelaCobro(origenInseguro); !errors.Is(err, ErrInicioOperacionCobroInvalido) {
			t.Errorf("la configuracion acepto ruta no canonica %q: %v", rutaPublicada, err)
		}
	}
	for nombre, muta := range map[string]func(*InicioOperacionCobro){
		"get libre":     func(i *InicioOperacionCobro) { i.Metodo = MetodoHandoffCobro("get") },
		"otro conector": func(i *InicioOperacionCobro) { i.Origen.ID = "otra_pasarela" },
		"otra version":  func(i *InicioOperacionCobro) { i.Origen.Version++ },
		"evidencia demasiado vieja": func(i *InicioOperacionCobro) {
			i.GeneradaEn = i.GeneradaEn.Add(3 * time.Minute)
			i.ExpiraEn = i.GeneradaEn.Add(time.Minute)
		},
		"ttl excesivo":         func(i *InicioOperacionCobro) { i.ExpiraEn = i.GeneradaEn.Add(16 * time.Minute) },
		"expiracion invertida": func(i *InicioOperacionCobro) { i.ExpiraEn = i.GeneradaEn },
		"config futura":        func(i *InicioOperacionCobro) { i.Origen.PublicadaEn = i.GeneradaEn.Add(time.Second) },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := valido
			muta(&alterado)
			if !errors.Is(alterado.Validar(), ErrInicioOperacionCobroInvalido) {
				t.Fatal("la alteracion fue aceptada")
			}
		})
	}
	origenDistinto := origen
	origenDistinto.BaseHTTPS = "https://otra-pasarela.example.test"
	huellaDistinta, err := CalcularHuellaConfiguracionOrigenPasarelaCobro(origenDistinto)
	if err != nil {
		t.Fatal(err)
	}
	origenDistinto.HuellaConfiguracionSHA256 = huellaDistinta
	if origenDistinto.Validar() != nil {
		t.Fatal("el segundo origen publicado de prueba debia ser estructuralmente valido")
	}
	if !errors.Is(valido.ValidarContra(comando, origenDistinto, instantePuertoCobro), ErrInicioOperacionCobroInvalido) {
		t.Fatal("se acepto un origen valido pero distinto del publicado para el comando")
	}
	if !errors.Is(valido.ValidarContra(comando, origen, valido.ExpiraEn), ErrInicioOperacionCobroInvalido) {
		t.Fatal("se entrego un handoff caducado")
	}
	otraAlta := altaPuertoCobro()
	otraAlta.ID = "cob_abcdefghijkl0123456789"
	otraAlta.LiquidacionRef = "liquidacion:dos"
	otraAlta.CorrelacionRef = "correlacion:dos"
	otraOrden, err := domain.NuevaOrdenCobro(
		otraAlta,
		contextoPuertoCobro(t, domain.AccionCobroCrearOrden, otraAlta.LiquidacionRef,
			otraAlta.Finalidad, otraAlta.CorrelacionRef, otraAlta.CreadaEn),
	)
	if err != nil {
		t.Fatal(err)
	}
	otroComando, err := otraOrden.PrepararInicioOperacion(
		"retorno:publico:dos", "notificacion:interna:dos", instantePuertoCobro,
		contextoOrdenPuertoCobro(t, otraOrden, domain.AccionCobroIniciarOperacion, instantePuertoCobro),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(valido.ValidarContra(otroComando, origen, instantePuertoCobro), ErrInicioOperacionCobroInvalido) {
		t.Fatal("se reutilizo el handoff con otro comando")
	}
}

func TestCargaHandoffEsAcotadaDefensivaYNoFiltraSecretos(t *testing.T) {
	permitidos := []string{"firma", "operacion"}
	carga, err := NuevaCargaHandoffCobro([]CampoHandoffCobro{{Nombre: "firma", Valor: "secreto-opaco"}}, permitidos)
	if err != nil {
		t.Fatal(err)
	}
	inicio, comando, origen := inicioPuertoCobroValido(t)
	inicio.Carga = carga
	primera, _ := inicio.CamposRespuestaPOSTContra(comando, origen, instantePuertoCobro)
	primera[0].Valor = "alterado"
	segunda, _ := inicio.CamposRespuestaPOSTContra(comando, origen, instantePuertoCobro)
	if segunda[0].Valor != "secreto-opaco" {
		t.Fatal("CamposRespuestaPOSTContra compartio memoria mutable")
	}
	if _, err := json.Marshal(carga); !errors.Is(err, ErrInicioOperacionCobroInvalido) {
		t.Fatalf("la carga se serializo genericamente: %v", err)
	}
	if strings.Contains(fmt.Sprintf("%+v", carga), "secreto-opaco") {
		t.Fatal("el formateo filtro la carga de handoff")
	}
	casos := [][]CampoHandoffCobro{
		{{Nombre: "no_permitido", Valor: "opaco"}},
		{{Nombre: "firma", Valor: "uno"}, {Nombre: "firma", Valor: "dos"}},
		{{Nombre: " firma", Valor: "opaco"}},
		{{Nombre: "firma ", Valor: "opaco"}},
		{{Nombre: "firma", Valor: " opaco"}},
		{{Nombre: "firma", Valor: "opaco "}},
		{{Nombre: "firma", Valor: "4111 1111 1111 1111"}},
		{{Nombre: "firma", Valor: "4111\u200b1111\u200b1111\u200b1111"}},
		{{Nombre: "firma", Valor: "４１１１-１１１１-１１１１-１１１１"}},
		{{Nombre: "firma", Valor: "card_number"}},
	}
	for indice, campos := range casos {
		if _, err := NuevaCargaHandoffCobro(campos, permitidos); !errors.Is(err, ErrInicioOperacionCobroInvalido) {
			t.Errorf("carga insegura %d fue aceptada", indice)
		}
	}
}

func TestNotificacionSoloEsReferenciaOpacaConCustodiaUnica(t *testing.T) {
	notificacion := NotificacionCobro{
		ConectorID: "pasarela_corporativa", VersionConector: 2,
		RecepcionRef: "rec_0123456789abcdefghijkl", Audiencia: "vec.cobros", RecibidaEn: instantePuertoCobro,
	}
	if err := notificacion.Validar(); err != nil {
		t.Fatalf("notificacion valida: %v", err)
	}
	metadatos := SolicitudCustodiarNotificacionCobro{
		ConectorID: notificacion.ConectorID, VersionConector: notificacion.VersionConector,
		RecepcionRef: notificacion.RecepcionRef, Audiencia: notificacion.Audiencia,
		TipoContenido: "application/json", Tamano: 512, HuellaSHA256: huellaPuertoCobro('4'),
		RecibidaEn: instantePuertoCobro, ExpiraEn: instantePuertoCobro.Add(10 * time.Minute),
	}
	if err := metadatos.Validar(); err != nil {
		t.Fatalf("metadatos validos: %v", err)
	}
	contenido := ContenidoNotificacionCobroUnico{Metadatos: metadatos, Contenido: io.NopCloser(strings.NewReader("opaco"))}
	if err := contenido.Validar(); err != nil {
		t.Fatalf("contenido unico valido: %v", err)
	}
	contenido.Contenido = nil
	if !errors.Is(contenido.Validar(), ErrNotificacionCobroInvalida) {
		t.Fatal("se acepto contenido consumido o ausente")
	}
	for nombre, muta := range map[string]func(*SolicitudCustodiarNotificacionCobro){
		"cuerpo vacio":        func(s *SolicitudCustodiarNotificacionCobro) { s.Tamano = 0 },
		"cuerpo grande":       func(s *SolicitudCustodiarNotificacionCobro) { s.Tamano = 1024*1024 + 1 },
		"huella":              func(s *SolicitudCustodiarNotificacionCobro) { s.HuellaSHA256 = "sha256" },
		"referencia":          func(s *SolicitudCustodiarNotificacionCobro) { s.RecepcionRef = "webhook-1" },
		"ttl":                 func(s *SolicitudCustodiarNotificacionCobro) { s.ExpiraEn = s.RecibidaEn.Add(16 * time.Minute) },
		"otra audiencia":      func(s *SolicitudCustodiarNotificacionCobro) { s.Audiencia = "otro.servicio" },
		"dato de tarjeta":     func(s *SolicitudCustodiarNotificacionCobro) { s.Audiencia = "PAN" },
		"mime no enumerado":   func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = "text/plain" },
		"mime con mayusculas": func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = "Application/JSON" },
		"mime con prefijo":    func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = " application/json" },
		"mime con sufijo":     func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = "application/json " },
		"mime con parametros": func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = "application/json; charset=utf-8" },
		"tipo con control":    func(s *SolicitudCustodiarNotificacionCobro) { s.TipoContenido = "application/json\nmalicioso" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterados := metadatos
			muta(&alterados)
			if !errors.Is(alterados.Validar(), ErrNotificacionCobroInvalida) {
				t.Fatal("metadatos inseguros aceptados")
			}
		})
	}
}

func mutacionCreacionPuertoCobro(t *testing.T) MutacionOrdenCobro {
	t.Helper()
	orden := nuevaOrdenPuertoCobro(t)
	mutacion, err := NuevaMutacionOrdenCobro(orden)
	if err != nil {
		t.Fatal(err)
	}
	return mutacion
}

func TestReservaRepositorioOCCAuditoriaYOutboxSonEstrictos(t *testing.T) {
	solicitud := SolicitudReservaOrdenCobro{
		OrdenRef:               "cob_0123456789abcdefghijkl",
		IndiceIdempotenciaHMAC: "hmac-sha256:pagos-v1:" + huellaPuertoCobro('5'),
		HuellaSolicitudHMAC:    "hmac-sha256:peticion-v1:" + huellaPuertoCobro('6'),
		PrincipalRef:           personaPuertoCobro, SolicitadaEn: instantePuertoCobro,
		ExpiraEn: instantePuertoCobro.Add(5 * time.Minute),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("reserva valida: %v", err)
	}
	for nombre, muta := range map[string]func(*SolicitudReservaOrdenCobro){
		"orden con prefijo": func(s *SolicitudReservaOrdenCobro) { s.OrdenRef = " " + s.OrdenRef },
		"idempotencia con sufijo": func(s *SolicitudReservaOrdenCobro) {
			s.IndiceIdempotenciaHMAC += " "
		},
		"peticion con prefijo": func(s *SolicitudReservaOrdenCobro) { s.HuellaSolicitudHMAC = " " + s.HuellaSolicitudHMAC },
		"principal con sufijo": func(s *SolicitudReservaOrdenCobro) { s.PrincipalRef += " " },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := solicitud
			muta(&alterada)
			if !errors.Is(alterada.Validar(), ErrReservaOrdenCobroInvalida) {
				t.Fatal("la reserva normalizo una referencia o un dominio")
			}
		})
	}
	if err := (ReservaOrdenCobro{Token: tokenReservaCobro}).Validar(); err != nil {
		t.Fatalf("resultado de reserva nueva invalido: %v", err)
	}
	for _, token := range []string{" " + tokenReservaCobro, tokenReservaCobro + " ", tokenReservaDevolucion} {
		if !errors.Is((ReservaOrdenCobro{Token: token}).Validar(), ErrReservaOrdenCobroInvalida) {
			t.Fatalf("se acepto token de reserva de alta no exacto: %q", token)
		}
	}
	reservaSecreta := ReservaOrdenCobro{Token: tokenReservaCobro}
	if _, err := json.Marshal(reservaSecreta); !errors.Is(err, ErrReservaOrdenCobroInvalida) ||
		strings.Contains(fmt.Sprintf("%#v", reservaSecreta), reservaSecreta.Token) {
		t.Fatal("la reserva interna filtro su token")
	}
	orden := nuevaOrdenPuertoCobro(t)
	if err := (ReservaOrdenCobro{Repetida: true, Orden: &orden}).Validar(); err != nil {
		t.Fatalf("resultado idempotente invalido: %v", err)
	}
	if (ReservaOrdenCobro{Repetida: true, Token: "reserva:indebida", Orden: &orden}).Validar() == nil ||
		(ReservaOrdenCobro{}).Validar() == nil {
		t.Fatal("una reserva ambigua fue aceptada")
	}
	solicitud.ExpiraEn = solicitud.SolicitadaEn.Add(16 * time.Minute)
	if !errors.Is(solicitud.Validar(), ErrReservaOrdenCobroInvalida) {
		t.Fatal("una reserva larga fue aceptada")
	}
	solicitud.ExpiraEn = solicitud.SolicitadaEn.Add(5 * time.Minute)
	solicitud.IndiceIdempotenciaHMAC = "hmac-sha256:devoluciones-v1:" + huellaPuertoCobro('5')
	if !errors.Is(solicitud.Validar(), ErrReservaOrdenCobroInvalida) {
		t.Fatal("se acepto una HMAC de devolucion para reservar un alta")
	}
	solicitud.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaPuertoCobro('5')
	if err := (SolicitudAbandonarReservaOrdenCobro{
		Token: tokenReservaCobro, OrdenRef: solicitud.OrdenRef, PrincipalRef: personaPuertoCobro,
		HuellaSolicitudHMAC: solicitud.HuellaSolicitudHMAC,
	}).Validar(); err != nil {
		t.Fatalf("abandono de alta tipado valido: %v", err)
	}

	mutacion := mutacionCreacionPuertoCobro(t)
	if err := mutacion.Validar(); err != nil {
		t.Fatalf("mutacion valida: %v", err)
	}
	datosCreacion, err := mutacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	decisionCreacion, _ := decisionPuertoCobro(
		t,
		domain.AccionCobroCrearOrden,
		datosCreacion.Orden.LiquidacionRef,
		datosCreacion.Orden.Finalidad,
		datosCreacion.Orden.CorrelacionRef,
		datosCreacion.Orden.CreadaEn,
	)
	evidenciaCreacion, err := NuevaEvidenciaUsoDecisionAutorizacion(
		decisionCreacion,
		datosCreacion.Orden.CreadaEn,
	)
	if err != nil {
		t.Fatalf("evidencia de autorizacion valida: %v", err)
	}
	contextoCreacion := contextoPuertoCobro(
		t,
		domain.AccionCobroCrearOrden,
		datosCreacion.Orden.LiquidacionRef,
		datosCreacion.Orden.Finalidad,
		datosCreacion.Orden.CorrelacionRef,
		datosCreacion.Orden.CreadaEn,
	)
	_, huellaEfectoCreacion, err := datosCreacion.Orden.ControlConcurrencia()
	if err != nil {
		t.Fatalf("huella del efecto valida: %v", err)
	}
	confirmacionCreacion := SolicitudConfirmarCreacionOrdenCobro{
		Token: tokenReservaCobro, OrdenRef: datosCreacion.Orden.ID,
		PrincipalRef:             datosCreacion.Auditoria.ActorRef,
		IndiceIdempotenciaHMAC:   datosCreacion.Orden.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:      "hmac-sha256:peticion-v1:" + huellaPuertoCobro('6'),
		ReservaSolicitadaEn:      datosCreacion.Orden.CreadaEn,
		ReservaExpiraEn:          datosCreacion.Orden.CreadaEn.Add(5 * time.Minute),
		DecisionAutorizacionRef:  datosCreacion.Auditoria.DecisionAutorizacionRef,
		HuellaDecisionSHA256:     datosCreacion.Auditoria.HuellaDecisionSHA256,
		DecisionValidaHasta:      datosCreacion.Auditoria.AutorizacionValidaHasta,
		HuellaEfectoSHA256:       huellaEfectoCreacion,
		EvidenciaAutorizacion:    evidenciaCreacion,
		ContextoAutorizacion:     contextoCreacion,
		SesionRef:                datosCreacion.Auditoria.SesionRef,
		HuellaSesionHMAC:         datosCreacion.Auditoria.HuellaSesionHMAC,
		SesionValidaHasta:        datosCreacion.Auditoria.AtestacionValidaHasta,
		LiquidacionRef:           datosCreacion.Orden.LiquidacionRef,
		LiquidacionRevision:      3,
		LiquidacionHuellaSHA256:  datosCreacion.Auditoria.HuellaEvidenciaSHA256,
		LiquidacionEstado:        EstadoControlLiquidacionCobroExigible,
		LiquidacionExigibleDesde: datosCreacion.Orden.CreadaEn.Add(-time.Minute),
		LiquidacionExigibleHasta: datosCreacion.Orden.CaducaEn,
		Mutacion:                 mutacion,
	}
	if err := confirmacionCreacion.Validar(); err != nil {
		t.Fatalf("confirmacion de creacion valida: %v", err)
	}
	for _, caso := range []struct {
		nombre string
		muta   func(*SolicitudConfirmarCreacionOrdenCobro)
	}{
		{"orden", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.OrdenRef = "cob_abcdefghijkl0123456789" }},
		{"principal", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.PrincipalRef = sujetoPuertoCobro }},
		{"indice HMAC", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaPuertoCobro('8')
		}},
		{"peticion HMAC", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.HuellaSolicitudHMAC = "huella-en-claro" }},
		{"inicio reserva", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.ReservaSolicitadaEn = s.ReservaSolicitadaEn.Add(time.Second)
		}},
		{"fin reserva", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.ReservaExpiraEn = s.ReservaSolicitadaEn }},
		{"decision", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.DecisionAutorizacionRef = "decision:otra" }},
		{"huella decision", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.HuellaDecisionSHA256 = huellaPuertoCobro('8') }},
		{"vigencia decision", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.DecisionValidaHasta = s.ReservaSolicitadaEn }},
		{"huella efecto", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.HuellaEfectoSHA256 = huellaPuertoCobro('8') }},
		{"evidencia decision", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.EvidenciaAutorizacion = EvidenciaUsoDecisionAutorizacion{}
		}},
		{"contexto decision", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.ContextoAutorizacion = domain.ContextoAutorizacionCobro{}
		}},
		{"sesion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.SesionRef = "ses_abcdefghijkl0123456789" }},
		{"huella sesion", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.HuellaSesionHMAC = "hmac-sha256:sesion-v1:" + huellaPuertoCobro('8')
		}},
		{"vigencia sesion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.SesionValidaHasta = s.ReservaSolicitadaEn }},
		{"liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.LiquidacionRef = "liquidacion:otra" }},
		{"revision liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.LiquidacionRevision = 0 }},
		{"huella liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.LiquidacionHuellaSHA256 = huellaPuertoCobro('8') }},
		{"estado liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) {
			s.LiquidacionEstado = EstadoControlLiquidacionCobro("anulada")
		}},
		{"inicio liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.LiquidacionExigibleDesde = s.ReservaExpiraEn }},
		{"fin liquidacion", func(s *SolicitudConfirmarCreacionOrdenCobro) { s.LiquidacionExigibleHasta = s.ReservaSolicitadaEn }},
	} {
		t.Run("confirmacion "+caso.nombre, func(t *testing.T) {
			alterada := confirmacionCreacion
			caso.muta(&alterada)
			if !errors.Is(alterada.Validar(), ErrMutacionOrdenCobroInvalida) {
				t.Fatal("se acepto un control de commit ausente, divergente o no canonico")
			}
		})
	}
	decisionMezclada := decisionCreacion
	decisionMezclada.AsignacionHuellaSHA256 = huellaPuertoCobro('7')
	evidenciaMezclada, err := NuevaEvidenciaUsoDecisionAutorizacion(
		decisionMezclada,
		datosCreacion.Orden.CreadaEn,
	)
	if err != nil {
		t.Fatalf("precondicion de decision B valida: %v", err)
	}
	confirmacionMezclada := confirmacionCreacion
	confirmacionMezclada.EvidenciaAutorizacion = evidenciaMezclada
	if !errors.Is(confirmacionMezclada.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("se mezclo el contexto opaco de la decision A con controles de la decision B")
	}
	casos := []struct {
		nombre string
		muta   func(*MutacionOrdenCobro)
	}{
		{"version auditoria", func(m *MutacionOrdenCobro) { m.datos.auditoria.VersionAnterior = 1 }},
		{"huella posterior", func(m *MutacionOrdenCobro) { m.datos.auditoria.HuellaPosteriorSHA256 = huellaPuertoCobro('f') }},
		{"huella origen creacion", func(m *MutacionOrdenCobro) { m.datos.auditoria.HuellaAnteriorSHA256 = huellaPuertoCobro('f') }},
		{"expediente", func(m *MutacionOrdenCobro) { m.datos.auditoria.ExpedienteRef = "expediente:dos" }},
		{"actor", func(m *MutacionOrdenCobro) { m.datos.auditoria.ActorRef = "persona:otra" }},
		{"perfil", func(m *MutacionOrdenCobro) { m.datos.auditoria.PerfilActivoRef = "perfil:otro" }},
		{"decision", func(m *MutacionOrdenCobro) { m.datos.auditoria.DecisionAutorizacionRef = "decision:otra" }},
		{"id auditoria", func(m *MutacionOrdenCobro) { m.datos.auditoria.ID = "aud_cob_0123456789abcdefghijkl" }},
		{"huella decision", func(m *MutacionOrdenCobro) { m.datos.auditoria.HuellaDecisionSHA256 = huellaPuertoCobro('1') }},
		{"instante decision", func(m *MutacionOrdenCobro) {
			m.datos.auditoria.AutorizacionEvaluadaEn = m.datos.auditoria.AutorizacionEvaluadaEn.Add(time.Second)
		}},
		{"evidencia", func(m *MutacionOrdenCobro) { m.datos.auditoria.EvidenciaRef = "evidencia:otra" }},
		{"atestacion", func(m *MutacionOrdenCobro) {
			m.datos.auditoria.AtestacionAutenticacionRef = "aut_otra23456789abcdefghijkl"
		}},
		{"sesion", func(m *MutacionOrdenCobro) { m.datos.auditoria.SesionRef = "ses_abcdefghijkl0123456789" }},
		{"huella sesion", func(m *MutacionOrdenCobro) {
			m.datos.auditoria.HuellaSesionHMAC = "hmac-sha256:sesion-v1:" + huellaPuertoCobro('8')
		}},
		{"metodo real", func(m *MutacionOrdenCobro) { m.datos.auditoria.MetodoAutenticacion = domain.AuthMethodKerberos }},
		{"garantia real", func(m *MutacionOrdenCobro) { m.datos.auditoria.GarantiaAutenticacion = domain.AuthAssuranceSubstantial }},
		{"accion", func(m *MutacionOrdenCobro) { m.datos.auditoria.Accion = domain.AccionCobroCancelar }},
		{"motivo", func(m *MutacionOrdenCobro) { m.datos.auditoria.Motivo = "Otro motivo" }},
		{"resultado", func(m *MutacionOrdenCobro) { m.datos.auditoria.Resultado = "confirmada" }},
		{"correlacion auditoria", func(m *MutacionOrdenCobro) { m.datos.auditoria.CorrelacionRef = "correlacion:dos" }},
		{"correlacion outbox", func(m *MutacionOrdenCobro) { m.datos.evento.CorrelacionRef = "correlacion:dos" }},
		{"version outbox", func(m *MutacionOrdenCobro) { m.datos.evento.VersionOrden++ }},
		{"tipo outbox", func(m *MutacionOrdenCobro) { m.datos.evento.Tipo = EventoCobroConfirmado }},
		{"hecho outbox", func(m *MutacionOrdenCobro) { m.datos.evento.Atributos.Hecho = domain.HechoCobroConfirmado }},
		{"estado outbox", func(m *MutacionOrdenCobro) { m.datos.evento.Atributos.Estado = domain.EstadoCobroConfirmada }},
		{"accion outbox", func(m *MutacionOrdenCobro) { m.datos.evento.Atributos.Accion = domain.AccionCobroCancelar }},
		{"metadato no enumerado", func(m *MutacionOrdenCobro) {
			m.datos.auditoria.Metadatos.Canal = CanalAuditoriaCobro("externo_inventado")
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := mutacion
			alterada.datos = &datosMutacionOrdenCobro{orden: mutacion.datos.orden.Clonar(), auditoria: mutacion.datos.auditoria, evento: mutacion.datos.evento}
			caso.muta(&alterada)
			if !errors.Is(alterada.Validar(), ErrMutacionOrdenCobroInvalida) {
				t.Fatal("mutacion incoherente aceptada")
			}
		})
	}
}

func TestReservaDevolucionEsUnicaTipadaYLigadaAOCC(t *testing.T) {
	anterior := ordenConfirmadaPuertoCobro(t)
	versionAnterior, huellaAnterior, err := anterior.ControlConcurrencia()
	if err != nil {
		t.Fatal(err)
	}
	instante := instantePuertoCobro.Add(3 * time.Minute)
	solicitudDominio := domain.SolicitudDevolucionOrdenCobro{
		DevolucionRef: "dev_abcdefghijkl0123456789", EvidenciaRef: "evidencia:devolucion:reserva",
		HuellaEvidenciaSHA256:  huellaPuertoCobro('4'),
		IndiceIdempotenciaHMAC: "hmac-sha256:devoluciones-v1:" + huellaPuertoCobro('5'),
		Motivo:                 "Resolucion administrativa", SolicitadaEn: instante,
	}
	posterior, _, repetida, err := anterior.SolicitarDevolucion(
		solicitudDominio,
		contextoOrdenPuertoCobro(t, anterior, domain.AccionCobroSolicitarDevolucion, instante),
	)
	if err != nil || repetida {
		t.Fatalf("preparar devolucion: repetida=%v err=%v", repetida, err)
	}
	mutacion, err := NuevaMutacionOrdenCobro(posterior)
	if err != nil {
		t.Fatal(err)
	}
	reserva := SolicitudReservaDevolucionCobro{
		OrdenRef: anterior.ID, DevolucionRef: solicitudDominio.DevolucionRef,
		IndiceIdempotenciaHMAC: solicitudDominio.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC:    "hmac-sha256:peticion-v1:" + huellaPuertoCobro('6'),
		PrincipalRef:           personaPuertoCobro, SolicitadaEn: instante, ExpiraEn: instante.Add(5 * time.Minute),
	}
	if err := reserva.Validar(); err != nil {
		t.Fatalf("reserva de devolucion valida rechazada: %v", err)
	}
	if err := (ReservaDevolucionCobro{Token: tokenReservaDevolucion}).Validar(); err != nil {
		t.Fatalf("token de devolucion valido rechazado: %v", err)
	}
	for nombre, muta := range map[string]func(*SolicitudReservaDevolucionCobro){
		"orden con sufijo":       func(s *SolicitudReservaDevolucionCobro) { s.OrdenRef += " " },
		"devolucion con prefijo": func(s *SolicitudReservaDevolucionCobro) { s.DevolucionRef = " " + s.DevolucionRef },
		"idempotencia con prefijo": func(s *SolicitudReservaDevolucionCobro) {
			s.IndiceIdempotenciaHMAC = " " + s.IndiceIdempotenciaHMAC
		},
		"peticion con sufijo":   func(s *SolicitudReservaDevolucionCobro) { s.HuellaSolicitudHMAC += " " },
		"principal con prefijo": func(s *SolicitudReservaDevolucionCobro) { s.PrincipalRef = " " + s.PrincipalRef },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := reserva
			muta(&alterada)
			if !errors.Is(alterada.Validar(), ErrReservaOrdenCobroInvalida) {
				t.Fatal("la reserva de devolucion normalizo una referencia o un dominio")
			}
		})
	}
	for _, token := range []string{" " + tokenReservaDevolucion, tokenReservaDevolucion + " ", tokenReservaCobro} {
		if !errors.Is((ReservaDevolucionCobro{Token: token}).Validar(), ErrReservaOrdenCobroInvalida) {
			t.Fatalf("se acepto token de reserva de devolucion no exacto: %q", token)
		}
	}
	confirmacion := SolicitudConfirmarReservaDevolucionCobro{
		Token: tokenReservaDevolucion, HuellaSolicitudHMAC: reserva.HuellaSolicitudHMAC,
		VersionEsperada: versionAnterior, HuellaEsperadaSHA256: huellaAnterior, Mutacion: mutacion,
	}
	if err := confirmacion.Validar(); err != nil {
		t.Fatalf("confirmacion reservada valida rechazada: %v", err)
	}
	if err := (SolicitudConfirmarTransicionOrdenCobro{
		VersionEsperada: versionAnterior, HuellaEsperadaSHA256: huellaAnterior, Mutacion: mutacion,
	}).Validar(); !errors.Is(err, ErrMutacionOrdenCobroInvalida) {
		t.Fatal("la devolucion eludio su reserva usando la confirmacion generica")
	}
	if err := (SolicitudAbandonarReservaDevolucionCobro{
		Token: tokenReservaDevolucion, OrdenRef: anterior.ID, DevolucionRef: solicitudDominio.DevolucionRef,
		PrincipalRef: personaPuertoCobro, HuellaSolicitudHMAC: reserva.HuellaSolicitudHMAC,
	}).Validar(); err != nil {
		t.Fatalf("abandono tipado valido rechazado: %v", err)
	}
	reserva.IndiceIdempotenciaHMAC = "hmac-sha256:pagos-v1:" + huellaPuertoCobro('5')
	if !errors.Is(reserva.Validar(), ErrReservaOrdenCobroInvalida) {
		t.Fatal("se acepto una HMAC de alta como idempotencia de devolucion")
	}
	confirmacion.HuellaSolicitudHMAC = "hmac-sha256:otro-dominio:" + huellaPuertoCobro('6')
	if !errors.Is(confirmacion.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("se acepto una HMAC de otro dominio al confirmar")
	}
}

func TestConfirmacionTransicionLigaOCCAnteriorConAuditoria(t *testing.T) {
	anterior := nuevaOrdenPuertoCobro(t)
	versionAnterior, huellaAnterior, err := anterior.ControlConcurrencia()
	if err != nil {
		t.Fatal(err)
	}
	instante := instantePuertoCobro.Add(time.Minute)
	evidencia, err := domain.NuevaEvidenciaInicioOperacionCobroVerificada(
		datosEvidenciaPuertoCobro(anterior, "transicion-occ", '7', instante),
	)
	if err != nil {
		t.Fatal(err)
	}
	posterior, repetida, err := anterior.RegistrarEnvio(
		evidencia, instante, contextoOrdenPuertoCobro(t, anterior, domain.AccionCobroIniciarOperacion, instante), "Inicio alojado",
	)
	if err != nil || repetida {
		t.Fatalf("preparar transicion: repetida=%v err=%v", repetida, err)
	}
	mutacion, err := NuevaMutacionOrdenCobro(posterior)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := SolicitudConfirmarTransicionOrdenCobro{
		VersionEsperada: versionAnterior, HuellaEsperadaSHA256: huellaAnterior, Mutacion: mutacion,
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("confirmacion OCC valida: %v", err)
	}
	solicitud.HuellaEsperadaSHA256 = huellaPuertoCobro('f')
	if !errors.Is(solicitud.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("se acepto OCC con otra huella anterior")
	}
}

func TestMetadatosAuditoriaSonAcotados(t *testing.T) {
	mutacion := mutacionCreacionPuertoCobro(t)
	mutacion.datos.auditoria.Metadatos = MetadatosAuditoriaCobro{Canal: CanalAuditoriaCobro("canal_no_enumerado")}
	if !errors.Is(mutacion.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("se aceptaron metadatos no enumerados")
	}
	mutacion = mutacionCreacionPuertoCobro(t)
	mutacion.datos.evento.Atributos = AtributosEventoSalidaCobro{}
	if !errors.Is(mutacion.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("se acepto un outbox sin atributos derivados")
	}
}

func TestMatrizAuditoriaLigaProcedenciaAlTipoYAccion(t *testing.T) {
	mutacionLocal := mutacionCreacionPuertoCobro(t)
	datosLocal, err := mutacionLocal.Datos()
	if err != nil {
		t.Fatal(err)
	}
	mutacionRemota, err := NuevaMutacionOrdenCobro(ordenEnviadaPuertoCobro(t))
	if err != nil {
		t.Fatal(err)
	}
	datosRemotos, err := mutacionRemota.Datos()
	if err != nil {
		t.Fatal(err)
	}

	localFalsificada := datosLocal.Auditoria
	localFalsificada.Hecho = domain.HechoCobroIncidenciaDetectada
	localFalsificada.Accion = domain.AccionCobroCancelar
	localFalsificada.VerificacionEvidenciaRef = datosRemotos.Auditoria.VerificacionEvidenciaRef
	localFalsificada.HuellaVerificacionSHA256 = datosRemotos.Auditoria.HuellaVerificacionSHA256
	localFalsificada.MetodoVerificacionEvidencia = datosRemotos.Auditoria.MetodoVerificacionEvidencia
	localFalsificada.AudienciaEvidencia = datosRemotos.Auditoria.AudienciaEvidencia
	localFalsificada.EvidenciaEmitidaEn = datosRemotos.Auditoria.EvidenciaEmitidaEn
	localFalsificada.EvidenciaRecibidaEn = datosRemotos.Auditoria.EvidenciaRecibidaEn
	localFalsificada.EvidenciaVerificadaEn = datosRemotos.Auditoria.EvidenciaVerificadaEn
	if matrizEvidenciaAuditoriaCobroValida(localFalsificada) {
		t.Fatal("una auditoria local pudo autodeclararse como remota")
	}

	remotaSinPrueba := datosRemotos.Auditoria
	remotaSinPrueba.Hecho = domain.HechoCobroIncidenciaDetectada
	remotaSinPrueba.Accion = domain.AccionCobroIniciarOperacion
	remotaSinPrueba.VerificacionEvidenciaRef = ""
	remotaSinPrueba.HuellaVerificacionSHA256 = ""
	remotaSinPrueba.MetodoVerificacionEvidencia = ""
	remotaSinPrueba.AudienciaEvidencia = ""
	remotaSinPrueba.EvidenciaEmitidaEn = time.Time{}
	remotaSinPrueba.EvidenciaRecibidaEn = time.Time{}
	remotaSinPrueba.EvidenciaVerificadaEn = time.Time{}
	if matrizEvidenciaAuditoriaCobroValida(remotaSinPrueba) {
		t.Fatal("una auditoria remota pudo omitir la prueba de verificacion")
	}
}

func TestMutacionCobroEsOpacaYEntregaCopiasDefensivas(t *testing.T) {
	mutacion := mutacionCreacionPuertoCobro(t)
	datos, err := mutacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Orden.Historial[0].Motivo = "Intento de alterar la copia"
	datos.Auditoria.ActorRef = "persona:otra"
	datos.Evento.Atributos.Hecho = domain.HechoCobroConfirmado
	if err := mutacion.Validar(); err != nil {
		t.Fatalf("una copia devuelta altero la mutacion interna: %v", err)
	}
	nuevaCopia, err := mutacion.Datos()
	if err != nil || nuevaCopia.Orden.Historial[0].Motivo == "Intento de alterar la copia" ||
		nuevaCopia.Auditoria.ActorRef == "persona:otra" || nuevaCopia.Evento.Atributos.Hecho == domain.HechoCobroConfirmado {
		t.Fatal("la mutacion compartio memoria mutable con el llamador")
	}
	if !errors.Is((MutacionOrdenCobro{}).Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("el valor cero de mutacion fue aceptado")
	}
	if _, err := json.Marshal(mutacion); !errors.Is(err, ErrMutacionOrdenCobroInvalida) ||
		strings.Contains(fmt.Sprintf("%+v", mutacion), mutacion.datos.auditoria.HuellaSesionHMAC) {
		t.Fatal("la mutacion interna se serializo o filtro evidencia forense")
	}
	tipoMetadatos := reflect.TypeOf(MetadatosAuditoriaCobro{})
	for indice := 0; indice < tipoMetadatos.NumField(); indice++ {
		if tipoMetadatos.Field(indice).Type.Kind() == reflect.Map {
			t.Fatal("los metadatos informativos recuperaron un mapa abierto")
		}
	}
}

func TestIdentificadorEventoCobroEsDeterministaYLigadoAlUltimoHecho(t *testing.T) {
	mutacion := mutacionCreacionPuertoCobro(t)
	datos, err := mutacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := NuevoEventoSalidaCobro(datos.Orden)
	if err != nil || repetido != datos.Evento {
		t.Fatalf("el mismo hecho no produjo el mismo evento: %+v, %v", repetido, err)
	}
	alterado := datos.Evento
	alterado.ID = idDeterministaEventoSalidaCobro(
		alterado.OrdenRef, alterado.VersionOrden+1, alterado.SecuenciaHecho, alterado.HuellaHechoSHA256,
		alterado.Atributos.Hecho, alterado.Atributos.Estado, alterado.Atributos.Accion,
	)
	if !errors.Is(alterado.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("un ID calculado para otra version fue aceptado")
	}
	tuplaDistinta := datos.Evento
	tuplaDistinta.Tipo = EventoCobroConfirmado
	tuplaDistinta.Atributos = AtributosEventoSalidaCobro{
		Hecho: domain.HechoCobroConfirmado, Estado: domain.EstadoCobroConfirmada,
		Accion: domain.AccionCobroProcesarResultado,
	}
	if !errors.Is(tuplaDistinta.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("el ID de un hecho se reutilizo para otra tupla cerrada")
	}
	posterior := ordenEnviadaPuertoCobro(t)
	eventoPosterior, err := NuevoEventoSalidaCobro(posterior)
	if err != nil || eventoPosterior.ID == datos.Evento.ID || eventoPosterior.SecuenciaHecho == datos.Evento.SecuenciaHecho {
		t.Fatalf("otro hecho reutilizo el identificador: %+v, %v", eventoPosterior, err)
	}
}

func TestOutboxCobroTieneMapeoCerradoYUnicoDesdeCadaHecho(t *testing.T) {
	hechos := []domain.TipoHechoCobro{
		domain.HechoCobroOrdenCreada,
		domain.HechoCobroOperacionEnviada,
		domain.HechoCobroResultadoPendiente,
		domain.HechoCobroResultadoDesconocido,
		domain.HechoCobroConfirmado,
		domain.HechoCobroRechazado,
		domain.HechoCobroCancelado,
		domain.HechoCobroCaducado,
		domain.HechoCobroConciliado,
		domain.HechoCobroDevolucionSolicitada,
		domain.HechoCobroDevolucionResultadoPendiente,
		domain.HechoCobroDevolucionResultadoDesconocido,
		domain.HechoCobroDevolucionRechazada,
		domain.HechoCobroDevuelto,
		domain.HechoCobroDevolucionConciliada,
		domain.HechoCobroIncidenciaDetectada,
		domain.HechoCobroEvidenciaAdicional,
	}
	vistos := make(map[TipoEventoSalidaCobro]domain.TipoHechoCobro, len(hechos))
	for _, hecho := range hechos {
		tipo, existe := tipoEventoSalidaCobroParaHecho(hecho)
		if !existe || !tipo.Valido() {
			t.Fatalf("hecho sin evento cerrado: %q", hecho)
		}
		if anterior, repetido := vistos[tipo]; repetido {
			t.Fatalf("los hechos %q y %q comparten tipo de evento %q", anterior, hecho, tipo)
		}
		vistos[tipo] = hecho
	}
	if tipo, existe := tipoEventoSalidaCobroParaHecho(domain.TipoHechoCobro("hecho_inventado")); existe || tipo != "" {
		t.Fatalf("un hecho no enumerado produjo evento: %q", tipo)
	}
	if TipoEventoSalidaCobro("cobro.todo").Valido() {
		t.Fatal("un tipo de evento no enumerado fue aceptado")
	}

	mutacion := mutacionCreacionPuertoCobro(t)
	datosMutacion, err := mutacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	eventoDerivado, err := NuevoEventoSalidaCobro(datosMutacion.Orden)
	if err != nil || eventoDerivado != datosMutacion.Evento {
		t.Fatalf("el constructor no derivo el evento exacto: evento=%+v err=%v", eventoDerivado, err)
	}
	eventoDerivado.Atributos.Hecho = domain.HechoCobroConfirmado
	eventoDerivado.Tipo = EventoCobroConfirmado
	if !errors.Is(eventoDerivado.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("el evento aislado acepto una tupla tipo, estado y accion incompatible")
	}
	mutacion.datos.evento = eventoDerivado
	if !errors.Is(mutacion.Validar(), ErrMutacionOrdenCobroInvalida) {
		t.Fatal("el outbox describio otro hecho y fue aceptado junto a la orden")
	}
}

func TestResultadosPuertoSoloAceptanEvidenciaVerificadaTipada(t *testing.T) {
	if !errors.Is((ResultadoOperacionCobro{}).Validar(), ErrResultadoPasarelaCobroInvalido) ||
		!errors.Is((ResultadoDevolucionCobro{}).Validar(), ErrResultadoPasarelaCobroInvalido) ||
		!errors.Is((ResultadoConciliacionCobro{}).Validar(), ErrResultadoPasarelaCobroInvalido) {
		t.Fatal("un resultado vacio fue aceptado")
	}
	orden := ordenEnviadaPuertoCobro(t)
	evidencia, err := domain.NuevaEvidenciaResultadoCobroVerificada(
		datosEvidenciaPuertoCobro(orden, "resultado", '7', instantePuertoCobro.Add(2*time.Minute)),
		domain.ResultadoOperacionCobroConfirmado,
	)
	if err != nil || (ResultadoOperacionCobro{Evidencia: evidencia}).Validar() != nil {
		t.Fatalf("resultado tipado valido rechazado: %v", err)
	}
}

func TestReferenciasConsultaSonExactas(t *testing.T) {
	referencia := ReferenciaOperacionCobro{
		ConectorID: "pasarela_corporativa", VersionConector: 2,
		OrdenRef: "cob_0123456789abcdefghijkl", OperacionProveedorRef: "operacion:opaca:uno",
		CorrelacionRef: "correlacion:uno",
	}
	if err := referencia.Validar(); err != nil {
		t.Fatalf("referencia de operacion valida: %v", err)
	}
	devolucion := ReferenciaDevolucionCobro{
		ConectorID: referencia.ConectorID, VersionConector: referencia.VersionConector,
		OrdenRef: referencia.OrdenRef, DevolucionRef: "dev_abcdefghijkl0123456789",
		OperacionProveedorRef: referencia.OperacionProveedorRef, CorrelacionRef: referencia.CorrelacionRef,
	}
	if err := devolucion.Validar(); err != nil {
		t.Fatalf("referencia de devolucion valida: %v", err)
	}
	devolucion.DevolucionRef = "dev_corta"
	if !errors.Is(devolucion.Validar(), ErrReferenciaOperacionCobroInvalida) {
		t.Fatal("una devolucion no opaca fue aceptada")
	}
}

func TestFabricasDeEvidenciaSoloSeInvocanEnAdaptadoresDePasarela(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	raiz := filepath.Clean(filepath.Join(directorio, "../../.."))
	permitido := filepath.ToSlash(filepath.Join("internal", "vec", "adapters", "pagos")) + "/"
	err = filepath.WalkDir(raiz, func(ruta string, entrada os.DirEntry, errorRecorrido error) error {
		if errorRecorrido != nil {
			return errorRecorrido
		}
		if entrada.IsDir() || filepath.Ext(ruta) != ".go" || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		relativa, errorRelativa := filepath.Rel(raiz, ruta)
		if errorRelativa != nil {
			return errorRelativa
		}
		relativa = filepath.ToSlash(relativa)
		if relativa == "internal/vec/domain/pagos.go" || strings.HasPrefix(relativa, permitido) {
			return nil
		}
		archivo, errorParseo := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
		if errorParseo != nil {
			return errorParseo
		}
		ast.Inspect(archivo, func(nodo ast.Node) bool {
			llamada, esLlamada := nodo.(*ast.CallExpr)
			if !esLlamada {
				return true
			}
			nombre := ""
			switch funcion := llamada.Fun.(type) {
			case *ast.Ident:
				nombre = funcion.Name
			case *ast.SelectorExpr:
				nombre = funcion.Sel.Name
			}
			fabricasReservadas := map[string]struct{}{
				"NuevaEvidenciaInicioOperacionCobroVerificada":     {},
				"NuevaEvidenciaResultadoCobroVerificada":           {},
				"NuevaEvidenciaResultadoDevolucionCobroVerificada": {},
				"NuevaEvidenciaConciliacionCobroVerificada":        {},
			}
			if _, reservada := fabricasReservadas[nombre]; reservada {
				t.Errorf("%s invoca una fabrica reservada al adaptador verificador", relativa)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("recorrer arquitectura: %v", err)
	}
}
