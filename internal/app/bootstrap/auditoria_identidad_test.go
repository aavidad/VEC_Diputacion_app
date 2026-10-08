package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/auditoria"
)

type identidadAuditoriaNominalPrueba struct{ fuentes []auditoria.FuenteConsulta }

func (i *identidadAuditoriaNominalPrueba) ResolverIdentidadConsulta(_ context.Context, _ *http.Request, fuente auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	i.fuentes = append(i.fuentes, fuente)
	return auditoria.IdentidadResuelta{}, nil
}

func TestIdentidadAuditoriaPorFuenteDespachaSoloFuenteExacta(t *testing.T) {
	opciones, ct, bolsa := &identidadAuditoriaNominalPrueba{}, &identidadAuditoriaNominalPrueba{}, &identidadAuditoriaNominalPrueba{}
	i := identidadAuditoriaPorFuenteRRHH{opciones: opciones, ct: ct, bolsa: bolsa}
	post := httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, nil)
	for _, fuente := range []auditoria.FuenteConsulta{auditoria.FuenteConsultaCT, auditoria.FuenteConsultaBolsa, "CT", "bolsa ", auditoria.FuenteConsultaGeneral, "auditoria_ct", "ct:expediente"} {
		_, err := i.ResolverIdentidadConsulta(context.Background(), post, fuente)
		if fuente == auditoria.FuenteConsultaCT || fuente == auditoria.FuenteConsultaBolsa {
			if err != nil {
				t.Fatalf("fuente %q no llegó a su identidad nominal: %v", fuente, err)
			}
		} else if !errors.Is(err, auditoria.ErrDenegada) {
			t.Fatalf("fuente %q no denegada: %v", fuente, err)
		}
	}
	if len(ct.fuentes) != 1 || ct.fuentes[0] != auditoria.FuenteConsultaCT ||
		len(bolsa.fuentes) != 1 || bolsa.fuentes[0] != auditoria.FuenteConsultaBolsa || len(opciones.fuentes) != 0 {
		t.Fatalf("fuentes mezcladas: ct=%v bolsa=%v opciones=%v", ct.fuentes, bolsa.fuentes, opciones.fuentes)
	}
	get := httptest.NewRequest(http.MethodGet, auditoria.RutaOpciones, nil)
	_, _ = i.ResolverIdentidadConsulta(context.Background(), get, auditoria.FuenteConsultaGeneral)
	if len(opciones.fuentes) != 1 || opciones.fuentes[0] != auditoria.FuenteConsultaGeneral || len(ct.fuentes) != 1 || len(bolsa.fuentes) != 1 {
		t.Fatal("GET opciones consumió un perfil nominal de consulta")
	}
	_, _ = i.ResolverIdentidadConsulta(context.Background(), post, auditoria.FuenteConsultaGeneral)
	if len(ct.fuentes) != 1 || len(bolsa.fuentes) != 1 {
		t.Fatal("POST sin fuente tipada consumió un perfil")
	}
	_, _ = i.ResolverIdentidadConsulta(context.Background(), get, auditoria.FuenteConsultaCT)
	if len(ct.fuentes) != 1 {
		t.Fatal("método ajeno consumió un perfil")
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err := i.ResolverIdentidadConsulta(ctxCancelado, post, auditoria.FuenteConsultaCT)
	if !errors.Is(err, auditoria.ErrDenegada) || len(ct.fuentes) != 1 {
		t.Fatal("petición cancelada alcanzó la identidad CT")
	}
	ajena := httptest.NewRequest(http.MethodPost, "/api/vec/contratacion-temporal/consultas", nil)
	_, err = i.ResolverIdentidadConsulta(context.Background(), ajena, auditoria.FuenteConsultaCT)
	if !errors.Is(err, auditoria.ErrDenegada) || len(ct.fuentes) != 1 {
		t.Fatal("ruta ajena alcanzó la identidad CT")
	}
}

func TestRutasAuditoriaConIdentidadesExigenCadaAutoridad(t *testing.T) {
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
	d := dependenciasIdentidadAuditoriaConsultaRRHH{
		PoolCT: &pgxpool.Pool{}, PoolBolsa: &pgxpool.Pool{},
		EmisorCT: &emisorAuditoriaConsultaPrueba{}, EmisorBolsa: &emisorAuditoriaConsultaPrueba{},
		IdentidadOpciones: &identidadAuditoriaNominalPrueba{}, IdentidadCT: &identidadAuditoriaNominalPrueba{},
		IdentidadBolsa: &identidadAuditoriaNominalPrueba{}, Opciones: &opcionesAuditoriaConsultaPrueba{},
		Intentos: configuracionIntentosConsultaPrueba(escenario.motivo),
	}
	if rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(d); err != nil || len(rutas) != 2 {
		t.Fatalf("factory completa: rutas=%v error=%v", rutas, err)
	}
	casos := []struct {
		nombre string
		quitar func(*dependenciasIdentidadAuditoriaConsultaRRHH)
	}{
		{"pool CT", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) { x.PoolCT = nil }},
		{"pool Bolsa", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) { x.PoolBolsa = nil }},
		{"emisor CT", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.EmisorCT = (*emisorAuditoriaConsultaPrueba)(nil)
		}},
		{"emisor Bolsa", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.EmisorBolsa = (*emisorAuditoriaConsultaPrueba)(nil)
		}},
		{"identidad opciones", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.IdentidadOpciones = (*identidadAuditoriaNominalPrueba)(nil)
		}},
		{"identidad CT", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.IdentidadCT = (*identidadAuditoriaNominalPrueba)(nil)
		}},
		{"identidad Bolsa", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.IdentidadBolsa = (*identidadAuditoriaNominalPrueba)(nil)
		}},
		{"opciones", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.Opciones = (*opcionesAuditoriaConsultaPrueba)(nil)
		}},
		{"intentos", func(x *dependenciasIdentidadAuditoriaConsultaRRHH) {
			x.Intentos.Registrador = (*registradorIntentoConsultaPrueba)(nil)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			incompleta := d
			caso.quitar(&incompleta)
			if rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(incompleta); rutas != nil || !errors.Is(err, auditoria.ErrNoDisponible) {
				t.Fatalf("dependencia ausente publicó rutas: %v, %v", rutas, err)
			}
		})
	}
}
