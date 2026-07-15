package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPoliticaCotejoNoEncontrada           = errors.New("vec: politica de cotejo no encontrada")
	ErrVersionPoliticaCotejoYaExiste        = errors.New("vec: version de politica de cotejo ya existente")
	ErrRevisionPoliticaCotejoConflicto      = errors.New("vec: revision de politica de cotejo en conflicto")
	ErrSecuenciaPoliticaCotejoInvalida      = errors.New("vec: secuencia de politica de cotejo invalida")
	ErrCodigoCotejoNoEncontrado             = errors.New("vec: codigo de cotejo no encontrado")
	ErrCodigoCotejoYaExiste                 = errors.New("vec: codigo de cotejo ya existente")
	ErrDocumentoConCodigoCotejo             = errors.New("vec: version documental ya vinculada a un codigo de cotejo")
	ErrIndiceCodigoCotejoYaExiste           = errors.New("vec: indice de codigo de cotejo ya existente")
	ErrIndicesCodigoCotejoAmbiguos          = errors.New("vec: varios codigos coinciden con los indices de cotejo")
	ErrRevisionCodigoCotejoConflicto        = errors.New("vec: revision de codigo de cotejo en conflicto")
	ErrClaveIdempotenciaCotejoInvalida      = errors.New("vec: clave de idempotencia de cotejo invalida")
	ErrClaveIdempotenciaCotejoReutilizada   = errors.New("vec: clave de idempotencia de cotejo reutilizada")
	ErrEmisionCodigoCotejoEnCurso           = errors.New("vec: emision de codigo de cotejo en curso")
	ErrReservaCodigoCotejoNoValida          = errors.New("vec: reserva de codigo de cotejo no valida")
	ErrMaterialCodigoCotejoInvalido         = errors.New("vec: material de codigo de cotejo invalido")
	ErrValorCodigoCotejoNoDisponible        = errors.New("vec: valor protegido del codigo de cotejo no disponible")
	ErrSerializacionCodigoCotejoProhibida   = errors.New("vec: serializacion del secreto de cotejo prohibida")
	ErrEvidenciaEmisionNoEncontrada         = errors.New("vec: evidencia de emision no encontrada")
	ErrContextoCustodiaCotejoInvalido       = errors.New("vec: contexto de custodia de codigo de cotejo invalido")
	ErrSerializacionContextoCotejoProhibida = errors.New("vec: serializacion del contexto de custodia de cotejo prohibida")
)

const (
	AccionProtegerCodigoCotejo         = "vec.documentos.cotejo.custodia.proteger"
	AccionRecuperarCodigoCotejo        = "vec.documentos.cotejo.custodia.recuperar"
	AccionEliminarCodigoCotejoHuerfano = "vec.documentos.cotejo.custodia.eliminar_huerfano"

	tipoRecursoCustodiaCodigoCotejo = "codigo_cotejo"
)

// CatalogoPoliticasCotejo solo permite leer versiones concretas. El nucleo no
// selecciona implicitamente «la ultima», porque eso cambiaria el significado
// de un documento ya emitido.
type CatalogoPoliticasCotejo interface {
	ObtenerPoliticaCotejo(context.Context, string, int) (domain.PoliticaCotejo, error)
	ListarVersionesPoliticaCotejo(context.Context, string) ([]domain.PoliticaCotejo, error)
}

