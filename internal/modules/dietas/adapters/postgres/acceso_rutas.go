package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"time"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

const funcionConsumirAccesoRutas = "SELECT decision_ref, efecto_ref, huella_efecto_sha256, consumo_huella_sha256, auditoria_ref, consumida_en, consumo_nuevo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::numeric,$6::numeric,$7::bytea,$8::bytea,$9::bytea,$10::bytea)"

type iniciadorAccesoRutas interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type RepositorioAccesoRutas struct{ pool iniciadorAccesoRutas }

var _ dietasports.ConsumidorAccesoRutasDietas = (*RepositorioAccesoRutas)(nil)

func NuevoRepositorioAccesoRutas(p *pgxpool.Pool) (*RepositorioAccesoRutas, error) {
	if nuloAccesoRutas(p) {
		return nil, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	return &RepositorioAccesoRutas{p}, nil
}
func (r *RepositorioAccesoRutas) ConsumirAccesoRutasDietas(ctx context.Context, o dietasports.OrdenConsumoAccesoRutasDietas) (dietasports.ReciboAccesoRutasDietas, error) {
	if r == nil || nuloAccesoRutas(r.pool) || ctx == nil || ctx.Err() != nil || o.Material.ValidarEstructura() != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	d, err := o.Solicitud.Datos()
	if err != nil || d.Recurso.Validar() != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	h, err := d.Recurso.HuellaContextoAutorizacionSHA256()
	z := o.Material.ResumenCapacidad()
	if err != nil || z.Operacion() != d.Accion || z.EfectoRef() != d.Recurso.Referencia || z.EfectoHuellaSHA256() != h || z.AudienciaConsumo() != dietasports.AudienciaAccesoRutasDietas {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return dietasports.ReciboAccesoRutasDietas{}, errorAccesoRutas(ctx, err)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	m := o.Material
	var recibo dietasports.ReciboAccesoRutasDietas
	err = tx.QueryRow(ctx, funcionConsumirAccesoRutas, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&recibo.DecisionRef, &recibo.EfectoRef, &recibo.HuellaEfectoSHA256, &recibo.ConsumoHuellaSHA256, &recibo.AuditoriaRef, &recibo.ConsumidaEn, &recibo.ConsumoNuevo)
	if err != nil {
		return dietasports.ReciboAccesoRutasDietas{}, errorAccesoRutas(ctx, err)
	}
	recibo.ConsumidaEn = recibo.ConsumidaEn.UTC()
	digest, errDigest := hex.DecodeString(recibo.ConsumoHuellaSHA256)
	if recibo.DecisionRef != z.DecisionRef() || recibo.EfectoRef != z.EfectoRef() ||
		recibo.HuellaEfectoSHA256 != z.EfectoHuellaSHA256() || !recibo.ConsumoNuevo ||
		recibo.AuditoriaRef == "" || errDigest != nil || len(digest) != 32 ||
		hex.EncodeToString(digest) != recibo.ConsumoHuellaSHA256 ||
		recibo.ConsumidaEn.Before(z.EmitidaEn()) || !recibo.ConsumidaEn.Before(z.ExpiraEn()) {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return dietasports.ReciboAccesoRutasDietas{}, errorAccesoRutas(ctx, err)
	}
	// SQL confirma consumo y auditoría antes de retornar; se conserva el recibo real.
	return recibo, nil
}
func errorAccesoRutas(ctx context.Context, e error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && (pg.Code == "42501" || pg.Code == "PDI03") {
		return dietasports.ErrAccesoRutasDietasDenegado
	}
	return dietasports.ErrAccesoRutasDietasNoDisponible
}
func nuloAccesoRutas(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Pointer || x.Kind() == reflect.Interface || x.Kind() == reflect.Map || x.Kind() == reflect.Slice || x.Kind() == reflect.Func || x.Kind() == reflect.Chan) && x.IsNil()
}
