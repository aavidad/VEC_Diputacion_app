package httpseguridad

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// relojSecuenciaIdentidad permite demostrar que una operacion que empieza
// vigente se cierra si cruza una frontera temporal durante una llamada a un
// puerto. Al agotarse conserva el ultimo instante para fallar de forma
// determinista ante llamadas adicionales.
type relojSecuenciaIdentidad struct {
	mu        sync.Mutex
	instantes []time.Time
	indice    int
}

func (r *relojSecuenciaIdentidad) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.instantes) == 0 {
		return time.Time{}
	}
	indice := r.indice
	if indice >= len(r.instantes) {
		indice = len(r.instantes) - 1
	} else {
		r.indice++
	}
	return r.instantes[indice]
}

func TestProyeccionRevalidaRevocacionCuentaYVigencia(t *testing.T) {
	crear := func(t *testing.T) (*ServicioIdentidad, IdentidadSesion, *registroMemoria, *relojFijo) {
		t.Helper()
		ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
		configuracion := configuracionInternaValida()
		verificador := &verificadorFalso{}
		registro := nuevoRegistroMemoria()
		reloj := &relojFijo{ahora: ahora}
		servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, reloj)
		canal := debeCanalTLS(t, servicio, configuracion)
		verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
		identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal))
		if err != nil {
			t.Fatalf("resolver: %v", err)
		}
		return servicio, identidad, registro, reloj
	}

	t.Run("sesion revocada", func(t *testing.T) {
		servicio, identidad, registro, _ := crear(t)
		registro.revocar("sesion-001")
		if _, _, err := servicio.ProyectarPrincipal(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesion revocada aceptada: %v", err)
		}
	})
	t.Run("cuenta inactiva", func(t *testing.T) {
		servicio, identidad, registro, _ := crear(t)
		registro.inactivar("cuenta-tecnica")
		if _, _, err := servicio.ProyectarPrincipal(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("cuenta inactiva aceptada: %v", err)
		}
	})
	t.Run("asercion caducada", func(t *testing.T) {
		servicio, identidad, _, reloj := crear(t)
		reloj.fijar(time.Date(2026, 7, 15, 8, 2, 0, 0, time.UTC))
		if _, _, err := servicio.ProyectarPrincipal(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesion caducada aceptada: %v", err)
		}
	})
}

func TestResolverCierraSiCaducaAutenticacionDuranteAltaDurable(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	frontera := ahora.Add(time.Second)
	reloj := &relojSecuenciaIdentidad{instantes: []time.Time{ahora, ahora, frontera}}
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, reloj)
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.AutenticacionVerificadaEn = frontera.Add(-configuracion.EdadMaximaAutenticacion)
	verificador.fijarAsercion(asercion)

	if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca-frontera-alta"), canal)); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("alta devuelta tras caducar la autenticacion: %v", err)
	}
	registro.mu.Lock()
	sesionesRegistradas := len(registro.sesiones)
	registro.mu.Unlock()
	if sesionesRegistradas != 1 {
		t.Fatalf("la prueba no cruzo la frontera despues del efecto durable: sesiones=%d", sesionesRegistradas)
	}
}

func TestResolverNoEscribeSiCaducaAutenticacionAntesDelAltaDurable(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	frontera := ahora.Add(time.Second)
	reloj := &relojSecuenciaIdentidad{instantes: []time.Time{ahora, frontera}}
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, reloj)
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.AutenticacionVerificadaEn = frontera.Add(-configuracion.EdadMaximaAutenticacion)
	verificador.fijarAsercion(asercion)

	if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca-frontera-previa"), canal)); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("alta iniciada con autenticacion ya caducada: %v", err)
	}
	registro.mu.Lock()
	sesionesRegistradas := len(registro.sesiones)
	registro.mu.Unlock()
	if sesionesRegistradas != 0 {
		t.Fatalf("se produjo un efecto durable tras caducar la autenticacion: sesiones=%d", sesionesRegistradas)
	}
}

func TestProyeccionCierraSiCaducaAutenticacionDuranteRevalidacion(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	relojInicial := &relojFijo{ahora: ahora}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, relojInicial)
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	frontera := ahora.Add(time.Second)
	asercion.AutenticacionVerificadaEn = frontera.Add(-configuracion.EdadMaximaAutenticacion)
	verificador.fijarAsercion(asercion)
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca-frontera-proyeccion"), canal))
	if err != nil {
		t.Fatalf("resolver identidad previa: %v", err)
	}

	servicio.reloj = &relojSecuenciaIdentidad{instantes: []time.Time{ahora, frontera}}
	if _, _, err = servicio.ProyectarPrincipal(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("principal devuelto tras caducar la autenticacion: %v", err)
	}
	registro.mu.Lock()
	consultaRealizada := registro.ultimaConsulta.SesionRef
	registro.mu.Unlock()
	if consultaRealizada == "" {
		t.Fatal("la prueba no cruzo la frontera despues de revalidar la sesion durable")
	}
}