// RepositorioGobiernoPoliticasCotejo confirma estado, auditoria y outbox en
// una unica transaccion. La huella anterior actua como control optimista.
type RepositorioGobiernoPoliticasCotejo interface {
	ConfirmarAltaBorradorPoliticaCotejo(context.Context, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarActualizacionBorradorPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaPoliticaCotejo(context.Context, string, domain.PoliticaCotejo, domain.AuditEntry, domain.Event) error
}

// SolicitudReservarEmisionCodigoCotejo fija el significado de una clave de
// idempotencia sin guardar los datos de la orden. La huella debe ser HMAC con
// una clave distinta de la empleada para indexar el CSV.
type SolicitudReservarEmisionCodigoCotejo struct {
	ClaveIdempotencia   string
	PrincipalID         string
	HuellaSolicitudHMAC string
	Documento           domain.ReferenciaDocumento
	Politica            domain.ReferenciaPoliticaCotejo
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

// ReservaEmisionCodigoCotejo devuelve un token nuevo o el agregado confirmado
// anteriormente. Incluso en una repeticion el valor visible se recupera por
// el protector; este contrato nunca lo persiste.
type ReservaEmisionCodigoCotejo struct {
	Token    string
	Repetida bool
	Codigo   domain.CodigoCotejo
}

// RepositorioCodigosCotejo mantiene tres unicidades permanentes: identificador
// interno, indice HMAC del CSV y un codigo por version documental. Todas las
// mutaciones confirman agregado, auditoria y outbox de forma atomica.
type RepositorioCodigosCotejo interface {
	ReservarEmisionCodigoCotejo(context.Context, SolicitudReservarEmisionCodigoCotejo) (ReservaEmisionCodigoCotejo, error)
	ConfirmarReservaCodigoCotejo(context.Context, string, string, time.Time, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	AbandonarReservaCodigoCotejo(context.Context, string) error
	ObtenerCodigoCotejo(context.Context, string) (domain.CodigoCotejo, error)
	ObtenerCodigoCotejoPorDocumento(context.Context, domain.ReferenciaDocumento) (domain.CodigoCotejo, error)
	BuscarCodigoCotejoPorIndices(context.Context, []string) (domain.CodigoCotejo, error)
	ConfirmarActivacionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	ConfirmarSustitucionCodigoCotejo(context.Context, string, domain.CodigoCotejo, domain.AuditEntry, domain.Event) error
	RegistrarConsultaCotejo(context.Context, domain.AuditEntry, domain.Event) error
}

// SecretoCodigoCotejo impide que el CSV acabe accidentalmente en JSON, texto,
// trazas o mensajes de formato. Revelar es la unica salida deliberada para el
// renderizador del sello/QR o para el DTO inicial estrictamente autorizado.
type SecretoCodigoCotejo struct {
	valor string
}

func NuevoSecretoCodigoCotejo(valor string) (SecretoCodigoCotejo, error) {
	canonico, err := domain.NormalizarValorCodigoCotejo(valor)
	if err != nil {
		return SecretoCodigoCotejo{}, ErrMaterialCodigoCotejoInvalido
	}
	return SecretoCodigoCotejo{valor: canonico}, nil
}

func (s SecretoCodigoCotejo) Validar() error {
	canonico, err := domain.NormalizarValorCodigoCotejo(s.valor)
	if err != nil || canonico != s.valor {
		return ErrMaterialCodigoCotejoInvalido
	}
	return nil
}

func (s SecretoCodigoCotejo) Revelar() string { return s.valor }

func (SecretoCodigoCotejo) String() string   { return "[CODIGO-COTEJO-OCULTO]" }
func (SecretoCodigoCotejo) GoString() string { return "ports.SecretoCodigoCotejo{[OCULTO]}" }

func (s SecretoCodigoCotejo) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SecretoCodigoCotejo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCodigoCotejoProhibida
}

func (SecretoCodigoCotejo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionCodigoCotejoProhibida
}

type ValorCodigoCotejoGenerado struct {
	Secreto          SecretoCodigoCotejo
	EntropiaBits     int
	VersionGenerador string
}

// GeneradorValorCodigoCotejo debe usar un CSPRNG y al menos 128 bits de
// entropia. El caso de uso vuelve a validar alfabeto, longitud y metadatos.
type GeneradorValorCodigoCotejo interface {
	GenerarValorCodigoCotejo(context.Context) (ValorCodigoCotejoGenerado, error)
}

type GeneradorIDCodigoCotejo interface {
	NuevoIDCodigoCotejo() (string, error)
}

// SelladorIndiceCodigoCotejo produce un indice determinista HMAC para buscar
// un CSV sin conservarlo ni permitir ataques de diccionario sin la clave.
type SelladorIndiceCodigoCotejo interface {
	SellarIndiceCodigoCotejo(context.Context, SecretoCodigoCotejo) (string, error)
	SellarIndicesConsultaCodigoCotejo(context.Context, SecretoCodigoCotejo) ([]string, error)
}

// SelladorSolicitudCotejo tiene una clave y ciclo de rotacion independientes
// del indice; evita que la idempotencia se convierta en un segundo indice.
type SelladorSolicitudCotejo interface {
	SellarSolicitudCotejo(context.Context, []byte) (string, error)
}

type datosContextoCustodiaCodigoCotejo struct {
	accion         string
	codigoRef      string
	clasificacion  string
	decisionRef    string
	finalidad      string
	correlacionRef string
	recursoHuella  string
	evidencia      EvidenciaUsoDecisionAutorizacion
}

// Los tres contextos son capacidades opacas e incompatibles. El valor cero es
// invalido y una decision concedida para reservar nunca puede rellenarlos ni
// reinterpretarse como permiso de custodia.
type ContextoProtegerCodigoCotejo struct {
	datos             *datosContextoCustodiaCodigoCotejo
	claveIdempotencia string
	indiceCodigoHMAC  string
}

type ContextoRecuperarCodigoCotejo struct {
	datos                    *datosContextoCustodiaCodigoCotejo
	proteccionRef            string
	indiceCodigoHMACEsperado string
}

type ContextoEliminarCodigoCotejoHuerfano struct {
	datos         *datosContextoCustodiaCodigoCotejo
	proteccionRef string
	evidenciaRef  string
	motivo        string
}

func NuevoContextoProtegerCodigoCotejo(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, claveIdempotencia, indiceCodigoHMAC string,
	verificadaEn time.Time,
) (ContextoProtegerCodigoCotejo, error) {
	datos, err := nuevoContextoCustodiaCodigoCotejo(
		decision, recurso, AccionProtegerCodigoCotejo, codigoRef, clasificacion, verificadaEn, false,
	)
	if err != nil || !textoContextoCotejoValido(claveIdempotencia, 512, false) ||
		!huellaHMACContextoCotejoValida(indiceCodigoHMAC) ||
		recurso.Atributos["clave_idempotencia"] != claveIdempotencia ||
		recurso.Atributos["indice_codigo_hmac"] != indiceCodigoHMAC {
		return ContextoProtegerCodigoCotejo{}, ErrContextoCustodiaCotejoInvalido
	}
	return ContextoProtegerCodigoCotejo{
		datos: datos, claveIdempotencia: claveIdempotencia, indiceCodigoHMAC: indiceCodigoHMAC,
	}, nil
}

func NuevoContextoRecuperarCodigoCotejo(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, proteccionRef, indiceCodigoHMACEsperado string,
	verificadaEn time.Time,
) (ContextoRecuperarCodigoCotejo, error) {
	datos, err := nuevoContextoCustodiaCodigoCotejo(
		decision, recurso, AccionRecuperarCodigoCotejo, codigoRef, clasificacion, verificadaEn, false,
	)
	if err != nil || !textoContextoCotejoValido(proteccionRef, 512, false) ||
		!huellaHMACContextoCotejoValida(indiceCodigoHMACEsperado) ||
		recurso.Atributos["proteccion_ref"] != proteccionRef ||
		recurso.Atributos["indice_codigo_hmac"] != indiceCodigoHMACEsperado {
		return ContextoRecuperarCodigoCotejo{}, ErrContextoCustodiaCotejoInvalido
	}
	return ContextoRecuperarCodigoCotejo{
		datos: datos, proteccionRef: proteccionRef, indiceCodigoHMACEsperado: indiceCodigoHMACEsperado,
	}, nil
}

// NuevoContextoEliminarCodigoCotejoHuerfano exige una decision distinta del
// PDP y, ademas, una cuenta privilegiada sobre la superficie administrativa.
// El flujo interactivo de emision no dispone de esa autoridad: la futura
// limpieza debe ejecutarla un worker tecnico interno expresamente cableado.
func NuevoContextoEliminarCodigoCotejoHuerfano(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	codigoRef, clasificacion, proteccionRef, evidenciaRef, motivo string,
	verificadaEn time.Time,
) (ContextoEliminarCodigoCotejoHuerfano, error) {
	datos, err := nuevoContextoCustodiaCodigoCotejo(
		decision, recurso, AccionEliminarCodigoCotejoHuerfano, codigoRef, clasificacion, verificadaEn, true,
	)
	if err != nil || !textoContextoCotejoValido(proteccionRef, 512, false) ||
		!textoContextoCotejoValido(evidenciaRef, 512, false) || !textoContextoCotejoValido(motivo, 512, true) ||
		recurso.Atributos["proteccion_ref"] != proteccionRef || recurso.Atributos["evidencia_ref"] != evidenciaRef ||
		recurso.Atributos["motivo"] != motivo {
		return ContextoEliminarCodigoCotejoHuerfano{}, ErrContextoCustodiaCotejoInvalido
	}
	return ContextoEliminarCodigoCotejoHuerfano{
		datos: datos, proteccionRef: proteccionRef, evidenciaRef: evidenciaRef, motivo: motivo,
	}, nil
}

func nuevoContextoCustodiaCodigoCotejo(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	accion, codigoRef, clasificacion string,
	verificadaEn time.Time,
	requiereAutoridadTecnica bool,
) (*datosContextoCustodiaCodigoCotejo, error) {
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || recurso.Tipo != tipoRecursoCustodiaCodigoCotejo || recurso.Referencia != codigoRef ||
		recurso.Ambitos["clasificacion"] != clasificacion || recurso.Atributos["operacion_custodia"] != accion ||
		!textoContextoCotejoValido(codigoRef, 512, false) || !textoContextoCotejoValido(clasificacion, 128, false) ||
		decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida || decision.Accion != accion ||
		decision.RecursoRef != recurso.Referencia || decision.ModuloID != recurso.ModuloID ||
		decision.TipoRecurso != recurso.Tipo || decision.ContextoRecursoHuellaSHA256 != huellaRecurso ||
		len(decision.CamposPermitidos) != 0 || len(decision.Obligaciones) != 0 {
		return nil, ErrContextoCustodiaCotejoInvalido
	}
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil || (requiereAutoridadTecnica && (!vinculo.CuentaPrivilegiada ||
		vinculo.Superficie != domain.SuperficieAutenticacionAdministracionPrivilegiadaV1)) {
		return nil, ErrContextoCustodiaCotejoInvalido
	}
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		return nil, ErrContextoCustodiaCotejoInvalido
	}
	return &datosContextoCustodiaCodigoCotejo{
		accion: accion, codigoRef: codigoRef, clasificacion: clasificacion,
		decisionRef: decision.DecisionRef, finalidad: decision.Finalidad,
		correlacionRef: decision.CorrelacionRef, recursoHuella: huellaRecurso, evidencia: evidencia,
	}, nil
}

func (c *datosContextoCustodiaCodigoCotejo) validarEn(accion string, instante time.Time) error {
	if c == nil || c.accion != accion || !textoContextoCotejoValido(c.codigoRef, 512, false) ||
		!textoContextoCotejoValido(c.clasificacion, 128, false) ||
		!textoContextoCotejoValido(c.decisionRef, 512, false) ||
		!textoContextoCotejoValido(c.finalidad, 512, false) ||
		!textoContextoCotejoValido(c.correlacionRef, 512, false) || len(c.recursoHuella) != 64 ||
		c.evidencia.ValidarEn(instante) != nil {
		return ErrContextoCustodiaCotejoInvalido
	}
	datos, err := c.evidencia.Datos()
	if err != nil || datos.Decision.DecisionRef != c.decisionRef || datos.Decision.Accion != accion ||
		datos.Decision.RecursoRef != c.codigoRef || datos.Decision.Finalidad != c.finalidad ||
		datos.Decision.CorrelacionRef != c.correlacionRef ||
		datos.Decision.ContextoRecursoHuellaSHA256 != c.recursoHuella {
		return ErrContextoCustodiaCotejoInvalido
	}
	return nil
}

type SolicitudProtegerCodigoCotejo struct {
	Contexto          ContextoProtegerCodigoCotejo
	ClaveIdempotencia string
	Secreto           SecretoCodigoCotejo
	IndiceCodigoHMAC  string
}

type SolicitudRecuperarCodigoCotejo struct {
	Contexto                 ContextoRecuperarCodigoCotejo
	ProteccionRef            string
	IndiceCodigoHMACEsperado string
}

type SolicitudEliminarCodigoCotejoHuerfano struct {
	Contexto      ContextoEliminarCodigoCotejoHuerfano
	ProteccionRef string
	EvidenciaRef  string
	Motivo        string
}

func (s SolicitudProtegerCodigoCotejo) ValidarEn(instante time.Time) error {
	if s.Contexto.datos == nil || s.Contexto.datos.validarEn(AccionProtegerCodigoCotejo, instante) != nil ||
		s.ClaveIdempotencia != s.Contexto.claveIdempotencia || s.IndiceCodigoHMAC != s.Contexto.indiceCodigoHMAC ||
		s.Secreto.Validar() != nil {
		return ErrContextoCustodiaCotejoInvalido
	}
	return nil
}

func (s SolicitudRecuperarCodigoCotejo) ValidarEn(instante time.Time) error {
	if s.Contexto.datos == nil || s.Contexto.datos.validarEn(AccionRecuperarCodigoCotejo, instante) != nil ||
		s.ProteccionRef != s.Contexto.proteccionRef ||
		s.IndiceCodigoHMACEsperado != s.Contexto.indiceCodigoHMACEsperado {
		return ErrContextoCustodiaCotejoInvalido
	}
	return nil
}

func (s SolicitudEliminarCodigoCotejoHuerfano) ValidarEn(instante time.Time) error {
	if s.Contexto.datos == nil || s.Contexto.datos.validarEn(AccionEliminarCodigoCotejoHuerfano, instante) != nil ||
		s.ProteccionRef != s.Contexto.proteccionRef || s.EvidenciaRef != s.Contexto.evidenciaRef ||
		s.Motivo != s.Contexto.motivo {
		return ErrContextoCustodiaCotejoInvalido
	}
	return nil
}

func (ContextoProtegerCodigoCotejo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}
func (ContextoRecuperarCodigoCotejo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}
func (ContextoEliminarCodigoCotejoHuerfano) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}

