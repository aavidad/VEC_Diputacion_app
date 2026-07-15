package ports

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrManifiestoEjecucionDocumentalV3Invalido = errors.New("vec: manifiesto de ejecucion documental v3 invalido")
	ErrReservaEjecucionDocumentalV3Invalida    = errors.New("vec: reserva de ejecucion documental v3 invalida")
	ErrTokenCercadoDocumentalV3Invalido        = errors.New("vec: token de cercado documental v3 invalido")
	ErrTransicionEjecucionDocumentalV3Invalida = errors.New("vec: transicion de ejecucion documental v3 invalida")
	ErrReconciliacionDocumentalV3Invalida      = errors.New("vec: reconciliacion documental v3 invalida")
	ErrSelloEvidenciaDocumentalV3Invalido      = errors.New("vec: sello de evidencia documental v3 invalido")
	ErrSerializacionSecretoDocumentalV3        = errors.New("vec: serializacion de secreto documental v3 prohibida")
)

const (
	EsquemaManifiestoEjecucionDocumentalV3 = "vec.documentos.manifiesto-ejecucion.v3"
	EsquemaEvidenciaRenderizadoV3          = "vec.documentos.evidencia-renderizado.v3"
	AlgoritmoSelloEvidenciaHMACSHA256V3    = "hmac-sha256"
	AudienciaSelloEvidenciaRenderizadoV3   = "vec.documentos.evidencia-renderizado.v3"
	AudienciaAtestacionTokenCercadoV3      = "vec.documentos.token-cercado.v3"

	duracionMaximaReservaEjecucionDocumentalV3 = 15 * time.Minute
	tamanoMaximoSalidaEjecucionDocumentalV3    = uint64(256 * 1024 * 1024)
	tamanoFirmaHMACSHA256V3                    = sha256.Size
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
func (ManifiestoEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*ManifiestoEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (ManifiestoEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
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
		!clavesHMACDocumentalesV3Distintas(
			s.IndiceIdempotenciaHMAC, s.HuellaSolicitudHMAC, datos.HuellaEntradaHMAC,
		) || !instanteEjecucionDocumentalV3Valido(s.SolicitadaEn) ||
		!instanteEjecucionDocumentalV3Valido(s.ExpiraEn) ||
		!s.ExpiraEn.After(s.SolicitadaEn) ||
		s.ExpiraEn.Sub(s.SolicitadaEn) > duracionMaximaReservaEjecucionDocumentalV3 {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
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

type PreparacionEjecucionDocumentalV3 struct {
	ReservaRef  string
	BorradorRef string
	EfectoRef   string
	Repetida    bool
	Estado      EstadoEjecucionDocumentalV3
}

func (p PreparacionEjecucionDocumentalV3) ValidarContra(
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

// TokenCercadoEjecucionDocumentalV3 combina un secreto aleatorio con una
// secuencia monotona. La secuencia de cercado es distinta de la secuencia
// operativa del perfil, aunque la huella liga ambas de forma inseparable.
type TokenCercadoEjecucionDocumentalV3 struct {
	valor                 string
	secuencia             uint64
	huellaVinculo         string
	claveAtestacionRef    string
	macAtestacion         []byte
	evidenciaOperacionRef string
}

// NuevoTokenCercadoEjecucionDocumentalV3 queda reservado al adaptador de
// RegistroEjecucionesDocumentalesV3. Solo construye el sobre: NO autentica el
// token. Antes de iniciar, confirmar o reconciliar es obligatorio obtener un
// ResultadoVerificacionTokenCercadoDocumentalV3 del verificador configurado.
func NuevoTokenCercadoEjecucionDocumentalV3(
	valor string,
	secuencia uint64,
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	claveAtestacionRef string,
	macAtestacion []byte,
	evidenciaOperacionRef string,
) (TokenCercadoEjecucionDocumentalV3, error) {
	if !referenciaEjecucionDocumentalV3Valida(valor) || secuencia == 0 ||
		!referenciaEjecucionDocumentalV3Valida(reservaRef) || consumo.ValidarContra(manifiesto) != nil ||
		!referenciaEjecucionDocumentalV3Valida(claveAtestacionRef) ||
		!referenciaEjecucionDocumentalV3Valida(evidenciaOperacionRef) ||
		len(macAtestacion) != tamanoFirmaHMACSHA256V3 {
		return TokenCercadoEjecucionDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	token := TokenCercadoEjecucionDocumentalV3{
		valor: valor, secuencia: secuencia,
		huellaVinculo: huellaVinculoCercadoEjecucionDocumentalV3(
			reservaRef, secuencia, manifiesto, consumo,
		),
		claveAtestacionRef:    claveAtestacionRef,
		macAtestacion:         append([]byte(nil), macAtestacion...),
		evidenciaOperacionRef: evidenciaOperacionRef,
	}
	if token.ValidarPara(reservaRef, manifiesto, consumo) != nil {
		return TokenCercadoEjecucionDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return token, nil
}

func (t TokenCercadoEjecucionDocumentalV3) ValidarPara(
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
) error {
	datos, err := manifiesto.Datos()
	if !referenciaEjecucionDocumentalV3Valida(t.valor) || t.secuencia == 0 ||
		!esSHA256Hexadecimal(t.huellaVinculo) ||
		!referenciaEjecucionDocumentalV3Valida(reservaRef) || err != nil ||
		consumo.ValidarContra(manifiesto) != nil ||
		!referenciaEjecucionDocumentalV3Valida(t.claveAtestacionRef) ||
		t.claveAtestacionRef == claveHMACDocumentalV3(datos.HuellaEntradaHMAC) ||
		len(t.macAtestacion) != tamanoFirmaHMACSHA256V3 ||
		bytesEjecucionDocumentalNulos(t.macAtestacion) ||
		!referenciaEjecucionDocumentalV3Valida(t.evidenciaOperacionRef) ||
		t.huellaVinculo != huellaVinculoCercadoEjecucionDocumentalV3(
			reservaRef, t.secuencia, manifiesto, consumo,
		) {
		return ErrTokenCercadoDocumentalV3Invalido
	}
	return nil
}

func (t TokenCercadoEjecucionDocumentalV3) Secuencia() uint64 { return t.secuencia }
func (t TokenCercadoEjecucionDocumentalV3) HuellaVinculoSHA256() string {
	return t.huellaVinculo
}
func (TokenCercadoEjecucionDocumentalV3) String() string {
	return "[TOKEN-CERCADO-DOCUMENTAL-V3-CONFIDENCIAL]"
}
func (t TokenCercadoEjecucionDocumentalV3) GoString() string { return t.String() }
func (t TokenCercadoEjecucionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, t.String())
}
func (TokenCercadoEjecucionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (TokenCercadoEjecucionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*TokenCercadoEjecucionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*TokenCercadoEjecucionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudVerificacionTokenCercadoDocumentalV3 struct {
	reservaRef string
	manifiesto ManifiestoEjecucionDocumentalV3
	consumo    ConsumoDecisionEjecucionDocumentalV3
	token      TokenCercadoEjecucionDocumentalV3
	mensaje    []byte
	huella     string
}

func NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
) (SolicitudVerificacionTokenCercadoDocumentalV3, error) {
	if token.ValidarPara(reservaRef, manifiesto, consumo) != nil {
		return SolicitudVerificacionTokenCercadoDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	mensaje := serializarAtestacionTokenCercadoDocumentalV3(reservaRef, manifiesto, consumo, token)
	solicitud := SolicitudVerificacionTokenCercadoDocumentalV3{
		reservaRef: reservaRef, manifiesto: manifiesto, consumo: consumo, token: token,
		mensaje: append([]byte(nil), mensaje...),
		huella:  huellaSolicitudVerificacionTokenCercadoDocumentalV3(mensaje, token.macAtestacion),
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificacionTokenCercadoDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return solicitud, nil
}

func (s SolicitudVerificacionTokenCercadoDocumentalV3) Validar() error {
	if s.token.ValidarPara(s.reservaRef, s.manifiesto, s.consumo) != nil ||
		len(s.mensaje) == 0 || !esSHA256Hexadecimal(s.huella) ||
		huellaSolicitudVerificacionTokenCercadoDocumentalV3(
			s.mensaje, s.token.macAtestacion,
		) != s.huella ||
		string(s.mensaje) != string(serializarAtestacionTokenCercadoDocumentalV3(
			s.reservaRef, s.manifiesto, s.consumo, s.token,
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

type ResultadoVerificacionTokenCercadoDocumentalV3 struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevoResultadoVerificacionTokenCercadoDocumentalV3 queda reservado al
// adaptador de VerificadorTokensCercadoDocumentalV3. El producto no puede
// invocarlo como sustituto de la comprobacion MAC con la clave gestionada.
func NuevoResultadoVerificacionTokenCercadoDocumentalV3(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (ResultadoVerificacionTokenCercadoDocumentalV3, error) {
	if solicitud.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(verificadaEn) {
		return ResultadoVerificacionTokenCercadoDocumentalV3{}, ErrTokenCercadoDocumentalV3Invalido
	}
	return ResultadoVerificacionTokenCercadoDocumentalV3{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}, nil
}

func (r ResultadoVerificacionTokenCercadoDocumentalV3) ValidarPara(
	solicitud SolicitudVerificacionTokenCercadoDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		!referenciaEjecucionDocumentalV3Valida(r.verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(r.verificadaEn) {
		return ErrTokenCercadoDocumentalV3Invalido
	}
	return nil
}

type VerificadorTokensCercadoDocumentalV3 interface {
	VerificarTokenCercadoDocumentalV3(
		context.Context,
		SolicitudVerificacionTokenCercadoDocumentalV3,
	) (ResultadoVerificacionTokenCercadoDocumentalV3, error)
}

type SolicitudActivarEjecucionDocumentalV3 struct {
	ReservaRef             string
	IndiceIdempotenciaHMAC string
	HuellaSolicitudHMAC    string
	Manifiesto             ManifiestoEjecucionDocumentalV3
	ConsumoDecision        ConsumoDecisionEjecucionDocumentalV3
	ActivadaEn             time.Time
}

func (s SolicitudActivarEjecucionDocumentalV3) Validar() error {
	datos, err := s.Manifiesto.Datos()
	if !referenciaEjecucionDocumentalV3Valida(s.ReservaRef) ||
		!hmacSHA256PuertoValido(s.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(s.HuellaSolicitudHMAC) || err != nil ||
		!clavesHMACDocumentalesV3Distintas(
			s.IndiceIdempotenciaHMAC, s.HuellaSolicitudHMAC, datos.HuellaEntradaHMAC,
		) || s.ConsumoDecision.ValidarContra(s.Manifiesto) != nil ||
		!instanteEjecucionDocumentalV3Valido(s.ActivadaEn) {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

type ActivacionEjecucionDocumentalV3 struct {
	Token    TokenCercadoEjecucionDocumentalV3
	Repetida bool
}

func (a ActivacionEjecucionDocumentalV3) ValidarContra(s SolicitudActivarEjecucionDocumentalV3) error {
	if s.Validar() != nil || a.Token.ValidarPara(s.ReservaRef, s.Manifiesto, s.ConsumoDecision) != nil {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

type SolicitudIniciarEfectoDocumentalV3 struct {
	ReservaRef          string
	Manifiesto          ManifiestoEjecucionDocumentalV3
	ConsumoDecision     ConsumoDecisionEjecucionDocumentalV3
	Token               TokenCercadoEjecucionDocumentalV3
	VerificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
	IniciadoEn          time.Time
}

func (s SolicitudIniciarEfectoDocumentalV3) Validar() error {
	if !instanteEjecucionDocumentalV3Valido(s.IniciadoEn) ||
		!verificacionTokenCercadoDocumentalV3Valida(
			s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.Token, s.VerificacionCercado,
		) {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

// ResultadoEfectoRenderizadoDocumentalV3 conserva una referencia exacta y
// versionada del objeto; nunca una URL temporal ni los bytes del documento.
type ResultadoEfectoRenderizadoDocumentalV3 struct {
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

func (r ResultadoEfectoRenderizadoDocumentalV3) ValidarContra(
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

// HuellasRecibosEjecucionDocumentalV3 es la proyeccion persistible de los
// tres recibos COSE ya verificados. No concede autoridad por si sola: la
// confirmacion exige tambien los valores tipados devueltos por el verificador
// criptografico de recibos.
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
	Render      ReciboEjecucionComponenteDocumentalVerificado
	Estructural ReciboEjecucionComponenteDocumentalVerificado
	Semantico   ReciboEjecucionComponenteDocumentalVerificado
}

func (r RecibosEjecucionDocumentalV3) ValidarContra(
	manifiesto ManifiestoEjecucionDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3,
	reservaRef string,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
) error {
	datosManifiesto, errManifiesto := manifiesto.Datos()
	solicitudCercado, errCercado := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		reservaRef, manifiesto, consumo, token,
	)
	datosRender, errRender := r.Render.Datos()
	datosEstructural, errEstructural := r.Estructural.Datos()
	datosSemantico, errSemantico := r.Semantico.Datos()
	if errManifiesto != nil || errCercado != nil || verificacion.ValidarPara(solicitudCercado) != nil ||
		resultado.ValidarContra(manifiesto) != nil ||
		errRender != nil || errEstructural != nil || errSemantico != nil ||
		!reciboEjecucionDocumentalV3Coincide(
			datosRender, OperacionRenderizadoDocumental,
			datosManifiesto.ComponenteRender, datosManifiesto, resultado,
			reservaRef, consumo, token, solicitudCercado, verificacion,
		) || !reciboEjecucionDocumentalV3Coincide(
		datosEstructural, OperacionValidacionEstructuralDocumental,
		datosManifiesto.ComponenteVerificador, datosManifiesto, resultado,
		reservaRef, consumo, token, solicitudCercado, verificacion,
	) || !reciboEjecucionDocumentalV3Coincide(
		datosSemantico, OperacionVerificacionSemanticaDocumental,
		datosManifiesto.ComponenteSemantico, datosManifiesto, resultado,
		reservaRef, consumo, token, solicitudCercado, verificacion,
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
	ReservaRef          string
	Manifiesto          ManifiestoEjecucionDocumentalV3
	EstadoEsperado      EstadoEjecucionDocumentalV3
	ConsumoDecision     ConsumoDecisionEjecucionDocumentalV3
	Token               TokenCercadoEjecucionDocumentalV3
	VerificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
	MotivoRef           string
	AbandonadaEn        time.Time
}

func (s SolicitudAbandonarEjecucionDocumentalV3) Validar() error {
	if !referenciaEjecucionDocumentalV3Valida(s.ReservaRef) || s.Manifiesto.Validar() != nil ||
		!referenciaEjecucionDocumentalV3Valida(s.MotivoRef) ||
		!instanteEjecucionDocumentalV3Valido(s.AbandonadaEn) {
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	switch s.EstadoEsperado {
	case EstadoEjecucionDocumentalV3Preparada:
		if !tokenCercadoDocumentalV3EsCero(s.Token) ||
			s.ConsumoDecision != (ConsumoDecisionEjecucionDocumentalV3{}) ||
			s.VerificacionCercado != (ResultadoVerificacionTokenCercadoDocumentalV3{}) {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Activa:
		if !verificacionTokenCercadoDocumentalV3Valida(
			s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.Token, s.VerificacionCercado,
		) {
			return ErrTransicionEjecucionDocumentalV3Invalida
		}
	default:
		return ErrTransicionEjecucionDocumentalV3Invalida
	}
	return nil
}

type SolicitudMarcarEjecucionDocumentalV3Indeterminada struct {
	ReservaRef          string
	Manifiesto          ManifiestoEjecucionDocumentalV3
	ConsumoDecision     ConsumoDecisionEjecucionDocumentalV3
	Token               TokenCercadoEjecucionDocumentalV3
	VerificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
	IncidenteRef        string
	MarcadaEn           time.Time
}

func (s SolicitudMarcarEjecucionDocumentalV3Indeterminada) Validar() error {
	if !verificacionTokenCercadoDocumentalV3Valida(
		s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.Token, s.VerificacionCercado,
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
		claveHMACDocumentalV3(c.IndiceIdempotenciaHMAC) != claveHMACDocumentalV3(c.HuellaSolicitudHMAC)
	if porReserva == porIndice {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	return nil
}

type InstantaneaEjecucionDocumentalV3 struct {
	ReservaRef                 string
	IndiceIdempotenciaHMAC     string
	HuellaSolicitudHMAC        string
	Manifiesto                 ManifiestoEjecucionDocumentalV3
	Estado                     EstadoEjecucionDocumentalV3
	SecuenciaCercado           uint64
	HuellaVinculoSHA256        string
	ConsumoDecision            ConsumoDecisionEjecucionDocumentalV3
	Resultado                  ResultadoEfectoRenderizadoDocumentalV3
	IncidenteRef               string
	EvidenciaRef               string
	HuellaEvidenciaSHA256      string
	EstadoOrigenAbandono       EstadoEjecucionDocumentalV3
	MotivoAbandonoRef          string
	ReconciliacionRef          string
	HuellaReconciliacionSHA256 string
	ActualizadaEn              time.Time
}

func (i InstantaneaEjecucionDocumentalV3) Validar() error {
	datos, err := i.Manifiesto.Datos()
	if err != nil || !referenciaEjecucionDocumentalV3Valida(i.ReservaRef) ||
		!hmacSHA256PuertoValido(i.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(i.HuellaSolicitudHMAC) ||
		!clavesHMACDocumentalesV3Distintas(
			i.IndiceIdempotenciaHMAC, i.HuellaSolicitudHMAC, datos.HuellaEntradaHMAC,
		) || !i.Estado.Valido() || !instanteEjecucionDocumentalV3Valido(i.ActualizadaEn) {
		return ErrReservaEjecucionDocumentalV3Invalida
	}
	switch i.Estado {
	case EstadoEjecucionDocumentalV3Preparada:
		if i.SecuenciaCercado != 0 || i.HuellaVinculoSHA256 != "" ||
			i.ConsumoDecision != (ConsumoDecisionEjecucionDocumentalV3{}) ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3{}) || i.IncidenteRef != "" ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionInstantaneaDocumentalV3Vacios(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Activa, EstadoEjecucionDocumentalV3EfectoIniciado:
		if !vinculoCercadoProyectadoDocumentalV3Valido(i) ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3{}) || i.IncidenteRef != "" ||
			i.EvidenciaRef != "" || i.HuellaEvidenciaSHA256 != "" ||
			!camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i) ||
			!camposReconciliacionInstantaneaDocumentalV3Vacios(i) {
			return ErrReservaEjecucionDocumentalV3Invalida
		}
	case EstadoEjecucionDocumentalV3Indeterminada:
		if !vinculoCercadoProyectadoDocumentalV3Valido(i) ||
			!referenciaEjecucionDocumentalV3Valida(i.IncidenteRef) ||
			i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3{}) ||
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
		if i.Resultado != (ResultadoEfectoRenderizadoDocumentalV3{}) ||
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
	Resultado                     ResultadoEfectoRenderizadoDocumentalV3
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
	if d.Esquema != EsquemaEvidenciaRenderizadoV3 || err != nil ||
		!referenciaEjecucionDocumentalV3Valida(d.ReservaRef) ||
		!hmacSHA256PuertoValido(d.IndiceIdempotenciaHMAC) ||
		!hmacSHA256PuertoValido(d.HuellaSolicitudHMAC) ||
		!clavesHMACDocumentalesV3Distintas(
			d.IndiceIdempotenciaHMAC, d.HuellaSolicitudHMAC, datosManifiesto.HuellaEntradaHMAC,
		) || d.SecuenciaCercado == 0 || !esSHA256Hexadecimal(d.HuellaVinculoSHA256) ||
		!referenciaEjecucionDocumentalV3Valida(d.ClaveAtestacionCercadoRef) ||
		!esSHA256Hexadecimal(d.HuellaMACCercadoSHA256) ||
		!referenciaEjecucionDocumentalV3Valida(d.EvidenciaAtestacionCercadoRef) ||
		!referenciaEjecucionDocumentalV3Valida(d.VerificacionCercadoRef) ||
		!instanteEjecucionDocumentalV3Valido(d.VerificadoCercadoEn) ||
		d.ConsumoDecision.ValidarContra(d.Manifiesto) != nil ||
		d.HuellaVinculoSHA256 != huellaVinculoCercadoEjecucionDocumentalV3(
			d.ReservaRef, d.SecuenciaCercado, d.Manifiesto, d.ConsumoDecision,
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
		perfil.ClaveID == claveHMACDocumentalV3(datos.IndiceIdempotenciaHMAC) ||
		perfil.ClaveID == claveHMACDocumentalV3(datos.HuellaSolicitudHMAC) ||
		perfil.ClaveID == datos.ClaveAtestacionCercadoRef {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	datosManifiesto, _ := datos.Manifiesto.Datos()
	if perfil.ClaveID == claveHMACDocumentalV3(datosManifiesto.HuellaEntradaHMAC) {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	mensaje := serializarEvidenciaRenderizadoDocumentalV3(datos)
	solicitud := SolicitudFirmaEvidenciaRenderizadoDocumentalV3{
		perfil: perfil, datos: datos, mensaje: append([]byte(nil), mensaje...),
		huella: huellaBytesFormatoDocumental(mensaje),
	}
	if solicitud.Validar() != nil {
		return SolicitudFirmaEvidenciaRenderizadoDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return solicitud, nil
}

func (s SolicitudFirmaEvidenciaRenderizadoDocumentalV3) Validar() error {
	if s.perfil.Validar() != nil || s.datos.Validar() != nil || len(s.mensaje) == 0 ||
		!esSHA256Hexadecimal(s.huella) ||
		huellaBytesFormatoDocumental(s.mensaje) != s.huella ||
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

type DatosSelloEvidenciaDocumentalV3 struct {
	Algoritmo             string
	ClaveID               string
	Audiencia             string
	HuellaMensajeSHA256   string
	Firma                 []byte
	EvidenciaOperacionRef string
	FirmadoEn             time.Time
}

type SelloEvidenciaDocumentalV3 struct {
	datos DatosSelloEvidenciaDocumentalV3
}

func NuevoSelloEvidenciaDocumentalV3(
	solicitud SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	firma []byte,
	evidenciaOperacionRef string,
	firmadoEn time.Time,
) (SelloEvidenciaDocumentalV3, error) {
	perfil, errPerfil := solicitud.Perfil()
	huella, errHuella := solicitud.HuellaMensajeSHA256()
	if errPerfil != nil || errHuella != nil {
		return SelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	sello := SelloEvidenciaDocumentalV3{datos: DatosSelloEvidenciaDocumentalV3{
		Algoritmo: perfil.Algoritmo, ClaveID: perfil.ClaveID, Audiencia: perfil.Audiencia,
		HuellaMensajeSHA256: huella, Firma: append([]byte(nil), firma...),
		EvidenciaOperacionRef: evidenciaOperacionRef, FirmadoEn: firmadoEn,
	}}
	if sello.ValidarPara(solicitud) != nil {
		return SelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return sello, nil
}

func (s SelloEvidenciaDocumentalV3) ValidarPara(
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

func (s SelloEvidenciaDocumentalV3) Datos() (DatosSelloEvidenciaDocumentalV3, error) {
	perfil := PerfilSelloEvidenciaDocumentalV3{
		Algoritmo: s.datos.Algoritmo, ClaveID: s.datos.ClaveID, Audiencia: s.datos.Audiencia,
	}
	if !datosSelloEvidenciaDocumentalV3ValidosPara(
		s.datos, perfil, s.datos.HuellaMensajeSHA256,
	) {
		return DatosSelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	datos := s.datos
	datos.Firma = append([]byte(nil), s.datos.Firma...)
	return datos, nil
}
func (SelloEvidenciaDocumentalV3) String() string     { return "[SELLO-EVIDENCIA-DOCUMENTAL-V3-OPACO]" }
func (s SelloEvidenciaDocumentalV3) GoString() string { return s.String() }
func (s SelloEvidenciaDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SelloEvidenciaDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SelloEvidenciaDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type SolicitudVerificacionEvidenciaDocumentalV3 struct {
	firma  SolicitudFirmaEvidenciaRenderizadoDocumentalV3
	sello  SelloEvidenciaDocumentalV3
	huella string
}

func NuevaSolicitudVerificacionEvidenciaDocumentalV3(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	sello SelloEvidenciaDocumentalV3,
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
// resultado con autoridad lo emite VerificadorEvidenciasRenderizadoDocumentalV3.
func NuevaSolicitudVerificacionEvidenciaDocumentalV3DesdeDatos(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	datos DatosSelloEvidenciaDocumentalV3,
) (SolicitudVerificacionEvidenciaDocumentalV3, error) {
	datos.Firma = append([]byte(nil), datos.Firma...)
	return NuevaSolicitudVerificacionEvidenciaDocumentalV3(
		firma, SelloEvidenciaDocumentalV3{datos: datos},
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
func (s SolicitudVerificacionEvidenciaDocumentalV3) Sello() (DatosSelloEvidenciaDocumentalV3, error) {
	if s.Validar() != nil {
		return DatosSelloEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return s.sello.Datos()
}

type ResultadoVerificacionEvidenciaDocumentalV3 struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevoResultadoVerificacionEvidenciaDocumentalV3 queda reservado al
// adaptador de VerificadorEvidenciasRenderizadoDocumentalV3. Construir un
// valor estructuralmente correcto no sustituye la comprobacion criptografica.
func NuevoResultadoVerificacionEvidenciaDocumentalV3(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (ResultadoVerificacionEvidenciaDocumentalV3, error) {
	if solicitud.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(verificadaEn) {
		return ResultadoVerificacionEvidenciaDocumentalV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	resultado := ResultadoVerificacionEvidenciaDocumentalV3{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}
	return resultado, nil
}

func (r ResultadoVerificacionEvidenciaDocumentalV3) ValidarPara(
	solicitud SolicitudVerificacionEvidenciaDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		!referenciaEjecucionDocumentalV3Valida(r.verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(r.verificadaEn) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

type FirmanteEvidenciasRenderizadoDocumentalV3 interface {
	FirmarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	) (SelloEvidenciaDocumentalV3, error)
}

// VerificadorEvidenciasRenderizadoDocumentalV3 es la unica frontera que
// convierte datos persistidos en una verificacion utilizable. No existe una
// funcion global de restauracion que acepte un mero SHA-256 autorreferente.
type VerificadorEvidenciasRenderizadoDocumentalV3 interface {
	VerificarEvidenciaRenderizadoDocumentalV3(
		context.Context,
		SolicitudVerificacionEvidenciaDocumentalV3,
	) (ResultadoVerificacionEvidenciaDocumentalV3, error)
}

type SolicitudConfirmarEjecucionDocumentalV3 struct {
	ReservaRef          string
	Manifiesto          ManifiestoEjecucionDocumentalV3
	ConsumoDecision     ConsumoDecisionEjecucionDocumentalV3
	Token               TokenCercadoEjecucionDocumentalV3
	VerificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
	Resultado           ResultadoEfectoRenderizadoDocumentalV3
	Recibos             RecibosEjecucionDocumentalV3
	Evidencia           DatosEvidenciaRenderizadoDocumentalV3
	Sello               SelloEvidenciaDocumentalV3
	Verificacion        ResultadoVerificacionEvidenciaDocumentalV3
}

func (s SolicitudConfirmarEjecucionDocumentalV3) Validar() error {
	if !verificacionTokenCercadoDocumentalV3Valida(
		s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.Token, s.VerificacionCercado,
	) ||
		s.Resultado.ValidarContra(s.Manifiesto) != nil || s.Evidencia.Validar() != nil ||
		s.Recibos.ValidarContra(
			s.Manifiesto, s.Resultado, s.ReservaRef, s.ConsumoDecision,
			s.Token, s.VerificacionCercado,
		) != nil ||
		s.Evidencia.ReservaRef != s.ReservaRef ||
		!manifiestosEjecucionDocumentalV3Coinciden(s.Evidencia.Manifiesto, s.Manifiesto) ||
		s.Evidencia.SecuenciaCercado != s.Token.Secuencia() ||
		s.Evidencia.HuellaVinculoSHA256 != s.Token.HuellaVinculoSHA256() ||
		s.Evidencia.ClaveAtestacionCercadoRef != s.Token.claveAtestacionRef ||
		s.Evidencia.HuellaMACCercadoSHA256 != huellaBytesFormatoDocumental(s.Token.macAtestacion) ||
		s.Evidencia.EvidenciaAtestacionCercadoRef != s.Token.evidenciaOperacionRef ||
		s.Evidencia.VerificacionCercadoRef != s.VerificacionCercado.verificacionRef ||
		!s.Evidencia.VerificadoCercadoEn.Equal(s.VerificacionCercado.verificadaEn) ||
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
	verificacion, err := NuevaSolicitudVerificacionEvidenciaDocumentalV3(firma, s.Sello)
	if err != nil || s.Verificacion.ValidarPara(verificacion) != nil {
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
	ReservaRef          string
	Manifiesto          ManifiestoEjecucionDocumentalV3
	ConsumoDecision     ConsumoDecisionEjecucionDocumentalV3
	Token               TokenCercadoEjecucionDocumentalV3
	VerificacionCercado ResultadoVerificacionTokenCercadoDocumentalV3
}

func (s SolicitudConsultarEfectoDocumentalV3) Validar() error {
	if !verificacionTokenCercadoDocumentalV3Valida(
		s.ReservaRef, s.Manifiesto, s.ConsumoDecision, s.Token, s.VerificacionCercado,
	) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

type SobreAtestacionReconciliacionDocumentalV3 struct {
	coseSign1 []byte
	huella    string
}

func NuevoSobreAtestacionReconciliacionDocumentalV3(
	coseSign1 []byte,
) (SobreAtestacionReconciliacionDocumentalV3, error) {
	sobre := SobreAtestacionReconciliacionDocumentalV3{
		coseSign1: append([]byte(nil), coseSign1...),
	}
	sobre.huella = huellaBytesFormatoDocumental(sobre.coseSign1)
	if sobre.Validar() != nil {
		return SobreAtestacionReconciliacionDocumentalV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	return sobre, nil
}

func (s SobreAtestacionReconciliacionDocumentalV3) Validar() error {
	if len(s.coseSign1) < minimoBytesSobreCOSEDocumental ||
		len(s.coseSign1) > maximoBytesSobreCOSEDocumental ||
		bytesEjecucionDocumentalNulos(s.coseSign1) || !esSHA256Hexadecimal(s.huella) ||
		huellaBytesFormatoDocumental(s.coseSign1) != s.huella {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

func (s SobreAtestacionReconciliacionDocumentalV3) COSESign1() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrReconciliacionDocumentalV3Invalida
	}
	return append([]byte(nil), s.coseSign1...), nil
}

func (s SobreAtestacionReconciliacionDocumentalV3) HuellaSHA256() (string, error) {
	if s.Validar() != nil {
		return "", ErrReconciliacionDocumentalV3Invalida
	}
	return s.huella, nil
}

func (SobreAtestacionReconciliacionDocumentalV3) String() string {
	return "[ATESTACION-RECONCILIACION-DOCUMENTAL-V3-OPACA]"
}
func (s SobreAtestacionReconciliacionDocumentalV3) GoString() string { return s.String() }
func (s SobreAtestacionReconciliacionDocumentalV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (SobreAtestacionReconciliacionDocumentalV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionDocumentalV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobreAtestacionReconciliacionDocumentalV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobreAtestacionReconciliacionDocumentalV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

type ResultadoConsultaEfectoDocumentalV3 struct {
	ReservaRef             string
	EfectoRef              string
	SecuenciaCercado       uint64
	HuellaVinculoSHA256    string
	HuellaPlanSHA256       string
	Estado                 EstadoResultadoReconciliacionDocumentalV3
	Resultado              ResultadoEfectoRenderizadoDocumentalV3
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	SobreAtestacion        SobreAtestacionReconciliacionDocumentalV3
	ConsultadaEn           time.Time
}

func (r ResultadoConsultaEfectoDocumentalV3) ValidarContra(
	s SolicitudConsultarEfectoDocumentalV3,
) error {
	datos, err := s.Manifiesto.Datos()
	if s.Validar() != nil || err != nil || r.ReservaRef != s.ReservaRef ||
		r.EfectoRef != datos.EfectoRef || r.SecuenciaCercado != s.Token.Secuencia() ||
		r.HuellaVinculoSHA256 != s.Token.HuellaVinculoSHA256() ||
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
	} else if r.Resultado != (ResultadoEfectoRenderizadoDocumentalV3{}) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	huellaSobre, _ := r.SobreAtestacion.HuellaSHA256()
	if huellaSobre != r.HuellaAtestacionSHA256 {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

type SolicitudVerificacionReconciliacionDocumentalV3 struct {
	consulta  SolicitudConsultarEfectoDocumentalV3
	resultado ResultadoConsultaEfectoDocumentalV3
	mensaje   []byte
	huella    string
}

func NuevaSolicitudVerificacionReconciliacionDocumentalV3(
	consulta SolicitudConsultarEfectoDocumentalV3,
	resultado ResultadoConsultaEfectoDocumentalV3,
) (SolicitudVerificacionReconciliacionDocumentalV3, error) {
	if resultado.ValidarContra(consulta) != nil {
		return SolicitudVerificacionReconciliacionDocumentalV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	mensaje := serializarResultadoReconciliacionDocumentalV3(resultado)
	solicitud := SolicitudVerificacionReconciliacionDocumentalV3{
		consulta: consulta, resultado: resultado,
		mensaje: append([]byte(nil), mensaje...),
		huella: huellaCanonicaFormatoDocumental([]string{
			"vec.documentos.solicitud-verificacion-reconciliacion.v3",
			huellaBytesFormatoDocumental(mensaje), resultado.HuellaAtestacionSHA256,
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
		huellaCanonicaFormatoDocumental([]string{
			"vec.documentos.solicitud-verificacion-reconciliacion.v3",
			huellaBytesFormatoDocumental(s.mensaje), s.resultado.HuellaAtestacionSHA256,
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

type ResultadoVerificacionReconciliacionDocumentalV3 struct {
	huellaSolicitud string
	verificacionRef string
	verificadaEn    time.Time
}

// NuevoResultadoVerificacionReconciliacionDocumentalV3 queda reservado al
// adaptador de VerificadorAtestacionesReconciliacionDocumentalV3. El producto
// no puede usar este constructor como sustituto de la verificacion COSE.
func NuevoResultadoVerificacionReconciliacionDocumentalV3(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
	verificacionRef string,
	verificadaEn time.Time,
) (ResultadoVerificacionReconciliacionDocumentalV3, error) {
	if solicitud.Validar() != nil || !referenciaEjecucionDocumentalV3Valida(verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(verificadaEn) {
		return ResultadoVerificacionReconciliacionDocumentalV3{}, ErrReconciliacionDocumentalV3Invalida
	}
	return ResultadoVerificacionReconciliacionDocumentalV3{
		huellaSolicitud: solicitud.huella, verificacionRef: verificacionRef, verificadaEn: verificadaEn,
	}, nil
}

func (r ResultadoVerificacionReconciliacionDocumentalV3) ValidarPara(
	solicitud SolicitudVerificacionReconciliacionDocumentalV3,
) error {
	if solicitud.Validar() != nil || r.huellaSolicitud != solicitud.huella ||
		!referenciaEjecucionDocumentalV3Valida(r.verificacionRef) ||
		!instanteEjecucionDocumentalV3Valido(r.verificadaEn) {
		return ErrReconciliacionDocumentalV3Invalida
	}
	return nil
}

type VerificadorAtestacionesReconciliacionDocumentalV3 interface {
	VerificarAtestacionReconciliacionDocumentalV3(
		context.Context,
		SolicitudVerificacionReconciliacionDocumentalV3,
	) (ResultadoVerificacionReconciliacionDocumentalV3, error)
}

type SolicitudAplicarReconciliacionDocumentalV3 struct {
	Consulta          SolicitudConsultarEfectoDocumentalV3
	ResultadoConsulta ResultadoConsultaEfectoDocumentalV3
	Verificacion      ResultadoVerificacionReconciliacionDocumentalV3
	TieneConfirmacion bool
	Confirmacion      SolicitudConfirmarEjecucionDocumentalV3
}

func (s SolicitudAplicarReconciliacionDocumentalV3) Validar() error {
	solicitudVerificacion, err := NuevaSolicitudVerificacionReconciliacionDocumentalV3(
		s.Consulta, s.ResultadoConsulta,
	)
	if err != nil || s.Verificacion.ValidarPara(solicitudVerificacion) != nil {
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
			) || s.Confirmacion.Evidencia.VerificacionReconciliacionRef !=
			s.Verificacion.verificacionRef ||
			!s.Confirmacion.Evidencia.ReconciliacionVerificadaEn.Equal(s.Verificacion.verificadaEn) {
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
	) (ResultadoConsultaEfectoDocumentalV3, error)
}

// RegistroEjecucionesDocumentalesV3 debe ser durable y transaccional. Las
// restricciones UNIQUE de indice HMAC, borrador, efecto y DecisionRef son
// permanentes. Confirmacion, manifiesto, evidencia, auditoria y outbox se
// confirman juntos; no se ofrece una implementacion en memoria productiva.
type RegistroEjecucionesDocumentalesV3 interface {
	PrepararEjecucionDocumentalV3(
		context.Context,
		SolicitudPrepararEjecucionDocumentalV3,
	) (PreparacionEjecucionDocumentalV3, error)
	ActivarEjecucionDocumentalV3(
		context.Context,
		SolicitudActivarEjecucionDocumentalV3,
	) (ActivacionEjecucionDocumentalV3, error)
	MarcarInicioEfectoDocumentalV3(context.Context, SolicitudIniciarEfectoDocumentalV3) error
	ConfirmarEjecucionDocumentalV3(context.Context, SolicitudConfirmarEjecucionDocumentalV3) error
	AbandonarEjecucionDocumentalV3(context.Context, SolicitudAbandonarEjecucionDocumentalV3) error
	MarcarEjecucionDocumentalV3Indeterminada(
		context.Context,
		SolicitudMarcarEjecucionDocumentalV3Indeterminada,
	) error
	ObtenerEjecucionDocumentalV3(
		context.Context,
		ConsultaEjecucionDocumentalV3,
	) (InstantaneaEjecucionDocumentalV3, error)
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
	recibo DatosReciboEjecucionComponenteDocumentalVerificado,
	operacion OperacionComponenteDocumental,
	componente DescriptorComponenteDocumentalAtestado,
	manifiesto DatosManifiestoEjecucionDocumentalV3,
	resultado ResultadoEfectoRenderizadoDocumentalV3,
	reservaRef string,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	solicitudCercado SolicitudVerificacionTokenCercadoDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
) bool {
	if !esSHA256Hexadecimal(recibo.HuellaCompromisoSHA256) || recibo.Operacion != operacion ||
		recibo.DescriptorPerfil != manifiesto.DescriptorPerfil ||
		recibo.SituacionOperativa != manifiesto.SituacionOperativa ||
		recibo.DescriptorComponente != componente || recibo.BorradorRef != manifiesto.BorradorRef ||
		recibo.ReservaRef != reservaRef || recibo.EfectoRef != manifiesto.EfectoRef ||
		recibo.HuellaPlanSHA256 != manifiesto.HuellaPlanSHA256 ||
		recibo.SecuenciaCercado != token.Secuencia() ||
		recibo.HuellaVinculoCercadoSHA256 != token.HuellaVinculoSHA256() ||
		recibo.DecisionRef != consumo.DecisionRef ||
		recibo.EsquemaHuellaDecision != consumo.EsquemaHuellaDecision ||
		recibo.HuellaDecisionSHA256 != consumo.HuellaDecisionSHA256 ||
		recibo.HuellaSolicitudCercadoSHA256 != solicitudCercado.huella ||
		recibo.VerificacionCercadoRef != verificacion.verificacionRef ||
		!recibo.VerificacionCercadoEn.Equal(verificacion.verificadaEn) ||
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
	return huellaCanonicaFormatoDocumental([]string{
		EsquemaManifiestoEjecucionDocumentalV3,
		d.Consulta.Identidad.Identificador(), d.Consulta.PerfilRef.Identificador(),
		strconv.FormatUint(d.Consulta.PerfilRef.Version(), 10), d.Consulta.DigestPerfilSHA256,
		strconv.FormatUint(d.Consulta.RevisionCatalogo.Numero(), 10),
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
	reservaRef string,
	secuencia uint64,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
) string {
	datos, err := manifiesto.Datos()
	if err != nil {
		return ""
	}
	return huellaCanonicaFormatoDocumental([]string{
		"vec.documentos.cercado-ejecucion.v3", reservaRef, strconvu64(secuencia),
		datos.HuellaPlanSHA256, datos.DescriptorPerfil.Perfil().DigestSHA256(),
		strconvu64(datos.SituacionOperativa.RevisionOperativa()),
		datos.SituacionOperativa.HuellaSHA256(),
		datos.ComponenteRender.DigestDeclaracionSHA256(),
		datos.ComponenteVerificador.DigestDeclaracionSHA256(),
		datos.ComponenteSemantico.DigestDeclaracionSHA256(),
		consumo.DecisionRef, consumo.EfectoRef, consumo.EsquemaHuellaDecision,
		consumo.HuellaDecisionSHA256, consumo.HuellaPlanSHA256,
	})
}

func vinculoCercadoProyectadoDocumentalV3Valido(i InstantaneaEjecucionDocumentalV3) bool {
	return i.SecuenciaCercado > 0 && esSHA256Hexadecimal(i.HuellaVinculoSHA256) &&
		i.ConsumoDecision.ValidarContra(i.Manifiesto) == nil &&
		i.HuellaVinculoSHA256 == huellaVinculoCercadoEjecucionDocumentalV3(
			i.ReservaRef, i.SecuenciaCercado, i.Manifiesto, i.ConsumoDecision,
		)
}

func camposOrigenAbandonoInstantaneaDocumentalV3Vacios(i InstantaneaEjecucionDocumentalV3) bool {
	return i.EstadoOrigenAbandono == "" && i.MotivoAbandonoRef == ""
}

func camposReconciliacionInstantaneaDocumentalV3Vacios(i InstantaneaEjecucionDocumentalV3) bool {
	return i.ReconciliacionRef == "" && i.HuellaReconciliacionSHA256 == ""
}

func camposReconciliacionConfirmadaDocumentalV3Validos(i InstantaneaEjecucionDocumentalV3) bool {
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
	var salida []byte
	for _, valor := range valores {
		salida = strconv.AppendInt(salida, int64(len(valor)), 10)
		salida = append(salida, ':')
		salida = append(salida, valor...)
		salida = append(salida, '\n')
	}
	return salida
}

func serializarAtestacionTokenCercadoDocumentalV3(
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
) []byte {
	huellaManifiesto, _ := manifiesto.HuellaSHA256()
	valores := []string{
		"vec.documentos.atestacion-token-cercado.v3", token.valor, reservaRef,
		strconvu64(token.secuencia), token.huellaVinculo, huellaManifiesto,
		consumo.DecisionRef, consumo.EfectoRef, consumo.EsquemaHuellaDecision,
		consumo.HuellaDecisionSHA256, consumo.HuellaPlanSHA256,
		AlgoritmoSelloEvidenciaHMACSHA256V3, AudienciaAtestacionTokenCercadoV3,
		token.claveAtestacionRef, token.evidenciaOperacionRef,
	}
	var salida []byte
	for _, valor := range valores {
		salida = strconv.AppendInt(salida, int64(len(valor)), 10)
		salida = append(salida, ':')
		salida = append(salida, valor...)
		salida = append(salida, '\n')
	}
	return salida
}

func serializarResultadoReconciliacionDocumentalV3(
	r ResultadoConsultaEfectoDocumentalV3,
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
	var salida []byte
	for _, valor := range valores {
		salida = strconv.AppendInt(salida, int64(len(valor)), 10)
		salida = append(salida, ':')
		salida = append(salida, valor...)
		salida = append(salida, '\n')
	}
	return salida
}

func huellaSolicitudVerificacionTokenCercadoDocumentalV3(
	mensaje, mac []byte,
) string {
	return huellaCanonicaFormatoDocumental([]string{
		"vec.documentos.solicitud-verificacion-token-cercado.v3",
		huellaBytesFormatoDocumental(mensaje), huellaBytesFormatoDocumental(mac),
	})
}

func huellaSolicitudVerificacionEvidenciaDocumentalV3(
	firma SolicitudFirmaEvidenciaRenderizadoDocumentalV3,
	sello SelloEvidenciaDocumentalV3,
) string {
	huellaMensaje, _ := firma.HuellaMensajeSHA256()
	datos, _ := sello.Datos()
	return huellaCanonicaFormatoDocumental([]string{
		"vec.documentos.solicitud-verificacion-evidencia.v3", huellaMensaje,
		datos.Algoritmo, datos.ClaveID, datos.Audiencia,
		huellaBytesFormatoDocumental(datos.Firma), datos.EvidenciaOperacionRef,
		datos.FirmadoEn.Format(time.RFC3339Nano),
	})
}

func verificacionTokenCercadoDocumentalV3Valida(
	reservaRef string,
	manifiesto ManifiestoEjecucionDocumentalV3,
	consumo ConsumoDecisionEjecucionDocumentalV3,
	token TokenCercadoEjecucionDocumentalV3,
	verificacion ResultadoVerificacionTokenCercadoDocumentalV3,
) bool {
	solicitud, err := NuevaSolicitudVerificacionTokenCercadoDocumentalV3(
		reservaRef, manifiesto, consumo, token,
	)
	return err == nil && verificacion.ValidarPara(solicitud) == nil
}

func tokenCercadoDocumentalV3EsCero(token TokenCercadoEjecucionDocumentalV3) bool {
	return token.valor == "" && token.secuencia == 0 && token.huellaVinculo == "" &&
		token.claveAtestacionRef == "" && len(token.macAtestacion) == 0 &&
		token.evidenciaOperacionRef == ""
}

func referenciaEjecucionDocumentalV3Valida(valor string) bool {
	if !referenciaDescriptorDocumentalValida.MatchString(valor) || strings.ContainsRune(valor, '*') ||
		strings.Contains(valor, "://") || strings.ContainsAny(valor, "/\\@?#%=") {
		return false
	}
	minusculas := strings.ToLower(valor)
	for _, prefijo := range []string{"dni:", "nif:", "nie:", "email:", "mailto:"} {
		if strings.HasPrefix(minusculas, prefijo) {
			return false
		}
	}
	return true
}

func instanteEjecucionDocumentalV3Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

func claveHMACDocumentalV3(valor string) string {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" {
		return ""
	}
	return partes[1]
}

func clavesHMACDocumentalesV3Distintas(valores ...string) bool {
	claves := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		clave := claveHMACDocumentalV3(valor)
		if clave == "" {
			return false
		}
		if _, existe := claves[clave]; existe {
			return false
		}
		claves[clave] = struct{}{}
	}
	return true
}

func solicitudConfirmacionDocumentalV3EsCero(
	s SolicitudConfirmarEjecucionDocumentalV3,
) bool {
	return s.ReservaRef == "" && s.Manifiesto.datos == nil &&
		s.ConsumoDecision == (ConsumoDecisionEjecucionDocumentalV3{}) &&
		tokenCercadoDocumentalV3EsCero(s.Token) &&
		s.VerificacionCercado == (ResultadoVerificacionTokenCercadoDocumentalV3{}) &&
		s.Resultado == (ResultadoEfectoRenderizadoDocumentalV3{}) &&
		reflect.ValueOf(s.Recibos).IsZero() &&
		s.Evidencia == (DatosEvidenciaRenderizadoDocumentalV3{}) &&
		s.Sello.datos.Algoritmo == "" && s.Sello.datos.ClaveID == "" &&
		s.Sello.datos.Audiencia == "" && s.Sello.datos.HuellaMensajeSHA256 == "" &&
		len(s.Sello.datos.Firma) == 0 && s.Sello.datos.EvidenciaOperacionRef == "" &&
		s.Sello.datos.FirmadoEn.IsZero() &&
		s.Verificacion == (ResultadoVerificacionEvidenciaDocumentalV3{})
}

func datosSelloEvidenciaDocumentalV3ValidosPara(
	datos DatosSelloEvidenciaDocumentalV3,
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

func strconvu64(valor uint64) string { return strconv.FormatUint(valor, 10) }
