package verificacioncose

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"

	gocose "github.com/veraison/go-cose"
)

type materialFirmaPrueba struct {
	algoritmo           Algoritmo
	algoritmoBiblioteca gocose.Algorithm
	publica             crypto.PublicKey
	privada             crypto.Signer
}

func TestVerificadorComunCompruebaEdDSAYES256SinAlias(t *testing.T) {
	for _, algoritmo := range []Algoritmo{AlgoritmoEdDSA, AlgoritmoES256} {
		t.Run(string(algoritmo), func(t *testing.T) {
			material := generarMaterialPrueba(t, algoritmo)
			claveID := []byte("clave:comun:activa")
			payload := []byte("mensaje-canonico-firmado")
			aad := []byte("vec.audiencia.prueba\x00recibo")
			verificador, err := NuevoVerificadorClave(claveID, algoritmo, material.publica)
			if err != nil {
				t.Fatalf("crear verificador: %v", err)
			}
			contenido := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
			inspeccion, err := InspeccionarSobreSign1(contenido, len(contenido))
			if err != nil {
				t.Fatalf("inspeccionar: %v", err)
			}
			payloadEsperado := append([]byte(nil), payload...)
			aadEsperado := append([]byte(nil), aad...)
			claveEsperada := append([]byte(nil), claveID...)

			// Ninguna entrada mutable sigue perteneciendo a los objetos creados.
			contenido[0] ^= 0xff
			payload[0] ^= 0xff
			aad[0] ^= 0xff
			claveID[0] ^= 0xff
			mutarClavePublicaPrueba(material.publica)

			if err := verificador.Verificar(inspeccion, payloadEsperado, aadEsperado); err != nil {
				t.Fatalf("verificar con copias defensivas: %v", err)
			}
			obtenidoAlgoritmo, err := inspeccion.Algoritmo()
			if err != nil || obtenidoAlgoritmo != algoritmo {
				t.Fatalf("algoritmo = %q, err=%v", obtenidoAlgoritmo, err)
			}
			obtenidaClave, err := inspeccion.ClaveID()
			if err != nil || !bytes.Equal(obtenidaClave, claveEsperada) {
				t.Fatalf("kid = %x, err=%v", obtenidaClave, err)
			}
			obtenidaClave[0] ^= 0xff
			segundaClave, _ := inspeccion.ClaveID()
			if !bytes.Equal(segundaClave, claveEsperada) {
				t.Fatal("el accessor expuso un alias mutable")
			}
		})
	}
}

func TestVerificadorComunCompruebaPayloadSeparadoSinReinterpretarIncrustado(t *testing.T) {
	for _, algoritmo := range []Algoritmo{AlgoritmoEdDSA, AlgoritmoES256} {
		t.Run(string(algoritmo), func(t *testing.T) {
			material := generarMaterialPrueba(t, algoritmo)
			claveID := []byte("clave:comun:payload-separado")
			payload := []byte("mensaje-canonico-no-duplicado-en-el-sobre")
			aad := []byte("vec.audiencia.prueba\x00payload-separado")
			contenido := firmarSobreSeparadoPrueba(t, material, claveID, payload, aad)
			inspeccion, err := InspeccionarSobreSign1(contenido, len(contenido))
			if err != nil {
				t.Fatalf("inspeccionar sobre separado: %v", err)
			}
			verificador, err := NuevoVerificadorClave(claveID, algoritmo, material.publica)
			if err != nil {
				t.Fatal(err)
			}
			if err := verificador.VerificarPayloadSeparado(
				inspeccion,
				payload,
				aad,
			); err != nil {
				t.Fatalf("verificar payload separado: %v", err)
			}
			if err := verificador.Verificar(
				inspeccion,
				payload,
				aad,
			); !errors.Is(err, ErrVerificacionFirmaSign1Fallida) {
				t.Fatalf("el modo incrustado acepto payload null: %v", err)
			}

			incrustado := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
			inspeccionIncrustada, err := InspeccionarSobreSign1(incrustado, len(incrustado))
			if err != nil {
				t.Fatal(err)
			}
			if err := verificador.VerificarPayloadSeparado(
				inspeccionIncrustada,
				payload,
				aad,
			); !errors.Is(err, ErrVerificacionFirmaSign1Fallida) {
				t.Fatalf("el modo separado acepto payload incrustado: %v", err)
			}
		})
	}
}

