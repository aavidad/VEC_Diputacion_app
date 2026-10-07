package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/ajustesreglas"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	operacionAjustesReglasCTSQL      = `SELECT vec_contratacion_temporal.operar_ajustes_reglas_v1($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	huellaMaterialAjustesCTSQL       = `SELECT encode(sha256(convert_to($1::jsonb::text,'UTF8')),'hex')`
	audienciaAjustesReglasCT         = "vec_contratacion_temporal.ajustes_reglas.v1"
	recursoAjustesReglasCT           = "vec.contratacion_temporal.reglas"
	organizacionAjustesReglasCT      = "organizacion:desarrollo:dipgra"
	maximoMaterialOperacionAjustesCT = 65536
	// La consulta puede devolver 50 versiones; cada una procede de un
	// material de hasta 64 KiB en CT-148.
	maximoRespuestaOperacionAjustesCT = 4 << 20
)

var (
	ErrOperacionAjustesReglasNoDisponible = errors.New("contratacion temporal: ajustes de reglas no disponibles")
	ErrOperacionAjustesReglasDenegada     = errors.New("contratacion temporal: ajustes de reglas denegados")
	ErrOperacionAjustesReglasInvalida     = errors.New("contratacion temporal: ajuste de reglas invalido")
	ErrOperacionAjustesReglasConflicto    = errors.New("contratacion temporal: version o clave de ajustes en conflicto")
)

