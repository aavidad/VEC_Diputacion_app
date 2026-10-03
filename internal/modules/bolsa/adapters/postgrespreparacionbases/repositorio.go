package postgrespreparacionbases

import (
	"context"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresql "vec-diputacion-granada/internal/shared/postgresql"
	core "vec-diputacion-granada/internal/vec/ports"
)

const columnasPreparacion = `resultado,preparacion_ref,revision,huella_material_sha256,material_canonico,recibo_ref,historia_ref,auditoria_efecto_ref,evento_ref,huella_intencion_sha256,confirmada_en,decision_ref,consumo_huella_sha256,auditoria_acceso_ref,recibo_acceso_ref,correlacion_ref,accedida_en`
const consultaGuardar = `SELECT ` + columnasPreparacion + ` FROM vec_bolsa_convocatorias.guardar_preparacion_bases_v3($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
const consultaObtener = `SELECT ` + columnasPreparacion + ` FROM vec_bolsa_convocatorias.obtener_preparacion_bases_v3($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`

type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// Repositorio usa cuentas/pools nominales segregados. Consultar tambien
// requiere READ WRITE porque consume autorizacion y conserva su auditoria.
type Repositorio struct {
	escritura iniciador
	lectura   iniciador
}

func NuevoRepositorio(escritura, lectura *pgxpool.Pool) (*Repositorio, error) {
	if escritura == lectura {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return nuevoRepositorio(escritura, lectura)
}

func nuevoRepositorio(escritura, lectura iniciador) (*Repositorio, error) {
	if nulo(escritura) || nulo(lectura) {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return &Repositorio{escritura, lectura}, nil
}

func (r *Repositorio) GuardarPreparacionBasesV3(ctx context.Context, o ports.OrdenGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	var cero, resultado ports.ResultadoPreparacionBasesV3
	if ctx == nil || r == nil || nulo(r.escritura) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := bolsa.ValidarMaterialGuardarPreparacionBasesV3(o); err != nil {
		return cero, err
	}
	p, err := bolsa.PrepararGuardadoPreparacionBasesV3(o.Solicitud)
	if err != nil {
		return cero, err
	}
	parametros := append([]any{string(p.Canonico), p.Contenido}, parametrosV3(o.Autorizacion)...)
	defer limpiarParametros(parametros)
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		resultado, fallo = r.ejecutar(ctx, r.escritura, consultaGuardar, parametros, o.Solicitud.Ambito, func(v ports.ResultadoPreparacionBasesV3) error {
			return bolsa.ValidarResultadoGuardarPreparacionBasesV3(o, v)
		})
		return fallo
	})
	if err != nil {
		return cero, clasificarError(ctx, err)
	}
	return resultado, nil
}

func (r *Repositorio) ConsultarPreparacionBasesV3(ctx context.Context, o ports.OrdenConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	var cero, resultado ports.ResultadoPreparacionBasesV3
	if ctx == nil || r == nil || nulo(r.lectura) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := bolsa.ValidarMaterialConsultarPreparacionBasesV3(o); err != nil {
		return cero, err
	}
	p, err := bolsa.PrepararConsultaPreparacionBasesV3(o.Solicitud)
	if err != nil {
		return cero, err
	}
	parametros := append([]any{string(p.Canonico)}, parametrosV3(o.Autorizacion)...)
	defer limpiarParametros(parametros)
	err = postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var fallo error
		resultado, fallo = r.ejecutar(ctx, r.lectura, consultaObtener, parametros, o.Solicitud.Ambito, func(v ports.ResultadoPreparacionBasesV3) error {
			return bolsa.ValidarResultadoConsultarPreparacionBasesV3(o, v)
		})
		return fallo
	})
	if err != nil {
		return cero, clasificarError(ctx, err)
	}
	return resultado, nil
}

func parametrosV3(a core.ExportacionMaterialConsumoAutorizacionAtestadaV3) []any {
	return []any{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PersonaVersion(), a.PerfilVersion(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
}

func limpiarParametros(ps []any) {
	for _, p := range ps {
		if b, ok := p.([]byte); ok {
			for i := range b {
				b[i] = 0
			}
		}
	}
}

func clasificarError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrResultadoPreparacionBasesInvalido) {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return ports.ErrPreparacionBasesDenegada
	}
	return ports.ErrPreparacionBasesNoDisponible
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

var _ ports.RepositorioPreparacionBasesV3 = (*Repositorio)(nil)
