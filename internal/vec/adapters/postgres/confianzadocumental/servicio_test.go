package confianzadocumental

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var instanteConfianzaDocumentalPrueba = time.Date(
	2026, time.July, 15, 17, 0, 0, 123_456_000, time.UTC,
)

const audienciaDespliegueAtestacionPDPPrueba = "vec-diputacion/pruebas/vec/autorizacion"

type relojConfianzaDocumentalFijo struct {
	ahora time.Time
}

func (r *relojConfianzaDocumentalFijo) Ahora() time.Time { return r.ahora }

type materialFirmaCOSEPrueba struct {
	algoritmoDocumental AlgoritmoCOSEDocumental
	algoritmoCOSE       cose.Algorithm
	claveID             []byte
	privada             crypto.Signer
	publica             crypto.PublicKey
}

func TestServicioVerificaCOSESign1RealEdDSAYES256(t *testing.T) {
	for _, algoritmo := range []AlgoritmoCOSEDocumental{
		AlgoritmoCOSEDocumentalEdDSA,
		AlgoritmoCOSEDocumentalES256,
	} {
		t.Run(string(algoritmo), func(t *testing.T) {
			material := generarMaterialFirmaCOSEPrueba(t, algoritmo, []byte("clave:documental:activa"))
			payloadOriginal := []byte(`{"operacion":"recibo:1","resultado":"conforme"}`)
			solicitud := nuevaSolicitudCOSEPrueba(
				t, payloadOriginal, AudienciaCOSEReciboComponenteDocumental,
			)
			payloadFirmado, err := solicitud.PayloadEsperado()
			if err != nil {
				t.Fatalf("obtener payload: %v", err)
			}
			payloadOriginal[0] ^= 0xff
			aad, err := solicitud.AADExterno()
			if err != nil {
				t.Fatalf("obtener AAD: %v", err)
			}
			aad[0] ^= 0xff
			sobre := firmarSobreCOSEPrueba(t, material, payloadFirmado, solicitud, nil, nil)
			servicio, configuracion := nuevoServicioCOSEPrueba(
				t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
				AudienciaCOSEReciboComponenteDocumental,
			)

			prueba, err := servicio.VerificarCOSESign1(context.Background(), solicitud, sobre)
			if err != nil {
				t.Fatalf("verificar COSE real: %v", err)
			}
			if prueba.Validar() != nil || prueba.ValidarPara(solicitud, sobre) != nil {
				t.Fatal("la autoridad local no conserva sus vinculos")
			}
			obtenido, err := prueba.Algoritmo()
			if err != nil || obtenido != algoritmo {
				t.Fatalf("algoritmo inesperado: %q, %v", obtenido, err)
			}
			verificadaEn, err := prueba.VerificadaEn()
			if err != nil || !verificadaEn.Equal(instanteConfianzaDocumentalPrueba) {
				t.Fatalf("el instante no procede del reloj local: %v, %v", verificadaEn, err)
			}
			revision, err := prueba.RevisionConfianza()
			if err != nil || revision != configuracion.revision {
				t.Fatalf("revision no ligada: %q, %v", revision, err)
			}
			huellaConfiguracion, err := prueba.HuellaConfiguracionConfianzaSHA256()
			if err != nil || huellaConfiguracion != configuracion.huellaSHA256 {
				t.Fatalf("configuracion no ligada: %q, %v", huellaConfiguracion, err)
			}
		})
	}
}

func TestServicioRechazaSobreCOSEConBSTRExteriorNoMinimoAunqueLaFirmaVerifica(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:cbor:bstr:no-minimo"),
	)
	payload := []byte("payload-no-minimo")
	solicitud := nuevaSolicitudCOSEPrueba(
		t, payload, AudienciaCOSEReciboComponenteDocumental,
	)
	sobreCanonico := firmarSobreCOSEPrueba(
		t, material, solicitud.payloadEsperado, solicitud, nil, nil,
	)
	contenido, err := sobreCanonico.COSESign1()
	if err != nil {
		t.Fatal(err)
	}
	patron := append([]byte{0x40 | byte(len(payload))}, payload...)
	if bytes.Count(contenido, patron) != 1 {
		t.Fatal("el vector no localizo de forma univoca el bstr del payload")
	}
	posicion := bytes.Index(contenido, patron)
	contenidoNoMinimo := make([]byte, 0, len(contenido)+1)
	contenidoNoMinimo = append(contenidoNoMinimo, contenido[:posicion]...)
	contenidoNoMinimo = append(contenidoNoMinimo, 0x58, byte(len(payload)))
	contenidoNoMinimo = append(contenidoNoMinimo, contenido[posicion+1:]...)
	sobreNoMinimo, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(contenidoNoMinimo)
	if err != nil {
		t.Fatalf("crear sobre semanticamente equivalente: %v", err)
	}
	verificarSobreDirectamenteConGoCOSEPrueba(t, material, solicitud, sobreNoMinimo)

	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material,
		EstadoConfianzaClaveDocumentalActiva, AudienciaCOSEReciboComponenteDocumental,
	)
	if _, err := servicio.VerificarCOSESign1(
		context.Background(), solicitud, sobreNoMinimo,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("CBOR exterior no determinista aceptado: %v", err)
	}
}

