package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteAutorizacionServicioPrueba struct {
	instantanea  domain.InstantaneaAutorizacion
	err          error
	invocaciones int
	despues      func()
}

func (f *fuenteAutorizacionServicioPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context,
	_, _ string,
) (domain.InstantaneaAutorizacion, error) {
	f.invocaciones++
	if f.despues != nil {
		f.despues()
	}
	return f.instantanea, f.err
}

type registroAutorizacionServicioPrueba struct {
	err          error
	invocaciones int
	concesiones  int
	denegaciones int
	decision     domain.DecisionAutorizacion
}

func (r *registroAutorizacionServicioPrueba) RegistrarDecisionSiInstantaneaVigente(
	_ context.Context,
	decision domain.DecisionAutorizacion,
) error {
	r.invocaciones++
	r.concesiones++
	r.decision = decision
	return r.err
}

func (r *registroAutorizacionServicioPrueba) RegistrarDenegacionAutorizacion(
	_ context.Context,
	decision domain.DecisionAutorizacion,
) error {
	r.invocaciones++
	r.denegaciones++
	r.decision = decision
	return r.err
}

func (r *registroAutorizacionServicioPrueba) ObtenerDecision(
	context.Context,
	string,
) (domain.DecisionAutorizacion, error) {
	return domain.DecisionAutorizacion{}, ports.ErrDecisionAutorizacionNoEncontrada
}

type relojAutorizacionServicioPrueba struct{ ahora time.Time }

func (r *relojAutorizacionServicioPrueba) Ahora() time.Time { return r.ahora }

type generadorAutorizacionServicioPrueba struct {
	referencia   string
	invocaciones int
}

func (g *generadorAutorizacionServicioPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	g.invocaciones++
	return g.referencia, nil
}

func TestNuevoServicioAutorizacionRechazaDependenciasNulasTipadas(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre    string
		fuente    ports.FuenteAutorizacion
		registro  ports.RegistroDecisionesAutorizacion
		denegadas ports.RegistroDenegacionesAutorizacion
		reloj     ports.Reloj
		generador ports.GeneradorReferenciaDecisionAutorizacion
	}{
		{
			nombre: "fuente", fuente: (*fuenteAutorizacionServicioPrueba)(nil),
			registro: &registroAutorizacionServicioPrueba{}, denegadas: &registroAutorizacionServicioPrueba{},
			reloj:     &relojAutorizacionServicioPrueba{ahora: ahora},
			generador: &generadorAutorizacionServicioPrueba{referencia: "decision:una"},
		},
		{
			nombre: "registro", fuente: &fuenteAutorizacionServicioPrueba{instantanea: instantanea},
			registro: (*registroAutorizacionServicioPrueba)(nil), denegadas: &registroAutorizacionServicioPrueba{},
			reloj:     &relojAutorizacionServicioPrueba{ahora: ahora},
			generador: &generadorAutorizacionServicioPrueba{referencia: "decision:una"},
		},
		{
			nombre: "registro_denegaciones", fuente: &fuenteAutorizacionServicioPrueba{instantanea: instantanea},
			registro: &registroAutorizacionServicioPrueba{}, denegadas: (*registroAutorizacionServicioPrueba)(nil),
			reloj:     &relojAutorizacionServicioPrueba{ahora: ahora},
			generador: &generadorAutorizacionServicioPrueba{referencia: "decision:una"},
		},
		{
			nombre: "reloj", fuente: &fuenteAutorizacionServicioPrueba{instantanea: instantanea},
			registro: &registroAutorizacionServicioPrueba{}, denegadas: &registroAutorizacionServicioPrueba{},
			reloj:     (*relojAutorizacionServicioPrueba)(nil),
			generador: &generadorAutorizacionServicioPrueba{referencia: "decision:una"},
		},
		{
			nombre: "generador", fuente: &fuenteAutorizacionServicioPrueba{instantanea: instantanea},
			registro: &registroAutorizacionServicioPrueba{}, denegadas: &registroAutorizacionServicioPrueba{},
			reloj:     &relojAutorizacionServicioPrueba{ahora: ahora},
			generador: (*generadorAutorizacionServicioPrueba)(nil),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, err := NuevoServicioAutorizacion(
				caso.fuente, caso.registro, caso.denegadas, caso.reloj, caso.generador,
				ConfiguracionServicioAutorizacion{},
			)
			if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
				t.Fatalf("dependencia nula tipada aceptada: servicio=%v err=%v", servicio, err)
			}
		})
	}
}

