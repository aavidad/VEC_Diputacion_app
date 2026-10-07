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
	exigirCAS              bool
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
	preimagenValida := (preimagen.VersionRol.Version == 4 && len(preimagen.VersionRol.Concesiones) == 6) ||
		(preimagen.VersionRol.Version == 5 && len(preimagen.VersionRol.Concesiones) == 7) ||
		(preimagen.VersionRol.Version == 6 && len(preimagen.VersionRol.Concesiones) == 9)
	if !preimagenValida ||
		(instantanea.AsignacionPerfil.Version == 2 && !a.permitirSucesion) {
		return errors.New("preimagen no admitida")
	}
	if a.exigirCAS && a.publicada.VersionRol.Version > 0 && preimagen.VersionRol.Version != a.publicada.VersionRol.Version {
		return errors.New("preimagen distinta del rol durable")
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

func TestPoliticaBorradorBolsaPublicaSieteConcesionesNominales(t *testing.T) {
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
	if instantanea.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef || len(instantanea.VersionRol.Concesiones) != 7 {
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
		puertosbolsa.AccionRegistrarContactoParticipacion:      puertosbolsa.FinalidadRegistrarContactoParticipacion,
		puertosbolsa.AccionConsultarContactoParticipacion:      puertosbolsa.FinalidadConsultarContactoParticipacion,
		puertosbolsa.AccionRegistrarDatosContactoParticipacion: puertosbolsa.FinalidadRegistrarDatosContactoParticipacion,
		puertosbolsa.AccionEmitirLlamamiento:                   puertosbolsa.FinalidadEmitirLlamamiento,
	} {
		concesion, existe := concesiones[accion]
		tipo := puertosbolsa.TipoRecursoBorradorLlamamiento
		if accion == puertosbolsa.AccionCambiarSituacionParticipacion || accion == puertosbolsa.AccionRegistrarContactoParticipacion || accion == puertosbolsa.AccionConsultarContactoParticipacion || accion == puertosbolsa.AccionRegistrarDatosContactoParticipacion {
			tipo = puertosbolsa.TipoRecursoSituacionParticipacion
		}
		if accion == puertosbolsa.AccionEmitirLlamamiento {
			tipo = puertosbolsa.TipoRecursoEmision
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

func TestMotivoPoliticaOfertasSoloConVersionSeisActiva(t *testing.T) {
	for _, motivo := range []dominiovec.ReferenciaEntradaCatalogo{motivoPublicarPoliticaOfertasBolsaDesarrollo(), motivoConsultarPoliticaOfertasBolsaDesarrollo()} {
		politicaCinco, _, _, _ := nuevaPoliticaBorradorBolsaPrueba(t)
		if err := politicaCinco.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, time.Now().UTC().Truncate(time.Microsecond)); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
			t.Fatalf("motivo B47 admitido antes de publicar: %v", err)
		}
		if err := politicaCinco.PublicarInicial(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := politicaCinco.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, time.Now().UTC().Truncate(time.Microsecond)); !errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
			t.Fatalf("rol v5 admitió B47: %v", err)
		}
		politicaSeis, _, _, _ := nuevaPoliticaBorradorBolsaPrueba(t)
		if err := politicaSeis.PublicarInicialConPoliticaOfertas(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := politicaSeis.ValidarReferenciaMotivoAutorizacionV2(context.Background(), motivo, time.Now().UTC().Truncate(time.Microsecond)); err != nil {
			t.Fatalf("rol v6 denegó motivo B47 publicado: %v", err)
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

func TestPoliticaBorradorBolsaEvolucionaSoloDesdeB7Exacta(t *testing.T) {
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
	if autoridad.publicada.VersionRol.Version != 5 || autoridad.publicada.AsignacionPerfil.Version != 2 || len(autoridad.publicada.VersionRol.Concesiones) != 7 {
		t.Fatalf("sucesión B4 no exacta: %+v", autoridad.publicada)
	}
}

func TestPoliticaBorradorBolsaConsultaDocumentalSoloEnVersionProvisionada(t *testing.T) {
	politica, soporte, _, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	contar := func(version int) int {
		i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef, politica.reloj.Ahora(), version)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range i.VersionRol.Concesiones {
			if c.Accion == puertosbolsa.AccionConsultarSolicitudesDocumentalesRRHH {
				n++
			}
		}
		return n
	}
	if contar(5) != 0 || contar(9) != 1 || contar(10) != 1 || contar(11) != 1 || contar(12) != 1 {
		t.Fatal("la lectura documental no conserva su versión de perfil fijo")
	}
}

func TestPoliticaBorradorBolsaAmpliaB5AConcesionB47Exacta(t *testing.T) {
	politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
	autoridad.permitirSucesion = true
	if err := politica.PublicarInicialConPoliticaOfertas(context.Background()); err != nil {
		t.Fatal(err)
	}
	if autoridad.publicada.VersionRol.Version != 6 || len(autoridad.publicada.VersionRol.Concesiones) != 9 {
		t.Fatalf("B47 no evolucionó conservadoramente B5: %+v", autoridad.publicada.VersionRol)
	}
	encontradas := map[string]*dominiovec.ConcesionRol{}
	for i := range autoridad.publicada.VersionRol.Concesiones {
		concesion := &autoridad.publicada.VersionRol.Concesiones[i]
		if concesion.Accion == puertosbolsa.AccionPublicarPoliticaOfertas || concesion.Accion == puertosbolsa.AccionConsultarPoliticaOfertas {
			encontradas[concesion.Accion] = concesion
		}
	}
	for _, accion := range []string{puertosbolsa.AccionPublicarPoliticaOfertas, puertosbolsa.AccionConsultarPoliticaOfertas} {
		encontrada := encontradas[accion]
		finalidad := puertosbolsa.FinalidadPoliticaOfertas
		if accion == puertosbolsa.AccionConsultarPoliticaOfertas {
			finalidad = puertosbolsa.FinalidadConsultarPoliticaOfertas
		}
		if encontrada == nil || encontrada.ModuloID != "bolsa" || encontrada.TipoRecurso != "bolsa_constituida" ||
			!reflect.DeepEqual(encontrada.Finalidades, []string{finalidad}) {
			t.Fatalf("concesión %s distinta: %+v", accion, encontrada)
		}
		if accion == puertosbolsa.AccionConsultarPoliticaOfertas && !reflect.DeepEqual(encontrada.CamposPermitidos, []string{puertosbolsa.CampoConsultarPoliticaOfertas}) {
			t.Fatalf("consulta sin campo exacto: %+v", encontrada)
		}
	}
}

func TestPoliticaBorradorBolsaReincorporacionTieneRamasExactas(t *testing.T) {
	for _, caso := range []struct {
		ofertas, version, concesiones int
	}{
		{0, 7, 8},
		{1, 8, 10},
	} {
		politica, _, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
		if err := politica.PublicarInicialConReincorporacion(context.Background(), caso.ofertas == 1); err != nil {
			t.Fatal(err)
		}
		rol := autoridad.publicada.VersionRol
		if rol.Version != caso.version || len(rol.Concesiones) != caso.concesiones {
			t.Fatalf("rama %d: versión=%d concesiones=%d", caso.ofertas, rol.Version, len(rol.Concesiones))
		}
		var lectura, edicionOfertas bool
		for _, concesion := range rol.Concesiones {
			if concesion.Accion == puertosbolsa.AccionConsultarReincorporacionTitular {
				lectura = concesion.ModuloID == "bolsa" && concesion.TipoRecurso == puertosbolsa.TipoRecursoSituacionParticipacion &&
					reflect.DeepEqual(concesion.Finalidades, []string{puertosbolsa.FinalidadConsultarReincorporacionTitular}) &&
					reflect.DeepEqual(concesion.CamposPermitidos, []string{puertosbolsa.CampoConsultarReincorporacionTitular}) && len(concesion.Obligaciones) == 0
			}
			if concesion.Accion == puertosbolsa.AccionPublicarPoliticaOfertas {
				edicionOfertas = true
			}
		}
		if !lectura || edicionOfertas != (caso.ofertas == 1) {
			t.Fatalf("rama %d mezcló permisos: lectura=%t edición_ofertas=%t", caso.ofertas, lectura, edicionOfertas)
		}
	}
}

func TestPoliticaBorradorBolsaReincorporacionNoAlternaB47ConHistoria(t *testing.T) {
	for _, caso := range []struct {
		anterior, nueva int
		ofertas         bool
	}{
		{7, 8, true},
		{8, 7, false},
	} {
		politica, soporte, autoridad, _ := nuevaPoliticaBorradorBolsaPrueba(t)
		datos, err := soporte.soporteCanal.contexto.Vinculo.Datos()
		if err != nil {
			t.Fatal(err)
		}
		anterior, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, soporte.unidadRef, soporte.ambitoRef,
			politica.reloj.Ahora(), caso.anterior)
		if err != nil {
			t.Fatal(err)
		}
		autoridad.publicada = anterior
		autoridad.exigirCAS = true
		autoridad.permitirSucesion = true
		autoridad.preparar = func(i dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
			i.AsignacionPerfil.Version = 2
			return i, nil
		}
		if err := politica.PublicarInicialConReincorporacion(context.Background(), caso.ofertas); !errors.Is(err, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible) ||
			autoridad.publicadas != 0 || autoridad.publicada.VersionRol.Version != caso.anterior || politica.publicada {
			t.Fatalf("transición v%d→v%d alteró historia: err=%v autoridad=%+v", caso.anterior, caso.nueva, err, autoridad)
		}
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
