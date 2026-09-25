package domain

import (
	"errors"
	"strings"
	"time"
)

var ErrCatalogoPermisoInvalido = errors.New("cronos catalogo de permisos invalido")

type ComputoPermiso string
type CircuitoPermiso string

const (
	ComputoLaborables                 ComputoPermiso  = "laborables"
	ComputoNaturales                  ComputoPermiso  = "naturales"
	CircuitoAdministracion            CircuitoPermiso = "A"
	CircuitoResponsableAdministracion CircuitoPermiso = "J-A"
)

// Cantidades de horas se expresan en minutos enteros; dias, en dias enteros.
// nil significa que la fuente no fija ese limite; cero no es un limite valido.
type CatalogoPermisoVersion struct {
	PermisoRef, VersionRef, FuenteRef, Nombre   string
	VigenteDesde, VigenteHasta                  time.Time // UTC; hasta exclusivo, cero = sin fin conocido
	Unidad                                      LeaveUnit
	Computo                                     ComputoPermiso
	Circuito                                    CircuitoPermiso
	Minimo                                      int64
	MaximoSolicitud, MaximoMensual, MaximoAnual *int64
	JustificanteExigido                         bool
}

func (c CatalogoPermisoVersion) Validar() error {
	if !referenciaMarcaje(c.PermisoRef) || !referenciaMarcaje(c.VersionRef) || !referenciaMarcaje(c.FuenteRef) || strings.TrimSpace(c.Nombre) == "" || c.VigenteDesde.IsZero() || c.VigenteDesde.Location() != time.UTC || c.VigenteDesde.Nanosecond()%1000 != 0 || (!c.VigenteHasta.IsZero() && (!c.VigenteHasta.After(c.VigenteDesde) || c.VigenteHasta.Location() != time.UTC || c.VigenteHasta.Nanosecond()%1000 != 0)) {
		return ErrCatalogoPermisoInvalido
	}
	if c.Unidad != LeaveUnitDay && c.Unidad != LeaveUnitHour || c.Computo != ComputoLaborables && c.Computo != ComputoNaturales || c.Circuito != CircuitoAdministracion && c.Circuito != CircuitoResponsableAdministracion || c.Minimo <= 0 {
		return ErrCatalogoPermisoInvalido
	}
	for _, max := range []*int64{c.MaximoSolicitud, c.MaximoMensual, c.MaximoAnual} {
		if max != nil && (*max <= 0 || *max < c.Minimo) {
			return ErrCatalogoPermisoInvalido
		}
	}
	return nil
}

func (c CatalogoPermisoVersion) VigenteEn(instante time.Time) bool {
	return c.Validar() == nil && !instante.Before(c.VigenteDesde) && (c.VigenteHasta.IsZero() || instante.Before(c.VigenteHasta))
}

// HechoResumenPermiso representa un hecho durable, no una cifra enviada por UI.
// Solicitado solo cuenta solicitudes aun pendientes. Concedido incluye las
// pendientes de justificar, que son un subconjunto, no un tercer consumo.
type HechoResumenPermiso struct {
	SolicitudRef        string
	CatalogoVersionRef  string
	Unidad              LeaveUnit
	Cantidad            int64
	Estado              EstadoSolicitudPermiso
	PendienteJustificar bool
}

type ResumenAnualPermiso struct {
	Anio                                       int
	PermisoRef, CatalogoVersionRef             string
	Unidad                                     LeaveUnit
	Solicitado, Concedido, PendienteJustificar int64
	Resta                                      *int64 // nil: sin maximo anual o sin conciliar
	// SinConciliar marca solo esta fila cuando la historia no cuadra con la
	// version vigente: hechos de otra unidad (no se suman) o un consumo que
	// supera un maximo anual rebajado. La resta queda entonces ausente.
	SinConciliar bool
}

func ProyectarPermisoAnual(c CatalogoPermisoVersion, anio int, hechos []HechoResumenPermiso) (ResumenAnualPermiso, error) {
	if c.Validar() != nil || anio < 1 || anio > 9999 {
		return ResumenAnualPermiso{}, ErrCatalogoPermisoInvalido
	}
	r := ResumenAnualPermiso{Anio: anio, PermisoRef: c.PermisoRef, CatalogoVersionRef: c.VersionRef, Unidad: c.Unidad}
	vistos := make(map[string]bool, len(hechos))
	for _, h := range hechos {
		if !referenciaMarcaje(h.SolicitudRef) || !referenciaMarcaje(h.CatalogoVersionRef) || (h.Unidad != LeaveUnitDay && h.Unidad != LeaveUnitHour) || vistos[h.SolicitudRef] || h.Cantidad <= 0 {
			return ResumenAnualPermiso{}, ErrCatalogoPermisoInvalido
		}
		vistos[h.SolicitudRef] = true
		if h.Estado != EstadoPermisoConcedido && h.PendienteJustificar {
			return ResumenAnualPermiso{}, ErrCatalogoPermisoInvalido
		}
		if h.Unidad != c.Unidad {
			// Otra version del catalogo con otra unidad: minutos y dias no se
			// suman; la fila se marca y el resto de la consulta sigue.
			switch h.Estado {
			case EstadoPermisoSolicitado, EstadoPermisoPendienteAdministracion, EstadoPermisoConcedido, EstadoPermisoDenegado, EstadoPermisoCancelado:
			default:
				return ResumenAnualPermiso{}, ErrCatalogoPermisoInvalido
			}
			r.SinConciliar = true
			continue
		}
		switch h.Estado {
		case EstadoPermisoSolicitado, EstadoPermisoPendienteAdministracion:
			r.Solicitado += h.Cantidad
		case EstadoPermisoConcedido:
			r.Concedido += h.Cantidad
			if h.PendienteJustificar {
				r.PendienteJustificar += h.Cantidad
			}
		case EstadoPermisoDenegado, EstadoPermisoCancelado:
		default:
			return ResumenAnualPermiso{}, ErrCatalogoPermisoInvalido
		}
	}
	if c.MaximoAnual != nil && !r.SinConciliar {
		resta := *c.MaximoAnual - r.Solicitado - r.Concedido
		if resta < 0 {
			r.SinConciliar = true
		} else {
			r.Resta = &resta
		}
	}
	return r, nil
}