func TestServicioRechazaMapaProtegidoNoCanonicoAunqueFueFirmadoYVerifica(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("kid-nc"),
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-protected-no-canonico"), AudienciaCOSEReciboComponenteDocumental,
	)
	sobreNoCanonico := firmarSobreCOSEPrueba(
		t, material, solicitud.payloadEsperado, solicitud,
		func(mensaje *cose.Sign1Message) {
			// Mapa a2 con 4:kid antes de 1:alg. Es CBOR valido, pero el
			// orden determinista exige primero la clave entera 1.
			mapa := []byte{0xa2, 0x04, 0x40 | byte(len(material.claveID))}
			mapa = append(mapa, material.claveID...)
			mapa = append(mapa, 0x01, 0x27) // algoritmo EdDSA = -8
			mensaje.Headers.RawProtected = append(
				[]byte{0x40 | byte(len(mapa))}, mapa...,
			)
		}, nil,
	)
	verificarSobreDirectamenteConGoCOSEPrueba(t, material, solicitud, sobreNoCanonico)

	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material,
		EstadoConfianzaClaveDocumentalActiva, AudienciaCOSEReciboComponenteDocumental,
	)
	if _, err := servicio.VerificarCOSESign1(
		context.Background(), solicitud, sobreNoCanonico,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("mapa protegido no determinista aceptado: %v", err)
	}
}

func TestServicioRechazaFirmaES256HighSAunqueGoCOSELaVerifica(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalES256, []byte("clave:es256:high-s"),
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-es256-high-s"), AudienciaCOSEReciboComponenteDocumental,
	)
	sobreBajo := firmarSobreCOSEPrueba(
		t, material, solicitud.payloadEsperado, solicitud, nil, nil,
	)
	sobreAlto := transformarSobreCOSEPrueba(
		t, sobreBajo, func(mensaje *cose.Sign1Message) {
			if len(mensaje.Signature) != 64 {
				t.Fatalf("firma ES256 de longitud inesperada: %d", len(mensaje.Signature))
			}
			orden := elliptic.P256().Params().N
			s := cose.OS2IP(mensaje.Signature[32:])
			mitad := cose.OS2IP(orden.Bytes())
			mitad.Rsh(mitad, 1)
			if s.Sign() <= 0 || s.Cmp(mitad) > 0 {
				t.Fatal("el helper positivo no entrego una firma low-S")
			}
			s.Sub(orden, s).FillBytes(mensaje.Signature[32:])
		},
	)
	verificarSobreDirectamenteConGoCOSEPrueba(t, material, solicitud, sobreAlto)

	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material,
		EstadoConfianzaClaveDocumentalActiva, AudienciaCOSEReciboComponenteDocumental,
	)
	if _, err := servicio.VerificarCOSESign1(
		context.Background(), solicitud, sobreAlto,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("firma ES256 high-S aceptada: %v", err)
	}
}

func TestRaizConfiguracionYServicioCortanAliasDeClaveYCatalogo(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:copias:1"),
	)
	raiz, err := nuevaRaizPublicaFijada(
		material.claveID, material.algoritmoDocumental, material.publica,
		AudienciaCOSEReciboComponenteDocumental, EstadoConfianzaClaveDocumentalActiva,
		instanteConfianzaDocumentalPrueba.Add(-time.Hour),
		instanteConfianzaDocumentalPrueba.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatalf("crear raiz: %v", err)
	}
	// La clave entregada al constructor deja de pertenecer a la configuracion.
	material.publica.(ed25519.PublicKey)[0] ^= 0xff
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:copias:1", instanteConfianzaDocumentalPrueba.Add(-time.Hour),
		instanteConfianzaDocumentalPrueba.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion tras mutar entrada: %v", err)
	}
	// La configuracion conserva su propia copia de la raiz.
	raiz.clavePublica.(ed25519.PublicKey)[1] ^= 0xff
	servicio, err := nuevoServicioConReloj(
		configuracion, &relojConfianzaDocumentalFijo{ahora: instanteConfianzaDocumentalPrueba},
	)
	if err != nil {
		t.Fatalf("crear servicio tras mutar raiz original: %v", err)
	}
	// El servicio conserva una instantanea independiente del catalogo.
	configuracion.raices[0].clavePublica.(ed25519.PublicKey)[2] ^= 0xff
	configuracion.revision = "manipulada"

	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-copias"), AudienciaCOSEReciboComponenteDocumental,
	)
	sobre := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)
	if _, err := servicio.VerificarCOSESign1(
		context.Background(), solicitud, sobre,
	); err != nil {
		t.Fatalf("un alias externo altero el servicio: %v", err)
	}
}

