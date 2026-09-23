package domain

import (
	"errors"
	"regexp"
	"time"
)

// Este cálculo representa un máximo orientativo para un borrador sintético.
// El alojamiento exige gasto justificado y la norma aplicable a la Diputación
// sigue pendiente de RRHH. Ningún tramo constituye una liquidación.
const RotuloTarifaProvisional = "PROVISIONAL · pendiente de confirmación por RRHH"

var ErrTramosProvisionalesNoDisponibles = errors.New("dietas: cálculo provisional no disponible")
var versionTarifaProvisional = regexp.MustCompile(`^provisional:[a-z0-9:-]{8,120}$`)

type TarifaNacionalProvisional struct {
	VersionRef              string
	Rotulo                  string
	PaisISO2                string
	Grupo                   int
	VigenteDesde            string
	VigenteHasta            string
	ManutencionCentimos     int64
	AlojamientoTopeCentimos int64
}

type TramoDietaProvisional struct {
	Fecha            string `json:"fecha"`
	Tipo             string `json:"tipo"`
	Porcentaje       int    `json:"porcentaje"`
	ImporteCentimos  int64  `json:"importe_centimos"`
	VersionTarifaRef string `json:"version_tarifa_ref"`
	Rotulo           string `json:"rotulo"`
}

type CalculoDietasProvisional struct {
	Tramos                  []TramoDietaProvisional `json:"tramos"`
	ManutencionCentimos     int64                   `json:"manutencion_centimos"`
	AlojamientoTopeCentimos int64                   `json:"alojamiento_tope_centimos"`
	TotalMaximoOrientativo  int64                   `json:"total_maximo_orientativo_centimos"`
	VersionTarifaRef        string                  `json:"version_tarifa_ref"`
	Rotulo                  string                  `json:"rotulo"`
}

// CalcularTramosNacionalesProvisionales aplica solo el caso ordinario nacional
// del artículo 12 del RD 462/2002 como referencia de ensayo. Excluye cena
// extraordinaria, extranjero y supuestos especiales; todos quedan pendientes
// de decisión expresa, nunca se completan con un cálculo por defecto.
func CalcularTramosNacionalesProvisionales(inicio, fin time.Time, zona *time.Location, tarifa TarifaNacionalProvisional) (CalculoDietasProvisional, error) {
	var vacio CalculoDietasProvisional
	if zona == nil || zona.String() != "Europe/Madrid" || inicio.IsZero() || fin.IsZero() || !fin.After(inicio) ||
		!versionTarifaProvisional.MatchString(tarifa.VersionRef) || tarifa.Rotulo != RotuloTarifaProvisional ||
		tarifa.PaisISO2 != "ES" || tarifa.Grupo < 1 || tarifa.Grupo > 3 ||
		tarifa.ManutencionCentimos < 1 || tarifa.AlojamientoTopeCentimos < 1 ||
		tarifa.ManutencionCentimos > 1000000 || tarifa.AlojamientoTopeCentimos > 1000000 {
		return vacio, ErrTramosProvisionalesNoDisponibles
	}
	desde, err := time.Parse("2006-01-02", tarifa.VigenteDesde)
	if err != nil || desde.Format("2006-01-02") != tarifa.VigenteDesde {
		return vacio, ErrTramosProvisionalesNoDisponibles
	}
	if tarifa.VigenteHasta != "" {
		hasta, err := time.Parse("2006-01-02", tarifa.VigenteHasta)
		if err != nil || hasta.Format("2006-01-02") != tarifa.VigenteHasta || !hasta.After(desde) {
			return vacio, ErrTramosProvisionalesNoDisponibles
		}
	}
	i, f := inicio.In(zona), fin.In(zona)
	fechaInicio, fechaFin := i.Format("2006-01-02"), f.Format("2006-01-02")
	if fechaInicio < tarifa.VigenteDesde || (tarifa.VigenteHasta != "" && fechaFin >= tarifa.VigenteHasta) {
		return vacio, ErrTramosProvisionalesNoDisponibles
	}
	resultado := CalculoDietasProvisional{Tramos: []TramoDietaProvisional{}, VersionTarifaRef: tarifa.VersionRef, Rotulo: tarifa.Rotulo}
	// Comparar fechas civiles evita que el cambio de hora convierta una noche
	// en dos días o elimine un día del itinerario.
	dia := time.Date(i.Year(), i.Month(), i.Day(), 12, 0, 0, 0, zona)
	ultimo := time.Date(f.Year(), f.Month(), f.Day(), 12, 0, 0, 0, zona)
	for n := 0; !dia.After(ultimo); n, dia = n+1, dia.AddDate(0, 0, 1) {
		if n >= 31 {
			return vacio, ErrTramosProvisionalesNoDisponibles
		}
		fecha := dia.Format("2006-01-02")
		porcentaje := 0
		switch {
		case fechaInicio == fechaFin:
			if fin.Sub(inicio) >= 5*time.Hour && horaAnterior(i, 14) && horaPosterior(f, 16) {
				porcentaje = 50
			}
		case fecha == fechaInicio:
			if horaAnterior(i, 14) {
				porcentaje = 100
			} else if horaPosterior(i, 14) && horaAnterior(i, 22) {
				porcentaje = 50
			}
		case fecha == fechaFin:
			if horaPosterior(f, 14) {
				porcentaje = 50
			}
		default:
			porcentaje = 100
		}
		if porcentaje > 0 {
			importe := tarifa.ManutencionCentimos * int64(porcentaje) / 100
			if porcentaje == 50 {
				importe = (tarifa.ManutencionCentimos + 1) / 2
			}
			resultado.Tramos = append(resultado.Tramos, tramoProvisional(fecha, "manutencion", porcentaje, importe, tarifa))
			resultado.ManutencionCentimos += importe
		}
		if fecha != fechaFin {
			resultado.Tramos = append(resultado.Tramos, tramoProvisional(fecha, "alojamiento_tope_pendiente_justificante", 100, tarifa.AlojamientoTopeCentimos, tarifa))
			resultado.AlojamientoTopeCentimos += tarifa.AlojamientoTopeCentimos
		}
	}
	resultado.TotalMaximoOrientativo = resultado.ManutencionCentimos + resultado.AlojamientoTopeCentimos
	return resultado, nil
}

func tramoProvisional(fecha, tipo string, porcentaje int, importe int64, tarifa TarifaNacionalProvisional) TramoDietaProvisional {
	return TramoDietaProvisional{fecha, tipo, porcentaje, importe, tarifa.VersionRef, tarifa.Rotulo}
}

func horaAnterior(t time.Time, hora int) bool { return t.Hour() < hora }
func horaPosterior(t time.Time, hora int) bool {
	return t.Hour() > hora || (t.Hour() == hora && (t.Minute() > 0 || t.Second() > 0 || t.Nanosecond() > 0))
}
