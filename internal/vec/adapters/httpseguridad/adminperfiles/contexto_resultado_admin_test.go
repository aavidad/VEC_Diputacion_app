package adminperfiles

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func respuestaContextoADMINPrueba(t *testing.T) ([]byte, ports.SolicitudResolucionRegistroContextoActorV2,
	VinculoSesionADMIN, string, string, string, string) {
	t.Helper()
	ahora := time.Date(2026, 10, 4, 15, 0, 0, 0, time.UTC)
	actor, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_"+strings.Repeat("a", 22), "prf_"+strings.Repeat("b", 22),
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	op, evento, correlacion := "oca_"+strings.Repeat("c", 32), "evento_"+strings.Repeat("d", 32), "correlacion_"+strings.Repeat("e", 32)
	v := vinculoContextoADMINPrueba(ahora)
	v.PersonaRef, v.CuentaRef, v.PerfilActivoRef = actor.Contexto.PersonaRef,
		actor.Contexto.Instantanea.CuentaRef, actor.Contexto.PerfilActivoRef
	s := ports.SolicitudResolucionRegistroContextoActorV2{OperacionRef: op, SolicitadoEn: ahora,
		Contexto: domain.SolicitudContextoActor{Cuenta: domain.CuentaAutenticadaContextoActor{
			CuentaRef: v.CuentaRef, Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceHigh},
			PerfilActivoRef: v.PerfilActivoRef}}
	e := map[string]any{"tipo_registro": "contexto_admin_pre_v2", "evento_ref": evento,
		"operador_login": "login_contexto", "actor_ref": v.PersonaRef,
		"perfil_activo_ref": v.PerfilActivoRef, "accion": "registrar_contexto_admin", "recurso_ref": op,
		"resultado": "permitido", "motivo_ref": "contexto_admin_pre_v2_permitido", "proceso": "vec_admin",
		"canal": "administracion_privilegiada", "finalidad_ref": "establecer_contexto_admin",
		"correlacion_ref": correlacion, "fuente_ref": v.FuenteRef, "fuente_sha256": v.FuenteSHA256}
	a := map[string]any{"auditoria_ref": "aud_v3_ap2_" + strings.Repeat("d", 32), "secuencia": 1,
		"huella_sha256": strings.Repeat("a", 64), "correlacion_ref": correlacion, "registrada_en": ahora}
	c := map[string]any{"operacion_ref": op, "registro_contexto_ref": actor.RegistroContextoRef,
		"representacion_canonica_base64":         base64.StdEncoding.EncodeToString(actor.RepresentacionCanonica),
		"huella_sha256":                          actor.HuellaSHA256,
		"manifiesto_procedencia_canonico_base64": base64.StdEncoding.EncodeToString(actor.ManifiestoProcedenciaCanonico),
		"manifiesto_procedencia_huella_sha256":   actor.ManifiestoProcedenciaHuellaSHA256,
		"autoridad_efectiva":                     actor.AutoridadEfectiva, "resuelto_en": actor.ResueltoEnAutoritativo}
	bruto, err := json.Marshal(map[string]any{"estado": "permitido", "motivo_ref": "contexto_admin_pre_v2_permitido",
		"evento": e, "acuse": a, "contexto": c})
	if err != nil {
		t.Fatal(err)
	}
	return bruto, s, v, actor.RegistroContextoRef, evento, correlacion, op
}

func TestRespuestaContextoADMINLigaAcuseEventoYActorAntesCommit(t *testing.T) {
	bruto, s, v, recibo, evento, correlacion, op := respuestaContextoADMINPrueba(t)
	var respuesta respuestaContextoADMIN
	confirmada, err := respuesta.validar(bruto, s, v, op, recibo, evento, correlacion, "vec_admin", "login_contexto")
	if err != nil || confirmada.OperacionRef != op || confirmada.RegistroContextoRef != recibo {
		t.Fatal("respuesta estructural positiva no se ligó a la solicitud")
	}
	for _, mutar := range []func(map[string]any){
		func(x map[string]any) {
			x["acuse"].(map[string]any)["correlacion_ref"] = "correlacion_" + strings.Repeat("f", 32)
		},
		func(x map[string]any) { delete(x["acuse"].(map[string]any), "secuencia") },
		func(x map[string]any) { x["evento"].(map[string]any)["fuente_sha256"] = strings.Repeat("f", 64) },
		func(x map[string]any) {
			x["contexto"].(map[string]any)["registro_contexto_ref"] = "rca_" + strings.Repeat("f", 32)
		},
		func(x map[string]any) { x["campo_desconocido"] = true },
	} {
		var x map[string]any
		if json.Unmarshal(bruto, &x) != nil {
			t.Fatal("fixture JSON")
		}
		mutar(x)
		b, _ := json.Marshal(x)
		var incompatible respuestaContextoADMIN
		if c, err := incompatible.validar(b, s, v, op, recibo, evento, correlacion, "vec_admin", "login_contexto"); err == nil || c.OperacionRef != "" {
			t.Fatal("acuse, fuente, recibo o campo ajeno pasó la frontera")
		}
	}
}