func TestSolicitudAplicaLimiteCerradoPorAudienciaYPDPVerificaEnLaFrontera(t *testing.T) {
	if _, err := NuevaSolicitudVerificacionCOSESign1(
		bytes.Repeat([]byte{1}, maximoBytesPayloadDocumentalV4),
		AudienciaCOSEReciboComponenteDocumental,
	); err != nil {
		t.Fatalf("frontera ordinaria rechazada: %v", err)
	}
	if _, err := NuevaSolicitudVerificacionCOSESign1(
		bytes.Repeat([]byte{1}, maximoBytesPayloadDocumentalV4+1),
		AudienciaCOSEReciboComponenteDocumental,
	); !errors.Is(err, ErrSolicitudVerificacionCOSESign1Invalida) {
		t.Fatalf("exceso ordinario aceptado: %v", err)
	}
	if _, err := NuevaSolicitudVerificacionCOSESign1(
		[]byte("payload"), AudienciaCOSEDocumental("audiencia_inventada"),
	); !errors.Is(err, ErrSolicitudVerificacionCOSESign1Invalida) {
		t.Fatalf("audiencia libre aceptada: %v", err)
	}

	payloadPDP := bytes.Repeat([]byte{0x5a}, domain.TamanoMaximoMensajeAtestacionAutorizacionV1)
	solicitudPDP := nuevaSolicitudCOSEPrueba(
		t, payloadPDP, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	if _, err := NuevaSolicitudVerificacionCOSESign1(
		bytes.Repeat([]byte{0x5a}, domain.TamanoMaximoMensajeAtestacionAutorizacionV1+1),
		AudienciaCOSEAtestacionAutorizacionPDP,
	); !errors.Is(err, ErrSolicitudVerificacionCOSESign1Invalida) {
		t.Fatalf("exceso PDP aceptado: %v", err)
	}
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:pdp:frontera"),
	)
	sobre := firmarSobreCOSEPrueba(t, material, payloadPDP, solicitudPDP, nil, nil)
	servicioPDP, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP,
	)
	materialOrdinario := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:recibo:frontera"),
	)
	solicitudOrdinaria := nuevaSolicitudCOSEPrueba(
		t, bytes.Repeat([]byte{0x41}, maximoBytesPayloadDocumentalV4),
		AudienciaCOSEReciboComponenteDocumental,
	)
	sobreOrdinario := firmarSobreCOSEPrueba(
		t, materialOrdinario, solicitudOrdinaria.payloadEsperado, solicitudOrdinaria, nil, nil,
	)
	servicioOrdinario, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, materialOrdinario,
		EstadoConfianzaClaveDocumentalActiva, AudienciaCOSEReciboComponenteDocumental,
	)
	if _, err := servicioOrdinario.VerificarCOSESign1(
		context.Background(), solicitudOrdinaria, sobreOrdinario,
	); err != nil {
		t.Fatalf("payload ordinario en frontera no verifico: %v", err)
	}
	if _, err := servicioPDP.VerificarCOSESign1(
		context.Background(), solicitudPDP, sobre,
	); err != nil {
		t.Fatalf("payload PDP en frontera no verifico: %v", err)
	}
	sobreDesproporcionado, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(
		bytes.Repeat([]byte{0x7f}, maximoBytesPayloadDocumentalV4+margenMaximoSobreCOSEDocumentalV4+1),
	)
	if err != nil {
		t.Fatalf("crear sobre global para prueba de limite por audiencia: %v", err)
	}
	if _, err := servicioOrdinario.VerificarCOSESign1(
		context.Background(), solicitudOrdinaria, sobreDesproporcionado,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("sobre desproporcionado para audiencia ordinaria aceptado: %v", err)
	}
}

func TestServicioRechazaFirmaPayloadAudienciaKidAlgoritmoYClaveManipulados(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:documental:principal"),
	)
	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEEvidenciaDocumental,
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte(`{"efecto":"generado"}`), AudienciaCOSEEvidenciaDocumental,
	)
	sobreValido := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)

	otraSolicitudPayload := nuevaSolicitudCOSEPrueba(
		t, []byte(`{"efecto":"alterado"}`), AudienciaCOSEEvidenciaDocumental,
	)
	otraSolicitudAudiencia := nuevaSolicitudCOSEPrueba(
		t, solicitud.payloadEsperado, AudienciaCOSEReconciliacionDocumental,
	)
	sobreValidoOtraAudiencia := firmarSobreCOSEPrueba(
		t, material, otraSolicitudAudiencia.payloadEsperado, otraSolicitudAudiencia, nil, nil,
	)
	sobreFirmaAlterada := transformarSobreCOSEPrueba(t, sobreValido, func(m *cose.Sign1Message) {
		m.Signature[0] ^= 0xff
	})
	sobreKidDesconocido := firmarSobreCOSEPrueba(
		t, material, solicitud.payloadEsperado, solicitud,
		func(m *cose.Sign1Message) {
			m.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:documental:desconocida")
		}, nil,
	)
	sobreAlgoritmoAlterado := transformarSobreCOSEPrueba(t, sobreValido, func(m *cose.Sign1Message) {
		m.Headers.Protected.SetAlgorithm(cose.AlgorithmES384)
	})
	otroMaterial := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, material.claveID,
	)
	sobreOtraClave := firmarSobreCOSEPrueba(
		t, otroMaterial, solicitud.payloadEsperado, solicitud, nil, nil,
	)

	casos := map[string]struct {
		solicitud SolicitudVerificacionCOSESign1
		sobre     ports.SobreCriptograficoDocumentalCrudoV4
	}{
		"firma":                         {solicitud, sobreFirmaAlterada},
		"payload":                       {otraSolicitudPayload, sobreValido},
		"audiencia":                     {otraSolicitudAudiencia, sobreValido},
		"clave ligada a otra audiencia": {otraSolicitudAudiencia, sobreValidoOtraAudiencia},
		"kid":                           {solicitud, sobreKidDesconocido},
		"algoritmo":                     {solicitud, sobreAlgoritmoAlterado},
		"clave":                         {solicitud, sobreOtraClave},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			prueba, err := servicio.VerificarCOSESign1(context.Background(), caso.solicitud, caso.sobre)
			if !errors.Is(err, ErrVerificacionCOSESign1Fallida) || prueba.Validar() == nil {
				t.Fatalf("manipulacion aceptada: prueba=%v, err=%v", prueba, err)
			}
		})
	}
}

