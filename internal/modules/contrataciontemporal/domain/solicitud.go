package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// DeclaracionRC refleja lo aportado por el centro. La validación autoritativa
// corresponde al análisis de RRHH y se conserva como un hecho distinto.
type DeclaracionRC struct {
	Existe       bool      `json:"existe"`
	Numero       string    `json:"numero,omitempty"`
	Fecha        time.Time `json:"fecha,omitempty"`
	Importe      Importe   `json:"importe,omitempty"`
	DocumentoRef string    `json:"documento_ref,omitempty"`
}

func (d DeclaracionRC) Validar() error {
	if !d.Existe {
		if d.Numero != "" || !d.Fecha.IsZero() || d.Importe != (Importe{}) ||
			d.DocumentoRef != "" {
			return ErrDatoInvalido
		}
		return nil
	}
	if !referenciaValida(d.Numero) || !fechaCivilCanonica(d.Fecha) ||
		d.Importe.Validar(false) != nil || !referenciaValida(d.DocumentoRef) {
		return ErrDatoInvalido
	}
	return nil
}

// PrefijoRCDeclarada marca la RC que declara el centro con el número de
// operación de SICAL. Es un dato que comprueban RRHH o Intervención: no es un
// documento custodiado ni acredita el crédito (regla provisional, duda 125).
const PrefijoRCDeclarada = "rc-declarado:"

var (
	patronDocumentoIdentidadRC = regexp.MustCompile(`(?i)(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]`)
	patronEtiquetaIdentidadRC  = regexp.MustCompile(`(?i)(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)`)
)

// ValidarDeclaradaPorCentro aplica a una petición nueva del centro: si hay
// RC, su referencia es la declarada con el mismo número de operación, que no
// puede tener forma de DNI o NIE. No se aplica al rehidratar lo ya guardado.
func (d DeclaracionRC) ValidarDeclaradaPorCentro() error {
	if err := d.Validar(); err != nil {
		return err
	}
	if !d.Existe {
		return nil
	}
	if d.DocumentoRef != PrefijoRCDeclarada+d.Numero ||
		patronDocumentoIdentidadRC.MatchString(d.Numero) ||
		patronEtiquetaIdentidadRC.MatchString(d.Numero) {
		return ErrDatoInvalido
	}
	return nil
}

type SolicitudCentro struct {
	CentroRef          string          `json:"centro_ref"`
	ContactoRef        string          `json:"contacto_ref"`
	CategoriaRef       string          `json:"categoria_ref"`
	GrupoSubgrupo      string          `json:"grupo_subgrupo"`
	MotivoClave        ClaveCatalogo   `json:"motivo_clave"`
	Detalle            string          `json:"detalle"`
	Periodo            PeriodoPrevisto `json:"periodo"`
	RC                 DeclaracionRC   `json:"rc"`
	DocumentosAdjuntos []string        `json:"documentos_adjuntos"`
	Observaciones      string          `json:"observaciones,omitempty"`
	// Estos datos proceden de la petición del centro. Su ausencia permite
	// recuperar las solicitudes anteriores sin inventar una jornada o puesto.
	JornadaMinutos   uint16 `json:"jornada_minutos,omitempty"`
	NumeroPersonas   uint32 `json:"numero_personas,omitempty"`
	PuestoSolicitado string `json:"puesto_solicitado,omitempty"`
	// Necesidad se fija por el servidor para el alta versionada. Su ausencia
	// conserva el contrato y los bytes históricos de SolicitudCentro.
	Necesidad *DatosNecesidadAlta `json:"necesidad,omitempty"`
}

func (s SolicitudCentro) Validar() error {
	if !referenciaValida(s.CentroRef) || !referenciaValida(s.ContactoRef) ||
		!referenciaValida(s.CategoriaRef) || !grupoValido(s.GrupoSubgrupo) ||
		!s.MotivoClave.Valida() || !textoValido(s.Detalle, 4000, false) ||
		s.Periodo.Validar() != nil || s.RC.Validar() != nil ||
		!referenciasUnicasValidas(s.DocumentosAdjuntos, 64) ||
		!textoValido(s.Observaciones, 4000, true) {
		return ErrDatoInvalido
	}
	conDatosPuesto := s.JornadaMinutos != 0 || s.NumeroPersonas != 0 || s.PuestoSolicitado != ""
	soloCantidadSellada := s.Necesidad != nil && s.NumeroPersonas != 0 &&
		s.JornadaMinutos == 0 && s.PuestoSolicitado == ""
	if soloCantidadSellada {
		cantidad, err := strconv.ParseUint(s.Necesidad.Campos["numero_personas"], 10, 32)
		if err != nil || cantidad == 0 || uint32(cantidad) != s.NumeroPersonas ||
			strconv.FormatUint(cantidad, 10) != s.Necesidad.Campos["numero_personas"] ||
			s.Necesidad.ValidarInstantanea() != nil {
			return ErrDatoInvalido
		}
	} else if conDatosPuesto && (s.JornadaMinutos == 0 || s.JornadaMinutos > 7*24*60 ||
		s.NumeroPersonas == 0 || !textoValido(s.PuestoSolicitado, 160, false) ||
		strings.IndexFunc(s.PuestoSolicitado, unicode.IsControl) >= 0) {
		return ErrDatoInvalido
	}
	if s.Necesidad != nil && (s.Necesidad.ValidarInstantanea() != nil ||
		s.MotivoClave != s.Necesidad.CausaClave || s.Periodo != s.Necesidad.Periodo) {
		return ErrDatoInvalido
	}
	return nil
}

func (s SolicitudCentro) clonar() SolicitudCentro {
	s.DocumentosAdjuntos = append([]string(nil), s.DocumentosAdjuntos...)
	if s.Necesidad != nil {
		copia := s.Necesidad.clonar()
		s.Necesidad = &copia
	}
	return s
}

// Clonar entrega una copia defensiva apta para cruzar los puertos del módulo.
// La copia nunca comparte la lista mutable de documentos con quien la aportó.
func (s SolicitudCentro) Clonar() (SolicitudCentro, error) {
	if s.Validar() != nil {
		return SolicitudCentro{}, ErrDatoInvalido
	}
	return s.clonar(), nil
}
