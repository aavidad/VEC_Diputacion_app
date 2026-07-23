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

type transicionDisponibilidad struct {
	disponible bool
	observar   func(bool)
}

// ComprobarDisponibilidad verifica que la cache de manifiesto ya fue cargada
// y que la ancla publicada sigue siendo la esperada. Las solicitudes
// concurrentes comparten una unica sonda con timeout propio: cancelar una
// solicitud deja de esperarla, pero no altera el resultado global.
func (f *Fuente) ComprobarDisponibilidad(ctx context.Context) error {
	if ctx == nil || f == nil || !f.configuracionValida() || f.cacheManifiesto.Load() == nil {
		return ErrPostgreSQLPublicoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrPostgreSQLPublicoNoDisponible, err)
	}

	for {
		f.integridadMu.RLock()
		if !f.integridadRecienteBloqueada() {
			f.disponibilidadMu.Lock()
			drenar := f.encolarEstadoDisponibilidadBloqueada(false)
			f.disponibilidadMu.Unlock()
			f.integridadMu.RUnlock()
			if drenar {
				f.drenarTransicionesDisponibilidad()
			}
			return ErrPostgreSQLPublicoNoDisponible
		}
		generacion := f.integridadGeneracion

		f.disponibilidadMu.Lock()
		if f.disponibilidadCerrada {
			f.disponibilidadMu.Unlock()
			f.integridadMu.RUnlock()
			return ErrPostgreSQLPublicoNoDisponible
		}
		if time.Now().Before(f.disponibilidadHasta) {
			err := f.disponibilidadErr
			f.disponibilidadMu.Unlock()
			f.integridadMu.RUnlock()
			return err
		}
		terminada := f.disponibilidadSondaTerminada
		if terminada == nil {
			ctxSonda, cancelar := context.WithTimeout(context.Background(), duracionSondaDisponibilidadPublica)
			terminada = make(chan struct{})
			f.disponibilidadSondaTerminada = terminada
			f.disponibilidadSondaCancelar = cancelar
			go func() {
				defer cancelar()
				f.completarSondaDisponibilidad(ctxSonda, generacion, terminada)
			}()
		}
		f.disponibilidadMu.Unlock()
		f.integridadMu.RUnlock()

		select {
		case <-ctx.Done():
			return errors.Join(ErrPostgreSQLPublicoNoDisponible, ctx.Err())
		case <-terminada:
			// Relee integridad y cache bajo el mismo orden de bloqueos. Una
			// invalidacion posterior a la sonda debe prevalecer en la respuesta.
		}
	}
}

func (f *Fuente) sondearDisponibilidad(ctx context.Context) error {
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

func (f *Fuente) completarSondaDisponibilidad(
	ctx context.Context,
	generacion uint64,
	terminada chan struct{},
) {
	err := f.sondearDisponibilidad(ctx)
	vida := duracionCacheDisponibilidadSana
	if err != nil {
		vida = duracionCacheDisponibilidadFallida
		err = errors.Join(ErrPostgreSQLPublicoNoDisponible, err)
	}

	f.integridadMu.RLock()
	if err == nil && (generacion != f.integridadGeneracion || !f.integridadRecienteBloqueada()) {
		err = errors.Join(ErrPostgreSQLPublicoNoDisponible, ErrDatosPostgreSQLPublicosNoConfiables)
		vida = duracionCacheDisponibilidadFallida
	}
	f.disponibilidadMu.Lock()
	drenar := false
	if f.disponibilidadSondaTerminada == terminada {
		if !f.disponibilidadCerrada {
			f.disponibilidadErr = err
			f.disponibilidadHasta = time.Now().Add(vida)
			drenar = f.encolarEstadoDisponibilidadBloqueada(err == nil)
		}
		f.disponibilidadSondaTerminada = nil
		f.disponibilidadSondaCancelar = nil
		close(terminada)
	}
	f.disponibilidadMu.Unlock()
	f.integridadMu.RUnlock()
	if drenar {
		f.drenarTransicionesDisponibilidad()
	}
}

func (f *Fuente) integridadReciente() bool {
	f.integridadMu.RLock()
	defer f.integridadMu.RUnlock()
	return f.integridadRecienteBloqueada()
}

func (f *Fuente) integridadRecienteBloqueada() bool {
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
	tx, err := f.iniciarLecturaConTimeout(ctx, true, duracionIntegridadDisponibilidad)
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
	f.integridadGeneracion++
	f.integridadErr = err
	if err == nil {
		f.integridadHasta = time.Now().Add(vigenciaIntegridadDisponibilidad)
	} else {
		f.integridadHasta = time.Time{}
	}
	drenar := false
	if err != nil {
		f.disponibilidadMu.Lock()
		drenar = f.encolarEstadoDisponibilidadBloqueada(false)
		f.disponibilidadMu.Unlock()
	}
	f.integridadMu.Unlock()
	if drenar {
		f.drenarTransicionesDisponibilidad()
	}
}

func (f *Fuente) registrarEstadoDisponibilidad(disponible bool) {
	f.disponibilidadMu.Lock()
	drenar := f.encolarEstadoDisponibilidadBloqueada(disponible)
	f.disponibilidadMu.Unlock()
	if drenar {
		f.drenarTransicionesDisponibilidad()
	}
}

func (f *Fuente) encolarEstadoDisponibilidadBloqueada(disponible bool) bool {
	transicion := !f.disponibilidadEstadoConocido || f.disponibilidadDisponible != disponible
	f.disponibilidadEstadoConocido = true
	f.disponibilidadDisponible = disponible
	if !transicion {
		return false
	}
	observador := f.observadorDisponibilidad
	if observador == nil {
		observador = observarTransicionDisponibilidad
	}
	f.disponibilidadPendientes = append(f.disponibilidadPendientes, transicionDisponibilidad{
		disponible: disponible,
		observar:   observador,
	})
	if f.disponibilidadNotificando {
		return false
	}
	f.disponibilidadNotificando = true
	return true
}

func (f *Fuente) drenarTransicionesDisponibilidad() {
	for {
		f.disponibilidadMu.Lock()
		if len(f.disponibilidadPendientes) == 0 {
			f.disponibilidadNotificando = false
			f.disponibilidadMu.Unlock()
			return
		}
		transicion := f.disponibilidadPendientes[0]
		f.disponibilidadPendientes[0] = transicionDisponibilidad{}
		f.disponibilidadPendientes = f.disponibilidadPendientes[1:]
		f.disponibilidadMu.Unlock()
		notificarTransicionDisponibilidad(transicion)
	}
}

func notificarTransicionDisponibilidad(transicion transicionDisponibilidad) {
	defer func() {
		_ = recover()
	}()
	transicion.observar(transicion.disponible)
}

func observarTransicionDisponibilidad(disponible bool) {
	// Deliberadamente no registra errores, SQL, DSN, nombres de filas ni
	// payloads: solo la transicion operativa necesaria para alerta y auditoria.
	log.Printf("bolsa publica: transicion de disponibilidad disponible=%t", disponible)
}
