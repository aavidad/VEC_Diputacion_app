package bootstrap

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func instantaneaBolsaConfirmacionCompatible(publicada dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) bool {
	return instantaneaBolsaGobernadaCompatible(publicada, vinculo, soporte, ahora,
		versionBase, concesionesConfirmacionDirectaExactas)
}

func contieneAccionConfirmacionDirecta(concesiones []dominiovec.ConcesionRol) bool {
	return slices.ContainsFunc(concesiones, func(c dominiovec.ConcesionRol) bool {
		return c.Accion == puertosbolsa.AccionConfirmarAceptacionDirecta
	})
}

func concesionesConfirmacionDirectaExactas(restantes []dominiovec.ConcesionRol) bool {
	if len(restantes) < 1 || len(restantes) > 4 {
		return false
	}
	confirmaciones, cargas := 0, 0
	var s6 []dominiovec.ConcesionRol
	for _, c := range restantes {
		switch c.Accion {
		case puertosbolsa.AccionConfirmarAceptacionDirecta:
			if c.ModuloID != "bolsa" || c.TipoRecurso != puertosbolsa.TipoRecursoConfirmarAceptacionDirecta ||
				!slices.Equal(c.Finalidades, []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta}) ||
				len(c.CamposPermitidos) != 0 || len(c.Obligaciones) != 0 ||
				c.GarantiaMinima != dominiovec.AuthAssuranceHigh {
				return false
			}
			confirmaciones++
		case puertosbolsa.AccionConfirmarCargaConvoca:
			if !concesionCargaConvocaExacta([]dominiovec.ConcesionRol{c}) {
				return false
			}
			cargas++
		case puertosbolsa.AccionConsultarTransicionesSituacion,
			puertosbolsa.AccionPublicarTransicionesSituacion:
			s6 = append(s6, c)
		default:
			return false
		}
	}
	return confirmaciones == 1 && cargas <= 1 &&
		(len(s6) == 0 || concesionesAdicionalesS6Exactas(s6))
}

// La falta de gobierno/material deja indisponible solo este acto. La copia
// privada de material CT conserva el resto de rutas ya compuestas.
func proveedorConfirmacionDirectaOpcional(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
	previos []descriptorMaterialConsumidorV3Desarrollo,
) *proveedorMaterialAltaContratacionTemporalDesarrollo {
	descriptores := append([]descriptorMaterialConsumidorV3Desarrollo(nil), previos...)
	descriptores = append(descriptores, descriptorMaterialConfirmacionDirectaBolsaDesarrollo())
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err == nil {
		var proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo
		proveedor, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno,
			material, reloj, catalogo, puertosbolsa.AudienciaConfirmarAceptacionDirecta)
		if err == nil {
			return proveedor
		}
	}
	registrarFalloPostgreSQLContratacionTemporalDesarrollo("material_confirmacion_aceptacion", causaFalloPostgreSQLCTDesarrollo(err))
	return nil
}
