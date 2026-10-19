package bootstrap

import (
	"context"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func instantaneaBolsaConfirmacionCompatible(publicada dominiovec.InstantaneaAutorizacion,
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	soporte *soporteSesionBorradorBolsaDesarrollo, ahora time.Time, versionBase int) bool {
	// Si B1 está publicado, conserva además su comprobación histórica. F171
	// también admite una versión gobernada sin B1.
	if slices.ContainsFunc(publicada.VersionRol.Concesiones, func(c dominiovec.ConcesionRol) bool {
		return c.Accion == puertosbolsa.AccionConfirmarCargaConvoca
	}) && !instantaneaBolsaCargaConvocaCompatible(publicada, vinculo, soporte, ahora, versionBase) {
		return false
	}
	return instantaneaBolsaGobernadaCompatible(publicada, vinculo, soporte, ahora,
		versionBase, concesionesConfirmacionDirectaExactas)
}

func contieneAccionConfirmacionDirecta(concesiones []dominiovec.ConcesionRol) bool {
	return slices.ContainsFunc(concesiones, func(c dominiovec.ConcesionRol) bool {
		return c.Accion == puertosbolsa.AccionConfirmarAceptacionDirecta
	})
}

func concesionesConfirmacionDirectaExactas(restantes []dominiovec.ConcesionRol) bool {
	if len(restantes) < 1 || len(restantes) > 13 {
		return false
	}
	confirmaciones, cargas, comprobaciones := 0, 0, 0
	var s6 []dominiovec.ConcesionRol
	vistas := make(map[string]bool, len(restantes))
	for _, c := range restantes {
		if vistas[c.Accion] {
			return false
		}
		vistas[c.Accion] = true
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
		case puertosbolsa.AccionComprobacionOfrecimiento:
			if !concesionComprobacionOfrecimientoExacta(c) {
				return false
			}
			comprobaciones++
		case puertosbolsa.AccionConsultarTransicionesSituacion,
			puertosbolsa.AccionPublicarTransicionesSituacion:
			s6 = append(s6, c)
		default:
			if !concesionB103CompatibleConConfirmacion(c) {
				return false
			}
		}
	}
	return confirmaciones == 1 && cargas <= 1 && comprobaciones <= 1 &&
		(len(s6) == 0 || concesionesAdicionalesS6Exactas(s6))
}

// La comprobación de F1-09 convive con la confirmación directa en el mismo
// rol con la forma exacta de su fuente gobernada, sin campos.
func concesionComprobacionOfrecimientoExacta(c dominiovec.ConcesionRol) bool {
	return c.ModuloID == "bolsa" && c.TipoRecurso == "participacion_bolsa" &&
		slices.Equal(c.Finalidades, []string{puertosbolsa.FinalidadComprobacionOfrecimiento}) &&
		len(c.CamposPermitidos) == 0 && slices.Equal(c.Obligaciones, []string{"auditar"}) &&
		c.GarantiaMinima == dominiovec.AuthAssuranceHigh && c.Validar() == nil
}

func concesionB103CompatibleConConfirmacion(c dominiovec.ConcesionRol) bool {
	if c.ModuloID != "bolsa" || c.GarantiaMinima != dominiovec.AuthAssuranceHigh || c.Validar() != nil {
		return false
	}
	switch c.Accion {
	case inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH,
		inscripcion.AccionConvocatoriasRRHH, inscripcion.AccionMotivosRRHH,
		inscripcion.AccionDecidir, inscripcion.AccionIncorporar:
		tipo, finalidad, ok := tipoFinalidadInscripcionBolsa(ClaveOperacionInscripcionBolsa{Accion: c.Accion, Canal: superficieInternaInscripcionBolsa})
		if !ok || c.TipoRecurso != tipo || !slices.Equal(c.Finalidades, []string{finalidad}) || len(c.Obligaciones) != 0 {
			return false
		}
		campos := slices.Clone(c.CamposPermitidos)
		slices.Sort(campos)
		return slices.Equal(campos, camposLecturaInscripcionBolsa(c.Accion, true))
	case accionLecturaRespuestasLlamamiento:
		return c.TipoRecurso == "llamamiento_bolsa" &&
			slices.Equal(c.Finalidades, []string{finalidadLecturaRespuestasLlamamiento}) &&
			slices.Equal(c.CamposPermitidos, camposLecturaRespuestasLlamamiento) &&
			slices.Equal(c.Obligaciones, []string{"auditar"})
	case dominiovec.AccionLecturaOrigenContactoBolsa:
		return c.TipoRecurso == "participacion_bolsa" &&
			slices.Equal(c.Finalidades, []string{"consulta_origen_datos_contacto_participacion"}) &&
			slices.Equal(c.CamposPermitidos, dominiovec.CamposLecturaOrigenContactoBolsa()) &&
			slices.Equal(c.Obligaciones, []string{"auditar"})
	default:
		return false
	}
}

// La falta de gobierno/material deja indisponible solo este acto. La copia
// privada de material CT conserva el resto de rutas ya compuestas.
func proveedorConfirmacionDirectaOpcional(ctx context.Context, gobierno *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
	catalogo catalogoMaterialAutorizacionComunDesarrollo,
) *proveedorMaterialAltaContratacionTemporalDesarrollo {
	proveedor, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno,
		material, reloj, catalogo, puertosbolsa.AudienciaConfirmarAceptacionDirecta)
	if err == nil {
		return proveedor
	}
	registrarFalloPostgreSQLContratacionTemporalDesarrollo("material_confirmacion_aceptacion", causaFalloPostgreSQLCTDesarrollo(err))
	return nil
}
