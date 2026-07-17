package application

import (
	"context"
	"errors"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestConsultaInternaConvocatoriaComponePDPYRepositorioEnOrden(t *testing.T) {
	version := nuevaVersionConsultaConvocatoriaAplicacionPrueba(t)
	consulta := &consultaVersionConvocatoriaPrueba{version: version}
	exigidor := &exigidorConsultaConvocatoriaPrueba{instante: instantePanelInternoPrueba}
	servicio, err := NuevoServicioConsultaVersionConvocatoria(
		consulta, exigidor, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := nuevaOrdenConsultaConvocatoriaAplicacionPrueba(t, version)
	exigidor.observar = func(recurso dominiovec.RecursoAutorizable, motivo string) {
		if consulta.llamadas != 0 || recurso.Referencia != version.Referencia() ||
			motivo != motivoConsultaInternaConvocatoria {
			t.Fatalf("PDP fuera de orden o recurso: repo=%d recurso=%+v motivo=%q",
				consulta.llamadas, recurso, motivo)
		}
	}
	consulta.antes = func(solicitud puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada) {
		if exigidor.llamadas != 1 || solicitud.Validar() != nil ||
			solicitud.Selector != orden.Selector || solicitud.IncluirInstanciaFlujo {
			t.Fatalf("solicitud durable inesperada: pdp=%d solicitud=%+v", exigidor.llamadas, solicitud)
		}
	}

	resultado, err := servicio.ObtenerExacta(context.Background(), orden)
	if err != nil || resultado.Version.Referencia() != version.Referencia() ||
		consulta.llamadas != 1 || exigidor.llamadas != 1 {
		t.Fatalf("composicion productiva rechazada: resultado=%+v error=%v", resultado, err)
	}
}

func TestConsultaInternaConvocatoriaRechazaSuperficieExternaAntesDelPDP(t *testing.T) {
	version := nuevaVersionConsultaConvocatoriaAplicacionPrueba(t)
	consulta := &consultaVersionConvocatoriaPrueba{version: version}
	exigidor := &exigidorConsultaConvocatoriaPrueba{instante: instantePanelInternoPrueba}
	servicio, err := NuevoServicioConsultaVersionConvocatoria(
		consulta, exigidor, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := nuevaOrdenConsultaConvocatoriaAplicacionPrueba(t, version)
	orden.ContextoActor, orden.VinculoAutenticacionActor = nuevoContextoYVinculoPanelPrueba(
		t, dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
		dominiovec.SuperficieAutenticacionExternaPersonalV1,
	)

	_, err = servicio.ObtenerExacta(context.Background(), orden)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
		!errors.Is(err, ErrOrdenConsultaConvocatoriaInvalida) ||
		exigidor.llamadas != 0 || consulta.llamadas != 0 {
		t.Fatalf("superficie externa alcanzo PDP/repositorio: %v", err)
	}
}

func TestConsultaInternaConvocatoriaRechazaResultadoNoLigado(t *testing.T) {
	version := nuevaVersionConsultaConvocatoriaAplicacionPrueba(t)
	consulta := &consultaVersionConvocatoriaPrueba{
		version: version,
		manipular: func(resultado *puertosbolsa.ResultadoConsultaVersionConvocatoria) {
			resultado.AuditoriaRef = resultado.ConsumoAutorizacionRef
		},
	}
	exigidor := &exigidorConsultaConvocatoriaPrueba{instante: instantePanelInternoPrueba}
	servicio, err := NuevoServicioConsultaVersionConvocatoria(
		consulta, exigidor, relojPanelInternoPrueba{ahora: instantePanelInternoPrueba},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = servicio.ObtenerExacta(
		context.Background(), nuevaOrdenConsultaConvocatoriaAplicacionPrueba(t, version),
	)
	if !errors.Is(err, ErrResultadoConsultaConvocatoriaInseguro) ||
		consulta.llamadas != 1 || exigidor.llamadas != 1 {
		t.Fatalf("resultado no ligado aceptado: %v", err)
	}
}

func TestConsultaInternaConvocatoriaFallaCerradaSinDependencias(t *testing.T) {
	if _, err := NuevoServicioConsultaVersionConvocatoria(nil, nil, nil); !errors.Is(
		err, ErrServicioConsultaConvocatoriaInvalido,
	) {
		t.Fatalf("constructor permisivo: %v", err)
	}
}
