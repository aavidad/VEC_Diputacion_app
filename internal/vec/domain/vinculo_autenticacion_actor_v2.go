package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"
)

const (
	VersionVinculoAutenticacionActorV2    uint16 = 2
	EsquemaVinculoAutenticacionActorV2           = "vec.autenticacion-actor.vinculo.v2.contexto-registrado"
	longitudMinimaRegistroContextoActorV2        = 24
	longitudMaximaRegistroContextoActorV2        = 128
)

var (
	ErrVinculoAutenticacionActorV2Invalido = errors.New(
		"vec: vinculo de autenticacion y actor v2 invalido",
	)
	ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida = errors.New(
		"vec: serializacion alternativa de vinculo de autenticacion y actor v2 prohibida",
	)
)

// ResultadoContextoActorRegistradoV2 es la proyeccion minimizada de un recibo
// durable. OperacionRef se excluye: identifica la invocacion del adaptador y no
// forma parte de la identidad que debe comprometer una decision.
type ResultadoContextoActorRegistradoV2 struct {
	RegistroContextoRef               string
	Contexto                          ContextoActor
	RepresentacionCanonica            []byte
	HuellaSHA256                      string
	ManifiestoProcedenciaCanonico     []byte
	ManifiestoProcedenciaHuellaSHA256 string
	AutoridadEfectiva                 AutoridadProcedenciaContextoActorV1
	ResueltoEnAutoritativo            time.Time
}

