package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func nuevoPreparadorSolicitudLigadaV3Prueba(
	t *testing.T,
	e *entornoAutorizacionSolicitudV3Prueba,
) *ServicioPreparacionSolicitudLigadaV3 {
	t.Helper()
	preparador, err := NuevoServicioPreparacionSolicitudLigadaV3(
		e.fuente, e.denegaciones, e.motivos,
		&relojAutorizacionServicioPrueba{ahora: e.ahora},
		&generadorAutorizacionServicioPrueba{
			referencia: "dec_preparada_0123456789abcdef01234567",
		},
		ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear preparador V3: %v", err)
	}
	return preparador
}

func TestPreparacionSolicitudLigadaV3PositivaNoRegistraYQuedaLigada(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)

	decision, orden, err := preparador.PrepararSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if err != nil {
		t.Fatalf("preparar solicitud: %v", err)
	}
	concedida, codigo, errDecision := decision.Resultado()
	datosOrden, errOrden := orden.Datos()
	if errDecision != nil || errOrden != nil || !concedida || codigo != "concedida" {
		t.Fatalf(
			"preparacion incoherente: concedida=%t codigo=%q decision=%v orden=%v",
			concedida, codigo, errDecision, errOrden,
		)
	}
	if e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf(
			"la preparacion positiva escribio: concesiones=%d denegaciones=%d",
			e.concesiones.invocaciones, e.denegaciones.invocaciones,
		)
	}
	datosSolicitud, errSolicitud := e.solicitud.Datos()
	_, errSolicitudOrden := datosOrden.Solicitud.Datos()
	if errSolicitud != nil || datosOrden.Decision.ValidarPara(e.solicitud) != nil ||
		errSolicitudOrden != nil ||
		datosOrden.ReferenciaMotivo != datosSolicitud.ReferenciaMotivo ||
		datosOrden.ResultadoContexto.Validar() != nil ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(datosOrden.ResultadoContexto) != nil {
		t.Fatalf("orden candidata no ligada exactamente: solicitud=%v orden=%v", errSolicitud, errOrden)
	}
	if decision.VigenteEn(e.ahora) {
		t.Fatal("la decision preparada se convirtio en capacidad ejecutable")
	}
}

func TestPreparacionSolicitudLigadaV3DenegadaSoloRegistraDenegacion(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	e.instantanea.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	e.fuente.instantanea = e.instantanea
	preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)

	decision, orden, err := preparador.PrepararSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	concedida, codigo, errDecision := decision.Resultado()
	_, errOrden := orden.Datos()
	if errDecision != nil || concedida || codigo != "accion_no_concedida" ||
		!errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(errOrden, ports.ErrOrdenRegistroAutorizacionLigadaV3Invalida) ||
		e.denegaciones.invocaciones != 1 || e.concesiones.invocaciones != 0 {
		t.Fatalf(
			"denegacion incoherente: concedida=%t codigo=%q decision=%v orden=%v error=%v",
			concedida, codigo, errDecision, errOrden, err,
		)
	}
}

func TestPreparacionSolicitudLigadaV3NoRetieneMutacionesDeFuentes(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)

	decision, orden, err := preparador.PrepararSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	e.fuente.instantanea.VersionRol.Concesiones[0].Accion = "adulterada"
	e.fuente.instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "otra_unidad"
	e.resultado.RepresentacionCanonica[0] ^= 1
	e.resultado.ManifiestoProcedenciaCanonico[0] ^= 1

	datosOrden, err := orden.Datos()
	concedida, _, errDecision := decision.Resultado()
	if err != nil || errDecision != nil || !concedida ||
		datosOrden.Decision.ValidarPara(datosOrden.Solicitud) != nil ||
		datosOrden.ResultadoContexto.Validar() != nil {
		t.Fatalf("mutacion externa alcanzo la preparacion: orden=%v decision=%v", err, errDecision)
	}
}

