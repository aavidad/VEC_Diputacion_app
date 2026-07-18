package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	josecipher "github.com/go-jose/go-jose/v4/cipher"

	"vec-diputacion-granada/config"
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

var ErrKMSDesarrolloNoDisponible = errors.New("bootstrap: KMS de desarrollo no disponible")

const (
	algoritmoContenidoDesarrollo         = "A256GCM"
	algoritmoEnvolturaDesarrollo         = "A256KW"
	algoritmoFirmaKMSDesarrollo          = "Ed25519"
	claveMaestraDesarrolloRef            = "clave:kms:desarrollo:emisor:v1"
	verificadorAtestacionDesarrolloRef   = "verificador:kms:desarrollo:atestacion:ed25519:v1"
	verificadorRevalidacionDesarrolloRef = "verificador:kms:desarrollo:revalidacion:ed25519:v1"
	proveedorEmisorKMSDesarrolloRef      = "proveedor:kms:desarrollo:emisor:v1"
	proveedorRevalidadorKMSDesarrolloRef = "proveedor:kms:desarrollo:revalidador:v1"
	proveedorVerificadorKMSDesarrolloRef = "proveedor:kms:desarrollo:verificador-recibo:v1"
)

// emisorKMSDesarrollo es la unica responsabilidad con acceso simultaneo a la
// clave de envoltura y a la privada de atestacion. No revalida ni verifica
// recibos confirmados.
type emisorKMSDesarrollo struct {
	claveEnvoltura          [sha256.Size]byte
	firmaAtestacion         ed25519.PrivateKey
	huellaPublicaAtestacion [sha256.Size]byte
	aleatorio               io.Reader
}

// revalidadorKMSDesarrollo solo conoce la publica del emisor y su segunda
// privada independiente. Verifica A y emite la evidencia B dentro del COMMIT.
type revalidadorKMSDesarrollo struct {
	verificadorAtestacion     ed25519.PublicKey
	huellaPublicaAtestacion   [sha256.Size]byte
	firmaRevalidacion         ed25519.PrivateKey
	huellaPublicaRevalidacion [sha256.Size]byte
}

// verificadorFirmasKMSDesarrollo nunca firma: conserva unicamente las dos
// publicas fijadas. La persistencia T20 lo usa tras releer el recibo.
type verificadorFirmasKMSDesarrollo struct {
	verificadorAtestacion     ed25519.PublicKey
	huellaPublicaAtestacion   [sha256.Size]byte
	verificadorRevalidacion   ed25519.PublicKey
	huellaPublicaRevalidacion [sha256.Size]byte
}

