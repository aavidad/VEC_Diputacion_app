package gobiernoconvocatorias

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	esquemaAADCifradoBorradorV1      = "bolsa.convocatoria.borrador.aad.v1"
	esquemaEnvolturaClaveBorradorV1  = "bolsa.convocatoria.borrador.clave-envuelta.v1"
	esquemaSobreCifradoBorradorV1    = "bolsa.convocatoria.borrador.sobre-aead.v1"
	esquemaAtestacionKMSBorradorV1   = "bolsa.convocatoria.borrador.atestacion-kms.v1"
	estadoAtestacionKMSVigente       = "vigente"
	estadoRevalidacionKMSAutorizada  = "autorizada"
	maximoBytesCifradoBorrador       = 16 << 20
	maximoBytesClaveEnvueltaBorrador = 64 << 10
)

var (
	ErrCifradoBorradorInvalido      = errors.New("gobierno convocatorias: cifrado de borrador invalido")
	ErrRevalidacionKMSBorradorFallo = errors.New("gobierno convocatorias: revalidacion KMS de borrador fallida")
)

// AADCanonicaCifradoBorrador liga el sobre a la version exacta, material,
// primaria, revision, cercado, arrendamiento, sellado y correlacion. Solo
// contiene referencias tecnicas, HMAC o SHA-256; nunca principal, perfil de
// usuario/actor, motivo ni contenido del borrador en claro.
type AADCanonicaCifradoBorrador struct {
	bloqueoSerializacionDiario
	representacion []byte
	huellaSHA256   string
}

func (a AADCanonicaCifradoBorrador) RepresentacionCanonica() ([]byte, error) {
	if !a.valida() {
		return nil, ErrCifradoBorradorInvalido
	}
	return append([]byte(nil), a.representacion...), nil
}

func (a AADCanonicaCifradoBorrador) HuellaSHA256() (string, error) {
	if !a.valida() {
		return "", ErrCifradoBorradorInvalido
	}
	return a.huellaSHA256, nil
}

func (a AADCanonicaCifradoBorrador) valida() bool {
	if len(a.representacion) == 0 || len(a.representacion) > 32<<10 ||
		!huellaHexValida(a.huellaSHA256) {
		return false
	}
	suma := sha256.Sum256(a.representacion)
	return coincideTextoConstante(hex.EncodeToString(suma[:]), a.huellaSHA256)
}

