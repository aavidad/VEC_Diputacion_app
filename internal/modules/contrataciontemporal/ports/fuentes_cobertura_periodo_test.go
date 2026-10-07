package ports

import (
	"bytes"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func materialPeriodoCoberturaPrueba(t *testing.T, p domain.PeriodoPrevisto) []byte {
	t.Helper()
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escribirPeriodoCobertura(escritor, p)
	material, err := escritor.resultado()
	if err != nil {
		t.Fatal(err)
	}
	return material
}

func TestPeriodoCoberturaConCausaSellaReglaYNoSerializaFechaInventada(t *testing.T) {
	inicio := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{
		Inicio:   inicio,
		CausaFin: "reincorporacion_titular",
		PoliticaFin: domain.PoliticaFin{
			ReglaRef: "regla:modalidad:sustitucion:v2", CatalogoVersion: 2,
			CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			FechaFin:             "no_aplica", CausaFin: "reincorporacion_titular",
		},
	}
	if !periodoCoberturaValido(periodo) || !periodoCoberturaV2(periodo) {
		t.Fatal("cobertura debe admitir el periodo catalogado sin fecha")
	}
	material := materialPeriodoCoberturaPrueba(t, periodo)
	if bytes.Contains(material, []byte("0001-01-01")) ||
		!bytes.Contains(material, []byte("reincorporacion_titular")) ||
		!bytes.Contains(material, []byte("regla:modalidad:sustitucion:v2")) {
		t.Fatal("el canon omitió la causa o incluyó una fecha inventada")
	}
	alterado := periodo
	alterado.PoliticaFin.CatalogoVersion++
	if bytes.Equal(material, materialPeriodoCoberturaPrueba(t, alterado)) ||
		periodosCoberturaCoinciden(periodo, alterado) {
		t.Fatal("la versión de la regla no quedó ligada al periodo")
	}
	sinInstantanea := periodo
	sinInstantanea.PoliticaFin = domain.PoliticaFin{}
	if periodoCoberturaValido(sinInstantanea) {
		t.Fatal("un periodo nuevo por causa necesita instantánea de la regla")
	}
}

func TestPeriodoCoberturaLegacyConservaMaterialDeDosFechas(t *testing.T) {
	p := domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
	}
	if !periodoCoberturaValido(p) || periodoCoberturaV2(p) {
		t.Fatal("periodo fechado histórico debe seguir V1")
	}
	antiguo := nuevoEscritorCanonFuenteAnalisis()
	antiguo.instante(p.Inicio)
	antiguo.instante(p.Fin)
	esperado, err := antiguo.resultado()
	if err != nil || !bytes.Equal(esperado, materialPeriodoCoberturaPrueba(t, p)) {
		t.Fatal("se alteraron los bytes del periodo V1")
	}
}
