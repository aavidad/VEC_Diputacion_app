package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrPlantillaDocumentoNoEncontrada = errors.New("vec: plantilla documental no encontrada")
	ErrVersionPlantillaYaExiste       = errors.New("vec: la version de plantilla ya existe")
	ErrDocumentoNoEncontrado          = errors.New("vec: documento no encontrado")
	ErrDocumentoYaExiste              = errors.New("vec: documento ya existe")
	ErrContenidoDocumentoNoEncontrado = errors.New("vec: contenido documental no encontrado")
	ErrHuellaContenidoNoCoincide      = errors.New("vec: la huella del contenido no coincide")
	ErrLimiteLecturaExcedido          = errors.New("vec: limite de lectura documental excedido")
	ErrDocumentoLogicoNoEncontrado    = errors.New("vec: documento logico no encontrado")
	ErrRepresentacionNoEncontrada     = errors.New("vec: representacion documental no encontrada")
	ErrClaveIdempotenciaInvalida      = errors.New("vec: clave de idempotencia documental invalida")
	ErrClaveIdempotenciaReutilizada   = errors.New("vec: clave de idempotencia reutilizada para otra solicitud")
	ErrGeneracionDocumentoEnCurso     = errors.New("vec: generacion documental en curso")
	ErrReservaDocumentoNoValida       = errors.New("vec: reserva documental no valida")
	ErrDecisionAutorizacionConsumida  = errors.New("vec: decision de autorizacion ya consumida por otro efecto")
)

// CatalogoPlantillasDocumento es deliberadamente de solo lectura. Toda alta o
// transicion pasa por el repositorio de gobierno con autorizacion, auditoria y
// outbox; ninguna importacion puede insertar una publicada por esta via.
type CatalogoPlantillasDocumento interface {
	ObtenerPlantilla(context.Context, string, int) (domain.PlantillaDocumento, error)
	ListarPlantillas(context.Context, string) ([]domain.PlantillaDocumento, error)
}

// RepositorioGobiernoPlantillasDocumento confirma cada alta o publicacion con
// su evidencia y outbox. Una interfaz separada evita que el caso de uso pueda
// sobrescribir una plantilla publicada mediante el catalogo de consulta.
type RepositorioGobiernoPlantillasDocumento interface {
	ConfirmarAltaBorradorPlantilla(context.Context, domain.PlantillaDocumento, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionPlantilla(context.Context, string, domain.PlantillaDocumento, domain.AuditEntry, domain.Event) error
}

type SolicitudGuardarContenido struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	DocumentoID       string
	Zona              ZonaAlmacen
	MIME              string
	HuellaSHA256      string
	Tamano            int64
	Contenido         []byte
}

