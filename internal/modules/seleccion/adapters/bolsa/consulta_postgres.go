// Package bolsa conecta Selección con la lectura nominal de versiones de Bolsa.
package bolsa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	postgresql "vec-diputacion-granada/internal/shared/postgresql"
)

const consultaVersionV3 = `SELECT resultado,version_canonica,huella_version_sha256,decision_ref,consumo_huella_sha256,auditoria_ref,recibo_ref,correlacion_ref,consultada_en FROM vec_bolsa_convocatorias.obtener_version_exacta_v3($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const ajustesVersionV3 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`
const maximoVersionCanonica = 32 << 20

type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioVersiones struct{ pool iniciador }

func NuevoRepositorioVersiones(pool *pgxpool.Pool) (*RepositorioVersiones, error) {
	return nuevoRepositorioVersiones(pool)
}
func nuevoRepositorioVersiones(pool iniciador) (*RepositorioVersiones, error) {
	if nulo(pool) {
		return nil, ports.ErrConvocatoriaNoDisponible
	}
	return &RepositorioVersiones{pool}, nil
}

func (r *RepositorioVersiones) ObtenerVersionExactaV3(ctx context.Context, o ports.OrdenConsultaConvocatoria) (ports.ResultadoVersionConvocatoria, error) {
	var cero, result ports.ResultadoVersionConvocatoria
	if ctx == nil || r == nil || nulo(r.pool) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := application.ValidarMaterialConsultaConvocatoria(o); err != nil {
		return cero, err
	}
	p, err := application.PrepararConsultaConvocatoria(o.Solicitud)
	if err != nil {
		return cero, err
	}
	a := o.Autorizacion
	parametros := []any{string(p.Canonico), a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PersonaVersion(), a.PerfilVersion(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, p := range parametros {
			if b, ok := p.([]byte); ok {
				for i := range b {
					b[i] = 0
				}
			}
		}
	}()
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var consultaErr error
		result, consultaErr = r.consultar(ctx, o, parametros)
		return consultaErr
	})
	if err != nil {
		if cancelacion := ctx.Err(); cancelacion != nil {
			return cero, cancelacion
		}
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	return result, nil
}

func (r *RepositorioVersiones) consultar(ctx context.Context, o ports.OrdenConsultaConvocatoria, parametros []any) (ports.ResultadoVersionConvocatoria, error) {
	var cero, result ports.ResultadoVersionConvocatoria
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return cero, err
	}
	if nulo(tx) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	defer func() {
		fin, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancelar()
		_ = tx.Rollback(fin)
	}()
	if _, err = tx.Exec(ctx, ajustesVersionV3); err != nil {
		return cero, err
	}
	var canon []byte
	var huella *string
	e := &result.Evidencia
	if err = tx.QueryRow(ctx, consultaVersionV3, parametros...).Scan(&result.Estado, &canon, &huella, &e.DecisionRef, &e.ConsumoHuellaSHA256, &e.AuditoriaRef, &e.ReciboRef, &e.CorrelacionRef, &e.ConsultadaEn); err != nil {
		return cero, err
	}
	defer func() {
		for i := range canon {
			canon[i] = 0
		}
	}()
	if result.Estado == "obtenida" {
		if len(canon) == 0 || len(canon) > maximoVersionCanonica || huella == nil {
			return cero, ports.ErrRespuestaConvocatoriaInvalida
		}
		h := sha256.Sum256(canon)
		if hex.EncodeToString(h[:]) != *huella {
			return cero, ports.ErrRespuestaConvocatoriaInvalida
		}
		result.Version, err = bolsadomain.DecodificarVersionConvocatoriaGobernadaCanonica(canon)
		if err != nil {
			return cero, ports.ErrRespuestaConvocatoriaInvalida
		}
		result.HuellaVersionSHA256 = *huella
	} else if len(canon) != 0 || huella != nil {
		return cero, ports.ErrRespuestaConvocatoriaInvalida
	}
	if application.ValidarResultadoVersionConvocatoria(o, result) != nil {
		return cero, ports.ErrRespuestaConvocatoriaInvalida
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, err
	}
	return result, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}

var _ ports.RepositorioConsultaConvocatoria = (*RepositorioVersiones)(nil)
