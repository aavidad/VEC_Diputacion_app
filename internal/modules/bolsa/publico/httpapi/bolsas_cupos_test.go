package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type fuenteBloqueadaPrueba struct {
	dentro chan struct{}
	soltar chan struct{}
}

func (f fuenteBloqueadaPrueba) BolsasPublicas(ctx context.Context) ([]BolsaPublica, time.Time, error) {
	f.dentro <- struct{}{}
	<-f.soltar
	return nil, time.Now().UTC(), nil
}

func (f fuenteBloqueadaPrueba) ListaPublica(context.Context, string) (BolsaPublica, []PosicionPublica, time.Time, error) {
	return BolsaPublica{}, nil, time.Time{}, ErrBolsaPublicaNoEncontrada
}

// Con todas las lecturas ocupadas, una consulta más responde 429 con
// Retry-After en lugar de lanzar otra lectura de la fuente.
func TestBolsasPublicasLimitaLecturasSimultaneas(t *testing.T) {
	fuente := fuenteBloqueadaPrueba{dentro: make(chan struct{}, concurrenciaBolsasPublicas), soltar: make(chan struct{})}
	manejador, err := NuevoManejadorBolsasPublicas(fuente)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range concurrenciaBolsasPublicas {
		wg.Go(func() {
			manejador.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, RutaBolsasPublicas, nil))
		})
	}
	for range concurrenciaBolsasPublicas {
		<-fuente.dentro
	}
	rec := httptest.NewRecorder()
	manejador.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaBolsasPublicas, nil))
	close(fuente.soltar)
	wg.Wait()
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("estado=%d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
}
