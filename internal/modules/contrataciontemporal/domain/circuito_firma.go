package domain

import (
	"errors"
	"regexp"
	"time"
)

// El circuito de firma de los borradores viene de un catálogo versionado: qué
// documentos, cuántos pasos, quién firma y qué ocurre ante una devolución. El
// dominio solo sabe calcular el estado de cada paso a partir de la historia
// de firmas registradas; no decide el circuito ni da eficacia a la firma. Una
// firma registrada aquí es una firma de prueba verificada, sin eficacia
// administrativa hasta el portafirmas corporativo.

var (
	// ErrCircuitoFirmaIncoherente: el circuito recibido no forma pasos 1..n.
	ErrCircuitoFirmaIncoherente = errors.New("contratacion temporal: circuito de firma incoherente")
	// ErrHistoriaFirmaIncoherente: la historia registrada no sigue el circuito.
	ErrHistoriaFirmaIncoherente = errors.New("contratacion temporal: historia de firmas incoherente")
)

// ResultadoFirmaDocumento es lo que registra un paso: firma o devolución.
type ResultadoFirmaDocumento string

const (
	ResultadoFirmaFirmado  ResultadoFirmaDocumento = "firmado"
	ResultadoFirmaDevuelto ResultadoFirmaDocumento = "devuelto"
)

// DevolucionPasoFirma es adónde vuelve el documento si el paso lo devuelve.
type DevolucionPasoFirma string

const (
	DevolucionVuelveARedaccion   DevolucionPasoFirma = "vuelve_a_redaccion"
	DevolucionVuelvePasoAnterior DevolucionPasoFirma = "vuelve_paso_anterior"
)

// EstadoPasoFirma es el estado visible de un paso del circuito.
type EstadoPasoFirma string

const (
	EstadoPasoPendienteFirma EstadoPasoFirma = "pendiente_firma"
	EstadoPasoEnEspera       EstadoPasoFirma = "en_espera"
	EstadoPasoFirmado        EstadoPasoFirma = "firmado"
	EstadoPasoDevuelto       EstadoPasoFirma = "devuelto"
)

// MaximoPasosCircuitoFirma coincide con el límite del catálogo y de CT118.
const MaximoPasosCircuitoFirma = 16

