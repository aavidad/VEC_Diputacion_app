package postgres

import (
	"strings"
	"testing"
)

func TestSelectorRecuperadoConEtiquetaNoPublicable(t *testing.T) {
	codigo, version, huella := "gestion_situacion", int64(2), strings.Repeat("a", 64)
	selector, valido := selectorCausaParticipacionRecuperada(&codigo, &version, &huella, nil, codigo)
	if !valido || selector == nil || selector.Codigo != codigo || selector.Version != version || selector.HuellaSHA256 != huella {
		t.Fatalf("selector completo con etiqueta NULL perdió replay: %#v, %v", selector, valido)
	}
	if selector, valido := selectorCausaParticipacionRecuperada(nil, nil, nil, nil, "motivo libre anterior"); !valido || selector != nil {
		t.Fatalf("historia legacy sin selector: %#v, %v", selector, valido)
	}
	for _, caso := range []struct {
		codigo   *string
		version  *int64
		huella   *string
		etiqueta *string
	}{
		{&codigo, nil, &huella, nil},
		{nil, &version, &huella, nil},
		{nil, nil, nil, &codigo},
	} {
		if _, valido := selectorCausaParticipacionRecuperada(caso.codigo, caso.version, caso.huella, caso.etiqueta, codigo); valido {
			t.Fatalf("selector parcial aceptado: %#v", caso)
		}
	}
}
