package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type generadorDecisionOperacionV3Prueba struct {
	referencia   string
	err          error
	invocaciones atomic.Int64
	despues      func()
}

func (g *generadorDecisionOperacionV3Prueba) NuevaReferenciaDecisionAutorizacion() (
	string,
	error,
) {
	g.invocaciones.Add(1)
	if g.despues != nil {
		g.despues()
	}
	return g.referencia, g.err
}

type fuenteAutorizacionCompuestaConcurrentePrueba struct {
	instantanea domain.InstantaneaAutorizacion
}

func (f fuenteAutorizacionCompuestaConcurrentePrueba) ObtenerInstantaneaAutorizacion(
	context.Context,
	string,
	string,
) (domain.InstantaneaAutorizacion, error) {
	return f.instantanea, nil
}

type validadorMotivoCompuestoConcurrentePrueba struct {
	referencia domain.ReferenciaEntradaCatalogo
}

func (v validadorMotivoCompuestoConcurrentePrueba) ValidarReferenciaMotivoAutorizacionV2(
	_ context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	_ time.Time,
) error {
	if referencia != v.referencia {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

type relojAutorizacionCompuestaConcurrentePrueba struct{ ahora time.Time }

func (r relojAutorizacionCompuestaConcurrentePrueba) Ahora() time.Time { return r.ahora }

type registroDenegacionesCompuestoConcurrentePrueba struct {
	invocaciones atomic.Int64
}

func (r *registroDenegacionesCompuestoConcurrentePrueba) RegistrarDenegacionAutorizacionLigadaV3(
	context.Context,
	ports.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	r.invocaciones.Add(1)
	return nil
}

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

func TestPreparacionRegistroCompuestoSolicitudLigadaV3NoEscribeNingunResultado(
	t *testing.T,
) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	generadorConcesion := &generadorDecisionOperacionV3Prueba{
		referencia: "dec_compuesta_concedida_0123456789abcdef",
	}
	decision, candidata, err := e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorConcesion,
	)
	concedida, ordenConcesion, ordenDenegacion, errCandidata := candidata.Resultado()
	if err != nil || errCandidata != nil || !concedida ||
		!ordenValidaSolicitudLigadaV3(ordenConcesion) ||
		ordenDenegacionValidaSolicitudLigadaV3(ordenDenegacion) ||
		referenciaDecisionAutorizacionV3Prueba(t, decision) != generadorConcesion.referencia ||
		generadorConcesion.invocaciones.Load() != 1 {
		t.Fatalf(
			"concesion compuesta incoherente: error=%v candidata=%v llamadas=%d",
			err, errCandidata, generadorConcesion.invocaciones.Load(),
		)
	}
	if e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf(
			"preparacion compuesta positiva escribio: concesiones=%d denegaciones=%d",
			e.concesiones.invocaciones, e.denegaciones.invocaciones,
		)
	}

	instantaneaDenegada := e.instantanea
	instantaneaDenegada.VersionRol.Concesiones = append(
		[]domain.ConcesionRol(nil), e.instantanea.VersionRol.Concesiones...,
	)
	instantaneaDenegada.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	e.fuente.instantanea = instantaneaDenegada
	generadorDenegacion := &generadorDecisionOperacionV3Prueba{
		referencia: "dec_compuesta_denegada_0123456789abcdef",
	}
	decision, candidata, err = e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorDenegacion,
	)
	concedida, ordenConcesion, ordenDenegacion, errCandidata = candidata.Resultado()
	resultadoDenegado, codigo, errDecision := decision.Resultado()
	if err != nil || errCandidata != nil || errDecision != nil ||
		concedida || resultadoDenegado || codigo != "accion_no_concedida" ||
		ordenValidaSolicitudLigadaV3(ordenConcesion) ||
		!ordenDenegacionValidaSolicitudLigadaV3(ordenDenegacion) ||
		referenciaDecisionAutorizacionV3Prueba(t, decision) != generadorDenegacion.referencia ||
		generadorDenegacion.invocaciones.Load() != 1 {
		t.Fatalf(
			"denegacion compuesta incoherente: error=%v candidata=%v decision=%v llamadas=%d",
			err, errCandidata, errDecision, generadorDenegacion.invocaciones.Load(),
		)
	}
	if e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf(
			"preparacion compuesta negativa escribio: concesiones=%d denegaciones=%d",
			e.concesiones.invocaciones, e.denegaciones.invocaciones,
		)
	}
}

