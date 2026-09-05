package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func solicitudReanudacionPrueba() SolicitudReservaEjecucionSeleccionLlamamiento {
	ref := func(n string) ReferenciaVersionadaIntegracionBolsa {
		return ReferenciaVersionadaIntegracionBolsa{Referencia: n, Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	}
	s := SolicitudReservaEjecucionSeleccionLlamamiento{
		ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11",
		OrganizacionRef:   "org:sintetica", ExpedienteRef: "exp:sintetico", VersionExpediente: 6,
		CorrelacionRef: "correlacion:sintetica", AutoridadSolicitante: "autoridad:sintetica",
		AutorizacionConsulta: ref("intencion:sintetica"), AccionConsulta: ref("accion:consulta"),
		RecursoConsulta: ref("necesidad:sintetica"), Necesidad: ref("necesidad:sintetica"),
		AccionOrden: ref("accion:orden"), Finalidad: ref("finalidad:sintetica"),
		Bolsa: ref("bolsa:sintetica"), Politica: ref("politica:sintetica"),
		MaximoPosiciones: 3, CantidadDisponible: 2,
	}
	s.HuellaSemantica = s.huellaEsperada()
	return s
}

func TestReanudacionSeleccionRecursoLigaIntencionCompleta(t *testing.T) {
	s := solicitudReanudacionPrueba()
	r, err := NuevoRecursoReanudacionSeleccionLlamamiento(s)
	if err != nil {
		t.Fatal(err)
	}
	canon := `{"organizacion_ref":"org:sintetica","expediente_ref":"exp:sintetico","version_expediente":6,"clave_idempotencia":"018f47a2-6b31-4c80-8a95-4d2e707c5a11","huella_semantica":"` + s.HuellaSemantica + `"}`
	h := sha256.Sum256([]byte(canon))
	if r.Referencia != s.ExpedienteRef || r.Tipo != TipoRecursoReanudacionSeleccionLlamamiento ||
		len(r.Ambitos) != 1 || r.Ambitos["organizacion_ref"] != s.OrganizacionRef ||
		len(r.Atributos) != 1 || r.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatal("recurso distinto del canon SQL")
	}
	for _, cambiar := range []func(*SolicitudReservaEjecucionSeleccionLlamamiento){
		func(x *SolicitudReservaEjecucionSeleccionLlamamiento) { x.OrganizacionRef += "b" },
		func(x *SolicitudReservaEjecucionSeleccionLlamamiento) { x.ExpedienteRef += "b" },
		func(x *SolicitudReservaEjecucionSeleccionLlamamiento) {
			x.ClaveIdempotencia = "118f47a2-6b31-4c80-8a95-4d2e707c5a11"
		},
		func(x *SolicitudReservaEjecucionSeleccionLlamamiento) { x.CantidadDisponible = 1 },
	} {
		otra := s
		cambiar(&otra)
		if _, err := NuevoRecursoReanudacionSeleccionLlamamiento(otra); err == nil {
			t.Fatal("solicitud alterada admitida sin validar huella")
		}
		otra.HuellaSemantica = otra.huellaEsperada()
		nuevo, err := NuevoRecursoReanudacionSeleccionLlamamiento(otra)
		if err != nil || nuevo.Atributos["material_sha256"] == r.Atributos["material_sha256"] {
			t.Fatal("intención diferente no ligada")
		}
	}
	s.VersionExpediente = 7
	s.HuellaSemantica = s.huellaEsperada()
	if _, err := NuevoRecursoReanudacionSeleccionLlamamiento(s); err == nil {
		t.Fatal("admitió versión fuera del corte")
	}
}
