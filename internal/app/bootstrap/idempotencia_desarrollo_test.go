package bootstrap

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

func materialIdempotenciaDeterministaPrueba(generaciones ...uint32) materialIdempotenciaDesarrollo {
	material := materialIdempotenciaDesarrollo{
		generaciones: make([]generacionIdempotenciaDesarrollo, 0, len(generaciones)),
	}
	for _, numero := range generaciones {
		generacion := generacionIdempotenciaDesarrollo{generacion: numero}
		generacion.localizador.referencia = referenciaLocalizadorIdempotenciaDesarrollo(numero)
		generacion.huellaSolicitud.referencia = referenciaHuellaIdempotenciaDesarrollo(numero)
		for indice := 0; indice < sha256.Size; indice++ {
			generacion.localizador.material[indice] = byte(numero*32 + uint32(indice) + 1)
			generacion.huellaSolicitud.material[indice] = byte(numero*32 + uint32(indice) + 129)
		}
		material.generaciones = append(material.generaciones, generacion)
	}
	return material
}

func nuevoDerivadorIdempotenciaPrueba(
	t *testing.T,
	generaciones ...uint32,
) *derivadorIdentidadOperacionDesarrollo {
	t.Helper()
	material := materialIdempotenciaDeterministaPrueba(generaciones...)
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material)
	if err != nil {
		t.Fatalf("crear derivador: %v", err)
	}
	if material.generaciones != nil {
		t.Fatal("el constructor conservo una segunda referencia propietaria a las claves")
	}
	t.Cleanup(derivador.borrar)
	return derivador
}

func TestDerivadorIdempotenciaGeneraPrimariaG2YAliasG1ConHMACSHA256(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	preimagenLocalizador := []byte("preimagen-localizador-efimera")
	preimagenHuella := []byte("preimagen-huella-efimera")
	resultados, err := derivador.calcularHMAC(preimagenLocalizador, preimagenHuella)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	if len(resultados) != 2 || resultados[0].generacion != 2 || resultados[1].generacion != 1 ||
		resultados[0].referenciaLocalizador != referenciaLocalizadorIdempotenciaDesarrollo(2) ||
		resultados[1].referenciaHuellaSolicitud != referenciaHuellaIdempotenciaDesarrollo(1) {
		t.Fatalf("ventana de rotacion incoherente: %+v", resultados)
	}
	mac := hmac.New(sha256.New, derivador.generaciones[0].localizador.material[:])
	_, _ = mac.Write(preimagenLocalizador)
	var esperado [sha256.Size]byte
	copy(esperado[:], mac.Sum(nil))
	if !hmac.Equal(resultados[0].localizador[:], esperado[:]) {
		t.Fatal("L primaria no es HMAC-SHA256 con la clave localizadora g2")
	}

	copiaLocalizador := append([]byte(nil), preimagenLocalizador...)
	copiaHuella := append([]byte(nil), preimagenHuella...)
	if _, err := derivador.derivarMaterialEfimero(
		context.Background(), copiaLocalizador, copiaHuella,
	); err != nil {
		t.Fatalf("derivar conjunto nominal: %v", err)
	}
	if !bytes.Equal(copiaLocalizador, make([]byte, len(copiaLocalizador))) ||
		!bytes.Equal(copiaHuella, make([]byte, len(copiaHuella))) {
		t.Fatal("el derivador no borro las copias efimeras de las preimagenes")
	}
}

