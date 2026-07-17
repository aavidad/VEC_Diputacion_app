// Package httpseguridad define la frontera de seguridad HTTP entre las
// superficies publica, personal, interna y de administracion. No contiene un
// servidor ni implementa protocolos de identidad: proporciona invariantes y
// contratos para que el arranque componga listeners y adaptadores separados.
package httpseguridad

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConfiguracionSuperficie = errors.New("configuracion de superficie no valida")
	ErrSuperficiesCompartidas  = errors.New("las superficies comparten un limite de seguridad")
)

const (
	duracionLimiteAsercion = 5 * time.Minute
	toleranciaLimiteReloj  = 2 * time.Minute
)

// Superficie es un conjunto cerrado de clases de ruta. La zona anonima y el
// area personal comparten el portal exterior, pero la primera no crea ni
// consume sesion. Interna y administracion son fronteras de despliegue aparte.
type Superficie string

const (
	SuperficiePublicaAnonima             Superficie = "publica_anonima"
	SuperficieExternaPersonal            Superficie = "externa_personal"
	SuperficieInternaCorporativa         Superficie = "interna_corporativa"
	SuperficieAdministracionPrivilegiada Superficie = "administracion_privilegiada"
)

// Valida informa de si la superficie pertenece al conjunto cerrado.
func (s Superficie) Valida() bool {
	switch s {
	case SuperficiePublicaAnonima, SuperficieExternaPersonal,
		SuperficieInternaCorporativa, SuperficieAdministracionPrivilegiada:
		return true
	default:
		return false
	}
}

// ZonaRed expresa la zona que debe alcanzar fisicamente un listener. No es una
// etiqueta procedente de una peticion: la fija el despliegue del proceso.
type ZonaRed string

const (
	ZonaRedPublica        ZonaRed = "publica"
	ZonaRedInterna        ZonaRed = "interna"
	ZonaRedAdministracion ZonaRed = "administracion"
)

// Valida informa de si la zona pertenece al conjunto cerrado.
func (z ZonaRed) Valida() bool {
	switch z {
	case ZonaRedPublica, ZonaRedInterna, ZonaRedAdministracion:
		return true
	default:
		return false
	}
}

// ConfiguracionSuperficie es la configuracion independiente de un listener.
// RedesPermitidas siempre es explicita; incluso la red publica debe declarar
// 0.0.0.0/0 y/o ::/0 cuando quiera aceptar Internet.
type ConfiguracionSuperficie struct {
	Superficie                          Superficie
	ZonaRed                             ZonaRed
	DireccionEscucha                    string
	Audiencia                           string
	EmisorIdentidad                     string
	RedesPermitidas                     []string
	HuellasProxyTLSPermitidas           []string
	IdentidadesSANProxyPermitidas       []string
	DuracionMaximaAsercion              time.Duration
	ToleranciaReloj                     time.Duration
	PermiteAnonimo                      bool
	MetodosAdmitidos                    []MetodoAutenticacion
	FactoresRequeridos                  []MetodoAutenticacion
	MinimoFactoresVerificados           int
	MinimoGruposCriptograficosDistintos int
	GarantiaMinima                      dominiovec.AuthAssurance
	RequiereCuentaPrivilegiada          bool
}

