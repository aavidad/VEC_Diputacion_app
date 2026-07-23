package ports

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type consumidorCoberturaFuncion func(
	context.Context,
	OrdenConsumoCobertura,
) (ReciboConsumoCobertura, error)

func (f consumidorCoberturaFuncion) ConsumirCobertura(
	ctx context.Context,
	orden OrdenConsumoCobertura,
) (ReciboConsumoCobertura, error) {
	return f(ctx, orden)
}

func TestCoberturaOcultaFallosPrivadosDeCadaDependencia(t *testing.T) {
	privado := &errorPrivadoCobertura{detalle: "token=secreto dni=12345678Z"}
	casos := []struct {
		nombre   string
		publico  error
		preparar func(*entornoCoberturaPrueba)
	}{
		{
			nombre:  "publicador",
			publico: ErrPublicadorCatalogoCoberturaNoDisponible,
			preparar: func(e *entornoCoberturaPrueba) {
				e.publicador.publicar = func(
					context.Context,
					SolicitudConsultarCobertura,
				) (ConfirmacionPublicacionCobertura, error) {
					return ConfirmacionPublicacionCobertura{}, privado
				}
			},
		},
		{
			nombre:  "verificador",
			publico: ErrVerificadorCoberturaNoDisponible,
			preparar: func(e *entornoCoberturaPrueba) {
				e.verificador.verificar = func(
					context.Context,
					SolicitudVerificarRespuestaCobertura,
				) (ConfirmacionRespuestaCobertura, error) {
					return ConfirmacionRespuestaCobertura{}, privado
				}
			},
		},
		{
			nombre:  "consumidor",
			publico: ErrConsumoCoberturaNoDisponible,
			preparar: func(e *entornoCoberturaPrueba) {
				e.consumidor = nil
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoCoberturaPrueba(t)
			caso.preparar(&entorno)
			var consumidor ConsumidorCobertura = entorno.consumidor
			if caso.nombre == "consumidor" {
				consumidor = consumidorCoberturaFuncion(func(
					context.Context,
					OrdenConsumoCobertura,
				) (ReciboConsumoCobertura, error) {
					return ReciboConsumoCobertura{}, privado
				})
			}
			_, err := ConsultarCoberturaConFuente(
				context.Background(),
				entorno.fuente,
				entorno.verificador,
				entorno.publicador,
				consumidor,
				entorno.confianza,
				entorno.reloj,
				entorno.solicitud,
				time.Second,
			)
			var filtrado *errorPrivadoCobertura
			if !errors.Is(err, caso.publico) ||
				errors.Is(err, privado) ||
				errors.As(err, &filtrado) ||
				strings.Contains(err.Error(), "secreto") {
				t.Fatalf("causa privada filtrada: %v", err)
			}
		})
	}
}

func TestCoberturaTimeoutTotalIncluyeFuente(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	entorno.fuente.consultar = func(
		ctx context.Context,
		_ SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		<-ctx.Done()
		return ResultadoConsultaCobertura{}, ctx.Err()
	}
	_, err := ConsultarCoberturaConFuente(
		context.Background(),
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		entorno.consumidor,
		entorno.confianza,
		entorno.reloj,
		entorno.solicitud,
		50*time.Millisecond,
	)
	if !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout total no impuesto: %v", err)
	}
}

func TestCoberturaRechazaNulosTipadosSinInvocarlos(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	var fuente *fuenteCoberturaNula
	var verificador *verificadorCoberturaNulo
	var publicador *publicadorCoberturaNulo
	var consumidor *consumidorCoberturaNulo
	var reloj *relojCoberturaNulo
	casos := []struct {
		nombre      string
		fuente      FuenteComprobacionCobertura
		verificador VerificadorRespuestaCobertura
		publicador  PublicadorCatalogoCobertura
		consumidor  ConsumidorCobertura
		reloj       Reloj
	}{
		{
			"fuente",
			fuente,
			entorno.verificador,
			entorno.publicador,
			entorno.consumidor,
			entorno.reloj,
		},
		{
			"verificador",
			entorno.fuente,
			verificador,
			entorno.publicador,
			entorno.consumidor,
			entorno.reloj,
		},
		{
			"publicador",
			entorno.fuente,
			entorno.verificador,
			publicador,
			entorno.consumidor,
			entorno.reloj,
		},
		{
			"consumidor",
			entorno.fuente,
			entorno.verificador,
			entorno.publicador,
			consumidor,
			entorno.reloj,
		},
		{
			"reloj",
			entorno.fuente,
			entorno.verificador,
			entorno.publicador,
			entorno.consumidor,
			reloj,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := ConsultarCoberturaConFuente(
				context.Background(),
				caso.fuente,
				caso.verificador,
				caso.publicador,
				caso.consumidor,
				entorno.confianza,
				caso.reloj,
				entorno.solicitud,
				time.Second,
			)
			if !errors.Is(err, ErrPeticionFuenteCoberturaInvalida) {
				t.Fatalf("nulo tipado aceptado: %v", err)
			}
		})
	}
}

