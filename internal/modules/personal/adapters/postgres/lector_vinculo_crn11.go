package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

const consultaVinculoPropioCRN11SQL = `SELECT vec_personal.consultar_vinculo_propio_historico_crn11_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const maxRespuestaVinculoCRN11 = 8 << 10

// RepositorioVinculoPropioCRN11PostgreSQL necesita la función y consumidor V3
// nominales. Su ausencia mantiene la lectura cerrada; nunca consulta tablas
// directamente ni cae a la ficha, Dietas o la consulta de RRHH.
type RepositorioVinculoPropioCRN11PostgreSQL struct{ pool iniciadorRegistroEmpleadoB2 }

func NuevoRepositorioVinculoPropioCRN11PostgreSQL(pool *pgxpool.Pool) (*RepositorioVinculoPropioCRN11PostgreSQL, error) {
	return nuevoRepositorioVinculoCRN11(pool)
}
func nuevoRepositorioVinculoCRN11(pool iniciadorRegistroEmpleadoB2) (*RepositorioVinculoPropioCRN11PostgreSQL, error) {
	if nuloRegistroEmpleadoB2(pool) {
		return nil, domain.ErrVinculoCRN11NoDisponible
	}
	return &RepositorioVinculoPropioCRN11PostgreSQL{pool: pool}, nil
}
func (r *RepositorioVinculoPropioCRN11PostgreSQL) ConsultarVinculoPropioCRN11(ctx context.Context, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
	var cero ports.ResultadoVinculoPropioCRN11
	if r == nil || ctx == nil || nuloRegistroEmpleadoB2(r.pool) {
		return cero, domain.ErrVinculoCRN11NoDisponible
	}
	m := o.Material
	reconstruido, err := domain.NuevoMaterialVinculoPropioCRN11(domain.SolicitudVinculoPropioCRN11{Actor: m.Actor(), EmpleadoRef: m.EmpleadoRef()})
	if err != nil || !bytes.Equal(reconstruido.Canonico(), m.Canonico()) || !autorizacionVinculoCRN11PostgresValida(o) {
		return cero, domain.ErrVinculoCRN11Invalido
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaVinculoPropioCRN11SQL, m.Canonico(), o.Autorizacion, maxRespuestaVinculoCRN11, func(b []byte) (ports.ResultadoVinculoPropioCRN11, error) { return decodificarVinculoCRN11(b, o) })
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return cero, err
		case errors.Is(err, errRegistroEmpleadoB2Denegado):
			return cero, domain.ErrVinculoCRN11Denegado
		default:
			return cero, domain.ErrVinculoCRN11NoDisponible
		}
	}
	return resultado, nil
}
func autorizacionVinculoCRN11PostgresValida(o ports.OrdenVinculoPropioCRN11) bool {
	m, a := o.Material, o.Autorizacion
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	h, errH := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && errH == nil && a.ValidarEstructura() == nil && bytes.Equal(canon, a.ContextoActorCanonico()) &&
		a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionVinculoPropioCRN11 && x.AudienciaConsumo() == domain.AudienciaVinculoPropioCRN11 && x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}
func decodificarVinculoCRN11(b []byte, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
	var cero ports.ResultadoVinculoPropioCRN11
	if len(b) == 0 || verificarJSONOrganizacionHistorica(b) != nil {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	var raiz map[string]json.RawMessage
	if json.Unmarshal(b, &raiz) != nil || !clavesRegistroB2(raiz, []string{"vinculo", "evidencia"}, nil) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	var v, e map[string]json.RawMessage
	if json.Unmarshal(raiz["vinculo"], &v) != nil || json.Unmarshal(raiz["evidencia"], &e) != nil ||
		!clavesRegistroB2(v, []string{"persona_ref", "empleado_ref", "vinculo_ref", "fuente_ref", "version"}, nil) ||
		!clavesRegistroB2(e, []string{"recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"}, nil) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	var r ports.ResultadoVinculoPropioCRN11
	if decodificarJSONRegistroB2(b, &r) != nil || r.Vinculo.ValidarPara(o.Material) != nil || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) || !o.Material.VigenteParaLectura(r.Evidencia.ConsultadaEn) {
		return cero, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

var _ ports.RepositorioVinculoPropioCRN11 = (*RepositorioVinculoPropioCRN11PostgreSQL)(nil)
