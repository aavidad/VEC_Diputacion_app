package application

import (
	"errors"
	"sort"
)

var ErrConsultaGlobalRRHHInvalida = errors.New("bolsa: consulta global no valida")

// La unidad del cuadro es una participación, no una persona distinta.
// No incluye nombres ni documentos: la ficha de su bolsa conserva esa lectura.
type ParticipacionGlobalRRHH struct {
	BolsaRef         string  `json:"bolsa_ref"`
	Categoria        string  `json:"categoria"`
	ParticipacionRef string  `json:"participacion_ref"`
	OrdenActa        int     `json:"orden_acta"`
	Estado           string  `json:"estado_clave"`
	EstadoDesde      string  `json:"estado_desde"`
	DisponibleDesde  *string `json:"disponible_desde"`
}

type LlamamientoGlobalRRHH struct {
	BolsaRef        string `json:"bolsa_ref"`
	Categoria       string `json:"categoria"`
	LlamamientoRef  string `json:"llamamiento_ref"`
	Referencia      string `json:"referencia"`
	EmitidoEn       string `json:"emitido_en"`
	Participaciones int    `json:"participaciones"`
}

type ConjuntoGlobalRRHH struct {
	GeneradoEn                  string
	Participaciones             []ParticipacionGlobalRRHH
	Llamamientos                []LlamamientoGlobalRRHH
	ListaLlamamientosDisponible bool
}

type ConsultaGlobalRRHH struct {
	Filtro, BolsaRef string
	Desde, Limite    int
}

type PaginaGlobalRRHH struct {
	Total, Desde, Hasta int
	Items               any
}

func (c ConsultaGlobalRRHH) Validar() error {
	if c.Limite < 1 || c.Limite > 100 || c.Desde < 0 || (c.BolsaRef != "" && c.Filtro != "llamamientos") {
		return ErrConsultaGlobalRRHHInvalida
	}
	switch c.Filtro {
	case "todos", "disponible", "renuncia", "llamamientos":
		return nil
	}
	return ErrConsultaGlobalRRHHInvalida
}

// Paginar selecciona una única vez el mismo predicado que usa el contador.
// El conjunto es inmutable y se conserva durante toda la continuación.
func (c ConjuntoGlobalRRHH) Paginar(q ConsultaGlobalRRHH) (PaginaGlobalRRHH, error) {
	if err := q.Validar(); err != nil {
		return PaginaGlobalRRHH{}, err
	}
	if q.Filtro == "llamamientos" {
		if !c.ListaLlamamientosDisponible {
			return PaginaGlobalRRHH{}, ErrConsultaGlobalRRHHInvalida
		}
		filas := make([]LlamamientoGlobalRRHH, 0)
		for _, f := range c.Llamamientos {
			if q.BolsaRef == "" || f.BolsaRef == q.BolsaRef {
				filas = append(filas, f)
			}
		}
		return paginaGlobal(filas, q)
	}
	filas := make([]ParticipacionGlobalRRHH, 0)
	for _, f := range c.Participaciones {
		if q.Filtro == "todos" || f.Estado == q.Filtro {
			filas = append(filas, f)
		}
	}
	return paginaGlobal(filas, q)
}

func paginaGlobal[T any](filas []T, q ConsultaGlobalRRHH) (PaginaGlobalRRHH, error) {
	if q.Desde > len(filas) {
		return PaginaGlobalRRHH{}, ErrConsultaGlobalRRHHInvalida
	}
	hasta := min(q.Desde+q.Limite, len(filas))
	desde := 0
	if hasta > q.Desde {
		desde = q.Desde + 1
	}
	return PaginaGlobalRRHH{Total: len(filas), Desde: desde, Hasta: hasta, Items: filas[q.Desde:hasta]}, nil
}

// Ordenar fija el orden de navegación sin confundir posición de acta con
// el turno actual (que se consulta al abrir la bolsa).
func (c *ConjuntoGlobalRRHH) Ordenar() {
	sort.Slice(c.Participaciones, func(i, j int) bool {
		a, b := c.Participaciones[i], c.Participaciones[j]
		if a.Categoria != b.Categoria {
			return a.Categoria < b.Categoria
		}
		if a.BolsaRef != b.BolsaRef {
			return a.BolsaRef < b.BolsaRef
		}
		if a.OrdenActa != b.OrdenActa {
			return a.OrdenActa < b.OrdenActa
		}
		return a.ParticipacionRef < b.ParticipacionRef
	})
	sort.Slice(c.Llamamientos, func(i, j int) bool {
		a, b := c.Llamamientos[i], c.Llamamientos[j]
		if a.EmitidoEn != b.EmitidoEn {
			return a.EmitidoEn > b.EmitidoEn
		}
		return a.LlamamientoRef < b.LlamamientoRef
	})
}
