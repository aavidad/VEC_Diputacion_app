package bootstrap

import (
	"context"
	"sync"
	"time"
)

const tiempoMaximoCargaCompartidaBolsasRRHH = 30 * time.Second

// cargaCompartidaBolsasRRHH une en una sola lectura las peticiones de resumen
// que llegan mientras otra está en curso. No es una caché: al terminar la
// lectura se olvida, y una mutación (invalidar) impide que las peticiones
// posteriores se sumen a una lectura empezada antes de ella.
type cargaCompartidaBolsasRRHH struct {
	mu         sync.Mutex
	generacion uint64
	enCurso    *lecturaCompartidaBolsasRRHH
}

type lecturaCompartidaBolsasRRHH struct {
	generacion uint64
	lista      chan struct{}
	datos      datasetBolsasRRHHDesarrollo
	err        error
}

func (c *cargaCompartidaBolsasRRHH) invalidar() {
	c.mu.Lock()
	c.generacion++
	c.mu.Unlock()
}

func (c *cargaCompartidaBolsasRRHH) obtener(ctx context.Context, cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)) (datasetBolsasRRHHDesarrollo, error) {
	c.mu.Lock()
	if lectura := c.enCurso; lectura != nil && lectura.generacion == c.generacion {
		c.mu.Unlock()
		select {
		case <-lectura.lista:
			return lectura.datos, lectura.err
		case <-ctx.Done():
			return datasetBolsasRRHHDesarrollo{}, ctx.Err()
		}
	}
	lectura := &lecturaCompartidaBolsasRRHH{generacion: c.generacion, lista: make(chan struct{})}
	c.enCurso = lectura
	c.mu.Unlock()
	// La lectura compartida no muere si se cancela la petición que la empezó
	// (otras esperan su resultado), pero sigue acotada en el tiempo.
	compartido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), tiempoMaximoCargaCompartidaBolsasRRHH)
	lectura.datos, lectura.err = cargar(compartido)
	cancelar()
	c.mu.Lock()
	if c.enCurso == lectura {
		c.enCurso = nil
	}
	c.mu.Unlock()
	close(lectura.lista)
	if err := ctx.Err(); err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	return lectura.datos, lectura.err
}
