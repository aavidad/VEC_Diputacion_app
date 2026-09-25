package bootstrap

import (
	"context"
	"strings"
	"sync"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// cargarCatalogoCorreoLlamamientoBolsa lee y valida el catálogo del correo
// personalizado. Un catálogo defectuoso deja B7 sin componer.
func cargarCatalogoCorreoLlamamientoBolsa() (dominiobolsa.CatalogoCorreoLlamamiento, error) {
	datos, err := config.CatalogoCorreoLlamamientoBolsa()
	if err != nil {
		return dominiobolsa.CatalogoCorreoLlamamiento{}, err
	}
	return dominiobolsa.CatalogoCorreoLlamamientoDesdeJSON(datos)
}

// fuentePersonalizacionB7 se crea antes que la fuente de bolsas constituidas
// (que se compone después de las rutas de Contratación) y se enlaza con ella
// una sola vez al terminar la composición. Sin fuente enlazada, un correo con
// marcadores personales se rechaza como no disponible.
type fuentePersonalizacionB7 struct {
	mu     sync.RWMutex
	fuente *fuenteConstituidaRRHHDesarrollo
}

func (f *fuentePersonalizacionB7) fijar(fuente *fuenteConstituidaRRHHDesarrollo) {
	if f == nil || fuente == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fuente == nil {
		f.fuente = fuente
	}
}

func (f *fuentePersonalizacionB7) DatosPersonalizacionLlamamiento(ctx context.Context, bolsaRef string, participaciones []string) (map[string]puertosbolsa.DatosPersonalizacionLlamamiento, error) {
	if f == nil || ctx == nil {
		return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
	}
	f.mu.RLock()
	fuente := f.fuente
	f.mu.RUnlock()
	if fuente == nil {
		return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
	}
	return fuente.datosPersonalizacion(ctx, bolsaRef, participaciones)
}

// datosPersonalizacion resuelve nombre, apellidos, posición vigente y
// denominación de la bolsa desde las mismas fuentes que la lectura RRHH: la
// constitución, el orden vigente y el staging protegido del acta.
func (f *fuenteConstituidaRRHHDesarrollo) datosPersonalizacion(ctx context.Context, bolsaRef string, participaciones []string) (map[string]puertosbolsa.DatosPersonalizacionLlamamiento, error) {
	if f == nil || f.repositorio == nil || f.orden == nil || f.recuperador == nil || bolsaRef == "" {
		return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
	}
	buscadas := make(map[string]bool, len(participaciones))
	for _, p := range participaciones {
		buscadas[p] = true
	}
	vigentes, err := f.repositorio.ListarVigentes(ctx)
	if err != nil {
		return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
	}
	for _, vigente := range vigentes {
		if vigente.Bolsa.BolsaRef != bolsaRef {
			continue
		}
		orden, err := f.orden.Consultar(ctx, bolsaRef)
		if err != nil {
			return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
		}
		posiciones := make(map[string]int, len(orden.Posiciones))
		for _, p := range orden.Posiciones {
			if p.OrdenVigente != nil {
				posiciones[p.ParticipacionRef] = int(*p.OrdenVigente)
			} else {
				posiciones[p.ParticipacionRef] = int(p.OrdenActa)
			}
		}
		entradas, err := f.repositorio.Entradas(ctx, vigente.Instantanea.InstantaneaRef, vigente.Instantanea.Version)
		if err != nil {
			return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
		}
		lote, _, existe, err := f.recuperador.RecuperarLote(ctx, vigente.Bolsa.HuellaListadoSHA256, vigente.CategoriaRef)
		if err != nil || !existe {
			return nil, puertosbolsa.ErrPersonalizacionLlamamientoNoDisponible
		}
		type identidad struct{ nombre, apellidos string }
		filas := make(map[int]identidad, len(lote.Aceptadas))
		for _, fila := range lote.Aceptadas {
			filas[fila.Numero] = identidad{
				nombre:    strings.Join(strings.Fields(fila.Identidad.Nombre), " "),
				apellidos: strings.Join(strings.Fields(fila.Identidad.PrimerApellido+" "+fila.Identidad.SegundoApellido), " "),
			}
		}
		denominacion := f.denominacion(vigente.CategoriaRef)
		out := make(map[string]puertosbolsa.DatosPersonalizacionLlamamiento, len(buscadas))
		for _, entrada := range entradas {
			if !buscadas[entrada.ParticipacionRef] {
				continue
			}
			visible, ok := filas[entrada.FilaNumero]
			if !ok {
				continue
			}
			out[entrada.ParticipacionRef] = puertosbolsa.DatosPersonalizacionLlamamiento{Nombre: visible.nombre, Apellidos: visible.apellidos, Posicion: posiciones[entrada.ParticipacionRef], Bolsa: denominacion}
		}
		return out, nil
	}
	return map[string]puertosbolsa.DatosPersonalizacionLlamamiento{}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararSolicitudVistaPreviaLlamamiento(ctx context.Context, entrada bolsahttp.EntradaVistaPreviaLlamamiento) (puertosbolsa.SolicitudVistaPreviaLlamamiento, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return puertosbolsa.SolicitudVistaPreviaLlamamiento{}, err
	}
	return puertosbolsa.SolicitudVistaPreviaLlamamiento{ContextoActor: contexto.Resultado.Contexto, BolsaRef: entrada.BolsaRef, ParticipacionRef: entrada.ParticipacionRef, Configuracion: entrada.Configuracion}, nil
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararConsultaPlantillaCorreoLlamamiento(ctx context.Context) (dominiovec.ContextoActor, error) {
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return dominiovec.ContextoActor{}, err
	}
	return contexto.Resultado.Contexto, nil
}
