package bootstrap

import (
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	mensajeFuenteConfiguracionArranque             = "bootstrap: fuente configurada no disponible"
	mensajeFuenteNecesidadesAltaNoConfigurada      = "contratacion temporal: falta configurar VEC_CT_NECESIDADES_ALTA_SOURCE_PATH para necesidades de alta"
	mensajeCatalogoRPTNecesidadesAltaNoConfigurado = "contratacion temporal: falta configurar VEC_RPT_CATALOGO_PATH para necesidades de alta"
)

// Este marcador identifica una fuente configurada sin transportar su ruta ni
// sustituir la causa original que necesitan errors.Is y errors.As.
type falloFuenteConfiguracionArranque struct{ causa error }

func (*falloFuenteConfiguracionArranque) Error() string              { return mensajeFuenteConfiguracionArranque }
func (f *falloFuenteConfiguracionArranque) Unwrap() error            { return f.causa }
func (*falloFuenteConfiguracionArranque) ClaseErrorArranque() string { return "configuracion" }

type falloConfiguracionNecesidadesAltaArranque struct{ mensaje string }

func (f *falloConfiguracionNecesidadesAltaArranque) Error() string { return f.mensaje }
func (*falloConfiguracionNecesidadesAltaArranque) Unwrap() error {
	return ports.ErrFuenteNecesidadesAltaNoDisponible
}
func (*falloConfiguracionNecesidadesAltaArranque) ClaseErrorArranque() string {
	return "configuracion"
}

var (
	errFuenteNecesidadesAltaNoConfigurada = marcarFalloComponenteArranque(
		"fuente_necesidades_alta",
		&falloConfiguracionNecesidadesAltaArranque{mensaje: mensajeFuenteNecesidadesAltaNoConfigurada},
	)
	errCatalogoRPTNecesidadesAltaNoConfigurado = marcarFalloComponenteArranque(
		"catalogo_rpt",
		&falloConfiguracionNecesidadesAltaArranque{mensaje: mensajeCatalogoRPTNecesidadesAltaNoConfigurado},
	)
)

func marcarFalloFuenteConfiguracionArranque(componente string, err error) error {
	if err == nil {
		return nil
	}
	switch componente {
	case "fuente_organizacion", "catalogo_rpt", "fuente_necesidades_alta":
	default:
		componente = "sin_etiqueta"
	}
	return marcarFalloComponenteArranque(componente, &falloFuenteConfiguracionArranque{causa: err})
}

// falloComponenteArranque conserva la causa y añade la etapa de composición
// conocida por el punto de llamada. La etiqueta procede solo del código.
type falloComponenteArranque struct {
	componente string
	causa      error
}

func (f *falloComponenteArranque) Error() string { return f.causa.Error() }
func (f *falloComponenteArranque) Unwrap() error { return f.causa }

func marcarFalloComponenteArranque(componente string, err error) error {
	if err == nil {
		return nil
	}
	var previo *falloComponenteArranque
	if errors.As(err, &previo) {
		return err // Conservar la etapa más precisa de un constructor interior.
	}
	if !componenteArranqueValido(componente) {
		componente = "sin_etiqueta"
	}
	return &falloComponenteArranque{componente: componente, causa: err}
}

// ComponenteFalloArranque entrega una etiqueta cerrada para el log fatal.
// Un error no etiquetado queda visible como una deuda de composición.
func ComponenteFalloArranque(err error) string {
	var fallo *falloComponenteArranque
	if errors.As(err, &fallo) && fallo != nil && componenteArranqueValido(fallo.componente) {
		return fallo.componente
	}
	return "sin_etiqueta"
}

func componenteArranqueValido(valor string) bool {
	if len(valor) == 0 || len(valor) > 64 {
		return false
	}
	for _, c := range valor {
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return false
	}
	return true
}
