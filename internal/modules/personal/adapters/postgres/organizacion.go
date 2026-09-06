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
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const funcionCatalogoOrganizacion = "vec_contratacion_temporal.obtener_catalogo_organizacion_v1"

// ProveedorCambioOrganizacion es la frontera nominal de identidad y
// autorizacion. El adaptador nunca fabrica una atestacion.
type ProveedorCambioOrganizacion interface {
	ActorOrganizacion(context.Context) (string, error)
	AutorizarCambioOrganizacion(context.Context, personalports.MaterialCambioOrganizacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type organizacionSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type RepositorioOrganizacionPostgreSQL struct {
	pool      organizacionSQL
	proveedor ProveedorCambioOrganizacion
	reloj     vecports.Reloj
}

var _ vecports.ConsultaCatalogosConfigurables = (*RepositorioOrganizacionPostgreSQL)(nil)
var _ personalports.RepositorioCambiosOrganizacion = (*RepositorioOrganizacionPostgreSQL)(nil)

func NuevoRepositorioOrganizacionPostgreSQL(pool *pgxpool.Pool, proveedor ProveedorCambioOrganizacion, reloj vecports.Reloj) (*RepositorioOrganizacionPostgreSQL, error) {
	return nuevoRepositorioOrganizacionPostgreSQL(pool, proveedor, reloj)
}

func nuevoRepositorioOrganizacionPostgreSQL(pool organizacionSQL, proveedor ProveedorCambioOrganizacion, reloj vecports.Reloj) (*RepositorioOrganizacionPostgreSQL, error) {
	if dependenciaNulaOrganizacion(pool) || dependenciaNulaOrganizacion(proveedor) || dependenciaNulaOrganizacion(reloj) {
		return nil, personalports.ErrCambioOrganizacionNoDisponible
	}
	return &RepositorioOrganizacionPostgreSQL{pool: pool, proveedor: proveedor, reloj: reloj}, nil
}

func (r *RepositorioOrganizacionPostgreSQL) ObtenerCatalogo(ctx context.Context, id string, version int) (core.CatalogoConfigurable, error) {
	if r == nil || dependenciaNulaOrganizacion(r.pool) || ctx == nil || id != personalports.IDCatalogoOrganizacion || version < 1 {
		return core.CatalogoConfigurable{}, personalports.ErrCambioOrganizacionNoDisponible
	}
	var contenido []byte
	if err := r.pool.QueryRow(ctx, "SELECT "+funcionCatalogoOrganizacion+"($1::int)", version).Scan(&contenido); err != nil {
		return core.CatalogoConfigurable{}, normalizarErrorOrganizacion(ctx, err)
	}
	c, err := catalogoOrganizacionDesdeJSON(contenido)
	if err != nil || c.ID != id || c.Version != version || c.Estado != core.EstadoCatalogoBorrador {
		return core.CatalogoConfigurable{}, personalports.ErrCambioOrganizacionNoDisponible
	}
	return c, nil
}

func (r *RepositorioOrganizacionPostgreSQL) ListarVersionesCatalogo(ctx context.Context, id string) ([]core.CatalogoConfigurable, error) {
	if r == nil || dependenciaNulaOrganizacion(r.pool) || ctx == nil || id != personalports.IDCatalogoOrganizacion {
		return nil, personalports.ErrCambioOrganizacionNoDisponible
	}
	rows, err := r.pool.Query(ctx, "SELECT catalogo FROM vec_contratacion_temporal.listar_catalogos_organizacion_v1() AS catalogo LIMIT 65")
	if err != nil {
		return nil, normalizarErrorOrganizacion(ctx, err)
	}
	defer rows.Close()
	resultado := make([]core.CatalogoConfigurable, 0, 8)
	for rows.Next() {
		if len(resultado) == vecports.MaximoVersionesConsultaCatalogosAcotada {
			return nil, personalports.ErrCambioOrganizacionNoDisponible
		}
		var contenido []byte
		if err := rows.Scan(&contenido); err != nil {
			return nil, normalizarErrorOrganizacion(ctx, err)
		}
		c, err := catalogoOrganizacionDesdeJSON(contenido)
		if err != nil || c.ID != id || c.Estado != core.EstadoCatalogoBorrador {
			return nil, personalports.ErrCambioOrganizacionNoDisponible
		}
		resultado = append(resultado, c)
	}
	if err := rows.Err(); err != nil {
		return nil, normalizarErrorOrganizacion(ctx, err)
	}
	return resultado, nil
}

func (r *RepositorioOrganizacionPostgreSQL) GuardarCambio(ctx context.Context, s personalports.SolicitudCambioOrganizacion) (personalports.ReciboCambioOrganizacion, error) {
	var vacio personalports.ReciboCambioOrganizacion
	if r == nil || dependenciaNulaOrganizacion(r.pool) || dependenciaNulaOrganizacion(r.proveedor) || dependenciaNulaOrganizacion(r.reloj) || ctx == nil {
		return vacio, personalports.ErrCambioOrganizacionNoDisponible
	}
	if err := s.Validar(); err != nil {
		return vacio, err
	}
	actor, err := r.proveedor.ActorOrganizacion(ctx)
	if err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	material, existe, err := r.materialIdempotente(ctx, s.ClaveIdempotencia)
	if err != nil {
		return vacio, err
	}
	if existe {
		if material.ActorID != actor || !solicitudesIguales(material.Solicitud, s) {
			return vacio, personalports.ErrClaveCambioOrganizacionUsada
		}
	} else {
		actual, err := r.ObtenerCatalogo(ctx, personalports.IDCatalogoOrganizacion, s.CatalogoVersion)
		if err != nil {
			return vacio, err
		}
		nuevo, err := personaldomain.PrepararCambioEstructuraOrganizativa(actual, s.CatalogoRevision, s.HuellaEsperada, actor, s.Motivo, personaldomain.CambioUnidadOrganizativa{Clave: s.Unidad.Clave, Etiqueta: s.Unidad.Etiqueta, Tipo: s.Unidad.Tipo, AdscripcionClave: s.Unidad.AdscripcionClave}, r.reloj.Ahora())
		if err != nil {
			return vacio, err
		}
		canon, err := nuevo.ClonarCanonico()
		if err != nil {
			return vacio, personalports.ErrSolicitudCambioOrganizacionInvalida
		}
		b, err := json.Marshal(canon)
		if err != nil || len(b) > personalports.MaximoBytesCatalogoOrganizacion {
			return vacio, personalports.ErrSolicitudCambioOrganizacionInvalida
		}
		material = personalports.MaterialCambioOrganizacion{Solicitud: s, ActorID: actor, CatalogoCanonico: string(b)}
	}
	if err := material.Validar(); err != nil {
		return vacio, err
	}
	aut, err := r.proveedor.AutorizarCambioOrganizacion(ctx, material)
	if err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	recurso, err := RecursoCambioOrganizacion(material)
	if err != nil {
		return vacio, personalports.ErrCambioOrganizacionDenegado
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := aut.ResumenCapacidad()
	if err != nil || aut.ValidarEstructura() != nil || resumen.Operacion() != personalports.AccionCambioOrganizacion || resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != personalports.AudienciaCambioOrganizacion {
		return vacio, personalports.ErrCambioOrganizacionDenegado
	}
	contenido, err := json.Marshal(material)
	if err != nil || len(contenido) > personalports.MaximoBytesCatalogoOrganizacion*2 {
		return vacio, personalports.ErrSolicitudCambioOrganizacionInvalida
	}
	return r.registrar(ctx, contenido, aut, material)
}

func RecursoCambioOrganizacion(m personalports.MaterialCambioOrganizacion) (core.RecursoAutorizable, error) {
	if err := m.Validar(); err != nil {
		return core.RecursoAutorizable{}, err
	}
	catalogo, err := m.Catalogo()
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return core.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	return core.RecursoAutorizable{Referencia: catalogo.Referencia(), ModuloID: "personal", Tipo: personalports.TipoCambioOrganizacion, Ambitos: map[string]string{"organizacion_ref": "organizacion:desarrollo:dipgra"}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}, nil
}

func (r *RepositorioOrganizacionPostgreSQL) registrar(ctx context.Context, material []byte, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, m personalports.MaterialCambioOrganizacion) (personalports.ReciboCambioOrganizacion, error) {
	var vacio personalports.ReciboCambioOrganizacion
	iniciador, ok := r.pool.(interface {
		BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	})
	if !ok {
		return vacio, personalports.ErrCambioOrganizacionNoDisponible
	}
	tx, err := iniciador.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	defer revertirTransaccionOrganizacion(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	secretos := [][]byte{a.CapacidadCanonica(), a.DecisionCanonica(), a.MotivoCanonico(), a.ContextoActorCanonico(), a.PayloadVECAD3(), a.SobreCOSESign1(), a.EvidenciaVerificacion(), a.RaizPublicaSPKI()}
	defer func() {
		for _, b := range secretos {
			borrarBytesOrganizacion(b)
		}
	}()
	var salida string
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_cambio_organizacion_v1($1::text,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`, string(material), secretos[0], secretos[1], secretos[2], secretos[3], int64(a.PersonaVersion()), int64(a.PerfilVersion()), secretos[4], secretos[5], secretos[6], secretos[7]).Scan(&salida)
	if err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	if len(salida) == 0 || len(salida) > 64*1024 || decodificarJSONEstrictoOrganizacion([]byte(salida), &vacio) != nil {
		return vacio, personalports.ErrResultadoCambioOrganizacionNoConfiable
	}
	vacio.RegistradoEn = normalizarInstanteOrganizacion(vacio.RegistradoEn)
	if err := vacio.ValidarPara(m); err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorOrganizacion(ctx, err)
	}
	return vacio, nil
}

func (r *RepositorioOrganizacionPostgreSQL) materialIdempotente(ctx context.Context, clave string) (personalports.MaterialCambioOrganizacion, bool, error) {
	var b []byte
	err := r.pool.QueryRow(ctx, "SELECT vec_contratacion_temporal.obtener_cambio_organizacion_v1($1::uuid)", clave).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return personalports.MaterialCambioOrganizacion{}, false, nil
	}
	if err != nil {
		return personalports.MaterialCambioOrganizacion{}, false, normalizarErrorOrganizacion(ctx, err)
	}
	if len(b) == 0 {
		return personalports.MaterialCambioOrganizacion{}, false, nil
	}
	var m personalports.MaterialCambioOrganizacion
	if len(b) == 0 || len(b) > personalports.MaximoBytesCatalogoOrganizacion*2 || decodificarJSONEstrictoOrganizacion(b, &m) != nil || m.Validar() != nil {
		return personalports.MaterialCambioOrganizacion{}, false, personalports.ErrCambioOrganizacionNoDisponible
	}
	return m, true, nil
}

func catalogoOrganizacionDesdeJSON(b []byte) (core.CatalogoConfigurable, error) {
	if len(b) == 0 || len(b) > personalports.MaximoBytesCatalogoOrganizacion {
		return core.CatalogoConfigurable{}, personalports.ErrCambioOrganizacionNoDisponible
	}
	var c core.CatalogoConfigurable
	if decodificarJSONEstrictoOrganizacion(b, &c) != nil {
		return core.CatalogoConfigurable{}, personalports.ErrCambioOrganizacionNoDisponible
	}
	canon, err := c.ClonarCanonico()
	if err != nil {
		return core.CatalogoConfigurable{}, personalports.ErrCambioOrganizacionNoDisponible
	}
	return canon, nil
}

func solicitudesIguales(a, b personalports.SolicitudCambioOrganizacion) bool { return a == b }

func normalizarErrorOrganizacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	for _, permitido := range []error{personaldomain.ErrCambioOrganizacionInvalido, personaldomain.ErrRevisionOrganizacionEnConflicto, personalports.ErrCambioOrganizacionDenegado, personalports.ErrClaveCambioOrganizacionUsada, personalports.ErrRevisionCambioOrganizacionEnConflicto, personalports.ErrCambioOrganizacionNoDisponible, personalports.ErrSolicitudCambioOrganizacionInvalida, personalports.ErrResultadoCambioOrganizacionNoConfiable} {
		if errors.Is(err, permitido) {
			return permitido
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501", "P0663":
			return personalports.ErrCambioOrganizacionDenegado
		case "P0660":
			return personaldomain.ErrCambioOrganizacionInvalido
		case "P0661":
			return personalports.ErrClaveCambioOrganizacionUsada
		case "P0662":
			return personaldomain.ErrRevisionOrganizacionEnConflicto
		}
	}
	return personalports.ErrCambioOrganizacionNoDisponible
}

func dependenciaNulaOrganizacion(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

func borrarBytesOrganizacion(v []byte) {
	for i := range v {
		v[i] = 0
	}
}

func decodificarJSONEstrictoOrganizacion(b []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	return func() error {
		var extra any
		if err := d.Decode(&extra); err != io.EOF {
			return errors.New("json: contenido adicional")
		}
		return nil
	}()
}

func normalizarInstanteOrganizacion(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }

func revertirTransaccionOrganizacion(tx pgx.Tx) {
	if tx != nil {
		_ = tx.Rollback(context.Background())
	}
}
