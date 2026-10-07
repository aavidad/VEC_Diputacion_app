package domain_test

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestAuditoriaFronteraSeleccionMinimizada(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/seleccion/preparacion-bases/guardar",
		"/api/vec/seleccion/preparacion-bases/consultar",
	} {
		orden := domain.OrdenAuditoriaFronteraRutaExacta{
			CorrelacionRef: "corr_no_disponible",
			Motivo:         domain.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
			Superficie:     domain.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases,
			Ruta:           ruta,
		}
		if err := orden.Validar(); err != nil {
			t.Fatalf("denegación previa a contexto no aceptada: %v", err)
		}
		for nombre, alterar := range map[string]func(*domain.OrdenAuditoriaFronteraRutaExacta){
			"actor libre": func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.ActorRef = "actor:aportado" },
			"actor opaco": func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.ActorRef = "per_0123456789abcdef0123456789abcdef" },
			"otra causa": func(o *domain.OrdenAuditoriaFronteraRutaExacta) {
				o.Motivo = domain.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
			},
			"detalle libre": func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Motivo = "csrf_cookie" },
			"alias CT": func(o *domain.OrdenAuditoriaFronteraRutaExacta) {
				o.Superficie = domain.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
			},
			"consulta":    func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "?persona=ajena" },
			"barra final": func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "/" },
			"subruta":     func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Ruta += "/detalle" },
			"base":        func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Ruta = "/api/vec/seleccion/preparacion-bases" },
			"otro módulo": func(o *domain.OrdenAuditoriaFronteraRutaExacta) { o.Ruta = "/api/vec/personal/organizacion-historica" },
		} {
			t.Run(ruta+"/"+nombre, func(t *testing.T) {
				copia := orden
				alterar(&copia)
				if !errors.Is(copia.Validar(), domain.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
					t.Fatal("ampliación de causa, actor, superficie o ruta aceptada")
				}
			})
		}
	}
}

func TestAuditoriaFronteraBaremoNoRecibeActor(t *testing.T) {
	for _, motivo := range []domain.MotivoAuditoriaFronteraRutaExacta{
		domain.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		domain.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
	} {
		orden := domain.OrdenAuditoriaFronteraRutaExacta{
			CorrelacionRef: "corr_no_disponible", Motivo: motivo,
			Superficie: domain.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo,
			Ruta:       "/api/vec/bolsa/reglas-baremo/borradores/alta",
		}
		if err := orden.Validar(); err != nil {
			t.Fatal(err)
		}
		orden.ActorRef = "actor:aportado"
		if !errors.Is(orden.Validar(), domain.ErrOrdenAuditoriaFronteraRutaExactaInvalida) {
			t.Fatal("actor recibido en rechazo anterior al contexto")
		}
	}
}
