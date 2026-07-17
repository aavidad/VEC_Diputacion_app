package domain

import (
	"bytes"
	"sort"
	"time"
)

const esquemaCompromisoTransicionFuenteAutoridadV1 = "vec.fuente_autoridad.compromiso_transicion.v1"

// CompromisoTransicionFuenteAutoridadV1 fija todos los datos que el
// comprobador debe atestar. Cambiar cualquiera de ellos invalida la evidencia.
type CompromisoTransicionFuenteAutoridadV1 struct {
	Esquema                    string                      `json:"esquema"`
	SolicitudRef               string                      `json:"solicitud_ref"`
	Fuente                     ReferenciaFuenteAutoridad   `json:"fuente"`
	RevisionPrevia             uint64                      `json:"revision_previa"`
	Secuencia                  uint64                      `json:"secuencia"`
	EstadoAnterior             EstadoFuenteAutoridad       `json:"estado_anterior"`
	EstadoNuevo                EstadoFuenteAutoridad       `json:"estado_nuevo"`
	Accion                     AccionActoFuenteAutoridad   `json:"accion"`
	ActorRef                   string                      `json:"actor_ref"`
	MotivoCodigo               CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	HuellaHistoriaPreviaSHA256 string                      `json:"huella_historia_previa_sha256"`
	PreparadaEn                time.Time                   `json:"preparada_en"`
	ExpiraEn                   time.Time                   `json:"expira_en"`
}

func (c CompromisoTransicionFuenteAutoridadV1) Validar() error {
	accion, valida := accionActoParaTransicionFuenteAutoridad(c.EstadoAnterior, c.EstadoNuevo)
	if c.Esquema != esquemaCompromisoTransicionFuenteAutoridadV1 ||
		!referenciaFuenteAutoridadValida(c.SolicitudRef) || c.Fuente.Validar() != nil ||
		c.RevisionPrevia == 0 || c.Secuencia == 0 || !c.EstadoAnterior.Valido() || !c.EstadoNuevo.Valido() ||
		!valida || c.Accion != accion || !referenciaPersonaFuenteAutoridadValida(c.ActorRef) ||
		!c.MotivoCodigo.Valido() || !esSHA256Autoridad(c.HuellaHistoriaPreviaSHA256) ||
		!instanteFuenteAutoridadCanonico(c.PreparadaEn) || !instanteFuenteAutoridadCanonico(c.ExpiraEn) ||
		!c.ExpiraEn.After(c.PreparadaEn) {
		return ErrTransicionAutoridadInvalida
	}
	return nil
}

func (c CompromisoTransicionFuenteAutoridadV1) HuellaSHA256() (string, error) {
	bytesCanonicos, err := c.BytesCanonicos()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(bytesCanonicos), nil
}

func (c CompromisoTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	return serializarCompromisoPersistibleAutoridadV1(c)
}

// MarshalJSON fuerza a todos los conectores a usar el mismo compromiso V1
// que se firma y cuya huella se conserva. No se serializa el tipo vivo.
func (c CompromisoTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return c.BytesCanonicos()
}

// SolicitudTransicionFuenteAutoridadV1 evita que quien integra el caso de uso
// repita actor, motivo, estado o instante entre la firma y la aplicacion.
type SolicitudTransicionFuenteAutoridadV1 struct {
	compromiso     CompromisoTransicionFuenteAutoridadV1
	bytesCanonicos []byte
}

type DatosPreparacionTransicionFuenteAutoridadV1 struct {
	EstadoNuevo  EstadoFuenteAutoridad
	ActorRef     string
	MotivoCodigo CodigoMotivoFuenteAutoridad
	SolicitudRef string
	PreparadaEn  time.Time
	ExpiraEn     time.Time
}

func nuevaSolicitudTransicionFuenteAutoridadV1(
	compromiso CompromisoTransicionFuenteAutoridadV1,
) (SolicitudTransicionFuenteAutoridadV1, error) {
	bytesCanonicos, err := compromiso.BytesCanonicos()
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	return SolicitudTransicionFuenteAutoridadV1{
		compromiso: compromiso, bytesCanonicos: append([]byte(nil), bytesCanonicos...),
	}, nil
}

func (s SolicitudTransicionFuenteAutoridadV1) Validar() error {
	bytesCanonicos, err := s.compromiso.BytesCanonicos()
	if err != nil || len(s.bytesCanonicos) == 0 || !bytes.Equal(bytesCanonicos, s.bytesCanonicos) {
		return ErrTransicionAutoridadInvalida
	}
	return nil
}

