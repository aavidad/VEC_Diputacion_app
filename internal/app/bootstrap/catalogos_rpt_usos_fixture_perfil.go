package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	vec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	actoAsignacionRPTUsosFixture     = "acto:rpt:usos:fixture:asignacion:v1"
	actoControlRPTUsosFixture        = "acto:rpt:usos:fixture:control-rol:v1"
	actoSesionRPTUsosFixture         = "acto:rpt:usos:fixture:sesion:v1"
	accionReservaRPTUsosFixture      = "vec.catalogos.categorias.reservar_uso"
	accionConfirmacionRPTUsosFixture = "vec.catalogos.categorias.confirmar_uso"
	accionCancelacionRPTUsosFixture  = "vec.catalogos.categorias.cancelar_uso"
	finalidadRPTUsosFixture          = "vincular_categoria_a_operacion"
)

func plantillaRPTUsosFixture(principal, perfil string, d ports.DescriptorCatalogoRPT, ahora time.Time) (vec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || (vec.RecursoAutorizable{Referencia: "fixture:rpt:perfil", ModuloID: d.ModuloID,
		Tipo: "uso_categoria", Ambitos: map[string]string{"catalogo_id": d.CatalogoID, "modulo_id": d.ModuloID, "consumidor": "contratacion_temporal"}}).Validar() != nil {
		return vec.InstantaneaAutorizacion{}, ErrSeguridadComunDesarrolloDenegada
	}
	rol := vec.VersionRol{RolID: "rpt_usos_fixture", Version: 1, Nombre: "usos_categorias",
		Estado: vec.EstadoVersionRolPublicada, PublicadaPor: "autoridad:rpt:usos:fixture", PublicadaEn: desde}
	for _, accion := range []string{accionReservaRPTUsosFixture, accionConfirmacionRPTUsosFixture, accionCancelacionRPTUsosFixture} {
		rol.Concesiones = append(rol.Concesiones, vec.ConcesionRol{Accion: accion, ModuloID: d.ModuloID, TipoRecurso: "uso_categoria",
			Finalidades: []string{finalidadRPTUsosFixture}, GarantiaMinima: vec.AuthAssuranceHigh,
			CamposPermitidos: []string{"recibo", "uso"}, Obligaciones: []string{}})
	}
	huella, err := vec.HuellaCatalogoPoliticasAutorizacion([]vec.PoliticaRestrictiva{})
	if err != nil {
		return vec.InstantaneaAutorizacion{}, err
	}
	i := vec.InstantaneaAutorizacion{VersionRol: rol,
		AsignacionPerfil: vec.AsignacionPerfil{AsignacionID: referenciaAltaContratacionTemporalDesarrollo("asg_", principal+"\x00"+perfil+"\x00rpt-usos-fixture"),
			Version: 1, PerfilActivoRef: perfil, PrincipalID: principal, VersionRolRef: rol.Referencia(), Estado: vec.EstadoAsignacionPerfilActiva,
			Ambitos:      []vec.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{d.CatalogoID}}, {Clave: "modulo_id", Valores: []string{d.ModuloID}}, {Clave: "consumidor", Valores: []string{"contratacion_temporal"}}},
			VigenteDesde: desde, VigenteHasta: hasta, EmitidaPor: rol.PublicadaPor, EmitidaEn: desde},
		ControlVigenciaVersionRol: vec.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: vec.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: desde},
		Politicas: []vec.PoliticaRestrictiva{}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	return i, i.Validar()
}

// La fuente sólo consume el perfil ya publicado. No prepara ni publica por
// petición, y cualquier asignación distinta o acto ajeno produce denegación.
type fuentePerfilRPTUsosFixture struct {
	pool      *pgxpool.Pool
	plantilla vec.InstantaneaAutorizacion
	reloj     ports.Reloj
}

func (f *fuentePerfilRPTUsosFixture) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (vec.InstantaneaAutorizacion, error) {
	if f == nil || f.pool == nil || dependenciaAutorizacionComunDesarrolloNula(f.reloj) || ctx == nil || ctx.Err() != nil ||
		principal != f.plantilla.AsignacionPerfil.PrincipalID || perfil != f.plantilla.AsignacionPerfil.PerfilActivoRef {
		return vec.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	p, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, f.pool, perfil)
	if err != nil || !encontrada {
		return vec.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	if _, exacta := instantaneaConsumible(p, f.plantilla, f.reloj.Ahora()); !exacta || p.actoControl != actoControlRPTUsosFixture || !origenOperativoPublicadoCTDesarrollo(p, actoAsignacionRPTUsosFixture, f.reloj.Ahora()) {
		return vec.InstantaneaAutorizacion{}, vec.ErrAutorizacionDenegada
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.instantanea), nil
}

// La aprobación inicial liga la plantilla; una sustitución liga la asignación
// vigente exacta. La autoridad común repite el CAS bajo su bloqueo.
func provisionarPerfilRPTUsosFixture(ctx context.Context, a autoridadPostgreSQLDesarrollo, plantilla vec.InstantaneaAutorizacion, preimagen string, ahora time.Time) error {
	p, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, a.pool, plantilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil || admitirPreimagenRPTUsosFixture(p, encontrada, plantilla, preimagen, ahora) != nil {
		return ErrSeguridadComunDesarrolloDenegada
	}
	if !encontrada {
		a.soloInicial = true
		objetivo, err := a.prepararInstantanea(ctx, plantilla, true)
		if err != nil {
			return ErrSeguridadComunDesarrolloDenegada
		}
		huella, err := objetivo.AsignacionPerfil.HuellaSHA256()
		if err != nil || huella != preimagen {
			return ErrSeguridadComunDesarrolloDenegada
		}
		return a.publicarInstantanea(ctx, objetivo)
	}
	if _, exacta := instantaneaConsumible(p, plantilla, ahora); exacta && origenOperativoPublicadoCTDesarrollo(p, actoAsignacionRPTUsosFixture, ahora) {
		return nil
	}
	objetivo, err := a.prepararInstantanea(ctx, plantilla, false)
	if err != nil {
		return ErrSeguridadComunDesarrolloDenegada
	}
	return a.publicarInstantaneaDesdePreimagen(ctx, objetivo, p.instantanea)
}

// La misma admisión se aplica antes de publicar gobierno/contexto y se repite
// al provisionar. La comprobación previa no sustituye el CAS transaccional.
func admitirPreimagenRPTUsosFixture(p instantaneaPublicadaDesarrollo, encontrada bool, plantilla vec.InstantaneaAutorizacion, preimagen string, ahora time.Time) error {
	if !encontrada {
		huella, err := plantilla.AsignacionPerfil.HuellaSHA256()
		if err != nil || huella != preimagen {
			return ErrSeguridadComunDesarrolloDenegada
		}
		return nil
	}
	if p.actoControl != actoControlRPTUsosFixture || !origenOperativoPublicadoCTDesarrollo(p, actoAsignacionRPTUsosFixture, ahora) {
		return ErrSeguridadComunDesarrolloDenegada
	}
	if _, exacta := instantaneaConsumible(p, plantilla, ahora); exacta {
		return nil
	}
	huella, err := p.instantanea.AsignacionPerfil.HuellaSHA256()
	if err != nil || huella != preimagen {
		return ErrSeguridadComunDesarrolloDenegada
	}
	return nil
}