// Validar aplica invariantes de una superficie individual. Ante cualquier
// omision la configuracion se rechaza; no se completan valores de seguridad.
func (c ConfiguracionSuperficie) Validar() error {
	if !c.Superficie.Valida() || !c.ZonaRed.Valida() {
		return fmt.Errorf("%w: superficie o zona desconocida", ErrConfiguracionSuperficie)
	}
	if err := validarCorrespondenciaZona(c.Superficie, c.ZonaRed); err != nil {
		return err
	}
	direccion, err := analizarDireccionEscucha(c.DireccionEscucha)
	if err != nil {
		return err
	}
	if c.ZonaRed != ZonaRedPublica && direccion.comodin {
		return fmt.Errorf("%w: una superficie interna no puede escuchar en todas las interfaces", ErrConfiguracionSuperficie)
	}
	if len(c.RedesPermitidas) == 0 {
		return fmt.Errorf("%w: se requiere una politica de red explicita", ErrConfiguracionSuperficie)
	}
	if _, err := NuevaPoliticaRed(c); err != nil {
		return err
	}
	if c.ToleranciaReloj < 0 || c.ToleranciaReloj > toleranciaLimiteReloj {
		return fmt.Errorf("%w: tolerancia de reloj fuera del limite permitido", ErrConfiguracionSuperficie)
	}

	if c.Superficie == SuperficiePublicaAnonima {
		if !c.PermiteAnonimo || strings.TrimSpace(c.Audiencia) != "" || strings.TrimSpace(c.EmisorIdentidad) != "" ||
			len(c.HuellasProxyTLSPermitidas) != 0 || len(c.IdentidadesSANProxyPermitidas) != 0 ||
			len(c.MetodosAdmitidos) != 0 || len(c.FactoresRequeridos) != 0 ||
			c.MinimoFactoresVerificados != 0 || c.MinimoGruposCriptograficosDistintos != 0 ||
			c.GarantiaMinima != "" || c.RequiereCuentaPrivilegiada || c.DuracionMaximaAsercion != 0 || c.ToleranciaReloj != 0 {
			return fmt.Errorf("%w: la superficie anonima no puede crear ni aceptar sesiones", ErrConfiguracionSuperficie)
		}
		return nil
	}

	if c.PermiteAnonimo {
		return fmt.Errorf("%w: solo la superficie publica admite anonimato", ErrConfiguracionSuperficie)
	}
	if validarAudienciaConfigurada(c.Audiencia) != nil || validarEmisorConfigurado(c.EmisorIdentidad) != nil ||
		c.DuracionMaximaAsercion <= 0 || c.DuracionMaximaAsercion > duracionLimiteAsercion ||
		!c.GarantiaMinima.Valida() {
		return fmt.Errorf("%w: emisor y duracion maxima son obligatorios", ErrConfiguracionSuperficie)
	}
	if err := validarConfianzaProxyTLS(c); err != nil {
		return err
	}
	if err := validarPoliticaFactores(c); err != nil {
		return err
	}

	switch c.Superficie {
	case SuperficieExternaPersonal:
		if !dominiovec.CumpleGarantiaAutenticacion(c.GarantiaMinima, dominiovec.AuthAssuranceSubstantial) {
			return fmt.Errorf("%w: el area personal exige garantia sustancial o superior", ErrConfiguracionSuperficie)
		}
	case SuperficieInternaCorporativa, SuperficieAdministracionPrivilegiada:
		if !contieneMetodo(c.FactoresRequeridos, MetodoKerberos) ||
			!contieneMetodo(c.FactoresRequeridos, MetodoCertificado) ||
			c.MinimoGruposCriptograficosDistintos < 2 || c.GarantiaMinima != dominiovec.AuthAssuranceHigh {
			return fmt.Errorf("%w: el acceso interno exige Kerberos y certificado con grupos criptograficos distintos", ErrConfiguracionSuperficie)
		}
	}
	if c.Superficie == SuperficieAdministracionPrivilegiada && !c.RequiereCuentaPrivilegiada {
		return fmt.Errorf("%w: administracion exige cuenta privilegiada separada", ErrConfiguracionSuperficie)
	}
	if c.Superficie != SuperficieAdministracionPrivilegiada && c.RequiereCuentaPrivilegiada {
		return fmt.Errorf("%w: la cuenta privilegiada solo pertenece a administracion", ErrConfiguracionSuperficie)
	}
	return nil
}

func validarCorrespondenciaZona(superficie Superficie, zona ZonaRed) error {
	esperada := ZonaRedPublica
	switch superficie {
	case SuperficieInternaCorporativa:
		esperada = ZonaRedInterna
	case SuperficieAdministracionPrivilegiada:
		esperada = ZonaRedAdministracion
	}
	if zona != esperada {
		return fmt.Errorf("%w: %s debe estar en la zona %s", ErrConfiguracionSuperficie, superficie, esperada)
	}
	return nil
}

type direccionListener struct {
	canonica string
	puerto   int
	ip       net.IP
	comodin  bool
}

