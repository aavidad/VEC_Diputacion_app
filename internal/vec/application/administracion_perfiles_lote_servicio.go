package application

import (
	"context"
	"slices"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const accionLoteOrdinario = "administracion.perfiles.aplicar_lote_ordinario"

// ServicioLotesAdministracionPerfiles coordina sólo el lote ordinario y su
// preparación. No ofrece actos singulares ni propuestas: el proceso que lo
// monta no tiene autoridad para ellos.
//
// La categoría «Administrador de Aplicación» del actor no se consulta aquí: el
// catálogo de roles administrables no la expone, y la comprueba la puerta del
// lote en PostgreSQL (AUT45/AUT51) antes y después de consumir la decisión
// (AD190). Este servicio exige lo que puede ver: rol de Aplicación publicado y
// habilitado, asignación vigente y la concesión exacta del lote en la versión.
type ServicioLotesAdministracionPerfiles struct {
	catalogo   ports.CatalogoRolesAdministrables
	lotes      ports.AutoridadLotesAdministracionPerfiles
	preparador ports.PreparadorLotesAdministracionPerfiles
	reloj      ports.Reloj
}

func NuevoServicioLotesAdministracionPerfiles(catalogo ports.CatalogoRolesAdministrables,
	lotes ports.AutoridadLotesAdministracionPerfiles, preparador ports.PreparadorLotesAdministracionPerfiles,
	reloj ports.Reloj) (*ServicioLotesAdministracionPerfiles, error) {
	if catalogo == nil || lotes == nil || preparador == nil || reloj == nil {
		return nil, ErrAdministracionPerfilesNoConfigurada
	}
	return &ServicioLotesAdministracionPerfiles{catalogo: catalogo, lotes: lotes, preparador: preparador, reloj: reloj}, nil
}

func (s *ServicioLotesAdministracionPerfiles) configurado() bool {
	return s != nil && s.catalogo != nil && s.lotes != nil && s.preparador != nil && s.reloj != nil
}

// validarAdministradorLote no fabrica la categoría: sólo descarta pronto lo que
// la autoridad durable rechazaría igualmente.
func (s *ServicioLotesAdministracionPerfiles) validarAdministradorLote(instantanea domain.InstantaneaAutorizacion) error {
	ahora := s.reloj.Ahora()
	if instantanea.Validar() != nil || instantanea.VersionRol.RolID != "administracion_perfiles" ||
		!domain.VersionRolAplicacionAdmitida(instantanea.VersionRol.Referencia()) ||
		instantanea.VersionRol.Estado != domain.EstadoVersionRolPublicada ||
		instantanea.ControlVigenciaVersionRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada ||
		!instantanea.AsignacionPerfil.VigenteEn(ahora) {
		return domain.ErrControlAdministracionPerfilesInvalido
	}
	for _, c := range instantanea.VersionRol.Concesiones {
		if c.Accion == accionLoteOrdinario && c.ModuloID == "administracion" && c.TipoRecurso == "persona" &&
			slices.Equal(c.Finalidades, []string{"gestion_perfiles"}) && c.GarantiaMinima == domain.AuthAssuranceHigh &&
			len(c.CamposPermitidos) == 0 && slices.Equal(c.Obligaciones, []string{"auditar"}) {
			return nil
		}
	}
	return domain.ErrControlAdministracionPerfilesInvalido
}

// AplicarLoteOrdinario valida la orden y coteja cada alta con el perfil
// registrado antes de llamar una sola vez a la autoridad del lote.
func (s *ServicioLotesAdministracionPerfiles) AplicarLoteOrdinario(ctx context.Context, solicitud domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error) {
	var vacio domain.ReciboLoteAdministracionPerfiles
	if !s.configurado() {
		return vacio, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	if err := s.validarAdministradorLote(solicitud.InstantaneaAutorizacion); err != nil {
		return vacio, err
	}
	ahora := s.reloj.Ahora()
	for _, cambio := range solicitud.Cambios {
		if cambio.Operacion == domain.OperacionOtorgarPerfil &&
			(!cambio.Objetivo.VigenteHasta.After(ahora) ||
				cambio.InicioVigencia == domain.InicioVigenciaLoteProgramado && !cambio.Objetivo.VigenteDesde.After(ahora)) {
			return vacio, domain.ErrActoAdministracionPerfilesInvalido
		}
		// Una baja no exige que el perfil siga ofreciéndose; la autoridad
		// comprueba que su versión esté registrada como ordinaria.
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			continue
		}
		rol, err := s.catalogo.ResolverRolAdministrable(ctx, cambio.RolVersionRef)
		if err != nil {
			return vacio, err
		}
		if rol.ValidarEn(ahora) != nil || rol.VersionRef != cambio.RolVersionRef ||
			rol.Clase != domain.ClaseControlPerfilOrdinario || (rol.UnidadRequerida && cambio.Objetivo.UnidadRef == "") ||
			cambio.InicioVigencia == domain.InicioVigenciaLoteProgramado && cambio.Objetivo.VigenteDesde.Before(rol.VigenteDesde) ||
			cambio.Objetivo.VigenteHasta.After(rol.VigenteHasta) {
			return vacio, domain.ErrActoAdministracionPerfilesInvalido
		}
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recibo, err := s.lotes.AplicarLoteOrdinario(ctx, solicitud)
	if err != nil {
		return vacio, err
	}
	if recibo.ValidarPara(solicitud) != nil {
		return vacio, domain.ErrActoAdministracionPerfilesInvalido
	}
	return recibo, nil
}

// PrepararLoteOrdinario no reserva nada: el lote vuelve a comparar todas las
// preimágenes cuando se aplica.
func (s *ServicioLotesAdministracionPerfiles) PrepararLoteOrdinario(ctx context.Context, solicitud domain.SolicitudPreparacionLoteAdministracionPerfiles) (domain.PreparacionLoteAdministracionPerfiles, error) {
	var vacia domain.PreparacionLoteAdministracionPerfiles
	if !s.configurado() {
		return vacia, ErrAdministracionPerfilesNoConfigurada
	}
	if ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		solicitud.Evidencia.ValidarEn(solicitud.Actor, s.reloj.Ahora()) != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	if err := s.validarAdministradorLote(solicitud.InstantaneaAutorizacion); err != nil {
		return vacia, err
	}
	p, err := s.preparador.PrepararLoteOrdinario(ctx, solicitud)
	if err != nil {
		return vacia, err
	}
	if p.ValidarPara(solicitud) != nil {
		return vacia, domain.ErrActoAdministracionPerfilesInvalido
	}
	return p, nil
}
