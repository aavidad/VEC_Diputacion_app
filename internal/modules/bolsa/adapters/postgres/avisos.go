package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ConsultaAvisosRRHHPostgreSQL struct{ pool *pgxpool.Pool }

var _ puertosbolsa.ConsultaAvisosRRHH = (*ConsultaAvisosRRHHPostgreSQL)(nil)

func NuevaConsultaAvisosRRHHPostgreSQL(pool *pgxpool.Pool) (*ConsultaAvisosRRHHPostgreSQL, error) {
	if pool == nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	return &ConsultaAvisosRRHHPostgreSQL{pool: pool}, nil
}

func (c *ConsultaAvisosRRHHPostgreSQL) ListarAvisosRRHH(ctx context.Context, corte time.Time, offset, limite int) ([]dominiobolsa.AvisoRRHH, error) {
	if c == nil || c.pool == nil || ctx == nil || corte.IsZero() || offset < 0 || limite < 1 || limite > 101 {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	filas, err := c.pool.Query(ctx, `SELECT tipo,bolsa_ref,referencia,detalle,fecha FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1($1) ORDER BY fecha DESC,tipo,referencia LIMIT $2 OFFSET $3`, corte, limite, offset)
	if err != nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	defer filas.Close()
	salida := make([]dominiobolsa.AvisoRRHH, 0, limite)
	for filas.Next() {
		var aviso dominiobolsa.AvisoRRHH
		var detalle []byte
		if err := filas.Scan(&aviso.Tipo, &aviso.BolsaRef, &aviso.Referencia, &detalle, &aviso.Fecha); err != nil || json.Unmarshal(detalle, &aviso.Detalle) != nil || aviso.Validar() != nil {
			return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
		}
		salida = append(salida, aviso)
	}
	if filas.Err() != nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	return salida, nil
}

func (c *ConsultaAvisosRRHHPostgreSQL) ContarAvisosRRHH(ctx context.Context, corte time.Time) (map[string]int, error) {
	if c == nil || c.pool == nil || ctx == nil || corte.IsZero() {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	filas, err := c.pool.Query(ctx, `SELECT tipo,count(*) FROM vec_bolsa_llamamientos.consultar_avisos_rrhh_v1($1) GROUP BY tipo`, corte)
	if err != nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	defer filas.Close()
	conteos := map[string]int{dominiobolsa.AvisoSaltoOrden: 0, dominiobolsa.AvisoTresAnos: 0}
	for filas.Next() {
		var tipo string
		var total int
		if err := filas.Scan(&tipo, &total); err != nil || (tipo != dominiobolsa.AvisoSaltoOrden && tipo != dominiobolsa.AvisoTresAnos) || total < 0 {
			return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
		}
		conteos[tipo] = total
	}
	if filas.Err() != nil {
		return nil, puertosbolsa.ErrConsultaAvisosNoDisponible
	}
	return conteos, nil
}
