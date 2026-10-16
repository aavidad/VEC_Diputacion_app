package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuenteUnidadSelectoresPrueba struct {
	filtro   ports.FiltroUnidadSelectoresRecuperacionRRHH
	err      error
	llamadas int
	actor    string
}

func (f *fuenteUnidadSelectoresPrueba) ResolverUnidadSelectoresRecuperacionRRHH(_ context.Context, c ports.ContextoConsultaRRHH) (ports.FiltroUnidadSelectoresRecuperacionRRHH, error) {
	f.llamadas++
	f.actor = c.ActorRef()
	return f.filtro, f.err
}

type sesionSelectoresPrueba struct {
	*sesionConsultaRRHHPrueba
	llamadasSelectores int
	filtro             ports.FiltroUnidadSelectoresRecuperacionRRHH
	filas              []ports.FirmaSelectorRecuperacionRRHH
}

func (s *sesionSelectoresPrueba) ConsultarDetalleConSelectoresYRegistrar(ctx context.Context, o ports.OrdenConsultaDetalleRRHH, filtro ports.FiltroUnidadSelectoresRecuperacionRRHH) (ports.DetalleExpedienteRRHH, error) {
	s.llamadasSelectores++
	s.filtro = filtro
	d, err := s.sesionConsultaRRHHPrueba.ConsultarDetalleYRegistrar(ctx, o)
	if err != nil {
		return d, err
	}
	return d.ConSelectoresRecuperacionFirmas(s.filas, ports.ReciboSelectoresRecuperacionRRHH{
		AccesoRef: "lectura:rrhh:001", SelloDetalleSHA256: strings.Repeat("a", 64),
		SelloSelectoresSHA256: strings.Repeat("b", 64),
	})
}

func TestDetalleRRHHSelectoresSoloConConcesionExactaYUnidadPropia(t *testing.T) {
	entorno := nuevoEntornoConsultaRRHH(t)
	entorno.emision.detalle.campos = []string{"selectores_recuperacion_firmas"}
	sesion := &sesionSelectoresPrueba{sesionConsultaRRHHPrueba: entorno.sesion}
	fuente := &fuenteUnidadSelectoresPrueba{filtro: ports.FiltroUnidadSelectoresRecuperacionRRHH{
		UnidadRef: "unidad:rrhh:001", PerfilAuxRef: "prf_firma_000001",
		VersionRolAuxRef: "rol:ct_cargo_tecnico_firma_dev:v1", AsignacionRef: "asignacion:rrhh:001:v1",
		AsignacionHuellaSHA256: strings.Repeat("c", 64),
	}}
	sesion.filas = []ports.FirmaSelectorRecuperacionRRHH{{FirmaRef: "firma:rrhh:001", Selector: ports.SelectorRecuperacionFirmaRRHH{
		ExpedienteRef: entorno.detalle.ExpedienteRef(), VersionExpediente: 1, Documento: "informe", PasoOrden: 1,
		ClaveIdempotencia: "clave-firma-rrhh-000001", CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC,
	}}}
	servicio, err := NuevoServicioConsultaDetalleRRHHConSelectores(entorno.autoridad, entorno.emisor, sesion, entorno.reloj, fuente)
	if err != nil {
		t.Fatal(err)
	}
	d, err := servicio.Consultar(context.Background(), entorno.detalle)
	filas, activo := d.SelectoresRecuperacionFirmas()
	if err != nil || !activo || len(filas) != 1 || sesion.llamadasSelectores != 1 || sesion.filtro != fuente.filtro ||
		fuente.llamadas != 1 || fuente.actor != entorno.contexto.ActorRef() {
		t.Fatalf("selección nominal: err=%v, activo=%v, filas=%d, fuente=%d, sesión=%d", err, activo, len(filas), fuente.llamadas, sesion.llamadasSelectores)
	}
}

func TestDetalleRRHHSelectoresLegacyYAjenos(t *testing.T) {
	for _, tc := range []struct {
		nombre        string
		campos        []string
		ext           bool
		errorEsperado bool
	}{
		{nombre: "legacy", campos: nil},
		{nombre: "extendido sin fuente", campos: []string{"selectores_recuperacion_firmas"}, ext: true},
		{nombre: "campo ajeno", campos: []string{"otro"}, errorEsperado: true},
		{nombre: "campo adicional", campos: []string{"selectores_recuperacion_firmas", "otro"}, errorEsperado: true},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.emision.detalle.campos = tc.campos
			sesion := &sesionSelectoresPrueba{sesionConsultaRRHHPrueba: entorno.sesion}
			servicio, err := NuevoServicioConsultaDetalleRRHH(entorno.autoridad, entorno.emisor, sesion, entorno.reloj)
			if err != nil {
				t.Fatal(err)
			}
			d, err := servicio.Consultar(context.Background(), entorno.detalle)
			if tc.errorEsperado {
				if !errors.Is(err, ErrConsultaRRHHNoObservable) || sesion.llamadasSelectores != 0 || entorno.sesion.llamadasDetalle != 0 {
					t.Fatalf("campo ajeno llegó a CT: %v", err)
				}
				return
			}
			_, activo := d.SelectoresRecuperacionFirmas()
			if err != nil || activo != tc.ext || sesion.llamadasSelectores != btoi(tc.ext) {
				t.Fatalf("contrato de detalle alterado: err=%v activo=%v selectores=%d", err, activo, sesion.llamadasSelectores)
			}
		})
	}
}
func btoi(v bool) int {
	if v {
		return 1
	}
	return 0
}
