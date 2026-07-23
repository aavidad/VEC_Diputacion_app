package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

var (
	ErrParseoAtestacionAutorizacionV3Invalido = errors.New(
		"vec: parseo no autoritativo VEC-AD-3 invalido",
	)
	ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida = errors.New(
		"vec: serializacion de proyeccion no autoritativa VEC-AD-3 prohibida",
	)
)

const (
	maximoBytesDecisionAtestacionAutorizacionV3 = 256 * 1024
	maximoBytesMotivoAtestacionAutorizacionV3   = 64 * 1024
	maximoProfundidadJSONAtestacionV3           = 8
	maximoCamposObjetoJSONAtestacionV3          = 64
	maximoElementosListaJSONAtestacionV3        = 1024
)

type datosProyeccionAtestacionAutorizacionV3 struct {
	referenciaDecision string
	huellaDecision     string
	huellaMotivo       string
	referenciaContexto string
	huellaContexto     string
	huellaManifiesto   string
	autoridadEfectiva  AutoridadProcedenciaContextoActorV1
	resueltoEn         time.Time
}

// ProyeccionAtestacionAutorizacionV3NoAutoritativa solo acredita forma
// canónica y coherencia interna del mensaje. No contiene una decisión viva,
// no verifica firma, confianza, vigencia ni consumo y no concede autoridad.
type ProyeccionAtestacionAutorizacionV3NoAutoritativa struct {
	bloqueoSerializacionProyeccionAtestacionAutorizacionV3
	cabecera CabeceraAtestacionAutorizacionV3
	datos    *datosProyeccionAtestacionAutorizacionV3
}

func ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
	contenido []byte,
) (ProyeccionAtestacionAutorizacionV3NoAutoritativa, error) {
	if len(contenido) < 8 ||
		len(contenido) > TamanoMaximoMensajeAtestacionAutorizacionV3 {
		return ProyeccionAtestacionAutorizacionV3NoAutoritativa{},
			errorParseoAtestacionAutorizacionV3()
	}
	longitudDeclarada := int64(binaryBigEndianUint64AtestacionV3(
		contenido[len(contenido)-8:],
	))
	if longitudDeclarada != int64(len(contenido)) {
		return ProyeccionAtestacionAutorizacionV3NoAutoritativa{},
			errorParseoAtestacionAutorizacionV3()
	}

	lector := lectorHistoricoAtestacionAutorizacionV1{contenido: contenido}
	lector.exigirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV3))
	lector.exigirByte(0)
	cabecera := CabeceraAtestacionAutorizacionV3{
		FormatoVersion: lector.leerUint16(),
		Suite:          lector.leerTexto(128),
		ClaveID:        lector.leerTexto(512),
		Audiencia:      lector.leerTexto(512),
	}
	decisionCanonica := leerBloqueAtestacionAutorizacionV3(
		&lector,
		maximoBytesDecisionAtestacionAutorizacionV3,
	)
	motivoCanonico := leerBloqueAtestacionAutorizacionV3(
		&lector,
		maximoBytesMotivoAtestacionAutorizacionV3,
	)
	referenciaContexto := lector.leerTexto(longitudMaximaRegistroContextoActorV2)
	contextoCanonico := leerBloqueAtestacionAutorizacionV3(
		&lector,
		TamanoMaximoRepresentacionContextoActorV2,
	)
	huellaContexto := lector.leerTexto(sha256.Size * 2)
	manifiestoCanonico := leerBloqueAtestacionAutorizacionV3(
		&lector,
		TamanoMaximoManifiestoProcedenciaContextoActorV1,
	)
	huellaManifiesto := lector.leerTexto(sha256.Size * 2)
	autoridad := AutoridadProcedenciaContextoActorV1(lector.leerTexto(64))
	resueltoEn := lector.leerInstante()
	longitudFinal := lector.leerUint64()
	if lector.err != nil || lector.posicion != len(contenido) ||
		longitudFinal != uint64(len(contenido)) || cabecera.Validar() != nil {
		return ProyeccionAtestacionAutorizacionV3NoAutoritativa{},
			errorParseoAtestacionAutorizacionV3()
	}

	decision, huellaDecision, err := validarDecisionCanonicaAtestacionAutorizacionV3(
		decisionCanonica,
	)
	referenciaMotivo, huellaMotivo, errMotivo :=
		validarMotivoCanonicoAtestacionAutorizacionV3(motivoCanonico)
	contexto, errContexto := RehidratarContextoActorVinculadoV2(contextoCanonico)
	manifiesto, errManifiesto := RehidratarManifiestoProcedenciaContextoActorV1(
		manifiestoCanonico,
	)
	huellaContextoCalculada, errHuellaContexto := contexto.HuellaSHA256VinculadaV2()
	huellaManifiestoCalculada, errHuellaManifiesto :=
		HuellaSHA256ManifiestoProcedenciaContextoActorV1(manifiestoCanonico)
	if err != nil || errMotivo != nil || errContexto != nil ||
		errManifiesto != nil || errHuellaContexto != nil ||
		errHuellaManifiesto != nil ||
		decision.MotivoHuellaSHA256 != huellaMotivo ||
		decision.VinculoAutenticacionActor.RegistroContextoRef != referenciaContexto ||
		decision.VinculoAutenticacionActor.ContextoActorEsquema !=
			EsquemaRepresentacionContextoActorV2 ||
		decision.VinculoAutenticacionActor.ContextoActorRef !=
			contexto.Instantanea.VinculoRef ||
		decision.VinculoAutenticacionActor.ContextoActorVersion !=
			contexto.Instantanea.VinculoVersion ||
		decision.VinculoAutenticacionActor.ContextoActorCuentaVersion !=
			contexto.Instantanea.CuentaVersion ||
		decision.VinculoAutenticacionActor.ContextoActorHuellaSHA256 !=
			huellaContexto ||
		huellaContextoCalculada != huellaContexto ||
		decision.VinculoAutenticacionActor.ManifiestoProcedenciaHuellaSHA256 !=
			huellaManifiesto ||
		huellaManifiestoCalculada != huellaManifiesto ||
		decision.VinculoAutenticacionActor.AutoridadEfectiva != autoridad ||
		autoridad != AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		manifiesto.AutoridadEfectiva != autoridad ||
		manifiesto.ValidarParaContexto(contexto) != nil ||
		!resueltoEn.Equal(contexto.ResueltoEn) ||
		decision.PrincipalID != contexto.Principal.ID ||
		decision.PerfilActivoRef != contexto.PerfilActivoRef ||
		decision.VinculoAutenticacionActor.PrincipalID != contexto.Principal.ID ||
		decision.VinculoAutenticacionActor.PerfilActivoRef !=
			contexto.PerfilActivoRef ||
		referenciaMotivo.CatalogoID == "" {
		return ProyeccionAtestacionAutorizacionV3NoAutoritativa{},
			errorParseoAtestacionAutorizacionV3()
	}

	proyeccion := ProyeccionAtestacionAutorizacionV3NoAutoritativa{
		cabecera: cabecera,
		datos: &datosProyeccionAtestacionAutorizacionV3{
			referenciaDecision: decision.DecisionRef,
			huellaDecision:     huellaDecision, huellaMotivo: huellaMotivo,
			referenciaContexto: referenciaContexto,
			huellaContexto:     huellaContexto, huellaManifiesto: huellaManifiesto,
			autoridadEfectiva: autoridad, resueltoEn: resueltoEn,
		},
	}
	if proyeccion.validar() != nil {
		return ProyeccionAtestacionAutorizacionV3NoAutoritativa{},
			errorParseoAtestacionAutorizacionV3()
	}
	return proyeccion, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) Cabecera() (
	CabeceraAtestacionAutorizacionV3,
	error,
) {
	if p.validar() != nil {
		return CabeceraAtestacionAutorizacionV3{},
			errorParseoAtestacionAutorizacionV3()
	}
	return p.cabecera, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) DecisionRef() (
	string,
	error,
) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV3()
	}
	return p.datos.referenciaDecision, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) HuellaDecisionSHA256() (
	string,
	error,
) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV3()
	}
	return p.datos.huellaDecision, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) HuellaMotivoSHA256() (
	string,
	error,
) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV3()
	}
	return p.datos.huellaMotivo, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) ReferenciaContextoActor() (
	string,
	error,
) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV3()
	}
	return p.datos.referenciaContexto, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) HuellaContextoActorSHA256() (
	string,
	error,
) {
	if p.validar() != nil {
		return "", errorParseoAtestacionAutorizacionV3()
	}
	return p.datos.huellaContexto, nil
}

