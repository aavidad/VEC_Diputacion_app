package ports

import (
	"bytes"
	"context"
	"encoding"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
)

func TestResultadoTransaccionalDistingueNoAplicadaDePosiblementeAplicada(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x21)
	prueba := pruebaNoAplicacionValidaPrueba(t, identificador, 0x31)

	noAplicada, err := NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil {
		t.Fatalf("NuevoErrorTransaccionBaremacionNoAplicada: %v", err)
	}
	if err := noAplicada.Validar(); err != nil {
		t.Fatalf("Validar no aplicada: %v", err)
	}
	if !noAplicada.NoAplicadaVerificada() || noAplicada.RequiereReconciliacion() ||
		noAplicada.Estado() != EstadoResultadoTransaccionalNoAplicadaVerificada {
		t.Fatalf("desenlace no aplicado incoherente: estado=%q acreditada=%t reconciliacion=%t",
			noAplicada.Estado(), noAplicada.NoAplicadaVerificada(), noAplicada.RequiereReconciliacion())
	}
	if !errors.Is(noAplicada, ErrTransaccionBaremacionNoAplicada) ||
		errors.Is(noAplicada, ErrResultadoTransaccionalBaremacionIndeterminado) ||
		errors.Is(noAplicada, ErrReconciliacionTransaccionalBaremacionRequerida) {
		t.Fatal("errors.Is no distingue el no aplicado acreditado")
	}

	indeterminado, err := NuevoErrorResultadoTransaccionalIndeterminadoBaremacion(identificador)
	if err != nil {
		t.Fatalf("NuevoErrorResultadoTransaccionalIndeterminadoBaremacion: %v", err)
	}
	if indeterminado.NoAplicadaVerificada() || !indeterminado.RequiereReconciliacion() ||
		indeterminado.Estado() != EstadoResultadoTransaccionalPodriaHaberseAplicado {
		t.Fatalf("desenlace indeterminado incoherente: estado=%q acreditada=%t reconciliacion=%t",
			indeterminado.Estado(), indeterminado.NoAplicadaVerificada(), indeterminado.RequiereReconciliacion())
	}
	if !errors.Is(indeterminado, ErrResultadoTransaccionalBaremacionIndeterminado) ||
		!errors.Is(indeterminado, ErrReconciliacionTransaccionalBaremacionRequerida) ||
		errors.Is(indeterminado, ErrTransaccionBaremacionNoAplicada) {
		t.Fatal("errors.Is no conserva la clasificacion fail-closed")
	}
	if _, existe := indeterminado.PruebaNoAplicacion(); existe {
		t.Fatal("el resultado indeterminado fabrico una prueba de no aplicacion")
	}

	var tipado *ErrorResultadoTransaccionalBaremacion
	compuesto := errors.Join(errors.New("fallo de transporte"), indeterminado)
	if !errors.As(compuesto, &tipado) || tipado != indeterminado ||
		!errors.Is(compuesto, ErrReconciliacionTransaccionalBaremacionRequerida) {
		t.Fatal("errors.As/Is no atraviesa un error compuesto")
	}
}

func TestIdentificadorOperacionSoloCoincideConReferenciaEIndiceExactos(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x35)
	clon, err := identificador.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	if !identificador.CoincideExactamenteCon(clon) || !clon.CoincideExactamenteCon(identificador) {
		t.Fatal("un identificador valido no coincide con su clon")
	}

	otraReferencia := identificadorResultadoTransaccionalValidoPrueba(t, 0x36)
	if identificador.CoincideExactamenteCon(otraReferencia) ||
		otraReferencia.CoincideExactamenteCon(identificador) {
		t.Fatal("se admitio otra referencia con el mismo indice")
	}

	referencia, _, err := identificador.DatosReconciliacion()
	if err != nil {
		t.Fatal(err)
	}
	otroIndice, err := NuevoIdentificadorOperacionTransaccionalBaremacion(
		referencia,
		"hmac-sha256:indice-reconciliacion-v1:"+strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	if identificador.CoincideExactamenteCon(otroIndice) ||
		otroIndice.CoincideExactamenteCon(identificador) {
		t.Fatal("se admitio otro indice para la misma referencia")
	}

	if identificador.CoincideExactamenteCon(IdentificadorOperacionTransaccionalBaremacion{}) ||
		(IdentificadorOperacionTransaccionalBaremacion{}).CoincideExactamenteCon(identificador) {
		t.Fatal("un identificador invalido no fallo en cerrado")
	}
}

