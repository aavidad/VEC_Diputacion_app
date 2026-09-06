package postgres

import (
	"bytes"
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
	AccionPropuestaFormalizacion         = "contratacion_temporal.formalizacion.propuesta.registrar"
	TipoRecursoPropuestaFormalizacion    = "propuesta_formalizacion_ct"
	maximoMaterialPropuestaFormalizacion = 64 * 1024
)

var ErrPersistenciaPropuestaFormalizacionNoDisponible = errors.New("contratacion temporal: persistencia de propuesta no disponible")

// El proveedor deriva la evidencia del terminal real de Bolsa antes de abrir
// la transacción CT. Cada etapa y replay exige autoridad nueva sobre sus bytes.
type ProveedorPropuestaFormalizacion interface {
	PrepararPropuestaFormalizacion(context.Context, ports.SolicitudPropuestaFormalizacion) (ports.MaterialPropuestaFormalizacion, error)
	AutorizarPropuestaFormalizacion(context.Context, ports.MaterialPropuestaFormalizacion) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type RegistroPropuestaFormalizacionPostgreSQL struct {
	pool      iniciadorTransacciones
	proveedor ProveedorPropuestaFormalizacion
}

var _ ports.TransaccionPropuestaFormalizacion = (*RegistroPropuestaFormalizacionPostgreSQL)(nil)

func NuevoRegistroPropuestaFormalizacionPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorPropuestaFormalizacion) (*RegistroPropuestaFormalizacionPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ErrPersistenciaPropuestaFormalizacionNoDisponible
	}
	return &RegistroPropuestaFormalizacionPostgreSQL{pool: pool, proveedor: proveedor}, nil
}

func materialPropuestaFormalizacion(m ports.MaterialPropuestaFormalizacion) ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > maximoMaterialPropuestaFormalizacion {
		return nil, ports.ErrSolicitudPropuestaFormalizacionInvalida
	}
	return b, nil
}

func RecursoPropuestaFormalizacion(m ports.MaterialPropuestaFormalizacion) (dominiovec.RecursoAutorizable, error) {
	b, err := materialPropuestaFormalizacion(m)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: m.Solicitud.ExpedienteRef, ModuloID: "contratacion_temporal", Tipo: TipoRecursoPropuestaFormalizacion,
		Ambitos:   map[string]string{"organizacion_ref": m.Solicitud.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}

func (r *RegistroPropuestaFormalizacionPostgreSQL) LeerAntecedente(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.AntecedentePropuestaFormalizacion, error) {
	var resultado ports.AntecedentePropuestaFormalizacion
	m := ports.MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s.Clonar()}
	err := r.ejecutar(ctx, m, &resultado, func() error {
		// Solo ubicación UTC; no se truncan instantes ni se reescriben recibos.
		resultado.Resolucion.ResueltaEn = resultado.Resolucion.ResueltaEn.UTC()
		j := &resultado.Justificante
		j.Respuesta.Solicitud.RecibidaEn = j.Respuesta.Solicitud.RecibidaEn.UTC()
		j.Respuesta.RegistradaEn = j.Respuesta.RegistradaEn.UTC()
		j.Seleccion.ConfirmadaEn = j.Seleccion.ConfirmadaEn.UTC()
		e := &j.Seleccion.Procedencia.Evidencia
		e.EmitidaEn, e.ValidaHasta, e.RetenerHasta = e.EmitidaEn.UTC(), e.ValidaHasta.UTC(), e.RetenerHasta.UTC()
		if c := j.Continuacion; c != nil {
			c.ConfirmadaEn = c.ConfirmadaEn.UTC()
			c.ReciboBolsa.ConfirmadaEn = c.ReciboBolsa.ConfirmadaEn.UTC()
		}
		return resultado.ValidarPara(s)
	})
	if err != nil {
		return ports.AntecedentePropuestaFormalizacion{}, normalizarErrorPropuestaFormalizacion(ctx, err)
	}
	return resultado, nil
}

