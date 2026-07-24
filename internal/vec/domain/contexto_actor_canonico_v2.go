package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

const (
	EsquemaRepresentacionContextoActorV2      = "vec.contexto-actor.vinculado.v2"
	TamanoMaximoRepresentacionContextoActorV2 = TamanoMaximoRepresentacionContextoActorV1
)

var ErrRepresentacionContextoActorV2Invalida = errors.New(
	"vec: representacion canonica de contexto de actor v2 invalida",
)

type vinculoReferenciaContextoActorCanonicoV2 = vinculoReferenciaContextoActorCanonico

type contextoActorCanonicoV2 struct {
	Esquema          string                                     `json:"esquema"`
	PrincipalRef     string                                     `json:"principal_ref"`
	Metodo           AuthMethod                                 `json:"metodo"`
	Garantia         AuthAssurance                              `json:"garantia"`
	PerfilActivoRef  string                                     `json:"perfil_activo_ref"`
	PersonaRef       string                                     `json:"persona_ref"`
	ContextoActorRef string                                     `json:"contexto_actor_ref"`
	ContextoVersion  uint64                                     `json:"contexto_version"`
	CuentaRef        string                                     `json:"cuenta_ref"`
	CuentaVersion    uint64                                     `json:"cuenta_version"`
	PersonaVersion   uint64                                     `json:"persona_version"`
	PerfilVersion    uint64                                     `json:"perfil_version"`
	Estado           string                                     `json:"estado"`
	VigenteDesde     string                                     `json:"vigente_desde"`
	VigenteHasta     string                                     `json:"vigente_hasta"`
	ResueltoEn       string                                     `json:"resuelto_en"`
	Vinculos         []vinculoReferenciaContextoActorCanonicoV2 `json:"vinculos"`
}

// RepresentacionCanonicaVinculadaV2 compromete tambien la revision exacta de
// la proyeccion de cuenta. No acepta snapshots V1 heredados, que representan
// CuentaVersion con cero porque esa revision no formaba parte de su preimagen.
func (c ContextoActor) RepresentacionCanonicaVinculadaV2() ([]byte, error) {
	canonica, err := c.Clonar()
	if err != nil || canonica.Instantanea.CuentaVersion == 0 {
		return nil, ErrContextoActorInvalido
	}
	documento := documentoContextoActorCanonicoV2(canonica)
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) == 0 || len(contenido) > TamanoMaximoRepresentacionContextoActorV2 {
		return nil, ErrContextoActorInvalido
	}
	return append([]byte(nil), contenido...), nil
}

func documentoContextoActorCanonicoV2(canonica ContextoActor) contextoActorCanonicoV2 {
	documento := contextoActorCanonicoV2{
		Esquema:      EsquemaRepresentacionContextoActorV2,
		PrincipalRef: canonica.Principal.ID, Metodo: canonica.Principal.AuthMethod,
		Garantia: canonica.Principal.AuthAssurance, PerfilActivoRef: canonica.PerfilActivoRef,
		PersonaRef: canonica.PersonaRef, ContextoActorRef: canonica.Instantanea.VinculoRef,
		ContextoVersion: canonica.Instantanea.VinculoVersion, CuentaRef: canonica.Instantanea.CuentaRef,
		CuentaVersion: canonica.Instantanea.CuentaVersion, PersonaVersion: canonica.Instantanea.PersonaVersion,
		PerfilVersion: canonica.Instantanea.PerfilVersion, Estado: string(canonica.Instantanea.Estado),
		VigenteDesde: formatearInstanteContextoActorCanonico(canonica.Instantanea.VigenteDesde),
		VigenteHasta: formatearInstanteContextoActorCanonico(canonica.Instantanea.VigenteHasta),
		ResueltoEn:   formatearInstanteContextoActorCanonico(canonica.ResueltoEn),
		Vinculos:     make([]vinculoReferenciaContextoActorCanonicoV2, 0, len(canonica.Instantanea.Vinculos)),
	}
	for _, vinculo := range canonica.Instantanea.Vinculos {
		documento.Vinculos = append(documento.Vinculos, vinculoReferenciaContextoActorCanonicoV2{
			VinculoRef: vinculo.VinculoRef, Version: vinculo.Version, Tipo: string(vinculo.Tipo),
			Referencia: vinculo.Referencia, Estado: string(vinculo.Estado),
			VigenteDesde: formatearInstanteContextoActorCanonico(vinculo.VigenteDesde),
			VigenteHasta: formatearInstanteContextoActorCanonico(vinculo.VigenteHasta),
		})
	}
	return documento
}

// HuellaSHA256VinculadaV2 identifica los bytes V2 que incluyen CuentaVersion.
func (c ContextoActor) HuellaSHA256VinculadaV2() (string, error) {
	contenido, err := c.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		return "", ErrContextoActorInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// RehidratarContextoActorVinculadoV2 solo acepta el documento V2 canónico y
// exige una version de cuenta positiva antes de reconstruir la capacidad.
func RehidratarContextoActorVinculadoV2(contenido []byte) (ContextoActor, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoRepresentacionContextoActorV2 {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}
	if err := validarJSONContextoActorCanonicoSinDuplicados(contenido); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}

	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var documento contextoActorCanonicoV2
	if err := decodificador.Decode(&documento); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}
	if err := exigirFinJSONContextoActorCanonico(decodificador); err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}
	resultado, err := rehidratarDocumentoContextoActorV2(documento)
	if err != nil {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}
	canonicos, err := resultado.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonicos, contenido) {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
	}
	return resultado, nil
}

func rehidratarDocumentoContextoActorV2(documento contextoActorCanonicoV2) (ContextoActor, error) {
	if documento.Esquema != EsquemaRepresentacionContextoActorV2 ||
		documento.PrincipalRef != documento.PersonaRef || documento.CuentaVersion == 0 {
		return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
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
		CuentaRef: documento.CuentaRef, CuentaVersion: documento.CuentaVersion,
		PersonaRef: documento.PersonaRef, PersonaVersion: documento.PersonaVersion,
		PerfilActivoRef: documento.PerfilActivoRef, PerfilVersion: documento.PerfilVersion,
		Estado:       EstadoVinculoContextoActor(documento.Estado),
		VigenteDesde: vigenteDesde, VigenteHasta: vigenteHasta,
		Vinculos: make([]VinculoReferenciaContextoActor, 0, len(documento.Vinculos)),
	}
	for _, vinculo := range documento.Vinculos {
		desde, errorDesde := parsearInstanteContextoActorCanonico(vinculo.VigenteDesde)
		hasta, errorHasta := parsearInstanteContextoActorCanonico(vinculo.VigenteHasta)
		if errorDesde != nil || errorHasta != nil {
			return ContextoActor{}, ErrRepresentacionContextoActorV2Invalida
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
