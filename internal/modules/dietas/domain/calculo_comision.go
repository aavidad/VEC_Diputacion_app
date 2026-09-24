package domain

import (
	"errors"
	"regexp"
	"strconv"
)

var ErrCalculoComisionInvalido = errors.New("dietas: cálculo de comisión inválido")
var decimalRuta = regexp.MustCompile(`^(0|[1-9][0-9]{0,4})\.[0-9]{4}$`)
var decimalTarifa = regexp.MustCompile(`^0\.[0-9]{4}$`)

// CalculoComision es una instantánea orientativa calculada en el servidor.
// Las tres opciones de dieta no eligen el grupo de la persona: esa autoridad
// pertenece a Personal y se conectará en el siguiente corte.
type CalculoComision struct {
	Procedencia                string                  `json:"procedencia"`
	VersionGrafo               string                  `json:"version_grafo"`
	Motor                      string                  `json:"motor"`
	VersionTarifa              string                  `json:"version_tarifa"`
	Rotulo                     string                  `json:"rotulo"`
	ReglaRef                   string                  `json:"regla_ref,omitempty"`
	ReglaHuellaSHA256          string                  `json:"regla_huella_sha256,omitempty"`
	HoraInicio                 string                  `json:"hora_inicio"`
	HoraFin                    string                  `json:"hora_fin"`
	Kilometros                 string                  `json:"kilometros"`
	EURPorKM                   string                  `json:"eur_por_km"`
	ImporteKilometrajeCentimos int64                   `json:"importe_kilometraje_centimos"`
	TramosRuta                 []TramoRutaComision     `json:"tramos_ruta"`
	OpcionesDieta              []OpcionDietaComision   `json:"opciones_dieta"`
	VehiculoPropio             bool                    `json:"vehiculo_propio,omitempty"`
	Rutas                      []RutaCalculadaComision `json:"rutas,omitempty"`
}

type RutaDeclaradaComision struct {
	CodigosRuta      []string `json:"codigos_ruta"`
	AjusteKilometros string   `json:"ajuste_kilometros"`
	MotivoAjuste     string   `json:"motivo_ajuste"`
}

type RutaCalculadaComision struct {
	CodigosRuta       []string            `json:"codigos_ruta"`
	VersionGrafo      string              `json:"version_grafo"`
	TramosRuta        []TramoRutaComision `json:"tramos_ruta"`
	KilometrosBase    string              `json:"kilometros_base"`
	AjusteKilometros  string              `json:"ajuste_kilometros"`
	MotivoAjuste      string              `json:"motivo_ajuste"`
	KilometrosFinales string              `json:"kilometros_finales"`
	ImporteCentimos   int64               `json:"importe_centimos"`
}

type TramoRutaComision struct {
	OrigenCodigo  string `json:"origen_codigo"`
	DestinoCodigo string `json:"destino_codigo"`
	Kilometros    string `json:"kilometros"`
}

type OpcionDietaComision struct {
	Grupo   int                      `json:"grupo"`
	Calculo CalculoDietasProvisional `json:"calculo"`
}

type TarifaComisionProvisional struct {
	Dieta      TarifaNacionalProvisional
	EURPorKM   string
	Vehiculo   string
	Referencia string
}

func (c CalculoComision) Validar(codigos []string) error {
	if c.Procedencia != "osrm_interno" || c.VersionGrafo == "" || len(c.VersionGrafo) > 160 || c.Motor != "OSRM" ||
		!versionTarifaProvisional.MatchString(c.VersionTarifa) || c.Rotulo != RotuloTarifaProvisional ||
		!horaCivil(c.HoraInicio) || !horaCivil(c.HoraFin) || !decimalRuta.MatchString(c.Kilometros) ||
		!decimalTarifa.MatchString(c.EURPorKM) || c.ImporteKilometrajeCentimos < 0 ||
		len(c.TramosRuta) != len(codigos)-1 || len(c.TramosRuta) < 1 || len(c.OpcionesDieta) != 3 {
		return ErrCalculoComisionInvalido
	}
	var total int64
	for i, tramo := range c.TramosRuta {
		if tramo.OrigenCodigo != codigos[i] || tramo.DestinoCodigo != codigos[i+1] || !decimalRuta.MatchString(tramo.Kilometros) {
			return ErrCalculoComisionInvalido
		}
		km, err := decimal4(tramo.Kilometros)
		if err != nil || km < 1 {
			return ErrCalculoComisionInvalido
		}
		total += km
	}
	km, err := decimal4(c.Kilometros)
	tarifa, errTarifa := decimal4(c.EURPorKM)
	if err != nil || errTarifa != nil || total != km || c.ImporteKilometrajeCentimos != (km*tarifa+500000)/1000000 {
		return ErrCalculoComisionInvalido
	}
	return c.validarOpciones()
}

func (c CalculoComision) validarOpciones() error {
	if (c.ReglaRef == "") != (c.ReglaHuellaSHA256 == "") {
		return ErrCalculoComisionInvalido
	}
	if c.ReglaRef != "" && (!referenciaReglaProvisional.MatchString(c.ReglaRef) || !huellaReglaProvisional.MatchString(c.ReglaHuellaSHA256)) {
		return ErrCalculoComisionInvalido
	}
	if len(c.OpcionesDieta) != 3 {
		return ErrCalculoComisionInvalido
	}
	for i, opcion := range c.OpcionesDieta {
		if c.ReglaRef != "" && (opcion.Calculo.ReglaRef != c.ReglaRef || opcion.Calculo.ReglaHuellaSHA256 != c.ReglaHuellaSHA256) {
			return ErrCalculoComisionInvalido
		}
		if opcion.Grupo != i+1 || opcion.Calculo.VersionTarifaRef != c.VersionTarifa || opcion.Calculo.Rotulo != c.Rotulo ||
			opcion.Calculo.TotalMaximoOrientativo != opcion.Calculo.ManutencionCentimos+opcion.Calculo.AlojamientoTopeCentimos || len(opcion.Calculo.Tramos) > 62 {
			return ErrCalculoComisionInvalido
		}
		for _, tramo := range opcion.Calculo.Tramos {
			if tramo.VersionTarifaRef != c.VersionTarifa || tramo.Rotulo != c.Rotulo || tramo.ImporteCentimos < 0 {
				return ErrCalculoComisionInvalido
			}
		}
	}
	return nil
}

func decimal4(s string) (int64, error) {
	if !decimalRuta.MatchString(s) && !decimalTarifa.MatchString(s) {
		return 0, ErrCalculoComisionInvalido
	}
	n, err := strconv.ParseInt(s[:len(s)-5]+s[len(s)-4:], 10, 64)
	return n, err
}

func horaCivil(s string) bool {
	return len(s) == 5 && s[2] == ':' && s[:2] >= "00" && s[:2] <= "23" && s[3:] >= "00" && s[3:] <= "59"
}
