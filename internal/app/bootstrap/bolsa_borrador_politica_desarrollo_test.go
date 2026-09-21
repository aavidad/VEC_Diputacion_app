package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadInicialBorradorBolsaPrueba struct {
	preparadas, publicadas int
	permitirInicial        bool
	permitirSucesion       bool
	preparar               func(dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error)
	publicar               func(dominiovec.InstantaneaAutorizacion) error
	publicada              dominiovec.InstantaneaAutorizacion
}

func (a *autoridadInicialBorradorBolsaPrueba) prepararInstantanea(
	_ context.Context, instantanea dominiovec.InstantaneaAutorizacion, permitirInicial bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	a.preparadas++
	a.permitirInicial = permitirInicial
	if a.preparar != nil {
		return a.preparar(instantanea)
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(instantanea), nil
}

func (a *autoridadInicialBorradorBolsaPrueba) publicarInstantaneaDesdePreimagen(
	_ context.Context, instantanea, preimagen dominiovec.InstantaneaAutorizacion,
) error {
	if preimagen.VersionRol.Version != 1 || len(preimagen.VersionRol.Concesiones) != 2 ||
		(instantanea.AsignacionPerfil.Version == 2 && !a.permitirSucesion) {
		return errors.New("preimagen no admitida")
	}
	a.publicadas++
	a.publicada = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(instantanea)
	if a.publicar != nil {
		return a.publicar(instantanea)
	}
	return nil
}

type registroBorradorBolsaPrueba struct{ concesiones, denegaciones int }

func (r *registroBorradorBolsaPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.concesiones++
	return time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC), nil
}

func (r *registroBorradorBolsaPrueba) RegistrarDenegacionAutorizacionLigadaV3(
	context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	r.denegaciones++
	return nil
}

func nuevaPoliticaBorradorBolsaPrueba(t *testing.T) (*politicaBorradorLlamamientoBolsaDesarrollo, *soporteSesionBorradorBolsaDesarrollo, *autoridadInicialBorradorBolsaPrueba, *registroBorradorBolsaPrueba) {
	t.Helper()
	directorio, soporteCT, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	soporte, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporteCT, ahora)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadInicialBorradorBolsaPrueba{}
	registro := &registroBorradorBolsaPrueba{}
	politica, err := nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(soporte, autoridad, registro, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	return politica, soporte, autoridad, registro
}

func TestPoliticaBorradorBolsaPublicaTresConcesionesNominales(t *testing.T) {
	politica, soporte, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = politica.ObtenerInstantaneaAutorizacion(context.Background(), datos.PrincipalID, datos.PerfilActivoRef); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("fuente antes de publicación=%v", err)
	}
	if err = politica.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 || !autoridad.permitirInicial {
		t.Fatalf("publicación inicial no usó la autoridad común: %+v", autoridad)
	}
	instantanea, err := politica.ObtenerInstantaneaAutorizacion(context.Background(), datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil {
		t.Fatal(err)
	}
	if instantanea.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef || len(instantanea.VersionRol.Concesiones) != 3 {
		t.Fatalf("perfil o concesiones inesperados: %+v", instantanea)
	}
	concesiones := map[string]dominiovec.ConcesionRol{}
	for _, concesion := range instantanea.VersionRol.Concesiones {
		concesiones[concesion.Accion] = concesion
	}
	for accion, finalidad := range map[string]string{
		puertosbolsa.AccionCrearBorradorLlamamientoInterno:     puertosbolsa.FinalidadCrearBorradorLlamamientoInterno,
		puertosbolsa.AccionConsultarBorradorLlamamientoInterno: puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno,
		puertosbolsa.AccionCambiarSituacionParticipacion:       puertosbolsa.FinalidadCambiarSituacionParticipacion,
	} {
		concesion, existe := concesiones[accion]
		tipo := puertosbolsa.TipoRecursoBorradorLlamamiento
		if accion == puertosbolsa.AccionCambiarSituacionParticipacion {
			tipo = puertosbolsa.TipoRecursoSituacionParticipacion
		}
		if !existe || concesion.ModuloID != puertosbolsa.ModuloBorradorLlamamiento || concesion.TipoRecurso != tipo || len(concesion.Finalidades) != 1 || concesion.Finalidades[0] != finalidad {
			t.Fatalf("concesión B-BACK no exacta para %s: %+v", accion, concesion)
		}
	}
	for _, concesion := range instantanea.VersionRol.Concesiones {
		if concesion.ModuloID == "contratacion_temporal" {
			t.Fatalf("la política Bolsa contiene una concesión CT: %+v", concesion)
		}
	}
}

