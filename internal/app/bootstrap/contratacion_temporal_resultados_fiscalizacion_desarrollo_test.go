package bootstrap

import (
	"testing"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// La regla c19 del catálogo de ejemplo recoge la respuesta de RRHH a la duda
// 5 y se entrega al servicio de fiscalización al arrancar.
func TestResultadosFiscalizacionDesdeCatalogoEjemplo(t *testing.T) {
	resolutor, err := nuevoResolutorReglasEjemplo(rutaReglasCTEjemploPrueba, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := fuenteResultadosFiscalizacionReglas{resolutor: resolutor}.ResultadosFiscalizacion(t.Context())
	if err != nil || len(p.Resultados()) != 3 {
		t.Fatalf("regla c19: %+v %v", p, err)
	}
	for r, esperado := range map[ctdomain.ResultadoFiscalizacion]ctdomain.EfectoResultadoFiscalizacion{
		ctdomain.FiscalizacionFavorable:                 ctdomain.EfectoFiscalizacionContinua,
		ctdomain.FiscalizacionFavorableConObservaciones: ctdomain.EfectoFiscalizacionContinua,
		ctdomain.FiscalizacionDesfavorable:              ctdomain.EfectoFiscalizacionVuelveUnidadGestora,
	} {
		if e, ok := p.Efecto(r); !ok || e != esperado {
			t.Errorf("%s: %s", r, e)
		}
	}
	servicio := &ctapplication.ServicioFiscalizaciones{}
	if err := gobernarResultadosFiscalizacionDesarrollo(servicio, resolutor); err != nil {
		t.Fatal(err)
	}
	if servicio.GobernarResultados(fuenteResultadosFiscalizacionReglas{resolutor: resolutor}) == nil {
		t.Fatal("la regla ya debía estar entregada")
	}
	if gobernarResultadosFiscalizacionDesarrollo(nil, nil) != nil {
		t.Fatal("sin catálogo rige la conducta de siempre")
	}
}

func TestResultadosFiscalizacionReglaIncoherenteNoArranca(t *testing.T) {
	regla := reglas.Regla{Unidad: reglas.UnidadLista, Valor: "favorable,desfavorable",
		Atributos: map[string]string{"efecto_favorable": "continua", "efecto_desfavorable": "continua"}}
	if _, err := politicaResultadosFiscalizacionDesdeRegla(regla); err == nil {
		t.Fatal("un desfavorable que continúa no se sabe aplicar")
	}
	regla.Atributos = map[string]string{"efecto_favorable": "continua"}
	if _, err := politicaResultadosFiscalizacionDesdeRegla(regla); err == nil {
		t.Fatal("resultado sin efecto declarado")
	}
}