func TestVentanasSolapadasRecuperanLaMismaIdentidadG2(t *testing.T) {
	anterior := nuevoDerivadorIdempotenciaPrueba(t, 3, 2)
	posterior := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	resultadosAnterior, err := anterior.calcularHMAC([]byte("L"), []byte("F"))
	if err != nil {
		t.Fatal(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultadosAnterior)
	resultadosPosterior, err := posterior.calcularHMAC([]byte("L"), []byte("F"))
	if err != nil {
		t.Fatal(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultadosPosterior)
	if resultadosAnterior[1] != resultadosPosterior[0] {
		t.Fatal("la generacion solapada g2 dejo de ser recuperable entre ventanas")
	}
}

func TestCambiarUnaClaveSoloAfectaSuDominioYGeneracion(t *testing.T) {
	base := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	alterado := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	alterado.generaciones[0].localizador.material[0] ^= 0xff

	resultadosBase, err := base.calcularHMAC([]byte("L"), []byte("F"))
	if err != nil {
		t.Fatal(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultadosBase)
	resultadosAlterados, err := alterado.calcularHMAC([]byte("L"), []byte("F"))
	if err != nil {
		t.Fatal(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultadosAlterados)
	if resultadosBase[0].localizador == resultadosAlterados[0].localizador {
		t.Fatal("cambiar la clave L/g2 no cambio su salida")
	}
	if resultadosBase[0].huellaSolicitud != resultadosAlterados[0].huellaSolicitud ||
		resultadosBase[1] != resultadosAlterados[1] {
		t.Fatal("cambiar L/g2 altero otro dominio o generacion")
	}
}

type contextoNuloPrueba struct{ context.Context }

func TestDerivadorFallaCerradoYBorraPreimagenesEnErrores(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	for nombre, ctx := range map[string]context.Context{
		"nulo":         nil,
		"nulo tipado":  (*contextoNuloPrueba)(nil),
		"ya cancelado": ctxCancelado,
	} {
		t.Run(nombre, func(t *testing.T) {
			preimagenL := []byte("L sensible")
			preimagenF := []byte("F sensible")
			if _, err := derivador.derivarMaterialEfimero(ctx, preimagenL, preimagenF); !errors.Is(
				err, gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida,
			) {
				t.Fatalf("error = %v", err)
			}
			if !bytes.Equal(preimagenL, make([]byte, len(preimagenL))) ||
				!bytes.Equal(preimagenF, make([]byte, len(preimagenF))) {
				t.Fatal("las preimagenes sobrevivieron a un desenlace de error")
			}
		})
	}
	var derivadorNulo *derivadorIdentidadOperacionDesarrollo
	preimagenL := []byte("L sensible")
	preimagenF := []byte("F sensible")
	if _, err := derivadorNulo.derivarMaterialEfimero(
		context.Background(), preimagenL, preimagenF,
	); !errors.Is(err, gobiernoconvocatorias.ErrRotacionIdempotenciaInvalida) {
		t.Fatalf("proveedor nulo: %v", err)
	}
	if !bytes.Equal(preimagenL, make([]byte, len(preimagenL))) ||
		!bytes.Equal(preimagenF, make([]byte, len(preimagenF))) {
		t.Fatal("el proveedor nulo no borro las preimagenes recibidas")
	}
}

func TestBorradoMejorEsfuerzoEliminaClavesDeLaCopiaPropietaria(t *testing.T) {
	material := materialIdempotenciaDeterministaPrueba(2, 1)
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&material)
	if err != nil {
		t.Fatal(err)
	}
	respaldoPropietario := derivador.generaciones
	derivador.borrar()
	if derivador.generaciones != nil {
		t.Fatal("el proveedor conservo la ventana tras borrarla")
	}
	for indice := range respaldoPropietario {
		if respaldoPropietario[indice].localizador.material != ([sha256.Size]byte{}) ||
			respaldoPropietario[indice].huellaSolicitud.material != ([sha256.Size]byte{}) {
			t.Fatal("el borrado no puso a cero el material de clave")
		}
	}
}

func TestDerivadorCompuestoNoExponeGettersDeClaves(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, bytes.NewBuffer(nil))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(composicion.derivadorIdempotencia.borrar)
	puerto, err := composicion.DerivadorIdentidadesBorrador()
	if err != nil {
		t.Fatal(err)
	}
	tipo := reflect.TypeOf(puerto)
	if tipo.NumMethod() != 1 || tipo.Method(0).Name != "Derivar" {
		t.Fatalf("superficie publica inesperada del derivador: %v", tipo)
	}
}

func TestConfiguracionIdempotenciaJSONEsCerradaYLaVentanaEsExacta(t *testing.T) {
	casos := map[string]func([]byte) []byte{
		"clave duplicada": func(valida []byte) []byte {
			return bytes.Replace(valida, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1)
		},
		"campo desconocido": func(valida []byte) []byte {
			return append(bytes.TrimSuffix(valida, []byte("}")), []byte(`,"desconocido":true}`)...)
		},
		"generaciones ascendentes": func(_ []byte) []byte {
			return configuracionIdempotenciaJSONPrueba(1, 2)
		},
		"referencia no exacta": func(valida []byte) []byte {
			return bytes.Replace(valida, []byte("localizador:desarrollo:v2"), []byte("localizador:desarrollo:v9"), 1)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			cfg, rutas := generarMaterialDesarrolloPrueba(t)
			valida, err := os.ReadFile(rutas.IdempotencyHMACConfig)
			if err != nil {
				t.Fatal(err)
			}
			defer borrarBytes(valida)
			alterada := mutar(valida)
			defer borrarBytes(alterada)
			if err := os.WriteFile(rutas.IdempotencyHMACConfig, alterada, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := NuevaComposicionSeguridadDesarrollo(cfg, bytes.NewBuffer(nil)); !errors.Is(
				err, ErrMaterialDesarrolloInvalido,
			) {
				t.Fatalf("configuracion ambigua aceptada: %v", err)
			}
		})
	}
}

func TestOrdenDeCamposJSONNoCambiaElAcuerdo(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	generaciones, err := json.Marshal([]referenciaGeneracionIdempotenciaDesarrollo{
		{
			Generacion: 2, ReferenciaLocalizador: referenciaLocalizadorIdempotenciaDesarrollo(2),
			ReferenciaHuellaSolicitud: referenciaHuellaIdempotenciaDesarrollo(2),
		},
		{
			Generacion: 1, ReferenciaLocalizador: referenciaLocalizadorIdempotenciaDesarrollo(1),
			ReferenciaHuellaSolicitud: referenciaHuellaIdempotenciaDesarrollo(1),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	reordenada := []byte(fmt.Sprintf(
		`{"generaciones":%s,"version_esquema_hmac":2,"autoridad":"%s","esquema":"%s","version":1}`,
		generaciones, AutoridadNoAutoritativa, esquemaMaterialIdempotenciaDesarrolloV1,
	))
	if err := os.WriteFile(rutas.IdempotencyHMACConfig, reordenada, 0o600); err != nil {
		t.Fatal(err)
	}
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, bytes.NewBuffer(nil))
	if err != nil {
		t.Fatalf("el orden irrelevante de campos JSON cambio el acuerdo: %v", err)
	}
	t.Cleanup(composicion.derivadorIdempotencia.borrar)
}

func TestComposicionRechazaReutilizarKMSOTSAComoClaveIdempotencia(t *testing.T) {
	for _, nombre := range []string{"KMS", "TSA"} {
		t.Run(nombre, func(t *testing.T) {
			cfg, rutas := generarMaterialDesarrolloPrueba(t)
			origen := rutas.KMSSecret
			if nombre == "TSA" {
				origen = rutas.TSASecret
			}
			contenido, err := os.ReadFile(origen)
			if err != nil {
				t.Fatal(err)
			}
			defer borrarBytes(contenido)
			destino := rutaClaveIdempotenciaDesarrollo(cfg.DevelopmentMaterialDir, 2, "localizador")
			if err := os.WriteFile(destino, contenido, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := NuevaComposicionSeguridadDesarrollo(cfg, bytes.NewBuffer(nil)); !errors.Is(
				err, ErrMaterialDesarrolloInvalido,
			) {
				t.Fatalf("se reutilizo la clave %s: %v", nombre, err)
			}
		})
	}
}

func TestComposicionRechazaCompartirClaveEntreLYF(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	origen := rutaClaveIdempotenciaDesarrollo(cfg.DevelopmentMaterialDir, 2, "localizador")
	destino := rutaClaveIdempotenciaDesarrollo(cfg.DevelopmentMaterialDir, 2, "huella-solicitud")
	contenido, err := os.ReadFile(origen)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(contenido)
	if err := os.WriteFile(destino, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevaComposicionSeguridadDesarrollo(cfg, bytes.NewBuffer(nil)); !errors.Is(
		err, ErrMaterialDesarrolloInvalido,
	) {
		t.Fatalf("L y F compartieron clave: %v", err)
	}
}

func configuracionIdempotenciaJSONPrueba(generaciones ...uint32) []byte {
	archivo := archivoMaterialIdempotenciaDesarrollo{
		Version: versionMaterialIdempotenciaDesarrollo, Esquema: esquemaMaterialIdempotenciaDesarrolloV1,
		Autoridad: AutoridadNoAutoritativa, VersionEsquemaHMAC: versionHMACIdempotenciaBorrador,
		Generaciones: make([]referenciaGeneracionIdempotenciaDesarrollo, 0, len(generaciones)),
	}
	for _, generacion := range generaciones {
		archivo.Generaciones = append(archivo.Generaciones, referenciaGeneracionIdempotenciaDesarrollo{
			Generacion: generacion, ReferenciaLocalizador: referenciaLocalizadorIdempotenciaDesarrollo(generacion),
			ReferenciaHuellaSolicitud: referenciaHuellaIdempotenciaDesarrollo(generacion),
		})
	}
	contenido, err := json.Marshal(archivo)
	if err != nil {
		panic(strings.TrimSpace(err.Error()))
	}
	return contenido
}
