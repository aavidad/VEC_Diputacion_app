package application

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestHuellaRegularizacionDocumentalCoincideConVectorB77(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	fin := time.Date(2026, 10, 2, 0, 0, 0, 0, madrid)
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: ports.SolicitudCambiarSituacionParticipacion{
			BolsaRef: "bolsa:sintetica:rrhh17", ParticipacionRef: "participacion:sintetica:rrhh17",
			Motivo: "documento validado", ClaveIdempotencia: "clave:regularizar:rrhh17",
		},
		Justificante: domain.JustificanteOperacionSituacion{Referencia: "documento:sintetico:rrhh17", SHA256: strings.Repeat("c", 64)},
		Validador:    "validador:sintetico", SituacionEsperadaDesde: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		CausaFinalizadaEn: &fin, SolicitudRef: "solicitud-documental:" + strings.Repeat("a", 64),
		SolicitudVersionEsperada: 1, SolicitudContenidoSHA256: strings.Repeat("b", 64),
	}
	huella, err := huellaRegularizacionDocumental(q, "actor:sintetico", "recibo:regularizar:rrhh17")
	if err != nil || huella != "c880f55833ff54587697e24638453ff9faeb441d24b185a7047a7490a09d50c6" {
		t.Fatalf("vector B77: huella=%s error=%v", huella, err)
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: q.ParticipacionRef, ModuloID: "bolsa", Tipo: "participacion_bolsa",
		Ambitos:   map[string]string{"ambito_ref": "ambito:sintetico", "unidad_ref": "unidad:sintetica"},
		Atributos: map[string]string{"regularizacion_documental_sha256": huella},
	}
	canon, err := contextoRecursoRegularizacionCanonico(recurso)
	esperado := `{"ambitos":{"ambito_ref":"ambito:sintetico","unidad_ref":"unidad:sintetica"},"atributos":{"regularizacion_documental_sha256":"c880f55833ff54587697e24638453ff9faeb441d24b185a7047a7490a09d50c6"}}`
	suma := sha256.Sum256(canon)
	if err != nil || string(canon) != esperado || hex.EncodeToString(suma[:]) != "61858286bdd9bbb779c7255d43ebddf4a5ce5d70d41e3f72f36aef8a10dac1de" {
		t.Fatalf("contexto V3 B77: %s, sha=%s error=%v", canon, hex.EncodeToString(suma[:]), err)
	}
	q.Validador += "\x1f"
	if _, err := huellaRegularizacionDocumental(q, "actor:sintetico", "recibo:regularizar:rrhh17"); err == nil {
		t.Fatal("separador de canon admitido")
	}
}
