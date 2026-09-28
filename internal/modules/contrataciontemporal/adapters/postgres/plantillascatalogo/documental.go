package plantillascatalogo

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaPublicadaDocumental = `SELECT vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1($1::jsonb,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
	audienciaDocumental         = "vec_contratacion_temporal.catalogo_plantillas_documental.v1"
)

var huellaDocumental = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ProveedorAutorizacionDocumental emite una decisión nueva para listar o
// descargar. La decisión de leer el expediente ya se consumió y no se reusa.
type ProveedorAutorizacionDocumental interface {
	AutorizarCatalogoDocumental(context.Context, vecdomain.ContextoActor, string, vecdomain.RecursoAutorizable) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// ProveedorDocumental sirve solo la última publicación a un expediente ya
// leído con V3 y vuelve a autorizar la lectura del catálogo dentro de CT133.
type ProveedorDocumental struct {
	pool            *pgxpool.Pool
	resolver        ResolverActor
	proveedor       ProveedorAutorizacionDocumental
	inicial         vecdomain.CatalogoConfigurable
	organizacionRef string
}

func NuevoProveedorDocumental(pool *pgxpool.Pool, resolver ResolverActor, proveedor ProveedorAutorizacionDocumental, inicial *vecdomain.CatalogoConfigurable, organizacionRef string) (*ProveedorDocumental, error) {
	if pool == nil || resolver == nil || proveedor == nil || inicial == nil || organizacionRef != organizacionCatalogoCT {
		return nil, app.ErrNoDisponible
	}
	c, err := inicial.ClonarCanonico()
	if err != nil || c.ID != app.CatalogoID || c.ModuloID != app.ModuloID || c.Estado != vecdomain.EstadoCatalogoPublicado {
		return nil, app.ErrNoDisponible
	}
	return &ProveedorDocumental{pool: pool, resolver: resolver, proveedor: proveedor, inicial: c, organizacionRef: organizacionRef}, nil
}

func (p *ProveedorDocumental) ObtenerPlantillasDocumento(ctx context.Context, s app.SolicitudDocumental, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	if p == nil || p.pool == nil || p.resolver == nil || p.proveedor == nil || ctx == nil || instante.IsZero() || s.Validar() != nil || s.OrganizacionRef != p.organizacionRef {
		return nil, app.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	actor, err := p.resolver.ResolverContextoActor(ctx)
	if err != nil || actor.Validar() != nil {
		return nil, app.ErrNoDisponible
	}
	material, err := json.Marshal(s)
	if err != nil || len(material) > 8<<10 {
		return nil, app.ErrEntradaInvalida
	}
	var huellaMaterial string
	if err = p.pool.QueryRow(ctx, `SELECT encode(sha256(convert_to($1::jsonb::text,'UTF8')),'hex')`, string(material)).Scan(&huellaMaterial); err != nil {
		return nil, normalizar(ctx, err)
	}
	if !huellaDocumental.MatchString(huellaMaterial) {
		return nil, app.ErrNoDisponible
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: s.ExpedienteRef, ModuloID: app.ModuloID, Tipo: "catalogo_plantillas_documental_ct", Ambitos: map[string]string{"organizacion_ref": p.organizacionRef}, Atributos: map[string]string{"material_sha256": huellaMaterial}}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return nil, app.ErrNoDisponible
	}
	accion := "contratacion_temporal.plantillas_documentos.documental_" + s.Operacion
	capacidad, err := p.proveedor.AutorizarCatalogoDocumental(ctx, actor, accion, recurso)
	if err != nil {
		return nil, normalizar(ctx, err)
	}
	r := capacidad.ResumenCapacidad()
	if capacidad.ValidarEstructura() != nil || capacidad.PersonaVersion() != actor.Instantanea.PersonaVersion || capacidad.PerfilVersion() != actor.Instantanea.PerfilVersion || r.Operacion() != accion || r.AudienciaConsumo() != audienciaDocumental || r.EfectoRef() != s.ExpedienteRef || r.EfectoHuellaSHA256() != h {
		return nil, ErrDenegado
	}
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, normalizar(ctx, err)
	}
	defer tx.Rollback(ctx)
	args := []any{string(material), capacidad.CapacidadCanonica(), capacidad.DecisionCanonica(), capacidad.MotivoCanonico(), capacidad.ContextoActorCanonico(), strconv.FormatUint(capacidad.PersonaVersion(), 10), strconv.FormatUint(capacidad.PerfilVersion(), 10), capacidad.PayloadVECAD3(), capacidad.SobreCOSESign1(), capacidad.EvidenciaVerificacion(), capacidad.RaizPublicaSPKI()}
	var b []byte
	if err = tx.QueryRow(ctx, consultaPublicadaDocumental, args...).Scan(&b); err != nil {
		return nil, normalizar(ctx, err)
	}
	if len(b) == 0 || len(b) > maximoRespuesta {
		return nil, app.ErrNoDisponible
	}
	var v struct {
		Catalogo             vecdomain.CatalogoConfigurable `json:"catalogo"`
		CatalogoHuellaSHA256 string                         `json:"catalogo_huella_sha256"`
		ContenidoJSONSHA256  string                         `json:"contenido_json_sha256"`
		Version              int                            `json:"version"`
		Revision             int                            `json:"revision"`
		ProcedenciaRef       string                         `json:"procedencia_ref"`
	}
	if json.Unmarshal(b, &v) != nil || v.Catalogo.ID != app.CatalogoID || v.Catalogo.ModuloID != app.ModuloID || v.Catalogo.Estado != vecdomain.EstadoCatalogoPublicado || v.Version != v.Catalogo.Version || v.Revision != v.Catalogo.Revision || !huellaDocumental.MatchString(v.CatalogoHuellaSHA256) || !huellaDocumental.MatchString(v.ContenidoJSONSHA256) || v.ProcedenciaRef == "" {
		return nil, app.ErrNoDisponible
	}
	huella, err := v.Catalogo.HuellaSHA256()
	if err != nil || huella != v.CatalogoHuellaSHA256 {
		return nil, app.ErrNoDisponible
	}
	if v.Version == p.inicial.Version {
		esperada, _ := p.inicial.HuellaSHA256()
		if huella != esperada {
			return nil, app.ErrNoDisponible
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, normalizar(ctx, err)
	}
	plantillas, err := informejuridico.NuevasPlantillasBorrador(v.Catalogo, instante)
	if err != nil {
		return nil, app.ErrNoDisponible
	}
	return plantillas, nil
}
