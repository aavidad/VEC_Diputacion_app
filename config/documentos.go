package config

import "errors"

// EnvDocumentosEnabled monta el servicio común de Documentos (consulta del
// expediente documental por RRHH y su vista web). Es un selector de
// despliegue, no un permiso: solo admite "true" y "false"; la ausencia
// equivale a apagado y cada petición la sigue autorizando el PDP común.
const EnvDocumentosEnabled = "VEC_DOCUMENTOS_ENABLED"

var (
	ErrConfiguracionDocumentosSelector   = errors.New("config: selector de Documentos invalido")
	ErrConfiguracionDocumentosActivacion = errors.New("config: Documentos fuera del perfil de desarrollo")
)

// DocumentosDesarrolloActivo valida la activación sin abrir ficheros ni
// conexiones. Un selector mal escrito o fuera de la doble llave falla cerrado.
func (c Config) DocumentosDesarrolloActivo() (bool, error) {
	c = c.Normalize()
	switch c.DocumentosEnabled {
	case "", "false":
		return false, nil
	case "true":
		if !c.DevelopmentEnabledByDoubleKey() {
			return false, ErrConfiguracionDocumentosActivacion
		}
		return true, nil
	default:
		return false, ErrConfiguracionDocumentosSelector
	}
}
