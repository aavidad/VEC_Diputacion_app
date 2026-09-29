package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type consultorPreparacionVigenteHTTPPrueba struct {
	llamadas int
	peticion application.SolicitudConsultarPreparacionCoberturaVigente
	err      error
}

func (c *consultorPreparacionVigenteHTTPPrueba) ConsultarParaAdaptador(
	_ context.Context,
	peticion application.SolicitudConsultarPreparacionCoberturaVigente,
) (application.ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador, error) {
	c.llamadas++
	c.peticion = peticion
	return application.ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{}, c.err
}

func TestGETPreparacionCoberturaVigenteDeniegaSinCatalogoParcial(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"perfil", application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil, 403, "datos_no_disponibles_perfil"},
		{"permiso", application.ErrPresentacionPropuestaCoberturaDenegada, 403, "acceso_denegado"},
		{"fuente", application.ErrPresentacionPropuestaCoberturaNoDisponible, 503, "servicio_no_disponible"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			consultor := &consultorPreparacionVigenteHTTPPrueba{
				err: errors.Join(caso.err, errors.New("catalogo:privado:documento_sae")),
			}
			h, err := NuevoManejadorPreparacionCoberturaVigente(
				autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, consultor,
			)
			if err != nil {
				t.Fatal(err)
			}
			respuesta := httptest.NewRecorder()
			h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil))
			if respuesta.Code != caso.estado || consultor.llamadas != 1 ||
				consultor.peticion.OrganizacionRef != contextoCoberturaValidoPrueba().OrganizacionRef {
				t.Fatalf("GET no pasó por contexto fiable: %d/%d", respuesta.Code, consultor.llamadas)
			}
			var salida map[string]any
			if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil || len(salida) != 1 {
				t.Fatalf("envelope no cerrado: %s %v", respuesta.Body.String(), err)
			}
			problema := salida["error"].(map[string]any)
			if problema["codigo"] != caso.codigo || problema["clave_i18n"] != "api.contratacion_temporal.cobertura.error."+caso.codigo {
				t.Fatalf("error sin clave traducible: %v", problema)
			}
			for _, privado := range []string{"catalogo:privado", "documento_sae", "\"catalogo\"", "\"vias\""} {
				if strings.Contains(respuesta.Body.String(), privado) {
					t.Fatalf("filtró %q: %s", privado, respuesta.Body.String())
				}
			}
			if respuesta.Header().Get("Cache-Control") != "no-store, no-transform" || respuesta.Header().Get("Set-Cookie") != "" {
				t.Fatal("cabeceras de consulta no seguras")
			}
		})
	}
}

func TestGETPreparacionCoberturaVigenteRechazaEntradaNoVacia(t *testing.T) {
	consultor := &consultorPreparacionVigenteHTTPPrueba{}
	h, err := NuevoManejadorPreparacionCoberturaVigente(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, consultor,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		r      *http.Request
		estado int
	}{
		{"query", httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente+"?via=sae", nil), 404},
		{"cuerpo", httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, strings.NewReader("{}")), 400},
		{"metodo", httptest.NewRequest(http.MethodPost, RutaPreparacionCoberturaVigente, nil), 405},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			h.ServeHTTP(respuesta, caso.r)
			if respuesta.Code != caso.estado || consultor.llamadas != 0 {
				t.Fatalf("entrada no vacía alcanzó el caso de uso: %d/%d", respuesta.Code, consultor.llamadas)
			}
		})
	}
	conCookie := httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil)
	conCookie.Header.Set("Cookie", "forjada=1")
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, conCookie)
	if respuesta.Code != 400 || consultor.llamadas != 0 {
		t.Fatal("aceptó cookie en la lectura previa")
	}
}

func TestGETPreparacionCoberturaVigenteSinContextoOResultadoNoExponeDatos(t *testing.T) {
	consultor := &consultorPreparacionVigenteHTTPPrueba{}
	sinContexto, err := NuevoManejadorPreparacionCoberturaVigente(
		autoridadCoberturaPrueba{err: ErrContextoCanalAusente}, consultor,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	sinContexto.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil))
	if respuesta.Code != http.StatusUnauthorized || consultor.llamadas != 0 ||
		codigoErrorCoberturaPrueba(t, respuesta) != "autenticacion_requerida" {
		t.Fatal("la lectura alcanzó el caso de uso sin identidad")
	}
	conContexto, err := NuevoManejadorPreparacionCoberturaVigente(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, consultor,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta = httptest.NewRecorder()
	conContexto.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil))
	if respuesta.Code != http.StatusServiceUnavailable || consultor.llamadas != 1 ||
		strings.Contains(respuesta.Body.String(), "\"catalogo\"") {
		t.Fatal("un resultado sin sello se presentó como catálogo")
	}
}

