package bootstrap

import (
	"context"
	"slices"
	"time"

	dietascomp "vec-diputacion-granada/internal/modules/dietas/adapters/composicion"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var camposLiquidacionEconomica = []string{"constancia", "economia", "esquema", "snapshot_sha256"}

const motivoAuditoriaLecturaLiquidacionEconomica = "lectura_liquidacion_dietas_correcta"

type capturadorLecturaLiquidacion struct {
	identidad interface {
		ResolverIdentidadRegistradaBorrador(context.Context) (dietascomp.IdentidadRegistradaBorrador, error)
	}
	fuente vecports.FuenteAutorizacion
}

func (c *capturadorLecturaLiquidacion) CapturarLecturaLiquidacionEconomica(ctx context.Context, ref string) (dietasports.CapturaLecturaLiquidacionEconomica, error) {
	var cero dietasports.CapturaLecturaLiquidacionEconomica
	if c == nil || ctx == nil || c.identidad == nil || c.fuente == nil || !referenciaRutaComisionDietas.MatchString(ref) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	base, err := c.identidad.ResolverIdentidadRegistradaBorrador(ctx)
	if err != nil || base.Contexto.Validar() != nil || base.Vinculo.ValidarPara(base.Contexto) != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	datos, err := base.Vinculo.Datos()
	if err != nil || datos.Superficie != "interna_corporativa" || datos.CuentaPrivilegiada {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	instantanea, err := c.fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil || instantanea.Validar() != nil ||
		instantanea.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		instantanea.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		instantanea.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	ambitos := map[string]string{}
	for _, a := range instantanea.AsignacionPerfil.Ambitos {
		if len(a.Valores) != 1 ||
			(a.Clave != "persona_ref" && a.Clave != "unidad_ref" && a.Clave != "empleado_ref") ||
			ambitos[a.Clave] != "" {
			return cero, dietasports.ErrAccesoCircuitoDenegado
		}
		ambitos[a.Clave] = a.Valores[0]
	}
	if ambitos["persona_ref"] != base.Contexto.Contexto.PersonaRef || len(ambitos) < 1 || len(ambitos) > 3 {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	if empleado, exigeEmpleado := ambitos["empleado_ref"]; exigeEmpleado {
		var canonico string
		for _, vinculo := range base.Contexto.Contexto.Instantanea.Vinculos {
			if vinculo.Tipo != vecdomain.TipoReferenciaContextoActorEmpleado {
				continue
			}
			if canonico != "" || !vinculo.VigenteEn(time.Now().UTC().Truncate(time.Microsecond)) {
				return cero, dietasports.ErrAccesoCircuitoDenegado
			}
			canonico = vinculo.Referencia
		}
		if canonico == "" || empleado != canonico {
			return cero, dietasports.ErrAccesoCircuitoDenegado
		}
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: ref, ModuloID: dietasports.ModuloDietas, Tipo: "liquidacion_economica", Ambitos: ambitos}
	if recurso.Validar() != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	const accion = "dietas.liquidacion_economica.consultar"
	const finalidad = "consultar_liquidacion_economica_dietas"
	e, err := vecdomain.EvaluarLecturaActualAutorizacionV3(base.Vinculo, base.Contexto, instantanea,
		accion, recurso, finalidad, camposLiquidacionEconomica, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil || !e.Permitido || !slices.Equal(e.CamposPermitidos, camposLiquidacionEconomica) ||
		(len(e.Obligaciones) != 0 && !slices.Equal(e.Obligaciones, []string{"auditar"})) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	captura, err := componerCapturaLecturaActual(ctx, base.Vinculo, base.Contexto, e,
		accion, dietasports.ModuloDietas, finalidad, motivoAuditoriaLecturaLiquidacionEconomica, ref, nil)
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return dietasports.CapturaLecturaLiquidacionEconomica{ContextoCanonico: captura.contexto,
		VinculoCanonico: captura.vinculo, Orden: captura.orden, ValidaHasta: captura.hasta}, nil
}

var _ dietasports.CapturadorLecturaLiquidacionEconomica = (*capturadorLecturaLiquidacion)(nil)
