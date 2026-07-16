package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteBaremacionPrueba = time.Date(2026, time.July, 15, 10, 0, 0, 0, time.UTC)

func TestServicioBaremacionCompletaRevisionConAutorizacionesIndependientes(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	ctx := context.Background()

	iniciada, err := entorno.servicio.IniciarRevision(ctx, ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatalf("IniciarRevision() error = %v", err)
	}
	adoptada, err := entorno.servicio.AdoptarDecision(ctx, ordenAdoptarBaremacionPrueba(iniciada, entorno.calculo.resultado.Calculo))
	if err != nil {
		t.Fatalf("AdoptarDecision() error = %v", err)
	}
	codificada, err := entorno.servicio.CodificarDecision(ctx, OrdenCodificarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Revision: adoptada,
		PoliticaFirmaRef:     entorno.politicas.politica.Referencia,
		PoliticaFirmaVersion: entorno.politicas.politica.Version,
		HuellaPoliticaSHA256: entorno.politicas.politica.HuellaSHA256,
	})
	if err != nil {
		t.Fatalf("CodificarDecision() error = %v", err)
	}
	custodiada, err := entorno.servicio.CustodiarDecision(ctx, OrdenCustodiarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Decision: codificada, OperacionRef: "operacion:decision:1",
		ClaveIdempotencia: "idempotencia:custodia:1", CargaRef: "carga:decision:1",
	})
	if err != nil {
		t.Fatalf("CustodiarDecision() error = %v", err)
	}
	preparada, err := entorno.servicio.PrepararFirma(ctx, OrdenPrepararFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Decision: custodiada, OperacionRef: "operacion:firma:1",
		ClaveIdempotencia: "idempotencia:firma:1",
	})
	if err != nil {
		t.Fatalf("PrepararFirma() error = %v", err)
	}
	finalizada, err := entorno.servicio.FinalizarFirma(ctx, OrdenFinalizarFirmaBaremacion{
		Actor: actorBaremacionPrueba(), Firma: preparada, OperacionRef: "operacion:finalizar:1",
		OperacionCustodiaRef: "operacion:custodia:firmado:1", ClaveIdempotenciaCustodia: "idempotencia:custodia:firmado:1",
		CargaDocumentoFirmadoRef: "carga:documento:firmado:1",
		ClaveIdempotenciaReserva: "idempotencia:reserva:confirmacion:1",
		MotivoClaveConfirmacion:  "decision_tecnica_firmada",
		MotivoConfirmacion:       "Incorporacion de la decision tecnica validada y firmada.",
	})
	if err != nil {
		t.Fatalf("FinalizarFirma() error = %v", err)
	}
	if finalizada.Decision.Validar() != nil || finalizada.Confirmacion.Validar() != nil ||
		finalizada.Confirmacion.Version.Referencia.Numero != 2 ||
		len(finalizada.Confirmacion.Version.Agregado.Decisiones) != 1 {
		t.Fatalf("resultado final no valido: %+v", finalizada.Confirmacion.Version.Referencia)
	}

	esperadas := []puertosbolsa.AccionOperacionBaremacion{
		puertosbolsa.AccionConsultarBaremacionVigente,
		puertosbolsa.AccionRecuperarCalculoBaremacion,
		puertosbolsa.AccionConsultarCriterioBaremacion,
		puertosbolsa.AccionConsultarEvidenciaBaremacion,
		puertosbolsa.AccionConsultarRepresentacionBaremacion,
		puertosbolsa.AccionAdoptarDecisionInicialBaremacion,
		puertosbolsa.AccionConsultarPoliticaFirmaBaremacion,
		puertosbolsa.AccionCodificarDecisionBaremacion,
		puertosbolsa.AccionCustodiarDecisionBaremacion,
		puertosbolsa.AccionPrepararFirmaDecisionBaremacion,
		puertosbolsa.AccionConsultarFirmaDecisionBaremacion,
		puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion,
		puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion,
		puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion,
		puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
		puertosbolsa.AccionConfirmarDecisionBaremacion,
	}
	if acciones := entorno.autorizador.acciones(); !accionesBaremacionIguales(acciones, esperadas) {
		t.Fatalf("acciones de autorizacion = %v; se esperaban %v", acciones, esperadas)
	}
	if entorno.autorizador.referenciasRepetidas() {
		t.Fatal("se reutilizo una decision de autorizacion entre operaciones")
	}
	capturada := entorno.repositorio.confirmacion
	if capturada == nil || capturada.Manifiesto == nil ||
		capturada.ContextoPrevalidacionArchivo.ValidarPara(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
			puertosbolsa.ClaseRecursoBaremacion, capturada.Agregado.ID,
		) != nil || !capturada.Contexto.MismoVinculoAutenticacionQue(
		capturada.ContextoPrevalidacionArchivo,
	) {
		t.Fatal("confirmacion no transporto la prevalidacion dedicada y ligada")
	}
	proyeccionConfirmacion := capturada.Contexto.Proyeccion()
	proyeccionPrevalidacion := capturada.ContextoPrevalidacionArchivo.Proyeccion()
	autorizacionesManifiesto := capturada.Manifiesto.Autorizaciones
	if proyeccionConfirmacion.AutorizacionRef == proyeccionPrevalidacion.AutorizacionRef ||
		len(autorizacionesManifiesto) < 2 ||
		autorizacionesManifiesto[len(autorizacionesManifiesto)-2].Accion !=
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion ||
		autorizacionesManifiesto[len(autorizacionesManifiesto)-2].AutorizacionRef !=
			proyeccionPrevalidacion.AutorizacionRef ||
		autorizacionesManifiesto[len(autorizacionesManifiesto)-1].Accion !=
			puertosbolsa.AccionConfirmarDecisionBaremacion ||
		autorizacionesManifiesto[len(autorizacionesManifiesto)-1].AutorizacionRef !=
			proyeccionConfirmacion.AutorizacionRef {
		t.Fatal("manifiesto no conserva prevalidacion y confirmacion distintas en orden canonico")
	}
	contextoFirmable, err := entorno.almacen.firmable.Contexto.Proyeccion()
	if err != nil {
		t.Fatalf("contexto opaco de custodia no proyectable: %v", err)
	}
	if contextoFirmable.SujetoSeudonimoHMAC == iniciada.sujetoRef ||
		strings.Contains(contextoFirmable.SujetoSeudonimoHMAC, iniciada.sujetoRef) {
		t.Fatal("el almacen recibio el identificador interno del sujeto")
	}
	if contextoFirmable.AccionTecnica != puertosvec.AccionAlmacenEscribir ||
		contextoFirmable.ModuloID != "bolsa" || contextoFirmable.RecursoRef != adoptada.contenido.ID {
		t.Fatalf("contexto de custodia no ligado: %+v", contextoFirmable)
	}
	if entorno.almacen.escrituras != 2 || entorno.almacen.retenciones != 1 ||
		finalizada.DocumentoFirmado.Objeto.RetenidoHasta.IsZero() {
		t.Fatalf("documento final no custodiado y retenido: escrituras=%d retenciones=%d", entorno.almacen.escrituras, entorno.almacen.retenciones)
	}
	if finalizada.Decision.Firma.ManifiestoProbatorioRef == "" ||
		finalizada.Decision.Firma.HuellaManifiestoProbatorioSHA256 == "" ||
		finalizada.Decision.Firma.SelloManifiestoProbatorioHMACSHA256 == "" {
		t.Fatalf("decision sin manifiesto probatorio sellado: %+v", finalizada.Decision.Firma)
	}
}

