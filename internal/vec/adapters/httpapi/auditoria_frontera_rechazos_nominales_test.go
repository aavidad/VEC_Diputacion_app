package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestRegistrarDenegacionFronteraPreparacionDelegaNominalmente(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/seleccion/preparacion-bases/guardar",
		"/api/vec/seleccion/preparacion-bases/consultar",
		"/api/vec/bolsa/reglas-baremo/borradores/alta",
		"/api/vec/bolsa/reglas-baremo/versiones/consultar",
		"/api/vec/bolsa/reglas-baremo/recibos/recuperar",
	} {
		motivos := []ports.MotivoAuditoriaFronteraRutaExacta{ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado}
		superficie := ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases
		if superficieAuditoriaFronteraRutaExacta(ruta) == ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo {
			superficie = ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo
			motivos = append(motivos, ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida)
		}
		for _, motivo := range motivos {
			r := &registradorAuditoriaFronteraRutaExactaEspia{}
			ctx := context.WithValue(context.Background(), claveActorAuditoriaBolsa{}, "persona:ajena")
			if err := RegistrarDenegacionFronteraPreparacion(ctx, r, http.MethodPost, ruta, motivo); err != nil {
				t.Fatal(err)
			}
			ordenes := r.ordenesRegistradas()
			if len(ordenes) != 1 || ordenes[0].Superficie != superficie || ordenes[0].Ruta != ruta ||
				ordenes[0].Motivo != motivo || ordenes[0].ActorRef != "" || ordenes[0].Validar() != nil {
				t.Fatal("export no delegó al registrador con orden nominal mínima")
			}
			_, cancelado, conPlazo := r.contextoRegistrado()
			if cancelado || !conPlazo {
				t.Fatal("auditoría sin contexto independiente y plazo común")
			}
		}
	}
}

func TestRegistrarDenegacionFronteraPreparacionFallaCerrado(t *testing.T) {
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	for _, caso := range []struct {
		nombre       string
		ctx          context.Context
		metodo, ruta string
		motivo       ports.MotivoAuditoriaFronteraRutaExacta
	}{
		{"sin contexto", nil, http.MethodPost, "/api/vec/seleccion/preparacion-bases/guardar", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"cancelado", ctxCancelado, http.MethodPost, "/api/vec/seleccion/preparacion-bases/guardar", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"metodo", context.Background(), http.MethodGet, "/api/vec/seleccion/preparacion-bases/guardar", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"query", context.Background(), http.MethodPost, "/api/vec/seleccion/preparacion-bases/guardar?actor=x", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"subruta", context.Background(), http.MethodPost, "/api/vec/bolsa/reglas-baremo/versiones/consultar/detalle", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"otra familia", context.Background(), http.MethodPost, "/api/vec/contratacion-temporal/expedientes", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"causa libre", context.Background(), http.MethodPost, "/api/vec/bolsa/reglas-baremo/borradores/alta", "detalle secreto"},
		{"causa ajena S2", context.Background(), http.MethodPost, "/api/vec/seleccion/preparacion-bases/consultar", ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := &registradorAuditoriaFronteraRutaExactaEspia{}
			err := RegistrarDenegacionFronteraPreparacion(caso.ctx, r, caso.metodo, caso.ruta, caso.motivo)
			if !errors.Is(err, ErrAutoridadRutaExactaNoDisponible) || len(r.ordenesRegistradas()) != 0 {
				t.Fatal("entrada inválida alcanzó auditor o afirmó éxito")
			}
		})
	}
	for _, r := range []ports.RegistradorAuditoriaFronteraRutaExacta{
		nil, (*registradorAuditoriaFronteraRutaExactaEspia)(nil),
		&registradorAuditoriaFronteraRutaExactaEspia{err: errors.New("fallo SQL")},
	} {
		if err := RegistrarDenegacionFronteraPreparacion(context.Background(), r, http.MethodPost,
			"/api/vec/seleccion/preparacion-bases/guardar", ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado); !errors.Is(err, ErrAutoridadRutaExactaNoDisponible) {
			t.Fatal("auditor ausente/fallido no produjo 503")
		}
	}
}