func analizarDireccionEscucha(direccion string) (direccionListener, error) {
	direccion = strings.TrimSpace(direccion)
	if direccion == "" {
		return direccionListener{}, fmt.Errorf("%w: direccion de escucha obligatoria", ErrConfiguracionSuperficie)
	}
	host, puertoTexto, err := net.SplitHostPort(direccion)
	if err != nil {
		return direccionListener{}, fmt.Errorf("%w: direccion de escucha %q: %v", ErrConfiguracionSuperficie, direccion, err)
	}
	puerto, err := strconv.Atoi(puertoTexto)
	if err != nil || puerto < 1 || puerto > 65535 {
		return direccionListener{}, fmt.Errorf("%w: el puerto de escucha debe ser numerico y estar entre 1 y 65535", ErrConfiguracionSuperficie)
	}
	host = strings.TrimSpace(host)
	resultado := direccionListener{puerto: puerto, comodin: host == ""}
	if host != "" {
		ip := net.ParseIP(host)
		if ip == nil {
			return direccionListener{}, fmt.Errorf("%w: el listener debe declarar una IP literal, no un nombre resoluble", ErrConfiguracionSuperficie)
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		}
		resultado.ip = append(net.IP(nil), ip...)
		resultado.comodin = ip.IsUnspecified()
		host = ip.String()
	}
	resultado.canonica = net.JoinHostPort(host, strconv.Itoa(puerto))
	return resultado, nil
}

func validarPoliticaFactores(c ConfiguracionSuperficie) error {
	if len(c.MetodosAdmitidos) == 0 || c.MinimoFactoresVerificados <= 0 ||
		c.MinimoGruposCriptograficosDistintos <= 0 {
		return fmt.Errorf("%w: politica de factores incompleta", ErrConfiguracionSuperficie)
	}
	admitidos := make(map[MetodoAutenticacion]struct{}, len(c.MetodosAdmitidos))
	for _, metodo := range c.MetodosAdmitidos {
		if !metodo.Valido() {
			return fmt.Errorf("%w: metodo de autenticacion desconocido", ErrConfiguracionSuperficie)
		}
		if _, existe := admitidos[metodo]; existe {
			return fmt.Errorf("%w: metodo de autenticacion duplicado", ErrConfiguracionSuperficie)
		}
		admitidos[metodo] = struct{}{}
	}
	requeridos := make(map[MetodoAutenticacion]struct{}, len(c.FactoresRequeridos))
	for _, metodo := range c.FactoresRequeridos {
		if _, admitido := admitidos[metodo]; !admitido {
			return fmt.Errorf("%w: un factor requerido no esta admitido", ErrConfiguracionSuperficie)
		}
		if _, existe := requeridos[metodo]; existe {
			return fmt.Errorf("%w: factor requerido duplicado", ErrConfiguracionSuperficie)
		}
		requeridos[metodo] = struct{}{}
	}
	if c.MinimoFactoresVerificados < len(requeridos) || c.MinimoFactoresVerificados > len(admitidos) ||
		c.MinimoGruposCriptograficosDistintos > len(admitidos) ||
		c.MinimoGruposCriptograficosDistintos > c.MinimoFactoresVerificados {
		return fmt.Errorf("%w: minimos de factores o grupos criptograficos incoherentes", ErrConfiguracionSuperficie)
	}
	return nil
}

// ValidarArquitecturaCompleta exige el despliegue declarado de las cuatro
// superficies. ValidarConjuntoSuperficies sigue siendo util para validar un
// proceso aislado, pero no sustituye esta comprobacion global de arranque.
func ValidarArquitecturaCompleta(configuraciones []ConfiguracionSuperficie) error {
	if len(configuraciones) != 4 {
		return fmt.Errorf("%w: la arquitectura debe declarar exactamente cuatro superficies", ErrConfiguracionSuperficie)
	}
	if err := ValidarConjuntoSuperficies(configuraciones); err != nil {
		return err
	}
	presentes := make(map[Superficie]struct{}, len(configuraciones))
	for _, configuracion := range configuraciones {
		presentes[configuracion.Superficie] = struct{}{}
	}
	for _, requerida := range []Superficie{
		SuperficiePublicaAnonima,
		SuperficieExternaPersonal,
		SuperficieInternaCorporativa,
		SuperficieAdministracionPrivilegiada,
	} {
		if _, existe := presentes[requerida]; !existe {
			return fmt.Errorf("%w: falta la superficie %s", ErrConfiguracionSuperficie, requerida)
		}
	}
	return nil
}

