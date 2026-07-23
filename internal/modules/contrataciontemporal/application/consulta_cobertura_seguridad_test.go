package application

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestServicioConsultaCoberturaLigaFuenteAlBackendGobernado(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.fuente.identidad = identidadCoberturaAplicacionPrueba(
		t,
		"fuente_cobertura_bolsa_012345",
		"backend_ajeno_a_la_definicion_01",
		claveEd25519CoberturaPrueba("fuente"),
		ports.RolFuenteCobertura,
	)
	var invocaciones atomic.Int32
	entorno.fuente.consultar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		invocaciones.Add(1)
		return ports.ResultadoConsultaCobertura{}, nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo de backend, recibido: %v", err)
	}
	if invocaciones.Load() != 0 {
		t.Fatal("no debe consultarse una fuente ajena a la definicion")
	}
}

func TestServicioConsultaCoberturaExigeAutoridadesSeparadas(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	claveComun := claveEd25519CoberturaPrueba("autoridad-comun")
	entorno.fuente.identidad = identidadCoberturaAplicacionPrueba(
		t,
		"autoridad_cobertura_compartida_01",
		entorno.solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
		claveComun,
		ports.RolFuenteCobertura,
	)
	entorno.verificador.identidad = identidadCoberturaAplicacionPrueba(
		t,
		"autoridad_cobertura_compartida_01",
		entorno.solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
		claveComun,
		ports.RolVerificadorCobertura,
	)
	entorno.publicador.identidad = identidadCoberturaAplicacionPrueba(
		t,
		"autoridad_cobertura_compartida_01",
		entorno.solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
		claveComun,
		ports.RolPublicadorCatalogoCobertura,
	)

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba separacion de funciones, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRechazaConfirmacionConFirmaAjena(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	claveAjena := claveEd25519CoberturaPrueba("verificador-ajeno")
	entorno.verificador.verificar = func(
		_ context.Context,
		solicitud ports.SolicitudVerificarRespuestaCobertura,
	) (ports.ConfirmacionRespuestaCobertura, error) {
		return verificarRespuestaCoberturaAplicacionPrueba(
			solicitud,
			entorno.verificador.identidad.AutoridadRef(),
			claveAjena,
			entorno.reloj.Ahora(),
		)
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo criptografico, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaNoFiltraErroresPrivados(
	t *testing.T,
) {
	casos := []struct {
		nombre   string
		publico  error
		preparar func(
			*entornoCoberturaAplicacionPrueba,
			*errorPrivadoCoberturaAplicacionPrueba,
		)
	}{
		{
			nombre:  "publicador",
			publico: ErrPublicadorCatalogoCoberturaNoDisponible,
			preparar: func(
				e *entornoCoberturaAplicacionPrueba,
				privado *errorPrivadoCoberturaAplicacionPrueba,
			) {
				e.publicador.publicar = func(
					context.Context,
					ports.SolicitudConsultarCobertura,
				) (ports.ConfirmacionPublicacionCobertura, error) {
					return ports.ConfirmacionPublicacionCobertura{}, privado
				}
			},
		},
		{
			nombre:  "fuente",
			publico: ErrFuenteCoberturaNoDisponible,
			preparar: func(
				e *entornoCoberturaAplicacionPrueba,
				privado *errorPrivadoCoberturaAplicacionPrueba,
			) {
				e.fuente.consultar = func(
					context.Context,
					ports.SolicitudConsultarCobertura,
				) (ports.ResultadoConsultaCobertura, error) {
					return ports.ResultadoConsultaCobertura{}, privado
				}
			},
		},
		{
			nombre:  "verificador",
			publico: ErrVerificadorCoberturaNoDisponible,
			preparar: func(
				e *entornoCoberturaAplicacionPrueba,
				privado *errorPrivadoCoberturaAplicacionPrueba,
			) {
				e.verificador.verificar = func(
					context.Context,
					ports.SolicitudVerificarRespuestaCobertura,
				) (ports.ConfirmacionRespuestaCobertura, error) {
					return ports.ConfirmacionRespuestaCobertura{}, privado
				}
			},
		},
		{
			nombre:  "consumidor",
			publico: ErrConsumoCoberturaNoDisponible,
			preparar: func(
				e *entornoCoberturaAplicacionPrueba,
				privado *errorPrivadoCoberturaAplicacionPrueba,
			) {
				e.consumidor.consumir = func(
					context.Context,
					ports.OrdenConsumoCobertura,
				) (ports.ReciboConsumoCobertura, error) {
					return ports.ReciboConsumoCobertura{}, privado
				}
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
			privado := &errorPrivadoCoberturaAplicacionPrueba{
				detalle: "dsn=secreto-interno-" + caso.nombre,
			}
			caso.preparar(entorno, privado)

			_, err := entorno.servicio.Consultar(
				context.Background(),
				entorno.solicitud,
			)

			if !errors.Is(err, caso.publico) {
				t.Fatalf("error publico inesperado: %v", err)
			}
			if errors.Is(err, privado) ||
				strings.Contains(err.Error(), privado.detalle) {
				t.Fatalf("se filtro la causa privada: %v", err)
			}
			var extraido *errorPrivadoCoberturaAplicacionPrueba
			if errors.As(err, &extraido) {
				t.Fatalf("la causa privada sigue accesible: %v", err)
			}
		})
	}
}

func TestNuevoServicioConsultaCoberturaRechazaDependenciaNulaTipada(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	var fuenteNula *fuenteCoberturaAplicacionPrueba

	_, err := NuevoServicioConsultaCobertura(
		fuenteNula,
		entorno.verificador,
		entorno.publicador,
		entorno.consumidor,
		entorno.autenticador,
		entorno.reloj,
		time.Second,
	)

	if !errors.Is(err, ErrServicioConsultaCoberturaInvalido) {
		t.Fatalf("se esperaba dependencia invalida, recibido: %v", err)
	}
}

func TestNuevoServicioConsultaCoberturaAcotaTiempoMaximo(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)

	_, err := NuevoServicioConsultaCobertura(
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		entorno.consumidor,
		entorno.autenticador,
		entorno.reloj,
		TiempoMaximoFuenteCobertura+time.Nanosecond,
	)

	if !errors.Is(err, ErrServicioConsultaCoberturaInvalido) {
		t.Fatalf("se esperaba limite temporal, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaRechazaOrganizacionNoGobernada(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	solicitud := entorno.solicitud
	solicitud.OrganizacionRef = "organizacion_ajena_0123456789"

	_, err := entorno.servicio.Consultar(context.Background(), solicitud)

	if !errors.Is(err, ports.ErrPeticionFuenteCoberturaInvalida) {
		t.Fatalf("se esperaba rechazo de organizacion, recibido: %v", err)
	}
}
