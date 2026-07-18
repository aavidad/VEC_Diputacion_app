package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"

	documentalcanonico "vec-diputacion-granada/internal/vec/canonico/documental"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrManifiestoEjecucionDocumentalV3Invalido = errors.New("vec: manifiesto de ejecucion documental v3 invalido")
	ErrReservaEjecucionDocumentalV3Invalida    = errors.New("vec: reserva de ejecucion documental v3 invalida")
	ErrTokenCercadoDocumentalV3Invalido        = errors.New("vec: token de cercado documental v3 invalido")
	ErrTransicionEjecucionDocumentalV3Invalida = errors.New("vec: transicion de ejecucion documental v3 invalida")
	ErrReconciliacionDocumentalV3Invalida      = errors.New("vec: reconciliacion documental v3 invalida")
	ErrSelloEvidenciaDocumentalV3Invalido      = errors.New("vec: sello de evidencia documental v3 invalido")
	ErrSerializacionSecretoDocumentalV3        = documentalcanonico.ErrSerializacionSecretoDocumentalV3
	ErrReciboInicioDocumentalV3Invalido        = errors.New("vec: recibo durable de inicio documental v3 invalido")
	ErrOrdenDespachoDocumentalV3Invalida       = documentalcanonico.ErrOrdenDespachoDocumentalV3Invalida
)

const (
	EsquemaManifiestoEjecucionDocumentalV3 = "vec.documentos.manifiesto-ejecucion.v3"
	EsquemaEvidenciaRenderizadoV3          = "vec.documentos.evidencia-renderizado.v3"
	AlgoritmoSelloEvidenciaHMACSHA256V3    = documentalcanonico.AlgoritmoHMACSHA256V3
	AudienciaSelloEvidenciaRenderizadoV3   = "vec.documentos.evidencia-renderizado.v3"
	AudienciaAtestacionTokenCercadoV3      = documentalcanonico.AudienciaTokenCercadoV3
	AudienciaAtestacionInicioEfectoV3      = documentalcanonico.AudienciaInicioEfectoV3
	AudienciaAtestacionReclamacionV3       = documentalcanonico.AudienciaReclamacionDespachoV3
	AudienciaComprobacionOrdenDespachoV3   = documentalcanonico.AudienciaComprobacionOrdenDespachoV3
	ContextoAtestacionTokenCercadoV3       = documentalcanonico.ContextoTokenCercadoV3
	ContextoAtestacionInicioEfectoV3       = documentalcanonico.ContextoInicioEfectoV3
	ContextoAtestacionReclamacionV3        = documentalcanonico.ContextoReclamacionDespachoV3
	ContextoComprobacionOrdenDespachoV3    = documentalcanonico.ContextoComprobacionOrdenDespachoV3

	duracionMaximaReservaEjecucionDocumentalV3 = 15 * time.Minute
	tamanoMaximoSalidaEjecucionDocumentalV3    = uint64(256 * 1024 * 1024)
	tamanoFirmaHMACSHA256V3                    = documentalcanonico.TamanoFirmaHMACSHA256V3
)

// DatosManifiestoEjecucionDocumentalV3 compromete una resolucion completa y
// carente de datos personales directos. Las referencias son opacas: no se
// admiten URL, rutas, comodines, direcciones de correo ni identificadores DNI.
type DatosManifiestoEjecucionDocumentalV3 struct {
	Esquema               string
	Consulta              ConsultaFormatoDocumental
	DescriptorPerfil      DescriptorPerfilDocumental
	SituacionOperativa    domain.SituacionOperativaPerfilDocumental
	ComponenteRender      DescriptorComponenteDocumentalAtestado
	ComponenteVerificador DescriptorComponenteDocumentalAtestado
	ComponenteSemantico   DescriptorComponenteDocumentalAtestado
	BorradorRef           string
	EfectoRef             string
	HuellaEntradaHMAC     string
	LimiteEfectivoBytes   uint64
	HuellaPlanSHA256      string
}

