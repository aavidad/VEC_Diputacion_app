package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaFichaPropiaSQL       = `SELECT vec_personal.consultar_ficha_propia_empleado_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	registrarDenegacionPropiaSQL = `SELECT vec_personal.registrar_denegacion_ficha_propia_v1($1::text,$2::text,$3::smallint,NULLIF($4::text,''))`
	maxRespuestaFichaPropia      = 256 << 10
	preflightFichaPropiaEjecutor = `SELECT has_function_privilege('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND NOT has_function_privilege('vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)','EXECUTE')`
	preflightFichaPropiaFrontera = `SELECT has_function_privilege('vec_personal.registrar_denegacion_ficha_propia_v1(text,text,smallint,text)','EXECUTE') AND NOT has_function_privilege('vec_personal.consultar_ficha_propia_empleado_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`
)

var (
	ErrFichaPropiaPostgreSQLNoDisponible = errors.New("personal: ficha propia PostgreSQL no disponible")
	correlacionDenegacionFichaPropia     = regexp.MustCompile(`^(corr_no_disponible|corr_[0-9a-f]{32})$`)
	actorDenegacionFichaPropia           = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,512}$`)
)

var _ ports.RepositorioFichaPropia = (*RepositorioRegistroEmpleadoB2PostgreSQL)(nil)

// ConsultarFichaPropia ejecuta la función nominal con el LOGIN ejecutor de
// Personal; consumo V3, comprobación de la proyección, lectura y recibo van
// en la misma transacción SERIALIZABLE, que solo confirma si la respuesta
// supera su contrato.
func (r *RepositorioRegistroEmpleadoB2PostgreSQL) ConsultarFichaPropia(ctx context.Context, o ports.OrdenFichaPropia) (ports.ResultadoFichaPropia, error) {
	var vacio ports.ResultadoFichaPropia
	if r == nil || !materialFichaPropiaValido(o.Material, o.Autorizacion) {
		return vacio, domain.ErrFichaPropiaInvalida
	}
	resultado, err := ejecutarRegistroEmpleadoB2(ctx, r.pool, consultaFichaPropiaSQL, o.Material.Canonico(), o.Autorizacion, maxRespuestaFichaPropia, func(bruto []byte) (ports.ResultadoFichaPropia, error) {
		return decodificarFichaPropia(bruto, o)
	})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return vacio, err
		case errors.Is(err, errRegistroEmpleadoB2Denegado):
			return vacio, domain.ErrFichaPropiaDenegada
		default:
			return vacio, domain.ErrFichaPropiaNoDisponible
		}
	}
	return resultado, nil
}

// PreflightEjecutorFichaPropia comprueba que el LOGIN del pool ejecuta la
// consulta y no el registro de denegaciones.
func PreflightEjecutorFichaPropia(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	return preflightFichaPropia(ctx, pool, preflightFichaPropiaEjecutor)
}

func materialFichaPropiaValido(m domain.MaterialFichaPropia, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	reconstruido, err := domain.NuevoMaterialFichaPropia(domain.SolicitudFichaPropia{Corte: m.Corte(), Actor: m.Actor()})
	if err != nil || !bytes.Equal(reconstruido.Canonico(), m.Canonico()) || a.ValidarEstructura() != nil {
		return false
	}
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	actor := m.Actor()
	return err == nil && a.PersonaVersion() == actor.Instantanea.PersonaVersion && a.PerfilVersion() == actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionFichaPropia && x.AudienciaConsumo() == domain.AudienciaFichaPropia &&
		x.EfectoRef() == m.EmpleadoRef() && x.EfectoHuellaSHA256() == h
}

func decodificarFichaPropia(bruto []byte, o ports.OrdenFichaPropia) (ports.ResultadoFichaPropia, error) {
	var vacio ports.ResultadoFichaPropia
	if len(bruto) == 0 || verificarJSONOrganizacionHistorica(bruto) != nil {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	var raiz map[string]json.RawMessage
	if json.Unmarshal(bruto, &raiz) != nil || !clavesRegistroB2(raiz, []string{"ficha", "evidencia"}, nil) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	var ficha map[string]json.RawMessage
	if json.Unmarshal(raiz["ficha"], &ficha) != nil || !clavesRegistroB2(ficha, []string{"corte", "relaciones", "servicios"}, nil) ||
		!filasExactasFichaPropia(ficha["relaciones"], []string{"inicio", "fin", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion"}) ||
		!filasExactasFichaPropia(ficha["servicios"], []string{"inicio", "fin", "clase", "dias", "estado"}) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	var r ports.ResultadoFichaPropia
	if err := decodificarJSONRegistroB2(bruto, &r); err != nil || r.Ficha.ValidarPara(o.Material) != nil || !evidenciaRegistroB2Valida(r.Evidencia, o.Autorizacion) {
		return vacio, errRegistroEmpleadoB2NoDisponible
	}
	return r, nil
}

func filasExactasFichaPropia(bruto json.RawMessage, claves []string) bool {
	var filas []map[string]json.RawMessage
	if len(bruto) == 0 || bruto[0] != '[' || json.Unmarshal(bruto, &filas) != nil {
		return false
	}
	for _, fila := range filas {
		if !clavesRegistroB2(fila, claves, nil) {
			return false
		}
	}
	return true
}

// RegistroDenegacionFichaPropiaPostgreSQL inscribe denegaciones de frontera
// con un LOGIN miembro exclusivo de vec_personal_registrador_frontera.
type RegistroDenegacionFichaPropiaPostgreSQL struct {
	pool *pgxpool.Pool
}

var _ ports.RegistroDenegacionFichaPropia = (*RegistroDenegacionFichaPropiaPostgreSQL)(nil)

func NuevoRegistroDenegacionFichaPropiaPostgreSQL(pool *pgxpool.Pool) (*RegistroDenegacionFichaPropiaPostgreSQL, error) {
	if pool == nil {
		return nil, ErrFichaPropiaPostgreSQLNoDisponible
	}
	return &RegistroDenegacionFichaPropiaPostgreSQL{pool: pool}, nil
}

func (r *RegistroDenegacionFichaPropiaPostgreSQL) PreflightFichaPropia(ctx context.Context) error {
	if r == nil {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	return preflightFichaPropia(ctx, r.pool, preflightFichaPropiaFrontera)
}

func (r *RegistroDenegacionFichaPropiaPostgreSQL) RegistrarDenegacionFichaPropia(ctx context.Context, d ports.DenegacionFichaPropia) error {
	if r == nil || r.pool == nil || ctx == nil || !correlacionDenegacionFichaPropia.MatchString(d.CorrelacionRef) ||
		d.EstadoHTTP < 400 || d.EstadoHTTP > 599 || (d.ActorRef != "" && !actorDenegacionFichaPropia.MatchString(d.ActorRef)) {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	var registrada bool
	if err := r.pool.QueryRow(ctx, registrarDenegacionPropiaSQL, d.CorrelacionRef, d.Motivo, int16(d.EstadoHTTP), d.ActorRef).Scan(&registrada); err != nil || !registrada {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	return nil
}

type consultorFilaFichaPropia interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func preflightFichaPropia(ctx context.Context, pool consultorFilaFichaPropia, consulta string) error {
	if ctx == nil || nuloRegistroEmpleadoB2(pool) {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	var ok bool
	if err := pool.QueryRow(ctx, consulta).Scan(&ok); err != nil || !ok {
		return ErrFichaPropiaPostgreSQLNoDisponible
	}
	return nil
}
