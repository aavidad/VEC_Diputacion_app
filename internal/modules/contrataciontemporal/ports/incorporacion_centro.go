package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Confirmación de la incorporación por el centro (CT124, AD3-88). El centro
// de la petición ratificada y entregada a RRHH confirma la fecha y el
// documento que la acredita; el tipo lo fija la regla del catálogo para la
// modalidad del expediente. Solo referencia y huella del documento.
const (
	AccionConfirmarIncorporacionCentro    = "contratacion_temporal.incorporacion.confirmar_centro"
	AccionConsultarIncorporacionesCentro  = "contratacion_temporal.incorporacion.consultar_centro"
	TipoRecursoIncorporacionCentro        = "incorporacion_centro_contratacion_temporal"
	FinalidadIncorporacionCentro          = "gestionar_peticion_centro"
	OperacionConfirmarIncorporacionCentro = "confirmar"
	maximoExpedientesIncorporacionCentro  = 50
)

var (
	ErrIncorporacionCentroInvalida       = errors.New("contratacion temporal: confirmacion de incorporacion del centro invalida")
	ErrIncorporacionCentroDenegada       = errors.New("contratacion temporal: confirmacion de incorporacion del centro denegada")
	ErrIncorporacionCentroNoAdmitida     = errors.New("contratacion temporal: el expediente no admite la confirmacion del centro")
	ErrClaveIncorporacionCentroUsada     = errors.New("contratacion temporal: clave de confirmacion del centro usada con otro contenido")
	ErrIncorporacionCentroNoDisponible   = errors.New("contratacion temporal: confirmacion de incorporacion del centro no disponible")
	ErrReglaAcreditacionNoDisponible     = errors.New("contratacion temporal: regla de acreditacion de la incorporacion no disponible")
	patronClaveIdempotenciaIncorporacion = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	patronFechaCivilIncorporacionCentro  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	patronReferenciaReglaIncorporacion   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,254}$`)
)

// ReglaDocumentoIncorporacion es la entrada del catálogo que fija el
// documento acreditativo para una modalidad, con su procedencia exacta.
type ReglaDocumentoIncorporacion struct {
	Referencia     string `json:"referencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	ModalidadClave string `json:"modalidad_clave"`
}

func (r ReglaDocumentoIncorporacion) Valida() bool {
	return patronReferenciaReglaIncorporacion.MatchString(r.Referencia) && huellaSHA256OperacionAnalisisValida(r.HuellaSHA256) &&
		domain.ClaveCatalogo(r.ModalidadClave).Valida()
}

// FuenteReglaAcreditacionIncorporacion resuelve con el catálogo vigente el
// tipo de documento que acredita la incorporación de una modalidad.
type FuenteReglaAcreditacionIncorporacion interface {
	DocumentoAcreditativo(ctx context.Context, modalidad domain.ClaveCatalogo) (domain.ClaveCatalogo, ReglaDocumentoIncorporacion, error)
}

// ConsultaIncorporacionesCentro es la bandeja del actor del centro.
type ConsultaIncorporacionesCentro struct {
	Modo            string                     `json:"modo"`
	OrganizacionRef string                     `json:"organizacion_ref"`
	Actor           domain.ActorPeticionCentro `json:"actor"`
}

func (c ConsultaIncorporacionesCentro) Validar() error {
	if c.Modo != "bandeja" || !domain.ReferenciaOpacaValida(c.OrganizacionRef) || c.Actor.Validar() != nil {
		return ErrIncorporacionCentroInvalida
	}
	return nil
}

// DocumentoIncorporacionCentro: tipo del catálogo, referencia y huella.
type DocumentoIncorporacionCentro struct {
	Tipo       string `json:"tipo"`
	Referencia string `json:"referencia"`
	SHA256     string `json:"sha256"`
}