func (r ResultadoContextoActorRegistradoV2) Validar() error {
	if !referenciaRegistroContextoActorV2Valida(r.RegistroContextoRef) ||
		r.Contexto.Validar() != nil || r.Contexto.Instantanea.CuentaVersion == 0 ||
		r.AutoridadEfectiva != AutoridadProcedenciaContextoActorMaestraAcreditadaV1 ||
		!instanteAutorizacionCanonico(r.ResueltoEnAutoritativo) ||
		!r.Contexto.ResueltoEn.Equal(r.ResueltoEnAutoritativo) {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	canon, err := r.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canon, r.RepresentacionCanonica) {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	huella, err := r.Contexto.HuellaSHA256VinculadaV2()
	if err != nil || huella != r.HuellaSHA256 {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	manifiesto, err := RehidratarManifiestoProcedenciaContextoActorV1(
		r.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.AutoridadEfectiva != r.AutoridadEfectiva ||
		manifiesto.ValidarParaContexto(r.Contexto) != nil {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	huellaManifiesto, err := HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		r.ManifiestoProcedenciaCanonico,
	)
	if err != nil || huellaManifiesto != r.ManifiestoProcedenciaHuellaSHA256 {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}

func (r ResultadoContextoActorRegistradoV2) Clonar() (ResultadoContextoActorRegistradoV2, error) {
	if r.Validar() != nil {
		return ResultadoContextoActorRegistradoV2{}, ErrVinculoAutenticacionActorV2Invalido
	}
	actor, err := r.Contexto.Clonar()
	if err != nil {
		return ResultadoContextoActorRegistradoV2{}, ErrVinculoAutenticacionActorV2Invalido
	}
	r.Contexto = actor
	r.RepresentacionCanonica = append([]byte(nil), r.RepresentacionCanonica...)
	r.ManifiestoProcedenciaCanonico = append([]byte(nil), r.ManifiestoProcedenciaCanonico...)
	return r, nil
}

// ResolutorContextoActorRegistradoV2 es una autoridad, no un repositorio de
// DTO. La composicion productiva debe adaptarlo desde ResolverRegistrado.
type ResolutorContextoActorRegistradoV2 interface {
	ResolverContextoActorRegistradoV2(
		context.Context,
		SolicitudContextoActor,
	) (ResultadoContextoActorRegistradoV2, error)
}

type RelojVinculoAutenticacionActorV2 interface {
	Ahora() time.Time
}

// DatosVinculoAutenticacionActorV2 es la evidencia cerrada que consume PDP
// V3. No contiene bytes, operacion, PII, roles, permisos ni claims libres.
type DatosVinculoAutenticacionActorV2 struct {
	Esquema                           string                              `json:"esquema"`
	BloqueVersion                     uint16                              `json:"bloque_version"`
	AutenticacionRef                  string                              `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256         string                              `json:"autenticacion_huella_sha256"`
	AsercionRef                       string                              `json:"asercion_ref"`
	SesionRef                         string                              `json:"sesion_ref"`
	ControlSesionRef                  string                              `json:"control_sesion_ref"`
	ControlSesionRevision             uint64                              `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256         string                              `json:"control_sesion_huella_sha256"`
	CuentaRef                         string                              `json:"cuenta_ref"`
	CuentaOrdinariaRef                string                              `json:"cuenta_ordinaria_ref"`
	PrincipalID                       string                              `json:"principal_id"`
	PerfilActivoRef                   string                              `json:"perfil_activo_ref"`
	CuentaPrivilegiada                bool                                `json:"cuenta_privilegiada"`
	Superficie                        SuperficieAutenticacionActorV1      `json:"superficie"`
	MetodoObservado                   AuthMethod                          `json:"metodo_observado"`
	GarantiaObservada                 AuthAssurance                       `json:"garantia_observada"`
	PoliticaGarantiaRef               string                              `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256      string                              `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn         time.Time                           `json:"autenticacion_verificada_en"`
	SesionEmitidaEn                   time.Time                           `json:"sesion_emitida_en"`
	SesionValidaHasta                 time.Time                           `json:"sesion_valida_hasta"`
	SesionRevalidadaEn                time.Time                           `json:"sesion_revalidada_en"`
	RegistroContextoRef               string                              `json:"registro_contexto_ref"`
	ContextoActorEsquema              string                              `json:"contexto_actor_esquema"`
	ContextoActorRef                  string                              `json:"contexto_actor_ref"`
	ContextoActorVersion              uint64                              `json:"contexto_actor_version"`
	ContextoActorCuentaVersion        uint64                              `json:"contexto_actor_cuenta_version"`
	ContextoActorHuellaSHA256         string                              `json:"contexto_actor_huella_sha256"`
	ManifiestoProcedenciaHuellaSHA256 string                              `json:"manifiesto_procedencia_huella_sha256"`
	AutoridadEfectiva                 AutoridadProcedenciaContextoActorV1 `json:"autoridad_efectiva"`
}

func (d DatosVinculoAutenticacionActorV2) Autenticacion() AutenticacionRevalidadaV1 {
	return AutenticacionRevalidadaV1{
		AutenticacionRef: d.AutenticacionRef, AutenticacionHuellaSHA256: d.AutenticacionHuellaSHA256,
		AsercionRef: d.AsercionRef, SesionRef: d.SesionRef, ControlSesionRef: d.ControlSesionRef,
		ControlSesionRevision: d.ControlSesionRevision, ControlSesionHuellaSHA256: d.ControlSesionHuellaSHA256,
		CuentaRef: d.CuentaRef, CuentaOrdinariaRef: d.CuentaOrdinariaRef,
		CuentaPrivilegiada: d.CuentaPrivilegiada, Superficie: d.Superficie,
		MetodoObservado: d.MetodoObservado, GarantiaObservada: d.GarantiaObservada,
		PoliticaGarantiaRef: d.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: d.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: d.AutenticacionVerificadaEn, SesionEmitidaEn: d.SesionEmitidaEn,
		SesionValidaHasta: d.SesionValidaHasta, SesionRevalidadaEn: d.SesionRevalidadaEn,
	}
}

func (d DatosVinculoAutenticacionActorV2) Validar() error {
	if d.Esquema != EsquemaVinculoAutenticacionActorV2 ||
		d.BloqueVersion != VersionVinculoAutenticacionActorV2 || d.Autenticacion().Validar() != nil ||
		!referenciaOpacaContextoActorValida(d.PrincipalID, "per_") ||
		!referenciaOpacaContextoActorValida(d.PerfilActivoRef, "prf_") ||
		!referenciaRegistroContextoActorV2Valida(d.RegistroContextoRef) ||
		d.ContextoActorEsquema != EsquemaRepresentacionContextoActorV2 ||
		!referenciaOpacaContextoActorValida(d.ContextoActorRef, "vca_") ||
		d.ContextoActorVersion == 0 || d.ContextoActorCuentaVersion == 0 ||
		!huellaSHA256AutorizacionValida(d.ContextoActorHuellaSHA256) ||
		!huellaSHA256AutorizacionValida(d.ManifiestoProcedenciaHuellaSHA256) ||
		d.AutoridadEfectiva != AutoridadProcedenciaContextoActorMaestraAcreditadaV1 {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}

type VinculoAutenticacionActorV2 struct {
	datos *DatosVinculoAutenticacionActorV2
}

func (v VinculoAutenticacionActorV2) Validar() error {
	if v.datos == nil || v.datos.Validar() != nil {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}

func (v VinculoAutenticacionActorV2) Datos() (DatosVinculoAutenticacionActorV2, error) {
	if v.Validar() != nil {
		return DatosVinculoAutenticacionActorV2{}, ErrVinculoAutenticacionActorV2Invalido
	}
	return *v.datos, nil
}

func (v VinculoAutenticacionActorV2) CoincideExactamenteCon(otro VinculoAutenticacionActorV2) bool {
	a, errA := v.Datos()
	b, errB := otro.Datos()
	return errA == nil && errB == nil && a == b
}

func (v VinculoAutenticacionActorV2) MarshalJSON() ([]byte, error) {
	datos, err := v.Datos()
	if err != nil {
		return nil, ErrVinculoAutenticacionActorV2Invalido
	}
	return json.Marshal(datos)
}

func (*VinculoAutenticacionActorV2) UnmarshalJSON([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) UnmarshalText([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) UnmarshalBinary([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) GobDecode([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) UnmarshalCBOR([]byte) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) UnmarshalYAML(func(any) error) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida
}
func (*VinculoAutenticacionActorV2) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrReconstruccionVinculoAutenticacionActorProhibida
}
func (VinculoAutenticacionActorV2) String() string     { return "[VINCULO-AUTENTICACION-ACTOR-V2-OPACO]" }
func (v VinculoAutenticacionActorV2) GoString() string { return v.String() }
func (v VinculoAutenticacionActorV2) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, v.String())
}
func (v VinculoAutenticacionActorV2) LogValue() slog.Value { return slog.StringValue(v.String()) }

// CrearVinculoAutenticacionActorV2 conserva el contrato anterior y delega en
// la fabrica nominal que devuelve tambien el resultado registrado exacto.
func CrearVinculoAutenticacionActorV2(
	ctx context.Context,
	revalidador RevalidadorAutenticacionActorV1,
	solicitudAutenticacion SolicitudRevalidacionAutenticacionActorV1,
	resolutor ResolutorContextoActorRegistradoV2,
	solicitudContexto SolicitudContextoActor,
	reloj RelojVinculoAutenticacionActorV2,
) (VinculoAutenticacionActorV2, error) {
	vinculo, _, err := CrearVinculoAutenticacionActorV2ConResultado(
		ctx, revalidador, solicitudAutenticacion, resolutor,
		solicitudContexto, reloj,
	)
	return vinculo, err
}

// CrearVinculoAutenticacionActorV2ConResultado invoca exactamente una vez las
// autoridades de autenticacion y contexto. Devuelve el vinculo junto al mismo
// resultado registrado que lo origino, clonado defensivamente. No admite un
// ContextoActor ni un recibo suministrados por el llamador como sustitutos de
// esa resolucion.
func CrearVinculoAutenticacionActorV2ConResultado(
	ctx context.Context,
	revalidador RevalidadorAutenticacionActorV1,
	solicitudAutenticacion SolicitudRevalidacionAutenticacionActorV1,
	resolutor ResolutorContextoActorRegistradoV2,
	solicitudContexto SolicitudContextoActor,
	reloj RelojVinculoAutenticacionActorV2,
) (
	VinculoAutenticacionActorV2,
	ResultadoContextoActorRegistradoV2,
	error,
) {
	if ctx == nil || ctx.Err() != nil || dependenciaVinculoAutenticacionActorV2Nula(revalidador) ||
		dependenciaVinculoAutenticacionActorV2Nula(resolutor) ||
		dependenciaVinculoAutenticacionActorV2Nula(reloj) || solicitudAutenticacion.Validar() != nil ||
		solicitudContexto.Validar() != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}
	autenticacion, err := revalidador.RevalidarAutenticacionActorV1(ctx, solicitudAutenticacion)
	if err != nil || ctx.Err() != nil || autenticacion.Validar() != nil ||
		autenticacion.AutenticacionRef != solicitudAutenticacion.AutenticacionRef ||
		autenticacion.SesionRef != solicitudAutenticacion.SesionRef {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, err, ctx.Err())
	}
	resultado, err := resolutor.ResolverContextoActorRegistradoV2(ctx, solicitudContexto)
	if err != nil || ctx.Err() != nil || resultado.Validar() != nil ||
		resultado.Contexto.Instantanea.CuentaRef != solicitudContexto.Cuenta.CuentaRef ||
		resultado.Contexto.PerfilActivoRef != solicitudContexto.PerfilActivoRef ||
		resultado.Contexto.Principal.AuthMethod != solicitudContexto.Cuenta.Metodo ||
		resultado.Contexto.Principal.AuthAssurance != solicitudContexto.Cuenta.Garantia {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, err, ctx.Err())
	}
	resultadoClonado, err := resultado.Clonar()
	if err != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}
	ahora := reloj.Ahora().UTC().Truncate(time.Microsecond)
	if errContexto := ctx.Err(); errContexto != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, errContexto)
	}
	if !instanteAutorizacionCanonico(ahora) ||
		ahora.Before(autenticacion.SesionRevalidadaEn) ||
		!ahora.Before(autenticacion.SesionValidaHasta) ||
		ahora.Before(resultadoClonado.ResueltoEnAutoritativo) ||
		!contextoActorV2VigenteEn(resultadoClonado.Contexto, ahora) {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}
	vinculo, err := nuevoVinculoAutenticacionActorV2(
		autenticacion,
		resultadoClonado,
	)
	if err != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			err
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			errors.Join(ErrVinculoAutenticacionActorV2Invalido, errContexto)
	}
	if vinculo.ValidarPara(resultadoClonado) != nil {
		return VinculoAutenticacionActorV2{}, ResultadoContextoActorRegistradoV2{},
			ErrVinculoAutenticacionActorV2Invalido
	}
	return vinculo, resultadoClonado, nil
}

