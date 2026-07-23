package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type generadorPeticionAnalisisDoble func(
	context.Context,
	TipoPeticionFuenteAnalisis,
) (string, error)

func (g generadorPeticionAnalisisDoble) NuevaReferenciaPeticionFuenteAnalisis(
	ctx context.Context,
	tipo TipoPeticionFuenteAnalisis,
) (string, error) {
	return g(ctx, tipo)
}

type selladorPeticionAnalisisDoble func(
	context.Context,
	PreimagenPeticionFuenteAnalisis,
) (string, error)

func (s selladorPeticionAnalisisDoble) SellarPeticionFuenteAnalisis(
	ctx context.Context,
	preimagen PreimagenPeticionFuenteAnalisis,
) (string, error) {
	return s(ctx, preimagen)
}

type relojFuenteAnalisisDoble func() time.Time

func (r relojFuenteAnalisisDoble) Ahora() time.Time {
	return r()
}

type fuentePresupuestariaDoble func(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error)

func (f fuentePresupuestariaDoble) ValidarRC(
	ctx context.Context,
	solicitud SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	return f(ctx, solicitud)
}

type calculadorCosteDoble func(
	context.Context,
	SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error)

func (c calculadorCosteDoble) CalcularCoste(
	ctx context.Context,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	return c(ctx, solicitud)
}

func TestValidarRCConFuenteDevuelveCopiaLigada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
	resultadoFuente, err := NuevoResultadoValidacionRC(
		solicitud,
		validacion,
		MotivoFuenteAnalisis{},
	)
	if err != nil {
		t.Fatal(err)
	}
	fuente := fuentePresupuestariaDoble(func(
		ctx context.Context,
		recibida SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		if ctx.Err() != nil || recibida.Validar() != nil {
			t.Fatal("la fuente recibió una solicitud inválida")
		}
		return resultadoFuente, nil
	})
	resultado, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		relojFijoFuenteAnalisis(inicio.Add(2*time.Second)),
		solicitud,
	)
	if err != nil || resultado.Validar() != nil ||
		resultado.Importe == validacion.Importe ||
		resultado.FechaRC == validacion.FechaRC {
		t.Fatalf("validar RC: %#v, %v", resultado, err)
	}
	*resultado.Importe = domain.Importe{Centimos: 1, Moneda: "EUR"}
	if validacion.Importe.Centimos == resultado.Importe.Centimos {
		t.Fatal("la copia comparte el importe de la fuente")
	}
}

func TestValidarRCConFuenteMaterializaSoloClaveI18N(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCNegativaPrueba(t, solicitud, inicio.Add(time.Second))
	motivo := motivoFuenteAnalisisPrueba(t)
	resultado, err := NuevoResultadoValidacionRC(solicitud, validacion, motivo)
	if err != nil {
		t.Fatal(err)
	}
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultado, nil
	})
	obtenida, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		relojFijoFuenteAnalisis(inicio.Add(2*time.Second)),
		solicitud,
	)
	if err != nil ||
		obtenida.Motivo != "contratacion_temporal.rc.no_requerida" ||
		!obtenida.HabilitaAvance() {
		t.Fatalf("motivo no materializado de forma segura: %#v, %v", obtenida, err)
	}
}

func TestFuentesDescartanValorCuandoLaDependenciaFalla(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitudRC, inicio.Add(time.Second))
	resultadoRC, _ := NuevoResultadoValidacionRC(
		solicitudRC,
		validacion,
		MotivoFuenteAnalisis{},
	)
	falloPrivado := errors.New("detalle privado del proveedor")
	fuente := fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultadoRC, falloPrivado
	})
	obtenidaRC, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		relojFijoFuenteAnalisis(inicio.Add(2*time.Second)),
		solicitudRC,
	)
	if !errors.Is(err, ErrFuentePresupuestariaNoDisponible) ||
		!errors.Is(err, falloPrivado) ||
		err.Error() != ErrFuentePresupuestariaNoDisponible.Error() ||
		obtenidaRC.HabilitaAvance() {
		t.Fatalf("la fuente filtró o aceptó valor con error: %#v, %v", obtenidaRC, err)
	}

	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	resultadoCoste, _ := NuevoResultadoCalculoCoste(
		solicitudCoste,
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
		inicio.Add(time.Second),
	)
	calculador := calculadorCosteDoble(func(
		context.Context,
		SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		return resultadoCoste, falloPrivado
	})
	obtenidoCoste, err := CalcularCosteConFuente(
		context.Background(),
		calculador,
		relojFijoFuenteAnalisis(inicio.Add(2*time.Second)),
		solicitudCoste,
	)
	if !errors.Is(err, ErrCalculadorCosteNoDisponible) ||
		!errors.Is(err, falloPrivado) ||
		err.Error() != ErrCalculadorCosteNoDisponible.Error() ||
		obtenidoCoste.datos != nil {
		t.Fatalf("el cálculo filtró o aceptó valor con error: %#v, %v", obtenidoCoste, err)
	}
}

