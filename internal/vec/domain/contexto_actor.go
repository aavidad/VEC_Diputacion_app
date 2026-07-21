package domain

import (
	"errors"
	"sort"
	"time"
)

var (
	ErrSolicitudContextoActorInvalida   = errors.New("vec: solicitud de contexto de actor invalida")
	ErrInstantaneaContextoActorInvalida = errors.New("vec: instantanea de contexto de actor invalida")
	ErrContextoActorInvalido            = errors.New("vec: contexto de actor invalido")
	ErrContextoActorNoResuelto          = errors.New("vec: contexto de actor no resuelto")
)

const (
	longitudMinimaTokenContextoActor = 22
	longitudMaximaTokenContextoActor = 128
	maximoVinculosContextoActor      = 128
)

// EstadoVinculoContextoActor es una lista cerrada. Una version revocada se
// conserva como evidencia historica, pero nunca puede producir un contexto.
type EstadoVinculoContextoActor string

const (
	EstadoVinculoContextoActorActivo   EstadoVinculoContextoActor = "activo"
	EstadoVinculoContextoActorRevocado EstadoVinculoContextoActor = "revocado"
)

func (e EstadoVinculoContextoActor) Valido() bool {
	return e == EstadoVinculoContextoActorActivo || e == EstadoVinculoContextoActorRevocado
}

// TipoReferenciaContextoActor identifica referencias opacas que pertenecen a
// otros modulos. El nucleo no conoce los datos de candidato ni de empleado.
type TipoReferenciaContextoActor string

const (
	TipoReferenciaContextoActorCandidato TipoReferenciaContextoActor = "candidato"
	TipoReferenciaContextoActorEmpleado  TipoReferenciaContextoActor = "empleado"
)

func (t TipoReferenciaContextoActor) Valido() bool {
	return t == TipoReferenciaContextoActorCandidato || t == TipoReferenciaContextoActorEmpleado
}

// CuentaAutenticadaContextoActor contiene exclusivamente el identificador
// tecnico opaco y la garantia ya acreditada por la frontera de identidad. No
// admite DNI, correo, nombre, roles, permisos ni atributos declarados.
type CuentaAutenticadaContextoActor struct {
	CuentaRef string        `json:"cuenta_ref"`
	Metodo    AuthMethod    `json:"metodo"`
	Garantia  AuthAssurance `json:"garantia"`
}

func (c CuentaAutenticadaContextoActor) Validar() error {
	if !referenciaOpacaContextoActorValida(c.CuentaRef, "cta_") ||
		!c.Metodo.Valido() || !c.Garantia.Valida() {
		return ErrSolicitudContextoActorInvalida
	}
	return nil
}

// SolicitudContextoActor exige un perfil concreto. Un perfil vacio no significa
// "el habitual" y nunca se obtiene seleccionando el primero disponible.
type SolicitudContextoActor struct {
	Cuenta          CuentaAutenticadaContextoActor `json:"cuenta"`
	PerfilActivoRef string                         `json:"perfil_activo_ref"`
}

func (s SolicitudContextoActor) Validar() error {
	if s.Cuenta.Validar() != nil || !referenciaOpacaContextoActorValida(s.PerfilActivoRef, "prf_") {
		return ErrSolicitudContextoActorInvalida
	}
	return nil
}

// VinculoReferenciaContextoActor enlaza la persona canonica con una referencia
// de modulo. Cada enlace tiene identidad, version, estado y vigencia propios.
type VinculoReferenciaContextoActor struct {
	VinculoRef   string                      `json:"vinculo_ref"`
	Version      uint64                      `json:"version"`
	Tipo         TipoReferenciaContextoActor `json:"tipo"`
	Referencia   string                      `json:"referencia"`
	Estado       EstadoVinculoContextoActor  `json:"estado"`
	VigenteDesde time.Time                   `json:"vigente_desde"`
	VigenteHasta time.Time                   `json:"vigente_hasta"`
}

func (v VinculoReferenciaContextoActor) Validar() error {
	prefijoReferencia := ""
	switch v.Tipo {
	case TipoReferenciaContextoActorCandidato:
		prefijoReferencia = "can_"
	case TipoReferenciaContextoActorEmpleado:
		prefijoReferencia = "emp_"
	default:
		return ErrInstantaneaContextoActorInvalida
	}
	if !referenciaOpacaContextoActorValida(v.VinculoRef, "vin_") || v.Version == 0 ||
		!referenciaOpacaContextoActorValida(v.Referencia, prefijoReferencia) || !v.Estado.Valido() ||
		!instanteContextoActorCanonico(v.VigenteDesde) || !instanteContextoActorCanonico(v.VigenteHasta) ||
		!v.VigenteHasta.After(v.VigenteDesde) {
		return ErrInstantaneaContextoActorInvalida
	}
	return nil
}

