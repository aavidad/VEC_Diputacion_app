package ports

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

const SuperficieAuditoriaFronteraRutaExactaContratacionTemporal = "api.contratacion_temporal.ruta_exacta"

var ErrOrdenAuditoriaFronteraRutaExactaInvalida = errors.New(
	"vec ports: orden de auditoria de frontera de ruta exacta invalida",
)

// MotivoAuditoriaFronteraRutaExacta es cerrado para no trasladar errores ni
// detalles de autenticación a la bitácora de frontera.
type MotivoAuditoriaFronteraRutaExacta string

const (
	MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida MotivoAuditoriaFronteraRutaExacta = "autenticacion_requerida"
	MotivoAuditoriaFronteraRutaExactaAccesoDenegado         MotivoAuditoriaFronteraRutaExacta = "acceso_denegado"
)

// OrdenAuditoriaFronteraRutaExacta conserva solo los datos minimizados de una
// denegación temprana. Ruta no contiene consulta y ActorRef solo puede llegar
// de una autoridad que ya haya verificado la identidad.
type OrdenAuditoriaFronteraRutaExacta struct {
	CorrelacionRef string
	Motivo         MotivoAuditoriaFronteraRutaExacta
	Superficie     string
	Ruta           string
	ActorRef       string
}

func (o OrdenAuditoriaFronteraRutaExacta) Validar() error {
	if !correlacionAuditoriaFronteraRutaExactaValida(o.CorrelacionRef) ||
		(o.Motivo != MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida &&
			o.Motivo != MotivoAuditoriaFronteraRutaExactaAccesoDenegado) ||
		o.Superficie != SuperficieAuditoriaFronteraRutaExactaContratacionTemporal ||
		!rutaAuditoriaFronteraRutaExactaValida(o.Ruta) ||
		(o.ActorRef != "" && !referenciaActorAuditoriaFronteraRutaExactaValida(o.ActorRef)) {
		return ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	return nil
}

func correlacionAuditoriaFronteraRutaExactaValida(valor string) bool {
	if valor == "corr_no_disponible" {
		return true
	}
	if len(valor) != len("corr_")+32 || !strings.HasPrefix(valor, "corr_") {
		return false
	}
	for _, caracter := range valor[len("corr_"):] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func rutaAuditoriaFronteraRutaExactaValida(ruta string) bool {
	if len(ruta) <= len("/api/vec/contratacion-temporal/") || len(ruta) > 512 ||
		!strings.HasPrefix(ruta, "/api/vec/contratacion-temporal/") ||
		strings.HasSuffix(ruta, "/") || strings.ContainsAny(ruta, "?#%\\") {
		return false
	}
	for _, segmento := range strings.Split(strings.TrimPrefix(ruta, "/api/vec/"), "/") {
		if segmento == "" || len(segmento) > 64 {
			return false
		}
		for _, caracter := range segmento {
			if (caracter < 'a' || caracter > 'z') &&
				(caracter < '0' || caracter > '9') && caracter != '-' && caracter != '_' {
				return false
			}
		}
	}
	return true
}

func referenciaActorAuditoriaFronteraRutaExactaValida(actor string) bool {
	if actor == "" || len(actor) > 512 || actor != strings.TrimSpace(actor) || !utf8.ValidString(actor) {
		return false
	}
	for _, caracter := range actor {
		if (caracter < 'a' || caracter > 'z') &&
			(caracter < 'A' || caracter > 'Z') &&
			(caracter < '0' || caracter > '9') &&
			caracter != ':' && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}

// RegistradorAuditoriaFronteraRutaExacta es una dependencia inyectada de la
// frontera HTTP. Su error nunca altera la denegación ya decidida.
type RegistradorAuditoriaFronteraRutaExacta interface {
	RegistrarAuditoriaFronteraRutaExacta(context.Context, OrdenAuditoriaFronteraRutaExacta) error
}