func TestPoliticaBorradorBolsaSoloInauguraSemillaExacta(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error)
		aceptada bool
	}{
		{
			nombre: "preimagen existente exacta",
			preparar: func(semilla dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
				return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla), nil
			},
			aceptada: true,
		},
		{
			nombre: "asignacion existente diferente",
			preparar: func(semilla dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
				preparada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
				preparada.AsignacionPerfil.Version++
				return preparada, nil
			},
		},
		{
			nombre: "asignacion existente revocada",
			preparar: func(semilla dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
				preparada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
				preparada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
				preparada.AsignacionPerfil.RevocadaPor = "seguridad:desarrollo:no-autoritativa"
				preparada.AsignacionPerfil.RevocadaEn = preparada.AsignacionPerfil.EmitidaEn
				preparada.AsignacionPerfil.RevocacionRef = "revocacion:bolsa-bback-prueba"
				return preparada, nil
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
			autoridad.preparar = caso.preparar
			err := politica.PublicarInicial(context.Background())
			if caso.aceptada {
				if err != nil || !politica.publicada || autoridad.publicadas != 1 {
					t.Fatalf("semilla exacta rechazada: err=%v politica=%+v autoridad=%+v", err, politica, autoridad)
				}
				return
			}
			if !errors.Is(err, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible) || politica.publicada || autoridad.publicadas != 0 {
				t.Fatalf("preimagen no inaugural aceptada: err=%v politica=%+v autoridad=%+v", err, politica, autoridad)
			}
		})
	}
}

func TestPoliticaBorradorBolsaPublicaLaSemillaParaCerrarCarrera(t *testing.T) {
	politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	var preparada dominiovec.InstantaneaAutorizacion
	autoridad.preparar = func(semilla dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
		preparada = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
		return preparada, nil
	}
	autoridad.publicar = func(recibida dominiovec.InstantaneaAutorizacion) error {
		if recibida.AsignacionPerfil.Referencia() != preparada.AsignacionPerfil.Referencia() ||
			recibida.AsignacionPerfil.Version != 1 {
			t.Fatalf("la publicación no recibió la semilla v1: %+v", recibida.AsignacionPerfil)
		}
		// Modela que otra transacción instaló una asignación antes de la
		// publicación: la autoridad durable debe rechazar la semilla v1.
		return errors.New("asignacion actual cambio")
	}
	if err := politica.PublicarInicial(context.Background()); !errors.Is(err, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible) || politica.publicada || autoridad.publicadas != 1 {
		t.Fatalf("carrera lógica aceptada: err=%v politica=%+v autoridad=%+v", err, politica, autoridad)
	}
}

func TestPoliticaBorradorBolsaEvolucionaSoloDesdeBBackV1Exacta(t *testing.T) {
	politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	autoridad.permitirSucesion = true
	autoridad.preparar = func(i dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
		preparada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
		preparada.AsignacionPerfil.Version = 2
		return preparada, nil
	}
	if err := politica.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	if autoridad.publicada.VersionRol.Version != 2 || autoridad.publicada.AsignacionPerfil.Version != 2 || len(autoridad.publicada.VersionRol.Concesiones) != 3 {
		t.Fatalf("sucesión B2 no exacta: %+v", autoridad.publicada)
	}
}

func TestPoliticaBorradorBolsaPublicaSemillaIntacta(t *testing.T) {
	politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	var semilla dominiovec.InstantaneaAutorizacion
	autoridad.preparar = func(entrada dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
		semilla = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(entrada)
		return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(entrada), nil
	}
	if err := politica.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(autoridad.publicada, semilla) {
		t.Fatalf("la autoridad no recibió la semilla íntegra: recibida=%+v semilla=%+v", autoridad.publicada, semilla)
	}
}

