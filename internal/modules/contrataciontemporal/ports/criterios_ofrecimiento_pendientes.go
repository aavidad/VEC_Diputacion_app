package ports

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
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
	Estado                 *EstadoComprobacionPendiente   `json:"estado,omitempty"`
}

// La ausencia actual es version_esperada=0. nil conserva el significado de
// una lectura histórica que todavía no proyectaba el estado del acto.
type EstadoComprobacionPendiente struct {
	VersionEsperada        uint64    `json:"version_esperada"`
	ComprobacionRef        string    `json:"comprobacion_ref"`
	ReciboRef              string    `json:"recibo_ref"`
	RegistradaEn           time.Time `json:"registrada_en"`
	VersionPoliticaOfertas uint64    `json:"version_politica_ofertas"`
	Resultado              string    `json:"resultado"`
	MotivoClave            string    `json:"motivo_clave"`
	Vigente                *bool     `json:"vigente"`
}

var referenciaActoComprobacion = regexp.MustCompile(`^comprobacion:[0-9a-f]{64}$`)

func (e EstadoComprobacionPendiente) ValidoPara(p ComprobacionOfrecimientoPendiente) bool {
	if p.Contexto == nil || !p.Contexto.ValidoPara(p) {
		return false
	}
	if e.VersionEsperada == 0 {
		return e.ComprobacionRef == "" && e.ReciboRef == "" && e.RegistradaEn.IsZero() &&
			e.VersionPoliticaOfertas == 0 && e.Resultado == "" && e.MotivoClave == "" && e.Vigente == nil
	}
	if e.VersionEsperada > 999999999 || !referenciaActoComprobacion.MatchString(e.ComprobacionRef) ||
		e.ReciboRef != "recibo:"+e.ComprobacionRef || e.VersionPoliticaOfertas == 0 ||
		e.VersionPoliticaOfertas > maximoEnteroContextoComprobacion || e.Vigente == nil ||
		e.RegistradaEn.IsZero() || e.RegistradaEn.Year() < 1 || e.RegistradaEn.Year() > 9999 ||
		e.RegistradaEn.Nanosecond()%1000 != 0 {
		return false
	}
	_, desfase := e.RegistradaEn.Zone()
	if desfase != 0 || (*e.Vigente && int64(e.VersionPoliticaOfertas) != p.Contexto.VersionPoliticaOfertas) {
		return false
	}
	return (e.Resultado == "no_aplicable" && e.MotivoClave == "sin_supuesto_articulo_9") ||
		(e.Resultado == "pendiente" && (e.MotivoClave == "antecedentes_incompletos" ||
			e.MotivoClave == "criterios_invalidos" || e.MotivoClave == "guarda_contratacion_pendiente" ||
			e.MotivoClave == "descuento_pendiente_criterio_rrhh"))
}

// time.Time con omitempty publica año 0001. La ausencia acreditada tiene
// por contrato exactamente una propiedad, sin fecha ni resultado fabricados.
func (e EstadoComprobacionPendiente) MarshalJSON() ([]byte, error) {
	if e.VersionEsperada == 0 {
		if e.ComprobacionRef != "" || e.ReciboRef != "" || !e.RegistradaEn.IsZero() ||
			e.VersionPoliticaOfertas != 0 || e.Resultado != "" || e.MotivoClave != "" || e.Vigente != nil {
			return nil, ErrRespuestaBolsaNoConfiable
		}
		return []byte(`{"version_esperada":0}`), nil
	}
	type sinMetodo EstadoComprobacionPendiente
	return json.Marshal(sinMetodo(e))
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