func aadCanonicaCifradoBorrador(
	version dominiobolsa.VersionConvocatoriaGobernada,
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
	reserva ProyeccionReservaDecision,
	control ResultadoOperacionDiario,
	sellado ProyeccionSelladoMotivoBorrador,
	perfil PerfilCifradoBorrador,
	correlacionRef string,
) (AADCanonicaCifradoBorrador, error) {
	estado, errEstado := puertosbolsa.EstadoVersionConvocatoria(version)
	huellaMaterial, errMaterial := material.HuellaSHA256()
	if errEstado != nil || errMaterial != nil || estado != material.EstadoPrincipalNuevo ||
		!reserva.valida() || !resultadoDiarioValido(control) ||
		control.Estado != ResultadoDiarioReservado || control.Revision == 0 || control.Cercado == 0 ||
		!control.ArrendamientoIniciaEn.Equal(reserva.ArrendamientoIniciaEn) ||
		!control.ArrendamientoVenceEn.Equal(reserva.ArrendamientoVenceEn) ||
		!sellado.validaEstructural() || !perfil.valida() ||
		!referenciaProyeccionValida(correlacionRef) {
		return AADCanonicaCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	huellaCorrelacion := sha256.Sum256([]byte(
		"bolsa.convocatoria.borrador.correlacion.v1\x00" + correlacionRef,
	))
	identidad := reserva.IdentidadPrimaria
	materialAAD := struct {
		Esquema                   string `json:"esquema"`
		VersionRef                string `json:"version_ref"`
		VersionRevision           int    `json:"version_revision"`
		HuellaVersionSHA256       string `json:"huella_version_sha256"`
		EsquemaMaterial           string `json:"esquema_material"`
		HuellaMaterialSHA256      string `json:"huella_material_sha256"`
		PerfilCifradoRef          string `json:"perfil_cifrado_ref"`
		PerfilCifradoVersion      uint32 `json:"perfil_cifrado_version"`
		HuellaPerfilSHA256        string `json:"huella_perfil_cifrado_sha256"`
		AlgoritmoAEAD             string `json:"algoritmo_aead"`
		AlgoritmoEnvolturaClave   string `json:"algoritmo_envoltura_clave"`
		LocalizadorEsquema        uint16 `json:"localizador_esquema"`
		LocalizadorDominio        string `json:"localizador_dominio"`
		LocalizadorClaveRef       string `json:"localizador_clave_ref"`
		LocalizadorGeneracion     uint32 `json:"localizador_generacion"`
		LocalizadorHMACSHA256     string `json:"localizador_hmac_sha256"`
		HuellaSolicitudEsquema    uint16 `json:"huella_solicitud_esquema"`
		HuellaSolicitudDominio    string `json:"huella_solicitud_dominio"`
		HuellaSolicitudClaveRef   string `json:"huella_solicitud_clave_ref"`
		HuellaSolicitudGeneracion uint32 `json:"huella_solicitud_generacion"`
		HuellaSolicitudHMACSHA256 string `json:"huella_solicitud_hmac_sha256"`
		RevisionDiario            uint64 `json:"revision_diario"`
		CercadoDiario             uint64 `json:"cercado_diario"`
		ArrendamientoIniciaEn     string `json:"arrendamiento_inicia_en"`
		ArrendamientoVenceEn      string `json:"arrendamiento_vence_en"`
		AtestacionSelladoRef      string `json:"atestacion_sellado_ref"`
		AtestacionSelladoVersion  uint32 `json:"atestacion_sellado_version"`
		HuellaAtestacionSellado   string `json:"huella_atestacion_sellado_sha256"`
		TokenConsumoSelladoRef    string `json:"token_consumo_sellado_ref"`
		HuellaCorrelacionSHA256   string `json:"huella_correlacion_sha256"`
	}{
		Esquema:    esquemaAADCifradoBorradorV1,
		VersionRef: estado.Referencia, VersionRevision: estado.Revision,
		HuellaVersionSHA256: estado.HuellaEstadoSHA256,
		EsquemaMaterial:     material.Esquema, HuellaMaterialSHA256: huellaMaterial,
		PerfilCifradoRef: perfil.Referencia, PerfilCifradoVersion: perfil.Version,
		HuellaPerfilSHA256:        perfil.HuellaContenidoSHA256,
		AlgoritmoAEAD:             perfil.AlgoritmoAEAD,
		AlgoritmoEnvolturaClave:   perfil.AlgoritmoEnvolturaClave,
		LocalizadorEsquema:        identidad.Localizador.VersionEsquema,
		LocalizadorDominio:        identidad.Localizador.Dominio,
		LocalizadorClaveRef:       identidad.Localizador.ClaveRef,
		LocalizadorGeneracion:     identidad.Localizador.GeneracionClave,
		LocalizadorHMACSHA256:     identidad.Localizador.ValorHMACSHA256,
		HuellaSolicitudEsquema:    identidad.HuellaSolicitud.VersionEsquema,
		HuellaSolicitudDominio:    identidad.HuellaSolicitud.Dominio,
		HuellaSolicitudClaveRef:   identidad.HuellaSolicitud.ClaveRef,
		HuellaSolicitudGeneracion: identidad.HuellaSolicitud.GeneracionClave,
		HuellaSolicitudHMACSHA256: identidad.HuellaSolicitud.ValorHMACSHA256,
		RevisionDiario:            control.Revision, CercadoDiario: control.Cercado,
		ArrendamientoIniciaEn:    control.ArrendamientoIniciaEn.Format(time.RFC3339Nano),
		ArrendamientoVenceEn:     control.ArrendamientoVenceEn.Format(time.RFC3339Nano),
		AtestacionSelladoRef:     sellado.AtestacionRef,
		AtestacionSelladoVersion: sellado.VersionAtestacion,
		HuellaAtestacionSellado:  sellado.HuellaAtestacionSHA256,
		TokenConsumoSelladoRef:   sellado.TokenConsumoRef,
		HuellaCorrelacionSHA256:  hex.EncodeToString(huellaCorrelacion[:]),
	}
	representacion, err := json.Marshal(materialAAD)
	if err != nil {
		return AADCanonicaCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	suma := sha256.Sum256(representacion)
	resultado := AADCanonicaCifradoBorrador{
		representacion: append([]byte(nil), representacion...),
		huellaSHA256:   hex.EncodeToString(suma[:]),
	}
	if !resultado.valida() {
		return AADCanonicaCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	return resultado, nil
}

// PerfilCifradoBorrador identifica una configuracion gobernada. El nucleo no
// fija un algoritmo ni proveedor: el KMS debe atestar la version exacta.
type PerfilCifradoBorrador struct {
	bloqueoSerializacionDiario
	Referencia              string
	Version                 uint32
	HuellaContenidoSHA256   string
	AlgoritmoAEAD           string
	AlgoritmoEnvolturaClave string
}

func NuevoPerfilCifradoBorrador(
	referencia string,
	version uint32,
	huellaContenidoSHA256, algoritmoAEAD, algoritmoEnvolturaClave string,
) (PerfilCifradoBorrador, error) {
	p := PerfilCifradoBorrador{
		Referencia: referencia, Version: version, HuellaContenidoSHA256: huellaContenidoSHA256,
		AlgoritmoAEAD: algoritmoAEAD, AlgoritmoEnvolturaClave: algoritmoEnvolturaClave,
	}
	if !p.valida() {
		return PerfilCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	return p, nil
}

func (p PerfilCifradoBorrador) valida() bool {
	return referenciaProyeccionValida(p.Referencia) && p.Version > 0 &&
		huellaHexValida(p.HuellaContenidoSHA256) && referenciaProyeccionValida(p.AlgoritmoAEAD) &&
		referenciaProyeccionValida(p.AlgoritmoEnvolturaClave) &&
		p.AlgoritmoAEAD != p.AlgoritmoEnvolturaClave
}

// SolicitudResolucionPerfilCifradoBorrador permite elegir una politica
// aprobada por accion y clasificacion sin exponer el agregado. El resolvedor
// debe leer un catalogo inmutable y versionado; el cifrador no elige el perfil.
type SolicitudResolucionPerfilCifradoBorrador struct {
	bloqueoSerializacionDiario
	Reserva       ProyeccionReservaDecision
	Control       ResultadoOperacionDiario
	Material      puertosbolsa.MaterialIntencionGobiernoConvocatoria
	SelladoMotivo ProyeccionSelladoMotivoBorrador
	SolicitadaEn  time.Time
}

func (s SolicitudResolucionPerfilCifradoBorrador) Validar() error {
	if !s.Reserva.valida() || s.Material.Validar() != nil ||
		s.Reserva.Accion != s.Material.Accion || s.Control.Estado != ResultadoDiarioReservado ||
		s.Control.Revision == 0 || s.Control.Cercado == 0 ||
		!s.Control.ArrendamientoIniciaEn.Equal(s.Reserva.ArrendamientoIniciaEn) ||
		!s.Control.ArrendamientoVenceEn.Equal(s.Reserva.ArrendamientoVenceEn) ||
		!instanteOperacionCanonico(s.SolicitadaEn) ||
		!s.SelladoMotivo.validaPara(s.Material, s.SolicitadaEn) ||
		s.SolicitadaEn.Before(s.Reserva.ArrendamientoIniciaEn) ||
		!s.SolicitadaEn.Before(s.Reserva.ArrendamientoVenceEn) {
		return ErrCifradoBorradorInvalido
	}
	return nil
}

type ResolvedorPerfilCifradoBorrador interface {
	ResolverPerfilCifradoBorrador(
		context.Context,
		SolicitudResolucionPerfilCifradoBorrador,
	) (PerfilCifradoBorrador, error)
}

// EnvolturaClaveKMSBorrador y SobreCifradoAEADBorrador mantienen los buffers
// privados, copian en entrada/salida y heredan el cierre de codecs y formato.
// Solo DatosParaPersistencia permite a un adaptador autorizado extraer copias.
type EnvolturaClaveKMSBorrador struct {
	bloqueoSerializacionDiario
	esquema          string
	perfil           PerfilCifradoBorrador
	claveMaestraRef  string
	versionClave     uint32
	materialEnvuelto []byte
	huellaAAD        string
	huellaSHA256     string
}

func NuevaEnvolturaClaveKMSBorrador(
	perfil PerfilCifradoBorrador,
	claveMaestraRef string,
	versionClave uint32,
	materialEnvuelto []byte,
	huellaAAD string,
) (EnvolturaClaveKMSBorrador, error) {
	e := EnvolturaClaveKMSBorrador{
		esquema: esquemaEnvolturaClaveBorradorV1, perfil: perfil,
		claveMaestraRef: claveMaestraRef, versionClave: versionClave,
		materialEnvuelto: append([]byte(nil), materialEnvuelto...), huellaAAD: huellaAAD,
	}
	e.huellaSHA256 = e.calcularHuella()
	if !e.valida() {
		return EnvolturaClaveKMSBorrador{}, ErrCifradoBorradorInvalido
	}
	return e, nil
}

func (e EnvolturaClaveKMSBorrador) valida() bool {
	if e.esquema != esquemaEnvolturaClaveBorradorV1 || !e.perfil.valida() ||
		!referenciaProyeccionValida(e.claveMaestraRef) || e.versionClave == 0 ||
		len(e.materialEnvuelto) < 16 || len(e.materialEnvuelto) > maximoBytesClaveEnvueltaBorrador ||
		!huellaHexValida(e.huellaAAD) || !huellaHexValida(e.huellaSHA256) {
		return false
	}
	return coincideTextoConstante(e.calcularHuella(), e.huellaSHA256)
}

func (e EnvolturaClaveKMSBorrador) calcularHuella() string {
	material := struct {
		Esquema, PerfilRef, PerfilHuella, Algoritmo, ClaveRef, HuellaAAD string
		PerfilVersion, VersionClave                                      uint32
		Material                                                         []byte
	}{e.esquema, e.perfil.Referencia, e.perfil.HuellaContenidoSHA256,
		e.perfil.AlgoritmoEnvolturaClave, e.claveMaestraRef, e.huellaAAD,
		e.perfil.Version, e.versionClave, e.materialEnvuelto}
	bytes, err := json.Marshal(material)
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:])
}

func (e EnvolturaClaveKMSBorrador) DatosParaPersistencia() (
	perfil PerfilCifradoBorrador,
	claveMaestraRef string,
	versionClave uint32,
	materialEnvuelto []byte,
	huellaAAD, huellaSHA256 string,
	err error,
) {
	if !e.valida() {
		return PerfilCifradoBorrador{}, "", 0, nil, "", "", ErrCifradoBorradorInvalido
	}
	return e.perfil, e.claveMaestraRef, e.versionClave,
		append([]byte(nil), e.materialEnvuelto...), e.huellaAAD, e.huellaSHA256, nil
}

type SobreCifradoAEADBorrador struct {
	bloqueoSerializacionDiario
	esquema      string
	perfil       PerfilCifradoBorrador
	nonce        []byte
	textoCifrado []byte
	huellaAAD    string
	huellaSHA256 string
}

func NuevoSobreCifradoAEADBorrador(
	perfil PerfilCifradoBorrador,
	nonce, textoCifrado []byte,
	huellaAAD string,
) (SobreCifradoAEADBorrador, error) {
	s := SobreCifradoAEADBorrador{
		esquema: esquemaSobreCifradoBorradorV1, perfil: perfil,
		nonce: append([]byte(nil), nonce...), textoCifrado: append([]byte(nil), textoCifrado...),
		huellaAAD: huellaAAD,
	}
	s.huellaSHA256 = s.calcularHuella()
	if !s.valida() {
		return SobreCifradoAEADBorrador{}, ErrCifradoBorradorInvalido
	}
	return s, nil
}

func (s SobreCifradoAEADBorrador) valida() bool {
	if s.esquema != esquemaSobreCifradoBorradorV1 || !s.perfil.valida() ||
		len(s.nonce) < 8 || len(s.nonce) > 64 || len(s.textoCifrado) < 16 ||
		len(s.textoCifrado) > maximoBytesCifradoBorrador ||
		!huellaHexValida(s.huellaAAD) || !huellaHexValida(s.huellaSHA256) {
		return false
	}
	return coincideTextoConstante(s.calcularHuella(), s.huellaSHA256)
}

func (s SobreCifradoAEADBorrador) calcularHuella() string {
	material := struct {
		Esquema, PerfilRef, PerfilHuella, Algoritmo, HuellaAAD string
		PerfilVersion                                          uint32
		Nonce, TextoCifrado                                    []byte
	}{s.esquema, s.perfil.Referencia, s.perfil.HuellaContenidoSHA256,
		s.perfil.AlgoritmoAEAD, s.huellaAAD, s.perfil.Version, s.nonce, s.textoCifrado}
	bytes, err := json.Marshal(material)
	if err != nil {
		return ""
	}
	suma := sha256.Sum256(bytes)
	return hex.EncodeToString(suma[:])
}

func (s SobreCifradoAEADBorrador) DatosParaPersistencia() (
	perfil PerfilCifradoBorrador,
	nonce, textoCifrado []byte,
	huellaAAD, huellaSHA256 string,
	err error,
) {
	if !s.valida() {
		return PerfilCifradoBorrador{}, nil, nil, "", "", ErrCifradoBorradorInvalido
	}
	return s.perfil, append([]byte(nil), s.nonce...), append([]byte(nil), s.textoCifrado...),
		s.huellaAAD, s.huellaSHA256, nil
}

type AtestacionKMSBorrador struct {
	bloqueoSerializacionDiario
	Esquema               string
	AtestacionRef         string
	VersionAtestacion     uint32
	Estado                string
	Perfil                PerfilCifradoBorrador
	ClaveMaestraRef       string
	VersionClave          uint32
	HuellaAAD             string
	HuellaEnvolturaSHA256 string
	HuellaSobreSHA256     string
	VerificadorRef        string
	EmitidaEn             time.Time
	ValidaHasta           time.Time
}

func NuevaAtestacionKMSBorrador(
	atestacionRef string,
	versionAtestacion uint32,
	perfil PerfilCifradoBorrador,
	claveMaestraRef string,
	versionClave uint32,
	huellaAAD, huellaEnvoltura, huellaSobre, verificadorRef string,
	emitidaEn, validaHasta time.Time,
) (AtestacionKMSBorrador, error) {
	a := AtestacionKMSBorrador{
		Esquema: esquemaAtestacionKMSBorradorV1, AtestacionRef: atestacionRef,
		VersionAtestacion: versionAtestacion, Estado: estadoAtestacionKMSVigente,
		Perfil: perfil, ClaveMaestraRef: claveMaestraRef, VersionClave: versionClave,
		HuellaAAD: huellaAAD, HuellaEnvolturaSHA256: huellaEnvoltura,
		HuellaSobreSHA256: huellaSobre, VerificadorRef: verificadorRef,
		EmitidaEn: emitidaEn, ValidaHasta: validaHasta,
	}
	if !a.validaEstructural() {
		return AtestacionKMSBorrador{}, ErrCifradoBorradorInvalido
	}
	return a, nil
}

func (a AtestacionKMSBorrador) validaEstructural() bool {
	return a.Esquema == esquemaAtestacionKMSBorradorV1 &&
		referenciaProyeccionValida(a.AtestacionRef) && a.VersionAtestacion > 0 &&
		a.Estado == estadoAtestacionKMSVigente && a.Perfil.valida() &&
		referenciaProyeccionValida(a.ClaveMaestraRef) && a.VersionClave > 0 &&
		huellaHexValida(a.HuellaAAD) && huellaHexValida(a.HuellaEnvolturaSHA256) &&
		huellaHexValida(a.HuellaSobreSHA256) && referenciaProyeccionValida(a.VerificadorRef) &&
		a.AtestacionRef != a.VerificadorRef && instanteOperacionCanonico(a.EmitidaEn) &&
		instanteOperacionCanonico(a.ValidaHasta) && a.ValidaHasta.After(a.EmitidaEn) &&
		a.ValidaHasta.Sub(a.EmitidaEn) <= 10*time.Minute
}

type ResultadoCifradoBorrador struct {
	bloqueoSerializacionDiario
	AAD            AADCanonicaCifradoBorrador
	EnvolturaClave EnvolturaClaveKMSBorrador
	SobreCifrado   SobreCifradoAEADBorrador
	AtestacionKMS  AtestacionKMSBorrador
	SolicitadaEn   time.Time
	CifradoEn      time.Time
}

func (r ResultadoCifradoBorrador) validaPara(s SolicitudCifradoBorrador) bool {
	huellaAAD, err := s.aad.HuellaSHA256()
	huellaSolicitud, errSolicitud := r.AAD.HuellaSHA256()
	_, claveRef, versionClave, _, huellaEnvolturaAAD, huellaEnvoltura, errEnvoltura :=
		r.EnvolturaClave.DatosParaPersistencia()
	perfilSobre, _, _, huellaSobreAAD, huellaSobre, errSobre := r.SobreCifrado.DatosParaPersistencia()
	perfilEnvoltura := r.EnvolturaClave.perfil
	return err == nil && errSolicitud == nil && errEnvoltura == nil && errSobre == nil &&
		aadCoincide(r.AAD, s.aad) && coincideTextoConstante(huellaAAD, huellaSolicitud) &&
		coincideTextoConstante(huellaAAD, huellaEnvolturaAAD) &&
		coincideTextoConstante(huellaAAD, huellaSobreAAD) &&
		perfilesCifradoCoinciden(perfilEnvoltura, perfilSobre) &&
		perfilesCifradoCoinciden(perfilSobre, s.PerfilEsperado) &&
		r.AtestacionKMS.validaEstructural() &&
		perfilesCifradoCoinciden(r.AtestacionKMS.Perfil, perfilSobre) &&
		r.AtestacionKMS.ClaveMaestraRef == claveRef && r.AtestacionKMS.VersionClave == versionClave &&
		coincideTextoConstante(r.AtestacionKMS.HuellaAAD, huellaAAD) &&
		coincideTextoConstante(r.AtestacionKMS.HuellaEnvolturaSHA256, huellaEnvoltura) &&
		coincideTextoConstante(r.AtestacionKMS.HuellaSobreSHA256, huellaSobre) &&
		instanteOperacionCanonico(r.SolicitadaEn) && r.SolicitadaEn.Equal(s.SolicitadaEn) &&
		instanteOperacionCanonico(r.CifradoEn) && !r.CifradoEn.Before(s.SolicitadaEn) &&
		!r.CifradoEn.Before(r.AtestacionKMS.EmitidaEn) && r.CifradoEn.Before(r.AtestacionKMS.ValidaHasta) &&
		r.CifradoEn.Before(s.Reserva.ArrendamientoVenceEn)
}

func perfilesCifradoCoinciden(a, b PerfilCifradoBorrador) bool {
	return a.valida() && b.valida() && a.Referencia == b.Referencia && a.Version == b.Version &&
		coincideTextoConstante(a.HuellaContenidoSHA256, b.HuellaContenidoSHA256) &&
		a.AlgoritmoAEAD == b.AlgoritmoAEAD && a.AlgoritmoEnvolturaClave == b.AlgoritmoEnvolturaClave
}

func aadCoincide(a, b AADCanonicaCifradoBorrador) bool {
	if !a.valida() || !b.valida() || len(a.representacion) != len(b.representacion) {
		return false
	}
	return subtle.ConstantTimeCompare(a.representacion, b.representacion) == 1
}

// SolicitudCifradoBorrador conserva el agregado solo en memoria y bloquea los
// codecs. El conector recibe una copia canonica deliberada y debe borrar ese
// buffer tras cifrarlo; el servicio nunca maneja una copia []byte en claro.
type SolicitudCifradoBorrador struct {
	bloqueoSerializacionDiario
	version        dominiobolsa.VersionConvocatoriaGobernada
	Reserva        ProyeccionReservaDecision
	Control        ResultadoOperacionDiario
	Material       puertosbolsa.MaterialIntencionGobiernoConvocatoria
	SelladoMotivo  ProyeccionSelladoMotivoBorrador
	PerfilEsperado PerfilCifradoBorrador
	CorrelacionRef string
	aad            AADCanonicaCifradoBorrador
	SolicitadaEn   time.Time
}

func nuevaSolicitudCifradoBorrador(
	version dominiobolsa.VersionConvocatoriaGobernada,
	reserva ProyeccionReservaDecision,
	control ResultadoOperacionDiario,
	material puertosbolsa.MaterialIntencionGobiernoConvocatoria,
	sellado ProyeccionSelladoMotivoBorrador,
	perfilEsperado PerfilCifradoBorrador,
	correlacionRef string,
	solicitadaEn time.Time,
) (SolicitudCifradoBorrador, error) {
	clon, err := version.ClonarCanonico()
	if err != nil {
		return SolicitudCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	aad, err := aadCanonicaCifradoBorrador(
		clon, material, reserva, control, sellado, perfilEsperado, correlacionRef,
	)
	if err != nil {
		return SolicitudCifradoBorrador{}, err
	}
	s := SolicitudCifradoBorrador{
		version: clon, Reserva: reserva, Control: control, Material: material,
		SelladoMotivo: sellado, PerfilEsperado: perfilEsperado,
		CorrelacionRef: correlacionRef, aad: aad,
		SolicitadaEn: solicitadaEn,
	}
	if s.Validar() != nil {
		return SolicitudCifradoBorrador{}, ErrCifradoBorradorInvalido
	}
	return s, nil
}

func (s SolicitudCifradoBorrador) VersionCanonicaParaCifrado() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrCifradoBorradorInvalido
	}
	return s.version.RepresentacionCanonica()
}

func (s SolicitudCifradoBorrador) AADCanonica() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrCifradoBorradorInvalido
	}
	return s.aad.RepresentacionCanonica()
}