func (p ProyeccionAtestacionAutorizacionV3NoAutoritativa) validar() error {
	if p.datos == nil || p.cabecera.Validar() != nil ||
		!textoAutorizacionSinComodinSeguro(p.datos.referenciaDecision, 512, false) ||
		!huellaSHA256AutorizacionValida(p.datos.huellaDecision) ||
		!huellaSHA256AutorizacionValida(p.datos.huellaMotivo) ||
		!referenciaRegistroContextoActorV2Valida(p.datos.referenciaContexto) ||
		!huellaSHA256AutorizacionValida(p.datos.huellaContexto) ||
		!huellaSHA256AutorizacionValida(p.datos.huellaManifiesto) ||
		p.datos.autoridadEfectiva !=
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		!instanteAutorizacionCanonico(p.datos.resueltoEn) {
		return errorParseoAtestacionAutorizacionV3()
	}
	return nil
}

func validarDecisionCanonicaAtestacionAutorizacionV3(
	contenido []byte,
) (decisionAutorizacionCanonicaV3, string, error) {
	var decision decisionAutorizacionCanonicaV3
	if validarJSONAtestacionAutorizacionV3SinDuplicados(contenido) != nil {
		return decision, "", errorParseoAtestacionAutorizacionV3()
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&decision); err != nil ||
		exigirFinJSONContextoActorCanonico(decodificador) != nil {
		return decision, "", errorParseoAtestacionAutorizacionV3()
	}
	canonico, err := json.Marshal(decision)
	if err != nil || !bytes.Equal(canonico, contenido) ||
		decision.Esquema != EsquemaHuellaDecisionAutorizacionV3 ||
		decision.BloqueVersion != VersionDecisionAutorizacionLigadaV3 ||
		!decision.Concedida || decision.Codigo != "concedida" ||
		!textoAutorizacionSinComodinSeguro(decision.DecisionRef, 512, false) ||
		decision.PrincipalID != decision.VinculoAutenticacionActor.PrincipalID ||
		decision.PerfilActivoRef !=
			decision.VinculoAutenticacionActor.PerfilActivoRef ||
		!huellaSHA256AutorizacionValida(decision.MotivoHuellaSHA256) {
		return decision, "", errorParseoAtestacionAutorizacionV3()
	}
	suma := sha256.Sum256(contenido)
	return decision, hex.EncodeToString(suma[:]), nil
}

func validarMotivoCanonicoAtestacionAutorizacionV3(
	contenido []byte,
) (ReferenciaEntradaCatalogo, string, error) {
	var documento motivoAutorizacionCanonicoV2
	if validarJSONAtestacionAutorizacionV3SinDuplicados(contenido) != nil {
		return ReferenciaEntradaCatalogo{}, "", errorParseoAtestacionAutorizacionV3()
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&documento); err != nil ||
		exigirFinJSONContextoActorCanonico(decodificador) != nil {
		return ReferenciaEntradaCatalogo{}, "", errorParseoAtestacionAutorizacionV3()
	}
	canonico, err := json.Marshal(documento)
	referencia := ReferenciaEntradaCatalogo{
		CatalogoID:           documento.Referencia.CatalogoID,
		CatalogoVersion:      documento.Referencia.CatalogoVersion,
		CatalogoHuellaSHA256: documento.Referencia.CatalogoHuellaSHA256,
		EntradaClave:         documento.Referencia.EntradaClave,
	}
	huella, errHuella := HuellaSHA256MotivoAutorizacionV2(referencia)
	if err != nil || errHuella != nil || !bytes.Equal(canonico, contenido) ||
		documento.Esquema != EsquemaHuellaMotivoAutorizacionV2 {
		return ReferenciaEntradaCatalogo{}, "", errorParseoAtestacionAutorizacionV3()
	}
	return referencia, huella, nil
}

