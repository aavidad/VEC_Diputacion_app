package bootstrap

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func fronteraSerializadaPrueba(base *autoridadConsultasContratacionTemporalDesarrollo, principal vecdomain.Principal, siguiente http.Handler) http.Handler {
	return &revalidadorConsultasContratacionTemporalDesarrollo{autoridad: base, siguiente: siguiente,
		resolverDemo: func(context.Context, *http.Request) (vecdomain.Principal, error) { return principal, nil }}
}

func peticionSerializadaPrueba(ctx context.Context) *http.Request {
	r := httptest.NewRequest(http.MethodPost, httpinterno.RutaSeguimientoCese, nil).WithContext(ctx)
	certificado := &x509.Certificate{NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{certificado}}, PeerCertificates: []*x509.Certificate{certificado}}
	return r
}

func TestSerializacionPerfilCTCancelaEsperaYNoLiberaAlTitular(t *testing.T) {
	var s serializacionPerfilesCT
	ctx, liberar, err := s.adquirir(context.Background(), "prf_uno")
	if err != nil {
		t.Fatal(err)
	}
	defer liberar()
	// Una segunda frontera del mismo handler conserva el turno exterior.
	_, liberarInterior, err := s.adquirir(ctx, "prf_uno")
	if err != nil {
		t.Fatal(err)
	}
	liberarInterior()
	cancelado, cancelar := context.WithCancel(context.Background())
	final := make(chan error, 1)
	go func() { _, _, err := s.adquirir(cancelado, "prf_uno"); final <- err }()
	cancelar()
	if err := <-final; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
	limite, fin := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer fin()
	if _, _, err := s.adquirir(limite, "prf_uno"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("plazo: %v", err)
	}
	// Otro perfil puede trabajar aunque el primero siga ocupado.
	_, soltarOtro, err := s.adquirir(context.Background(), "prf_dos")
	if err != nil {
		t.Fatal(err)
	}
	soltarOtro()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.perfiles) != 1 || s.perfiles["prf_uno"].usuarios != 1 {
		t.Fatal("una espera cancelada liberó al titular o dejó entradas")
	}
}

func TestSerializacionPerfilCTLimitaEsperaSinPlazoDelCliente(t *testing.T) {
	var s serializacionPerfilesCT
	_, liberar, err := s.adquirir(context.Background(), "prf_uno")
	if err != nil {
		t.Fatal(err)
	}
	defer liberar()
	if _, _, err := s.adquirir(context.Background(), "prf_uno"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sin límite de espera: %v", err)
	}
}

func TestSerializacionPerfilCTMantieneExclusionHastaRetorno(t *testing.T) {
	var s serializacionPerfilesCT
	var dentro atomic.Int32
	var solapadas atomic.Int32
	final := make(chan error, 20)
	for range 20 {
		go func() {
			_, liberar, err := s.adquirir(context.Background(), "prf_uno")
			if err == nil {
				if dentro.Add(1) != 1 {
					solapadas.Add(1)
				}
				time.Sleep(time.Millisecond)
				dentro.Add(-1)
				liberar()
			}
			final <- err
		}()
	}
	for range 20 {
		if err := <-final; err != nil {
			t.Fatal(err)
		}
	}
	if solapadas.Load() != 0 || len(s.perfiles) != 0 {
		t.Fatal("solapamiento o turnos sin retirar")
	}
}

// El turno se toma en la frontera HTTP real antes de llegar al handler,
// también para el perfil de la segunda persona de RRHH.
func TestSerializacionPerfilCTFronteraCanceladaNoEjecutaHandler(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	base := &autoridadConsultasContratacionTemporalDesarrollo{sello: soporte.sello, resolvedor: &resolvedorIdentidadDesarrollo{}}
	_, liberar, err := base.serializacionPerfiles.adquirir(context.Background(), soporte.contexto.Resultado.Contexto.PerfilActivoRef)
	if err != nil {
		t.Fatal(err)
	}
	defer liberar()
	h := fronteraSerializadaPrueba(base, principal, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("se ejecutó la petición cancelada") }))
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelar()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionSerializadaPrueba(ctx))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado: %d", w.Code)
	}
	var sobre map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &sobre); err != nil {
		t.Fatal(err)
	}
	if sobre["error"]["codigo"] != "servicio_no_disponible" || sobre["error"]["clave_i18n"] != "api.vec.ruta_exacta.error.servicio_no_disponible" ||
		w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store, no-transform" {
		t.Fatalf("error sin catálogo o cabeceras: %s", w.Body.String())
	}
}
