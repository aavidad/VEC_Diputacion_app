package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// maximoFilasResumenBolsas acota lo que se acepta de la base antes de crecer.
const maximoFilasResumenBolsas = 200000

// LectorResumenBolsasPostgreSQL lee las dos funciones de conjunto de Bolsa
// 000082. Ambas comprueban en la base el rol de quien llama.
type LectorResumenBolsasPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.LectorResumenBolsas = (*LectorResumenBolsasPostgreSQL)(nil)
var _ ports.LectorResumenBolsasNominal = (*LectorResumenBolsasPostgreSQL)(nil)

func NuevoLectorResumenBolsasPostgreSQL(pool *pgxpool.Pool) (*LectorResumenBolsasPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &LectorResumenBolsasPostgreSQL{pool: pool}, nil
}

// consultaResumenBolsas es lo que necesitan las dos lecturas de una transacción.
type consultaResumenBolsas interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// LeerResumen lee situaciones y políticas en una misma transacción
// REPEATABLE READ de solo lectura: las dos ven la misma instantánea. Una
// participación NULL (constitución sin entradas) o cualquier fila
// incoherente hace fallar la lectura entera.
func (l *LectorResumenBolsasPostgreSQL) LeerResumen(ctx context.Context, corte time.Time) ([]ports.SituacionResumenParticipacion, map[string]dominiobolsa.PoliticaOrdenBolsa, error) {
	if l == nil || l.pool == nil || ctx == nil || corte.IsZero() {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := leerSituacionesResumen(ctx, tx, corte)
	if err != nil {
		return nil, nil, err
	}
	politicas, err := leerPoliticasResumen(ctx, tx, corte)
	if err != nil {
		return nil, nil, err
	}
	if tx.Commit(ctx) != nil {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	return filas, politicas, nil
}

func leerSituacionesResumen(ctx context.Context, consulta consultaResumenBolsas, corte time.Time) ([]ports.SituacionResumenParticipacion, error) {
	filas, err := consulta.Query(ctx, `SELECT bolsa_ref,categoria_ref,confirmada_en,instantanea_ref,version_instantanea,orden,participacion_ref,situacion,desde,fecha_disponible,
		cese_fecha_efecto,cese_disponible_desde,cese_en_restriccion,cese_trabajo_cesado
		FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1($1)`, corte.UTC())
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	var salida []ports.SituacionResumenParticipacion
	for filas.Next() {
		if len(salida) >= maximoFilasResumenBolsas {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var fila ports.SituacionResumenParticipacion
		var version, orden int64
		var situacion *string
		var desde, disponible *time.Time
		var efecto, disponibleCese *time.Time
		var restringida, cesado *bool
		if err := filas.Scan(&fila.BolsaRef, &fila.CategoriaRef, &fila.ConfirmadaEn, &fila.InstantaneaRef, &version, &orden, &fila.ParticipacionRef, &situacion, &desde, &disponible,
			&efecto, &disponibleCese, &restringida, &cesado); err != nil || version <= 0 || orden <= 0 || fila.ParticipacionRef == "" || fila.BolsaRef == "" || fila.CategoriaRef == "" || fila.ConfirmadaEn.IsZero() {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		fila.VersionInstantanea, fila.Orden = uint64(version), uint64(orden)
		if situacion != nil {
			if desde == nil {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			fila.Situacion = &ports.SituacionParticipacion{ParticipacionRef: fila.ParticipacionRef, Situacion: *situacion, Desde: *desde, FechaDisponible: disponible}
		}
		if efecto != nil {
			if disponibleCese == nil || restringida == nil || cesado == nil {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			estado, presente, err := validarEstadoCese(*efecto, *disponibleCese, *restringida, *cesado, corte)
			if err != nil || !presente {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			fila.Cese = &estado
		}
		salida = append(salida, fila)
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
}

func leerPoliticasResumen(ctx context.Context, consulta consultaResumenBolsas, en time.Time) (map[string]dominiobolsa.PoliticaOrdenBolsa, error) {
	filas, err := consulta.Query(ctx, `SELECT bolsa_ref,politica_ref,version_politica,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde
		FROM vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1($1)`, en.UTC())
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	salida := map[string]dominiobolsa.PoliticaOrdenBolsa{}
	for filas.Next() {
		if len(salida) >= maximoFilasResumenBolsas {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var p dominiobolsa.PoliticaOrdenBolsa
		var version int64
		if err := filas.Scan(&p.BolsaRef, &p.PoliticaRef, &version, &p.Criterio, &p.TipoLista, &p.Reposicion, &p.Provisional, &p.Rotulo, &p.Actor, &p.VigenteDesde); err != nil || version <= 0 {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		p.Version = uint64(version)
		if _, repetida := salida[p.BolsaRef]; repetida || p.Validar() != nil {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		salida[p.BolsaRef] = p
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
}

// LeerResumenNominal consume la decisión, inserta la auditoría común y lee
// situaciones, políticas y contadores bajo la misma transacción. El SQL B84
// no concede lectura directa de las tablas ni del staging al LOGIN.
func (l *LectorResumenBolsasPostgreSQL) LeerResumenNominal(ctx context.Context, accion string, ceseActivo bool, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResumenBolsasNominal, error) {
	var vacio ports.ResumenBolsasNominal
	if l == nil || l.pool == nil || ctx == nil || ctx.Err() != nil || material.ValidarEstructura() != nil ||
		(accion != ports.AccionRRHHBolsasConsultar && accion != ports.AccionRRHHEstadisticasConsultar) ||
		material.ResumenCapacidad().Operacion() != accion {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	audiencia, recurso := ports.AudienciaRRHHBolsasConsultar, "coleccion:bolsa:rrhh:bolsas"
	if accion == ports.AccionRRHHEstadisticasConsultar {
		audiencia, recurso = ports.AudienciaRRHHEstadisticasConsultar, "coleccion:bolsa:rrhh:estadisticas"
	}
	if material.ResumenCapacidad().AudienciaConsumo() != audiencia || material.ResumenCapacidad().EfectoRef() != recurso {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	var crudo []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`,
		accion, ceseActivo, material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI()).Scan(&crudo)
	if err != nil {
		return vacio, errorResumenNominal(err)
	}
	if len(crudo) == 0 || len(crudo) > 32<<20 {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	resultado, err := decodificarResumenNominal(crudo, accion)
	if err != nil || ctx.Err() != nil || tx.Commit(ctx) != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	return resultado, nil
}

func errorResumenNominal(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return vd.ErrAutorizacionDenegada
	}
	if errors.As(err, &pg) && pg.Code == "VBR04" {
		return ports.ErrBolsaRRHHNoEncontrada
	}
	if errors.As(err, &pg) && pg.Code == "VBR09" {
		return ports.ErrCursorRRHHNoEncontrado
	}
	return ports.ErrResumenBolsasNoDisponible
}

type bolsaAgregadaSQL struct {
	BolsaRef     string     `json:"bolsa_ref"`
	CategoriaRef string     `json:"categoria_ref"`
	ConfirmadaEn time.Time  `json:"confirmada_en"`
	VigenteHasta *time.Time `json:"vigente_hasta"`
	Politica     struct {
		BolsaRef     string    `json:"bolsa_ref"`
		PoliticaRef  string    `json:"politica_ref"`
		Version      int64     `json:"version_politica"`
		Criterio     string    `json:"criterio"`
		TipoLista    string    `json:"tipo_lista"`
		Reposicion   string    `json:"reposicion"`
		Provisional  bool      `json:"provisional"`
		Rotulo       string    `json:"rotulo"`
		Actor        string    `json:"actor"`
		VigenteDesde time.Time `json:"vigente_desde"`
	} `json:"politica"`
	PorEstado           map[string]int `json:"por_estado"`
	LlamamientosEnCurso int            `json:"llamamientos_en_curso"`
}

type estadisticasAgregadasSQL struct {
	Bolsas struct {
		Total       int `json:"total"`
		Vigentes    int `json:"vigentes"`
		Sustituidas int `json:"sustituidas"`
	} `json:"bolsas"`
	Personas struct {
		Total     int            `json:"total"`
		PorEstado map[string]int `json:"por_estado"`
	} `json:"personas"`
	Llamamientos struct {
		EnCurso             int            `json:"en_curso"`
		HistoricoDisponible bool           `json:"historico_disponible"`
		Total               *int           `json:"total"`
		PorCanal            map[string]int `json:"por_canal"`
		PorResultado        map[string]int `json:"por_resultado"`
	} `json:"llamamientos"`
	PorBolsa []struct {
		BolsaRef     string         `json:"bolsa_ref"`
		CategoriaRef string         `json:"categoria_ref"`
		TipoLista    string         `json:"tipo_lista"`
		Vigente      bool           `json:"vigente"`
		Total        int            `json:"total"`
		PorEstado    map[string]int `json:"por_estado"`
	} `json:"por_bolsa"`
}

func decodificarJSONResumenExclusivo(raw []byte, destino any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destino); err != nil {
		return err
	}
	var sobrante any
	if err := decoder.Decode(&sobrante); err != io.EOF {
		return ports.ErrResumenBolsasNoDisponible
	}
	return nil
}

func decodificarResumenNominal(crudo []byte, accion string) (ports.ResumenBolsasNominal, error) {
	var vacio ports.ResumenBolsasNominal
	var campos map[string]json.RawMessage
	if json.Unmarshal(crudo, &campos) != nil || len(campos) != 2 || campos["generado_en"] == nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	var generado time.Time
	if json.Unmarshal(campos["generado_en"], &generado) != nil || generado.IsZero() {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	out := ports.ResumenBolsasNominal{GeneradoEn: generado}
	if accion == ports.AccionRRHHBolsasConsultar {
		var bolsas []bolsaAgregadaSQL
		if campos["bolsas"] == nil || decodificarJSONResumenExclusivo(campos["bolsas"], &bolsas) != nil ||
			len(bolsas) > 10000 {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		vistas := make(map[string]bool, len(bolsas))
		for _, b := range bolsas {
			politica := dominiobolsa.PoliticaOrdenBolsa{BolsaRef: b.Politica.BolsaRef, PoliticaRef: b.Politica.PoliticaRef,
				Version: uint64(b.Politica.Version), Criterio: b.Politica.Criterio, TipoLista: b.Politica.TipoLista,
				Reposicion: b.Politica.Reposicion, Provisional: b.Politica.Provisional, Rotulo: b.Politica.Rotulo,
				Actor: b.Politica.Actor, VigenteDesde: b.Politica.VigenteDesde}
			if b.BolsaRef == "" || b.CategoriaRef == "" || b.ConfirmadaEn.IsZero() || b.Politica.Version <= 0 ||
				politica.BolsaRef != b.BolsaRef || politica.Validar() != nil || vistas[b.BolsaRef] ||
				!conteosResumenNominalValidos(b.PorEstado, -1) || b.LlamamientosEnCurso < 0 {
				return vacio, ports.ErrResumenBolsasNoDisponible
			}
			vistas[b.BolsaRef] = true
			out.Bolsas = append(out.Bolsas, ports.BolsaResumenNominal{BolsaRef: b.BolsaRef, CategoriaRef: b.CategoriaRef,
				ConfirmadaEn: b.ConfirmadaEn, VigenteHasta: b.VigenteHasta, Politica: politica,
				PorEstado: b.PorEstado, LlamamientosEnCurso: b.LlamamientosEnCurso})
		}
		return out, nil
	}
	if accion != ports.AccionRRHHEstadisticasConsultar || campos["estadisticas"] == nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	var s estadisticasAgregadasSQL
	if decodificarJSONResumenExclusivo(campos["estadisticas"], &s) != nil || s.Bolsas.Total < 0 ||
		s.Bolsas.Total != s.Bolsas.Vigentes+s.Bolsas.Sustituidas || len(s.PorBolsa) != s.Bolsas.Total ||
		!conteosResumenNominalValidos(s.Personas.PorEstado, s.Personas.Total) || s.Llamamientos.EnCurso < 0 ||
		s.Llamamientos.HistoricoDisponible || s.Llamamientos.Total != nil ||
		s.Llamamientos.PorCanal != nil || s.Llamamientos.PorResultado != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	resumen := &ports.EstadisticasBolsasNominal{BolsasTotal: s.Bolsas.Total, BolsasVigentes: s.Bolsas.Vigentes,
		BolsasSustituidas: s.Bolsas.Sustituidas, PersonasTotal: s.Personas.Total,
		PersonasPorEstado: s.Personas.PorEstado, LlamamientosEnCurso: s.Llamamientos.EnCurso}
	vistas := make(map[string]bool, len(s.PorBolsa))
	var totalPersonas int
	for _, b := range s.PorBolsa {
		if b.BolsaRef == "" || b.CategoriaRef == "" || b.TipoLista == "" || vistas[b.BolsaRef] ||
			!conteosResumenNominalValidos(b.PorEstado, b.Total) {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		vistas[b.BolsaRef] = true
		totalPersonas += b.Total
		resumen.PorBolsa = append(resumen.PorBolsa, ports.EstadisticasBolsaNominal{
			BolsaRef: b.BolsaRef, CategoriaRef: b.CategoriaRef, TipoLista: b.TipoLista,
			Vigente: b.Vigente, Total: b.Total, PorEstado: b.PorEstado})
	}
	if totalPersonas != s.Personas.Total {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	out.Estadisticas = resumen
	return out, nil
}

func conteosResumenNominalValidos(estados map[string]int, total int) bool {
	if estados == nil {
		return false
	}
	var suma int
	for estado, n := range estados {
		switch estado {
		case "disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision":
		default:
			return false
		}
		if n < 0 || n > maximoFilasResumenBolsas-suma {
			return false
		}
		suma += n
	}
	return total < 0 || suma == total
}
