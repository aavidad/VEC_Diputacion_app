package bootstrap

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	vecapp "vec-diputacion-granada/internal/vec/application"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrOriginalFirmableCTMontajeNoDisponible = errors.New("bootstrap: montaje original firmable CT no disponible")

// La puerta SQL es independiente de las rutas: Documentos sigue sirviendo sus
// consultas anteriores aunque DOC13 o AD158 aún no estén instaladas.
type puertaOriginalFirmableCTDesarrollo interface {
	Verificar(context.Context) error
}

type preparacionOriginalFirmableCTDesarrollo struct {
	servicio *docapp.Servicio
	catalogo *conservacion.Catalogo
	puerta   puertaOriginalFirmableCTDesarrollo
}

type puertaSQLOriginalFirmableCTDesarrollo struct{ ejecutor *pgxpool.Pool }

// Verificar exige las dos fachadas DOC13 con EXECUTE exclusivo del rol nominal
// y el consumidor AD158 con ambas acciones. No escribe ni concede autoridad.
func (p puertaSQLOriginalFirmableCTDesarrollo) Verificar(ctx context.Context) error {
	if p.ejecutor == nil || ctx == nil || ctx.Err() != nil {
		return ErrOriginalFirmableCTMontajeNoDisponible
	}
	var ok bool
	err := p.ejecutor.QueryRow(ctx, `WITH firmas AS (
 SELECT to_regprocedure('vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AS reserva,
        to_regprocedure('vec_documentos.confirmar_original_firmable_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AS confirmacion,
        (SELECT p.oid FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_autorizacion_atestada_v3'
           AND p.proname='consumir_decision_mutacion_v3_interna'
           AND p.proargtypes='25 17 17 17 17 1700 1700 17 17 17 17'::oidvector) AS consumidor,
        (SELECT p.oid FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_autorizacion_atestada_v3'
           AND p.proname='consumir_operacion_documentos_replay_v3_atestada'
           AND p.proargtypes='17 17 17 17 1700 1700 17 17 17 17'::oidvector) AS consumidor_replay
), fachadas AS (
 SELECT p.oid, p.proowner, p.prosecdef, p.proacl
 FROM firmas f, pg_catalog.pg_proc p WHERE p.oid IN (f.reserva, f.confirmacion)
)
SELECT session_user=current_user
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_roles r
   JOIN pg_catalog.pg_auth_members m ON m.member=r.oid
   JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
   WHERE r.rolname=session_user AND g.rolname='vec_documentos_ejecutor'
     AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 AND (SELECT count(*)=2 AND bool_and(pg_catalog.pg_get_userbyid(proowner)='vec_documentos_propietario'
     AND prosecdef AND pg_catalog.has_function_privilege(current_user,oid,'EXECUTE')
     AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(proacl,pg_catalog.acldefault('f',proowner))) acl
                     WHERE acl.grantee NOT IN (proowner,'vec_documentos_ejecutor'::regrole)
                        OR acl.privilege_type<>'EXECUTE' OR acl.is_grantable)) FROM fachadas)
 AND (SELECT consumidor IS NOT NULL FROM firmas)
 AND EXISTS (SELECT 1 FROM firmas f JOIN pg_catalog.pg_proc p ON p.oid=f.consumidor
   WHERE pg_catalog.pg_get_userbyid(p.proowner)='vec_autorizacion_atestada_v3_propietario'
     AND p.prosecdef AND pg_catalog.strpos(p.prosrc,'documentos.original_firmable.reservar')>0
     AND pg_catalog.strpos(p.prosrc,'documentos.original_firmable.confirmar')>0)
 AND EXISTS (SELECT 1 FROM firmas f JOIN pg_catalog.pg_proc p ON p.oid=f.consumidor_replay
   WHERE pg_catalog.pg_get_userbyid(p.proowner)='vec_autorizacion_atestada_v3_propietario'
     AND p.prosecdef
     AND pg_catalog.length(p.prosrc)-pg_catalog.length(pg_catalog.replace(p.prosrc,'documentos.original_firmable.reservar',''))
         >=2*pg_catalog.length('documentos.original_firmable.reservar')
     AND pg_catalog.length(p.prosrc)-pg_catalog.length(pg_catalog.replace(p.prosrc,'documentos.original_firmable.confirmar',''))
         >=2*pg_catalog.length('documentos.original_firmable.confirmar')
     AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) acl
                 WHERE acl.grantee='vec_documentos_propietario'::regrole AND acl.privilege_type='EXECUTE' AND NOT acl.is_grantable)
     AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) acl
                     WHERE acl.grantee NOT IN (p.proowner,'vec_documentos_propietario'::regrole)
                        OR acl.privilege_type<>'EXECUTE' OR acl.is_grantable))
`).Scan(&ok)
	if err != nil || !ok {
		return ErrOriginalFirmableCTMontajeNoDisponible
	}
	return nil
}

