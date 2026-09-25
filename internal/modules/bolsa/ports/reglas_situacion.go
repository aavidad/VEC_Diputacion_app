package ports

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrReglasSituacionNoConfiguradas: no hay catálogo de reglas compuesto.
	// Bolsa conserva su conducta sin reglas (tabla compilada, fecha a mano y
	// motivo libre).
	ErrReglasSituacionNoConfiguradas = errors.New("bolsa: reglas de situacion no configuradas")
	// ErrReglasSituacionNoDisponibles: hay catálogo pero no puede leerse o
	// una regla no cumple su contrato. Nunca se sustituye por un valor supuesto.
	ErrReglasSituacionNoDisponibles = errors.New("bolsa: reglas de situacion no disponibles")
	// ErrReposicionNoCalculable: los datos de la propuesta no son válidos.
	ErrReposicionNoCalculable = errors.New("bolsa: propuesta de reposicion no calculable")
)

// ProcedenciaRegla identifica la entrada del catálogo que respalda un valor.
// Solo se muestra a RRHH.
type ProcedenciaRegla struct {
	Clave      string
	Referencia string
	Articulo   string
	Norma      string
	Ejemplo    bool
}

// CausaBajaSituacion es una causa de baja definitiva ofrecida por el catálogo.
type CausaBajaSituacion struct {
	Codigo      string
	Etiqueta    string
	Procedencia ProcedenciaRegla
}

// ModalidadReposicion es una modalidad de nombramiento con un periodo de no
// disponibilidad distinto del general.
type ModalidadReposicion struct {
	Codigo string
	Meses  int
}

// PropuestaReposicion es la fecha de disponibilidad que el catálogo propone
// al terminar una relación. RRHH la confirma o la cambia.
type PropuestaReposicion struct {
	// FechaDisponible es el primer instante en que vuelve a estar disponible.
	FechaDisponible time.Time
	// UltimoDiaNoDisponible es la fecha civil en que termina el periodo.
	UltimoDiaNoDisponible string
	Meses                 int
	Procedencia           ProcedenciaRegla
}

// ReglasTransicionesSituacion restringe las transiciones de situación con el
// catálogo versionado. configurada=false significa que el catálogo no dice
// nada para ese origen y rige la tabla compilada del dominio.
type ReglasTransicionesSituacion interface {
	DestinosSituacion(ctx context.Context, origen string) (destinos []string, configurada bool, err error)
}

// ConsultaReglasSituacion reúne lo que la pantalla de RRHH necesita del
// catálogo para cambiar la situación de una participación.
type ConsultaReglasSituacion interface {
	ReglasTransicionesSituacion
	Configurada() bool
	CausasBaja(ctx context.Context) ([]CausaBajaSituacion, error)
	ModalidadesReposicion(ctx context.Context) ([]ModalidadReposicion, error)
	ProponerReposicion(ctx context.Context, finRelacion time.Time, modalidad string) (PropuestaReposicion, error)
}