func TestNoAplicadaExigePruebaAutenticadaYEnlazada(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x41)
	prueba := pruebaNoAplicacionValidaPrueba(t, identificador, 0x51)
	resultado, err := NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil {
		t.Fatal(err)
	}

	pruebaObtenida, existe := resultado.PruebaNoAplicacion()
	if !existe {
		t.Fatal("se perdio la prueba de no aplicacion")
	}
	evidenciaObtenida, err := pruebaObtenida.Evidencia()
	if err != nil {
		t.Fatal(err)
	}
	identificadorPrueba, referenciaEvidencia, selloEvidencia, err := evidenciaObtenida.DatosVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	if identificadorPrueba != identificador || referenciaEvidencia == "" || selloEvidencia == "" {
		t.Fatal("la prueba no conserva su enlace probatorio")
	}

	otroIdentificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x61)
	cruzado := *resultado
	cruzado.IdentificadorOperacion = otroIdentificador
	if !errors.Is(cruzado.Validar(), ErrResultadoTransaccionalBaremacionInvalido) ||
		cruzado.NoAplicadaVerificada() || !cruzado.RequiereReconciliacion() ||
		!errors.Is(&cruzado, ErrResultadoTransaccionalBaremacionIndeterminado) ||
		!errors.Is(&cruzado, ErrReconciliacionTransaccionalBaremacionRequerida) {
		t.Fatal("una prueba cruzada no fallo en cerrado")
	}

	sinPrueba := ErrorResultadoTransaccionalBaremacion{
		IdentificadorOperacion: identificador,
		EstadoAplicacion:       EstadoResultadoTransaccionalNoAplicadaVerificada,
	}
	if sinPrueba.Validar() == nil || sinPrueba.NoAplicadaVerificada() || !sinPrueba.RequiereReconciliacion() {
		t.Fatal("se admitio afirmar no aplicacion sin prueba")
	}

	pruebaEnIndeterminado := ErrorResultadoTransaccionalBaremacion{
		IdentificadorOperacion:       identificador,
		EstadoAplicacion:             EstadoResultadoTransaccionalPodriaHaberseAplicado,
		PruebaNoAplicacionVerificada: prueba,
	}
	if pruebaEnIndeterminado.Validar() == nil || !pruebaEnIndeterminado.RequiereReconciliacion() {
		t.Fatal("un estado indeterminado acepto evidencia incoherente")
	}
}

func TestEvidenciaSintacticaNoPermiteAfirmarNoAplicacion(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x65)
	evidencia, err := NuevaEvidenciaNoAplicacionBaremacion(
		identificador,
		referenciaOpacaResultadoPrueba(prefijoReferenciaEvidenciaBaremacion, 0x66),
		"hmac-sha256:evidencia-no-aplicacion-v1:"+strings.Repeat("c", 64),
	)
	if err != nil {
		t.Fatal(err)
	}

	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(context.Background(), nil, evidencia); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrVerificadorNoAplicacionBaremacionRequerido) {
		t.Fatalf("verificador ausente = (%v, %v)", prueba, err)
	}
	var nulo *verificadorNoAplicacionNuloPrueba
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(context.Background(), nulo, evidencia); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrVerificadorNoAplicacionBaremacionRequerido) {
		t.Fatalf("verificador typed nil = (%v, %v)", prueba, err)
	}
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(nil, &verificadorNoAplicacionPrueba{}, evidencia); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrContextoVerificacionNoAplicacionBaremacionInvalido) {
		t.Fatalf("contexto ausente = (%v, %v)", prueba, err)
	}
	var contextoNulo *contextoVerificacionNuloPrueba
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(contextoNulo, &verificadorNoAplicacionPrueba{}, evidencia); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrContextoVerificacionNoAplicacionBaremacionInvalido) {
		t.Fatalf("contexto typed nil = (%v, %v)", prueba, err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	verificadorNoInvocado := &verificadorNoAplicacionPrueba{}
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(ctxCancelado, verificadorNoInvocado, evidencia); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrEvidenciaNoAplicacionBaremacionNoVerificada) || !errors.Is(err, context.Canceled) ||
		verificadorNoInvocado.llamadas != 0 {
		t.Fatalf("contexto cancelado = (%v, %v), llamadas=%d", prueba, err, verificadorNoInvocado.llamadas)
	}

	causa := errors.New("conexion-secreta-postgresql")
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(
		context.Background(), &verificadorNoAplicacionPrueba{err: causa}, evidencia,
	); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrEvidenciaNoAplicacionBaremacionNoVerificada) || errors.Is(err, causa) ||
		strings.Contains(fmt.Sprint(err), causa.Error()) {
		t.Fatalf("el fallo del verificador se conservo o admitio: prueba=%v err=%v", prueba, err)
	}
	if prueba, err := VerificarEvidenciaNoAplicacionBaremacion(
		context.Background(), &verificadorNoAplicacionPrueba{err: context.DeadlineExceeded}, evidencia,
	); prueba != (PruebaNoAplicacionVerificadaBaremacion{}) ||
		!errors.Is(err, ErrEvidenciaNoAplicacionBaremacionNoVerificada) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline del verificador no propagado = (%v, %v)", prueba, err)
	}

	prueba, err := VerificarEvidenciaNoAplicacionBaremacion(context.Background(), &verificadorNoAplicacionPrueba{}, evidencia)
	if err != nil || prueba.Validar() != nil {
		t.Fatalf("verificacion positiva = (%v, %v)", prueba, err)
	}
	resultado, err := NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil || !resultado.NoAplicadaVerificada() {
		t.Fatalf("resultado verificado = (%v, %v)", resultado, err)
	}
}