func nuevosProveedoresKMSDesarrollo(
	claveMaestra [sha256.Size]byte,
	firmaAtestacion ed25519.PrivateKey,
	verificadorAtestacion ed25519.PublicKey,
	huellaPublicaAtestacion [sha256.Size]byte,
	firmaRevalidacion ed25519.PrivateKey,
	verificadorRevalidacion ed25519.PublicKey,
	huellaPublicaRevalidacion [sha256.Size]byte,
) (*emisorKMSDesarrollo, *revalidadorKMSDesarrollo, *verificadorFirmasKMSDesarrollo, error) {
	if !parejaEd25519Fijada(firmaAtestacion, verificadorAtestacion, huellaPublicaAtestacion) ||
		!parejaEd25519Fijada(firmaRevalidacion, verificadorRevalidacion, huellaPublicaRevalidacion) ||
		subtle.ConstantTimeCompare(verificadorAtestacion, verificadorRevalidacion) == 1 {
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	emisor := &emisorKMSDesarrollo{
		claveEnvoltura:          derivarClaveDesarrollo(claveMaestra, "vec.kms.desarrollo.envoltura.v1"),
		firmaAtestacion:         append(ed25519.PrivateKey(nil), firmaAtestacion...),
		huellaPublicaAtestacion: huellaPublicaAtestacion,
		aleatorio:               rand.Reader,
	}
	revalidador := &revalidadorKMSDesarrollo{
		verificadorAtestacion:     append(ed25519.PublicKey(nil), verificadorAtestacion...),
		huellaPublicaAtestacion:   huellaPublicaAtestacion,
		firmaRevalidacion:         append(ed25519.PrivateKey(nil), firmaRevalidacion...),
		huellaPublicaRevalidacion: huellaPublicaRevalidacion,
	}
	verificador := &verificadorFirmasKMSDesarrollo{
		verificadorAtestacion:     append(ed25519.PublicKey(nil), verificadorAtestacion...),
		huellaPublicaAtestacion:   huellaPublicaAtestacion,
		verificadorRevalidacion:   append(ed25519.PublicKey(nil), verificadorRevalidacion...),
		huellaPublicaRevalidacion: huellaPublicaRevalidacion,
	}
	return emisor, revalidador, verificador, nil
}

func parejaEd25519Fijada(
	privada ed25519.PrivateKey,
	publica ed25519.PublicKey,
	huellaEsperada [sha256.Size]byte,
) bool {
	if len(privada) != ed25519.PrivateKeySize || len(publica) != ed25519.PublicKeySize ||
		subtle.ConstantTimeCompare(privada.Public().(ed25519.PublicKey), publica) != 1 {
		return false
	}
	der, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return false
	}
	huella := sha256.Sum256(der)
	borrarBytes(der)
	return subtle.ConstantTimeCompare(huella[:], huellaEsperada[:]) == 1
}

