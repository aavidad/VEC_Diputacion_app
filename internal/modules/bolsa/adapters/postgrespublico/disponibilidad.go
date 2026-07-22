package postgrespublico

import (
	"context"
	"errors"
	"log"
	"time"
)

const (
	duracionSondaDisponibilidadPublica = 2 * time.Second
	duracionCacheDisponibilidadSana    = 10 * time.Second
	duracionCacheDisponibilidadFallida = time.Second
	periodoIntegridadDisponibilidad    = 5 * time.Minute
	reintentoIntegridadMinimo          = time.Minute
	reintentoIntegridadMaximo          = 15 * time.Minute
	vigenciaIntegridadDisponibilidad   = 7 * time.Minute
	duracionIntegridadDisponibilidad   = 30 * time.Second
)

// ComprobarDisponibilidad verifica que la cache de manifiesto ya fue cargada
// y que la ancla publicada sigue siendo la esperada. Solo la primera solicitud
// inicia una sonda: las seguidoras fallan inmediatamente mientras siga en
// curso, para no agotar ni el pool ni las goroutines del servidor.
func (f *Fuente) ComprobarDisponibilidad(ctx context.Context) error {
	if ctx == nil || f == nil || !f.configuracionValida() || f.cacheManifiesto.Load() == nil {
		return ErrPostgreSQLPublicoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrPostgreSQLPublicoNoDisponible, err)
	}
	if !f.integridadReciente() {
		f.registrarEstadoDisponibilidad(false)
		return ErrPostgreSQLPublicoNoDisponible
	}
	f.disponibilidadMu.Lock()
	if time.Now().Before(f.disponibilidadHasta) {
		err := f.disponibilidadErr
		cancelada := f.disponibilidadCancelada
		f.disponibilidadMu.Unlock()
		if !cancelada {
			f.registrarEstadoDisponibilidad(err == nil)
		}
		return err
	}
	if f.disponibilidadEnCurso {
		f.disponibilidadMu.Unlock()
		return ErrPostgreSQLPublicoNoDisponible
	}
	f.disponibilidadEnCurso = true
	f.disponibilidadMu.Unlock()

	err := f.sondearDisponibilidad(ctx)
	// La cancelacion del cliente no describe una transicion de la dependencia,
	// pero se cachea brevemente para impedir churn de Begin/CancelQuery.
	if causa := ctx.Err(); causa != nil {
		errorCancelacion := errors.Join(ErrPostgreSQLPublicoNoDisponible, causa)
		f.disponibilidadMu.Lock()
		f.disponibilidadErr = errorCancelacion
		f.disponibilidadCancelada = true
		f.disponibilidadHasta = time.Now().Add(duracionCacheDisponibilidadFallida)
		f.disponibilidadEnCurso = false
		f.disponibilidadMu.Unlock()
		return errorCancelacion
	}
	// La worker puede invalidar la proyeccion mientras la sonda ligera esta en
	// vuelo. Nunca conviertas ese resultado obsoleto en un nuevo verde.
	if err == nil && !f.integridadReciente() {
		err = ErrDatosPostgreSQLPublicosNoConfiables
	}
	vida := duracionCacheDisponibilidadSana
	if err != nil {
		vida = duracionCacheDisponibilidadFallida
		err = errors.Join(ErrPostgreSQLPublicoNoDisponible, err)
	}
	f.disponibilidadMu.Lock()
	f.disponibilidadErr = err
	f.disponibilidadCancelada = false
	f.disponibilidadHasta = time.Now().Add(vida)
	f.disponibilidadEnCurso = false
	disponible := err == nil
	f.disponibilidadMu.Unlock()
	f.registrarEstadoDisponibilidad(disponible)
	return err
}