// ProveedorAutorizacionAjustesReglasCT obtiene una decisión V3 nueva y ligada
// al actor de la frontera confiable y al material completo normalizado por PG.
// El perfil fijo se provisiona al arrancar; este puerto nunca lo publica.
type ProveedorAutorizacionAjustesReglasCT interface {
	AutorizarAjustesReglasCT(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ComprobarCapacidadAjustesReglasCT(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (bool, error)
}

type ejecutorOperacionAjustesCT interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RepositorioAjustesReglasCT ejecuta CT-148 con el rol ejecutor CT. La
// autorización se consume dentro de la misma transacción serializable que
// guarda versión, recibo, auditoría y outbox, también en un replay.
type RepositorioAjustesReglasCT struct {
	pool            ejecutorOperacionAjustesCT
	proveedor       ProveedorAutorizacionAjustesReglasCT
	organizacionRef string
}

func NuevoRepositorioAjustesReglasCT(pool *pgxpool.Pool, proveedor ProveedorAutorizacionAjustesReglasCT, organizacionRef string) (*RepositorioAjustesReglasCT, error) {
	return nuevoRepositorioAjustesReglasCT(pool, proveedor, organizacionRef)
}

func nuevoRepositorioAjustesReglasCT(pool ejecutorOperacionAjustesCT, proveedor ProveedorAutorizacionAjustesReglasCT, organizacionRef string) (*RepositorioAjustesReglasCT, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) || organizacionRef != organizacionAjustesReglasCT {
		return nil, ErrOperacionAjustesReglasNoDisponible
	}
	return &RepositorioAjustesReglasCT{pool: pool, proveedor: proveedor, organizacionRef: organizacionRef}, nil
}

var _ app.Repositorio = (*RepositorioAjustesReglasCT)(nil)

func (r *RepositorioAjustesReglasCT) Consultar(ctx context.Context, actor vecdomain.ContextoActor, limite int, antes *int64) (app.Lectura, error) {
	if r == nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	if limite < 1 || limite > 50 || (antes != nil && (*antes < 2 || *antes > 10000000)) {
		return app.Lectura{}, app.ErrEntradaInvalida
	}
	material := map[string]any{"operacion": "consultar", "organizacion_ref": r.organizacionRef, "catalogo_id": catalogoAjustesCT, "limite": limite}
	if antes != nil {
		material["antes_de_version"] = *antes
	}
	b, err := json.Marshal(material)
	if err != nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	respuesta, err := r.operar(ctx, actor, "consultar", b)
	if err != nil {
		return app.Lectura{}, errorAjustesCTAplicacion(err)
	}
	var sql struct {
		Vigente *struct {
			Version          int                          `json:"version"`
			HuellaSHA256     string                       `json:"huella_sha256"`
			Ajustes          map[string]map[string]string `json:"ajustes"`
			VigenteDesde     time.Time                    `json:"vigente_desde"`
			BaseVersion      int                          `json:"base_version"`
			BaseHuellaSHA256 string                       `json:"base_huella_sha256"`
		} `json:"vigente"`
		Historial []app.CambioHistorico `json:"historial"`
		HayMas    bool                  `json:"hay_mas"`
	}
	if json.Unmarshal(respuesta, &sql) != nil || sql.Historial == nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	lectura := app.Lectura{Historial: sql.Historial, HayMas: sql.HayMas}
	if sql.Vigente != nil {
		if sql.Vigente.Version < 1 || sql.Vigente.Version > maximoVersionAjustesCT || sql.Vigente.VigenteDesde.IsZero() ||
			sql.Vigente.BaseVersion < 1 || sql.Vigente.BaseVersion > maximoVersionAjustesCT || !huellaSHA256CTValida(sql.Vigente.BaseHuellaSHA256) {
			return app.Lectura{}, app.ErrNoDisponible
		}
		if err := validarHuellaAjustesCT(sql.Vigente.Ajustes, sql.Vigente.HuellaSHA256); err != nil {
			return app.Lectura{}, err
		}
		lectura.Vigente = &reglas.VersionAjustes{CatalogoID: catalogoAjustesCT, Version: sql.Vigente.Version, HuellaSHA256: sql.Vigente.HuellaSHA256,
			Ajustes: sql.Vigente.Ajustes, VigenteDesde: sql.Vigente.VigenteDesde}
		lectura.VigenteBaseVersion = sql.Vigente.BaseVersion
		lectura.VigenteBaseHuella = sql.Vigente.BaseHuellaSHA256
	}
	for _, h := range lectura.Historial {
		if h.Version < 1 || h.Version > maximoVersionAjustesCT || h.VigenteDesde.IsZero() || h.ReciboRef == "" || len(h.Cambios) == 0 {
			return app.Lectura{}, app.ErrNoDisponible
		}
	}
	// Indicador de interfaz sin emisión ni consumo. Operar requerirá una
	// decisión nueva sobre el material exacto.
	indicador := vecdomain.RecursoAutorizable{Referencia: recursoAjustesReglasCT, ModuloID: "contratacion_temporal", Tipo: "catalogo_reglas", Ambitos: map[string]string{"organizacion_ref": r.organizacionRef}, Atributos: map[string]string{"operacion": "ajustar"}}
	lectura.PuedeAjustar, _ = r.proveedor.ComprobarCapacidadAjustesReglasCT(ctx, actor, "contratacion_temporal.reglas.ajustar", indicador)
	return lectura, nil
}

func (r *RepositorioAjustesReglasCT) Operar(ctx context.Context, actor vecdomain.ContextoActor, material app.Material) (app.Resultado, error) {
	if r == nil || material.Operacion != "ajustar" || material.CatalogoID != catalogoAjustesCT ||
		(material.OrganizacionRef != "" && material.OrganizacionRef != r.organizacionRef) {
		return app.Resultado{}, app.ErrEntradaInvalida
	}
	material.OrganizacionRef = r.organizacionRef
	b, err := json.Marshal(material)
	if err != nil || len(b) > maximoMaterialOperacionAjustesCT {
		return app.Resultado{}, app.ErrEntradaInvalida
	}
	respuesta, err := r.operar(ctx, actor, "ajustar", b)
	if err != nil {
		return app.Resultado{}, errorAjustesCTAplicacion(err)
	}
	var resultado app.Resultado
	if json.Unmarshal(respuesta, &resultado) != nil || resultado.Recibo.ReciboRef == "" || resultado.Recibo.ClaveIdempotencia != material.ClaveIdempotencia ||
		resultado.Recibo.Version < 1 || resultado.Recibo.Version > maximoVersionAjustesCT || resultado.Recibo.VigenteDesde.IsZero() ||
		resultado.Recibo.DecisionRef == "" || resultado.Recibo.AuditoriaRef == "" || len(resultado.Recibo.ConsumoHuellaSHA256) != 64 ||
		(!resultado.Replay && (resultado.Recibo.Version != material.VersionEsperada+1 || resultado.Recibo.HuellaSHA256 != material.AjustesHuellaSHA256)) {
		return app.Resultado{}, app.ErrNoDisponible
	}
	if err := validarHuellaAjustesCT(resultado.Ajustes, resultado.Recibo.HuellaSHA256); err != nil {
		return app.Resultado{}, err
	}
	return resultado, nil
}

func validarHuellaAjustesCT(ajustes map[string]map[string]string, esperada string) error {
	if !huellaSHA256CTValida(esperada) || ajustes == nil {
		return app.ErrNoDisponible
	}
	canonico, err := reglas.CanonicoAjustes(ajustes)
	if err != nil {
		return errorAjustesCTAplicacion(err)
	}
	if len(canonico) > maximoCanonicoAjustesCT {
		return app.ErrNoDisponible
	}
	suma := sha256.Sum256(canonico)
	if hex.EncodeToString(suma[:]) != esperada {
		return app.ErrNoDisponible
	}
	return nil
}

func huellaSHA256CTValida(valor string) bool {
	if len(valor) != 64 || strings.ToLower(valor) != valor {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}

func errorAjustesCTAplicacion(err error) error {
	switch {
	case errors.Is(err, ErrOperacionAjustesReglasDenegada):
		return vecdomain.ErrAutorizacionDenegada
	case errors.Is(err, ErrOperacionAjustesReglasInvalida):
		return app.ErrEntradaInvalida
	case errors.Is(err, ErrOperacionAjustesReglasConflicto):
		return app.ErrConflicto
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	default:
		return app.ErrNoDisponible
	}
}

// operar recibe exclusivamente material creado por aplicación. El llamador
// liga antes las constantes de recurso y organización de la configuración
// confiable, nunca valores procedentes del cuerpo HTTP.
func (r *RepositorioAjustesReglasCT) operar(ctx context.Context, actor vecdomain.ContextoActor, operacion string, material json.RawMessage) (json.RawMessage, error) {
	if r == nil || dependenciaNula(r.pool) || dependenciaNula(r.proveedor) || ctx == nil || actor.Validar() != nil || len(material) == 0 || len(material) > maximoMaterialOperacionAjustesCT || !json.Valid(material) {
		return nil, ErrOperacionAjustesReglasNoDisponible
	}
	if operacion != "consultar" && operacion != "ajustar" {
		return nil, ErrOperacionAjustesReglasInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var cabecera struct {
		Operacion       string `json:"operacion"`
		OrganizacionRef string `json:"organizacion_ref"`
		CatalogoID      string `json:"catalogo_id"`
	}
	if json.Unmarshal(material, &cabecera) != nil || cabecera.Operacion != operacion || cabecera.OrganizacionRef != r.organizacionRef || cabecera.CatalogoID != catalogoAjustesCT {
		return nil, ErrOperacionAjustesReglasInvalida
	}
	var huellaMaterial string
	if err := r.pool.QueryRow(ctx, huellaMaterialAjustesCTSQL, string(material)).Scan(&huellaMaterial); err != nil {
		return nil, normalizarErrorOperacionAjustesCT(ctx, err)
	}
	if len(huellaMaterial) != 64 {
		return nil, ErrOperacionAjustesReglasNoDisponible
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: recursoAjustesReglasCT, ModuloID: "contratacion_temporal", Tipo: "catalogo_reglas", Ambitos: map[string]string{"organizacion_ref": r.organizacionRef}, Atributos: map[string]string{"material_sha256": huellaMaterial}}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, ErrOperacionAjustesReglasNoDisponible
	}
	accion := "contratacion_temporal.reglas." + operacion
	if operacion == "consultar" {
		accion = "contratacion_temporal.reglas.consultar_ajustes"
	}
	autorizacion, err := r.proveedor.AutorizarAjustesReglasCT(ctx, actor, accion, recurso)
	if err != nil {
		return nil, normalizarErrorOperacionAjustesCT(ctx, err)
	}
	resumen := autorizacion.ResumenCapacidad()
	if autorizacion.ValidarEstructura() != nil || autorizacion.PersonaVersion() != actor.Instantanea.PersonaVersion || autorizacion.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaAjustesReglasCT || resumen.EfectoRef() != recursoAjustesReglasCT || resumen.EfectoHuellaSHA256() != huellaRecurso {
		return nil, ErrOperacionAjustesReglasDenegada
	}
	args := []any{string(material), autorizacion.CapacidadCanonica(), autorizacion.DecisionCanonica(), autorizacion.MotivoCanonico(), autorizacion.ContextoActorCanonico(),
		strconv.FormatUint(autorizacion.PersonaVersion(), 10), strconv.FormatUint(autorizacion.PerfilVersion(), 10), autorizacion.PayloadVECAD3(), autorizacion.SobreCOSESign1(), autorizacion.EvidenciaVerificacion(), autorizacion.RaizPublicaSPKI()}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, normalizarErrorOperacionAjustesCT(ctx, err)
	}
	defer tx.Rollback(ctx)
	var respuesta json.RawMessage
	if err = tx.QueryRow(ctx, operacionAjustesReglasCTSQL, args...).Scan(&respuesta); err != nil {
		return nil, normalizarErrorOperacionAjustesCT(ctx, err)
	}
	if len(respuesta) == 0 || len(respuesta) > maximoRespuestaOperacionAjustesCT || !json.Valid(respuesta) {
		return nil, ErrOperacionAjustesReglasNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, normalizarErrorOperacionAjustesCT(ctx, err)
	}
	return respuesta, nil
}

func normalizarErrorOperacionAjustesCT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	// El servicio V3 envuelve la caída del registro junto con la denegación.
	// La causa de infraestructura prevalece para no presentarla como un 403.
	if errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) {
		return ErrOperacionAjustesReglasNoDisponible
	}
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return ErrOperacionAjustesReglasDenegada
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ErrOperacionAjustesReglasDenegada
		case "23505", "40001", "55P03":
			return ErrOperacionAjustesReglasConflicto
		case "22023", "22001":
			return ErrOperacionAjustesReglasInvalida
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return ErrOperacionAjustesReglasNoDisponible
}
