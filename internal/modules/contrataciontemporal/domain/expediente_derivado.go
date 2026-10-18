package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
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
}

type AltaExpedienteDerivado struct {
	Padre         Expediente
	Referencia    string
	NumeroVisible string
	OfertaRef     string
	NumeroPlaza   uint32
	Actuacion     DatosActuacion
}

func NuevoExpedienteDerivado(alta AltaExpedienteDerivado) (Expediente, error) {
	// Esta guarda precede a Validar y Clonar: un padre cíclico no debe recorrerse.
	if alta.Padre.InstruccionCompartida != nil ||
		alta.Padre.Validar() != nil || !padreDerivable(alta.Padre) ||
		!referenciaValida(alta.Referencia) || alta.Referencia == alta.Padre.Referencia ||
		!numeroDerivadoValido(alta.NumeroVisible, alta.Padre.NumeroVisible, alta.NumeroPlaza) ||
		!referenciaValida(alta.OfertaRef) || alta.NumeroPlaza == 0 ||
		alta.NumeroPlaza > alta.Padre.Solicitud.NumeroPersonas ||
		alta.Actuacion.validar() != nil ||
		alta.Actuacion.AccionClave != AccionDerivarNombramiento ||
		alta.Actuacion.FaseDestino != FaseFiscalizacion ||
		alta.Actuacion.EstadoDestino != EstadoEnCurso ||
		alta.Actuacion.UnidadRef != alta.Padre.Asignacion.UnidadRef ||
		alta.Actuacion.RealizadaEn.Before(alta.Padre.ActualizadoEn) ||
		len(alta.Actuacion.DocumentosRef) != 0 || alta.Actuacion.RetornoRef != "" {
		return Expediente{}, ErrExpedienteInvalido
	}
	for _, a := range alta.Padre.Actuaciones {
		if a.ReciboRef == alta.Actuacion.ReciboRef {
			return Expediente{}, ErrExpedienteInvalido
		}
	}
	padre := alta.Padre.Clonar()
	huella, err := huellaExpedientePadre(padre)
	if err != nil {
		return Expediente{}, ErrExpedienteInvalido
	}
	hijo := padre.Clonar()
	hijo.Referencia, hijo.NumeroVisible, hijo.Version = alta.Referencia, alta.NumeroVisible, 6
	hijo.Solicitud.NumeroPersonas = 1
	hijo.CreadoEn, hijo.ActualizadoEn = alta.Actuacion.RealizadaEn, alta.Actuacion.RealizadaEn
	hijo.FaseActual, hijo.EstadoActual = FaseFiscalizacion, EstadoEnCurso
	hijo.InstruccionCompartida = &InstruccionCompartida{
		ExpedientePadre: padre, OfertaRef: alta.OfertaRef, NumeroPlaza: alta.NumeroPlaza,
		ReciboDerivacionRef: alta.Actuacion.ReciboRef, HuellaPadreSHA256: huella,
	}
	hijo.Actuaciones = []Actuacion{hijo.nuevaActuacion(
		padre.FaseActual, padre.EstadoActual, alta.Actuacion, 6,
	)}
	if hijo.Validar() != nil {
		return Expediente{}, ErrExpedienteInvalido
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

func huellaExpedientePadre(p Expediente) (string, error) {
	material, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func (e Expediente) instruccionCompartidaValida() bool {
	i := e.InstruccionCompartida
	if i == nil || i.ExpedientePadre.InstruccionCompartida != nil {
		return false
	}
	p := i.ExpedientePadre
	huella, err := huellaExpedientePadre(p)
	if err != nil || p.Validar() != nil || !padreDerivable(p) ||
		!referenciaValida(i.OfertaRef) || i.NumeroPlaza == 0 ||
		i.NumeroPlaza > p.Solicitud.NumeroPersonas ||
		!referenciaValida(i.ReciboDerivacionRef) || i.HuellaPadreSHA256 != huella ||
		e.Referencia == p.Referencia ||
		!numeroDerivadoValido(e.NumeroVisible, p.NumeroVisible, i.NumeroPlaza) ||
		e.OrganizacionRef != p.OrganizacionRef || e.Flujo != p.Flujo {
		return false
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
		return false
	}
	primera := e.Actuaciones[0]
	if primera.AccionClave != AccionDerivarNombramiento ||
		primera.FaseOrigen != p.FaseActual || primera.EstadoOrigen != p.EstadoActual ||
		primera.FaseDestino != FaseFiscalizacion || primera.EstadoDestino != EstadoEnCurso ||
		primera.UnidadRef != p.Asignacion.UnidadRef ||
		primera.ReciboRef != i.ReciboDerivacionRef ||
		primera.RealizadaEn.Before(p.ActualizadoEn) ||
		len(primera.DocumentosRef) != 0 || primera.RetornoRef != "" {
		return false
	}
	for _, a := range p.Actuaciones {
		if a.ReciboRef == primera.ReciboRef {
			return false
		}
	}
	return true
}
