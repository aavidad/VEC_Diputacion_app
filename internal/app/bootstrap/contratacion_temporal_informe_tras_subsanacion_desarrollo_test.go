package bootstrap

import (
	"errors"
	"testing"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/reglas"
)

// El catálogo de ejemplo recoge la duda 5: tras subsanar, informe nuevo y
// nueva firma del informe definitivo. Exigirlo sin CT123 impide arrancar.
func TestInformeTrasSubsanacionDesdeCatalogoEjemplo(t *testing.T) {
	resolutor, err := nuevoResolutorReglasEjemplo(rutaReglasCTEjemploPrueba, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := fuenteInformeTrasSubsanacionReglas{resolutor: resolutor}.InformeTrasSubsanacion(t.Context())
	if err != nil || !p.ExigeInformeNuevo || p.DocumentoFirma != "informe_definitivo" {
		t.Fatalf("regla c12: %+v %v", p, err)
	}
	_, err = gobernarInformeTrasSubsanacionDesarrollo(resolutor, nil, &ctapplication.ServicioFiscalizaciones{}, &ctapplication.ServicioInformesJuridicos{})
	if !errors.Is(err, errInformeTrasSubsanacionSinCT123) {
		t.Fatalf("sin CT123 no se arranca: %v", err)
	}
	if fuente, err := gobernarInformeTrasSubsanacionDesarrollo(nil, nil, nil, nil); fuente != nil || err != nil {
		t.Fatal("sin catálogo rige la conducta de siempre")
	}
}

func TestInformeTrasSubsanacionAtributosDeLaRegla(t *testing.T) {
	casos := []struct {
		atributos map[string]string
		exige     bool
		valida    bool
	}{
		{nil, false, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "no"}, false, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si"}, true, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si", atributoDocumentoFirmaInformeNuevo: "informe_definitivo"}, true, true},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "quizas"}, false, false},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "no", atributoDocumentoFirmaInformeNuevo: "informe_definitivo"}, false, false},
		{map[string]string{atributoInformeNuevoTrasSubsanacion: "si", atributoDocumentoFirmaInformeNuevo: "Informe Definitivo"}, false, false},
	}
	for _, c := range casos {
		p, err := politicaInformeTrasSubsanacionDesdeRegla(reglas.Regla{Atributos: c.atributos})
		if (err == nil) != c.valida || (c.valida && p.ExigeInformeNuevo != c.exige) {
			t.Errorf("%v: %+v %v", c.atributos, p, err)
		}
	}
}