func (v VinculoReferenciaContextoActor) VigenteEn(instante time.Time) bool {
	return instanteContextoActorCanonico(instante) && v.Validar() == nil &&
		v.Estado == EstadoVinculoContextoActorActivo &&
		!instante.Before(v.VigenteDesde) && instante.Before(v.VigenteHasta)
}

// InstantaneaContextoActor es el resultado versionado de unir cuenta, persona y
// perfil en el servidor. La fuente devuelve todas las coincidencias y la capa de
// aplicacion exige exactamente una; esta estructura no concede por si sola.
type InstantaneaContextoActor struct {
	VinculoRef     string `json:"vinculo_ref"`
	VinculoVersion uint64 `json:"vinculo_version"`
	CuentaRef      string `json:"cuenta_ref"`
	// CuentaVersion vale cero exclusivamente en snapshots heredados cuyo canon
	// V1 no comprometia esa revision. Toda capacidad durable V2 exige un valor
	// positivo y lo liga tanto al canon como al manifiesto de procedencia.
	CuentaVersion   uint64                           `json:"cuenta_version,omitempty"`
	PersonaRef      string                           `json:"persona_ref"`
	PersonaVersion  uint64                           `json:"persona_version"`
	PerfilActivoRef string                           `json:"perfil_activo_ref"`
	PerfilVersion   uint64                           `json:"perfil_version"`
	Estado          EstadoVinculoContextoActor       `json:"estado"`
	VigenteDesde    time.Time                        `json:"vigente_desde"`
	VigenteHasta    time.Time                        `json:"vigente_hasta"`
	Vinculos        []VinculoReferenciaContextoActor `json:"vinculos"`
}

func (i InstantaneaContextoActor) Validar() error {
	if !referenciaOpacaContextoActorValida(i.VinculoRef, "vca_") || i.VinculoVersion == 0 ||
		!referenciaOpacaContextoActorValida(i.CuentaRef, "cta_") ||
		!referenciaOpacaContextoActorValida(i.PersonaRef, "per_") || i.PersonaVersion == 0 ||
		!referenciaOpacaContextoActorValida(i.PerfilActivoRef, "prf_") || i.PerfilVersion == 0 ||
		!i.Estado.Valido() || !instanteContextoActorCanonico(i.VigenteDesde) ||
		!instanteContextoActorCanonico(i.VigenteHasta) || !i.VigenteHasta.After(i.VigenteDesde) ||
		len(i.Vinculos) > maximoVinculosContextoActor {
		return ErrInstantaneaContextoActorInvalida
	}

	vinculos := make(map[string]struct{}, len(i.Vinculos))
	referencias := make(map[string]struct{}, len(i.Vinculos))
	tipos := make(map[TipoReferenciaContextoActor]struct{}, len(i.Vinculos))
	for _, vinculo := range i.Vinculos {
		if vinculo.Validar() != nil {
			return ErrInstantaneaContextoActorInvalida
		}
		if _, repetido := vinculos[vinculo.VinculoRef]; repetido {
			return ErrInstantaneaContextoActorInvalida
		}
		vinculos[vinculo.VinculoRef] = struct{}{}
		claveReferencia := string(vinculo.Tipo) + "\x00" + vinculo.Referencia
		if _, repetida := referencias[claveReferencia]; repetida {
			return ErrInstantaneaContextoActorInvalida
		}
		referencias[claveReferencia] = struct{}{}
		if _, repetido := tipos[vinculo.Tipo]; repetido {
			// El contexto entrega la referencia canonica de cada modulo. Dos
			// candidatos o dos empleados volverian a trasladar la ambiguedad al
			// consumidor y permitirian que este eligiese el primero.
			return ErrInstantaneaContextoActorInvalida
		}
		tipos[vinculo.Tipo] = struct{}{}
	}
	return nil
}

func (i InstantaneaContextoActor) VigenteEn(instante time.Time) bool {
	return instanteContextoActorCanonico(instante) && i.Validar() == nil &&
		i.Estado == EstadoVinculoContextoActorActivo &&
		!instante.Before(i.VigenteDesde) && instante.Before(i.VigenteHasta)
}

// ClonarCanonica devuelve una copia defensiva y ordena los enlaces sin elegir
// uno de ellos ni eliminar duplicados silenciosamente.
func (i InstantaneaContextoActor) ClonarCanonica() (InstantaneaContextoActor, error) {
	if err := i.Validar(); err != nil {
		return InstantaneaContextoActor{}, err
	}
	copia := i
	copia.Vinculos = append([]VinculoReferenciaContextoActor(nil), i.Vinculos...)
	sort.Slice(copia.Vinculos, func(a, b int) bool {
		primero, segundo := copia.Vinculos[a], copia.Vinculos[b]
		if primero.Tipo != segundo.Tipo {
			return primero.Tipo < segundo.Tipo
		}
		if primero.Referencia != segundo.Referencia {
			return primero.Referencia < segundo.Referencia
		}
		if primero.Version != segundo.Version {
			return primero.Version < segundo.Version
		}
		return primero.VinculoRef < segundo.VinculoRef
	})
	return copia, nil
}

