package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"maps"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// materialListaCategoriasRPT autoriza una página del listado con el perfil
// nominal propio del listado. El recurso y la huella del material son los que
// coteja el lector común (AD3-117): catálogo y módulo de la configuración,
// cursor y límite; nada llega del canal salvo el cursor ya validado.
func (a *autoridadIncorporacionPersonalB2) materialListaCategoriasRPT(ctx context.Context, cursor string) (core.SolicitudAutorizacionLigadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vp.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || a.rptPool == nil || a.catalogoRPTID == "" || a.moduloRPTID == "" || ctx == nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, ct.ErrAutorizacionDenegada
	}
	var c *string
	if cursor != "" {
		c = &cursor
	}
	b, e := json.Marshal(map[string]any{"catalogo_id": a.catalogoRPTID, "modulo_id": a.moduloRPTID, "cursor_categoria_id": c, "limite": httpct.LimiteCategoriasRPTB2})
	if e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, e
	}
	defer clear(b)
	var sha string
	// Misma huella que calcula el lector: jsonb::text de PostgreSQL.
	if e = a.rptPool.QueryRow(ctx, "SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex')", string(b)).Scan(&sha); e != nil {
		return core.SolicitudAutorizacionLigadaV3{}, cero, errors.Join(ct.ErrConsultaRRHHNoDisponible, e)
	}
	r := core.RecursoAutorizable{Referencia: a.catalogoRPTID, ModuloID: a.moduloRPTID, Tipo: "catalogo_configurable",
		Ambitos: map[string]string{"catalogo_id": a.catalogoRPTID, "modulo_id": a.moduloRPTID}, Atributos: map[string]string{"material_sha256": sha}}
	return a.emitirRecurso(ctx, accionListarCategoriasRPTB2, r)
}

// emisorListaCategoriasRPTB2 separa la emisión de la autorización para poder
// probar la fachada sin PostgreSQL ni material V3.
type emisorListaCategoriasRPTB2 interface {
	materialListaCategoriasRPT(context.Context, string) (core.SolicitudAutorizacionLigadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// fachadaCategoriasRPTB2 reutiliza el lector común de categorías RPT
// (pgvec.NuevoLectorCategoriasRPTPostgreSQL) que el montaje B2 ya compone, con
// la misma autoridad nominal B2: una autorización nueva por página.
type fachadaCategoriasRPTB2 struct {
	catalogoID string
	autoridad  emisorListaCategoriasRPTB2
	lector     vp.LectorCategoriasRPT
}

func (f *fachadaCategoriasRPTB2) ListarCategoriasRPT(ctx context.Context, cursor string) (httpct.PaginaCategoriasRPTB2, error) {
	var cero httpct.PaginaCategoriasRPTB2
	if f == nil || f.autoridad == nil || f.lector == nil || ctx == nil {
		return cero, httpct.ErrManejadorIncorporacionPersonalB2
	}
	s, x, e := f.autoridad.materialListaCategoriasRPT(ctx, cursor)
	if e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	r, e := f.lector.ListarCategoriasHabilitadasRPT(ctx, vp.OrdenCategoriasHabilitadasRPT{
		Consulta:  vp.ConsultaCategoriasHabilitadasRPT{CatalogoID: f.catalogoID, CursorCategoriaID: cursor, Limite: httpct.LimiteCategoriasRPTB2},
		Solicitud: s, Autorizacion: x})
	if e != nil {
		return cero, errorHTTPNominalB2(ctx, e)
	}
	return paginaCategoriasRPTB2(r)
}

// paginaCategoriasRPTB2 toma fuente y aprobación del documento publicado del
// que procede cada categoría; el lector ya verificó su huella y su contenido.
func paginaCategoriasRPTB2(r vp.ResultadoCategoriasHabilitadasRPT) (httpct.PaginaCategoriasRPTB2, error) {
	var cero httpct.PaginaCategoriasRPTB2
	if !r.Encontrado {
		return cero, errors.Join(httpct.ErrPreparacionPendienteIncorporacionPersonalB2, ct.ErrPreparacionIncorporacionPendiente)
	}
	documentos := make(map[vp.ReferenciaPublicacionRPT]core.CatalogoConfigurable, len(r.Publicaciones))
	for _, p := range r.Publicaciones {
		var c core.CatalogoConfigurable
		if json.Unmarshal([]byte(p.DocumentoCanonico), &c) != nil {
			return cero, httpct.ErrManejadorIncorporacionPersonalB2
		}
		documentos[p.Referencia] = c
	}
	pagina := httpct.PaginaCategoriasRPTB2{Categorias: make([]httpct.CategoriaRPTB2, 0, len(r.Categorias)), HayMas: r.HayMas}
	if r.SiguienteCursor != nil {
		pagina.SiguienteCursor = *r.SiguienteCursor
	}
	for _, c := range r.Categorias {
		d, ok := documentos[c.Publicacion]
		if !ok || c.Publicacion.Version < 1 {
			return cero, httpct.ErrManejadorIncorporacionPersonalB2
		}
		atributos := maps.Clone(c.Definicion.Atributos)
		if atributos == nil {
			atributos = map[string]string{}
		}
		pagina.Categorias = append(pagina.Categorias, httpct.CategoriaRPTB2{CategoriaID: c.CategoriaID, Etiqueta: c.Etiqueta,
			Atributos: atributos, CatalogoVersion: uint64(c.Publicacion.Version), CatalogoHuellaSHA256: c.Publicacion.HuellaSHA256,
			FuenteRef: d.FuenteRef, AprobacionRef: d.AprobacionRef})
	}
	return pagina, nil
}
