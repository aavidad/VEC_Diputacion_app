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

// MaterialEventoPlazoLlamamiento es el material autorizado y persistido por
// CT111: la solicitud de RRHH y, solo con el contacto efectivo, el plazo que
// el servidor calculó con el catálogo. Su JSON directo es el contrato SQL.
type MaterialEventoPlazoLlamamiento struct {
	Solicitud ports.SolicitudRegistrarEventoPlazoLlamamiento
	Plazo     *ports.PlazoRespuestaGobernado `json:",omitempty"`
}

func (m MaterialEventoPlazoLlamamiento) Validar() error {
	if m.Solicitud.Validar() != nil ||
		(m.Solicitud.Tipo == ports.EventoPlazoContactoEfectivo) != (m.Plazo != nil) ||
		(m.Plazo != nil && m.Plazo.ValidarDesde(m.Solicitud.InstanteEn) != nil) {
		return ports.ErrSolicitudEventoPlazoInvalida
	}
	return nil
}

// ProveedorEventoPlazoLlamamiento emite, en cada petición y también en un
// replay, una decisión nueva del permiso de validación manual de respuesta y
// plazo (el mismo consumidor AD3-17 que la resolución), ligada al material.
type ProveedorEventoPlazoLlamamiento interface {
	AutorizarEventoPlazo(context.Context, MaterialEventoPlazoLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoEventoPlazoLlamamiento(m MaterialEventoPlazoLlamamiento) (dominiovec.RecursoAutorizable, error) {
	b, err := codificarMaterialEventoPlazo(m)
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

func codificarMaterialEventoPlazo(m MaterialEventoPlazoLlamamiento) ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialComunicacionLlamamiento {
		return nil, ports.ErrSolicitudEventoPlazoInvalida
	}
	return b, nil
}

type RegistroEventosPlazoLlamamientoPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorEventoPlazoLlamamiento
}

var _ ports.RegistroEventosPlazoLlamamiento = (*RegistroEventosPlazoLlamamientoPostgreSQL)(nil)

func NuevoRegistroEventosPlazoLlamamientoPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorEventoPlazoLlamamiento) (*RegistroEventosPlazoLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrEventoPlazoNoDisponible
	}
	return &RegistroEventosPlazoLlamamientoPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (t *RegistroEventosPlazoLlamamientoPostgreSQL) RegistrarEventoPlazo(
	ctx context.Context,
	s ports.SolicitudRegistrarEventoPlazoLlamamiento,
	plazo *ports.PlazoRespuestaGobernado,
) (ports.EventoPlazoLlamamientoRegistrado, error) {
	vacio := ports.EventoPlazoLlamamientoRegistrado{}
	if ctx == nil || t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) {
		return vacio, ports.ErrEventoPlazoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	// Copia defensiva: el material no comparte el plazo del llamante.
	m := MaterialEventoPlazoLlamamiento{Solicitud: s}
	if plazo != nil {
		copia := *plazo
		m.Plazo = &copia
	}
	contenido, err := codificarMaterialEventoPlazo(m)
	if err != nil {
		return vacio, err
	}
	a, err := t.proveedor.AutorizarEventoPlazo(ctx, m)
	if err != nil {
		return vacio, normalizarErrorEventoPlazo(ctx, err)
	}
	recurso, err := RecursoEventoPlazoLlamamiento(m)
	huella, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || errHuella != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionResolucionManualLlamamiento || r.EfectoRef() != s.ExpedienteRef ||
		r.EfectoHuellaSHA256() != huella || r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	recibo, err := t.registrar(ctx, s, contenido, a)
	if err != nil {
		return vacio, normalizarErrorEventoPlazo(ctx, err)
	}
	return recibo, nil
}

func (t *RegistroEventosPlazoLlamamientoPostgreSQL) registrar(ctx context.Context, s ports.SolicitudRegistrarEventoPlazoLlamamiento, contenido []byte, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EventoPlazoLlamamientoRegistrado, error) {
	vacio := ports.EventoPlazoLlamamientoRegistrado{}
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
	var contenidoRecibo string
	if err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_evento_plazo_llamamiento_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&contenidoRecibo); err != nil {
		return vacio, err
	}
	var recibo ports.EventoPlazoLlamamientoRegistrado
	if len(contenidoRecibo) == 0 || len(contenidoRecibo) > 2*maximoMaterialComunicacionLlamamiento ||
		decodificarJSONEstricto([]byte(contenidoRecibo), &recibo) != nil {
		return vacio, ports.ErrResultadoEventoPlazoNoConfiable
	}
	recibo.Solicitud.InstanteEn = normalizarInstantePostgreSQL(recibo.Solicitud.InstanteEn)
	recibo.RegistradoEn = normalizarInstantePostgreSQL(recibo.RegistradoEn)
	if recibo.Plazo != nil {
		recibo.Plazo.RespuestaHasta = normalizarInstantePostgreSQL(recibo.Plazo.RespuestaHasta)
	}
	if err := recibo.ValidarPara(s); err != nil {
		return vacio, err
	}
	// Un COMMIT incierto no entrega recibo: el reintento con la misma clave y
	// autorización nueva recupera el original sin otro evento.
	if err = tx.Commit(ctx); err != nil {
		return vacio, err
	}
	return recibo, nil
}

func normalizarErrorEventoPlazo(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, permitido := range []error{context.Canceled, context.DeadlineExceeded,
		ports.ErrOperacionEventoPlazoDenegada, ports.ErrClaveEventoPlazoUsada,
		ports.ErrEventoPlazoEnConflicto, ports.ErrSolicitudEventoPlazoInvalida,
		ports.ErrResultadoEventoPlazoNoConfiable} {
		if errors.Is(err, permitido) {
			return permitido
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501", "P0593":
			return ports.ErrOperacionEventoPlazoDenegada
		case "P0590":
			return ports.ErrSolicitudEventoPlazoInvalida
		case "P0591":
			return ports.ErrClaveEventoPlazoUsada
		case "P0592":
			return ports.ErrEventoPlazoEnConflicto
		}
	}
	return ports.ErrEventoPlazoNoDisponible
}
