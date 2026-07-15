package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"
)

var (
	ErrAutenticacionRevalidadaInvalida                  = errors.New("vec: autenticacion revalidada invalida")
	ErrVinculoAutenticacionActorInvalido                = errors.New("vec: vinculo de autenticacion y actor invalido")
	ErrReconstruccionVinculoAutenticacionActorProhibida = errors.New("vec: reconstruccion de vinculo de autenticacion y actor prohibida")
)

const (
	// VersionVinculoAutenticacionActorV1 identifica el bloque cerrado que liga
	// una decision con la sesion revalidada y el documento de actor resuelto.
	// El valor cero nunca identifica una version compatible.
	VersionVinculoAutenticacionActorV1 uint16 = 1

	esquemaHuellaContextoActorV1 = "vec.contexto-actor.vinculado.v1"
)

// SuperficieAutenticacionActorV1 es deliberadamente cerrada. La superficie
// publica anonima no puede producir una decision asociada a una persona.
type SuperficieAutenticacionActorV1 string

const (
	SuperficieAutenticacionExternaPersonalV1            SuperficieAutenticacionActorV1 = "externa_personal"
	SuperficieAutenticacionInternaCorporativaV1         SuperficieAutenticacionActorV1 = "interna_corporativa"
	SuperficieAutenticacionAdministracionPrivilegiadaV1 SuperficieAutenticacionActorV1 = "administracion_privilegiada"
)

func (s SuperficieAutenticacionActorV1) Valida() bool {
	switch s {
	case SuperficieAutenticacionExternaPersonalV1,
		SuperficieAutenticacionInternaCorporativaV1,
		SuperficieAutenticacionAdministracionPrivilegiadaV1:
		return true
	default:
		return false
	}
}

