package domain

import (
	"errors"
	"strings"
)

var ErrDatosIntentoAuditoriaInvalidos = errors.New("vec: datos de intento de auditoria invalidos")

// ResultadoIntentoAuditoria describe la respuesta observada al terminar una
// operación fallida. No representa una decisión de autorización ni un consumo.
type ResultadoIntentoAuditoria string

const (
	ResultadoIntentoAuditoriaDenegado ResultadoIntentoAuditoria = "denegado"
	ResultadoIntentoAuditoriaError    ResultadoIntentoAuditoria = "error"
)

// DatosIntentoAuditoria contiene solo referencias y claves de catálogo. El
// origen procede de configuración del proceso, nunca del cuerpo o cabeceras HTTP.
type DatosIntentoAuditoria struct {
	Accion         string
	ModuloID       string
	RecursoRef     string
	FinalidadRef   string
	Resultado      ResultadoIntentoAuditoria
	Motivo         ReferenciaEntradaCatalogo
	Proceso        string
	Canal          string
	CorrelacionRef string
}

func (d DatosIntentoAuditoria) Validar() error {
	if !claveIntentoAuditoriaValida(d.Accion, 128) ||
		!claveIntentoAuditoriaValida(d.ModuloID, 128) ||
		!referenciaRecursoIntentoAuditoriaValida(d.RecursoRef) ||
		!claveIntentoAuditoriaValida(d.FinalidadRef, 128) ||
		(d.Resultado != ResultadoIntentoAuditoriaDenegado &&
			d.Resultado != ResultadoIntentoAuditoriaError) ||
		d.Motivo.Validar() != nil || !claveIntentoAuditoriaValida(d.Motivo.Referencia(), 160) ||
		!procesoIntentoAuditoriaValido(d.Proceso) ||
		!claveIntentoAuditoriaValida(d.Canal, 128) ||
		!ReferenciaCorrelacionAutorizacionV2Valida(d.CorrelacionRef) {
		return ErrDatosIntentoAuditoriaInvalidos
	}
	return nil
}

func claveIntentoAuditoriaValida(valor string, limite int) bool {
	if valor == "" || len(valor) > limite || valor[0] < 'a' || valor[0] > 'z' ||
		strings.Contains(valor, "..") {
		return false
	}
	for _, r := range valor {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') &&
			r != '_' && r != '-' && r != '.' && r != ':' {
			return false
		}
	}
	return true
}

// La referencia puede incluir el prefijo del módulo, pero no rutas, correos,
// consultas, espacios ni texto libre susceptible de contener datos personales.
func referenciaRecursoIntentoAuditoriaValida(valor string) bool {
	if valor == "" || len(valor) > 200 || strings.Contains(valor, "..") ||
		strings.Contains(valor, "@") || strings.HasPrefix(valor, "http") {
		return false
	}
	primero := valor[0]
	if (primero < 'a' || primero > 'z') && (primero < '0' || primero > '9') {
		return false
	}
	for _, r := range valor[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') &&
			r != '_' && r != '-' && r != '.' && r != ':' {
			return false
		}
	}
	return strings.ContainsAny(valor, ":_")
}

func procesoIntentoAuditoriaValido(valor string) bool {
	if len(valor) < 2 || len(valor) > 80 || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for _, r := range valor[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') &&
			r != '_' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
