package application

import (
	"context"
	"errors"
	"testing"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

type lectorPreparacionPrueba struct {
	resultado ports.HechosPreparados
	err       error
	llamadas  int
}

func (l *lectorPreparacionPrueba) LeerHechosSinteticos(context.Context, ports.SelectorHechosPreparacion) (ports.HechosPreparados, error) {
	l.llamadas++
	return l.resultado, l.err
}

func TestConsumidorNoEntregaLecturaIncompletaOSustituida(t *testing.T) {
	s := SolicitudHechosPreparacion{Alcance: meritos.AlcanceSintetico,
		Contexto: ContextoHechosPreparacion{ConvocatoriaRef: "convocatoria:ensayo", BasesRef: "bases:ensayo", BasesVersion: 1,
			SolicitudRef: "solicitud:ensayo", HitoRef: "hito:ensayo", Uso: "requisito"},
		Selector: ports.SelectorHechosPreparacion{PersonaRef: "persona:ensayo", FechaCorte: "2026-10-01", Hechos: []ports.ReferenciaHechoPreparacion{{Referencia: "hecho:01", VersionEsperada: 2}}}}
	for _, l := range []*lectorPreparacionPrueba{
		{resultado: ports.HechosPreparados{Alcance: "real", VersionPaquete: "paquete:1"}},
		{resultado: ports.HechosPreparados{Alcance: meritos.AlcanceSintetico, VersionPaquete: "paquete:1", Hechos: []ports.HechoPreparado{{Referencia: "hecho:otro", Version: 2, Pendientes: []string{"pendiente"}}}}},
		{resultado: ports.HechosPreparados{Alcance: meritos.AlcanceSintetico, VersionPaquete: "paquete:1", Hechos: []ports.HechoPreparado{{Referencia: "hecho:01", Version: 1, Pendientes: []string{"pendiente"}}}}},
		{err: errors.New("dependencia")},
	} {
		if r, err := ConsultarHechosPreparacion(context.Background(), s, l); err == nil || r.Hechos != nil || r.Contexto.ConvocatoriaRef != "" {
			t.Fatal("lectura fallida expone resultado")
		}
	}
	l := &lectorPreparacionPrueba{}
	s.Contexto.Uso = "admision"
	if _, err := ConsultarHechosPreparacion(context.Background(), s, l); err == nil || l.llamadas != 0 {
		t.Fatal("uso inventado alcanza fuente")
	}
}