func TestContratoJSONPreparacionCoberturaVigenteSoloListaPublicada(t *testing.T) {
	datos := application.PreparacionCatalogoPropuestaCobertura{
		Identidad: domain.IdentidadCatalogoViasCobertura{
			Referencia: "catalogo:ct:cobertura:v2", Version: 2,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		Canon: domain.CanonHuellaCatalogoCoberturaV2(), EsEjemplo: true,
		Vias: []application.PreparacionViaPropuestaCobertura{{
			Clave: "oferta_sae", Orden: 2,
			Datos: []domain.ElementoPreparacionViaCobertura{{
				Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
			}},
		}},
	}
	proyeccion := proyectarPreparacionCatalogoCobertura(&datos)
	if proyeccion == nil {
		t.Fatal("faltó proyección")
	}
	respuesta := httptest.NewRecorder()
	responderJSONCobertura(respuesta, httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil),
		200, envoltorioPreparacionCoberturaVigente{Data: preparacionCoberturaVigenteJSON{
			Esquema: "vec.contratacion-temporal.preparacion-cobertura.v1", Catalogo: *proyeccion,
		}})
	if respuesta.Code != 200 || respuesta.Body.Len() > MaximoRespuestaCoberturaBytes {
		t.Fatalf("salida inválida o excesiva: %d/%d", respuesta.Code, respuesta.Body.Len())
	}
	var salida map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil || len(salida) != 1 {
		t.Fatalf("envelope inválido: %s %v", respuesta.Body.String(), err)
	}
	data := salida["data"].(map[string]any)
	if len(data) != 2 || data["esquema"] != "vec.contratacion-temporal.preparacion-cobertura.v1" {
		t.Fatalf("esquema inválido: %v", data)
	}
	catalogo := data["catalogo"].(map[string]any)
	exigirClavesCoberturaV2(t, catalogo, "referencia", "version", "huella_sha256", "es_ejemplo", "vias")
	via := catalogo["vias"].([]any)[0].(map[string]any)
	exigirClavesCoberturaV2(t, via, "clave", "orden", "documentos", "datos")
	if via["clave"] != "oferta_sae" || len(via["documentos"].([]any)) != 0 ||
		len(via["datos"].([]any)) != 1 {
		t.Fatalf("SAE no quedó como lista de preparación: %v", via)
	}
	for _, prohibido := range []string{"contacto", "envio", "seleccion", "comprobaciones", "procedencia"} {
		if strings.Contains(respuesta.Body.String(), prohibido) {
			t.Fatalf("campo operativo %q en la vista", prohibido)
		}
	}
}

func TestGETPreparacionCoberturaVigenteAdmiteMaximoV2Bajo256KiB(t *testing.T) {
	catalogo := catalogoPreparacionCoberturaJSON{
		Referencia: "catalogo:ct:cobertura:maximo", Version: 2,
		HuellaSHA256: strings.Repeat("a", 64), EsEjemplo: true,
	}
	for indice := 0; indice < 64; indice++ {
		via := viaPreparacionCoberturaJSON{Clave: fmt.Sprintf("via_%02d", indice), Orden: uint16(indice + 1)}
		for elemento := 0; elemento < 4; elemento++ {
			dato := elementoPreparacionCoberturaJSON{
				Clave: fmt.Sprintf("%s%02d", strings.Repeat("c", 78), elemento),
				Orden: uint16(elemento + 1), ClaveI18n: "ct." + strings.Repeat("i", 77),
			}
			via.Documentos = append(via.Documentos, dato)
			via.Datos = append(via.Datos, dato)
		}
		catalogo.Vias = append(catalogo.Vias, via)
	}
	respuesta := httptest.NewRecorder()
	responderJSONCobertura(respuesta, httptest.NewRequest(http.MethodGet, RutaPreparacionCoberturaVigente, nil),
		200, envoltorioPreparacionCoberturaVigente{Data: preparacionCoberturaVigenteJSON{
			Esquema: "vec.contratacion-temporal.preparacion-cobertura.v1", Catalogo: catalogo,
		}})
	if respuesta.Code != 200 || respuesta.Body.Len() > MaximoRespuestaCoberturaBytes {
		t.Fatalf("máximo V2 excede 256 KiB: estado=%d bytes=%d", respuesta.Code, respuesta.Body.Len())
	}
}