func (r *RegistroPropuestaFormalizacionPostgreSQL) ConfirmarPropuesta(ctx context.Context, s ports.SolicitudPropuestaFormalizacion) (ports.ResultadoPropuestaFormalizacion, error) {
	var resultado ports.ResultadoPropuestaFormalizacion
	if ctx == nil || r == nil || dependenciaNula(r.pool) || dependenciaNula(r.proveedor) {
		return resultado, ErrPersistenciaPropuestaFormalizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return resultado, err
	}
	if (ports.MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil {
		return resultado, ports.ErrSolicitudPropuestaFormalizacionInvalida
	}
	m, err := r.proveedor.PrepararPropuestaFormalizacion(ctx, s.Clonar())
	if err != nil {
		return resultado, normalizarErrorPropuestaFormalizacion(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return resultado, err
	}
	esperada, _ := json.Marshal(s)
	recibida, err := json.Marshal(m.Solicitud)
	if err != nil || m.Etapa != "confirmacion" || m.Validar() != nil || !bytes.Equal(esperada, recibida) {
		return resultado, ports.ErrResultadoPropuestaFormalizacionNoConfiable
	}
	err = r.ejecutar(ctx, m.Clonar(), &resultado, func() error {
		resultado.ConfirmadaEn = resultado.ConfirmadaEn.UTC()
		return resultado.ValidarPara(s)
	})
	if err != nil {
		return ports.ResultadoPropuestaFormalizacion{}, normalizarErrorPropuestaFormalizacion(ctx, err)
	}
	return resultado.Clonar(), nil
}

func (r *RegistroPropuestaFormalizacionPostgreSQL) ejecutar(ctx context.Context, m ports.MaterialPropuestaFormalizacion, destino any, validar func() error) error {
	if ctx == nil || r == nil || dependenciaNula(r.pool) || dependenciaNula(r.proveedor) {
		return ErrPersistenciaPropuestaFormalizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	contenido, err := materialPropuestaFormalizacion(m)
	if err != nil {
		return err
	}
	recurso, err := RecursoPropuestaFormalizacion(m)
	if err != nil {
		return err
	}
	a, err := r.proveedor.AutorizarPropuestaFormalizacion(ctx, m.Clonar())
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	c := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil || c.Operacion() != AccionPropuestaFormalizacion ||
		c.EfectoRef() != m.Solicitud.ExpedienteRef || c.EfectoHuellaSHA256() != h ||
		c.AudienciaConsumo() != AudienciaRegistroComunicacionLlamamiento {
		return ports.ErrOperacionPropuestaFormalizacionDenegada
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
	var salida string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_propuesta_formalizacion_v1(
  $1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		string(contenido), secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()),
		secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return err
	}
	if len(salida) == 0 || len(salida) > maximoMaterialPropuestaFormalizacion || decodificarJSONEstricto([]byte(salida), destino) != nil {
		return ports.ErrResultadoPropuestaFormalizacionNoConfiable
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

func normalizarErrorPropuestaFormalizacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, propio := range []error{ports.ErrSolicitudPropuestaFormalizacionInvalida, ports.ErrResultadoPropuestaFormalizacionNoConfiable,
		ports.ErrOperacionPropuestaFormalizacionDenegada, ports.ErrVersionPropuestaFormalizacionEnConflicto,
		ports.ErrClavePropuestaFormalizacionUsada, ports.ErrResolucionLlamamientoNoAceptada, ErrPersistenciaPropuestaFormalizacionNoDisponible} {
		if errors.Is(err, propio) {
			return propio
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg != nil {
		switch pg.Code {
		case "P0610":
			return ports.ErrSolicitudPropuestaFormalizacionInvalida
		case "P0611", "23505":
			return ports.ErrClavePropuestaFormalizacionUsada
		case "P0612":
			return ports.ErrVersionPropuestaFormalizacionEnConflicto
		case "P0613", "42501":
			return ports.ErrOperacionPropuestaFormalizacionDenegada
		case "P0615":
			return ports.ErrResolucionLlamamientoNoAceptada
		}
	}
	return ErrPersistenciaPropuestaFormalizacionNoDisponible
}
