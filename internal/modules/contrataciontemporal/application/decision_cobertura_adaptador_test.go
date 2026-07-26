package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestResultadoPropuestaParaAdaptadorReplicaCoherenciaDominioEstadoVia(t *testing.T) {
	casos := []struct {
		nombre   string
		esperado domain.EstadoPropuestaDecisionCobertura
		preparar func(*escenarioPresentacionCobertura)
	}{
		{"viable", domain.PropuestaCoberturaViable, func(*escenarioPresentacionCobertura) {}},
		{"incompleta", domain.PropuestaCoberturaIncompleta, func(e *escenarioPresentacionCobertura) {
			e.gobierno.politica = politicaGlobalC1(t, e.global.catalogo, e.global.entorno.reloj.Ahora(), "politica_bloqueante_adaptador")
			configurarResultadosPresentacionCobertura(t, e, func(ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion {
				return domain.ComprobacionNoConsta
			})
		}},
		{"conflictiva", domain.PropuestaCoberturaConflictiva, func(e *escenarioPresentacionCobertura) {
			configurarResultadosPresentacionCobertura(t, e, func(s ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion {
				if s.ViaClave == "via_global_01" {
					return domain.ComprobacionAfirmativa
				}
				return domain.ComprobacionNegativa
			})
		}},
		{"sin_via", domain.PropuestaCoberturaSinVia, func(e *escenarioPresentacionCobertura) {
			configurarResultadosPresentacionCobertura(t, e, func(ports.SolicitudConsultarCobertura) domain.ResultadoComprobacion {
				return domain.ComprobacionNegativa
			})
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			vias := viasPresentacionCoberturaPrueba(2)
			if caso.esperado == domain.PropuestaCoberturaConflictiva {
				vias[1].Comprobaciones[0].Clave = vias[0].Comprobaciones[0].Clave
			}
			escenario := nuevoEscenarioPresentacionCobertura(t, vias)
			caso.preparar(escenario)
			resultado, err := escenario.servicio.ProponerParaAdaptador(context.Background(), escenario.solicitud)
			if err != nil {
				t.Fatal(err)
			}
			datos, ok := resultado.DatosParaAdaptador()
			if !ok || datos.Estado != caso.esperado || (datos.Estado == domain.PropuestaCoberturaViable) != datos.ViaRecomendada.Valida() {
				t.Fatalf("estado/vía no coherentes: %+v", datos)
			}
		})
	}
	base := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(1))
	presentacion, err := base.servicio.Proponer(context.Background(), base.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	presentacion.Estado = domain.PropuestaCoberturaSinVia
	if _, err := nuevaResultadoPropuestaCoberturaParaAdaptador(presentacion); !errors.Is(err, ErrPresentacionPropuestaCoberturaNoConfiable) {
		t.Fatalf("sin vía incoherente=%v", err)
	}
}
