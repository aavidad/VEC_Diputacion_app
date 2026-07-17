package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	claveMotivoAutorizacionV2Prueba           = "motivo_11111111111111111111111111111111"
	claveMotivoAutorizacionV2Alternativa      = "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	referenciaCorrelacionAutorizacionV2Prueba = "correlacion_11111111111111111111111111111111"
)

type generadorCorrelacionAplicacionPrueba struct{ valor string }

func (g generadorCorrelacionAplicacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func referenciaCorrelacionAplicacionPrueba(
	valor string,
) domain.ReferenciaCorrelacionAutorizacionV2 {
	referencia, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionAplicacionPrueba{valor: valor},
	)
	if err != nil {
		panic("fixture de correlacion nominal V2 invalido: " + err.Error())
	}
	return referencia
}

type registroAutorizacionSolicitudV2Prueba struct {
	err          error
	concesiones  int
	denegaciones int
	decision     domain.DecisionAutorizacion
	motivo       domain.ReferenciaEntradaCatalogo
}

type validadorMotivoAutorizacionV2Prueba struct {
	referencia domain.ReferenciaEntradaCatalogo
}

func (v *validadorMotivoAutorizacionV2Prueba) ValidarReferenciaMotivoAutorizacionV2(
	_ context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	_ time.Time,
) error {
	if v == nil || referencia != v.referencia {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func nuevoValidadorMotivoAutorizacionV2Prueba() *validadorMotivoAutorizacionV2Prueba {
	return &validadorMotivoAutorizacionV2Prueba{
		referencia: referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba),
	}
}

func (r *registroAutorizacionSolicitudV2Prueba) RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
	_ context.Context,
	orden ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	r.concesiones++
	r.decision = datos.Decision
	r.motivo = datos.ReferenciaMotivo
	return r.err
}

func (r *registroAutorizacionSolicitudV2Prueba) RegistrarDenegacionAutorizacionSolicitudLigadaV2(
	_ context.Context,
	orden ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2,
) error {
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	r.denegaciones++
	r.decision = datos.Decision
	r.motivo = datos.ReferenciaMotivo
	return r.err
}

func TestServicioAutorizacionSolicitudLigadaV2EmiteYRegistraSoloV2(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
	registro := &registroAutorizacionSolicitudV2Prueba{}
	servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
		fuente,
		registro,
		registro,
		nuevoValidadorMotivoAutorizacionV2Prueba(),
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "decision:solicitud-ligada:v2:una"},
		ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V2: %v", err)
	}
	if _, implementaV1 := any(servicio).(ports.Autorizador); implementaV1 {
		t.Fatal("el servicio V2 implementa accidentalmente el autorizador V1")
	}
	solicitud := solicitudAutorizacionServicioV2Prueba()
	decision, err := servicio.ExigirSolicitudLigadaV2(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("exigir V2: %v", err)
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(datosSolicitud.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		decision.SolicitudHuellaSHA256 != huellaSolicitud || decision.MotivoHuellaSHA256 != huellaMotivo ||
		registro.concesiones != 1 || registro.decision.DecisionRef != decision.DecisionRef ||
		registro.motivo != datosSolicitud.ReferenciaMotivo {
		t.Fatalf("decision V2 no ligada/registrada: decision=%+v registro=%+v", decision, registro.decision)
	}
}

func TestServicioAutorizacionSolicitudLigadaV2RegistraDenegacionV2(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
	registro := &registroAutorizacionSolicitudV2Prueba{}
	servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
		fuente, registro, registro, nuevoValidadorMotivoAutorizacionV2Prueba(),
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "decision:solicitud-ligada:v2:denegada"},
		ConfiguracionServicioAutorizacion{},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudAutorizacionServicioV2Prueba()
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud.Accion = "bolsa.expediente.eliminar"
	solicitud = nuevaSolicitudAutorizacionServicioV2Prueba(t, datosSolicitud)
	decision, err := servicio.ExigirSolicitudLigadaV2(context.Background(), solicitud)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Concedida ||
		!decision.TieneSolicitudLigadaV2() || registro.denegaciones != 1 || registro.concesiones != 0 ||
		registro.motivo != datosSolicitud.ReferenciaMotivo {
		t.Fatalf("denegacion V2 incorrecta: decision=%+v registro=%+v err=%v", decision, registro.decision, err)
	}
}

