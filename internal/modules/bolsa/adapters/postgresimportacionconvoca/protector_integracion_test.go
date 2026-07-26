package postgresimportacionconvoca

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// protectorAEADIntegracion solo cifra fixtures sintéticos de la prueba. Simula
// el contrato KMS/HSM sin introducir una clave o un protector de producción.
type protectorAEADIntegracion struct {
	claveCifrado    [32]byte
	claveCiega      [32]byte
	claveAtestacion [32]byte
}

func nuevoProtectorAEADIntegracion() *protectorAEADIntegracion {
	return &protectorAEADIntegracion{
		claveCifrado: sha256.Sum256([]byte("fixture-sintetico-cifrado-convoca-b1")),
		claveCiega:   sha256.Sum256([]byte("fixture-sintetico-derivacion-convoca-b1")),
		claveAtestacion: sha256.Sum256(
			[]byte("fixture-sintetico-atestacion-fila-convoca-b1"),
		),
	}
}

func (p *protectorAEADIntegracion) ProtegerStaging(
	ctx context.Context,
	solicitud SolicitudProteccionStaging,
) (ResultadoProteccionStaging, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoProteccionStaging{}, err
	}
	aead, err := p.aead()
	if err != nil {
		return ResultadoProteccionStaging{}, err
	}
	resultado := ResultadoProteccionStaging{
		Filas: make([]FilaStagingProtegida, len(solicitud.Filas)),
	}
	for i := range solicitud.Filas {
		if err := ctx.Err(); err != nil {
			return ResultadoProteccionStaging{}, err
		}
		claro, err := json.Marshal(solicitud.Filas[i])
		if err != nil {
			return ResultadoProteccionStaging{}, err
		}
		nonce := make([]byte, aead.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			return ResultadoProteccionStaging{}, err
		}
		aad := aadIntegracion(solicitud.ImportacionRef, solicitud.HuellaFicheroSHA256,
			solicitud.Esquema, solicitud.Filas[i].Numero)
		mac := hmac.New(sha256.New, p.claveCiega[:])
		_, _ = mac.Write([]byte(solicitud.Filas[i].Identidad.Documento))
		atestacion := p.atestarFila(solicitud, solicitud.Filas[i].Numero, claro)
		resultado.Filas[i] = FilaStagingProtegida{
			Numero:                        solicitud.Filas[i].Numero,
			EsquemaProteccion:             EsquemaProteccionStagingV1,
			ClaveRef:                      "kms:fixture:convoca:b1:v1",
			ClaveDerivacionRef:            "kms:fixture:convoca:derivacion-documento:v1",
			Nonce:                         nonce,
			ClaveAtestacionRef:            "kms:fixture:convoca:atestacion-fila:v1",
			ContenidoCifrado:              aead.Seal(nil, nonce, claro, aad),
			DerivacionDocumentoHMACSHA256: mac.Sum(nil),
			AtestacionFilaHMACSHA256:      atestacion,
		}
		borrarBytes(claro)
	}
	return resultado, nil
}

