package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrSubidaLocalPendiente = errors.New("documentos: subida local pendiente")
var ErrSubidaLocalRechazada = errors.New("documentos: subida local rechazada")
var ErrSubidaLocalDenegada = errors.New("documentos: subida local denegada")
var ErrSubidaLocalNoEncontrada = errors.New("documentos: subida local propia no encontrada")
var ErrSubidaLocalNoDisponible = errors.New("documentos: subida local no disponible")

type EstadoSubidaLocal string

const (
	EstadoSubidaLocalReservada  EstadoSubidaLocal = "reservada"
	EstadoSubidaLocalRecibida   EstadoSubidaLocal = "recibido"
	EstadoSubidaLocalAnalizando EstadoSubidaLocal = "analizando"
	EstadoSubidaLocalAdmitida   EstadoSubidaLocal = "admitido"
	EstadoSubidaLocalRechazada  EstadoSubidaLocal = "rechazado"
)

func (e EstadoSubidaLocal) Valido() bool {
	switch e {
	case EstadoSubidaLocalReservada, EstadoSubidaLocalRecibida,
		EstadoSubidaLocalAnalizando, EstadoSubidaLocalAdmitida, EstadoSubidaLocalRechazada:
		return true
	default:
		return false
	}
}

// La petición pública sólo aporta la clave de declaración. Identidad y
// vínculo llegan de la frontera certificada; el recurso dueño se resuelve en
// el módulo, no se acepta desde multipart ni JSON.
type SolicitudSubidaLocal struct {
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	VinculoV2         domain.VinculoAutenticacionActorV2
	ClaveIdempotencia string
}

type SolicitudConsultaLocal struct {
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	VinculoV2         domain.VinculoAutenticacionActorV2
	CargaID           string
	DocumentoID       string
	ClaveIdempotencia string
	CorrelacionRef    string
}

// La identidad de consulta procede de la frontera. La versión solicitada
// identifica el original exacto; la persona y su huella no llegan del cliente.
type SolicitudOriginalSubidaLocalPropio struct {
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	VinculoV2         domain.VinculoAutenticacionActorV2
	DocumentoID       string
	DocumentoVersion  uint64
	CorrelacionRef    string
}

// CustodioID identifica el conector que guardó los bytes admitidos. No
// atribuye una custodia jurídica distinta de la evidencia de Documentos.
type OriginalSubidaLocalAcreditado struct {
	DocumentoID         string            `json:"documento_id"`
	DocumentoVersion    uint64            `json:"documento_version"`
	HuellaSHA256Bytes   string            `json:"huella_sha256_bytes"`
	ReciboRef           string            `json:"recibo_ref"`
	RecursoRef          string            `json:"recurso_ref"`
	PersonaTitularRef   string            `json:"persona_titular_ref"`
	PersonaAportanteRef string            `json:"persona_aportante_ref"`
	CustodioID          string            `json:"custodio_id"`
	ModuloID            string            `json:"modulo_id"`
	TipoRecurso         string            `json:"tipo_recurso"`
	Finalidad           string            `json:"finalidad"`
	EstadoAdmision      EstadoSubidaLocal `json:"estado_admision"`
}

type ConsultaOriginalSubidaLocalPropio interface {
	ConsultarOriginalAdmitido(context.Context, SolicitudOriginalSubidaLocalPropio) (OriginalSubidaLocalAcreditado, error)
}

type ResultadoSubidaLocal struct {
	CargaID          string            `json:"carga_id"`
	RecursoRef       string            `json:"-"`
	ComprobanteRef   string            `json:"comprobante_ref"`
	DocumentoVersion int               `json:"documento_version"`
	ReciboRef        string            `json:"recibo_ref"`
	Estado           EstadoSubidaLocal `json:"estado"`
	CodigoRechazo    string            `json:"codigo_rechazo"`
	RegistradoEn     time.Time         `json:"registrado_en"`
}

