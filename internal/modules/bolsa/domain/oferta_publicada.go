package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// Ofertas publicadas (Petición RRHH p. 3; Reglamento de bolsas, art. 8.1):
// RRHH publica una oferta de una bolsa; quienes figuran en ella manifiestan
// disposición en el plazo de la regla b10; al vencer se adjudica a la mejor
// posición del orden vigente entre ellas o se pasa a llamamiento directo.

// Estados deducidos de una oferta. No se almacenan: salen del instante de la
// consulta, del vencimiento y de los actos confirmados sobre sus plazas.
const (
	EstadoOfertaAbierta               = "abierta"
	EstadoOfertaPendienteResolucion   = "pendiente_resolucion"
	EstadoOfertaEnCurso               = "en_curso"
	EstadoOfertaAdjudicada            = "adjudicada"
	EstadoOfertaLlamamientoDirecto    = "llamamiento_directo"
	EstadoOfertaCerrada               = "cerrada"
	PropuestaOfertaAdjudicar          = "adjudicar"
	PropuestaOfertaLlamamientoDirecto = "llamamiento_directo"
)

// Una oferta cubre de 1 a MaximoPlazasOferta plazas (duda 75). Cada plaza se
// resuelve con actos de solo adición; el último fija su estado.
const (
	MaximoPlazasOferta = 100

	ActoPlazaAdjudicada         = "adjudicada"
	ActoPlazaAceptada           = "aceptada"
	ActoPlazaRenuncia           = "renuncia"
	ActoPlazaSinRespuesta       = "sin_respuesta"
	ActoPlazaLlamamientoDirecto = "llamamiento_directo"

	EstadoPlazaVacante            = "vacante"
	EstadoPlazaPendienteRespuesta = "pendiente_respuesta"
	EstadoPlazaCubierta           = "cubierta"
	EstadoPlazaLlamamientoDirecto = "llamamiento_directo"
)

// NumeroPlazasValido acota el número de plazas de una oferta.
func NumeroPlazasValido(n int) bool { return n >= 1 && n <= MaximoPlazasOferta }

// ActoPlazaValido exige persona en todos los actos salvo el paso a
// llamamiento directo, que no la lleva.
func ActoPlazaValido(tipo string, conPersona bool) bool {
	switch tipo {
	case ActoPlazaAdjudicada, ActoPlazaAceptada, ActoPlazaRenuncia, ActoPlazaSinRespuesta:
		return conPersona
	case ActoPlazaLlamamientoDirecto:
		return !conPersona
	}
	return false
}

const (
	minimoTextoOferta = 2
	maximoTextoOferta = 2000
	formatoFechaDia   = "2006-01-02"
)

var ErrDatosOfertaInvalidos = errors.New("bolsa: datos de oferta invalidos")

// DatosOferta es lo que RRHH publica y ven las personas de la bolsa. Las
// fechas son días civiles (AAAA-MM-DD); la de fin es opcional.
type DatosOferta struct {
	Categoria   string `json:"categoria"`
	Centro      string `json:"centro"`
	FechaInicio string `json:"fecha_inicio"`
	FechaFin    string `json:"fecha_fin,omitempty"`
	Descripcion string `json:"descripcion"`
}

// Validar exige textos recortados y acotados y fechas coherentes.
func (d DatosOferta) Validar() error {
	for _, texto := range []string{d.Categoria, d.Centro, d.FechaInicio, d.Descripcion} {
		if !textoOfertaValido(texto) {
			return ErrDatosOfertaInvalidos
		}
	}
	inicio, err := time.Parse(formatoFechaDia, d.FechaInicio)
	if err != nil {
		return ErrDatosOfertaInvalidos
	}
	if d.FechaFin == "" {
		return nil
	}
	fin, err := time.Parse(formatoFechaDia, d.FechaFin)
	if err != nil || fin.Before(inicio) {
		return ErrDatosOfertaInvalidos
	}
	return nil
}

func textoOfertaValido(texto string) bool {
	return strings.TrimSpace(texto) == texto && len(texto) >= minimoTextoOferta && len(texto) <= maximoTextoOferta
}

// NotificacionOferta conserva la declaración de RRHH sobre el correo externo
// con el extracto de la oferta. No acredita entrega a cada destinatario.
type NotificacionOferta struct {
	NotificadaEn       time.Time `json:"notificada_en"`
	ReferenciaCorreo   string    `json:"referencia_correo"`
	HuellaCorreoSHA256 string    `json:"huella_correo_sha256"`
	Fuente             string    `json:"fuente"`
}

var referenciaCorreoOferta = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}$`)
var huellaCorreoOferta = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ValidarPara exige un hecho pasado o presente y una referencia sin contenido
// personal. La autoridad de la declaración es RRHH, revalidada al publicar.
func (n NotificacionOferta) ValidarPara(ahora time.Time) error {
	_, offset := n.NotificadaEn.Zone()
	if ahora.IsZero() || n.NotificadaEn.IsZero() || offset != 0 ||
		n.NotificadaEn.Nanosecond()%1000 != 0 || n.NotificadaEn.After(ahora) ||
		!referenciaCorreoOferta.MatchString(n.ReferenciaCorreo) || !referenciaLlamamientoOpacaValida(n.ReferenciaCorreo) ||
		!huellaCorreoOferta.MatchString(n.HuellaCorreoSHA256) ||
		n.Fuente != "correo_externo_declarado_rrhh" {
		return ErrDatosOfertaInvalidos
	}
	return nil
}
