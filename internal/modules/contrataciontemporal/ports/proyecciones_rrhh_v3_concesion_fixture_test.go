package ports_test

import (
	"context"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func concesionConsultaRRHHPrueba(
	t *testing.T,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
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
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{
			Clave: clave, Valores: []string{valor},
		})
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh", Version: 1, Nombre: "Técnico de RRHH",
		Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: []dominiovec.ConcesionRol{{
			Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:    datos.Recurso.Tipo,
			Finalidades:    []string{datos.Finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  instante.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: dominiovec.AsignacionPerfil{
			AsignacionID: "asig-contratacion-temporal", Version: 1,
			PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID:     vinculo.PrincipalID, VersionRolRef: version.Referencia(),
			Estado: dominiovec.EstadoAsignacionPerfilActiva, Ambitos: ambitos,
			VigenteDesde: instante.Add(-time.Hour),
			VigenteHasta: instante.Add(time.Hour),
			EmitidaPor:   "administrador-identidades",
			EmitidaEn:    instante.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea,
		"dec_0123456789abcdef0123456789abcdef",
		instante, instante.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(
		solicitud, evidencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden, err :=
		puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
			solicitud, decision, motivo, resultado,
		)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err :=
		puertosvec.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroConcesionRRHHPrueba{
				instante: instante.Add(500 * time.Microsecond),
			},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	return decision, confirmacion
}
