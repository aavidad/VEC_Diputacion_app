// Package pruebas contiene fabricas exclusivas para dobles automatizados. No
// forma parte de ninguna composicion productiva ni ofrece un modo degradado.
package pruebas

import (
	"context"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type revalidadorAutenticacionActor struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionActor) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

// NuevoContextoYVinculo crea una sesion autoritativa simulada y la cruza por
// la misma fabrica sellada usada por el dominio. Persona y perfil deben ser
// referencias opacas validas; la funcion no los corrige ni inventa defaults.
func NuevoContextoYVinculo(
	instante time.Time,
	personaRef string,
	perfilRef string,
	metodo domain.AuthMethod,
	garantia domain.AuthAssurance,
) (domain.ContextoActor, domain.VinculoAutenticacionActorV1, error) {
	instante = instante.UTC().Truncate(time.Microsecond)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: metodo, Garantia: garantia,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: personaRef, PersonaVersion: 3,
		PerfilActivoRef: perfilRef, PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		return domain.ContextoActor{}, domain.VinculoAutenticacionActorV1{}, err
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: metodo, GarantiaObservada: garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute),
		SesionEmitidaEn:              instante.Add(-4 * time.Minute),
		SesionRevalidadaEn:           instante.Add(-3 * time.Minute),
		SesionValidaHasta:            instante.Add(10 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorAutenticacionActor{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		actor, instante,
	)
	if err != nil {
		return domain.ContextoActor{}, domain.VinculoAutenticacionActorV1{}, err
	}
	return actor, vinculo, nil
}

// NuevoVinculoGenerico sirve para decisiones aisladas que no ejercitan el
// cruce con una solicitud. Sigue pasando por revalidacion y fabrica sellada.
func NuevoVinculoGenerico(instante time.Time) (domain.VinculoAutenticacionActorV1, error) {
	_, vinculo, err := NuevoContextoYVinculo(
		instante,
		"per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate,
		domain.AuthAssuranceHigh,
	)
	return vinculo, err
}
