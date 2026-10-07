package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRegistro = errors.New("orden_postgres_no_disponible")

// MaterialV3 is private infrastructure input to the real AD143 guard, not a
// capability factory. SQL must verify all pieces against current V3 authority.
// The provider is the central authority facade, never an HTTP material DTO.
type MaterialV3 struct {
	Capacidad, Decision, Motivo, Contexto, Payload, Sobre, Evidencia, Raiz []byte
	PersonaVersion, PerfilVersion                                          uint64
}
type MaterializadorV3 interface {
	MaterializarOrdenV3(context.Context, domain.Orden, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (MaterialV3, error)
}
type Registro struct {
	pool     *pgxpool.Pool
	material MaterializadorV3
}

func Nuevo(pool *pgxpool.Pool, m MaterializadorV3) (*Registro, error) {
	if pool == nil || m == nil {
		return nil, ErrRegistro
	}
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, ErrRegistro
	}
	return &Registro{pool, m}, nil
}

// ComprometerOrden issues exactly one server transaction. No FS/KMS call occurs
// here. SQL's absence, denied current guard or rollback remains an error.
func (r *Registro) ComprometerOrden(ctx context.Context, o domain.Orden, s vecdomain.SolicitudAutorizacionLigadaV3, d vecdomain.DecisionAutorizacionLigadaV3, c vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) error {
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil || d.ValidarPara(s) != nil {
		return ErrRegistro
	}
	granted, _, grantErr := d.Resultado()
	sd, requestErr := s.Datos()
	od, orderErr := o.Datos()
	confirmation, confirmationErr := c.Datos()
	decisionHash, hashErr := vecdomain.HuellaSHA256DecisionAutorizacionV3(d)
	if grantErr != nil || !granted || requestErr != nil || orderErr != nil || confirmationErr != nil || hashErr != nil ||
		sd.Accion != "administracion.copias.orden.emitir" || sd.Finalidad != "emitir_orden_copia" ||
		sd.Recurso.ModuloID != "administracion" || sd.Recurso.Tipo != "orden_copia" || sd.Recurso.Referencia != od.Orden ||
		decisionHash != od.DecisionSHA256 || confirmation.DecisionRef != od.DecisionV3 || confirmation.DecisionHuellaSHA256 != decisionHash {
		return ErrRegistro
	}
	b, err := o.Bytes()
	if err != nil {
		return ErrRegistro
	}
	plan, err := o.PlanBytes()
	if err != nil {
		return ErrRegistro
	}
	m, err := r.material.MaterializarOrdenV3(ctx, o, s, d, c)
	if err != nil || !materialValido(m) {
		return ErrRegistro
	}
	canonical, err := vecdomain.RepresentacionCanonicaDecisionAutorizacionV3(d)
	if err != nil || !bytes.Equal(canonical, m.Decision) {
		return ErrRegistro
	}
	var returned []byte
	err = r.pool.QueryRow(ctx, `SELECT vec_administracion_copias.comprometer_orden_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, b, plan, m.Capacidad, m.Decision, m.Motivo, m.Contexto, m.PersonaVersion, m.PerfilVersion, m.Payload, m.Sobre, m.Evidencia, m.Raiz).Scan(&returned)
	if err != nil || !bytes.Equal(returned, b) || ctx.Err() != nil {
		return ErrRegistro
	}
	return nil
}
func (r *Registro) LeerOrdenComprometida(ctx context.Context, ref string) (domain.Orden, error) {
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil || len(ref) == 0 || len(ref) > 128 {
		return domain.Orden{}, ErrRegistro
	}
	var b []byte
	if err := r.pool.QueryRow(ctx, `SELECT vec_administracion_copias.leer_orden_comprometida_v1($1)`, ref).Scan(&b); err != nil || len(b) == 0 || len(b) > 16384 {
		return domain.Orden{}, ErrRegistro
	}
	var raw struct {
		Esquema string       `json:"esquema"`
		Datos   domain.Datos `json:"datos"`
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&raw) != nil || raw.Esquema != domain.Esquema {
		return domain.Orden{}, ErrRegistro
	}
	o, err := domain.Nueva(raw.Datos)
	if err != nil {
		return domain.Orden{}, ErrRegistro
	}
	actual, _ := o.Bytes()
	if !bytes.Equal(actual, b) || raw.Datos.Orden != ref || ctx.Err() != nil {
		return domain.Orden{}, ErrRegistro
	}
	return o, nil
}
func materialValido(m MaterialV3) bool {
	if m.PersonaVersion == 0 || m.PerfilVersion == 0 {
		return false
	}
	for _, b := range [][]byte{m.Capacidad, m.Decision, m.Motivo, m.Contexto, m.Payload, m.Sobre, m.Evidencia, m.Raiz} {
		if len(b) == 0 || len(b) > 1<<20 {
			return false
		}
	}
	return true
}

var _ puerto.ConsumidorV3 = (*Registro)(nil)
var _ puerto.LectorCommit = (*Registro)(nil)
