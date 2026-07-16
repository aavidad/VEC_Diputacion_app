package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestIntentoNominalConfirmacionV2LigaOperacionYSobreCanonico(t *testing.T) {
	solicitud := intentoNominalConfirmacionV2ValidoPrueba(t, 0x41)
	canonica, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	canonicaConfirmacionV2, err := RepresentacionCanonicaConfirmacionBaremacion(solicitud.Confirmacion)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonica.Revelar(), canonicaConfirmacionV2.Revelar()) {
		t.Fatal("el sobre nominal V2 descendio a la confirmacion ordinaria V2")
	}
	sumaConfirmacionV2 := sha256.Sum256(canonicaConfirmacionV2.Revelar())
	const vectorConfirmacionV2Esperado = "d22293713f205638e903dbaff6bb03761790843235aa8041ef6a1dc942e57c16"
	if obtenido := hex.EncodeToString(sumaConfirmacionV2[:]); obtenido != vectorConfirmacionV2Esperado {
		t.Fatalf("el vector canonico vigente cambio sin versionado: %s", obtenido)
	}

	mismaConfirmacionOtraOperacion := intentoNominalConfirmacionV2ValidoPrueba(t, 0x42)
	mismaConfirmacionOtraOperacion.Confirmacion = solicitud.Confirmacion
	canonicaOtraOperacion, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(
		mismaConfirmacionOtraOperacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonica.Revelar(), canonicaOtraOperacion.Revelar()) {
		t.Fatal("cambiar la referencia de operacion no cambio el canonico")
	}

	referencia, _, err := solicitud.IdentificadorOperacion.DatosReconciliacion()
	if err != nil {
		t.Fatal(err)
	}
	otroIndice, err := NuevoIdentificadorOperacionTransaccionalBaremacion(
		referencia,
		"hmac-sha256:indice-reconciliacion-v1:"+strings.Repeat("c", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	mismaOperacionOtroIndice := solicitud
	mismaOperacionOtroIndice.IdentificadorOperacion = otroIndice
	canonicaOtroIndice, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(mismaOperacionOtroIndice)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonica.Revelar(), canonicaOtroIndice.Revelar()) {
		t.Fatal("cambiar el indice de operacion no cambio el canonico")
	}

	otroEfecto, err := solicitud.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	otroEfecto.Confirmacion.Trazabilidad.Motivo = "Alta administrativa corregida."
	canonicaOtroEfecto, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(otroEfecto)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonica.Revelar(), canonicaOtroEfecto.Revelar()) {
		t.Fatal("cambiar el efecto exacto no cambio el canonico")
	}

	otroSello, err := solicitud.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	otroSello.Confirmacion.HuellaSolicitudHMAC = "hmac-sha256:confirmacion_v2:" + strings.Repeat("e", 64)
	canonicaOtroSello, err := RepresentacionCanonicaSobreProbatorioConfirmacionBaremacionV2(otroSello)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonica.Revelar(), canonicaOtroSello.Revelar()) {
		t.Fatal("el sello se incluyo circularmente en su propia preimagen")
	}

	verificacion := SolicitudVerificarSelloBaremacion{
		Finalidad:              FinalidadSelloSobreProbatorioConfirmacionBaremacionV2,
		RepresentacionCanonica: canonica,
		SelloHMAC:              solicitud.Confirmacion.HuellaSolicitudHMAC,
	}
	if err := verificacion.Validar(); err != nil {
		t.Fatalf("la finalidad V2 no es verificable: %v", err)
	}

	suma := sha256.Sum256(canonica.Revelar())
	const vectorV2Esperado = "371e064cf0673dea27d2741803604a804e08246b9f447497deb025bf49790f6a"
	if obtenido := hex.EncodeToString(suma[:]); obtenido != vectorV2Esperado {
		t.Fatalf("vector canonico V2 inesperado: %s", obtenido)
	}
}

func TestResultadoNominalConfirmacionV2ExigeLaMismaOperacion(t *testing.T) {
	solicitud := intentoNominalConfirmacionV2ValidoPrueba(t, 0x51)
	resultado := resultadoNominalConfirmacionV2ValidoPrueba(t, solicitud)
	if err := resultado.ValidarFormaPara(solicitud); err != nil {
		t.Fatalf("resultado V2 valido rechazado: %v", err)
	}

	cruzado := resultado
	cruzado.IdentificadorOperacion = intentoNominalConfirmacionV2ValidoPrueba(
		t, 0x52,
	).IdentificadorOperacion
	if err := cruzado.ValidarFormaPara(solicitud); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
		t.Fatalf("resultado de otra operacion admitido: %v", err)
	}

	clon, err := resultado.ClonarFormaPara(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	clon.Resultado.Version.Agregado.CalculoInicial.ResultadoRef = "resultado-alterado"
	if resultado.Resultado.Version.Agregado.CalculoInicial.ResultadoRef == "resultado-alterado" {
		t.Fatal("el resultado V2 clonado comparte el agregado")
	}
}