func (s SolicitudCifradoBorrador) Validar() error {
	estado, errEstado := puertosbolsa.EstadoVersionConvocatoria(s.version)
	aad, errAAD := aadCanonicaCifradoBorrador(
		s.version, s.Material, s.Reserva, s.Control, s.SelladoMotivo,
		s.PerfilEsperado, s.CorrelacionRef,
	)
	if errEstado != nil || errAAD != nil || estado != s.Material.EstadoPrincipalNuevo ||
		!instanteOperacionCanonico(s.SolicitadaEn) ||
		s.SolicitadaEn.Before(s.SelladoMotivo.AtestacionEmitidaEn) ||
		!s.SolicitadaEn.Before(s.SelladoMotivo.AtestacionValidaHasta) ||
		s.SolicitadaEn.Before(s.Reserva.ArrendamientoIniciaEn) ||
		!s.SolicitadaEn.Before(s.Reserva.ArrendamientoVenceEn) || !aadCoincide(aad, s.aad) {
		return ErrCifradoBorradorInvalido
	}
	return nil
}

// CifradorAEADKMSBorrador es el puerto intercambiable. La implementacion debe
// generar una DEK nueva, cifrar con AEAD, envolver la DEK mediante KMS/HSM y
// devolver una atestacion autoritativa; no se admite texto cifrado sintetico.
type CifradorAEADKMSBorrador interface {
	CifrarBorrador(context.Context, SolicitudCifradoBorrador) (ResultadoCifradoBorrador, error)
}

