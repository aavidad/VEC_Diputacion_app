package application

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestFinalizarFirmaClasificaRespuestaInvalidaComoIndeterminada(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.repositorio.alterarConfirmacion = func(
		resultado puertosbolsa.ResultadoConfirmarCambioBaremacion,
	) puertosbolsa.ResultadoConfirmarCambioBaremacion {
		resultado.Version.Referencia.Numero++
		return resultado
	}

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "resultado-invalido"),
	)
	exigirDesenlaceIndeterminadoBaremacionPrueba(t, err)
	if entorno.repositorio.confirmaciones != 1 || entorno.repositorio.intentosAbandono != 0 ||
		entorno.repositorio.version.Referencia.Numero != 2 {
		t.Fatalf("efectos confirmaciones=%d abandonos=%d version=%d",
			entorno.repositorio.confirmaciones, entorno.repositorio.intentosAbandono,
			entorno.repositorio.version.Referencia.Numero)
	}
}

func TestFinalizarFirmaConservaResultadoTransaccionalIndeterminadoTipado(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	identificador := identificadorTransaccionalAplicacionPrueba(t, 0x31)
	indeterminado, err := puertosbolsa.NuevoErrorResultadoTransaccionalIndeterminadoBaremacion(identificador)
	if err != nil {
		t.Fatal(err)
	}
	entorno.repositorio.errorConfirmar = indeterminado

	_, err = entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "indeterminado-tipado"),
	)
	exigirDesenlaceIndeterminadoBaremacionPrueba(t, err)
	var recuperado *puertosbolsa.ErrorResultadoTransaccionalBaremacion
	if !errors.As(err, &recuperado) || recuperado != indeterminado ||
		entorno.repositorio.version.Referencia.Numero != 2 ||
		entorno.repositorio.intentosAbandono != 0 {
		t.Fatal("se perdio el identificador o se compenso el desenlace indeterminado")
	}
}

func TestFinalizarFirmaRechazaResultadoAplicadoYNoAplicacionContradictorios(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.repositorio.errorConfirmar = errorNoAplicacionAcreditadaBaremacionPrueba(t, 0x49)
	entorno.repositorio.devolverResultadoConError = true

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "aplicada-y-no-aplicada"),
	)
	exigirDesenlaceIndeterminadoBaremacionPrueba(t, err)
	if entorno.repositorio.version.Referencia.Numero != 2 ||
		entorno.repositorio.confirmaciones != 1 || entorno.repositorio.intentosAbandono != 0 {
		t.Fatal("la contradiccion altero o compenso el efecto confirmado")
	}
}

func TestFinalizarFirmaDistingueNoAplicacionAcreditadaSinReintentar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	noAplicada := errorNoAplicacionAcreditadaBaremacionPrueba(t, 0x41)
	entorno.repositorio.errorConfirmarSinEfecto = noAplicada

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "no-aplicada-acreditada"),
	)
	if !errors.Is(err, puertosbolsa.ErrTransaccionBaremacionNoAplicada) ||
		errors.Is(err, puertosbolsa.ErrResultadoTransaccionalBaremacionIndeterminado) ||
		errors.Is(err, puertosbolsa.ErrReconciliacionTransaccionalBaremacionRequerida) {
		t.Fatalf("prueba de no aplicacion mal clasificada: %v", err)
	}
	var recuperada *puertosbolsa.ErrorResultadoTransaccionalBaremacion
	if !errors.As(err, &recuperada) || recuperada != noAplicada ||
		entorno.repositorio.confirmaciones != 1 || entorno.repositorio.intentosAbandono != 0 ||
		entorno.repositorio.version.Referencia.Numero != 1 {
		t.Fatal("la no aplicacion acreditada genero efecto, abandono o reintento")
	}
}

func TestClasificacionTransaccionalRechazaAfirmacionesNoAcreditadasOContradictorias(t *testing.T) {
	solicitud := solicitudConfirmacionValidaClasificacionPrueba(t)
	noAplicada := errorNoAplicacionAcreditadaBaremacionPrueba(t, 0x51)
	otraNoAplicada := errorNoAplicacionAcreditadaBaremacionPrueba(t, 0x61)
	indeterminadoA := errorIndeterminadoBaremacionPrueba(t, 0x52)
	indeterminadoB := errorIndeterminadoBaremacionPrueba(t, 0x62)
	var tipadoNulo *puertosbolsa.ErrorResultadoTransaccionalBaremacion
	var errorTipadoNulo error = tipadoNulo
	casos := []struct {
		nombre string
		err    error
	}{
		{"marca desnuda", puertosbolsa.ErrTransaccionBaremacionNoAplicada},
		{"composicion contradictoria", errors.Join(
			noAplicada, puertosbolsa.ErrResultadoTransaccionalBaremacionIndeterminado,
		)},
		{"dos pruebas distintas", errors.Join(noAplicada, otraNoAplicada)},
		{"resultado tipado invalido", &puertosbolsa.ErrorResultadoTransaccionalBaremacion{}},
		{"resultado tipado nulo", errorTipadoNulo},
		{"dos indeterminados distintos", errors.Join(indeterminadoA, indeterminadoB)},
		{"desenvoltura con panico", errorDesenvolturaPanicoBaremacionPrueba{}},
		{"desenvoltura ciclica", &errorDesenvolturaCiclicaBaremacionPrueba{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			err := clasificarDesenlaceConfirmacionBaremacion(
				puertosbolsa.ResultadoConfirmarCambioBaremacion{},
				solicitud,
				caso.err,
			)
			exigirDesenlaceIndeterminadoBaremacionPrueba(t, err)
			var tipado *puertosbolsa.ErrorResultadoTransaccionalBaremacion
			if errors.As(err, &tipado) {
				t.Fatalf("la clasificacion eligio un resultado tipado ambiguo: %v", err)
			}
		})
	}
}

