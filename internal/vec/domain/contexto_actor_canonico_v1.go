package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"
)

const (
	// TamanoMaximoRepresentacionContextoActorV1 limita la entrada antes de
	// decodificarla. La cota supera holgadamente cualquier documento valido con
	// los tipos V1 cerrados, pero evita usar el rehidratador como analizador JSON
	// generico.
	TamanoMaximoRepresentacionContextoActorV1    = 64 * 1024
	formatoInstanteContextoActorCanonico         = "2006-01-02T15:04:05.000000Z"
	profundidadMaximaJSONContextoActorCanonico   = 4
	camposMaximosObjetoJSONContextoActorCanonico = 32
)

var ErrRepresentacionContextoActorV1Invalida = errors.New("vec: representacion canonica de contexto de actor v1 invalida")

type vinculoReferenciaContextoActorCanonico struct {
	VinculoRef   string `json:"vinculo_ref"`
	Version      uint64 `json:"version"`
	Tipo         string `json:"tipo"`
	Referencia   string `json:"referencia"`
	Estado       string `json:"estado"`
	VigenteDesde string `json:"vigente_desde"`
	VigenteHasta string `json:"vigente_hasta"`
}

type contextoActorCanonicoV1 struct {
	Esquema          string                                   `json:"esquema"`
	PrincipalRef     string                                   `json:"principal_ref"`
	Metodo           AuthMethod                               `json:"metodo"`
	Garantia         AuthAssurance                            `json:"garantia"`
	PerfilActivoRef  string                                   `json:"perfil_activo_ref"`
	PersonaRef       string                                   `json:"persona_ref"`
	ContextoActorRef string                                   `json:"contexto_actor_ref"`
	ContextoVersion  uint64                                   `json:"contexto_version"`
	CuentaRef        string                                   `json:"cuenta_ref"`
	PersonaVersion   uint64                                   `json:"persona_version"`
	PerfilVersion    uint64                                   `json:"perfil_version"`
	Estado           string                                   `json:"estado"`
	VigenteDesde     string                                   `json:"vigente_desde"`
	VigenteHasta     string                                   `json:"vigente_hasta"`
	ResueltoEn       string                                   `json:"resuelto_en"`
	Vinculos         []vinculoReferenciaContextoActorCanonico `json:"vinculos"`
}

// RepresentacionCanonicaVinculadaV1 entrega una copia de los bytes exactos
// comprometidos por HuellaSHA256VinculadaV1. Este es el unico documento V1 que
// un adaptador durable puede registrar o recuperar; no contiene claims libres.
func (c ContextoActor) RepresentacionCanonicaVinculadaV1() ([]byte, error) {
	canonica, err := c.Clonar()
	if err != nil || canonica.Instantanea.CuentaVersion != 0 {
		// V1 nunca tuvo cuenta_version. Rechazar una capacidad V2 evita producir
		// una preimagen que silenciosamente deje esa revision fuera de la huella.
		return nil, ErrContextoActorInvalido
	}
	documento := documentoContextoActorCanonicoV1(canonica)
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) == 0 || len(contenido) > TamanoMaximoRepresentacionContextoActorV1 {
		return nil, ErrContextoActorInvalido
	}
	return append([]byte(nil), contenido...), nil
}