func (p *protectorAEADIntegracion) RecuperarStaging(
	ctx context.Context,
	solicitud SolicitudRecuperacionStaging,
) ([]dominio.FilaAceptada, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	aead, err := p.aead()
	if err != nil {
		return nil, err
	}
	resultado := make([]dominio.FilaAceptada, len(solicitud.Filas))
	for i := range solicitud.Filas {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if solicitud.Filas[i].EsquemaProteccion !=
			EsquemaProteccionStagingV1 ||
			solicitud.Filas[i].ClaveRef !=
				"kms:fixture:convoca:b1:v1" ||
			solicitud.Filas[i].ClaveAtestacionRef !=
				"kms:fixture:convoca:atestacion-fila:v1" ||
			solicitud.Filas[i].ClaveDerivacionRef !=
				"kms:fixture:convoca:derivacion-documento:v1" {
			return nil, ErrMaterialNoConfiable
		}
		aad := aadIntegracion(solicitud.ImportacionRef, solicitud.HuellaFicheroSHA256,
			solicitud.Esquema, solicitud.Filas[i].Numero)
		claro, err := aead.Open(nil, solicitud.Filas[i].Nonce,
			solicitud.Filas[i].ContenidoCifrado, aad)
		if err != nil {
			return nil, ErrMaterialNoConfiable
		}
		if !bytesIgualesConstantes(
			solicitud.Filas[i].AtestacionFilaHMACSHA256,
			p.atestarFilaProtegida(solicitud, solicitud.Filas[i].Numero, claro),
		) {
			borrarBytes(claro)
			return nil, ErrMaterialNoConfiable
		}
		if err := json.Unmarshal(claro, &resultado[i]); err != nil {
			borrarBytes(claro)
			return nil, ErrMaterialNoConfiable
		}
		derivacion := hmac.New(sha256.New, p.claveCiega[:])
		_, _ = derivacion.Write([]byte(resultado[i].Identidad.Documento))
		if !bytesIgualesConstantes(
			solicitud.Filas[i].DerivacionDocumentoHMACSHA256,
			derivacion.Sum(nil),
		) {
			borrarBytes(claro)
			return nil, ErrMaterialNoConfiable
		}
		borrarBytes(claro)
	}
	return resultado, nil
}

