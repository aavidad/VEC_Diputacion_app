package ports

import (
	"context"
	"errors"
	"strings"
)

const (
	SuperficieAuditoriaFronteraComision = "api.dietas.comisiones"
	RutaAuditoriaFronteraComision       = "/api/vec/dietas/comisiones"
	RutaAuditoriaFronteraDetalle        = "/api/vec/dietas/comisiones/detalle"
	MotivoFronteraAutenticacion         = "autenticacion_requerida"
	MotivoFronteraAccesoDenegado        = "acceso_denegado"
	MotivoFronteraDependencia           = "dependencia_no_disponible"
	AccionFronteraListar                = "listar"
	AccionFronteraCrear                 = "crear"
	AccionFronteraConsultarDetalle      = "consultar_detalle"
	AccionFronteraMetodoNoAdmitido      = "metodo_no_admitido"
)

var ErrAuditoriaFronteraComisionInvalida = errors.New("dietas: auditoria de frontera de comision invalida")

// La ruta identifica una clase y nunca transporta una referencia de comisión.
// El actor vacío representa el rechazo anterior a una identidad verificada.
type OrdenAuditoriaFronteraComision struct {
	CorrelacionRef string
	Motivo         string
	Ruta           string
	Accion         string
	ActorRef       string
}

func (o OrdenAuditoriaFronteraComision) Validar() error {
	if o.CorrelacionRef != "corr_no_disponible" {
		if len(o.CorrelacionRef) != 37 || !strings.HasPrefix(o.CorrelacionRef, "corr_") || !soloHexMinuscula(o.CorrelacionRef[5:]) {
			return ErrAuditoriaFronteraComisionInvalida
		}
	}
	if (o.Motivo != MotivoFronteraAutenticacion && o.Motivo != MotivoFronteraAccesoDenegado && o.Motivo != MotivoFronteraDependencia) ||
		(o.Ruta != RutaAuditoriaFronteraComision && o.Ruta != RutaAuditoriaFronteraDetalle) ||
		(o.Accion != AccionFronteraListar && o.Accion != AccionFronteraCrear && o.Accion != AccionFronteraConsultarDetalle && o.Accion != AccionFronteraMetodoNoAdmitido) ||
		len(o.ActorRef) > 512 || (o.Motivo == MotivoFronteraAutenticacion && o.ActorRef != "") {
		return ErrAuditoriaFronteraComisionInvalida
	}
	for _, c := range o.ActorRef {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == ':' || c == '-' || c == '_') {
			return ErrAuditoriaFronteraComisionInvalida
		}
	}
	return nil
}

func soloHexMinuscula(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

type RegistradorAuditoriaFronteraComision interface {
	RegistrarAuditoriaFronteraComision(context.Context, OrdenAuditoriaFronteraComision) error
}
