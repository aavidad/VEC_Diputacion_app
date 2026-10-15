package bootstrap

import (
	"context"
	"errors"
	"time"

	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Solo prepara la propuesta de ámbito con la asignación nominal propia.
// La fachada documental consume V3 y coteja la relación real de Bolsa antes
// de devolver metadatos; el perfil no constituye una lectura de la renuncia.
func ambitoPropioJustificanteRRHH(ctx context.Context, fuente vecports.FuenteAutorizacion,
	actor vec.ResultadoContextoActorRegistradoV2, vinculo vec.VinculoAutenticacionActorV2) (unidad, ambito string, err error) {
	denegado := vec.ErrAutorizacionDenegada
	if ctx == nil || fuente == nil || actor.Validar() != nil || vinculo.ValidarPara(actor) != nil {
		return "", "", denegado
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.PrincipalID != actor.Contexto.PersonaRef || datos.PerfilActivoRef != actor.Contexto.PerfilActivoRef ||
		datos.Superficie != vec.SuperficieAutenticacionInternaCorporativaV1 || datos.CuentaPrivilegiada {
		return "", "", denegado
	}
	i, err := fuente.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, datos.PerfilActivoRef)
	if err != nil {
		return "", "", errors.Join(vecports.ErrFuenteAutorizacionNoDisponible, err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if i.Validar() != nil || i.AsignacionPerfil.PrincipalID != datos.PrincipalID || i.AsignacionPerfil.PerfilActivoRef != datos.PerfilActivoRef ||
		!i.AsignacionPerfil.VigenteEn(ahora) || !vinculo.VigenteEn(ahora, actor) ||
		i.ControlVigenciaVersionRol.Estado != vec.EstadoControlVigenciaVersionRolHabilitada ||
		i.VersionRol.Estado != vec.EstadoVersionRolPublicada || i.VersionRol.PublicadaEn.After(ahora) ||
		i.AsignacionPerfil.EmitidaPor == "identidad:desarrollo:no-autoritativa" ||
		i.VersionRol.PublicadaPor == "seguridad:desarrollo:no-autoritativa" ||
		i.ControlVigenciaVersionRol.ActualizadoPor == "seguridad:desarrollo:no-autoritativa" ||
		len(i.AsignacionPerfil.Ambitos) != 2 {
		return "", "", denegado
	}
	for _, a := range i.AsignacionPerfil.Ambitos {
		if len(a.Valores) != 1 {
			return "", "", denegado
		}
		switch a.Clave {
		case "unidad_ref":
			unidad = a.Valores[0]
		case "ambito_ref":
			ambito = a.Valores[0]
		default:
			return "", "", denegado
		}
	}
	if unidad == "" || ambito == "" {
		return "", "", denegado
	}
	return unidad, ambito, nil
}