func TestVerificadorComunPayloadSeparadoRechazaCruces(t *testing.T) {
	material := generarMaterialPrueba(t, AlgoritmoEdDSA)
	claveID := []byte("clave:comun:separado:cruces")
	payload := []byte("payload-separado-original")
	aad := []byte("audiencia-separada-original")
	contenido := firmarSobreSeparadoPrueba(t, material, claveID, payload, aad)
	inspeccion, err := InspeccionarSobreSign1(contenido, len(contenido))
	if err != nil {
		t.Fatal(err)
	}
	verificador, err := NuevoVerificadorClave(claveID, AlgoritmoEdDSA, material.publica)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre  string
		payload []byte
		aad     []byte
	}{
		{"payload_distinto", []byte("payload-separado-ajeno"), aad},
		{"aad_distinto", payload, []byte("audiencia-separada-ajena")},
		{"payload_vacio", nil, aad},
		{"aad_vacio", payload, nil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := verificador.VerificarPayloadSeparado(
				inspeccion,
				caso.payload,
				caso.aad,
			); !errors.Is(err, ErrVerificacionFirmaSign1Fallida) {
				t.Fatalf("cruce aceptado: %v", err)
			}
		})
	}
}

func TestVerificadorComunRechazaCrucesYClavesIncompatibles(t *testing.T) {
	material := generarMaterialPrueba(t, AlgoritmoEdDSA)
	claveID := []byte("clave:cruces:1")
	payload := []byte("payload-cruces")
	aad := []byte("audiencia-cruces")
	contenido := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
	inspeccion, err := InspeccionarSobreSign1(contenido, len(contenido))
	if err != nil {
		t.Fatal(err)
	}
	verificador, err := NuevoVerificadorClave(claveID, AlgoritmoEdDSA, material.publica)
	if err != nil {
		t.Fatal(err)
	}
	otroMaterial := generarMaterialPrueba(t, AlgoritmoEdDSA)
	verificadorOtraClave, _ := NuevoVerificadorClave(claveID, AlgoritmoEdDSA, otroMaterial.publica)
	verificadorOtroID, _ := NuevoVerificadorClave([]byte("clave:cruces:2"), AlgoritmoEdDSA, material.publica)

	casos := map[string]struct {
		verificador *VerificadorClave
		payload     []byte
		aad         []byte
	}{
		"payload":   {verificador, []byte("payload-distinto"), aad},
		"aad":       {verificador, payload, []byte("audiencia-distinta")},
		"clave":     {verificadorOtraClave, payload, aad},
		"clave_id":  {verificadorOtroID, payload, aad},
		"nulo":      {nil, payload, aad},
		"payload_0": {verificador, nil, aad},
		"aad_0":     {verificador, payload, nil},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := caso.verificador.Verificar(
				inspeccion,
				caso.payload,
				caso.aad,
			); !errors.Is(err, ErrVerificacionFirmaSign1Fallida) {
				t.Fatalf("cruce aceptado: %v", err)
			}
		})
	}

	if _, err := NuevoVerificadorClave(
		claveID,
		AlgoritmoES256,
		material.publica,
	); !errors.Is(err, ErrConfiguracionClaveInvalida) {
		t.Fatalf("algoritmo y clave incompatibles aceptados: %v", err)
	}
}

func TestInspeccionComunRechazaCabecerasFormaYFirmaNoCanonicas(t *testing.T) {
	material := generarMaterialPrueba(t, AlgoritmoEdDSA)
	claveID := []byte("kid-canonico")
	payload := []byte("payload-canonico")
	aad := []byte("audiencia-canonica")

	casos := map[string][]byte{
		"protegida_adicional": firmarSobrePrueba(
			t, material, claveID, payload, aad,
			func(m *gocose.Sign1Message) {
				m.Headers.Protected[gocose.HeaderLabelContentType] = "application/json"
			}, nil,
		),
		"no_protegida": firmarSobrePrueba(
			t, material, claveID, payload, aad,
			func(m *gocose.Sign1Message) {
				m.Headers.Unprotected[gocose.HeaderLabelContentType] = "application/json"
			}, nil,
		),
		"kid_nulo": firmarSobrePrueba(
			t, material, make([]byte, 4), payload, aad, nil, nil,
		),
		"algoritmo_no_admitido": firmarSobrePrueba(
			t, material, claveID, payload, aad, nil,
			func(m *gocose.Sign1Message) {
				m.Headers.Protected.SetAlgorithm(gocose.AlgorithmES384)
			},
		),
		"firma_corta": firmarSobrePrueba(
			t, material, claveID, payload, aad, nil,
			func(m *gocose.Sign1Message) { m.Signature = m.Signature[:63] },
		),
	}
	canonico := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
	patron := append([]byte{0x40 | byte(len(payload))}, payload...)
	posicion := bytes.Index(canonico, patron)
	if posicion < 0 || bytes.Count(canonico, patron) != 1 {
		t.Fatal("el vector no localizo el payload de forma univoca")
	}
	noMinimo := make([]byte, 0, len(canonico)+1)
	noMinimo = append(noMinimo, canonico[:posicion]...)
	noMinimo = append(noMinimo, 0x58, byte(len(payload)))
	noMinimo = append(noMinimo, canonico[posicion+1:]...)
	casos["bstr_no_minimo"] = noMinimo

	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := InspeccionarSobreSign1(
				contenido,
				len(contenido),
			); !errors.Is(err, ErrSobreSign1Invalido) {
				t.Fatalf("forma hostil aceptada: %v", err)
			}
		})
	}
}

