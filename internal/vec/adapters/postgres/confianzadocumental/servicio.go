package confianzadocumental

import (
	"bytes"
	"context"
	"crypto/elliptic"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"time"

	"github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/vec/ports"
)

type relojConfiable interface {
	Ahora() time.Time
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

type raizVerificacion struct {
	algoritmo                        AlgoritmoCOSEDocumental
	verificador                      cose.Verifier
	huellaClaveSHA256                string
	audiencia                        AudienciaCOSEDocumental
	suiteAtestacionPDP               string
	audienciaDespliegueAtestacionPDP string
	estado                           EstadoConfianzaClaveDocumental
	revocadaEn                       time.Time
	validaDesde                      time.Time
	validaHasta                      time.Time
}

// Servicio contiene una instantanea defensiva de la lista positiva. El reloj
// y la configuracion son internos; el solicitante no puede proponer una fecha.
type Servicio struct {
	raices                    map[string]raizVerificacion
	reloj                     relojConfiable
	repositorioEjecucionV4    repositorioEjecucionDocumentalV4
	revisionConfianza         string
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
}

func (*Servicio) String() string {
	return "[SERVICIO-CONFIANZA-DOCUMENTAL-REDACTADO]"
}

func (s *Servicio) GoString() string { return s.String() }
func (s *Servicio) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s *Servicio) LogValue() slog.Value { return slog.StringValue(s.String()) }
func (*Servicio) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*Servicio) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*Servicio) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*Servicio) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*Servicio) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*Servicio) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}

func nuevoServicioConRelojSistema(configuracion ConfiguracionConfianzaFijada) (*Servicio, error) {
	return nuevoServicioConReloj(configuracion, relojSistema{})
}

