package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// TestReciboAdmiteInstantesUTCConDesplazamientoCero reproduce la forma en que
// PostgreSQL serializa timestamptz dentro de jsonb con TimeZone=UTC
// («…+00:00», no «…Z»). encoding/json lo decodifica con una zona fija de
// desplazamiento cero, distinta de time.UTC, y el dominio exige UTC canónico.
// El acto ordinario y el cierre con recibo tienen que aceptarlo y devolver los
// instantes en time.UTC; un desplazamiento distinto de cero sigue rechazado
// (caso «zona» de TestRevocacionConservaVentanaHistoricaCompletaAntesDeCommit).
func TestReciboAdmiteInstantesUTCConDesplazamientoCero(t *testing.T) {
	for _, ruta := range []string{"ordinario", "cierre"} {
		t.Run(ruta, func(t *testing.T) {
			s, rol, pool, recibo := contratoV2Prueba(t)
			s.Operacion = domain.OperacionRevocarPerfil
			s.Objetivo.PerfilVersion, s.Objetivo.VinculoVersion = 1, 2
			s.Objetivo.VigenteDesde, s.Objetivo.VigenteHasta = time.Time{}, time.Time{}
			recibo.EstadoPosterior, recibo.VersionPosterior = domain.EstadoVinculoContextoActorRevocado, 3
			if err := s.Validar(); err != nil {
				t.Fatal(err)
			}
			cierre := domain.SolicitudCierrePropuestaAdministracionPerfiles{
				OperacionRef: "cierre_admin:" + strings.Repeat("a", 32), PropuestaRef: "propuesta_admin:" + strings.Repeat("b", 32),
				PropuestaHuellaSHA256: strings.Repeat("c", 64), ProponentePersonaRef: "per_" + strings.Repeat("x", 22),
				ObjetivoPersonaRef: s.Objetivo.PersonaRef, Aprobador: s.Actor, Evidencia: s.Evidencia,
				InstantaneaAutorizacion: s.InstantaneaAutorizacion, Decision: domain.DecisionAprobarPropuestaPerfil,
				Motivo: s.Motivo, CorrelacionRef: s.CorrelacionRef}
			if ruta == "cierre" {
				recibo.OperacionRef, recibo.PropuestaRef = cierre.OperacionRef, cierre.PropuestaRef
			}
			x := proyeccionReciboPrueba(recibo)
			x["vigente_desde"] = comoJSONBUTC(recibo.VigenteDesde)
			x["vigente_hasta"] = comoJSONBUTC(recibo.VigenteHasta)
			x["confirmado_en"] = comoJSONBUTC(recibo.ConfirmadoEn)
			var salida any = x
			e, _ := materialActo(s, rol, false)
			if ruta == "cierre" {
				salida = map[string]any{"operacion_ref": cierre.OperacionRef, "propuesta_ref": cierre.PropuestaRef,
					"propuesta_huella_sha256": cierre.PropuestaHuellaSHA256, "decision": cierre.Decision,
					"huella_cierre_sha256": strings.Repeat("d", 64), "confirmado_en": comoJSONBUTC(recibo.ConfirmadoEn), "recibo": x}
				e, _ = materialCierre(cierre)
			}
			b, _ := json.Marshal(salida)
			if !strings.Contains(string(b), "+00:00") {
				t.Fatalf("la respuesta de prueba no reproduce la forma de jsonb: %s", b)
			}
			tx := &txFalsa{fila: filaFalsa{dato: b}}
			pool.tx = tx
			a := &Autoridad{pool: pool, emisor: &emisorFalso{material: materialSintetico(t, e, recibo.ConfirmadoEn)},
				reloj: relojFijo(recibo.ConfirmadoEn)}
			var r domain.ReciboAdministracionPerfiles
			var confirmado time.Time
			var err error
			if ruta == "cierre" {
				c, errCierre := a.CerrarPropuestaSensible(context.Background(), cierre)
				err, confirmado = errCierre, c.ConfirmadoEn
				if c.Recibo != nil {
					r = *c.Recibo
				}
			} else {
				r, err = a.AplicarActoOrdinario(context.Background(), s)
				confirmado = r.ConfirmadoEn
			}
			if err != nil || tx.commits != 1 {
				t.Fatalf("instantes UTC «+00:00» rechazados: %v (commits=%d)", err, tx.commits)
			}
			for nombre, instante := range map[string]time.Time{"confirmado_en": confirmado,
				"recibo.confirmado_en": r.ConfirmadoEn, "vigente_desde": r.VigenteDesde, "vigente_hasta": r.VigenteHasta} {
				if instante.Location() != time.UTC {
					t.Fatalf("%s sale con zona %v, no time.UTC", nombre, instante.Location())
				}
			}
		})
	}
}

// comoJSONBUTC escribe el instante como lo hace jsonb para timestamptz con
// TimeZone=UTC: «AAAA-MM-DDTHH:MM:SS[.ffffff]+00:00».
func comoJSONBUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.999999") + "+00:00"
}
