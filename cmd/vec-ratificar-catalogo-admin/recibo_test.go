package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRespuestaNoConfirmaHuellaNiAprobacionDivergentes(t *testing.T) {
	sha := strings.Repeat("a", 64)
	pre := strings.Repeat("b", 64)
	rol := strings.Repeat("c", 64)
	control := strings.Repeat("d", 64)
	cat := strings.Repeat("e", 64)
	desc := strings.Repeat("f", 64)
	p := documentoPlan{OperacionRef: "rca_0123456789012345678901", RolRef: "rol:administracion_perfiles:v7", RolSHA: rol,
		ControlRevision: json.Number("1"), ControlSHA: control, CatalogoSHA: cat, PreimagenSHA: pre}
	a := aprobacionPrivada{AprobacionRef: "aprobacion:prueba", AprobacionSHA256: sha}
	r := respuestaRatificacion{Estado: "permitido", Codigo: "ratificacion_registrada", AuditoriaIntento: auditoriaIntento{
		AuditoriaRef: "aud_v3_rcai_" + strings.Repeat("1", 32), Secuencia: 2,
		HuellaSHA256: sha, CorrelacionRef: "correlacion_" + strings.Repeat("2", 32),
		RegistradaEn: "2026-10-09T01:00:02+00:00",
	}, Recibo: &reciboRatificacion{
		Esquema: "vec.admin.ratificacion-catalogo.recibo.v1", OperacionRef: p.OperacionRef,
		PlanSHA: sha, PreimagenSHA: pre, RolRef: p.RolRef, RolSHA: rol,
		ControlRevision: 1, ControlSHA: control, CatalogoPrevioSHA: cat,
		DescriptoresSHA: desc, CatalogoPosterSHA: sha, VigenteDesde: "2026-10-09T01:00:00+00:00",
		AprobacionRef: a.AprobacionRef, AprobacionSHA: a.AprobacionSHA256,
		OperadorLogin: "login_prueba", AuditoriaRef: "aud_v3_rca_" + sha[:32],
		AuditoriaSecuencia: 1, AuditoriaSHA: sha, ConfirmadoEn: "2026-10-09T01:00:01+00:00",
	}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validarRespuesta(b, p, sha, desc, a, "login_prueba"); err != nil {
		t.Fatal(err)
	}
	r.Recibo.DescriptoresSHA = cat
	b, _ = json.Marshal(r)
	if _, err := validarRespuesta(b, p, sha, desc, a, "login_prueba"); err == nil {
		t.Fatal("aceptó descriptores divergentes")
	}
	r.Recibo.DescriptoresSHA = desc
	r.Recibo.AprobacionSHA = cat
	b, _ = json.Marshal(r)
	if _, err := validarRespuesta(b, p, sha, desc, a, "login_prueba"); err == nil {
		t.Fatal("aceptó una aprobación distinta")
	}
}

type transaccionFalsa struct {
	canon     string
	respuesta []byte
	ratifico  bool
	confirmo  bool
	cerrada   bool
}

func (f *transaccionFalsa) Preparar(context.Context) (string, error) { return "login_prueba", nil }
func (f *transaccionFalsa) CanonYDescriptores(context.Context, string) (string, string, error) {
	return f.canon, strings.Repeat("f", 64), nil
}
func (f *transaccionFalsa) Ratificar(context.Context, string, string) ([]byte, error) {
	f.ratifico = true
	return f.respuesta, nil
}
func (f *transaccionFalsa) Confirmar(context.Context) error { f.confirmo = true; return nil }
func (f *transaccionFalsa) Cerrar(context.Context)          { f.cerrada = true }

func TestTransaccionNoInvocaFachadaSiPlanNoEsCanonico(t *testing.T) {
	f := &transaccionFalsa{canon: `{}`}
	plan := []byte(`{"descriptores":[]}`)
	abrir := func(context.Context, conexionPrivada) (transaccion, error) { return f, nil }
	_, err := ejecutarOperacion(context.Background(), conexionPrivada{}, plan, strings.Repeat("a", 64), documentoPlan{}, aprobacionPrivada{}, abrir)
	if err == nil || f.ratifico || f.confirmo || !f.cerrada {
		t.Fatal("un plan no canónico llegó a la fachada o al COMMIT")
	}
}

func TestTransaccionNoConfirmaRespuestaInvalida(t *testing.T) {
	plan := []byte(`{"descriptores":[]}`)
	f := &transaccionFalsa{canon: string(plan), respuesta: []byte(`{"recibo":{},"replay":false}`)}
	abrir := func(context.Context, conexionPrivada) (transaccion, error) { return f, nil }
	_, err := ejecutarOperacion(context.Background(), conexionPrivada{}, plan, strings.Repeat("a", 64), documentoPlan{}, aprobacionPrivada{}, abrir)
	if err == nil || !f.ratifico || f.confirmo || !f.cerrada {
		t.Fatal("una respuesta incompleta alcanzó COMMIT")
	}
}

func TestDenegacionAuditadaConfirmaIntentoSinRecibo(t *testing.T) {
	plan := []byte(`{"descriptores":[]}`)
	denegado := respuestaRatificacion{Estado: "denegado", Codigo: "ratificacion_rechazada",
		AuditoriaIntento: auditoriaIntento{AuditoriaRef: "aud_v3_rcai_" + strings.Repeat("1", 32),
			Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64),
			CorrelacionRef: "correlacion_" + strings.Repeat("2", 32), RegistradaEn: "2026-10-09T01:00:00+00:00"}}
	b, _ := json.Marshal(denegado)
	f := &transaccionFalsa{canon: string(plan), respuesta: b}
	abrir := func(context.Context, conexionPrivada) (transaccion, error) { return f, nil }
	r, err := ejecutarOperacion(context.Background(), conexionPrivada{}, plan, strings.Repeat("a", 64), documentoPlan{}, aprobacionPrivada{}, abrir)
	if err != nil || r.Estado != "denegado" || len(r.Acuse) != 0 || !f.confirmo || !f.cerrada {
		t.Fatal("una denegación auditada no confirmó únicamente el intento")
	}
}
