package ginpixapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOperacionGINPIXRecuperableEmiteCargaYConsultaMinimizada(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)
	preparacion, err := prepararOperacionRecuperable(solicitud)
	if err != nil {
		t.Fatalf("preparar solicitud recuperable: %v", err)
	}
	cuerpoCompleto, _ := preparacion.Cuerpo()
	var cuerpos [][]byte
	var destinos []string
	var cerrados atomic.Int32
	transporte := &transporteFalso{funcion: func(_ int, peticion *http.Request) (*http.Response, error) {
		contenido, err := io.ReadAll(peticion.Body)
		if err != nil {
			t.Fatalf("leer peticion: %v", err)
		}
		cuerpos = append(cuerpos, append([]byte(nil), contenido...))
		destinos = append(destinos, peticion.URL.Path)
		return respuestaValidaPrueba(t, preparacion, http.StatusOK, &cerrados), nil
	}}
	adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())

	reciboEmision, err := adaptador.EmitirOperacionGINPIX(
		context.Background(), solicitud, reserva,
	)
	if err != nil || reciboEmision.ValidarPara(solicitud) != nil {
		t.Fatalf("emitir por puerto: %#v / %v", reciboEmision, err)
	}
	reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
	reciboConsulta, err := adaptador.ConsultarOperacionGINPIX(
		context.Background(), solicitud, reserva,
	)
	if err != nil || reciboConsulta.ValidarPara(solicitud) != nil ||
		!reflect.DeepEqual(reciboEmision, reciboConsulta) {
		t.Fatalf("consultar por puerto: %#v / %v", reciboConsulta, err)
	}
	replayConsulta, err := adaptador.ConsultarOperacionGINPIX(
		context.Background(), solicitud, reserva,
	)
	if err != nil || !reflect.DeepEqual(reciboConsulta, replayConsulta) {
		t.Fatalf("replay de consulta divergente: %#v / %v", replayConsulta, err)
	}
	if transporte.total() != 3 || cerrados.Load() != 3 ||
		len(cuerpos) != 3 || !bytes.Equal(cuerpos[0], cuerpoCompleto) ||
		bytes.Equal(cuerpos[1], cuerpoCompleto) || !bytes.Equal(cuerpos[1], cuerpos[2]) ||
		bytes.Contains(cuerpos[1], []byte(datoPersonalSinteticoPrueba)) ||
		!bytes.Contains(cuerpos[1], []byte("idempotencia_ginpix_api_0001")) {
		t.Fatalf("cuerpos no respetan emision/consulta: llamadas=%d", transporte.total())
	}
	if len(destinos) != 3 || destinos[0] != "/operaciones/enviar" ||
		destinos[1] != "/operaciones/consultar" || destinos[2] != "/operaciones/consultar" {
		t.Fatalf("destinos incorrectos: %#v", destinos)
	}
	datosSolicitud, _ := solicitud.Datos()
	if reciboEmision.ClaveOperacionRef != datosSolicitud.ClaveOperacionRef {
		t.Fatal("el recibo neutral no conserva la clave de operacion")
	}
}