func documentoContextoActorCanonicoV1(canonica ContextoActor) contextoActorCanonicoV1 {
	documento := contextoActorCanonicoV1{
		Esquema:      esquemaHuellaContextoActorV1,
		PrincipalRef: canonica.Principal.ID, Metodo: canonica.Principal.AuthMethod,
		Garantia: canonica.Principal.AuthAssurance, PerfilActivoRef: canonica.PerfilActivoRef,
		PersonaRef: canonica.PersonaRef, ContextoActorRef: canonica.Instantanea.VinculoRef,
		ContextoVersion: canonica.Instantanea.VinculoVersion, CuentaRef: canonica.Instantanea.CuentaRef,
		PersonaVersion: canonica.Instantanea.PersonaVersion,
		PerfilVersion:  canonica.Instantanea.PerfilVersion, Estado: string(canonica.Instantanea.Estado),
		VigenteDesde: formatearInstanteContextoActorCanonico(canonica.Instantanea.VigenteDesde),
		VigenteHasta: formatearInstanteContextoActorCanonico(canonica.Instantanea.VigenteHasta),
		ResueltoEn:   formatearInstanteContextoActorCanonico(canonica.ResueltoEn),
		Vinculos:     make([]vinculoReferenciaContextoActorCanonico, 0, len(canonica.Instantanea.Vinculos)),
	}
	for _, vinculo := range canonica.Instantanea.Vinculos {
		documento.Vinculos = append(documento.Vinculos, vinculoReferenciaContextoActorCanonico{
			VinculoRef: vinculo.VinculoRef, Version: vinculo.Version, Tipo: string(vinculo.Tipo),
			Referencia: vinculo.Referencia, Estado: string(vinculo.Estado),
			VigenteDesde: formatearInstanteContextoActorCanonico(vinculo.VigenteDesde),
			VigenteHasta: formatearInstanteContextoActorCanonico(vinculo.VigenteHasta),
		})
	}
	return documento
}