func TestServicioAutorizacionSolicitudLigadaV2ClonaAntesDeConsultarFuente(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	iniciada := make(chan struct{})
	continuar := make(chan struct{})
	fuente := &fuenteAutorizacionServicioPrueba{
		instantanea: instantaneaAutorizacionServicioPrueba(t),
		despues: func() {
			close(iniciada)
			<-continuar
		},
	}
	registro := &registroAutorizacionSolicitudV2Prueba{}
	servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
		fuente, registro, registro, nuevoValidadorMotivoAutorizacionV2Prueba(),
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "decision:solicitud-ligada:v2:instantanea"},
		ConfiguracionServicioAutorizacion{},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosOriginales := datosSolicitudAutorizacionServicioV2Prueba()
	datosOriginales.Recurso.Atributos = map[string]string{"estado": "presentado"}
	solicitud := nuevaSolicitudAutorizacionServicioV2Prueba(t, datosOriginales)
	huellaEsperada, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	type resultado struct {
		decision domain.DecisionAutorizacion
		err      error
	}
	resultadoServicio := make(chan resultado, 1)
	go func() {
		decision, err := servicio.ExigirSolicitudLigadaV2(context.Background(), solicitud)
		resultadoServicio <- resultado{decision: decision, err: err}
	}()
	select {
	case <-iniciada:
	case <-time.After(2 * time.Second):
		t.Fatal("el servicio no alcanzo la fuente")
	}
	// La mutacion sucede despues de construir la capacidad y antes de evaluar la
	// instantanea. Constructor y Datos deben conservar copias independientes.
	datosOriginales.Recurso.Ambitos["unidad"] = "nominas"
	datosOriginales.Recurso.Atributos["estado"] = "borrado"
	datosOriginales.ReferenciaMotivo.EntradaClave = "dni_12345678z"
	entregada, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	entregada.Recurso.Ambitos["unidad"] = "intervencion"
	entregada.Recurso.Atributos["estado"] = "eliminado"
	entregada.ReferenciaMotivo.EntradaClave = claveMotivoAutorizacionV2Alternativa
	close(continuar)
	obtenido := <-resultadoServicio
	if obtenido.err != nil || obtenido.decision.SolicitudHuellaSHA256 != huellaEsperada ||
		registro.decision.SolicitudHuellaSHA256 != huellaEsperada {
		t.Fatalf("la mutacion cruzo la instantanea V2: decision=%+v err=%v", obtenido.decision, obtenido.err)
	}
}

func TestNuevoServicioAutorizacionSolicitudLigadaV2RechazaNulosTipados(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	casos := []struct {
		nombre       string
		fuente       ports.FuenteAutorizacion
		concesiones  ports.RegistroDecisionesAutorizacionSolicitudLigadaV2
		denegaciones ports.RegistroDenegacionesAutorizacionSolicitudLigadaV2
		validador    ports.ValidadorReferenciaMotivoAutorizacionV2
		reloj        ports.Reloj
		generador    ports.GeneradorReferenciaDecisionAutorizacion
	}{
		{"fuente", (*fuenteAutorizacionServicioPrueba)(nil), &registroAutorizacionSolicitudV2Prueba{}, &registroAutorizacionSolicitudV2Prueba{}, nuevoValidadorMotivoAutorizacionV2Prueba(), &relojAutorizacionServicioPrueba{ahora: ahora}, &generadorAutorizacionServicioPrueba{referencia: "decision:v2:nula"}},
		{"concesiones", &fuenteAutorizacionServicioPrueba{instantanea: instantanea}, (*registroAutorizacionSolicitudV2Prueba)(nil), &registroAutorizacionSolicitudV2Prueba{}, nuevoValidadorMotivoAutorizacionV2Prueba(), &relojAutorizacionServicioPrueba{ahora: ahora}, &generadorAutorizacionServicioPrueba{referencia: "decision:v2:nula"}},
		{"denegaciones", &fuenteAutorizacionServicioPrueba{instantanea: instantanea}, &registroAutorizacionSolicitudV2Prueba{}, (*registroAutorizacionSolicitudV2Prueba)(nil), nuevoValidadorMotivoAutorizacionV2Prueba(), &relojAutorizacionServicioPrueba{ahora: ahora}, &generadorAutorizacionServicioPrueba{referencia: "decision:v2:nula"}},
		{"validador", &fuenteAutorizacionServicioPrueba{instantanea: instantanea}, &registroAutorizacionSolicitudV2Prueba{}, &registroAutorizacionSolicitudV2Prueba{}, (*validadorMotivoAutorizacionV2Prueba)(nil), &relojAutorizacionServicioPrueba{ahora: ahora}, &generadorAutorizacionServicioPrueba{referencia: "decision:v2:nula"}},
		{"reloj", &fuenteAutorizacionServicioPrueba{instantanea: instantanea}, &registroAutorizacionSolicitudV2Prueba{}, &registroAutorizacionSolicitudV2Prueba{}, nuevoValidadorMotivoAutorizacionV2Prueba(), (*relojAutorizacionServicioPrueba)(nil), &generadorAutorizacionServicioPrueba{referencia: "decision:v2:nula"}},
		{"generador", &fuenteAutorizacionServicioPrueba{instantanea: instantanea}, &registroAutorizacionSolicitudV2Prueba{}, &registroAutorizacionSolicitudV2Prueba{}, nuevoValidadorMotivoAutorizacionV2Prueba(), &relojAutorizacionServicioPrueba{ahora: ahora}, (*generadorAutorizacionServicioPrueba)(nil)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
				caso.fuente, caso.concesiones, caso.denegaciones, caso.validador, caso.reloj, caso.generador,
				ConfiguracionServicioAutorizacion{},
			)
			if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
				t.Fatalf("nulo tipado aceptado: servicio=%v err=%v", servicio, err)
			}
		})
	}
}