func TestOperacionGINPIXRecuperablePreparacionEquivaleAlContratoIntegrado(t *testing.T) {
	mapeo, orden, incorporacion := insumosPreparacionAPIGINPIXPrueba(t)
	solicitud, err := ports.NuevaSolicitudOperacionGINPIX(mapeo, orden, incorporacion)
	if err != nil {
		t.Fatal(err)
	}
	desdeOperacion, err := prepararOperacionRecuperable(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	desdeContratos, err := Preparar(mapeo, orden, incorporacion)
	if err != nil {
		t.Fatal(err)
	}
	cuerpoOperacion, _ := desdeOperacion.Cuerpo()
	cuerpoContratos, _ := desdeContratos.Cuerpo()
	metadatosOperacion, _ := desdeOperacion.Metadatos()
	metadatosContratos, _ := desdeContratos.Metadatos()
	if !bytes.Equal(cuerpoOperacion, cuerpoContratos) || metadatosOperacion != metadatosContratos {
		t.Fatal("la traduccion diverge de Preparar O7-04")
	}
}

func TestOperacionGINPIXRecuperableClasificaFronteraRoundTrip(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)
	preparacion, _ := prepararOperacionRecuperable(solicitud)

	t.Run("autenticacion previa", func(t *testing.T) {
		transporte := transporteQueNoDebeInvocarse(t)
		autenticador := &autenticadorFalso{err: errors.New("detalle privado")}
		adaptador := nuevoAdaptadorPrueba(t, transporte, autenticador, politicaPrueba())
		_, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva)
		if !errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) ||
			errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) || transporte.total() != 0 {
			t.Fatalf("fallo previo mal clasificado: %v", err)
		}
	})

	t.Run("transporte posterior", func(t *testing.T) {
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			return nil, errors.New("detalle transporte")
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		_, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva)
		if !errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) ||
			errors.Is(err, ports.ErrEmisionOperacionGINPIXNoIniciada) || transporte.total() != 1 {
			t.Fatalf("fallo posterior mal clasificado: %v", err)
		}
	})

	t.Run("resultado mas error", func(t *testing.T) {
		var cerrados atomic.Int32
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			return respuestaValidaPrueba(t, preparacion, http.StatusOK, &cerrados),
				errors.New("resultado ambiguo")
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		recibo, err := adaptador.EmitirOperacionGINPIX(context.Background(), solicitud, reserva)
		if recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
			!errors.Is(err, ports.ErrEmisionOperacionGINPIXIndeterminada) ||
			cerrados.Load() != 1 || transporte.total() != 1 {
			t.Fatalf("resultado+error no fallo cerrado: %#v / %v", recibo, err)
		}
	})

	t.Run("consulta resultado mas error", func(t *testing.T) {
		reservaConsulta := reserva
		reservaConsulta.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
		var cerrados atomic.Int32
		transporte := &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
			return respuestaValidaPrueba(t, preparacion, http.StatusOK, &cerrados),
				errors.New("consulta ambigua")
		}}
		adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
		recibo, err := adaptador.ConsultarOperacionGINPIX(
			context.Background(), solicitud, reservaConsulta,
		)
		if recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
			!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) ||
			cerrados.Load() != 1 || transporte.total() != 1 {
			t.Fatalf("consulta resultado+error no fallo cerrada: %#v / %v", recibo, err)
		}
	})
}

func TestOperacionGINPIXRecuperableConsultaNoEncontradaNoEmite(t *testing.T) {
	solicitud, reserva := solicitudOperacionRecuperablePrueba(t)
	reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
	var cuerpo []byte
	var cerrados atomic.Int32
	transporte := &transporteFalso{funcion: func(_ int, peticion *http.Request) (*http.Response, error) {
		cuerpo, _ = io.ReadAll(peticion.Body)
		if peticion.URL.Path != "/operaciones/consultar" {
			t.Fatalf("la consulta activo otro destino: %s", peticion.URL.Path)
		}
		return respuestaSimplePrueba(http.StatusNotFound, "application/json", []byte(`{}`), &cerrados), nil
	}}
	adaptador := nuevoAdaptadorPrueba(t, transporte, &autenticadorFalso{}, politicaPrueba())
	recibo, err := adaptador.ConsultarOperacionGINPIX(context.Background(), solicitud, reserva)
	if recibo != (ports.ReciboExternoOperacionGINPIX{}) ||
		!errors.Is(err, ports.ErrConsultaOperacionGINPIXNoDisponible) ||
		transporte.total() != 1 || cerrados.Load() != 1 ||
		bytes.Contains(cuerpo, []byte(datoPersonalSinteticoPrueba)) {
		t.Fatalf("consulta no encontrada produjo exito o emision: %#v / %v", recibo, err)
	}
}

func solicitudOperacionRecuperablePrueba(
	t *testing.T,
) (ports.SolicitudOperacionGINPIX, ports.ReservaOperacionGINPIX) {
	t.Helper()
	mapeo, orden, incorporacion := insumosPreparacionAPIGINPIXPrueba(t)
	solicitud, err := ports.NuevaSolicitudOperacionGINPIX(mapeo, orden, incorporacion)
	if err != nil {
		t.Fatalf("crear solicitud recuperable: %v", err)
	}
	datos, _ := solicitud.Datos()
	return solicitud, ports.ReservaOperacionGINPIX{
		ReservaRef: "reserva_ginpix_api_0001", ClaveOperacionRef: datos.ClaveOperacionRef,
		Intento: 1, Situacion: ports.ReservaOperacionGINPIXEmisionAutorizada,
	}
}

func transporteQueNoDebeInvocarse(t *testing.T) *transporteFalso {
	t.Helper()
	return &transporteFalso{funcion: func(_ int, _ *http.Request) (*http.Response, error) {
		t.Fatal("se alcanzo RoundTrip")
		return nil, nil
	}}
}
