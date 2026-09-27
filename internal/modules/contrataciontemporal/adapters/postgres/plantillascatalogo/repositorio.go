package plantillascatalogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcion   = `SELECT vec_contratacion_temporal.operar_catalogo_plantillas_v1($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	audiencia = "vec_contratacion_temporal.catalogo_plantillas.v1"
	// CT131 admite hasta 16 MiB de catálogo y 17 MB de material JSONB.
	// El límite de lectura cubre esa representación sin aceptar una respuesta
	// mayor que la frontera SQL.
	maximoRespuesta = 17_000_000
)

var ErrDenegado = errors.New("contratacion temporal: gobierno de plantillas denegado")

// ProveedorAutorizacion debe resolver la concesión positiva y el actor desde
// la frontera confiable. El paquete no admite credenciales del cuerpo HTTP.
type ProveedorAutorizacion interface {
	AutorizarCatalogoPlantillas(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	ComprobarCapacidadCatalogoPlantillas(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (bool, error)
}

type Repositorio struct {
	pool      *pgxpool.Pool
	proveedor ProveedorAutorizacion
}

func NuevoRepositorio(pool *pgxpool.Pool, proveedor ProveedorAutorizacion) (*Repositorio, error) {
	if pool == nil || proveedor == nil {
		return nil, app.ErrNoDisponible
	}
	return &Repositorio{pool: pool, proveedor: proveedor}, nil
}

func (r *Repositorio) Consultar(ctx context.Context, actor vecdomain.ContextoActor) (app.Lectura, error) {
	if r == nil || r.pool == nil || r.proveedor == nil || ctx == nil || actor.Validar() != nil {
		return app.Lectura{}, app.ErrNoDisponible
	}
	b := []byte(`{"operacion":"consultar"}`)
	var z app.Lectura
	if err := r.operar(ctx, actor, "consultar", b, &z); err != nil {
		return app.Lectura{}, err
	}
	return z, nil
}

// ComprobarAccion consulta el PDP para el recurso funcional actual. El
// resultado sólo gobierna la presentación del control; Cambiar emite y
// consume una decisión V3 nueva ligada a la petición exacta.
func (r *Repositorio) ComprobarAccion(ctx context.Context, actor vecdomain.ContextoActor, operacion string, lectura app.Lectura) (bool, error) {
	if r == nil || r.proveedor == nil || ctx == nil || actor.Validar() != nil || (operacion != "editar" && operacion != "publicar") {
		return false, app.ErrNoDisponible
	}
	c := lectura.Borrador
	if c == nil {
		c = lectura.Publicado
	}
	if c == nil || c.ID != app.CatalogoID || c.ModuloID != app.ModuloID {
		return false, nil
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: app.CatalogoID, ModuloID: app.ModuloID, Tipo: "catalogo_plantillas_contratacion_temporal", Ambitos: map[string]string{},
		Atributos: map[string]string{"operacion": operacion, "estado": string(c.Estado), "version": strconv.Itoa(c.Version), "revision": strconv.Itoa(c.Revision)}}
	if recurso.Validar() != nil {
		return false, app.ErrNoDisponible
	}
	ok, err := r.proveedor.ComprobarCapacidadCatalogoPlantillas(ctx, actor, "contratacion_temporal.plantillas_documentos."+operacion, recurso)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (r *Repositorio) Cambiar(ctx context.Context, actor vecdomain.ContextoActor, material app.MaterialCambio) (app.ResultadoCambio, error) {
	if r == nil || r.pool == nil || r.proveedor == nil || ctx == nil || actor.Validar() != nil || (material.Operacion != "editar" && material.Operacion != "publicar") {
		return app.ResultadoCambio{}, app.ErrNoDisponible
	}
	b, err := json.Marshal(material)
	if err != nil || len(b) > 17_000_000 {
		return app.ResultadoCambio{}, app.ErrEntradaInvalida
	}
	var z app.ResultadoCambio
	if err = r.operar(ctx, actor, material.Operacion, b, &z); err != nil {
		return app.ResultadoCambio{}, err
	}
	if z.Catalogo.Validar() != nil || z.Catalogo.ID != app.CatalogoID || z.Catalogo.ModuloID != app.ModuloID ||
		z.Recibo.ReciboRef == "" || z.Recibo.RegistradoEn.IsZero() || (z.Recibo.EstadoReplay != "registrado" && z.Recibo.EstadoReplay != "replay") {
		return app.ResultadoCambio{}, app.ErrNoDisponible
	}
	if z.Recibo.EstadoReplay == "registrado" && material.Catalogo != nil {
		h, _ := z.Catalogo.HuellaSHA256()
		if h != material.CatalogoHuellaSHA256 {
			return app.ResultadoCambio{}, app.ErrNoDisponible
		}
	}
	return z, nil
}

func (r *Repositorio) operar(ctx context.Context, actor vecdomain.ContextoActor, operacion string, material []byte, destino any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// PostgreSQL canoniza jsonb antes de fijar la huella del material. Se usa
	// exactamente esa representación para la capacidad y se vuelve a comprobar
	// dentro de la función que consume V3.
	var huellaMaterial string
	if err := r.pool.QueryRow(ctx, `SELECT encode(sha256(convert_to($1::jsonb::text,'UTF8')),'hex')`, string(material)).Scan(&huellaMaterial); err != nil {
		return normalizar(ctx, err)
	}
	if len(huellaMaterial) != 64 {
		return app.ErrNoDisponible
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: app.CatalogoID, ModuloID: app.ModuloID, Tipo: "catalogo_plantillas_contratacion_temporal", Ambitos: map[string]string{}, Atributos: map[string]string{"material_sha256": huellaMaterial}}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return app.ErrNoDisponible
	}
	accion := "contratacion_temporal.plantillas_documentos." + operacion
	autorizacion, err := r.proveedor.AutorizarCatalogoPlantillas(ctx, actor, accion, recurso)
	if err != nil {
		return normalizar(ctx, err)
	}
	resumen := autorizacion.ResumenCapacidad()
	if autorizacion.ValidarEstructura() != nil || autorizacion.PersonaVersion() != actor.Instantanea.PersonaVersion || autorizacion.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia || resumen.EfectoRef() != app.CatalogoID || resumen.EfectoHuellaSHA256() != huellaRecurso {
		return ErrDenegado
	}
	args := []any{string(material), autorizacion.CapacidadCanonica(), autorizacion.DecisionCanonica(), autorizacion.MotivoCanonico(), autorizacion.ContextoActorCanonico(),
		strconv.FormatUint(autorizacion.PersonaVersion(), 10), strconv.FormatUint(autorizacion.PerfilVersion(), 10), autorizacion.PayloadVECAD3(), autorizacion.SobreCOSESign1(), autorizacion.EvidenciaVerificacion(), autorizacion.RaizPublicaSPKI()}
	var respuesta []byte
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return normalizar(ctx, err)
	}
	defer tx.Rollback(ctx)
	if err = tx.QueryRow(ctx, funcion, args...).Scan(&respuesta); err != nil {
		return normalizar(ctx, err)
	}
	if len(respuesta) == 0 || len(respuesta) > maximoRespuesta {
		return app.ErrNoDisponible
	}
	if err = json.Unmarshal(respuesta, destino); err != nil {
		return app.ErrNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return normalizar(ctx, err)
	}
	return nil
}

func normalizar(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return ErrDenegado
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return ErrDenegado
		case "23505", "40001", "55P03", "P0001":
			return app.ErrConflicto
		case "22023", "22001":
			return app.ErrEntradaInvalida
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return fmt.Errorf("%w: persistencia", app.ErrNoDisponible)
}

// ResolverActor se implementa en la composición a partir del contexto
// autenticado del canal, nunca desde una cabecera o el JSON del navegador.
type ResolverActor interface {
	ResolverContextoActor(context.Context) (vecdomain.ContextoActor, error)
}

// ProveedorVivo selecciona la última versión publicada en PostgreSQL en cada
// petición. El catálogo inicial debe haberse provisionado por el migrador;
// una base vacía falla cerrada y nunca publica un fichero desde HTTP.
type ProveedorVivo struct {
	repo     *Repositorio
	resolver ResolverActor
	inicial  *vecdomain.CatalogoConfigurable
}

func NuevoProveedorVivo(repo *Repositorio, resolver ResolverActor, inicial *vecdomain.CatalogoConfigurable) (*ProveedorVivo, error) {
	if repo == nil || resolver == nil || inicial == nil {
		return nil, app.ErrNoDisponible
	}
	c, err := inicial.ClonarCanonico()
	if err != nil || c.ID != app.CatalogoID || c.Estado != vecdomain.EstadoCatalogoPublicado {
		return nil, app.ErrNoDisponible
	}
	return &ProveedorVivo{repo: repo, resolver: resolver, inicial: &c}, nil
}

func (p *ProveedorVivo) ObtenerPlantillas(ctx context.Context, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	if p == nil || p.repo == nil || p.resolver == nil || ctx == nil || instante.IsZero() {
		return nil, app.ErrNoDisponible
	}
	actor, err := p.resolver.ResolverContextoActor(ctx)
	if err != nil {
		return nil, normalizar(ctx, err)
	}
	l, err := p.repo.Consultar(ctx, actor)
	if err != nil {
		return nil, err
	}
	c := l.Publicado
	if c == nil {
		return nil, app.ErrNoDisponible
	}
	if c.Version == p.inicial.Version {
		actual, _ := c.HuellaSHA256()
		esperada, _ := p.inicial.HuellaSHA256()
		if actual != esperada {
			return nil, app.ErrNoDisponible
		}
	}
	return informejuridico.NuevasPlantillasBorrador(*c, instante)
}

var _ app.Repositorio = (*Repositorio)(nil)