// nuevoServicioOriginalFirmableCTDesarrollo monta solo puertos reales. Las
// autorizaciones se obtienen de la identidad CT ya verificada por otra
// frontera; una petición o un DTO CT no pueden aportar concesiones. Hasta que
// esa autoridad nominal exista, el llamador debe pasar nil y el montaje para.
func nuevoServicioOriginalFirmableCTDesarrollo(
	ctx context.Context,
	documentos *autoridadDocumentosDesarrollo,
	actual, historico ctports.ConsultorOriginalFirmableRRHH,
	render ctports.RenderizadorBorradorRRHH,
	autorizaciones almacenvec.AutorizacionesDocumentosOriginalCT,
) (*vecapp.ServicioOriginalFirmableCT, error) {
	if ctx == nil || ctx.Err() != nil || documentos == nil || documentos.originalCT == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(actual) || dependenciaEsNulaContratacionTemporalDesarrollo(historico) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(render) || dependenciaEsNulaContratacionTemporalDesarrollo(autorizaciones) {
		return nil, ErrOriginalFirmableCTMontajeNoDisponible
	}
	p := documentos.originalCT
	if p.servicio == nil || p.catalogo == nil || dependenciaEsNulaContratacionTemporalDesarrollo(p.puerta) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.servicio.Repositorio) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.servicio.Almacen) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.servicio.ContextosLectura) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.servicio.Reloj) ||
		p.servicio.Politicas != p.catalogo {
		return nil, ErrOriginalFirmableCTMontajeNoDisponible
	}
	if _, ok := p.servicio.Repositorio.(docports.RepositorioOriginalFirmable); !ok {
		return nil, ErrOriginalFirmableCTMontajeNoDisponible
	}
	if err := p.puerta.Verificar(ctx); err != nil {
		return nil, errors.Join(ErrOriginalFirmableCTMontajeNoDisponible, err)
	}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(p.catalogo)
	if err != nil {
		return nil, errors.Join(ErrOriginalFirmableCTMontajeNoDisponible, err)
	}
	fuente, err := ctapp.NuevaFuenteOriginalFirmableRRHH(actual, historico, render, tipos)
	if err != nil {
		return nil, errors.Join(ErrOriginalFirmableCTMontajeNoDisponible, err)
	}
	custodia, err := almacenvec.NuevaCustodiaDocumentosOriginalCT(p.servicio, autorizaciones,
		almacenvec.FuncionMapeoExpedienteOriginalCT(ctapp.ReferenciaExpedienteDocumentalFormalizacion), tipos)
	if err != nil {
		return nil, errors.Join(ErrOriginalFirmableCTMontajeNoDisponible, err)
	}
	servicio, err := vecapp.NuevoServicioOriginalFirmableCT(fuente, custodia, tipos)
	if err != nil {
		return nil, errors.Join(ErrOriginalFirmableCTMontajeNoDisponible, err)
	}
	return servicio, nil
}

var _ vecports.CustodiaOriginalFirmableCT = (*almacenvec.CustodiaDocumentosOriginalCT)(nil)
