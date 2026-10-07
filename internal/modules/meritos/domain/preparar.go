package domain

import "errors"

const AlcanceSintetico = "preparacion_sintetica"

var ErrPreparacion = errors.New("meritos.error.preparacion_invalida")

type PersonaSintetica struct {
	Referencia string `json:"referencia"`
	Nombre     string `json:"nombre"`
}

type Paquete struct {
	Alcance    string             `json:"alcance"`
	Version    string             `json:"version"`
	FechaCorte string             `json:"fecha_corte"`
	Personas   []PersonaSintetica `json:"personas"`
	Hechos     []Hecho            `json:"hechos"`
}

type RevisionPreparacion struct {
	HechoRef        string   `json:"hecho_ref"`
	Version         int      `json:"version"`
	VigenciaEnCorte string   `json:"vigencia_en_corte"`
	Pendientes      []string `json:"pendientes"`
}

type Preparacion struct {
	Paquete          Paquete               `json:"paquete"`
	Estado           string                `json:"estado"`
	Persistido       bool                  `json:"persistido"`
	AcreditacionReal bool                  `json:"acreditacion_real"`
	Revisiones       []RevisionPreparacion `json:"revisiones"`
}

type origen struct{ persona, fuente, hecho, tipo string }
type identidad struct{ persona, fuente, hecho, tipo string }

// Preparar conserva las versiones aportadas. Toda correspondencia o revisión
// real queda pendiente; ejecutar esta utilidad no incorpora hechos al RUM.
func Preparar(p Paquete) (Preparacion, error) {
	if p.Alcance != AlcanceSintetico || !ReferenciaValida(p.Version) || !fechaValida(p.FechaCorte) ||
		len(p.Personas) == 0 || len(p.Personas) > 100 || len(p.Hechos) == 0 || len(p.Hechos) > 1000 {
		return Preparacion{}, ErrPreparacion
	}
	personas := make(map[string]bool)
	for _, persona := range p.Personas {
		if !ReferenciaValida(persona.Referencia) || !textoValido(persona.Nombre, 256) || personas[persona.Referencia] {
			return Preparacion{}, ErrPreparacion
		}
		personas[persona.Referencia] = true
	}
	versiones := make(map[string]int)
	identidades := make(map[string]identidad)
	origenes := make(map[origen]string)
	out := Preparacion{Paquete: copiar(p), Estado: "pendiente_comprobacion_fuentes", Revisiones: []RevisionPreparacion{}}
	for _, h := range p.Hechos {
		id := identidad{h.PersonaRef, h.Procedencia.FuenteRef, h.Procedencia.HechoOrigenRef, h.Tipo}
		ori := origen{id.persona, id.fuente, id.hecho, id.tipo}
		if h.Validar() != nil || !personas[h.PersonaRef] || h.Version != versiones[h.Referencia]+1 ||
			(versiones[h.Referencia] > 0 && identidades[h.Referencia] != id) ||
			(origenes[ori] != "" && origenes[ori] != h.Referencia) {
			return Preparacion{}, ErrPreparacion
		}
		versiones[h.Referencia], identidades[h.Referencia], origenes[ori] = h.Version, id, h.Referencia
		vigencia := "vigente"
		if p.FechaCorte < h.Vigencia.Desde {
			vigencia = "no_iniciada"
		} else if h.Vigencia.Hasta != "" && p.FechaCorte > h.Vigencia.Hasta {
			vigencia = "finalizada"
		}
		pendientes := []string{"meritos.pendiente.persona", "meritos.pendiente.procedencia", "meritos.pendiente.documentos", "meritos.pendiente.autoridad_revision"}
		if h.Estado == Declarado || h.Estado == Pendiente {
			pendientes = append(pendientes, "meritos.pendiente.resolver_estado")
		}
		out.Revisiones = append(out.Revisiones, RevisionPreparacion{h.Referencia, h.Version, vigencia, pendientes})
	}
	return out, nil
}

func copiar(p Paquete) Paquete {
	p.Personas = append([]PersonaSintetica{}, p.Personas...)
	p.Hechos = append([]Hecho{}, p.Hechos...)
	for i := range p.Hechos {
		h := &p.Hechos[i]
		h.Evidencias = append(h.Evidencias[:0:0], h.Evidencias...)
		if h.Horas != nil {
			valor := *h.Horas
			h.Horas = &valor
		}
		if h.Revision != nil {
			valor := *h.Revision
			h.Revision = &valor
		}
	}
	return p
}
