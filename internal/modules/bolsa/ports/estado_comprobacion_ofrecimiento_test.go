package ports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEstadoComprobacionOfrecimientoDistingueHistoriaAusenciaYActo(t *testing.T) {
	var historica EntradaOrigenLlamamientoCT
	if err := json.Unmarshal([]byte(`{"orden_acta":1}`), &historica); err != nil || historica.ComprobacionOfrecimiento != nil {
		t.Fatalf("lectura histórica: %+v %v", historica, err)
	}
	var ausencia EntradaOrigenLlamamientoCT
	if err := json.Unmarshal([]byte(`{"comprobacion_ofrecimiento":{"version_esperada":0}}`), &ausencia); err != nil || ausencia.ComprobacionOfrecimiento == nil || ausencia.ComprobacionOfrecimiento.VersionEsperada != 0 {
		t.Fatalf("ausencia acreditada: %+v %v", ausencia, err)
	}
	b, err := json.Marshal(ausencia.ComprobacionOfrecimiento)
	if err != nil || string(b) != `{"version_esperada":0}` {
		t.Fatalf("ausencia publicó campos: %s %v", b, err)
	}
	ref := "comprobacion:" + strings.Repeat("a", 64)
	for _, vigente := range []bool{true, false} {
		estado := EstadoComprobacionOfrecimiento{VersionEsperada: 3, ComprobacionRef: ref, ReciboRef: "recibo:" + ref,
			RegistradaEn: time.Date(2026, 10, 18, 8, 0, 0, 123000, time.UTC), VersionPoliticaOfertas: 7,
			Resultado: "pendiente", MotivoClave: "antecedentes_incompletos", Vigente: &vigente}
		b, err := json.Marshal(estado)
		var vuelta EstadoComprobacionOfrecimiento
		if err != nil || json.Unmarshal(b, &vuelta) != nil || vuelta.Validar() != nil || vuelta.Vigente == nil || *vuelta.Vigente != vigente || vuelta.VersionEsperada != 3 {
			t.Fatalf("acto vigente=%t: %s %v", vigente, b, err)
		}
		estado.Resultado, estado.MotivoClave = "no_aplicable", "sin_supuesto_articulo_9"
		if estado.Validar() != nil {
			t.Fatal("resultado no aplicable de B133 rechazado")
		}
	}
}

func TestEstadoComprobacionOfrecimientoRechazaCamposAjenosEIncompletos(t *testing.T) {
	ref := "comprobacion:" + strings.Repeat("a", 64)
	base := `{"version_esperada":3,"comprobacion_ref":"` + ref + `","recibo_ref":"recibo:` + ref + `","registrada_en":"2026-10-18T08:00:00Z","version_politica_ofertas":7,"resultado":"pendiente","motivo_clave":"antecedentes_incompletos","vigente":false}`
	for nombre, raw := range map[string]string{
		"estado nulo":                  `null`,
		"ausencia con dato":            `{"version_esperada":0,"vigente":false}`,
		"versión ausente":              `{"vigente":false}`,
		"versión fraccionaria":         strings.Replace(base, `"version_esperada":3`, `"version_esperada":3.0`, 1),
		"política fuera del contrato":  strings.Replace(base, `"version_politica_ofertas":7`, `"version_politica_ofertas":9007199254740992`, 1),
		"fecha no finita":              strings.Replace(base, `"2026-10-18T08:00:00Z"`, `"infinity"`, 1),
		"año cero":                     strings.Replace(base, `"2026-10-18T08:00:00Z"`, `"0000-10-18T08:00:00Z"`, 1),
		"dato personal":                strings.Replace(base, `"vigente":false`, `"vigente":false,"actor_ref":"persona:uno"`, 1),
		"campo obligatorio nulo":       strings.Replace(base, `"vigente":false`, `"vigente":null`, 1),
		"recibo ajeno":                 strings.Replace(base, `"recibo_ref":"recibo:`+ref+`"`, `"recibo_ref":"recibo:otro"`, 1),
		"motivo ajeno":                 strings.Replace(base, `"antecedentes_incompletos"`, `"motivo_no_publicado"`, 1),
		"resultado y motivo distintos": strings.Replace(base, `"resultado":"pendiente"`, `"resultado":"no_aplicable"`, 1),
		"versión repetida":             strings.Replace(base, `"version_esperada":3`, `"version_esperada":3,"version_esperada":3`, 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			var entrada EntradaOrigenLlamamientoCT
			if err := json.Unmarshal([]byte(`{"comprobacion_ofrecimiento":`+raw+`}`), &entrada); err == nil {
				t.Fatalf("proyección inválida admitida: %s", raw)
			}
		})
	}
	var entrada EntradaOrigenLlamamientoCT
	if json.Unmarshal([]byte(`{"COMPROBACION_OFRECIMIENTO":null}`), &entrada) == nil {
		t.Fatal("nombre no canónico interpretado como lectura histórica")
	}
}
