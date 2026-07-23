package ports

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestPuenteArtefactoAnalisisIntegraEvidenciaO3VerificadaYConsumida(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararYConsumirArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	datos, err := artefacto.DatosPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := DerivarAnalisisDesdeArtefacto(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	pruebas, err := artefacto.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if len(consumidor.recibos) != 1 ||
		consumidor.llamadas != 1 ||
		datos.ArtefactoHuellaSHA256 == "" ||
		datos.ConsumoRCRef == "" ||
		datos.ConsumoCosteRef == "" ||
		pruebas.ReciboConsumoCoste == nil ||
		analisis.ValidacionRC.FuenteRef != datos.FuenteRCRef ||
		analisis.CostePrevisto == datos.CostePrevisto ||
		analisis.CostePrevisto.Centimos != datos.CostePrevisto.Centimos {
		t.Fatal("el puente perdió evidencia verificada o consumo durable")
	}
	for _, nombre := range []string{
		"ResultadoRC", "ConfirmacionRC", "OrdenConsumoRC",
		"ReciboConsumoRC", "Credencial", "Raiz", "Autoridad",
	} {
		if _, existe := reflect.TypeOf(
			SolicitudPrepararArtefactoAnalisis{},
		).FieldByName(nombre); existe {
			t.Fatalf("el DTO acepta autoridad mediante %s", nombre)
		}
	}
}

func TestPuenteArtefactoAnalisisRepeticionExactaEsIdempotente(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	primero := prepararYConsumirArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	segundo := prepararYConsumirArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	datosPrimero, _ := primero.DatosPara(solicitud)
	datosSegundo, _ := segundo.DatosPara(solicitud)
	pruebasPrimero, err := primero.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	pruebasSegundo, err := segundo.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if datosPrimero.ArtefactoHuellaSHA256 !=
		datosSegundo.ArtefactoHuellaSHA256 ||
		datosPrimero.ConsumoRCRef != datosSegundo.ConsumoRCRef ||
		datosPrimero.ConsumoCosteRef != datosSegundo.ConsumoCosteRef ||
		len(consumidor.recibos) != 1 ||
		consumidor.llamadas != 2 ||
		pruebasPrimero.ReciboConsumoConjunto == nil ||
		pruebasSegundo.ReciboConsumoConjunto == nil ||
		!recibosConsumoConjuntoIgualesO3(
			*pruebasPrimero.ReciboConsumoConjunto,
			*pruebasSegundo.ReciboConsumoConjunto,
		) {
		t.Fatal("la repetición exacta no reutilizó los consumos")
	}
}

func TestReciboConsumoConjuntoRechazaConfirmacionParcialRCConCoste(
	t *testing.T,
) {
	solicitud, capacidad, consumidor := capacidadArtefactoAnalisisPrueba(t)
	artefacto, err := capacidad.PrepararArtefactoAnalisis(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := artefacto.pruebas.ordenConjunto
	datosOrden, err := orden.Datos()
	if err != nil || datosOrden.OrdenCoste == nil {
		t.Fatal("la orden de prueba debe incluir RC y coste")
	}
	reciboRC, err := NuevoReciboConsumoRespuestaFuenteAnalisis(
		datosOrden.OrdenRC,
		"consumo_rc_parcial_prohibido_012345",
		consumidor.instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoReciboConsumoConjuntoFuentesAnalisisO3(
		orden,
		"consumo_conjunto_parcial_prohibido_012345",
		reciboRC,
		nil,
		consumidor.instante,
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("se aceptó un recibo RC sin su coste indivisible: %v", err)
	}
	if len(consumidor.recibos) != 0 || consumidor.llamadas != 0 {
		t.Fatal("la validación del recibo parcial produjo un efecto durable")
	}
}

func TestPruebasArtefactoAnalisisClonaConfirmacionPublicacion(
	t *testing.T,
) {
	solicitud, capacidad, _ := capacidadArtefactoAnalisisPrueba(t)
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	inicio := instanteFuenteAnalisisPrueba()
	validacion := validacionRCNegativaPrueba(
		t,
		solicitudes.ValidacionRC,
		inicio.Add(time.Second),
	)
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	resultado := resultadoRCFirmadoPrueba(
		t,
		solicitudes.ValidacionRC,
		validacion,
		motivoFuenteAnalisisPrueba(t),
		metadatos,
	)
	capacidad.fuenteRC = fuentePresupuestariaDoble(func(
		context.Context,
		SolicitudValidarRC,
	) (ResultadoValidacionRC, error) {
		return resultado, nil
	})
	capacidad.publicador = verificadorPublicacionPrueba(
		inicio.Add(2500 * time.Millisecond),
	)
	artefacto := prepararYConsumirArtefactoAnalisisPrueba(
		t,
		capacidad,
		solicitud,
	)
	primera, err := artefacto.PruebasParaO3(solicitud)
	if err != nil || primera.ConfirmacionPublicacion == nil {
		t.Fatalf("confirmación de publicación ausente: %v", err)
	}
	datosOriginales, err := primera.ConfirmacionPublicacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	primera.ConfirmacionPublicacion.datos.PublicacionRef =
		"publicacion_adulterada_por_consumidor_012345"

	segunda, err := artefacto.PruebasParaO3(solicitud)
	if err != nil || segunda.ConfirmacionPublicacion == nil {
		t.Fatalf("la mutación externa contaminó el artefacto: %v", err)
	}
	datosSegunda, err := segunda.ConfirmacionPublicacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datosSegunda != datosOriginales {
		t.Fatalf(
			"PruebasParaO3 devolvió un alias mutable: %#v",
			datosSegunda,
		)
	}
	if _, err := artefacto.DatosPara(solicitud); err != nil {
		t.Fatalf("la mutación externa alteró el artefacto original: %v", err)
	}
}
