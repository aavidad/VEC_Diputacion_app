package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type MaterialResolucionPortal struct {
	Solicitud ports.SolicitudResolucionPortal
	Politica  ports.ReferenciaGobernadaComunicacionLlamamiento
}

func (m MaterialResolucionPortal) validar() error {
	if m.Solicitud.Validar() != nil || m.Politica.Validar() != nil ||
		m.Politica.Referencia != m.Solicitud.CriterioValidacionRef {
		return ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

type ProveedorResolucionPortal interface {
	PrepararResolucionPortal(context.Context, ports.SolicitudResolucionPortal) (MaterialResolucionPortal, error)
	AutorizarResolucionPortal(context.Context, MaterialResolucionPortal) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoResolucionPortal(m MaterialResolucionPortal) (dominiovec.RecursoAutorizable, error) {
	r, _, err := recursoResolucionPortal(m)
	return r, err
}

func recursoResolucionPortal(m MaterialResolucionPortal) (dominiovec.RecursoAutorizable, []byte, error) {
	if err := m.validar(); err != nil {
		return dominiovec.RecursoAutorizable{}, nil, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialComunicacionLlamamiento {
		return dominiovec.RecursoAutorizable{}, nil, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: m.Solicitud.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoResolucionManualLlamamiento,
		Ambitos:   map[string]string{"organizacion_ref": m.Solicitud.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, b, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) ResolverPortal(
	ctx context.Context, s ports.SolicitudResolucionPortal,
) (ports.ResultadoResolucionPortal, error) {
	var resultado ports.ResultadoResolucionPortal
	if ctx == nil {
		return resultado, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	err := ejecutarConReintentoSerializable(ctx, func() error {
		resultado = ports.ResultadoResolucionPortal{}
		var fallo error
		resultado, fallo = t.resolverPortalIntento(ctx, s)
		return fallo
	})
	if err != nil {
		return ports.ResultadoResolucionPortal{}, normalizarErrorPortal(ctx, err)
	}
	return resultado, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) resolverPortalIntento(
	ctx context.Context, s ports.SolicitudResolucionPortal,
) (ports.ResultadoResolucionPortal, error) {
	vacio := ports.ResultadoResolucionPortal{}
	if s.Validar() != nil {
		return vacio, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	if t == nil || dependenciaNula(t.pool) {
		return vacio, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	p, ok := t.proveedor.(ProveedorResolucionPortal)
	if !ok || dependenciaNula(p) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	m, err := p.PrepararResolucionPortal(ctx, s)
	if err != nil {
		return vacio, err
	}
	if m.Solicitud != s {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	recurso, contenido, err := recursoResolucionPortal(m)
	if err != nil {
		return vacio, err
	}
	a, err := p.AutorizarResolucionPortal(ctx, m)
	if err != nil {
		return vacio, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionResolucionManualLlamamiento ||
		r.EfectoRef() != s.ExpedienteRef || r.EfectoHuellaSHA256() != huella ||
		r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, err
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true), set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, err
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var salida string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_resolucion_respuesta_portal_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6::text::numeric,$7::text::numeric,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		strconv.FormatUint(a.PersonaVersion(), 10), strconv.FormatUint(a.PerfilVersion(), 10),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return vacio, err
	}
	var recibo ports.ResultadoResolucionPortal
	if len(salida) == 0 || len(salida) > maximoMaterialComunicacionLlamamiento ||
		decodificarJSONEstricto([]byte(salida), &recibo) != nil {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	recibo.ResueltaEn = recibo.ResueltaEn.UTC()
	recibo.RespuestaPortal.RespondidaEn = recibo.RespuestaPortal.RespondidaEn.UTC()
	if recibo.ValidarPara(s) != nil || recibo.Politica != m.Politica {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	return recibo, nil
}

func normalizarErrorPortal(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P2490":
			return ports.ErrSolicitudComunicacionLlamamientoInvalida
		case "P2491":
			return ports.ErrClaveComunicacionLlamamientoUsada
		case "P2492":
			return ports.ErrVersionComunicacionLlamamientoEnConflicto
		case "P2493":
			return ports.ErrOperacionComunicacionLlamamientoDenegada
		}
	}
	return normalizarErrorResolucionManualLlamamiento(ctx, err)
}