// SolicitudRevalidacionAtestacionKMSBorrador se crea en persistencia despues
// de bloquear diario/agregado y antes del COMMIT. Una proyeccion copiada por el
// servicio nunca sustituye esta consulta autoritativa al KMS.
type SolicitudRevalidacionAtestacionKMSBorrador struct {
	bloqueoSerializacionDiario
	AtestacionKMS            AtestacionKMSBorrador
	IdentidadPrimaria        ProyeccionIdentidadOperacion
	HuellaAAD                string
	Revision                 uint64
	Cercado                  uint64
	ArrendamientoVenceEn     time.Time
	ConfirmacionSolicitadaEn time.Time
	SolicitadaEn             time.Time
}

func NuevaSolicitudRevalidacionAtestacionKMSBorrador(
	confirmacion SolicitudConfirmacionBorrador,
	solicitadaEn time.Time,
) (SolicitudRevalidacionAtestacionKMSBorrador, error) {
	huellaAAD, err := confirmacion.Cifrado.AAD.HuellaSHA256()
	s := SolicitudRevalidacionAtestacionKMSBorrador{
		AtestacionKMS:     confirmacion.Cifrado.AtestacionKMS,
		IdentidadPrimaria: confirmacion.Reserva.IdentidadPrimaria, HuellaAAD: huellaAAD,
		Revision: confirmacion.Control.Revision, Cercado: confirmacion.Control.Cercado,
		ArrendamientoVenceEn:     confirmacion.Control.ArrendamientoVenceEn,
		ConfirmacionSolicitadaEn: confirmacion.SolicitadaEn,
		SolicitadaEn:             solicitadaEn,
	}
	if err != nil || confirmacion.Validar() != nil || s.Validar() != nil {
		return SolicitudRevalidacionAtestacionKMSBorrador{}, ErrRevalidacionKMSBorradorFallo
	}
	return s, nil
}