func TestServicioExigeSoloAlgoritmoYKidProtegidos(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:cabeceras:1"),
	)
	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSETokenCercadoDocumental,
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-cabeceras"), AudienciaCOSETokenCercadoDocumental,
	)

	algoritmo := material.algoritmoCOSE
	casos := map[string]ports.SobreCriptograficoDocumentalCrudoV4{
		"sin algoritmo": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, nil, material.claveID, nil, nil,
		),
		"sin kid": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, &algoritmo, nil, nil, nil,
		),
		"kid no protegido": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, &algoritmo, nil,
			map[any]any{cose.HeaderLabelKeyID: append([]byte(nil), material.claveID...)}, nil,
		),
		"cabecera protegida adicional": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, &algoritmo, material.claveID, nil,
			func(m *cose.Sign1Message) {
				m.Headers.Protected[cose.HeaderLabelContentType] = "application/json"
			},
		),
		"cabecera no protegida": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, &algoritmo, material.claveID,
			map[any]any{cose.HeaderLabelContentType: "application/json"}, nil,
		),
		"crit": firmarSobreCOSEConCabecerasPrueba(
			t, material, solicitud, &algoritmo, material.claveID, nil,
			func(m *cose.Sign1Message) {
				m.Headers.Protected[cose.HeaderLabelCritical] = []any{cose.HeaderLabelAlgorithm}
			},
		),
	}
	for nombre, sobre := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := servicio.VerificarCOSESign1(
				context.Background(), solicitud, sobre,
			); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
				t.Fatalf("cabeceras no permitidas aceptadas: %v", err)
			}
		})
	}
}

func TestServicioFallaCerradoConClaveRevocadaOConfiguracionFueraDeVentana(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalES256, []byte("clave:revocacion:1"),
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-revocacion"), AudienciaCOSEEscrituraAlmacenDocumental,
	)
	sobre := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)

	servicioRevocado, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalRevocada,
		AudienciaCOSEEscrituraAlmacenDocumental,
	)
	if _, err := servicioRevocado.VerificarCOSESign1(
		context.Background(), solicitud, sobre,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("una clave revocada autorizo: %v", err)
	}

	raiz := nuevaRaizCOSEPrueba(
		t, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEEscrituraAlmacenDocumental, instanteConfianzaDocumentalPrueba,
	)
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:documental:ventana:1",
		instanteConfianzaDocumentalPrueba.Add(-time.Hour),
		instanteConfianzaDocumentalPrueba,
		raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion caducada: %v", err)
	}
	servicioCaducado, err := nuevoServicioConReloj(
		configuracion, &relojConfianzaDocumentalFijo{ahora: instanteConfianzaDocumentalPrueba},
	)
	if err != nil {
		t.Fatalf("crear servicio caducado: %v", err)
	}
	if _, err := servicioCaducado.VerificarCOSESign1(
		context.Background(), solicitud, sobre,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("una configuracion caducada autorizo: %v", err)
	}

	configuracionFutura, err := nuevaConfiguracionConfianzaFijada(
		"confianza:documental:futura:1",
		instanteConfianzaDocumentalPrueba.Add(time.Minute),
		instanteConfianzaDocumentalPrueba.Add(time.Hour),
		raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion futura: %v", err)
	}
	servicioFuturo, err := nuevoServicioConReloj(
		configuracionFutura, &relojConfianzaDocumentalFijo{ahora: instanteConfianzaDocumentalPrueba},
	)
	if err != nil {
		t.Fatalf("crear servicio futuro: %v", err)
	}
	if _, err := servicioFuturo.VerificarCOSESign1(
		context.Background(), solicitud, sobre,
	); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
		t.Fatalf("una configuracion aun no publicada autorizo: %v", err)
	}
}

