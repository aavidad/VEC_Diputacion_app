package bootstrap

import (
	"context"
	"errors"
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
	// alEsperar es un gancho de prueba: se llama (con mu tomado) cuando una
	// petición se suma a la lectura en curso.
	alEsperar func()
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
		if c.alEsperar != nil {
			c.alEsperar()
		}
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
	c.ejecutar(ctx, lectura, cargar)
	if err := ctx.Err(); err != nil {
		return datasetBolsasRRHHDesarrollo{}, err
	}
	return lectura.datos, lectura.err
}

var errCargaCompartidaInterrumpida = errors.New("bolsa rrhh: lectura compartida interrumpida")

// ejecutar hace la lectura y, pase lo que pase (también un pánico), la
// retira de «en curso» y despierta a quienes esperan; si no, todas las
// peticiones siguientes del cuadro quedarían colgadas.
func (c *cargaCompartidaBolsasRRHH) ejecutar(ctx context.Context, lectura *lecturaCompartidaBolsasRRHH, cargar func(context.Context) (datasetBolsasRRHHDesarrollo, error)) {
	terminada := false
	defer func() {
		if !terminada {
			lectura.datos, lectura.err = datasetBolsasRRHHDesarrollo{}, errCargaCompartidaInterrumpida
		}
		c.mu.Lock()
		if c.enCurso == lectura {
			c.enCurso = nil
		}
		c.mu.Unlock()
		close(lectura.lista)
	}()
	// La lectura compartida no muere si se cancela la petición que la empezó
	// (otras esperan su resultado), pero sigue acotada en el tiempo.
	compartido, cancelar := context.WithTimeout(context.WithoutCancel(ctx), tiempoMaximoCargaCompartidaBolsasRRHH)
	defer cancelar()
	lectura.datos, lectura.err = cargar(compartido)
	terminada = true
}
