package ports

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// El material V1 ya publicado conserva exactamente sus dos fechas. La
// instantánea de la modalidad separa las operaciones posteriores en V2.
func periodoCoberturaV2(periodo domain.PeriodoPrevisto) bool {
	return periodo.PoliticaFin != (domain.PoliticaFin{})
}

func periodoCoberturaValido(periodo domain.PeriodoPrevisto) bool {
	if periodo.Validar() != nil ||
		!instanteFuenteAnalisisCanonico(periodo.Inicio) ||
		(!periodo.Fin.IsZero() &&
			(!instanteFuenteAnalisisCanonico(periodo.Fin) ||
				periodo.Fin.After(periodo.Inicio.AddDate(maximoAniosPeriodoFuente, 0, 0)))) {
		return false
	}
	if periodoCoberturaV2(periodo) {
		return periodo.PoliticaFin.Validar() == nil
	}
	return !periodo.Fin.IsZero() && periodo.CausaFin == ""
}

func periodosCoberturaCoinciden(a, b domain.PeriodoPrevisto) bool {
	return a.Inicio.Equal(b.Inicio) && a.Fin.Equal(b.Fin) &&
		a.CausaFin == b.CausaFin && a.PoliticaFin == b.PoliticaFin
}

func escribirPeriodoCobertura(
	escritor *escritorCanonFuenteAnalisis,
	periodo domain.PeriodoPrevisto,
) {
	escritor.instante(periodo.Inicio)
	if !periodoCoberturaV2(periodo) {
		escritor.instante(periodo.Fin)
		return
	}
	if periodo.Fin.IsZero() {
		escritor.texto("causa")
		escritor.texto(string(periodo.CausaFin))
	} else {
		escritor.texto("fecha")
		escritor.instante(periodo.Fin)
	}
	escritor.texto(periodo.PoliticaFin.ReglaRef)
	escritor.entero64(periodo.PoliticaFin.CatalogoVersion)
	escritor.texto(periodo.PoliticaFin.CatalogoHuellaSHA256)
	escritor.texto(periodo.PoliticaFin.FechaFin)
	escritor.texto(string(periodo.PoliticaFin.CausaFin))
}
