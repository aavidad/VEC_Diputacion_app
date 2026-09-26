package ports

import (
	"strings"
	"testing"
	"time"
)

func antecedenteNoIncorporacionPrueba(t *testing.T) (SolicitudContinuarLlamamiento, AntecedenteContinuacionLlamamiento) {
	t.Helper()
	sr := solicitudResolverComunicacionPrueba(RespuestaLlamamientoAceptada)
	sr.VersionEsperada = 2
	sr.RevisionRespuestaRRHH, sr.RevisionPlazoRRHH = true, true
	sr.CriterioValidacionRef = "politica:revision-sintetica"
	r := resultadoResolucionComunicacionPrueba(sr, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoVigente, nil)
	n := AntecedenteNoIncorporacion{ReciboRef: "recibo:no-incorporacion", IntencionRef: "intencion:ct124:" + strings.Repeat("a", 64),
		ComandoRef: "comando:ct124:" + strings.Repeat("b", 64), VersionResultante: 8, RegistradaEn: r.ResueltaEn.Add(time.Hour)}
	s := SolicitudContinuarLlamamiento{"31111111-1111-4111-8111-111111111111", sr.OrganizacionRef, sr.ExpedienteRef, r.ResolucionRef, n.IntencionRef}
	a := AntecedenteContinuacionLlamamiento{Resolucion: r, ComandoSiguienteRef: n.ComandoRef, NoIncorporacion: &n,
		ComandoSiguiente: ComandoSiguienteLlamamiento{Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1", ComandoRef: n.ComandoRef,
			IntencionRef: n.IntencionRef, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: sr.LlamamientoRef,
			JustificanteRef: sr.PruebaRespuestaRef, SeleccionClave: sr.ClaveIdempotencia}}
	return s, a
}

func TestContinuacionTrasNoIncorporacion(t *testing.T) {
	s, a := antecedenteNoIncorporacionPrueba(t)
	if !a.EsNoIncorporacion() || a.EsExpiracion() || a.ValidarPara(s) != nil {
		t.Fatal("aceptación con no incorporación rechazada como antecedente", a.ValidarPara(s))
	}
	for nombre, mutar := range map[string]func(*AntecedenteContinuacionLlamamiento){
		"sin_no_incorporacion": func(a *AntecedenteContinuacionLlamamiento) { a.NoIncorporacion = nil },
		"intencion_ajena": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.NoIncorporacion
			c.IntencionRef = "intencion:ajena"
			a.NoIncorporacion = &c
		},
		"comando_ajeno": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.NoIncorporacion
			c.ComandoRef = "comando:ajeno"
			a.NoIncorporacion = &c
		},
		"version_nombramiento": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.NoIncorporacion
			c.VersionResultante = 7
			a.NoIncorporacion = &c
		},
		"anterior_a_aceptar": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.NoIncorporacion
			c.RegistradaEn = a.Resolucion.ResueltaEn.Add(-time.Second)
			a.NoIncorporacion = &c
		},
		"renuncia": func(a *AntecedenteContinuacionLlamamiento) {
			a.Resolucion.Solicitud.Respuesta = RespuestaLlamamientoRenunciada
		},
		"con_seleccion": func(a *AntecedenteContinuacionLlamamiento) { a.Seleccion = &ReciboSolicitudLlamamientoBolsa{} },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := a
			mutar(&otra)
			if otra.ValidarPara(s) == nil {
				t.Fatal("antecedente de no incorporación incoherente admitido")
			}
		})
	}
}

func TestReglaNoIncorporacionAcotada(t *testing.T) {
	r := ReglaNoIncorporacion{Motivos: []MotivoNoIncorporacion{{Clave: "no_presentado", Etiqueta: "No se presenta", ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo"}}}
	if !r.Valida() {
		t.Fatal("regla válida rechazada")
	}
	if m, ok := r.Motivo("no_presentado"); !ok || m.ConsecuenciaClave != "b24.sancion.baja_llamamiento_directo" {
		t.Fatal("motivo no encontrado")
	}
	for _, m := range []MotivoNoIncorporacion{{Clave: "no_presentado", Etiqueta: "otra", ConsecuenciaClave: "b24.x"},
		{Clave: "Mayusculas", Etiqueta: "x", ConsecuenciaClave: "b24.x"}, {Clave: "sin_consecuencia", Etiqueta: "x"}, {Clave: "sin_etiqueta", ConsecuenciaClave: "b24.x"}} {
		otra := r
		otra.Motivos = append(append([]MotivoNoIncorporacion(nil), r.Motivos...), m)
		if otra.Valida() {
			t.Fatalf("motivo inválido o repetido admitido: %+v", m)
		}
	}
	if (ReglaNoIncorporacion{}).Valida() {
		t.Fatal("regla sin motivos admitida")
	}
}
