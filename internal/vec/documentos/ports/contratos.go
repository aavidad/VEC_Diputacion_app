package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrSolicitudInvalida = errors.New("documentos: solicitud invalida")
var ErrCapacidadNoDisponible = errors.New("documentos: capacidad no disponible")
var ErrOriginalNoDisponible = errors.New("documentos: original no disponible")

const (
	AccionAlta                 = "documentos.generado.alta"
	AccionListar               = "documentos.expediente.listar"
	AccionDescargar            = "documentos.original.descargar"
	AccionPrepararNotificacion = "documentos.notificacion.preparar"
	AccionRegistrarExterno     = "documentos.externo.registrar"
	AudienciaV3                = "vec_documentos.operacion.v1"
)

// AutorizacionV3 transporta el material nominal al adaptador duradero. Su
// estructura no concede permiso: el adaptador debe consumirlo en AD3 dentro
// de la misma transaccion que metadatos, auditoria y outbox.
type AutorizacionV3 struct {
	Material        vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Accion          string
	Finalidad       string
	RecursoRef      string
	AmbitoRef       string
	PrincipalID     string
	PerfilActivoRef string
	CorrelacionRef  string
}

func (a AutorizacionV3) ValidarPara(accion string, ahora time.Time) error {
	if a.Material.ValidarEstructura() != nil || a.Accion != accion ||
		!domain.ReferenciaOpacaValida(a.RecursoRef) ||
		!domain.ReferenciaOpacaValida(a.AmbitoRef) ||
		!domain.ReferenciaValida(a.PrincipalID) ||
		!domain.ReferenciaValida(a.PerfilActivoRef) ||
		!domain.ReferenciaValida(a.CorrelacionRef) ||
		ahora.IsZero() {
		return ErrSolicitudInvalida
	}
	resumen := a.Material.ResumenCapacidad()
	if resumen.ValidarEstructura() != nil ||
		resumen.AudienciaConsumo() != AudienciaV3 ||
		resumen.EfectoRef() != a.RecursoRef ||
		ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ErrSolicitudInvalida
	}
	switch accion {
	case AccionAlta:
		if a.Finalidad != "alta_documento_generado" {
			return ErrSolicitudInvalida
		}
	case AccionListar:
		if a.Finalidad != "listar_documentos_expediente" {
			return ErrSolicitudInvalida
		}
	case AccionDescargar:
		if a.Finalidad != "descargar_documento_original" {
			return ErrSolicitudInvalida
		}
	case AccionPrepararNotificacion:
		if a.Finalidad != "preparar_notificacion" {
			return ErrSolicitudInvalida
		}
	case AccionRegistrarExterno:
		if a.Finalidad != "registrar_documento_externo" {
			return ErrSolicitudInvalida
		}
	default:
		return ErrSolicitudInvalida
	}
	return nil
}

// AltaGenerado requiere contexto opaco de AlmacenObjetos derivado del
// manifiesto y plantilla publicados. Un llamante no puede fabricarlo.
type AltaGenerado struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	Contenido                                               []byte
	ContextoAlmacen                                         vecports.ContextoOperacionAlmacen
	SolicitudPolitica                                       vecports.SolicitudPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

// AltaPersistente se confirma despues de validar el recibo del almacen.
// Ningun adaptador debe aceptar solo la referencia declarada del objeto.
type AltaPersistente struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	HuellaSHA256                                            string
	Tamano                                                  int64
	Objeto                                                  vecports.ResultadoOperacionObjeto
	Politica                                                vecports.ResultadoPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

// AltaExterna registra un original que custodia otro sistema. No tiene campo
// de contenido: el tipo impide subir bytes cuando la custodia es externa.
// MIME y Tamano son opcionales ("" y 0 significan no declarados).
type AltaExterna struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	Tamano                                                  int64
	Custodia                                                domain.ReferenciaCustodiaExterna
	SolicitudPolitica                                       vecports.SolicitudPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

