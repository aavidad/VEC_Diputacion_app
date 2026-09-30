package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioErroresPeticionCentroPrueba struct{ err error }

func (r *repositorioErroresPeticionCentroPrueba) ListarPeticiones(context.Context, domain.ActorPeticionCentro) ([]domain.DatosPeticionCentro, error) {
	return nil, r.err
}
func (r *repositorioErroresPeticionCentroPrueba) ConsultarOperacion(context.Context, domain.ActorPeticionCentro, string) (*ports.MaterialPeticionCentro, error) {
	return nil, r.err
}
func (r *repositorioErroresPeticionCentroPrueba) ObtenerPeticion(context.Context, domain.ActorPeticionCentro, string) (domain.DatosPeticionCentro, error) {
	return domain.DatosPeticionCentro{}, r.err
}
func (r *repositorioErroresPeticionCentroPrueba) ConfirmarPeticion(context.Context, ports.MaterialPeticionCentro) (ports.ReciboPeticionCentro, error) {
	return ports.ReciboPeticionCentro{}, r.err
}

type catalogoErrorPeticionCentroPrueba struct{ error }

func (c catalogoErrorPeticionCentroPrueba) ObtenerCatalogo(context.Context, string, int) (vecdomain.CatalogoConfigurable, error) {
	return vecdomain.CatalogoConfigurable{}, c.error
}
func (c catalogoErrorPeticionCentroPrueba) ListarVersionesCatalogo(context.Context, string) ([]vecdomain.CatalogoConfigurable, error) {
	return nil, c.error
}

func TestHTTPPeticionCentroDistingueDenegacionDeIndisponibilidad(t *testing.T) {
	id, _ := escenarioIdentidadCentroPrueba(t, "http-errores")
	repo := &repositorioErroresPeticionCentroPrueba{}
	proveedor := &proveedorPeticionCentroDesarrollo{
		actores:   map[string]*identidadPeticionCentroDesarrollo{id.principal.ID: id},
		catalogos: catalogoCentroPrueba{}, reloj: id.soporte.reloj,
	}
	servicio, err := application.NuevoServicioPeticionCentro(proveedor, repo, id.soporte.reloj)
	if err != nil {
		t.Fatal(err)
	}
	m := &manejadorPeticionCentroDesarrollo{proveedor: proveedor, servicio: servicio, bandeja: repo}
	var registro bytes.Buffer
	previo := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(previo) })
	secreto := "per_" + strings.Repeat("d", 32)
	comando := ports.ComandoPeticionCentro{Operacion: ports.OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc",
		Solicitud: &domain.SolicitudCentro{CentroRef: "centro-520", ContactoRef: "contacto:sintetico:001", CategoriaRef: "categoria:tecnica",
			GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal", Detalle: "Necesidad sintética",
			Periodo: domain.PeriodoPrevisto{Inicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Fin: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}}}
	cuerpo, err := json.Marshal(comando)
	if err != nil || comando.Validar() != nil {
		t.Fatalf("comando de prueba inválido: %v", err)
	}
	casos := []struct {
		nombre, ruta, metodo, causa string
		err                         error
		catalogo, cancelarContexto  bool
		estado                      int
	}{
		{"bandeja_denegada", rutaBandejaPeticionCentro, http.MethodGet, "autorizacion_denegada", errors.Join(domain.ErrRatificacionCentroDenegada, errors.New(secreto)), false, false, 403},
		{"bandeja_fuente_envuelta", rutaBandejaPeticionCentro, http.MethodGet, "fuente_autorizacion_no_disponible", errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrFuenteAutorizacionNoDisponible, errors.New(secreto)), false, false, 503},
		{"bandeja_registro_envuelto", rutaBandejaPeticionCentro, http.MethodGet, "registro_decision_no_disponible", errors.Join(vecdomain.ErrAutorizacionDenegada, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, errors.New(secreto)), false, false, 503},
		{"bandeja_timeout", rutaBandejaPeticionCentro, http.MethodGet, "peticion_cancelada_o_vencida", errors.Join(context.DeadlineExceeded, errors.New(secreto)), false, false, 503},
		{"bandeja_contexto_cancelado", rutaBandejaPeticionCentro, http.MethodGet, "peticion_cancelada_o_vencida", domain.ErrRatificacionCentroDenegada, false, true, 503},
		{"contexto_catalogo", rutaContextoPeticionCentro, http.MethodGet, "catalogo_organizacion_no_disponible", errors.Join(personalports.ErrCambioOrganizacionNoDisponible, errors.New(secreto)), true, false, 503},
		{"operacion_denegada", rutaOperacionesPeticionCentro, http.MethodPost, "autorizacion_denegada", errors.Join(domain.ErrRatificacionCentroDenegada, errors.New(secreto)), false, false, 403},
		{"operacion_configuracion", rutaOperacionesPeticionCentro, http.MethodPost, "configuracion_autorizacion_no_disponible", errors.Join(vecdomain.ErrAutorizacionDenegada, vecdomain.ErrConfiguracionAccesoInvalida, errors.New(secreto)), false, false, 503},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			registro.Reset()
			repo.err = c.err
			proveedor.catalogos = catalogoCentroPrueba{}
			if c.catalogo {
				proveedor.catalogos = catalogoErrorPeticionCentroPrueba{c.err}
			}
			var lector *bytes.Reader
			if c.metodo == http.MethodPost {
				lector = bytes.NewReader(cuerpo)
			} else {
				lector = bytes.NewReader(nil)
			}
			ctx := contextoRutaCentroPrueba(id, c.ruta)
			if c.cancelarContexto {
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				cancelar()
			}
			r := httptest.NewRequest(c.metodo, c.ruta, lector).WithContext(ctx)
			if c.metodo == http.MethodPost {
				r.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			m.ServeHTTP(w, r)
			codigo := "operacion_denegada"
			if c.estado == 503 {
				codigo = "servicio_no_disponible"
			}
			if w.Code != c.estado || !strings.Contains(w.Body.String(), `"codigo":"`+codigo+`"`) {
				t.Fatalf("HTTP %d %s; esperado %d %s", w.Code, w.Body.String(), c.estado, codigo)
			}
			linea := registro.String()
			if !strings.Contains(linea, `"causa":"`+c.causa+`"`) || !strings.Contains(linea, c.ruta) ||
				strings.Contains(linea, secreto) || strings.Contains(w.Body.String(), secreto) {
				t.Fatalf("registro o respuesta no minimizados: %q, %q", linea, w.Body.String())
			}
		})
	}
}