// nuevoServicioConReloj permanece privado: permite pruebas deterministas sin
// abrir a adaptadores una fabrica parametrizable de autoridad.
func nuevoServicioConReloj(
	configuracion ConfiguracionConfianzaFijada,
	reloj relojConfiable,
) (*Servicio, error) {
	if configuracion.validar() != nil || reloj == nil {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	servicio := &Servicio{
		raices:                    make(map[string]raizVerificacion, len(configuracion.raices)),
		reloj:                     reloj,
		revisionConfianza:         configuracion.revision,
		huellaConfiguracionSHA256: configuracion.huellaSHA256,
		configuracionPublicadaEn:  configuracion.publicadaEn,
		configuracionExpiraEn:     configuracion.expiraEn,
	}
	for _, raiz := range configuracion.raices {
		claveClonada, huellaClave, err := clonarClavePublicaDocumental(
			raiz.algoritmo, raiz.clavePublica,
		)
		if raiz.validar() != nil || err != nil || huellaClave != raiz.huellaClaveSHA256 {
			return nil, ErrConfiguracionConfianzaDocumentalInvalida
		}
		algoritmoCOSE, err := algoritmoBiblioteca(raiz.algoritmo)
		if err != nil {
			return nil, ErrConfiguracionConfianzaDocumentalInvalida
		}
		verificador, err := cose.NewVerifier(algoritmoCOSE, claveClonada)
		if err != nil {
			return nil, ErrConfiguracionConfianzaDocumentalInvalida
		}
		indice := string(raiz.claveID)
		if _, duplicada := servicio.raices[indice]; duplicada {
			return nil, ErrConfiguracionConfianzaDocumentalInvalida
		}
		servicio.raices[indice] = raizVerificacion{
			algoritmo: raiz.algoritmo, verificador: verificador,
			huellaClaveSHA256: huellaClave, audiencia: raiz.audiencia,
			suiteAtestacionPDP:               raiz.suiteAtestacionPDP,
			audienciaDespliegueAtestacionPDP: raiz.audienciaDespliegueAtestacionPDP,
			estado:                           raiz.estado, revocadaEn: raiz.revocadaEn,
			validaDesde: raiz.validaDesde, validaHasta: raiz.validaHasta,
		}
	}
	return servicio, nil
}

// VerificarCOSESign1 interpreta y verifica localmente el sobre. Solo acepta
// las cabeceras protegidas alg y kid, ninguna cabecera no protegida, el
// payload exacto y la audiencia cerrada ligada mediante external_aad.
func (s *Servicio) VerificarCOSESign1(
	ctx context.Context,
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (PruebaCOSESign1DocumentalVerificada, error) {
	if ctx == nil || s == nil || s.reloj == nil || len(s.raices) == 0 ||
		solicitud.Validar() != nil || sobre.ValidarSintaxis() != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	if err := ctx.Err(); err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, err
	}
	return s.verificarCOSESign1En(ctx, solicitud, sobre, s.reloj.Ahora())
}

func (s *Servicio) verificarCOSESign1En(
	ctx context.Context,
	solicitud SolicitudVerificacionCOSESign1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
	verificadaEn time.Time,
) (PruebaCOSESign1DocumentalVerificada, error) {
	if ctx == nil || s == nil || len(s.raices) == 0 || solicitud.Validar() != nil ||
		sobre.ValidarSintaxis() != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	if err := ctx.Err(); err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, err
	}
	if !instanteCanonicoDocumental(verificadaEn) ||
		verificadaEn.Before(s.configuracionPublicadaEn) ||
		!verificadaEn.Before(s.configuracionExpiraEn) {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	contenido, err := sobre.COSESign1()
	if err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	limitePayload, audienciaValida := limitePayloadPorAudiencia(solicitud.audiencia)
	if !audienciaValida || len(contenido) > limitePayload+margenMaximoSobreCOSEDocumentalV4 {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	var mensaje cose.Sign1Message
	if err := mensaje.UnmarshalCBOR(contenido); err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	// COSE autentica el contenido, pero una representacion CBOR equivalente
	// puede conservar una firma valida y cambiar la huella del sobre. Se exige
	// una unica representacion determinista antes de convertir esa huella en
	// evidencia. Construir otro mensaje sin Raw* obliga a recodificar tambien
	// ambos mapas, no solo el tag, array y bstr exteriores, y deja intacto el
	// mensaje original que despues se somete a verificacion criptografica.
	mensajeDeterminista := cose.Sign1Message{
		Headers: cose.Headers{
			Protected:   mensaje.Headers.Protected,
			Unprotected: mensaje.Headers.Unprotected,
		},
		Payload:   append([]byte(nil), mensaje.Payload...),
		Signature: append([]byte(nil), mensaje.Signature...),
	}
	contenidoDeterminista, err := mensajeDeterminista.MarshalCBOR()
	if err != nil || !bytes.Equal(contenidoDeterminista, contenido) {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	if len(mensaje.Headers.Protected) != 2 || len(mensaje.Headers.Unprotected) != 0 {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	algoritmoCOSE, err := mensaje.Headers.Protected.Algorithm()
	if err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	algoritmo, err := algoritmoDocumental(algoritmoCOSE)
	if err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	// ECDSA admite (r, N-s) para la misma firma. Low-S fija una unica forma y
	// evita que una firma valida tenga dos huellas de sobre distintas.
	if algoritmo == AlgoritmoCOSEDocumentalES256 && !firmaCOSEES256Canonica(mensaje.Signature) {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	claveIDValor, protegida := mensaje.Headers.Protected[cose.HeaderLabelKeyID]
	claveID, tipoCorrecto := claveIDValor.([]byte)
	if !protegida || !tipoCorrecto || !claveIDDocumentalValida(claveID) {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	raiz, conocida := s.raices[string(claveID)]
	if !conocida || raiz.algoritmo != algoritmo || raiz.audiencia != solicitud.audiencia ||
		raiz.estado != EstadoConfianzaClaveDocumentalActiva || !raiz.revocadaEn.IsZero() ||
		verificadaEn.Before(raiz.validaDesde) || !verificadaEn.Before(raiz.validaHasta) ||
		!bytes.Equal(mensaje.Payload, solicitud.payloadEsperado) {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	aad, err := solicitud.AADExterno()
	if err != nil || mensaje.Verify(aad, raiz.verificador) != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	if err := ctx.Err(); err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, err
	}
	huellaSobre, err := sobre.HuellaSHA256()
	if err != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	prueba, err := nuevaPruebaCOSESign1DocumentalVerificada(
		comprobacionCriptograficaSatisfactoria{
			algoritmo: algoritmo, claveID: claveID,
			huellaClaveSHA256:   raiz.huellaClaveSHA256,
			estadoConfianza:     EstadoConfianzaClaveDocumentalActiva,
			audiencia:           solicitud.audiencia,
			huellaPayloadSHA256: huellaBytesDocumentales(solicitud.payloadEsperado),
			huellaSobreSHA256:   huellaSobre, verificadaEn: verificadaEn,
			raizValidaDesde: raiz.validaDesde, raizValidaHasta: raiz.validaHasta,
			revisionConfianza:         s.revisionConfianza,
			huellaConfiguracionSHA256: s.huellaConfiguracionSHA256,
			configuracionPublicadaEn:  s.configuracionPublicadaEn,
			configuracionExpiraEn:     s.configuracionExpiraEn,
		},
	)
	if err != nil || prueba.ValidarPara(solicitud, sobre) != nil {
		return PruebaCOSESign1DocumentalVerificada{}, ErrVerificacionCOSESign1Fallida
	}
	return prueba, nil
}

func firmaCOSEES256Canonica(firma []byte) bool {
	const bytesComponenteP256 = 32
	if len(firma) != 2*bytesComponenteP256 {
		return false
	}
	orden := elliptic.P256().Params().N
	mitadOrden := new(big.Int).Rsh(new(big.Int).Set(orden), 1)
	r := new(big.Int).SetBytes(firma[:bytesComponenteP256])
	s := new(big.Int).SetBytes(firma[bytesComponenteP256:])
	return r.Sign() > 0 && r.Cmp(orden) < 0 &&
		s.Sign() > 0 && s.Cmp(mitadOrden) <= 0
}

func algoritmoBiblioteca(algoritmo AlgoritmoCOSEDocumental) (cose.Algorithm, error) {
	switch algoritmo {
	case AlgoritmoCOSEDocumentalEdDSA:
		return cose.AlgorithmEdDSA, nil
	case AlgoritmoCOSEDocumentalES256:
		return cose.AlgorithmES256, nil
	default:
		return cose.AlgorithmReserved, ErrConfiguracionConfianzaDocumentalInvalida
	}
}

func algoritmoDocumental(algoritmo cose.Algorithm) (AlgoritmoCOSEDocumental, error) {
	switch algoritmo {
	case cose.AlgorithmEdDSA:
		return AlgoritmoCOSEDocumentalEdDSA, nil
	case cose.AlgorithmES256:
		return AlgoritmoCOSEDocumentalES256, nil
	default:
		return "", ErrVerificacionCOSESign1Fallida
	}
}