// Validar exige que la solicitud coincida byte a byte con el paso actualmente
// seleccionado del manifiesto autorizado. Un contexto de escritura generico,
// otro paso del mismo manifiesto o metadatos parcialmente coincidentes se
// deniegan; no existe compatibilidad permisiva para la generacion documental.
func (s SolicitudGuardarContenido) Validar() error {
	if !referenciaOpacaAlmacenValida(s.DocumentoID, 512) ||
		!referenciaOpacaAlmacenValida(s.ClaveIdempotencia, 512) || !s.Zona.Valida() ||
		!textoSeguroAlmacen(s.MIME, 255) || s.Tamano < 1 ||
		int64(len(s.Contenido)) != s.Tamano || !esSHA256Hexadecimal(s.HuellaSHA256) ||
		contieneComodinContextoAlmacen(s.DocumentoID, s.ClaveIdempotencia, s.MIME) {
		return ErrSolicitudAlmacenInvalida
	}
	suma := sha256.Sum256(s.Contenido)
	if hex.EncodeToString(suma[:]) != s.HuellaSHA256 ||
		s.Contexto.validarPasoGeneracionDocumental(
			s.DocumentoID, s.ClaveIdempotencia, s.Zona, s.MIME, s.Tamano, s.HuellaSHA256,
		) != nil {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ValidarEn debe ejecutarse inmediatamente antes del efecto remoto. Ademas
// del manifiesto exacto, revalida la vigencia temporal de la decision.
func (s SolicitudGuardarContenido) ValidarEn(instante time.Time) error {
	if s.Validar() != nil || s.Contexto.ValidarParaEn(AccionAlmacenEscribir, instante) != nil {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type ContenidoDocumentoGuardado struct {
	ReferenciaLogica   string
	Referencia         string
	Version            string
	ConectorID         string
	Zona               ZonaAlmacen
	MIME               string
	HuellaSHA256       string
	Tamano             int64
	EvidenciaOperacion EvidenciaOperacionAlmacen
}

func (g ContenidoDocumentoGuardado) ValidarContra(s SolicitudGuardarContenido) error {
	objeto := ReferenciaObjetoAlmacen{Referencia: g.Referencia, Version: g.Version}
	if s.Validar() != nil || objeto.Validar() != nil || g.ReferenciaLogica != s.DocumentoID ||
		!referenciaOpacaAlmacenValida(g.ConectorID, 128) || g.Zona != s.Zona ||
		g.MIME != s.MIME || g.HuellaSHA256 != s.HuellaSHA256 || g.Tamano != s.Tamano ||
		g.EvidenciaOperacion.Validar() != nil || g.EvidenciaOperacion.Objeto != objeto ||
		g.EvidenciaOperacion.ConectorID != g.ConectorID ||
		g.EvidenciaOperacion.Accion != AccionAlmacenEscribir ||
		g.EvidenciaOperacion.FundamentoRef != "" ||
		!evidenciaAlmacenLigada(g.EvidenciaOperacion, s.Contexto) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudLeerContenido struct {
	Contexto   ContextoOperacionAlmacen
	Referencia string
	Zona       ZonaAlmacen
	Limite     int64
}

type ContenidoDocumentoLeido struct {
	Contenido          []byte
	ConectorID         string
	Zona               ZonaAlmacen
	HuellaSHA256       string
	Tamano             int64
	EvidenciaOperacion EvidenciaOperacionAlmacen
}

// AlmacenContenidoDocumento es implementable por S3 compatible, filesystem
// cifrado o un gestor documental. La referencia devuelta debe ser opaca y
// estable; las URL temporales solo se crean al descargar.
type AlmacenContenidoDocumento interface {
	GuardarContenido(context.Context, SolicitudGuardarContenido) (ContenidoDocumentoGuardado, error)
	LeerContenido(context.Context, SolicitudLeerContenido) (ContenidoDocumentoLeido, error)
}

// RepositorioDocumentos conserva el agregado documental historico. No es el
// mecanismo que habilita un efecto remoto: toda generacion nueva debe reservar
// primero mediante RegistroEfectosGeneracionDocumental. La confirmacion de
// metadatos, auditoria y outbox sigue siendo atomica y la evidencia opaca nunca
// autoriza por si sola.
type RepositorioDocumentos interface {
	ConfirmarGeneracion(context.Context, domain.DocumentoGenerado, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ObtenerDocumento(context.Context, string) (domain.DocumentoGenerado, error)
	ListarDocumentosExpediente(context.Context, string) ([]domain.DocumentoGenerado, error)
}

// SolicitudReservarGeneracionDocumento fija una clave idempotente dentro del
// ambito del principal. La huella HMAC vincula todos los datos con efecto sin
// persistirlos en claro dentro del control de concurrencia.
type SolicitudReservarGeneracionDocumento struct {
	ClaveIdempotencia   string
	PrincipalID         string
	HuellaSolicitudHMAC string
	SolicitadaEn        time.Time
	ExpiraEn            time.Time
}

// ReservaGeneracionDocumento tiene dos resultados excluyentes: una reserva
// nueva con Token, o el resultado confirmado anteriormente con Repetida=true.
type ReservaGeneracionDocumento struct {
	Token     string
	Repetida  bool
	Resultado domain.ResultadoGeneracionDocumento
}

// RepositorioDocumentosLogicos protege la idempotencia funcional y confirma en
// una sola transaccion el agregado, sus representaciones, la auditoria y el
// outbox. No sustituye la reserva tecnica previa de cada efecto remoto.
// En produccion debe implementarse con una restriccion unica por
// principal+clave y bloqueo transaccional o mecanismo equivalente.
type RepositorioDocumentosLogicos interface {
	ReservarGeneracion(context.Context, SolicitudReservarGeneracionDocumento) (ReservaGeneracionDocumento, error)
	ConfirmarGeneracionLogica(context.Context, string, string, time.Time, domain.ResultadoGeneracionDocumento, domain.AuditEntry, domain.Event) error
	AbandonarGeneracion(context.Context, string) error
	ObtenerDocumentoLogico(context.Context, domain.ReferenciaDocumento) (domain.DocumentoLogico, error)
	ListarRepresentacionesDocumento(context.Context, domain.ReferenciaDocumento) ([]domain.RepresentacionDocumento, error)
}

// RenderizadorDocumento mantiene el nucleo ajeno a librerias PDF, DOCX o a
// servicios de conversion externos.
type RenderizadorDocumento interface {
	Formato() domain.FormatoDocumento
	Renderizar(context.Context, domain.ContenidoDocumento) ([]byte, error)
	ValidarSalida(context.Context, []byte) error
}

// SelladorDatosDocumento calcula una huella autenticada (por ejemplo,
// HMAC-SHA-256 con clave en KMS). No debe usarse SHA sin clave para campos de
// baja entropia como DNI, porque permitiria ataques de diccionario.
type SelladorDatosDocumento interface {
	SellarDatos(context.Context, []byte) (string, error)
}

// SelladorSolicitudDocumento usa una clave criptografica separada y estable
// durante la ventana de idempotencia. Separarlo del sellado de datos permite
// rotar una clave sin cambiar silenciosamente el significado de otra.
type SelladorSolicitudDocumento interface {
	SellarSolicitudDocumento(context.Context, []byte) (string, error)
}

type GeneradorIDDocumento interface {
	NuevoIDDocumento() (string, error)
}

type Reloj interface {
	Ahora() time.Time
}
