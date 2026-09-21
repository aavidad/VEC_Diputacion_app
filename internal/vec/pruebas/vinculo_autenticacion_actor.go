// Package pruebas contiene fabricas exclusivas para dobles automatizados. No
// forma parte de ninguna composicion productiva ni ofrece un modo degradado.
package pruebas

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

type resolutorContextoActorRegistradoV2 struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoActorRegistradoV2) ResolverContextoActorRegistradoV2(
	ctx context.Context,
	solicitud domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil ||
		solicitud.Cuenta.CuentaRef != r.resultado.Contexto.Instantanea.CuentaRef ||
		solicitud.PerfilActivoRef != r.resultado.Contexto.PerfilActivoRef {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.ErrVinculoAutenticacionActorV2Invalido
	}
	return r.resultado.Clonar()
}

type relojFijoVinculoAutenticacionActorV2 struct{ instante time.Time }

func (r relojFijoVinculoAutenticacionActorV2) Ahora() time.Time { return r.instante }

// NuevoContextoRegistradoYVinculoV2 crea el par de prueba V2 a partir de la
// fabrica V1. El vinculo se obtiene exclusivamente mediante la fabrica sellada
// V2 y el resultado contiene una procedencia acreditada verificable.
func NuevoContextoRegistradoYVinculoV2(
	instante time.Time,
	personaRef string,
	perfilRef string,
	metodo domain.AuthMethod,
	garantia domain.AuthAssurance,
) (domain.ResultadoContextoActorRegistradoV2, domain.VinculoAutenticacionActorV2, error) {
	instante = instante.UTC().Truncate(time.Microsecond)
	actor, _, err := NuevoContextoYVinculo(instante, personaRef, perfilRef, metodo, garantia)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.VinculoAutenticacionActorV2{}, err
	}
	actor.Instantanea.CuentaVersion = 1
	resultado, err := nuevoResultadoContextoActorRegistradoV2Prueba(actor)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.VinculoAutenticacionActorV2{}, err
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          referenciaVinculoAutenticacionActorV2Prueba("aut_", "autenticacion"),
		AutenticacionHuellaSHA256: huellaVinculoAutenticacionActorV2Prueba("autenticacion"),
		AsercionRef:               referenciaVinculoAutenticacionActorV2Prueba("ase_", "asercion"),
		SesionRef:                 referenciaVinculoAutenticacionActorV2Prueba("ses_", "sesion"),
		ControlSesionRef:          referenciaVinculoAutenticacionActorV2Prueba("cse_", "control-sesion"),
		ControlSesionRevision:     1,
		ControlSesionHuellaSHA256: huellaVinculoAutenticacionActorV2Prueba("control-sesion"),
		CuentaRef:                 actor.Instantanea.CuentaRef,
		CuentaOrdinariaRef:        actor.Instantanea.CuentaRef,
		Superficie:                domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:           metodo,
		GarantiaObservada:         garantia,
		PoliticaGarantiaRef:       referenciaVinculoAutenticacionActorV2Prueba("pga_", "politica-garantia"),
		PoliticaGarantiaHuellaSHA256: huellaVinculoAutenticacionActorV2Prueba(
			"politica-garantia",
		),
		AutenticacionVerificadaEn: instante.Add(-5 * time.Minute),
		SesionEmitidaEn:           instante.Add(-4 * time.Minute),
		SesionRevalidadaEn:        instante.Add(-3 * time.Minute),
		SesionValidaHasta:         instante.Add(10 * time.Minute),
	}
	vinculo, resultadoLigado, err := domain.CrearVinculoAutenticacionActorV2ConResultado(
		context.Background(),
		revalidadorAutenticacionActor{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorContextoActorRegistradoV2{resultado: resultado},
		domain.SolicitudContextoActor{
			Cuenta: domain.CuentaAutenticadaContextoActor{
				CuentaRef: actor.Instantanea.CuentaRef,
				Metodo:    metodo,
				Garantia:  garantia,
			},
			PerfilActivoRef: actor.PerfilActivoRef,
		},
		relojFijoVinculoAutenticacionActorV2{instante: instante},
	)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, domain.VinculoAutenticacionActorV2{}, err
	}
	return resultadoLigado, vinculo, nil
}

func nuevoResultadoContextoActorRegistradoV2Prueba(
	actor domain.ContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          referenciaVinculoAutenticacionActorV2Prueba("prc_", "procedencia"),
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: huellaVinculoAutenticacionActorV2Prueba("procedencia"),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: actor.Instantanea.CuentaRef, Version: actor.Instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: actor.PersonaRef, Version: actor.Instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: actor.PerfilActivoRef, Version: actor.Instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: actor.Instantanea.VinculoRef, Version: actor.Instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []domain.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	manifiestoHuella, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanon)
	if err != nil {
		return domain.ResultadoContextoActorRegistradoV2{}, err
	}
	return domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef:               referenciaVinculoAutenticacionActorV2Prueba("rca_", "registro-contexto"),
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}, nil
}

func referenciaVinculoAutenticacionActorV2Prueba(prefijo, material string) string {
	suma := sha256.Sum256([]byte("vec.pruebas.vinculo-autenticacion-actor.v2\\x00" + material))
	return prefijo + hex.EncodeToString(suma[:16])
}

func huellaVinculoAutenticacionActorV2Prueba(material string) string {
	suma := sha256.Sum256([]byte("vec.pruebas.vinculo-autenticacion-actor.v2\\x00" + material))
	return hex.EncodeToString(suma[:])
}
