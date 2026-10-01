package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const registrarAltaPlanCTSQL = `SELECT vec_personal.registrar_acto_plan_incorporacion_ct_v1('alta',$1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const registrarHechoPlanCTSQL = `SELECT vec_personal.registrar_acto_plan_incorporacion_ct_v1('hecho',$1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

// El servicio de actos B2 sigue siendo la autoridad de alta/relación/ocupación.
// Esta fachada enlaza su efecto al plan y coteja su fuente en esa misma TX.
type RepositorioActosPlanIncorporacionCTPostgreSQL struct {
	base *RepositorioRegistroEmpleadoB2PostgreSQL
}

var _ ports.RepositorioActosRegistroEmpleadoB2 = (*RepositorioActosPlanIncorporacionCTPostgreSQL)(nil)

func NuevoRepositorioActosPlanIncorporacionCTPostgreSQL(pool *pgxpool.Pool) (*RepositorioActosPlanIncorporacionCTPostgreSQL, error) {
	return nuevoRepositorioActosPlanCT(pool)
}
func nuevoRepositorioActosPlanCT(pool iniciadorRegistroEmpleadoB2) (*RepositorioActosPlanIncorporacionCTPostgreSQL, error) {
	r, e := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	if e != nil {
		return nil, e
	}
	return &RepositorioActosPlanIncorporacionCTPostgreSQL{base: r}, nil
}
func (r *RepositorioActosPlanIncorporacionCTPostgreSQL) RegistrarEmpleadoRRHH(ctx context.Context, o ports.OrdenAltaEmpleadoB2) (ports.ResultadoAltaEmpleadoB2, error) {
	var vacio ports.ResultadoAltaEmpleadoB2
	if r == nil || r.base == nil || !materialActoEmpleadoB2Valido(o.Material, o.Autorizacion, "alta", domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, e := ejecutarRegistroEmpleadoB2(ctx, r.base.pool, registrarAltaPlanCTSQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaActoEmpleadoB2, func(b []byte) (ports.ResultadoAltaEmpleadoB2, error) {
		var z ports.ResultadoAltaEmpleadoB2
		if formaReciboActoB2(b, "alta") != nil || decodificarJSONRegistroB2(b, &z) != nil || !reciboActoEmpleadoB2Valido(z.Recibo, z.AccesoActual, o.Material, o.Autorizacion, true) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if e != nil {
		return vacio, errorDominioRegistroEmpleadoB2(e)
	}
	return resultado, nil
}
func (r *RepositorioActosPlanIncorporacionCTPostgreSQL) RegistrarHechoEmpleadoRRHH(ctx context.Context, o ports.OrdenHechoEmpleadoB2) (ports.ResultadoHechoEmpleadoB2, error) {
	var vacio ports.ResultadoHechoEmpleadoB2
	if r == nil || r.base == nil || !materialActoEmpleadoB2Valido(o.Material, o.Autorizacion, "hecho", domain.AccionHechoEmpleadoB2, domain.AudienciaHechoEmpleadoB2) {
		return vacio, domain.ErrRegistroEmpleadoB2Invalido
	}
	resultado, e := ejecutarRegistroEmpleadoB2(ctx, r.base.pool, registrarHechoPlanCTSQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaActoEmpleadoB2, func(b []byte) (ports.ResultadoHechoEmpleadoB2, error) {
		var z ports.ResultadoHechoEmpleadoB2
		if formaReciboActoB2(b, "hecho") != nil || decodificarJSONRegistroB2(b, &z) != nil || !reciboActoEmpleadoB2Valido(z.Recibo, z.AccesoActual, o.Material, o.Autorizacion, false) {
			return vacio, errRegistroEmpleadoB2NoDisponible
		}
		return z, nil
	})
	if e != nil {
		return vacio, errorDominioRegistroEmpleadoB2(e)
	}
	return resultado, nil
}
