package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultaJustificanteRespuestaRecibida      = "contratacion_temporal.llamamiento.respuesta.consultar_justificante"
	TipoRecursoConsultaJustificanteRespuestaRecibida = "justificante_respuesta_recibida_ct"
	maximoJustificanteRespuestaRecibida              = 64 * 1024
)

// ProveedorConsultaJustificanteRespuestaRecibida emite autorización nominal
// nueva, ligada a los ocho campos de la intención, no un permiso de registro.
type ProveedorConsultaJustificanteRespuestaRecibida interface {
	AutorizarConsultaJustificanteRespuestaRecibida(context.Context, ports.SolicitudResolverLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoConsultaJustificanteRespuestaRecibida(s ports.SolicitudResolverLlamamiento) (dominiovec.RecursoAutorizable, error) {
	if s.Validar() != nil || (s.Respuesta != ports.RespuestaLlamamientoAceptada && s.Respuesta != ports.RespuestaLlamamientoRenunciada) {
		return dominiovec.RecursoAutorizable{}, ports.ErrSolicitudRespuestaRecibidaInvalida
	}
	if s.VersionEsperada != 2 {
		return dominiovec.RecursoAutorizable{}, ports.ErrVersionRespuestaRecibidaEnConflicto
	}
	// Material directo, sin envoltorio ni cambio de nombres: mismos ocho campos.
	b, err := json.Marshal(s)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, ports.ErrSolicitudRespuestaRecibidaInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoConsultaJustificanteRespuestaRecibida,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

type LectorJustificantesRespuestaRecibidaPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorConsultaJustificanteRespuestaRecibida
}

var _ ports.LectorJustificantesRespuestaRecibida = (*LectorJustificantesRespuestaRecibidaPostgreSQL)(nil)

func NuevoLectorJustificantesRespuestaRecibidaPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorConsultaJustificanteRespuestaRecibida) (*LectorJustificantesRespuestaRecibidaPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrRespuestaRecibidaNoDisponible
	}
	return &LectorJustificantesRespuestaRecibidaPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (l *LectorJustificantesRespuestaRecibidaPostgreSQL) ConsultarJustificanteRespuestaRecibida(ctx context.Context, s ports.SolicitudResolverLlamamiento) (ports.JustificanteRespuestaRecibida, error) {
	vacio := ports.JustificanteRespuestaRecibida{}
	if ctx == nil || l == nil || dependenciaNula(l.pool) || dependenciaNula(l.proveedor) {
		return vacio, ports.ErrRespuestaRecibidaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recurso, err := RecursoConsultaJustificanteRespuestaRecibida(s)
	if err != nil {
		return vacio, err
	}
	a, err := l.proveedor.AutorizarConsultaJustificanteRespuestaRecibida(ctx, s)
	if err != nil {
		return vacio, normalizarErrorConsultaJustificanteRespuestaRecibida(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || r.Operacion() != AccionConsultaJustificanteRespuestaRecibida ||
		r.EfectoRef() != s.ExpedienteRef || r.EfectoHuellaSHA256() != huella ||
		r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrOperacionRespuestaRecibidaDenegada
	}
	contenido, err := json.Marshal(s)
	if err != nil {
		return vacio, ports.ErrSolicitudRespuestaRecibidaInvalida
	}
	j, err := l.consultar(ctx, s, contenido, a)
	if err != nil {
		return vacio, normalizarErrorConsultaJustificanteRespuestaRecibida(ctx, err)
	}
	return j, nil
}

func (l *LectorJustificantesRespuestaRecibidaPostgreSQL) consultar(ctx context.Context, s ports.SolicitudResolverLlamamiento, contenido []byte, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.JustificanteRespuestaRecibida, error) {
	vacio := ports.JustificanteRespuestaRecibida{}
	// Es lectura de negocio, pero el consumo de autorización debe confirmarse.
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
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
	if err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.consultar_justificante_respuesta_recibida_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&resultado); err != nil {
		return vacio, err
	}
	var justificante ports.JustificanteRespuestaRecibida
	if len(resultado) == 0 || len(resultado) > maximoJustificanteRespuestaRecibida ||
		decodificarJSONEstricto([]byte(resultado), &justificante) != nil {
		return vacio, ports.ErrResultadoRespuestaRecibidaNoConfiable
	}
	// PostgreSQL representa UTC también como +00:00. Normalizar solo ubicación,
	// nunca truncar fracciones: la validación conserva la precisión requerida.
	justificante.Respuesta.Solicitud.RecibidaEn = justificante.Respuesta.Solicitud.RecibidaEn.UTC()
	justificante.Respuesta.RegistradaEn = justificante.Respuesta.RegistradaEn.UTC()
	justificante.Seleccion.ConfirmadaEn = justificante.Seleccion.ConfirmadaEn.UTC()
	e := &justificante.Seleccion.Procedencia.Evidencia
	e.EmitidaEn, e.ValidaHasta, e.RetenerHasta = e.EmitidaEn.UTC(), e.ValidaHasta.UTC(), e.RetenerHasta.UTC()
	if err := justificante.ValidarPara(s); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	// Un commit incierto no devuelve datos ni activa reintento automático.
	if err := tx.Commit(ctx); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := justificante.ValidarPara(s); err != nil {
		return vacio, err
	}
	return justificante, nil
}

// Vocabulario propio de CT57; CT56 y su registro permanecen sin cambios.
func normalizarErrorConsultaJustificanteRespuestaRecibida(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg != nil {
		switch pg.Code {
		case "P0570":
			return ports.ErrSolicitudRespuestaRecibidaInvalida
		case "P0572":
			return ports.ErrVersionRespuestaRecibidaEnConflicto
		case "P0573":
			return ports.ErrOperacionRespuestaRecibidaDenegada
		case "P0574":
			return ports.ErrRespuestaRecibidaNoDisponible
		}
	}
	return normalizarErrorRespuestaRecibida(ctx, err)
}
