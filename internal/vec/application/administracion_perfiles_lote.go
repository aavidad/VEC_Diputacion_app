package application

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// validarAdministrador exige el rol nominal existente y su categoría central, nunca su
// nombre visible ni Principal.Roles. El puerto durable revalida bajo bloqueo.
func (s *ServicioAdministracionPerfiles) validarAdministrador(ctx context.Context, instantanea domain.InstantaneaAutorizacion) error {
	if s == nil || s.catalogo == nil || s.reloj == nil || instantanea.Validar() != nil ||
		instantanea.VersionRol.RolID != "administracion_perfiles" ||
		!instantanea.AsignacionPerfil.VigenteEn(s.reloj.Ahora()) ||
		instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	rol, err := s.catalogo.ResolverRolAdministrable(ctx, instantanea.VersionRol.Referencia())
	if err != nil {
		return err
	}
	huella, err := instantanea.VersionRol.HuellaSHA256()
	if err != nil || rol.ValidarEn(s.reloj.Ahora()) != nil ||
		rol.VersionRef != instantanea.VersionRol.Referencia() || rol.HuellaSHA256 != huella ||
		rol.Clase != domain.ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "aplicacion" {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	return nil
}

// AplicarLoteOrdinario valida toda la orden antes de llamar una sola vez a la
// extensión tipada. Una autoridad antigua sin lote deniega; no hay fallback.
func (s *ServicioAdministracionPerfiles) AplicarLoteOrdinario(ctx context.Context, solicitud domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	if s == nil || s.catalogo == nil || s.actos == nil || s.reloj == nil {
		return domain.ReciboLoteAdministracionPerfiles{}, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	autoridad, ok := s.actos.(ports.AutoridadLotesAdministracionPerfiles)
	if !ok {
		return domain.ReciboLoteAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := s.validarAdministrador(ctx, solicitud.InstantaneaAutorizacion); err != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, err
	}
	ahora := s.reloj.Ahora()
	for _, cambio := range solicitud.Cambios {
		// Un alta programada empieza en el futuro y ninguna alta puede terminar
		// ya: la autoridad confirmaría un efecto que el recibo no podría acreditar.
		if cambio.Operacion == domain.OperacionOtorgarPerfil &&
			(!cambio.Objetivo.VigenteHasta.After(ahora) ||
				cambio.InicioVigencia == domain.InicioVigenciaLoteProgramado && !cambio.Objetivo.VigenteDesde.After(ahora)) {
			return domain.ReciboLoteAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
		}
		// Las bajas no exigen que el perfil siga ofreciéndose; la autoridad
		// comprueba que su versión esté registrada como ordinaria.
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			continue
		}
		rol, err := s.catalogo.ResolverRolAdministrable(ctx, cambio.RolVersionRef)
		if err != nil {
			return domain.ReciboLoteAdministracionPerfiles{}, err
		}
		if rol.ValidarEn(s.reloj.Ahora()) != nil || rol.VersionRef != cambio.RolVersionRef ||
			rol.Clase != domain.ClaseControlPerfilOrdinario || (rol.UnidadRequerida && cambio.Objetivo.UnidadRef == "") ||
			(cambio.Operacion == domain.OperacionOtorgarPerfil &&
				(cambio.InicioVigencia == domain.InicioVigenciaLoteProgramado && cambio.Objetivo.VigenteDesde.Before(rol.VigenteDesde) ||
					cambio.Objetivo.VigenteHasta.After(rol.VigenteHasta))) {
			return domain.ReciboLoteAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
		}
	}
	if err := ctx.Err(); err != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, err
	}
	recibo, err := autoridad.AplicarLoteOrdinario(ctx, solicitud)
	if err != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, err
	}
	if recibo.ValidarPara(solicitud) != nil {
		return domain.ReciboLoteAdministracionPerfiles{}, domain.ErrActoAdministracionPerfilesInvalido
	}
	return recibo, nil
}