func TestResultadoTransaccionalTypedNilNoRelajaElEstado(t *testing.T) {
	var resultado *ErrorResultadoTransaccionalBaremacion
	var comoError error = resultado

	if !errors.Is(resultado.Validar(), ErrResultadoTransaccionalBaremacionInvalido) ||
		resultado.NoAplicadaVerificada() || !resultado.RequiereReconciliacion() ||
		resultado.Estado() != EstadoResultadoTransaccionalPodriaHaberseAplicado ||
		!errors.Is(comoError, ErrResultadoTransaccionalBaremacionInvalido) ||
		!errors.Is(comoError, ErrResultadoTransaccionalBaremacionIndeterminado) ||
		!errors.Is(comoError, ErrReconciliacionTransaccionalBaremacionRequerida) ||
		errors.Is(comoError, ErrTransaccionBaremacionNoAplicada) {
		t.Fatal("typed nil no fallo en cerrado")
	}
	if clon, err := resultado.Clonar(); clon != nil || !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
		t.Fatalf("Clonar typed nil = (%v, %v)", clon, err)
	}
	if _, err := resultado.Identificador(); !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
		t.Fatalf("Identificador typed nil = %v", err)
	}

	salidas := []string{
		fmt.Sprintf("%s", resultado), fmt.Sprintf("%v", resultado), fmt.Sprintf("%+v", resultado),
		fmt.Sprintf("%#v", resultado), fmt.Sprintf("%q", resultado), fmt.Sprint(resultado),
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Error("resultado", "error", resultado)
	salidas = append(salidas, registro.String())
	for _, salida := range salidas {
		if salida == "" {
			t.Fatal("representacion typed nil vacia")
		}
	}
	serializado, err := json.Marshal(resultado)
	if err != nil || string(serializado) != "null" {
		t.Fatalf("JSON typed nil = %q, %v", serializado, err)
	}
	var extraido *ErrorResultadoTransaccionalBaremacion
	if !errors.As(comoError, &extraido) || extraido != nil {
		t.Fatalf("errors.As typed nil = %#v", extraido)
	}
	if texto, err := resultado.MarshalText(); texto != nil || !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) {
		t.Fatalf("MarshalText typed nil = %q, %v", texto, err)
	}
	if binario, err := resultado.MarshalBinary(); binario != nil || !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) {
		t.Fatalf("MarshalBinary typed nil = %q, %v", binario, err)
	}
}

