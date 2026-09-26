package ports

import "context"

// FuenteListaDefinitiva identifica la autoridad que fijó el orden. Bolsa no
// vuelve a ordenar ni a desempatar las posiciones aprobadas.
type FuenteListaDefinitiva string

const (
	FuenteImportacionConvoca FuenteListaDefinitiva = "importacion_convoca"
	FuenteSeleccionNativa    FuenteListaDefinitiva = "seleccion_nativa"
)

// ConsultaListaDefinitiva usa una referencia opaca propia de la fuente. En
// CONVOCA es la huella del fichero importado; en Selección será la referencia
// de la lista publicada.
type ConsultaListaDefinitiva struct {
	Fuente       FuenteListaDefinitiva
	Referencia   string
	CategoriaRef string
}

// PosicionListaDefinitiva no contiene identidad civil. SujetoRef y
// ParticipacionRef son referencias opacas históricas de CONVOCA. Bolsa deriva
// las de Selección a partir de PersonaRef. CandidatoRef enlaza Mi bolsa cuando
// la fuente dispone de una referencia can_*.
type PosicionListaDefinitiva struct {
	Posicion         uint64
	PersonaRef       string
	CandidatoRef     string
	SujetoRef        string
	ParticipacionRef string
	Puntuacion       string
	FilaOrigenNumero int
}

// ListaDefinitivaAutorizada transporta la versión y el orden aprobado. La
// evidencia de CONVOCA es su acta confirmada y la huella del fichero; no se
// interpreta como firma jurídica ni como publicación nativa de Selección.
type ListaDefinitivaAutorizada struct {
	Fuente                     FuenteListaDefinitiva
	Referencia                 string
	Version                    uint64
	ConvocatoriaRef            string
	CategoriaRef               string
	BolsaRef                   string
	HuellaListadoSHA256        string
	AutorizacionPublicacionRef string
	HuellaAutorizacionSHA256   string
	Posiciones                 []PosicionListaDefinitiva
}

// RecuperadorListaDefinitiva es el puerto de entrada de Bolsa. Selección
// podrá implementarlo sin que Bolsa consulte sus tablas.
type RecuperadorListaDefinitiva interface {
	RecuperarListaDefinitiva(context.Context, ConsultaListaDefinitiva) (ListaDefinitivaAutorizada, bool, error)
}
