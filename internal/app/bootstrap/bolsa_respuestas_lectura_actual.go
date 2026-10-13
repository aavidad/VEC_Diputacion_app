package bootstrap

import (
	"context"
	"slices"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	accionLecturaRespuestasLlamamiento    = "bolsa.rrhh.respuestas_llamamiento.consultar"
	finalidadLecturaRespuestasLlamamiento = "seguimiento_contacto_rrhh"
)

var camposLecturaRespuestasLlamamiento = []string{
	"llamamiento_ref", "respuestas[].participacion_ref", "respuestas[].respuesta",
}

type contratoLecturaBolsa struct {
	accion, finalidad, tipo, motivo string
	campos                          []string
}

func contratoRespuestasLlamamiento() contratoLecturaBolsa {
	return contratoLecturaBolsa{accionLecturaRespuestasLlamamiento, finalidadLecturaRespuestasLlamamiento, "llamamiento_bolsa", "lectura_respuestas_llamamiento_correcta", camposLecturaRespuestasLlamamiento}
}

func concesionLecturaRespuestasLlamamientoExacta(i vecdomain.InstantaneaAutorizacion) bool {
	return concesionLecturaBolsaExacta(i, contratoRespuestasLlamamiento())
}

func concesionLecturaBolsaExacta(i vecdomain.InstantaneaAutorizacion, contrato contratoLecturaBolsa) bool {
	if i.VersionRol.RolID != "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo" {
		return false
	}
	coincidencias := 0
	for _, c := range i.VersionRol.Concesiones {
		if c.Accion != contrato.accion {
			continue
		}
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		if c.ModuloID != "bolsa" || c.TipoRecurso != contrato.tipo ||
			!slices.Equal(c.Finalidades, []string{contrato.finalidad}) ||
			!slices.Equal(campos, contrato.campos) ||
			c.GarantiaMinima != vecdomain.AuthAssuranceHigh ||
			(len(c.Obligaciones) != 0 && !slices.Equal(c.Obligaciones, []string{"auditar"})) {
			return false
		}
		coincidencias++
	}
	return coincidencias == 1
}

// CapturarLecturaRespuestasLlamamiento evalúa una concesión actual para el
// vínculo real de la petición. Unidad y ámbito proceden del soporte interno;
// bolsaRef identifica el dato consultado y no amplía la asignación. La lectura
// de Bolsa vuelve a cotejar la captura dentro de su transacción.
func CapturarLecturaRespuestasLlamamiento(
	ctx context.Context, fuente vecports.FuenteAutorizacion,
	vinculo vecdomain.VinculoAutenticacionActorV2,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	unidadRef, ambitoRef, bolsaRef, llamamientoRef string,
) (vecports.CapturaLecturaNominalRespuestasLlamamiento, error) {
	return capturarLecturaBolsa(ctx, fuente, vinculo, resultado, unidadRef, ambitoRef, bolsaRef, llamamientoRef, contratoRespuestasLlamamiento())
}

func capturarLecturaBolsa(ctx context.Context, fuente vecports.FuenteAutorizacion,
	vinculo vecdomain.VinculoAutenticacionActorV2, resultado vecdomain.ResultadoContextoActorRegistradoV2,
	unidadRef, ambitoRef, bolsaRef, recursoRef string, contrato contratoLecturaBolsa,
) (vecports.CapturaLecturaNominalBolsa, error) {
	vacia := vecports.CapturaLecturaNominalRespuestasLlamamiento{}
	if ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(fuente) ||
		resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		unidadRef == "" || ambitoRef == "" || bolsaRef == "" || recursoRef == "" {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	recurso := vecdomain.RecursoAutorizable{
		Referencia: recursoRef, ModuloID: "bolsa", Tipo: contrato.tipo,
		Ambitos:   map[string]string{"unidad_ref": unidadRef, "ambito_ref": ambitoRef},
		Atributos: map[string]string{contrato.tipo + "_ref": recursoRef},
	}
	if contrato.tipo == "llamamiento_bolsa" {
		recurso.Atributos = map[string]string{"llamamiento_ref": recursoRef}
	}
	if recurso.Validar() != nil {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if !vinculo.VigenteEn(ahora, resultado) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	instantanea, err := fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || ctx.Err() != nil || instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		!concesionLecturaBolsaExacta(instantanea, contrato) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	ahora = time.Now().UTC().Truncate(time.Microsecond)
	evaluacion, err := vecdomain.EvaluarLecturaActualAutorizacionV3(vinculo, resultado,
		instantanea, contrato.accion, recurso,
		contrato.finalidad, contrato.campos, ahora)
	if err != nil || !evaluacion.Permitido ||
		!slices.Equal(evaluacion.CamposPermitidos, contrato.campos) ||
		(len(evaluacion.Obligaciones) != 0 && !slices.Equal(evaluacion.Obligaciones, []string{"auditar"})) {
		return vacia, vecdomain.ErrAutorizacionDenegada
	}
	captura, err := componerCapturaLecturaActual(ctx, vinculo, resultado, evaluacion,
		contrato.accion, "bolsa", contrato.finalidad, contrato.motivo, recursoRef,
		map[string]any{"unidad_ref": unidadRef, "ambito_ref": ambitoRef, "bolsa_ref": bolsaRef})
	if err != nil {
		return vacia, err
	}
	return vecports.CapturaLecturaNominalRespuestasLlamamiento{
		ContextoCanonico: captura.contexto, VinculoCanonico: captura.vinculo, Orden: captura.orden,
		BolsaRef: bolsaRef, RecursoRef: recursoRef, LlamamientoRef: recursoRef, ValidaHasta: captura.hasta,
	}, nil
}