func (c ContextoActor) huellaSHA256RepresentacionCanonicaVinculadaV1() (string, error) {
	contenido, err := c.RepresentacionCanonicaVinculadaV1()
	if err != nil {
		return "", ErrContextoActorInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// RehidratarContextoActorVinculadoV1 reconstruye exclusivamente una evidencia
// V1 canónica. La canonicidad no acredita autenticidad, vigencia actual ni
// existencia en el registro autoritativo: el resultado no puede autorizar por
// si solo. Un adaptador debe cotejarlo con la huella esperada y con el registro
// y la version actuales antes de convertirlo en una capacidad de uso.
//
// Rechaza extensiones, claves repetidas, orden alternativo, espacios, null,
// numeros no canónicos y cualquier instante que no sea UTC con seis cifras
// decimales.
func RehidratarContextoActorVinculadoV1(contenido []byte) (ContextoActor, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoRepresentacionContextoActorV1 {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	if err := validarJSONContextoActorCanonicoSinDuplicados(contenido); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}

	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var documento contextoActorCanonicoV1
	if err := decodificador.Decode(&documento); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	if err := exigirFinJSONContextoActorCanonico(decodificador); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	resultado, err := rehidratarDocumentoContextoActorV1(documento)
	if err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	canonicos, err := resultado.RepresentacionCanonicaVinculadaV1()
	if err != nil || !bytes.Equal(canonicos, contenido) {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	return resultado, nil
}

func rehidratarDocumentoContextoActorV1(documento contextoActorCanonicoV1) (ContextoActor, error) {
	if documento.Esquema != esquemaHuellaContextoActorV1 || documento.PrincipalRef != documento.PersonaRef {
		return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
	}
	vigenteDesde, err := parsearInstanteContextoActorCanonico(documento.VigenteDesde)
	if err != nil {
		return ContextoActor{}, err
	}
	vigenteHasta, err := parsearInstanteContextoActorCanonico(documento.VigenteHasta)
	if err != nil {
		return ContextoActor{}, err
	}
	resueltoEn, err := parsearInstanteContextoActorCanonico(documento.ResueltoEn)
	if err != nil {
		return ContextoActor{}, err
	}

	instantanea := InstantaneaContextoActor{
		VinculoRef: documento.ContextoActorRef, VinculoVersion: documento.ContextoVersion,
		CuentaRef: documento.CuentaRef, PersonaRef: documento.PersonaRef,
		PersonaVersion: documento.PersonaVersion, PerfilActivoRef: documento.PerfilActivoRef,
		PerfilVersion: documento.PerfilVersion, Estado: EstadoVinculoContextoActor(documento.Estado),
		VigenteDesde: vigenteDesde, VigenteHasta: vigenteHasta,
		Vinculos: make([]VinculoReferenciaContextoActor, 0, len(documento.Vinculos)),
	}
	for _, vinculo := range documento.Vinculos {
		desde, errorDesde := parsearInstanteContextoActorCanonico(vinculo.VigenteDesde)
		hasta, errorHasta := parsearInstanteContextoActorCanonico(vinculo.VigenteHasta)
		if errorDesde != nil || errorHasta != nil {
			return ContextoActor{}, ErrRepresentacionContextoActorV1Invalida
		}
		instantanea.Vinculos = append(instantanea.Vinculos, VinculoReferenciaContextoActor{
			VinculoRef: vinculo.VinculoRef, Version: vinculo.Version,
			Tipo: TipoReferenciaContextoActor(vinculo.Tipo), Referencia: vinculo.Referencia,
			Estado: EstadoVinculoContextoActor(vinculo.Estado), VigenteDesde: desde, VigenteHasta: hasta,
		})
	}
	cuenta := CuentaAutenticadaContextoActor{
		CuentaRef: documento.CuentaRef, Metodo: documento.Metodo, Garantia: documento.Garantia,
	}
	return NuevoContextoActor(cuenta, instantanea, resueltoEn)
}

func parsearInstanteContextoActorCanonico(valor string) (time.Time, error) {
	instante, err := time.Parse(formatoInstanteContextoActorCanonico, valor)
	if err != nil || formatearInstanteContextoActorCanonico(instante) != valor {
		return time.Time{}, ErrRepresentacionContextoActorV1Invalida
	}
	return instante, nil
}

func formatearInstanteContextoActorCanonico(instante time.Time) string {
	return instante.UTC().Format(formatoInstanteContextoActorCanonico)
}

func validarJSONContextoActorCanonicoSinDuplicados(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	primero, err := decodificador.Token()
	if err != nil || primero != json.Delim('{') {
		return ErrRepresentacionContextoActorV1Invalida
	}
	if err = consumirCompuestoJSONContextoActorCanonico(decodificador, json.Delim('{'), 1); err != nil {
		return err
	}
	return exigirFinJSONContextoActorCanonico(decodificador)
}

func consumirCompuestoJSONContextoActorCanonico(decodificador *json.Decoder, apertura json.Delim, profundidad int) error {
	if profundidad > profundidadMaximaJSONContextoActorCanonico {
		return ErrRepresentacionContextoActorV1Invalida
	}
	claves := make(map[string]struct{})
	elementos := 0
	for decodificador.More() {
		elementos++
		if (apertura == json.Delim('{') && elementos > camposMaximosObjetoJSONContextoActorCanonico) ||
			(apertura == json.Delim('[') && elementos > maximoVinculosContextoActor) {
			return ErrRepresentacionContextoActorV1Invalida
		}
		if apertura == json.Delim('{') {
			token, err := decodificador.Token()
			clave, valida := token.(string)
			if err != nil || !valida {
				return ErrRepresentacionContextoActorV1Invalida
			}
			if _, repetida := claves[clave]; repetida {
				return ErrRepresentacionContextoActorV1Invalida
			}
			claves[clave] = struct{}{}
		}
		valor, err := decodificador.Token()
		if err != nil {
			return ErrRepresentacionContextoActorV1Invalida
		}
		if delimitador, compuesto := valor.(json.Delim); compuesto {
			if delimitador != json.Delim('{') && delimitador != json.Delim('[') {
				return ErrRepresentacionContextoActorV1Invalida
			}
			if err = consumirCompuestoJSONContextoActorCanonico(decodificador, delimitador, profundidad+1); err != nil {
				return err
			}
		}
	}
	cierre, err := decodificador.Token()
	if err != nil || (apertura == json.Delim('{') && cierre != json.Delim('}')) ||
		(apertura == json.Delim('[') && cierre != json.Delim(']')) {
		return ErrRepresentacionContextoActorV1Invalida
	}
	return nil
}

func exigirFinJSONContextoActorCanonico(decodificador *json.Decoder) error {
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return ErrRepresentacionContextoActorV1Invalida
	}
	return nil
}
