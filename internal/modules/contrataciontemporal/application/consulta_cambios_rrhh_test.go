package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sesionCambiosRRHHPrueba struct {
	*sesionConsultaRRHHPrueba
	llamadasCambios int
	alterar         func(*ports.ResultadoConsultaCambiosRRHH)
}

func (s *sesionCambiosRRHHPrueba) ConsultarCambiosYRegistrar(_ context.Context, orden ports.OrdenConsultaDetalleRRHH) (ports.ResultadoConsultaCambiosRRHH, error) {
	s.llamadasCambios++
	anterior, nuevo, huella := "C2", "C1", "sha256:"+strings.Repeat("c", 64)
	r := ports.ResultadoConsultaCambiosRRHH{
		ExpedienteRef: orden.Solicitud().ExpedienteRef(), VersionExpediente: 3,
		Cambios: []ports.CambioExpedienteRRHH{
			{VersionExpediente: 2, RegistradaEn: orden.Instante(), OrigenVersion: "analisis_o3", OperacionRef: "op:ct:2", Ruta: "solicitud.grupo_subgrupo", ValorAnterior: &anterior, ValorNuevo: &nuevo},
			{VersionExpediente: 3, RegistradaEn: orden.Instante(), OrigenVersion: "cobertura_o4", OperacionRef: "op:ct:3", Ruta: "analisis.observaciones", ValorNuevo: &huella},
		},
		ConsumoHuellaSHA256: strings.Repeat("b", 64), AuditoriaRef: "auditoria:rrhh:001",
		AuditoriaHuellaSHA256: strings.Repeat("a", 64), ConsumidaEn: orden.Instante(),
	}
	if s.alterar != nil {
		s.alterar(&r)
	}
	return r, nil
}

func servicioCambiosRRHHPrueba(t *testing.T, campos []string, alterar func(*ports.ResultadoConsultaCambiosRRHH)) (*ServicioConsultaDetalleRRHH, *sesionCambiosRRHHPrueba, *entornoConsultaRRHH) {
	t.Helper()
	entorno := nuevoEntornoConsultaRRHH(t)
	entorno.emision.detalle.campos = campos
	// El histórico llega hasta la versión 3 observada.
	detalle, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:001", 3)
	if err != nil {
		t.Fatal(err)
	}
	entorno.detalle = detalle
	sesion := &sesionCambiosRRHHPrueba{sesionConsultaRRHHPrueba: entorno.sesion, alterar: alterar}
	servicio, err := NuevoServicioConsultaDetalleRRHH(entorno.autoridad, entorno.emisor, sesion, entorno.reloj)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, sesion, entorno
}

// Petición RRHH p.4: el histórico de cambios usa el permiso del detalle
// completo y valida cada cambio antes de entregarlo.
func TestConsultaCambiosRRHHDevuelveCambiosValidados(t *testing.T) {
	t.Parallel()
	servicio, sesion, entorno := servicioCambiosRRHHPrueba(t, nil, nil)
	resultado, err := servicio.ConsultarCambios(context.Background(), entorno.detalle)
	if err != nil || sesion.llamadasCambios != 1 || entorno.sesion.llamadasDetalle != 0 || len(resultado.Cambios) != 2 ||
		*resultado.Cambios[0].ValorAnterior != "C2" || resultado.Cambios[1].ValorAnterior != nil {
		t.Fatalf("cambios=%+v err=%v llamadas=%d", resultado.Cambios, err, sesion.llamadasCambios)
	}
}

func TestConsultaCambiosRRHHNoAlcanzaConConcesionDeSeguimiento(t *testing.T) {
	t.Parallel()
	servicio, sesion, entorno := servicioCambiosRRHHPrueba(t, camposSeguimientoRRHHPrueba, nil)
	_, err := servicio.ConsultarCambios(context.Background(), entorno.detalle)
	if !errors.Is(err, ErrConsultaRRHHNoObservable) || sesion.llamadasCambios != 0 {
		t.Fatalf("la concesión de seguimiento alcanzó los cambios: %v", err)
	}
}

func TestConsultaCambiosRRHHRechazaResultadosNoConfiables(t *testing.T) {
	t.Parallel()
	claro := "600123456 con texto libre\n"
	for nombre, alterar := range map[string]func(*ports.ResultadoConsultaCambiosRRHH){
		"otro expediente":    func(r *ports.ResultadoConsultaCambiosRRHH) { r.ExpedienteRef = "expediente:ajeno" },
		"versión posterior":  func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[1].VersionExpediente = 4 },
		"desordenado":        func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[0], r.Cambios[1] = r.Cambios[1], r.Cambios[0] },
		"ruta no canónica":   func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[0].Ruta = "solicitud..x" },
		"texto con saltos":   func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[0].ValorNuevo = &claro },
		"sin valores":        func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[1].ValorNuevo = nil },
		"sin cambio":         func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios[0].ValorNuevo = r.Cambios[0].ValorAnterior },
		"demasiados cambios": func(r *ports.ResultadoConsultaCambiosRRHH) { r.Cambios = make([]ports.CambioExpedienteRRHH, 501) },
	} {
		t.Run(nombre, func(t *testing.T) {
			servicio, _, entorno := servicioCambiosRRHHPrueba(t, nil, alterar)
			if _, err := servicio.ConsultarCambios(context.Background(), entorno.detalle); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("aceptado: %v", err)
			}
		})
	}
}
