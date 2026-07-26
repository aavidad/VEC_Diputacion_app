package cobertura

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

var errLecturaResultadoHistoricoTCBPrueba = errors.New("lectura historica tcb de prueba")

type sesionLecturaResultadoHistoricoTCBPrueba struct {
	mu       sync.Mutex
	datos    DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB
	err      error
	panico   bool
	llamadas int
}

func (s *sesionLecturaResultadoHistoricoTCBPrueba) LeerResultadoHistoricoTCB(
	ctx context.Context,
	consulta ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) (DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB, error) {
	if err := ctx.Err(); err != nil {
		return DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			err
	}
	if _, err := consulta.DatosLectura(); err != nil {
		return DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
			err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.llamadas++
	if s.panico {
		panic("panic privado de sesión TCB")
	}
	return s.datos, s.err
}

func (s *sesionLecturaResultadoHistoricoTCBPrueba) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.llamadas
}

type ejecutorLecturaResultadoHistoricoTCBPrueba struct {
	sesion    *sesionLecturaResultadoHistoricoTCBPrueba
	modo      string
	iniciada  chan struct{}
	liberar   chan struct{}
	terminada chan struct{}
}

func (e *ejecutorLecturaResultadoHistoricoTCBPrueba) EjecutarLecturaResultadoHistoricoTCB(
	ctx context.Context,
	callback func(SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB) error,
) error {
	switch e.modo {
	case "panic_ejecutor":
		panic("panic privado de ejecutor TCB")
	case "panic_despues":
		if err := callback(e.sesion); err != nil {
			return err
		}
		panic("panic de ejecutor TCB tras callback")
	case "error_antes":
		return errLecturaResultadoHistoricoTCBPrueba
	case "sin_callback":
		return nil
	case "doble":
		if err := callback(e.sesion); err != nil {
			return err
		}
		return callback(e.sesion)
	case "concurrente":
		inicio := make(chan struct{})
		errores := make(chan error, 2)
		for range 2 {
			go func() {
				<-inicio
				errores <- callback(e.sesion)
			}()
		}
		close(inicio)
		primero, segundo := <-errores, <-errores
		if primero != nil {
			return primero
		}
		return segundo
	case "retenido":
		go func() {
			close(e.iniciada)
			<-e.liberar
			_ = callback(e.sesion)
			close(e.terminada)
		}()
		<-e.iniciada
		return nil
	default:
		return callback(e.sesion)
	}
}

func escenarioLecturaResultadoHistoricoTCBPrueba(
	t *testing.T,
) (
	SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) {
	t.Helper()
	base := identidadOperacionDecisionCoberturaPrueba()
	solicitud := solicitudRecuperacionOperacionDecisionCoberturaPrueba(
		t,
		base.expedienteRef,
	)
	_, reserva := solicitudReservaOperacionDecisionCoberturaPrueba(t, base)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, reserva)
	return solicitud,
		DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
			Encontrado: true,
			Reserva: datosReservaTerminalOperacionDecisionCoberturaPrueba(
				t,
				reserva,
			),
			Recibo:      recibo,
			ObservadaEn: recibo.ConfirmadaEn,
		}
}

func nuevoLectorResultadoHistoricoTCBPrueba(
	t *testing.T,
	ejecutor EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) LectorResultadoHistoricoOperacionDecisionCobertura {
	t.Helper()
	lector, err :=
		NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	return lector
}

func TestLectorResultadoHistoricoTCBConstruyeUnionEnNucleo(t *testing.T) {
	solicitud, datos := escenarioLecturaResultadoHistoricoTCBPrueba(t)
	sesion := &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos}
	lector := nuevoLectorResultadoHistoricoTCBPrueba(
		t,
		&ejecutorLecturaResultadoHistoricoTCBPrueba{sesion: sesion},
	)
	resultado, err := lector.LeerResultadoHistoricoOperacionDecisionCobertura(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); !confirmado {
		t.Fatal("el núcleo no construyó el terminal confirmado")
	}
	if sesion.total() != 1 {
		t.Fatalf("número de lecturas inesperado: %d", sesion.total())
	}

	datos.Encontrado = false
	datos.Reserva = DatosReservaTerminalOperacionDecisionCobertura{}
	datos.Recibo = ReciboOperacionDecisionCobertura{}
	sesion = &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos}
	lector = nuevoLectorResultadoHistoricoTCBPrueba(
		t,
		&ejecutorLecturaResultadoHistoricoTCBPrueba{sesion: sesion},
	)
	resultado, err = lector.LeerResultadoHistoricoOperacionDecisionCobertura(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !resultado.NoObservablePara(solicitud) {
		t.Fatal("el núcleo no construyó el terminal no observable")
	}
}

