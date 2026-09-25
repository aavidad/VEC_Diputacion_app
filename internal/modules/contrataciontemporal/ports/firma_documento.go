package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Firma de prueba de los borradores del expediente. La firma se hace en el
// equipo de la persona con AutoFirma, la verifica el validador y se registra
// en CT118 ligada al expediente, su versión, la huella del documento y el
// paso del catálogo. Nunca tiene eficacia administrativa: esa la aporta el
// portafirmas corporativo.
const (
	AccionFirmarDocumento      = "contratacion_temporal.documento.firmar"
	AudienciaFirmaDocumentoV3  = "vec_contratacion_temporal.firma_documento.v1"
	TipoRecursoFirmaDocumento  = "firma_documento_contratacion_temporal"
	FinalidadFirmaDocumento    = "gestionar_contratacion_temporal"
	PrefijoRecursoFirma        = "operacion-firma-ct:"
	PoliticaVerificacionFirma  = "politica:vec:firma:verificacion-autonoma:v1"
	MaximoDocumentoFirmaBytes  = 1 << 20
	MaximoMotivoDevolucionRune = 500
)

var (
	ErrSolicitudFirmaDocumentoInvalida    = errors.New("contratacion temporal: solicitud de firma no valida")
	ErrFirmaDocumentoDenegada             = errors.New("contratacion temporal: firma denegada")
	ErrClaveFirmaDocumentoUsada           = errors.New("contratacion temporal: clave de firma reutilizada")
	ErrFirmaDocumentoEnConflicto          = errors.New("contratacion temporal: firma en conflicto con la historia")
	ErrCadenaFirmaDocumentoRota           = errors.New("contratacion temporal: la firma no continua la del paso anterior")
	ErrRegistroFirmaDocumentoNoDisponible = errors.New("contratacion temporal: registro de firmas no disponible")
	ErrResultadoFirmaDocumentoInvalido    = errors.New("contratacion temporal: resultado del registro de firmas no confiable")
)

