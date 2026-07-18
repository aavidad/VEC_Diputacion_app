package bootstrap

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
	"time"

	josecipher "github.com/go-jose/go-jose/v4/cipher"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

func nuevosProveedoresKMSPrueba(
	t *testing.T,
) (*emisorKMSDesarrollo, *revalidadorKMSDesarrollo, *verificadorFirmasKMSDesarrollo) {
	t.Helper()
	publicaAtestacion, privadaAtestacion, err := ed25519.GenerateKey(
		bytes.NewReader(bytes.Repeat([]byte{0x11}, ed25519.SeedSize)),
	)
	if err != nil {
		t.Fatal(err)
	}
	publicaRevalidacion, privadaRevalidacion, err := ed25519.GenerateKey(
		bytes.NewReader(bytes.Repeat([]byte{0x22}, ed25519.SeedSize)),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaAtestacion := huellaPublicaEd25519Prueba(t, publicaAtestacion)
	huellaRevalidacion := huellaPublicaEd25519Prueba(t, publicaRevalidacion)
	var maestra [sha256.Size]byte
	for indice := range maestra {
		maestra[indice] = byte(indice + 1)
	}
	emisor, revalidador, verificador, err := nuevosProveedoresKMSDesarrollo(
		maestra, privadaAtestacion, publicaAtestacion, huellaAtestacion,
		privadaRevalidacion, publicaRevalidacion, huellaRevalidacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return emisor, revalidador, verificador
}

func huellaPublicaEd25519Prueba(t *testing.T, publica ed25519.PublicKey) [sha256.Size]byte {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(der)
}

func TestA256KWUsaVectorConocidoRFC3394(t *testing.T) {
	kek, _ := hex.DecodeString("000102030405060708090A0B0C0D0E0F101112131415161718191A1B1C1D1E1F")
	clave, _ := hex.DecodeString("00112233445566778899AABBCCDDEEFF0001020304050607")
	esperada, _ := hex.DecodeString("A8F9BC1612C68B3FF6E6F4FBE30E71E4769C8B80A32CB8958CD5D17D6B254DA1")
	bloque, err := aes.NewCipher(kek)
	if err != nil {
		t.Fatal(err)
	}
	envuelta, err := josecipher.KeyWrap(bloque, clave)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(envuelta, esperada) {
		t.Fatalf("A256KW = %X; esperado %X", envuelta, esperada)
	}
	recuperada, err := josecipher.KeyUnwrap(bloque, envuelta)
	if err != nil || !bytes.Equal(recuperada, clave) {
		t.Fatalf("round-trip RFC3394 = %X, %v", recuperada, err)
	}
}

func TestKMSDesarrolloCifraEnvuelveYDetectaAlteraciones(t *testing.T) {
	kms, _, _ := nuevosProveedoresKMSPrueba(t)
	kms.aleatorio = bytes.NewReader(bytes.Repeat([]byte{0x5a}, 44))
	claro := []byte("borrador de convocatoria no autoritativo")
	aad := []byte("aad:reserva:revision:1")
	nonce, cifrado, envuelta, err := kms.cifrarContenido(claro, aad)
	if err != nil {
		t.Fatal(err)
	}
	bloqueEnvoltura, _ := aes.NewCipher(kms.claveEnvoltura[:])
	dek, err := josecipher.KeyUnwrap(bloqueEnvoltura, envuelta)
	if err != nil {
		t.Fatal(err)
	}
	bloqueContenido, _ := aes.NewCipher(dek)
	aead, _ := cipher.NewGCM(bloqueContenido)
	recuperado, err := aead.Open(nil, nonce, cifrado, aad)
	if err != nil || !bytes.Equal(recuperado, claro) {
		t.Fatalf("round-trip = %q, %v", recuperado, err)
	}

	cifradoAlterado := append([]byte(nil), cifrado...)
	cifradoAlterado[0] ^= 0x80
	if _, err := aead.Open(nil, nonce, cifradoAlterado, aad); err == nil {
		t.Fatal("AES-GCM acepto texto alterado")
	}
	envueltaAlterada := append([]byte(nil), envuelta...)
	envueltaAlterada[len(envueltaAlterada)-1] ^= 0x01
	if _, err := josecipher.KeyUnwrap(bloqueEnvoltura, envueltaAlterada); err == nil {
		t.Fatal("A256KW acepto clave envuelta alterada")
	}
}

func TestKMSDesarrolloSeparaEmisionRevalidacionYVerificacionTrasReinicio(t *testing.T) {
	emisor, revalidador, verificador := nuevosProveedoresKMSPrueba(t)
	perfil, err := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(
		"perfil:cifrado:desarrollo:v1", 1, strings.Repeat("a", sha256.Size*2),
		algoritmoContenidoDesarrollo, algoritmoEnvolturaDesarrollo,
	)
	if err != nil {
		t.Fatal(err)
	}
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		"desarrollo", gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		"proveedor:seguridad:desarrollo:t21", false,
	)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	atestacion, err := gobiernoconvocatorias.NuevaAtestacionKMSBorrador(
		"atestacion:kms:desarrollo:prueba:001", 1, perfil, claveMaestraDesarrolloRef, 1,
		strings.Repeat("b", sha256.Size*2), strings.Repeat("c", sha256.Size*2),
		strings.Repeat("d", sha256.Size*2), verificadorAtestacionDesarrolloRef,
		procedencia, algoritmoFirmaKMSDesarrollo,
		hex.EncodeToString(emisor.huellaPublicaAtestacion[:]), base, base.Add(4*time.Minute),
		func(preimagen []byte) ([]byte, error) {
			return ed25519.Sign(emisor.firmaAtestacion, preimagen), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	identidad := gobiernoconvocatorias.ProyeccionIdentidadOperacion{
		Localizador: gobiernoconvocatorias.ProyeccionHMACDiario{
			VersionEsquema: 1, Dominio: "localizador",
			ClaveRef:        "clave:hmac:convocatorias:localizador:desarrollo-v1",
			GeneracionClave: 1, ValorHMACSHA256: strings.Repeat("e", sha256.Size*2),
		},
		HuellaSolicitud: gobiernoconvocatorias.ProyeccionHMACDiario{
			VersionEsquema: 1, Dominio: "huella_solicitud",
			ClaveRef:        "clave:hmac:convocatorias:huella:desarrollo-v1",
			GeneracionClave: 1, ValorHMACSHA256: strings.Repeat("f", sha256.Size*2),
		},
	}
	solicitud := gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador{
		AtestacionKMS: atestacion, IdentidadPrimaria: identidad,
		HuellaAAD:                atestacion.HuellaAAD,
		HuellaCuerpoReciboSHA256: strings.Repeat("9", sha256.Size*2),
		Revision:                 7, Cercado: 3,
		ArrendamientoVenceEn:     base.Add(6 * time.Minute),
		ConfirmacionSolicitadaEn: base.Add(30 * time.Second),
		SolicitadaEn:             base.Add(time.Minute),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatal(err)
	}
	resultado, err := revalidador.RevalidarAtestacionKMS(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	// Simula un reinicio: el nuevo proceso recibe solo copias de las publicas.
	reiniciado := &verificadorFirmasKMSDesarrollo{
		verificadorAtestacion:     append(ed25519.PublicKey(nil), verificador.verificadorAtestacion...),
		huellaPublicaAtestacion:   verificador.huellaPublicaAtestacion,
		verificadorRevalidacion:   append(ed25519.PublicKey(nil), verificador.verificadorRevalidacion...),
		huellaPublicaRevalidacion: verificador.huellaPublicaRevalidacion,
	}
	if err := reiniciado.VerificarAtestacion(atestacion); err != nil {
		t.Fatalf("atestacion Ed25519 tras reinicio: %v", err)
	}
	if err := reiniciado.VerificarRevalidacion(solicitud, resultado); err != nil {
		t.Fatalf("revalidacion Ed25519 tras reinicio: %v", err)
	}

	procedenciaAjena, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		"desarrollo", gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		"proveedor:seguridad:desarrollo:ajeno", false,
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionAjena, err := gobiernoconvocatorias.NuevaAtestacionKMSBorrador(
		"atestacion:kms:desarrollo:prueba:ajena", 1, perfil, claveMaestraDesarrolloRef, 1,
		atestacion.HuellaAAD, atestacion.HuellaEnvolturaSHA256, atestacion.HuellaSobreSHA256,
		verificadorAtestacionDesarrolloRef, procedenciaAjena, algoritmoFirmaKMSDesarrollo,
		hex.EncodeToString(emisor.huellaPublicaAtestacion[:]), base, base.Add(4*time.Minute),
		func(preimagen []byte) ([]byte, error) {
			return ed25519.Sign(emisor.firmaAtestacion, preimagen), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudAjena := solicitud
	solicitudAjena.AtestacionKMS = atestacionAjena
	if solicitudAjena.Validar() != nil {
		t.Fatal("la evidencia ajena debía ser estructuralmente válida para probar la frontera T21")
	}
	procedenciaReetiquetada := procedencia
	procedenciaReetiquetada.Autoridad = gobiernoconvocatorias.AutoridadActoAutoritativa
	procedenciaReetiquetada.MigrableProduccion = true
	if procedenciaKMSDesarrolloValida(procedenciaAjena) ||
		procedenciaKMSDesarrolloValida(procedenciaReetiquetada) {
		t.Fatal("la frontera KMS aceptó una procedencia reetiquetada")
	}
	if _, err := revalidador.RevalidarAtestacionKMS(context.Background(), solicitudAjena); err == nil {
		t.Fatal("el revalidador T21 firmó una atestación de otro proveedor")
	}
	if err := reiniciado.VerificarAtestacion(atestacionAjena); err == nil {
		t.Fatal("el verificador T21 aceptó una atestación de otro proveedor")
	}

	preimagen, algoritmo, referencia, huella, firma, err := atestacion.DatosParaVerificacionFirma()
	if err != nil {
		t.Fatal(err)
	}
	firma[0] ^= 0x80
	firmaAlterada, err := gobiernoconvocatorias.RestaurarFirmaEvidenciaBorrador(
		algoritmo, referencia, huella, preimagen, base64.RawURLEncoding.EncodeToString(firma),
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionAlterada := atestacion
	atestacionAlterada.Firma = firmaAlterada
	if err := reiniciado.VerificarAtestacion(atestacionAlterada); err == nil {
		t.Fatal("el verificador acepto una firma de atestacion alterada")
	}
	preimagenResultado, algoritmoResultado, referenciaResultado, huellaResultado,
		firmaResultado, err := resultado.DatosParaVerificacionFirma(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	firmaResultado[len(firmaResultado)-1] ^= 0x01
	firmaResultadoAlterada, err := gobiernoconvocatorias.RestaurarFirmaEvidenciaBorrador(
		algoritmoResultado, referenciaResultado, huellaResultado, preimagenResultado,
		base64.RawURLEncoding.EncodeToString(firmaResultado),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultadoAlterado := resultado
	resultadoAlterado.Firma = firmaResultadoAlterada
	if err := reiniciado.VerificarRevalidacion(solicitud, resultadoAlterado); err == nil {
		t.Fatal("el verificador acepto una firma de revalidacion alterada")
	}

	publicaAjena, _, err := ed25519.GenerateKey(
		bytes.NewReader(bytes.Repeat([]byte{0x33}, ed25519.SeedSize)),
	)
	if err != nil {
		t.Fatal(err)
	}
	verificadorAjeno := *reiniciado
	verificadorAjeno.verificadorAtestacion = publicaAjena
	if err := verificadorAjeno.VerificarAtestacion(atestacion); err == nil {
		t.Fatal("el verificador acepto una clave publica ajena")
	}
}

func TestKMSDesarrolloRechazaReutilizarParejaEntreResponsabilidades(t *testing.T) {
	publica, privada, err := ed25519.GenerateKey(
		bytes.NewReader(bytes.Repeat([]byte{0x44}, ed25519.SeedSize)),
	)
	if err != nil {
		t.Fatal(err)
	}
	huella := huellaPublicaEd25519Prueba(t, publica)
	if _, _, _, err := nuevosProveedoresKMSDesarrollo(
		[sha256.Size]byte{1}, privada, publica, huella, privada, publica, huella,
	); err == nil {
		t.Fatal("se acepto la misma pareja Ed25519 para emision y revalidacion")
	}
}

func TestKMSDesarrolloSeparaCredencialesRolesYPrivadas(t *testing.T) {
	emisor, revalidador, verificador := nuevosProveedoresKMSPrueba(t)
	identidades := []gobiernoconvocatorias.IdentidadAutoridadBorrador{
		emisor.IdentidadAutoridadBorrador(),
		revalidador.IdentidadAutoridadBorrador(),
		verificador.IdentidadAutoridadBorrador(),
	}
	for indice, identidad := range identidades {
		if identidad.ProveedorRef == "" || identidad.InstanciaRef == "" ||
			identidad.CredencialRef == "" || identidad.RolRef == "" {
			t.Fatalf("identidad %d incompleta: %+v", indice, identidad)
		}
		for anterior := 0; anterior < indice; anterior++ {
			otra := identidades[anterior]
			if identidad.ProveedorRef == otra.ProveedorRef || identidad.InstanciaRef == otra.InstanciaRef ||
				identidad.CredencialRef == otra.CredencialRef || identidad.RolRef == otra.RolRef {
				t.Fatalf("responsabilidades KMS comparten autoridad: %+v / %+v", otra, identidad)
			}
		}
	}
	tipoPrivada := reflect.TypeOf(ed25519.PrivateKey(nil))
	tipoVerificador := reflect.TypeOf(*verificador)
	for indice := 0; indice < tipoVerificador.NumField(); indice++ {
		if tipoVerificador.Field(indice).Type == tipoPrivada {
			t.Fatalf("el verificador publico conserva una privada en %s", tipoVerificador.Field(indice).Name)
		}
	}
}
