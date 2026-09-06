package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type editorOrganizacionPrueba struct {
	denegado bool
	llamadas int
}

func (e *editorOrganizacionPrueba) ActorOrganizacion(context.Context) (string, error) {
	if e.denegado {
		return "", personalports.ErrCambioOrganizacionDenegado
	}
	return "actor:sintetico", nil
}
func (*editorOrganizacionPrueba) AutorizarCambioOrganizacion(context.Context, personalports.MaterialCambioOrganizacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, personalports.ErrCambioOrganizacionDenegado
}
func (e *editorOrganizacionPrueba) GuardarCambio(_ context.Context, s personalports.SolicitudCambioOrganizacion) (personalports.ReciboCambioOrganizacion, error) {
	e.llamadas++
	return personalports.ReciboCambioOrganizacion{ReciboRef: "recibo:sintetico", CatalogoVersion: s.CatalogoVersion, CatalogoRevision: s.CatalogoRevision + 1}, nil
}

func TestEdicionOrganizacionFronteraHTTP(t *testing.T) {
	s := personalports.SolicitudCambioOrganizacion{CatalogoVersion: 1, CatalogoRevision: 1,
		HuellaEsperada: strings.Repeat("a", 64), ClaveIdempotencia: "a77d3f10-a635-46fd-b9eb-a00000000011",
		Unidad: personalports.UnidadCambioOrganizacion{Clave: "local-a77d3f10-a635-46fd-b9eb-a00000000012", Etiqueta: "Unidad de ejemplo", Tipo: "centro"}, Motivo: "Preparación sintética"}
	b, _ := json.Marshal(s)
	for _, c := range []struct {
		nombre, metodo, cuerpo, cabecera string
		denegado                         bool
		estado                           int
	}{
		{"valida", http.MethodPost, string(b), "", false, 200},
		{"identidad_denegada", http.MethodPost, string(b), "", true, 401},
		{"cookie", http.MethodPost, string(b), "Cookie", false, 400},
		{"actor_libre", http.MethodPost, string(b), "X-User", false, 400},
		{"campo_desconocido", http.MethodPost, strings.TrimSuffix(string(b), "}") + `,"actor_id":"otro"}`, "", false, 400},
		{"campo_duplicado", http.MethodPost, strings.TrimSuffix(string(b), "}") + `,"catalogo_version":1}`, "", false, 400},
		{"lectura", http.MethodGet, "", "", false, 405},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			e := &editorOrganizacionPrueba{denegado: c.denegado}
			m := &manejadorEdicionOrganizacionDesarrollo{proveedor: e, repositorio: e, version: 1}
			r := httptest.NewRequest(c.metodo, rutaCambiosOrganizacionContratacionTemporalDesarrollo, strings.NewReader(c.cuerpo))
			r.Header.Set("Content-Type", "application/json")
			if c.cabecera != "" {
				r.Header.Set(c.cabecera, "sintetico")
			}
			w := httptest.NewRecorder()
			m.ServeHTTP(w, r)
			if w.Code != c.estado {
				t.Fatalf("HTTP %d, esperado %d", w.Code, c.estado)
			}
			if c.estado != 200 && e.llamadas != 0 {
				t.Fatal("escritura tras rechazo")
			}
			if w.Header().Get("Set-Cookie") != "" {
				t.Fatal("no admite cookies")
			}
		})
	}
}

func TestEdicionOrganizacionUtilizaRegistroDurableDeAutorizacion(t *testing.T) {
	// Este predicado selecciona publicación de la instantánea y registro
	// PostgreSQL, no la confirmación en memoria de las rutas históricas.
	if !rutaMutacionDurableContratacionTemporalDesarrollo(rutaCambiosOrganizacionContratacionTemporalDesarrollo) {
		t.Fatal("la escritura organizativa debe registrar autorización persistente")
	}
	if rutaMutacionDurableContratacionTemporalDesarrollo(rutaOrganizacionContratacionTemporalDesarrollo) ||
		rutaMutacionDurableContratacionTemporalDesarrollo(rutaCambiosOrganizacionContratacionTemporalDesarrollo+"/otra") {
		t.Fatal("no ampliar la autoridad a lecturas ni rutas derivadas")
	}
}

func TestEdicionOrganizacionNoComponeEscrituraPorDefecto(t *testing.T) {
	rutas, err := nuevasRutasOrganizacionContratacionTemporalDesarrollo(config.Config{}, nil, relojContratacionTemporalDesarrollo{})
	if err != nil || len(rutas) != 2 {
		t.Fatal("composición opcional inesperada", err)
	}
	w := httptest.NewRecorder()
	rutas[1].Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodPost, rutas[1].Ruta, nil))
	if w.Code != 503 {
		t.Fatal("editor habilitado sin dependencias")
	}
	if _, err := nuevasRutasOrganizacionContratacionTemporalDesarrollo(config.Config{PersonalOrganizacionPostgreSQL: true}, nil, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("fallback silencioso de PostgreSQL")
	}
}