func TestServicioAutorizacionDeniegaContextoNuloOCanceladoSinConsultarAutoridad(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
	}{
		{nombre: "nulo", ctx: nil},
		{nombre: "cancelado", ctx: contextoCanceladoAutorizacionPrueba()},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
			registro := &registroAutorizacionServicioPrueba{}
			generador := &generadorAutorizacionServicioPrueba{referencia: "decision:no-usar"}
			servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, ahora)
			decision, err := servicio.Exigir(caso.ctx, solicitudAutorizacionServicioPrueba())
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.DecisionRef != "" {
				t.Fatalf("contexto %s no denegado: decision=%+v err=%v", caso.nombre, decision, err)
			}
			if fuente.invocaciones != 0 || generador.invocaciones != 0 || registro.invocaciones != 0 {
				t.Fatalf("contexto %s produjo efectos: fuente=%d generador=%d registro=%d",
					caso.nombre, fuente.invocaciones, generador.invocaciones, registro.invocaciones)
			}
		})
	}

	var servicioNulo *ServicioAutorizacion
	if _, err := servicioNulo.Exigir(context.Background(), solicitudAutorizacionServicioPrueba()); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("receptor nulo no denegado: %v", err)
	}
}

func TestServicioAutorizacionDeniegaSinVinculoAutenticacionActor(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	for _, caso := range []struct {
		nombre string
		mutar  func(*domain.SolicitudAutorizacion)
	}{
		{"sin contexto actor", func(s *domain.SolicitudAutorizacion) { s.ContextoActor = domain.ContextoActor{} }},
		{"sin vinculo", func(s *domain.SolicitudAutorizacion) {
			s.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
			registro := &registroAutorizacionServicioPrueba{}
			generador := &generadorAutorizacionServicioPrueba{referencia: "decision:no-usar"}
			servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, ahora)
			solicitud := solicitudAutorizacionServicioPrueba()
			caso.mutar(&solicitud)
			decision, err := servicio.Exigir(context.Background(), solicitud)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Validar() == nil ||
				fuente.invocaciones != 0 || registro.invocaciones != 0 || generador.invocaciones != 0 {
				t.Fatalf("omision no denegada antes de efectos: decision=%v err=%v", decision, err)
			}
		})
	}
}

func TestServicioAutorizacionRecompruebaCancelacionTrasLeerInstantanea(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea, despues: cancelar}
	registro := &registroAutorizacionServicioPrueba{}
	generador := &generadorAutorizacionServicioPrueba{referencia: "decision:no-usar"}
	servicio := nuevoServicioAutorizacionServicioPrueba(
		t, fuente, registro, generador, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC),
	)
	if decision, err := servicio.Exigir(ctx, solicitudAutorizacionServicioPrueba()); !errors.Is(err, context.Canceled) || !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		decision.DecisionRef != "" {
		t.Fatalf("cancelacion concurrente no cerrada: decision=%+v err=%v", decision, err)
	}
	if generador.invocaciones != 0 || registro.invocaciones != 0 {
		t.Fatalf("cancelacion produjo efectos: generador=%d registro=%d", generador.invocaciones, registro.invocaciones)
	}
}

func contextoCanceladoAutorizacionPrueba() context.Context {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	return ctx
}

