package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type repositorioGuardaEntregaPrueba struct{ llamadas int }

type repositorioNumeroMOADRechazadoPrueba struct {
	*repositorioGuardaEntregaPrueba
	err error
}

func (r *repositorioNumeroMOADRechazadoPrueba) PrepararEntrega(context.Context, ports.ComandoEntregarPeticionCentro) (ports.EntregaPeticionCentro, error) {
	return ports.EntregaPeticionCentro{}, r.err
}

func TestHTTPEntregaNumeroMOADInvalidoYConflictoNoSonTemporales(t *testing.T) {
	for _, caso := range []struct {
		err    error
		estado int
	}{
		{application.ErrSolicitudRegistroInvalida, http.StatusUnprocessableEntity},
		{ports.ErrClaveIdempotenciaUsada, http.StatusConflict},
	} {
		e := nuevaSesionConsultaPrueba(t)
		e.soporte.sesionOperativa = e.p
		canal := e.contexto().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		canal.ruta, canal.metodo = rutaEntregaPeticionCentro, http.MethodPost
		canal.contextoOperacion = &contextoOperacionCTDesarrollo{}
		ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
		p := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: e.soporte}, auditor: &auditorDenegacionEntregaPreV3Prueba{}}
		repo := &repositorioNumeroMOADRechazadoPrueba{&repositorioGuardaEntregaPrueba{}, caso.err}
		servicio, err := application.NuevoServicioEntregaPeticionCentro(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		m := &manejadorEntregaPeticionDesarrollo{p, repo, servicio, nil}
		body := `{"peticion_ref":"peticion:centro:sintetica","version_esperada":2,"numero_expediente_moad":"2026/5487"}`
		r := httptest.NewRequest(http.MethodPost, rutaEntregaPeticionCentro, strings.NewReader(body)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)
		if w.Code != caso.estado {
			t.Fatalf("rechazo definitivo presentado como temporal: estado=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
}

func (r *repositorioGuardaEntregaPrueba) ListarPeticionesRRHH(context.Context) ([]ports.EntregaPeticionCentro, error) {
	r.llamadas++
	return nil, ports.ErrPeticionCentroNoDisponible
}
func (r *repositorioGuardaEntregaPrueba) PrepararEntrega(context.Context, ports.ComandoEntregarPeticionCentro) (ports.EntregaPeticionCentro, error) {
	r.llamadas++
	return ports.EntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
}
func (r *repositorioGuardaEntregaPrueba) ConfirmarEntrega(context.Context, ports.ComandoEntregarPeticionCentro, ports.AltaDePeticionCentro) (ports.EntregaPeticionCentro, error) {
	r.llamadas++
	return ports.EntregaPeticionCentro{}, ports.ErrPeticionCentroNoDisponible
}

func TestHTTPEntregaGuardaDistingueSesionCaidaCancelacionYRevocacion(t *testing.T) {
	const secreto = "referencia-privada-envuelta-en-error"
	casos := []struct {
		nombre, causa string
		err           error
		cancelada     bool
		estado        int
	}{
		{"lector_caido", "sesion_no_disponible", errors.New(secreto), false, 503},
		{"sesion_revocada", "autorizacion_denegada", vecdomain.ErrAutenticacionRevalidadaInvalida, false, 403},
		{"cancelacion", "peticion_cancelada_o_vencida", nil, true, 503},
	}
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		for _, caso := range casos {
			t.Run(metodo+"/"+caso.nombre, func(t *testing.T) {
				e := nuevaSesionConsultaPrueba(t)
				e.soporte.sesionOperativa = e.p
				e.revalidador.err = caso.err
				canal := e.contexto().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
				canal.ruta, canal.metodo = rutaEntregaPeticionCentro, metodo
				canal.contextoOperacion = &contextoOperacionCTDesarrollo{}
				ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
				if caso.cancelada {
					var cancelar context.CancelFunc
					ctx, cancelar = context.WithCancel(ctx)
					cancelar()
				}
				p := &proveedorEntregaPeticionDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: e.soporte}, auditor: &auditorDenegacionEntregaPreV3Prueba{}}
				repo := &repositorioGuardaEntregaPrueba{}
				servicio, err := application.NuevoServicioEntregaPeticionCentro(repo, p)
				if err != nil {
					t.Fatal(err)
				}
				m := &manejadorEntregaPeticionDesarrollo{p, repo, servicio, nil}
				var registro bytes.Buffer
				previo := slog.Default()
				slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
				t.Cleanup(func() { slog.SetDefault(previo) })
				w := httptest.NewRecorder()
				m.ServeHTTP(w, httptest.NewRequest(metodo, rutaEntregaPeticionCentro, nil).WithContext(ctx))
				if w.Code != caso.estado || repo.llamadas != 0 || !strings.Contains(registro.String(), `"causa":"`+caso.causa+`"`) ||
					strings.Contains(registro.String(), secreto) || strings.Contains(w.Body.String(), secreto) {
					t.Fatalf("guarda incorrecta HTTP=%d repositorio=%d registro=%q", w.Code, repo.llamadas, registro.String())
				}
				if !caso.cancelada && e.revalidador.llamadas != 1 {
					t.Fatalf("no se ejercitó la sesión nominal: %d", e.revalidador.llamadas)
				}
			})
		}
	}
}
