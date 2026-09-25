package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Petición RRHH p. 2 y duda 62: histórico de sanciones de una participación.
// Una sanción es la resolución que impone una consecuencia del catálogo de
// reglas (baja definitiva, pasar al final, suspensión...). No es una situación
// nueva: si la consecuencia cambia la situación lo hace mediante una operación
// B8 existente, de modo que el histórico de situaciones sigue siendo uno solo.

var ErrSancionParticipacionInvalida = errors.New("bolsa: sancion de participacion invalida")

// EfectoSancionNinguno marca una consecuencia que no cambia la situación
// (por ejemplo, pasar al final). Los demás efectos son operaciones B8.
const EfectoSancionNinguno = "ninguna"

var (
	patronClaveCatalogoSancion = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	patronEstadoRecurso        = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

// EfectoSancionValido admite «ninguna» y las operaciones B8 que imponen una
// consecuencia. Reactivar no es una sanción.
func EfectoSancionValido(efecto string) bool {
	return efecto == EfectoSancionNinguno || efecto == OperacionPausar || efecto == OperacionExcluir
}

// ClaveConsecuenciaSancionValida valida la clave de catálogo de una
// consecuencia o de un estado del recurso sin fijar sus valores.
func ClaveConsecuenciaSancionValida(clave string) bool {
	return patronClaveCatalogoSancion.MatchString(clave)
}

// EstadoRecursoValido valida la forma del estado; los valores admitidos los
// fija el catálogo de reglas.
func EstadoRecursoValido(estado string) bool { return patronEstadoRecurso.MatchString(estado) }

// DocumentoSancion identifica la resolución o el escrito del recurso en su
// custodia: VEC conserva la referencia y la huella, nunca el contenido.
type DocumentoSancion struct {
	Referencia string
	SHA256     string
}

func (d DocumentoSancion) Validar() error {
	if !referenciaLlamamientoOpacaValida(d.Referencia) || len(d.Referencia) > 256 || !patronHuellaJustificante.MatchString(d.SHA256) {
		return ErrSancionParticipacionInvalida
	}
	return nil
}

// FechaCivilSancion interpreta «AAAA-MM-DD» como fecha civil.
func FechaCivilSancion(valor string) (time.Time, bool) {
	fecha, err := parsearFechaCivilSancion(valor)
	return fecha, err == nil
}

func parsearFechaCivilSancion(valor string) (time.Time, error) {
	fecha, err := time.Parse(time.DateOnly, valor)
	if err != nil {
		return time.Time{}, errors.Join(ErrSancionParticipacionInvalida, err)
	}
	if fecha.Format(time.DateOnly) != valor {
		return time.Time{}, ErrSancionParticipacionInvalida
	}
	return fecha, nil
}

// DatosSancion son los hechos que declara RRHH al registrar la sanción.
type DatosSancion struct {
	Consecuencia      string
	Causa             string
	FechaNotificacion string
	Resolucion        DocumentoSancion
	ResueltaPor       string
}

// Validar comprueba los datos frente al instante de registro: la
// notificación no puede ser posterior al día del registro.
func (d DatosSancion) Validar(ahora time.Time) error {
	fecha, ok := FechaCivilSancion(d.FechaNotificacion)
	if !ClaveConsecuenciaSancionValida(d.Consecuencia) || !textoSancionValido(d.Causa, 1000) || !ok ||
		fecha.After(ahora.UTC().AddDate(0, 0, 1)) || fecha.Year() < 2000 || d.Resolucion.Validar() != nil ||
		!identidadOperacionValida(d.ResueltaPor) {
		return ErrSancionParticipacionInvalida
	}
	return nil
}

// EventoRecursoSancion anota un estado del recurso de reposición. Los
// eventos solo se añaden: el estado vigente es el último.
type EventoRecursoSancion struct {
	Estado       string
	Fecha        string
	Documento    *DocumentoSancion
	Actor        string
	RegistradaEn time.Time
}

// ValidarDatos comprueba lo que aporta RRHH, antes de conocer el actor.
func (e EventoRecursoSancion) ValidarDatos(ahora time.Time) error {
	fecha, ok := FechaCivilSancion(e.Fecha)
	if !EstadoRecursoValido(e.Estado) || !ok || fecha.After(ahora.UTC().AddDate(0, 0, 1)) || fecha.Year() < 2000 {
		return ErrSancionParticipacionInvalida
	}
	if e.Documento != nil && e.Documento.Validar() != nil {
		return ErrSancionParticipacionInvalida
	}
	return nil
}

// SancionParticipacion es una entrada del histórico de sanciones.
type SancionParticipacion struct {
	SancionRef           string
	ParticipacionRef     string
	Consecuencia         string
	ConsecuenciaEtiqueta string
	Efecto               string
	Datos                DatosSancion
	// ReglaRef y RecursoReglaRef son catalogo:version:entrada; sus huellas
	// fijan el catálogo exacto que se aplicó.
	ReglaRef           string
	ReglaHuella        string
	SuspensionHasta    string
	RecursoVence       string
	RecursoReglaRef    string
	RecursoReglaHuella string
	// SituacionDesde enlaza con el cambio de situación que produjo; nulo
	// cuando el efecto es «ninguna».
	SituacionDesde *time.Time
	Actor          string
	RegistradaEn   time.Time
	Recursos       []EventoRecursoSancion
	// Efecto aplicado: la situación que dejó la sanción (con su fecha de
	// vuelta al turno si la suspensión termina sola) y si la colocó al final
	// del orden vigente.
	SituacionAplicada string
	FechaDisponible   *time.Time
	OrdenFinal        bool
	// Reversion es la readmisión por un recurso revocatorio, si la hubo.
	Reversion *ReversionSancion
}

// ReversionSancion deja sin efecto una sanción por un recurso estimado. La
// situación restaurada es vacía si no hubo que cambiarla (por ejemplo, al
// levantar solo la penalización del orden).
type ReversionSancion struct {
	EstadoRecurso       string
	ReglaRef            string
	EfectoRevertido     string
	SituacionRestaurada string
	SituacionDesde      *time.Time
	ResueltaPor         string
	Actor               string
	ReciboRef           string
	RegistradaEn        time.Time
}

// EstadoRecurso devuelve el último estado anotado o «» si no hay recurso.
func (s SancionParticipacion) EstadoRecurso() string {
	if len(s.Recursos) == 0 {
		return ""
	}
	return s.Recursos[len(s.Recursos)-1].Estado
}

func textoSancionValido(valor string, maximo int) bool {
	return strings.TrimSpace(valor) == valor && valor != "" && len(valor) <= maximo
}

// IdentidadResolucionValida valida la persona que resuelve una sanción o su
// reversión, con las mismas reglas que el validador de una operación B8.
func IdentidadResolucionValida(valor string) bool { return identidadOperacionValida(valor) }