func (s SolicitudTransicionFuenteAutoridadV1) Compromiso() (CompromisoTransicionFuenteAutoridadV1, error) {
	if err := s.Validar(); err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	return s.compromiso, nil
}

func (s SolicitudTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := s.Validar(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.bytesCanonicos...), nil
}

// MarshalJSON evita que la opacidad de la solicitud se convierta
// accidentalmente en {}. La representación es el compromiso V1 canónico que
// puede custodiarse mientras un portafirmas completa el acto.
func (s SolicitudTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return s.BytesCanonicos()
}

// EvidenciaActoFuenteAutoridad es una atestacion neutral producida por un
// puerto de comprobacion. Validar comprueba coherencia estructural, no firma,
// competencia ni procedencia criptografica.
type EvidenciaActoFuenteAutoridad struct {
	EvidenciaRef                string                    `json:"evidencia_ref"`
	Accion                      AccionActoFuenteAutoridad `json:"accion"`
	FuenteID                    string                    `json:"fuente_id"`
	FuenteVersion               uint64                    `json:"fuente_version"`
	HuellaContenidoSHA256       string                    `json:"huella_contenido_sha256"`
	ActoRef                     string                    `json:"acto_ref"`
	DocumentoRef                string                    `json:"documento_ref"`
	RepresentacionRef           string                    `json:"representacion_ref"`
	HuellaDocumentoSHA256       string                    `json:"huella_documento_sha256"`
	OrganoRef                   string                    `json:"organo_ref"`
	FirmasRefs                  []string                  `json:"firmas_refs"`
	ComprobadorRef              string                    `json:"comprobador_ref"`
	AtestacionRef               string                    `json:"atestacion_ref"`
	HuellaAtestacionSHA256      string                    `json:"huella_atestacion_sha256"`
	FirmaAtestacionRef          string                    `json:"firma_atestacion_ref"`
	HuellaCompromisoSHA256      string                    `json:"huella_compromiso_sha256"`
	HuellaMensajeAtestadoSHA256 string                    `json:"huella_mensaje_atestado_sha256"`
	ActoOcurridoEn              time.Time                 `json:"acto_ocurrido_en"`
	ComprobadaEn                time.Time                 `json:"comprobada_en"`
}

