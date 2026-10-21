package bootstrap

import (
	"context"
	"sync"
	"time"
)

const esperaMaximaPerfilCT = 30 * time.Second

// Esta exclusión solo coordina el proceso de desarrollo. La autoridad y las
// revalidaciones de PostgreSQL siguen decidiendo y consumiendo los permisos.
type serializacionPerfilesCT struct {
	mu       sync.Mutex
	perfiles map[string]*turnoPerfilCT
}

type turnoPerfilCT struct {
	ocupado  chan struct{}
	usuarios int
}

type claveTurnoPerfilCT struct{}
type turnoPerfilCTEnContexto struct {
	serializacion *serializacionPerfilesCT
	perfil        string
}

func (s *serializacionPerfilesCT) adquirir(ctx context.Context, perfil string) (context.Context, func(), error) {
	if ctx == nil || perfil == "" {
		return ctx, nil, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ctx, nil, err
	}
	// Organización vuelve a atravesar la misma frontera dentro del handler.
	// Solo el turno del mismo coordinador y perfil puede reutilizarse.
	if actual, ok := ctx.Value(claveTurnoPerfilCT{}).(turnoPerfilCTEnContexto); ok && actual.serializacion == s && actual.perfil == perfil {
		return ctx, func() {}, nil
	}
	s.mu.Lock()
	if s.perfiles == nil {
		s.perfiles = make(map[string]*turnoPerfilCT)
	}
	turno := s.perfiles[perfil]
	if turno == nil {
		turno = &turnoPerfilCT{ocupado: make(chan struct{}, 1)}
		s.perfiles[perfil] = turno
	}
	turno.usuarios++
	s.mu.Unlock()
	retirar := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		turno.usuarios--
		if turno.usuarios == 0 {
			delete(s.perfiles, perfil)
		}
	}
	espera, cancelar := context.WithTimeout(ctx, esperaMaximaPerfilCT)
	defer cancelar()
	select {
	case turno.ocupado <- struct{}{}:
		if err := espera.Err(); err != nil {
			<-turno.ocupado
			retirar()
			return ctx, nil, err
		}
		liberar := func() { <-turno.ocupado; retirar() }
		return context.WithValue(ctx, claveTurnoPerfilCT{}, turnoPerfilCTEnContexto{s, perfil}), liberar, nil
	case <-espera.Done():
		retirar()
		return ctx, nil, espera.Err()
	}
}
