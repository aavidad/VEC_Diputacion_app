package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
)

const (
	EsquemaManifiestoProcedenciaContextoActorV1       = "vec.contexto-actor.procedencia-manifiesto.v1"
	TamanoMaximoManifiestoProcedenciaContextoActorV1  = 64 * 1024
	longitudMinimaReferenciaComponenteContextoActorV1 = 22
)

var ErrManifiestoProcedenciaContextoActorV1Invalido = errors.New(
	"vec: manifiesto de procedencia de contexto de actor v1 invalido",
)

type AutoridadProcedenciaContextoActorV1 string

const (
	AutoridadProcedenciaContextoActorMaestraAcreditadaV1 AutoridadProcedenciaContextoActorV1 = "autoridad_maestra_acreditada"
	AutoridadProcedenciaContextoActorNoAutoritativaV1    AutoridadProcedenciaContextoActorV1 = "no_autoritativa"
)

func (a AutoridadProcedenciaContextoActorV1) Valida() bool {
	return a == AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		a == AutoridadProcedenciaContextoActorNoAutoritativaV1
}

type AcreditacionProcedenciaComponenteContextoActorV1 struct {
	ProcedenciaRef          string                              `json:"procedencia_ref"`
	ProcedenciaVersion      uint64                              `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string                              `json:"procedencia_huella_sha256"`
	ProcedenciaAutoridad    AutoridadProcedenciaContextoActorV1 `json:"procedencia_autoridad"`
}

func (a AcreditacionProcedenciaComponenteContextoActorV1) validar(
	autoridad AutoridadProcedenciaContextoActorV1,
) bool {
	return referenciaComponenteContextoActorV1Valida(a.ProcedenciaRef, "prc_") &&
		a.ProcedenciaVersion > 0 && huellaContextoActorV1Valida(a.ProcedenciaHuellaSHA256) &&
		a.ProcedenciaAutoridad == autoridad
}

type ProcedenciaCuentaContextoActorV1 struct {
	CuentaRef string `json:"cuenta_ref"`
	Version   uint64 `json:"version"`
	AcreditacionProcedenciaComponenteContextoActorV1
}

type ProcedenciaPersonaContextoActorV1 struct {
	PersonaRef string `json:"persona_ref"`
	Version    uint64 `json:"version"`
	AcreditacionProcedenciaComponenteContextoActorV1
}

type ProcedenciaPerfilContextoActorV1 struct {
	PerfilRef string `json:"perfil_ref"`
	Version   uint64 `json:"version"`
	AcreditacionProcedenciaComponenteContextoActorV1
}

type ProcedenciaVinculoContextoActorV1 struct {
	VinculoRef string `json:"vinculo_ref"`
	Version    uint64 `json:"version"`
	AcreditacionProcedenciaComponenteContextoActorV1
}

type ProcedenciaVinculoReferenciaContextoActorV1 struct {
	VinculoRef string                      `json:"vinculo_ref"`
	Version    uint64                      `json:"version"`
	Tipo       TipoReferenciaContextoActor `json:"tipo"`
	Referencia string                      `json:"referencia"`
	AcreditacionProcedenciaComponenteContextoActorV1
}

type ManifiestoProcedenciaContextoActorV1 struct {
	Esquema           string                                        `json:"esquema"`
	AutoridadEfectiva AutoridadProcedenciaContextoActorV1           `json:"autoridad_efectiva"`
	Cuenta            ProcedenciaCuentaContextoActorV1              `json:"cuenta"`
	Persona           ProcedenciaPersonaContextoActorV1             `json:"persona"`
	Perfil            ProcedenciaPerfilContextoActorV1              `json:"perfil"`
	Contexto          ProcedenciaVinculoContextoActorV1             `json:"contexto"`
	Vinculos          []ProcedenciaVinculoReferenciaContextoActorV1 `json:"vinculos"`
}

func (m ManifiestoProcedenciaContextoActorV1) Validar() error {
	if m.Esquema != EsquemaManifiestoProcedenciaContextoActorV1 ||
		!m.AutoridadEfectiva.Valida() ||
		!referenciaComponenteContextoActorV1Valida(m.Cuenta.CuentaRef, "cta_") ||
		m.Cuenta.Version == 0 || !m.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1.validar(m.AutoridadEfectiva) ||
		!referenciaComponenteContextoActorV1Valida(m.Persona.PersonaRef, "per_") ||
		m.Persona.Version == 0 || !m.Persona.AcreditacionProcedenciaComponenteContextoActorV1.validar(m.AutoridadEfectiva) ||
		!referenciaComponenteContextoActorV1Valida(m.Perfil.PerfilRef, "prf_") ||
		m.Perfil.Version == 0 || !m.Perfil.AcreditacionProcedenciaComponenteContextoActorV1.validar(m.AutoridadEfectiva) ||
		!referenciaComponenteContextoActorV1Valida(m.Contexto.VinculoRef, "vca_") ||
		m.Contexto.Version == 0 || !m.Contexto.AcreditacionProcedenciaComponenteContextoActorV1.validar(m.AutoridadEfectiva) ||
		m.Vinculos == nil || len(m.Vinculos) > 128 {
		return ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	vinculos := make(map[string]struct{}, len(m.Vinculos))
	tipos := make(map[TipoReferenciaContextoActor]struct{}, len(m.Vinculos))
	for _, vinculo := range m.Vinculos {
		prefijo := ""
		switch vinculo.Tipo {
		case TipoReferenciaContextoActorCandidato:
			prefijo = "can_"
		case TipoReferenciaContextoActorEmpleado:
			prefijo = "emp_"
		default:
			return ErrManifiestoProcedenciaContextoActorV1Invalido
		}
		if !referenciaComponenteContextoActorV1Valida(vinculo.VinculoRef, "vin_") ||
			vinculo.Version == 0 || !referenciaComponenteContextoActorV1Valida(vinculo.Referencia, prefijo) ||
			!vinculo.AcreditacionProcedenciaComponenteContextoActorV1.validar(m.AutoridadEfectiva) {
			return ErrManifiestoProcedenciaContextoActorV1Invalido
		}
		if _, existe := vinculos[vinculo.VinculoRef]; existe {
			return ErrManifiestoProcedenciaContextoActorV1Invalido
		}
		vinculos[vinculo.VinculoRef] = struct{}{}
		if _, existe := tipos[vinculo.Tipo]; existe {
			return ErrManifiestoProcedenciaContextoActorV1Invalido
		}
		tipos[vinculo.Tipo] = struct{}{}
	}
	if !sort.SliceIsSorted(m.Vinculos, func(i, j int) bool {
		a, b := m.Vinculos[i], m.Vinculos[j]
		if a.Tipo != b.Tipo {
			return a.Tipo < b.Tipo
		}
		if a.Referencia != b.Referencia {
			return a.Referencia < b.Referencia
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.VinculoRef < b.VinculoRef
	}) {
		return ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	return nil
}

func (m ManifiestoProcedenciaContextoActorV1) ValidarParaContexto(
	contexto ContextoActor,
) error {
	if m.Validar() != nil || contexto.Validar() != nil ||
		m.Cuenta.CuentaRef != contexto.Instantanea.CuentaRef ||
		m.Cuenta.Version != contexto.Instantanea.CuentaVersion ||
		m.Persona.PersonaRef != contexto.PersonaRef || m.Persona.Version != contexto.Instantanea.PersonaVersion ||
		m.Perfil.PerfilRef != contexto.PerfilActivoRef || m.Perfil.Version != contexto.Instantanea.PerfilVersion ||
		m.Contexto.VinculoRef != contexto.Instantanea.VinculoRef || m.Contexto.Version != contexto.Instantanea.VinculoVersion ||
		len(m.Vinculos) != len(contexto.Instantanea.Vinculos) {
		return ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	for i := range m.Vinculos {
		a, b := m.Vinculos[i], contexto.Instantanea.Vinculos[i]
		if a.VinculoRef != b.VinculoRef || a.Version != b.Version ||
			a.Tipo != b.Tipo || a.Referencia != b.Referencia {
			return ErrManifiestoProcedenciaContextoActorV1Invalido
		}
	}
	return nil
}

func (m ManifiestoProcedenciaContextoActorV1) RepresentacionCanonicaV1() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	contenido, err := json.Marshal(m)
	if err != nil || len(contenido) == 0 || len(contenido) > TamanoMaximoManifiestoProcedenciaContextoActorV1 {
		return nil, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	return append([]byte(nil), contenido...), nil
}

func RehidratarManifiestoProcedenciaContextoActorV1(
	contenido []byte,
) (ManifiestoProcedenciaContextoActorV1, error) {
	if len(contenido) == 0 || len(contenido) > TamanoMaximoManifiestoProcedenciaContextoActorV1 {
		return ManifiestoProcedenciaContextoActorV1{}, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var manifiesto ManifiestoProcedenciaContextoActorV1
	if err := decodificador.Decode(&manifiesto); err != nil {
		return ManifiestoProcedenciaContextoActorV1{}, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return ManifiestoProcedenciaContextoActorV1{}, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	canonicos, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil || !bytes.Equal(canonicos, contenido) {
		return ManifiestoProcedenciaContextoActorV1{}, ErrManifiestoProcedenciaContextoActorV1Invalido
	}
	return manifiesto, nil
}

func HuellaSHA256ManifiestoProcedenciaContextoActorV1(contenido []byte) (string, error) {
	if _, err := RehidratarManifiestoProcedenciaContextoActorV1(contenido); err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func referenciaComponenteContextoActorV1Valida(valor, prefijo string) bool {
	if len(prefijo) == 0 || len(valor) < len(prefijo)+longitudMinimaReferenciaComponenteContextoActorV1 ||
		len(valor) > len(prefijo)+128 || valor[:len(prefijo)] != prefijo {
		return false
	}
	for i := len(prefijo); i < len(valor); i++ {
		c := valor[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func huellaContextoActorV1Valida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for i := range valor {
		if valor[i] < '0' || (valor[i] > '9' && valor[i] < 'a') || valor[i] > 'f' {
			return false
		}
	}
	return true
}