// AltaExternaPersistente lleva ya resuelta la politica de conservacion.
type AltaExternaPersistente struct {
	ID, ClaveIdempotencia, ModuloID, ExpedienteRef, TipoRef string
	Version                                                 uint64
	MIME                                                    string
	Tamano                                                  int64
	Custodia                                                domain.ReferenciaCustodiaExterna
	Politica                                                vecports.ResultadoPoliticaConservacionDocumental
	Autorizacion                                            AutorizacionV3
}

// PreimagenExterna fija los campos que autoriza V3 y coteja la fachada SQL.
func (a AltaExternaPersistente) PreimagenExterna() ([]byte, error) {
	if a.Politica.Validar() != nil || a.Custodia.Validar() != nil ||
		!domain.ReferenciaOpacaValida(a.ID) || !domain.ReferenciaOpacaValida(a.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(a.ModuloID) || !domain.ReferenciaOpacaValida(a.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(a.TipoRef) || a.Version == 0 || a.Tamano < 0 ||
		(a.MIME != "" && !domain.MIMEValido(a.MIME)) {
		return nil, ErrSolicitudInvalida
	}
	p := a.Politica.Politica()
	s := p.Solicitud()
	return json.Marshal(struct {
		Accion            string `json:"accion"`
		ID                string `json:"id"`
		Clave             string `json:"clave_idempotencia"`
		Modulo            string `json:"modulo_id"`
		Expediente        string `json:"expediente_ref"`
		Tipo              string `json:"tipo_ref"`
		Version           uint64 `json:"version"`
		MIME              string `json:"mime"`
		Tamano            int64  `json:"tamano"`
		Huella            string `json:"huella_sha256"`
		Custodio          string `json:"custodio_id"`
		CustodiaRef       string `json:"custodia_ref"`
		Politica          string `json:"politica_ref"`
		VersionPolitica   uint64 `json:"version_politica"`
		HuellaPolitica    string `json:"huella_politica_sha256"`
		Proteccion        string `json:"proteccion"`
		ConservacionHasta string `json:"conservacion_hasta"`
	}{AccionRegistrarExterno, a.ID, a.ClaveIdempotencia, a.ModuloID, a.ExpedienteRef,
		a.TipoRef, a.Version, a.MIME, a.Tamano, a.Custodia.HuellaSHA256,
		a.Custodia.CustodioID, a.Custodia.Referencia,
		s.PoliticaRef(), s.VersionPolitica(), hex.EncodeToString(s.HuellaPoliticaSHA256()),
		string(p.Proteccion()), p.ConservacionHasta().UTC().Format(time.RFC3339Nano)})
}

type ConsultaExpediente struct {
	ExpedienteRef string
	Cursor        string
	Limite        uint32
	Autorizacion  AutorizacionV3
}

type PaginaDocumentos struct {
	Items           []domain.Documento
	SiguienteCursor string
}

type ConsultaDocumento struct {
	DocumentoID  string
	Version      uint64
	Autorizacion AutorizacionV3
}

// FabricaContextoLectura recibe el objeto solo tras la consulta SQL V3.
// Debe resolver una decision de almacen positiva propia para ese objeto y
// usar NuevoContextoLeerDocumentoGeneradoAlmacen; no admite objeto generico.
type FabricaContextoLectura interface {
	ContextoLecturaOriginal(context.Context, domain.Documento, AutorizacionV3) (vecports.ContextoOperacionAlmacen, error)
}

type Original struct {
	Contenido    []byte
	MIME         string
	HuellaSHA256 string
}

type PreparacionNotificacion struct {
	ID, ClaveIdempotencia, DocumentoID, DestinatarioRef, Canal string
	Version                                                    uint64
	Autorizacion                                               AutorizacionV3
}

type Repositorio interface {
	ConfirmarAlta(context.Context, AltaPersistente) (domain.Documento, error)
	ListarExpediente(context.Context, ConsultaExpediente) (PaginaDocumentos, error)
	Obtener(context.Context, ConsultaDocumento) (domain.Documento, error)
	ConfirmarPreparacion(context.Context, PreparacionNotificacion) (domain.NotificacionPreparada, error)
	ConfirmarReferenciaExterna(context.Context, AltaExternaPersistente) (domain.Documento, error)
}

func (q ConsultaExpediente) PreimagenListar() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(q.ExpedienteRef) || q.Limite == 0 || q.Limite > 100 ||
		(q.Cursor != "" && !domain.ReferenciaValida(q.Cursor)) {
		return nil, ErrSolicitudInvalida
	}
	return json.Marshal(struct {
		Accion     string `json:"accion"`
		Expediente string `json:"expediente_ref"`
		Cursor     string `json:"cursor"`
		Limite     uint32 `json:"limite"`
	}{AccionListar, q.ExpedienteRef, q.Cursor, q.Limite})
}

func (q ConsultaDocumento) PreimagenDescargar() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(q.DocumentoID) || q.Version == 0 {
		return nil, ErrSolicitudInvalida
	}
	return json.Marshal(struct {
		Accion    string `json:"accion"`
		Documento string `json:"documento_id"`
		Version   uint64 `json:"version"`
	}{AccionDescargar, q.DocumentoID, q.Version})
}