func TestServicioAutorizacionNoInventaEvidenciaSiLaFuenteNoEstaDisponible(t *testing.T) {
	errFuente := errors.New("catalogo no disponible")
	fuente := &fuenteAutorizacionServicioPrueba{err: errFuente}
	registro := &registroAutorizacionServicioPrueba{}
	generador := &generadorAutorizacionServicioPrueba{referencia: "decision:no-debe-usarse"}
	servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC))

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, errFuente) {
		t.Fatalf("fallo de fuente no cerro acceso: decision=%+v err=%v", decision, err)
	}
	if decision.DecisionRef != "" || registro.invocaciones != 0 || generador.invocaciones != 0 {
		t.Fatalf("se fingio evidencia sin instantanea: decision=%+v registro=%d generador=%d", decision, registro.invocaciones, generador.invocaciones)
	}
}

func TestServicioAutorizacionNoRegistraInstantaneaNoFiable(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.CatalogoPoliticasHuellaSHA256 = "no-es-una-huella"
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{}
	generador := &generadorAutorizacionServicioPrueba{referencia: "decision:no-debe-usarse"}
	servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC))

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.DecisionRef != "" {
		t.Fatalf("instantanea no fiable no cerro acceso: decision=%+v err=%v", decision, err)
	}
	if registro.invocaciones != 0 || generador.invocaciones != 0 {
		t.Fatalf("se intento registrar una instantanea no fiable: registro=%d generador=%d", registro.invocaciones, generador.invocaciones)
	}
}

func TestServicioAutorizacionConvierteConflictoCASenDenegacionNoRegistrada(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{err: ports.ErrInstantaneaAutorizacionObsoleta}
	generador := &generadorAutorizacionServicioPrueba{referencia: "decision:cas-obsoleto"}
	servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC))

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
		t.Fatalf("conflicto CAS no cerro acceso: decision=%+v err=%v", decision, err)
	}
	if decision.Concedida || decision.Codigo != "instantanea_obsoleta" || registro.invocaciones != 1 {
		t.Fatalf("resultado CAS inseguro: decision=%+v registro=%d", decision, registro.invocaciones)
	}
	if !registro.decision.Concedida || registro.decision.Codigo != "concedida" {
		t.Fatalf("el registro no recibio la decision evaluada original: %+v", registro.decision)
	}
	if err := registro.decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("el CAS recibio evidencia incompleta: %v", err)
	}
}

func TestServicioAutorizacionRegistraDenegacionSinCrearCapacidad(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{}
	servicio := nuevoServicioAutorizacionServicioPrueba(
		t, fuente, registro,
		&generadorAutorizacionServicioPrueba{referencia: "decision:denegada"},
		time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC),
	)

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || decision.Codigo != "accion_no_concedida" {
		t.Fatalf("resultado inesperado: decision=%+v err=%v", decision, err)
	}
	if registro.concesiones != 0 || registro.denegaciones != 1 || registro.decision.Concedida {
		t.Fatalf("la denegacion alcanzo el registro equivocado: concesiones=%d denegaciones=%d decision=%+v",
			registro.concesiones, registro.denegaciones, registro.decision)
	}
}

func TestServicioAutorizacionFalloDeTrazaNoOcultaMotivoDeDenegacion(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{err: errors.New("traza no disponible")}
	servicio := nuevoServicioAutorizacionServicioPrueba(
		t, fuente, registro,
		&generadorAutorizacionServicioPrueba{referencia: "decision:denegada-sin-traza"},
		time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC),
	)

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if !errors.Is(err, domain.ErrAutorizacionDenegada) ||
		!errors.Is(err, ports.ErrRegistroDenegacionNoDisponible) {
		t.Fatalf("fallo de traza no propagado: decision=%+v err=%v", decision, err)
	}
	if decision.Concedida || decision.Codigo != "accion_no_concedida" ||
		registro.concesiones != 0 || registro.denegaciones != 1 {
		t.Fatalf("el fallo de traza altero la decision: decision=%+v registro=%+v", decision, registro)
	}
}

