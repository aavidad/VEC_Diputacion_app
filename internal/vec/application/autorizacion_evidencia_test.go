package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func referenciaIdentidadAutorizacionPrueba(prefijo, semilla string) string {
	suma := sha256.Sum256([]byte(prefijo + "\x00" + semilla))
	return prefijo + hex.EncodeToString(suma[:16])
}

func personaAutorizacionPrueba(semilla string) string {
	return referenciaIdentidadAutorizacionPrueba("per_", semilla)
}

func perfilAutorizacionPrueba(semilla string) string {
	return referenciaIdentidadAutorizacionPrueba("prf_", semilla)
}

// completarDecisionAutorizacionPrueba solo existe en binarios de prueba. Los
// dobles antiguos construian concesiones mínimas; el PEP productivo exige ahora
// la misma evidencia reforzada que el servicio real y nunca la completa.
func completarDecisionAutorizacionPrueba(
	solicitud domain.SolicitudAutorizacion,
	decision domain.DecisionAutorizacion,
) domain.DecisionAutorizacion {
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
			panic("recurso invalido en doble de autorizacion: " + err.Error())
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
		huella, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(
			decision.PoliticasEvaluadasRefs,
			decision.PoliticasEvaluadasHuellasSHA256,
		)
		if err != nil {
			panic("catalogo invalido en doble de autorizacion: " + err.Error())
		}
		decision.CatalogoPoliticasHuellaSHA256 = huella
	}
	return decision
}

type revalidadorVinculoAutenticacionAplicacionPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorVinculoAutenticacionAplicacionPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

func contextoYVinculoAutenticacionAplicacionPrueba(
	instante time.Time,
) (domain.ContextoActor, domain.VinculoAutenticacionActorV1) {
	instante = instante.UTC().Truncate(time.Microsecond)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		panic("actor de prueba invalido: " + err.Error())
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute),
		SesionEmitidaEn:              instante.Add(-4 * time.Minute),
		SesionRevalidadaEn:           instante.Add(-3 * time.Minute),
		SesionValidaHasta:            instante.Add(10 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		actor, instante,
	)
	if err != nil {
		panic("vinculo de prueba invalido: " + err.Error())
	}
	return actor, vinculo
}