func TestPreparacionSolicitudLigadaV3FallaCerradoEnErroresYCancelacion(t *testing.T) {
	t.Run("fuente", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		secreto := "postgres://usuario:clave@interno persona=12345678Z"
		fallo := errors.New(secreto)
		e.fuente.err = fallo
		preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)
		decision, orden, err := preparador.PrepararSolicitudLigadaV3(
			context.Background(), e.solicitud, e.resultado,
		)
		if decision.Validar() == nil || ordenValidaSolicitudLigadaV3(orden) ||
			!errors.Is(err, domain.ErrAutorizacionDenegada) ||
			!errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) ||
			!errors.Is(err, fallo) || strings.Contains(err.Error(), secreto) ||
			e.denegaciones.invocaciones != 0 {
			t.Fatalf("fallo de fuente abierto o filtrado: %v", err)
		}
	})
	t.Run("cancelacion previa", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		_, orden, err := preparador.PrepararSolicitudLigadaV3(ctx, e.solicitud, e.resultado)
		if !errors.Is(err, context.Canceled) || ordenValidaSolicitudLigadaV3(orden) ||
			e.fuente.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
			t.Fatalf("cancelacion previa ignorada: %v", err)
		}
	})
	t.Run("cancelacion tras fuente", func(t *testing.T) {
		e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		e.fuente.despues = cancelar
		preparador := nuevoPreparadorSolicitudLigadaV3Prueba(t, e)
		_, orden, err := preparador.PrepararSolicitudLigadaV3(ctx, e.solicitud, e.resultado)
		if !errors.Is(err, context.Canceled) || ordenValidaSolicitudLigadaV3(orden) ||
			e.denegaciones.invocaciones != 0 {
			t.Fatalf("cancelacion tras fuente ignorada: %v", err)
		}
	})
}

func TestNuevoPreparadorSolicitudLigadaV3RechazaNulosTipadosYConfiguracion(t *testing.T) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	var fuente *fuenteAutorizacionServicioPrueba
	var denegaciones *registroDenegacionesAutorizacionV3Prueba
	var motivos *validadorMotivoAutorizacionV2Prueba
	var reloj *relojAutorizacionServicioPrueba
	var generador *generadorAutorizacionServicioPrueba
	casos := []struct {
		fuente       ports.FuenteAutorizacion
		denegaciones ports.RegistroDenegacionesAutorizacionLigadaV3
		motivos      ports.ValidadorReferenciaMotivoAutorizacionV2
		reloj        ports.Reloj
		generador    ports.GeneradorReferenciaDecisionAutorizacion
		config       ConfiguracionServicioAutorizacion
	}{
		{fuente, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}, ConfiguracionServicioAutorizacion{}},
		{e.fuente, denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}, ConfiguracionServicioAutorizacion{}},
		{e.fuente, e.denegaciones, motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}, ConfiguracionServicioAutorizacion{}},
		{e.fuente, e.denegaciones, e.motivos, reloj, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}, ConfiguracionServicioAutorizacion{}},
		{e.fuente, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, generador, ConfiguracionServicioAutorizacion{}},
		{e.fuente, e.denegaciones, e.motivos, &relojAutorizacionServicioPrueba{ahora: e.ahora}, &generadorAutorizacionServicioPrueba{referencia: "dec_ok"}, ConfiguracionServicioAutorizacion{VigenciaDecision: time.Nanosecond}},
	}
	for indice, caso := range casos {
		preparador, err := NuevoServicioPreparacionSolicitudLigadaV3(
			caso.fuente, caso.denegaciones, caso.motivos, caso.reloj,
			caso.generador, caso.config,
		)
		if preparador != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
			t.Fatalf("caso inseguro %d aceptado: preparador=%v error=%v", indice, preparador, err)
		}
	}

	var preparador *ServicioPreparacionSolicitudLigadaV3
	_, orden, err := preparador.PrepararSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado,
	)
	if !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) ||
		ordenValidaSolicitudLigadaV3(orden) {
		t.Fatalf("receptor nil aceptado: %v", err)
	}
}

func ordenValidaSolicitudLigadaV3(
	orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) bool {
	_, err := orden.Datos()
	return err == nil
}