func contieneMetodo(metodos []MetodoAutenticacion, buscado MetodoAutenticacion) bool {
	for _, metodo := range metodos {
		if metodo == buscado {
			return true
		}
	}
	return false
}

// ValidarConjuntoSuperficies impide compartir audiencia. Solo las dos clases
// del portal exterior pueden compartir listener; interna y administracion
// siempre utilizan entradas diferentes.
func ValidarConjuntoSuperficies(configuraciones []ConfiguracionSuperficie) error {
	if len(configuraciones) == 0 {
		return fmt.Errorf("%w: conjunto vacio", ErrConfiguracionSuperficie)
	}
	superficies := make(map[Superficie]struct{}, len(configuraciones))
	audiencias := make(map[string]Superficie, len(configuraciones))
	listeners := make([]listenerRegistrado, 0, len(configuraciones))
	for _, configuracion := range configuraciones {
		if err := configuracion.Validar(); err != nil {
			return err
		}
		if _, existe := superficies[configuracion.Superficie]; existe {
			return fmt.Errorf("%w: superficie %s duplicada", ErrSuperficiesCompartidas, configuracion.Superficie)
		}
		superficies[configuracion.Superficie] = struct{}{}
		if strings.TrimSpace(configuracion.Audiencia) != "" {
			if err := registrarLimiteUnico(audiencias, configuracion.Audiencia, configuracion.Superficie, "audiencia"); err != nil {
				return err
			}
		}
		if err := registrarListener(&listeners, configuracion.DireccionEscucha, configuracion.Superficie); err != nil {
			return err
		}
	}
	if err := exigirListenerExteriorComun(listeners); err != nil {
		return err
	}
	return nil
}

type listenerRegistrado struct {
	direccion  direccionListener
	superficie Superficie
}

func registrarListener(usados *[]listenerRegistrado, valor string, superficie Superficie) error {
	direccion, err := analizarDireccionEscucha(valor)
	if err != nil {
		return err
	}
	for _, anterior := range *usados {
		if !listenersSolapados(anterior.direccion, direccion) {
			continue
		}
		if esParejaPortalExterior(anterior.superficie, superficie) &&
			anterior.direccion.canonica == direccion.canonica {
			continue
		}
		return fmt.Errorf("%w: listeners solapados por %s y %s", ErrSuperficiesCompartidas, anterior.superficie, superficie)
	}
	*usados = append(*usados, listenerRegistrado{direccion: direccion, superficie: superficie})
	return nil
}

func listenersSolapados(primero, segundo direccionListener) bool {
	if primero.puerto != segundo.puerto {
		return false
	}
	if primero.comodin || segundo.comodin {
		return true
	}
	return primero.ip.Equal(segundo.ip)
}

func exigirListenerExteriorComun(listeners []listenerRegistrado) error {
	var publica, personal *direccionListener
	for indice := range listeners {
		switch listeners[indice].superficie {
		case SuperficiePublicaAnonima:
			publica = &listeners[indice].direccion
		case SuperficieExternaPersonal:
			personal = &listeners[indice].direccion
		}
	}
	if publica != nil && personal != nil && publica.canonica != personal.canonica {
		return fmt.Errorf("%w: las clases anonima y personal deben compartir exactamente el listener exterior", ErrSuperficiesCompartidas)
	}
	return nil
}

func esParejaPortalExterior(primera, segunda Superficie) bool {
	return (primera == SuperficiePublicaAnonima && segunda == SuperficieExternaPersonal) ||
		(primera == SuperficieExternaPersonal && segunda == SuperficiePublicaAnonima)
}

func registrarLimiteUnico(usados map[string]Superficie, valor string, superficie Superficie, clase string) error {
	clave := strings.ToLower(strings.TrimSpace(valor))
	if anterior, existe := usados[clave]; existe {
		return fmt.Errorf("%w: %s compartido por %s y %s", ErrSuperficiesCompartidas, clase, anterior, superficie)
	}
	usados[clave] = superficie
	return nil
}