func TestLectorResultadoHistoricoTCBFallaCerradoAnteProtocoloHostil(t *testing.T) {
	solicitud, datos := escenarioLecturaResultadoHistoricoTCBPrueba(t)
	casos := map[string]*ejecutorLecturaResultadoHistoricoTCBPrueba{
		"sin_callback": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
			modo:   "sin_callback",
		},
		"doble": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
			modo:   "doble",
		},
		"concurrente": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
			modo:   "concurrente",
		},
		"panic_ejecutor": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
			modo:   "panic_ejecutor",
		},
		"panic_despues": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
			modo:   "panic_despues",
		},
		"panic_sesion": {
			sesion: &sesionLecturaResultadoHistoricoTCBPrueba{
				datos:  datos,
				panico: true,
			},
		},
	}
	for nombre, ejecutor := range casos {
		t.Run(nombre, func(t *testing.T) {
			lector := nuevoLectorResultadoHistoricoTCBPrueba(t, ejecutor)
			resultado, err :=
				lector.LeerResultadoHistoricoOperacionDecisionCobertura(
					context.Background(),
					solicitud,
				)
			if !errors.Is(
				err,
				ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
			) {
				t.Fatalf("protocolo hostil no cerrado: %v", err)
			}
			if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); confirmado ||
				resultado.NoObservablePara(solicitud) {
				t.Fatal("protocolo hostil elevó una unión terminal")
			}
			if (nombre == "doble" || nombre == "concurrente") &&
				ejecutor.sesion.total() != 1 {
				t.Fatalf(
					"%s inició %d lecturas",
					nombre,
					ejecutor.sesion.total(),
				)
			}
		})
	}

	ejecutor := &ejecutorLecturaResultadoHistoricoTCBPrueba{
		sesion:    &sesionLecturaResultadoHistoricoTCBPrueba{datos: datos},
		modo:      "retenido",
		iniciada:  make(chan struct{}),
		liberar:   make(chan struct{}),
		terminada: make(chan struct{}),
	}
	lector := nuevoLectorResultadoHistoricoTCBPrueba(t, ejecutor)
	_, err := lector.LeerResultadoHistoricoOperacionDecisionCobertura(
		context.Background(),
		solicitud,
	)
	if !errors.Is(
		err,
		ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	) {
		t.Fatalf("callback retenido no cerrado: %v", err)
	}
	close(ejecutor.liberar)
	select {
	case <-ejecutor.terminada:
	case <-time.After(time.Second):
		t.Fatal("callback retenido no terminó tras su liberación")
	}
}

func TestLectorResultadoHistoricoTCBClasificaDatosYDisponibilidad(t *testing.T) {
	solicitud, datos := escenarioLecturaResultadoHistoricoTCBPrueba(t)
	t.Run("proyeccion_cruzada_no_confiable", func(t *testing.T) {
		cruzados := datos
		cruzados.Reserva.ExpedienteRef = "expediente_historico_tcb_cruzado_01"
		lector := nuevoLectorResultadoHistoricoTCBPrueba(
			t,
			&ejecutorLecturaResultadoHistoricoTCBPrueba{
				sesion: &sesionLecturaResultadoHistoricoTCBPrueba{
					datos: cruzados,
				},
			},
		)
		_, err := lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			context.Background(),
			solicitud,
		)
		if !errors.Is(
			err,
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		) {
			t.Fatalf("proyección cruzada mal clasificada: %v", err)
		}
	})
	t.Run("historia_divergente_tipificada", func(t *testing.T) {
		lector := nuevoLectorResultadoHistoricoTCBPrueba(
			t,
			&ejecutorLecturaResultadoHistoricoTCBPrueba{
				sesion: &sesionLecturaResultadoHistoricoTCBPrueba{
					datos: datos,
					err:   ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
				},
			},
		)
		resultado, err :=
			lector.LeerResultadoHistoricoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			)
		if !errors.Is(
			err,
			ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
		) {
			t.Fatalf("divergencia tipificada mal clasificada: %v", err)
		}
		if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); confirmado {
			t.Fatal("resultado no cero junto con error se elevó a confirmado")
		}
	})
	t.Run("lectura_no_disponible", func(t *testing.T) {
		lector := nuevoLectorResultadoHistoricoTCBPrueba(
			t,
			&ejecutorLecturaResultadoHistoricoTCBPrueba{
				sesion: &sesionLecturaResultadoHistoricoTCBPrueba{
					datos: datos,
					err:   errLecturaResultadoHistoricoTCBPrueba,
				},
			},
		)
		resultado, err := lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			context.Background(),
			solicitud,
		)
		if !errors.Is(
			err,
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
		) {
			t.Fatalf("indisponibilidad mal clasificada: %v", err)
		}
		if _, confirmado := resultado.ReciboConfirmadoPara(solicitud); confirmado {
			t.Fatal("resultado no cero junto con error se elevó a confirmado")
		}
	})
	t.Run("proyeccion_no_confiable", func(t *testing.T) {
		noConfiables := datos
		noConfiables.ObservadaEn = time.Time{}
		lector := nuevoLectorResultadoHistoricoTCBPrueba(
			t,
			&ejecutorLecturaResultadoHistoricoTCBPrueba{
				sesion: &sesionLecturaResultadoHistoricoTCBPrueba{
					datos: noConfiables,
				},
			},
		)
		_, err := lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			context.Background(),
			solicitud,
		)
		if !errors.Is(
			err,
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
		) {
			t.Fatalf("proyección no confiable mal clasificada: %v", err)
		}
	})
}

func TestSuperficieTCBResultadoHistoricoSoloExponeDatosCrudos(t *testing.T) {
	tipoSesion := reflect.TypeOf(
		(*SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB)(nil),
	).Elem()
	metodo, existe := tipoSesion.MethodByName("LeerResultadoHistoricoTCB")
	if !existe || metodo.Type.NumOut() != 2 ||
		metodo.Type.Out(0) != reflect.TypeOf(
			DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
		) ||
		metodo.Type.Out(0) == reflect.TypeOf(
			ResultadoHistoricoOperacionDecisionCobertura{},
		) {
		t.Fatalf("sesión TCB expone una unión autoritativa: %v", metodo.Type)
	}
	if _, existe := reflect.TypeOf(
		DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{},
	).FieldByName("HuellaOrdenSHA256"); existe {
		t.Fatal("la evidencia exterior conserva la huella privada de la orden")
	}
}

var _ EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*ejecutorLecturaResultadoHistoricoTCBPrueba)(nil)
var _ SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*sesionLecturaResultadoHistoricoTCBPrueba)(nil)