// MaterialIncorporacionCentro es la intención exacta que se autoriza y se
// persiste; su serialización JSON es la que SQL recibe y resume.
type MaterialIncorporacionCentro struct {
	Operacion          string                       `json:"operacion"`
	ClaveIdempotencia  string                       `json:"clave_idempotencia"`
	OrganizacionRef    string                       `json:"organizacion_ref"`
	Actor              domain.ActorPeticionCentro   `json:"actor"`
	PeticionRef        string                       `json:"peticion_ref"`
	ExpedienteRef      string                       `json:"expediente_ref"`
	FechaIncorporacion string                       `json:"fecha_incorporacion"`
	Documento          DocumentoIncorporacionCentro `json:"documento"`
	Regla              ReglaDocumentoIncorporacion  `json:"regla"`
}

func (m MaterialIncorporacionCentro) Validar() error {
	if m.Operacion != OperacionConfirmarIncorporacionCentro || !patronClaveIdempotenciaIncorporacion.MatchString(m.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(m.OrganizacionRef) || m.Actor.Validar() != nil ||
		!domain.ReferenciaOpacaValida(m.PeticionRef) || !domain.ReferenciaOpacaValida(m.ExpedienteRef) ||
		!fechaCivilIncorporacionCentroValida(m.FechaIncorporacion) ||
		!domain.ClaveCatalogo(m.Documento.Tipo).Valida() || !domain.ReferenciaOpacaValida(m.Documento.Referencia) ||
		!huellaSHA256OperacionAnalisisValida(m.Documento.SHA256) || m.Documento.SHA256 == "0000000000000000000000000000000000000000000000000000000000000000" ||
		!m.Regla.Valida() {
		return ErrIncorporacionCentroInvalida
	}
	return nil
}

func fechaCivilIncorporacionCentroValida(v string) bool {
	if !patronFechaCivilIncorporacionCentro.MatchString(v) {
		return false
	}
	f, err := time.Parse(time.DateOnly, v)
	return err == nil && f.Format(time.DateOnly) == v
}

// RecursoIncorporacionCentro es el recurso autorizable: la referencia
// (expediente o bandeja del centro), los ámbitos de organización y centro y
// la huella del material exacto. SQL lo recompone y lo coteja.
func RecursoIncorporacionCentro(referencia, organizacion, centro string, material []byte) vd.RecursoAutorizable {
	h := sha256.Sum256(material)
	return vd.RecursoAutorizable{Referencia: referencia, ModuloID: ModuloContratacion, Tipo: TipoRecursoIncorporacionCentro,
		Ambitos:   map[string]string{"organizacion_ref": organizacion, "centro_ref": centro},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
}

// ReferenciaBandejaIncorporacionesCentro es el recurso de la bandeja.
func ReferenciaBandejaIncorporacionesCentro(centro string) string {
	return "incorporaciones:centro:" + centro
}

// SerializarConsultaIncorporacionesCentro y SerializarMaterialIncorporacionCentro
// fijan los bytes exactos que se autorizan y se envían a SQL.
func SerializarConsultaIncorporacionesCentro(c ConsultaIncorporacionesCentro) ([]byte, error) {
	if c.Validar() != nil {
		return nil, ErrIncorporacionCentroInvalida
	}
	return json.Marshal(c)
}

func SerializarMaterialIncorporacionCentro(m MaterialIncorporacionCentro) ([]byte, error) {
	if m.Validar() != nil {
		return nil, ErrIncorporacionCentroInvalida
	}
	return json.Marshal(m)
}

// ConfirmacionIncorporacionCentro es lo que el centro ve de su confirmación.
type ConfirmacionIncorporacionCentro struct {
	FechaIncorporacion string    `json:"fecha_incorporacion"`
	DocumentoTipo      string    `json:"documento_tipo"`
	DocumentoRef       string    `json:"documento_ref"`
	ReciboRef          string    `json:"recibo_ref"`
	RegistradaEn       time.Time `json:"registrada_en"`
}

// PeriodoIncorporacionCentro es el periodo solicitado por el centro.
type PeriodoIncorporacionCentro struct {
	Inicio string `json:"inicio"`
	Fin    string `json:"fin"`
}

// ExpedienteIncorporacionCentro es una fila de la bandeja del centro: sin
// datos personales; solo referencias, fase, modalidad y periodo solicitado.
type ExpedienteIncorporacionCentro struct {
	PeticionRef    string                           `json:"peticion_ref"`
	ExpedienteRef  string                           `json:"expediente_ref"`
	NumeroVisible  string                           `json:"numero_visible"`
	Version        uint64                           `json:"version"`
	Fase           string                           `json:"fase"`
	Estado         string                           `json:"estado"`
	ModalidadClave string                           `json:"modalidad_clave"`
	CategoriaRef   string                           `json:"categoria_ref"`
	Periodo        *PeriodoIncorporacionCentro      `json:"periodo"`
	Confirmacion   *ConfirmacionIncorporacionCentro `json:"confirmacion"`
}

// AdmiteConfirmacion: expediente en nombramiento, en curso y sin confirmar.
func (e ExpedienteIncorporacionCentro) AdmiteConfirmacion() bool {
	return e.Fase == string(domain.FaseNombramiento) && e.Estado == string(domain.EstadoEnCurso) && e.Confirmacion == nil
}

func (e ExpedienteIncorporacionCentro) Valido() bool {
	if !domain.ReferenciaOpacaValida(e.PeticionRef) || !domain.ReferenciaOpacaValida(e.ExpedienteRef) || e.NumeroVisible == "" ||
		len(e.NumeroVisible) > 64 || e.Version == 0 || !domain.ClaveFase(e.Fase).Valida() || !domain.EstadoOperativo(e.Estado).Valido() ||
		(e.ModalidadClave != "" && !domain.ClaveCatalogo(e.ModalidadClave).Valida()) || len(e.CategoriaRef) > 160 {
		return false
	}
	if c := e.Confirmacion; c != nil && (!fechaCivilIncorporacionCentroValida(c.FechaIncorporacion) || !domain.ClaveCatalogo(c.DocumentoTipo).Valida() ||
		!domain.ReferenciaOpacaValida(c.DocumentoRef) || !domain.ReferenciaOpacaValida(c.ReciboRef) || c.RegistradaEn.IsZero()) {
		return false
	}
	return true
}

// ReciboIncorporacionCentro es el recibo durable de la confirmación.
type ReciboIncorporacionCentro struct {
	ReciboRef          string    `json:"recibo_ref"`
	ConfirmacionRef    string    `json:"confirmacion_ref"`
	PeticionRef        string    `json:"peticion_ref"`
	ExpedienteRef      string    `json:"expediente_ref"`
	NumeroVisible      string    `json:"numero_visible"`
	FechaIncorporacion string    `json:"fecha_incorporacion"`
	DocumentoTipo      string    `json:"documento_tipo"`
	ActorRef           string    `json:"actor_ref"`
	RegistradoEn       time.Time `json:"registrado_en"`
	EstadoLocal        string    `json:"estado_local"`
}

func (r ReciboIncorporacionCentro) ValidarPara(m MaterialIncorporacionCentro) error {
	if !domain.ReferenciaOpacaValida(r.ReciboRef) || !domain.ReferenciaOpacaValida(r.ConfirmacionRef) ||
		r.PeticionRef != m.PeticionRef || r.ExpedienteRef != m.ExpedienteRef || r.FechaIncorporacion != m.FechaIncorporacion ||
		r.DocumentoTipo != m.Documento.Tipo || r.ActorRef != m.Actor.ActorRef || r.RegistradoEn.IsZero() ||
		(r.EstadoLocal != "registrado" && r.EstadoLocal != "replay_confirmado") {
		return ErrIncorporacionCentroNoDisponible
	}
	return nil
}

// RepositorioIncorporacionCentro lista la bandeja y confirma; cada llamada
// consume una autorización nueva ligada al material exacto.
type RepositorioIncorporacionCentro interface {
	ListarIncorporacionesCentro(ctx context.Context, c ConsultaIncorporacionesCentro) ([]ExpedienteIncorporacionCentro, error)
	ConfirmarIncorporacionCentro(ctx context.Context, m MaterialIncorporacionCentro) (ReciboIncorporacionCentro, error)
}

// LimiteExpedientesIncorporacionCentro es la cota de la bandeja en SQL.
func LimiteExpedientesIncorporacionCentro() int { return maximoExpedientesIncorporacionCentro }
