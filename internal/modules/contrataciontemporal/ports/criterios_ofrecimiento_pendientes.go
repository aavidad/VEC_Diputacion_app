package ports

import (
	"errors"
	"regexp"
	"strings"
)

// La selección se suspende hasta que RRHH compruebe los hechos contextuales
// de una persona situada antes del siguiente ofrecimiento.
var ErrCriteriosOfrecimientoPendientesRRHH = errors.New("contratacion temporal: comprobacion de ofrecimiento pendiente por RRHH")

type ComprobacionOfrecimientoPendiente struct {
	BolsaRef               string                         `json:"bolsa_ref"`
	OrdenActa              uint64                         `json:"orden_acta"`
	VersionPoliticaOfertas int64                          `json:"version_politica_ofertas,omitempty"`
	MotivoClave            string                         `json:"motivo_clave"`
	Contexto               *ContextoComprobacionPendiente `json:"contexto,omitempty"`
}

// Referencias y versiones del corte que RRHH puede enviar a Bolsa para
// registrar la comprobación. No incluye hechos ni datos de la persona.
type ContextoComprobacionPendiente struct {
	AmbitoRef                   string `json:"ambito_ref"`
	BolsaRef                    string `json:"bolsa_ref"`
	CategoriaRef                string `json:"categoria_ref"`
	CausaOfrecida               string `json:"causa_ofrecida"`
	ExpedienteRef               string `json:"expediente_ref"`
	HuellaPoliticaOfertasSHA256 string `json:"huella_politica_ofertas_sha256"`
	ModalidadOfrecida           string `json:"modalidad_ofrecida"`
	NecesidadRef                string `json:"necesidad_ref"`
	OrdenActa                   int64  `json:"orden_acta"`
	OrganizacionRef             string `json:"organizacion_ref"`
	ParticipacionRef            string `json:"participacion_ref"`
	UnidadRef                   string `json:"unidad_ref"`
	VersionBolsa                int64  `json:"version_bolsa"`
	VersionExpediente           int64  `json:"version_expediente"`
	VersionInstantanea          int64  `json:"version_instantanea"`
	VersionPoliticaOfertas      int64  `json:"version_politica_ofertas"`
}

var referenciaContextoComprobacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]+$`)
var huellaContextoComprobacion = regexp.MustCompile(`^[a-f0-9]{64}$`)
var documentoPersonalContextoComprobacion = regexp.MustCompile(`(?i)(([0-9][._:/#-]?){8}|[XYZ][._:/#-]?([0-9][._:/#-]?){7})[A-Z]`)

// Las referencias que emite el sistema (espacio de nombres y huella SHA-256)
// no pueden llevar un documento escrito por una persona, pero sus cifras
// hexadecimales casan por azar con el patrón de DNI; quedan fuera de ese
// filtro, igual que en Bolsa.
var referenciaPropiaSistemaContextoComprobacion = regexp.MustCompile(`^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$`)
var etiquetaPersonalContextoComprobacion = regexp.MustCompile(`(?i)(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)`)

const maximoEnteroContextoComprobacion = 9007199254740991

// ValidoPara impide publicar un contexto parcial o de otra posición en el 409.
func (c ContextoComprobacionPendiente) ValidoPara(p ComprobacionOfrecimientoPendiente) bool {
	for _, ref := range []string{c.AmbitoRef, c.BolsaRef, c.CategoriaRef, c.ExpedienteRef,
		c.NecesidadRef, c.OrganizacionRef, c.ParticipacionRef, c.UnidadRef} {
		if len(ref) > 256 || !referenciaContextoComprobacion.MatchString(ref) ||
			(!referenciaPropiaSistemaContextoComprobacion.MatchString(ref) && documentoPersonalContextoComprobacion.MatchString(ref)) ||
			etiquetaPersonalContextoComprobacion.MatchString(ref) {
			return false
		}
	}
	return len(c.ModalidadOfrecida) <= 256 && strings.TrimSpace(c.ModalidadOfrecida) != "" &&
		len(c.CausaOfrecida) <= 256 && huellaContextoComprobacion.MatchString(c.HuellaPoliticaOfertasSHA256) &&
		c.OrdenActa > 0 && c.VersionBolsa > 0 && c.VersionExpediente > 0 &&
		c.VersionInstantanea > 0 && c.VersionPoliticaOfertas > 0 &&
		c.OrdenActa <= maximoEnteroContextoComprobacion && c.VersionBolsa <= maximoEnteroContextoComprobacion &&
		c.VersionExpediente <= maximoEnteroContextoComprobacion && c.VersionInstantanea <= maximoEnteroContextoComprobacion &&
		c.VersionPoliticaOfertas <= maximoEnteroContextoComprobacion &&
		c.BolsaRef == p.BolsaRef && uint64(c.OrdenActa) == p.OrdenActa &&
		c.VersionPoliticaOfertas == p.VersionPoliticaOfertas && p.MotivoClave == "pendiente_comprobacion_rrhh"
}

type ErrorCriteriosOfrecimientoPendientesRRHH struct {
	Comprobacion ComprobacionOfrecimientoPendiente
}

func (e *ErrorCriteriosOfrecimientoPendientesRRHH) Error() string {
	return ErrCriteriosOfrecimientoPendientesRRHH.Error()
}

func (e *ErrorCriteriosOfrecimientoPendientesRRHH) Unwrap() error {
	return ErrCriteriosOfrecimientoPendientesRRHH
}
