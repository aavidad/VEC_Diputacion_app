package postgres

import (
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestDecodificarPoliticaFinConfirmadaDistingueLegadoYRechazaMaterialInvalido(t *testing.T) {
	legado, err := decodificarPoliticaFinConfirmada(nil, true)
	if err != nil || legado != (domain.PoliticaFin{}) {
		t.Fatal("una confirmación anterior a c12 debe conservar su canon")
	}
	politica := domain.PoliticaFin{
		ReglaRef: "regla:modalidad-sintetica", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		FechaFin:             "opcional", CausaFin: "fin_sustitucion",
	}
	contenido, err := json.Marshal(politica)
	if err != nil {
		t.Fatal(err)
	}
	restaurada, err := decodificarPoliticaFinConfirmada(contenido, true)
	if err != nil || restaurada != politica {
		t.Fatal("la consulta no conservó la instantánea exacta")
	}
	for _, invalido := range [][]byte{
		[]byte(`null`), []byte(`{}`),
		[]byte(`{"regla_ref":"regla:modalidad-sintetica","catalogo_version":3,"catalogo_huella_sha256":"` + strings.Repeat("a", 64) + `","fecha_fin":"opcional","causa_fin":"fin_sustitucion","dato_ajeno":true}`),
	} {
		if _, err := decodificarPoliticaFinConfirmada(invalido, true); err == nil {
			t.Fatal("la política histórica inválida debe denegarse")
		}
	}
}
