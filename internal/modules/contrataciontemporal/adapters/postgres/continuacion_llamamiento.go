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
	AccionContinuacionLlamamiento         = "contratacion_temporal.llamamiento.siguiente.continuar"
	TipoRecursoContinuacionLlamamiento    = "continuacion_llamamiento_ct"
	maximoMaterialContinuacionLlamamiento = 64 * 1024
)

// Cada llamada emite autoridad nueva sobre etapa y material exactos. No
// reutilizar el permiso de renuncia, lectura de justificante o apertura Bolsa.
type ProveedorContinuacionLlamamiento interface {
	AutorizarContinuacionLlamamiento(context.Context, ports.MaterialContinuacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

func RecursoContinuacionLlamamiento(m ports.MaterialContinuacionLlamamiento) (dominiovec.RecursoAutorizable, error) {
	b, err := materialContinuacion(m)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: m.Solicitud.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoContinuacionLlamamiento,
		Ambitos:   map[string]string{"organizacion_ref": m.Solicitud.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

func materialContinuacion(m ports.MaterialContinuacionLlamamiento) ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialContinuacionLlamamiento {
		return nil, ports.ErrOperacionContinuacionInvalida
	}
	return b, nil
}

type RegistroContinuacionLlamamientoPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorContinuacionLlamamiento
}

var _ ports.RegistroContinuacionLlamamiento = (*RegistroContinuacionLlamamientoPostgreSQL)(nil)

func NuevoRegistroContinuacionLlamamientoPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorContinuacionLlamamiento) (*RegistroContinuacionLlamamientoPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrOperacionContinuacionNoDisponible
	}
	return &RegistroContinuacionLlamamientoPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func (r *RegistroContinuacionLlamamientoPostgreSQL) LeerAntecedente(ctx context.Context, s ports.SolicitudContinuarLlamamiento) (ports.AntecedenteContinuacionLlamamiento, error) {
	var resultado ports.AntecedenteContinuacionLlamamiento
	m := ports.MaterialContinuacionLlamamiento{Etapa: "consulta", Solicitud: s}
	err := r.ejecutar(ctx, m, &resultado, func() error {
		resultado.Resolucion.ResueltaEn = resultado.Resolucion.ResueltaEn.UTC()
		resultado.Resolucion.IntencionSiguiente.ActualizadaEn = resultado.Resolucion.IntencionSiguiente.ActualizadaEn.UTC()
		return resultado.ValidarPara(s)
	})
	if err != nil {
		return ports.AntecedenteContinuacionLlamamiento{}, normalizarErrorContinuacion(ctx, err)
	}
	return resultado, nil
}

func (r *RegistroContinuacionLlamamientoPostgreSQL) Confirmar(ctx context.Context, s ports.SolicitudContinuarLlamamiento, b ports.ReciboBolsaContinuacion) (ports.ResultadoContinuacionLlamamiento, error) {
	var resultado ports.ResultadoContinuacionLlamamiento
	// Copia por valor: el proveedor no puede modificar el material que se envía.
	m := ports.MaterialContinuacionLlamamiento{Etapa: "confirmacion", Solicitud: s, ReciboBolsa: &b}
	err := r.ejecutar(ctx, m, &resultado, func() error {
		resultado.ConfirmadaEn = resultado.ConfirmadaEn.UTC()
		resultado.ReciboBolsa.ConfirmadaEn = resultado.ReciboBolsa.ConfirmadaEn.UTC()
		if resultado.ReciboBolsa != b {
			return ports.ErrOperacionContinuacionNoDisponible
		}
		return resultado.ValidarPara(s)
	})
	if err != nil {
		return ports.ResultadoContinuacionLlamamiento{}, normalizarErrorContinuacion(ctx, err)
	}
	return resultado, nil
}

// Una sola frontera SQL para las dos etapas. No mantiene transacciones durante
// la llamada a Bolsa, no reintenta efectos ni devuelve recibos antes del COMMIT.
func (r *RegistroContinuacionLlamamientoPostgreSQL) ejecutar(ctx context.Context, m ports.MaterialContinuacionLlamamiento, destino any, validar func() error) error {
	if ctx == nil || r == nil || dependenciaNula(r.pool) || dependenciaNula(r.proveedor) {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	contenido, err := materialContinuacion(m)
	if err != nil {
		return err
	}
	recurso, err := RecursoContinuacionLlamamiento(m)
	if err != nil {
		return err
	}
	paraAutorizar := m
	if m.ReciboBolsa != nil {
		copia := *m.ReciboBolsa
		paraAutorizar.ReciboBolsa = &copia
	}
	a, err := r.proveedor.AutorizarContinuacionLlamamiento(ctx, paraAutorizar)
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || c.Operacion() != AccionContinuacionLlamamiento ||
		c.EfectoRef() != m.Solicitud.ExpedienteRef || c.EfectoHuellaSHA256() != h ||
		c.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return ports.ErrOperacionContinuacionDenegada
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	defer revertirTransaccion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true), set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return err
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(),
		a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytes(b)
		}
	}()
	var jsonResultado string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.continuar_llamamiento_rrhh_v1(
		$1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&jsonResultado)
	if err != nil {
		return err
	}
	if len(jsonResultado) == 0 || len(jsonResultado) > maximoMaterialContinuacionLlamamiento ||
		decodificarJSONEstricto([]byte(jsonResultado), destino) != nil {
		return ports.ErrOperacionContinuacionNoDisponible
	}
	if err = validar(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return validar()
}

func normalizarErrorContinuacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, propio := range []error{ports.ErrOperacionContinuacionInvalida, ports.ErrOperacionContinuacionDenegada,
		ports.ErrOperacionContinuacionConflicto, ports.ErrOperacionContinuacionNoDisponible} {
		if errors.Is(err, propio) {
			return propio
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg != nil {
		switch pg.Code {
		case "P0600":
			return ports.ErrOperacionContinuacionInvalida
		case "P0601", "P0602", "23505":
			return ports.ErrOperacionContinuacionConflicto
		case "P0603", "42501":
			return ports.ErrOperacionContinuacionDenegada
		}
	}
	return ports.ErrOperacionContinuacionNoDisponible
}
