package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const AccionDerivarNombramiento ClaveCatalogo = "contratacion_temporal.nombramiento.derivar"

// InstruccionCompartida conserva la instrucción original con sus actuaciones y
// recibos. Las actuaciones del hijo comienzan en la secuencia 6.
type InstruccionCompartida struct {
	ExpedientePadre     Expediente `json:"expediente_padre"`
	OfertaRef           string     `json:"oferta_ref"`
	NumeroPlaza         uint32     `json:"numero_plaza"`
	ReciboDerivacionRef string     `json:"recibo_derivacion_ref"`
	HuellaPadreSHA256   string     `json:"huella_padre_sha256"`
	PreimagenPadreJSON  string     `json:"preimagen_padre_json"`
}

type AltaExpedienteDerivado struct {
	Padre              Expediente
	Referencia         string
	OfertaRef          string
	NumeroPlaza        uint32
	ActorRef           string
	ReciboRef          string
	CreadaEn           time.Time
	PreimagenPadreJSON string
}

func NuevoExpedienteDerivado(alta AltaExpedienteDerivado) (Expediente, error) {
	// Esta guarda precede a Validar y Clonar: un padre cíclico no debe recorrerse.
	if alta.Padre.InstruccionCompartida != nil || alta.Padre.Validar() != nil ||
		!padreDerivable(alta.Padre) || len(alta.PreimagenPadreJSON) < 2 || len(alta.PreimagenPadreJSON) > 262144 {
		return Expediente{}, ErrExpedienteInvalido
	}
	actuacion := DatosActuacion{
		AccionClave: AccionDerivarNombramiento, ActorRef: alta.ActorRef,
		UnidadRef: alta.Padre.Asignacion.UnidadRef, ReciboRef: alta.ReciboRef,
		RealizadaEn: alta.CreadaEn, FaseDestino: FaseFiscalizacion,
		EstadoDestino: EstadoEnCurso,
	}
	padre := alta.Padre.Clonar()
	huella := huellaPreimagenPadre(alta.PreimagenPadreJSON)
	hijo := padre.Clonar()
	hijo.Referencia, hijo.NumeroVisible, hijo.Version = alta.Referencia,
		padre.NumeroVisible+"-P"+strconv.FormatUint(uint64(alta.NumeroPlaza), 10), 6
	hijo.Solicitud.NumeroPersonas = 1
	hijo.CreadoEn, hijo.ActualizadoEn = alta.CreadaEn, alta.CreadaEn
	hijo.FaseActual, hijo.EstadoActual = FaseFiscalizacion, EstadoEnCurso
	hijo.InstruccionCompartida = &InstruccionCompartida{
		ExpedientePadre: padre, OfertaRef: alta.OfertaRef, NumeroPlaza: alta.NumeroPlaza,
		ReciboDerivacionRef: alta.ReciboRef, HuellaPadreSHA256: huella,
		PreimagenPadreJSON: alta.PreimagenPadreJSON,
	}
	hijo.Actuaciones = []Actuacion{hijo.nuevaActuacion(
		padre.FaseActual, padre.EstadoActual, actuacion, 6,
	)}
	if err := hijo.Validar(); err != nil {
		return Expediente{}, err
	}
	return hijo, nil
}

func padreDerivable(p Expediente) bool {
	return p.InstruccionCompartida == nil && p.Version >= 6 &&
		p.Solicitud.NumeroPersonas >= 2 && p.Asignacion != nil &&
		p.Fiscalizacion != nil &&
		(p.Fiscalizacion.Resultado == FiscalizacionFavorable ||
			p.Fiscalizacion.Resultado == FiscalizacionFavorableConObservaciones) &&
		p.FaseActual == FaseFiscalizacion && p.EstadoActual == EstadoEnCurso
}

func numeroDerivadoValido(numero, numeroPadre string, plaza uint32) bool {
	return plaza != 0 &&
		numero == numeroPadre+"-P"+strconv.FormatUint(uint64(plaza), 10)
}

