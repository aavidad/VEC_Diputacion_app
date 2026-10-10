package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type emisorListaRPTPrueba struct {
	cursor string
	err    error
}

func (e *emisorListaRPTPrueba) materialListaCategoriasRPT(_ context.Context, cursor string) (core.SolicitudAutorizacionLigadaV3, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.cursor = cursor
	return core.SolicitudAutorizacionLigadaV3{}, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, e.err
}

type lectorListaRPTPrueba struct {
	vp.LectorCategoriasRPT
	orden     vp.OrdenCategoriasHabilitadasRPT
	resultado vp.ResultadoCategoriasHabilitadasRPT
	err       error
}

func (l *lectorListaRPTPrueba) ListarCategoriasHabilitadasRPT(_ context.Context, o vp.OrdenCategoriasHabilitadasRPT) (vp.ResultadoCategoriasHabilitadasRPT, error) {
	l.orden = o
	return l.resultado, l.err
}

func resultadoListaRPTPrueba(t *testing.T) vp.ResultadoCategoriasHabilitadasRPT {
	t.Helper()
	doc, err := json.Marshal(core.CatalogoConfigurable{ID: "categorias_rpt", Version: 2, FuenteRef: "ejemplo:rpt-publica", AprobacionRef: "ejemplo:sin-aprobacion-juridica"})
	if err != nil {
		t.Fatal(err)
	}
	ref := vp.ReferenciaPublicacionRPT{CatalogoID: "categorias_rpt", Version: 2, HuellaSHA256: strings.Repeat("c", 64)}
	siguiente := "categoria:rpt:administrativo"
	return vp.ResultadoCategoriasHabilitadasRPT{Encontrado: true, HayMas: true, SiguienteCursor: &siguiente,
		Publicaciones: []vp.PublicacionRPT{{Referencia: ref, DocumentoCanonico: string(doc)}},
		Categorias: []vp.CategoriaHabilitadaRPT{{CategoriaID: siguiente, Publicacion: ref, Revision: 1, Estado: "habilitada", Etiqueta: "Administrativo",
			Definicion: core.EntradaCatalogoConfigurable{Clave: siguiente, Etiqueta: "Administrativo", Atributos: map[string]string{"etiqueta_en": "Administrative officer"}}}}}
}

// La fachada pide una autorización por página, consulta el catálogo de la
// configuración con límite 100 y toma fuente y aprobación del documento
// publicado del que procede cada categoría.
func TestCategoriasRPTB2FachadaUsaLectorComunYDocumentoPublicado(t *testing.T) {
	emisor := &emisorListaRPTPrueba{}
	lector := &lectorListaRPTPrueba{resultado: resultadoListaRPTPrueba(t)}
	f := &fachadaCategoriasRPTB2{catalogoID: "categorias_rpt", autoridad: emisor, lector: lector}
	p, err := f.ListarCategoriasRPT(context.Background(), "categoria:rpt:a")
	if err != nil || emisor.cursor != "categoria:rpt:a" || lector.orden.Consulta != (vp.ConsultaCategoriasHabilitadasRPT{CatalogoID: "categorias_rpt", CursorCategoriaID: "categoria:rpt:a", Limite: httpct.LimiteCategoriasRPTB2}) {
		t.Fatalf("consulta distinta: %+v %v", lector.orden.Consulta, err)
	}
	if len(p.Categorias) != 1 || !p.HayMas || p.SiguienteCursor != "categoria:rpt:administrativo" {
		t.Fatalf("página: %+v", p)
	}
	c := p.Categorias[0]
	if c.CatalogoVersion != 2 || c.CatalogoHuellaSHA256 != strings.Repeat("c", 64) || c.FuenteRef != "ejemplo:rpt-publica" ||
		c.AprobacionRef != "ejemplo:sin-aprobacion-juridica" || c.Atributos["etiqueta_en"] != "Administrative officer" {
		t.Fatalf("categoría: %+v", c)
	}
}

// Negativa, catálogo sin publicar y publicación ausente se traducen sin
// inventar datos: 403, 409 y 503.
func TestCategoriasRPTB2FachadaTraduceErrores(t *testing.T) {
	f := &fachadaCategoriasRPTB2{catalogoID: "categorias_rpt", autoridad: &emisorListaRPTPrueba{err: ct.ErrAutorizacionDenegada}, lector: &lectorListaRPTPrueba{}}
	if _, err := f.ListarCategoriasRPT(context.Background(), ""); !errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) {
		t.Fatalf("negativa: %v", err)
	}
	f = &fachadaCategoriasRPTB2{catalogoID: "categorias_rpt", autoridad: &emisorListaRPTPrueba{}, lector: &lectorListaRPTPrueba{err: vp.ErrLecturaRPTDenegada}}
	if _, err := f.ListarCategoriasRPT(context.Background(), ""); !errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) {
		t.Fatalf("lectura denegada: %v", err)
	}
	f = &fachadaCategoriasRPTB2{catalogoID: "categorias_rpt", autoridad: &emisorListaRPTPrueba{}, lector: &lectorListaRPTPrueba{}}
	if _, err := f.ListarCategoriasRPT(context.Background(), ""); !errors.Is(err, httpct.ErrPreparacionPendienteIncorporacionPersonalB2) {
		t.Fatalf("catálogo sin publicar: %v", err)
	}
	sinDocumento := resultadoListaRPTPrueba(t)
	sinDocumento.Publicaciones = nil
	f = &fachadaCategoriasRPTB2{catalogoID: "categorias_rpt", autoridad: &emisorListaRPTPrueba{}, lector: &lectorListaRPTPrueba{resultado: sinDocumento}}
	if _, err := f.ListarCategoriasRPT(context.Background(), ""); !errors.Is(err, httpct.ErrManejadorIncorporacionPersonalB2) {
		t.Fatalf("publicación ausente: %v", err)
	}
}