func TestPoliticaBorradorBolsaValidaSoloMotivosCerradosVigentes(t *testing.T) {
	politica, _, _, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	if err := politica.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for _, motivo := range []dominiovec.ReferenciaEntradaCatalogo{motivoCrearBorradorLlamamientoBolsaDesarrollo(), motivoConsultarBorradorLlamamientoBolsaDesarrollo()} {
		if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, ahora); err != nil {
			t.Fatalf("motivo B-BACK válido rechazado: %v", err)
		}
	}
	if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivoAltaContratacionTemporalDesarrolloReferenciaPrueba(), ahora); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("motivo CT cruzado=%v", err)
	}
	ajeno := motivoCrearBorradorLlamamientoBolsaDesarrollo()
	ajeno.EntradaClave = referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-bback-ajeno")
	if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), ajeno, ahora); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("motivo arbitrario=%v", err)
	}
	if err := politica.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivoCrearBorradorLlamamientoBolsaDesarrollo(), ahora.In(time.FixedZone("ajena", 3600))); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("instante no UTC=%v", err)
	}
}

func TestPoliticaBorradorBolsaDelegaRegistros(t *testing.T) {
	politica, soporte, _, registro := nuevaPoliticaBorradorBolsaPrueba(t)
	if _, err := politica.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}); !errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		t.Fatalf("orden vacía antes de publicar=%v", err)
	}
	if err := politica.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := politica.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}); !errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		t.Fatalf("orden vacía publicada=%v", err)
	}
	orden, _ := ordenBorradorLlamamientoPoliticaPrueba(t, politica, soporte, puertosbolsa.AccionCrearBorradorLlamamientoInterno, motivoCrearBorradorLlamamientoBolsaDesarrollo(), false)
	if _, err := politica.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), orden); err != nil {
		t.Fatal(err)
	}
	if registro.concesiones != 1 {
		t.Fatalf("concesión nominal no delegada: %+v", registro)
	}
	cruzada, _ := ordenBorradorLlamamientoPoliticaPrueba(t, politica, soporte, puertosbolsa.AccionCrearBorradorLlamamientoInterno, motivoConsultarBorradorLlamamientoBolsaDesarrollo(), true)
	if _, err := politica.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), cruzada); !errors.Is(err, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		t.Fatalf("motivo cruzado aceptado=%v", err)
	}
	if registro.concesiones != 1 {
		t.Fatalf("motivo cruzado llegó al registro: %+v", registro)
	}
}

type validadorMotivoBorradorPoliticaPrueba struct{}

func (validadorMotivoBorradorPoliticaPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}

func ordenBorradorLlamamientoPoliticaPrueba(
	t *testing.T,
	politica *politicaBorradorLlamamientoBolsaDesarrollo,
	soporte *soporteSesionBorradorBolsaDesarrollo,
	accion string,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	permitirMotivoCruzado bool,
) (puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3) {
	t.Helper()
	if !permitirMotivoCruzado && !motivoBorradorLlamamientoCorresponde(accion, motivo) {
		t.Fatal("el auxiliar requiere un motivo correspondiente")
	}
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	finalidad := puertosbolsa.FinalidadCrearBorradorLlamamientoInterno
	if accion == puertosbolsa.AccionConsultarBorradorLlamamientoInterno {
		finalidad = puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: soporte.soporteCanal.contexto.Vinculo, ReferenciaMotivo: motivo,
		Accion:    accion,
		Recurso:   dominiovec.RecursoAutorizable{Referencia: "borrador-llamamiento:alta:" + strings.Repeat("a", 64), ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoBorradorLlamamiento, Ambitos: map[string]string{"unidad_ref": soporte.unidadRef, "ambito_ref": soporte.ambitoRef}},
		Finalidad: finalidad, Correlacion: correlacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	validador := puertosvec.ValidadorReferenciaMotivoAutorizacionV2(politica)
	if permitirMotivoCruzado {
		validador = validadorMotivoBorradorPoliticaPrueba{}
	}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(politica, politica, politica, validador, relojContratacionTemporalDesarrollo{}, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	decision, orden, err := servicio.PrepararSolicitudLigadaV3(context.Background(), solicitud, soporte.soporteCanal.contexto.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	if datos.PrincipalID == "" || decision.ValidarPara(solicitud) != nil {
		t.Fatal("la orden de prueba no quedó ligada al contexto Bolsa")
	}
	return orden, decision
}

func motivoAltaContratacionTemporalDesarrolloReferenciaPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "crear-solicitud")}
}
