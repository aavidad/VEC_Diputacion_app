package ports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrCorreosNoAutenticado = errors.New("usuarios correos: no autenticado")
	ErrCorreosProhibido     = errors.New("usuarios correos: prohibido")
	ErrCorreosConflicto     = errors.New("usuarios correos: conflicto")
	ErrCorreosInvalidos     = errors.New("usuarios correos: peticion invalida")
	ErrCorreosNoDisponible  = errors.New("usuarios correos: no disponible")
	ErrCorreosLimite        = errors.New("usuarios correos: limite alcanzado")
)

const (
	FinalidadCorreosPropios          = "finalidad:usuarios:correos-propios:v1"
	MaxReenviosCorreoPorHora         = 3
	MaxIntentosCodigoCorreo          = 5
	AccionConsultarCorreos           = "vec.correos.consultar"
	AccionAnadirCorreo               = "vec.correos.anadir"
	AccionReenviarCorreo             = "vec.correos.reenviar"
	AccionVerificarCorreo            = "vec.correos.verificar"
	AccionActivarCorreo              = "vec.correos.activar"
	AccionRetirarCorreo              = "vec.correos.retirar"
	AudienciaConsultarCorreosInterna = "vec_usuarios.correos.consultar.interna_corporativa.v1"
	AudienciaAnadirCorreoInterna     = "vec_usuarios.correos.anadir.interna_corporativa.v1"
	AudienciaReenviarCorreoInterna   = "vec_usuarios.correos.reenviar.interna_corporativa.v1"
	AudienciaVerificarCorreoInterna  = "vec_usuarios.correos.verificar.interna_corporativa.v1"
	AudienciaActivarCorreoInterna    = "vec_usuarios.correos.activar.interna_corporativa.v1"
	AudienciaRetirarCorreoInterna    = "vec_usuarios.correos.retirar.interna_corporativa.v1"
	AudienciaConsultarCorreosExterna = "vec_usuarios.correos.consultar.externa_personal.v1"
	AudienciaAnadirCorreoExterna     = "vec_usuarios.correos.anadir.externa_personal.v1"
	AudienciaReenviarCorreoExterna   = "vec_usuarios.correos.reenviar.externa_personal.v1"
	AudienciaVerificarCorreoExterna  = "vec_usuarios.correos.verificar.externa_personal.v1"
	AudienciaActivarCorreoExterna    = "vec_usuarios.correos.activar.externa_personal.v1"
	AudienciaRetirarCorreoExterna    = "vec_usuarios.correos.retirar.externa_personal.v1"
)

func AudienciaCorreos(accion string, superficie vecdomain.SuperficieAutenticacionActorV1) (string, error) {
	switch superficie {
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		switch accion {
		case AccionConsultarCorreos:
			return AudienciaConsultarCorreosInterna, nil
		case AccionAnadirCorreo:
			return AudienciaAnadirCorreoInterna, nil
		case AccionReenviarCorreo:
			return AudienciaReenviarCorreoInterna, nil
		case AccionVerificarCorreo:
			return AudienciaVerificarCorreoInterna, nil
		case AccionActivarCorreo:
			return AudienciaActivarCorreoInterna, nil
		case AccionRetirarCorreo:
			return AudienciaRetirarCorreoInterna, nil
		}
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		switch accion {
		case AccionConsultarCorreos:
			return AudienciaConsultarCorreosExterna, nil
		case AccionAnadirCorreo:
			return AudienciaAnadirCorreoExterna, nil
		case AccionReenviarCorreo:
			return AudienciaReenviarCorreoExterna, nil
		case AccionVerificarCorreo:
			return AudienciaVerificarCorreoExterna, nil
		case AccionActivarCorreo:
			return AudienciaActivarCorreoExterna, nil
		case AccionRetirarCorreo:
			return AudienciaRetirarCorreoExterna, nil
		}
	}
	return "", ErrCorreosProhibido
}

