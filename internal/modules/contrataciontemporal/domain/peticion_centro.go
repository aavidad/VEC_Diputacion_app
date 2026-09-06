package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

const (
	estadoPeticionCentroPendiente  = "pendiente_ratificacion"
	estadoPeticionCentroRatificada = "ratificada"
)

var (
	ErrPeticionCentroInvalida           = errors.New("contratacion temporal: peticion de centro invalida")
	ErrRatificacionCentroDenegada       = errors.New("contratacion temporal: ratificacion de centro denegada")
	ErrVersionPeticionCentroEnConflicto = errors.New("contratacion temporal: version de peticion de centro en conflicto")
)

type ActorPeticionCentro struct {
	ActorRef  string `json:"actor_ref"`
	PerfilRef string `json:"perfil_ref"`
	CentroRef string `json:"centro_ref"`
	PuestoRef string `json:"puesto_ref"`
}

func (a ActorPeticionCentro) Validar() error {
	if !referenciaValida(a.ActorRef) || !referenciaValida(a.PerfilRef) ||
		!referenciaValida(a.CentroRef) || !referenciaValida(a.PuestoRef) {
		return ErrPeticionCentroInvalida
	}
	return nil
}

type ConfiguracionPeticionCentro struct {
	Referencia  string              `json:"referencia"`
	Version     uint64              `json:"version"`
	Solicitante ActorPeticionCentro `json:"solicitante"`
	Ratificador ActorPeticionCentro `json:"ratificador"`
}

func (c ConfiguracionPeticionCentro) Validar() error {
	if !referenciaValida(c.Referencia) || c.Version == 0 ||
		c.Solicitante.Validar() != nil || c.Ratificador.Validar() != nil ||
		c.Solicitante.ActorRef == c.Ratificador.ActorRef ||
		c.Solicitante.CentroRef != c.Ratificador.CentroRef {
		return ErrPeticionCentroInvalida
	}
	return nil
}

type DatosPeticionCentro struct {
	Referencia         string                      `json:"referencia"`
	Version            uint64                      `json:"version"`
	Configuracion      ConfiguracionPeticionCentro `json:"configuracion"`
	Solicitud          SolicitudCentro             `json:"solicitud"`
	Estado             string                      `json:"estado"`
	CreadaEn           time.Time                   `json:"creada_en"`
	RatificadaEn       time.Time                   `json:"ratificada_en,omitzero"`
	MotivoRatificacion string                      `json:"motivo_ratificacion,omitempty"`
}

type PeticionCentro struct{ datos DatosPeticionCentro }

func NuevaPeticionCentro(
	referencia string,
	config ConfiguracionPeticionCentro,
	solicitud SolicitudCentro,
	ahora time.Time,
) (PeticionCentro, error) {
	datos := DatosPeticionCentro{
		Referencia: referencia, Version: 1, Configuracion: config,
		Solicitud: solicitud, Estado: estadoPeticionCentroPendiente, CreadaEn: ahora,
	}
	return RehidratarPeticionCentro(datos)
}

func RehidratarPeticionCentro(datos DatosPeticionCentro) (PeticionCentro, error) {
	solicitud, err := datos.Solicitud.Clonar()
	if err != nil || !referenciaValida(datos.Referencia) ||
		datos.Configuracion.Validar() != nil || !instanteCanonico(datos.CreadaEn) ||
		datos.Solicitud.CentroRef != datos.Configuracion.Solicitante.CentroRef {
		return PeticionCentro{}, ErrPeticionCentroInvalida
	}
	datos.Solicitud = solicitud
	switch datos.Estado {
	case estadoPeticionCentroPendiente:
		if datos.Version != 1 || datos.RatificadaEn != (time.Time{}) ||
			datos.MotivoRatificacion != "" {
			return PeticionCentro{}, ErrPeticionCentroInvalida
		}
	case estadoPeticionCentroRatificada:
		if datos.Version != 2 || !instanteCanonico(datos.RatificadaEn) ||
			datos.RatificadaEn.Before(datos.CreadaEn) ||
			!motivoRatificacionCentroValido(datos.MotivoRatificacion) {
			return PeticionCentro{}, ErrPeticionCentroInvalida
		}
	default:
		return PeticionCentro{}, ErrPeticionCentroInvalida
	}
	return PeticionCentro{datos: datos}, nil
}

func (p PeticionCentro) Datos() DatosPeticionCentro {
	datos := p.datos
	datos.Solicitud = datos.Solicitud.clonar()
	return datos
}

func (p PeticionCentro) Ratificar(
	actor ActorPeticionCentro,
	versionEsperada uint64,
	motivo string,
	ahora time.Time,
) (PeticionCentro, error) {
	actual, err := RehidratarPeticionCentro(p.Datos())
	if err != nil {
		return PeticionCentro{}, ErrPeticionCentroInvalida
	}
	if versionEsperada != actual.datos.Version {
		return PeticionCentro{}, ErrVersionPeticionCentroEnConflicto
	}
	if actual.datos.Estado != estadoPeticionCentroPendiente ||
		actor != actual.datos.Configuracion.Ratificador ||
		!motivoRatificacionCentroValido(motivo) || !instanteCanonico(ahora) ||
		ahora.Before(actual.datos.CreadaEn) {
		return PeticionCentro{}, ErrRatificacionCentroDenegada
	}
	siguiente := actual.Datos()
	siguiente.Version = 2
	siguiente.Estado = estadoPeticionCentroRatificada
	siguiente.RatificadaEn = ahora
	siguiente.MotivoRatificacion = motivo
	return RehidratarPeticionCentro(siguiente)
}

func motivoRatificacionCentroValido(motivo string) bool {
	return len(motivo) <= 1000 && textoValido(motivo, 1000, false) &&
		strings.IndexFunc(motivo, unicode.IsControl) < 0
}
