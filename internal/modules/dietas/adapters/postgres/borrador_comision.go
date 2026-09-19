package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const (
	AudienciaBorradorPropio       = "vec_dietas_v1.borrador_propio.v1"
	AccionCrearBorradorPropio     = "dietas.borrador.crear_propio"
	AccionRecuperarBorradorPropio = "dietas.borrador.recuperar_propio"
	AccionListarBorradoresPropios = "dietas.borrador.listar_propios"
	FinalidadBorradorPropio       = "gestionar_borrador_propio"
)

var referencia = regexp.MustCompile(`^[a-z][A-Za-z0-9_:-]{2,180}$`)
var clave = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)

// ProveedorAutorizacionBorrador recibe exclusivamente contexto del servidor y
// el recurso construido por este adaptador. Nunca recibe material V3 de HTTP.
// AutorizacionBorrador conserva el enlace sellado por separado del material
// exportado para contrastarlo antes de consumirlo en la transacción de negocio.
type AutorizacionBorrador struct {
	Material      vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ResultadoBase EnlaceAutorizacionBorrador
}
type EnlaceAutorizacionBorrador struct {
	DecisionRef, ContextoRef, ActorRef, PerfilRef, Accion, RecursoRef string
	DecisionHuellaSHA256, ContextoHuellaSHA256, RecursoHuellaSHA256   string
	CorrelacionRef                                                    string
}
type ProveedorAutorizacionBorrador interface {
	AutorizarBorradorPropio(context.Context, core.ContextoActor, string, core.RecursoAutorizable) (AutorizacionBorrador, error)
}
type iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type RepositorioBorradorComision struct {
	pool      iniciador
	proveedor ProveedorAutorizacionBorrador
}

var _ ports.UnidadTrabajoBorradorComision = (*RepositorioBorradorComision)(nil)

func NuevoRepositorioBorradorComision(pool *pgxpool.Pool, proveedor ProveedorAutorizacionBorrador) (*RepositorioBorradorComision, error) {
	if pool == nil || nulo(proveedor) {
		return nil, ports.ErrBorradorNoDisponible
	}
	return &RepositorioBorradorComision{pool: pool, proveedor: proveedor}, nil
}

type actorMaterial struct {
	ActorRef    string `json:"actor_ref"`
	PerfilRef   string `json:"perfil_ref"`
	PersonaRef  string `json:"persona_ref"`
	EmpleadoRef string `json:"empleado_ref"`
}
type materialCrear struct {
	Actor           actorMaterial           `json:"actor"`
	ClaveOperacion  string                  `json:"clave_operacion"`
	VersionEsperada uint64                  `json:"version_esperada"`
	Borrador        domain.BorradorComision `json:"borrador"`
}
type materialRecuperar struct {
	Actor       actorMaterial `json:"actor"`
	ComisionRef string        `json:"comision_ref"`
}
type materialListar struct {
	Actor   actorMaterial `json:"actor"`
	Limite  int           `json:"limite"`
	Despues string        `json:"despues"`
}