func (e EvidenciaActoFuenteAutoridad) Validar() error {
	if !referenciaFuenteAutoridadValida(e.EvidenciaRef) || !e.Accion.Valida() ||
		!esClaveDocumentalCanonica(e.FuenteID) || e.FuenteVersion < 1 || !esSHA256Autoridad(e.HuellaContenidoSHA256) ||
		!referenciaFuenteAutoridadValida(e.ActoRef) || !referenciaFuenteAutoridadValida(e.DocumentoRef) ||
		!referenciaFuenteAutoridadValida(e.RepresentacionRef) || !esSHA256Autoridad(e.HuellaDocumentoSHA256) ||
		!referenciaFuenteAutoridadValida(e.OrganoRef) || len(e.FirmasRefs) == 0 ||
		len(e.FirmasRefs) > maximoFirmasActoFuenteAutoridad ||
		!referenciaFuenteAutoridadValida(e.ComprobadorRef) || !referenciaFuenteAutoridadValida(e.AtestacionRef) ||
		!esSHA256Autoridad(e.HuellaAtestacionSHA256) || !referenciaFuenteAutoridadValida(e.FirmaAtestacionRef) ||
		!esSHA256Autoridad(e.HuellaCompromisoSHA256) || !esSHA256Autoridad(e.HuellaMensajeAtestadoSHA256) ||
		!instanteFuenteAutoridadCanonico(e.ActoOcurridoEn) ||
		!instanteFuenteAutoridadCanonico(e.ComprobadaEn) || e.ComprobadaEn.Before(e.ActoOcurridoEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	vistas := make(map[string]struct{}, len(e.FirmasRefs))
	for _, firma := range e.FirmasRefs {
		if !referenciaFuenteAutoridadValida(firma) {
			return ErrEvidenciaActoAutoridadInvalida
		}
		if _, repetida := vistas[firma]; repetida {
			return ErrEvidenciaActoAutoridadInvalida
		}
		vistas[firma] = struct{}{}
	}
	return nil
}

func (e EvidenciaActoFuenteAutoridad) ClonarCanonica() (EvidenciaActoFuenteAutoridad, error) {
	if err := e.Validar(); err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	clon := e
	clon.FirmasRefs = append([]string(nil), e.FirmasRefs...)
	sort.Strings(clon.FirmasRefs)
	return clon, nil
}

const esquemaMensajeAtestacionActoFuenteAutoridadV1 = "vec.fuente_autoridad.mensaje_atestacion_acto.v1"

// MensajeAtestacionActoFuenteAutoridadV1 es el mensaje completo que cubre la
// atestacion externa. Excluye unicamente el sobre criptografico que lo firma
// para evitar una dependencia circular.
type MensajeAtestacionActoFuenteAutoridadV1 struct {
	Esquema               string                                `json:"esquema"`
	Compromiso            CompromisoTransicionFuenteAutoridadV1 `json:"compromiso"`
	EvidenciaRef          string                                `json:"evidencia_ref"`
	ActoRef               string                                `json:"acto_ref"`
	DocumentoRef          string                                `json:"documento_ref"`
	RepresentacionRef     string                                `json:"representacion_ref"`
	HuellaDocumentoSHA256 string                                `json:"huella_documento_sha256"`
	OrganoRef             string                                `json:"organo_ref"`
	FirmasRefs            []string                              `json:"firmas_refs"`
	ComprobadorRef        string                                `json:"comprobador_ref"`
	ActoOcurridoEn        time.Time                             `json:"acto_ocurrido_en"`
	ComprobadaEn          time.Time                             `json:"comprobada_en"`
}

// DatosMensajeAtestacionActoFuenteAutoridadV1 son hechos producidos por el
// comprobador. No incluyen campos derivados ni el sobre que todavía debe
// firmar el mensaje.
type DatosMensajeAtestacionActoFuenteAutoridadV1 struct {
	EvidenciaRef          string
	ActoRef               string
	DocumentoRef          string
	RepresentacionRef     string
	HuellaDocumentoSHA256 string
	OrganoRef             string
	FirmasRefs            []string
	ComprobadorRef        string
	ActoOcurridoEn        time.Time
	ComprobadaEn          time.Time
}

type DatosSobreAtestacionActoFuenteAutoridadV1 struct {
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	FirmaAtestacionRef     string
}

// PrepararMensajeAtestacionActoFuenteAutoridadV1 construye el único mensaje
// que un conector puede firmar. El adaptador no serializa el compromiso ni
// repite actor, recurso, revisión o acción.
func PrepararMensajeAtestacionActoFuenteAutoridadV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	datos DatosMensajeAtestacionActoFuenteAutoridadV1,
) (MensajeAtestacionActoFuenteAutoridadV1, error) {
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		return MensajeAtestacionActoFuenteAutoridadV1{}, err
	}
	mensaje := MensajeAtestacionActoFuenteAutoridadV1{
		Esquema: esquemaMensajeAtestacionActoFuenteAutoridadV1, Compromiso: compromiso,
		EvidenciaRef: datos.EvidenciaRef, ActoRef: datos.ActoRef,
		DocumentoRef: datos.DocumentoRef, RepresentacionRef: datos.RepresentacionRef,
		HuellaDocumentoSHA256: datos.HuellaDocumentoSHA256, OrganoRef: datos.OrganoRef,
		FirmasRefs: append([]string(nil), datos.FirmasRefs...), ComprobadorRef: datos.ComprobadorRef,
		ActoOcurridoEn: normalizarInstanteFuenteAutoridad(datos.ActoOcurridoEn),
		ComprobadaEn:   normalizarInstanteFuenteAutoridad(datos.ComprobadaEn),
	}
	if _, err := mensaje.BytesCanonicos(); err != nil {
		return MensajeAtestacionActoFuenteAutoridadV1{}, err
	}
	return mensaje, nil
}