func (s SolicitudRevalidacionAtestacionKMSBorrador) Validar() error {
	if !s.AtestacionKMS.validaEstructural() ||
		!s.IdentidadPrimaria.valida() ||
		!coincideTextoConstante(s.AtestacionKMS.HuellaAAD, s.HuellaAAD) ||
		s.Revision == 0 || s.Cercado == 0 || !instanteOperacionCanonico(s.SolicitadaEn) ||
		!instanteOperacionCanonico(s.ArrendamientoVenceEn) ||
		!instanteOperacionCanonico(s.ConfirmacionSolicitadaEn) ||
		s.SolicitadaEn.Before(s.ConfirmacionSolicitadaEn) ||
		!s.SolicitadaEn.Before(s.ArrendamientoVenceEn) ||
		s.SolicitadaEn.Before(s.AtestacionKMS.EmitidaEn) ||
		!s.SolicitadaEn.Before(s.AtestacionKMS.ValidaHasta) {
		return ErrRevalidacionKMSBorradorFallo
	}
	return nil
}

type ResultadoRevalidacionAtestacionKMSBorrador struct {
	bloqueoSerializacionDiario
	AtestacionRef            string
	VersionAtestacion        uint32
	Estado                   string
	HuellaAAD                string
	ComprobacionRef          string
	HuellaComprobacionSHA256 string
	ComprobadaEn             time.Time
}