func TestClasificacionNoAplicadaRechazaHermanosTecnicos(t *testing.T) {
	solicitud := solicitudConfirmacionValidaClasificacionPrueba(t)
	noAplicada := errorNoAplicacionAcreditadaBaremacionPrueba(t, 0x71)
	causaTecnica := errors.New("detalle tecnico sensible simulado")
	err := clasificarDesenlaceConfirmacionBaremacion(
		puertosbolsa.ResultadoConfirmarCambioBaremacion{},
		solicitud,
		errors.Join(causaTecnica, noAplicada),
	)
	exigirDesenlaceIndeterminadoBaremacionPrueba(t, err)
	var recuperada *puertosbolsa.ErrorResultadoTransaccionalBaremacion
	if errors.Is(err, causaTecnica) || errors.As(err, &recuperada) {
		t.Fatalf("la clasificacion conservo una rama ambigua: %v", err)
	}
}

func solicitudConfirmacionValidaClasificacionPrueba(
	t *testing.T,
) puertosbolsa.SolicitudConfirmarCambioBaremacion {
	t.Helper()
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.repositorio.errorConfirmarSinEfecto = errors.New("captura expurgada de solicitud")
	_, _ = entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "captura-clasificacion"),
	)
	if entorno.repositorio.confirmacion == nil || entorno.repositorio.confirmacion.Validar() != nil {
		t.Fatal("no se capturo una solicitud de confirmacion valida")
	}
	clon, err := entorno.repositorio.confirmacion.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	return clon
}

func exigirDesenlaceIndeterminadoBaremacionPrueba(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrResultadoBaremacionNoConfiable) ||
		!errors.Is(err, puertosbolsa.ErrResultadoTransaccionalBaremacionIndeterminado) ||
		!errors.Is(err, puertosbolsa.ErrReconciliacionTransaccionalBaremacionRequerida) ||
		errors.Is(err, puertosbolsa.ErrTransaccionBaremacionNoAplicada) {
		t.Fatalf("desenlace no fallo en cerrado: %v", err)
	}
}

func identificadorTransaccionalAplicacionPrueba(
	t *testing.T,
	relleno byte,
) puertosbolsa.IdentificadorOperacionTransaccionalBaremacion {
	t.Helper()
	referencia := "brc1_" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{relleno}, 32))
	identificador, err := puertosbolsa.NuevoIdentificadorOperacionTransaccionalBaremacion(
		referencia,
		"hmac-sha256:indice-reconciliacion-v1:"+strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	return identificador
}

func errorNoAplicacionAcreditadaBaremacionPrueba(
	t *testing.T,
	relleno byte,
) *puertosbolsa.ErrorResultadoTransaccionalBaremacion {
	t.Helper()
	identificador := identificadorTransaccionalAplicacionPrueba(t, relleno)
	referencia := "bre1_" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{relleno + 1}, 32))
	evidencia, err := puertosbolsa.NuevaEvidenciaNoAplicacionBaremacion(
		identificador, referencia,
		"hmac-sha256:evidencia-no-aplicacion-v1:"+strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := puertosbolsa.VerificarEvidenciaNoAplicacionBaremacion(
		context.Background(), verificadorNoAplicacionAplicacionPrueba{}, evidencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := puertosbolsa.NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func errorIndeterminadoBaremacionPrueba(
	t *testing.T,
	relleno byte,
) *puertosbolsa.ErrorResultadoTransaccionalBaremacion {
	t.Helper()
	resultado, err := puertosbolsa.NuevoErrorResultadoTransaccionalIndeterminadoBaremacion(
		identificadorTransaccionalAplicacionPrueba(t, relleno),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

type verificadorNoAplicacionAplicacionPrueba struct{}

type errorDesenvolturaPanicoBaremacionPrueba struct{}

func (errorDesenvolturaPanicoBaremacionPrueba) Error() string { return "[ERROR-PROTEGIDO]" }
func (errorDesenvolturaPanicoBaremacionPrueba) Unwrap() []error {
	panic("desenvoltura hostil simulada")
}

type errorDesenvolturaCiclicaBaremacionPrueba struct{}

func (*errorDesenvolturaCiclicaBaremacionPrueba) Error() string {
	return "[ERROR-PROTEGIDO]"
}
func (e *errorDesenvolturaCiclicaBaremacionPrueba) Unwrap() error { return e }

func (verificadorNoAplicacionAplicacionPrueba) VerificarNoAplicacionBaremacion(
	context.Context,
	puertosbolsa.EvidenciaNoAplicacionBaremacion,
) error {
	return nil
}
