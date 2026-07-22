package postgrespublico

import (
	"context"
	"errors"
	"time"
)

const (
	duracionSondaDisponibilidadPublica = 2 * time.Second
	duracionCacheDisponibilidadSana    = 10 * time.Second
	duracionCacheDisponibilidadFallida = time.Second
)

// ComprobarDisponibilidad verifica que la cache de manifiesto ya fue cargada
// y que la ancla publicada sigue siendo la esperada. Solo la primera solicitud
// inicia una sonda: las seguidoras fallan inmediatamente mientras siga en
// curso, para no agotar ni el pool ni las goroutines del servidor.
func (f *Fuente) ComprobarDisponibilidad(_ context.Context) error {
	if f == nil || !f.configuracionValida() || f.cacheManifiesto.Load() == nil {
		return ErrPostgreSQLPublicoNoDisponible
	}
	f.disponibilidadMu.Lock()
	if time.Now().Before(f.disponibilidadHasta) {
		err := f.disponibilidadErr
		f.disponibilidadMu.Unlock()
		return err
	}
	if f.disponibilidadEnCurso {
		f.disponibilidadMu.Unlock()
		return ErrPostgreSQLPublicoNoDisponible
	}
	f.disponibilidadEnCurso = true
	f.disponibilidadMu.Unlock()

	err := f.sondearDisponibilidad()
	vida := duracionCacheDisponibilidadSana
	if err != nil {
		vida = duracionCacheDisponibilidadFallida
		err = ErrPostgreSQLPublicoNoDisponible
	}
	f.disponibilidadMu.Lock()
	f.disponibilidadErr = err
	f.disponibilidadHasta = time.Now().Add(vida)
	f.disponibilidadEnCurso = false
	f.disponibilidadMu.Unlock()
	return err
}

func (f *Fuente) sondearDisponibilidad() error {
	ctx, cancelar := context.WithTimeout(context.Background(), duracionSondaDisponibilidadPublica)
	defer cancelar()
	if f.sondaDisponibilidadPrueba != nil {
		return f.sondaDisponibilidadPrueba(ctx)
	}
	var ancla string
	err := f.pool.QueryRow(ctx, `
		SELECT manifiesto_sha256
		  FROM vec_bolsa_publica_lectura.fuente_publica_v2
		 WHERE control_id IS TRUE`).Scan(&ancla)
	if err != nil || !huellasIguales(ancla, f.manifiestoSHA256) {
		return errors.New("bolsa publica: disponibilidad no confirmada")
	}
	return nil
}