func TestResultadoTransaccionalInvalidoSiempreFallaEnCerrado(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x71)
	casos := []ErrorResultadoTransaccionalBaremacion{
		{},
		{IdentificadorOperacion: identificador},
		{IdentificadorOperacion: identificador, EstadoAplicacion: EstadoResultadoTransaccionalBaremacion("aplicada")},
		{IdentificadorOperacion: identificador, EstadoAplicacion: EstadoResultadoTransaccionalBaremacion("no_aplicada")},
	}
	for indice := range casos {
		caso := casos[indice]
		if caso.Validar() == nil || caso.NoAplicadaVerificada() || !caso.RequiereReconciliacion() ||
			caso.Estado() != EstadoResultadoTransaccionalPodriaHaberseAplicado ||
			!errors.Is(&caso, ErrResultadoTransaccionalBaremacionInvalido) ||
			!errors.Is(&caso, ErrResultadoTransaccionalBaremacionIndeterminado) ||
			!errors.Is(&caso, ErrReconciliacionTransaccionalBaremacionRequerida) ||
			errors.Is(&caso, ErrTransaccionBaremacionNoAplicada) {
			t.Fatalf("caso invalido %d no fallo en cerrado", indice)
		}
		if clon, err := caso.Clonar(); clon != nil || !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
			t.Fatalf("Clonar invalido %d = (%v, %v)", indice, clon, err)
		}
	}
}

func TestReferenciasYSellosDelResultadoTransaccionalSonCanonicos(t *testing.T) {
	referenciaValida := referenciaOpacaResultadoPrueba(prefijoReferenciaOperacionBaremacion, 0x81)
	hmacValido := "hmac-sha256:reconciliacion-v1:" + strings.Repeat("a", 64)
	if _, err := NuevoIdentificadorOperacionTransaccionalBaremacion(referenciaValida, hmacValido); err != nil {
		t.Fatalf("identificador valido rechazado: %v", err)
	}

	referenciasInvalidas := []string{
		"", "DNI-12345678Z", strings.TrimPrefix(referenciaValida, prefijoReferenciaOperacionBaremacion),
		prefijoReferenciaOperacionBaremacion + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 31)),
		prefijoReferenciaOperacionBaremacion + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 33)),
		referenciaValida + "=", referenciaValida + " ", "brc2_" + strings.TrimPrefix(referenciaValida, prefijoReferenciaOperacionBaremacion),
	}
	for _, referencia := range referenciasInvalidas {
		if _, err := NuevoIdentificadorOperacionTransaccionalBaremacion(referencia, hmacValido); !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
			t.Fatalf("referencia opaca invalida admitida %q: %v", referencia, err)
		}
	}

	hmacInvalidos := []string{
		"", strings.Repeat("a", 64), "sha256:reconciliacion-v1:" + strings.Repeat("a", 64),
		"hmac-sha256:RECONCILIACION:" + strings.Repeat("a", 64),
		"hmac-sha256:reconciliacion-v1:" + strings.Repeat("A", 64),
		"hmac-sha256:reconciliacion-v1:" + strings.Repeat("a", 63),
	}
	for _, hmac := range hmacInvalidos {
		if _, err := NuevoIdentificadorOperacionTransaccionalBaremacion(referenciaValida, hmac); !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
			t.Fatalf("HMAC invalido admitido %q: %v", hmac, err)
		}
	}

	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0x91)
	if _, err := NuevaEvidenciaNoAplicacionBaremacion(
		identificador,
		referenciaOpacaResultadoPrueba(prefijoReferenciaOperacionBaremacion, 0x92),
		"hmac-sha256:evidencia-v1:"+strings.Repeat("b", 64),
	); !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
		t.Fatalf("la evidencia admitio el prefijo de una operacion: %v", err)
	}
	if _, err := NuevaEvidenciaNoAplicacionBaremacion(
		identificador,
		referenciaOpacaResultadoPrueba(prefijoReferenciaEvidenciaBaremacion, 0x93),
		strings.Repeat("b", 64),
	); !errors.Is(err, ErrResultadoTransaccionalBaremacionInvalido) {
		t.Fatalf("la evidencia admitio una huella sin HMAC: %v", err)
	}
}