func TestServicioFallaCerradoFueraDeVentanaDeRaiz(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:ventana-raiz:1"),
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-ventana-raiz"), AudienciaCOSEReconciliacionDocumental,
	)
	sobre := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)
	ventanas := []struct {
		nombre      string
		validaDesde time.Time
		validaHasta time.Time
	}{
		{
			nombre:      "aun no valida",
			validaDesde: instanteConfianzaDocumentalPrueba.Add(time.Microsecond),
			validaHasta: instanteConfianzaDocumentalPrueba.Add(time.Hour),
		},
		{
			nombre:      "limite superior exclusivo",
			validaDesde: instanteConfianzaDocumentalPrueba.Add(-time.Hour),
			validaHasta: instanteConfianzaDocumentalPrueba,
		},
	}
	for _, ventana := range ventanas {
		t.Run(ventana.nombre, func(t *testing.T) {
			raiz, err := nuevaRaizPublicaFijada(
				material.claveID, material.algoritmoDocumental, material.publica,
				AudienciaCOSEReconciliacionDocumental,
				EstadoConfianzaClaveDocumentalActiva,
				ventana.validaDesde, ventana.validaHasta, time.Time{},
			)
			if err != nil {
				t.Fatalf("crear raiz: %v", err)
			}
			configuracion, err := nuevaConfiguracionConfianzaFijada(
				"confianza:ventana-raiz:1",
				instanteConfianzaDocumentalPrueba.Add(-time.Hour),
				instanteConfianzaDocumentalPrueba.Add(time.Hour), raiz,
			)
			if err != nil {
				t.Fatalf("crear configuracion: %v", err)
			}
			servicio, err := nuevoServicioConReloj(
				configuracion,
				&relojConfianzaDocumentalFijo{ahora: instanteConfianzaDocumentalPrueba},
			)
			if err != nil {
				t.Fatalf("crear servicio: %v", err)
			}
			if _, err := servicio.VerificarCOSESign1(
				context.Background(), solicitud, sobre,
			); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
				t.Fatalf("ventana de raiz invalida autorizo: %v", err)
			}
		})
	}
}

func TestServicioRechazaRelojNoCanonicoYNoPermiteBackdating(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:reloj:1"),
	)
	raiz := nuevaRaizCOSEPrueba(
		t, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEAtestacionAutorizacionPDP, instanteConfianzaDocumentalPrueba,
	)
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:reloj:1", instanteConfianzaDocumentalPrueba.Add(-time.Hour),
		instanteConfianzaDocumentalPrueba.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion: %v", err)
	}
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-reloj"), AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobre := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)

	instantesInvalidos := []time.Time{
		instanteConfianzaDocumentalPrueba.Add(time.Nanosecond),
		instanteConfianzaDocumentalPrueba.In(time.FixedZone("UTC-equivalente", 0)),
		time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	for _, instante := range instantesInvalidos {
		servicio, err := nuevoServicioConReloj(
			configuracion, &relojConfianzaDocumentalFijo{ahora: instante},
		)
		if err != nil {
			t.Fatalf("crear servicio con reloj controlado: %v", err)
		}
		if _, err := servicio.VerificarCOSESign1(
			context.Background(), solicitud, sobre,
		); !errors.Is(err, ErrVerificacionCOSESign1Fallida) {
			t.Fatalf("reloj no canonico autorizo (%v): %v", instante, err)
		}
	}

	// SolicitudVerificacionCOSESign1 no contiene ningun campo temporal: la
	// fecha que se liga a la autoridad procede exclusivamente del reloj local.
	tipo := reflect.TypeOf(solicitud)
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).Type == reflect.TypeOf(time.Time{}) {
			t.Fatalf("la solicitud conserva una via temporal de backdating: %s", tipo.Field(indice).Name)
		}
	}
}