// ContextoActor es la proyeccion consumible por los casos de uso. Principal usa
// la persona canonica como sujeto; cuenta y versiones permanecen en la
// instantanea para auditoria y revalidacion posteriores.
type ContextoActor struct {
	Principal       Principal                `json:"principal"`
	PerfilActivoRef string                   `json:"perfil_activo_ref"`
	PersonaRef      string                   `json:"persona_ref"`
	Instantanea     InstantaneaContextoActor `json:"instantanea"`
	ResueltoEn      time.Time                `json:"resuelto_en"`
}

func NuevoContextoActor(
	cuenta CuentaAutenticadaContextoActor,
	instantanea InstantaneaContextoActor,
	resueltoEn time.Time,
) (ContextoActor, error) {
	if cuenta.Validar() != nil || !instanteContextoActorCanonico(resueltoEn) ||
		instantanea.Validar() != nil || instantanea.CuentaRef != cuenta.CuentaRef ||
		!instantanea.VigenteEn(resueltoEn) {
		return ContextoActor{}, ErrContextoActorInvalido
	}
	for _, vinculo := range instantanea.Vinculos {
		if !vinculo.VigenteEn(resueltoEn) {
			return ContextoActor{}, ErrContextoActorInvalido
		}
	}
	canonica, err := instantanea.ClonarCanonica()
	if err != nil {
		return ContextoActor{}, ErrContextoActorInvalido
	}
	resultado := ContextoActor{
		Principal: Principal{
			ID:            canonica.PersonaRef,
			AuthMethod:    cuenta.Metodo,
			AuthAssurance: cuenta.Garantia,
		},
		PerfilActivoRef: canonica.PerfilActivoRef,
		PersonaRef:      canonica.PersonaRef,
		Instantanea:     canonica,
		ResueltoEn:      resueltoEn,
	}
	if resultado.Validar() != nil {
		return ContextoActor{}, ErrContextoActorInvalido
	}
	return resultado, nil
}

func (c ContextoActor) Validar() error {
	if c.Principal.Validate() != nil || c.Principal.ID != c.PersonaRef ||
		c.Principal.DisplayName != "" || c.Principal.Email != "" ||
		len(c.Principal.Roles) != 0 || len(c.Principal.Permissions) != 0 || len(c.Principal.Attributes) != 0 ||
		!referenciaOpacaContextoActorValida(c.PersonaRef, "per_") ||
		!referenciaOpacaContextoActorValida(c.PerfilActivoRef, "prf_") ||
		!instanteContextoActorCanonico(c.ResueltoEn) || c.Instantanea.Validar() != nil ||
		c.Instantanea.PersonaRef != c.PersonaRef || c.Instantanea.PerfilActivoRef != c.PerfilActivoRef ||
		!c.Instantanea.VigenteEn(c.ResueltoEn) {
		return ErrContextoActorInvalido
	}
	for _, vinculo := range c.Instantanea.Vinculos {
		if !vinculo.VigenteEn(c.ResueltoEn) {
			return ErrContextoActorInvalido
		}
	}
	return nil
}

func (c ContextoActor) Clonar() (ContextoActor, error) {
	if err := c.Validar(); err != nil {
		return ContextoActor{}, err
	}
	copia := c
	instantanea, err := c.Instantanea.ClonarCanonica()
	if err != nil {
		return ContextoActor{}, ErrContextoActorInvalido
	}
	copia.Instantanea = instantanea
	return copia, nil
}

// Referencias devuelve copias de las referencias opacas vigentes de un tipo.
// Una clase desconocida nunca se interpreta como "todas".
func (c ContextoActor) Referencias(tipo TipoReferenciaContextoActor) ([]string, error) {
	if c.Validar() != nil || !tipo.Valido() {
		return nil, ErrContextoActorInvalido
	}
	resultado := make([]string, 0)
	for _, vinculo := range c.Instantanea.Vinculos {
		if vinculo.Tipo == tipo {
			resultado = append(resultado, vinculo.Referencia)
		}
	}
	return append([]string(nil), resultado...), nil
}

func referenciaOpacaContextoActorValida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+longitudMinimaTokenContextoActor ||
		len(valor) > len(prefijo)+longitudMaximaTokenContextoActor ||
		len(prefijo) == 0 || len(valor) <= len(prefijo) || valor[:len(prefijo)] != prefijo {
		return false
	}
	for indice := len(prefijo); indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func instanteContextoActorCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}