func TestEstadosIntermediosBaremacionNoSonDTOTransportables(t *testing.T) {
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(SesionAutenticadaBaremacion{}),
		reflect.TypeOf(RevisionBaremacionIniciada{}), reflect.TypeOf(RevisionBaremacionAdoptada{}),
		reflect.TypeOf(DecisionBaremacionCodificada{}), reflect.TypeOf(DecisionBaremacionCustodiada{}),
		reflect.TypeOf(FirmaBaremacionPreparada{}),
	} {
		for indice := 0; indice < tipo.NumField(); indice++ {
			if tipo.Field(indice).IsExported() {
				t.Fatalf("%s expone estado fabricable: %s", tipo.Name(), tipo.Field(indice).Name)
			}
		}
	}
	for _, caso := range []struct {
		valor    any
		etiqueta string
	}{
		{SesionAutenticadaBaremacion{}, "[SESION-BAREMACION-OPACA]"},
		{RevisionBaremacionIniciada{}, "[REVISION-BAREMACION-INICIADA-OPACA]"},
		{RevisionBaremacionAdoptada{}, "[REVISION-BAREMACION-ADOPTADA-OPACA]"},
		{DecisionBaremacionCodificada{}, "[DECISION-BAREMACION-CODIFICADA-OPACA]"},
		{DecisionBaremacionCustodiada{}, "[DECISION-BAREMACION-CUSTODIADA-OPACA]"},
		{FirmaBaremacionPreparada{}, "[FIRMA-BAREMACION-PREPARADA-OPACA]"},
	} {
		if obtenido := fmt.Sprintf("%v %+v %#v %s", caso.valor, caso.valor, caso.valor, caso.valor); obtenido != strings.Repeat(caso.etiqueta+" ", 3)+caso.etiqueta {
			t.Errorf("el formato de %T filtró estado: %s", caso.valor, obtenido)
		}
		if obtenido := slog.AnyValue(caso.valor).Resolve().String(); obtenido != caso.etiqueta {
			t.Errorf("slog de %T filtró estado: %s", caso.valor, obtenido)
		}
	}
	entorno := nuevoEntornoBaremacionPrueba(t)
	sesion := entorno.sesiones.sesiones[0]
	if serializada := fmt.Sprintf("%v %+v %#v", sesion, sesion, sesion); serializada !=
		"[SESION-BAREMACION-OPACA] [SESION-BAREMACION-OPACA] [SESION-BAREMACION-OPACA]" {
		t.Fatalf("el formato de la sesion filtro identidad: %s", serializada)
	}
	if _, err := json.Marshal(sesion); !errors.Is(err, ErrOrdenBaremacionInvalida) {
		t.Fatalf("la sesion opaca se serializo: %v", err)
	}
	iniciada, err := entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(iniciada); err == nil {
		t.Fatal("la reserva opaca se pudo serializar como DTO")
	}
	adoptada, err := entorno.servicio.AdoptarDecision(
		context.Background(), ordenAdoptarBaremacionPrueba(iniciada, entorno.calculo.resultado.Calculo),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(adoptada); err == nil {
		t.Fatal("la autorizacion opaca de adopcion se pudo serializar como DTO")
	}
}

