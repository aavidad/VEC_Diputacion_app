package calculoexperienciaoficial

import (
	"context"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorSesionPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorSesionPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

func sesionPrueba(
	t *testing.T,
	ahora time.Time,
	superficie dominiovec.SuperficieAutenticacionActorV1,
	garantia dominiovec.AuthAssurance,
) (dominiovec.ContextoActor, dominiovec.VinculoAutenticacionActorV1) {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate, Garantia: garantia,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl",
		PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor := debePrueba(dominiovec.NuevoContextoActor(
		cuenta, instantanea, ahora.Add(-2*time.Minute),
	))
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie: superficie, MetodoObservado: dominiovec.AuthMethodCertificate,
		GarantiaObservada:            garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-5 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-4 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(30 * time.Minute),
	}
	vinculo := debePrueba(dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorSesionPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		}, actor, ahora,
	))
	return actor, vinculo
}

type generadorCorrelacionPrueba struct{ valor string }

func (g generadorCorrelacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func correlacionPrueba(t *testing.T, sufijo string) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	return debePrueba(dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionPrueba{
			valor: "correlacion_0123456789abcdef0123456789abcde" + sufijo,
		},
	))
}
