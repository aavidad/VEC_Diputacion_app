package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type fuenteRCCredencialCambianteAplicacionPrueba struct {
	base        fuenteRCAplicacionPrueba
	resultado   ports.ResultadoValidacionRC
	presentadas int
}

func (f *fuenteRCCredencialCambianteAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	f.presentadas++
	autoridad := f.base.autoridad
	if f.presentadas > 1 {
		autoridad.datos.Serie++
	}
	return autoridad.presentar(desafio)
}

func (f *fuenteRCCredencialCambianteAplicacionPrueba) ValidarRC(
	context.Context,
	ports.SolicitudValidarRC,
) (ports.ResultadoValidacionRC, error) {
	return f.resultado, nil
}

type fuenteRCCancelaRevalidacionAplicacionPrueba struct {
	base        fuenteRCAplicacionPrueba
	resultado   ports.ResultadoValidacionRC
	cancelar    context.CancelFunc
	presentadas int
}

func (f *fuenteRCCancelaRevalidacionAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	f.presentadas++
	if f.presentadas > 1 {
		f.cancelar()
		return ports.PresentacionAutoridadFuenteAnalisis{},
			context.Canceled
	}
	return f.base.autoridad.presentar(desafio)
}

func (f *fuenteRCCancelaRevalidacionAplicacionPrueba) ValidarRC(
	context.Context,
	ports.SolicitudValidarRC,
) (ports.ResultadoValidacionRC, error) {
	return f.resultado, nil
}

func TestPreparadorArtefactoCoordinaRevalidacionFueraDePorts(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-revalidacion-aplicacion-sintetica",
	)
	preparador := nuevoPreparadorArtefactoAnalisisO3AplicacionPrueba(
		t,
		escenario.instante,
	)
	capacidad, correcto := preparador.(*CapacidadPrepararArtefactoAnalisisO3)
	if !correcto {
		t.Fatal("la composición de prueba no devolvió la capacidad real")
	}
	solicitud := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      escenario.registrar.ArtefactoRef,
		OrganizacionRef:   escenario.registrar.OrganizacionRef,
		ExpedienteRef:     escenario.registrar.ExpedienteRef,
		VersionExpediente: escenario.registrar.VersionEsperada,
		DatosFuncionales:  escenario.registrar.DatosFuncionales,
		SolicitadaEn:      escenario.instante,
	}
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(
			context.Background(),
			solicitud,
		)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := capacidad.fuenteRC.ValidarRC(
		context.Background(),
		solicitudes.ValidacionRC,
	)
	if err != nil {
		t.Fatal(err)
	}
	base, correcto := capacidad.fuenteRC.(fuenteRCAplicacionPrueba)
	if !correcto {
		t.Fatal("la fuente de prueba no permite simular rotación")
	}
	cambiante := &fuenteRCCredencialCambianteAplicacionPrueba{
		base:      base,
		resultado: resultado,
	}
	capacidad.fuenteRC = cambiante

	if _, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	); !errors.Is(err, ports.ErrArtefactoAnalisisNoConfiable) ||
		cambiante.presentadas != 2 {
		t.Fatalf("la revalidación aceptó otra credencial: %v", err)
	}
}

func TestPreparadorArtefactoConservaCancelacionDuranteRevalidacion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-revalidacion-cancelada-sintetica",
	)
	preparador := nuevoPreparadorArtefactoAnalisisO3AplicacionPrueba(
		t,
		escenario.instante,
	)
	capacidad := preparador.(*CapacidadPrepararArtefactoAnalisisO3)
	solicitud := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      escenario.registrar.ArtefactoRef,
		OrganizacionRef:   escenario.registrar.OrganizacionRef,
		ExpedienteRef:     escenario.registrar.ExpedienteRef,
		VersionExpediente: escenario.registrar.VersionEsperada,
		DatosFuncionales:  escenario.registrar.DatosFuncionales,
		SolicitadaEn:      escenario.instante,
	}
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(
			context.Background(),
			solicitud,
		)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := capacidad.fuenteRC.ValidarRC(
		context.Background(),
		solicitudes.ValidacionRC,
	)
	if err != nil {
		t.Fatal(err)
	}
	base := capacidad.fuenteRC.(fuenteRCAplicacionPrueba)
	ctx, cancelar := context.WithCancel(context.Background())
	capacidad.fuenteRC =
		&fuenteRCCancelaRevalidacionAplicacionPrueba{
			base:      base,
			resultado: resultado,
			cancelar:  cancelar,
		}

	if _, err := capacidad.PrepararArtefactoAnalisis(
		ctx,
		solicitud,
	); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ports.ErrArtefactoAnalisisNoDisponible) {
		t.Fatalf("la cancelación quedó degradada: %v", err)
	}
}
