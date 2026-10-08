package reglas

import (
	"errors"
	"testing"
	"time"
)

func TestPrepararRepeticionAjustesConservaCASOriginalYUsaCabezaReal(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	cabeza := versionAjustes(t, 2, ahora.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	preparada, err := PrepararRepeticionAjustes(ahora, nil, 0, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}})
	if err != nil {
		t.Fatal(err)
	}
	d := preparada.Datos()
	if d.VersionEsperada != 0 || d.BaseVersion != 3 || d.BaseHuellaSHA256 !=
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" ||
		len(d.Cambios) != 1 || d.Cambios[0].Anterior != "7" || d.Cambios[0].Nuevo != "7" ||
		d.Ajustes[CTPlazoFiscalizacion][CampoCantidad] != "7" ||
		string(d.Canonico) != `{"c03.plazo_fiscalizacion":{"cantidad":"7"}}` || d.HuellaSHA256 != cabeza.HuellaSHA256 {
		t.Fatalf("repetición modificó CAS o cabeza: %+v", d)
	}
	if _, err := PrepararRepeticionAjustes(ahora, nil, cabeza.Version, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}}); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("material de repetición admitido para alta: %v", err)
	}
}

func TestPrepararRepeticionAjustesRechazaCabezaFalsaYFormaInvalida(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC)
	cabeza := versionAjustes(t, 2, ahora.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	solicitud := []SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "8"}}
	cabeza.HuellaSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := PrepararRepeticionAjustes(ahora, nil, 0, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", solicitud); !errors.Is(err, ErrAjustesNoDisponibles) {
		t.Fatalf("cabeza sin integridad admitida: %v", err)
	}
	cabeza = versionAjustes(t, 2, ahora.Add(-time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "7"},
	})
	if _, err := PrepararRepeticionAjustes(ahora, nil, 0, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		append(solicitud, solicitud[0])); !errors.Is(err, ErrAjusteInvalido) {
		t.Fatalf("cambio duplicado admitido: %v", err)
	}
}

func TestRepeticionConservaEfectoOriginalAnteriorALaCabeza(t *testing.T) {
	ahora := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	efecto := ahora.Add(-24 * time.Hour)
	cabeza := versionAjustes(t, 3, ahora.Add(24*time.Hour), map[string]map[string]string{
		CTPlazoFiscalizacion: {CampoCantidad: "8"},
	})
	preparada, err := PrepararRepeticionAjustes(ahora, &efecto, 1, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}})
	if err != nil {
		t.Fatal(err)
	}
	d := preparada.Datos()
	if d.EfectoDesde == nil || !d.EfectoDesde.Equal(efecto) || d.VersionEsperada != 1 {
		t.Fatalf("se alteró la petición original: %+v", d)
	}
	sinFecha, err := PrepararRepeticionAjustes(ahora, nil, 1, cabeza, 3,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		[]SolicitudCambioAjuste{{ReglaClave: CTPlazoFiscalizacion, Campo: CampoCantidad, Nuevo: "7"}})
	if err != nil || sinFecha.Datos().EfectoDesde != nil {
		t.Fatalf("omisión de fecha no conservada: %v", err)
	}
}
