// Package separacionportales fija qué credenciales de base de datos y qué
// material de claves puede ver cada proceso de VEC cuando el portal externo
// (Área personal de aspirantes y consulta pública) y el interno (RRHH y
// empleados) se ejecutan en procesos distintos.
//
// La separación protege de verdad solo si cada proceso arranca sin las
// credenciales ni las claves del otro: quien comprometa el proceso expuesto a
// Internet no debe poder abrir datos de trabajadores. Por eso las
// comprobaciones se hacen antes de componer nada, sobre el entorno y el
// directorio de material, y fallan cerradas: un elemento del otro portal o
// uno sin clasificar en el portal externo impiden arrancar.
//
// Sin separación configurada (portal combinado) no se comprueba nada y VEC
// se comporta como antes. El paquete no abre conexiones ni lee secretos: solo
// nombres de variables, rutas relativas y huellas SHA-256 para comparar.
package separacionportales

import (
	"errors"
	"fmt"
)

// Portal identifica qué superficie atiende un proceso.
type Portal string

const (
	// PortalCombinado es el funcionamiento histórico: un proceso atiende a
	// los dos portales. Es el valor por defecto (variable vacía).
	PortalCombinado Portal = ""
	// PortalInterno atiende al portal de RRHH y del empleado.
	PortalInterno Portal = "interno"
	// PortalExterno atiende al Área personal y a la consulta pública.
	PortalExterno Portal = "externo"
)

var (
	// ErrPortalDesconocido rechaza un valor de portal fuera del conjunto
	// cerrado; una errata nunca se interpreta como portal combinado.
	ErrPortalDesconocido = errors.New("separacion de portales: portal de proceso desconocido")
	// ErrSeparacionPortales agrupa todos los rechazos de la comprobación.
	ErrSeparacionPortales = errors.New("separacion de portales")
)

// Parsear traduce el valor configurado. Solo admite la cadena vacía,
// «interno» y «externo», sin espacios ni variaciones de mayúsculas.
func Parsear(valor string) (Portal, error) {
	switch Portal(valor) {
	case PortalCombinado, PortalInterno, PortalExterno:
		return Portal(valor), nil
	default:
		return PortalCombinado, ErrPortalDesconocido
	}
}

// Separado indica si el proceso atiende a un solo portal.
func (p Portal) Separado() bool {
	return p == PortalInterno || p == PortalExterno
}

// otro devuelve el portal contrario de un proceso separado.
func (p Portal) otro() Portal {
	if p == PortalInterno {
		return PortalExterno
	}
	return PortalInterno
}

// ErrorSeparacion describe el primer elemento que impide arrancar. Elemento
// es un nombre de variable o una ruta relativa al material, nunca un valor.
type ErrorSeparacion struct {
	Motivo   string
	Elemento string
}

func (e *ErrorSeparacion) Error() string {
	if e.Elemento == "" {
		return fmt.Sprintf("%s: %s", ErrSeparacionPortales, e.Motivo)
	}
	return fmt.Sprintf("%s: %s: %s", ErrSeparacionPortales, e.Motivo, e.Elemento)
}

func (e *ErrorSeparacion) Unwrap() error { return ErrSeparacionPortales }

func rechazo(motivo, elemento string) error {
	return &ErrorSeparacion{Motivo: motivo, Elemento: elemento}
}
