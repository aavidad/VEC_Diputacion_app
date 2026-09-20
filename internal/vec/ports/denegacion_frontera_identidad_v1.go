package ports

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrOrdenDenegacionFronteraIdentidadV1Invalida = errors.New(
		"vec: orden de registro de denegacion de frontera de identidad v1 invalida",
	)
	ErrRegistradorDenegacionFronteraIdentidadV1NoDisponible = errors.New(
		"vec: registro de denegacion de frontera de identidad v1 no disponible",
	)
)

// SuperficieDenegacionFronteraIdentidadV1 reutiliza las tres superficies que
// pueden contener una identidad. La superficie publica anonima no pertenece a
// este registro: no puede producir una denegacion asociada a identidad.
type SuperficieDenegacionFronteraIdentidadV1 string

const (
	SuperficieDenegacionFronteraIdentidadV1ExternaPersonal            SuperficieDenegacionFronteraIdentidadV1 = "externa_personal"
	SuperficieDenegacionFronteraIdentidadV1InternaCorporativa         SuperficieDenegacionFronteraIdentidadV1 = "interna_corporativa"
	SuperficieDenegacionFronteraIdentidadV1AdministracionPrivilegiada SuperficieDenegacionFronteraIdentidadV1 = "administracion_privilegiada"
)

func (s SuperficieDenegacionFronteraIdentidadV1) Valida() bool {
	switch s {
	case SuperficieDenegacionFronteraIdentidadV1ExternaPersonal,
		SuperficieDenegacionFronteraIdentidadV1InternaCorporativa,
		SuperficieDenegacionFronteraIdentidadV1AdministracionPrivilegiada:
		return true
	default:
		return false
	}
}

type AccionDenegacionFronteraIdentidadV1 string

const AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias AccionDenegacionFronteraIdentidadV1 = "bolsa.participaciones_propias.consultar"

func (a AccionDenegacionFronteraIdentidadV1) Valida() bool {
	return a == AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias
}

type MotivoDenegacionFronteraIdentidadV1 string

const (
	MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida MotivoDenegacionFronteraIdentidadV1 = "autenticacion_requerida"
	MotivoDenegacionFronteraIdentidadV1AccesoDenegado         MotivoDenegacionFronteraIdentidadV1 = "acceso_denegado"
)

func (m MotivoDenegacionFronteraIdentidadV1) Valida() bool {
	return m == MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida ||
		m == MotivoDenegacionFronteraIdentidadV1AccesoDenegado
}

const RutaExactaDenegacionFronteraIdentidadV1ParticipacionesPropias = "/api/vec/bolsa/mi-bolsa"

// OrdenDenegacionFronteraIdentidadV1 es la evidencia minimizada de
// una denegacion previa a B11. CorrelacionRef procede de un generador servidor
// CSPRNG y no hay instante ni material aportado por el cliente. CanalRef es
// una referencia opaca al canal ya autenticado; ActorRef se reserva para una
// denegacion posterior a identidad, por lo que B11 preautenticado lo exige
// vacio.
type OrdenDenegacionFronteraIdentidadV1 struct {
	CorrelacionRef string
	Superficie     string
	RutaExacta     string
	Accion         string
	Motivo         string
	CanalRef       string
	ActorRef       string
}

func (o OrdenDenegacionFronteraIdentidadV1) Validar() error {
	if !domain.ReferenciaCorrelacionAutorizacionV2Valida(o.CorrelacionRef) ||
		!SuperficieDenegacionFronteraIdentidadV1(o.Superficie).Valida() ||
		o.RutaExacta != RutaExactaDenegacionFronteraIdentidadV1ParticipacionesPropias ||
		!AccionDenegacionFronteraIdentidadV1(o.Accion).Valida() ||
		!MotivoDenegacionFronteraIdentidadV1(o.Motivo).Valida() ||
		(o.CanalRef != "" && !canalRefDenegacionFronteraIdentidadV1Valido(o.CanalRef)) ||
		// La única acción de V1 es B11 preautenticada: no existe actor que
		// pueda atribuirse sin haber resuelto y revalidado identidad.
		o.ActorRef != "" {
		return ErrOrdenDenegacionFronteraIdentidadV1Invalida
	}
	return nil
}

func canalRefDenegacionFronteraIdentidadV1Valido(valor string) bool {
	const prefijo = "tls-exportador:sha256:"
	if len(valor) != len(prefijo)+64 || !strings.HasPrefix(valor, prefijo) ||
		!utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor[len(prefijo):] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

// RegistradorDenegacionFronteraIdentidadV1 es append-only y segregado de la
// autorizacion. Un error al registrarla no concede el acceso; quien decide la
// frontera conserva la denegacion y propaga el fallo para operar con traza.
type RegistradorDenegacionFronteraIdentidadV1 interface {
	RegistrarDenegacionFronteraIdentidadV1(
		context.Context,
		OrdenDenegacionFronteraIdentidadV1,
	) error
}
