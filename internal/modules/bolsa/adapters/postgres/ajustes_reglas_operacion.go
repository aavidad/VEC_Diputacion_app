package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	app "vec-diputacion-granada/internal/modules/bolsa/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	operarAjustesBolsaSQL         = `SELECT vec_bolsa_llamamientos.operar_ajustes_reglas_v1($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	huellaMaterialAjustesBolsaSQL = `SELECT encode(sha256(convert_to($1::jsonb::text,'UTF8')),'hex')`
	audienciaAjustesBolsa         = "vec_bolsa_llamamientos.ajustes_reglas.v1"
	organizacionAjustesBolsa      = "organizacion:desarrollo:dipgra"
	recursoAjustesBolsa           = "vec.bolsa.reglas"
)

// La raíz aporta el proveedor ACTO nominal. Este puerto no publica perfiles
// ni material de autorización y nunca recibe identidad desde HTTP.
type ProveedorAutorizacionAjustesReglasBolsa interface {
	AutorizarAjustesReglasBolsa(context.Context, vecdomain.ContextoActor, string, string, vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ComprobarCapacidadAjustesReglasBolsa(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (bool, error)
}

// W/V implementa estas lecturas owner-only con sesión certificada y auditoría
// común en la misma transacción. Sin este puerto no se construye repositorio.
type LectorAjustesReglasBolsa interface {
	ConsultarAjustesReglasBolsa(context.Context, vecdomain.ContextoActor, int, *int64) (json.RawMessage, error)
	LeerPreimagenAjustesReglasBolsa(context.Context, vecdomain.ContextoActor, string) (json.RawMessage, bool, error)
}

type ejecutorAjustesBolsaSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioAjustesReglasBolsa struct {
	pool         ejecutorAjustesBolsaSQL
	consulta     *ConsultaAjustesReglasBolsaPostgreSQL
	proveedor    ProveedorAutorizacionAjustesReglasBolsa
	lector       LectorAjustesReglasBolsa
	organizacion string
}

func NuevoRepositorioAjustesReglasBolsa(pool *pgxpool.Pool, proveedor ProveedorAutorizacionAjustesReglasBolsa, lector LectorAjustesReglasBolsa, organizacion string) (*RepositorioAjustesReglasBolsa, error) {
	if pool == nil || nulaAjustesBolsa(proveedor) || nulaAjustesBolsa(lector) || organizacion != organizacionAjustesBolsa {
		return nil, app.ErrNoDisponible
	}
	return &RepositorioAjustesReglasBolsa{pool: pool, consulta: &ConsultaAjustesReglasBolsaPostgreSQL{db: pool}, proveedor: proveedor, lector: lector, organizacion: organizacion}, nil
}

func nulaAjustesBolsa(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

var _ app.Repositorio = (*RepositorioAjustesReglasBolsa)(nil)

func (r *RepositorioAjustesReglasBolsa) LeerCabeza(ctx context.Context) (*reglas.VersionAjustes, error) {
	if r == nil || r.consulta == nil {
		return nil, app.ErrNoDisponible
	}
	return r.consulta.LeerCabeza(ctx)
}

func (r *RepositorioAjustesReglasBolsa) Consultar(ctx context.Context, actor vecdomain.ContextoActor, limite int, antes *int64) (app.Lectura, error) {
	if r == nil || r.lector == nil || r.proveedor == nil || ctx == nil || actor.Validar() != nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	if limite < 1 || limite > 50 || antes != nil && (*antes < 2 || *antes > 10_000_000) {
		return app.Lectura{}, app.ErrEntradaInvalida
	}
	b, err := r.lector.ConsultarAjustesReglasBolsa(ctx, actor, limite, antes)
	if err != nil {
		return app.Lectura{}, mapearErrorAjustesBolsa(err)
	}
	if len(b) == 0 || len(b) > 4<<20 || !json.Valid(b) {
		return app.Lectura{}, app.ErrNoDisponible
	}
	var sql struct {
		ConsultadaEn time.Time               `json:"consultada_en"`
		Cabeza       *versionLecturaBolsa    `json:"cabeza"`
		VigenteHoy   *versionLecturaBolsa    `json:"vigente_hoy"`
		Programados  []app.VersionProgramada `json:"programados"`
		Historial    []app.CambioHistorico   `json:"historial"`
		HayMas       bool                    `json:"hay_mas"`
	}
	if json.Unmarshal(b, &sql) != nil || sql.ConsultadaEn.IsZero() || sql.Programados == nil || sql.Historial == nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	l := app.Lectura{ConsultadaEn: sql.ConsultadaEn, Programados: sql.Programados, Historial: sql.Historial, HayMas: sql.HayMas}
	if sql.Cabeza != nil {
		v, e := sql.Cabeza.version()
		if e != nil {
			return app.Lectura{}, e
		}
		l.Cabeza = &v
		l.CabezaPublicadaEn = sql.Cabeza.PublicadaEn
	}
	if sql.VigenteHoy != nil {
		v, e := sql.VigenteHoy.version()
		if e != nil {
			return app.Lectura{}, e
		}
		l.VigenteHoy = &v
		l.VigentePublicadaEn = sql.VigenteHoy.PublicadaEn
	}
	if l.Cabeza == nil && (l.VigenteHoy != nil || len(l.Programados) > 0) {
		return app.Lectura{}, app.ErrNoDisponible
	}
	indicador := vecdomain.RecursoAutorizable{Referencia: recursoAjustesBolsa, ModuloID: "bolsa", Tipo: "catalogo_reglas", Ambitos: map[string]string{"organizacion_ref": r.organizacion}, Atributos: map[string]string{"operacion": "ajustar"}}
	l.PuedeAjustar, _ = r.proveedor.ComprobarCapacidadAjustesReglasBolsa(ctx, actor, "bolsa.reglas.ajustar", indicador)
	return l, nil
}

type versionLecturaBolsa struct {
	Version      int                          `json:"version"`
	HuellaSHA256 string                       `json:"huella_sha256"`
	Ajustes      map[string]map[string]string `json:"ajustes"`
	VigenteDesde time.Time                    `json:"vigente_desde"`
	PublicadaEn  time.Time                    `json:"publicada_en"`
}

func (v versionLecturaBolsa) version() (reglas.VersionAjustes, error) {
	if v.PublicadaEn.IsZero() || v.VigenteDesde.Before(v.PublicadaEn) {
		return reglas.VersionAjustes{}, app.ErrNoDisponible
	}
	canon, e := reglas.CanonicoAjustes(v.Ajustes)
	if e != nil {
		return reglas.VersionAjustes{}, app.ErrNoDisponible
	}
	return restaurarVersionAjustesBolsa(app.CatalogoAjustes, int64(v.Version), v.HuellaSHA256, string(canon), v.VigenteDesde)
}

func (r *RepositorioAjustesReglasBolsa) LeerPreimagen(ctx context.Context, actor vecdomain.ContextoActor, clave string) (app.Material, bool, error) {
	if r == nil || r.lector == nil || ctx == nil || actor.Validar() != nil {
		return app.Material{}, false, app.ErrNoDisponible
	}
	b, existe, err := r.lector.LeerPreimagenAjustesReglasBolsa(ctx, actor, clave)
	if err != nil {
		return app.Material{}, false, mapearErrorAjustesBolsa(err)
	}
	if !existe {
		return app.Material{}, false, nil
	}
	var m app.Material
	if len(b) == 0 || len(b) > 65536 || json.Unmarshal(b, &m) != nil || m.Operacion != "ajustar" || m.CatalogoID != app.CatalogoAjustes || m.ClaveIdempotencia != clave || m.OrganizacionRef != r.organizacion || m.BaseVersion < 1 || len(m.Cambios) == 0 {
		return app.Material{}, false, app.ErrNoDisponible
	}
	var ajustes map[string]map[string]string
	if json.Unmarshal([]byte(m.AjustesCanonico), &ajustes) != nil || ajustes == nil {
		return app.Material{}, false, app.ErrNoDisponible
	}
	canon, err := reglas.CanonicoAjustes(ajustes)
	huella, errHuella := reglas.HuellaAjustes(ajustes)
	if err != nil || errHuella != nil || string(canon) != m.AjustesCanonico || huella != m.AjustesHuellaSHA256 {
		return app.Material{}, false, app.ErrNoDisponible
	}
	return m, true, nil
}

func (r *RepositorioAjustesReglasBolsa) Operar(ctx context.Context, actor vecdomain.ContextoActor, m app.Material) (app.Resultado, error) {
	if r == nil || r.pool == nil || r.proveedor == nil || ctx == nil || actor.Validar() != nil || m.Operacion != "ajustar" || m.CatalogoID != app.CatalogoAjustes || m.OrganizacionRef != "" && m.OrganizacionRef != r.organizacion {
		return app.Resultado{}, app.ErrEntradaInvalida
	}
	m.OrganizacionRef = r.organizacion
	b, err := json.Marshal(m)
	if err != nil || len(b) > 65536 {
		return app.Resultado{}, app.ErrEntradaInvalida
	}
	var huella string
	if err = r.pool.QueryRow(ctx, huellaMaterialAjustesBolsaSQL, string(b)).Scan(&huella); err != nil || len(huella) != 64 {
		return app.Resultado{}, app.ErrNoDisponible
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: recursoAjustesBolsa, ModuloID: "bolsa", Tipo: "catalogo_reglas", Ambitos: map[string]string{"organizacion_ref": r.organizacion}, Atributos: map[string]string{"material_sha256": huella}}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return app.Resultado{}, app.ErrNoDisponible
	}
	aut, err := r.proveedor.AutorizarAjustesReglasBolsa(ctx, actor, "bolsa.reglas.ajustar", m.MotivoClave, recurso)
	if err != nil {
		return app.Resultado{}, mapearErrorAjustesBolsa(err)
	}
	resumen := aut.ResumenCapacidad()
	if aut.ValidarEstructura() != nil || aut.PersonaVersion() != actor.Instantanea.PersonaVersion || aut.PerfilVersion() != actor.Instantanea.PerfilVersion || resumen.Operacion() != "bolsa.reglas.ajustar" || resumen.AudienciaConsumo() != audienciaAjustesBolsa || resumen.EfectoRef() != recursoAjustesBolsa || resumen.EfectoHuellaSHA256() != huellaRecurso {
		return app.Resultado{}, app.ErrProhibido
	}
	actorCanonico, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actorCanonico, aut.ContextoActorCanonico()) {
		return app.Resultado{}, app.ErrProhibido
	}
	var motivo struct {
		Esquema    string                              `json:"esquema"`
		Referencia vecdomain.ReferenciaEntradaCatalogo `json:"referencia"`
	}
	if json.Unmarshal(aut.MotivoCanonico(), &motivo) != nil || motivo.Esquema != vecdomain.EsquemaHuellaMotivoAutorizacionV2 ||
		motivo.Referencia.CatalogoID != app.CatalogoMotivosID || motivo.Referencia.EntradaClave != m.MotivoClave {
		return app.Resultado{}, app.ErrProhibido
	}
	motivoCanonico, err := vecdomain.RepresentacionCanonicaMotivoAutorizacionV2(motivo.Referencia)
	if err != nil || !bytes.Equal(motivoCanonico, aut.MotivoCanonico()) {
		return app.Resultado{}, app.ErrProhibido
	}
	motivoHash := sha256.Sum256(motivoCanonico)
	if resumen.MotivoHuellaSHA256() != hex.EncodeToString(motivoHash[:]) {
		return app.Resultado{}, app.ErrProhibido
	}
	args := []any{string(b), aut.CapacidadCanonica(), aut.DecisionCanonica(), aut.MotivoCanonico(), aut.ContextoActorCanonico(), strconv.FormatUint(aut.PersonaVersion(), 10), strconv.FormatUint(aut.PerfilVersion(), 10), aut.PayloadVECAD3(), aut.SobreCOSESign1(), aut.EvidenciaVerificacion(), aut.RaizPublicaSPKI()}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return app.Resultado{}, mapearErrorAjustesBolsa(err)
	}
	defer tx.Rollback(ctx)
	var raw json.RawMessage
	if err = tx.QueryRow(ctx, operarAjustesBolsaSQL, args...).Scan(&raw); err != nil {
		return app.Resultado{}, mapearErrorAjustesBolsa(err)
	}
	if len(raw) == 0 || len(raw) > 4<<20 || !json.Valid(raw) {
		return app.Resultado{}, app.ErrNoDisponible
	}
	var result app.Resultado
	if json.Unmarshal(raw, &result) != nil || result.Ajustes == nil || result.Recibo.ClaveIdempotencia != m.ClaveIdempotencia ||
		result.Recibo.ReciboRef == "" || result.Recibo.Version < 1 || result.Recibo.Version > 9_999_999 ||
		result.Recibo.VigenteDesde.IsZero() || result.Recibo.PublicadaEn.IsZero() || result.Recibo.DecisionRef == "" ||
		result.Recibo.AuditoriaRef == "" || result.Recibo.ConsumoHuellaSHA256 == "" {
		return app.Resultado{}, app.ErrNoDisponible
	}
	huellaResultado, err := reglas.HuellaAjustes(result.Ajustes)
	if err != nil || huellaResultado != result.Recibo.HuellaSHA256 ||
		!result.Replay && (result.Recibo.Version != m.VersionEsperada+1 || result.Recibo.HuellaSHA256 != m.AjustesHuellaSHA256) {
		return app.Resultado{}, app.ErrNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return app.Resultado{}, mapearErrorAjustesBolsa(err)
	}
	return result, nil
}

func mapearErrorAjustesBolsa(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return app.ErrProhibido
	}
	var p *pgconn.PgError
	if errors.As(err, &p) {
		switch p.Code {
		case "23505", "40001", "55P03":
			return app.ErrConflicto
		case "22023":
			return app.ErrEntradaInvalida
		case "42501":
			return app.ErrProhibido
		}
	}
	return app.ErrNoDisponible
}
