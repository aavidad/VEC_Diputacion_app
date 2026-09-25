package domain

import (
	"errors"
	"time"
)

var ErrSolicitudPermisoInvalida = errors.New("cronos solicitud de permiso invalida")
var ErrTransicionPermisoInvalida = errors.New("cronos transicion de permiso invalida")

type EstadoSolicitudPermiso string

const (
	EstadoPermisoSolicitado              EstadoSolicitudPermiso = "solicitado"
	EstadoPermisoPendienteAdministracion EstadoSolicitudPermiso = "pendiente_administracion"
	EstadoPermisoConcedido               EstadoSolicitudPermiso = "concedido"
	EstadoPermisoDenegado                EstadoSolicitudPermiso = "denegado"
	EstadoPermisoCancelado               EstadoSolicitudPermiso = "cancelado"
)

type PasoPermiso string

const (
	PasoResponsable    PasoPermiso = "responsable"
	PasoAdministracion PasoPermiso = "administracion"
)

type DecisionPermiso string

const (
	DecisionAprobar DecisionPermiso = "aprobar"
	DecisionDenegar DecisionPermiso = "denegar"
)

// Fechas son civiles ISO y el fin es inclusivo. Cantidad procede de la
// autoridad de calendario/computo, nunca de una cifra enviada por el cliente.
type SolicitudPermiso struct {
	Referencia, EmpleadoRef, CatalogoVersionRef, PermisoRef string
	Desde, Hasta                                            string
	Cantidad                                                int64
	Unidad                                                  LeaveUnit
	Estado                                                  EstadoSolicitudPermiso
	Version                                                 int64
	ClaveOperacion, HuellaMaterial, ReciboRef               string
	CreadaUTC                                               time.Time
}

func (s SolicitudPermiso) Validar(c CatalogoPermisoVersion) error {
	if c.Validar() != nil || !referenciaMarcaje(s.Referencia) || !referenciaMarcaje(s.EmpleadoRef) || s.CatalogoVersionRef != c.VersionRef || s.PermisoRef != c.PermisoRef || s.Unidad != c.Unidad || s.Cantidad < c.Minimo || !claveOperacionMarcaje(s.ClaveOperacion) || !referenciaMarcaje(s.ReciboRef) || len(s.HuellaMaterial) != 64 || s.Version < 1 || s.CreadaUTC.IsZero() || s.CreadaUTC.Location() != time.UTC || s.CreadaUTC.Nanosecond()%1000 != 0 {
		return ErrSolicitudPermisoInvalida
	}
	if _, err := fechaPermiso(s.Desde); err != nil {
		return err
	}
	if hasta, err := fechaPermiso(s.Hasta); err != nil {
		return err
	} else if desde, _ := fechaPermiso(s.Desde); hasta.Before(desde) {
		return ErrSolicitudPermisoInvalida
	}
	if c.MaximoSolicitud != nil && s.Cantidad > *c.MaximoSolicitud {
		return ErrSolicitudPermisoInvalida
	}
	switch s.Estado {
	case EstadoPermisoSolicitado, EstadoPermisoPendienteAdministracion, EstadoPermisoConcedido, EstadoPermisoDenegado, EstadoPermisoCancelado:
		return nil
	default:
		return ErrSolicitudPermisoInvalida
	}
}

func fechaPermiso(valor string) (time.Time, error) {
	if len(valor) != 10 {
		return time.Time{}, ErrSolicitudPermisoInvalida
	}
	f, err := time.Parse("2006-01-02", valor)
	if err != nil || f.Format("2006-01-02") != valor {
		return time.Time{}, ErrSolicitudPermisoInvalida
	}
	return f, nil
}

// SiguienteEstadoPermiso mantiene distintos los hechos de jefatura y RRHH.
// La decisión final de administración nunca se infiere de la primera.
func SiguienteEstadoPermiso(c CircuitoPermiso, actual EstadoSolicitudPermiso, paso PasoPermiso, decision DecisionPermiso) (EstadoSolicitudPermiso, error) {
	if decision != DecisionAprobar && decision != DecisionDenegar {
		return "", ErrTransicionPermisoInvalida
	}
	if c == CircuitoResponsableAdministracion && actual == EstadoPermisoSolicitado && paso == PasoResponsable {
		if decision == DecisionDenegar {
			return EstadoPermisoDenegado, nil
		}
		return EstadoPermisoPendienteAdministracion, nil
	}
	if ((c == CircuitoResponsableAdministracion && actual == EstadoPermisoPendienteAdministracion) || (c == CircuitoAdministracion && actual == EstadoPermisoSolicitado)) && paso == PasoAdministracion {
		if decision == DecisionDenegar {
			return EstadoPermisoDenegado, nil
		}
		return EstadoPermisoConcedido, nil
	}
	return "", ErrTransicionPermisoInvalida
}
