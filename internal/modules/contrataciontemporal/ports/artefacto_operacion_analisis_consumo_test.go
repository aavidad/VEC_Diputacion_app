package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestHuellaAnalisisRehidratadoReutilizaExactamenteCanonO3(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
	esperada, err := HuellaAnalisisDerivadoO3(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	analisis, err := DerivarAnalisisDesdeArtefacto(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	analisis.ActuacionRegistro = &domain.VinculoActuacionAnalisis{
		Secuencia:         2,
		VersionExpediente: 2,
		AccionClave:       domain.ClaveCatalogo(AccionRegistrarAnalisis),
		FaseDestino:       "analisis_rrhh",
		ReciboRef:         "recibo_confirmacion_analisis_o3_01",
	}
	obtenida, err := HuellaAnalisisRRHHRehidratadoO3(analisis)
	if err != nil {
		t.Fatal(err)
	}
	if obtenida != esperada {
		t.Fatalf("el canon rehidratado divergió de O3: %s != %s", obtenida, esperada)
	}
}

func TestHuellaAnalisisRehidratadoRechazaContenidoInvalido(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
	analisis, err := DerivarAnalisisDesdeArtefacto(solicitud, artefacto)
	if err != nil {
		t.Fatal(err)
	}
	analisis.CategoriaRef = ""
	if huella, err := HuellaAnalisisRRHHRehidratadoO3(analisis); !errors.Is(
		err,
		ErrArtefactoAnalisisNoConfiable,
	) || huella != "" {
		t.Fatalf("análisis inválido aceptado: huella=%q err=%v", huella, err)
	}
}

func TestPuenteArtefactoAnalisisIntegraEvidenciaYOrdenNoConsumida(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
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
	orden, err := pruebas.OrdenConsumoConjunto.Datos()
	pruebaCanonica, errPrueba :=
		pruebas.OrdenConsumoConjunto.PruebaCanonica()
	huellaPrueba := sha256.Sum256(pruebaCanonica)
	if err != nil ||
		errPrueba != nil ||
		orden.ArtefactoRef != datos.ArtefactoRef ||
		orden.HuellaSHA256 == "" ||
		hex.EncodeToString(huellaPrueba[:]) != orden.HuellaSHA256 ||
		datos.ArtefactoHuellaSHA256 == "" ||
		analisis.ValidacionRC.FuenteRef != datos.FuenteRCRef ||
		analisis.CostePrevisto == datos.CostePrevisto ||
		analisis.CostePrevisto.Centimos != datos.CostePrevisto.Centimos {
		t.Fatal("el puente perdió evidencia verificada u orden pendiente")
	}
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(SolicitudPrepararArtefactoAnalisis{}),
		reflect.TypeOf(DatosArtefactoAnalisis{}),
		reflect.TypeOf(PruebasArtefactoAnalisisO3{}),
	} {
		for _, nombre := range []string{
			"ReciboConsumoRC",
			"ReciboConsumoCoste",
			"ReciboConsumoConjunto",
			"ConsumoRCRef",
			"ConsumoCosteRef",
		} {
			if _, existe := tipo.FieldByName(nombre); existe {
				t.Fatalf("%s anticipa consumo mediante %s", tipo.Name(), nombre)
			}
		}
	}
}

func TestPuenteArtefactoAnalisisRepeticionExactaGeneraMismaOrdenPendiente(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	primero := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
	segundo := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
	datosPrimero, err := primero.DatosPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	datosSegundo, err := segundo.DatosPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	pruebasPrimero, err := primero.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	pruebasSegundo, err := segundo.PruebasParaO3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	ordenPrimera, err := pruebasPrimero.OrdenConsumoConjunto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ordenSegunda, err := pruebasSegundo.OrdenConsumoConjunto.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datosPrimero.ArtefactoHuellaSHA256 !=
		datosSegundo.ArtefactoHuellaSHA256 ||
		!reflect.DeepEqual(ordenPrimera, ordenSegunda) {
		t.Fatal("la repetición exacta no conservó la orden pendiente")
	}
}

func TestReciboConsumoConjuntoRechazaConfirmacionParcialRCConCoste(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
	orden := artefacto.pruebas.ordenConjunto
	datosOrden, err := orden.Datos()
	if err != nil || datosOrden.OrdenCoste == nil {
		t.Fatal("la orden de prueba debe incluir RC y coste")
	}
	consumidaEn := instanteFuenteAnalisisPrueba().Add(4 * time.Second)
	reciboRC, err := NuevoReciboConsumoRespuestaFuenteAnalisis(
		datosOrden.OrdenRC,
		"consumo_rc_parcial_prohibido_012345",
		consumidaEn,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoReciboConsumoConjuntoFuentesAnalisisO3(
		orden,
		"consumo_conjunto_parcial_prohibido_012345",
		reciboRC,
		nil,
		consumidaEn,
	); !errors.Is(err, ErrArtefactoAnalisisNoConfiable) {
		t.Fatalf("se aceptó un recibo RC sin su coste indivisible: %v", err)
	}
}

func TestPruebasArtefactoAnalisisClonaConfirmacionPublicacion(
	t *testing.T,
) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
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
	artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, solicitud)
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
		t.Fatalf("PruebasParaO3 devolvió un alias mutable: %#v", datosSegunda)
	}
	if _, err := artefacto.DatosPara(solicitud); err != nil {
		t.Fatalf("la mutación externa alteró el artefacto original: %v", err)
	}
}