func TestConstructorBaremacionRechazaDependenciaNulaTipada(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	var repositorioNulo *repositorioBaremacionPrueba
	_, err := NuevoServicioBaremacion(
		repositorioNulo, entorno.servicio.fuenteDatos, entorno.servicio.calculador,
		entorno.servicio.catalogoFirma, entorno.servicio.codificador, entorno.servicio.almacen,
		entorno.servicio.firmador, entorno.servicio.recuperadorBinario, entorno.servicio.validadorFirma,
		entorno.servicio.selladorTiempo, entorno.servicio.aumentadorFirma, entorno.servicio.selladorSolicitud,
		entorno.servicio.seudonimizador, entorno.servicio.generador, entorno.servicio.autorizador,
		entorno.servicio.sesiones, entorno.servicio.reloj,
		OpcionesServicioBaremacion{
			DuracionReserva: entorno.servicio.duracionReserva, DuracionFirma: entorno.servicio.duracionFirma,
			ClasificacionDocumental:  entorno.servicio.clasificacion,
			ConectorAlmacenPermitido: entorno.servicio.conectorAlmacen,
			PoliticaRetencionRef:     entorno.servicio.politicaRetencion,
			DuracionRetencion:        entorno.servicio.duracionRetencion,
		},
	)
	if !errors.Is(err, ErrDependenciaBaremacionRequerida) {
		t.Fatalf("dependencia nula tipada admitida: %v", err)
	}
}

