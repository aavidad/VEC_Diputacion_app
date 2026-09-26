package domain

import "testing"

func TestPoliticaResultadosFiscalizacion(t *testing.T) {
	p := PoliticaResultadosFiscalizacionPredeterminada()
	if len(p.Resultados()) != 3 || !p.Admite(FiscalizacionFavorableConObservaciones) {
		t.Fatalf("sin catálogo rigen los tres resultados: %+v", p)
	}
	if e, _ := p.Efecto(FiscalizacionDesfavorable); e != EfectoFiscalizacionVuelveUnidadGestora {
		t.Fatal("el desfavorable vuelve a la unidad gestora")
	}
	solo := map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{
		FiscalizacionFavorable: EfectoFiscalizacionContinua, FiscalizacionDesfavorable: EfectoFiscalizacionVuelveUnidadGestora,
	}
	restringida, err := NuevaPoliticaResultadosFiscalizacion([]ResultadoFiscalizacion{FiscalizacionFavorable, FiscalizacionDesfavorable}, solo)
	if err != nil || restringida.Admite(FiscalizacionFavorableConObservaciones) {
		t.Fatalf("el catálogo puede retirar un resultado: %v", err)
	}
	for nombre, caso := range map[string]struct {
		lista   []ResultadoFiscalizacion
		efectos map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion
	}{
		"efecto no aplicable": {[]ResultadoFiscalizacion{FiscalizacionFavorableConObservaciones},
			map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{FiscalizacionFavorableConObservaciones: EfectoFiscalizacionVuelveUnidadGestora}},
		"resultado desconocido": {[]ResultadoFiscalizacion{"reparo_suspensivo"},
			map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{"reparo_suspensivo": EfectoFiscalizacionContinua}},
		"sin efecto": {[]ResultadoFiscalizacion{FiscalizacionFavorable}, nil},
		"repetido": {[]ResultadoFiscalizacion{FiscalizacionFavorable, FiscalizacionFavorable},
			map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{FiscalizacionFavorable: EfectoFiscalizacionContinua}},
		"nada continúa": {[]ResultadoFiscalizacion{FiscalizacionDesfavorable},
			map[ResultadoFiscalizacion]EfectoResultadoFiscalizacion{FiscalizacionDesfavorable: EfectoFiscalizacionVuelveUnidadGestora}},
		"efecto sin resultado": {[]ResultadoFiscalizacion{FiscalizacionFavorable}, solo},
		"vacía":                {nil, nil},
	} {
		if _, err := NuevaPoliticaResultadosFiscalizacion(caso.lista, caso.efectos); err == nil {
			t.Errorf("%s: admitida", nombre)
		}
	}
}
