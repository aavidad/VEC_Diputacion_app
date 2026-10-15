package domain

import (
	"errors"
	"strings"
	"testing"
)

func reglaRevisionPrueba(respuesta, resultado, destino string) ReglaResolucionRespuestaRevision {
	return ReglaResolucionRespuestaRevision{CatalogoID: "motivos_revision_bolsa",
		CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64),
		MotivoClave: "motivo_revision", Respuesta: respuesta, Resultado: resultado,
		Procedimiento: ProcedimientoLlamamientoDirecto, Destino: destino,
		Publicada: true, PoliticaVersion: 2, PoliticaHuellaSHA256: strings.Repeat("b", 64),
		PoliticaPublicada: true, Politica: PoliticaTransicionesSituacionCompilada()}
}

func TestResolucionRespuestaRevisionDirectaDerivaEfectoYNoInventaCambio(t *testing.T) {
	casos := []struct {
		respuesta, resultado, destino string
		justificada                   bool
		cambia                        bool
	}{
		{"acepta", ResultadoConfirmarAceptacion, SituacionPendienteIncorporacion, false, true},
		{"renuncia_justificada", ResultadoAdmitirJustificacion, SituacionDisponible, true, false},
		{"renuncia_justificada", ResultadoRechazarJustificacion, SituacionExcluido, true, true},
	}
	for _, c := range casos {
		t.Run(c.resultado, func(t *testing.T) {
			h := HechosResolucionRespuestaRevision{Respuesta: c.respuesta,
				Procedimiento: ProcedimientoLlamamientoDirecto, SituacionActual: SituacionDisponible,
				RespuestaPendiente: true, LlamamientoAcreditado: true}
			if c.justificada {
				h.Causa, h.JustificanteRef = "enfermedad", "documento:justificante:1"
				h.JustificanteSHA256, h.JustificanteRevisado = strings.Repeat("c", 64), true
			}
			e, err := DerivarResolucionRespuestaRevision(h,
				reglaRevisionPrueba(c.respuesta, c.resultado, c.destino), c.resultado, "motivo_revision")
			if err != nil || e.Destino != c.destino || e.CambiaSituacion != c.cambia {
				t.Fatalf("efecto directo incorrecto: %+v %v", e, err)
			}
		})
	}
}

func TestResolucionRespuestaRevisionRechazaResultadoYAutoridadNoAcreditados(t *testing.T) {
	h := HechosResolucionRespuestaRevision{Respuesta: "renuncia_justificada",
		Procedimiento: ProcedimientoLlamamientoDirecto, SituacionActual: SituacionDisponible,
		Causa: "enfermedad", JustificanteRef: "documento:justificante:1",
		JustificanteSHA256: strings.Repeat("c", 64), JustificanteRevisado: true,
		RespuestaPendiente: true, LlamamientoAcreditado: true}
	r := reglaRevisionPrueba(h.Respuesta, ResultadoAdmitirJustificacion, SituacionDisponible)
	for _, alterar := range []func(*HechosResolucionRespuestaRevision, *ReglaResolucionRespuestaRevision){
		func(_ *HechosResolucionRespuestaRevision, r *ReglaResolucionRespuestaRevision) {
			r.Resultado = ResultadoConfirmarAceptacion
		},
		func(h *HechosResolucionRespuestaRevision, _ *ReglaResolucionRespuestaRevision) {
			h.Procedimiento = "publicado"
		},
		func(h *HechosResolucionRespuestaRevision, _ *ReglaResolucionRespuestaRevision) {
			h.JustificanteRevisado = false
		},
		func(h *HechosResolucionRespuestaRevision, _ *ReglaResolucionRespuestaRevision) {
			h.RespuestaPendiente = false
		},
		func(_ *HechosResolucionRespuestaRevision, r *ReglaResolucionRespuestaRevision) { r.Publicada = false },
	} {
		hActual, rActual := h, r
		alterar(&hActual, &rActual)
		if _, err := DerivarResolucionRespuestaRevision(hActual, rActual, ResultadoAdmitirJustificacion, "motivo_revision"); !errors.Is(err, ErrResolucionRespuestaRevisionInvalida) {
			t.Fatalf("resolución no acreditada admitida: %+v %+v %v", hActual, rActual, err)
		}
	}
	politicaCerrada := r
	politicaCerrada.Politica = r.Politica.Restringir(SituacionDisponible, []string{SituacionExcluido})
	hAcepta := HechosResolucionRespuestaRevision{Respuesta: "acepta", Procedimiento: ProcedimientoLlamamientoDirecto,
		SituacionActual: SituacionDisponible, RespuestaPendiente: true, LlamamientoAcreditado: true}
	politicaCerrada.Respuesta, politicaCerrada.Resultado, politicaCerrada.Destino = "acepta", ResultadoConfirmarAceptacion, SituacionPendienteIncorporacion
	if _, err := DerivarResolucionRespuestaRevision(hAcepta, politicaCerrada, ResultadoConfirmarAceptacion, "motivo_revision"); !errors.Is(err, ErrResolucionRespuestaRevisionInvalida) {
		t.Fatal("transición S6 cerrada admitida")
	}
}
