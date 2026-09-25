package domain

import (
	"testing"
	"time"
)

func TestPlazoRespuestaVenceAlFinalDelUltimoDia(t *testing.T) {
	hasta := time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)
	if SituacionPlazo(hasta, hasta.Add(-time.Microsecond)) != PlazoRespuestaEnPlazo ||
		SituacionPlazo(hasta, hasta) != PlazoRespuestaVencido {
		t.Fatal("el plazo incluye todo el último día y vence exactamente al acabar")
	}
	if ProcedePropuestaExpiracion(hasta, hasta.Add(-time.Second), false) ||
		!ProcedePropuestaExpiracion(hasta, hasta, false) || ProcedePropuestaExpiracion(hasta, hasta.Add(time.Hour), true) {
		t.Fatal("la propuesta de expiración exige vencimiento sin respuesta")
	}
}

func TestRespuestaFueraDePlazoSegunLaRegla(t *testing.T) {
	hasta := time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)
	tarde, aTiempo := hasta.Add(time.Minute), hasta.Add(-time.Minute)
	casos := []struct {
		tratamiento TratamientoRespuestaFueraDePlazo
		recibida    time.Time
		causa       bool
		esperado    AdmisionRespuesta
	}{
		{TratamientoFueraDePlazoNoAdmitir, aTiempo, false, RespuestaAdmitidaEnPlazo},
		{TratamientoFueraDePlazoAdmitir, tarde, false, RespuestaAdmitidaFueraDePlazo},
		{TratamientoFueraDePlazoExigeCausaJustificada, tarde, false, RespuestaRequiereCausaJustificada},
		{TratamientoFueraDePlazoExigeCausaJustificada, tarde, true, RespuestaAdmitidaFueraDePlazo},
		{TratamientoFueraDePlazoNoAdmitir, tarde, true, RespuestaNoAdmitida},
		{TratamientoRespuestaFueraDePlazo("inventado"), tarde, true, RespuestaNoAdmitida},
	}
	for _, caso := range casos {
		if obtenida := EvaluarAdmisionRespuesta(caso.tratamiento, hasta, caso.recibida, caso.causa); obtenida != caso.esperado {
			t.Errorf("%s %v causa=%v: %s", caso.tratamiento, caso.recibida, caso.causa, obtenida)
		}
	}
	if TratamientoRespuestaFueraDePlazo("").Valido() || !TratamientoFueraDePlazoAdmitir.Valido() ||
		ConfirmacionExpiracionLlamamiento("automatica").Valida() || !ConfirmacionExpiracionRRHH.Valida() {
		t.Fatal("solo se admiten los mecanismos implementados")
	}
}
