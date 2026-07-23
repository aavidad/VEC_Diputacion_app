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
	entorno.autenticador.identidades[ports.RolFuenteCobertura] =
		entorno.fuente.identidad
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

func TestServicioConsultaCoberturaUsaRelojPosteriorACadaPresentacion(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	incremento := 100 * time.Millisecond
	esperados := map[ports.RolAutoridadFuenteAnalisis]time.Time{
		ports.RolFuenteCobertura: entorno.inicio.Add(
			2*time.Second + incremento,
		),
		ports.RolVerificadorCobertura: entorno.inicio.Add(
			2*time.Second + 2*incremento,
		),
		ports.RolPublicadorCatalogoCobertura: entorno.inicio.Add(
			2*time.Second + 3*incremento,
		),
	}
	var presentaciones atomic.Int32
	avanzar := func(
		identidad ports.IdentidadAutoridadFuenteAnalisis,
	) func(
		context.Context,
		ports.DesafioAutoridadFuenteAnalisis,
	) (ports.PresentacionAutoridadFuenteAnalisis, error) {
		return func(
			_ context.Context,
			_ ports.DesafioAutoridadFuenteAnalisis,
		) (ports.PresentacionAutoridadFuenteAnalisis, error) {
			presentaciones.Add(1)
			entorno.reloj.mu.Lock()
			entorno.reloj.ahora = entorno.reloj.ahora.Add(incremento)
			entorno.reloj.mu.Unlock()
			return presentacionEstructuralCoberturaPrueba(identidad)
		}
	}
	entorno.fuente.presentar = avanzar(entorno.fuente.identidad)
	entorno.verificador.presentar = avanzar(entorno.verificador.identidad)
	entorno.publicador.presentar = avanzar(entorno.publicador.identidad)
	var verificaciones atomic.Int32
	entorno.autenticador.antes = func(
		rol ports.RolAutoridadFuenteAnalisis,
		comprobadaEn time.Time,
	) error {
		if !comprobadaEn.Equal(esperados[rol]) {
			t.Fatalf(
				"%s se verificó con instante previo: %s",
				rol,
				comprobadaEn,
			)
		}
		verificaciones.Add(1)
		return nil
	}

	if _, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	if presentaciones.Load() != 3 || verificaciones.Load() != 4 {
		t.Fatalf(
			"presentaciones=%d verificaciones=%d",
			presentaciones.Load(),
			verificaciones.Load(),
		)
	}
}

func TestServicioConsultaCoberturaRechazaAutoridadCaducadaDurantePresentacion(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	validaHasta := entorno.inicio.Add(4 * time.Second)
	entorno.fuente.presentar = func(
		_ context.Context,
		_ ports.DesafioAutoridadFuenteAnalisis,
	) (ports.PresentacionAutoridadFuenteAnalisis, error) {
		entorno.reloj.fijar(validaHasta)
		return presentacionEstructuralCoberturaPrueba(
			entorno.fuente.identidad,
		)
	}
	entorno.autenticador.antes = func(
		rol ports.RolAutoridadFuenteAnalisis,
		comprobadaEn time.Time,
	) error {
		if rol == ports.RolFuenteCobertura &&
			!comprobadaEn.Before(validaHasta) {
			return ports.ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	var consultas atomic.Int32
	entorno.fuente.consultar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		consultas.Add(1)
		return ports.ResultadoConsultaCobertura{}, nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("autoridad caducada durante presentación aceptada: %v", err)
	}
	if consultas.Load() != 0 {
		t.Fatal("la autoridad caducada alcanzó la fuente funcional")
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
	entorno.autenticador.identidades[ports.RolFuenteCobertura] =
		entorno.fuente.identidad
	entorno.autenticador.identidades[ports.RolVerificadorCobertura] =
		entorno.verificador.identidad
	entorno.autenticador.identidades[ports.RolPublicadorCatalogoCobertura] =
		entorno.publicador.identidad

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