func nuevoVinculoAutenticacionActorV2(
	a AutenticacionRevalidadaV1,
	r ResultadoContextoActorRegistradoV2,
) (VinculoAutenticacionActorV2, error) {
	actor := r.Contexto
	if a.CuentaRef != actor.Instantanea.CuentaRef || a.MetodoObservado != actor.Principal.AuthMethod ||
		a.GarantiaObservada != actor.Principal.AuthAssurance ||
		actor.ResueltoEn.Before(a.SesionRevalidadaEn) || !actor.ResueltoEn.Before(a.SesionValidaHasta) {
		return VinculoAutenticacionActorV2{}, ErrVinculoAutenticacionActorV2Invalido
	}
	d := DatosVinculoAutenticacionActorV2{
		Esquema: EsquemaVinculoAutenticacionActorV2, BloqueVersion: VersionVinculoAutenticacionActorV2,
		AutenticacionRef: a.AutenticacionRef, AutenticacionHuellaSHA256: a.AutenticacionHuellaSHA256,
		AsercionRef: a.AsercionRef, SesionRef: a.SesionRef, ControlSesionRef: a.ControlSesionRef,
		ControlSesionRevision: a.ControlSesionRevision, ControlSesionHuellaSHA256: a.ControlSesionHuellaSHA256,
		CuentaRef: a.CuentaRef, CuentaOrdinariaRef: a.CuentaOrdinariaRef,
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		CuentaPrivilegiada: a.CuentaPrivilegiada, Superficie: a.Superficie,
		MetodoObservado: a.MetodoObservado, GarantiaObservada: a.GarantiaObservada,
		PoliticaGarantiaRef: a.PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256: a.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn: a.AutenticacionVerificadaEn, SesionEmitidaEn: a.SesionEmitidaEn,
		SesionValidaHasta: a.SesionValidaHasta, SesionRevalidadaEn: a.SesionRevalidadaEn,
		RegistroContextoRef: r.RegistroContextoRef, ContextoActorEsquema: EsquemaRepresentacionContextoActorV2,
		ContextoActorRef: actor.Instantanea.VinculoRef, ContextoActorVersion: actor.Instantanea.VinculoVersion,
		ContextoActorCuentaVersion: actor.Instantanea.CuentaVersion, ContextoActorHuellaSHA256: r.HuellaSHA256,
		ManifiestoProcedenciaHuellaSHA256: r.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 r.AutoridadEfectiva,
	}
	v := VinculoAutenticacionActorV2{datos: &d}
	if v.ValidarPara(r) != nil {
		return VinculoAutenticacionActorV2{}, ErrVinculoAutenticacionActorV2Invalido
	}
	return v, nil
}

