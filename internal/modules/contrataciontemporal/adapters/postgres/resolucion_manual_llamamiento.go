package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionResolucionManualLlamamiento      = "contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar"
	TipoRecursoResolucionManualLlamamiento = "resolucion_manual_respuesta_ct"
)

// MaterialResolucionManualLlamamiento liga la revisión expresa de RRHH a la
// política resuelta por composición confiable, no a una regla enviada por HTTP.
// Su JSON directo conserva Solicitud (once campos) y Politica (tres campos).
type MaterialResolucionManualLlamamiento struct {
	Solicitud ports.SolicitudResolverLlamamiento
	Politica  ports.ReferenciaGobernadaComunicacionLlamamiento
}

func (m MaterialResolucionManualLlamamiento) Validar() error {
	if m.Solicitud.Validar() != nil || !m.Solicitud.RevisionManualConfirmada() ||
		m.Politica.Validar() != nil || m.Solicitud.CriterioValidacionRef != m.Politica.Referencia {
		return ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

// ProveedorResolucionManualLlamamiento es opcional y separado del registro del
// aviso. Debe reacreditar actor, justificante y criterio vigente en cada petición,
// también en replay, y emitir permiso propio ligado a todo el material. Estas
// referencias y revisiones no son una verificación automática del correo.
type ProveedorResolucionManualLlamamiento interface {
	PrepararResolucionManual(context.Context, ports.SolicitudResolverLlamamiento) (MaterialResolucionManualLlamamiento, error)
	AutorizarResolucionManual(context.Context, MaterialResolucionManualLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoResolucionManualLlamamiento(m MaterialResolucionManualLlamamiento) (dominiovec.RecursoAutorizable, error) {
	b, err := codificarMaterialResolucionManualLlamamiento(m)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: m.Solicitud.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoResolucionManualLlamamiento,
		Ambitos:   map[string]string{"organizacion_ref": m.Solicitud.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

func codificarMaterialResolucionManualLlamamiento(m MaterialResolucionManualLlamamiento) ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialComunicacionLlamamiento {
		return nil, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return b, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) resolverManual(ctx context.Context, s ports.SolicitudResolverLlamamiento) (ports.ResultadoResolucionLlamamiento, error) {
	vacio := ports.ResultadoResolucionLlamamiento{}
	if ctx == nil {
		return vacio, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	if !s.RevisionManualConfirmada() || t == nil || dependenciaNula(t.proveedor) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	p, ok := t.proveedor.(ProveedorResolucionManualLlamamiento)
	if !ok || dependenciaNula(p) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if dependenciaNula(t.pool) {
		return vacio, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	m, err := p.PrepararResolucionManual(ctx, s)
	if err != nil {
		return vacio, normalizarErrorResolucionManualLlamamiento(ctx, err)
	}
	if m.Solicitud != s || m.Validar() != nil {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	a, err := p.AutorizarResolucionManual(ctx, m)
	if err != nil {
		return vacio, normalizarErrorResolucionManualLlamamiento(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recurso, err := RecursoResolucionManualLlamamiento(m)
	huella, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || errHuella != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionResolucionManualLlamamiento || r.EfectoRef() != s.ExpedienteRef ||
		r.EfectoHuellaSHA256() != huella || r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	contenido, err := codificarMaterialResolucionManualLlamamiento(m)
	if err != nil {
		return vacio, err
	}
	resultado, err := t.registrarResolucionManual(ctx, m, contenido, a)
	if err != nil {
		return vacio, normalizarErrorResolucionManualLlamamiento(ctx, err)
	}
	return resultado, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) registrarResolucionManual(ctx context.Context, m MaterialResolucionManualLlamamiento, contenido []byte, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoResolucionLlamamiento, error) {
	vacio := ports.ResultadoResolucionLlamamiento{}
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
	var resultado string
	if err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_resolucion_manual_respuesta_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&resultado); err != nil {
		return vacio, err
	}
	var recibo ports.ResultadoResolucionLlamamiento
	if len(resultado) == 0 || len(resultado) > maximoMaterialComunicacionLlamamiento ||
		decodificarJSONEstricto([]byte(resultado), &recibo) != nil {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	// Solo ubicación: nunca esconder precisión submicrosegundo truncándola.
	recibo.ResueltaEn = recibo.ResueltaEn.UTC()
	recibo.IntencionSiguiente.ActualizadaEn = recibo.IntencionSiguiente.ActualizadaEn.UTC()
	if recibo.ValidarPara(m.Solicitud) != nil || recibo.Politica != m.Politica {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	// El permiso se consume junto al asiento también al recuperar. Un COMMIT
	// incierto no entrega recibo ni provoca un segundo intento automático.
	if err := tx.Commit(ctx); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if recibo.ValidarPara(m.Solicitud) != nil || recibo.Politica != m.Politica {
		return vacio, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return recibo, nil
}

func normalizarErrorResolucionManualLlamamiento(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrSolicitudComunicacionLlamamientoInvalida) {
		return ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P0580":
			return ports.ErrSolicitudComunicacionLlamamientoInvalida
		case "P0581":
			return ports.ErrClaveComunicacionLlamamientoUsada
		case "P0582":
			return ports.ErrVersionComunicacionLlamamientoEnConflicto
		case "P0583":
			return ports.ErrOperacionComunicacionLlamamientoDenegada
		case "P0584":
			return ErrPersistenciaComunicacionLlamamientoNoDisponible
		}
	}
	return normalizarErrorComunicacionLlamamiento(ctx, err)
}
