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
	AccionRegistroComunicacionLlamamiento      = "contratacion_temporal.llamamiento.comunicacion.registrar"
	TipoRecursoRegistroComunicacionLlamamiento = "comunicacion_llamamiento_contratacion_temporal"
	AudienciaRegistroComunicacionLlamamiento   = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
	FinalidadRegistroComunicacionLlamamiento   = "gestionar_contratacion_temporal"
	maximoMaterialComunicacionLlamamiento      = 16384
)

var ErrPersistenciaComunicacionLlamamientoNoDisponible = errors.New(
	"contratacion temporal: persistencia de comunicacion de llamamiento no disponible",
)

// MaterialRegistroComunicacionLlamamiento procede de composición confiable.
// SQL comprueba la ejecución confirmada de CT, nunca tablas de Bolsa.
// Canal y política se resuelven en sus autoridades, no desde HTTP.
// No incluye dirección de correo, destinatario ni texto libre.
// Solicitud.VersionEsperada es 1 (versión inicial del llamamiento, no del
// expediente). PruebaEntregaRef identifica su recibo de selección confirmado:
// es un antecedente y nunca una prueba de entrega en esta implementación local.
type MaterialRegistroComunicacionLlamamiento struct {
	Solicitud ports.SolicitudRegistrarComunicacionLlamamiento  `json:"solicitud"`
	Canal     ports.ReferenciaGobernadaComunicacionLlamamiento `json:"canal"`
	Politica  ports.ReferenciaGobernadaComunicacionLlamamiento `json:"politica"`
}