func TestPreparacionRegistroCompuestoSolicitudLigadaV3FallaCerradoEnGenerador(
	t *testing.T,
) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	ctxCancelado, cancelarAntes := context.WithCancel(context.Background())
	cancelarAntes()
	generadorNoInvocado := &generadorDecisionOperacionV3Prueba{
		referencia: "dec_compuesta_no_invocada_0123456789abcdef",
	}
	_, candidata, err := e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctxCancelado, e.solicitud, e.resultado, generadorNoInvocado,
	)
	if !errors.Is(err, context.Canceled) ||
		candidataRegistroDecisionV3Valida(candidata) ||
		generadorNoInvocado.invocaciones.Load() != 0 ||
		e.fuente.invocaciones != 0 {
		t.Fatalf("cancelacion previa alcanzo dependencias: %v", err)
	}

	var generadorNulo *generadorDecisionOperacionV3Prueba
	_, candidata, err = e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorNulo,
	)
	if !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) ||
		candidataRegistroDecisionV3Valida(candidata) ||
		e.fuente.invocaciones != 0 {
		t.Fatalf("generador nulo tipado aceptado: %v", err)
	}

	var servicioNulo *ServicioAutorizacionSolicitudLigadaV3
	_, candidata, err = servicioNulo.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorNoInvocado,
	)
	if !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) ||
		candidataRegistroDecisionV3Valida(candidata) {
		t.Fatalf("receptor nulo aceptado: %v", err)
	}

	generadorInvalido := &generadorDecisionOperacionV3Prueba{referencia: "decision:*"}
	_, candidata, err = e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorInvalido,
	)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		candidataRegistroDecisionV3Valida(candidata) ||
		generadorInvalido.invocaciones.Load() != 1 ||
		e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf("referencia insegura aceptada: %v", err)
	}

	secreto := "postgres://usuario:clave@interno persona=12345678Z"
	fallo := errors.New(secreto)
	generadorFallido := &generadorDecisionOperacionV3Prueba{err: fallo}
	_, candidata, err = e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		context.Background(), e.solicitud, e.resultado, generadorFallido,
	)
	if !errors.Is(err, fallo) || strings.Contains(err.Error(), secreto) ||
		candidataRegistroDecisionV3Valida(candidata) ||
		e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf("fallo de generador abierto o filtrado: %v", err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	generadorCancelado := &generadorDecisionOperacionV3Prueba{
		referencia: "dec_compuesta_cancelada_0123456789abcdef",
		despues:    cancelar,
	}
	_, candidata, err = e.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx, e.solicitud, e.resultado, generadorCancelado,
	)
	if !errors.Is(err, context.Canceled) ||
		candidataRegistroDecisionV3Valida(candidata) ||
		e.concesiones.invocaciones != 0 || e.denegaciones.invocaciones != 0 {
		t.Fatalf("cancelacion tras generador ignorada: %v", err)
	}
}

func TestPreparacionRegistroCompuestoSolicitudLigadaV3AislaCienReferenciasConcurrentes(
	t *testing.T,
) {
	e := nuevoEntornoAutorizacionSolicitudV3Prueba(t)
	registro := &registroDenegacionesCompuestoConcurrentePrueba{}
	generadorConfigurado := &generadorDecisionOperacionV3Prueba{
		referencia: "dec_configurada_no_usada_0123456789abcdef",
	}
	preparador, err := NuevoServicioPreparacionSolicitudLigadaV3(
		fuenteAutorizacionCompuestaConcurrentePrueba{instantanea: e.instantanea},
		registro,
		validadorMotivoCompuestoConcurrentePrueba{
			referencia: referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba),
		},
		relojAutorizacionCompuestaConcurrentePrueba{ahora: e.ahora},
		generadorConfigurado,
		ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}

	const total = 100
	var grupo sync.WaitGroup
	errores := make(chan error, total)
	for indice := 0; indice < total; indice++ {
		indice := indice
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			referencia := fmt.Sprintf("dec_compuesta_%032x", indice+1)
			generador := &generadorDecisionOperacionV3Prueba{referencia: referencia}
			decision, candidata, errPreparacion :=
				preparador.PrepararRegistroCompuestoSolicitudLigadaV3(
					context.Background(), e.solicitud, e.resultado, generador,
				)
			concedida, orden, denegacion, errCandidata := candidata.Resultado()
			if errPreparacion != nil || errCandidata != nil || !concedida ||
				!ordenValidaSolicitudLigadaV3(orden) ||
				ordenDenegacionValidaSolicitudLigadaV3(denegacion) ||
				generador.invocaciones.Load() != 1 {
				errores <- fmt.Errorf(
					"operacion %d incoherente: preparacion=%v candidata=%v",
					indice, errPreparacion, errCandidata,
				)
				return
			}
			if obtenida := referenciaDecisionAutorizacionV3PruebaSinFatal(decision); obtenida != referencia {
				errores <- fmt.Errorf(
					"operacion %d cruzo referencias: esperada=%q obtenida=%q",
					indice, referencia, obtenida,
				)
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Error(err)
	}
	if registro.invocaciones.Load() != 0 ||
		generadorConfigurado.invocaciones.Load() != 0 {
		t.Fatalf(
			"la composicion toco dependencias prohibidas: denegaciones=%d generador_configurado=%d",
			registro.invocaciones.Load(), generadorConfigurado.invocaciones.Load(),
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

func ordenDenegacionValidaSolicitudLigadaV3(
	orden ports.OrdenRegistroDenegacionAutorizacionLigadaV3,
) bool {
	_, err := orden.Datos()
	return err == nil
}

func candidataRegistroDecisionV3Valida(
	candidata ports.CandidataRegistroDecisionAutorizacionLigadaV3,
) bool {
	_, _, _, err := candidata.Resultado()
	return err == nil
}

func referenciaDecisionAutorizacionV3Prueba(
	t *testing.T,
	decision domain.DecisionAutorizacionLigadaV3,
) string {
	t.Helper()
	referencia := referenciaDecisionAutorizacionV3PruebaSinFatal(decision)
	if referencia == "" {
		t.Fatal("decision V3 sin referencia canonica")
	}
	return referencia
}

func referenciaDecisionAutorizacionV3PruebaSinFatal(
	decision domain.DecisionAutorizacionLigadaV3,
) string {
	canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return ""
	}
	var documento struct {
		DecisionRef string `json:"decision_ref"`
	}
	if json.Unmarshal(canon, &documento) != nil {
		return ""
	}
	return documento.DecisionRef
}