func TestResultadoTransaccionalBloqueaRepresentacionesGenericas(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0xa1)
	prueba := pruebaNoAplicacionValidaPrueba(t, identificador, 0xa2)
	resultado, err := NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil {
		t.Fatal(err)
	}
	referenciaOperacion, indiceHMAC, err := identificador.DatosReconciliacion()
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := prueba.Evidencia()
	if err != nil {
		t.Fatal(err)
	}
	_, referenciaEvidencia, selloEvidencia, err := evidencia.DatosVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	secretos := []string{referenciaOperacion, indiceHMAC, referenciaEvidencia, selloEvidencia}

	objetos := []any{identificador, &identificador, evidencia, &evidencia, prueba, &prueba, resultado}
	for indice, objeto := range objetos {
		salidas := []string{
			fmt.Sprintf("%s", objeto), fmt.Sprintf("%v", objeto), fmt.Sprintf("%+v", objeto),
			fmt.Sprintf("%#v", objeto), fmt.Sprintf("%q", objeto), fmt.Sprint(objeto),
		}
		if serializado, err := json.Marshal(objeto); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) || serializado != nil {
			t.Fatalf("objeto %d JSON no bloqueado: bytes=%q err=%v", indice, serializado, err)
		}
		mariscalTexto, ok := objeto.(encoding.TextMarshaler)
		if !ok {
			t.Fatalf("objeto %d no implementa TextMarshaler", indice)
		}
		if serializado, err := mariscalTexto.MarshalText(); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) || serializado != nil {
			t.Fatalf("objeto %d texto no bloqueado: bytes=%q err=%v", indice, serializado, err)
		}
		mariscalBinario, ok := objeto.(encoding.BinaryMarshaler)
		if !ok {
			t.Fatalf("objeto %d no implementa BinaryMarshaler", indice)
		}
		if serializado, err := mariscalBinario.MarshalBinary(); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) || serializado != nil {
			t.Fatalf("objeto %d binario no bloqueado: bytes=%q err=%v", indice, serializado, err)
		}

		var registro bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&registro, nil))
		logger.Error("resultado", "valor", objeto)
		salidas = append(salidas, registro.String())
		for _, salida := range salidas {
			for _, secreto := range secretos {
				if strings.Contains(salida, secreto) {
					t.Fatalf("objeto %d expone material probatorio %q en %q", indice, secreto, salida)
				}
			}
		}
	}
	// Incluso una desreferenciacion deliberada solo puede mostrar los envoltorios
	// protegidos de sus campos privados; el error publico se usa siempre por puntero.
	for _, salida := range []string{
		fmt.Sprintf("%v", *resultado), fmt.Sprintf("%+v", *resultado), fmt.Sprintf("%#v", *resultado),
	} {
		for _, secreto := range secretos {
			if strings.Contains(salida, secreto) {
				t.Fatalf("el valor desreferenciado expone %q en %q", secreto, salida)
			}
		}
	}
}

