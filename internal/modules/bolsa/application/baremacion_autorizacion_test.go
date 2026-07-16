package application

import (
	"context"
	"strings"
	"testing"
	"time"

	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

// Estos dobles viven junto al módulo para que sus pruebas no dependan de
// símbolos privados del paquete de aplicación del núcleo. La dependencia real
// sigue siendo el puerto Autorizador; ServicioAutorizacion sólo se usa en la
// prueba de integración entre ambas fronteras.
type fuenteAutorizacionServicioPrueba struct {
	instantanea  dominiovec.InstantaneaAutorizacion
	err          error
	invocaciones int
}

func (f *fuenteAutorizacionServicioPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context,
	_, _ string,
) (dominiovec.InstantaneaAutorizacion, error) {
	f.invocaciones++
	return f.instantanea, f.err
}

type registroAutorizacionServicioPrueba struct {
	err          error
	invocaciones int
	concesiones  int
	denegaciones int
	decision     dominiovec.DecisionAutorizacion
}

func (r *registroAutorizacionServicioPrueba) RegistrarDecisionSiInstantaneaVigente(
	_ context.Context,
	decision dominiovec.DecisionAutorizacion,
) error {
	r.invocaciones++
	r.concesiones++
	r.decision = decision
	return r.err
}

func (r *registroAutorizacionServicioPrueba) RegistrarDenegacionAutorizacion(
	_ context.Context,
	decision dominiovec.DecisionAutorizacion,
) error {
	r.invocaciones++
	r.denegaciones++
	r.decision = decision
	return r.err
}

func (*registroAutorizacionServicioPrueba) ObtenerDecision(
	context.Context,
	string,
) (dominiovec.DecisionAutorizacion, error) {
	return dominiovec.DecisionAutorizacion{}, puertosvec.ErrDecisionAutorizacionNoEncontrada
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

func nuevoServicioAutorizacionServicioPrueba(
	t *testing.T,
	instantes puertosvec.FuenteAutorizacion,
	registro puertosvec.RegistroDecisionesAutorizacion,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
	ahora time.Time,
) *aplicacionvec.ServicioAutorizacion {
	t.Helper()
	servicio, err := aplicacionvec.NuevoServicioAutorizacion(
		instantes,
		registro,
		registro.(puertosvec.RegistroDenegacionesAutorizacion),
		&relojAutorizacionServicioPrueba{ahora: ahora},
		generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio de autorización real: %v", err)
	}
	return servicio
}

type revalidadorVinculoAutenticacionAplicacionPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorVinculoAutenticacionAplicacionPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

func contextoYVinculoAutenticacionAplicacionPrueba(
	instante time.Time,
) (dominiovec.ContextoActor, dominiovec.VinculoAutenticacionActorV1) {
	instante = instante.UTC().Truncate(time.Microsecond)
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: dominiovec.AuthMethodCertificate,
		Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		panic("actor de prueba inválido: " + err.Error())
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: cuenta.CuentaRef,
		CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:         dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:    cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute),
		SesionEmitidaEn:              instante.Add(-4 * time.Minute), SesionRevalidadaEn: instante.Add(-3 * time.Minute),
		SesionValidaHasta: instante.Add(10 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		actor,
		instante,
	)
	if err != nil {
		panic("vínculo de prueba inválido: " + err.Error())
	}
	return actor, vinculo
}

func sesionAutenticadaBaremacionIdentidadPrueba(
	t *testing.T,
	principalRef string,
	perfilRef string,
) SesionAutenticadaBaremacion {
	t.Helper()
	contextoActor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instanteBaremacionPrueba,
		principalRef,
		perfilRef,
		dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatalf("crear identidad alternativa de prueba: %v", err)
	}
	sesion, err := NuevaSesionAutenticadaBaremacion(contextoActor, vinculo)
	if err != nil {
		t.Fatalf("crear sesion alternativa de prueba: %v", err)
	}
	return sesion
}

func completarDecisionAutorizacionPrueba(
	solicitud dominiovec.SolicitudAutorizacion,
	decision dominiovec.DecisionAutorizacion,
) dominiovec.DecisionAutorizacion {
	if decision.VinculoAutenticacionActor.Validar() != nil {
		_, vinculo, err := pruebasvec.NuevoContextoYVinculo(
			decision.EmitidaEn,
			decision.PrincipalID,
			decision.PerfilActivoRef,
			solicitud.Principal.AuthMethod,
			solicitud.Principal.AuthAssurance,
		)
		if err == nil {
			decision.VinculoAutenticacionActor = vinculo
		}
	}
	if decision.AsignacionRef == "" {
		decision.AsignacionRef = "asignacion:doble-prueba:v1"
	}
	if decision.AsignacionHuellaSHA256 == "" {
		decision.AsignacionHuellaSHA256 = strings.Repeat("a", 64)
	}
	if decision.VersionRolRef == "" {
		decision.VersionRolRef = "rol:doble-prueba:v1"
	}
	if decision.VersionRolHuellaSHA256 == "" {
		decision.VersionRolHuellaSHA256 = strings.Repeat("b", 64)
	}
	if decision.ControlVigenciaVersionRolRef == "" {
		decision.ControlVigenciaVersionRolRef = decision.VersionRolRef
	}
	if decision.ControlVigenciaVersionRolRevision == 0 {
		decision.ControlVigenciaVersionRolRevision = 1
	}
	if decision.ControlVigenciaVersionRolHuellaSHA256 == "" {
		decision.ControlVigenciaVersionRolHuellaSHA256 = strings.Repeat("c", 64)
	}
	if decision.ModuloID == "" {
		decision.ModuloID = solicitud.Recurso.ModuloID
	}
	if decision.TipoRecurso == "" {
		decision.TipoRecurso = solicitud.Recurso.Tipo
	}
	if decision.ContextoRecursoHuellaSHA256 == "" {
		huella, err := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
		if err != nil {
			panic("recurso inválido en doble de autorización: " + err.Error())
		}
		decision.ContextoRecursoHuellaSHA256 = huella
	}
	if decision.PoliticasEvaluadasRefs == nil {
		decision.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasRefs...)
	}
	if decision.PoliticasEvaluadasHuellasSHA256 == nil {
		decision.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, len(decision.PoliticasHuellasSHA256))
		for referencia, huella := range decision.PoliticasHuellasSHA256 {
			decision.PoliticasEvaluadasHuellasSHA256[referencia] = huella
		}
	}
	if decision.RevisionCatalogoPoliticas == 0 {
		decision.RevisionCatalogoPoliticas = 1
	}
	if decision.CatalogoPoliticasHuellaSHA256 == "" {
		huella, err := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(
			decision.PoliticasEvaluadasRefs,
			decision.PoliticasEvaluadasHuellasSHA256,
		)
		if err != nil {
			panic("catálogo inválido en doble de autorización: " + err.Error())
		}
		decision.CatalogoPoliticasHuellaSHA256 = huella
	}
	return decision
}
