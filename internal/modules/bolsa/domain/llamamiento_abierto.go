package domain

import "errors"

const maximoEnteroSeguroVersionLlamamiento = uint64(9_007_199_254_740_991)

var (
	ErrLlamamientoAbiertoInvalido = errors.New(
		"bolsa: llamamiento abierto invalido",
	)
	ErrVersionLlamamientoEnConflicto = errors.New(
		"bolsa: version de llamamiento en conflicto",
	)
	ErrTerminalLlamamientoInvalido = errors.New(
		"bolsa: terminal de llamamiento invalido",
	)
	ErrTerminalLlamamientoIncompatible = errors.New(
		"bolsa: terminal de llamamiento incompatible",
	)
)

// EstadoLlamamiento contiene solo el ciclo tecnico cerrado necesario para
// proteger un llamamiento abierto. Las causas y reglas administrativas
// permanecen en sus autoridades gobernadas y no se deducen en este agregado.
type EstadoLlamamiento string

const (
	EstadoLlamamientoAbierto    EstadoLlamamiento = "abierto"
	EstadoLlamamientoAceptado   EstadoLlamamiento = "aceptacion"
	EstadoLlamamientoRenunciado EstadoLlamamiento = "renuncia"
	EstadoLlamamientoExpirado   EstadoLlamamiento = "expiracion_gobernada"
)

func (e EstadoLlamamiento) esTerminal() bool {
	return e == EstadoLlamamientoAceptado ||
		e == EstadoLlamamientoRenunciado ||
		e == EstadoLlamamientoExpirado
}

func (e EstadoLlamamiento) valido() bool {
	return e == EstadoLlamamientoAbierto || e.esTerminal()
}

// DatosLlamamientoAbierto enlaza el agregado con hechos opacos ya existentes.
// No transporta identidad, posicion, destinatario ni datos de contacto.
type DatosLlamamientoAbierto struct {
	LlamamientoRef string
	BolsaRef       string
	NecesidadRef   string
	PropuestaRef   string
	Version        uint64
}

func (d DatosLlamamientoAbierto) validarParaApertura() error {
	if !referenciaLlamamientoOpacaValida(d.LlamamientoRef) ||
		!referenciaLlamamientoOpacaValida(d.BolsaRef) ||
		!referenciaLlamamientoOpacaValida(d.NecesidadRef) ||
		!referenciaLlamamientoOpacaValida(d.PropuestaRef) ||
		d.Version == 0 || d.Version >= maximoEnteroSeguroVersionLlamamiento {
		return ErrLlamamientoAbiertoInvalido
	}
	return nil
}

// TerminalLlamamiento identifica un unico hecho terminal. OperacionRef es la
// identidad opaca que permite distinguir un replay exacto de otra operacion;
// no acredita por si misma competencia, autorizacion ni persistencia.
type TerminalLlamamiento struct {
	Estado       EstadoLlamamiento
	OperacionRef string
}

func (t TerminalLlamamiento) Validar() error {
	if !t.Estado.esTerminal() ||
		!referenciaLlamamientoOpacaValida(t.OperacionRef) {
		return ErrTerminalLlamamientoInvalido
	}
	return nil
}

// LlamamientoAbierto es un valor inmutable. Cada transicion devuelve otro
// valor, de modo que varias derivaciones concurrentes no comparten mutacion.
type LlamamientoAbierto struct {
	datos    DatosLlamamientoAbierto
	estado   EstadoLlamamiento
	terminal TerminalLlamamiento
}

func NuevoLlamamientoAbierto(
	datos DatosLlamamientoAbierto,
) (LlamamientoAbierto, error) {
	if err := datos.validarParaApertura(); err != nil {
		return LlamamientoAbierto{}, err
	}
	return LlamamientoAbierto{
		datos:  datos,
		estado: EstadoLlamamientoAbierto,
	}, nil
}

func (l LlamamientoAbierto) Validar() error {
	if !l.estado.valido() ||
		!referenciaLlamamientoOpacaValida(l.datos.LlamamientoRef) ||
		!referenciaLlamamientoOpacaValida(l.datos.BolsaRef) ||
		!referenciaLlamamientoOpacaValida(l.datos.NecesidadRef) ||
		!referenciaLlamamientoOpacaValida(l.datos.PropuestaRef) ||
		l.datos.Version == 0 ||
		l.datos.Version > maximoEnteroSeguroVersionLlamamiento {
		return ErrLlamamientoAbiertoInvalido
	}
	if l.estado == EstadoLlamamientoAbierto {
		if l.datos.Version >= maximoEnteroSeguroVersionLlamamiento ||
			l.terminal != (TerminalLlamamiento{}) {
			return ErrLlamamientoAbiertoInvalido
		}
		return nil
	}
	if l.datos.Version < 2 || l.terminal.Validar() != nil ||
		l.terminal.Estado != l.estado {
		return ErrLlamamientoAbiertoInvalido
	}
	return nil
}

func (l LlamamientoAbierto) Datos() DatosLlamamientoAbierto {
	return l.datos
}

func (l LlamamientoAbierto) Estado() EstadoLlamamiento {
	return l.estado
}

func (l LlamamientoAbierto) EsTerminal() bool {
	return l.estado.esTerminal()
}

func (l LlamamientoAbierto) Terminal() (TerminalLlamamiento, bool) {
	if !l.EsTerminal() {
		return TerminalLlamamiento{}, false
	}
	return l.terminal, true
}

// TransicionarATerminal aplica CAS sobre la version abierta. Un replay exacto
// conserva el valor ya producido; otra operacion o estado tras el cierre se
// rechazan sin alterar el receptor.
func (l LlamamientoAbierto) TransicionarATerminal(
	versionEsperada uint64,
	terminal *TerminalLlamamiento,
) (LlamamientoAbierto, error) {
	if l.Validar() != nil {
		return LlamamientoAbierto{}, ErrLlamamientoAbiertoInvalido
	}
	if terminal == nil || terminal.Validar() != nil {
		return LlamamientoAbierto{}, ErrTerminalLlamamientoInvalido
	}
	terminalCanonico := *terminal
	if l.EsTerminal() {
		if l.terminal != terminalCanonico {
			return LlamamientoAbierto{}, ErrTerminalLlamamientoIncompatible
		}
		if versionEsperada == 0 ||
			versionEsperada+1 != l.datos.Version {
			return LlamamientoAbierto{}, ErrVersionLlamamientoEnConflicto
		}
		return l, nil
	}
	if versionEsperada == 0 || versionEsperada != l.datos.Version ||
		versionEsperada >= maximoEnteroSeguroVersionLlamamiento {
		return LlamamientoAbierto{}, ErrVersionLlamamientoEnConflicto
	}
	siguiente := l
	siguiente.datos.Version++
	siguiente.estado = terminalCanonico.Estado
	siguiente.terminal = terminalCanonico
	if siguiente.Validar() != nil {
		return LlamamientoAbierto{}, ErrLlamamientoAbiertoInvalido
	}
	return siguiente, nil
}
