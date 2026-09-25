package ports

import "testing"

// La expiración confirmada es antecedente de continuación con la selección
// original y sin justificante; la renuncia no admite selección añadida.
func TestContinuacionLlamamientoAntecedenteExpiracion(t *testing.T) {
	_, j := justificanteRespuestaRecibidaPrueba(t)
	seleccion := j.Seleccion
	sr := solicitudResolverComunicacionPrueba(RespuestaLlamamientoExpirada)
	sr.OrganizacionRef, sr.ExpedienteRef, sr.LlamamientoRef = seleccion.OrganizacionRef, seleccion.ExpedienteRef, seleccion.LlamamientoRef
	sr.VersionEsperada = 2
	sr.RevisionRespuestaRRHH, sr.RevisionPlazoRRHH = true, true
	sr.CriterioValidacionRef = "politica:revision-sintetica"
	r := resultadoResolucionComunicacionPrueba(sr, ResultadoComunicacionLlamamientoConfirmado, PlazoLlamamientoExpirado, estadoOutbox(OutboxSiguienteCandidatoPendiente))
	if r.ValidarPara(sr) != nil {
		t.Fatal("resolución de expiración de prueba incoherente")
	}
	s := SolicitudContinuarLlamamiento{"31111111-1111-4111-8111-111111111112", sr.OrganizacionRef, sr.ExpedienteRef, r.ResolucionRef, r.IntencionSiguiente.IntencionRef}
	a := AntecedenteContinuacionLlamamiento{Resolucion: r, ComandoSiguienteRef: r.IntencionSiguiente.ComandoOpacoRef,
		ComandoSiguiente: ComandoSiguienteLlamamiento{
			Esquema: "vec.contratacion-temporal.siguiente-candidato.intencion.v1", ComandoRef: r.IntencionSiguiente.ComandoOpacoRef,
			IntencionRef: s.IntencionRef, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
			LlamamientoRef: sr.LlamamientoRef, SeleccionClave: sr.ClaveIdempotencia,
		}, Seleccion: &seleccion}
	if !a.EsExpiracion() || a.ValidarPara(s) != nil {
		t.Fatal("expiración confirmada rechazada como antecedente", a.ValidarPara(s))
	}
	for nombre, mutar := range map[string]func(*AntecedenteContinuacionLlamamiento){
		"sin_seleccion":    func(a *AntecedenteContinuacionLlamamiento) { a.Seleccion = nil },
		"plazo_vigente":    func(a *AntecedenteContinuacionLlamamiento) { a.Resolucion.EstadoPlazo = PlazoLlamamientoVigente },
		"con_justificante": func(a *AntecedenteContinuacionLlamamiento) { a.ComandoSiguiente.JustificanteRef = "justificante:ajeno" },
		"seleccion_ajena": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.Seleccion
			c.LlamamientoRef += "otro"
			a.Seleccion = &c
		},
		"seleccion_sin_propuesta": func(a *AntecedenteContinuacionLlamamiento) {
			c := *a.Seleccion
			c.PropuestaGenerada = false
			a.Seleccion = &c
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := a
			mutar(&otra)
			if otra.ValidarPara(s) == nil {
				t.Fatal("antecedente de expiración incoherente admitido")
			}
		})
	}
	sRen, aRen, _ := continuacionPrueba(t)
	aRen.Seleccion = &seleccion
	if aRen.ValidarPara(sRen) == nil {
		t.Fatal("renuncia con selección añadida admitida")
	}
}