func (m MaterialRegistroComunicacionLlamamiento) Validar() error {
	if m.Solicitud.Validar() != nil ||
		m.Canal.Validar() != nil || m.Politica.Validar() != nil {
		return ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return nil
}

// ProveedorRegistroComunicacionLlamamiento debe reacreditar identidad y permisos
// también en replay. Autorizar recibe exactamente el material validado que se
// escribirá; debe usar RecursoRegistroComunicacionLlamamiento para ligar toda
// la petición a la capacidad V3. No puede fabricar una decisión positiva.
type ProveedorRegistroComunicacionLlamamiento interface {
	PrepararRegistroComunicacion(context.Context, ports.SolicitudRegistrarComunicacionLlamamiento) (MaterialRegistroComunicacionLlamamiento, error)
	AutorizarRegistroComunicacion(context.Context, MaterialRegistroComunicacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoRegistroComunicacionLlamamiento(m MaterialRegistroComunicacionLlamamiento) (dominiovec.RecursoAutorizable, error) {
	contenido, err := codificarMaterialComunicacionLlamamiento(m)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	huella := sha256.Sum256(contenido)
	return dominiovec.RecursoAutorizable{
		Referencia: m.Solicitud.ExpedienteRef,
		ModuloID:   "contratacion_temporal",
		Tipo:       TipoRecursoRegistroComunicacionLlamamiento,
		Ambitos:    map[string]string{"organizacion_ref": m.Solicitud.OrganizacionRef},
		Atributos:  map[string]string{"material_sha256": hex.EncodeToString(huella[:])},
	}, nil
}

type TransaccionComunicacionLlamamientoPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorRegistroComunicacionLlamamiento
}

var _ ports.TransaccionComunicacionLlamamiento = (*TransaccionComunicacionLlamamientoPostgreSQL)(nil)

func NuevaTransaccionComunicacionLlamamientoPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorRegistroComunicacionLlamamiento) (*TransaccionComunicacionLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	return &TransaccionComunicacionLlamamientoPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) RegistrarComunicacion(ctx context.Context, solicitud ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) {
		return ports.ComunicacionProbatoria{}, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	if solicitud.Validar() != nil {
		return ports.ComunicacionProbatoria{}, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	if solicitud.VersionEsperada != 1 {
		return ports.ComunicacionProbatoria{}, ports.ErrVersionComunicacionLlamamientoEnConflicto
	}
	if err := ctx.Err(); err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	preparacion, err := t.proveedor.PrepararRegistroComunicacion(ctx, solicitud)
	if err != nil {
		return ports.ComunicacionProbatoria{}, normalizarErrorComunicacionLlamamiento(ctx, err)
	}
	if preparacion.Solicitud != solicitud || preparacion.Validar() != nil {
		return ports.ComunicacionProbatoria{}, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	autorizacion, err := t.proveedor.AutorizarRegistroComunicacion(ctx, preparacion)
	if err != nil {
		return ports.ComunicacionProbatoria{}, normalizarErrorComunicacionLlamamiento(ctx, err)
	}
	contenido, err := validarMaterialAtestadoComunicacion(preparacion, autorizacion)
	if err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	// Una única transacción. Un COMMIT incierto nunca se reintenta aquí: la misma
	// intención con autorización renovada recuperará el recibo durable.
	recibo, err := t.registrar(ctx, solicitud, preparacion, contenido, autorizacion)
	if err != nil {
		return ports.ComunicacionProbatoria{}, normalizarErrorComunicacionLlamamiento(ctx, err)
	}
	return recibo, nil
}

func (t *TransaccionComunicacionLlamamientoPostgreSQL) registrar(ctx context.Context, solicitud ports.SolicitudRegistrarComunicacionLlamamiento, preparacion MaterialRegistroComunicacionLlamamiento, contenido []byte, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ComunicacionProbatoria, error) {
	tx, err := t.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	secretos := [][]byte{material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(), material.PayloadVECAD3(),
		material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var reciboJSON string
	err = tx.QueryRow(ctx, `
		SELECT vec_contratacion_temporal.registrar_comunicacion_llamamiento_local_v1(
			$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3],
		int64(material.PersonaVersion()), int64(material.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&reciboJSON)
	if err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	recibo, err := decodificarComunicacionLlamamientoLocal(reciboJSON, solicitud)
	if err != nil || recibo.Canal != preparacion.Canal || recibo.Politica != preparacion.Politica {
		return ports.ComunicacionProbatoria{}, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	return recibo, nil
}

func codificarMaterialComunicacionLlamamiento(m MaterialRegistroComunicacionLlamamiento) ([]byte, error) {
	if m.Validar() != nil {
		return nil, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialComunicacionLlamamiento {
		return nil, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return b, nil
}

func validarMaterialAtestadoComunicacion(m MaterialRegistroComunicacionLlamamiento, a puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]byte, error) {
	recurso, err := RecursoRegistroComunicacionLlamamiento(m)
	huella, errHuella := recurso.HuellaContextoAutorizacionSHA256()
	r := a.ResumenCapacidad()
	if err != nil || errHuella != nil || a.ValidarEstructura() != nil ||
		r.Operacion() != AccionRegistroComunicacionLlamamiento ||
		r.EfectoRef() != m.Solicitud.ExpedienteRef || r.EfectoHuellaSHA256() != huella ||
		r.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return nil, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return codificarMaterialComunicacionLlamamiento(m)
}

func decodificarComunicacionLlamamientoLocal(contenido string, solicitud ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	if len(contenido) == 0 || len(contenido) > maximoMaterialComunicacionLlamamiento {
		return ports.ComunicacionProbatoria{}, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	var r ports.ComunicacionProbatoria
	if decodificarJSONEstricto([]byte(contenido), &r) != nil {
		return ports.ComunicacionProbatoria{}, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	r.RegistradaEn = normalizarInstantePostgreSQL(r.RegistradaEn)
	if r.ValidarPara(solicitud) != nil || !r.EsRegistroLocal() {
		return ports.ComunicacionProbatoria{}, ports.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return r, nil
}

// El registro de salida local no es una prueba de entrega ni de respuesta.
// Mantener denegada la resolución evita inventar aceptación o plazo vencido.
// Este método no abre transacción y no modifica el llamamiento de Bolsa.
func (t *TransaccionComunicacionLlamamientoPostgreSQL) ResolverLlamamiento(ctx context.Context, solicitud ports.SolicitudResolverLlamamiento) (ports.ResultadoResolucionLlamamiento, error) {
	if ctx == nil {
		return ports.ResultadoResolucionLlamamiento{}, ErrPersistenciaComunicacionLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoResolucionLlamamiento{}, err
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoResolucionLlamamiento{}, ports.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return ports.ResultadoResolucionLlamamiento{}, ports.ErrOperacionComunicacionLlamamientoDenegada
}

func normalizarErrorComunicacionLlamamiento(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, permitido := range []error{context.Canceled, context.DeadlineExceeded,
		ports.ErrOperacionComunicacionLlamamientoDenegada,
		ports.ErrClaveComunicacionLlamamientoUsada,
		ports.ErrVersionComunicacionLlamamientoEnConflicto,
		ports.ErrResultadoComunicacionLlamamientoNoConfiable} {
		if errors.Is(err, permitido) {
			return permitido
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ports.ErrOperacionComunicacionLlamamientoDenegada
		case "P0541":
			return ports.ErrClaveComunicacionLlamamientoUsada
		case "P0542":
			return ports.ErrVersionComunicacionLlamamientoEnConflicto
		}
	}
	return ErrPersistenciaComunicacionLlamamientoNoDisponible
}