func huellaPreimagenPadre(raw string) string {
	huella := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(huella[:])
}

func validarPreimagenPadre(raw string, padre Expediente) error {
	if len(raw) < 2 || len(raw) > 262144 || !utf8.ValidString(raw) ||
		padre.InstruccionCompartida != nil {
		return ErrExpedienteInvalido
	}
	decodificador := json.NewDecoder(strings.NewReader(raw))
	decodificador.DisallowUnknownFields()
	var desdeRaw Expediente
	if err := decodificador.Decode(&desdeRaw); err != nil {
		return err
	}
	var resto any
	if err := decodificador.Decode(&resto); err != io.EOF {
		if err != nil {
			return err
		}
		return ErrExpedienteInvalido
	}
	if desdeRaw.InstruccionCompartida != nil {
		return ErrExpedienteInvalido
	}

	if !reflect.DeepEqual(desdeRaw.Clonar(), padre.Clonar()) {
		return ErrExpedienteInvalido
	}
	return nil
}

func (e Expediente) validarInstruccionCompartida() error {
	i := e.InstruccionCompartida
	if i == nil || i.ExpedientePadre.InstruccionCompartida != nil {
		return ErrExpedienteInvalido
	}
	p := i.ExpedientePadre
	if err := validarPreimagenPadre(i.PreimagenPadreJSON, p); err != nil {
		return errors.Join(ErrExpedienteInvalido, err)
	}
	huella := huellaPreimagenPadre(i.PreimagenPadreJSON)
	if p.Validar() != nil || !padreDerivable(p) ||
		!referenciaValida(i.OfertaRef) || i.NumeroPlaza == 0 ||
		i.NumeroPlaza > p.Solicitud.NumeroPersonas ||
		!referenciaValida(i.ReciboDerivacionRef) || i.HuellaPadreSHA256 != huella ||
		e.Referencia == p.Referencia ||
		!numeroDerivadoValido(e.NumeroVisible, p.NumeroVisible, i.NumeroPlaza) ||
		e.OrganizacionRef != p.OrganizacionRef || e.Flujo != p.Flujo {
		return ErrExpedienteInvalido
	}
	solicitud := p.Solicitud.clonar()
	solicitud.NumeroPersonas = 1
	if !reflect.DeepEqual(e.Solicitud, solicitud) ||
		!reflect.DeepEqual(e.Analisis, p.Analisis) ||
		!reflect.DeepEqual(e.ViaCobertura, p.ViaCobertura) ||
		!reflect.DeepEqual(e.DecisionesCobertura, p.DecisionesCobertura) ||
		!reflect.DeepEqual(e.Asignacion, p.Asignacion) ||
		!reflect.DeepEqual(e.InformeJuridico, p.InformeJuridico) ||
		!reflect.DeepEqual(e.Fiscalizacion, p.Fiscalizacion) ||
		!reflect.DeepEqual(e.Circuito, p.Circuito) {
		return ErrExpedienteInvalido
	}
	primera := e.Actuaciones[0]
	if primera.AccionClave != AccionDerivarNombramiento ||
		primera.FaseOrigen != p.FaseActual || primera.EstadoOrigen != p.EstadoActual ||
		primera.FaseDestino != FaseFiscalizacion || primera.EstadoDestino != EstadoEnCurso ||
		primera.UnidadRef != p.Asignacion.UnidadRef ||
		primera.ReciboRef != i.ReciboDerivacionRef ||
		primera.RealizadaEn.Before(p.ActualizadoEn) ||
		len(primera.DocumentosRef) != 0 || primera.RetornoRef != "" {
		return ErrExpedienteInvalido
	}
	for _, a := range p.Actuaciones {
		if a.ReciboRef == primera.ReciboRef {
			return ErrExpedienteInvalido
		}
	}
	return nil
}