type ProveedorMaterialCorreos interface {
	ProveerMaterialCorreos(context.Context, vecdomain.VinculoAutenticacionActorV2, MaterialCorreos) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenCorreos sólo se obtiene a partir de la persona canónica del actor.
// PersonaRef nunca forma parte de una petición HTTP.
type OrdenCorreos struct {
	actor      vecdomain.ContextoActor
	vinculo    vecdomain.VinculoAutenticacionActorV2
	superficie vecdomain.SuperficieAutenticacionActorV1
	proveedor  ProveedorMaterialCorreos
}

func cotejarIdentidadCorreo(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2) bool {
	if actor.Validar() != nil || (actor.Principal.AuthMethod != vecdomain.AuthMethodCertificate && actor.Principal.AuthMethod != vecdomain.AuthMethodDNIe) || actor.Instantanea.CuentaVersion == 0 {
		return false
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.CuentaPrivilegiada {
		return false
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	return err == nil && datos.PrincipalID == actor.PersonaRef && datos.PerfilActivoRef == actor.PerfilActivoRef &&
		datos.CuentaRef == actor.Instantanea.CuentaRef && datos.CuentaOrdinariaRef == actor.Instantanea.CuentaRef &&
		datos.MetodoObservado == actor.Principal.AuthMethod && datos.GarantiaObservada == actor.Principal.AuthAssurance &&
		datos.ContextoActorRef == actor.Instantanea.VinculoRef && datos.ContextoActorVersion == actor.Instantanea.VinculoVersion &&
		datos.ContextoActorCuentaVersion == actor.Instantanea.CuentaVersion && datos.ContextoActorHuellaSHA256 == huella
}

func cotejarOrdenCorreo(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficie vecdomain.SuperficieAutenticacionActorV1) bool {
	if !cotejarIdentidadCorreo(actor, vinculo) {
		return false
	}
	datos, _ := vinculo.Datos()
	return (superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1 || superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1) && datos.Superficie == superficie
}

func NuevaOrdenCorreos(actor vecdomain.ContextoActor, vinculo vecdomain.VinculoAutenticacionActorV2, superficieRuta vecdomain.SuperficieAutenticacionActorV1, proveedor ProveedorMaterialCorreos) (OrdenCorreos, error) {
	if proveedor == nil || !cotejarIdentidadCorreo(actor, vinculo) {
		return OrdenCorreos{}, ErrCorreosNoAutenticado
	}
	if !cotejarOrdenCorreo(actor, vinculo, superficieRuta) {
		return OrdenCorreos{}, ErrCorreosProhibido
	}
	copia, err := actor.Clonar()
	if err != nil {
		return OrdenCorreos{}, ErrCorreosNoAutenticado
	}
	return OrdenCorreos{actor: copia, vinculo: vinculo, superficie: superficieRuta, proveedor: proveedor}, nil
}

func (o OrdenCorreos) ContextoActor() (vecdomain.ContextoActor, error) {
	if o.proveedor == nil || !cotejarOrdenCorreo(o.actor, o.vinculo, o.superficie) {
		return vecdomain.ContextoActor{}, ErrCorreosNoAutenticado
	}
	return o.actor.Clonar()
}
func (o OrdenCorreos) Vinculo() (vecdomain.VinculoAutenticacionActorV2, error) {
	if o.proveedor == nil || !cotejarOrdenCorreo(o.actor, o.vinculo, o.superficie) {
		return vecdomain.VinculoAutenticacionActorV2{}, ErrCorreosNoAutenticado
	}
	return o.vinculo, nil
}
func (o OrdenCorreos) Superficie() (vecdomain.SuperficieAutenticacionActorV1, error) {
	if o.proveedor == nil || !cotejarOrdenCorreo(o.actor, o.vinculo, o.superficie) {
		return "", ErrCorreosNoAutenticado
	}
	return o.superficie, nil
}
func (o OrdenCorreos) Proveedor() ProveedorMaterialCorreos { return o.proveedor }

type PeticionCorreo struct {
	VersionEsperada uint64 `json:"version_esperada"`
	ClaveOperacion  string `json:"clave_operacion"`
	CorreoRef       string `json:"correo_ref,omitempty"`
	Direccion       string `json:"direccion,omitempty"`
	Codigo          string `json:"codigo,omitempty"`
	SustitutoRef    string `json:"sustituto_ref,omitempty"`
}

// La colección contiene HMAC dedicados y versionados de la preimagen completa.
// El activo se persiste; retenidos sirven exclusivamente para comparar replay
// tras rotación, sin reescribir el recibo ni ampliar el ámbito de autorización.
type HuellaSemanticaCorreo struct {
	ClaveRef string `json:"clave_ref"`
	Valor    string `json:"valor"`
}

type HuellasSemanticasCorreo struct {
	Activa    HuellaSemanticaCorreo   `json:"activa"`
	Retenidas []HuellaSemanticaCorreo `json:"retenidas"`
}

func (h HuellasSemanticasCorreo) Validar() bool {
	if len(h.Retenidas) > 8 {
		return false
	}
	vistas := make(map[string]bool, len(h.Retenidas)+1)
	for _, sello := range append([]HuellaSemanticaCorreo{h.Activa}, h.Retenidas...) {
		valor, err := hex.DecodeString(sello.Valor)
		if sello.ClaveRef == "" || vistas[sello.ClaveRef] || err != nil || len(valor) != 32 {
			return false
		}
		vistas[sello.ClaveRef] = true
	}
	return true
}

// Sellar acepta bytes sólo en memoria y nunca registra ni devuelve la
// preimagen. Usa HMAC-SHA256 con clave de huella semántica propia del módulo;
// no la clave de desafío ni de igualdad de direcciones. Retener generaciones
// válidas permite comparar la misma petición con su clave histórica.
type SelladorHuellaCorreos interface {
	SellarHuellaCorreo(context.Context, []byte) (HuellasSemanticasCorreo, error)
}

// Sólo lleva HMAC semánticos de la entrada. Nunca dirección ni código.
type MaterialCorreos struct {
	Superficie      vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
	PersonaRef      string                                   `json:"persona_ref"`
	PerfilRef       string                                   `json:"perfil_ref"`
	Accion          string                                   `json:"accion"`
	FinalidadRef    string                                   `json:"finalidad_ref"`
	VersionEsperada uint64                                   `json:"version_esperada"`
	ClaveOperacion  string                                   `json:"clave_operacion"`
	HuellasPeticion HuellasSemanticasCorreo                  `json:"huellas_peticion"`
	CorreoRef       string                                   `json:"correo_ref"`
	SustitutoRef    string                                   `json:"sustituto_ref"`
}

func (m MaterialCorreos) MarshalJSON() ([]byte, error) { return SerializarMaterialCorreos(m) }

// SerializarMaterialCorreos es la única preimagen V3/SQL p_material text.
// El proveedor V3 calcula material_sha256 de estos bytes exactos y el adaptador
// PG envía esos mismos bytes, sin reconstruir JSON ni incorporar secretos.
func SerializarMaterialCorreos(m MaterialCorreos) ([]byte, error) {
	if _, err := AudienciaCorreos(m.Accion, m.Superficie); err != nil || m.PersonaRef == "" || m.PerfilRef == "" || m.FinalidadRef != FinalidadCorreosPropios {
		return nil, ErrCorreosInvalidos
	}
	if m.Accion == AccionConsultarCorreos {
		if m.ClaveOperacion != "" || m.HuellasPeticion.Activa != (HuellaSemanticaCorreo{}) || len(m.HuellasPeticion.Retenidas) != 0 || m.CorreoRef != "" || m.SustitutoRef != "" || m.VersionEsperada != 0 {
			return nil, ErrCorreosInvalidos
		}
	} else if m.ClaveOperacion == "" || !m.HuellasPeticion.Validar() {
		return nil, ErrCorreosInvalidos
	}
	m.HuellasPeticion.Retenidas = append([]HuellaSemanticaCorreo{}, m.HuellasPeticion.Retenidas...)
	sort.Slice(m.HuellasPeticion.Retenidas, func(i, j int) bool {
		return m.HuellasPeticion.Retenidas[i].ClaveRef < m.HuellasPeticion.Retenidas[j].ClaveRef
	})
	huellaJSON := json.RawMessage("{}")
	if m.Accion != AccionConsultarCorreos {
		var err error
		huellaJSON, err = json.Marshal(m.HuellasPeticion)
		if err != nil {
			return nil, ErrCorreosInvalidos
		}
	}
	b, err := json.Marshal(struct {
		Superficie      vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
		PersonaRef      string                                   `json:"persona_ref"`
		PerfilRef       string                                   `json:"perfil_ref"`
		Accion          string                                   `json:"accion"`
		FinalidadRef    string                                   `json:"finalidad_ref"`
		VersionEsperada uint64                                   `json:"version_esperada"`
		ClaveOperacion  string                                   `json:"clave_operacion"`
		HuellasPeticion json.RawMessage                          `json:"huellas_peticion"`
		CorreoRef       string                                   `json:"correo_ref"`
		SustitutoRef    string                                   `json:"sustituto_ref"`
	}{m.Superficie, m.PersonaRef, m.PerfilRef, m.Accion, m.FinalidadRef, m.VersionEsperada, m.ClaveOperacion, huellaJSON, m.CorreoRef, m.SustitutoRef})
	if err != nil || len(b) > 8192 {
		return nil, ErrCorreosInvalidos
	}
	return b, nil
}

type VistaCorreos struct {
	PersonaRef string                `json:"persona_ref"`
	Version    uint64                `json:"version"`
	Correos    []domain.CorreoPropio `json:"correos"`
}

type ReciboCorreos struct {
	ReciboRef  string    `json:"recibo_ref"`
	PersonaRef string    `json:"persona_ref"`
	Accion     string    `json:"accion"`
	CorreoRef  string    `json:"correo_ref"`
	Version    uint64    `json:"version"`
	FechaUTC   time.Time `json:"fecha_utc"`
	Replay     bool      `json:"replay"`
}

// ReservaDesafio sólo circula en memoria hasta la transacción. El registro
// conserva huella/estado/clave_ref y escribe Desafio exclusivamente en outbox.
// La dirección cifrada debe ligar AAD a CorreoRef y VersionNueva; el sobre
// ContactoUsuario existente no cumple esa ligadura por sí solo.
type ReservaDesafio struct {
	DesafioRef   string
	Desafio      []byte
	HuellaCodigo []byte
	ClaveRef     string
	VenceUTC     time.Time
}

// El protector cifra con AAD que incorpora persona, correo_ref y versión del
// conjunto. La huella de igualdad debe ser HMAC separada para la unicidad;
// nunca SHA-256 simple de una dirección adivinable.
type SobreDireccionCorreo struct {
	CorreoRef      string
	Version        uint64
	ClaveRef       string
	Nonce          []byte
	Cifrado        []byte
	HuellaIgualdad []byte
}

type ProtectorDireccionCorreo interface {
	CifrarDireccionCorreo(context.Context, string, string, uint64, []byte) (SobreDireccionCorreo, error)
}

type PreparadorDesafioCorreo interface {
	// Debe usar CSPRNG >=128 bits y clave HMAC dedicada/versionada. El código
	// se deriva del dominio, persona, correo_ref, desafío y vencimiento.
	PrepararDesafioCorreo(context.Context, string, string, time.Time) (ReservaDesafio, error)
}

type MetadatosDesafioCorreo struct {
	PersonaRef   string
	CorreoRef    string
	DesafioRef   string
	HuellaCodigo []byte
	ClaveRef     string
	VenceUTC     time.Time
}

// El registro invoca Comprobar bajo bloqueo de la fila pendiente; consume
// atómicamente intento o éxito. Una clave revocada debe producir false/error.
type ComprobadorCodigoCorreo interface {
	Comprobar(context.Context, MetadatosDesafioCorreo) (bool, error)
}

type ValidadorCodigoCorreo interface {
	ComprobarCodigoCorreo(context.Context, MetadatosDesafioCorreo, string) (bool, error)
}

// Aplicar es la única puerta durable. Verifica/consume V3 y CAS en la misma
// transacción que cuota persistente (3 reenvíos/hora, sin contar replay),
// intentos, estado, historia, auditoría, recibo y outbox. El primer alta
// pendiente no se convierte en activo. Activar exige verificado y reserva
// aviso al anterior; retirar activo exige SustitutoRef verificado distinto y
// realiza ambos cambios de forma atómica. No elige sustituto por defecto.
// Si una llamada concurrente ya confirmó misma persona+clave+huella de una
// generación válida, devuelve el recibo ORIGINAL con Replay=true, incluso
// cuando el correo_ref aleatorio de la segunda alta sea distinto. Descarta su
// sobre/desafío nuevos y no consume cuota, historia, auditoría ni outbox extra.
// La misma clave con huella distinta devuelve ErrCorreosConflicto.
type RegistroCorreos interface {
	ConsultarPropios(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (VistaCorreos, error)
	RecuperarOperacion(context.Context, OrdenCorreos, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboCorreos, bool, error)
	Aplicar(context.Context, OrdenCorreos, PeticionCorreo, MaterialCorreos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, SobreDireccionCorreo, ReservaDesafio, ComprobadorCodigoCorreo) (ReciboCorreos, error)
}

// Lectura para CT/Bolsa. El consumidor aporta finalidad y concesión exacta;
// el adaptador audita la lectura y sólo expone el activo verificado. No se
// permite consultar tablas de usuarios desde el módulo consumidor.
type SolicitudCorreoActivo struct {
	PersonaRef   string
	FinalidadRef string
	Material     vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type CorreoActivoVerificado struct {
	PersonaRef string
	CorreoRef  string
	Direccion  string
	Version    uint64
}

type LectorCorreoActivoVerificado interface {
	ConsultarActivoVerificado(context.Context, SolicitudCorreoActivo) (CorreoActivoVerificado, bool, error)
}

// El dispatcher consume una reserva ya confirmada por el registro. Sólo
// Desafio sale por outbox; el código se deriva en memoria y el transporte no
// informa entrega jurídica, únicamente aceptación SMTP.
type DespachoVerificacionCorreo struct {
	OutboxRef  string
	PersonaRef string
	CorreoRef  string
	DesafioRef string
	Desafio    []byte
	ClaveRef   string
	VenceUTC   time.Time
}

// Resuelve y descifra la dirección en memoria después de autorizar el intento
// de despacho. La dirección en claro no figura en el outbox.
type LectorDireccionDespachoCorreo interface {
	ConDireccionDespachoCorreo(context.Context, string, string, func(string) error) error // persona_ref, correo_ref
}

type DerivadorCodigoDespachoCorreo interface {
	DerivarCodigoCorreo(context.Context, DespachoVerificacionCorreo) (string, error)
}

type TransportadorVerificacionCorreo interface {
	AceptarVerificacionCorreo(context.Context, string, string, string) error // outbox_ref, direccion, codigo
}