func TestIntentoNominalConfirmacionV2BloqueaRepresentacionesGenericas(t *testing.T) {
	solicitud := intentoNominalConfirmacionV2ValidoPrueba(t, 0x61)
	resultado := resultadoNominalConfirmacionV2ValidoPrueba(t, solicitud)
	for nombre, valor := range map[string]any{
		"solicitud": solicitud,
		"resultado": resultado,
	} {
		t.Run(nombre, func(t *testing.T) {
			texto := fmt.Sprintf("%v %+v %#v", valor, valor, valor)
			if strings.Contains(texto, solicitud.Confirmacion.Trazabilidad.Motivo) ||
				strings.Contains(texto, "hmac-sha256") || strings.Contains(texto, "brc1_") {
				t.Fatalf("el formateo filtro material protegido: %s", texto)
			}
			serializado, err := json.Marshal(valor)
			if serializado != nil || !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("JSON generico = (%q, %v)", serializado, err)
			}
			mariscalTexto, ok := valor.(encoding.TextMarshaler)
			if !ok {
				t.Fatal("falta bloqueo de texto")
			}
			if contenido, err := mariscalTexto.MarshalText(); contenido != nil || !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("texto generico = (%q, %v)", contenido, err)
			}
			mariscalBinario, ok := valor.(encoding.BinaryMarshaler)
			if !ok {
				t.Fatal("falta bloqueo binario")
			}
			if contenido, err := mariscalBinario.MarshalBinary(); contenido != nil || !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("binario generico = (%q, %v)", contenido, err)
			}
		})
	}

	for nombre, destino := range map[string]any{
		"solicitud": &IntentoNominalConfirmacionBaremacionV2{},
		"resultado": &ResultadoNominalConfirmacionBaremacionV2{},
	} {
		t.Run("deserializar_"+nombre, func(t *testing.T) {
			if err := json.Unmarshal([]byte(`{"dato":"fabricado"}`), destino); !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("JSON fabricado admitido: %v", err)
			}
			if err := destino.(encoding.TextUnmarshaler).UnmarshalText([]byte("fabricado")); !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("texto fabricado admitido: %v", err)
			}
			if err := destino.(encoding.BinaryUnmarshaler).UnmarshalBinary([]byte("fabricado")); !errors.Is(
				err, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida,
			) {
				t.Fatalf("binario fabricado admitido: %v", err)
			}
		})
	}

	if err := (IntentoNominalConfirmacionBaremacionV2{}).ValidarForma(); !errors.Is(err, ErrSolicitudBaremacionInvalida) {
		t.Fatalf("el valor cero no fallo en cerrado: %v", err)
	}
}

func intentoNominalConfirmacionV2ValidoPrueba(
	t *testing.T,
	relleno byte,
) IntentoNominalConfirmacionBaremacionV2 {
	t.Helper()
	baremacion := baremacionValidaPrueba(t)
	token, err := NuevoTokenReservaBaremacion(
		base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx")),
	)
	if err != nil {
		t.Fatal(err)
	}
	identificador, err := NuevoIdentificadorOperacionTransaccionalBaremacion(
		referenciaOpacaResultadoPrueba(prefijoReferenciaOperacionBaremacion, relleno),
		"hmac-sha256:indice-reconciliacion-v1:"+strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := IntentoNominalConfirmacionBaremacionV2{
		IdentificadorOperacion: identificador,
		Confirmacion: SolicitudConfirmarCambioBaremacion{
			Contexto:            contextoOperacionValido(AccionConfirmarAltaBaremacion, baremacion.ID),
			Token:               token,
			Clase:               ClaseCambioAltaBaremacion,
			HuellaSolicitudHMAC: "hmac-sha256:confirmacion_v2:" + strings.Repeat("b", 64),
			Agregado:            baremacion,
			Trazabilidad: TrazabilidadCambioBaremacion{
				MotivoClave: "alta_merito",
				Motivo:      "Alta administrativa del merito.",
			},
			ConfirmadaEn: instantePuertosPrueba.Add(time.Minute),
		},
	}
	if err := solicitud.ValidarForma(); err != nil {
		t.Fatalf("crear solicitud V2: %v", err)
	}
	return solicitud
}

func resultadoNominalConfirmacionV2ValidoPrueba(
	t *testing.T,
	solicitud IntentoNominalConfirmacionBaremacionV2,
) ResultadoNominalConfirmacionBaremacionV2 {
	t.Helper()
	agregado, err := solicitud.Confirmacion.Agregado.ClonarCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := agregado.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resultado := ResultadoNominalConfirmacionBaremacionV2{
		IdentificadorOperacion: solicitud.IdentificadorOperacion,
		Resultado: ResultadoConfirmarCambioBaremacion{
			Version: VersionBaremacion{
				Referencia: ReferenciaVersionBaremacion{
					BaremacionMeritoRef: agregado.ID,
					Numero:              1,
					HuellaEstadoSHA256:  huella,
				},
				Agregado:     agregado,
				ConfirmadaEn: solicitud.Confirmacion.ConfirmadaEn,
			},
			Evidencia: EvidenciaTransaccionBaremacion{
				AuditoriaRef:             "auditoria:confirmacion:v2:1",
				HuellaAuditoriaSHA256:    strings.Repeat("c", 64),
				EventoOutboxRef:          "evento:confirmacion:v2:1",
				HuellaEventoOutboxSHA256: strings.Repeat("d", 64),
				ConfirmadaEn:             solicitud.Confirmacion.ConfirmadaEn,
			},
		},
	}
	if err := resultado.ValidarFormaPara(solicitud); err != nil {
		t.Fatalf("crear resultado V2: %v", err)
	}
	return resultado
}