func TestResultadoTransaccionalBloqueaDeserializacionGenerica(t *testing.T) {
	casos := []struct {
		nombre  string
		destino any
	}{
		{"identificador", &IdentificadorOperacionTransaccionalBaremacion{}},
		{"evidencia", &EvidenciaNoAplicacionBaremacion{}},
		{"prueba", &PruebaNoAplicacionVerificadaBaremacion{}},
		{"resultado", &ErrorResultadoTransaccionalBaremacion{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{"referencia_opaca":"fabricada"}`), caso.destino); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) {
				t.Fatalf("JSON admitido: %v", err)
			}
			if err := caso.destino.(encoding.TextUnmarshaler).UnmarshalText([]byte("fabricado")); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) {
				t.Fatalf("texto admitido: %v", err)
			}
			if err := caso.destino.(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte("fabricado")); !errors.Is(err, ErrSerializacionResultadoTransaccionalBaremacionProhibida) {
				t.Fatalf("binario admitido: %v", err)
			}
		})
	}
}

func TestResultadoTransaccionalClonaSinCompartirEstado(t *testing.T) {
	identificador := identificadorResultadoTransaccionalValidoPrueba(t, 0xb1)
	prueba := pruebaNoAplicacionValidaPrueba(t, identificador, 0xb2)
	resultado, err := NuevoErrorTransaccionBaremacionNoAplicada(prueba)
	if err != nil {
		t.Fatal(err)
	}
	clon, err := resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	if clon == resultado || *clon != *resultado {
		t.Fatal("Clonar no produjo una copia equivalente e independiente")
	}
	clon.EstadoAplicacion = EstadoResultadoTransaccionalPodriaHaberseAplicado
	clon.PruebaNoAplicacionVerificada = PruebaNoAplicacionVerificadaBaremacion{}
	if !resultado.NoAplicadaVerificada() || resultado.RequiereReconciliacion() {
		t.Fatal("mutar el clon altero el original")
	}

	clonIdentificador, err := identificador.Clonar()
	if err != nil || clonIdentificador != identificador {
		t.Fatalf("clon identificador = (%v, %v)", clonIdentificador, err)
	}
	clonPrueba, err := prueba.Clonar()
	if err != nil || clonPrueba != prueba {
		t.Fatalf("clon prueba = (%v, %v)", clonPrueba, err)
	}
}

func TestContratoResultadoTransaccionalNoContieneAutoridadNiCapacidades(t *testing.T) {
	tipos := []reflect.Type{
		reflect.TypeFor[IdentificadorOperacionTransaccionalBaremacion](),
		reflect.TypeFor[EvidenciaNoAplicacionBaremacion](),
		reflect.TypeFor[PruebaNoAplicacionVerificadaBaremacion](),
		reflect.TypeFor[ErrorResultadoTransaccionalBaremacion](),
	}
	prohibidos := []string{
		"context", "contexto", "sesion", "autoriz", "token", "principal", "actor",
		"credencial", "cookie", "cabecera", "causa", "payload", "dni", "correo",
	}
	visitados := make(map[reflect.Type]bool)
	for _, tipo := range tipos {
		verificarTipoResultadoTransaccionalSeguro(t, tipo, prohibidos, visitados)
	}
}

func verificarTipoResultadoTransaccionalSeguro(
	t *testing.T,
	tipo reflect.Type,
	prohibidos []string,
	visitados map[reflect.Type]bool,
) {
	t.Helper()
	if visitados[tipo] {
		return
	}
	visitados[tipo] = true
	if tipo.Kind() != reflect.Struct {
		t.Fatalf("%v no es struct", tipo)
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		campo := tipo.Field(indice)
		if campo.IsExported() {
			esResultado := tipo == reflect.TypeFor[ErrorResultadoTransaccionalBaremacion]()
			permitido := campo.Name == "IdentificadorOperacion" || campo.Name == "EstadoAplicacion" ||
				campo.Name == "PruebaNoAplicacionVerificada"
			if !esResultado || !permitido {
				t.Fatalf("%v.%s expone material interno", tipo, campo.Name)
			}
		}
		nombre := strings.ToLower(campo.Name)
		for _, prohibido := range prohibidos {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("%v.%s contiene el concepto prohibido %q", tipo, campo.Name, prohibido)
			}
		}
		switch campo.Type.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
			t.Fatalf("%v.%s permite estado compartido o autoridad dinamica: %v", tipo, campo.Name, campo.Type)
		case reflect.Struct:
			verificarTipoResultadoTransaccionalSeguro(t, campo.Type, prohibidos, visitados)
		}
	}
}

func identificadorResultadoTransaccionalValidoPrueba(
	t *testing.T,
	relleno byte,
) IdentificadorOperacionTransaccionalBaremacion {
	t.Helper()
	identificador, err := NuevoIdentificadorOperacionTransaccionalBaremacion(
		referenciaOpacaResultadoPrueba(prefijoReferenciaOperacionBaremacion, relleno),
		"hmac-sha256:indice-reconciliacion-v1:"+strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatalf("crear identificador: %v", err)
	}
	return identificador
}

func pruebaNoAplicacionValidaPrueba(
	t *testing.T,
	identificador IdentificadorOperacionTransaccionalBaremacion,
	relleno byte,
) PruebaNoAplicacionVerificadaBaremacion {
	t.Helper()
	evidencia, err := NuevaEvidenciaNoAplicacionBaremacion(
		identificador,
		referenciaOpacaResultadoPrueba(prefijoReferenciaEvidenciaBaremacion, relleno),
		"hmac-sha256:evidencia-no-aplicacion-v1:"+strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatalf("crear evidencia: %v", err)
	}
	prueba, err := VerificarEvidenciaNoAplicacionBaremacion(
		context.Background(), &verificadorNoAplicacionPrueba{}, evidencia,
	)
	if err != nil {
		t.Fatalf("verificar evidencia: %v", err)
	}
	return prueba
}

type verificadorNoAplicacionPrueba struct {
	err      error
	llamadas int
}

func (v *verificadorNoAplicacionPrueba) VerificarNoAplicacionBaremacion(
	_ context.Context,
	_ EvidenciaNoAplicacionBaremacion,
) error {
	v.llamadas++
	return v.err
}

type verificadorNoAplicacionNuloPrueba struct{}

func (*verificadorNoAplicacionNuloPrueba) VerificarNoAplicacionBaremacion(
	context.Context,
	EvidenciaNoAplicacionBaremacion,
) error {
	panic("no debe invocarse un verificador typed nil")
}

type contextoVerificacionNuloPrueba struct{ context.Context }

func referenciaOpacaResultadoPrueba(prefijo string, relleno byte) string {
	return prefijo + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{relleno}, longitudReferenciaOpacaBaremacion))
}