func (e *emisorKMSDesarrollo) CifrarBorrador(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudCifradoBorrador,
) (gobiernoconvocatorias.ResultadoCifradoBorrador, error) {
	if e == nil || ctx == nil || ctx.Err() != nil || e.aleatorio == nil ||
		len(e.firmaAtestacion) != ed25519.PrivateKeySize || solicitud.Validar() != nil ||
		!procedenciaKMSDesarrolloValida(solicitud.Procedencia) ||
		solicitud.PerfilEsperado.AlgoritmoAEAD != algoritmoContenidoDesarrollo ||
		solicitud.PerfilEsperado.AlgoritmoEnvolturaClave != algoritmoEnvolturaDesarrollo {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	claro, err := solicitud.VersionCanonicaParaCifrado()
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	defer borrarBytes(claro)
	aad, err := solicitud.AADCanonica()
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	defer borrarBytes(aad)
	huellaAADBytes := sha256.Sum256(aad)
	huellaAAD := hex.EncodeToString(huellaAADBytes[:])

	nonce, textoCifrado, claveEnvuelta, err := e.cifrarContenido(claro, aad)
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, err
	}
	defer borrarBytes(nonce)
	defer borrarBytes(textoCifrado)
	defer borrarBytes(claveEnvuelta)
	if err := ctx.Err(); err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, err
	}

	envoltura, err := gobiernoconvocatorias.NuevaEnvolturaClaveKMSBorrador(
		solicitud.PerfilEsperado, claveMaestraDesarrolloRef, 1, claveEnvuelta, huellaAAD,
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	sobre, err := gobiernoconvocatorias.NuevoSobreCifradoAEADBorrador(
		solicitud.PerfilEsperado, nonce, textoCifrado, huellaAAD,
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	_, _, _, _, _, huellaEnvoltura, err := envoltura.DatosParaPersistencia()
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	_, _, _, _, huellaSobre, err := sobre.DatosParaPersistencia()
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}

	emitidaEn := solicitud.SolicitadaEn
	validaHasta := emitidaEn.Add(4 * time.Minute)
	if !validaHasta.Before(solicitud.Reserva.ArrendamientoVenceEn) {
		validaHasta = solicitud.Reserva.ArrendamientoVenceEn
	}
	if !validaHasta.After(emitidaEn) {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	atestacionRef, err := e.nuevaReferenciaAtestacion()
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, err
	}
	atestacion, err := gobiernoconvocatorias.NuevaAtestacionKMSBorrador(
		atestacionRef, 1, solicitud.PerfilEsperado, claveMaestraDesarrolloRef, 1,
		huellaAAD, huellaEnvoltura, huellaSobre, verificadorAtestacionDesarrolloRef,
		solicitud.Procedencia, algoritmoFirmaKMSDesarrollo,
		hex.EncodeToString(e.huellaPublicaAtestacion[:]), emitidaEn, validaHasta,
		func(preimagen []byte) ([]byte, error) {
			if len(preimagen) == 0 {
				return nil, ErrKMSDesarrolloNoDisponible
			}
			return ed25519.Sign(e.firmaAtestacion, preimagen), nil
		},
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	resultado, err := gobiernoconvocatorias.NuevoResultadoCifradoBorrador(
		solicitud, envoltura, sobre, atestacion, emitidaEn,
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoCifradoBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	return resultado, nil
}

func (e *emisorKMSDesarrollo) cifrarContenido(claro, aad []byte) ([]byte, []byte, []byte, error) {
	if e == nil || e.aleatorio == nil || len(claro) == 0 || len(aad) == 0 {
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	var dek [sha256.Size]byte
	if _, err := io.ReadFull(e.aleatorio, dek[:]); err != nil {
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	defer borrarBytes(dek[:])
	bloqueContenido, err := aes.NewCipher(dek[:])
	if err != nil {
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	aead, err := cipher.NewGCM(bloqueContenido)
	if err != nil {
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(e.aleatorio, nonce); err != nil {
		borrarBytes(nonce)
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	textoCifrado := aead.Seal(nil, nonce, claro, aad)
	bloqueEnvoltura, err := aes.NewCipher(e.claveEnvoltura[:])
	if err != nil {
		borrarBytes(nonce)
		borrarBytes(textoCifrado)
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	claveEnvuelta, err := josecipher.KeyWrap(bloqueEnvoltura, dek[:])
	if err != nil {
		borrarBytes(nonce)
		borrarBytes(textoCifrado)
		return nil, nil, nil, ErrKMSDesarrolloNoDisponible
	}
	return nonce, textoCifrado, claveEnvuelta, nil
}

func (e *emisorKMSDesarrollo) nuevaReferenciaAtestacion() (string, error) {
	var opaco [16]byte
	if e == nil || e.aleatorio == nil {
		return "", ErrKMSDesarrolloNoDisponible
	}
	if _, err := io.ReadFull(e.aleatorio, opaco[:]); err != nil {
		return "", ErrKMSDesarrolloNoDisponible
	}
	return "atestacion:kms:desarrollo:ed25519:v1:" + hex.EncodeToString(opaco[:]), nil
}

func (r *revalidadorKMSDesarrollo) RevalidarAtestacionKMS(
	ctx context.Context,
	solicitud gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador,
) (gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador, error) {
	if r == nil || ctx == nil || ctx.Err() != nil ||
		len(r.verificadorAtestacion) != ed25519.PublicKeySize ||
		len(r.firmaRevalidacion) != ed25519.PrivateKeySize || solicitud.Validar() != nil ||
		!procedenciaKMSDesarrolloValida(solicitud.AtestacionKMS.Procedencia) {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	atestacion := solicitud.AtestacionKMS
	if atestacion.ClaveMaestraRef != claveMaestraDesarrolloRef || atestacion.VersionClave != 1 ||
		atestacion.Perfil.AlgoritmoAEAD != algoritmoContenidoDesarrollo ||
		atestacion.Perfil.AlgoritmoEnvolturaClave != algoritmoEnvolturaDesarrollo {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	preimagen, firma, err := verificarAtestacionKMSDesarrollo(
		atestacion, r.verificadorAtestacion, r.huellaPublicaAtestacion,
	)
	if err != nil {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, err
	}
	defer borrarBytes(preimagen)
	defer borrarBytes(firma)
	huellaComprobacion := huellaComprobacionAtestacion(preimagen, firma)
	comprobacionRef := "comprobacion:kms:desarrollo:ed25519:v1:" + huellaComprobacion
	resultado, err := gobiernoconvocatorias.NuevoResultadoRevalidacionAtestacionKMSBorrador(
		solicitud, comprobacionRef, huellaComprobacion, solicitud.SolicitadaEn,
		algoritmoFirmaKMSDesarrollo, verificadorRevalidacionDesarrolloRef,
		hex.EncodeToString(r.huellaPublicaRevalidacion[:]),
		func(datos []byte) ([]byte, error) {
			if len(datos) == 0 {
				return nil, ErrKMSDesarrolloNoDisponible
			}
			return ed25519.Sign(r.firmaRevalidacion, datos), nil
		},
	)
	if err != nil || ctx.Err() != nil {
		return gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador{}, ErrKMSDesarrolloNoDisponible
	}
	return resultado, nil
}

func verificarAtestacionKMSDesarrollo(
	atestacion gobiernoconvocatorias.AtestacionKMSBorrador,
	publica ed25519.PublicKey,
	huellaPublica [sha256.Size]byte,
) ([]byte, []byte, error) {
	preimagen, algoritmo, verificadorRef, huellaDeclarada, firma, err :=
		atestacion.DatosParaVerificacionFirma()
	if err != nil || !procedenciaKMSDesarrolloValida(atestacion.Procedencia) ||
		len(publica) != ed25519.PublicKeySize ||
		!textoConstanteIgual(algoritmo, algoritmoFirmaKMSDesarrollo) ||
		!textoConstanteIgual(verificadorRef, verificadorAtestacionDesarrolloRef) ||
		!textoConstanteIgual(huellaDeclarada, hex.EncodeToString(huellaPublica[:])) ||
		!ed25519.Verify(publica, preimagen, firma) {
		borrarBytes(preimagen)
		borrarBytes(firma)
		return nil, nil, ErrKMSDesarrolloNoDisponible
	}
	return preimagen, firma, nil
}

func (v *verificadorFirmasKMSDesarrollo) VerificarAtestacion(
	atestacion gobiernoconvocatorias.AtestacionKMSBorrador,
) error {
	if v == nil {
		return ErrKMSDesarrolloNoDisponible
	}
	preimagen, firma, err := verificarAtestacionKMSDesarrollo(
		atestacion, v.verificadorAtestacion, v.huellaPublicaAtestacion,
	)
	borrarBytes(preimagen)
	borrarBytes(firma)
	return err
}

func (v *verificadorFirmasKMSDesarrollo) VerificarRevalidacion(
	solicitud gobiernoconvocatorias.SolicitudRevalidacionAtestacionKMSBorrador,
	resultado gobiernoconvocatorias.ResultadoRevalidacionAtestacionKMSBorrador,
) error {
	if v == nil || solicitud.Validar() != nil ||
		!procedenciaKMSDesarrolloValida(solicitud.AtestacionKMS.Procedencia) ||
		resultado.ValidarPara(solicitud) != nil {
		return ErrKMSDesarrolloNoDisponible
	}
	preimagen, algoritmo, verificadorRef, huellaDeclarada, firma, err :=
		resultado.DatosParaVerificacionFirma(solicitud)
	defer borrarBytes(preimagen)
	defer borrarBytes(firma)
	if err != nil || len(v.verificadorRevalidacion) != ed25519.PublicKeySize ||
		!textoConstanteIgual(algoritmo, algoritmoFirmaKMSDesarrollo) ||
		!textoConstanteIgual(verificadorRef, verificadorRevalidacionDesarrolloRef) ||
		!textoConstanteIgual(huellaDeclarada, hex.EncodeToString(v.huellaPublicaRevalidacion[:])) ||
		!ed25519.Verify(v.verificadorRevalidacion, preimagen, firma) {
		return ErrKMSDesarrolloNoDisponible
	}
	return nil
}

// VerificarEvidenciasRecibo valida el vinculo canonico del recibo y las dos
// firmas. No sustituye la relectura durable: el adaptador PostgreSQL T20 debe
// invocarlo sobre la fila inmutable recuperada tras COMMIT.
func (v *verificadorFirmasKMSDesarrollo) VerificarEvidenciasRecibo(
	ctx context.Context,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	if v == nil || ctx == nil || ctx.Err() != nil ||
		!procedenciaKMSDesarrolloValida(recibo.Procedencia) {
		return ErrKMSDesarrolloNoDisponible
	}
	atestacion, solicitud, revalidacion, err := recibo.EvidenciasKMSParaVerificacion()
	if err != nil || v.VerificarAtestacion(atestacion) != nil ||
		v.VerificarRevalidacion(solicitud, revalidacion) != nil || ctx.Err() != nil {
		return ErrKMSDesarrolloNoDisponible
	}
	return nil
}

func procedenciaKMSDesarrolloValida(procedencia gobiernoconvocatorias.ProcedenciaActoBorrador) bool {
	esperada, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		config.ExecutionProfileDevelopment,
		gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		proveedorSeguridadDesarrolloRef,
		false,
	)
	return err == nil && procedencia == esperada
}

func (e *emisorKMSDesarrollo) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		proveedorEmisorKMSDesarrolloRef, "adaptador:kms:desarrollo:emisor:v1",
		"credencial:kms:desarrollo:atestacion-ed25519:v1", "rol:kms:desarrollo:emisor-atestacion:v1",
	)
	return identidad
}

func (r *revalidadorKMSDesarrollo) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		proveedorRevalidadorKMSDesarrolloRef, "adaptador:kms:desarrollo:revalidador:v1",
		"credencial:kms:desarrollo:revalidacion-ed25519:v1", "rol:kms:desarrollo:revalidador:v1",
	)
	return identidad
}

func (v *verificadorFirmasKMSDesarrollo) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		proveedorVerificadorKMSDesarrolloRef, "adaptador:kms:desarrollo:verificador-recibo:v1",
		"credencial:kms:desarrollo:publicas-ed25519:v1", "rol:kms:desarrollo:verificador-recibo:v1",
	)
	return identidad
}

func huellaComprobacionAtestacion(preimagen, firma []byte) string {
	huellaPreimagen := sha256.Sum256(preimagen)
	huellaFirma := sha256.Sum256(firma)
	canonico, _ := json.Marshal(struct {
		Dominio         string `json:"dominio"`
		HuellaPreimagen string `json:"huella_preimagen_sha256"`
		HuellaFirma     string `json:"huella_firma_sha256"`
	}{
		Dominio:         "vec.kms.desarrollo.comprobacion-atestacion.v1",
		HuellaPreimagen: hex.EncodeToString(huellaPreimagen[:]),
		HuellaFirma:     hex.EncodeToString(huellaFirma[:]),
	})
	defer borrarBytes(canonico)
	huella := sha256.Sum256(canonico)
	return hex.EncodeToString(huella[:])
}

func textoConstanteIgual(izquierda, derecha string) bool {
	return len(izquierda) == len(derecha) &&
		subtle.ConstantTimeCompare([]byte(izquierda), []byte(derecha)) == 1
}

func derivarClaveDesarrollo(maestra [sha256.Size]byte, dominio string) [sha256.Size]byte {
	mac := hmac.New(sha256.New, maestra[:])
	_, _ = mac.Write([]byte(dominio))
	var resultado [sha256.Size]byte
	copy(resultado[:], mac.Sum(nil))
	return resultado
}

var (
	_ gobiernoconvocatorias.CifradorAEADKMSBorrador          = (*emisorKMSDesarrollo)(nil)
	_ gobiernoconvocatorias.RevalidadorAtestacionKMSBorrador = (*revalidadorKMSDesarrollo)(nil)
	_ gobiernoconvocatorias.DescriptorAutoridadBorrador      = (*emisorKMSDesarrollo)(nil)
	_ gobiernoconvocatorias.DescriptorAutoridadBorrador      = (*revalidadorKMSDesarrollo)(nil)
	_ gobiernoconvocatorias.DescriptorAutoridadBorrador      = (*verificadorFirmasKMSDesarrollo)(nil)
)