func (v VinculoAutenticacionActorV2) ValidarPara(r ResultadoContextoActorRegistradoV2) error {
	d, err := v.Datos()
	actor := r.Contexto
	if err != nil || r.Validar() != nil || d.RegistroContextoRef != r.RegistroContextoRef ||
		d.CuentaRef != actor.Instantanea.CuentaRef || d.PrincipalID != actor.Principal.ID ||
		d.PrincipalID != actor.PersonaRef || d.PerfilActivoRef != actor.PerfilActivoRef ||
		d.MetodoObservado != actor.Principal.AuthMethod || d.GarantiaObservada != actor.Principal.AuthAssurance ||
		d.ContextoActorRef != actor.Instantanea.VinculoRef || d.ContextoActorVersion != actor.Instantanea.VinculoVersion ||
		d.ContextoActorCuentaVersion != actor.Instantanea.CuentaVersion || d.ContextoActorHuellaSHA256 != r.HuellaSHA256 ||
		d.ManifiestoProcedenciaHuellaSHA256 != r.ManifiestoProcedenciaHuellaSHA256 ||
		d.AutoridadEfectiva != r.AutoridadEfectiva || actor.ResueltoEn.Before(d.SesionRevalidadaEn) ||
		!actor.ResueltoEn.Before(d.SesionValidaHasta) {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}

func (v VinculoAutenticacionActorV2) VigenteEn(instante time.Time, r ResultadoContextoActorRegistradoV2) bool {
	d, err := v.Datos()
	return err == nil && instanteAutorizacionCanonico(instante) && v.ValidarPara(r) == nil &&
		!instante.Before(d.SesionRevalidadaEn) && instante.Before(d.SesionValidaHasta) &&
		!instante.Before(r.ResueltoEnAutoritativo) && contextoActorV2VigenteEn(r.Contexto, instante)
}

func contextoActorV2VigenteEn(actor ContextoActor, instante time.Time) bool {
	if actor.Validar() != nil || actor.Instantanea.CuentaVersion == 0 || instante.Before(actor.ResueltoEn) ||
		!actor.Instantanea.VigenteEn(instante) {
		return false
	}
	for _, vinculo := range actor.Instantanea.Vinculos {
		if !vinculo.VigenteEn(instante) {
			return false
		}
	}
	return true
}

func dependenciaVinculoAutenticacionActorV2Nula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func referenciaRegistroContextoActorV2Valida(valor string) bool {
	const prefijo = "rca_"
	if len(valor) < len(prefijo)+longitudMinimaRegistroContextoActorV2 ||
		len(valor) > len(prefijo)+longitudMaximaRegistroContextoActorV2 ||
		valor[:len(prefijo)] != prefijo {
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
