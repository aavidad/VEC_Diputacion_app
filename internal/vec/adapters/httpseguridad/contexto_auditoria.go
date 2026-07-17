package httpseguridad

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var ErrContextoAuditoriaNoSerializable = errors.New("el contexto de auditoria autenticada no se reconstruye desde una serializacion")

const contextoAuditoriaRedactado = "[CONTEXTO DE AUDITORIA AUTENTICADA CONFIDENCIAL]"

// ResumenFactorAuditoria no contiene secretos ni concede autorizacion.
type ResumenFactorAuditoria struct {
	Metodo                MetodoAutenticacion
	EvidenciaRef          string
	GrupoCriptograficoRef string
	VerificadoEn          time.Time
}

// ContextoAuditoriaAutenticada conserva referencias canonicas de cuenta,
// sesion y politica verificadas. Deliberadamente no conserva los IDs crudos de
// sujeto, cuenta, asercion o sesion del IdP, ni roles, permisos o atributos.
type ContextoAuditoriaAutenticada struct {
	autenticacionRef          string
	asercionRef               string
	sesionRef                 string
	controlSesionRef          string
	cuentaRef                 string
	cuentaOrdinariaRef        string
	autenticacionHuellaSHA256 string
	controlSesionRevision     uint64
	controlSesionEstado       EstadoControlSesion
	controlSesionHuellaSHA256 string
	sesionRevalidadaEn        time.Time
	sesionValidaHasta         time.Time
	emisor                    string
	audiencia                 string
	cuentaPrivilegiada        bool
	superficie                Superficie
	metodoPrimario            MetodoAutenticacion
	metodoObservado           dominiovec.AuthMethod
	autenticacionVerificadaEn time.Time
	garantia                  dominiovec.AuthAssurance
	emitidaEn                 time.Time
	noAntesDe                 time.Time
	expiraEn                  time.Time
	politicaGarantiaRef       string
	huellaPolitica            string
	huellaConfiguracion       string
	canalVinculadoRef         string
	factores                  []ResumenFactorAuditoria
}

func (c ContextoAuditoriaAutenticada) AutenticacionRef() string   { return c.autenticacionRef }
func (c ContextoAuditoriaAutenticada) AsercionRef() string        { return c.asercionRef }
func (c ContextoAuditoriaAutenticada) SesionRef() string          { return c.sesionRef }
func (c ContextoAuditoriaAutenticada) ControlSesionRef() string   { return c.controlSesionRef }
func (c ContextoAuditoriaAutenticada) CuentaRef() string          { return c.cuentaRef }
func (c ContextoAuditoriaAutenticada) CuentaOrdinariaRef() string { return c.cuentaOrdinariaRef }
func (c ContextoAuditoriaAutenticada) AutenticacionHuellaSHA256() string {
	return c.autenticacionHuellaSHA256
}
func (c ContextoAuditoriaAutenticada) ControlSesionRevision() uint64 {
	return c.controlSesionRevision
}
func (c ContextoAuditoriaAutenticada) ControlSesionEstado() EstadoControlSesion {
	return c.controlSesionEstado
}
func (c ContextoAuditoriaAutenticada) ControlSesionHuellaSHA256() string {
	return c.controlSesionHuellaSHA256
}
func (c ContextoAuditoriaAutenticada) SesionRevalidadaEn() time.Time { return c.sesionRevalidadaEn }
func (c ContextoAuditoriaAutenticada) SesionValidaHasta() time.Time  { return c.sesionValidaHasta }
func (c ContextoAuditoriaAutenticada) Emisor() string                { return c.emisor }
func (c ContextoAuditoriaAutenticada) Audiencia() string             { return c.audiencia }
func (c ContextoAuditoriaAutenticada) CuentaPrivilegiada() bool      { return c.cuentaPrivilegiada }
func (c ContextoAuditoriaAutenticada) Superficie() Superficie        { return c.superficie }
func (c ContextoAuditoriaAutenticada) MetodoPrimario() MetodoAutenticacion {
	return c.metodoPrimario
}
func (c ContextoAuditoriaAutenticada) MetodoObservado() dominiovec.AuthMethod {
	return c.metodoObservado
}
func (c ContextoAuditoriaAutenticada) AutenticacionVerificadaEn() time.Time {
	return c.autenticacionVerificadaEn
}
func (c ContextoAuditoriaAutenticada) Garantia() dominiovec.AuthAssurance { return c.garantia }
func (c ContextoAuditoriaAutenticada) GarantiaObservada() dominiovec.AuthAssurance {
	return c.garantia
}
func (c ContextoAuditoriaAutenticada) EmitidaEn() time.Time        { return c.emitidaEn }
func (c ContextoAuditoriaAutenticada) SesionEmitidaEn() time.Time  { return c.emitidaEn }
func (c ContextoAuditoriaAutenticada) NoAntesDe() time.Time        { return c.noAntesDe }
func (c ContextoAuditoriaAutenticada) ExpiraEn() time.Time         { return c.expiraEn }
func (c ContextoAuditoriaAutenticada) PoliticaGarantiaRef() string { return c.politicaGarantiaRef }
func (c ContextoAuditoriaAutenticada) HuellaPolitica() string      { return c.huellaPolitica }
func (c ContextoAuditoriaAutenticada) PoliticaGarantiaHuellaSHA256() string {
	return strings.TrimPrefix(c.huellaPolitica, "sha256:")
}
func (c ContextoAuditoriaAutenticada) HuellaConfiguracion() string { return c.huellaConfiguracion }
func (c ContextoAuditoriaAutenticada) CanalVinculadoRef() string   { return c.canalVinculadoRef }
func (c ContextoAuditoriaAutenticada) Factores() []ResumenFactorAuditoria {
	return append([]ResumenFactorAuditoria(nil), c.factores...)
}

func (ContextoAuditoriaAutenticada) String() string   { return contextoAuditoriaRedactado }
func (ContextoAuditoriaAutenticada) GoString() string { return contextoAuditoriaRedactado }
func (ContextoAuditoriaAutenticada) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(contextoAuditoriaRedactado))
}
func (ContextoAuditoriaAutenticada) LogValue() slog.Value {
	return slog.StringValue(contextoAuditoriaRedactado)
}
func (ContextoAuditoriaAutenticada) MarshalJSON() ([]byte, error) {
	return []byte(`{"contexto_auditoria_autenticada":"[CONFIDENCIAL]"}`), nil
}
func (*ContextoAuditoriaAutenticada) UnmarshalJSON([]byte) error {
	return ErrContextoAuditoriaNoSerializable
}
func (ContextoAuditoriaAutenticada) MarshalText() ([]byte, error) {
	return []byte(contextoAuditoriaRedactado), nil
}
func (*ContextoAuditoriaAutenticada) UnmarshalText([]byte) error {
	return ErrContextoAuditoriaNoSerializable
}
func (ContextoAuditoriaAutenticada) MarshalBinary() ([]byte, error) {
	return []byte(contextoAuditoriaRedactado), nil
}
func (*ContextoAuditoriaAutenticada) UnmarshalBinary([]byte) error {
	return ErrContextoAuditoriaNoSerializable
}
func (ContextoAuditoriaAutenticada) GobEncode() ([]byte, error) {
	return []byte(contextoAuditoriaRedactado), nil
}
func (*ContextoAuditoriaAutenticada) GobDecode([]byte) error {
	return ErrContextoAuditoriaNoSerializable
}