func (f *Fuente) sondearDisponibilidad(ctxPadre context.Context) error {
	ctx, cancelar := context.WithTimeout(ctxPadre, duracionSondaDisponibilidadPublica)
	defer cancelar()
	if f.sondaDisponibilidadPrueba != nil {
		return f.sondaDisponibilidadPrueba(ctx)
	}
	var ancla string
	err := f.pool.QueryRow(ctx, `
		SELECT manifiesto_sha256
		  FROM vec_bolsa_publica_lectura.fuente_publica_v2
		 WHERE control_id IS TRUE`).Scan(&ancla)
	if err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	if !huellasIguales(ancla, f.manifiestoSHA256) {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	return nil
}

func (f *Fuente) integridadReciente() bool {
	f.integridadMu.RLock()
	defer f.integridadMu.RUnlock()
	return f.integridadErr == nil && time.Now().Before(f.integridadHasta)
}

func (f *Fuente) iniciarVigilanciaIntegridad() {
	ctx, cancelar := context.WithCancel(context.Background())
	terminada := make(chan struct{})
	f.integridadMu.Lock()
	f.integridadErr = nil
	f.integridadHasta = time.Now().Add(vigenciaIntegridadDisponibilidad)
	f.integridadCancelar = cancelar
	f.integridadTerminada = terminada
	f.integridadMu.Unlock()
	go f.vigilarIntegridad(ctx, terminada)
}

func (f *Fuente) vigilarIntegridad(ctx context.Context, terminada chan<- struct{}) {
	defer close(terminada)
	espera := duracionIntegridadConJitter(periodoIntegridadDisponibilidad, time.Now())
	reintento := reintentoIntegridadMinimo
	for {
		temporizador := time.NewTimer(espera)
		select {
		case <-ctx.Done():
			if !temporizador.Stop() {
				<-temporizador.C
			}
			return
		case <-temporizador.C:
		}
		ctxSonda, cancelar := context.WithTimeout(ctx, duracionIntegridadDisponibilidad)
		err := f.ejecutarIntegridadProtegida(ctxSonda)
		cancelar()
		if ctx.Err() != nil {
			return
		}
		f.actualizarIntegridad(err)
		if err == nil {
			reintento = reintentoIntegridadMinimo
			espera = duracionIntegridadConJitter(periodoIntegridadDisponibilidad, time.Now())
		} else {
			espera = duracionIntegridadConJitter(reintento, time.Now())
			if reintento < reintentoIntegridadMaximo/2 {
				reintento *= 2
			} else {
				reintento = reintentoIntegridadMaximo
			}
		}
	}
}

func (f *Fuente) ejecutarIntegridadProtegida(ctx context.Context) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrPostgreSQLPublicoNoDisponible
		}
	}()
	return f.comprobarIntegridadProyeccion(ctx)
}

func duracionIntegridadConJitter(base time.Duration, ahora time.Time) time.Duration {
	if base <= 0 {
		return 0
	}
	margen := base / 10
	if margen == 0 {
		return base
	}
	semilla := ahora.UnixNano() & int64(^uint64(0)>>1)
	desplazamiento := time.Duration(semilla%int64(2*margen+1)) - margen
	resultado := base + desplazamiento
	if resultado > reintentoIntegridadMaximo {
		return reintentoIntegridadMaximo
	}
	return resultado
}

func (f *Fuente) comprobarIntegridadProyeccion(ctx context.Context) error {
	if f.sondaIntegridadPrueba != nil {
		return f.sondaIntegridadPrueba(ctx)
	}
	tx, err := f.iniciarLectura(ctx, true)
	if err != nil {
		return err
	}
	defer rollbackPostgreSQLPublico(tx)
	// Reutiliza la verificacion criptografica integral del arranque. Su coste
	// queda limitado a 12.000 filas/256 MiB, una unica worker y 30 segundos. La
	// cadencia sana de cinco minutos limita su ocupacion teorica al 10% de una
	// conexion; el jitter evita sincronizar replicas y el fallo aplica backoff.
	if _, err := f.construirCacheManifiesto(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errorPostgreSQLPublico(ctx, err)
	}
	return nil
}

func (f *Fuente) actualizarIntegridad(err error) {
	f.integridadMu.Lock()
	f.integridadErr = err
	if err == nil {
		f.integridadHasta = time.Now().Add(vigenciaIntegridadDisponibilidad)
	} else {
		f.integridadHasta = time.Time{}
	}
	f.integridadMu.Unlock()
	if err != nil {
		f.registrarEstadoDisponibilidad(false)
	}
}

func (f *Fuente) registrarEstadoDisponibilidad(disponible bool) {
	f.disponibilidadMu.Lock()
	transicion := !f.disponibilidadEstadoConocido || f.disponibilidadDisponible != disponible
	f.disponibilidadEstadoConocido = true
	f.disponibilidadDisponible = disponible
	observador := f.observadorDisponibilidad
	f.disponibilidadMu.Unlock()
	if !transicion {
		return
	}
	if observador == nil {
		observador = observarTransicionDisponibilidad
	}
	observador(disponible)
}

func observarTransicionDisponibilidad(disponible bool) {
	// Deliberadamente no registra errores, SQL, DSN, nombres de filas ni
	// payloads: solo la transicion operativa necesaria para alerta y auditoria.
	log.Printf("bolsa publica: transicion de disponibilidad disponible=%t", disponible)
}