// ConstituirEvidenciaAtestadaV1 incorpora el sobre criptográfico después de
// firmar/verificar el mensaje y calcula todos los campos derivados.
func (m MensajeAtestacionActoFuenteAutoridadV1) ConstituirEvidenciaAtestadaV1(
	sobre DatosSobreAtestacionActoFuenteAutoridadV1,
) (EvidenciaActoFuenteAutoridad, error) {
	if !referenciaFuenteAutoridadValida(sobre.AtestacionRef) ||
		!esSHA256Autoridad(sobre.HuellaAtestacionSHA256) ||
		!referenciaFuenteAutoridadValida(sobre.FirmaAtestacionRef) {
		return EvidenciaActoFuenteAutoridad{}, ErrEvidenciaActoAutoridadInvalida
	}
	huellaCompromiso, err := m.Compromiso.HuellaSHA256()
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	huellaMensaje, err := m.HuellaSHA256()
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	evidencia := EvidenciaActoFuenteAutoridad{
		EvidenciaRef: m.EvidenciaRef, Accion: m.Compromiso.Accion,
		FuenteID: m.Compromiso.Fuente.FuenteID, FuenteVersion: m.Compromiso.Fuente.Version,
		HuellaContenidoSHA256: m.Compromiso.Fuente.HuellaContenidoSHA256,
		ActoRef:               m.ActoRef, DocumentoRef: m.DocumentoRef, RepresentacionRef: m.RepresentacionRef,
		HuellaDocumentoSHA256: m.HuellaDocumentoSHA256, OrganoRef: m.OrganoRef,
		FirmasRefs: append([]string(nil), m.FirmasRefs...), ComprobadorRef: m.ComprobadorRef,
		AtestacionRef: sobre.AtestacionRef, HuellaAtestacionSHA256: sobre.HuellaAtestacionSHA256,
		FirmaAtestacionRef: sobre.FirmaAtestacionRef, HuellaCompromisoSHA256: huellaCompromiso,
		HuellaMensajeAtestadoSHA256: huellaMensaje,
		ActoOcurridoEn:              m.ActoOcurridoEn, ComprobadaEn: m.ComprobadaEn,
	}
	return evidencia.ClonarCanonica()
}

func (m MensajeAtestacionActoFuenteAutoridadV1) Validar() error {
	if m.Esquema != esquemaMensajeAtestacionActoFuenteAutoridadV1 || m.Compromiso.Validar() != nil ||
		!referenciaFuenteAutoridadValida(m.EvidenciaRef) || !referenciaFuenteAutoridadValida(m.ActoRef) ||
		!referenciaFuenteAutoridadValida(m.DocumentoRef) || !referenciaFuenteAutoridadValida(m.RepresentacionRef) ||
		!esSHA256Autoridad(m.HuellaDocumentoSHA256) || !referenciaFuenteAutoridadValida(m.OrganoRef) ||
		len(m.FirmasRefs) == 0 || len(m.FirmasRefs) > maximoFirmasActoFuenteAutoridad ||
		!referenciaFuenteAutoridadValida(m.ComprobadorRef) || !instanteFuenteAutoridadCanonico(m.ActoOcurridoEn) ||
		!instanteFuenteAutoridadCanonico(m.ComprobadaEn) || m.ComprobadaEn.Before(m.ActoOcurridoEn) ||
		m.ComprobadaEn.Before(m.Compromiso.PreparadaEn) || !m.ComprobadaEn.Before(m.Compromiso.ExpiraEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	vistas := make(map[string]struct{}, len(m.FirmasRefs))
	for _, firma := range m.FirmasRefs {
		if !referenciaFuenteAutoridadValida(firma) {
			return ErrEvidenciaActoAutoridadInvalida
		}
		if _, repetida := vistas[firma]; repetida {
			return ErrEvidenciaActoAutoridadInvalida
		}
		vistas[firma] = struct{}{}
	}
	return nil
}

func (m MensajeAtestacionActoFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	m.FirmasRefs = append([]string(nil), m.FirmasRefs...)
	sort.Strings(m.FirmasRefs)
	return serializarMensajeAtestacionPersistibleAutoridadV1(m)
}

// MarshalJSON evita que el orden recibido de las firmas u otro detalle del
// tipo vivo produzca unos bytes distintos de los entregados a Portafirmas.
func (m MensajeAtestacionActoFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return m.BytesCanonicos()
}

func (m MensajeAtestacionActoFuenteAutoridadV1) HuellaSHA256() (string, error) {
	bytesCanonicos, err := m.BytesCanonicos()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(bytesCanonicos), nil
}

func mensajeAtestacionActoFuenteAutoridad(
	compromiso CompromisoTransicionFuenteAutoridadV1,
	evidencia EvidenciaActoFuenteAutoridad,
) MensajeAtestacionActoFuenteAutoridadV1 {
	return MensajeAtestacionActoFuenteAutoridadV1{
		Esquema: esquemaMensajeAtestacionActoFuenteAutoridadV1, Compromiso: compromiso,
		EvidenciaRef: evidencia.EvidenciaRef, ActoRef: evidencia.ActoRef,
		DocumentoRef: evidencia.DocumentoRef, RepresentacionRef: evidencia.RepresentacionRef,
		HuellaDocumentoSHA256: evidencia.HuellaDocumentoSHA256, OrganoRef: evidencia.OrganoRef,
		FirmasRefs: append([]string(nil), evidencia.FirmasRefs...), ComprobadorRef: evidencia.ComprobadorRef,
		ActoOcurridoEn: evidencia.ActoOcurridoEn, ComprobadaEn: evidencia.ComprobadaEn,
	}
}