func actorDesde(a core.ContextoActor) (actorMaterial, error) {
	empleado, e := application.EmpleadoBorradorPropio(a)
	if e != nil {
		return actorMaterial{}, e
	}
	return actorMaterial{a.Principal.ID, a.PerfilActivoRef, a.PersonaRef, empleado}, nil
}
func materialCreacion(x ports.SolicitudCrearBorradorPropio) ([]byte, core.RecursoAutorizable, error) {
	a, e := actorDesde(x.ContextoActor)
	if e != nil {
		return nil, core.RecursoAutorizable{}, e
	}
	if !clave.MatchString(x.ClaveOperacion) || x.VersionEsperada != 0 || x.Borrador.Validar() != nil || x.Borrador.PersonaRef != a.PersonaRef {
		return nil, core.RecursoAutorizable{}, domain.ErrBorradorComisionInvalido
	}
	b := x.Borrador
	return codificarMaterial("dietas:borrador:"+x.ClaveOperacion, a, materialCrear{a, x.ClaveOperacion, 0, b})
}
func materialListado(a core.ContextoActor, q ports.ConsultaBorradoresPropios) ([]byte, core.RecursoAutorizable, error) {
	actor, e := actorDesde(a)
	if e != nil {
		return nil, core.RecursoAutorizable{}, e
	}
	if q.Limite < 1 || q.Limite > 20 || (q.Despues != "" && !referencia.MatchString(q.Despues)) {
		return nil, core.RecursoAutorizable{}, domain.ErrBorradorComisionInvalido
	}
	return codificarMaterial("dietas:borradores:propios", actor, materialListar{actor, q.Limite, q.Despues})
}
func materialLectura(a core.ContextoActor, ref string) ([]byte, core.RecursoAutorizable, error) {
	actor, e := actorDesde(a)
	if e != nil {
		return nil, core.RecursoAutorizable{}, e
	}
	if !referencia.MatchString(ref) {
		return nil, core.RecursoAutorizable{}, domain.ErrBorradorComisionInvalido
	}
	return codificarMaterial(ref, actor, materialRecuperar{actor, ref})
}
func codificarMaterial(ref string, a actorMaterial, m any) ([]byte, core.RecursoAutorizable, error) {
	b, e := json.Marshal(m)
	if e != nil || len(b) > ports.MaxBytesMaterialBorrador {
		return nil, core.RecursoAutorizable{}, domain.ErrBorradorComisionInvalido
	}
	h := sha256.Sum256(b)
	recurso := core.RecursoAutorizable{Referencia: ref, ModuloID: "dietas", Tipo: "borrador_comision",
		Ambitos:   map[string]string{"persona_ref": a.PersonaRef, "empleado_ref": a.EmpleadoRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil {
		return nil, core.RecursoAutorizable{}, domain.ErrBorradorComisionInvalido
	}
	return b, recurso, nil
}
func (r *RepositorioBorradorComision) CrearBorradorPropio(ctx context.Context, x ports.SolicitudCrearBorradorPropio) (ports.ReciboBorradorComision, error) {
	b, recurso, e := materialCreacion(x)
	if e != nil {
		return ports.ReciboBorradorComision{}, e
	}
	defer clear(b)
	var recibo ports.ReciboBorradorComision
	e = r.ejecutar(ctx, x.ContextoActor, AccionCrearBorradorPropio, recurso, b, func(salida []byte) error {
		if decodificar(salida, &recibo) != nil {
			return ports.ErrBorradorNoDisponible
		}
		recibo.RegistradoEn = recibo.RegistradoEn.UTC()
		return validarRecibo(recibo, recurso.Referencia)
	})
	if e != nil {
		return ports.ReciboBorradorComision{}, e
	}
	return recibo, nil
}
func (r *RepositorioBorradorComision) RecuperarBorradorPropio(ctx context.Context, a core.ContextoActor, ref string) (domain.BorradorComision, ports.ReciboBorradorComision, error) {
	b, recurso, e := materialLectura(a, ref)
	if e != nil {
		return domain.BorradorComision{}, ports.ReciboBorradorComision{}, e
	}
	defer clear(b)
	var resultado struct {
		Borrador domain.BorradorComision      `json:"borrador"`
		Recibo   ports.ReciboBorradorComision `json:"recibo"`
	}
	e = r.ejecutar(ctx, a, AccionRecuperarBorradorPropio, recurso, b, func(salida []byte) error {
		if decodificar(salida, &resultado) != nil {
			return ports.ErrBorradorNoDisponible
		}
		resultado.Recibo.RegistradoEn = resultado.Recibo.RegistradoEn.UTC()
		resultado.Borrador.Inicio = resultado.Borrador.Inicio.UTC()
		resultado.Borrador.Fin = resultado.Borrador.Fin.UTC()
		if resultado.Borrador.Validar() != nil || resultado.Borrador.PersonaRef != a.PersonaRef {
			return ports.ErrBorradorNoDisponible
		}
		return validarRecibo(resultado.Recibo, ref)
	})
	if e != nil {
		return domain.BorradorComision{}, ports.ReciboBorradorComision{}, e
	}
	return resultado.Borrador, resultado.Recibo, nil
}
func validarRecibo(r ports.ReciboBorradorComision, ref string) error {
	if r.ComisionRef != ref || !referencia.MatchString(r.ReciboRef) || !referencia.MatchString(r.CorrelacionRef) || r.Version != 1 || r.RegistradoEn.IsZero() || r.RegistradoEn.Nanosecond()%1000 != 0 {
		return ports.ErrBorradorNoDisponible
	}
	return nil
}

// ListarBorradoresPropios consume una concesión nueva por página. La titularidad
// limita las filas devueltas, pero nunca concede permiso de lectura.
func (r *RepositorioBorradorComision) ListarBorradoresPropios(ctx context.Context, a core.ContextoActor, q ports.ConsultaBorradoresPropios) (ports.PaginaBorradoresPropios, error) {
	b, recurso, e := materialListado(a, q)
	if e != nil {
		return ports.PaginaBorradoresPropios{}, e
	}
	defer clear(b)
	var pagina ports.PaginaBorradoresPropios
	e = r.ejecutar(ctx, a, AccionListarBorradoresPropios, recurso, b, func(salida []byte) error {
		if decodificar(salida, &pagina) != nil {
			return ports.ErrBorradorNoDisponible
		}
		return validarPagina(pagina, a.PersonaRef, q)
	})
	if e != nil {
		return ports.PaginaBorradoresPropios{}, e
	}
	return pagina, nil
}
func validarPagina(p ports.PaginaBorradoresPropios, persona string, q ports.ConsultaBorradoresPropios) error {
	if p.Borradores == nil || len(p.Borradores) > q.Limite {
		return ports.ErrBorradorNoDisponible
	}
	anterior := q.Despues
	for _, item := range p.Borradores {
		ref := item.Recibo.ComisionRef
		if !referencia.MatchString(ref) || ref <= anterior || item.Borrador.ValidarResumen() != nil || item.Borrador.PersonaRef != persona || validarRecibo(item.Recibo, ref) != nil {
			return ports.ErrBorradorNoDisponible
		}
		anterior = ref
	}
	if p.Siguiente != "" && (len(p.Borradores) != q.Limite || p.Siguiente != anterior) {
		return ports.ErrBorradorNoDisponible
	}
	return nil
}
func (r *RepositorioBorradorComision) ejecutar(ctx context.Context, a core.ContextoActor, accion string, recurso core.RecursoAutorizable, b []byte, validar func([]byte) error) error {
	if r == nil || nulo(r.pool) || nulo(r.proveedor) || ctx == nil {
		return ports.ErrBorradorNoDisponible
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	var consulta string
	switch accion {
	case AccionCrearBorradorPropio:
		consulta = "SELECT vec_dietas_v1.crear_borrador_propio_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text"
	case AccionRecuperarBorradorPropio:
		consulta = "SELECT vec_dietas_v1.recuperar_borrador_propio_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text"
	case AccionListarBorradoresPropios:
		consulta = "SELECT vec_dietas_v1.listar_borradores_propios_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text"
	default:
		return ports.ErrAccesoBorradorDenegado
	}
	autorizacion, e := r.proveedor.AutorizarBorradorPropio(ctx, a, accion, recurso)
	if e != nil {
		return e
	}
	material := autorizacion.Material
	evento := autorizacion.ResultadoBase
	h, e := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if e != nil || material.ValidarEstructura() != nil || resumen.Operacion() != accion || resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != AudienciaBorradorPropio ||
		resumen.DecisionRef() != evento.DecisionRef || resumen.DecisionHuellaSHA256() != evento.DecisionHuellaSHA256 ||
		resumen.ContextoRef() != evento.ContextoRef || resumen.ContextoHuellaSHA256() != evento.ContextoHuellaSHA256 ||
		evento.ActorRef != a.Principal.ID || evento.PerfilRef != a.PerfilActivoRef || evento.Accion != accion ||
		evento.RecursoRef != recurso.Referencia || evento.RecursoHuellaSHA256 != h {
		return ports.ErrBorradorNoDisponible
	}
	_, _, err := r.transaccion(ctx, consulta, accion, b, material, validar)
	return err
}

// transaccion conserva consumo, estado, recibo, historia y auditoría juntos.
// Un COMMIT sin respuesta exige recuperación con la misma clave idempotente;
// no acredita un rechazo ni permite crear otra operación.
func (r *RepositorioBorradorComision) transaccion(ctx context.Context, consulta, accion string, b []byte, material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, validar func([]byte) error) (resultado, causa string, err error) {
	resultado, causa = "fallo_confirmado", "persistencia"
	tx, e := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if e != nil {
		return resultado, causa, errorSQL(ctx, e)
	}
	confirmacionIntentada := false
	defer func() {
		if err == nil {
			return
		}
		rollbackCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rollbackErr := tx.Rollback(rollbackCtx)
		if !confirmacionIntentada && rollbackErr != nil {
			resultado, causa = "resultado_indeterminado", "rollback"
			err = ports.ErrResultadoBorradorIncierto
		}
	}()
	secretos := [][]byte{material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(), material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI()}
	defer func() {
		for _, s := range secretos {
			clear(s)
		}
	}()
	var salida []byte
	e = tx.QueryRow(ctx, consulta, string(b), secretos[0], secretos[1], secretos[2], secretos[3], int64(material.PersonaVersion()), int64(material.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if e != nil {
		err = errorSQL(ctx, e)
		if errors.Is(err, ports.ErrConflictoBorrador) {
			causa = "conflicto"
		}
		return resultado, causa, err
	}
	defer clear(salida)
	limiteSalida := ports.MaxBytesReciboBorrador
	switch accion {
	case AccionRecuperarBorradorPropio:
		limiteSalida = ports.MaxBytesDetalleBorrador
	case AccionListarBorradoresPropios:
		limiteSalida = ports.MaxBytesListadoBorrador
	}
	if len(salida) == 0 || len(salida) > limiteSalida {
		return resultado, "recibo_invalido", ports.ErrBorradorNoDisponible
	}
	if e = validar(salida); e != nil {
		return resultado, "recibo_invalido", e
	}
	confirmacionIntentada = true
	if e = tx.Commit(ctx); e != nil {
		var pg *pgconn.PgError
		if errors.Is(e, pgx.ErrTxCommitRollback) || (errors.As(e, &pg) && pg.Code == "40001") {
			return "fallo_confirmado", "commit_revertido", ports.ErrBorradorNoDisponible
		}
		return "resultado_indeterminado", "commit", ports.ErrResultadoBorradorIncierto
	}
	return "", "", nil
}
func decodificar(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return ports.ErrBorradorNoDisponible
	}
	return nil
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
func errorSQL(ctx context.Context, e error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) {
		switch pg.Code {
		case "PDI00":
			return domain.ErrBorradorComisionInvalido
		case "PDI01":
			return ports.ErrConflictoBorrador
		case "PDI03", "42501":
			return ports.ErrAccesoBorradorDenegado
		case "PDI04":
			return ports.ErrBorradorNoEncontrado
		}
	}
	return ports.ErrBorradorNoDisponible
}