func TestServicioBaremacionDeniegaSesionNoVerificadaAntesDeEfectos(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	entorno.sesiones.sesiones = nil

	_, err := entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("IniciarRevision() error = %v", err)
	}
	if len(entorno.autorizador.acciones()) != 0 || entorno.repositorio.consultas != 0 || entorno.repositorio.reservas != 0 {
		t.Fatalf("hubo efectos con sesion no verificada: autorizaciones=%d consultas=%d reservas=%d",
			len(entorno.autorizador.acciones()), entorno.repositorio.consultas, entorno.repositorio.reservas)
	}
}

func TestServicioBaremacionDeniegaSesionAmbiguaAntesDeAutorizar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	entorno.sesiones.sesiones = append(entorno.sesiones.sesiones, entorno.sesiones.sesiones[0])

	_, err := entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("IniciarRevision() error = %v", err)
	}
	if len(entorno.autorizador.acciones()) != 0 || entorno.repositorio.consultas != 0 {
		t.Fatalf("la fuente ambigua produjo efectos: autorizaciones=%d consultas=%d",
			len(entorno.autorizador.acciones()), entorno.repositorio.consultas)
	}
}

func TestServicioBaremacionDeniegaContextoNuloOCanceladoAntesDeEfectos(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
	}{
		{nombre: "nulo", ctx: nil},
		{nombre: "cancelado", ctx: contextoCanceladoBaremacionPrueba()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoBaremacionPrueba(t)
			_, err := entorno.servicio.IniciarRevision(caso.ctx, ordenIniciarBaremacionPrueba())
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
				t.Fatalf("IniciarRevision() error = %v", err)
			}
			if len(entorno.autorizador.acciones()) != 0 || entorno.repositorio.consultas != 0 ||
				entorno.repositorio.reservas != 0 || entorno.repositorio.confirmaciones != 0 {
				t.Fatalf(
					"hubo efectos con contexto %s: autorizaciones=%d consultas=%d reservas=%d confirmaciones=%d",
					caso.nombre, len(entorno.autorizador.acciones()), entorno.repositorio.consultas,
					entorno.repositorio.reservas, entorno.repositorio.confirmaciones,
				)
			}
		})
	}
}

func contextoCanceladoBaremacionPrueba() context.Context {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	return ctx
}