func (ContextoProtegerCodigoCotejo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}
func (ContextoRecuperarCodigoCotejo) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}
func (ContextoEliminarCodigoCotejoHuerfano) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionContextoCotejoProhibida
}

func (ContextoProtegerCodigoCotejo) String() string {
	return "[CONTEXTO-PROTEGER-COTEJO-OPACO]"
}
func (ContextoRecuperarCodigoCotejo) String() string {
	return "[CONTEXTO-RECUPERAR-COTEJO-OPACO]"
}
func (ContextoEliminarCodigoCotejoHuerfano) String() string {
	return "[CONTEXTO-ELIMINAR-COTEJO-HUERFANO-OPACO]"
}

func textoContextoCotejoValido(valor string, maximo int, permiteEspacios bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo || !utf8.ValidString(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || (!permiteEspacios && unicode.IsSpace(caracter)) ||
			(permiteEspacios && unicode.IsSpace(caracter) && caracter != ' ') {
			return false
		}
	}
	return true
}

func huellaHMACContextoCotejoValida(valor string) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || !textoContextoCotejoValido(partes[1], 128, false) || len(partes[2]) != 64 {
		return false
	}
	for _, caracter := range partes[2] {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

type CustodiaCodigoCotejo struct {
	ProteccionRef string
	ConectorID    string
	EvidenciaRef  string
}

type RecuperacionCodigoCotejo struct {
	Secreto      SecretoCodigoCotejo
	ConectorID   string
	EvidenciaRef string
}

// ProtectorCodigoCotejo representa Vault, KMS/HSM o un servicio equivalente.
// Las implementaciones productivas deben cifrar con autenticacion, versionar
// claves, llamar ValidarEn con reloj fiable justo antes de cada operacion y
// eliminar solo referencias huerfanas con autoridad tecnica expresa.
type ProtectorCodigoCotejo interface {
	ProtegerCodigoCotejo(context.Context, SolicitudProtegerCodigoCotejo) (CustodiaCodigoCotejo, error)
	RecuperarCodigoCotejo(context.Context, SolicitudRecuperarCodigoCotejo) (RecuperacionCodigoCotejo, error)
	EliminarCodigoCotejoHuerfano(context.Context, SolicitudEliminarCodigoCotejoHuerfano) error
}

type SolicitudObtenerEvidenciaEmisionDocumento struct {
	Documento        domain.ReferenciaDocumento
	RepresentacionID string
	SolicitanteID    string
	AutorizacionRef  string
	Finalidad        string
	CorrelacionRef   string
}

// FuenteEvidenciaEmisionDocumento consulta exclusivamente fuentes internas
// confiables (firma, validacion, registro y almacenamiento). Nunca acepta
// huellas o referencias declaradas por el cliente HTTP.
type FuenteEvidenciaEmisionDocumento interface {
	ObtenerEvidenciaEmisionDocumento(context.Context, SolicitudObtenerEvidenciaEmisionDocumento) (domain.EvidenciaEmisionDocumento, error)
}