func TestProtectorIntegracionRechazaMutacionesCampoACampo(t *testing.T) {
	t.Parallel()
	protector := nuevoProtectorAEADIntegracion()
	fila := dominio.FilaAceptada{
		Numero: 2, Esquema: dominio.EsquemaResumenPersona,
		Identidad: dominio.IdentidadEnmascarada{
			Documento: "***0001**", PrimerApellido: "Sintetica",
			SegundoApellido: "Prueba", Nombre: "Persona",
		},
		Turno: "Libre",
		Resumen: &dominio.ResumenPersona{
			Experiencia: "1", Formacion: "1", Total: "2",
		},
	}
	proteger := SolicitudProteccionStaging{
		ImportacionRef:      "importacion:convoca:" + huellaIntegracion("8"),
		HuellaFicheroSHA256: huellaIntegracion("8"),
		Esquema:             dominio.EsquemaResumenPersona,
		Filas:               []dominio.FilaAceptada{fila},
	}
	protegido, err := protector.ProtegerStaging(context.Background(), proteger)
	if err != nil {
		t.Fatalf("proteger fixture para mutaciones: %v", err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	recuperar := SolicitudRecuperacionStaging{
		ImportacionRef: proteger.ImportacionRef, HuellaFicheroSHA256: proteger.HuellaFicheroSHA256,
		Esquema: proteger.Esquema,
	}
	mutaciones := map[string]func(*FilaStagingProtegida){
		"numero": func(valor *FilaStagingProtegida) {
			valor.Numero++
		},
		"esquema": func(valor *FilaStagingProtegida) {
			valor.EsquemaProteccion += ".alterado"
		},
		"clave_ref": func(valor *FilaStagingProtegida) {
			valor.ClaveRef = "kms:fixture:convoca:b1:v2"
		},
		"clave_derivacion_ref": func(valor *FilaStagingProtegida) {
			valor.ClaveDerivacionRef = "kms:fixture:convoca:derivacion-documento:v2"
		},
		"clave_atestacion_ref": func(valor *FilaStagingProtegida) {
			valor.ClaveAtestacionRef = "kms:fixture:convoca:atestacion-fila:v2"
		},
		"nonce": func(valor *FilaStagingProtegida) {
			valor.Nonce[0] ^= 0xff
		},
		"cifrado": func(valor *FilaStagingProtegida) {
			valor.ContenidoCifrado[0] ^= 0xff
		},
		"derivacion": func(valor *FilaStagingProtegida) {
			valor.DerivacionDocumentoHMACSHA256[0] ^= 0xff
		},
		"atestacion": func(valor *FilaStagingProtegida) {
			valor.AtestacionFilaHMACSHA256[0] ^= 0xff
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			filas := clonarFilasProtegidas(protegido.Filas)
			defer borrarFilasProtegidas(filas)
			mutar(&filas[0])
			recuperar.Filas = filas
			if _, err := protector.RecuperarStaging(
				context.Background(), recuperar,
			); err == nil {
				t.Fatalf("protector recupero material mutado en %s", nombre)
			}
		})
	}
}

type contextoCanceladoEnIteracion struct {
	context.Context
	consultas int
	limite    int
}

func (c *contextoCanceladoEnIteracion) Err() error {
	c.consultas++
	if c.consultas >= c.limite {
		return context.Canceled
	}
	return nil
}

func TestProtectorIntegracionAtiendeCancelacionDuranteRecuperacion(t *testing.T) {
	t.Parallel()
	protector := nuevoProtectorAEADIntegracion()
	fila := dominio.FilaAceptada{
		Numero: 2, Esquema: dominio.EsquemaResumenPersona,
		Identidad: dominio.IdentidadEnmascarada{
			Documento: "***0001**", PrimerApellido: "Sintetica",
			SegundoApellido: "Prueba", Nombre: "Persona",
		},
		Turno: "Libre",
		Resumen: &dominio.ResumenPersona{
			Experiencia: "1", Formacion: "1", Total: "2",
		},
	}
	solicitud := SolicitudProteccionStaging{
		ImportacionRef:      "importacion:convoca:" + huellaIntegracion("6"),
		HuellaFicheroSHA256: huellaIntegracion("6"),
		Esquema:             dominio.EsquemaResumenPersona,
		Filas:               []dominio.FilaAceptada{fila},
	}
	protegido, err := protector.ProtegerStaging(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("proteger fixture para cancelacion: %v", err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	filas := []FilaStagingProtegida{protegido.Filas[0], protegido.Filas[0]}
	ctx := &contextoCanceladoEnIteracion{
		Context: context.Background(), limite: 3,
	}
	_, err = protector.RecuperarStaging(ctx, SolicitudRecuperacionStaging{
		ImportacionRef: solicitud.ImportacionRef, HuellaFicheroSHA256: solicitud.HuellaFicheroSHA256,
		Esquema: solicitud.Esquema, Filas: filas,
	})
	if err != context.Canceled {
		t.Fatalf("recuperacion no atendio cancelacion entre filas: %v", err)
	}
}

func (p *protectorAEADIntegracion) atestarFila(
	solicitud SolicitudProteccionStaging,
	numero int,
	canon []byte,
) []byte {
	return p.atestar(
		solicitud.ImportacionRef, solicitud.HuellaFicheroSHA256,
		solicitud.Esquema, numero, canon,
	)
}

func (p *protectorAEADIntegracion) atestarFilaProtegida(
	solicitud SolicitudRecuperacionStaging,
	numero int,
	canon []byte,
) []byte {
	return p.atestar(
		solicitud.ImportacionRef, solicitud.HuellaFicheroSHA256,
		solicitud.Esquema, numero, canon,
	)
}

func (p *protectorAEADIntegracion) atestar(
	importacionRef string,
	huella string,
	esquema dominio.EsquemaExportacion,
	numero int,
	canon []byte,
) []byte {
	mac := hmac.New(sha256.New, p.claveAtestacion[:])
	_, _ = mac.Write(aadIntegracion(
		importacionRef, huella, esquema, numero,
	))
	_, _ = mac.Write([]byte{0x1f})
	_, _ = mac.Write(canon)
	return mac.Sum(nil)
}

func (p *protectorAEADIntegracion) aead() (cipher.AEAD, error) {
	bloque, err := aes.NewCipher(p.claveCifrado[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(bloque)
}

func aadIntegracion(
	importacionRef string,
	huella string,
	esquema dominio.EsquemaExportacion,
	numero int,
) []byte {
	return []byte(fmt.Sprintf("%s\x1f%s\x1f%s\x1f%d",
		importacionRef, huella, esquema, numero))
}
