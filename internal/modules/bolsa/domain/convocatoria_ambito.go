package domain

import (
	"encoding/json"
	"strings"
)

const (
	prefijoOrganizacionConvocatoria  = "org_"
	prefijoUnidadGestionConvocatoria = "uni_"
	minimoCargaAmbitoConvocatoria    = 16
	maximoCargaAmbitoConvocatoria    = 80
)

// AmbitoOrganizativoConvocatoria fija el alcance organizativo de toda la
// cadena de versiones. Sus referencias son opacas y su estado no se puede
// modificar tras construirlo.
type AmbitoOrganizativoConvocatoria struct {
	organizacionRef  string
	unidadGestionRef string
}

// NuevoAmbitoOrganizativoConvocatoria no normaliza ni completa referencias:
// solo admite la representacion canonica exacta recibida.
func NuevoAmbitoOrganizativoConvocatoria(
	organizacionRef, unidadGestionRef string,
) (AmbitoOrganizativoConvocatoria, error) {
	ambito := AmbitoOrganizativoConvocatoria{
		organizacionRef: organizacionRef, unidadGestionRef: unidadGestionRef,
	}
	if ambito.Validar() != nil {
		return AmbitoOrganizativoConvocatoria{}, ErrVersionConvocatoriaGobernadaInvalida
	}
	return ambito, nil
}

func (a AmbitoOrganizativoConvocatoria) Validar() error {
	if !referenciaAmbitoConvocatoriaValida(a.organizacionRef, prefijoOrganizacionConvocatoria) ||
		(a.unidadGestionRef != "" &&
			!referenciaAmbitoConvocatoriaValida(a.unidadGestionRef, prefijoUnidadGestionConvocatoria)) {
		return ErrVersionConvocatoriaGobernadaInvalida
	}
	return nil
}

func (a AmbitoOrganizativoConvocatoria) OrganizacionRef() string { return a.organizacionRef }

func (a AmbitoOrganizativoConvocatoria) UnidadGestionRef() string { return a.unidadGestionRef }

func (a AmbitoOrganizativoConvocatoria) ClonarCanonico() (
	AmbitoOrganizativoConvocatoria,
	error,
) {
	if a.Validar() != nil {
		return AmbitoOrganizativoConvocatoria{}, ErrVersionConvocatoriaGobernadaInvalida
	}
	return a, nil
}

func (a AmbitoOrganizativoConvocatoria) MarshalJSON() ([]byte, error) {
	if a.Validar() != nil {
		return nil, ErrVersionConvocatoriaGobernadaInvalida
	}
	return json.Marshal(materialAmbitoOrganizativoConvocatoriaDe(a))
}

type materialAmbitoOrganizativoConvocatoria struct {
	OrganizacionRef  string `json:"organizacion_ref"`
	UnidadGestionRef string `json:"unidad_gestion_ref,omitempty"`
}

func materialAmbitoOrganizativoConvocatoriaDe(
	ambito AmbitoOrganizativoConvocatoria,
) materialAmbitoOrganizativoConvocatoria {
	return materialAmbitoOrganizativoConvocatoria{
		OrganizacionRef: ambito.OrganizacionRef(), UnidadGestionRef: ambito.UnidadGestionRef(),
	}
}

func (m materialAmbitoOrganizativoConvocatoria) restaurar() (
	AmbitoOrganizativoConvocatoria,
	error,
) {
	return NuevoAmbitoOrganizativoConvocatoria(m.OrganizacionRef, m.UnidadGestionRef)
}

func referenciaAmbitoConvocatoriaValida(referencia, prefijo string) bool {
	if !strings.HasPrefix(referencia, prefijo) {
		return false
	}
	carga := referencia[len(prefijo):]
	if len(carga) < minimoCargaAmbitoConvocatoria || len(carga) > maximoCargaAmbitoConvocatoria {
		return false
	}
	for indice := range len(carga) {
		caracter := carga[indice]
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') {
			return false
		}
	}
	return true
}