func TestServicioAutorizacionSolicitudLigadaV2RechazaMotivoNoResuelto(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	for nombre, mutar := range map[string]func(*domain.DatosSolicitudAutorizacionLigadaV2){
		"referencia cero": func(s *domain.DatosSolicitudAutorizacionLigadaV2) {
			s.ReferenciaMotivo = domain.ReferenciaEntradaCatalogo{}
		},
		"huella cero": func(s *domain.DatosSolicitudAutorizacionLigadaV2) {
			s.ReferenciaMotivo.CatalogoHuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		},
		"referencia legible": func(s *domain.DatosSolicitudAutorizacionLigadaV2) {
			s.ReferenciaMotivo.EntradaClave = "otra_entrada"
		},
		"dato personal con referencia fabricada": func(s *domain.DatosSolicitudAutorizacionLigadaV2) {
			s.ReferenciaMotivo = referenciaMotivoAutorizacionV2Prueba("dni_12345678z")
			s.ReferenciaMotivo.CatalogoHuellaSHA256 = "9999999999999999999999999999999999999999999999999999999999999999"
		},
		"capacidad de correlacion cero": func(s *domain.DatosSolicitudAutorizacionLigadaV2) {
			s.Correlacion = domain.ReferenciaCorrelacionAutorizacionV2{}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
			registro := &registroAutorizacionSolicitudV2Prueba{}
			servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
				fuente, registro, registro, nuevoValidadorMotivoAutorizacionV2Prueba(),
				&relojAutorizacionServicioPrueba{ahora: ahora},
				&generadorAutorizacionServicioPrueba{referencia: "decision:motivo-no-resuelto:v2"},
				ConfiguracionServicioAutorizacion{},
			)
			if err != nil {
				t.Fatal(err)
			}
			datos := datosSolicitudAutorizacionServicioV2Prueba()
			mutar(&datos)
			if _, err := domain.NuevaSolicitudAutorizacionLigadaV2(datos); !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
				registro.concesiones != 0 || registro.denegaciones != 0 || fuente.invocaciones != 0 {
				t.Fatalf("entrada invalida alcanzo dependencias: fuente=%d registro=%+v err=%v", fuente.invocaciones, registro, err)
			}
			if _, err := servicio.ExigirSolicitudLigadaV2(
				context.Background(), domain.SolicitudAutorizacionLigadaV2{},
			); !errors.Is(err, domain.ErrAutorizacionDenegada) ||
				!errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) || fuente.invocaciones != 0 {
				t.Fatalf("capacidad cero alcanzo dependencias: fuente=%d err=%v", fuente.invocaciones, err)
			}
		})
	}
}