func TestCoberturaAdmiteViaNuevaSoloPorCatalogoPublicado(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave:       "consulta_nueva_fuente",
		Orden:       1,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "sistema_nuevo",
			DefinicionFuenteRef: "conector_sistema_nuevo_v1",
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo_dinamico_cobertura_01",
			Version:     1,
			PublicadoEn: entorno.solicitud.SolicitadaEn.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: entorno.solicitud.SolicitadaEn.Add(-time.Hour),
				Hasta: entorno.solicitud.SolicitadaEn.Add(time.Hour),
			},
			ProcedenciaRef: "acto_publicacion_catalogo_012345",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "via_nueva_sin_recompilar",
				Orden: 1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{
					comprobacion,
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	entorno.solicitud.Catalogo = catalogo.Identidad()
	entorno.solicitud.ViaClave = "via_nueva_sin_recompilar"
	entorno.solicitud.Comprobacion = comprobacion
	entorno.fuente.presentador.datos.BackendRef =
		comprobacion.Procedencia.DefinicionFuenteRef
	entorno.publicador.publicar = func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ConfirmacionPublicacionCobertura, error) {
		return NuevaConfirmacionPublicacionCobertura(
			"publicador_catalogo_cobertura_01",
			catalogo.Publicacion(),
			entorno.reloj.ahora,
		)
	}
	if _, err := entorno.consultar(context.Background()); err != nil {
		t.Fatalf("vía dinámica gobernada rechazada: %v", err)
	}
}

func TestObjetosSensiblesCoberturaSeFormateanRedactados(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	resultado := resultadoCoberturaFirmadoPrueba(t, entorno.solicitud)
	objetos := []any{
		resultado,
		resultado.preimagen,
		resultado.atestacion,
	}
	for _, objeto := range objetos {
		texto := fmt.Sprintf("%v %#v", objeto, objeto)
		registro := slog.AnyValue(objeto).Resolve().String()
		if strings.Contains(texto, entorno.solicitud.PeticionRef) ||
			strings.Contains(registro, entorno.solicitud.PeticionRef) ||
			!strings.Contains(texto, "REDACTAD") {
			t.Fatalf("formato no redactado: %s / %s", texto, registro)
		}
	}
}

type fuenteCoberturaNula struct{}

func (*fuenteCoberturaNula) PresentarAutoridadFuenteAnalisis(
	context.Context,
	DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	panic("no debe invocarse")
}
func (*fuenteCoberturaNula) ConsultarCobertura(
	context.Context,
	SolicitudConsultarCobertura,
) (ResultadoConsultaCobertura, error) {
	panic("no debe invocarse")
}

type verificadorCoberturaNulo struct{}

func (*verificadorCoberturaNulo) PresentarAutoridadFuenteAnalisis(
	context.Context,
	DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	panic("no debe invocarse")
}
func (*verificadorCoberturaNulo) VerificarRespuestaCobertura(
	context.Context,
	SolicitudVerificarRespuestaCobertura,
) (ConfirmacionRespuestaCobertura, error) {
	panic("no debe invocarse")
}

type publicadorCoberturaNulo struct{}

func (*publicadorCoberturaNulo) PresentarAutoridadFuenteAnalisis(
	context.Context,
	DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	panic("no debe invocarse")
}
func (*publicadorCoberturaNulo) ConsultarPublicacionCobertura(
	context.Context,
	SolicitudConsultarCobertura,
) (ConfirmacionPublicacionCobertura, error) {
	panic("no debe invocarse")
}

type consumidorCoberturaNulo struct{}

func (*consumidorCoberturaNulo) ConsumirCobertura(
	context.Context,
	OrdenConsumoCobertura,
) (ReciboConsumoCobertura, error) {
	panic("no debe invocarse")
}

type relojCoberturaNulo struct{}

func (*relojCoberturaNulo) Ahora() time.Time {
	panic("no debe invocarse")
}