func TestOrdenBaremacionNoRepiteIdentidadNiAtributosPersonales(t *testing.T) {
	tipo := reflect.TypeOf(ActorBaremacion{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "Motivo" {
		t.Fatalf("ActorBaremacion expone datos ajenos a la motivacion: %v", tipo)
	}
	serializada, err := json.Marshal(ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatalf("serializar orden de entrada: %v", err)
	}
	for _, campoProhibido := range []string{
		"display_name", "email", "roles", "permissions", "attributes",
		"auth_method", "auth_assurance", "perfil_activo", "principal",
	} {
		if bytes.Contains(serializada, []byte(campoProhibido)) {
			t.Fatalf("la orden repite %q: %s", campoProhibido, serializada)
		}
	}
}

func TestServicioBaremacionDeniegaDecisionLigadaAOtraSesionV1(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	contextoActor, vinculoBase := contextoYVinculoAutenticacionAplicacionPrueba(instanteBaremacionPrueba)
	datos, err := vinculoBase.Datos()
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := datos.Autenticacion()
	autenticacion.AutenticacionRef = "aut_otra_autenticacion_abcdefghijkl"
	autenticacion.AsercionRef = "ase_otra_asercion_abcdefghijkl"
	autenticacion.SesionRef = "ses_otra_sesion_abcdefghijkl"
	autenticacion.ControlSesionRef = "cse_otro_control_abcdefghijkl"
	vinculoAjeno, err := dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		contextoActor,
		instanteBaremacionPrueba,
	)
	if err != nil {
		t.Fatalf("crear vinculo alternativo: %v", err)
	}
	entorno.autorizador.vinculoDecision = vinculoAjeno

	_, err = entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		!errors.Is(err, dominiovec.ErrDecisionAutorizacionInvalida) {
		t.Fatalf("decision de otra sesion no denegada: %v", err)
	}
	if entorno.repositorio.consultas != 0 {
		t.Fatal("la decision cruzada alcanzo el repositorio")
	}
}

func TestServicioBaremacionNoTransfiereRevisionAOtraIdentidad(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	revision, err := entorno.servicio.IniciarRevision(context.Background(), ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatalf("iniciar revision: %v", err)
	}
	entorno.sesiones.sesiones = []SesionAutenticadaBaremacion{
		sesionAutenticadaBaremacionIdentidadPrueba(
			t,
			"per_otra_persona_abcdefghijkl",
			"prf_otro_perfil_abcdefghijkl",
		),
	}
	_, err = entorno.servicio.AdoptarDecision(
		context.Background(),
		ordenAdoptarBaremacionPrueba(revision, entorno.calculo.resultado.Calculo),
	)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("revision transferida a otra identidad: %v", err)
	}
	if entorno.calculo.recuperaciones != 0 {
		t.Fatalf("la identidad distinta alcanzo el calculador: %d", entorno.calculo.recuperaciones)
	}
}

func TestCustodiarDecisionRevalidaSesionAntesDeSeudonimizarOEscribir(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	ctx := context.Background()
	revision, err := entorno.servicio.IniciarRevision(ctx, ordenIniciarBaremacionPrueba())
	if err != nil {
		t.Fatal(err)
	}
	adoptada, err := entorno.servicio.AdoptarDecision(
		ctx,
		ordenAdoptarBaremacionPrueba(revision, entorno.calculo.resultado.Calculo),
	)
	if err != nil {
		t.Fatal(err)
	}
	codificada, err := entorno.servicio.CodificarDecision(ctx, OrdenCodificarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Revision: adoptada,
		PoliticaFirmaRef:     entorno.politicas.politica.Referencia,
		PoliticaFirmaVersion: entorno.politicas.politica.Version,
		HuellaPoliticaSHA256: entorno.politicas.politica.HuellaSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	entorno.sesiones.sesiones = nil
	seudonimizador := &seudonimizadorContadorBaremacionPrueba{}
	entorno.servicio.seudonimizador = seudonimizador
	escriturasPrevias := entorno.almacen.escrituras

	_, err = entorno.servicio.CustodiarDecision(ctx, OrdenCustodiarDecisionBaremacion{
		Actor: actorBaremacionPrueba(), Decision: codificada,
		OperacionRef:      "operacion:decision:sesion-revocada",
		ClaveIdempotencia: "idempotencia:custodia:sesion-revocada",
		CargaRef:          "carga:decision:sesion-revocada",
	})
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("sesion revocada admitida: %v", err)
	}
	if seudonimizador.llamadas != 0 || entorno.almacen.escrituras != escriturasPrevias {
		t.Fatalf(
			"hubo efectos antes de revalidar: seudonimizaciones=%d escrituras=%d/%d",
			seudonimizador.llamadas, entorno.almacen.escrituras, escriturasPrevias,
		)
	}
}