func TestConfiguracionRechazaAlgoritmoClaveEstadoVentanasYDuplicadosInvalidos(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:configuracion:1"),
	)
	desde := instanteConfianzaDocumentalPrueba.Add(-time.Hour)
	hasta := instanteConfianzaDocumentalPrueba.Add(time.Hour)

	if _, err := nuevaRaizPublicaFijada(
		material.claveID, AlgoritmoCOSEDocumentalES256, material.publica,
		AudienciaCOSEReciboComponenteDocumental,
		EstadoConfianzaClaveDocumentalActiva, desde, hasta, time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("algoritmo/clave incompatible aceptado: %v", err)
	}
	if _, err := nuevaRaizPublicaFijada(
		material.claveID, material.algoritmoDocumental, material.publica,
		AudienciaCOSEReciboComponenteDocumental,
		EstadoConfianzaClaveDocumentalRevocada, desde, hasta, time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("revocacion sin instante aceptada: %v", err)
	}
	if _, err := nuevaRaizPublicaFijada(
		material.claveID, material.algoritmoDocumental, material.publica,
		AudienciaCOSEReciboComponenteDocumental,
		EstadoConfianzaClaveDocumentalActiva, desde.Add(time.Nanosecond), hasta, time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("ventana submicrosegundo aceptada: %v", err)
	}
	raiz := nuevaRaizCOSEPrueba(
		t, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEReciboComponenteDocumental, instanteConfianzaDocumentalPrueba,
	)
	raizMismaClaveOtraAudiencia, err := nuevaRaizPublicaFijada(
		[]byte("clave:configuracion:alias"), material.algoritmoDocumental, material.publica,
		AudienciaCOSEEvidenciaDocumental, EstadoConfianzaClaveDocumentalActiva,
		desde, hasta, time.Time{},
	)
	if err != nil {
		t.Fatalf("crear alias de clave para prueba: %v", err)
	}
	if _, err := nuevaConfiguracionConfianzaFijada(
		"confianza:material-duplicado:1", desde, hasta, raiz, raizMismaClaveOtraAudiencia,
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("misma clave bajo otro kid/audiencia aceptada: %v", err)
	}
	if _, err := nuevaConfiguracionConfianzaFijada(
		"confianza:duplicada:1", desde, hasta, raiz, raiz,
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("kid duplicado aceptado: %v", err)
	}
	if _, err := nuevaConfiguracionConfianzaFijada(
		"confianza:demasiado-larga:1", desde,
		desde.Add(maximaVigenciaConfiguracionConfianzaV4+time.Microsecond), raiz,
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("configuracion sin frescura acotada aceptada: %v", err)
	}
	if _, err := nuevaConfiguracionConfianzaFijada(
		"confianza:fuera-rfc3339:1", time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(10_000, 1, 1, 1, 0, 0, 0, time.UTC), raiz,
	); !errors.Is(err, ErrConfiguracionConfianzaDocumentalInvalida) {
		t.Fatalf("ano fuera de RFC3339 aceptado: %v", err)
	}
}

func TestAutoridadCOSEEsOpacaCopiaKidYSeRedacta(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:redaccion:secreta"),
	)
	servicio, _ := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEReciboComponenteDocumental,
	)
	solicitud := nuevaSolicitudCOSEPrueba(
		t, []byte("payload-redaccion-secreto"), AudienciaCOSEReciboComponenteDocumental,
	)
	sobre := firmarSobreCOSEPrueba(t, material, solicitud.payloadEsperado, solicitud, nil, nil)
	prueba, err := servicio.VerificarCOSESign1(context.Background(), solicitud, sobre)
	if err != nil {
		t.Fatalf("verificar: %v", err)
	}

	claveID, err := prueba.ClaveID()
	if err != nil {
		t.Fatalf("obtener kid: %v", err)
	}
	claveID[0] ^= 0xff
	segunda, err := prueba.ClaveID()
	if err != nil || !bytes.Equal(segunda, material.claveID) {
		t.Fatalf("el accessor expuso un alias mutable: %x, %v", segunda, err)
	}
	texto := fmt.Sprintf("%v|%+v|%#v|%s", prueba, prueba, prueba, prueba)
	for _, secreto := range []string{"clave:redaccion:secreta", "payload-redaccion-secreto"} {
		if strings.Contains(texto, secreto) {
			t.Fatalf("la autoridad filtro %q: %s", secreto, texto)
		}
	}
	if _, err := json.Marshal(prueba); !errors.Is(err, ErrSerializacionAutoridadCOSESign1Prohibida) {
		t.Fatalf("la autoridad se serializo por JSON: %v", err)
	}
	if _, err := prueba.MarshalText(); !errors.Is(err, ErrSerializacionAutoridadCOSESign1Prohibida) {
		t.Fatalf("la autoridad se serializo como texto: %v", err)
	}
	if _, err := prueba.MarshalBinary(); !errors.Is(err, ErrSerializacionAutoridadCOSESign1Prohibida) {
		t.Fatalf("la autoridad se serializo como binario: %v", err)
	}
}

func TestServicioConfiguracionYRaizSeRedactanEnFmtSlogYSerializadores(t *testing.T) {
	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:no-volcar:1"),
	)
	raiz := nuevaRaizCOSEPrueba(
		t, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEReciboComponenteDocumental, instanteConfianzaDocumentalPrueba,
	)
	servicio, configuracion := nuevoServicioCOSEPrueba(
		t, instanteConfianzaDocumentalPrueba, material, EstadoConfianzaClaveDocumentalActiva,
		AudienciaCOSEReciboComponenteDocumental,
	)
	valores := map[string]any{"raiz": raiz, "configuracion": configuracion, "servicio": servicio}
	for nombre, valor := range valores {
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, "clave:no-volcar:1") ||
			strings.Contains(texto, "confianza:documental:revision:7") {
			t.Fatalf("%s filtro configuracion mediante fmt: %s", nombre, texto)
		}
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionAutoridadCOSESign1Prohibida) {
			t.Fatalf("%s se serializo por JSON: %v", nombre, err)
		}
	}
	var salida bytes.Buffer
	registrador := slog.New(slog.NewTextHandler(&salida, nil))
	registrador.Info("confianza", "raiz", raiz, "configuracion", configuracion, "servicio", servicio)
	if strings.Contains(salida.String(), "clave:no-volcar:1") ||
		strings.Contains(salida.String(), "confianza:documental:revision:7") {
		t.Fatalf("slog filtro el mapa de confianza: %s", salida.String())
	}
}

