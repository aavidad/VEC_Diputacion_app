package domain

import (
	"strings"
	"testing"
	"time"
)

func TestPoliticaCeseSoloAdmiteVersionSinteticaCompleta(t *testing.T) {
	p := PoliticaCese{Version: 1, CatalogoRef: "catalogo:bolsa:cese:ejemplo-sintetico:v1", CatalogoSHA256: strings.Repeat("a", 64),
		Mapeo:        map[string]string{"interinidad": "general", "interinidad|acumulacion_tareas": "acumulacion_tareas"},
		MesesGeneral: 5, MesesAcumulacion: 9, Computo: "fecha_cese_meses_calendario_ajuste_fin_mes",
		Estado: "ejemplo_sintetico", PublicadaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
	if p.Validar() != nil {
		t.Fatal("política B45 válida rechazada")
	}
	copia := p.Clonar()
	copia.Mapeo["interinidad"] = "acumulacion_tareas"
	if p.Mapeo["interinidad"] != "general" {
		t.Fatal("mapa compartido con consumidor")
	}
	for _, cambio := range []func(*PoliticaCese){
		func(x *PoliticaCese) { x.Estado = "vigente" },
		func(x *PoliticaCese) { x.MesesGeneral = 0 },
		func(x *PoliticaCese) { x.Mapeo["interinidad"] = "otro" },
		func(x *PoliticaCese) { x.Mapeo["dni 123"] = "general" },
	} {
		q := p.Clonar()
		cambio(&q)
		if q.Validar() == nil {
			t.Fatalf("política ajena admitida: %+v", q)
		}
	}
}
