package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Descarga de un borrador RRHH (PDF o DOCX) servido por la consulta de detalle
// del expediente. Es una acción propia, distinta de la consulta: la decisión
// V3 liga el tipo de borrador, el formato y la huella del archivo generado, y
// su consumo (AD199/CT177) deja el asiento en la auditoría común y una fila en
// el registro de descargas de CT. La finalidad es la de tramitar el expediente.
const (
	AccionDescargarBorradorRRHH            = "contratacion_temporal.borrador_rrhh.descargar"
	FinalidadDescargarBorradorRRHH         = FinalidadConsultarDetalleRRHH
	AudienciaConsumoDescargaBorradorRRHHV3 = "vec_contratacion_temporal.borrador_rrhh.descargar.v1"
	DominioHuellaDescargaBorradorRRHH      = "vec.contratacion_temporal.descarga_borrador_rrhh.v1"
	FormatoBorradorRRHHPDF                 = "pdf"
	FormatoBorradorRRHHDOCX                = "docx"
	maximoBorradorDescargaRRHHBytes        = 16 << 20
)

var (
	ErrDescargaBorradorRRHHInvalida       = errors.New("contratacion temporal: descarga de borrador RRHH invalida")
	ErrDescargaBorradorRRHHNoDisponible   = errors.New("contratacion temporal: registro de descarga de borrador RRHH no disponible")
	ErrDescargaBorradorRRHHVersionAusente = errors.New("contratacion temporal: version de expediente de la descarga inexistente")
	// ErrDescargaBorradorRRHHSinAnotar indica que no había nada que anotar:
	// sin registrador AD169 o sin actor resuelto en la petición.
	ErrDescargaBorradorRRHHSinAnotar = errors.New("contratacion temporal: fallo de descarga sin actor que anotar")

	tipoBorradorDescargaValido = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	huellaDescargaValida       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	// Misma forma que admite CT177 para la referencia del expediente.
	expedienteDescargaValido = regexp.MustCompile(`^expediente:[A-Za-z0-9._:/#-]{1,149}$`)
)

// SolicitudDescargaBorradorRRHH describe el archivo ya generado y validado que
// se va a entregar. Nunca lleva su contenido, solo su huella y su tamaño.
type SolicitudDescargaBorradorRRHH struct {
	ExpedienteRef        string
	VersionExpediente    uint64
	Tipo                 TipoBorradorRRHH
	Formato              string
	DocumentoSHA256      string
	TamanoBytes          int
	ConsultaHuellaSHA256 string
}

func (s SolicitudDescargaBorradorRRHH) Validar() error {
	if !expedienteDescargaValido.MatchString(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionExpediente < 1 || s.VersionExpediente > 9_007_199_254_740_991 ||
		!tipoBorradorDescargaValido.MatchString(string(s.Tipo)) ||
		(s.Formato != FormatoBorradorRRHHPDF && s.Formato != FormatoBorradorRRHHDOCX) ||
		!huellaDescargaValida.MatchString(s.DocumentoSHA256) ||
		s.TamanoBytes < 1 || s.TamanoBytes > maximoBorradorDescargaRRHHBytes ||
		!huellaDescargaValida.MatchString(s.ConsultaHuellaSHA256) {
		return ErrDescargaBorradorRRHHInvalida
	}
	return nil
}

// CanonDescargaBorradorRRHH es la misma cadena que calcula CT177
// (canon_descarga_borrador_rrhh_v1): campos de forma cerrada separados por
// saltos de línea.
func CanonDescargaBorradorRRHH(s SolicitudDescargaBorradorRRHH) ([]byte, error) {
	if err := s.Validar(); err != nil {
		return nil, err
	}
	return []byte(DominioHuellaDescargaBorradorRRHH + "\n" + s.ExpedienteRef + "\n" +
		strconv.FormatUint(s.VersionExpediente, 10) + "\n" + string(s.Tipo) + "\n" + s.Formato + "\n" +
		s.DocumentoSHA256 + "\n" + strconv.Itoa(s.TamanoBytes) + "\n" + s.ConsultaHuellaSHA256), nil
}

// AlcanceDescargaBorradorRRHH es el ámbito fijo de la consulta RRHH que ya
// autorizó el detalle; la petición nunca lo elige.
type AlcanceDescargaBorradorRRHH struct {
	OrganizacionRef string
	ClaseAmbito     ClaseAmbitoConsultaRRHH
	AmbitoRef       string
}

// RecursoDescargaBorradorRRHH es el recurso de la decisión: el expediente, con
// el ámbito de la consulta y, como atributo, la huella del canon. CT177 rehace
// la misma huella de contexto antes de consumir.
func RecursoDescargaBorradorRRHH(s SolicitudDescargaBorradorRRHH, a AlcanceDescargaBorradorRRHH) (vecdomain.RecursoAutorizable, error) {
	canon, err := CanonDescargaBorradorRRHH(s)
	if err != nil || !domain.ReferenciaOpacaValida(a.OrganizacionRef) || !domain.ReferenciaOpacaValida(a.AmbitoRef) || a.ClaseAmbito == "" {
		return vecdomain.RecursoAutorizable{}, ErrDescargaBorradorRRHHInvalida
	}
	h := sha256.Sum256(canon)
	r := vecdomain.RecursoAutorizable{
		Referencia: s.ExpedienteRef, ModuloID: ModuloContratacion, Tipo: TipoRecursoExpediente,
		Ambitos: map[string]string{"organizacion_ref": a.OrganizacionRef, "clase_ambito": string(a.ClaseAmbito), "ambito_ref": a.AmbitoRef},
		Atributos: map[string]string{"consulta_dominio": DominioHuellaDescargaBorradorRRHH,
			"consulta_huella_sha256": hex.EncodeToString(h[:])},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ErrDescargaBorradorRRHHInvalida
	}
	return r, nil
}

// ReciboDescargaBorradorRRHH es el acuse del registro: la fila de CT177 y el
// asiento de la auditoría común que dejó el consumo.
type ReciboDescargaBorradorRRHH struct {
	DescargaRef  string
	AuditoriaRef string
	DecisionRef  string
	RegistradaEn time.Time
}

// RegistradorDescargaBorradorRRHH lo usa el transporte justo antes de escribir
// el archivo. RegistrarDescarga autoriza, consume y registra; si falla, no se
// entrega nada. RegistrarFalloDescarga deja en la auditoría común un intento
// fallido anterior (consulta denegada, error al generar o validar el archivo).
// Los errores de ambos, cuando el intento quedó registrado, implementan
// FalloLecturaAuditado.
type RegistradorDescargaBorradorRRHH interface {
	RegistrarDescarga(context.Context, SolicitudDescargaBorradorRRHH) (ReciboDescargaBorradorRRHH, error)
	RegistrarFalloDescarga(ctx context.Context, expedienteRef string, causa error) error
}

// RepositorioDescargaBorradorRRHH consume el material y escribe la fila en una
// sola transacción SERIALIZABLE.
type RepositorioDescargaBorradorRRHH interface {
	RegistrarDescargaBorrador(context.Context, AlcanceDescargaBorradorRRHH, SolicitudDescargaBorradorRRHH, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboDescargaBorradorRRHH, error)
}
