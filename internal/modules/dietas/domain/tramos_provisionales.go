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
var referenciaReglaProvisional = regexp.MustCompile(`^provisional:regla:[a-z0-9:-]{8,120}$`)
var huellaReglaProvisional = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ConfiguracionDevengoProvisional es un esquema cerrado de datos. Ningún
// campo contiene expresiones, SQL o código ejecutable.
type ConfiguracionDevengoProvisional struct {
	Regla                        string `json:"regla"`
	Zona                         string `json:"zona"`
	DuracionMinimaMismoDiaHoras  int    `json:"duracion_minima_mismo_dia_horas"`
	HoraSalida100AntesDe         int    `json:"hora_salida_100_antes_de"`
	HoraSalida50AntesDe          int    `json:"hora_salida_50_antes_de"`
	HoraRegreso50DespuesDe       int    `json:"hora_regreso_50_despues_de"`
	HoraRegresoMismoDiaDespuesDe int    `json:"hora_regreso_mismo_dia_despues_de"`
	DiasMaximos                  int    `json:"dias_maximos"`
	Alojamiento                  string `json:"alojamiento"`
	Liquidable                   bool   `json:"liquidable"`
	PorcentajeMismoDia           int    `json:"porcentaje_mismo_dia"`
	PorcentajeSalidaTemprana     int    `json:"porcentaje_salida_temprana"`
	PorcentajeSalidaMedia        int    `json:"porcentaje_salida_media"`
	PorcentajeRegreso            int    `json:"porcentaje_regreso"`
	PorcentajeIntermedio         int    `json:"porcentaje_intermedio"`
	PorcentajeAlojamientoTope    int    `json:"porcentaje_alojamiento_tope"`
}

type ReglaDevengoProvisional struct {
	ReglaRef         string                          `json:"regla_ref"`
	VersionTarifaRef string                          `json:"version_tarifa_ref"`
	PaisISO2         string                          `json:"pais_iso2"`
	Variante         string                          `json:"variante"`
	Configuracion    ConfiguracionDevengoProvisional `json:"configuracion"`
	HuellaSHA256     string                          `json:"huella_sha256"`
	VigenteDesde     string                          `json:"vigente_desde,omitempty"`
	VigenteHasta     string                          `json:"vigente_hasta,omitempty"`
}

func (r ReglaDevengoProvisional) Validar() error {
	c := r.Configuracion
	if !referenciaReglaProvisional.MatchString(r.ReglaRef) || !versionTarifaProvisional.MatchString(r.VersionTarifaRef) || r.PaisISO2 != "ES" || r.Variante != "nacional_ordinaria" || !huellaReglaProvisional.MatchString(r.HuellaSHA256) ||
		c.Regla != "nacional_ordinaria_provisional_v1" || c.Zona != "Europe/Madrid" || c.Alojamiento != "tope_pendiente_justificante" || c.Liquidable ||
		c.DuracionMinimaMismoDiaHoras < 1 || c.DuracionMinimaMismoDiaHoras > 24 || c.DiasMaximos < 1 || c.DiasMaximos > 31 ||
		!horaCatalogo(c.HoraSalida100AntesDe) || !horaCatalogo(c.HoraSalida50AntesDe) || !horaCatalogo(c.HoraRegreso50DespuesDe) || !horaCatalogo(c.HoraRegresoMismoDiaDespuesDe) ||
		c.HoraSalida100AntesDe >= c.HoraSalida50AntesDe || c.HoraRegreso50DespuesDe >= c.HoraRegresoMismoDiaDespuesDe ||
		!porcentajeCatalogo(c.PorcentajeMismoDia) || !porcentajeCatalogo(c.PorcentajeSalidaTemprana) || !porcentajeCatalogo(c.PorcentajeSalidaMedia) || !porcentajeCatalogo(c.PorcentajeRegreso) || !porcentajeCatalogo(c.PorcentajeIntermedio) || !porcentajeCatalogo(c.PorcentajeAlojamientoTope) {
		return ErrTramosProvisionalesNoDisponibles
	}
	return nil
}

func VersionTarifaProvisionalValida(version string) bool {
	return versionTarifaProvisional.MatchString(version)
}

func horaCatalogo(v int) bool       { return v >= 0 && v <= 23 }
func porcentajeCatalogo(v int) bool { return v >= 0 && v <= 100 }

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
	ReglaRef                string                  `json:"regla_ref,omitempty"`
	ReglaHuellaSHA256       string                  `json:"regla_huella_sha256,omitempty"`
}