type ActoSubidaLocal struct {
	Recurso       domain.RecursoAutorizable
	PersonaRef    string
	Finalidad     string
	Clasificacion string
	VersionDueno  uint64
}

const (
	AccionMeritosComprobanteSubirLocal      = "meritos.cartera.comprobante.subir_propio"
	AccionMeritosComprobanteDescargarLocal  = "meritos.cartera.comprobante.descargar_propio"
	AccionDietasJustificanteSubirLocal      = "dietas.justificante.subir_propio"
	AccionDietasJustificanteDescargarLocal  = "dietas.justificante.descargar_propio"
	AccionCronosJustificacionSubirLocal     = "cronos.justificacion.subir_propio"
	AccionCronosJustificacionDescargarLocal = "cronos.justificacion.descargar_propio"
)

// El módulo y las acciones se fijan al componer el servicio, nunca en HTTP.
func ParejaActoSubidaLocalValida(modulo, tipo, finalidad, subir, descargar string) bool {
	switch {
	case modulo == "meritos" && tipo == "hecho" && finalidad == "custodiar_evidencia_meritos":
		return subir == AccionMeritosComprobanteSubirLocal && descargar == AccionMeritosComprobanteDescargarLocal
	case modulo == "dietas" && tipo == "comision_borrador" && finalidad == "custodiar_justificante_dietas":
		return subir == AccionDietasJustificanteSubirLocal && descargar == AccionDietasJustificanteDescargarLocal
	case modulo == "cronos" && tipo == "justificacion" && finalidad == "custodiar_justificacion_cronos":
		return subir == AccionCronosJustificacionSubirLocal && descargar == AccionCronosJustificacionDescargarLocal
	default:
		return false
	}
}

// El módulo dueño resuelve propiedad y versión en una fuente durable. El
// repositorio de Documentos las vuelve a comprobar en la misma transacción.
type ResolutorActoSubidaLocal interface {
	ResolverActoSubidaLocal(context.Context, SolicitudSubidaLocal) (ActoSubidaLocal, error)
}

type DerivadorCorrelacionSubidaLocal interface {
	CorrelacionSubida(context.Context, string, string) (string, error)
}

