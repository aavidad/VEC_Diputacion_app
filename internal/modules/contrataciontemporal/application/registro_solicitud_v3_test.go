package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func selloHMACRegistroPrueba(referencia, digito string) string {
	return "hmac-sha256:" + referencia + ":" + strings.Repeat(digito, 64)
}

type revalidadorVinculoPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (d revalidadorVinculoPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return d.resultado, nil
}

type resolutorResultadoVinculoPrueba struct {
	resultado dominiovec.ResultadoContextoActorRegistradoV2
}

func (d resolutorResultadoVinculoPrueba) ResolverContextoActorRegistradoV2(
	context.Context,
	dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado, nil
}

type relojVinculoPrueba struct{ instante time.Time }

func (d relojVinculoPrueba) Ahora() time.Time { return d.instante }

func contextoAutorizacionAltaV3Prueba(
	t *testing.T,
	ahora time.Time,
) ports.ContextoAutorizacionAltaV3 {
	return contextoAutorizacionAltaV3PruebaConMarcas(t, ahora, "a", "a")
}

func contextoAutorizacionAltaV3PruebaConMarcas(
	t *testing.T,
	ahora time.Time,
	marcaActor string,
	marcaPerfil string,
) ports.ContextoAutorizacionAltaV3 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl" + marcaActor + marcaPerfil,
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl" + marcaActor,
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl" + marcaPerfil,
		PerfilVersion:   5,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := dominiovec.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := dominiovec.ManifiestoProcedenciaContextoActorV1{
		Esquema:           dominiovec.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: dominiovec.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: dominiovec.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: dominiovec.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: dominiovec.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: dominiovec.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []dominiovec.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err := dominiovec.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := dominiovec.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn" +
			marcaActor + marcaPerfil,
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: dominiovec.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_0123456789abcdefghijkl",
		SesionRef:                    "ses_0123456789abcdefghijkl",
		ControlSesionRef:             "cse_0123456789abcdefghijkl",
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorVinculoPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		resolutorResultadoVinculoPrueba{resultado: resultado},
		dominiovec.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		relojVinculoPrueba{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}
}

func concesionAutorizacionV3Prueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	ahora time.Time,
	referenciaDecision string,
	conceder bool,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	version := dominiovec.VersionRol{
		RolID:   "tecnico_rrhh",
		Version: 1,
		Nombre:  "Técnico de RRHH",
		Estado:  dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:    datos.Recurso.Tipo,
			Finalidades:    []string{datos.Finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  ahora.Add(-24 * time.Hour),
	}
	if !conceder {
		version.Concesiones[0].Accion =
			"contratacion_temporal.solicitud.denegada"
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID:    "asig-contratacion-temporal",
			Version:         1,
			PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID:     vinculo.PrincipalID,
			VersionRolRef:   version.Referencia(),
			Estado:          dominiovec.EstadoAsignacionPerfilActiva,
			Ambitos:         ambitos,
			VigenteDesde:    ahora.Add(-time.Hour),
			VigenteHasta:    ahora.Add(time.Hour),
			EmitidaPor:      "administrador-identidades",
			EmitidaEn:       ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef:  version.Referencia(),
			Revision:       1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud,
		instantanea,
		referenciaDecision,
		ahora,
		ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	if !conceder {
		return decision,
			puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{},
			nil
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		solicitud,
		decision,
		motivo,
		resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := puertosvec.
		RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroConcesionV3Doble{registradaEn: ahora},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion, nil
}
