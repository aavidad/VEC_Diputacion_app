package bootstrap

import (
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestOrigenCTAcuseLigadoAlIntentoAceptaNonceConFormaDeDNI(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sesion := contextoInscripcionCanalPrueba(t, ahora, false)
	nonce := "12345678a" + strings.Repeat("0", 23) // Hexadecimal ficticio, no documento personal.
	datos := core.DatosIntentoAuditoria{
		Accion: bolsa.AccionConsultarOrigenCT, ModuloID: "bolsa", RecursoRef: "bolsa:prueba",
		FinalidadRef: bolsa.FinalidadConsultarOrigenCT, Resultado: core.ResultadoIntentoAuditoriaExito,
		Motivo: motivoLlamamientoDesarrollo(false), Proceso: "vec-rrhh", Canal: "interna_corporativa",
		CorrelacionRef: "correlacion_" + strings.Repeat("b", 32),
	}
	orden, err := core.NuevaOrdenIntentoAuditoria("intento_"+nonce, sesion.Resultado, sesion.Vinculo, datos)
	if err != nil {
		t.Fatal(err)
	}
	acuse := core.AcuseIntentoAuditoria{
		AuditoriaRef: "aud_v3_i_" + nonce, Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
		CorrelacionRef: datos.CorrelacionRef, RegistradaEn: ahora,
	}
	if acuse.ValidarPara(orden) != nil || bolsa.ReferenciaOpacaLlamamientoValida(acuse.AuditoriaRef) ||
		!acuseOrigenCTVinculadoAlIntento(acuse, orden) {
		t.Fatal("el acuse nominal correcto fue rechazado por la heurística de documento")
	}

	otroIntento, err := core.NuevaOrdenIntentoAuditoria("intento_"+strings.Repeat("c", 32), sesion.Resultado, sesion.Vinculo, datos)
	if err != nil || acuseOrigenCTVinculadoAlIntento(acuse, otroIntento) {
		t.Fatal("el acuse de otra orden fue aceptado")
	}
	prefijoAjeno := acuse
	prefijoAjeno.AuditoriaRef = "aud_v3_x_" + nonce
	if acuseOrigenCTVinculadoAlIntento(prefijoAjeno, orden) {
		t.Fatal("el acuse de otra familia fue aceptado")
	}
	otroNonce := acuse
	otroNonce.AuditoriaRef = "aud_v3_i_" + strings.Repeat("d", 32)
	if acuseOrigenCTVinculadoAlIntento(otroNonce, orden) {
		t.Fatal("el acuse de otro intento fue aceptado")
	}
	corrupto := acuse
	corrupto.Secuencia = 0
	if acuseOrigenCTVinculadoAlIntento(corrupto, orden) ||
		acuseOrigenCTVinculadoAlIntento(acuse, core.OrdenIntentoAuditoria{}) {
		t.Fatal("un acuse sin secuencia o sin orden válida fue aceptado")
	}
}