func TestValorCeroNuncaConservaAutoridadCOSE(t *testing.T) {
	var prueba PruebaCOSESign1DocumentalVerificada
	if !errors.Is(prueba.Validar(), ErrPruebaCOSESign1VerificadaInvalida) {
		t.Fatal("el valor cero conservo autoridad")
	}
	if _, err := prueba.ClaveID(); !errors.Is(err, ErrPruebaCOSESign1VerificadaInvalida) {
		t.Fatalf("el valor cero revelo datos: %v", err)
	}
}

func generarMaterialFirmaCOSEPrueba(
	t *testing.T,
	algoritmo AlgoritmoCOSEDocumental,
	claveID []byte,
) materialFirmaCOSEPrueba {
	t.Helper()
	material := materialFirmaCOSEPrueba{
		algoritmoDocumental: algoritmo,
		claveID:             append([]byte(nil), claveID...),
	}
	switch algoritmo {
	case AlgoritmoCOSEDocumentalEdDSA:
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generar Ed25519: %v", err)
		}
		material.algoritmoCOSE = cose.AlgorithmEdDSA
		material.privada, material.publica = privada, publica
	case AlgoritmoCOSEDocumentalES256:
		privada, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generar P-256: %v", err)
		}
		material.algoritmoCOSE = cose.AlgorithmES256
		material.privada, material.publica = privada, &privada.PublicKey
	default:
		t.Fatalf("algoritmo de prueba no soportado: %s", algoritmo)
	}
	return material
}

func nuevaSolicitudCOSEPrueba(
	t *testing.T,
	payload []byte,
	audiencia AudienciaCOSEDocumental,
) SolicitudVerificacionCOSESign1 {
	t.Helper()
	solicitud, err := NuevaSolicitudVerificacionCOSESign1(payload, audiencia)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	return solicitud
}

func nuevaRaizCOSEPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	estado EstadoConfianzaClaveDocumental,
	audiencia AudienciaCOSEDocumental,
	ahora time.Time,
) RaizPublicaFijada {
	t.Helper()
	revocadaEn := time.Time{}
	if estado == EstadoConfianzaClaveDocumentalRevocada {
		revocadaEn = ahora.Add(-90 * time.Minute)
	}
	var raiz RaizPublicaFijada
	var err error
	if audiencia == AudienciaCOSEAtestacionAutorizacionPDP {
		raiz, err = nuevaRaizPublicaFijadaAtestacionPDP(
			material.claveID, material.algoritmoDocumental, material.publica,
			suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
			audienciaDespliegueAtestacionPDPPrueba, estado,
			ahora.Add(-2*time.Hour), ahora.Add(2*time.Hour), revocadaEn,
		)
	} else {
		raiz, err = nuevaRaizPublicaFijada(
			material.claveID, material.algoritmoDocumental, material.publica, audiencia, estado,
			ahora.Add(-2*time.Hour), ahora.Add(2*time.Hour), revocadaEn,
		)
	}
	if err != nil {
		t.Fatalf("crear raiz: %v", err)
	}
	return raiz
}

func nuevoServicioCOSEPrueba(
	t *testing.T,
	ahora time.Time,
	material materialFirmaCOSEPrueba,
	estado EstadoConfianzaClaveDocumental,
	audiencia AudienciaCOSEDocumental,
) (*Servicio, ConfiguracionConfianzaFijada) {
	t.Helper()
	raiz := nuevaRaizCOSEPrueba(t, material, estado, audiencia, ahora)
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:documental:revision:7", ahora.Add(-time.Hour), ahora.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion: %v", err)
	}
	servicio, err := nuevoServicioConReloj(
		configuracion, &relojConfianzaDocumentalFijo{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio, configuracion
}

func firmarSobreCOSEPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	payload []byte,
	solicitud SolicitudVerificacionCOSESign1,
	antesDeFirmar func(*cose.Sign1Message),
	despuesDeFirmar func(*cose.Sign1Message),
) ports.SobreCriptograficoDocumentalCrudoV4 {
	t.Helper()
	algoritmo := material.algoritmoCOSE
	return firmarSobreCOSEConCabecerasYPayloadPrueba(
		t, material, payload, solicitud, &algoritmo, material.claveID, nil,
		antesDeFirmar, despuesDeFirmar,
	)
}

func firmarSobreCOSEConCabecerasPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	solicitud SolicitudVerificacionCOSESign1,
	algoritmo *cose.Algorithm,
	kidProtegido []byte,
	noProtegidas map[any]any,
	antesDeFirmar func(*cose.Sign1Message),
) ports.SobreCriptograficoDocumentalCrudoV4 {
	t.Helper()
	return firmarSobreCOSEConCabecerasYPayloadPrueba(
		t, material, solicitud.payloadEsperado, solicitud, algoritmo, kidProtegido,
		noProtegidas, antesDeFirmar, nil,
	)
}