func TestServicioAutorizacionSolicitudLigadaV2RechazaReferenciaOpacaInexistente(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	catalogo := catalogoMotivosAutorizacionV2Prueba(t, ahora)
	validador, err := NuevoValidadorReferenciaMotivoCatalogoV2(
		&consultaCatalogosMotivoAutorizacionV2Prueba{catalogo: catalogo},
		catalogo.ID,
	)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
	registro := &registroAutorizacionSolicitudV2Prueba{}
	servicio, err := NuevoServicioAutorizacionSolicitudLigadaV2(
		fuente, registro, registro, validador,
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "decision:motivo-pii-fabricado:v2"},
		ConfiguracionServicioAutorizacion{},
	)
	if err != nil {
		t.Fatal(err)
	}
	datosSolicitud := datosSolicitudAutorizacionServicioV2Prueba()
	datosSolicitud.ReferenciaMotivo = referenciaCatalogoMotivoAutorizacionV2Prueba(
		t,
		catalogo,
		claveMotivoAutorizacionV2Alternativa,
	)
	solicitud := nuevaSolicitudAutorizacionServicioV2Prueba(t, datosSolicitud)
	if _, err := servicio.ExigirSolicitudLigadaV2(context.Background(), solicitud); !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) || fuente.invocaciones != 0 ||
		registro.concesiones != 0 || registro.denegaciones != 0 {
		t.Fatalf("referencia opaca inexistente alcanzo el PDP: fuente=%d registro=%+v err=%v", fuente.invocaciones, registro, err)
	}
}

func TestServicioAutorizacionV1YV2NoSeAceptanEntreRegistros(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
	registro := &registroAutorizacionServicioPrueba{}
	servicioV1 := nuevoServicioAutorizacionServicioPrueba(
		t,
		fuente,
		registro,
		&generadorAutorizacionServicioPrueba{referencia: "decision:separacion:v1"},
		ahora,
	)
	decisionV1, err := servicioV1.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if err != nil || decisionV1.TieneSolicitudLigadaV2() {
		t.Fatalf("servicio V1 produjo V2: decision=%+v err=%v", decisionV1, err)
	}
	if _, implementaV2 := any(servicioV1).(ports.AutorizadorSolicitudLigadaV2); implementaV2 {
		t.Fatal("el servicio V1 implementa accidentalmente el autorizador V2")
	}
	proyeccionV2, err := proyectarSolicitudAutorizacionLigadaV2(solicitudAutorizacionServicioV2Prueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicioV1.Exigir(context.Background(), proyeccionV2); !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("el servicio V1 acepto una referencia de motivo V2: %v", err)
	}
	var servicioNulo *ServicioAutorizacionSolicitudLigadaV2
	if _, err := servicioNulo.ExigirSolicitudLigadaV2(
		context.Background(),
		solicitudAutorizacionServicioV2Prueba(),
	); !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("receptor V2 nulo no fallo cerrado: %v", err)
	}
}

func solicitudAutorizacionServicioV2Prueba() domain.SolicitudAutorizacionLigadaV2 {
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		datosSolicitudAutorizacionServicioV2Prueba(),
	)
	if err != nil {
		panic("fixture de solicitud de autorizacion V2 invalido: " + err.Error())
	}
	return solicitud
}

func datosSolicitudAutorizacionServicioV2Prueba() domain.DatosSolicitudAutorizacionLigadaV2 {
	solicitud := solicitudAutorizacionServicioPrueba()
	return domain.DatosSolicitudAutorizacionLigadaV2{
		ContextoActor: solicitud.ContextoActor, VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		ReferenciaMotivo: referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba),
		Accion:           solicitud.Accion, Recurso: solicitud.Recurso, Finalidad: solicitud.Finalidad,
		Correlacion: referenciaCorrelacionAplicacionPrueba(
			referenciaCorrelacionAutorizacionV2Prueba,
		),
	}
}

func nuevaSolicitudAutorizacionServicioV2Prueba(
	t *testing.T,
	datos domain.DatosSolicitudAutorizacionLigadaV2,
) domain.SolicitudAutorizacionLigadaV2 {
	t.Helper()
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(datos)
	if err != nil {
		t.Fatalf("crear solicitud nominal V2: %v", err)
	}
	return solicitud
}

func referenciaMotivoAutorizacionV2Prueba(clave string) domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		EntradaClave:         clave,
	}
}
