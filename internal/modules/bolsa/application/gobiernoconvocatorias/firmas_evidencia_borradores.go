package gobiernoconvocatorias

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const maximoFirmaEvidenciaBorradorBytes = 16 << 10

// FirmaEvidenciaBorrador conserva una firma opaca y su clave pública fijada.
// El algoritmo es una referencia gobernada: desarrollo usa Ed25519, mientras
// producción puede incorporar otro conector sin cambiar el núcleo.
type FirmaEvidenciaBorrador struct {
	bloqueoSerializacionDiario
	AlgoritmoFirma           string
	VerificadorRef           string
	HuellaClavePublicaSHA256 string
	HuellaPreimagenSHA256    string
	firmaBase64URLSinRelleno string
}

type FuncionFirmaEvidenciaBorrador func([]byte) ([]byte, error)

func FirmarEvidenciaBorrador(
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256 string,
	preimagen []byte,
	firmar FuncionFirmaEvidenciaBorrador,
) (FirmaEvidenciaBorrador, error) {
	if firmar == nil || len(preimagen) == 0 || len(preimagen) > 1<<20 {
		return FirmaEvidenciaBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	firma, err := firmar(append([]byte(nil), preimagen...))
	if err != nil {
		return FirmaEvidenciaBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	defer borrarBytesFirmaBorrador(firma)
	return RestaurarFirmaEvidenciaBorrador(
		algoritmoFirma, verificadorRef, huellaClavePublicaSHA256,
		preimagen, base64.RawURLEncoding.EncodeToString(firma),
	)
}

// RestaurarFirmaEvidenciaBorrador permite al adaptador de lectura reconstruir
// la evidencia persistida. Solo valida forma y compromiso; la autoridad nace
// de verificar criptográficamente la firma con la clave pública fijada.
func RestaurarFirmaEvidenciaBorrador(
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256 string,
	preimagen []byte,
	firmaBase64URLSinRelleno string,
) (FirmaEvidenciaBorrador, error) {
	huella := sha256.Sum256(preimagen)
	f := FirmaEvidenciaBorrador{
		AlgoritmoFirma: algoritmoFirma, VerificadorRef: verificadorRef,
		HuellaClavePublicaSHA256: huellaClavePublicaSHA256,
		HuellaPreimagenSHA256:    hex.EncodeToString(huella[:]),
		firmaBase64URLSinRelleno: firmaBase64URLSinRelleno,
	}
	if !f.validaPara(preimagen) {
		return FirmaEvidenciaBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return f, nil
}

// RestaurarFirmaEvidenciaBorradorPersistida recompone la forma durable sin
// atribuirle autoridad. El agregado que la contiene debe recalcular después
// la preimagen canónica y un verificador independiente debe comprobar la
// firma con la clave pública fijada antes de devolver éxito.
func RestaurarFirmaEvidenciaBorradorPersistida(
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256,
	huellaPreimagenSHA256, firmaBase64URLSinRelleno string,
) (FirmaEvidenciaBorrador, error) {
	f := FirmaEvidenciaBorrador{
		AlgoritmoFirma: algoritmoFirma, VerificadorRef: verificadorRef,
		HuellaClavePublicaSHA256: huellaClavePublicaSHA256,
		HuellaPreimagenSHA256:    huellaPreimagenSHA256,
		firmaBase64URLSinRelleno: firmaBase64URLSinRelleno,
	}
	if !f.formaValida() {
		return FirmaEvidenciaBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return f, nil
}

func (f FirmaEvidenciaBorrador) validaPara(preimagen []byte) bool {
	if !f.formaValida() {
		return false
	}
	huella := sha256.Sum256(preimagen)
	return coincideTextoConstante(f.HuellaPreimagenSHA256, hex.EncodeToString(huella[:]))
}

func (f FirmaEvidenciaBorrador) formaValida() bool {
	firma, err := base64.RawURLEncoding.DecodeString(f.firmaBase64URLSinRelleno)
	if err != nil || len(firma) == 0 || len(firma) > maximoFirmaEvidenciaBorradorBytes ||
		base64.RawURLEncoding.EncodeToString(firma) != f.firmaBase64URLSinRelleno ||
		!referenciaProyeccionValida(f.AlgoritmoFirma) ||
		!referenciaProyeccionValida(f.VerificadorRef) ||
		!huellaHexValida(f.HuellaClavePublicaSHA256) ||
		!huellaHexValida(f.HuellaPreimagenSHA256) {
		return false
	}
	return true
}

func (f FirmaEvidenciaBorrador) DatosParaVerificacion(
	preimagen []byte,
) (
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256 string,
	firma []byte,
	err error,
) {
	if !f.validaPara(preimagen) {
		return "", "", "", nil, ErrRevalidacionKMSBorradorFallo
	}
	firma, _ = base64.RawURLEncoding.DecodeString(f.firmaBase64URLSinRelleno)
	return f.AlgoritmoFirma, f.VerificadorRef, f.HuellaClavePublicaSHA256,
		append([]byte(nil), firma...), nil
}

func (f FirmaEvidenciaBorrador) FirmaBase64URLParaPersistencia() (string, error) {
	if f.firmaBase64URLSinRelleno == "" {
		return "", ErrRevalidacionKMSBorradorFallo
	}
	return f.firmaBase64URLSinRelleno, nil
}

func borrarBytesFirmaBorrador(datos []byte) {
	for indice := range datos {
		datos[indice] = 0
	}
}
