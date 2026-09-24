package catalogosvec

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type fuenteImportacionCatalogos struct {
	catalogos map[string]vecdomain.CatalogoConfigurable
}

func (f *fuenteImportacionCatalogos) ObtenerCatalogo(_ context.Context, id string, _ int) (vecdomain.CatalogoConfigurable, error) {
	c, ok := f.catalogos[id]
	if !ok {
		return vecdomain.CatalogoConfigurable{}, errors.New("ausente")
	}
	return c, nil
}
func (f *fuenteImportacionCatalogos) ListarVersionesCatalogo(_ context.Context, id string) ([]vecdomain.CatalogoConfigurable, error) {
	return []vecdomain.CatalogoConfigurable{f.catalogos[id]}, nil
}

func pruebaImportacionCatalogos(t *testing.T) (*fuenteImportacionCatalogos, domain.ReferenciaCatalogoImportacion, domain.ReferenciaCatalogoImportacion) {
	t.Helper()
	fecha := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	unidades := catalogoEstructura("estructura-organizativa-dipgra", 1, fecha, []vecdomain.EntradaCatalogoConfigurable{
		entradaEstructura("centro-uno", "Centro uno", "centro", "", "fuente"),
	})
	clasificaciones := vecdomain.CatalogoConfigurable{
		ID: "categorias-profesionales", Version: 1, Revision: 1, ModuloID: "personal",
		Nombre: "Clasificaciones", FuenteRef: "fuente:clasificaciones", MotivoCreacion: "prueba",
		Estado: vecdomain.EstadoCatalogoBorrador, CreadoPor: "tecnico-personal", CreadoEn: fecha,
		Entradas: []vecdomain.EntradaCatalogoConfigurable{{Clave: "categoria-uno", Etiqueta: "Categoría uno", Orden: 1, VigenteDesde: fecha,
			Atributos: map[string]string{"area": "administracion_general", "area_etiqueta": "Administración general"}}},
	}
	if err := unidades.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := clasificaciones.Validar(); err != nil {
		t.Fatal(err)
	}
	referencia := func(c vecdomain.CatalogoConfigurable) domain.ReferenciaCatalogoImportacion {
		huella, err := c.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		return domain.ReferenciaCatalogoImportacion{ID: c.ID, Version: c.Version, Revision: c.Revision, HuellaSHA256: huella}
	}
	return &fuenteImportacionCatalogos{map[string]vecdomain.CatalogoConfigurable{unidades.ID: unidades, clasificaciones.ID: clasificaciones}}, referencia(unidades), referencia(clasificaciones)
}

func hechosImportacionCatalogos() []domain.HechoImportacionOrganizacion {
	return []domain.HechoImportacionOrganizacion{
		{Clase: "nodo", UnidadRef: "unidad:tecnica:uno", CatalogoEntradaClave: "centro-uno", TipoUnidad: "centro", VigenteDesde: "2026-01-01"},
		{Clase: "puesto_tipo", UnidadRef: "unidad:tecnica:uno", ClasificacionRef: "categoria-uno", VigenteDesde: "2026-01-01"},
	}
}

func TestImportacionCatalogosFijaReferenciaYAdmiteBorradorEnPreparacion(t *testing.T) {
	fuente, u, c := pruebaImportacionCatalogos(t)
	v, err := NuevoVerificadorCatalogosImportacionOrganizacion(fuente)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.ValidarReferencias(context.Background(), u, c); err != nil {
		t.Fatal(err)
	}
	if err := v.ValidarHechos(context.Background(), u, c, hechosImportacionCatalogos()); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*domain.ReferenciaCatalogoImportacion){
		"huella":   func(r *domain.ReferenciaCatalogoImportacion) { r.HuellaSHA256 = strings.Repeat("b", 64) },
		"revision": func(r *domain.ReferenciaCatalogoImportacion) { r.Revision++ },
		"version":  func(r *domain.ReferenciaCatalogoImportacion) { r.Version++ },
		"id":       func(r *domain.ReferenciaCatalogoImportacion) { r.ID = "catalogo-ajeno" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := u
			cambiar(&alterada)
			if !errors.Is(v.ValidarReferencias(context.Background(), alterada, c), ErrCatalogosImportacionOrganizacionNoDisponibles) {
				t.Fatal("referencia manipulada aceptada")
			}
		})
	}
}

func TestImportacionCatalogosRechazaClaveAjenaYVigencia(t *testing.T) {
	fuente, u, c := pruebaImportacionCatalogos(t)
	v, _ := NuevoVerificadorCatalogosImportacionOrganizacion(fuente)
	for nombre, cambiar := range map[string]func([]domain.HechoImportacionOrganizacion){
		"unidad":                func(h []domain.HechoImportacionOrganizacion) { h[0].CatalogoEntradaClave = "otro-centro" },
		"clasificacion":         func(h []domain.HechoImportacionOrganizacion) { h[1].ClasificacionRef = "otra-categoria" },
		"referencia sin enlace": func(h []domain.HechoImportacionOrganizacion) { h[1].UnidadRef = "unidad:ajena" },
		"clave sin nodo":        func(h []domain.HechoImportacionOrganizacion) { h[1].UnidadRef = "centro-uno" },
		"fecha anterior":        func(h []domain.HechoImportacionOrganizacion) { h[1].VigenteDesde = "2019-01-01" },
		"fecha final anterior":  func(h []domain.HechoImportacionOrganizacion) { h[1].VigenteHasta = "2019-01-01" },
	} {
		t.Run(nombre, func(t *testing.T) {
			hechos := hechosImportacionCatalogos()
			cambiar(hechos)
			if !errors.Is(v.ValidarHechos(context.Background(), u, c, hechos), ErrCatalogosImportacionOrganizacionNoDisponibles) {
				t.Fatal("hecho ajeno aceptado")
			}
		})
	}
}

func TestImportacionCatalogosRechazaRetiradoYDependenciaCaida(t *testing.T) {
	fuente, u, c := pruebaImportacionCatalogos(t)
	v, _ := NuevoVerificadorCatalogosImportacionOrganizacion(fuente)
	cat := fuente.catalogos[c.ID]
	cat.Estado = vecdomain.EstadoCatalogoRetirado
	cat.PublicadoPor = "publicador-personal"
	cat.PublicadoEn = cat.CreadoEn.Add(time.Hour)
	cat.AprobacionRef = "aprobacion:una"
	cat.MotivoPublicacion = "aprobado"
	cat.RetiradoPor = "retirador-personal"
	cat.RetiradoEn = cat.PublicadoEn.Add(time.Hour)
	cat.RetiradaAprobacionRef = "aprobacion:retirada"
	cat.MotivoRetirada = "retirado"
	fuente.catalogos[c.ID] = cat
	if !errors.Is(v.ValidarReferencias(context.Background(), u, c), ErrCatalogosImportacionOrganizacionNoDisponibles) {
		t.Fatal("catalogo retirado aceptado")
	}
	delete(fuente.catalogos, c.ID)
	if !errors.Is(v.ValidarReferencias(context.Background(), u, c), ErrCatalogosImportacionOrganizacionNoDisponibles) {
		t.Fatal("fuente caida aceptada")
	}
	var nula *fuenteImportacionCatalogos
	if _, err := NuevoVerificadorCatalogosImportacionOrganizacion(nula); !errors.Is(err, ErrCatalogosImportacionOrganizacionNoDisponibles) {
		t.Fatal("dependencia nula aceptada")
	}
}