func TestServicioAutorizacionCanonizaRelojAMicrosegundo(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	registro := &registroAutorizacionServicioPrueba{}
	generador := &generadorAutorizacionServicioPrueba{referencia: "decision:precision"}
	ahora := time.Date(2026, 7, 15, 10, 0, 0, 123_456_789, time.FixedZone("CEST", 2*60*60))
	servicio := nuevoServicioAutorizacionServicioPrueba(t, fuente, registro, generador, ahora)

	decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
	if err != nil {
		t.Fatalf("autorizar: %v", err)
	}
	esperado := ahora.UTC().Truncate(time.Microsecond)
	if decision.EmitidaEn != esperado || decision.EmitidaEn.Location() != time.UTC || decision.EmitidaEn.Nanosecond()%1_000 != 0 {
		t.Fatalf("hora no canonica: recibida=%s esperada=%s", decision.EmitidaEn, esperado)
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision canonica invalida: %v", err)
	}
}

func TestNuevoServicioAutorizacionRechazaVigenciaSubmicrosegundo(t *testing.T) {
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	servicio, err := NuevoServicioAutorizacion(
		&fuenteAutorizacionServicioPrueba{instantanea: instantanea},
		&registroAutorizacionServicioPrueba{},
		&registroAutorizacionServicioPrueba{},
		&relojAutorizacionServicioPrueba{ahora: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)},
		&generadorAutorizacionServicioPrueba{referencia: "decision:precision"},
		ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute + time.Nanosecond},
	)
	if servicio != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("vigencia no portable aceptada: servicio=%v err=%v", servicio, err)
	}
}

func TestServicioAutorizacionNoSuperaFronterasTemporalesConocidas(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre   string
		preparar func(*domain.InstantaneaAutorizacion)
		espera   time.Duration
	}{
		{
			nombre: "fin de asignacion",
			preparar: func(i *domain.InstantaneaAutorizacion) {
				i.AsignacionPerfil.VigenteHasta = ahora.Add(20 * time.Second)
			},
			espera: 20 * time.Second,
		},
		{
			nombre: "inicio futuro de politica",
			preparar: func(i *domain.InstantaneaAutorizacion) {
				i.Politicas = []domain.PoliticaRestrictiva{politicaTemporalAutorizacionServicioPrueba(
					ahora.Add(30*time.Second), ahora.Add(time.Hour),
				)}
			},
			espera: 30 * time.Second,
		},
		{
			nombre: "fin de politica vigente",
			preparar: func(i *domain.InstantaneaAutorizacion) {
				i.Politicas = []domain.PoliticaRestrictiva{politicaTemporalAutorizacionServicioPrueba(
					ahora.Add(-time.Hour), ahora.Add(15*time.Second),
				)}
			},
			espera: 15 * time.Second,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			instantanea := instantaneaAutorizacionServicioPrueba(t)
			caso.preparar(&instantanea)
			huella, err := domain.HuellaCatalogoPoliticasAutorizacion(instantanea.Politicas)
			if err != nil {
				t.Fatalf("huella de catalogo: %v", err)
			}
			instantanea.RevisionCatalogoPoliticas++
			instantanea.CatalogoPoliticasHuellaSHA256 = huella
			fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
			registro := &registroAutorizacionServicioPrueba{}
			servicio := nuevoServicioAutorizacionServicioPrueba(
				t, fuente, registro,
				&generadorAutorizacionServicioPrueba{referencia: "decision:frontera-temporal"},
				ahora,
			)

			decision, err := servicio.Exigir(context.Background(), solicitudAutorizacionServicioPrueba())
			if err != nil {
				t.Fatalf("autorizar: %v", err)
			}
			if vigencia := decision.ValidaHasta.Sub(decision.EmitidaEn); vigencia != caso.espera {
				t.Fatalf("vigencia = %v, se esperaba %v", vigencia, caso.espera)
			}
		})
	}
}