func TestInspeccionComunRechazaES256HighS(t *testing.T) {
	material := generarMaterialPrueba(t, AlgoritmoES256)
	contenido := firmarSobrePrueba(
		t, material, []byte("clave:high-s"), []byte("payload-high-s"),
		[]byte("audiencia-high-s"), nil,
		func(m *gocose.Sign1Message) {
			orden := elliptic.P256().Params().N
			s := gocose.OS2IP(m.Signature[32:])
			s.Sub(orden, s).FillBytes(m.Signature[32:])
		},
	)
	if _, err := InspeccionarSobreSign1(
		contenido,
		len(contenido),
	); !errors.Is(err, ErrSobreSign1Invalido) {
		t.Fatalf("firma high-S aceptada: %v", err)
	}
}

func TestTiposComunesSonNominalesRedactadosYNoSerializables(t *testing.T) {
	material := generarMaterialPrueba(t, AlgoritmoEdDSA)
	claveID := []byte("clave:no-filtrar")
	payload := []byte("payload-no-filtrar")
	aad := []byte("audiencia-no-filtrar")
	contenido := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
	inspeccion, _ := InspeccionarSobreSign1(contenido, len(contenido))
	verificador, _ := NuevoVerificadorClave(claveID, AlgoritmoEdDSA, material.publica)

	texto := fmt.Sprintf("%v|%+v|%#v|%v", inspeccion, inspeccion, inspeccion, verificador)
	if strings.Contains(texto, string(claveID)) || strings.Contains(texto, string(payload)) {
		t.Fatalf("se filtraron datos: %s", texto)
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info(
		"prueba",
		"sobre", inspeccion,
		"verificador", verificador,
	)
	if strings.Contains(registro.String(), string(claveID)) ||
		strings.Contains(registro.String(), string(payload)) {
		t.Fatalf("slog filtro datos: %s", registro.String())
	}
	if _, err := json.Marshal(inspeccion); !errors.Is(err, ErrSerializacionCOSEProhibida) {
		t.Fatalf("sobre serializado: %v", err)
	}
	if _, err := json.Marshal(verificador); !errors.Is(err, ErrSerializacionCOSEProhibida) {
		t.Fatalf("verificador serializado: %v", err)
	}
	if _, err := inspeccion.MarshalBinary(); !errors.Is(err, ErrSerializacionCOSEProhibida) {
		t.Fatalf("sobre binario: %v", err)
	}
	if _, err := verificador.MarshalText(); !errors.Is(err, ErrSerializacionCOSEProhibida) {
		t.Fatalf("verificador texto: %v", err)
	}

	var cero SobreSign1Estricto
	if _, err := cero.ClaveID(); !errors.Is(err, ErrSobreSign1Invalido) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
	if err := (*VerificadorClave)(nil).Verificar(
		inspeccion,
		payload,
		aad,
	); !errors.Is(err, ErrVerificacionFirmaSign1Fallida) {
		t.Fatalf("verificador nulo aceptado: %v", err)
	}
}

func TestInspeccionComunAplicaLimiteAntesDeInterpretar(t *testing.T) {
	for _, caso := range []struct {
		contenido []byte
		limite    int
	}{
		{nil, tamanoMinimoSobreSign1},
		{make([]byte, tamanoMinimoSobreSign1), tamanoMinimoSobreSign1 - 1},
		{make([]byte, tamanoMinimoSobreSign1), TamanoMaximoAbsolutoSobreSign1 + 1},
		{make([]byte, TamanoMaximoAbsolutoSobreSign1+1), TamanoMaximoAbsolutoSobreSign1},
	} {
		if _, err := InspeccionarSobreSign1(
			caso.contenido,
			caso.limite,
		); !errors.Is(err, ErrSobreSign1Invalido) {
			t.Fatalf("limite invalido aceptado: len=%d limite=%d err=%v", len(caso.contenido), caso.limite, err)
		}
	}
}

func FuzzInspeccionarSobreSign1NoEntraEnPanico(f *testing.F) {
	f.Add([]byte{})
	f.Add(bytes.Repeat([]byte{0xff}, tamanoMinimoSobreSign1))
	f.Fuzz(func(t *testing.T, contenido []byte) {
		_, _ = InspeccionarSobreSign1(contenido, TamanoMaximoAbsolutoSobreSign1)
	})
}

func generarMaterialPrueba(t *testing.T, algoritmo Algoritmo) materialFirmaPrueba {
	t.Helper()
	material := materialFirmaPrueba{algoritmo: algoritmo}
	switch algoritmo {
	case AlgoritmoEdDSA:
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		material.algoritmoBiblioteca = gocose.AlgorithmEdDSA
		material.publica, material.privada = publica, privada
	case AlgoritmoES256:
		privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		material.algoritmoBiblioteca = gocose.AlgorithmES256
		material.publica, material.privada = &privada.PublicKey, privada
	default:
		t.Fatalf("algoritmo no soportado: %q", algoritmo)
	}
	return material
}

func firmarSobrePrueba(
	t *testing.T,
	material materialFirmaPrueba,
	claveID, payload, aad []byte,
	antesDeFirmar func(*gocose.Sign1Message),
	despuesDeFirmar func(*gocose.Sign1Message),
) []byte {
	t.Helper()
	mensaje := gocose.NewSign1Message()
	mensaje.Headers.Protected.SetAlgorithm(material.algoritmoBiblioteca)
	mensaje.Headers.Protected[gocose.HeaderLabelKeyID] = append([]byte(nil), claveID...)
	mensaje.Payload = append([]byte(nil), payload...)
	if antesDeFirmar != nil {
		antesDeFirmar(mensaje)
	}
	firmante, err := gocose.NewSigner(material.algoritmoBiblioteca, material.privada)
	if err != nil {
		t.Fatal(err)
	}
	if err := mensaje.Sign(rand.Reader, aad, firmante); err != nil {
		t.Fatal(err)
	}
	normalizarFirmaES256Prueba(t, material.algoritmo, mensaje)
	if despuesDeFirmar != nil {
		despuesDeFirmar(mensaje)
	}
	mensaje.Headers.RawProtected = nil
	mensaje.Headers.RawUnprotected = nil
	contenido, err := mensaje.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func firmarSobreSeparadoPrueba(
	t *testing.T,
	material materialFirmaPrueba,
	claveID, payload, aad []byte,
) []byte {
	t.Helper()
	contenido := firmarSobrePrueba(t, material, claveID, payload, aad, nil, nil)
	var mensaje gocose.Sign1Message
	if err := mensaje.UnmarshalCBOR(contenido); err != nil {
		t.Fatal(err)
	}
	mensaje.Payload = nil
	mensaje.Headers.RawProtected = nil
	mensaje.Headers.RawUnprotected = nil
	separado, err := mensaje.MarshalCBOR()
	if err != nil {
		t.Fatal(err)
	}
	return separado
}

func normalizarFirmaES256Prueba(
	t *testing.T,
	algoritmo Algoritmo,
	mensaje *gocose.Sign1Message,
) {
	t.Helper()
	if algoritmo != AlgoritmoES256 {
		return
	}
	if len(mensaje.Signature) != 64 {
		t.Fatalf("firma ES256 inesperada: %d", len(mensaje.Signature))
	}
	orden := elliptic.P256().Params().N
	s := gocose.OS2IP(mensaje.Signature[32:])
	mitad := gocose.OS2IP(orden.Bytes())
	mitad.Rsh(mitad, 1)
	if s.Cmp(mitad) > 0 {
		s.Sub(orden, s).FillBytes(mensaje.Signature[32:])
	}
}

func mutarClavePublicaPrueba(clave crypto.PublicKey) {
	switch publica := clave.(type) {
	case ed25519.PublicKey:
		publica[0] ^= 0xff
	case *ecdsa.PublicKey:
		publica.X.Add(publica.X, bigUnoPrueba())
	}
}

func bigUnoPrueba() *big.Int {
	return big.NewInt(1)
}
