package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const AccionResolucionFormalizacion = "contratacion_temporal.formalizacion.resolucion.manual_ejercicio.registrar"
const TipoRecursoResolucionFormalizacion = "resolucion_formalizacion_ct"

// La composición prepara el PDF original v7 y autoriza todo el material.
// El cliente HTTP nunca proporciona documento, organización o exportación V3.
type ProveedorResolucionFormalizacion interface {
	PrepararResolucionFormalizacion(context.Context, ports.SolicitudResolucionFormalizacion) (ports.MaterialResolucionFormalizacion, error)
	AutorizarResolucionFormalizacion(context.Context, ports.MaterialResolucionFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type RegistroResolucionFormalizacionPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorResolucionFormalizacion
}

func NuevoRegistroResolucionFormalizacionPostgreSQL(pool *pgxpool.Pool, p ProveedorResolucionFormalizacion) (*RegistroResolucionFormalizacionPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(p) {
		return nil, ports.ErrResolucionFormalizacionNoDisponible
	}
	return &RegistroResolucionFormalizacionPostgreSQL{pool, p}, nil
}
func RecursoResolucionFormalizacion(m ports.MaterialResolucionFormalizacion) (dominiovec.RecursoAutorizable, error) {
	if m.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	b, e := json.Marshal(m)
	if e != nil || len(b) > 16384 {
		return dominiovec.RecursoAutorizable{}, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{Referencia: m.Solicitud.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoResolucionFormalizacion,
		Ambitos: map[string]string{"organizacion_ref": m.OrganizacionRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}
func (r *RegistroResolucionFormalizacionPostgreSQL) RegistrarResolucionFormalizacion(ctx context.Context, s ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	z := ports.ResultadoResolucionFormalizacion{}
	if ctx == nil || r == nil || dependenciaNula(r.pool) || dependenciaNula(r.proveedor) {
		return z, ports.ErrResolucionFormalizacionNoDisponible
	}
	if e := ctx.Err(); e != nil {
		return z, e
	}
	if s.Validar() != nil {
		return z, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	m, e := r.proveedor.PrepararResolucionFormalizacion(ctx, s)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	if m.Validar() != nil || m.Solicitud != s {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	recurso, e := RecursoResolucionFormalizacion(m)
	if e != nil {
		return z, e
	}
	a, e := r.proveedor.AutorizarResolucionFormalizacion(ctx, m)
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	h, e := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if e != nil || a.ValidarEstructura() != nil || c.Operacion() != AccionResolucionFormalizacion ||
		c.EfectoRef() != s.ExpedienteRef || c.EfectoHuellaSHA256() != h || c.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return z, ports.ErrResolucionFormalizacionDenegada
	}
	b, e := json.Marshal(m)
	if e != nil || len(b) > 16384 {
		return z, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	defer revertirTransaccion(tx)
	if _, e = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),
 set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
 set_config('idle_in_transaction_session_timeout','20s',true)`); e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	material := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range material {
			borrarBytes(b)
		}
	}()
	var raw string
	e = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_resolucion_formalizacion_v1(
 $1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`, string(b), material[0], material[1], material[2], material[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), material[4], material[5], material[6], material[7]).Scan(&raw)
	if e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	var out ports.ResultadoResolucionFormalizacion
	if len(raw) == 0 || len(raw) > 16384 || decodificarJSONEstricto([]byte(raw), &out) != nil {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	out.RegistradaEn = out.RegistradaEn.UTC()
	if out.ValidarPara(s) != nil || out.DocumentoRef != m.DocumentoRef || out.DocumentoVersion != m.DocumentoVersion ||
		out.DocumentoSHA256 != m.DocumentoSHA256 || out.RegistradaEn.Before(m.PropuestaConfirmadaEn) {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	if e = tx.Commit(ctx); e != nil {
		return z, errorResolucionFormalizacion(ctx, e)
	}
	if ctx.Err() != nil {
		return z, ctx.Err()
	}
	return out, nil
}
func errorResolucionFormalizacion(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, v := range []error{ports.ErrSolicitudResolucionFormalizacionInvalida, ports.ErrResolucionFormalizacionDenegada,
		ports.ErrResolucionFormalizacionEnConflicto, ports.ErrClaveResolucionFormalizacionUsada, ports.ErrResultadoResolucionFormalizacionNoConfiable,
		context.Canceled, context.DeadlineExceeded} {
		if errors.Is(e, v) {
			return v
		}
	}
	var p *pgconn.PgError
	if errors.As(e, &p) && p != nil {
		switch p.Code {
		case "P0690":
			return ports.ErrSolicitudResolucionFormalizacionInvalida
		case "P0691", "23505":
			return ports.ErrClaveResolucionFormalizacionUsada
		case "P0692":
			return ports.ErrResolucionFormalizacionEnConflicto
		case "P0693", "42501":
			return ports.ErrResolucionFormalizacionDenegada
		}
	}
	return ports.ErrResolucionFormalizacionNoDisponible
}

func (s *SesionConsultaRRHHPostgreSQL) ConsultarPreparacionYRegistrarAcceso(ctx context.Context, orden ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, ports.PreparacionResolucionFormalizacion, error) {
	dz, pz := ports.DetalleExpedienteRRHH{}, ports.PreparacionResolucionFormalizacion{}
	if err := s.validarContexto(ctx); err != nil {
		return dz, pz, err
	}
	if orden.Solicitud().VersionObservada() != 0 {
		return dz, pz, ports.ErrSolicitudResolucionFormalizacionInvalida
	}
	m, err := orden.ExportacionParaSQL()
	if err != nil || m.ValidarEstructura() != nil {
		return dz, pz, ports.ErrConsultaRRHHNoDisponible
	}
	args, err := nuevosArgumentosMaterialConsultaRRHH(m)
	if err != nil {
		return dz, pz, err
	}
	defer args.limpiar()
	c, cap, q := orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	var salida salidaDetalleConsultaRRHH
	defer clear(salida.contenidoCanonico)
	var raw string
	// Misma fila de detalle/recibo de acceso; añade solo la preparación ligada
	// en servidor a la misma instantánea nominal. No usa el pool de efectos.
	consulta := strings.Replace(consultaDetalleRRHHPostgreSQL, "recibo_sello_sha256", "recibo_sello_sha256, preparacion::text", 1)
	consulta = strings.Replace(consulta, "consultar_detalle_rrhh_atestado_v1", "consultar_preparacion_resolucion_v1", 1)
	destinos := append(destinosDetalleConsultaRRHH(&salida), &raw)
	type resultado struct {
		detalle     ports.DetalleExpedienteRRHH
		preparacion ports.PreparacionResolucionFormalizacion
	}
	r, err := ejecutarConsultaRRHHEnTransaccion(ctx, s.pool, consulta,
		argumentosSQLDetalleConsultaRRHH(c.OrganizacionRef(), string(cap.ClaseAmbito()), cap.AmbitoRef(), q, args), destinos,
		func() (resultado, error) {
			z := resultado{}
			if ctx.Err() != nil {
				return z, ctx.Err()
			}
			salida.cierre.normalizarInstantesSQL()
			recibo, e := salida.cierre.construirRecibo(c, cap)
			if e != nil {
				return z, e
			}
			_, version, _, e := salida.cierre.enterosSeguros()
			if e != nil {
				return z, e
			}
			entrada, e := s.analizador.analizarDetalle(salida.contenidoCanonico, salida.cierre.generadaEn, salida.cierre.expedienteRef, version)
			if e != nil {
				return z, e
			}
			d, e := ports.NuevoDetalleExpedienteRRHHMinimizado(entrada, recibo)
			if e != nil || d.ValidarParaEjecucionInterna(orden) != nil {
				return z, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			p, e := decodificarPreparacionResolucion(raw, q.ExpedienteRef(), version)
			if e != nil {
				return z, e
			}
			return resultado{d, p}, nil
		})
	if ctx.Err() != nil {
		return dz, pz, ctx.Err()
	}
	if err != nil {
		return dz, pz, err
	}
	return r.detalle, r.preparacion.Clonar(), nil
}

func decodificarPreparacionResolucion(raw, expediente string, version uint64) (ports.PreparacionResolucionFormalizacion, error) {
	z := ports.PreparacionResolucionFormalizacion{}
	var p ports.PreparacionResolucionFormalizacion
	if len(raw) == 0 || len(raw) > 16384 || decodificarJSONEstricto([]byte(raw), &p) != nil ||
		p.ValidarPara(expediente) != nil || p.VersionActual != version {
		return z, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}
	return p, nil
}