func (DatosManifiestoEjecucionDocumentalV3) String() string {
	return "[DATOS-MANIFIESTO-EJECUCION-DOCUMENTAL-V3-HMAC-REDACTADOS]"
}
func (d DatosManifiestoEjecucionDocumentalV3) GoString() string { return d.String() }
func (d DatosManifiestoEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosManifiestoEjecucionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosManifiestoEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosManifiestoEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosManifiestoEjecucionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosManifiestoEjecucionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// ManifiestoEjecucionDocumentalV3 es inmutable y opaco. Su huella SHA-256
// identifica el plan; no sustituye el sello criptografico de la evidencia.
type ManifiestoEjecucionDocumentalV3 struct {
	datos *DatosManifiestoEjecucionDocumentalV3
}

func NuevoManifiestoEjecucionDocumentalV3(
	consulta ConsultaFormatoDocumental,
	descriptor DescriptorPerfilDocumental,
	situacion domain.SituacionOperativaPerfilDocumental,
	render, verificador, semantico DescriptorComponenteDocumentalAtestado,
	borradorRef, efectoRef, huellaEntradaHMAC string,
	limiteEfectivoBytes uint64,
) (ManifiestoEjecucionDocumentalV3, error) {
	datos := DatosManifiestoEjecucionDocumentalV3{
		Esquema:  EsquemaManifiestoEjecucionDocumentalV3,
		Consulta: consulta, DescriptorPerfil: descriptor, SituacionOperativa: situacion,
		ComponenteRender: render, ComponenteVerificador: verificador,
		ComponenteSemantico: semantico,
		BorradorRef:         borradorRef, EfectoRef: efectoRef,
		HuellaEntradaHMAC: huellaEntradaHMAC, LimiteEfectivoBytes: limiteEfectivoBytes,
	}
	datos.HuellaPlanSHA256 = huellaManifiestoEjecucionDocumentalV3(datos)
	manifiesto := ManifiestoEjecucionDocumentalV3{datos: &datos}
	if manifiesto.Validar() != nil {
		return ManifiestoEjecucionDocumentalV3{}, ErrManifiestoEjecucionDocumentalV3Invalido
	}
	return manifiesto, nil
}

func (m ManifiestoEjecucionDocumentalV3) Validar() error {
	if m.datos == nil {
		return ErrManifiestoEjecucionDocumentalV3Invalido
	}
	d := *m.datos
	perfil := d.DescriptorPerfil.Perfil()
	consultaRender := consultaComponenteEjecucionDocumentalV3(
		d.DescriptorPerfil, domain.RolComponenteRenderizador,
	)
	consultaVerificador := consultaComponenteEjecucionDocumentalV3(
		d.DescriptorPerfil, domain.RolComponenteValidadorEstructural,
	)
	consultaSemantico := consultaComponenteEjecucionDocumentalV3(
		d.DescriptorPerfil, domain.RolComponenteVerificadorSemantico,
	)
	if d.Esquema != EsquemaManifiestoEjecucionDocumentalV3 || d.Consulta.Validar() != nil ||
		d.DescriptorPerfil.Validar() != nil || !d.DescriptorPerfil.Coincide(d.Consulta) ||
		d.SituacionOperativa.Validar() != nil ||
		d.SituacionOperativa.PublicacionRef() != d.DescriptorPerfil.PublicacionRef() ||
		d.SituacionOperativa.PerfilRef() != perfil.Referencia() ||
		d.SituacionOperativa.DigestPerfilSHA256() != perfil.DigestSHA256() ||
		d.SituacionOperativa.RevisionCatalogo() != d.DescriptorPerfil.Revision() ||
		!d.SituacionOperativa.AutorizaEjecucion(perfil, d.DescriptorPerfil.Revision()) ||
		d.ComponenteRender.Validar() != nil || d.ComponenteVerificador.Validar() != nil ||
		d.ComponenteSemantico.Validar() != nil ||
		d.ComponenteRender.Componente().Rol() != domain.RolComponenteRenderizador ||
		d.ComponenteVerificador.Componente().Rol() != domain.RolComponenteValidadorEstructural ||
		d.ComponenteSemantico.Componente().Rol() != domain.RolComponenteVerificadorSemantico ||
		!d.ComponenteRender.Coincide(consultaRender) ||
		!d.ComponenteVerificador.Coincide(consultaVerificador) ||
		!d.ComponenteSemantico.Coincide(consultaSemantico) ||
		!d.ComponenteRender.IndependienteDe(d.ComponenteVerificador) ||
		!d.ComponenteRender.IndependienteDe(d.ComponenteSemantico) ||
		!d.ComponenteVerificador.IndependienteDe(d.ComponenteSemantico) ||
		!referenciaEjecucionDocumentalV3Valida(d.BorradorRef) ||
		!referenciaEjecucionDocumentalV3Valida(d.EfectoRef) || d.BorradorRef == d.EfectoRef ||
		!hmacSHA256PuertoValido(d.HuellaEntradaHMAC) || d.LimiteEfectivoBytes == 0 ||
		d.LimiteEfectivoBytes > tamanoMaximoSalidaEjecucionDocumentalV3 ||
		d.LimiteEfectivoBytes > perfil.MaximoBytes() ||
		d.LimiteEfectivoBytes > d.ComponenteRender.MaximoBytes() ||
		d.LimiteEfectivoBytes > d.ComponenteVerificador.MaximoBytes() ||
		d.LimiteEfectivoBytes > d.ComponenteSemantico.MaximoBytes() ||
		!esSHA256Hexadecimal(d.HuellaPlanSHA256) ||
		huellaManifiestoEjecucionDocumentalV3(d) != d.HuellaPlanSHA256 {
		return ErrManifiestoEjecucionDocumentalV3Invalido
	}
	return nil
}

func (m ManifiestoEjecucionDocumentalV3) Datos() (DatosManifiestoEjecucionDocumentalV3, error) {
	if m.Validar() != nil {
		return DatosManifiestoEjecucionDocumentalV3{}, ErrManifiestoEjecucionDocumentalV3Invalido
	}
	return *m.datos, nil
}

func (m ManifiestoEjecucionDocumentalV3) HuellaSHA256() (string, error) {
	datos, err := m.Datos()
	if err != nil {
		return "", err
	}
	return datos.HuellaPlanSHA256, nil
}

func (ManifiestoEjecucionDocumentalV3) String() string {
	return "[MANIFIESTO-EJECUCION-DOCUMENTAL-V3-OPACO]"
}

func (m ManifiestoEjecucionDocumentalV3) GoString() string { return m.String() }
func (m ManifiestoEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (m ManifiestoEjecucionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(m.String())
}
func (ManifiestoEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ManifiestoEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ManifiestoEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ManifiestoEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ManifiestoEjecucionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ManifiestoEjecucionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudPrepararEjecucionDocumentalV3 struct {
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	SolicitadaEn           time.Time
	ExpiraEn               time.Time
}

func (s SolicitudPrepararEjecucionDocumentalV3) Validar() error {
	datos, err := s.Manifiesto.Datos()
	if !hmacSHA256PuertoValido(s.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(s.HuellaSolicitudHMAC) || err != nil ||
		!documentalcanonico.ClavesHMACSHA256V3Distintas(
			s.IndiceIdempotenciaHMAC, s.HuellaSolicitudHMAC, datos.HuellaEntradaHMAC,
		) || !instanteEjecucionDocumentalV3Valido(s.SolicitadaEn) ||
		!instanteEjecucionDocumentalV3Valido(s.ExpiraEn) ||
		!s.ExpiraEn.After(s.SolicitadaEn) ||
		s.ExpiraEn.Sub(s.SolicitadaEn) > duracionMaximaReservaEjecucionDocumentalV3 {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

func (SolicitudPrepararEjecucionDocumentalV3) String() string {
	return "[SOLICITUD-PREPARAR-EJECUCION-DOCUMENTAL-V3-HMAC-REDACTADA]"
}
func (s SolicitudPrepararEjecucionDocumentalV3) GoString() string { return s.String() }
func (s SolicitudPrepararEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudPrepararEjecucionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudPrepararEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudPrepararEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudPrepararEjecucionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudPrepararEjecucionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type EstadoEjecucionDocumentalV3 string

const (
	EstadoEjecucionDocumentalV3Preparada           EstadoEjecucionDocumentalV3 = "preparada"
	EstadoEjecucionDocumentalV3Activa              EstadoEjecucionDocumentalV3 = "activa"
	EstadoEjecucionDocumentalV3EfectoIniciado      EstadoEjecucionDocumentalV3 = "efecto_iniciado"
	EstadoEjecucionDocumentalV3Indeterminada       EstadoEjecucionDocumentalV3 = "indeterminada"
	EstadoEjecucionDocumentalV3Confirmada          EstadoEjecucionDocumentalV3 = "confirmada"
	EstadoEjecucionDocumentalV3AbandonadaSinEfecto EstadoEjecucionDocumentalV3 = "abandonada_sin_efecto"
)

func (e EstadoEjecucionDocumentalV3) Valido() bool {
	switch e {
	case EstadoEjecucionDocumentalV3Preparada, EstadoEjecucionDocumentalV3Activa,
		EstadoEjecucionDocumentalV3EfectoIniciado, EstadoEjecucionDocumentalV3Indeterminada,
		EstadoEjecucionDocumentalV3Confirmada, EstadoEjecucionDocumentalV3AbandonadaSinEfecto:
		return true
	default:
		return false
	}
}

func (e EstadoEjecucionDocumentalV3) PuedeTransicionarA(siguiente EstadoEjecucionDocumentalV3) bool {
	switch e {
	case EstadoEjecucionDocumentalV3Preparada:
		return siguiente == EstadoEjecucionDocumentalV3Activa ||
			siguiente == EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	case EstadoEjecucionDocumentalV3Activa:
		return siguiente == EstadoEjecucionDocumentalV3EfectoIniciado ||
			siguiente == EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	case EstadoEjecucionDocumentalV3EfectoIniciado:
		return siguiente == EstadoEjecucionDocumentalV3Confirmada ||
			siguiente == EstadoEjecucionDocumentalV3Indeterminada
	case EstadoEjecucionDocumentalV3Indeterminada:
		return siguiente == EstadoEjecucionDocumentalV3Confirmada ||
			siguiente == EstadoEjecucionDocumentalV3AbandonadaSinEfecto
	default:
		return false
	}
}

type PreparacionEjecucionDocumentalV3Nominal struct {
	ReservaRef  string
	BorradorRef string
	EfectoRef   string
	Repetida    bool
	Estado      EstadoEjecucionDocumentalV3
}

func (p PreparacionEjecucionDocumentalV3Nominal) ValidarContra(
	s SolicitudPrepararEjecucionDocumentalV3,
) error {
	datos, err := s.Manifiesto.Datos()
	if s.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(p.ReservaRef) ||
		err != nil || p.BorradorRef != datos.BorradorRef || p.EfectoRef != datos.EfectoRef ||
		!p.Estado.Valido() || (!p.Repetida && p.Estado != EstadoEjecucionDocumentalV3Preparada) {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

// ConsumoDecisionEjecucionDocumentalV3 se reclama con UNIQUE(DecisionRef) en
// la misma transaccion que activa la reserva. Abandono o caducidad no liberan
// nunca esa referencia para otra ejecucion.
type ConsumoDecisionEjecucionDocumentalV3 struct {
	DecisionRef           string
	EfectoRef             string
	EsquemaHuellaDecision string
	HuellaDecisionSHA256  string
	HuellaPlanSHA256      string
}

func (c ConsumoDecisionEjecucionDocumentalV3) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
) error {
	datos, err := manifiesto.Datos()
	if err != nil || !referenciaEjecucionDocumentalV3Valida(c.DecisionRef) ||
		!referenciaEjecucionDocumentalV3Valida(c.EfectoRef) ||
		c.EfectoRef != datos.EfectoRef ||
		c.EsquemaHuellaDecision != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!esSHA256Hexadecimal(c.HuellaDecisionSHA256) ||
		c.HuellaPlanSHA256 != datos.HuellaPlanSHA256 {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

// VinculoEstableActivacionDocumentalV3 es el DTO nominal unico que compromete
// la intencion estable de activacion. Excluye ActivadaEn deliberadamente para
// que un reintento posterior recupere el mismo cercado. Sus campos son
// inspeccionables, pero su forma y huella no conceden autoridad: el registro
// debe reconstruirlo desde las filas V3/V4 durables y compararlo dentro de la
// transaccion que autoriza el efecto.
type VinculoEstableActivacionDocumentalV3 struct {
	ReservaRef               string
	IndiceIdempotenciaHMAC   string
	HuellaSolicitudHMAC      string
	Manifiesto               ManifiestoEjecucionDocumentalV3
	ConsumoDecision          ConsumoDecisionEjecucionDocumentalV3
	OrdenConsumoDurableV4Ref string
}

func (v VinculoEstableActivacionDocumentalV3) Validar() error {
	proyeccion, valida := proyectarVinculoActivacionDocumentalV3(v)
	if !valida || !proyeccion.Validar() {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

func (v VinculoEstableActivacionDocumentalV3) HuellaSHA256() (string, error) {
	if v.Validar() != nil {
		return "", ErrReservaEjecucionDocumentalV3Invalida
	}
	proyeccion, valida := proyectarVinculoActivacionDocumentalV3(v)
	if !valida {
		return "", ErrReservaEjecucionDocumentalV3Invalida
	}
	return proyeccion.HuellaSHA256(), nil
}

func (VinculoEstableActivacionDocumentalV3) String() string {
	return "[VINCULO-ESTABLE-ACTIVACION-DOCUMENTAL-V3-NOMINAL-REDACTADO]"
}
func (v VinculoEstableActivacionDocumentalV3) GoString() string { return v.String() }
func (v VinculoEstableActivacionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}
func (v VinculoEstableActivacionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(v.String())
}
func (VinculoEstableActivacionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculoEstableActivacionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (VinculoEstableActivacionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculoEstableActivacionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (VinculoEstableActivacionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*VinculoEstableActivacionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// TokenCercadoEjecucionDocumentalV3Nominal combina un valor aleatorio con una
// secuencia monotona y una MAC restaurable. La secuencia de cercado es distinta
// de la secuencia operativa del perfil, aunque la huella liga ambas de forma
// inseparable. Su construccion publica solo acredita forma nominal: el servicio
// privado debe verificar la MAC con la clave gestionada antes de cualquier uso.
type TokenCercadoEjecucionDocumentalV3Nominal struct {
	valor                 string
	secuencia             uint64
	vinculoEstable        VinculoEstableActivacionDocumentalV3
	huellaVinculoEstable  string
	huellaVinculo         string
	claveAtestacionRef    string
	revisionClave         uint64
	macAtestacion         []byte
	evidenciaOperacionRef string
}

// NuevoTokenCercadoEjecucionDocumentalV3Nominal queda reservado al adaptador de
// RegistroEjecucionesDocumentalesV3. Solo construye el sobre: NO autentica el
// token. El MAC compromete la huella canonica del vinculo estable; el registro
// debe comprobarlo con su clave gestionada antes de cualquier efecto.
func NuevoTokenCercadoEjecucionDocumentalV3Nominal(
	valor string,
	secuencia uint64,
	vinculo VinculoEstableActivacionDocumentalV3,
	claveAtestacionRef string,
	revisionClave uint64,
	macAtestacion []byte,
	evidenciaOperacionRef string,
) (TokenCercadoEjecucionDocumentalV3Nominal, error) {
	huellaVinculoEstable, errVinculo := vinculo.HuellaSHA256()
	if errVinculo != nil {
		return TokenCercadoEjecucionDocumentalV3Nominal{}, ErrTokenCercadoDocumentalV3Invalido
	}
	token := TokenCercadoEjecucionDocumentalV3Nominal{
		valor: valor, secuencia: secuencia,
		vinculoEstable:       vinculo,
		huellaVinculoEstable: huellaVinculoEstable,
		huellaVinculo: huellaVinculoCercadoEjecucionDocumentalV3(
			secuencia, huellaVinculoEstable,
		),
		claveAtestacionRef:    claveAtestacionRef,
		revisionClave:         revisionClave,
		macAtestacion:         append([]byte(nil), macAtestacion...),
		evidenciaOperacionRef: evidenciaOperacionRef,
	}
	if token.ValidarPara(vinculo) != nil {
		return TokenCercadoEjecucionDocumentalV3Nominal{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return token, nil
}

func (t TokenCercadoEjecucionDocumentalV3Nominal) ValidarPara(
	vinculo VinculoEstableActivacionDocumentalV3,
) error {
	proyeccion, valida := proyectarTokenCercadoDocumentalV3(t, vinculo)
	if !valida || !proyeccion.Validar() {
		return ErrTokenCercadoDocumentalV3Invalido
	}
	return nil
}

func (t TokenCercadoEjecucionDocumentalV3Nominal) Secuencia() uint64 { return t.secuencia }
func (t TokenCercadoEjecucionDocumentalV3Nominal) RevisionClaveGestionada() uint64 {
	return t.revisionClave
}
func (t TokenCercadoEjecucionDocumentalV3Nominal) HuellaVinculoSHA256() string {
	return t.huellaVinculo
}
func (TokenCercadoEjecucionDocumentalV3Nominal) String() string {
	return "[TOKEN-CERCADO-DOCUMENTAL-V3-NOMINAL-NO-AUTORITATIVO-REDACTADO]"
}
func (t TokenCercadoEjecucionDocumentalV3Nominal) GoString() string { return t.String() }
func (t TokenCercadoEjecucionDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (t TokenCercadoEjecucionDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(t.String())
}
func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (TokenCercadoEjecucionDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*TokenCercadoEjecucionDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudVerificacionTokenCercadoDocumentalV3 struct {
	vinculo VinculoEstableActivacionDocumentalV3
	token   TokenCercadoEjecucionDocumentalV3Nominal
	mensaje []byte
	huella  string
}

func NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) (SolicitudVerificacionTokenCercadoDocumentalV3, error) {
	if token.ValidarPara(vinculo) != nil {
		return SolicitudVerificacionTokenCercadoDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	mensaje := serializarAtestacionTokenCercadoDocumentalV3(vinculo, token)
	solicitud := SolicitudVerificacionTokenCercadoDocumentalV3{
		vinculo: vinculo, token: token,
		mensaje: append([]byte(nil), mensaje...),
		huella:  huellaSolicitudVerificacionTokenCercadoDocumentalV3(mensaje, token.macAtestacion),
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificacionTokenCercadoDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return solicitud, nil
}

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Validar() error {
	if s.token.ValidarPara(s.vinculo) != nil ||
		len(s.mensaje) == 0 || !esSHA256Hexadecimal(s.huella) ||
		huellaSolicitudVerificacionTokenCercadoDocumentalV3(
			s.mensaje, s.token.macAtestacion,
		) != s.huella ||
		string(s.mensaje) != string(serializarAtestacionTokenCercadoDocumentalV3(
			s.vinculo, s.token,
		)) {
		return ErrTokenCercadoDocumentalV3Invalido
	}
	return nil
}

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrTokenCercadoDocumentalV3Invalido
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (s SolicitudVerificacionTokenCercadoDocumentalV3) MAC() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrTokenCercadoDocumentalV3Invalido
	}
	return append([]byte(nil), s.token.macAtestacion...), nil
}

func (s SolicitudVerificacionTokenCercadoDocumentalV3) ClaveAtestacionRef() (string, error) {
	if s.Validar() != nil {
		return "", ErrTokenCercadoDocumentalV3Invalido
	}
	return s.token.claveAtestacionRef, nil
}

func (SolicitudVerificacionTokenCercadoDocumentalV3) String() string {
	return "[SOLICITUD-VERIFICACION-TOKEN-CERCADO-V3-CONFIDENCIAL]"
}
func (s SolicitudVerificacionTokenCercadoDocumentalV3) GoString() string { return s.String() }
func (s SolicitudVerificacionTokenCercadoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudVerificacionTokenCercadoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionTokenCercadoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionTokenCercadoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// PruebaCrudaAtestacionDespachoDocumentalV3 transporta material suficiente
// para que un adaptador criptografico real compruebe una HMAC. Es restaurable
// y, por tanto, nunca acredita por si misma que la comprobacion haya ocurrido.
type PruebaCrudaAtestacionDespachoDocumentalV3 struct {
	algoritmo               string
	audiencia               string
	contexto                string
	claveGestionadaRef      string
	revisionClaveGestionada uint64
	evidenciaOperacionRef   string
	mensajeCanonico         []byte
	sobreCriptografico      []byte
	huellaMensajeSHA256     string
	huellaSobreSHA256       string
}

func NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	evidenciaOperacionRef string,
	mensajeCanonico, sobreCriptografico []byte,
) (PruebaCrudaAtestacionDespachoDocumentalV3, error) {
	prueba := PruebaCrudaAtestacionDespachoDocumentalV3{
		algoritmo: algoritmo, audiencia: audiencia, contexto: contexto,
		claveGestionadaRef:      claveGestionadaRef,
		revisionClaveGestionada: revisionClaveGestionada,
		evidenciaOperacionRef:   evidenciaOperacionRef,
		mensajeCanonico:         append([]byte(nil), mensajeCanonico...),
		sobreCriptografico:      append([]byte(nil), sobreCriptografico...),
		huellaMensajeSHA256:     documentalcanonico.HuellaBytesSHA256(mensajeCanonico),
		huellaSobreSHA256:       documentalcanonico.HuellaBytesSHA256(sobreCriptografico),
	}
	if prueba.Validar() != nil {
		return PruebaCrudaAtestacionDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return prueba, nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) Validar() error {
	if !proyectarPruebaAtestacionDespachoDocumentalV3(p).Validar() {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) Perfil() (
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	err error,
) {
	if p.Validar() != nil {
		return "", "", "", "", 0, ErrOrdenDespachoDocumentalV3Invalida
	}
	return p.algoritmo, p.audiencia, p.contexto, p.claveGestionadaRef,
		p.revisionClaveGestionada, nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) MensajeCanonico() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), p.mensajeCanonico...), nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) SobreCriptografico() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), p.sobreCriptografico...), nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) EvidenciaOperacionRef() (string, error) {
	if p.Validar() != nil {
		return "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return p.evidenciaOperacionRef, nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) HuellasSHA256() (
	mensaje, sobre string,
	err error,
) {
	if p.Validar() != nil {
		return "", "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return p.huellaMensajeSHA256, p.huellaSobreSHA256, nil
}

func (p PruebaCrudaAtestacionDespachoDocumentalV3) huellaSHA256() string {
	return proyectarPruebaAtestacionDespachoDocumentalV3(p).HuellaSHA256()
}

func (PruebaCrudaAtestacionDespachoDocumentalV3) String() string {
	return "[PRUEBA-CRUDA-ATESTACION-DESPACHO-V3-NOMINAL-NO-AUTORITATIVA-REDACTADA]"
}
func (p PruebaCrudaAtestacionDespachoDocumentalV3) GoString() string { return p.String() }
func (p PruebaCrudaAtestacionDespachoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PruebaCrudaAtestacionDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PruebaCrudaAtestacionDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PruebaCrudaAtestacionDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type MetadatosComprobacionTokenCercadoDocumentalV3Nominal struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal solo correlaciona una
// solicitud con referencia e instante. Es una fabrica publica autocreable y,
// por tanto, estos metadatos son nominales y NO acreditan que se
// verificara el MAC. Nunca habilitan el inicio ni otro efecto por si solos.
func NuevosMetadatosComprobacionTokenCercadoDocumentalV3Nominal(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionTokenCercadoDocumentalV3Nominal, error) {
	proyeccion := documentalcanonico.DatosMetadatosComprobacionV3{
		HuellaSolicitud: solicitud.huella, HuellaSolicitudEsperada: solicitud.huella,
		VerificacionRef: verificacionRef, VerificadaEn: verificadaEn,
	}
	if solicitud.Validar() != nil || !proyeccion.Validar() {
		return MetadatosComprobacionTokenCercadoDocumentalV3Nominal{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return MetadatosComprobacionTokenCercadoDocumentalV3Nominal{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}, nil
}

func (r MetadatosComprobacionTokenCercadoDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
) error {
	proyeccion := documentalcanonico.DatosMetadatosComprobacionV3{
		HuellaSolicitud: r.huellaSolicitud, HuellaSolicitudEsperada: solicitud.huella,
		VerificacionRef: r.verificacionRef, VerificadaEn: r.verificadaEn,
	}
	if solicitud.Validar() != nil || !proyeccion.Validar() {
		return ErrTokenCercadoDocumentalV3Invalido
	}
	return nil
}

func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) String() string {
	return "[METADATOS-COMPROBACION-TOKEN-CERCADO-V3-NOMINALES-NO-AUTORITATIVOS]"
}
func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) GoString() string { return m.String() }
func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (m MetadatosComprobacionTokenCercadoDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(m.String())
}
func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionTokenCercadoDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionTokenCercadoDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudActivarEjecucionDocumentalV3 struct {
	ReservaRef             string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	// OrdenConsumoDurableV4Ref es una referencia nominal, nunca autoridad.
	// El registro debe releer la orden V4 y adoptarla en su propio COMMIT.
	OrdenConsumoDurableV4Ref string
	ActivadaEn               time.Time
}

func (s SolicitudActivarEjecucionDocumentalV3) Validar() error {
	if _, err := s.VinculoEstable(); err != nil ||
		!instanteEjecucionDocumentalV3Valido(s.ActivadaEn) {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

func (s SolicitudActivarEjecucionDocumentalV3) VinculoEstable() (
	VinculoEstableActivacionDocumentalV3,
	error,
) {
	vinculo := VinculoEstableActivacionDocumentalV3{
		ReservaRef: s.ReservaRef, IndiceIdempotenciaHMAC: s.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC: s.HuellaSolicitudHMAC, Manifiesto: s.Manifiesto,
		ConsumoDecision:          s.ConsumoDecision,
		OrdenConsumoDurableV4Ref: s.OrdenConsumoDurableV4Ref,
	}
	if vinculo.Validar() != nil {
		return VinculoEstableActivacionDocumentalV3{}, ErrReservaEjecucionDocumentalV3Invalida
	}
	return vinculo, nil
}

func (SolicitudActivarEjecucionDocumentalV3) String() string {
	return "[SOLICITUD-ACTIVAR-EJECUCION-DOCUMENTAL-V3-REDACTADA]"
}

func (s SolicitudActivarEjecucionDocumentalV3) GoString() string { return s.String() }
func (s SolicitudActivarEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudActivarEjecucionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudActivarEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudActivarEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudActivarEjecucionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudActivarEjecucionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type ActivacionEjecucionDocumentalV3Nominal struct {
	Token    TokenCercadoEjecucionDocumentalV3Nominal
	Repetida bool
}

func (a ActivacionEjecucionDocumentalV3Nominal) ValidarContra(s SolicitudActivarEjecucionDocumentalV3) error {
	vinculo, err := s.VinculoEstable()
	if s.Validar() != nil || err != nil || a.Token.ValidarPara(vinculo) != nil {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

type SolicitudIniciarEfectoDocumentalV3 struct {
	VinculoActivacion VinculoEstableActivacionDocumentalV3
	Token             TokenCercadoEjecucionDocumentalV3Nominal
	IniciadoEn        time.Time
}

func (s SolicitudIniciarEfectoDocumentalV3) Validar() error {
	if !instanteEjecucionDocumentalV3Valido(s.IniciadoEn) ||
		s.VinculoActivacion.Validar() != nil || s.Token.ValidarPara(s.VinculoActivacion) != nil {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

func (SolicitudIniciarEfectoDocumentalV3) String() string {
	return "[SOLICITUD-INICIAR-EFECTO-DOCUMENTAL-V3-ESTRUCTURAL-REDACTADA]"
}
func (s SolicitudIniciarEfectoDocumentalV3) GoString() string { return s.String() }
func (s SolicitudIniciarEfectoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudIniciarEfectoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudIniciarEfectoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudIniciarEfectoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudIniciarEfectoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudIniciarEfectoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// DatosReciboInicioEfectoDocumentalV3Nominal es la proyeccion nominal del COMMIT que
// hizo el CAS activa -> iniciada. Puede persistirse como columnas, pero una
// copia construida por el llamador nunca concede autoridad de despacho.
type DatosReciboInicioEfectoDocumentalV3Nominal struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	AtestacionInicio           PruebaCrudaAtestacionDespachoDocumentalV3
	IniciadoEn                 time.Time
}

func (DatosReciboInicioEfectoDocumentalV3Nominal) String() string {
	return "[DATOS-RECIBO-INICIO-EFECTO-V3-NOMINALES-HMAC-REDACTADOS]"
}
func (d DatosReciboInicioEfectoDocumentalV3Nominal) GoString() string { return d.String() }
func (d DatosReciboInicioEfectoDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosReciboInicioEfectoDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosReciboInicioEfectoDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosReciboInicioEfectoDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// ReciboInicioEfectoDocumentalV3Nominal es un recibo durable nominal y atestado. Lo
// devuelve el registro en el mismo COMMIT del inicio, pero solo la posterior
// relectura y reclamacion CAS de su outbox puede convertirlo en candidato a
// despacho. Su fabrica publica restaura datos; no certifica su procedencia.
type ReciboInicioEfectoDocumentalV3Nominal struct {
	datos *DatosReciboInicioEfectoDocumentalV3Nominal
}

func MensajeCanonicoAtestacionInicioEfectoDocumentalV3(
	solicitud SolicitudIniciarEfectoDocumentalV3,
	inicioRef string,
	versionInicioCAS uint64,
	auditoriaInicioRef, outboxInicioRef, evidenciaOperacionRef string,
) ([]byte, error) {
	huellaVinculo, err := solicitud.VinculoActivacion.HuellaSHA256()
	if solicitud.Validar() != nil || err != nil {
		return nil, ErrReciboInicioDocumentalV3Invalido
	}
	proyeccion := documentalcanonico.DatosAtestacionInicioEfectoV3{
		InicioRef: inicioRef, ReservaRef: solicitud.VinculoActivacion.ReservaRef,
		HuellaVinculoEstableSHA256: huellaVinculo,
		SecuenciaCercado:           solicitud.Token.Secuencia(),
		HuellaVinculoCercadoSHA256: solicitud.Token.HuellaVinculoSHA256(),
		OrdenConsumoDurableV4Ref:   solicitud.VinculoActivacion.OrdenConsumoDurableV4Ref,
		VersionInicioCAS:           versionInicioCAS, AuditoriaInicioRef: auditoriaInicioRef,
		OutboxInicioRef: outboxInicioRef, ClaveAtestacionRef: solicitud.Token.claveAtestacionRef,
		RevisionClave: solicitud.Token.revisionClave, EvidenciaOperacionRef: evidenciaOperacionRef,
		IniciadoEn: solicitud.IniciadoEn,
	}
	mensaje := proyeccion.Bytes()
	if len(mensaje) == 0 {
		return nil, ErrReciboInicioDocumentalV3Invalido
	}
	return mensaje, nil
}

func NuevoReciboInicioEfectoDocumentalV3Nominal(
	solicitud SolicitudIniciarEfectoDocumentalV3,
	inicioRef string,
	versionInicioCAS uint64,
	auditoriaInicioRef, outboxInicioRef string,
	atestacionInicio PruebaCrudaAtestacionDespachoDocumentalV3,
) (ReciboInicioEfectoDocumentalV3Nominal, error) {
	huellaVinculo, err := solicitud.VinculoActivacion.HuellaSHA256()
	evidenciaRef, errEvidencia := atestacionInicio.EvidenciaOperacionRef()
	mensajeEsperado, errMensaje := MensajeCanonicoAtestacionInicioEfectoDocumentalV3(
		solicitud, inicioRef, versionInicioCAS, auditoriaInicioRef, outboxInicioRef,
		evidenciaRef,
	)
	algoritmo, audiencia, contexto, claveRef, revisionClave, errPerfil := atestacionInicio.Perfil()
	mensajeAtestacion, errMaterial := atestacionInicio.MensajeCanonico()
	if solicitud.Validar() != nil || err != nil || errEvidencia != nil || errMensaje != nil ||
		errPerfil != nil || errMaterial != nil ||
		algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 ||
		audiencia != AudienciaAtestacionInicioEfectoV3 ||
		contexto != ContextoAtestacionInicioEfectoV3 ||
		claveRef != solicitud.Token.claveAtestacionRef ||
		revisionClave != solicitud.Token.revisionClave ||
		!documentalcanonico.BytesIguales(mensajeAtestacion, mensajeEsperado) {
		return ReciboInicioEfectoDocumentalV3Nominal{}, ErrReciboInicioDocumentalV3Invalido
	}
	datos := &DatosReciboInicioEfectoDocumentalV3Nominal{
		InicioRef: inicioRef, ReservaRef: solicitud.VinculoActivacion.ReservaRef,
		HuellaVinculoEstableSHA256: huellaVinculo,
		SecuenciaCercado:           solicitud.Token.Secuencia(),
		HuellaVinculoCercadoSHA256: solicitud.Token.HuellaVinculoSHA256(),
		OrdenConsumoDurableV4Ref:   solicitud.VinculoActivacion.OrdenConsumoDurableV4Ref,
		VersionInicioCAS:           versionInicioCAS, AuditoriaInicioRef: auditoriaInicioRef,
		OutboxInicioRef:  outboxInicioRef,
		AtestacionInicio: clonarPruebaCrudaAtestacionDespachoDocumentalV3(atestacionInicio),
		IniciadoEn:       solicitud.IniciadoEn,
	}
	recibo := ReciboInicioEfectoDocumentalV3Nominal{datos: datos}
	if recibo.ValidarContra(solicitud) != nil {
		return ReciboInicioEfectoDocumentalV3Nominal{}, ErrReciboInicioDocumentalV3Invalido
	}
	return recibo, nil
}

func (r ReciboInicioEfectoDocumentalV3Nominal) Validar() error {
	if r.datos == nil {
		return ErrReciboInicioDocumentalV3Invalido
	}
	proyeccion, valida := proyectarReciboInicioEfectoDocumentalV3(*r.datos)
	if !valida || !proyeccion.Validar() {
		return ErrReciboInicioDocumentalV3Invalido
	}
	return nil
}

func (r ReciboInicioEfectoDocumentalV3Nominal) ValidarContra(
	solicitud SolicitudIniciarEfectoDocumentalV3,
) error {
	// Validar primero el receptor opaco evita desreferenciar datos restaurados
	// nulos. El valor cero debe fallar cerrado igual que cualquier recibo
	// malformado, nunca provocar panic en un adaptador o registro.
	if r.Validar() != nil || solicitud.Validar() != nil {
		return ErrReciboInicioDocumentalV3Invalido
	}
	datos := r.datos
	huellaVinculo, err := solicitud.VinculoActivacion.HuellaSHA256()
	evidenciaRef, errEvidencia := datos.AtestacionInicio.EvidenciaOperacionRef()
	mensajeEsperado, errMensaje := MensajeCanonicoAtestacionInicioEfectoDocumentalV3(
		solicitud, datos.InicioRef, datos.VersionInicioCAS,
		datos.AuditoriaInicioRef, datos.OutboxInicioRef, evidenciaRef,
	)
	mensajeAtestacion, errMaterial := datos.AtestacionInicio.MensajeCanonico()
	if err != nil ||
		errEvidencia != nil || errMensaje != nil || errMaterial != nil ||
		!documentalcanonico.BytesIguales(mensajeAtestacion, mensajeEsperado) ||
		datos.ReservaRef != solicitud.VinculoActivacion.ReservaRef ||
		datos.HuellaVinculoEstableSHA256 != huellaVinculo ||
		datos.SecuenciaCercado != solicitud.Token.Secuencia() ||
		datos.HuellaVinculoCercadoSHA256 != solicitud.Token.HuellaVinculoSHA256() ||
		datos.OrdenConsumoDurableV4Ref != solicitud.VinculoActivacion.OrdenConsumoDurableV4Ref ||
		!datos.IniciadoEn.Equal(solicitud.IniciadoEn) {
		return ErrReciboInicioDocumentalV3Invalido
	}
	return nil
}

func (r ReciboInicioEfectoDocumentalV3Nominal) Datos() (DatosReciboInicioEfectoDocumentalV3Nominal, error) {
	if r.Validar() != nil {
		return DatosReciboInicioEfectoDocumentalV3Nominal{}, ErrReciboInicioDocumentalV3Invalido
	}
	datos := *r.datos
	datos.AtestacionInicio = clonarPruebaCrudaAtestacionDespachoDocumentalV3(r.datos.AtestacionInicio)
	return datos, nil
}

func (r ReciboInicioEfectoDocumentalV3Nominal) HuellaSHA256() (string, error) {
	if r.Validar() != nil {
		return "", ErrReciboInicioDocumentalV3Invalido
	}
	proyeccion, _ := proyectarReciboInicioEfectoDocumentalV3(*r.datos)
	return proyeccion.HuellaSHA256(), nil
}

func (ReciboInicioEfectoDocumentalV3Nominal) String() string {
	return "[RECIBO-INICIO-EFECTO-DOCUMENTAL-V3-NOMINAL-REDACTADO]"
}
func (r ReciboInicioEfectoDocumentalV3Nominal) GoString() string { return r.String() }
func (r ReciboInicioEfectoDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ReciboInicioEfectoDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ReciboInicioEfectoDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ReciboInicioEfectoDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ReciboInicioEfectoDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ReciboInicioEfectoDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// SolicitudReclamarOrdenDespachoDocumentalV3 identifica una unica reclamacion
// CAS del evento outbox de inicio. El llamador solo aporta referencias opacas;
// nunca un resultado de verificacion ni un token.
type SolicitudReclamarOrdenDespachoDocumentalV3 = documentalcanonico.DatosSolicitudReclamacionV3

type DatosOrdenDespachoDocumentalV3Nominal struct {
	ReciboInicio             DatosReciboInicioEfectoDocumentalV3Nominal
	HuellaReciboInicioSHA256 string
	ReclamacionRef           string
	ConsumidorRef            string
	VersionReclamacionCAS    uint64
	AuditoriaReclamacionRef  string
	AtestacionReclamacion    PruebaCrudaAtestacionDespachoDocumentalV3
	ReclamadaEn              time.Time
	ExpiraEn                 time.Time
}

func (DatosOrdenDespachoDocumentalV3Nominal) String() string {
	return "[DATOS-ORDEN-DESPACHO-V3-NOMINALES-HMAC-REDACTADOS]"
}
func (d DatosOrdenDespachoDocumentalV3Nominal) GoString() string { return d.String() }
func (d DatosOrdenDespachoDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosOrdenDespachoDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosOrdenDespachoDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosOrdenDespachoDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosOrdenDespachoDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosOrdenDespachoDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// OrdenDespachoDocumentalV3Nominal es la salida restaurable de la reclamacion
// CAS. Aunque contenga atestaciones, su fabrica publica nunca la promueve a
// autoridad. Solo sirve como entrada cruda al servicio de aplicacion privado.
type OrdenDespachoDocumentalV3Nominal struct {
	datos *DatosOrdenDespachoDocumentalV3Nominal
}

func MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
	recibo ReciboInicioEfectoDocumentalV3Nominal,
	solicitud SolicitudReclamarOrdenDespachoDocumentalV3,
	versionReclamacionCAS uint64,
	auditoriaReclamacionRef, evidenciaOperacionRef string,
) ([]byte, error) {
	datosRecibo, errDatos := recibo.Datos()
	huellaRecibo, errHuella := recibo.HuellaSHA256()
	algoritmo, _, _, claveRef, revisionClave, errPerfil := datosRecibo.AtestacionInicio.Perfil()
	if solicitud.Validar() != nil || errDatos != nil || errHuella != nil || errPerfil != nil ||
		algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	proyeccion := documentalcanonico.DatosAtestacionReclamacionV3{
		Solicitud:                proyectarSolicitudReclamacionDocumentalV3(solicitud),
		HuellaReciboInicioSHA256: huellaRecibo, InicioReciboRef: datosRecibo.InicioRef,
		OutboxInicioReciboRef: datosRecibo.OutboxInicioRef, IniciadoEn: datosRecibo.IniciadoEn,
		VersionReclamacionCAS:   versionReclamacionCAS,
		AuditoriaReclamacionRef: auditoriaReclamacionRef,
		ClaveAtestacionRef:      claveRef, RevisionClave: revisionClave,
		EvidenciaOperacionRef:      evidenciaOperacionRef,
		SecuenciaCercado:           datosRecibo.SecuenciaCercado,
		HuellaVinculoEstableSHA256: datosRecibo.HuellaVinculoEstableSHA256,
		HuellaVinculoCercadoSHA256: datosRecibo.HuellaVinculoCercadoSHA256,
		OrdenConsumoDurableV4Ref:   datosRecibo.OrdenConsumoDurableV4Ref,
	}
	mensaje := proyeccion.Bytes()
	if len(mensaje) == 0 {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return mensaje, nil
}

func NuevaOrdenDespachoDocumentalV3Nominal(
	recibo ReciboInicioEfectoDocumentalV3Nominal,
	solicitud SolicitudReclamarOrdenDespachoDocumentalV3,
	versionReclamacionCAS uint64,
	auditoriaReclamacionRef string,
	atestacionReclamacion PruebaCrudaAtestacionDespachoDocumentalV3,
) (OrdenDespachoDocumentalV3Nominal, error) {
	datosRecibo, errDatos := recibo.Datos()
	huellaRecibo, errHuella := recibo.HuellaSHA256()
	evidenciaRef, errEvidencia := atestacionReclamacion.EvidenciaOperacionRef()
	mensajeEsperado, errMensaje := MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
		recibo, solicitud, versionReclamacionCAS, auditoriaReclamacionRef, evidenciaRef,
	)
	algoritmo, audiencia, contexto, claveRef, revisionClave, errPerfil := atestacionReclamacion.Perfil()
	algoritmoInicio, _, _, claveInicioRef, revisionInicio, errPerfilInicio := datosRecibo.AtestacionInicio.Perfil()
	mensajeAtestacion, errMaterial := atestacionReclamacion.MensajeCanonico()
	if solicitud.Validar() != nil || errDatos != nil || errHuella != nil ||
		errEvidencia != nil || errMensaje != nil || errPerfil != nil || errPerfilInicio != nil ||
		errMaterial != nil || algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 ||
		algoritmoInicio != algoritmo || audiencia != AudienciaAtestacionReclamacionV3 ||
		contexto != ContextoAtestacionReclamacionV3 || claveRef != claveInicioRef ||
		revisionClave != revisionInicio ||
		!documentalcanonico.BytesIguales(mensajeAtestacion, mensajeEsperado) ||
		solicitud.InicioRef != datosRecibo.InicioRef || solicitud.OutboxRef != datosRecibo.OutboxInicioRef ||
		solicitud.ReclamadaEn.Before(datosRecibo.IniciadoEn) {
		return OrdenDespachoDocumentalV3Nominal{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	datos := &DatosOrdenDespachoDocumentalV3Nominal{
		ReciboInicio: datosRecibo, HuellaReciboInicioSHA256: huellaRecibo,
		ReclamacionRef: solicitud.ReclamacionRef, ConsumidorRef: solicitud.ConsumidorRef,
		VersionReclamacionCAS:   versionReclamacionCAS,
		AuditoriaReclamacionRef: auditoriaReclamacionRef,
		AtestacionReclamacion:   clonarPruebaCrudaAtestacionDespachoDocumentalV3(atestacionReclamacion),
		ReclamadaEn:             solicitud.ReclamadaEn, ExpiraEn: solicitud.ExpiraEn,
	}
	orden := OrdenDespachoDocumentalV3Nominal{datos: datos}
	if orden.Validar() != nil {
		return OrdenDespachoDocumentalV3Nominal{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return orden, nil
}

func (o OrdenDespachoDocumentalV3Nominal) Validar() error {
	if o.datos == nil {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	d := o.datos
	recibo := ReciboInicioEfectoDocumentalV3Nominal{datos: &d.ReciboInicio}
	huellaRecibo, err := recibo.HuellaSHA256()
	evidenciaRef, errEvidencia := d.AtestacionReclamacion.EvidenciaOperacionRef()
	solicitud := SolicitudReclamarOrdenDespachoDocumentalV3{
		ReclamacionRef: d.ReclamacionRef, InicioRef: d.ReciboInicio.InicioRef,
		OutboxRef: d.ReciboInicio.OutboxInicioRef, ConsumidorRef: d.ConsumidorRef,
		ReclamadaEn: d.ReclamadaEn, ExpiraEn: d.ExpiraEn,
	}
	mensajeEsperado, errMensaje := MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
		recibo, solicitud, d.VersionReclamacionCAS, d.AuditoriaReclamacionRef,
		evidenciaRef,
	)
	mensajeAtestacion, errMaterial := d.AtestacionReclamacion.MensajeCanonico()
	proyeccion := documentalcanonico.DatosOrdenDespachoV3{
		Solicitud:                   proyectarSolicitudReclamacionDocumentalV3(solicitud),
		HuellaReciboInicioSHA256:    d.HuellaReciboInicioSHA256,
		HuellaReciboCalculadaSHA256: huellaRecibo,
		VersionReclamacionCAS:       d.VersionReclamacionCAS,
		AuditoriaReclamacionRef:     d.AuditoriaReclamacionRef,
		EvidenciaOperacionRef:       evidenciaRef,
		AtestacionValida:            d.AtestacionReclamacion.Validar() == nil,
		HuellaAtestacionSHA256:      d.AtestacionReclamacion.huellaSHA256(),
		MensajeAtestacion:           mensajeAtestacion, MensajeEsperado: mensajeEsperado,
		IniciadoEn: d.ReciboInicio.IniciadoEn,
	}
	if err != nil || errEvidencia != nil || errMensaje != nil || errMaterial != nil ||
		!proyeccion.Validar() {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (o OrdenDespachoDocumentalV3Nominal) Datos() (DatosOrdenDespachoDocumentalV3Nominal, error) {
	if o.Validar() != nil {
		return DatosOrdenDespachoDocumentalV3Nominal{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	datos := *o.datos
	datos.ReciboInicio.AtestacionInicio = clonarPruebaCrudaAtestacionDespachoDocumentalV3(
		o.datos.ReciboInicio.AtestacionInicio,
	)
	datos.AtestacionReclamacion = clonarPruebaCrudaAtestacionDespachoDocumentalV3(
		o.datos.AtestacionReclamacion,
	)
	return datos, nil
}

func (o OrdenDespachoDocumentalV3Nominal) HuellaSHA256() (string, error) {
	if o.Validar() != nil {
		return "", ErrOrdenDespachoDocumentalV3Invalida
	}
	proyeccion, valida := proyectarOrdenDespachoDocumentalV3(*o.datos)
	if !valida {
		return "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return proyeccion.HuellaSHA256(), nil
}

func (OrdenDespachoDocumentalV3Nominal) String() string {
	return "[ORDEN-DESPACHO-DOCUMENTAL-V3-NOMINAL-REDACTADA]"
}
func (o OrdenDespachoDocumentalV3Nominal) GoString() string { return o.String() }
func (o OrdenDespachoDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, o.String())
}
func (o OrdenDespachoDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(o.String())
}
func (OrdenDespachoDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (OrdenDespachoDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (OrdenDespachoDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type VinculosCrudosVerificacionOrdenDespachoDocumentalV3 = documentalcanonico.VinculosMaterialDespachoV3

// MaterialCrudoVerificacionOrdenDespachoDocumentalV3 agrupa el mensaje
// canonico, tres pruebas HMAC y todos los vinculos durables que debe cotejar un
// adaptador. Es material nominal autocreable: verificarlo no concede autoridad
// fuera del servicio de aplicacion que ejecuta despues el consumo CAS.
type MaterialCrudoVerificacionOrdenDespachoDocumentalV3 struct {
	orden       OrdenDespachoDocumentalV3Nominal
	vinculo     VinculoEstableActivacionDocumentalV3
	token       TokenCercadoEjecucionDocumentalV3Nominal
	cercado     PruebaCrudaAtestacionDespachoDocumentalV3
	inicio      PruebaCrudaAtestacionDespachoDocumentalV3
	reclamacion PruebaCrudaAtestacionDespachoDocumentalV3
	vinculos    VinculosCrudosVerificacionOrdenDespachoDocumentalV3
	mensaje     []byte
	huella      string
}

func nuevoMaterialCrudoVerificacionOrdenDespachoDocumentalV3(
	orden OrdenDespachoDocumentalV3Nominal,
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) (MaterialCrudoVerificacionOrdenDespachoDocumentalV3, error) {
	datos, errDatos := orden.Datos()
	huellaOrden, errOrden := orden.HuellaSHA256()
	huellaVinculo, errVinculo := vinculo.HuellaSHA256()
	huellaRecibo := datos.HuellaReciboInicioSHA256
	if errDatos != nil || errOrden != nil || errVinculo != nil || token.ValidarPara(vinculo) != nil ||
		datos.ReciboInicio.ReservaRef != vinculo.ReservaRef ||
		datos.ReciboInicio.HuellaVinculoEstableSHA256 != huellaVinculo ||
		datos.ReciboInicio.HuellaVinculoCercadoSHA256 != token.HuellaVinculoSHA256() ||
		datos.ReciboInicio.SecuenciaCercado != token.Secuencia() ||
		datos.ReciboInicio.OrdenConsumoDurableV4Ref != vinculo.OrdenConsumoDurableV4Ref {
		return MaterialCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	solicitudToken, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(vinculo, token)
	if err != nil {
		return MaterialCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	mensajeToken, _ := solicitudToken.Mensaje()
	macToken, _ := solicitudToken.MAC()
	cercado, err := NuevaPruebaCrudaAtestacionDespachoDocumentalV3(
		AlgoritmoSelloEvidenciaHMACSHA256V3, AudienciaAtestacionTokenCercadoV3,
		ContextoAtestacionTokenCercadoV3, token.claveAtestacionRef,
		token.revisionClave, token.evidenciaOperacionRef, mensajeToken, macToken,
	)
	if err != nil {
		return MaterialCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	inicioRef, _ := datos.ReciboInicio.AtestacionInicio.EvidenciaOperacionRef()
	reclamacionRef, _ := datos.AtestacionReclamacion.EvidenciaOperacionRef()
	vinculos := VinculosCrudosVerificacionOrdenDespachoDocumentalV3{
		InicioRef: datos.ReciboInicio.InicioRef, AtestacionInicioRef: inicioRef,
		ReclamacionRef: datos.ReclamacionRef, AtestacionReclamacionRef: reclamacionRef,
		OrdenConsumoDurableV4Ref:  datos.ReciboInicio.OrdenConsumoDurableV4Ref,
		HuellaOrdenDespachoSHA256: huellaOrden, HuellaReciboInicioSHA256: huellaRecibo,
		HuellaVinculoEstableSHA256: huellaVinculo,
		HuellaVinculoCercadoSHA256: datos.ReciboInicio.HuellaVinculoCercadoSHA256,
		SecuenciaCercado:           datos.ReciboInicio.SecuenciaCercado,
		VersionInicioCAS:           datos.ReciboInicio.VersionInicioCAS,
		VersionReclamacionCAS:      datos.VersionReclamacionCAS,
	}
	mensaje := serializarMaterialCrudoVerificacionOrdenDespachoDocumentalV3(
		vinculos, cercado, datos.ReciboInicio.AtestacionInicio, datos.AtestacionReclamacion,
	)
	material := MaterialCrudoVerificacionOrdenDespachoDocumentalV3{
		orden: orden, vinculo: vinculo, token: token,
		cercado:     cercado,
		inicio:      clonarPruebaCrudaAtestacionDespachoDocumentalV3(datos.ReciboInicio.AtestacionInicio),
		reclamacion: clonarPruebaCrudaAtestacionDespachoDocumentalV3(datos.AtestacionReclamacion),
		vinculos:    vinculos, mensaje: append([]byte(nil), mensaje...),
		huella: documentalcanonico.HuellaBytesSHA256(mensaje),
	}
	if material.Validar() != nil {
		return MaterialCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return material, nil
}

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Validar() error {
	datos, errDatos := m.orden.Datos()
	huellaOrden, errOrden := m.orden.HuellaSHA256()
	huellaVinculo, errVinculo := m.vinculo.HuellaSHA256()
	proyeccion := documentalcanonico.DatosMaterialDespachoV3{
		Vinculos:         proyectarVinculosMaterialDespachoDocumentalV3(m.vinculos),
		Cercado:          proyectarPerfilMaterialDespachoDocumentalV3(m.cercado),
		Inicio:           proyectarPerfilMaterialDespachoDocumentalV3(m.inicio),
		Reclamacion:      proyectarPerfilMaterialDespachoDocumentalV3(m.reclamacion),
		ClaveEsperadaRef: m.token.claveAtestacionRef, RevisionEsperada: m.token.revisionClave,
		HuellaOrdenEsperadaSHA256:       huellaOrden,
		HuellaReciboEsperadaSHA256:      datos.HuellaReciboInicioSHA256,
		HuellaVinculoEsperadaSHA256:     huellaVinculo,
		HuellaCercadoEsperadaSHA256:     m.token.HuellaVinculoSHA256(),
		SecuenciaEsperada:               m.token.Secuencia(),
		VersionInicioEsperada:           datos.ReciboInicio.VersionInicioCAS,
		VersionReclamacionEsperada:      datos.VersionReclamacionCAS,
		HuellaInicioEsperadaSHA256:      datos.ReciboInicio.AtestacionInicio.huellaSHA256(),
		HuellaReclamacionEsperadaSHA256: datos.AtestacionReclamacion.huellaSHA256(),
		Mensaje:                         m.mensaje, HuellaMensajeSHA256: m.huella,
	}
	if errDatos != nil || errOrden != nil || errVinculo != nil ||
		m.token.ValidarPara(m.vinculo) != nil ||
		!proyeccion.Validar() {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MensajeCanonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), m.mensaje...), nil
}

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Pruebas() (
	cercado, inicio, reclamacion PruebaCrudaAtestacionDespachoDocumentalV3,
	err error,
) {
	if m.Validar() != nil {
		return PruebaCrudaAtestacionDespachoDocumentalV3{},
			PruebaCrudaAtestacionDespachoDocumentalV3{},
			PruebaCrudaAtestacionDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.cercado),
		clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.inicio),
		clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.reclamacion), nil
}

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Vinculos() (
	VinculosCrudosVerificacionOrdenDespachoDocumentalV3,
	error,
) {
	if m.Validar() != nil {
		return VinculosCrudosVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return m.vinculos, nil
}

func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) HuellaSHA256() (string, error) {
	if m.Validar() != nil {
		return "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return m.huella, nil
}

func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) String() string {
	return "[MATERIAL-CRUDO-VERIFICACION-DESPACHO-V3-NOMINAL-NO-AUTORITATIVO-REDACTADO]"
}
func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) GoString() string { return m.String() }
func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, m.String())
}
func (m MaterialCrudoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(m.String())
}
func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MaterialCrudoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MaterialCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudComprobarOrdenDespachoDocumentalV3 struct {
	orden    OrdenDespachoDocumentalV3Nominal
	vinculo  VinculoEstableActivacionDocumentalV3
	token    TokenCercadoEjecucionDocumentalV3Nominal
	material MaterialCrudoVerificacionOrdenDespachoDocumentalV3
	mensaje  []byte
	huella   string
}

func NuevaSolicitudComprobarOrdenDespachoDocumentalV3(
	orden OrdenDespachoDocumentalV3Nominal,
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) (SolicitudComprobarOrdenDespachoDocumentalV3, error) {
	material, err := nuevoMaterialCrudoVerificacionOrdenDespachoDocumentalV3(orden, vinculo, token)
	if err != nil {
		return SolicitudComprobarOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	mensaje, _ := material.MensajeCanonico()
	huella, _ := material.HuellaSHA256()
	solicitud := SolicitudComprobarOrdenDespachoDocumentalV3{
		orden: orden, vinculo: vinculo, token: token,
		material: clonarMaterialCrudoVerificacionOrdenDespachoDocumentalV3(material),
		mensaje:  append([]byte(nil), mensaje...), huella: huella,
	}
	if solicitud.Validar() != nil {
		return SolicitudComprobarOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return solicitud, nil
}

func (s SolicitudComprobarOrdenDespachoDocumentalV3) Validar() error {
	huellaOrden, errOrden := s.orden.HuellaSHA256()
	huellaOrdenMaterial, errOrdenMaterial := s.material.orden.HuellaSHA256()
	huellaVinculo, errVinculo := s.vinculo.HuellaSHA256()
	huellaVinculoMaterial, errVinculoMaterial := s.material.vinculo.HuellaSHA256()
	solicitudToken, errToken := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(s.vinculo, s.token)
	mensajeToken, errMensajeToken := solicitudToken.Mensaje()
	macToken, errMACToken := solicitudToken.MAC()
	if s.material.Validar() != nil || errOrden != nil || errOrdenMaterial != nil ||
		errVinculo != nil || errVinculoMaterial != nil ||
		errToken != nil || errMensajeToken != nil || errMACToken != nil ||
		huellaOrden != huellaOrdenMaterial || huellaVinculo != huellaVinculoMaterial ||
		!documentalcanonico.BytesIguales(mensajeToken, s.material.cercado.mensajeCanonico) ||
		!documentalcanonico.BytesIguales(macToken, s.material.cercado.sobreCriptografico) ||
		s.token.ValidarPara(s.vinculo) != nil || len(s.mensaje) == 0 ||
		!esSHA256Hexadecimal(s.huella) || s.huella != documentalcanonico.HuellaBytesSHA256(s.mensaje) ||
		s.huella != s.material.huella ||
		!documentalcanonico.BytesIguales(s.mensaje, s.material.mensaje) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (s SolicitudComprobarOrdenDespachoDocumentalV3) MaterialCrudo() (
	MaterialCrudoVerificacionOrdenDespachoDocumentalV3,
	error,
) {
	if s.Validar() != nil {
		return MaterialCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return clonarMaterialCrudoVerificacionOrdenDespachoDocumentalV3(s.material), nil
}

func (s SolicitudComprobarOrdenDespachoDocumentalV3) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (SolicitudComprobarOrdenDespachoDocumentalV3) String() string {
	return "[SOLICITUD-COMPROBAR-ORDEN-DESPACHO-V3-CRUDA-REDACTADA]"
}
func (s SolicitudComprobarOrdenDespachoDocumentalV3) GoString() string { return s.String() }
func (s SolicitudComprobarOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudComprobarOrdenDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudComprobarOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudComprobarOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// ResultadoCrudoVerificacionOrdenDespachoDocumentalV3 es la respuesta nominal
// del conector KMS. Su fabrica publica permite restaurarla y, por eso, nunca es
// autoridad aislada.
type ResultadoCrudoVerificacionOrdenDespachoDocumentalV3 struct {
	huellaSolicitud         string
	huellaMaterialCrudo     string
	comprobacionRef         string
	algoritmo               string
	audiencia               string
	contexto                string
	claveGestionadaRef      string
	revisionClaveGestionada uint64
	evidenciaOperacionRef   string
	huellaAtestacionSHA256  string
	comprobadaEn            time.Time
}

// DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3 permite que un
// adaptador externo persista y enlace la respuesta nominal del KMS. Los campos
// son material crudo restaurable: no prueban por si solos que la comprobacion
// criptografica haya ocurrido ni autorizan a ejecutar un efecto.
type DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3 struct {
	HuellaSolicitudSHA256      string
	HuellaMaterialCrudoSHA256  string
	ComprobacionRef            string
	Algoritmo                  string
	Audiencia                  string
	Contexto                   string
	ClaveGestionadaRef         string
	RevisionClaveGestionada    uint64
	EvidenciaOperacionRef      string
	HuellaAtestacionSHA256     string
	ComprobadaEn               time.Time
	HuellaResultadoCrudoSHA256 string
}

func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) String() string {
	return "[DATOS-CRUDOS-RESULTADO-VERIFICACION-DESPACHO-V3-NO-AUTORITATIVOS-REDACTADOS]"
}
func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) GoString() string {
	return d.String()
}
func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

func NuevoResultadoCrudoVerificacionOrdenDespachoDocumentalV3(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	comprobacionRef, evidenciaOperacionRef string,
	huellaAtestacionSHA256 string,
	comprobadaEn time.Time,
) (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error) {
	huellaMaterial, errMaterial := solicitud.material.HuellaSHA256()
	algoritmo, _, _, claveGestionadaRef, revisionClave, errPerfil := solicitud.material.cercado.Perfil()
	resultado := ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{
		huellaSolicitud: solicitud.huella, huellaMaterialCrudo: huellaMaterial,
		comprobacionRef: comprobacionRef,
		algoritmo:       algoritmo, audiencia: AudienciaComprobacionOrdenDespachoV3,
		contexto:           ContextoComprobacionOrdenDespachoV3,
		claveGestionadaRef: claveGestionadaRef, revisionClaveGestionada: revisionClave,
		evidenciaOperacionRef:  evidenciaOperacionRef,
		huellaAtestacionSHA256: huellaAtestacionSHA256, comprobadaEn: comprobadaEn,
	}
	if errMaterial != nil || errPerfil != nil || resultado.ValidarPara(solicitud) != nil {
		return ResultadoCrudoVerificacionOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return resultado, nil
}

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) ValidarPara(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		r.huellaMaterialCrudo != solicitud.material.huella ||
		r.claveGestionadaRef != solicitud.material.cercado.claveGestionadaRef ||
		r.revisionClaveGestionada != solicitud.material.cercado.revisionClaveGestionada ||
		r.validarEstructuraParaOrden(solicitud.orden) != nil {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

// DatosCrudos devuelve una copia agrupada para adaptadores KMS y de
// persistencia. La operacion solo valida forma nominal; el servicio de
// aplicacion conserva la obligacion de llamar ValidarPara y efectuar la
// relectura/CAS dentro de su frontera privada.
func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) DatosCrudos() (
	DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3,
	error,
) {
	if r.validarEstructuraNominal() != nil {
		return DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3{},
			ErrOrdenDespachoDocumentalV3Invalida
	}
	return DatosCrudosResultadoVerificacionOrdenDespachoDocumentalV3{
		HuellaSolicitudSHA256:      r.huellaSolicitud,
		HuellaMaterialCrudoSHA256:  r.huellaMaterialCrudo,
		ComprobacionRef:            r.comprobacionRef,
		Algoritmo:                  r.algoritmo,
		Audiencia:                  r.audiencia,
		Contexto:                   r.contexto,
		ClaveGestionadaRef:         r.claveGestionadaRef,
		RevisionClaveGestionada:    r.revisionClaveGestionada,
		EvidenciaOperacionRef:      r.evidenciaOperacionRef,
		HuellaAtestacionSHA256:     r.huellaAtestacionSHA256,
		ComprobadaEn:               r.comprobadaEn,
		HuellaResultadoCrudoSHA256: r.huellaSHA256(),
	}, nil
}

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) validarEstructuraNominal() error {
	referencias := []string{r.comprobacionRef, r.claveGestionadaRef, r.evidenciaOperacionRef}
	for indice, referencia := range referencias {
		if !referenciaEjecucionDocumentalV3Valida(referencia) {
			return ErrOrdenDespachoDocumentalV3Invalida
		}
		for otra := indice + 1; otra < len(referencias); otra++ {
			if referencia == referencias[otra] {
				return ErrOrdenDespachoDocumentalV3Invalida
			}
		}
	}
	if !esSHA256Hexadecimal(r.huellaSolicitud) ||
		!esSHA256Hexadecimal(r.huellaMaterialCrudo) ||
		r.algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 ||
		r.audiencia != AudienciaComprobacionOrdenDespachoV3 ||
		r.contexto != ContextoComprobacionOrdenDespachoV3 ||
		r.revisionClaveGestionada == 0 ||
		!esSHA256Hexadecimal(r.huellaAtestacionSHA256) ||
		!instanteEjecucionDocumentalV3Valido(r.comprobadaEn) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) validarEstructuraParaOrden(
	orden OrdenDespachoDocumentalV3Nominal,
) error {
	datos, err := orden.Datos()
	if err != nil || r.validarEstructuraNominal() != nil ||
		r.comprobadaEn.Before(datos.ReclamadaEn) || !r.comprobadaEn.Before(datos.ExpiraEn) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) huellaSHA256() string {
	return documentalcanonico.HuellaCamposSHA256V3([]string{
		"vec.documentos.resultado-crudo-verificacion-despacho.v3", r.huellaSolicitud,
		r.huellaMaterialCrudo, r.comprobacionRef, r.algoritmo, r.audiencia,
		r.contexto, r.claveGestionadaRef, strconvu64(r.revisionClaveGestionada),
		r.evidenciaOperacionRef,
		r.huellaAtestacionSHA256, r.comprobadaEn.Format(time.RFC3339Nano),
	})
}

func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) String() string {
	return "[RESULTADO-CRUDO-VERIFICACION-DESPACHO-V3-NO-AUTORITATIVO-REDACTADO]"
}
func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) GoString() string { return r.String() }
func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoCrudoVerificacionOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// EstadoCrudoOrdenDespachoDocumentalV3 es la relectura nominal del registro
// privado posterior a la reclamacion. Nunca concede autoridad por si solo.
type EstadoCrudoOrdenDespachoDocumentalV3 struct {
	huellaOrdenSHA256        string
	huellaResultadoKMSSHA256 string
	estadoRef                string
	auditoriaRef             string
	consumoRef               string
	outboxConsumoRef         string
	versionReclamacionCAS    uint64
	versionConsumoCAS        uint64
	consumidaEn              time.Time
}

func NuevoEstadoCrudoOrdenDespachoDocumentalV3(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	estadoRef, auditoriaRef, consumoRef, outboxConsumoRef string,
	versionConsumoCAS uint64,
	consumidaEn time.Time,
) (EstadoCrudoOrdenDespachoDocumentalV3, error) {
	// La correlacion solicitud/resultado debe cerrarse antes de formar cualquier
	// estado que represente el CAS. Así un resultado KMS valido para B nunca se
	// puede aplicar a la orden A y descubrirse solo despues de consumirla.
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil {
		return EstadoCrudoOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	orden := solicitud.orden
	huellaOrden, err := orden.HuellaSHA256()
	datosOrden, errDatos := orden.Datos()
	estado := EstadoCrudoOrdenDespachoDocumentalV3{
		huellaOrdenSHA256: huellaOrden, huellaResultadoKMSSHA256: resultado.huellaSHA256(),
		estadoRef: estadoRef, auditoriaRef: auditoriaRef,
		consumoRef: consumoRef, outboxConsumoRef: outboxConsumoRef,
		versionConsumoCAS: versionConsumoCAS,
		consumidaEn:       consumidaEn,
	}
	if errDatos == nil {
		estado.versionReclamacionCAS = datosOrden.VersionReclamacionCAS
	}
	if err != nil || errDatos != nil ||
		estado.validarEstructuraPara(orden, resultado) != nil {
		return EstadoCrudoOrdenDespachoDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return estado, nil
}

func (e EstadoCrudoOrdenDespachoDocumentalV3) validarEstructuraPara(
	orden OrdenDespachoDocumentalV3Nominal,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
) error {
	datos, errDatos := orden.Datos()
	huellaOrden, errHuella := orden.HuellaSHA256()
	referencias := []string{
		e.estadoRef, e.auditoriaRef, e.consumoRef, e.outboxConsumoRef,
		resultado.comprobacionRef, resultado.evidenciaOperacionRef,
	}
	for indice, referencia := range referencias {
		if !referenciaEjecucionDocumentalV3Valida(referencia) {
			return ErrOrdenDespachoDocumentalV3Invalida
		}
		for otra := indice + 1; otra < len(referencias); otra++ {
			if referencia == referencias[otra] {
				return ErrOrdenDespachoDocumentalV3Invalida
			}
		}
	}
	if errDatos != nil || errHuella != nil || e.huellaOrdenSHA256 != huellaOrden ||
		e.huellaResultadoKMSSHA256 != resultado.huellaSHA256() ||
		e.versionReclamacionCAS != datos.VersionReclamacionCAS ||
		e.versionConsumoCAS <= e.versionReclamacionCAS ||
		!instanteEjecucionDocumentalV3Valido(e.consumidaEn) ||
		e.consumidaEn.Before(resultado.comprobadaEn) || !e.consumidaEn.Before(datos.ExpiraEn) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (e EstadoCrudoOrdenDespachoDocumentalV3) ValidarPara(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
) error {
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil ||
		e.validarEstructuraPara(solicitud.orden, resultado) != nil {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (e EstadoCrudoOrdenDespachoDocumentalV3) huellaSHA256() string {
	return documentalcanonico.HuellaCamposSHA256V3([]string{
		"vec.documentos.estado-crudo-despacho.v3", e.huellaOrdenSHA256,
		e.huellaResultadoKMSSHA256, e.estadoRef, e.auditoriaRef, e.consumoRef,
		e.outboxConsumoRef,
		strconvu64(e.versionReclamacionCAS), strconvu64(e.versionConsumoCAS),
		e.consumidaEn.Format(time.RFC3339Nano),
	})
}

func (EstadoCrudoOrdenDespachoDocumentalV3) String() string {
	return "[ESTADO-CRUDO-ORDEN-DESPACHO-V3-NOMINAL-REDACTADO]"
}
func (e EstadoCrudoOrdenDespachoDocumentalV3) GoString() string { return e.String() }
func (e EstadoCrudoOrdenDespachoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (e EstadoCrudoOrdenDespachoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (EstadoCrudoOrdenDespachoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*EstadoCrudoOrdenDespachoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// VerificadorOrdenDespachoDocumentalV3 es un puerto intercambiable de KMS.
// Devuelve un resultado crudo nominal; no puede promover capacidades.
type VerificadorOrdenDespachoDocumentalV3 interface {
	VerificarOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudComprobarOrdenDespachoDocumentalV3,
	) (ResultadoCrudoVerificacionOrdenDespachoDocumentalV3, error)
}

// ConsumidorPrivadoOrdenDespachoDocumentalV3 relee y consume por CAS la orden
// reclamada dentro de una unica transaccion que incrementa VersionConsumoCAS,
// inmoviliza un ConsumoDespachoRef UNIQUE y escribe auditoria y outbox. Debe
// ejecutar Resultado.ValidarPara(Solicitud) antes de mutar la fila y repetir
// esa correlacion dentro de la transaccion, junto con la relectura durable. No
// existe una operacion publica read-then-use: el segundo consumo o un resultado
// KMS perteneciente a otra solicitud deben fallar sin ejecutar el CAS.
type ConsumidorPrivadoOrdenDespachoDocumentalV3 interface {
	ReleerYConsumirOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudComprobarOrdenDespachoDocumentalV3,
		ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	) (EstadoCrudoOrdenDespachoDocumentalV3, error)
}

// OrdenDespachoDocumentalV3ConsumidaNominal es el comando restaurable que el
// servicio de aplicacion crea despues de verificar por KMS y consumir por CAS.
// Su fabrica publica solo coteja forma y correlacion: NO acredita que esas dos
// operaciones ocurrieran y nunca concede autoridad por si sola. HTTP, CLI, MCP
// y handlers no deben recibirla ni poseer el despachador que la consume.
type OrdenDespachoDocumentalV3ConsumidaNominal struct {
	solicitud     SolicitudComprobarOrdenDespachoDocumentalV3
	resultado     ResultadoCrudoVerificacionOrdenDespachoDocumentalV3
	estado        EstadoCrudoOrdenDespachoDocumentalV3
	huellaConsumo string
}

func NuevaOrdenDespachoDocumentalV3ConsumidaNominal(
	solicitud SolicitudComprobarOrdenDespachoDocumentalV3,
	resultado ResultadoCrudoVerificacionOrdenDespachoDocumentalV3,
	estado EstadoCrudoOrdenDespachoDocumentalV3,
) (OrdenDespachoDocumentalV3ConsumidaNominal, error) {
	orden := OrdenDespachoDocumentalV3ConsumidaNominal{
		solicitud: solicitud, resultado: resultado, estado: estado,
		huellaConsumo: documentalcanonico.HuellaCamposSHA256V3([]string{
			"vec.documentos.orden-despacho-consumida-nominal.v3", solicitud.huella,
			resultado.huellaSHA256(), estado.huellaSHA256(),
		}),
	}
	if orden.ValidarEn(estado.consumidaEn) != nil {
		return OrdenDespachoDocumentalV3ConsumidaNominal{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return orden, nil
}

func (o OrdenDespachoDocumentalV3ConsumidaNominal) ValidarEn(instante time.Time) error {
	if o.solicitud.Validar() != nil ||
		o.resultado.ValidarPara(o.solicitud) != nil ||
		o.estado.ValidarPara(o.solicitud, o.resultado) != nil ||
		!esSHA256Hexadecimal(o.huellaConsumo) ||
		o.huellaConsumo != documentalcanonico.HuellaCamposSHA256V3([]string{
			"vec.documentos.orden-despacho-consumida-nominal.v3", o.solicitud.huella,
			o.resultado.huellaSHA256(), o.estado.huellaSHA256(),
		}) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	datos, err := o.solicitud.orden.Datos()
	if err != nil || !instanteEjecucionDocumentalV3Valido(instante) ||
		instante.Before(o.estado.consumidaEn) || !instante.Before(datos.ExpiraEn) {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (o OrdenDespachoDocumentalV3ConsumidaNominal) VinculoActivacion() (
	VinculoEstableActivacionDocumentalV3,
	error,
) {
	if o.ValidarEn(o.estado.consumidaEn) != nil {
		return VinculoEstableActivacionDocumentalV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return o.solicitud.vinculo, nil
}

func (o OrdenDespachoDocumentalV3ConsumidaNominal) DatosOrden() (
	DatosOrdenDespachoDocumentalV3Nominal,
	error,
) {
	if o.ValidarEn(o.estado.consumidaEn) != nil {
		return DatosOrdenDespachoDocumentalV3Nominal{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return o.solicitud.orden.Datos()
}

func (OrdenDespachoDocumentalV3ConsumidaNominal) String() string {
	return "[ORDEN-DESPACHO-DOCUMENTAL-V3-CONSUMIDA-NOMINAL-NO-AUTORITATIVA-REDACTADA]"
}
func (o OrdenDespachoDocumentalV3ConsumidaNominal) GoString() string { return o.String() }
func (o OrdenDespachoDocumentalV3ConsumidaNominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, o.String())
}
func (o OrdenDespachoDocumentalV3ConsumidaNominal) LogValue() slog.Value {
	return slog.StringValue(o.String())
}
func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (OrdenDespachoDocumentalV3ConsumidaNominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*OrdenDespachoDocumentalV3ConsumidaNominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// ResultadoEfectoRenderizadoDocumentalV3Crudo conserva una referencia exacta y
// versionada del objeto; nunca una URL temporal ni los bytes del documento.
type ResultadoEfectoRenderizadoDocumentalV3Crudo struct {
	BorradorRef           string
	EfectoRef             string
	ContenidoRef          string
	ContenidoVersion      string
	ConectorRef           string
	MIME                  string
	HuellaSalidaSHA256    string
	TamanoSalida          uint64
	EvidenciaOperacionRef string
}

func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
) error {
	datos, err := manifiesto.Datos()
	if err != nil || r.BorradorRef != datos.BorradorRef || r.EfectoRef != datos.EfectoRef ||
		!referenciaEjecucionDocumentalV3Valida(r.ContenidoRef) ||
		!referenciaEjecucionDocumentalV3Valida(r.ContenidoVersion) ||
		!referenciaEjecucionDocumentalV3Valida(r.ConectorRef) ||
		!referenciaEjecucionDocumentalV3Valida(r.EvidenciaOperacionRef) ||
		r.MIME != datos.DescriptorPerfil.Perfil().MIME() ||
		!esSHA256Hexadecimal(r.HuellaSalidaSHA256) || r.TamanoSalida == 0 ||
		r.TamanoSalida > datos.LimiteEfectivoBytes {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

func (ResultadoEfectoRenderizadoDocumentalV3Crudo) String() string {
	return "[RESULTADO-EFECTO-RENDERIZADO-DOCUMENTAL-V3-CRUDO-REFERENCIAS-REDACTADAS]"
}
func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) GoString() string { return r.String() }
func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoEfectoRenderizadoDocumentalV3Crudo) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoEfectoRenderizadoDocumentalV3Crudo) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoEfectoRenderizadoDocumentalV3Crudo) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// HuellasRecibosEjecucionDocumentalV3 es la proyeccion persistible de tres
// sobres COSE nominalmente correlacionados. No acredita su comprobacion ni
// concede autoridad: el registro debe releer y comprobar los recibos mediante
// dependencias privadas antes del CAS de confirmacion.
type HuellasRecibosEjecucionDocumentalV3 struct {
	ReciboRenderRef               string
	HuellaSobreRenderSHA256       string
	HuellaReciboRenderSHA256      string
	ReciboEstructuralRef          string
	HuellaSobreEstructuralSHA256  string
	HuellaReciboEstructuralSHA256 string
	ReciboSemanticoRef            string
	HuellaSobreSemanticoSHA256    string
	HuellaReciboSemanticoSHA256   string
}

func (h HuellasRecibosEjecucionDocumentalV3) Validar() error {
	referencias := []string{h.ReciboRenderRef, h.ReciboEstructuralRef, h.ReciboSemanticoRef}
	huella := []string{
		h.HuellaSobreRenderSHA256, h.HuellaReciboRenderSHA256,
		h.HuellaSobreEstructuralSHA256, h.HuellaReciboEstructuralSHA256,
		h.HuellaSobreSemanticoSHA256, h.HuellaReciboSemanticoSHA256,
	}
	for indice, referencia := range referencias {
		if !referenciaEjecucionDocumentalV3Valida(referencia) {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
		for otro := indice + 1; otro < len(referencias); otro++ {
			if referencia == referencias[otro] {
				return ErrTransicionEjecucionDocumentalV3Invalida
			}
		}
	}
	for indice, valor := range huella {
		if !esSHA256Hexadecimal(valor) {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
		for otro := indice + 1; otro < len(huella); otro++ {
			if valor == huella[otro] {
				return ErrTransicionEjecucionDocumentalV3Invalida
			}
		}
	}
	return nil
}

type RecibosEjecucionDocumentalV3 struct {
	Render      ReciboEjecucionComponenteDocumentalNominal
	Estructural ReciboEjecucionComponenteDocumentalNominal
	Semantico   ReciboEjecucionComponenteDocumentalNominal
}

func (r RecibosEjecucionDocumentalV3) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3Crudo,
	reservaRef string,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
	instante time.Time,
) error {
	datosManifiesto, errManifiesto := manifiesto.Datos()
	vinculo, errVinculo := ordenConsumida.VinculoActivacion()
	datosOrden, errOrden := ordenConsumida.DatosOrden()
	datosRender, errRender := r.Render.Datos()
	datosEstructural, errEstructural := r.Estructural.Datos()
	datosSemantico, errSemantico := r.Semantico.Datos()
	if errManifiesto != nil || errVinculo != nil || errOrden != nil ||
		ordenConsumida.ValidarEn(instante) != nil || vinculo.ReservaRef != reservaRef ||
		!manifiestosEjecucionDocumentalV3Coinciden(vinculo.Manifiesto, manifiesto) ||
		vinculo.ConsumoDecision != consumo ||
		resultado.ValidarContra(manifiesto) != nil ||
		errRender != nil || errEstructural != nil || errSemantico != nil ||
		!reciboEjecucionDocumentalV3Coincide(
			datosRender, OperacionRenderizadoDocumental,
			datosManifiesto.ComponenteRender, datosManifiesto, resultado,
			reservaRef, consumo, datosOrden, ordenConsumida,
		) || !reciboEjecucionDocumentalV3Coincide(
		datosEstructural, OperacionValidacionEstructuralDocumental,
		datosManifiesto.ComponenteVerificador, datosManifiesto, resultado,
		reservaRef, consumo, datosOrden, ordenConsumida,
	) || !reciboEjecucionDocumentalV3Coincide(
		datosSemantico, OperacionVerificacionSemanticaDocumental,
		datosManifiesto.ComponenteSemantico, datosManifiesto, resultado,
		reservaRef, consumo, datosOrden, ordenConsumida,
	) || !r.Render.IndependienteDe(r.Estructural) ||
		!r.Render.IndependienteDe(r.Semantico) || !r.Estructural.IndependienteDe(r.Semantico) {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

func (r RecibosEjecucionDocumentalV3) Huellas() (HuellasRecibosEjecucionDocumentalV3, error) {
	datosRender, errRender := r.Render.Datos()
	datosEstructural, errEstructural := r.Estructural.Datos()
	datosSemantico, errSemantico := r.Semantico.Datos()
	if errRender != nil || errEstructural != nil || errSemantico != nil {
		return HuellasRecibosEjecucionDocumentalV3{}, ErrTransicionEjecucionDocumentalV3Invalida
	}
	huella := HuellasRecibosEjecucionDocumentalV3{
		ReciboRenderRef:               datosRender.ReciboRef,
		HuellaSobreRenderSHA256:       datosRender.HuellaSobreCOSESHA256,
		HuellaReciboRenderSHA256:      datosRender.HuellaReciboSHA256,
		ReciboEstructuralRef:          datosEstructural.ReciboRef,
		HuellaSobreEstructuralSHA256:  datosEstructural.HuellaSobreCOSESHA256,
		HuellaReciboEstructuralSHA256: datosEstructural.HuellaReciboSHA256,
		ReciboSemanticoRef:            datosSemantico.ReciboRef,
		HuellaSobreSemanticoSHA256:    datosSemantico.HuellaSobreCOSESHA256,
		HuellaReciboSemanticoSHA256:   datosSemantico.HuellaReciboSHA256,
	}
	if huella.Validar() != nil {
		return HuellasRecibosEjecucionDocumentalV3{}, ErrTransicionEjecucionDocumentalV3Invalida
	}
	return huella, nil
}

type SolicitudAbandonarEjecucionDocumentalV3 struct {
	ReservaRef      string
	Manifiesto      ManifiestoEjecucionDocumentalV3
	EstadoEsperado  EstadoEjecucionDocumentalV3
	ConsumoDecision ConsumoDecisionEjecucionDocumentalV3
	MotivoRef       string
	AbandonadaEn    time.Time
}

func (s SolicitudAbandonarEjecucionDocumentalV3) Validar() error {
	if !referenciaEjecucionDocumentalV3Valida(s.ReservaRef) || s.Manifiesto.Validar() != nil ||
		!referenciaEjecucionDocumentalV3Valida(s.MotivoRef) ||
		!instanteEjecucionDocumentalV3Valido(s.AbandonadaEn) {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	switch s.EstadoEsperado {
	case EstadoEjecucionDocumentalV3Preparada:
		if s.ConsumoDecision != (ConsumoDecisionEjecucionDocumentalV3{}) {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Activa:
		// La solicitud es estructural. El registro debe releer la activacion,
		// adopcion V4 y version CAS dentro de la transaccion de abandono.
		if s.ConsumoDecision.ValidarContra(s.Manifiesto) != nil {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
	default:
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

type SolicitudMarcarEjecucionDocumentalV3Indeterminada struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	IncidenteRef           string
	MarcadaEn              time.Time
}

func (s SolicitudMarcarEjecucionDocumentalV3Indeterminada) Validar() error {
	if !ordenDespachoDocumentalV3ConsumidaNominalCoincide(
		s.OrdenDespachoConsumida, s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.MarcadaEn,
	) ||
		!referenciaEjecucionDocumentalV3Valida(s.IncidenteRef) ||
		!instanteEjecucionDocumentalV3Valido(s.MarcadaEn) {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

type ConsultaEjecucionDocumentalV3 struct {
	ReservaRef             string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
}

func (c ConsultaEjecucionDocumentalV3) Validar() error {
	porReserva := c.ReservaRef != "" && c.IndiceIdempotenciaHMAC == "" && c.HuellaSolicitudHMAC == "" &&
		referenciaEjecucionDocumentalV3Valida(c.ReservaRef)
	porIndice := c.ReservaRef == "" && hmacSHA256PuertoValido(c.IndiceIdempotenciaHMAC) &&
		hmacSHA256PuertoValido(c.HuellaSolicitudHMAC) &&
		documentalcanonico.ClaveHMACSHA256V3(c.IndiceIdempotenciaHMAC) !=
			documentalcanonico.ClaveHMACSHA256V3(c.HuellaSolicitudHMAC)
	if porReserva == porIndice {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

func (ConsultaEjecucionDocumentalV3) String() string {
	return "[CONSULTA-EJECUCION-DOCUMENTAL-V3-HMAC-REDACTADA]"
}
func (c ConsultaEjecucionDocumentalV3) GoString() string { return c.String() }
func (c ConsultaEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConsultaEjecucionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
func (ConsultaEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ConsultaEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ConsultaEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ConsultaEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ConsultaEjecucionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ConsultaEjecucionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type InstantaneaEjecucionDocumentalV3Nominal struct {
	ReservaRef                 string
	IndiceIdempotenciaHMAC     string
	HuellaSolicitudHMAC        string
	Manifiesto                 ManifiestoEjecucionDocumentalV3
	Estado                     EstadoEjecucionDocumentalV3
	SecuenciaCercado           uint64
	HuellaVinculoSHA256        string
	ConsumoDecision            ConsumoDecisionEjecucionDocumentalV3
	OrdenConsumoDurableV4Ref   string
	Resultado                  ResultadoEfectoRenderizadoDocumentalV3Crudo
	IncidenteRef               string
	EvidenciaRef               string
	HuellaEvidenciaSHA256      string
	EstadoOrigenAbandono       EstadoEjecucionDocumentalV3
	MotivoAbandonoRef          string
	ReconciliacionRef          string
	HuellaReconciliacionSHA256 string
	ActualizadaEn              time.Time
}

func (i InstantaneaEjecucionDocumentalV3Nominal) Validar() error {
	datos, err := i.Manifiesto.Datos()
	if err != nil || !referenciaEjecucionDocumentalV3Valida(i.ReservaRef) ||
		!hmacSHA256PuertoValido(i.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(i.HuellaSolicitudHMAC) ||
		!documentalcanonico.ClavesHMACSHA256V3Distintas(
			i.IndiceIdempotenciaHMAC, i.HuellaSolicitudHMAC, datos.HuellaEntradaHMAC,
		) || !i.Estado.Valido() || !instanteEjecucionDocumentalV3Valido(i.ActualizadaEn) {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	switch i.Estado {
	case EstadoEjecucionDocumentalV3Preparada:
		if i.SecuenciaCercado != 0 || i.HuellaVinculoSHA256 != "" ||
			i.ConsumoDecision != (ConsumoDecisionEjecucionDocumentalV3{}) ||
			i.OrdenConsumoDurableV4Ref != "" ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) || i.IncidenteRef != "" ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionInstantaneaDocumentalV3Vacios(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Activa, EstadoEjecucionDocumentalV3EfectoIniciado:
		if !vinculoCercadoProyectadoDocumentalV3Valido(i) ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) || i.IncidenteRef != "" ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionInstantaneaDocumentalV3Vacios(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Indeterminada:
		if !vinculoCercadoProyectadoDocumentalV3Valido(i) ||
			!referenciaEjecucionDocumentalV3Valida(i.IncidenteRef) ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionInstantaneaDocumentalV3Vacios(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Confirmada:
		if !vinculoCercadoProyectadoDocumentalV3Valido(i) || i.Resultado.ValidarContra(i.Manifiesto) != nil ||
			i.IncidenteRef != "" || !referenciaEjecucionDocumentalV3Valida(i.EvidenciaRef) ||
			!esSHA256Hexadecimal(i.HuellaEvidenciaSHA256) ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionConfirmadaDocumentalV3Validos(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3AbandonadaSinEfecto:
		if i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!referenciaEjecucionDocumentalV3Valida(i.MotivoAbandonoRef) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
		switch i.EstadoOrigenAbandono {
		case EstadoEjecucionDocumentalV3Preparada:
			if i.SecuenciaCercado != 0 || i.HuellaVinculoSHA256 != "" ||
				i.ConsumoDecision != (ConsumoDecisionEjecucionDocumentalV3{}) || i.IncidenteRef != "" ||
				i.ReconciliacionRef != "" || i.HuellaReconciliacionSHA256 != "" {
				return ErrReservaEjecucionDocumentalV3Invalida
			}
		case EstadoEjecucionDocumentalV3Activa:
			if !vinculoCercadoProyectadoDocumentalV3Valido(i) || i.IncidenteRef != "" ||
				i.ReconciliacionRef != "" || i.HuellaReconciliacionSHA256 != "" {
				return ErrReservaEjecucionDocumentalV3Invalida
			}
		case EstadoEjecucionDocumentalV3Indeterminada:
			if !vinculoCercadoProyectadoDocumentalV3Valido(i) ||
				!referenciaEjecucionDocumentalV3Valida(i.IncidenteRef) ||
				!referenciaEjecucionDocumentalV3Valida(i.ReconciliacionRef) ||
				!esSHA256Hexadecimal(i.HuellaReconciliacionSHA256) {
				return ErrReservaEjecucionDocumentalV3Invalida
			}
		default:
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	}
	return nil
}

func (InstantaneaEjecucionDocumentalV3Nominal) String() string {
	return "[INSTANTANEA-EJECUCION-DOCUMENTAL-V3-NOMINAL-HMAC-REDACTADA]"
}
func (i InstantaneaEjecucionDocumentalV3Nominal) GoString() string { return i.String() }
func (i InstantaneaEjecucionDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}
func (i InstantaneaEjecucionDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(i.String())
}
func (InstantaneaEjecucionDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (InstantaneaEjecucionDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (InstantaneaEjecucionDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*InstantaneaEjecucionDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type DatosEvidenciaRenderizadoDocumentalV3 struct {
	Esquema                       string
	ReservaRef                    string
	IndiceIdempotenciaHMAC        string
	HuellaSolicitudHMAC           string
	Manifiesto                    ManifiestoEjecucionDocumentalV3
	SecuenciaCercado              uint64
	HuellaVinculoSHA256           string
	ClaveAtestacionCercadoRef     string
	HuellaMACCercadoSHA256        string
	EvidenciaAtestacionCercadoRef string
	VerificacionCercadoRef        string
	VerificadoCercadoEn           time.Time
	ConsumoDecision               ConsumoDecisionEjecucionDocumentalV3
	Resultado                     ResultadoEfectoRenderizadoDocumentalV3Crudo
	Recibos                       HuellasRecibosEjecucionDocumentalV3
	GeneradoEn                    time.Time
	ConfirmadoEn                  time.Time
	ReconciliacionRef             string
	HuellaReconciliacionSHA256    string
	ReconciliacionConsultadaEn    time.Time
	VerificacionReconciliacionRef string
	ReconciliacionVerificadaEn    time.Time
}

func (d DatosEvidenciaRenderizadoDocumentalV3) Validar() error {
	datosManifiesto, err := d.Manifiesto.Datos()
	vinculo := VinculoEstableActivacionDocumentalV3{
		ReservaRef: d.ReservaRef, IndiceIdempotenciaHMAC: d.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC: d.HuellaSolicitudHMAC, Manifiesto: d.Manifiesto,
		ConsumoDecision:          d.ConsumoDecision,
		OrdenConsumoDurableV4Ref: d.ConsumoDecision.EfectoRef,
	}
	huellaVinculoEstable, errVinculo := vinculo.HuellaSHA256()
	if d.Esquema != EsquemaEvidenciaRenderizadoV3 || err != nil ||
		!referenciaEjecucionDocumentalV3Valida(d.ReservaRef) ||
		!hmacSHA256PuertoValido(d.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(d.HuellaSolicitudHMAC) ||
		!documentalcanonico.ClavesHMACSHA256V3Distintas(
			d.IndiceIdempotenciaHMAC, d.HuellaSolicitudHMAC, datosManifiesto.HuellaEntradaHMAC,
		) || d.SecuenciaCercado == 0 || !esSHA256Hexadecimal(d.HuellaVinculoSHA256) ||
		!referenciaEjecucionDocumentalV3Valida(d.ClaveAtestacionCercadoRef) ||
		!esSHA256Hexadecimal(d.HuellaMACCercadoSHA256) ||
		!referenciaEjecucionDocumentalV3Valida(d.EvidenciaAtestacionCercadoRef) ||
		!referenciaEjecucionDocumentalV3Valida(d.VerificacionCercadoRef) ||
		!instanteEjecucionDocumentalV3Valido(d.VerificadoCercadoEn) ||
		errVinculo != nil ||
		d.HuellaVinculoSHA256 != huellaVinculoCercadoEjecucionDocumentalV3(
			d.SecuenciaCercado, huellaVinculoEstable,
		) || d.Resultado.ValidarContra(d.Manifiesto) != nil ||
		d.Recibos.Validar() != nil ||
		!instanteEjecucionDocumentalV3Valido(d.GeneradoEn) ||
		!instanteEjecucionDocumentalV3Valido(d.ConfirmadoEn) ||
		d.VerificadoCercadoEn.After(d.GeneradoEn) || d.ConfirmadoEn.Before(d.GeneradoEn) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	if d.ReconciliacionRef == "" {
		if d.HuellaReconciliacionSHA256 != "" || !d.ReconciliacionConsultadaEn.IsZero() ||
			d.VerificacionReconciliacionRef != "" || !d.ReconciliacionVerificadaEn.IsZero() {
			return ErrSelloEvidenciaDocumentalV3Invalido
		}
	} else if !referenciaEjecucionDocumentalV3Valida(d.ReconciliacionRef) ||
		!esSHA256Hexadecimal(d.HuellaReconciliacionSHA256) ||
		!instanteEjecucionDocumentalV3Valido(d.ReconciliacionConsultadaEn) ||
		!referenciaEjecucionDocumentalV3Valida(d.VerificacionReconciliacionRef) ||
		!instanteEjecucionDocumentalV3Valido(d.ReconciliacionVerificadaEn) ||
		d.ReconciliacionConsultadaEn.Before(d.GeneradoEn) ||
		d.ReconciliacionVerificadaEn.Before(d.ReconciliacionConsultadaEn) ||
		d.ConfirmadoEn.Before(d.ReconciliacionVerificadaEn) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (DatosEvidenciaRenderizadoDocumentalV3) String() string {
	return "[DATOS-EVIDENCIA-RENDERIZADO-V3-CONFIDENCIALES-REDACTADOS]"
}
func (d DatosEvidenciaRenderizadoDocumentalV3) GoString() string { return d.String() }
func (d DatosEvidenciaRenderizadoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosEvidenciaRenderizadoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosEvidenciaRenderizadoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosEvidenciaRenderizadoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosEvidenciaRenderizadoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosEvidenciaRenderizadoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type PerfilSelloEvidenciaDocumentalV3 struct {
	Algoritmo string
	ClaveID   string
	Audiencia string
}

func NuevoPerfilSelloEvidenciaHMACSHA256V3(claveID string) (PerfilSelloEvidenciaDocumentalV3, error) {
	perfil := PerfilSelloEvidenciaDocumentalV3{
		Algoritmo: AlgoritmoSelloEvidenciaHMACSHA256V3,
		ClaveID:   claveID, Audiencia: AudienciaSelloEvidenciaRenderizadoV3,
	}
	if perfil.Validar() != nil {
		return PerfilSelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return perfil, nil
}

func (p PerfilSelloEvidenciaDocumentalV3) Validar() error {
	if p.Algoritmo != AlgoritmoSelloEvidenciaHMACSHA256V3 ||
		!referenciaEjecucionDocumentalV3Valida(p.ClaveID) ||
		p.Audiencia != AudienciaSelloEvidenciaRenderizadoV3 {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (PerfilSelloEvidenciaDocumentalV3) String() string {
	return "[PERFIL-SELLO-EVIDENCIA-V3-HMAC-REDACTADO]"
}
func (p PerfilSelloEvidenciaDocumentalV3) GoString() string { return p.String() }
func (p PerfilSelloEvidenciaDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PerfilSelloEvidenciaDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (PerfilSelloEvidenciaDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PerfilSelloEvidenciaDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PerfilSelloEvidenciaDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudFirmaEvidenciaRenderizadoDocumentalV3 struct {
	perfil  PerfilSelloEvidenciaDocumentalV3
	datos   DatosEvidenciaRenderizadoDocumentalV3
	mensaje []byte
	huella  string
}

func NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(
	perfil PerfilSelloEvidenciaDocumentalV3,
	datos DatosEvidenciaRenderizadoDocumentalV3,
) (SolicitudFirmaEvidenciaRenderizadoDocumentalV3, error) {
	if perfil.Validar() != nil || datos.Validar() != nil ||
		perfil.ClaveID == documentalcanonico.ClaveHMACSHA256V3(datos.IndiceIdempotenciaHMAC) ||
		perfil.ClaveID == documentalcanonico.ClaveHMACSHA256V3(datos.HuellaSolicitudHMAC) ||
		perfil.ClaveID == datos.ClaveAtestacionCercadoRef {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	datosManifiesto, _ := datos.Manifiesto.Datos()
	if perfil.ClaveID == documentalcanonico.ClaveHMACSHA256V3(datosManifiesto.HuellaEntradaHMAC) {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	mensaje := serializarEvidenciaRenderizadoDocumentalV3(datos)
	solicitud := SolicitudFirmaEvidenciaRenderizadoDocumentalV3{
		perfil: perfil, datos: datos, mensaje: append([]byte(nil), mensaje...),
		huella: documentalcanonico.HuellaBytesSHA256(mensaje),
	}
	if solicitud.Validar() != nil {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return solicitud, nil
}

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Validar() error {
	if s.perfil.Validar() != nil || s.datos.Validar() != nil || len(s.mensaje) == 0 ||
		!esSHA256Hexadecimal(s.huella) ||
		documentalcanonico.HuellaBytesSHA256(s.mensaje) != s.huella ||
		string(s.mensaje) != string(serializarEvidenciaRenderizadoDocumentalV3(s.datos)) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return append([]byte(nil), s.mensaje...), nil
}
func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) HuellaMensajeSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrSelloEvidenciaDocumentalV3Invalido
	}
	return s.huella, nil
}
func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Perfil() (PerfilSelloEvidenciaDocumentalV3, error) {
	if s.Validar() != nil {
		return PerfilSelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return s.perfil, nil
}

func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) String() string {
	return "[SOLICITUD-FIRMA-EVIDENCIA-RENDERIZADO-V3-CONFIDENCIAL]"
}
func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) GoString() string { return s.String() }
func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudFirmaEvidenciaRenderizadoDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudFirmaEvidenciaRenderizadoDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type DatosSelloEvidenciaDocumentalV3Crudos struct {
	Algoritmo             string
	ClaveID               string
	Audiencia             string
	HuellaMensajeSHA256   string
	Firma                 []byte
	EvidenciaOperacionRef string
	FirmadoEn             time.Time
}

func (DatosSelloEvidenciaDocumentalV3Crudos) String() string {
	return "[DATOS-SELLO-EVIDENCIA-V3-CONFIDENCIALES-REDACTADOS]"
}
func (d DatosSelloEvidenciaDocumentalV3Crudos) GoString() string { return d.String() }
func (d DatosSelloEvidenciaDocumentalV3Crudos) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosSelloEvidenciaDocumentalV3Crudos) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSelloEvidenciaDocumentalV3Crudos) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaDocumentalV3Crudos) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SelloEvidenciaDocumentalV3Nominal struct {
	datos DatosSelloEvidenciaDocumentalV3Crudos
}

func NuevoSelloEvidenciaDocumentalV3Nominal(
	solicitud SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	firma []byte,
	evidenciaOperacionRef string,
	firmadoEn time.Time,
) (SelloEvidenciaDocumentalV3Nominal, error) {
	perfil, errPerfil := solicitud.Perfil()
	huella, errHuella := solicitud.HuellaMensajeSHA256()
	if errPerfil != nil || errHuella != nil {
		return SelloEvidenciaDocumentalV3Nominal{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	sello := SelloEvidenciaDocumentalV3Nominal{datos: DatosSelloEvidenciaDocumentalV3Crudos{
		Algoritmo: perfil.Algoritmo, ClaveID: perfil.ClaveID, Audiencia: perfil.Audiencia,
		HuellaMensajeSHA256: huella, Firma: append([]byte(nil), firma...),
		EvidenciaOperacionRef: evidenciaOperacionRef, FirmadoEn: firmadoEn,
	}}
	if sello.ValidarPara(solicitud) != nil {
		return SelloEvidenciaDocumentalV3Nominal{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return sello, nil
}

func (s SelloEvidenciaDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
) error {
	perfil, errPerfil := solicitud.Perfil()
	huella, errHuella := solicitud.HuellaMensajeSHA256()
	if errPerfil != nil || errHuella != nil ||
		!datosSelloEvidenciaDocumentalV3ValidosPara(s.datos, perfil, huella) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (s SelloEvidenciaDocumentalV3Nominal) Datos() (DatosSelloEvidenciaDocumentalV3Crudos, error) {
	perfil := PerfilSelloEvidenciaDocumentalV3{
		Algoritmo: s.datos.Algoritmo, ClaveID: s.datos.ClaveID, Audiencia: s.datos.Audiencia,
	}
	if !datosSelloEvidenciaDocumentalV3ValidosPara(
		s.datos, perfil, s.datos.HuellaMensajeSHA256,
	) {
		return DatosSelloEvidenciaDocumentalV3Crudos{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	datos := s.datos
	datos.Firma = append([]byte(nil), s.datos.Firma...)
	return datos, nil
}
func (SelloEvidenciaDocumentalV3Nominal) String() string {
	return "[SELLO-EVIDENCIA-DOCUMENTAL-V3-NOMINAL-NO-AUTORITATIVO-REDACTADO]"
}
func (s SelloEvidenciaDocumentalV3Nominal) GoString() string { return s.String() }
func (s SelloEvidenciaDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SelloEvidenciaDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SelloEvidenciaDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SelloEvidenciaDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SelloEvidenciaDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudVerificacionEvidenciaDocumentalV3 struct {
	firma  SolicitudFirmaEvidenciaRenderizadoDocumentalV3
	sello  SelloEvidenciaDocumentalV3Nominal
	huella string
}

func NuevaSolicitudVerificacionEvidenciaDocumentalV3(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	sello SelloEvidenciaDocumentalV3Nominal,
) (SolicitudVerificacionEvidenciaDocumentalV3, error) {
	solicitud := SolicitudVerificacionEvidenciaDocumentalV3{
		firma: firma, sello: sello,
		huella: huellaSolicitudVerificacionEvidenciaDocumentalV3(firma, sello),
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificacionEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return solicitud, nil
}

// NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos permite restaurar
// un sobre persistido, pero no lo convierte en evidencia confiable: el unico
// la salida del conector tambien es nominal. Solo el servicio de aplicacion
// precompuesto puede cotejarla con relectura durable y CAS dentro de su llamada.
func NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	datos DatosSelloEvidenciaDocumentalV3Crudos,
) (SolicitudVerificacionEvidenciaDocumentalV3, error) {
	datos.Firma = append([]byte(nil), datos.Firma...)
	return NuevaSolicitudVerificacionEvidenciaDocumentalV3(
		firma, SelloEvidenciaDocumentalV3Nominal{datos: datos},
	)
}

func (s SolicitudVerificacionEvidenciaDocumentalV3) Validar() error {
	if s.firma.Validar() != nil || s.sello.ValidarPara(s.firma) != nil ||
		!esSHA256Hexadecimal(s.huella) ||
		s.huella != huellaSolicitudVerificacionEvidenciaDocumentalV3(s.firma, s.sello) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (s SolicitudVerificacionEvidenciaDocumentalV3) Mensaje() ([]byte, error) {
	return s.firma.Mensaje()
}
func (s SolicitudVerificacionEvidenciaDocumentalV3) Sello() (DatosSelloEvidenciaDocumentalV3Crudos, error) {
	if s.Validar() != nil {
		return DatosSelloEvidenciaDocumentalV3Crudos{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return s.sello.Datos()
}

func (SolicitudVerificacionEvidenciaDocumentalV3) String() string {
	return "[SOLICITUD-VERIFICACION-EVIDENCIA-V3-CONFIDENCIAL]"
}
func (s SolicitudVerificacionEvidenciaDocumentalV3) GoString() string { return s.String() }
func (s SolicitudVerificacionEvidenciaDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudVerificacionEvidenciaDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionEvidenciaDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionEvidenciaDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type MetadatosComprobacionEvidenciaDocumentalV3Nominal struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevosMetadatosComprobacionEvidenciaDocumentalV3Nominal restaura la salida
// nominal del conector. No sustituye la comprobacion criptografica privada del
// servicio de aplicacion ni autoriza una confirmacion.
func NuevosMetadatosComprobacionEvidenciaDocumentalV3Nominal(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionEvidenciaDocumentalV3Nominal, error) {
	if solicitud.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(verificadaEn) {
		return MetadatosComprobacionEvidenciaDocumentalV3Nominal{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	resultado := MetadatosComprobacionEvidenciaDocumentalV3Nominal{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}
	return resultado, nil
}

func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		!referenciaEjecucionDocumentalV3Valida(r.verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(r.verificadaEn) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) String() string {
	return "[METADATOS-COMPROBACION-EVIDENCIA-V3-NOMINALES-NO-AUTORITATIVOS]"
}
func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) GoString() string { return r.String() }
func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r MetadatosComprobacionEvidenciaDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionEvidenciaDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionEvidenciaDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type FirmanteEvidenciasRenderizadoDocumentalV3 interface {
	FirmarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	) (SelloEvidenciaDocumentalV3Nominal, error)
}

// VerificadorEvidenciasRenderizadoDocumentalV3 es un conector intercambiable.
// Su salida es nominal; solo el servicio precompuesto puede usarla junto con
// relectura durable y CAS, sin exponerla como autoridad a handlers.
type VerificadorEvidenciasRenderizadoDocumentalV3 interface {
	VerificarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudVerificacionEvidenciaDocumentalV3,
	) (MetadatosComprobacionEvidenciaDocumentalV3Nominal, error)
}

type SolicitudConfirmarEjecucionDocumentalV3 struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	Resultado              ResultadoEfectoRenderizadoDocumentalV3Crudo
	Recibos                RecibosEjecucionDocumentalV3
	Evidencia              DatosEvidenciaRenderizadoDocumentalV3
	Sello                  SelloEvidenciaDocumentalV3Nominal
}

func (s SolicitudConfirmarEjecucionDocumentalV3) Validar() error {
	datosOrden, errOrden := s.OrdenDespachoConsumida.DatosOrden()
	if errOrden != nil || !ordenDespachoDocumentalV3ConsumidaNominalCoincide(
		s.OrdenDespachoConsumida, s.ReservaRef, s.Manifiesto, s.ConsumoDecision,
		s.Evidencia.ConfirmadoEn,
	) ||
		s.Resultado.ValidarContra(s.Manifiesto) != nil || s.Evidencia.Validar() != nil ||
		s.Recibos.ValidarContra(
			s.Manifiesto, s.Resultado, s.ReservaRef, s.ConsumoDecision,
			s.OrdenDespachoConsumida, s.Evidencia.ConfirmadoEn,
		) != nil ||
		s.Evidencia.ReservaRef != s.ReservaRef ||
		!manifiestosEjecucionDocumentalV3Coinciden(s.Evidencia.Manifiesto, s.Manifiesto) ||
		s.Evidencia.SecuenciaCercado != datosOrden.ReciboInicio.SecuenciaCercado ||
		s.Evidencia.HuellaVinculoSHA256 != datosOrden.ReciboInicio.HuellaVinculoCercadoSHA256 ||
		s.Evidencia.ConsumoDecision != s.ConsumoDecision || s.Evidencia.Resultado != s.Resultado {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	huellasRecibos, err := s.Recibos.Huellas()
	if err != nil || huellasRecibos != s.Evidencia.Recibos {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	perfil, err := NuevoPerfilSelloEvidenciaHMACSHA256V3(s.Sello.datos.ClaveID)
	if err != nil {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	firma, err := NuevaSolicitudFirmaEvidenciaRenderizadoDocumentalV3(perfil, s.Evidencia)
	if err != nil || s.Sello.ValidarPara(firma) != nil {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

type EstadoResultadoReconciliacionDocumentalV3 string

const (
	ResultadoReconciliacionDocumentalV3AplicadoExacto EstadoResultadoReconciliacionDocumentalV3 = "aplicado_exacto"
	ResultadoReconciliacionDocumentalV3NoAplicado     EstadoResultadoReconciliacionDocumentalV3 = "no_aplicado_atestado"
	ResultadoReconciliacionDocumentalV3Desconocido    EstadoResultadoReconciliacionDocumentalV3 = "desconocido"
	ResultadoReconciliacionDocumentalV3Conflictivo    EstadoResultadoReconciliacionDocumentalV3 = "conflictivo"
)

func (e EstadoResultadoReconciliacionDocumentalV3) Valido() bool {
	return e == ResultadoReconciliacionDocumentalV3AplicadoExacto ||
		e == ResultadoReconciliacionDocumentalV3NoAplicado ||
		e == ResultadoReconciliacionDocumentalV3Desconocido ||
		e == ResultadoReconciliacionDocumentalV3Conflictivo
}

type SolicitudConsultarEfectoDocumentalV3 struct {
	ReservaRef             string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	OrdenDespachoConsumida OrdenDespachoDocumentalV3ConsumidaNominal
	SolicitadaEn           time.Time
}

func (s SolicitudConsultarEfectoDocumentalV3) Validar() error {
	if !ordenDespachoDocumentalV3ConsumidaNominalCoincide(
		s.OrdenDespachoConsumida, s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.SolicitadaEn,
	) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

type SobreAtestacionReconciliacionDocumentalV3Crudo struct {
	coseSign1 []byte
	huella    string
}

func NuevoSobreAtestacionReconciliacionDocumentalV3Crudo(
	coseSign1 []byte,
) (SobreAtestacionReconciliacionDocumentalV3Crudo, error) {
	sobre := SobreAtestacionReconciliacionDocumentalV3Crudo{
		coseSign1: append([]byte(nil), coseSign1...),
	}
	sobre.huella = documentalcanonico.HuellaBytesSHA256(sobre.coseSign1)
	if sobre.Validar() != nil {
		return SobreAtestacionReconciliacionDocumentalV3Crudo{}, ErrReconciliacionDocumentalV3Invalida
	}
	return sobre, nil
}

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) Validar() error {
	if len(s.coseSign1) < minimoBytesSobreCOSEDocumental ||
		len(s.coseSign1) > maximoBytesSobreCOSEDocumental ||
		bytesEjecucionDocumentalNulos(s.coseSign1) || !esSHA256Hexadecimal(s.huella) ||
		documentalcanonico.HuellaBytesSHA256(s.coseSign1) != s.huella {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) COSESign1() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrReconciliacionDocumentalV3Invalida
	}
	return append([]byte(nil), s.coseSign1...), nil
}

func (s SobreAtestacionReconciliacionDocumentalV3Crudo) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrReconciliacionDocumentalV3Invalida
	}
	return s.huella, nil
}

func (SobreAtestacionReconciliacionDocumentalV3Crudo) String() string {
	return "[ATESTACION-RECONCILIACION-DOCUMENTAL-V3-CRUDA-NO-AUTORITATIVA-REDACTADA]"
}
func (s SobreAtestacionReconciliacionDocumentalV3Crudo) GoString() string { return s.String() }
func (s SobreAtestacionReconciliacionDocumentalV3Crudo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SobreAtestacionReconciliacionDocumentalV3Crudo) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SobreAtestacionReconciliacionDocumentalV3Crudo) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionDocumentalV3Crudo) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type ResultadoConsultaEfectoDocumentalV3Crudo struct {
	ReservaRef             string
	EfectoRef              string
	SecuenciaCercado       uint64
	HuellaVinculoSHA256    string
	HuellaPlanSHA256       string
	Estado                 EstadoResultadoReconciliacionDocumentalV3
	Resultado              ResultadoEfectoRenderizadoDocumentalV3Crudo
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	SobreAtestacion        SobreAtestacionReconciliacionDocumentalV3Crudo
	ConsultadaEn           time.Time
}

func (r ResultadoConsultaEfectoDocumentalV3Crudo) ValidarContra(
	s SolicitudConsultarEfectoDocumentalV3,
) error {
	datos, err := s.Manifiesto.Datos()
	datosOrden, errOrden := s.OrdenDespachoConsumida.DatosOrden()
	if s.Validar() != nil || err != nil || r.ReservaRef != s.ReservaRef ||
		errOrden != nil || r.EfectoRef != datos.EfectoRef ||
		r.SecuenciaCercado != datosOrden.ReciboInicio.SecuenciaCercado ||
		r.HuellaVinculoSHA256 != datosOrden.ReciboInicio.HuellaVinculoCercadoSHA256 ||
		r.HuellaPlanSHA256 != datos.HuellaPlanSHA256 || !r.Estado.Valido() ||
		!referenciaEjecucionDocumentalV3Valida(r.AtestacionRef) ||
		!esSHA256Hexadecimal(r.HuellaAtestacionSHA256) || r.SobreAtestacion.Validar() != nil ||
		!instanteEjecucionDocumentalV3Valido(r.ConsultadaEn) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	if r.Estado == ResultadoReconciliacionDocumentalV3AplicadoExacto {
		if r.Resultado.ValidarContra(s.Manifiesto) != nil {
			return ErrReconciliacionDocumentalV3Invalida
		}
	} else if r.Resultado != (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	huellaSobre, _ := r.SobreAtestacion.HuellaSHA256()
	if huellaSobre != r.HuellaAtestacionSHA256 {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (ResultadoConsultaEfectoDocumentalV3Crudo) String() string {
	return "[RESULTADO-CONSULTA-EFECTO-DOCUMENTAL-V3-CRUDO-COSE-Y-REFERENCIAS-REDACTADOS]"
}
func (r ResultadoConsultaEfectoDocumentalV3Crudo) GoString() string { return r.String() }
func (r ResultadoConsultaEfectoDocumentalV3Crudo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoConsultaEfectoDocumentalV3Crudo) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ResultadoConsultaEfectoDocumentalV3Crudo) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ResultadoConsultaEfectoDocumentalV3Crudo) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudVerificacionReconciliacionDocumentalV3 struct {
	consulta  SolicitudConsultarEfectoDocumentalV3
	resultado ResultadoConsultaEfectoDocumentalV3Crudo
	mensaje   []byte
	huella    string
}

func NuevaSolicitudVerificacionReconciliacionDocumentalV3(
	consulta SolicitudConsultarEfectoDocumentalV3,
	resultado ResultadoConsultaEfectoDocumentalV3Crudo,
) (SolicitudVerificacionReconciliacionDocumentalV3, error) {
	if resultado.ValidarContra(consulta) != nil {
		return SolicitudVerificacionReconciliacionDocumentalV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	mensaje := serializarResultadoReconciliacionDocumentalV3(resultado)
	solicitud := SolicitudVerificacionReconciliacionDocumentalV3{
		consulta: consulta, resultado: resultado,
		mensaje: append([]byte(nil), mensaje...),
		huella: documentalcanonico.HuellaCamposSHA256V3([]string{
			"vec.documentos.solicitud-verificacion-reconciliacion.v3",
			documentalcanonico.HuellaBytesSHA256(mensaje), resultado.HuellaAtestacionSHA256,
		}),
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificacionReconciliacionDocumentalV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	return solicitud, nil
}

func (s SolicitudVerificacionReconciliacionDocumentalV3) Validar() error {
	if s.resultado.ValidarContra(s.consulta) != nil || len(s.mensaje) == 0 ||
		!esSHA256Hexadecimal(s.huella) ||
		documentalcanonico.HuellaCamposSHA256V3([]string{
			"vec.documentos.solicitud-verificacion-reconciliacion.v3",
			documentalcanonico.HuellaBytesSHA256(s.mensaje), s.resultado.HuellaAtestacionSHA256,
		}) != s.huella ||
		string(s.mensaje) != string(serializarResultadoReconciliacionDocumentalV3(s.resultado)) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (s SolicitudVerificacionReconciliacionDocumentalV3) Mensaje() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrReconciliacionDocumentalV3Invalida
	}
	return append([]byte(nil), s.mensaje...), nil
}

func (s SolicitudVerificacionReconciliacionDocumentalV3) Sobre() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrReconciliacionDocumentalV3Invalida
	}
	return s.resultado.SobreAtestacion.COSESign1()
}

func (SolicitudVerificacionReconciliacionDocumentalV3) String() string {
	return "[SOLICITUD-VERIFICACION-RECONCILIACION-V3-CONFIDENCIAL]"
}
func (s SolicitudVerificacionReconciliacionDocumentalV3) GoString() string { return s.String() }
func (s SolicitudVerificacionReconciliacionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudVerificacionReconciliacionDocumentalV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SolicitudVerificacionReconciliacionDocumentalV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SolicitudVerificacionReconciliacionDocumentalV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type MetadatosComprobacionReconciliacionDocumentalV3Nominal struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevosMetadatosComprobacionReconciliacionDocumentalV3Nominal restaura una
// salida nominal del conector. Nunca sustituye la verificacion COSE privada.
func NuevosMetadatosComprobacionReconciliacionDocumentalV3Nominal(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (MetadatosComprobacionReconciliacionDocumentalV3Nominal, error) {
	if solicitud.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(verificadaEn) {
		return MetadatosComprobacionReconciliacionDocumentalV3Nominal{}, ErrReconciliacionDocumentalV3Invalida
	}
	return MetadatosComprobacionReconciliacionDocumentalV3Nominal{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}, nil
}

func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) ValidarPara(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		!referenciaEjecucionDocumentalV3Valida(r.verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(r.verificadaEn) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) String() string {
	return "[METADATOS-COMPROBACION-RECONCILIACION-V3-NOMINALES-NO-AUTORITATIVOS]"
}
func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) GoString() string {
	return r.String()
}
func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r MetadatosComprobacionReconciliacionDocumentalV3Nominal) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (MetadatosComprobacionReconciliacionDocumentalV3Nominal) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*MetadatosComprobacionReconciliacionDocumentalV3Nominal) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type VerificadorAtestacionesReconciliacionDocumentalV3 interface {
	VerificarAtestacionReconciliacionDocumentalV3(
		context.Context,
		SolicitudVerificacionReconciliacionDocumentalV3,
	) (MetadatosComprobacionReconciliacionDocumentalV3Nominal, error)
}

type SolicitudAplicarReconciliacionDocumentalV3 struct {
	Consulta          SolicitudConsultarEfectoDocumentalV3
	ResultadoConsulta ResultadoConsultaEfectoDocumentalV3Crudo
	TieneConfirmacion bool
	Confirmacion      SolicitudConfirmarEjecucionDocumentalV3
}

func (s SolicitudAplicarReconciliacionDocumentalV3) Validar() error {
	solicitudVerificacion, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(
		s.Consulta, s.ResultadoConsulta,
	)
	if err != nil || solicitudVerificacion.Validar() != nil {
		return ErrReconciliacionDocumentalV3Invalida
	}
	if s.ResultadoConsulta.Estado == ResultadoReconciliacionDocumentalV3AplicadoExacto {
		if !s.TieneConfirmacion || s.Confirmacion.Validar() != nil ||
			s.Confirmacion.ReservaRef != s.Consulta.ReservaRef ||
			s.Confirmacion.Resultado != s.ResultadoConsulta.Resultado ||
			s.Confirmacion.Evidencia.ReconciliacionRef != s.ResultadoConsulta.AtestacionRef ||
			s.Confirmacion.Evidencia.HuellaReconciliacionSHA256 !=
				s.ResultadoConsulta.HuellaAtestacionSHA256 ||
			!s.Confirmacion.Evidencia.ReconciliacionConsultadaEn.Equal(
				s.ResultadoConsulta.ConsultadaEn,
			) {
			return ErrReconciliacionDocumentalV3Invalida
		}
	} else if s.TieneConfirmacion || !solicitudConfirmacionDocumentalV3EsCero(s.Confirmacion) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

type ReconciliadorEfectosRenderizadoDocumentalV3 interface {
	ConsultarEfectoRenderizadoDocumentalV3(
		context.Context,
		SolicitudConsultarEfectoDocumentalV3,
	) (ResultadoConsultaEfectoDocumentalV3Crudo, error)
}

// RegistroEjecucionesDocumentalesV3 debe ser durable y transaccional. Las
// restricciones UNIQUE de indice HMAC, borrador, efecto y DecisionRef son
// permanentes. Confirmacion, manifiesto, evidencia, auditoria y outbox se
// confirman juntos; no se ofrece una implementacion en memoria productiva.
//
// ActivarEjecucionDocumentalV3AdoptandoOrdenV4 es la unica frontera de
// adopcion. El adaptador PostgreSQL debe abrir una sola transaccion, bloquear
// y releer por OrdenConsumoDurableV4Ref la fila V4 autoritativa, exigir que su
// orden_ref sea el EfectoRef y cotejar DecisionRef, HuellaDecisionSHA256,
// HuellaPlanSHA256, estado pendiente y contexto comun completos. En el mismo
// COMMIT debe: insertar una adopcion durable 1:1/UNIQUE que referencie la orden
// V4 sin mutarla; guardar en la adopcion sus huellas autoritativas de
// aplicacion, orden y contexto; reclamar UNIQUE por orden, DecisionRef, efecto
// y reserva; activar la reserva; incrementar el cercado; y escribir
// auditoria/outbox. Cualquier cero, cruce, discordancia o replay revierte todo.
// Un reintento con la misma intencion estable (reserva, HMAC, manifiesto,
// orden, terna y huellas) y otro ActivadaEn recupera el mismo token con
// Repetida=true; la marca temporal efimera no crea otra activacion.
//
// MarcarInicioEfectoDocumentalV3 debe, en una sola transaccion, bloquear y
// releer el registro V3 activo, su adopcion durable y la orden V4 autoritativa;
// reconstruir y comparar el VinculoEstableActivacionDocumentalV3 completo;
// verificar el MAC del token con la clave gestionada indicada por su referencia
// (nunca mediante MetadatosComprobacion autocreables); y solo entonces ejecutar
// el CAS activa -> iniciada junto con auditoria y outbox. Cualquier ausencia,
// cruce o discordancia revierte toda la transaccion y no inicia el efecto.
// Devuelve el ReciboInicioEfectoDocumentalV3Nominal creado en ese mismo COMMIT.
//
// ReclamarOrdenDespachoDocumentalV3 debe bloquear el evento outbox y releer el
// recibo de inicio, V3, adopcion y V4. En una sola transaccion ejecuta CAS
// pendiente -> reclamada, incrementa VersionReclamacionCAS y escribe auditoria
// y atestacion. Una segunda reclamacion, incluso identica, falla cerrada. La
// orden devuelta sigue siendo nominal. El servicio de aplicacion privado debe
// verificarla una vez por KMS y entregarla al consumo CAS transaccional; solo
// entonces crea un comando ConsumidaNominal y lo usa dentro de la misma llamada.
//
// ConfirmarEjecucionDocumentalV3 y AplicarReconciliacionDocumentalV3 no pueden
// confiar en comandos nominales, metadatos publicos ni recibos restaurados. El
// registro debe releer inicio, reclamacion, consumo, adopcion V3/V4 y versiones;
// verificar criptograficamente COSE/firmas con dependencias privadas; ejecutar
// el CAS de estado; y confirmar evidencia, auditoria y outbox en un solo COMMIT.
// Abandono e indeterminacion aplican la misma regla de relectura y CAS cerrado.
// El sistema permanece NO-GO hasta existir y probar el servicio application
// precompuesto que mantenga KMS, consumidor, despachador y almacen fuera de
// HTTP, CLI, MCP y modulos funcionales.
type RegistroEjecucionesDocumentalesV3 interface {
	PrepararEjecucionDocumentalV3(
		context.Context,
		SolicitudPrepararEjecucionDocumentalV3,
	) (PreparacionEjecucionDocumentalV3Nominal, error)
	ActivarEjecucionDocumentalV3AdoptandoOrdenV4(
		context.Context,
		SolicitudActivarEjecucionDocumentalV3,
	) (ActivacionEjecucionDocumentalV3Nominal, error)
	MarcarInicioEfectoDocumentalV3(
		context.Context,
		SolicitudIniciarEfectoDocumentalV3,
	) (ReciboInicioEfectoDocumentalV3Nominal, error)
	ReclamarOrdenDespachoDocumentalV3(
		context.Context,
		SolicitudReclamarOrdenDespachoDocumentalV3,
	) (OrdenDespachoDocumentalV3Nominal, error)
	ConfirmarEjecucionDocumentalV3(context.Context, SolicitudConfirmarEjecucionDocumentalV3) error
	AbandonarEjecucionDocumentalV3(context.Context, SolicitudAbandonarEjecucionDocumentalV3) error
	MarcarEjecucionDocumentalV3Indeterminada(
		context.Context,
		SolicitudMarcarEjecucionDocumentalV3Indeterminada,
	) error
	ObtenerEjecucionDocumentalV3(
		context.Context,
		ConsultaEjecucionDocumentalV3,
	) (InstantaneaEjecucionDocumentalV3Nominal, error)
	AplicarReconciliacionDocumentalV3(context.Context, SolicitudAplicarReconciliacionDocumentalV3) error
}

func consultaComponenteEjecucionDocumentalV3(
	descriptor DescriptorPerfilDocumental,
	rol domain.RolComponenteDocumental,
) ConsultaComponenteDocumentalAtestado {
	return ConsultaComponenteDocumentalAtestado{
		Rol: rol, DescriptorPerfilRef: descriptor.Referencia(),
		PublicacionRef: descriptor.PublicacionRef(), PerfilRef: descriptor.Perfil().Referencia(),
		DigestPerfil: descriptor.Perfil().DigestSHA256(), RevisionCatalogo: descriptor.Revision(),
	}
}

func manifiestosEjecucionDocumentalV3Coinciden(
	a, b ManifiestoEjecucionDocumentalV3,
) bool {
	datosA, errA := a.Datos()
	datosB, errB := b.Datos()
	return errA == nil && errB == nil && datosA == datosB
}

func reciboEjecucionDocumentalV3Coincide(
	recibo DatosReciboEjecucionComponenteDocumentalNominal,
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	manifiesto DatosManifiestoEjecucionDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3Crudo,
	reservaRef string,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	datosOrden DatosOrdenDespachoDocumentalV3Nominal,
	ordenConsumida OrdenDespachoDocumentalV3ConsumidaNominal,
) bool {
	huellaOrden, errHuella := ordenConsumida.solicitud.orden.HuellaSHA256()
	if !esSHA256Hexadecimal(recibo.HuellaCompromisoSHA256) || recibo.Operacion != operacion ||
		errHuella != nil ||
		recibo.DescriptorPerfil != manifiesto.DescriptorPerfil ||
		recibo.SituacionOperativa != manifiesto.SituacionOperativa ||
		recibo.DescriptorComponente != componente || recibo.BorradorRef != manifiesto.BorradorRef ||
		recibo.ReservaRef != reservaRef || recibo.EfectoRef != manifiesto.EfectoRef ||
		recibo.HuellaPlanSHA256 != manifiesto.HuellaPlanSHA256 ||
		recibo.SecuenciaCercado != datosOrden.ReciboInicio.SecuenciaCercado ||
		recibo.HuellaVinculoCercadoSHA256 != datosOrden.ReciboInicio.HuellaVinculoCercadoSHA256 ||
		recibo.DecisionRef != consumo.DecisionRef ||
		recibo.EsquemaHuellaDecision != consumo.EsquemaHuellaDecision ||
		recibo.HuellaDecisionSHA256 != consumo.HuellaDecisionSHA256 ||
		recibo.InicioEfectoRef != datosOrden.ReciboInicio.InicioRef ||
		recibo.OutboxInicioRef != datosOrden.ReciboInicio.OutboxInicioRef ||
		recibo.ReclamacionDespachoRef != datosOrden.ReclamacionRef ||
		recibo.ConsumoDespachoRef != ordenConsumida.estado.consumoRef ||
		recibo.OutboxConsumoRef != ordenConsumida.estado.outboxConsumoRef ||
		recibo.VersionInicioCAS != datosOrden.ReciboInicio.VersionInicioCAS ||
		recibo.VersionReclamacionCAS != datosOrden.VersionReclamacionCAS ||
		recibo.VersionConsumoCAS != ordenConsumida.estado.versionConsumoCAS ||
		recibo.HuellaOrdenDespachoSHA256 != huellaOrden ||
		recibo.ComprobacionKMSRef != ordenConsumida.resultado.comprobacionRef ||
		!recibo.ConsumidaEn.Equal(ordenConsumida.estado.consumidaEn) ||
		recibo.HuellaContenidoNeutralHMAC != manifiesto.HuellaEntradaHMAC ||
		recibo.LimiteBytes != manifiesto.LimiteEfectivoBytes ||
		recibo.HuellaSalidaSHA256 != resultado.HuellaSalidaSHA256 ||
		recibo.TamanoSalida != resultado.TamanoSalida {
		return false
	}
	if operacion == OperacionRenderizadoDocumental {
		return recibo.HuellaDocumentoSHA256 == "" && recibo.TamanoDocumento == 0
	}
	return recibo.HuellaDocumentoSHA256 == resultado.HuellaSalidaSHA256 &&
		recibo.TamanoDocumento == resultado.TamanoSalida
}

func huellaManifiestoEjecucionDocumentalV3(d DatosManifiestoEjecucionDocumentalV3) string {
	perfil := d.DescriptorPerfil.Perfil()
	situacion := d.SituacionOperativa
	render := d.ComponenteRender
	verificador := d.ComponenteVerificador
	semantico := d.ComponenteSemantico
	return documentalcanonico.HuellaCamposSHA256V3([]string{
		EsquemaManifiestoEjecucionDocumentalV3,
		d.Consulta.Identidad.Identificador(), d.Consulta.PerfilRef.Identificador(),
		documentalcanonico.Uint64Decimal(d.Consulta.PerfilRef.Version()), d.Consulta.DigestPerfilSHA256,
		documentalcanonico.Uint64Decimal(d.Consulta.RevisionCatalogo.Numero()),
		d.Consulta.RevisionCatalogo.HuellaSHA256(), d.DescriptorPerfil.Referencia(),
		d.DescriptorPerfil.PublicacionRef(), perfil.DigestSHA256(),
		strconvu64(d.DescriptorPerfil.Revision().Numero()), d.DescriptorPerfil.Revision().HuellaSHA256(),
		situacion.PublicacionRef(), strconvu64(situacion.RevisionOperativa()),
		string(situacion.Estado()), situacion.HuellaSHA256(),
		render.Referencia(), render.DigestDeclaracionSHA256(),
		verificador.Referencia(), verificador.DigestDeclaracionSHA256(),
		semantico.Referencia(), semantico.DigestDeclaracionSHA256(),
		d.BorradorRef, d.EfectoRef, d.HuellaEntradaHMAC,
		strconvu64(d.LimiteEfectivoBytes),
	})
}

func huellaVinculoCercadoEjecucionDocumentalV3(
	secuencia uint64,
	huellaVinculoEstable string,
) string {
	return documentalcanonico.HuellaVinculoCercadoV3(secuencia, huellaVinculoEstable)
}

func vinculoCercadoProyectadoDocumentalV3Valido(i InstantaneaEjecucionDocumentalV3Nominal) bool {
	vinculo := VinculoEstableActivacionDocumentalV3{
		ReservaRef: i.ReservaRef, IndiceIdempotenciaHMAC: i.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC: i.HuellaSolicitudHMAC, Manifiesto: i.Manifiesto,
		ConsumoDecision:          i.ConsumoDecision,
		OrdenConsumoDurableV4Ref: i.OrdenConsumoDurableV4Ref,
	}
	huellaVinculoEstable, err := vinculo.HuellaSHA256()
	return err == nil && i.SecuenciaCercado > 0 && esSHA256Hexadecimal(i.HuellaVinculoSHA256) &&
		i.HuellaVinculoSHA256 == huellaVinculoCercadoEjecucionDocumentalV3(
			i.SecuenciaCercado, huellaVinculoEstable,
		)
}

func camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i InstantaneaEjecucionDocumentalV3Nominal) bool {
	return i.EstadoOrigenAbandono == "" && i.MotivoAbandonoRef == ""
}

func camposReconciliacionInstantaneaDocumentalV3Vacios(i InstantaneaEjecucionDocumentalV3Nominal) bool {
	return i.ReconciliacionRef == "" && i.HuellaReconciliacionSHA256 == ""
}

func camposReconciliacionConfirmadaDocumentalV3Validos(i InstantaneaEjecucionDocumentalV3Nominal) bool {
	return camposReconciliacionInstantaneaDocumentalV3Vacios(i) ||
		(referenciaEjecucionDocumentalV3Valida(i.ReconciliacionRef) &&
			esSHA256Hexadecimal(i.HuellaReconciliacionSHA256))
}

func serializarEvidenciaRenderizadoDocumentalV3(d DatosEvidenciaRenderizadoDocumentalV3) []byte {
	huellaManifiesto, _ := d.Manifiesto.HuellaSHA256()
	valores := []string{
		EsquemaEvidenciaRenderizadoV3, d.ReservaRef, d.IndiceIdempotenciaHMAC,
		d.HuellaSolicitudHMAC, huellaManifiesto, strconvu64(d.SecuenciaCercado),
		d.HuellaVinculoSHA256, d.ClaveAtestacionCercadoRef,
		d.HuellaMACCercadoSHA256, d.EvidenciaAtestacionCercadoRef,
		d.VerificacionCercadoRef, d.VerificadoCercadoEn.Format(time.RFC3339Nano),
		d.ConsumoDecision.DecisionRef,
		d.ConsumoDecision.EfectoRef, d.ConsumoDecision.EsquemaHuellaDecision,
		d.ConsumoDecision.HuellaDecisionSHA256, d.ConsumoDecision.HuellaPlanSHA256,
		d.Resultado.BorradorRef, d.Resultado.EfectoRef, d.Resultado.ContenidoRef,
		d.Resultado.ContenidoVersion, d.Resultado.ConectorRef, d.Resultado.MIME,
		d.Resultado.HuellaSalidaSHA256, strconvu64(d.Resultado.TamanoSalida),
		d.Resultado.EvidenciaOperacionRef,
		d.Recibos.ReciboRenderRef, d.Recibos.HuellaSobreRenderSHA256,
		d.Recibos.HuellaReciboRenderSHA256, d.Recibos.ReciboEstructuralRef,
		d.Recibos.HuellaSobreEstructuralSHA256, d.Recibos.HuellaReciboEstructuralSHA256,
		d.Recibos.ReciboSemanticoRef, d.Recibos.HuellaSobreSemanticoSHA256,
		d.Recibos.HuellaReciboSemanticoSHA256, d.GeneradoEn.Format(time.RFC3339Nano),
		d.ConfirmadoEn.Format(time.RFC3339Nano), d.ReconciliacionRef,
		d.HuellaReconciliacionSHA256, d.ReconciliacionConsultadaEn.Format(time.RFC3339Nano),
		d.VerificacionReconciliacionRef, d.ReconciliacionVerificadaEn.Format(time.RFC3339Nano),
	}
	return documentalcanonico.SerializarCamposV3(valores)
}

func serializarAtestacionTokenCercadoDocumentalV3(
	vinculo VinculoEstableActivacionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3Nominal,
) []byte {
	proyeccion, valida := proyectarTokenCercadoDocumentalV3(token, vinculo)
	if !valida || token.ValidarPara(vinculo) != nil {
		return nil
	}
	return proyeccion.MensajeAtestacion()
}

func serializarMaterialCrudoVerificacionOrdenDespachoDocumentalV3(
	v VinculosCrudosVerificacionOrdenDespachoDocumentalV3,
	cercado, inicio, reclamacion PruebaCrudaAtestacionDespachoDocumentalV3,
) []byte {
	proyeccion := documentalcanonico.DatosMaterialDespachoV3{
		Vinculos:    proyectarVinculosMaterialDespachoDocumentalV3(v),
		Cercado:     proyectarPerfilMaterialDespachoDocumentalV3(cercado),
		Inicio:      proyectarPerfilMaterialDespachoDocumentalV3(inicio),
		Reclamacion: proyectarPerfilMaterialDespachoDocumentalV3(reclamacion),
	}
	return proyeccion.Bytes()
}

func clonarPruebaCrudaAtestacionDespachoDocumentalV3(
	p PruebaCrudaAtestacionDespachoDocumentalV3,
) PruebaCrudaAtestacionDespachoDocumentalV3 {
	p.mensajeCanonico = append([]byte(nil), p.mensajeCanonico...)
	p.sobreCriptografico = append([]byte(nil), p.sobreCriptografico...)
	return p
}

func clonarMaterialCrudoVerificacionOrdenDespachoDocumentalV3(
	m MaterialCrudoVerificacionOrdenDespachoDocumentalV3,
) MaterialCrudoVerificacionOrdenDespachoDocumentalV3 {
	m.orden = clonarOrdenDespachoDocumentalV3Nominal(m.orden)
	m.vinculo = clonarVinculoEstableActivacionDocumentalV3(m.vinculo)
	m.token = clonarTokenCercadoEjecucionDocumentalV3(m.token)
	m.cercado = clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.cercado)
	m.inicio = clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.inicio)
	m.reclamacion = clonarPruebaCrudaAtestacionDespachoDocumentalV3(m.reclamacion)
	m.mensaje = append([]byte(nil), m.mensaje...)
	return m
}

func clonarVinculoEstableActivacionDocumentalV3(
	v VinculoEstableActivacionDocumentalV3,
) VinculoEstableActivacionDocumentalV3 {
	if v.Manifiesto.datos != nil {
		datos := *v.Manifiesto.datos
		v.Manifiesto.datos = &datos
	}
	return v
}

func clonarTokenCercadoEjecucionDocumentalV3(
	t TokenCercadoEjecucionDocumentalV3Nominal,
) TokenCercadoEjecucionDocumentalV3Nominal {
	t.vinculoEstable = clonarVinculoEstableActivacionDocumentalV3(t.vinculoEstable)
	t.macAtestacion = append([]byte(nil), t.macAtestacion...)
	return t
}

func clonarOrdenDespachoDocumentalV3Nominal(
	o OrdenDespachoDocumentalV3Nominal,
) OrdenDespachoDocumentalV3Nominal {
	if o.datos == nil {
		return o
	}
	datos := *o.datos
	datos.ReciboInicio.AtestacionInicio = clonarPruebaCrudaAtestacionDespachoDocumentalV3(
		o.datos.ReciboInicio.AtestacionInicio,
	)
	datos.AtestacionReclamacion = clonarPruebaCrudaAtestacionDespachoDocumentalV3(
		o.datos.AtestacionReclamacion,
	)
	o.datos = &datos
	return o
}

func serializarResultadoReconciliacionDocumentalV3(
	r ResultadoConsultaEfectoDocumentalV3Crudo,
) []byte {
	valores := []string{
		"vec.documentos.resultado-reconciliacion.v3", r.ReservaRef, r.EfectoRef,
		strconvu64(r.SecuenciaCercado), r.HuellaVinculoSHA256, r.HuellaPlanSHA256,
		string(r.Estado), r.Resultado.BorradorRef, r.Resultado.EfectoRef,
		r.Resultado.ContenidoRef, r.Resultado.ContenidoVersion, r.Resultado.ConectorRef,
		r.Resultado.MIME, r.Resultado.HuellaSalidaSHA256,
		strconvu64(r.Resultado.TamanoSalida), r.Resultado.EvidenciaOperacionRef,
		r.AtestacionRef, r.ConsultadaEn.Format(time.RFC3339Nano),
	}
	return documentalcanonico.SerializarCamposV3(valores)
}

func huellaSolicitudVerificacionTokenCercadoDocumentalV3(
	mensaje, mac []byte,
) string {
	return documentalcanonico.HuellaSolicitudVerificacionTokenV3(mensaje, mac)
}

func huellaSolicitudVerificacionEvidenciaDocumentalV3(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	sello SelloEvidenciaDocumentalV3Nominal,
) string {
	huellaMensaje, _ := firma.HuellaMensajeSHA256()
	datos, _ := sello.Datos()
	return documentalcanonico.HuellaCamposSHA256V3([]string{
		"vec.documentos.solicitud-verificacion-evidencia.v3", huellaMensaje,
		datos.Algoritmo, datos.ClaveID, datos.Audiencia,
		documentalcanonico.HuellaBytesSHA256(datos.Firma), datos.EvidenciaOperacionRef,
		datos.FirmadoEn.Format(time.RFC3339Nano),
	})
}

func ordenDespachoDocumentalV3ConsumidaNominalCoincide(
	orden OrdenDespachoDocumentalV3ConsumidaNominal,
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	instante time.Time,
) bool {
	vinculo, err := orden.VinculoActivacion()
	return err == nil && orden.ValidarEn(instante) == nil &&
		vinculo.ReservaRef == reservaRef &&
		manifiestosEjecucionDocumentalV3Coinciden(vinculo.Manifiesto, manifiesto) &&
		vinculo.ConsumoDecision == consumo
}

func ordenDespachoDocumentalV3ConsumidaNominalEsCero(
	orden OrdenDespachoDocumentalV3ConsumidaNominal,
) bool {
	return reflect.ValueOf(orden).IsZero()
}

func proyectarVinculoActivacionDocumentalV3(
	v VinculoEstableActivacionDocumentalV3,
) (documentalcanonico.DatosVinculoActivacionV3, bool) {
	datos, errDatos := v.Manifiesto.Datos()
	huellaManifiesto, errHuella := v.Manifiesto.HuellaSHA256()
	if errDatos != nil || errHuella != nil {
		return documentalcanonico.DatosVinculoActivacionV3{}, false
	}
	return documentalcanonico.DatosVinculoActivacionV3{
		ReservaRef: v.ReservaRef, IndiceIdempotenciaHMAC: v.IndiceIdempotenciaHMAC,
		HuellaSolicitudHMAC: v.HuellaSolicitudHMAC, HuellaEntradaHMAC: datos.HuellaEntradaHMAC,
		HuellaManifiestoSHA256: huellaManifiesto, EfectoManifiestoRef: datos.EfectoRef,
		HuellaPlanManifiestoSHA256: datos.HuellaPlanSHA256,
		OrdenConsumoDurableV4Ref:   v.OrdenConsumoDurableV4Ref,
		DecisionRef:                v.ConsumoDecision.DecisionRef, EfectoDecisionRef: v.ConsumoDecision.EfectoRef,
		EsquemaHuellaDecision:         v.ConsumoDecision.EsquemaHuellaDecision,
		EsquemaHuellaDecisionEsperado: EsquemaHuellaDecisionAutorizacionReforzadaV1,
		HuellaDecisionSHA256:          v.ConsumoDecision.HuellaDecisionSHA256,
		HuellaPlanDecisionSHA256:      v.ConsumoDecision.HuellaPlanSHA256,
	}, true
}

func proyectarTokenCercadoDocumentalV3(
	t TokenCercadoEjecucionDocumentalV3Nominal,
	vinculo VinculoEstableActivacionDocumentalV3,
) (documentalcanonico.DatosTokenCercadoV3, bool) {
	datos, errDatos := vinculo.Manifiesto.Datos()
	huellaEsperada, errEsperada := vinculo.HuellaSHA256()
	huellaInterna, errInterna := t.vinculoEstable.HuellaSHA256()
	if errDatos != nil || errEsperada != nil || errInterna != nil {
		return documentalcanonico.DatosTokenCercadoV3{}, false
	}
	return documentalcanonico.DatosTokenCercadoV3{
		Valor: t.valor, Secuencia: t.secuencia,
		HuellaVinculoEstableSHA256:  t.huellaVinculoEstable,
		HuellaVinculoEsperadoSHA256: huellaEsperada,
		HuellaVinculoInternoSHA256:  huellaInterna,
		HuellaVinculoCercadoSHA256:  t.huellaVinculo,
		ClaveAtestacionRef:          t.claveAtestacionRef, RevisionClave: t.revisionClave,
		MACAtestacion: t.macAtestacion, EvidenciaOperacionRef: t.evidenciaOperacionRef,
		ClaveHuellaEntradaHMAC: documentalcanonico.ClaveHMACSHA256V3(datos.HuellaEntradaHMAC),
	}, true
}

func proyectarPruebaAtestacionDespachoDocumentalV3(
	p PruebaCrudaAtestacionDespachoDocumentalV3,
) documentalcanonico.DatosPruebaAtestacionDespachoV3 {
	return documentalcanonico.DatosPruebaAtestacionDespachoV3{
		Algoritmo: p.algoritmo, Audiencia: p.audiencia, Contexto: p.contexto,
		ClaveGestionadaRef:      p.claveGestionadaRef,
		RevisionClaveGestionada: p.revisionClaveGestionada,
		EvidenciaOperacionRef:   p.evidenciaOperacionRef,
		MensajeCanonico:         p.mensajeCanonico, SobreCriptografico: p.sobreCriptografico,
		HuellaMensajeSHA256: p.huellaMensajeSHA256, HuellaSobreSHA256: p.huellaSobreSHA256,
	}
}

func proyectarReciboInicioEfectoDocumentalV3(
	d DatosReciboInicioEfectoDocumentalV3Nominal,
) (documentalcanonico.DatosReciboInicioEfectoV3, bool) {
	prueba := proyectarPruebaAtestacionDespachoDocumentalV3(d.AtestacionInicio)
	return documentalcanonico.DatosReciboInicioEfectoV3{
		InicioRef: d.InicioRef, ReservaRef: d.ReservaRef,
		HuellaVinculoEstableSHA256: d.HuellaVinculoEstableSHA256,
		SecuenciaCercado:           d.SecuenciaCercado,
		HuellaVinculoCercadoSHA256: d.HuellaVinculoCercadoSHA256,
		OrdenConsumoDurableV4Ref:   d.OrdenConsumoDurableV4Ref,
		VersionInicioCAS:           d.VersionInicioCAS, AuditoriaInicioRef: d.AuditoriaInicioRef,
		OutboxInicioRef: d.OutboxInicioRef, EvidenciaOperacionRef: d.AtestacionInicio.evidenciaOperacionRef,
		AtestacionValida: prueba.Validar(), HuellaAtestacionSHA256: prueba.HuellaSHA256(),
		IniciadoEn: d.IniciadoEn,
	}, true
}

func proyectarSolicitudReclamacionDocumentalV3(
	s SolicitudReclamarOrdenDespachoDocumentalV3,
) documentalcanonico.DatosSolicitudReclamacionV3 {
	return documentalcanonico.DatosSolicitudReclamacionV3{
		ReclamacionRef: s.ReclamacionRef, InicioRef: s.InicioRef,
		OutboxRef: s.OutboxRef, ConsumidorRef: s.ConsumidorRef,
		ReclamadaEn: s.ReclamadaEn, ExpiraEn: s.ExpiraEn,
	}
}

func proyectarOrdenDespachoDocumentalV3(
	d DatosOrdenDespachoDocumentalV3Nominal,
) (documentalcanonico.DatosOrdenDespachoV3, bool) {
	recibo := ReciboInicioEfectoDocumentalV3Nominal{datos: &d.ReciboInicio}
	huellaRecibo, errRecibo := recibo.HuellaSHA256()
	evidenciaRef, errEvidencia := d.AtestacionReclamacion.EvidenciaOperacionRef()
	solicitud := SolicitudReclamarOrdenDespachoDocumentalV3{
		ReclamacionRef: d.ReclamacionRef, InicioRef: d.ReciboInicio.InicioRef,
		OutboxRef: d.ReciboInicio.OutboxInicioRef, ConsumidorRef: d.ConsumidorRef,
		ReclamadaEn: d.ReclamadaEn, ExpiraEn: d.ExpiraEn,
	}
	mensajeEsperado, errMensaje := MensajeCanonicoAtestacionReclamacionDespachoDocumentalV3(
		recibo, solicitud, d.VersionReclamacionCAS, d.AuditoriaReclamacionRef, evidenciaRef,
	)
	mensajeAtestacion, errMaterial := d.AtestacionReclamacion.MensajeCanonico()
	if errRecibo != nil || errEvidencia != nil || errMensaje != nil || errMaterial != nil {
		return documentalcanonico.DatosOrdenDespachoV3{}, false
	}
	return documentalcanonico.DatosOrdenDespachoV3{
		Solicitud:                   proyectarSolicitudReclamacionDocumentalV3(solicitud),
		HuellaReciboInicioSHA256:    d.HuellaReciboInicioSHA256,
		HuellaReciboCalculadaSHA256: huellaRecibo,
		VersionReclamacionCAS:       d.VersionReclamacionCAS,
		AuditoriaReclamacionRef:     d.AuditoriaReclamacionRef,
		EvidenciaOperacionRef:       evidenciaRef,
		AtestacionValida:            d.AtestacionReclamacion.Validar() == nil,
		HuellaAtestacionSHA256:      d.AtestacionReclamacion.huellaSHA256(),
		MensajeAtestacion:           mensajeAtestacion, MensajeEsperado: mensajeEsperado,
		IniciadoEn: d.ReciboInicio.IniciadoEn,
	}, true
}

func proyectarVinculosMaterialDespachoDocumentalV3(
	v VinculosCrudosVerificacionOrdenDespachoDocumentalV3,
) documentalcanonico.VinculosMaterialDespachoV3 {
	return documentalcanonico.VinculosMaterialDespachoV3{
		InicioRef: v.InicioRef, AtestacionInicioRef: v.AtestacionInicioRef,
		ReclamacionRef: v.ReclamacionRef, AtestacionReclamacionRef: v.AtestacionReclamacionRef,
		OrdenConsumoDurableV4Ref:   v.OrdenConsumoDurableV4Ref,
		HuellaOrdenDespachoSHA256:  v.HuellaOrdenDespachoSHA256,
		HuellaReciboInicioSHA256:   v.HuellaReciboInicioSHA256,
		HuellaVinculoEstableSHA256: v.HuellaVinculoEstableSHA256,
		HuellaVinculoCercadoSHA256: v.HuellaVinculoCercadoSHA256,
		SecuenciaCercado:           v.SecuenciaCercado, VersionInicioCAS: v.VersionInicioCAS,
		VersionReclamacionCAS: v.VersionReclamacionCAS,
	}
}

func proyectarPerfilMaterialDespachoDocumentalV3(
	p PruebaCrudaAtestacionDespachoDocumentalV3,
) documentalcanonico.PerfilMaterialDespachoV3 {
	return documentalcanonico.PerfilMaterialDespachoV3{
		Valido: p.Validar() == nil, Audiencia: p.audiencia,
		ClaveGestionadaRef:      p.claveGestionadaRef,
		RevisionClaveGestionada: p.revisionClaveGestionada, HuellaSHA256: p.huellaSHA256(),
	}
}

func referenciaEjecucionDocumentalV3Valida(valor string) bool {
	return documentalcanonico.ReferenciaEjecucionV3Valida(valor)
}

func instanteEjecucionDocumentalV3Valido(instante time.Time) bool {
	return documentalcanonico.InstanteV3Valido(instante)
}

func solicitudConfirmacionDocumentalV3EsCero(
	s SolicitudConfirmarEjecucionDocumentalV3,
) bool {
	return s.ReservaRef == "" && s.Manifiesto.datos == nil &&
		s.ConsumoDecision == (ConsumoDecisionEjecucionDocumentalV3{}) &&
		ordenDespachoDocumentalV3ConsumidaNominalEsCero(s.OrdenDespachoConsumida) &&
		s.Resultado == (ResultadoEfectoRenderizadoDocumentalV3Crudo{}) &&
		reflect.ValueOf(s.Recibos).IsZero() &&
		s.Evidencia == (DatosEvidenciaRenderizadoDocumentalV3{}) &&
		s.Sello.datos.Algoritmo == "" && s.Sello.datos.ClaveID == "" &&
		s.Sello.datos.Audiencia == "" && s.Sello.datos.HuellaMensajeSHA256 == "" &&
		len(s.Sello.datos.Firma) == 0 && s.Sello.datos.EvidenciaOperacionRef == "" &&
		s.Sello.datos.FirmadoEn.IsZero()
}

func datosSelloEvidenciaDocumentalV3ValidosPara(
	datos DatosSelloEvidenciaDocumentalV3Crudos,
	perfil PerfilSelloEvidenciaDocumentalV3,
	huellaMensaje string,
) bool {
	return perfil.Validar() == nil && datos.Algoritmo == perfil.Algoritmo &&
		datos.ClaveID == perfil.ClaveID && datos.Audiencia == perfil.Audiencia &&
		esSHA256Hexadecimal(huellaMensaje) && datos.HuellaMensajeSHA256 == huellaMensaje &&
		len(datos.Firma) == tamanoFirmaHMACSHA256V3 &&
		!bytesEjecucionDocumentalNulos(datos.Firma) &&
		referenciaEjecucionDocumentalV3Valida(datos.EvidenciaOperacionRef) &&
		instanteEjecucionDocumentalV3Valido(datos.FirmadoEn)
}

func strconvu64(valor uint64) string { return documentalcanonico.Uint64Decimal(valor) }