func solicitudValidarRCPrueba(
	t *testing.T,
	instante time.Time,
) SolicitudValidarRC {
	t.Helper()
	solicitud, err := NuevaSolicitudValidarRC(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacionValidarRCPrueba(),
	)
	if err != nil {
		t.Fatalf("crear solicitud RC: %v", err)
	}
	return solicitud
}

func solicitudCalcularCostePrueba(
	t *testing.T,
	instante time.Time,
) SolicitudCalcularCoste {
	t.Helper()
	solicitud, err := NuevaSolicitudCalcularCoste(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_abcdefghij0123456789klmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacionCalcularCostePrueba(),
	)
	if err != nil {
		t.Fatalf("crear solicitud de coste: %v", err)
	}
	return solicitud
}

func preparacionValidarRCPrueba() PreparacionSolicitudValidarRC {
	fechaRC := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return PreparacionSolicitudValidarRC{
		OrganizacionRef:   "organizacion_diputacion_granada",
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 2,
		Entrada: domain.VinculoEntradaRC{
			Referencia:   "entrada_rc_0123456789",
			HuellaSHA256: strings.Repeat("a", 64),
		},
		Declaracion: domain.DeclaracionRC{
			Existe: true, Numero: "rc_2026_0123456789", Fecha: fechaRC,
			Importe:      domain.Importe{Centimos: 3_245_000, Moneda: "EUR"},
			DocumentoRef: "documento_rc_0123456789",
		},
	}
}

func preparacionCalcularCostePrueba() PreparacionSolicitudCalcularCoste {
	return PreparacionSolicitudCalcularCoste{
		OrganizacionRef:   "organizacion_diputacion_granada",
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 2,
		CategoriaRef:      "categoria_trabajo_social",
		GrupoSubgrupo:     "A2", ModalidadClave: "sustitucion",
		CausaClave: "incapacidad_temporal",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		Jornada: domain.JornadaCompletaDiezmilesimas,
	}
}

func validacionRCPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validadaEn time.Time,
) domain.ValidacionRC {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	fechaRC := datos.Declaracion.Fecha
	importe := datos.Declaracion.Importe
	return domain.ValidacionRC{
		Resultado:           domain.RCValidada,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           "fuente_presupuesto_0123456789",
		ReciboRef:           "recibo_presupuesto_0123456789",
		ValidadaEn:          validadaEn, FechaRC: &fechaRC,
		Numero: datos.Declaracion.Numero, Importe: &importe,
		DocumentoRef: datos.Declaracion.DocumentoRef,
	}
}

func validacionRCNegativaPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validadaEn time.Time,
) domain.ValidacionRC {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return domain.ValidacionRC{
		Resultado:           domain.RCNoRequerida,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           "fuente_presupuesto_0123456789",
		ReciboRef:           "recibo_presupuesto_0123456789",
		ValidadaEn:          validadaEn,
	}
}

func motivoFuenteAnalisisPrueba(t *testing.T) MotivoFuenteAnalisis {
	t.Helper()
	motivo, err := NuevoMotivoFuenteAnalisis(
		"catalogo_motivos_rc_0123456789",
		7,
		strings.Repeat("b", 64),
		"rc_no_requerida",
		"contratacion_temporal.rc.no_requerida",
		[]ParametroMotivoFuenteAnalisis{
			{Clave: ParametroMotivoCausa, Valor: "no_consta_rc"},
			{Clave: ParametroMotivoResultado, Valor: "no_requerida"},
		},
	)
	if err != nil {
		t.Fatalf("crear motivo: %v", err)
	}
	return motivo
}

func generadorFijoFuenteAnalisis(
	referencia string,
) generadorPeticionAnalisisDoble {
	return func(
		_ context.Context,
		_ TipoPeticionFuenteAnalisis,
	) (string, error) {
		return referencia, nil
	}
}

func selladorHMACFuenteAnalisisPrueba() selladorPeticionAnalisisDoble {
	return func(
		_ context.Context,
		preimagen PreimagenPeticionFuenteAnalisis,
	) (string, error) {
		contenido, err := preimagen.Bytes()
		if err != nil {
			return "", err
		}
		mac := hmac.New(sha256.New, []byte("clave-prueba-fuente-analisis"))
		_, _ = mac.Write(contenido)
		return dominioSelloPeticionAnalisis + hex.EncodeToString(mac.Sum(nil)), nil
	}
}

func relojFijoFuenteAnalisis(instante time.Time) relojFuenteAnalisisDoble {
	return func() time.Time {
		return instante
	}
}

func instanteFuenteAnalisisPrueba() time.Time {
	return time.Date(2026, 7, 23, 9, 0, 0, 123_456_000, time.UTC)
}