// CalcularTramosNacionalesProvisionales aplica solo el caso ordinario nacional
// del artículo 12 del RD 462/2002 como referencia de ensayo. Excluye cena
// extraordinaria, extranjero y supuestos especiales; todos quedan pendientes
// de decisión expresa, nunca se completan con un cálculo por defecto.
func CalcularTramosNacionalesProvisionales(inicio, fin time.Time, zona *time.Location, tarifa TarifaNacionalProvisional, regla ReglaDevengoProvisional) (CalculoDietasProvisional, error) {
	var vacio CalculoDietasProvisional
	if zona == nil || zona.String() != "Europe/Madrid" || inicio.IsZero() || fin.IsZero() || !fin.After(inicio) ||
		regla.Validar() != nil || regla.VersionTarifaRef != tarifa.VersionRef || regla.PaisISO2 != tarifa.PaisISO2 || regla.Configuracion.Zona != zona.String() ||
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
	resultado := CalculoDietasProvisional{Tramos: []TramoDietaProvisional{}, VersionTarifaRef: tarifa.VersionRef, Rotulo: tarifa.Rotulo, ReglaRef: regla.ReglaRef, ReglaHuellaSHA256: regla.HuellaSHA256}
	cfg := regla.Configuracion
	// Comparar fechas civiles evita que el cambio de hora convierta una noche
	// en dos días o elimine un día del itinerario.
	dia := time.Date(i.Year(), i.Month(), i.Day(), 12, 0, 0, 0, zona)
	ultimo := time.Date(f.Year(), f.Month(), f.Day(), 12, 0, 0, 0, zona)
	for n := 0; !dia.After(ultimo); n, dia = n+1, dia.AddDate(0, 0, 1) {
		if n >= cfg.DiasMaximos {
			return vacio, ErrTramosProvisionalesNoDisponibles
		}
		fecha := dia.Format("2006-01-02")
		porcentaje := 0
		switch {
		case fechaInicio == fechaFin:
			if fin.Sub(inicio) >= time.Duration(cfg.DuracionMinimaMismoDiaHoras)*time.Hour && horaAnterior(i, cfg.HoraSalida100AntesDe) && horaPosterior(f, cfg.HoraRegresoMismoDiaDespuesDe) {
				porcentaje = cfg.PorcentajeMismoDia
			}
		case fecha == fechaInicio:
			if horaAnterior(i, cfg.HoraSalida100AntesDe) {
				porcentaje = cfg.PorcentajeSalidaTemprana
			} else if horaPosterior(i, cfg.HoraSalida100AntesDe) && horaAnterior(i, cfg.HoraSalida50AntesDe) {
				porcentaje = cfg.PorcentajeSalidaMedia
			}
		case fecha == fechaFin:
			if horaPosterior(f, cfg.HoraRegreso50DespuesDe) {
				porcentaje = cfg.PorcentajeRegreso
			}
		default:
			porcentaje = cfg.PorcentajeIntermedio
		}
		if porcentaje > 0 {
			importe := porcentajeCentimos(tarifa.ManutencionCentimos, porcentaje)
			resultado.Tramos = append(resultado.Tramos, tramoProvisional(fecha, "manutencion", porcentaje, importe, tarifa))
			resultado.ManutencionCentimos += importe
		}
		if fecha != fechaFin && cfg.PorcentajeAlojamientoTope > 0 {
			tope := porcentajeCentimos(tarifa.AlojamientoTopeCentimos, cfg.PorcentajeAlojamientoTope)
			resultado.Tramos = append(resultado.Tramos, tramoProvisional(fecha, "alojamiento_tope_pendiente_justificante", cfg.PorcentajeAlojamientoTope, tope, tarifa))
			resultado.AlojamientoTopeCentimos += tope
		}
	}
	resultado.TotalMaximoOrientativo = resultado.ManutencionCentimos + resultado.AlojamientoTopeCentimos
	return resultado, nil
}

func porcentajeCentimos(base int64, porcentaje int) int64 { return (base*int64(porcentaje) + 99) / 100 }

func tramoProvisional(fecha, tipo string, porcentaje int, importe int64, tarifa TarifaNacionalProvisional) TramoDietaProvisional {
	return TramoDietaProvisional{fecha, tipo, porcentaje, importe, tarifa.VersionRef, tarifa.Rotulo}
}

func horaAnterior(t time.Time, hora int) bool { return t.Hour() < hora }
func horaPosterior(t time.Time, hora int) bool {
	return t.Hour() > hora || (t.Hour() == hora && (t.Minute() > 0 || t.Second() > 0 || t.Nanosecond() > 0))
}
