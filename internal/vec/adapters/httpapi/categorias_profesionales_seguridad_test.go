package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	personalmodule "vec-diputacion-granada/internal/modules/personal"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

func TestCategoriasProfesionalesRechazaConsultaMalCodificadaOSemicolon(t *testing.T) {
	handler := newTestHandler(t)
	for nombre, consulta := range map[string]string{
		"porcentaje mal formado": "q=%ZZ",
		"separador semicolon":    "q=tecnico;area=administracion_general",
	} {
		t.Run(nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories", nil)
			peticion.URL.RawQuery = consulta
			respuesta := httptest.NewRecorder()

			servirCatalogoPersonalConPermisosExpresos(
				handler, respuesta, peticion, personalmodule.PermissionPositionRead,
			)

			if respuesta.Code != http.StatusBadRequest ||
				!strings.Contains(respuesta.Body.String(), "filtro_categorias_profesionales_invalido") {
				t.Fatalf("consulta %q = %d: %s", consulta, respuesta.Code, respuesta.Body.String())
			}
		})
	}
}

func TestCategoriasProfesionalesListaYDetalleDerivanEstadoDeFuente(t *testing.T) {
	handler := newTestHandler(t)

	lista := httptest.NewRecorder()
	servirCatalogoPersonalConPermisosExpresos(
		handler,
		lista,
		httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories?limit=1", nil),
		personalmodule.PermissionPositionRead,
	)
	if lista.Code != http.StatusOK {
		t.Fatalf("lista = %d: %s", lista.Code, lista.Body.String())
	}
	var respuestaLista struct {
		Datos struct {
			Categorias paginaCategoriasProfesionalesHTTP `json:"categories"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lista.Body.Bytes(), &respuestaLista); err != nil {
		t.Fatal(err)
	}
	if !respuestaLista.Datos.Categorias.Fuente.Demostracion || len(respuestaLista.Datos.Categorias.Items) != 1 ||
		respuestaLista.Datos.Categorias.Items[0].Estado != "Demostración pendiente de validación RRHH" {
		t.Fatalf("lista no deriva estado de la fuente: %#v; cuerpo=%s", respuestaLista.Datos.Categorias, lista.Body.String())
	}

	detalle := httptest.NewRecorder()
	servirCatalogoPersonalConPermisosExpresos(
		handler,
		detalle,
		httptest.NewRequest(http.MethodGet, "/api/vec/personal/categories/administrativo", nil),
		personalmodule.PermissionPositionRead,
	)
	if detalle.Code != http.StatusOK {
		t.Fatalf("detalle = %d: %s", detalle.Code, detalle.Body.String())
	}
	var respuestaDetalle struct {
		Datos struct {
			Categoria categoriaProfesionalHTTP                               `json:"category"`
			Fuente    personalports.FuenteCategoriasProfesionalesConsultable `json:"fuente"`
		} `json:"data"`
	}
	if err := json.Unmarshal(detalle.Body.Bytes(), &respuestaDetalle); err != nil {
		t.Fatal(err)
	}
	if !respuestaDetalle.Datos.Fuente.Demostracion || respuestaDetalle.Datos.Fuente.Revision == "" ||
		respuestaDetalle.Datos.Categoria.Estado != "Demostración pendiente de validación RRHH" {
		t.Fatalf("detalle sin fuente segura o estado coherente: %#v", respuestaDetalle)
	}
	for _, prohibido := range []string{"source_path", "creado_por", "publicado_por", "aprobacion_ref", "motivo_publicacion"} {
		if strings.Contains(detalle.Body.String(), prohibido) {
			t.Fatalf("detalle expone procedencia interna %q: %s", prohibido, detalle.Body.String())
		}
	}
}

type consultaCategoriasProfesionalesSeguridad struct {
	catalogo personalports.CatalogoCategoriasProfesionalesConsultable
}

func (c consultaCategoriasProfesionalesSeguridad) ObtenerVigentes(
	ctx context.Context,
	_ time.Time,
) (personalports.CatalogoCategoriasProfesionalesConsultable, error) {
	if err := ctx.Err(); err != nil {
		return personalports.CatalogoCategoriasProfesionalesConsultable{}, err
	}
	return c.catalogo.Clonar(), nil
}

type relojCategoriasProfesionalesSeguridad struct {
	instante time.Time
}

func (r relojCategoriasProfesionalesSeguridad) Ahora() time.Time { return r.instante }

func TestCategoriasProfesionalesNoInfiereDemostracionDeDescripcion(t *testing.T) {
	instante := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	catalogo := personalports.CatalogoCategoriasProfesionalesConsultable{
		Referencia: personalports.ReferenciaCatalogoCategoriasProfesionales{
			CatalogoID:           "categorias-profesionales",
			CatalogoVersion:      2,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
		},
		Fuente: personalports.FuenteCategoriasProfesionalesConsultable{
			Revision: "fuente-oficial-v2", ActualizadaEn: instante,
			Demostracion: false, Aviso: "Catálogo validado por RRHH.",
		},
		Categorias: []personalports.CategoriaProfesionalConsultable{{
			Clave: "administrativo", Etiqueta: "Administrativo",
			Descripcion: "Texto histórico pendiente de validacion que no determina el estado.",
			Orden:       1, Area: "administracion_general", AreaEtiqueta: "Administración general",
		}},
	}
	servicio, err := personalapp.NuevoServicioConsultaCategoriasProfesionales(
		consultaCategoriasProfesionalesSeguridad{catalogo: catalogo},
		relojCategoriasProfesionalesSeguridad{instante: instante},
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := newTestHandlerWithOptions(t, HandlerOptions{CategoriasProfesionales: servicio})

	for _, ruta := range []string{
		"/api/vec/personal/categories",
		"/api/vec/personal/categories/administrativo",
	} {
		respuesta := httptest.NewRecorder()
		servirCatalogoPersonalConPermisosExpresos(
			handler,
			respuesta,
			httptest.NewRequest(http.MethodGet, ruta, nil),
			personalmodule.PermissionPositionRead,
		)
		if respuesta.Code != http.StatusOK || !strings.Contains(respuesta.Body.String(), `"state":"Vigente"`) ||
			strings.Contains(respuesta.Body.String(), "Demostración pendiente") {
			t.Fatalf("%s infirio estado desde descripcion: %d: %s", ruta, respuesta.Code, respuesta.Body.String())
		}
	}
}

func TestFuenteCategoriasProfesionalesRechazaControlesUnicodeYCf(t *testing.T) {
	base := personalports.FuenteCategoriasProfesionalesConsultable{
		Revision:      "fuente-oficial-v2",
		ActualizadaEn: time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC),
		Aviso:         "Catálogo validado por RRHH.",
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("precondicion invalida: %v", err)
	}
	for nombre, sufijo := range map[string]string{
		"control nulo":    "\x00",
		"salto de linea":  "\n",
		"formato bidi":    "\u202e",
		"union invisible": "\u200d",
	} {
		t.Run(nombre, func(t *testing.T) {
			fuente := base
			fuente.Aviso += sufijo
			if err := fuente.Validar(); !errors.Is(err, personalports.ErrCatalogoCategoriasProfesionalesNoDisponible) {
				t.Fatalf("aviso con %s aceptado: %v", nombre, err)
			}
		})
	}
}
