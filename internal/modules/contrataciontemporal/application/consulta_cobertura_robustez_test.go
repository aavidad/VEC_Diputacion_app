package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestServicioConsultaCoberturaRechazaCoordenadasReselladas(
	t *testing.T,
) {
	casos := []struct {
		nombre  string
		alterar func(
			*ports.DatosResultadoConsultaCobertura,
		)
	}{
		{
			nombre: "peticion",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.PeticionRef = "peticion_cobertura_distinta_01"
			},
		},
		{
			nombre: "huella_peticion",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.HuellaPeticionSHA256 = strings.Repeat("0", 64)
			},
		},
		{
			nombre: "organizacion",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.OrganizacionRef = "organizacion_cobertura_distinta_01"
			},
		},
		{
			nombre: "expediente",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.ExpedienteRef = "expediente_cobertura_distinto_01"
			},
		},
		{
			nombre: "version",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.VersionExpediente++
			},
		},
		{
			nombre: "via",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.ViaClave = "via_cobertura_distinta"
			},
		},
		{
			nombre: "procedencia",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.ProcedenciaClave = "procedencia_distinta"
			},
		},
		{
			nombre: "categoria",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.CategoriaRef = "categoria_cobertura_distinta_01"
			},
		},
		{
			nombre: "periodo",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.Periodo.Fin = d.Periodo.Fin.Add(-24 * time.Hour)
			},
		},
		{
			nombre: "comprobacion",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.Comprobacion.Clave = "comprobacion_distinta"
			},
		},
		{
			nombre: "fuente",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.Comprobacion.FuenteRef =
					"fuente_cobertura_distinta_012345"
			},
		},
		{
			nombre: "definicion_fuente",
			alterar: func(d *ports.DatosResultadoConsultaCobertura) {
				d.DefinicionFuenteRef =
					"definicion_fuente_distinta_012345"
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
			entorno.fuente.consultar = func(
				_ context.Context,
				solicitud ports.SolicitudConsultarCobertura,
			) (ports.ResultadoConsultaCobertura, error) {
				return resultadoCoberturaAplicacionPrueba(
					t,
					solicitud,
					caso.alterar,
				), nil
			}

			_, err := entorno.servicio.Consultar(
				context.Background(),
				entorno.solicitud,
			)

			if !errors.Is(
				err,
				ports.ErrResultadoFuenteCoberturaNoConfiable,
			) {
				t.Fatalf("coordenada alterada aceptada: %v", err)
			}
		})
	}
}

func TestServicioConsultaCoberturaReplayConcurrenteProduceUnSoloEfecto(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	resultado := resultadoCoberturaAplicacionPrueba(
		t,
		entorno.solicitud,
		nil,
	)
	entorno.fuente.consultar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		return resultado, nil
	}
	const concurrencia = 24
	var espera sync.WaitGroup
	errores := make(chan error, concurrencia)
	espera.Add(concurrencia)
	for indice := 0; indice < concurrencia; indice++ {
		go func() {
			defer espera.Done()
			_, err := entorno.servicio.Consultar(
				context.Background(),
				entorno.solicitud,
			)
			errores <- err
		}()
	}
	espera.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("replay concurrente rechazado: %v", err)
		}
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.registros) != 1 ||
		len(entorno.consumidor.ordenes) != concurrencia {
		t.Fatalf(
			"efecto concurrente incorrecto: registros=%d ordenes=%d",
			len(entorno.consumidor.registros),
			len(entorno.consumidor.ordenes),
		)
	}
}

func TestServicioConsultaCoberturaPruebaPertenenciaExactaAlCatalogo(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	solicitud := entorno.solicitud
	solicitud.Comprobacion = domain.ComprobacionExigibleCobertura{
		Clave:       "comprobacion_ausente_catalogo",
		Orden:       2,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
		},
	}
	var consultas atomic.Int32
	entorno.fuente.consultar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		consultas.Add(1)
		return ports.ResultadoConsultaCobertura{}, nil
	}

	_, err := entorno.servicio.Consultar(context.Background(), solicitud)

	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se esperaba rechazo de catalogo, recibido: %v", err)
	}
	if consultas.Load() != 0 {
		t.Fatal("no debe consultarse la fuente para una regla no publicada")
	}
}

func TestServicioConsultaCoberturaTiempoMaximoIncluyeLaFuente(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	entorno.reconstruirServicio(t, 5*time.Millisecond)
	entorno.fuente.consultar = func(
		_ context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		time.Sleep(15 * time.Millisecond)
		return resultadoCoberturaAplicacionPrueba(t, solicitud, nil), nil
	}

	_, err := entorno.servicio.Consultar(
		context.Background(),
		entorno.solicitud,
	)

	if !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("se esperaba plazo total agotado, recibido: %v", err)
	}
}

func TestServicioConsultaCoberturaPriorizaCancelacionSobreErrorPrivado(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	privado := &errorPrivadoCoberturaAplicacionPrueba{
		detalle: "credencial-interna-no-publicable",
	}
	entorno.fuente.consultar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		cancelar()
		return ports.ResultadoConsultaCobertura{}, privado
	}

	_, err := entorno.servicio.Consultar(ctx, entorno.solicitud)

	if !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, context.Canceled) ||
		errors.Is(err, privado) ||
		strings.Contains(err.Error(), privado.detalle) {
		t.Fatalf("cancelacion o redaccion incorrectas: %v", err)
	}
}
