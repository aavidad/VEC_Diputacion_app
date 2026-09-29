package pruebas

import (
	"context"
	"sort"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DatosConcesionV3Prueba describe la concesión que se quiere obtener. El rol
// concede exactamente Accion sobre el módulo y tipo del recurso, con esa
// finalidad, esos campos y esas obligaciones; la asignación cubre los ámbitos
// del recurso.
type DatosConcesionV3Prueba struct {
	Instante        time.Time
	PersonaRef      string
	PerfilRef       string
	Accion          string
	AccionConcedida string // vacía: la misma que Accion
	Recurso         domain.RecursoAutorizable
	Finalidad       string
	Campos          []string
	Obligaciones    []string
	DecisionRef     string
}

// ConcesionV3Prueba es el trío que produce el PDP V3 tras registrar la
// decisión. Confirmacion es cero si la decisión no es una concesión.
type ConcesionV3Prueba struct {
	Solicitud    domain.SolicitudAutorizacionLigadaV3
	Decision     domain.DecisionAutorizacionLigadaV3
	Confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
}

type registroConcesionV3Prueba struct{ registradaEn time.Time }

func (r registroConcesionV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return r.registradaEn, nil
}

type generadorCorrelacionV3Prueba struct{ valor string }

func (g generadorCorrelacionV3Prueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return g.valor, nil
}

// NuevaConcesionV3Prueba evalúa una solicitud V3 real con una instantánea de
// autorización sintética y, si concede, la registra con un registro de
// prueba (un segundo después de emitirla) que NO persiste nada: sirve para
// pruebas unitarias de adaptadores y no acredita el registro durable, que se
// prueba con PostgreSQL real en la composición. Solo para pruebas.
func NuevaConcesionV3Prueba(d DatosConcesionV3Prueba) (ConcesionV3Prueba, error) {
	var cero ConcesionV3Prueba
	instante := d.Instante.UTC().Truncate(time.Microsecond)
	resultado, vinculo, err := NuevoContextoRegistradoYVinculoV2(instante, d.PersonaRef, d.PerfilRef,
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		return cero, err
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaVinculoAutenticacionActorV2Prueba("motivos"),
		EntradaClave:         "motivo_" + huellaVinculoAutenticacionActorV2Prueba("motivo-v3")[:32],
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(),
		generadorCorrelacionV3Prueba{valor: "correlacion_" + huellaVinculoAutenticacionActorV2Prueba(d.DecisionRef)[:32]})
	if err != nil {
		return cero, err
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(domain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: d.Accion,
		Recurso: d.Recurso, Finalidad: d.Finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return cero, err
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		return cero, err
	}
	concedida := d.AccionConcedida
	if concedida == "" {
		concedida = d.Accion
	}
	version := domain.VersionRol{
		RolID: "rol_prueba_v3", Version: 1, Nombre: "Rol de prueba V3",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion: concedida, ModuloID: d.Recurso.ModuloID, TipoRecurso: d.Recurso.Tipo,
			Finalidades: []string{d.Finalidad}, GarantiaMinima: domain.AuthAssuranceSubstantial,
			CamposPermitidos: append([]string(nil), d.Campos...), Obligaciones: append([]string(nil), d.Obligaciones...),
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: instante.Add(-24 * time.Hour),
	}
	ambitos := make([]domain.AmbitoPerfil, 0, len(d.Recurso.Ambitos))
	for clave, valor := range d.Recurso.Ambitos {
		ambitos = append(ambitos, domain.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	sort.Slice(ambitos, func(i, j int) bool { return ambitos[i].Clave < ambitos[j].Clave })
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return cero, err
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-prueba-v3", Version: 1, PerfilActivoRef: datosVinculo.PerfilActivoRef,
			PrincipalID: datosVinculo.PrincipalID, VersionRolRef: version.Referencia(),
			Estado: domain.EstadoAsignacionPerfilActiva, Ambitos: ambitos,
			VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: instante.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, d.DecisionRef, instante, instante.Add(90*time.Second))
	if err != nil {
		return cero, err
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		return cero, err
	}
	salida := ConcesionV3Prueba{Solicitud: solicitud, Decision: decision}
	if ok, _, err := decision.Resultado(); err != nil || !ok {
		return salida, err
	}
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, motivo, resultado)
	if err != nil {
		return cero, err
	}
	salida.Confirmacion, err = ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), registroConcesionV3Prueba{registradaEn: instante.Add(time.Second)}, orden)
	if err != nil {
		return cero, err
	}
	return salida, nil
}