func TestServicioAutorizacionNoSuperaSesionNiContextoActor(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	for _, caso := range []struct {
		nombre   string
		preparar func(domain.ContextoActor, domain.AutenticacionRevalidadaV1) (domain.ContextoActor, domain.AutenticacionRevalidadaV1, time.Time)
	}{
		{"fin de sesion", func(actor domain.ContextoActor, autenticacion domain.AutenticacionRevalidadaV1) (domain.ContextoActor, domain.AutenticacionRevalidadaV1, time.Time) {
			autenticacion.SesionValidaHasta = ahora.Add(20 * time.Second)
			return actor, autenticacion, autenticacion.SesionValidaHasta
		}},
		{"fin de contexto actor", func(actor domain.ContextoActor, autenticacion domain.AutenticacionRevalidadaV1) (domain.ContextoActor, domain.AutenticacionRevalidadaV1, time.Time) {
			actor.Instantanea.VigenteHasta = ahora.Add(15 * time.Second)
			return actor, autenticacion, actor.Instantanea.VigenteHasta
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			actor, base := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
			datos, _ := base.Datos()
			actor, autenticacion, esperado := caso.preparar(actor, datos.Autenticacion())
			vinculo, err := domain.CrearVinculoAutenticacionActorV1(
				context.Background(), revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
				domain.SolicitudRevalidacionAutenticacionActorV1{
					AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
				},
				actor, ahora,
			)
			if err != nil {
				t.Fatalf("crear vinculo acotado: %v", err)
			}
			solicitud := solicitudAutorizacionServicioPrueba()
			solicitud.Principal = actor.Principal
			solicitud.PerfilActivoRef = actor.PerfilActivoRef
			solicitud.ContextoActor = actor
			solicitud.VinculoAutenticacionActor = vinculo
			fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantaneaAutorizacionServicioPrueba(t)}
			registro := &registroAutorizacionServicioPrueba{}
			servicio := nuevoServicioAutorizacionServicioPrueba(
				t, fuente, registro, &generadorAutorizacionServicioPrueba{referencia: "decision:limite-vinculo"}, ahora,
			)
			decision, err := servicio.Exigir(context.Background(), solicitud)
			if err != nil || !decision.ValidaHasta.Equal(esperado) {
				t.Fatalf("limite=%s; esperado=%s; err=%v", decision.ValidaHasta, esperado, err)
			}
		})
	}
}

func politicaTemporalAutorizacionServicioPrueba(desde, hasta time.Time) domain.PoliticaRestrictiva {
	return domain.PoliticaRestrictiva{
		PoliticaID: "frontera_temporal", Version: 1, Nombre: "Frontera temporal",
		Estado: domain.EstadoPoliticaRestrictivaPublicada, Efecto: domain.EfectoPoliticaRestringir,
		Acciones: []string{"bolsa.expediente.leer"}, Modulos: []string{"bolsa"},
		TiposRecurso: []string{"expediente"}, VigenteDesde: desde, VigenteHasta: hasta,
		PublicadaPor: "responsable-seguridad", PublicadaEn: time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC),
	}
}

func nuevoServicioAutorizacionServicioPrueba(
	t *testing.T,
	fuente ports.FuenteAutorizacion,
	registro ports.RegistroDecisionesAutorizacion,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	ahora time.Time,
) *ServicioAutorizacion {
	t.Helper()
	servicio, err := NuevoServicioAutorizacion(
		fuente,
		registro,
		registro.(ports.RegistroDenegacionesAutorizacion),
		&relojAutorizacionServicioPrueba{ahora: ahora},
		generador,
		ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func instantaneaAutorizacionServicioPrueba(t *testing.T) domain.InstantaneaAutorizacion {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	version := domain.VersionRol{
		RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico de bolsa", Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente", Finalidades: []string{"gestion_bolsa"},
			GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	asignacion := domain.AsignacionPerfil{
		AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PrincipalID: "per_0123456789abcdefghijkl",
		VersionRolRef: version.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatalf("huella de catalogo vacio: %v", err)
	}
	return domain.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
}

func solicitudAutorizacionServicioPrueba() domain.SolicitudAutorizacion {
	instante := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(instante)
	return domain.SolicitudAutorizacion{
		Principal: actor.Principal, PerfilActivoRef: actor.PerfilActivoRef,
		ContextoActor: actor, VinculoAutenticacionActor: vinculo,
		Accion: "bolsa.expediente.leer",
		Recurso: domain.RecursoAutorizable{
			Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
			Ambitos: map[string]string{"unidad": "seleccion"},
		},
		Finalidad: "gestion_bolsa", CorrelacionRef: "corr-servicio", Motivo: "Consulta administrativa",
	}
}