func firmarSobreCOSEConCabecerasYPayloadPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	payload []byte,
	solicitud SolicitudVerificacionCOSESign1,
	algoritmo *cose.Algorithm,
	kidProtegido []byte,
	noProtegidas map[any]any,
	antesDeFirmar func(*cose.Sign1Message),
	despuesDeFirmar func(*cose.Sign1Message),
) ports.SobreCriptograficoDocumentalCrudoV4 {
	t.Helper()
	mensaje := cose.NewSign1Message()
	mensaje.Payload = append([]byte(nil), payload...)
	if algoritmo != nil {
		mensaje.Headers.Protected.SetAlgorithm(*algoritmo)
	}
	if kidProtegido != nil {
		mensaje.Headers.Protected[cose.HeaderLabelKeyID] = append([]byte(nil), kidProtegido...)
	}
	for etiqueta, valor := range noProtegidas {
		mensaje.Headers.Unprotected[etiqueta] = valor
	}
	if antesDeFirmar != nil {
		antesDeFirmar(mensaje)
	}
	firmante, err := cose.NewSigner(material.algoritmoCOSE, material.privada)
	if err != nil {
		t.Fatalf("crear firmante COSE: %v", err)
	}
	aad, err := solicitud.AADExterno()
	if err != nil {
		t.Fatalf("crear AAD: %v", err)
	}
	if err := mensaje.Sign(rand.Reader, aad, firmante); err != nil {
		t.Fatalf("firmar COSE: %v", err)
	}
	normalizarFirmaES256BajaPrueba(t, material, mensaje)
	if despuesDeFirmar != nil {
		despuesDeFirmar(mensaje)
	}
	contenido, err := mensaje.MarshalCBOR()
	if err != nil {
		t.Fatalf("codificar COSE: %v", err)
	}
	sobre, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(contenido)
	if err != nil {
		t.Fatalf("crear sobre crudo: %v", err)
	}
	return sobre
}

func normalizarFirmaES256BajaPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	mensaje *cose.Sign1Message,
) {
	t.Helper()
	if material.algoritmoDocumental != AlgoritmoCOSEDocumentalES256 {
		return
	}
	if len(mensaje.Signature) != 64 {
		t.Fatalf("firma ES256 de longitud inesperada: %d", len(mensaje.Signature))
	}
	orden := elliptic.P256().Params().N
	s := cose.OS2IP(mensaje.Signature[32:])
	mitad := cose.OS2IP(orden.Bytes())
	mitad.Rsh(mitad, 1)
	if s.Sign() <= 0 || s.Cmp(orden) >= 0 {
		t.Fatal("firma ES256 fuera del orden de P-256")
	}
	if s.Cmp(mitad) > 0 {
		s.Sub(orden, s).FillBytes(mensaje.Signature[32:])
	}
}

func verificarSobreDirectamenteConGoCOSEPrueba(
	t *testing.T,
	material materialFirmaCOSEPrueba,
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) {
	t.Helper()
	contenido, err := sobre.COSESign1()
	if err != nil {
		t.Fatal(err)
	}
	var mensaje cose.Sign1Message
	if err := mensaje.UnmarshalCBOR(contenido); err != nil {
		t.Fatalf("go-cose no interpreto el vector adversarial: %v", err)
	}
	verificador, err := cose.NewVerifier(material.algoritmoCOSE, material.publica)
	if err != nil {
		t.Fatalf("crear verificador directo: %v", err)
	}
	aad, err := solicitud.AADExterno()
	if err != nil {
		t.Fatal(err)
	}
	if err := mensaje.Verify(aad, verificador); err != nil {
		t.Fatalf("el vector adversarial no conserva una firma valida para go-cose: %v", err)
	}
}

func transformarSobreCOSEPrueba(
	t *testing.T,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
	transformar func(*cose.Sign1Message),
) ports.SobreCriptograficoDocumentalCrudoV4 {
	t.Helper()
	contenido, err := sobre.COSESign1()
	if err != nil {
		t.Fatalf("obtener sobre: %v", err)
	}
	var mensaje cose.Sign1Message
	if err := mensaje.UnmarshalCBOR(contenido); err != nil {
		t.Fatalf("interpretar COSE: %v", err)
	}
	transformar(&mensaje)
	// go-cose conserva las cabeceras CBOR originales al decodificar. Para que
	// la manipulacion de prueba se recodifique, se descartan esas caches.
	mensaje.Headers.RawProtected = nil
	mensaje.Headers.RawUnprotected = nil
	contenido, err = mensaje.MarshalCBOR()
	if err != nil {
		t.Fatalf("recodificar COSE: %v", err)
	}
	transformado, err := ports.NuevoSobreCriptograficoDocumentalCrudoV4(contenido)
	if err != nil {
		t.Fatalf("crear sobre transformado: %v", err)
	}
	return transformado
}