func leerBloqueAtestacionAutorizacionV3(
	lector *lectorHistoricoAtestacionAutorizacionV1,
	maximo int,
) []byte {
	if lector == nil || maximo <= 0 {
		return nil
	}
	longitud := lector.leerUint32()
	if lector.err != nil || longitud == 0 || uint64(longitud) > uint64(maximo) ||
		uint64(longitud) > uint64(len(lector.contenido)-lector.posicion) {
		lector.err = errorParseoAtestacionAutorizacionV3()
		return nil
	}
	return lector.tomar(int(longitud))
}

func validarJSONAtestacionAutorizacionV3SinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	primero, err := decodificador.Token()
	if err != nil || primero != json.Delim('{') ||
		consumirJSONAtestacionAutorizacionV3(
			decodificador,
			json.Delim('{'),
			1,
		) != nil ||
		exigirFinJSONContextoActorCanonico(decodificador) != nil {
		return errorParseoAtestacionAutorizacionV3()
	}
	return nil
}

func consumirJSONAtestacionAutorizacionV3(
	decodificador *json.Decoder,
	apertura json.Delim,
	profundidad int,
) error {
	if profundidad > maximoProfundidadJSONAtestacionV3 {
		return errorParseoAtestacionAutorizacionV3()
	}
	claves := make(map[string]struct{})
	elementos := 0
	for decodificador.More() {
		elementos++
		if apertura == json.Delim('{') &&
			elementos > maximoCamposObjetoJSONAtestacionV3 ||
			apertura == json.Delim('[') &&
				elementos > maximoElementosListaJSONAtestacionV3 {
			return errorParseoAtestacionAutorizacionV3()
		}
		if apertura == json.Delim('{') {
			token, err := decodificador.Token()
			clave, valida := token.(string)
			if err != nil || !valida {
				return errorParseoAtestacionAutorizacionV3()
			}
			if _, repetida := claves[clave]; repetida {
				return errorParseoAtestacionAutorizacionV3()
			}
			claves[clave] = struct{}{}
		}
		valor, err := decodificador.Token()
		if err != nil {
			return errorParseoAtestacionAutorizacionV3()
		}
		if delimitador, compuesto := valor.(json.Delim); compuesto {
			if delimitador != json.Delim('{') && delimitador != json.Delim('[') ||
				consumirJSONAtestacionAutorizacionV3(
					decodificador,
					delimitador,
					profundidad+1,
				) != nil {
				return errorParseoAtestacionAutorizacionV3()
			}
		}
	}
	cierre, err := decodificador.Token()
	if err != nil ||
		apertura == json.Delim('{') && cierre != json.Delim('}') ||
		apertura == json.Delim('[') && cierre != json.Delim(']') {
		return errorParseoAtestacionAutorizacionV3()
	}
	return nil
}

func binaryBigEndianUint64AtestacionV3(contenido []byte) uint64 {
	if len(contenido) != 8 {
		return 0
	}
	return uint64(contenido[0])<<56 |
		uint64(contenido[1])<<48 |
		uint64(contenido[2])<<40 |
		uint64(contenido[3])<<32 |
		uint64(contenido[4])<<24 |
		uint64(contenido[5])<<16 |
		uint64(contenido[6])<<8 |
		uint64(contenido[7])
}

func errorParseoAtestacionAutorizacionV3() error {
	return errors.Join(
		ErrParseoAtestacionAutorizacionV3Invalido,
		ErrMensajeAtestacionAutorizacionInvalido,
	)
}

type bloqueoSerializacionProyeccionAtestacionAutorizacionV3 struct{}

func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) GobDecode([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalCBOR([]byte) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalYAML() (any, error) {
	return nil, ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (*bloqueoSerializacionProyeccionAtestacionAutorizacionV3) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida
}
func (bloqueoSerializacionProyeccionAtestacionAutorizacionV3) String() string {
	return "[PROYECCION-VEC-AD-3-NOMINAL-NO-AUTORITATIVA-REDACTADA]"
}
func (b bloqueoSerializacionProyeccionAtestacionAutorizacionV3) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionProyeccionAtestacionAutorizacionV3) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionProyeccionAtestacionAutorizacionV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