var (
	claveDocumentoFirmaValida = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	huellaFirmaValida         = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ClaveDocumentoFirmaValida admite claves de documento del catálogo.
func ClaveDocumentoFirmaValida(v string) bool { return claveDocumentoFirmaValida.MatchString(v) }

// HuellaSHA256FirmaValida admite una huella SHA-256 hexadecimal no nula.
func HuellaSHA256FirmaValida(v string) bool {
	return huellaFirmaValida.MatchString(v) && v != "0000000000000000000000000000000000000000000000000000000000000000"
}

// PasoCircuitoFirma es un paso resuelto del catálogo.
type PasoCircuitoFirma struct {
	Orden      int
	Cargo      string
	PerfilRef  string
	Accion     string
	Devolucion DevolucionPasoFirma
	// Referencia es catalogo:version:entrada del paso.
	Referencia string
}

// CircuitoFirmaDocumento son los pasos ordenados de un documento.
type CircuitoFirmaDocumento struct {
	Documento string
	Etiqueta  string
	Pasos     []PasoCircuitoFirma
}

// Validar exige pasos 1..n, una referencia por paso y una devolución conocida
// (el primer paso no puede volver a un paso anterior).
func (c CircuitoFirmaDocumento) Validar() error {
	if !ClaveDocumentoFirmaValida(c.Documento) || c.Etiqueta == "" || len(c.Pasos) == 0 || len(c.Pasos) > MaximoPasosCircuitoFirma {
		return ErrCircuitoFirmaIncoherente
	}
	for i, p := range c.Pasos {
		if p.Orden != i+1 || p.Referencia == "" || p.Cargo == "" ||
			(p.Devolucion != DevolucionVuelveARedaccion && p.Devolucion != DevolucionVuelvePasoAnterior) ||
			(p.Orden == 1 && p.Devolucion == DevolucionVuelvePasoAnterior) {
			return ErrCircuitoFirmaIncoherente
		}
	}
	return nil
}

// CircuitoFirma es el catálogo vigente con su procedencia.
type CircuitoFirma struct {
	CatalogoRef    string
	HuellaCatalogo string
	Ejemplo        bool
	Documentos     []CircuitoFirmaDocumento
}

// Documento devuelve el circuito de un documento del catálogo.
func (c CircuitoFirma) Documento(clave string) (CircuitoFirmaDocumento, bool) {
	for _, d := range c.Documentos {
		if d.Documento == clave {
			return d, true
		}
	}
	return CircuitoFirmaDocumento{}, false
}

// EventoFirmaDocumento es una fila de la historia de firmas de un documento.
type EventoFirmaDocumento struct {
	Secuencia      int
	CatalogoHuella string
	PasoOrden      int
	Resultado      ResultadoFirmaDocumento
	// ConMotivoDevolucion indica que la devolución consta con su motivo; el
	// texto del motivo no se lee: la consulta de la historia no consume una
	// decisión atestada y solo devuelve campos no personales.
	ConMotivoDevolucion bool
	OriginalHuella      string
	FirmadoHuella       string
	ReciboRef           string
	RegistradaEn        time.Time
}

// EstadoPasoCalculado es el estado de un paso derivado de la historia.
type EstadoPasoCalculado struct {
	Orden        int
	Estado       EstadoPasoFirma
	ReciboRef    string
	RegistradaEn time.Time
}

// EstadoCircuitoDocumento resume el circuito de un documento: el paso que
// espera firma (0 si el circuito está completo), la última secuencia
// registrada y la huella que debe tener el original del paso pendiente: cada
// paso firma por separado el mismo borrador que firmó el paso anterior; el
// primer paso fija ese borrador (huella vacía).
type EstadoCircuitoDocumento struct {
	Documento               string
	Pasos                   []EstadoPasoCalculado
	PasoPendiente           int
	UltimaSecuencia         int
	OriginalEsperadoHuella  string
	Completo                bool
	EventosCatalogoAnterior int
}

type marcaPaso struct {
	huella, recibo string
	en             time.Time
}

// CalcularEstadoCircuitoFirma aplica la historia en orden de secuencia. Un
// evento de otro catálogo (huella distinta) reinicia el circuito: el
// catálogo cambió y sus pasos ya no son comparables. Cada evento del
// catálogo vigente debe afectar al paso pendiente en ese momento; si no, la
// historia es incoherente y no se presenta ningún estado.
func CalcularEstadoCircuitoFirma(c CircuitoFirmaDocumento, huellaCatalogo string, eventos []EventoFirmaDocumento) (EstadoCircuitoDocumento, error) {
	if c.Validar() != nil || !HuellaSHA256FirmaValida(huellaCatalogo) {
		return EstadoCircuitoDocumento{}, ErrCircuitoFirmaIncoherente
	}
	n := len(c.Pasos)
	firmados := map[int]marcaPaso{}
	devueltos := map[int]marcaPaso{}
	pendiente := func() int {
		for o := 1; o <= n; o++ {
			if _, ok := firmados[o]; !ok {
				return o
			}
		}
		return 0
	}
	anteriores := 0
	for i, e := range eventos {
		if e.Secuencia != i+1 {
			return EstadoCircuitoDocumento{}, ErrHistoriaFirmaIncoherente
		}
		if e.CatalogoHuella != huellaCatalogo {
			firmados, devueltos = map[int]marcaPaso{}, map[int]marcaPaso{}
			anteriores++
			continue
		}
		if e.PasoOrden < 1 || e.PasoOrden > n || e.PasoOrden != pendiente() {
			return EstadoCircuitoDocumento{}, ErrHistoriaFirmaIncoherente
		}
		marca := marcaPaso{huella: e.OriginalHuella, recibo: e.ReciboRef, en: e.RegistradaEn}
		switch e.Resultado {
		case ResultadoFirmaFirmado:
			esperado := ""
			if e.PasoOrden > 1 {
				esperado = firmados[e.PasoOrden-1].huella
			}
			if !HuellaSHA256FirmaValida(e.FirmadoHuella) || !HuellaSHA256FirmaValida(e.OriginalHuella) ||
				(esperado != "" && e.OriginalHuella != esperado) {
				return EstadoCircuitoDocumento{}, ErrHistoriaFirmaIncoherente
			}
			firmados[e.PasoOrden] = marca
			delete(devueltos, e.PasoOrden)
		case ResultadoFirmaDevuelto:
			if !e.ConMotivoDevolucion {
				return EstadoCircuitoDocumento{}, ErrHistoriaFirmaIncoherente
			}
			if c.Pasos[e.PasoOrden-1].Devolucion == DevolucionVuelveARedaccion {
				firmados = map[int]marcaPaso{}
			} else {
				delete(firmados, e.PasoOrden-1)
			}
			devueltos[e.PasoOrden] = marca
		default:
			return EstadoCircuitoDocumento{}, ErrHistoriaFirmaIncoherente
		}
	}
	estado := EstadoCircuitoDocumento{Documento: c.Documento, PasoPendiente: pendiente(), UltimaSecuencia: len(eventos), EventosCatalogoAnterior: anteriores}
	estado.Completo = estado.PasoPendiente == 0
	if estado.PasoPendiente > 1 {
		estado.OriginalEsperadoHuella = firmados[estado.PasoPendiente-1].huella
	}
	for _, p := range c.Pasos {
		calc := EstadoPasoCalculado{Orden: p.Orden}
		if f, ok := firmados[p.Orden]; ok {
			calc.Estado, calc.ReciboRef, calc.RegistradaEn = EstadoPasoFirmado, f.recibo, f.en
		} else if d, ok := devueltos[p.Orden]; ok {
			calc.Estado, calc.ReciboRef, calc.RegistradaEn = EstadoPasoDevuelto, d.recibo, d.en
		} else if p.Orden == estado.PasoPendiente {
			calc.Estado = EstadoPasoPendienteFirma
		} else {
			calc.Estado = EstadoPasoEnEspera
		}
		estado.Pasos = append(estado.Pasos, calc)
	}
	return estado, nil
}
