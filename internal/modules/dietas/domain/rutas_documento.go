package domain

import (
	"regexp"
	"strconv"
	"strings"
)

var ajusteRutaDocumento = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,3})\.[0-9]{4}$`)

func ajusteEscalado(s string) (int64, bool) {
	if !ajusteRutaDocumento.MatchString(s) || s == "-0.0000" {
		return 0, false
	}
	negativo := strings.HasPrefix(s, "-")
	if negativo {
		s = s[1:]
	}
	n, err := strconv.ParseInt(s[:len(s)-5]+s[len(s)-4:], 10, 64)
	if err != nil || n > 1000*10000 {
		return 0, false
	}
	if negativo {
		n = -n
	}
	return n, true
}

func ajusteTextoValido(ajuste, motivo string) bool {
	n, ok := ajusteEscalado(ajuste)
	if !ok {
		return false
	}
	if n == 0 {
		return motivo == ""
	}
	return len(motivo) >= 3 && len(motivo) <= 500 && motivo == strings.TrimSpace(motivo) && !strings.ContainsAny(motivo, "\r\n\x00")
}

func (r RutaDeclaradaComision) Validar() error {
	if len(r.CodigosRuta) < 2 || len(r.CodigosRuta) > 12 || !ajusteTextoValido(r.AjusteKilometros, r.MotivoAjuste) {
		return ErrDocumentoComisionInvalido
	}
	for i, codigo := range r.CodigosRuta {
		if !codigoRutaDieta.MatchString(codigo) || (i > 0 && codigo == r.CodigosRuta[i-1]) {
			return ErrDocumentoComisionInvalido
		}
	}
	return nil
}

func (r RutaDeclaradaComision) AjusteEscalado() (int64, error) {
	if r.Validar() != nil {
		return 0, ErrDocumentoComisionInvalido
	}
	n, _ := ajusteEscalado(r.AjusteKilometros)
	return n, nil
}

// ValidarDocumento coteja la instantánea OSRM de cada ruta independiente.
// El grupo y los máximos de dieta continúan siendo alternativas provisionales.
func (c CalculoComision) ValidarDocumento(codigos []string, declaradas []RutaDeclaradaComision, vehiculo bool) error {
	if !versionTarifaProvisional.MatchString(c.VersionTarifa) || c.Rotulo != RotuloTarifaProvisional || !referenciaReglaProvisional.MatchString(c.ReglaRef) || !huellaReglaProvisional.MatchString(c.ReglaHuellaSHA256) ||
		!horaCivil(c.HoraInicio) || !horaCivil(c.HoraFin) || !decimalTarifa.MatchString(c.EURPorKM) ||
		len(codigos) < 2 || len(codigos) > 12 || len(c.TramosRuta) != 0 || c.VehiculoPropio != vehiculo ||
		len(c.Rutas) != len(declaradas) || c.validarOpciones() != nil {
		return ErrCalculoComisionInvalido
	}
	if vehiculo && (c.Procedencia != "osrm_interno" || c.Motor != "OSRM" || c.VersionGrafo == "" || len(c.VersionGrafo) > 160) {
		return ErrCalculoComisionInvalido
	}
	if !vehiculo && (c.Procedencia != "sin_vehiculo_propio" || c.Motor != "no_aplica" || c.VersionGrafo != "no_aplica") {
		return ErrCalculoComisionInvalido
	}
	if !vehiculo && len(declaradas) != 0 {
		return ErrCalculoComisionInvalido
	}
	if vehiculo && (len(declaradas) < 1 || len(declaradas) > 8) {
		return ErrCalculoComisionInvalido
	}
	tarifa, err := decimal4(c.EURPorKM)
	if err != nil {
		return ErrCalculoComisionInvalido
	}
	var totalKM, totalImporte int64
	for i, r := range c.Rutas {
		d := declaradas[i]
		if d.Validar() != nil || len(r.CodigosRuta) != len(d.CodigosRuta) || len(r.TramosRuta) != len(d.CodigosRuta)-1 || r.VersionGrafo == "" || len(r.VersionGrafo) > 160 || r.VersionGrafo != c.VersionGrafo || r.AjusteKilometros != d.AjusteKilometros || r.MotivoAjuste != d.MotivoAjuste {
			return ErrCalculoComisionInvalido
		}
		var base int64
		for j, t := range r.TramosRuta {
			if r.CodigosRuta[j] != d.CodigosRuta[j] || t.OrigenCodigo != d.CodigosRuta[j] || t.DestinoCodigo != d.CodigosRuta[j+1] || t.OrigenCodigo == t.DestinoCodigo {
				return ErrCalculoComisionInvalido
			}
			km, e := decimal4(t.Kilometros)
			if e != nil || km < 1 {
				return ErrCalculoComisionInvalido
			}
			base += km
		}
		if r.CodigosRuta[len(d.CodigosRuta)-1] != d.CodigosRuta[len(d.CodigosRuta)-1] || decimal4ComisionDominio(base) != r.KilometrosBase {
			return ErrCalculoComisionInvalido
		}
		ajuste, _ := ajusteEscalado(d.AjusteKilometros)
		final := base + ajuste
		if final < 1 || final > 10000*10000 || decimal4ComisionDominio(final) != r.KilometrosFinales || r.ImporteCentimos != (final*tarifa+500000)/1000000 {
			return ErrCalculoComisionInvalido
		}
		totalKM += final
		totalImporte += r.ImporteCentimos
	}
	if totalKM > 10000*10000 || c.Kilometros != decimal4ComisionDominio(totalKM) || c.ImporteKilometrajeCentimos != totalImporte {
		return ErrCalculoComisionInvalido
	}
	return nil
}

func decimal4ComisionDominio(n int64) string {
	return strconv.FormatInt(n/10000, 10) + "." + func() string {
		s := strconv.FormatInt(n%10000, 10)
		for len(s) < 4 {
			s = "0" + s
		}
		return s
	}()
}