var claveFirmaDocumentoValida = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$`)

// ClaveIdempotenciaFirmaValida admite la gramática de CT118.
func ClaveIdempotenciaFirmaValida(v string) bool { return claveFirmaDocumentoValida.MatchString(v) }

// MotivoDevolucionFirmaValido exige texto llano de 3 a 500 caracteres, sin
// controles ni espacios en los extremos, igual que CT118.
func MotivoDevolucionFirmaValido(v string) bool {
	n := utf8.RuneCountInString(v)
	if !utf8.ValidString(v) || n < 3 || n > MaximoMotivoDevolucionRune || strings.TrimSpace(v) != v {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// MaterialFirmaDocumento es exactamente lo que registra CT118 y lo que la
// decisión V3 ata al recurso por su huella. Lo construye la aplicación: el
// actor y el perfil no forman parte, los aporta la decisión V3.
type MaterialFirmaDocumento struct {
	OrganizacionRef      string
	ExpedienteRef        string
	VersionExpediente    uint64
	Documento            string
	CatalogoRef          string
	CatalogoHuella       string
	PasoRef              string
	PasoOrden            int
	Secuencia            int
	Resultado            domain.ResultadoFirmaDocumento
	MotivoDevolucion     string
	OriginalHuella       string
	FirmadoHuella        string
	CertificadoHuella    string
	FirmanteRef          string
	PoliticaVerificacion string
	RevocacionEstado     string
	SelloTiempoEstado    string
	ClaveIdempotencia    string
}

// Validar comprueba la forma que CT118 exige: una firma lleva todo su
// dictamen y una devolución solo su motivo.
func (m MaterialFirmaDocumento) Validar() error {
	if !domain.ReferenciaOpacaValida(m.OrganizacionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		m.VersionExpediente == 0 || m.VersionExpediente > 9007199254740991 ||
		!domain.ClaveDocumentoFirmaValida(m.Documento) || !domain.ReferenciaOpacaValida(m.CatalogoRef) ||
		!domain.HuellaSHA256FirmaValida(m.CatalogoHuella) || m.PasoRef == "" || len(m.PasoRef) > 256 ||
		m.PasoOrden < 1 || m.PasoOrden > domain.MaximoPasosCircuitoFirma || m.Secuencia < 1 || m.Secuencia > 100000 ||
		!ClaveIdempotenciaFirmaValida(m.ClaveIdempotencia) {
		return ErrSolicitudFirmaDocumentoInvalida
	}
	switch m.Resultado {
	case domain.ResultadoFirmaFirmado:
		if m.MotivoDevolucion != "" || !domain.HuellaSHA256FirmaValida(m.OriginalHuella) ||
			!domain.HuellaSHA256FirmaValida(m.FirmadoHuella) || m.OriginalHuella == m.FirmadoHuella ||
			!domain.HuellaSHA256FirmaValida(m.CertificadoHuella) || !domain.ReferenciaOpacaValida(m.FirmanteRef) ||
			m.PoliticaVerificacion != PoliticaVerificacionFirma || m.RevocacionEstado != "vigente" ||
			(m.SelloTiempoEstado != "no_presente" && m.SelloTiempoEstado != "valido" && m.SelloTiempoEstado != "no_comprobado") {
			return ErrSolicitudFirmaDocumentoInvalida
		}
	case domain.ResultadoFirmaDevuelto:
		if !MotivoDevolucionFirmaValido(m.MotivoDevolucion) || m.OriginalHuella != "" || m.FirmadoHuella != "" ||
			m.CertificadoHuella != "" || m.FirmanteRef != "" || m.PoliticaVerificacion != "" ||
			m.RevocacionEstado != "" || m.SelloTiempoEstado != "" {
			return ErrSolicitudFirmaDocumentoInvalida
		}
	default:
		return ErrSolicitudFirmaDocumentoInvalida
	}
	return nil
}

func nulo(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// Canonico devuelve el JSON de orden fijo que CT118 recibe y cuya huella
// liga la decisión V3. Los campos ausentes viajan como null.
func (m MaterialFirmaDocumento) Canonico() ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrSolicitudFirmaDocumentoInvalida
	}
	return json.Marshal(struct {
		OrganizacionRef      string  `json:"OrganizacionRef"`
		ExpedienteRef        string  `json:"ExpedienteRef"`
		VersionExpediente    uint64  `json:"VersionExpediente"`
		Documento            string  `json:"Documento"`
		CatalogoRef          string  `json:"CatalogoRef"`
		CatalogoHuella       string  `json:"CatalogoHuella"`
		PasoRef              string  `json:"PasoRef"`
		PasoOrden            int     `json:"PasoOrden"`
		Secuencia            int     `json:"Secuencia"`
		Resultado            string  `json:"Resultado"`
		MotivoDevolucion     *string `json:"MotivoDevolucion"`
		OriginalHuella       *string `json:"OriginalHuella"`
		FirmadoHuella        *string `json:"FirmadoHuella"`
		CertificadoHuella    *string `json:"CertificadoHuella"`
		FirmanteRef          *string `json:"FirmanteRef"`
		PoliticaVerificacion *string `json:"PoliticaVerificacion"`
		RevocacionEstado     *string `json:"RevocacionEstado"`
		SelloTiempoEstado    *string `json:"SelloTiempoEstado"`
		ClaveIdempotencia    string  `json:"ClaveIdempotencia"`
	}{m.OrganizacionRef, m.ExpedienteRef, m.VersionExpediente, m.Documento, m.CatalogoRef, m.CatalogoHuella,
		m.PasoRef, m.PasoOrden, m.Secuencia, string(m.Resultado), nulo(m.MotivoDevolucion), nulo(m.OriginalHuella),
		nulo(m.FirmadoHuella), nulo(m.CertificadoHuella), nulo(m.FirmanteRef), nulo(m.PoliticaVerificacion),
		nulo(m.RevocacionEstado), nulo(m.SelloTiempoEstado), m.ClaveIdempotencia})
}

// HuellaSHA256 es la huella del JSON canónico.
func (m MaterialFirmaDocumento) HuellaSHA256() (string, error) {
	c, err := m.Canonico()
	if err != nil {
		return "", err
	}
	s := sha256.Sum256(c)
	return hex.EncodeToString(s[:]), nil
}

// RecursoRef es el efecto V3 de la operación: estable por clave.
func (m MaterialFirmaDocumento) RecursoRef() string { return PrefijoRecursoFirma + m.ClaveIdempotencia }

// CapacidadFirmaDocumento transporta solo el material V3 opaco.
type CapacidadFirmaDocumento struct {
	material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func TransportarMaterialFirmaDocumento(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) CapacidadFirmaDocumento {
	return CapacidadFirmaDocumento{material: m}
}

func (c CapacidadFirmaDocumento) ExportarMaterialParaConsumidor() vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	return c.material
}

// AutorizadorFirmaDocumento obtiene de la frontera confiable la concesión
// ligada al material exacto. La identidad nunca viene de la petición.
type AutorizadorFirmaDocumento interface {
	AutorizarFirmaDocumento(context.Context, MaterialFirmaDocumento) (CapacidadFirmaDocumento, error)
}

// ReciboFirmaDocumento es lo que devuelve CT118 tras escribir o recuperar.
type ReciboFirmaDocumento struct {
	FirmaRef          string
	ReciboRef         string
	Secuencia         int
	Resultado         domain.ResultadoFirmaDocumento
	ExpedienteVersion uint64
	ActorRef          string
	PerfilRef         string
	RegistradaEn      time.Time
	SolicitudHuella   string
	YaRegistrada      bool
}

// FirmaRegistrada es una fila de la historia de un expediente.
type FirmaRegistrada struct {
	FirmaRef          string
	ReciboRef         string
	Documento         string
	Secuencia         int
	ExpedienteVersion uint64
	CatalogoRef       string
	CatalogoHuella    string
	PasoRef           string
	PasoOrden         int
	Resultado         domain.ResultadoFirmaDocumento
	// ConMotivoDevolucion: la devolución consta con motivo; su texto libre no
	// se devuelve en la lectura de la historia.
	ConMotivoDevolucion bool
	OriginalHuella      string
	FirmadoHuella       string
	CertificadoHuella   string
	FirmanteRef         string
	SelloTiempoEstado   string
	ActorRef            string
	PerfilRef           string
	RegistradaEn        time.Time
}

// RegistroFirmasDocumento es el almacén de solo adición de CT118.
type RegistroFirmasDocumento interface {
	RegistrarFirma(context.Context, MaterialFirmaDocumento, CapacidadFirmaDocumento) (ReciboFirmaDocumento, error)
	ConsultarFirmas(ctx context.Context, organizacionRef, expedienteRef string) ([]FirmaRegistrada, error)
}

// FuenteCircuitoFirma entrega el catálogo de circuito vigente.
type FuenteCircuitoFirma interface {
	CircuitoFirma(context.Context) (domain.CircuitoFirma, error)
}