func (r ResultadoRevalidacionAtestacionKMSBorrador) ValidarPara(
	s SolicitudRevalidacionAtestacionKMSBorrador,
) error {
	if s.Validar() != nil || r.AtestacionRef != s.AtestacionKMS.AtestacionRef ||
		r.VersionAtestacion != s.AtestacionKMS.VersionAtestacion ||
		r.Estado != estadoRevalidacionKMSAutorizada ||
		!coincideTextoConstante(r.HuellaAAD, s.HuellaAAD) ||
		!referenciaProyeccionValida(r.ComprobacionRef) ||
		!huellaHexValida(r.HuellaComprobacionSHA256) ||
		r.ComprobacionRef == r.AtestacionRef || !instanteOperacionCanonico(r.ComprobadaEn) ||
		r.ComprobadaEn.Before(s.SolicitadaEn) || !r.ComprobadaEn.Before(s.AtestacionKMS.ValidaHasta) {
		return ErrRevalidacionKMSBorradorFallo
	}
	return nil
}

// RevalidadorAtestacionKMSBorrador debe ser propiedad del adaptador de
// persistencia y ejecutarse dentro de su seccion critica transaccional.
type RevalidadorAtestacionKMSBorrador interface {
	RevalidarAtestacionKMS(
		context.Context,
		SolicitudRevalidacionAtestacionKMSBorrador,
	) (ResultadoRevalidacionAtestacionKMSBorrador, error)
}
