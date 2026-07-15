package ports

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	personaVinculoPuertoPrueba = "per_0123456789abcdefghijkl"
	perfilVinculoPuertoPrueba  = "prf_0123456789abcdefghijkl"
)

type revalidadorVinculoPuertoPrueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorVinculoPuertoPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

func vinculoAutenticacionActorPuertoPrueba(t *testing.T, instante time.Time) domain.VinculoAutenticacionActorV1 {
	t.Helper()
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: personaVinculoPuertoPrueba, PersonaVersion: 3,
		PerfilActivoRef: perfilVinculoPuertoPrueba, PerfilVersion: 4,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("crear actor de puerto: %v", err)
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
		context.Background(), revalidadorVinculoPuertoPrueba{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		actor, instante,
	)
	if err != nil {
		t.Fatalf("crear vinculo de puerto: %v", err)
	}
	return vinculo
}

var _ RevalidadorAutenticacionActorV1 = revalidadorVinculoPuertoPrueba{}
