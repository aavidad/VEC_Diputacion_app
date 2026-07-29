package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type contextoCanceladoTrasDecodificarPrueba struct {
	context.Context
	err error
}

func (c *contextoCanceladoTrasDecodificarPrueba) Err() error {
	return c.err
}

type cuerpoCanceladorConsultaRRHHPrueba struct {
	lector     *strings.Reader
	cancelar   func()
	finalizado bool
}

func (c *cuerpoCanceladorConsultaRRHHPrueba) Read(
	destino []byte,
) (int, error) {
	leidos, err := c.lector.Read(destino)
	if errors.Is(err, io.EOF) && !c.finalizado {
		c.finalizado = true
		c.cancelar()
	}
	return leidos, err
}

func (*cuerpoCanceladorConsultaRRHHPrueba) Close() error {
	return nil
}

func TestManejadoresConsultaRRHHCancelacionTrasDecodificarNoInvocaConsultor(
	t *testing.T,
) {
	t.Parallel()
	for _, fallo := range []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{
			nombre: "cancelación",
			err:    context.Canceled,
			estado: http.StatusRequestTimeout,
			codigo: "peticion_cancelada",
		},
		{
			nombre: "plazo",
			err:    context.DeadlineExceeded,
			estado: http.StatusGatewayTimeout,
			codigo: "plazo_agotado",
		},
	} {
		fallo := fallo
		for _, superficie := range []string{"cuadro", "detalle"} {
			superficie := superficie
			t.Run(fallo.nombre+"/"+superficie, func(t *testing.T) {
				t.Parallel()
				var (
					manejador       http.Handler
					ruta            string
					contenido       string
					llamadas        *int
					errConstruccion error
				)
				switch superficie {
				case "cuadro":
					consultor := &consultorCuadroRRHHPrueba{
						pagina: paginaRRHHPrueba(),
					}
					manejador, errConstruccion =
						NuevoManejadorConsultaCuadroRRHH(consultor)
					ruta = RutaConsultaCuadroRRHH
					contenido = cuerpoCuadroRRHHPrueba()
					llamadas = &consultor.llamadas
				case "detalle":
					consultor := &consultorDetalleRRHHPrueba{
						detalle: detalleRRHHPrueba(),
					}
					manejador, errConstruccion =
						NuevoManejadorConsultaDetalleRRHH(consultor)
					ruta = RutaConsultaDetalleRRHH
					contenido = cuerpoDetalleRRHHPrueba()
					llamadas = &consultor.llamadas
				default:
					t.Fatalf("superficie no contemplada: %s", superficie)
				}
				if errConstruccion != nil {
					t.Fatalf("crear manejador: %v", errConstruccion)
				}
				contexto := &contextoCanceladoTrasDecodificarPrueba{
					Context: context.Background(),
				}
				cuerpo := &cuerpoCanceladorConsultaRRHHPrueba{
					lector: strings.NewReader(contenido),
					cancelar: func() {
						contexto.err = fallo.err
					},
				}
				peticion := nuevaPeticionConsultaRRHHPrueba(
					ruta,
					contenido,
				).WithContext(contexto)
				peticion.Body = cuerpo
				respuesta := httptest.NewRecorder()

				manejador.ServeHTTP(respuesta, peticion)

				if !cuerpo.finalizado || *llamadas != 0 {
					t.Fatalf(
						"decodificación=%t llamadas=%d",
						cuerpo.finalizado,
						*llamadas,
					)
				}
				var salida envoltorioErrorConsultaRRHH
				if err := json.Unmarshal(
					respuesta.Body.Bytes(),
					&salida,
				); err != nil {
					t.Fatalf("decodificar error público: %v", err)
				}
				if respuesta.Code != fallo.estado ||
					salida.Error.Codigo != fallo.codigo {
					t.Fatalf(
						"estado=%d código=%q cuerpo=%s",
						respuesta.Code,
						salida.Error.Codigo,
						respuesta.Body,
					)
				}
			})
		}
	}
}
