package bootstrap

import (
	"bytes"
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type intentoOrganizacionC4Registrador interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}

type motivosIntentoOrganizacionC4 struct {
	Denegado, Invalido, NoDisponible vecdomain.ReferenciaEntradaCatalogo
}

type contextoOrganizacionC4 struct {
	unidad      personalports.ResultadoUnidadTrabajoPropiaCronos
	identidad   resultadoIdentidadOrganizacionC4
	correlacion string
}
type resultadoIdentidadOrganizacionC4 struct {
	resultado vecdomain.ResultadoContextoActorRegistradoV2
	vinculo   vecdomain.VinculoAutenticacionActorV2
}
type claveOrganizacionC4 struct{}

type lectorOrganizacionC4 struct {
	servicio  *personalapp.ServicioConsultaOrganizacionHistorica
	autoridad *autoridadCronosEmpleadoDesarrollo
}

func (l *lectorOrganizacionC4) ConsultarC4(ctx context.Context, s personaldomain.SolicitudConsultaOrganizacionHistorica,
	u personalports.ResultadoUnidadTrabajoPropiaCronos) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio personalports.ResultadoConsultaOrganizacionHistorica
	if l == nil || l.servicio == nil || l.autoridad == nil || ctx == nil || u.Cobertura != "completa" || u.Unidad == nil ||
		u.Evidencia.ReciboRef == "" || u.Evidencia.DecisionRef == "" || u.Evidencia.AuditoriaRef == "" ||
		s.Actor.Validar() != nil || s.Selector.Validar() != nil || s.Selector.UnidadClave != "" ||
		s.Selector.OrganismoRef != u.Unidad.OrganismoRef || s.Selector.VigenteEn.AntesDe(u.Unidad.Desde) ||
		(u.Unidad.HastaExclusivo != "" && !s.Selector.VigenteEn.AntesDe(u.Unidad.HastaExclusivo)) {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	id, err := (seguridadCronosEmpleadoDesarrollo{autoridad: l.autoridad}).ResolverIdentidadRegistradaCronos(ctx)
	if err != nil || id.Contexto.Validar() != nil || id.Vinculo.ValidarPara(id.Contexto) != nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	canon, err := s.Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canon, id.Contexto.RepresentacionCanonica) {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	empleados, err := s.Actor.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 || empleados[0] != u.Unidad.EmpleadoRef {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	ctx = context.WithValue(ctx, claveOrganizacionC4{}, contextoOrganizacionC4{unidad: u, identidad: resultadoIdentidadOrganizacionC4{id.Contexto, id.Vinculo}, correlacion: ref})
	return l.servicio.Consultar(ctx, s)
}

type proveedorOrganizacionC4 struct {
	emisor    emisorUnidadC4
	motivo    vecdomain.ReferenciaEntradaCatalogo
	autoridad *autoridadCronosEmpleadoDesarrollo
}

func (p proveedorOrganizacionC4) AutorizarConsultaOrganizacionHistorica(ctx context.Context, m personaldomain.MaterialConsultaOrganizacionHistorica) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	d, ok := ctx.Value(claveOrganizacionC4{}).(contextoOrganizacionC4)
	s := m.Solicitud()
	if !ok || p.emisor == nil || p.autoridad == nil || p.autoridad.reloj == nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(p.motivo) || d.unidad.Unidad == nil ||
		d.unidad.Cobertura != "completa" || s.Selector.Validar() != nil || s.Selector.UnidadClave != "" ||
		s.Selector.OrganismoRef != d.unidad.Unidad.OrganismoRef || d.identidad.resultado.Validar() != nil ||
		d.identidad.vinculo.ValidarPara(d.identidad.resultado) != nil || !d.identidad.vinculo.VigenteEn(p.autoridad.reloj.Ahora(), d.identidad.resultado) {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	canon, err := s.Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canon, d.identidad.resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	sol, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: d.identidad.vinculo, ReferenciaMotivo: p.motivo, Accion: personaldomain.AccionConsultaOrganizacionHistorica,
		Recurso: m.Recurso(), Finalidad: "consultar_organizacion_historica", Correlacion: correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, sol, d.identidad.resultado)
	if err != nil {
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return vacio, personaldomain.ErrConsultaOrganizacionHistoricaDenegada
		}
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	if decision.ValidarPara(sol) != nil || exportador == nil {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(sol, decision, confirmacion, d.identidad.resultado, p.motivo, material, personaldomain.AudienciaConsultaOrganizacionHistorica) {
		return vacio, personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	return material, nil
}

type registroIntentosOrganizacionC4 struct {
	registrador intentoOrganizacionC4Registrador
	motivos     motivosIntentoOrganizacionC4
	proceso     string
}

func (r registroIntentosOrganizacionC4) VerificarRegistroConsultaOrganizacionHistorica(ctx context.Context) error {
	if ctx == nil || r.registrador == nil || r.proceso == "" || r.registrador.PreflightIntentoAuditoria(ctx) != nil {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	if _, ok := ctx.Value(claveOrganizacionC4{}).(contextoOrganizacionC4); !ok {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}

func (r registroIntentosOrganizacionC4) RegistrarIntentoConsultaOrganizacionHistorica(ctx context.Context, in personalports.IntentoConsultaOrganizacionHistorica) error {
	if r.VerificarRegistroConsultaOrganizacionHistorica(ctx) != nil {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	d := ctx.Value(claveOrganizacionC4{}).(contextoOrganizacionC4)
	motivo, resultado := r.motivos.NoDisponible, vecdomain.ResultadoIntentoAuditoriaError
	switch in.Motivo {
	case "denegado":
		motivo, resultado = r.motivos.Denegado, vecdomain.ResultadoIntentoAuditoriaDenegado
	case "entrada_invalida":
		motivo = r.motivos.Invalido
	case "no_disponible":
	default:
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	datos := vecdomain.DatosIntentoAuditoria{Accion: personaldomain.AccionConsultaOrganizacionHistorica, ModuloID: "personal",
		RecursoRef: "personal:organizacion_historica:" + d.unidad.Unidad.OrganismoRef, FinalidadRef: "consultar_organizacion_historica",
		Resultado: resultado, Motivo: motivo, Proceso: r.proceso, Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1), CorrelacionRef: d.correlacion}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(ref, d.identidad.resultado, d.identidad.vinculo, datos)
	if err != nil {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	acuse, err := r.registrador.AppendIntentoAuditoria(ctx, orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return personaldomain.ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}

func nuevoLectorOrganizacionCalendarioC4(ctx context.Context, pool *pgxpool.Pool, a *autoridadCronosEmpleadoDesarrollo,
	emisor emisorUnidadC4, motivo vecdomain.ReferenciaEntradaCatalogo, registro intentoOrganizacionC4Registrador,
	motivos motivosIntentoOrganizacionC4) (lectorOrganizacionCalendarioC4, error) {
	if ctx == nil || pool == nil || a == nil || a.procesoIntentos == "" || emisor == nil || registro == nil ||
		registro.PreflightIntentoAuditoria(ctx) != nil || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	var permite bool
	if pool.QueryRow(ctx, `SELECT COALESCE(has_function_privilege(to_regprocedure('vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE'),false)
	 AND NOT has_table_privilege('vec_personal.org_nodo_historia','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')`).Scan(&permite) != nil || !permite {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	for _, m := range []vecdomain.ReferenciaEntradaCatalogo{motivos.Denegado, motivos.Invalido, motivos.NoDisponible} {
		if !vecdomain.ReferenciaMotivoAutorizacionV2Valida(m) {
			return nil, ErrMontajeCalendarioLaboralC4
		}
	}
	repo, err := personalpg.NuevoRepositorioOrganizacionHistoricaPostgreSQL(pool)
	if err != nil {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	svc, err := personalapp.NuevoServicioConsultaOrganizacionHistorica(proveedorOrganizacionC4{emisor, motivo, a}, repo, registroIntentosOrganizacionC4{registrador: registro, motivos: motivos, proceso: a.procesoIntentos})
	if err != nil {
		return nil, ErrMontajeCalendarioLaboralC4
	}
	return &lectorOrganizacionC4{servicio: svc, autoridad: a}, nil
}
