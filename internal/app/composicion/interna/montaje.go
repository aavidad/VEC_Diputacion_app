package interna

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

// puenteIdentidadCT existe solo dentro de la raíz. El servidor C4 crea el
// token de canal después de recibir su API; entonces se sella una única
// fachada sobre ese servidor antes de entregar la aplicación al llamador.
type puenteIdentidadCT struct {
	fuente  extractorAsercionInstitucional
	api     http.Handler
	fachada atomic.Pointer[FachadaIdentidadOffline]
}

func (p *puenteIdentidadCT) sellar(f *FachadaIdentidadOffline) bool {
	return p != nil && f != nil && p.fachada.CompareAndSwap(nil, f)
}

func (p *puenteIdentidadCT) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || r == nil || r.URL == nil || interfazNulaIdentidadOffline(p.fuente) ||
		manejadorNulo(p.api) || p.fachada.Load() == nil {
		http.Error(w, "servicio no disponible", http.StatusServiceUnavailable)
		return
	}
	asercion, err := p.fuente.ExtraerAsercionProtegida(r)
	if err != nil || len(asercion) == 0 {
		clear(asercion)
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	defer clear(asercion)
	ctx, err := p.fachada.Load().AutenticarYVincular(r.Context(), asercion)
	if err != nil || ctx == nil {
		http.Error(w, "autenticación requerida", http.StatusUnauthorized)
		return
	}
	p.api.ServeHTTP(w, r.WithContext(ctx))
}

type recursoPoolConsultasCT struct {
	pool      *postgresct.PoolConsultasRRHHPostgreSQL
	propiedad atomic.Bool
	unaVez    sync.Once
}

func (r *recursoPoolConsultasCT) reclamarPropiedad() bool {
	return r != nil && r.pool != nil && r.propiedad.CompareAndSwap(false, true)
}

func (r *recursoPoolConsultasCT) cerrar() error {
	if r == nil || r.pool == nil || !r.propiedad.Load() {
		return ErrAplicacionInternaNoDisponible
	}
	r.unaVez.Do(r.pool.Cerrar)
	return nil
}

func nuevaAplicacionLecturaCT(ctx context.Context, cfg Configuracion, p proveedoresLecturaCT) (*AplicacionInterna, error) {
	if ctx == nil || ctx.Err() != nil || cfg.Validar() != nil || p.identidad == nil ||
		interfazNulaIdentidadOffline(p.extractor) || p.servicioVEC == nil ||
		interfazNulaIdentidadOffline(p.autoridadRutas) ||
		interfazNulaIdentidadOffline(p.auditoriaRutas) ||
		interfazNulaIdentidadOffline(p.ambitos) ||
		interfazNulaIdentidadOffline(p.actor.selector) ||
		interfazNulaIdentidadOffline(p.actor.revalidador) ||
		interfazNulaIdentidadOffline(p.actor.resolutor) ||
		interfazNulaIdentidadOffline(p.actor.reloj) || p.consultas.pool == nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	transferida := false
	defer func() {
		if transferida {
			return
		}
		// La adquisición todavía pertenece a la raíz si algún constructor
		// falla. Cerrar del pool es idempotente; recursos ya reclamados por
		// nuevaAplicacionInterna se cierran allí y no se reclaman otra vez.
		p.consultas.pool.Cerrar()
		for _, recurso := range p.recursos {
			if !interfazNulaIdentidadOffline(recurso) && recurso.reclamarPropiedad() {
				_ = cerrarRecursoAplicacionInterna(recurso)
			}
		}
	}()
	p.actor.identidad = p.identidad
	rutas, err := nuevasRutasConsultasRRHH(p.consultas, autoridadContextoConsultaRRHH{
		actor: p.actor, ambitos: p.ambitos,
	})
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	api, err := nuevaAPIInternaCT(p.servicioVEC, rutas, p.autoridadRutas, p.auditoriaRutas)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	puente := &puenteIdentidadCT{fuente: p.extractor, api: api}
	servidor, err := construirServidorInterno(cfg, puente)
	if err != nil {
		return nil, err
	}
	fachada, err := NuevaFachadaIdentidadOffline(p.identidad, servidor)
	if err != nil || !puente.sellar(fachada) {
		_ = servidor.Apagar(context.Background())
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	recursos := append([]recursoCerrableAplicacionInterna(nil), p.recursos...)
	recursos = append(recursos, &recursoPoolConsultasCT{pool: p.consultas.pool})
	aplicacion, err := nuevaAplicacionInterna(servidor, recursos...)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	transferida = true
	return aplicacion, nil
}
