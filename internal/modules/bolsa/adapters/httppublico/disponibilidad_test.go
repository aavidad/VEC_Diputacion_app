package httppublico

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
)

type servicioDisponibilidadPrueba struct {
	listar   func(context.Context, aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error)
	llamadas atomic.Int32
}

func (s *servicioDisponibilidadPrueba) Listar(
	ctx context.Context,
	solicitud aplicacionbolsa.SolicitudListadoPublico,
) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
	s.llamadas.Add(1)
	return s.listar(ctx, solicitud)
}

func (*servicioDisponibilidadPrueba) Obtener(
	context.Context,
	string,
) (aplicacionbolsa.DetalleConvocatoriaPublica, error) {
	return aplicacionbolsa.DetalleConvocatoriaPublica{}, nil
}

func (*servicioDisponibilidadPrueba) ListarCategorias(
	context.Context,
) (aplicacionbolsa.DirectorioCategoriasPublicas, error) {
	return aplicacionbolsa.DirectorioCategoriasPublicas{}, nil
}

func TestHTTPPublicoCancelaAlAgotarPresupuestoTotal(t *testing.T) {
	servicio := &servicioDisponibilidadPrueba{
		listar: func(ctx context.Context, _ aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			<-ctx.Done()
			return aplicacionbolsa.ListadoConvocatoriasPublicas{}, ctx.Err()
		},
	}
	handler := nuevoHandler(servicio, 1, 10*time.Millisecond)
	rec := httptest.NewRecorder()
	inicio := time.Now()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if rec.Code != http.StatusGatewayTimeout || time.Since(inicio) > time.Second {
		t.Fatalf("deadline total = %d en %s: %s", rec.Code, time.Since(inicio), rec.Body.String())
	}
	if servicio.llamadas.Load() != 1 {
		t.Fatalf("llamadas = %d", servicio.llamadas.Load())
	}
}

func TestHTTPPublicoRechazaAntesDeInvocarAlAgotarCupos(t *testing.T) {
	entrada := make(chan struct{}, 1)
	liberar := make(chan struct{})
	servicio := &servicioDisponibilidadPrueba{
		listar: func(ctx context.Context, _ aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			entrada <- struct{}{}
			select {
			case <-liberar:
				return aplicacionbolsa.ListadoConvocatoriasPublicas{}, nil
			case <-ctx.Done():
				return aplicacionbolsa.ListadoConvocatoriasPublicas{}, ctx.Err()
			}
		},
	}
	handler := nuevoHandler(servicio, 1, time.Second)
	primera := httptest.NewRecorder()
	terminada := make(chan struct{})
	go func() {
		defer close(terminada)
		handler.ServeHTTP(primera, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	}()
	<-entrada

	segunda := httptest.NewRecorder()
	handler.ServeHTTP(segunda, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if segunda.Code != http.StatusTooManyRequests || segunda.Header().Get("Retry-After") != "1" {
		t.Fatalf("segundo acceso = %d, Retry-After=%q", segunda.Code, segunda.Header().Get("Retry-After"))
	}
	if servicio.llamadas.Load() != 1 {
		t.Fatalf("la petición rechazada invocó el servicio: %d", servicio.llamadas.Load())
	}
	close(liberar)
	<-terminada
	if primera.Code != http.StatusOK {
		t.Fatalf("primera petición = %d", primera.Code)
	}
}