func (p PreparacionNotificacion) PreimagenPreparar() ([]byte, error) {
	if !domain.ReferenciaOpacaValida(p.ID) || !domain.ReferenciaOpacaValida(p.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(p.DocumentoID) || !domain.ReferenciaOpacaValida(p.DestinatarioRef) ||
		!domain.IdentificadorTecnicoValido(p.Canal) || p.Version == 0 {
		return nil, ErrSolicitudInvalida
	}
	return json.Marshal(struct {
		Accion       string `json:"accion"`
		ID           string `json:"id"`
		Clave        string `json:"clave_idempotencia"`
		Documento    string `json:"documento_id"`
		Version      uint64 `json:"version"`
		Destinatario string `json:"destinatario_ref"`
		Canal        string `json:"canal"`
	}{AccionPrepararNotificacion, p.ID, p.ClaveIdempotencia, p.DocumentoID, p.Version, p.DestinatarioRef, p.Canal})
}

// PreimagenAlta evita autorizar un conjunto de campos y confirmar otro.
// El adaptador SQL coteja los campos y la huella exacta con AD3.
func (a AltaPersistente) PreimagenAlta() ([]byte, error) {
	if a.Politica.Validar() != nil || !domain.HuellaValida(a.HuellaSHA256) ||
		!domain.ReferenciaOpacaValida(a.ID) || !domain.ReferenciaOpacaValida(a.ClaveIdempotencia) ||
		!domain.IdentificadorTecnicoValido(a.ModuloID) || !domain.ReferenciaOpacaValida(a.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(a.TipoRef) || a.Version == 0 || a.Tamano < 1 {
		return nil, ErrSolicitudInvalida
	}
	p := a.Politica.Politica()
	s := p.Solicitud()
	return json.Marshal(struct {
		Accion            string `json:"accion"`
		ID                string `json:"id"`
		Clave             string `json:"clave_idempotencia"`
		Modulo            string `json:"modulo_id"`
		Expediente        string `json:"expediente_ref"`
		Tipo              string `json:"tipo_ref"`
		Version           uint64 `json:"version"`
		MIME              string `json:"mime"`
		Tamano            int64  `json:"tamano"`
		Huella            string `json:"huella_sha256"`
		Politica          string `json:"politica_ref"`
		VersionPolitica   uint64 `json:"version_politica"`
		HuellaPolitica    string `json:"huella_politica_sha256"`
		Proteccion        string `json:"proteccion"`
		ConservacionHasta string `json:"conservacion_hasta"`
	}{AccionAlta, a.ID, a.ClaveIdempotencia, a.ModuloID, a.ExpedienteRef,
		a.TipoRef, a.Version, a.MIME, a.Tamano, a.HuellaSHA256,
		s.PoliticaRef(), s.VersionPolitica(), hex.EncodeToString(s.HuellaPoliticaSHA256()),
		string(p.Proteccion()), p.ConservacionHasta().UTC().Format(time.RFC3339Nano)})
}

func HuellaPreimagen(preimagen []byte) string {
	suma := sha256.Sum256(preimagen)
	return hex.EncodeToString(suma[:])
}
