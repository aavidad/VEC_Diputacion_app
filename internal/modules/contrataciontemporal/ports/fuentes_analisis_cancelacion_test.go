package ports

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestContextoSeCompruebaTrasSelladorYRelojFinal(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	t.Run("sellador", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		sellador := selladorPeticionAnalisisDoble(func(
			context.Context,
			PreimagenPeticionFuenteAnalisis,
		) (string, error) {
			cancelar()
			return dominioSelloPeticionAnalisis + strings.Repeat("a", 64), nil
		})
		_, err := NuevaSolicitudValidarRC(
			ctx,
			generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
			sellador,
			relojFijoFuenteAnalisis(inicio),
			preparacionValidarRCPrueba(),
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("no comprobó ctx tras sellador: %v", err)
		}
	})

	t.Run("reloj final", func(t *testing.T) {
		solicitud := solicitudValidarRCPrueba(t, inicio)
		validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
		resultado, err := NuevoResultadoValidacionRC(
			solicitud,
			validacion,
			MotivoFuenteAnalisis{},
		)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancelar := context.WithCancel(context.Background())
		fuente := fuentePresupuestariaDoble(func(
			context.Context,
			SolicitudValidarRC,
		) (ResultadoValidacionRC, error) {
			return resultado, nil
		})
		_, err = ValidarRCConFuente(
			ctx,
			fuente,
			relojFuenteAnalisisDoble(func() time.Time {
				cancelar()
				return inicio.Add(2 * time.Second)
			}),
			solicitud,
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("no comprobó ctx tras reloj final: %v", err)
		}
	})
}

func TestSolicitudesDetectanAdulteracionInternaDePreimagenYSello(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	copiaRC := solicitudRC
	datosRC := *solicitudRC.datos
	copiaRC.datos = &datosRC
	copiaRC.datos.VersionExpediente++
	if copiaRC.Validar() == nil {
		t.Fatal("solicitud RC alterada aceptada")
	}
	copiaRC = solicitudRC
	datosRC = *solicitudRC.datos
	copiaRC.datos = &datosRC
	copiaRC.datos.HuellaPeticionHMAC =
		dominioSelloPeticionAnalisis + strings.Repeat("c", 64)
	if copiaRC.Validar() == nil {
		t.Fatal("solicitud RC con sello sustituido aceptada")
	}

	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	copiaCoste := solicitudCoste
	datosCoste := *solicitudCoste.datos
	copiaCoste.datos = &datosCoste
	copiaCoste.datos.Jornada = 5_000
	if copiaCoste.Validar() == nil {
		t.Fatal("solicitud de coste alterada aceptada")
	}
	copiaCoste = solicitudCoste
	datosCoste = *solicitudCoste.datos
	copiaCoste.datos = &datosCoste
	copiaCoste.preimagen = append([]byte(nil), solicitudCoste.preimagen...)
	copiaCoste.preimagen[0] ^= 1
	if copiaCoste.Validar() == nil {
		t.Fatal("preimagen de coste alterada aceptada")
	}
}

func TestDependenciasNulasTipadasFallanCerrado(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	var fuente *fuentePresupuestariaNulaPrueba
	if _, err := ValidarRCConFuente(
		context.Background(),
		fuente,
		relojFijoFuenteAnalisis(inicio),
		solicitudValidarRCPrueba(t, inicio),
	); !errors.Is(err, ErrPeticionFuenteAnalisisInvalida) {
		t.Fatalf("fuente nula tipada aceptada: %v", err)
	}
}

func TestSalidasInvalidasDeInfraestructuraNoCreanCausaNula(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	_, err := NuevaSolicitudValidarRC(
		context.Background(),
		generadorFijoFuenteAnalisis("referencia_no_generada_por_el_puerto"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacionValidarRCPrueba(),
	)
	if !errors.Is(err, ErrInfraestructuraFuenteAnalisisNoDisponible) {
		t.Fatalf("salida inválida no falló cerrada: %v", err)
	}
}

type fuentePresupuestariaNulaPrueba struct{}

func (*fuentePresupuestariaNulaPrueba) ValidarRC(
	context.Context,
	SolicitudValidarRC,
) (ResultadoValidacionRC, error) {
	panic("no debe invocarse")
}