// AutorizacionActoSubidaLocalV3 sólo transporta material emitido por el PDP
// nativo del módulo. El SQL consumidor debe cotejar acción, recurso, persona,
// contexto, finalidad, audiencia y la huella de la preimagen exacta.
type AutorizacionActoSubidaLocalV3 struct {
	Accion         string
	CargaID        string
	RecursoRef     string
	Finalidad      string
	CorrelacionRef string
	Material       vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type AutoridadActoSubidaLocalV3 interface {
	AutorizarSubidaLocal(context.Context, []byte, domain.ResultadoContextoActorRegistradoV2,
		string, string, string, string) (AutorizacionActoSubidaLocalV3, error)
	AutorizarAlmacenSubidaLocal(context.Context, domain.ResultadoContextoActorRegistradoV2,
		ActoSubidaLocal, string, string, vecports.FaseAlmacenSubidaLocal,
		vecports.VinculosOperacionAlmacen) (vecports.ContextoOperacionAlmacen, error)
}

// ReservaSubidaLocal se confirma antes de escribir el primer byte en
// cuarentena. Una repetición de clave con otra huella se rechaza en SQL.
type ReservaSubidaLocal struct {
	ActorRegistradoV2   domain.ResultadoContextoActorRegistradoV2
	VinculoV2           domain.VinculoAutenticacionActorV2
	Acto                ActoSubidaLocal
	AccionSubir         string
	AccionDescargar     string
	CargaID             string
	CorrelacionRef      string
	ClaveHMAC           string
	HuellaSolicitudHMAC string
	HuellaPDFSHA256     string
	Tamano              int64
	MIME                string
	AutorizacionV3      AutorizacionActoSubidaLocalV3
}

// Orden de 18 claves fijado con D20 local. El mapa no admite campos libres;
// SHA256 de estos bytes forma el atributo preimagen_sha256 del recurso PDP.
func PreimagenReservaSubidaLocal(q ReservaSubidaLocal) ([]byte, error) {
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.Acto.Recurso.Validar() != nil || q.Acto.PersonaRef != q.ActorRegistradoV2.Contexto.PersonaRef ||
		!ParejaActoSubidaLocalValida(q.Acto.Recurso.ModuloID, q.Acto.Recurso.Tipo, q.Acto.Finalidad,
			q.AccionSubir, q.AccionDescargar) ||
		q.Acto.VersionDueno == 0 || q.CargaID == "" || q.CorrelacionRef == "" ||
		q.ClaveHMAC == "" || q.HuellaSolicitudHMAC == "" || q.HuellaPDFSHA256 == "" ||
		q.Tamano < 1 || q.Tamano > 16<<20 || q.MIME != "application/pdf" {
		return nil, ErrSubidaLocalDenegada
	}
	return json.Marshal(struct {
		Accion               string `json:"accion"`
		Fase                 string `json:"fase"`
		CargaID              string `json:"carga_id"`
		RecursoRef           string `json:"recurso_ref"`
		PersonaRef           string `json:"persona_ref"`
		PerfilActivoRef      string `json:"perfil_activo_ref"`
		ModuloID             string `json:"modulo_id"`
		TipoRecurso          string `json:"tipo_recurso"`
		Finalidad            string `json:"finalidad"`
		ClaveHMAC            string `json:"clave_idempotencia_hmac"`
		HuellaSolicitudHMAC  string `json:"huella_solicitud_hmac"`
		HuellaPDFSHA256      string `json:"huella_pdf_sha256"`
		Tamano               int64  `json:"tamano"`
		MIME                 string `json:"mime"`
		PropietarioVersion   uint64 `json:"propietario_version"`
		RegistroContextoRef  string `json:"registro_contexto_ref"`
		HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
		CorrelacionRef       string `json:"correlacion_ref"`
	}{q.AccionSubir, "reservar", q.CargaID,
		q.Acto.Recurso.Referencia, q.Acto.PersonaRef,
		q.ActorRegistradoV2.Contexto.PerfilActivoRef, q.Acto.Recurso.ModuloID,
		q.Acto.Recurso.Tipo, q.Acto.Finalidad, q.ClaveHMAC,
		q.HuellaSolicitudHMAC, q.HuellaPDFSHA256, q.Tamano, q.MIME,
		q.Acto.VersionDueno, q.ActorRegistradoV2.RegistroContextoRef,
		q.ActorRegistradoV2.HuellaSHA256, q.CorrelacionRef})
}

func HuellaPreimagenSubidaLocal(pre []byte) string {
	suma := sha256.Sum256(pre)
	return hex.EncodeToString(suma[:])
}

type RegistroSubidaLocal interface {
	Reservar(context.Context, ReservaSubidaLocal) (InstantaneaSubidaLocal, error)
	ConfirmarRecepcion(context.Context, MovimientoSubidaLocal) (InstantaneaSubidaLocal, error)
	IniciarAnalisis(context.Context, MovimientoSubidaLocal) (InstantaneaSubidaLocal, error)
	RegistrarAnalisis(context.Context, MovimientoSubidaLocal) (InstantaneaSubidaLocal, error)
	CerrarAnalisis(context.Context, MovimientoSubidaLocal) (InstantaneaSubidaLocal, error)
	ConsultarPropia(context.Context, ConsultaPersistenciaSubidaLocal) (InstantaneaSubidaLocal, error)
}

// InstantaneaSubidaLocal es la postimagen durable devuelta por Documentos.
// El objeto y la evidencia nunca salen del adaptador HTTP.
type InstantaneaSubidaLocal struct {
	Resultado           ResultadoSubidaLocal                 `json:"resultado"`
	RecursoRef          string                               `json:"recurso_ref"`
	PersonaRef          string                               `json:"persona_ref"`
	CorrelacionRef      string                               `json:"correlacion_ref"`
	Clasificacion       string                               `json:"clasificacion"`
	HuellaSolicitudHMAC string                               `json:"huella_solicitud_hmac"`
	HuellaPDFSHA256     string                               `json:"huella_pdf_sha256"`
	MIME                string                               `json:"mime"`
	Tamano              int64                                `json:"tamano"`
	Version             uint64                               `json:"version"`
	Cuarentena          *vecports.ResultadoOperacionObjeto   `json:"cuarentena"`
	Analisis            *vecports.ResultadoAnalisisContenido `json:"analisis"`
	Admitido            *vecports.ResultadoOperacionObjeto   `json:"admitido"`
}

type FasePersistenciaSubidaLocal string

const (
	FaseSubidaLocalConfirmarRecepcion FasePersistenciaSubidaLocal = "confirmar_recepcion"
	FaseSubidaLocalIniciarAnalisis    FasePersistenciaSubidaLocal = "iniciar_analisis"
	FaseSubidaLocalRegistrarAnalisis  FasePersistenciaSubidaLocal = "registrar_analisis"
	FaseSubidaLocalCerrarAnalisis     FasePersistenciaSubidaLocal = "cerrar_analisis"
	FaseSubidaLocalConsultarCarga     FasePersistenciaSubidaLocal = "consulta_carga"
	FaseSubidaLocalConsultarDocumento FasePersistenciaSubidaLocal = "consulta_documento"
	FaseSubidaLocalAbrir              FasePersistenciaSubidaLocal = "abrir"
)

type MovimientoSubidaLocal struct {
	ActorRegistradoV2                              domain.ResultadoContextoActorRegistradoV2
	VinculoV2                                      domain.VinculoAutenticacionActorV2
	Accion                                         string
	Fase                                           FasePersistenciaSubidaLocal
	CargaID, RecursoRef, Finalidad, CorrelacionRef string
	VersionEsperada                                uint64
	HuellaPDFSHA256                                string
	ReciboRef                                      string
	EstadoFinal                                    EstadoSubidaLocal
	CodigoRechazo                                  string
	Cuarentena                                     *vecports.ResultadoOperacionObjeto
	Analisis                                       *vecports.ResultadoAnalisisContenido
	Promocion                                      *vecports.ResultadoOperacionObjeto
	AutorizacionV3                                 AutorizacionActoSubidaLocalV3
}

// PreimagenMovimientoSubidaLocal transporta el resultado técnico íntegro;
// la decisión V3 cubre todos sus bytes y SQL vuelve a cotejarlos con la fila.
func PreimagenMovimientoSubidaLocal(q MovimientoSubidaLocal) ([]byte, error) {
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.CargaID == "" || q.RecursoRef == "" || q.Finalidad == "" || q.CorrelacionRef == "" ||
		q.VersionEsperada == 0 ||
		(q.Accion != AccionMeritosComprobanteSubirLocal &&
			q.Accion != AccionDietasJustificanteSubirLocal && q.Accion != AccionCronosJustificacionSubirLocal) {
		return nil, ErrSubidaLocalDenegada
	}
	base := struct {
		Accion, Fase, CargaID, RecursoRef, PersonaRef, PerfilActivoRef, Finalidad string
		VersionEsperada                                                           uint64
		RegistroContextoRef, HuellaContextoSHA256, CorrelacionRef                 string
	}{q.Accion, string(q.Fase), q.CargaID, q.RecursoRef,
		q.ActorRegistradoV2.Contexto.PersonaRef, q.ActorRegistradoV2.Contexto.PerfilActivoRef,
		q.Finalidad, q.VersionEsperada, q.ActorRegistradoV2.RegistroContextoRef,
		q.ActorRegistradoV2.HuellaSHA256, q.CorrelacionRef}
	switch q.Fase {
	case FaseSubidaLocalConfirmarRecepcion:
		if q.Cuarentena == nil || q.Cuarentena.Validar() != nil || q.Analisis != nil || q.Promocion != nil ||
			q.HuellaPDFSHA256 == "" {
			return nil, ErrSubidaLocalDenegada
		}
		return json.Marshal(struct {
			Accion               string                             `json:"accion"`
			Fase                 string                             `json:"fase"`
			CargaID              string                             `json:"carga_id"`
			RecursoRef           string                             `json:"recurso_ref"`
			PersonaRef           string                             `json:"persona_ref"`
			PerfilActivoRef      string                             `json:"perfil_activo_ref"`
			Finalidad            string                             `json:"finalidad"`
			VersionEsperada      uint64                             `json:"version_esperada"`
			HuellaPDFSHA256      string                             `json:"huella_pdf_sha256"`
			Objeto               vecports.ObjetoAlmacenado          `json:"objeto"`
			EvidenciaAlmacen     vecports.EvidenciaOperacionAlmacen `json:"evidencia_almacen"`
			RegistroContextoRef  string                             `json:"registro_contexto_ref"`
			HuellaContextoSHA256 string                             `json:"huella_contexto_sha256"`
			CorrelacionRef       string                             `json:"correlacion_ref"`
		}{base.Accion, base.Fase, base.CargaID, base.RecursoRef, base.PersonaRef, base.PerfilActivoRef,
			base.Finalidad, base.VersionEsperada, q.HuellaPDFSHA256, q.Cuarentena.Objeto,
			q.Cuarentena.Evidencia, base.RegistroContextoRef, base.HuellaContextoSHA256, base.CorrelacionRef})
	case FaseSubidaLocalIniciarAnalisis:
		if q.Cuarentena != nil || q.Analisis != nil || q.Promocion != nil || q.ReciboRef == "" {
			return nil, ErrSubidaLocalDenegada
		}
		return json.Marshal(struct {
			Accion               string `json:"accion"`
			Fase                 string `json:"fase"`
			CargaID              string `json:"carga_id"`
			RecursoRef           string `json:"recurso_ref"`
			PersonaRef           string `json:"persona_ref"`
			PerfilActivoRef      string `json:"perfil_activo_ref"`
			Finalidad            string `json:"finalidad"`
			VersionEsperada      uint64 `json:"version_esperada"`
			ReciboRef            string `json:"recibo_ref"`
			RegistroContextoRef  string `json:"registro_contexto_ref"`
			HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
			CorrelacionRef       string `json:"correlacion_ref"`
		}{base.Accion, base.Fase, base.CargaID, base.RecursoRef, base.PersonaRef, base.PerfilActivoRef,
			base.Finalidad, base.VersionEsperada, q.ReciboRef, base.RegistroContextoRef,
			base.HuellaContextoSHA256, base.CorrelacionRef})
	case FaseSubidaLocalRegistrarAnalisis:
		if q.Analisis == nil || q.Analisis.Validar() != nil || q.Cuarentena != nil || q.Promocion != nil ||
			q.EstadoFinal != "" || q.CodigoRechazo != "" {
			return nil, ErrSubidaLocalDenegada
		}
		return json.Marshal(struct {
			Accion               string                              `json:"accion"`
			Fase                 string                              `json:"fase"`
			CargaID              string                              `json:"carga_id"`
			RecursoRef           string                              `json:"recurso_ref"`
			PersonaRef           string                              `json:"persona_ref"`
			PerfilActivoRef      string                              `json:"perfil_activo_ref"`
			Finalidad            string                              `json:"finalidad"`
			VersionEsperada      uint64                              `json:"version_esperada"`
			Analisis             vecports.ResultadoAnalisisContenido `json:"analisis"`
			RegistroContextoRef  string                              `json:"registro_contexto_ref"`
			HuellaContextoSHA256 string                              `json:"huella_contexto_sha256"`
			CorrelacionRef       string                              `json:"correlacion_ref"`
		}{base.Accion, base.Fase, base.CargaID, base.RecursoRef, base.PersonaRef, base.PerfilActivoRef,
			base.Finalidad, base.VersionEsperada, *q.Analisis, base.RegistroContextoRef,
			base.HuellaContextoSHA256, base.CorrelacionRef})
	case FaseSubidaLocalCerrarAnalisis:
		if q.Analisis == nil || q.Analisis.Validar() != nil ||
			(q.EstadoFinal != EstadoSubidaLocalAdmitida && q.EstadoFinal != EstadoSubidaLocalRechazada) ||
			(q.EstadoFinal == EstadoSubidaLocalAdmitida && (q.Promocion == nil || q.Promocion.Validar() != nil ||
				q.Analisis.Estado != vecports.EstadoAnalisisContenidoLimpio || q.CodigoRechazo != "")) ||
			(q.EstadoFinal == EstadoSubidaLocalRechazada && (q.Promocion != nil ||
				(q.CodigoRechazo != "pdf_no_valido" && q.CodigoRechazo != "contenido_no_permitido" &&
					q.CodigoRechazo != "analisis_no_concluyente"))) {
			return nil, ErrSubidaLocalDenegada
		}
		return json.Marshal(struct {
			Accion               string                              `json:"accion"`
			Fase                 string                              `json:"fase"`
			CargaID              string                              `json:"carga_id"`
			RecursoRef           string                              `json:"recurso_ref"`
			PersonaRef           string                              `json:"persona_ref"`
			PerfilActivoRef      string                              `json:"perfil_activo_ref"`
			Finalidad            string                              `json:"finalidad"`
			VersionEsperada      uint64                              `json:"version_esperada"`
			EstadoFinal          EstadoSubidaLocal                   `json:"estado_final"`
			CodigoRechazo        string                              `json:"codigo_rechazo"`
			Analisis             vecports.ResultadoAnalisisContenido `json:"analisis"`
			Promocion            *vecports.ResultadoOperacionObjeto  `json:"promocion"`
			RegistroContextoRef  string                              `json:"registro_contexto_ref"`
			HuellaContextoSHA256 string                              `json:"huella_contexto_sha256"`
			CorrelacionRef       string                              `json:"correlacion_ref"`
		}{base.Accion, base.Fase, base.CargaID, base.RecursoRef, base.PersonaRef, base.PerfilActivoRef,
			base.Finalidad, base.VersionEsperada, q.EstadoFinal, q.CodigoRechazo, *q.Analisis, q.Promocion,
			base.RegistroContextoRef, base.HuellaContextoSHA256, base.CorrelacionRef})
	default:
		return nil, ErrSubidaLocalDenegada
	}
}

type ConsultaPersistenciaSubidaLocal struct {
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	VinculoV2         domain.VinculoAutenticacionActorV2
	Accion            string
	Fase              FasePersistenciaSubidaLocal
	CargaID           string
	DocumentoID       string
	RecursoRef        string
	Objeto            vecports.ReferenciaObjetoAlmacen
	HuellaPDFSHA256   string
	Tamano            int64
	MIME              string
	CorrelacionRef    string
	AutorizacionV3    AutorizacionActoSubidaLocalV3
}

// Consulta por clave usa carga_id derivado servidor. La primera consulta por
// documento autoriza su identificador; abrir requiere después carga y
// documento exactos devueltos por SQL bajo otra decisión propia.
func PreimagenConsultaSubidaLocal(q ConsultaPersistenciaSubidaLocal) ([]byte, error) {
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.CorrelacionRef == "" ||
		(q.Accion != AccionMeritosComprobanteSubirLocal && q.Accion != AccionMeritosComprobanteDescargarLocal &&
			q.Accion != AccionDietasJustificanteSubirLocal && q.Accion != AccionDietasJustificanteDescargarLocal &&
			q.Accion != AccionCronosJustificacionSubirLocal && q.Accion != AccionCronosJustificacionDescargarLocal) {
		return nil, ErrSubidaLocalDenegada
	}
	switch q.Fase {
	case FaseSubidaLocalConsultarCarga:
		if q.CargaID == "" || q.DocumentoID != "" ||
			(q.Accion != AccionMeritosComprobanteSubirLocal && q.Accion != AccionDietasJustificanteSubirLocal &&
				q.Accion != AccionCronosJustificacionSubirLocal) {
			return nil, ErrSubidaLocalDenegada
		}
	case FaseSubidaLocalConsultarDocumento:
		if q.CargaID != "" || q.DocumentoID == "" ||
			(q.Accion != AccionMeritosComprobanteDescargarLocal && q.Accion != AccionDietasJustificanteDescargarLocal &&
				q.Accion != AccionCronosJustificacionDescargarLocal) {
			return nil, ErrSubidaLocalDenegada
		}
	case FaseSubidaLocalAbrir:
		if q.CargaID == "" || q.DocumentoID == "" ||
			q.Objeto.Validar() != nil || q.HuellaPDFSHA256 == "" || q.Tamano < 1 || q.MIME != "application/pdf" ||
			(q.Accion != AccionMeritosComprobanteDescargarLocal && q.Accion != AccionDietasJustificanteDescargarLocal &&
				q.Accion != AccionCronosJustificacionDescargarLocal) {
			return nil, ErrSubidaLocalDenegada
		}
	default:
		return nil, ErrSubidaLocalDenegada
	}
	if q.Fase == FaseSubidaLocalAbrir {
		return json.Marshal(struct {
			Accion               string `json:"accion"`
			Fase                 string `json:"fase"`
			CargaID              string `json:"carga_id"`
			DocumentoID          string `json:"documento_id"`
			PersonaRef           string `json:"persona_ref"`
			PerfilActivoRef      string `json:"perfil_activo_ref"`
			ObjetoRef            string `json:"objeto_ref"`
			ObjetoVersion        string `json:"objeto_version"`
			HuellaPDFSHA256      string `json:"huella_pdf_sha256"`
			Tamano               int64  `json:"tamano"`
			MIME                 string `json:"mime"`
			RegistroContextoRef  string `json:"registro_contexto_ref"`
			HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
			CorrelacionRef       string `json:"correlacion_ref"`
		}{q.Accion, string(q.Fase), q.CargaID, q.DocumentoID,
			q.ActorRegistradoV2.Contexto.PersonaRef, q.ActorRegistradoV2.Contexto.PerfilActivoRef,
			q.Objeto.Referencia, q.Objeto.Version, q.HuellaPDFSHA256, q.Tamano, q.MIME,
			q.ActorRegistradoV2.RegistroContextoRef, q.ActorRegistradoV2.HuellaSHA256, q.CorrelacionRef})
	}
	return json.Marshal(struct {
		Accion               string `json:"accion"`
		Fase                 string `json:"fase"`
		CargaID              string `json:"carga_id"`
		DocumentoID          string `json:"documento_id"`
		PersonaRef           string `json:"persona_ref"`
		PerfilActivoRef      string `json:"perfil_activo_ref"`
		RegistroContextoRef  string `json:"registro_contexto_ref"`
		HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
		CorrelacionRef       string `json:"correlacion_ref"`
	}{q.Accion, string(q.Fase), q.CargaID, q.DocumentoID,
		q.ActorRegistradoV2.Contexto.PersonaRef, q.ActorRegistradoV2.Contexto.PerfilActivoRef,
		q.ActorRegistradoV2.RegistroContextoRef, q.ActorRegistradoV2.HuellaSHA256, q.CorrelacionRef})
}

type SubidorLocal interface {
	Subir(context.Context, SolicitudSubidaLocal, []byte) (ResultadoSubidaLocal, error)
	Consultar(context.Context, SolicitudConsultaLocal) (ResultadoSubidaLocal, error)
	AbrirOriginal(context.Context, SolicitudConsultaLocal) (io.ReadCloser, error)
}
