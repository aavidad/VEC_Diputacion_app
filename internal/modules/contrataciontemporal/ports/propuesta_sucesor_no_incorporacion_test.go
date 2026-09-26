package ports

import (
	"testing"
	"time"
)

// Tras una no incorporación (CT124) el expediente vuelve a la fiscalización
// en una versión posterior a la de la selección original; la propuesta del
// sucesor (CT128) parte de esa versión con la selección de siempre.
func TestPropuestaSucesorTrasNoIncorporacionAdmiteVersionPosterior(t *testing.T) {
	s, a, _ := propuestaDesarrolloPrueba(t)
	raiz := a.Justificante.Seleccion
	if raiz.VersionExpediente != 6 {
		t.Fatalf("selección de prueba en la versión %d", raiz.VersionExpediente)
	}
	// Sin continuación, la versión debe ser la de la selección.
	sinContinuacion := s
	sinContinuacion.VersionEsperada = 8
	if a.ValidarPara(sinContinuacion) == nil {
		t.Fatal("propuesta sin continuación en otra versión admitida")
	}
	sc, _, b := continuacionPrueba(t)
	sc.OrganizacionRef, sc.ExpedienteRef = s.OrganizacionRef, s.ExpedienteRef
	b.ConfirmadaEn = raiz.ConfirmadaEn.Add(time.Second)
	c := ResultadoContinuacionLlamamiento{Solicitud: sc, LlamamientoAnteriorRef: raiz.LlamamientoRef, ReciboBolsa: b,
		ReciboRef: "recibo:continuacion", AuditoriaRef: "auditoria:continuacion", ConfirmadaEn: b.ConfirmadaEn.Add(time.Second), Estado: "confirmado"}
	a.Justificante.Continuacion = &c
	s.LlamamientoRef, a.Resolucion.Solicitud.LlamamientoRef = b.LlamamientoRef, b.LlamamientoRef
	a.Justificante.Respuesta.Solicitud.LlamamientoRef = b.LlamamientoRef
	for _, v := range []uint64{6, 8, 12} {
		s.VersionEsperada = v
		if err := a.ValidarPara(s); err != nil {
			t.Fatalf("propuesta del sucesor en la versión %d rechazada: %v", v, err)
		}
	}
	s.VersionEsperada = 5
	if a.ValidarPara(s) == nil {
		t.Fatal("propuesta anterior a la selección admitida")
	}
}

func TestVersionSeleccionPropuestaValida(t *testing.T) {
	j := JustificanteRespuestaRecibida{Seleccion: ReciboSolicitudLlamamientoBolsa{VersionExpediente: 6}}
	casos := []struct {
		continuacion bool
		esperada     uint64
		valida       bool
	}{{false, 6, true}, {false, 8, false}, {true, 6, true}, {true, 8, true}, {true, 5, false}}
	for _, c := range casos {
		j.Continuacion = nil
		if c.continuacion {
			j.Continuacion = &ResultadoContinuacionLlamamiento{}
		}
		if versionSeleccionPropuestaValida(j, c.esperada) != c.valida {
			t.Fatalf("continuación %v, versión %d: se esperaba %v", c.continuacion, c.esperada, c.valida)
		}
	}
}