// AutenticacionRevalidadaV1 es el resultado tipado de la autoridad de sesion.
// No se construye a partir de un DTO HTTP: el puerto de revalidacion debe
// resolver estos datos desde sus registros autoritativos y devolverlos todos.
// Una omision no se completa ni se interpreta como un valor predeterminado.
type AutenticacionRevalidadaV1 struct {
	AutenticacionRef             string                         `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string                         `json:"autenticacion_huella_sha256"`
	AsercionRef                  string                         `json:"asercion_ref"`
	SesionRef                    string                         `json:"sesion_ref"`
	ControlSesionRef             string                         `json:"control_sesion_ref"`
	ControlSesionRevision        uint64                         `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string                         `json:"control_sesion_huella_sha256"`
	CuentaRef                    string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef           string                         `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiada           bool                           `json:"cuenta_privilegiada"`
	Superficie                   SuperficieAutenticacionActorV1 `json:"superficie"`
	MetodoObservado              AuthMethod                     `json:"metodo_observado"`
	GarantiaObservada            AuthAssurance                  `json:"garantia_observada"`
	PoliticaGarantiaRef          string                         `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string                         `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    time.Time                      `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              time.Time                      `json:"sesion_emitida_en"`
	SesionValidaHasta            time.Time                      `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           time.Time                      `json:"sesion_revalidada_en"`
}

// SolicitudRevalidacionAutenticacionActorV1 solo transporta referencias
// opacas. La cuenta, persona, perfil, metodo, garantia y superficie nunca se
// aceptan como atributos declarados en esta solicitud.
type SolicitudRevalidacionAutenticacionActorV1 struct {
	AutenticacionRef string `json:"autenticacion_ref"`
	SesionRef        string `json:"sesion_ref"`
}

// RevalidadorAutenticacionActorV1 es una autoridad inyectada, no un DTO. Su
// implementacion debe consultar la sesion y sus controles en el origen
// autoritativo. El nucleo solo emitira la capacidad opaca tras cruzar el
// resultado con ContextoActor.
type RevalidadorAutenticacionActorV1 interface {
	RevalidarAutenticacionActorV1(
		context.Context,
		SolicitudRevalidacionAutenticacionActorV1,
	) (AutenticacionRevalidadaV1, error)
}

func (s SolicitudRevalidacionAutenticacionActorV1) Validar() error {
	if !referenciaOpacaContextoActorValida(s.AutenticacionRef, "aut_") ||
		!referenciaOpacaContextoActorValida(s.SesionRef, "ses_") {
		return ErrAutenticacionRevalidadaInvalida
	}
	return nil
}

func (a AutenticacionRevalidadaV1) Validar() error {
	if !referenciaOpacaContextoActorValida(a.AutenticacionRef, "aut_") ||
		!huellaSHA256AutorizacionValida(a.AutenticacionHuellaSHA256) ||
		!referenciaOpacaContextoActorValida(a.AsercionRef, "ase_") ||
		!referenciaOpacaContextoActorValida(a.SesionRef, "ses_") ||
		!referenciaOpacaContextoActorValida(a.ControlSesionRef, "cse_") ||
		a.ControlSesionRevision == 0 ||
		!huellaSHA256AutorizacionValida(a.ControlSesionHuellaSHA256) ||
		!referenciaOpacaContextoActorValida(a.CuentaRef, "cta_") ||
		!referenciaOpacaContextoActorValida(a.CuentaOrdinariaRef, "cta_") ||
		!a.Superficie.Valida() || !a.MetodoObservado.Valido() ||
		a.MetodoObservado == AuthMethodDemo || !a.GarantiaObservada.Valida() ||
		!referenciaOpacaContextoActorValida(a.PoliticaGarantiaRef, "pga_") ||
		!huellaSHA256AutorizacionValida(a.PoliticaGarantiaHuellaSHA256) ||
		!instanteAutorizacionCanonico(a.AutenticacionVerificadaEn) ||
		!instanteAutorizacionCanonico(a.SesionEmitidaEn) ||
		!instanteAutorizacionCanonico(a.SesionValidaHasta) ||
		!instanteAutorizacionCanonico(a.SesionRevalidadaEn) ||
		a.AutenticacionVerificadaEn.After(a.SesionEmitidaEn) ||
		a.SesionRevalidadaEn.Before(a.AutenticacionVerificadaEn) ||
		a.SesionRevalidadaEn.Before(a.SesionEmitidaEn) ||
		!a.SesionValidaHasta.After(a.SesionRevalidadaEn) {
		return ErrAutenticacionRevalidadaInvalida
	}

	if a.CuentaPrivilegiada {
		if a.Superficie != SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
			a.CuentaRef == a.CuentaOrdinariaRef {
			return ErrAutenticacionRevalidadaInvalida
		}
		return nil
	}
	if a.Superficie == SuperficieAutenticacionAdministracionPrivilegiadaV1 ||
		a.CuentaRef != a.CuentaOrdinariaRef {
		return ErrAutenticacionRevalidadaInvalida
	}
	return nil
}

// DatosVinculoAutenticacionActorV1 contiene exactamente los 25 datos que
// quedan ligados a una decision. Es una proyeccion defensiva para firmar y
// persistir evidencia, no una capacidad reconstruible.
type DatosVinculoAutenticacionActorV1 struct {
	BloqueVersion                uint16                         `json:"bloque_version"`
	AutenticacionRef             string                         `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string                         `json:"autenticacion_huella_sha256"`
	AsercionRef                  string                         `json:"asercion_ref"`
	SesionRef                    string                         `json:"sesion_ref"`
	ControlSesionRef             string                         `json:"control_sesion_ref"`
	ControlSesionRevision        uint64                         `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string                         `json:"control_sesion_huella_sha256"`
	CuentaRef                    string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef           string                         `json:"cuenta_ordinaria_ref"`
	PrincipalID                  string                         `json:"principal_id"`
	PerfilActivoRef              string                         `json:"perfil_activo_ref"`
	CuentaPrivilegiada           bool                           `json:"cuenta_privilegiada"`
	Superficie                   SuperficieAutenticacionActorV1 `json:"superficie"`
	MetodoObservado              AuthMethod                     `json:"metodo_observado"`
	GarantiaObservada            AuthAssurance                  `json:"garantia_observada"`
	PoliticaGarantiaRef          string                         `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string                         `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    time.Time                      `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              time.Time                      `json:"sesion_emitida_en"`
	SesionValidaHasta            time.Time                      `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           time.Time                      `json:"sesion_revalidada_en"`
	ContextoActorRef             string                         `json:"contexto_actor_ref"`
	ContextoActorVersion         uint64                         `json:"contexto_actor_version"`
	ContextoActorHuellaSHA256    string                         `json:"contexto_actor_huella_sha256"`
}

func (v DatosVinculoAutenticacionActorV1) Autenticacion() AutenticacionRevalidadaV1 {
	return AutenticacionRevalidadaV1{
		AutenticacionRef: v.AutenticacionRef, AutenticacionHuellaSHA256: v.AutenticacionHuellaSHA256,
		AsercionRef: v.AsercionRef, SesionRef: v.SesionRef,
		ControlSesionRef: v.ControlSesionRef, ControlSesionRevision: v.ControlSesionRevision,
		ControlSesionHuellaSHA256: v.ControlSesionHuellaSHA256,
		CuentaRef:                 v.CuentaRef, CuentaOrdinariaRef: v.CuentaOrdinariaRef,
		CuentaPrivilegiada: v.CuentaPrivilegiada, Superficie: v.Superficie,
		MetodoObservado: v.MetodoObservado, GarantiaObservada: v.GarantiaObservada,
		PoliticaGarantiaRef:          v.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: v.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    v.AutenticacionVerificadaEn,
		SesionEmitidaEn:              v.SesionEmitidaEn, SesionValidaHasta: v.SesionValidaHasta,
		SesionRevalidadaEn: v.SesionRevalidadaEn,
	}
}

func (v DatosVinculoAutenticacionActorV1) Validar() error {
	if v.BloqueVersion != VersionVinculoAutenticacionActorV1 ||
		v.Autenticacion().Validar() != nil ||
		!referenciaOpacaContextoActorValida(v.PrincipalID, "per_") ||
		!referenciaOpacaContextoActorValida(v.PerfilActivoRef, "prf_") ||
		!referenciaOpacaContextoActorValida(v.ContextoActorRef, "vca_") ||
		v.ContextoActorVersion == 0 ||
		!huellaSHA256AutorizacionValida(v.ContextoActorHuellaSHA256) {
		return ErrVinculoAutenticacionActorInvalido
	}
	return nil
}

type datosVinculoAutenticacionActorV1 struct {
	DatosVinculoAutenticacionActorV1
}

// VinculoAutenticacionActorV1 es una capacidad opaca. El valor cero es
// invalido y otro paquete no puede rellenar sus 25 datos mediante un literal.
// MarshalJSON permite guardar la evidencia; UnmarshalJSON se mantiene cerrado
// hasta disponer de un rehidratador que revalide sesion y actor en una misma
// transaccion autoritativa.
type VinculoAutenticacionActorV1 struct {
	datos *datosVinculoAutenticacionActorV1
}

func (v VinculoAutenticacionActorV1) Validar() error {
	if v.datos == nil || v.datos.DatosVinculoAutenticacionActorV1.Validar() != nil {
		return ErrVinculoAutenticacionActorInvalido
	}
	return nil
}

func (v VinculoAutenticacionActorV1) Datos() (DatosVinculoAutenticacionActorV1, error) {
	if v.Validar() != nil {
		return DatosVinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
	}
	return v.datos.DatosVinculoAutenticacionActorV1, nil
}

// CoincideExactamenteCon compara dos capacidades validas por todos sus datos
// ligados. No normaliza, completa ni ignora campos; un valor cero o invalido
// nunca coincide, ni siquiera con otro valor cero o invalido.
func (v VinculoAutenticacionActorV1) CoincideExactamenteCon(otro VinculoAutenticacionActorV1) bool {
	primeros, errPrimeros := v.Datos()
	segundos, errSegundos := otro.Datos()
	return errPrimeros == nil && errSegundos == nil && primeros == segundos
}

func (v VinculoAutenticacionActorV1) MarshalJSON() ([]byte, error) {
	datos, err := v.Datos()
	if err != nil {
		return nil, ErrVinculoAutenticacionActorInvalido
	}
	return json.Marshal(datos)
}

func (*VinculoAutenticacionActorV1) UnmarshalJSON([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}

func (VinculoAutenticacionActorV1) MarshalText() ([]byte, error) {
	return nil, ErrReconstruccionVinculoAutenticacionActorProhibida
}

func (*VinculoAutenticacionActorV1) UnmarshalText([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}

func (VinculoAutenticacionActorV1) String() string {
	return "[VINCULO-AUTENTICACION-ACTOR-OPACO]"
}

func (v VinculoAutenticacionActorV1) GoString() string { return v.String() }

func (v VinculoAutenticacionActorV1) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}

func (v VinculoAutenticacionActorV1) LogValue() slog.Value {
	return slog.StringValue(v.String())
}

func nuevoVinculoAutenticacionActorV1(
	autenticacion AutenticacionRevalidadaV1,
	actor ContextoActor,
) (VinculoAutenticacionActorV1, error) {
	if autenticacion.Validar() != nil || actor.Validar() != nil ||
		autenticacion.CuentaRef != actor.Instantanea.CuentaRef ||
		autenticacion.MetodoObservado != actor.Principal.AuthMethod ||
		autenticacion.GarantiaObservada != actor.Principal.AuthAssurance ||
		actor.ResueltoEn.Before(autenticacion.SesionRevalidadaEn) ||
		!actor.ResueltoEn.Before(autenticacion.SesionValidaHasta) {
		return VinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
	}
	huellaActor, err := actor.HuellaSHA256VinculadaV1()
	if err != nil {
		return VinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
	}
	datos := DatosVinculoAutenticacionActorV1{
		BloqueVersion:             VersionVinculoAutenticacionActorV1,
		AutenticacionRef:          autenticacion.AutenticacionRef,
		AutenticacionHuellaSHA256: autenticacion.AutenticacionHuellaSHA256,
		AsercionRef:               autenticacion.AsercionRef, SesionRef: autenticacion.SesionRef,
		ControlSesionRef:          autenticacion.ControlSesionRef,
		ControlSesionRevision:     autenticacion.ControlSesionRevision,
		ControlSesionHuellaSHA256: autenticacion.ControlSesionHuellaSHA256,
		CuentaRef:                 autenticacion.CuentaRef, CuentaOrdinariaRef: autenticacion.CuentaOrdinariaRef,
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		CuentaPrivilegiada: autenticacion.CuentaPrivilegiada, Superficie: autenticacion.Superficie,
		MetodoObservado:              autenticacion.MetodoObservado,
		GarantiaObservada:            autenticacion.GarantiaObservada,
		PoliticaGarantiaRef:          autenticacion.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: autenticacion.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    autenticacion.AutenticacionVerificadaEn,
		SesionEmitidaEn:              autenticacion.SesionEmitidaEn,
		SesionValidaHasta:            autenticacion.SesionValidaHasta,
		SesionRevalidadaEn:           autenticacion.SesionRevalidadaEn,
		ContextoActorRef:             actor.Instantanea.VinculoRef,
		ContextoActorVersion:         actor.Instantanea.VinculoVersion,
		ContextoActorHuellaSHA256:    huellaActor,
	}
	vinculo := VinculoAutenticacionActorV1{datos: &datosVinculoAutenticacionActorV1{
		DatosVinculoAutenticacionActorV1: datos,
	}}
	if vinculo.ValidarPara(actor) != nil {
		return VinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
	}
	return vinculo, nil
}

// CrearVinculoAutenticacionActorV1 es la unica fabrica publica. La fuente
// autoritativa se invoca dentro de la operacion; no se acepta como argumento un
// resultado de autenticacion que el llamador haya podido rellenar directamente.
func CrearVinculoAutenticacionActorV1(
	ctx context.Context,
	revalidador RevalidadorAutenticacionActorV1,
	solicitud SolicitudRevalidacionAutenticacionActorV1,
	actor ContextoActor,
	ahora time.Time,
) (VinculoAutenticacionActorV1, error) {
	if ctx == nil || dependenciaVinculoAutenticacionActorNula(revalidador) || ctx.Err() != nil || solicitud.Validar() != nil ||
		actor.Validar() != nil || !instanteAutorizacionCanonico(ahora) {
		return VinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
	}
	autenticacion, err := revalidador.RevalidarAutenticacionActorV1(ctx, solicitud)
	if err != nil || ctx.Err() != nil || autenticacion.Validar() != nil ||
		autenticacion.AutenticacionRef != solicitud.AutenticacionRef ||
		autenticacion.SesionRef != solicitud.SesionRef ||
		ahora.Before(autenticacion.SesionRevalidadaEn) ||
		!ahora.Before(autenticacion.SesionValidaHasta) || ahora.Before(actor.ResueltoEn) ||
		!actor.Instantanea.VigenteEn(ahora) {
		return VinculoAutenticacionActorV1{}, errors.Join(ErrVinculoAutenticacionActorInvalido, err, ctx.Err())
	}
	for _, referencia := range actor.Instantanea.Vinculos {
		if !referencia.VigenteEn(ahora) {
			return VinculoAutenticacionActorV1{}, ErrVinculoAutenticacionActorInvalido
		}
	}
	return nuevoVinculoAutenticacionActorV1(autenticacion, actor)
}

func dependenciaVinculoAutenticacionActorNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

// ValidarPara demuestra la correspondencia exacta con el documento de actor.
func (v VinculoAutenticacionActorV1) ValidarPara(actor ContextoActor) error {
	datos, err := v.Datos()
	if err != nil || actor.Validar() != nil ||
		datos.CuentaRef != actor.Instantanea.CuentaRef ||
		datos.PrincipalID != actor.Principal.ID ||
		datos.PrincipalID != actor.PersonaRef ||
		datos.PerfilActivoRef != actor.PerfilActivoRef ||
		datos.MetodoObservado != actor.Principal.AuthMethod ||
		datos.GarantiaObservada != actor.Principal.AuthAssurance ||
		datos.ContextoActorRef != actor.Instantanea.VinculoRef ||
		datos.ContextoActorVersion != actor.Instantanea.VinculoVersion ||
		actor.ResueltoEn.Before(datos.SesionRevalidadaEn) ||
		!actor.ResueltoEn.Before(datos.SesionValidaHasta) {
		return ErrVinculoAutenticacionActorInvalido
	}
	huella, err := actor.HuellaSHA256VinculadaV1()
	if err != nil || huella != datos.ContextoActorHuellaSHA256 {
		return ErrVinculoAutenticacionActorInvalido
	}
	return nil
}

// VigenteEn exige simultaneamente la sesion y el documento de actor vigentes.
func (v VinculoAutenticacionActorV1) VigenteEn(instante time.Time, actor ContextoActor) bool {
	datos, err := v.Datos()
	if err != nil || !instanteAutorizacionCanonico(instante) || v.ValidarPara(actor) != nil ||
		instante.Before(datos.SesionRevalidadaEn) || !instante.Before(datos.SesionValidaHasta) ||
		instante.Before(actor.ResueltoEn) || !actor.Instantanea.VigenteEn(instante) {
		return false
	}
	for _, referencia := range actor.Instantanea.Vinculos {
		if !referencia.VigenteEn(instante) {
			return false
		}
	}
	return true
}

type vinculoReferenciaContextoActorCanonicoV1 struct {
	VinculoRef   string `json:"vinculo_ref"`
	Version      uint64 `json:"version"`
	Tipo         string `json:"tipo"`
	Referencia   string `json:"referencia"`
	Estado       string `json:"estado"`
	VigenteDesde string `json:"vigente_desde"`
	VigenteHasta string `json:"vigente_hasta"`
}

type contextoActorCanonicoV1 struct {
	Esquema          string                                     `json:"esquema"`
	PrincipalRef     string                                     `json:"principal_ref"`
	Metodo           AuthMethod                                 `json:"metodo"`
	Garantia         AuthAssurance                              `json:"garantia"`
	PerfilActivoRef  string                                     `json:"perfil_activo_ref"`
	PersonaRef       string                                     `json:"persona_ref"`
	ContextoActorRef string                                     `json:"contexto_actor_ref"`
	ContextoVersion  uint64                                     `json:"contexto_version"`
	CuentaRef        string                                     `json:"cuenta_ref"`
	PersonaVersion   uint64                                     `json:"persona_version"`
	PerfilVersion    uint64                                     `json:"perfil_version"`
	Estado           string                                     `json:"estado"`
	VigenteDesde     string                                     `json:"vigente_desde"`
	VigenteHasta     string                                     `json:"vigente_hasta"`
	ResueltoEn       string                                     `json:"resuelto_en"`
	Vinculos         []vinculoReferenciaContextoActorCanonicoV1 `json:"vinculos"`
}

// HuellaSHA256VinculadaV1 compromete la identidad canonica, cuenta, perfil,
// versiones, vigencias y referencias de modulo en un documento independiente.
func (c ContextoActor) HuellaSHA256VinculadaV1() (string, error) {
	canonica, err := c.Clonar()
	if err != nil {
		return "", ErrContextoActorInvalido
	}
	documento := contextoActorCanonicoV1{
		Esquema:      esquemaHuellaContextoActorV1,
		PrincipalRef: canonica.Principal.ID, Metodo: canonica.Principal.AuthMethod,
		Garantia: canonica.Principal.AuthAssurance, PerfilActivoRef: canonica.PerfilActivoRef,
		PersonaRef: canonica.PersonaRef, ContextoActorRef: canonica.Instantanea.VinculoRef,
		ContextoVersion: canonica.Instantanea.VinculoVersion, CuentaRef: canonica.Instantanea.CuentaRef,
		PersonaVersion: canonica.Instantanea.PersonaVersion,
		PerfilVersion:  canonica.Instantanea.PerfilVersion, Estado: string(canonica.Instantanea.Estado),
		VigenteDesde: instanteVinculoAutenticacionActorV1(canonica.Instantanea.VigenteDesde),
		VigenteHasta: instanteVinculoAutenticacionActorV1(canonica.Instantanea.VigenteHasta),
		ResueltoEn:   instanteVinculoAutenticacionActorV1(canonica.ResueltoEn),
		Vinculos:     make([]vinculoReferenciaContextoActorCanonicoV1, 0, len(canonica.Instantanea.Vinculos)),
	}
	for _, vinculo := range canonica.Instantanea.Vinculos {
		documento.Vinculos = append(documento.Vinculos, vinculoReferenciaContextoActorCanonicoV1{
			VinculoRef: vinculo.VinculoRef, Version: vinculo.Version, Tipo: string(vinculo.Tipo),
			Referencia: vinculo.Referencia, Estado: string(vinculo.Estado),
			VigenteDesde: instanteVinculoAutenticacionActorV1(vinculo.VigenteDesde),
			VigenteHasta: instanteVinculoAutenticacionActorV1(vinculo.VigenteHasta),
		})
	}
	contenido, err := json.Marshal(documento)
	if err != nil {
		return "", ErrContextoActorInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func instanteVinculoAutenticacionActorV1(instante time.Time) string {
	return instante.UTC().Format("2006-01-02T15:04:05.000000Z")
}
