package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrCargaDocumentalInvalida        = errors.New("vec: carga documental invalida")
	ErrTransicionCargaNoPermitida     = errors.New("vec: transicion de carga documental no permitida")
	ErrContenidoCargaNoCorresponde    = errors.New("vec: contenido de carga documental no corresponde")
	ErrAnalisisCargaNoCorresponde     = errors.New("vec: analisis de carga documental no corresponde")
	ErrCargaDocumentalFueraDeVigencia = errors.New("vec: carga documental fuera de vigencia")
	ErrManifiestoPreparacionInvalido  = errors.New("vec: manifiesto de preparacion de carga directa invalido")
)

const duracionMaximaReservaCargaDocumental = 10 * time.Minute

const (
	EsquemaManifiestoPreparacionCargaDirectaV1 = "vec.carga-directa.manifiesto-preparacion.v1"

	esquemaContextoAlmacenManifiestoV1 = "vec.almacen.contexto-operacion.v1"
	esquemaHuellaDecisionManifiestoV1  = "vec.autorizacion.decision.reforzada.v1.autenticacion-actor"
	accionPrepararCargaManifiestoV1    = "vec.documentos.carga.preparar"
	accionTecnicaPrepararManifiestoV1  = "preparar_carga_directa"
	pasoPrepararCargaManifiestoV1      = "01_preparar_carga_directa"
)

// ContextoManifiestoPreparacionCargaDirectaV1 contiene la proyeccion no
// autoritativa del plan que existia al preparar la carga. La fabrica la cruza
// con el agregado preparado y fija una huella canonica; nunca permite
// reconstruir una capacidad ni sustituye la decision original.
type ContextoManifiestoPreparacionCargaDirectaV1 struct {
	CargaRef                string
	SujetoSeudonimoHMAC     string
	HuellaRecursoBaseSHA256 string
	HuellaRecursoSHA256     string
	ConectorAlmacenID       string
	EsquemaContexto         string
	AccionNegocio           string
	AccionTecnica           string
	PasoRef                 string
	EfectoRef               string
	HuellaPlanEfectoSHA256  string
	EsquemaHuellaDecision   string
	DecisionRef             string
	HuellaDecisionSHA256    string
	ContextoVerificadoEn    time.Time
	DecisionValidaHasta     time.Time
}

// DatosManifiestoPreparacionCargaDirectaV1 es una copia destinada a un
// adaptador durable. Todos sus campos son escalares y la huella se recalcula
// al leer; una mutacion o decodificacion parcial falla cerrada.
type DatosManifiestoPreparacionCargaDirectaV1 struct {
	Esquema                 string    `json:"esquema"`
	CargaID                 string    `json:"carga_id"`
	VersionCarga            int       `json:"version_carga"`
	HuellaCargaSHA256       string    `json:"huella_carga_sha256"`
	IndiceIdempotenciaHMAC  string    `json:"indice_idempotencia_hmac"`
	OperacionRef            string    `json:"operacion_ref"`
	CorrelacionRef          string    `json:"correlacion_ref"`
	CargaRef                string    `json:"carga_ref"`
	Finalidad               string    `json:"finalidad"`
	Clasificacion           string    `json:"clasificacion"`
	SujetoSeudonimoHMAC     string    `json:"sujeto_seudonimo_hmac"`
	HuellaSolicitudHMAC     string    `json:"huella_solicitud_hmac"`
	RecursoRef              string    `json:"recurso_ref"`
	ModuloID                string    `json:"modulo_id"`
	TipoRecurso             string    `json:"tipo_recurso"`
	HuellaRecursoBaseSHA256 string    `json:"huella_recurso_base_sha256"`
	HuellaRecursoSHA256     string    `json:"huella_recurso_sha256"`
	MIME                    string    `json:"mime"`
	Tamano                  int64     `json:"tamano"`
	HuellaContenidoSHA256   string    `json:"huella_contenido_sha256"`
	VinculoSesionHMAC       string    `json:"vinculo_sesion_hmac"`
	ConectorAlmacenID       string    `json:"conector_almacen_id"`
	EsquemaContexto         string    `json:"esquema_contexto"`
	AccionNegocio           string    `json:"accion_negocio"`
	AccionTecnica           string    `json:"accion_tecnica"`
	PasoRef                 string    `json:"paso_ref"`
	EfectoRef               string    `json:"efecto_ref"`
	HuellaPlanEfectoSHA256  string    `json:"huella_plan_efecto_sha256"`
	EsquemaHuellaDecision   string    `json:"esquema_huella_decision"`
	DecisionRef             string    `json:"decision_ref"`
	HuellaDecisionSHA256    string    `json:"huella_decision_sha256"`
	ContextoVerificadoEn    time.Time `json:"contexto_verificado_en"`
	DecisionValidaHasta     time.Time `json:"decision_valida_hasta"`
	PreparadaEn             time.Time `json:"preparada_en"`
	ExpiraEn                time.Time `json:"expira_en"`
	HuellaManifiestoSHA256  string    `json:"huella_manifiesto_sha256"`
}

// ManifiestoPreparacionCargaDirectaV1 es opaco e inmutable fuera del dominio.
// Solo conserva hechos historicos no autoritativos; no contiene la sesion ni
// el recibo en claro y no puede emplearse para acunar otra autorizacion.
type ManifiestoPreparacionCargaDirectaV1 struct {
	datos DatosManifiestoPreparacionCargaDirectaV1
}

func NuevoManifiestoPreparacionCargaDirectaV1(
	carga CargaDocumental,
	contexto ContextoManifiestoPreparacionCargaDirectaV1,
) (ManifiestoPreparacionCargaDirectaV1, error) {
	if carga.Validar() != nil || carga.Estado != EstadoCargaDocumentalPreparada {
		return ManifiestoPreparacionCargaDirectaV1{}, ErrManifiestoPreparacionInvalido
	}
	huellaCarga, err := carga.HuellaSHA256()
	if err != nil {
		return ManifiestoPreparacionCargaDirectaV1{}, ErrManifiestoPreparacionInvalido
	}
	datos := DatosManifiestoPreparacionCargaDirectaV1{
		Esquema: EsquemaManifiestoPreparacionCargaDirectaV1,
		CargaID: carga.ID, VersionCarga: carga.Version, HuellaCargaSHA256: huellaCarga,
		IndiceIdempotenciaHMAC: carga.IndiceIdempotenciaHMAC,
		OperacionRef:           carga.OperacionRef, CorrelacionRef: carga.CorrelacionRef,
		CargaRef: contexto.CargaRef, Finalidad: carga.Finalidad, Clasificacion: carga.Clasificacion,
		SujetoSeudonimoHMAC: contexto.SujetoSeudonimoHMAC,
		HuellaSolicitudHMAC: carga.HuellaSolicitudHMAC,
		RecursoRef:          carga.RecursoRef, ModuloID: carga.ModuloID, TipoRecurso: carga.TipoRecurso,
		HuellaRecursoBaseSHA256: contexto.HuellaRecursoBaseSHA256,
		HuellaRecursoSHA256:     contexto.HuellaRecursoSHA256,
		MIME:                    carga.MIMEDeclarado, Tamano: carga.TamanoDeclarado,
		HuellaContenidoSHA256: carga.HuellaDeclaradaSHA256,
		VinculoSesionHMAC:     carga.VinculoSesionHMAC, ConectorAlmacenID: contexto.ConectorAlmacenID,
		EsquemaContexto: contexto.EsquemaContexto, AccionNegocio: contexto.AccionNegocio,
		AccionTecnica: contexto.AccionTecnica, PasoRef: contexto.PasoRef,
		EfectoRef: contexto.EfectoRef, HuellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256,
		EsquemaHuellaDecision: contexto.EsquemaHuellaDecision, DecisionRef: contexto.DecisionRef,
		HuellaDecisionSHA256: contexto.HuellaDecisionSHA256,
		ContextoVerificadoEn: contexto.ContextoVerificadoEn.UTC(),
		DecisionValidaHasta:  contexto.DecisionValidaHasta.UTC(),
		PreparadaEn:          carga.PreparadaEn.UTC(), ExpiraEn: carga.ExpiraEn.UTC(),
	}
	datos.HuellaManifiestoSHA256 = huellaCanonicaManifiestoPreparacionCargaDirectaV1(datos)
	manifiesto := ManifiestoPreparacionCargaDirectaV1{datos: datos}
	if manifiesto.ValidarContraCarga(carga) != nil {
		return ManifiestoPreparacionCargaDirectaV1{}, ErrManifiestoPreparacionInvalido
	}
	return manifiesto, nil
}

// RestaurarManifiestoPreparacionCargaDirectaV1 rehidrata exclusivamente los
// hechos exactos que un adaptador durable obtuvo antes mediante Datos. No
// completa, normaliza ni deriva ningun campo ausente: una fila parcial,
// alterada o perteneciente a otro esquema falla cerrada.
func RestaurarManifiestoPreparacionCargaDirectaV1(
	datos DatosManifiestoPreparacionCargaDirectaV1,
) (ManifiestoPreparacionCargaDirectaV1, error) {
	manifiesto := ManifiestoPreparacionCargaDirectaV1{datos: datos}
	if manifiesto.Validar() != nil {
		return ManifiestoPreparacionCargaDirectaV1{}, ErrManifiestoPreparacionInvalido
	}
	return manifiesto, nil
}

func (m ManifiestoPreparacionCargaDirectaV1) Validar() error {
	d := m.datos
	if d.Esquema != EsquemaManifiestoPreparacionCargaDirectaV1 ||
		!referenciaManifiestoCargaValida(d.CargaID) || d.VersionCarga != 2 || !esSHA256(d.HuellaCargaSHA256) ||
		!esHuellaHMACSHA256(d.IndiceIdempotenciaHMAC) || !referenciaManifiestoCargaValida(d.OperacionRef) ||
		!referenciaManifiestoCargaValida(d.CorrelacionRef) || d.CargaRef != d.IndiceIdempotenciaHMAC ||
		!textoDocumentalNoVacioValido(d.Finalidad) || strings.ContainsRune(d.Finalidad, '*') ||
		!esClaveDocumentalCanonica(d.Clasificacion) || !esHuellaHMACSHA256(d.SujetoSeudonimoHMAC) ||
		!esHuellaHMACSHA256(d.HuellaSolicitudHMAC) || !referenciaManifiestoCargaValida(d.RecursoRef) ||
		!esClaveDocumentalCanonica(d.ModuloID) || !esClaveDocumentalCanonica(d.TipoRecurso) ||
		!esSHA256(d.HuellaRecursoBaseSHA256) || !esSHA256(d.HuellaRecursoSHA256) ||
		!mimeManifiestoCargaValido(d.MIME) || d.Tamano < 1 ||
		!esSHA256(d.HuellaContenidoSHA256) || !esHuellaHMACSHA256(d.VinculoSesionHMAC) ||
		!referenciaManifiestoCargaValida(d.ConectorAlmacenID) ||
		d.EsquemaContexto != esquemaContextoAlmacenManifiestoV1 ||
		d.AccionNegocio != accionPrepararCargaManifiestoV1 ||
		d.AccionTecnica != accionTecnicaPrepararManifiestoV1 || d.PasoRef != pasoPrepararCargaManifiestoV1 ||
		!referenciaManifiestoCargaValida(d.EfectoRef) || !esSHA256(d.HuellaPlanEfectoSHA256) ||
		d.EsquemaHuellaDecision != esquemaHuellaDecisionManifiestoV1 ||
		!referenciaManifiestoCargaValida(d.DecisionRef) || !esSHA256(d.HuellaDecisionSHA256) ||
		d.ContextoVerificadoEn.IsZero() || d.DecisionValidaHasta.IsZero() || d.PreparadaEn.IsZero() ||
		d.ExpiraEn.IsZero() || d.ContextoVerificadoEn.Location() != time.UTC ||
		d.DecisionValidaHasta.Location() != time.UTC || d.PreparadaEn.Location() != time.UTC ||
		d.ExpiraEn.Location() != time.UTC || !d.DecisionValidaHasta.After(d.ContextoVerificadoEn) ||
		d.PreparadaEn.Before(d.ContextoVerificadoEn) || !d.PreparadaEn.Before(d.ExpiraEn) ||
		d.DecisionValidaHasta.Before(d.ExpiraEn) || !esSHA256(d.HuellaManifiestoSHA256) ||
		d.HuellaManifiestoSHA256 != huellaCanonicaManifiestoPreparacionCargaDirectaV1(d) {
		return ErrManifiestoPreparacionInvalido
	}
	return nil
}

func (m ManifiestoPreparacionCargaDirectaV1) ValidarContraCarga(carga CargaDocumental) error {
	if m.Validar() != nil || carga.Validar() != nil || carga.Estado != EstadoCargaDocumentalPreparada {
		return ErrManifiestoPreparacionInvalido
	}
	huellaCarga, err := carga.HuellaSHA256()
	d := m.datos
	if err != nil || d.CargaID != carga.ID || d.VersionCarga != carga.Version || d.HuellaCargaSHA256 != huellaCarga ||
		d.IndiceIdempotenciaHMAC != carga.IndiceIdempotenciaHMAC || d.OperacionRef != carga.OperacionRef ||
		d.CorrelacionRef != carga.CorrelacionRef || d.CargaRef != carga.IndiceIdempotenciaHMAC ||
		d.Finalidad != carga.Finalidad || d.Clasificacion != carga.Clasificacion ||
		d.HuellaSolicitudHMAC != carga.HuellaSolicitudHMAC || d.RecursoRef != carga.RecursoRef ||
		d.ModuloID != carga.ModuloID || d.TipoRecurso != carga.TipoRecurso || d.MIME != carga.MIMEDeclarado ||
		d.Tamano != carga.TamanoDeclarado || d.HuellaContenidoSHA256 != carga.HuellaDeclaradaSHA256 ||
		d.VinculoSesionHMAC != carga.VinculoSesionHMAC || d.DecisionRef != carga.AutorizacionPreparacionRef ||
		!d.PreparadaEn.Equal(carga.PreparadaEn) || !d.ExpiraEn.Equal(carga.ExpiraEn) {
		return ErrManifiestoPreparacionInvalido
	}
	return nil
}

func (m ManifiestoPreparacionCargaDirectaV1) Datos() (DatosManifiestoPreparacionCargaDirectaV1, error) {
	if m.Validar() != nil {
		return DatosManifiestoPreparacionCargaDirectaV1{}, ErrManifiestoPreparacionInvalido
	}
	return m.datos, nil
}

func (m ManifiestoPreparacionCargaDirectaV1) HuellaSHA256() (string, error) {
	datos, err := m.Datos()
	if err != nil {
		return "", err
	}
	return datos.HuellaManifiestoSHA256, nil
}

func (ManifiestoPreparacionCargaDirectaV1) String() string {
	return "[MANIFIESTO-PREPARACION-CARGA-DIRECTA-V1]"
}

func (m ManifiestoPreparacionCargaDirectaV1) GoString() string { return m.String() }

func huellaCanonicaManifiestoPreparacionCargaDirectaV1(d DatosManifiestoPreparacionCargaDirectaV1) string {
	valores := []string{
		d.Esquema, d.CargaID, strconv.Itoa(d.VersionCarga), d.HuellaCargaSHA256,
		d.IndiceIdempotenciaHMAC, d.OperacionRef, d.CorrelacionRef, d.CargaRef,
		d.Finalidad, d.Clasificacion, d.SujetoSeudonimoHMAC, d.HuellaSolicitudHMAC,
		d.RecursoRef, d.ModuloID, d.TipoRecurso, d.HuellaRecursoBaseSHA256, d.HuellaRecursoSHA256,
		d.MIME, strconv.FormatInt(d.Tamano, 10), d.HuellaContenidoSHA256,
		d.VinculoSesionHMAC, d.ConectorAlmacenID, d.EsquemaContexto,
		d.AccionNegocio, d.AccionTecnica, d.PasoRef, d.EfectoRef,
		d.HuellaPlanEfectoSHA256, d.EsquemaHuellaDecision, d.DecisionRef,
		d.HuellaDecisionSHA256, d.ContextoVerificadoEn.UTC().Format(time.RFC3339Nano),
		d.DecisionValidaHasta.UTC().Format(time.RFC3339Nano), d.PreparadaEn.UTC().Format(time.RFC3339Nano),
		d.ExpiraEn.UTC().Format(time.RFC3339Nano),
	}
	var canonico strings.Builder
	for _, valor := range valores {
		canonico.WriteString(strconv.Itoa(len(valor)))
		canonico.WriteByte(':')
		canonico.WriteString(valor)
		canonico.WriteByte('\n')
	}
	suma := sha256.Sum256([]byte(canonico.String()))
	return hex.EncodeToString(suma[:])
}

func referenciaManifiestoCargaValida(valor string) bool {
	return referenciaDocumentalValida(valor) && !strings.ContainsRune(valor, '*') && !strings.ContainsAny(valor, " \t\r\n")
}

func mimeManifiestoCargaValido(valor string) bool {
	return textoDocumentalNoVacioValido(valor) && valor == strings.ToLower(valor) &&
		!strings.ContainsAny(valor, " \t\r\n;*") && strings.Count(valor, "/") == 1
}

// EstadoCargaDocumental expresa estados de seguridad, no estados de merito o
// tramitacion administrativa. Una carga admitida aun puede ser rechazada por
// RRHH por no cumplir las bases de una convocatoria.
type EstadoCargaDocumental string

const (
	EstadoCargaDocumentalReservada         EstadoCargaDocumental = "reservada"
	EstadoCargaDocumentalPreparada         EstadoCargaDocumental = "preparada"
	EstadoCargaDocumentalCuarentena        EstadoCargaDocumental = "cuarentena"
	EstadoCargaDocumentalAnalizadaLimpia   EstadoCargaDocumental = "analizada_limpia"
	EstadoCargaDocumentalRetenidaSeguridad EstadoCargaDocumental = "retenida_seguridad"
	EstadoCargaDocumentalAdmitida          EstadoCargaDocumental = "admitida"
	EstadoCargaDocumentalAbandonada        EstadoCargaDocumental = "abandonada"
	EstadoCargaDocumentalExpirada          EstadoCargaDocumental = "expirada"
)

func (e EstadoCargaDocumental) Valido() bool {
	switch e {
	case EstadoCargaDocumentalReservada, EstadoCargaDocumentalPreparada,
		EstadoCargaDocumentalCuarentena, EstadoCargaDocumentalAnalizadaLimpia,
		EstadoCargaDocumentalRetenidaSeguridad, EstadoCargaDocumentalAdmitida,
		EstadoCargaDocumentalAbandonada, EstadoCargaDocumentalExpirada:
		return true
	default:
		return false
	}
}

type ZonaContenidoCarga string

const (
	ZonaContenidoCargaCuarentena ZonaContenidoCarga = "cuarentena"
	ZonaContenidoCargaAdmitida   ZonaContenidoCarga = "admitida"
)

func (z ZonaContenidoCarga) Valida() bool {
	return z == ZonaContenidoCargaCuarentena || z == ZonaContenidoCargaAdmitida
}

// ContenidoCargaDocumental conserva referencias opacas y evidencia tecnica;
// nunca rutas, nombres originales, URL temporales ni credenciales del almacen.
type ContenidoCargaDocumental struct {
	ConectorID   string             `json:"conector_id"`
	Referencia   string             `json:"referencia"`
	Version      string             `json:"version"`
	Zona         ZonaContenidoCarga `json:"zona"`
	MIME         string             `json:"mime"`
	Tamano       int64              `json:"tamano"`
	HuellaSHA256 string             `json:"huella_sha256"`
	EvidenciaRef string             `json:"evidencia_ref"`
	RegistradoEn time.Time          `json:"registrado_en"`
}

func (c ContenidoCargaDocumental) Validar() error {
	if !referenciaDocumentalValida(c.ConectorID) || !referenciaDocumentalValida(c.Referencia) ||
		!referenciaDocumentalValida(c.Version) || !c.Zona.Valida() || !textoDocumentalNoVacioValido(c.MIME) ||
		c.Tamano < 1 || !esSHA256(c.HuellaSHA256) || !referenciaDocumentalValida(c.EvidenciaRef) ||
		c.RegistradoEn.IsZero() {
		return ErrContenidoCargaNoCorresponde
	}
	return nil
}

type EstadoAnalisisCarga string

const (
	EstadoAnalisisCargaLimpio        EstadoAnalisisCarga = "limpio"
	EstadoAnalisisCargaMalicioso     EstadoAnalisisCarga = "malicioso"
	EstadoAnalisisCargaSospechoso    EstadoAnalisisCarga = "sospechoso"
	EstadoAnalisisCargaNoConcluyente EstadoAnalisisCarga = "no_concluyente"
	EstadoAnalisisCargaError         EstadoAnalisisCarga = "error"
)

func (e EstadoAnalisisCarga) Valido() bool {
	switch e {
	case EstadoAnalisisCargaLimpio, EstadoAnalisisCargaMalicioso,
		EstadoAnalisisCargaSospechoso, EstadoAnalisisCargaNoConcluyente,
		EstadoAnalisisCargaError:
		return true
	default:
		return false
	}
}

// AnalisisCargaDocumental fija el resultado normalizado del motor sobre una
// version exacta. La salida cruda del antivirus no entra en el dominio.
type AnalisisCargaDocumental struct {
	ObjetoReferencia      string              `json:"objeto_referencia"`
	ObjetoVersion         string              `json:"objeto_version"`
	HuellaObjetoSHA256    string              `json:"huella_objeto_sha256"`
	ConectorAnalizadorID  string              `json:"conector_analizador_id"`
	VersionConector       int                 `json:"version_conector"`
	Estado                EstadoAnalisisCarga `json:"estado"`
	CodigoResultado       string              `json:"codigo_resultado"`
	EvidenciaRef          string              `json:"evidencia_ref"`
	HuellaEvidenciaSHA256 string              `json:"huella_evidencia_sha256"`
	CompletadoEn          time.Time           `json:"completado_en"`
}

func (a AnalisisCargaDocumental) Validar() error {
	if !referenciaDocumentalValida(a.ObjetoReferencia) || !referenciaDocumentalValida(a.ObjetoVersion) ||
		!esSHA256(a.HuellaObjetoSHA256) || !referenciaDocumentalValida(a.ConectorAnalizadorID) ||
		a.VersionConector < 1 || !a.Estado.Valido() || !textoDocumentalNoVacioValido(a.CodigoResultado) ||
		!referenciaDocumentalValida(a.EvidenciaRef) || !esSHA256(a.HuellaEvidenciaSHA256) || a.CompletadoEn.IsZero() {
		return ErrAnalisisCargaNoCorresponde
	}
	return nil
}

// CargaDocumental es el agregado tecnico que impide que una sesion temporal,
// un resultado del antivirus o una promocion se apliquen a otro expediente.
// VinculoSesionHMAC permite comprobar la sesion sin persistirla en claro.
type CargaDocumental struct {
	ID                         string                    `json:"id"`
	Version                    int                       `json:"version"`
	PrincipalID                string                    `json:"principal_id"`
	RecursoRef                 string                    `json:"recurso_ref"`
	ModuloID                   string                    `json:"modulo_id"`
	TipoRecurso                string                    `json:"tipo_recurso"`
	OperacionRef               string                    `json:"operacion_ref"`
	CorrelacionRef             string                    `json:"correlacion_ref"`
	Finalidad                  string                    `json:"finalidad"`
	Clasificacion              string                    `json:"clasificacion"`
	MIMEDeclarado              string                    `json:"mime_declarado"`
	TamanoDeclarado            int64                     `json:"tamano_declarado"`
	HuellaDeclaradaSHA256      string                    `json:"huella_declarada_sha256"`
	IndiceIdempotenciaHMAC     string                    `json:"indice_idempotencia_hmac"`
	HuellaSolicitudHMAC        string                    `json:"huella_solicitud_hmac"`
	VinculoSesionHMAC          string                    `json:"vinculo_sesion_hmac,omitempty"`
	Estado                     EstadoCargaDocumental     `json:"estado"`
	AutorizacionPreparacionRef string                    `json:"autorizacion_preparacion_ref,omitempty"`
	AutorizacionRecepcionRef   string                    `json:"autorizacion_recepcion_ref,omitempty"`
	AutorizacionAnalisisRef    string                    `json:"autorizacion_analisis_ref,omitempty"`
	AutorizacionPromocionRef   string                    `json:"autorizacion_promocion_ref,omitempty"`
	ContenidoCuarentena        *ContenidoCargaDocumental `json:"contenido_cuarentena,omitempty"`
	Analisis                   *AnalisisCargaDocumental  `json:"analisis,omitempty"`
	ContenidoAdmitido          *ContenidoCargaDocumental `json:"contenido_admitido,omitempty"`
	CreadaEn                   time.Time                 `json:"creada_en"`
	PreparadaEn                time.Time                 `json:"preparada_en,omitempty"`
	ExpiraEn                   time.Time                 `json:"expira_en"`
	ActualizadaEn              time.Time                 `json:"actualizada_en"`
}

func NuevaCargaDocumental(
	id, principalID, recursoRef, moduloID, tipoRecurso, operacionRef, correlacionRef,
	finalidad, clasificacion, mime string,
	tamano int64,
	huellaSHA256, indiceIdempotenciaHMAC, huellaSolicitudHMAC string,
	creadaEn, expiraEn time.Time,
) (CargaDocumental, error) {
	carga := CargaDocumental{
		ID: id, Version: 1, PrincipalID: principalID, RecursoRef: recursoRef,
		ModuloID: moduloID, TipoRecurso: tipoRecurso, OperacionRef: operacionRef,
		CorrelacionRef: correlacionRef, Finalidad: finalidad, Clasificacion: clasificacion,
		MIMEDeclarado: mime, TamanoDeclarado: tamano, HuellaDeclaradaSHA256: huellaSHA256,
		IndiceIdempotenciaHMAC: indiceIdempotenciaHMAC, HuellaSolicitudHMAC: huellaSolicitudHMAC,
		Estado:   EstadoCargaDocumentalReservada,
		CreadaEn: creadaEn.UTC(), ExpiraEn: expiraEn.UTC(), ActualizadaEn: creadaEn.UTC(),
	}
	if err := carga.Validar(); err != nil {
		return CargaDocumental{}, err
	}
	return carga, nil
}

func (c CargaDocumental) Validar() error {
	if !referenciaDocumentalValida(c.ID) || c.Version < 1 || !referenciaDocumentalValida(c.PrincipalID) ||
		!referenciaDocumentalValida(c.RecursoRef) || !esClaveDocumentalCanonica(c.ModuloID) ||
		!esClaveDocumentalCanonica(c.TipoRecurso) || !referenciaDocumentalValida(c.OperacionRef) ||
		!referenciaDocumentalValida(c.CorrelacionRef) || !textoDocumentalNoVacioValido(c.Finalidad) ||
		!esClaveDocumentalCanonica(c.Clasificacion) || !textoDocumentalNoVacioValido(c.MIMEDeclarado) ||
		c.TamanoDeclarado < 1 || !esSHA256(c.HuellaDeclaradaSHA256) ||
		!esHuellaHMACSHA256(c.IndiceIdempotenciaHMAC) || !esHuellaHMACSHA256(c.HuellaSolicitudHMAC) ||
		!c.Estado.Valido() || c.CreadaEn.IsZero() ||
		c.ExpiraEn.IsZero() || !c.ExpiraEn.After(c.CreadaEn) ||
		c.ExpiraEn.Sub(c.CreadaEn) > duracionMaximaReservaCargaDocumental || c.ActualizadaEn.Before(c.CreadaEn) {
		return ErrCargaDocumentalInvalida
	}
	return c.validarEstado()
}

func (c CargaDocumental) validarEstado() error {
	sesionPreparada := esHuellaHMACSHA256(c.VinculoSesionHMAC) &&
		referenciaDocumentalValida(c.AutorizacionPreparacionRef) && !c.PreparadaEn.IsZero() &&
		c.PreparadaEn.After(c.CreadaEn) && c.PreparadaEn.Before(c.ExpiraEn)
	recepcion := c.ContenidoCuarentena != nil && c.ContenidoCuarentena.Validar() == nil &&
		c.ContenidoCuarentena.Zona == ZonaContenidoCargaCuarentena &&
		referenciaDocumentalValida(c.AutorizacionRecepcionRef) && sesionPreparada &&
		c.ContenidoCuarentena.RegistradoEn.After(c.PreparadaEn)
	analisis := c.Analisis != nil && c.Analisis.Validar() == nil && referenciaDocumentalValida(c.AutorizacionAnalisisRef) &&
		recepcion && c.Analisis.CompletadoEn.After(c.ContenidoCuarentena.RegistradoEn)
	admision := c.ContenidoAdmitido != nil && c.ContenidoAdmitido.Validar() == nil &&
		c.ContenidoAdmitido.Zona == ZonaContenidoCargaAdmitida && referenciaDocumentalValida(c.AutorizacionPromocionRef) &&
		analisis && c.ContenidoAdmitido.RegistradoEn.After(c.Analisis.CompletadoEn)
	sinAutorizacionesPosterioresAPreparacion := c.AutorizacionRecepcionRef == "" &&
		c.AutorizacionAnalisisRef == "" && c.AutorizacionPromocionRef == ""
	sinAutorizacionesPosterioresARecepcion := c.AutorizacionAnalisisRef == "" && c.AutorizacionPromocionRef == ""

	switch c.Estado {
	case EstadoCargaDocumentalReservada:
		if c.Version != 1 || c.VinculoSesionHMAC != "" || c.AutorizacionPreparacionRef != "" ||
			!sinAutorizacionesPosterioresAPreparacion || !c.PreparadaEn.IsZero() ||
			c.ContenidoCuarentena != nil || c.Analisis != nil || c.ContenidoAdmitido != nil ||
			!c.ActualizadaEn.Equal(c.CreadaEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalPreparada:
		if c.Version != 2 || !sesionPreparada || !sinAutorizacionesPosterioresAPreparacion ||
			c.ContenidoCuarentena != nil || c.Analisis != nil || c.ContenidoAdmitido != nil ||
			!c.ActualizadaEn.Equal(c.PreparadaEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalCuarentena:
		if c.Version != 3 || !sesionPreparada || !recepcion || !sinAutorizacionesPosterioresARecepcion ||
			c.Analisis != nil || c.ContenidoAdmitido != nil ||
			c.ActualizadaEn.Before(c.ContenidoCuarentena.RegistradoEn) || !c.ActualizadaEn.Before(c.ExpiraEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalAnalizadaLimpia:
		if c.Version != 4 || !sesionPreparada || !recepcion || !analisis ||
			c.Analisis.Estado != EstadoAnalisisCargaLimpio || c.AutorizacionPromocionRef != "" || admision ||
			!c.ActualizadaEn.Equal(c.Analisis.CompletadoEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalRetenidaSeguridad:
		if c.Version != 4 || !sesionPreparada || !recepcion || !analisis ||
			c.Analisis.Estado == EstadoAnalisisCargaLimpio || c.AutorizacionPromocionRef != "" || admision ||
			!c.ActualizadaEn.Equal(c.Analisis.CompletadoEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalAdmitida:
		if c.Version != 5 || !sesionPreparada || !recepcion || !analisis ||
			c.Analisis.Estado != EstadoAnalisisCargaLimpio || !admision ||
			c.ActualizadaEn.Before(c.ContenidoAdmitido.RegistradoEn) ||
			!c.ActualizadaEn.After(c.Analisis.CompletadoEn) {
			return ErrCargaDocumentalInvalida
		}
	case EstadoCargaDocumentalAbandonada, EstadoCargaDocumentalExpirada:
		if recepcion || analisis || admision {
			return ErrCargaDocumentalInvalida
		}
	default:
		return ErrCargaDocumentalInvalida
	}
	return nil
}

func (c CargaDocumental) Preparar(vinculoSesionHMAC, autorizacionRef string, preparadaEn time.Time) (CargaDocumental, error) {
	if c.Validar() != nil || c.Estado != EstadoCargaDocumentalReservada || !preparadaEn.UTC().After(c.ActualizadaEn) ||
		!preparadaEn.UTC().Before(c.ExpiraEn) {
		return CargaDocumental{}, ErrTransicionCargaNoPermitida
	}
	actualizada := c.clonar()
	actualizada.Version++
	actualizada.VinculoSesionHMAC = vinculoSesionHMAC
	actualizada.AutorizacionPreparacionRef = autorizacionRef
	actualizada.PreparadaEn = preparadaEn.UTC()
	actualizada.ActualizadaEn = preparadaEn.UTC()
	actualizada.Estado = EstadoCargaDocumentalPreparada
	if err := actualizada.Validar(); err != nil {
		return CargaDocumental{}, err
	}
	return actualizada, nil
}

func (c CargaDocumental) RegistrarCuarentena(
	contenido ContenidoCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error) {
	instante = instante.UTC()
	if c.Validar() != nil || c.Estado != EstadoCargaDocumentalPreparada || contenido.Validar() != nil ||
		contenido.Zona != ZonaContenidoCargaCuarentena || contenido.MIME != c.MIMEDeclarado ||
		contenido.Tamano != c.TamanoDeclarado || contenido.HuellaSHA256 != c.HuellaDeclaradaSHA256 ||
		!instante.After(c.ActualizadaEn) || !instante.Before(c.ExpiraEn) ||
		!contenido.RegistradoEn.UTC().After(c.ActualizadaEn) || contenido.RegistradoEn.UTC().After(instante) {
		return CargaDocumental{}, ErrContenidoCargaNoCorresponde
	}
	contenido.RegistradoEn = contenido.RegistradoEn.UTC()
	actualizada := c.clonar()
	actualizada.Version++
	actualizada.ContenidoCuarentena = &contenido
	actualizada.AutorizacionRecepcionRef = autorizacionRef
	actualizada.ActualizadaEn = instante
	actualizada.Estado = EstadoCargaDocumentalCuarentena
	if err := actualizada.Validar(); err != nil {
		return CargaDocumental{}, err
	}
	return actualizada, nil
}

func (c CargaDocumental) RegistrarAnalisis(
	analisis AnalisisCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error) {
	instante = instante.UTC()
	if c.Validar() != nil || c.Estado != EstadoCargaDocumentalCuarentena || analisis.Validar() != nil ||
		analisis.ObjetoReferencia != c.ContenidoCuarentena.Referencia ||
		analisis.ObjetoVersion != c.ContenidoCuarentena.Version ||
		analisis.HuellaObjetoSHA256 != c.ContenidoCuarentena.HuellaSHA256 ||
		!instante.After(c.ActualizadaEn) || !instante.Equal(analisis.CompletadoEn.UTC()) {
		return CargaDocumental{}, ErrAnalisisCargaNoCorresponde
	}
	analisis.CompletadoEn = analisis.CompletadoEn.UTC()
	actualizada := c.clonar()
	actualizada.Version++
	actualizada.Analisis = &analisis
	actualizada.AutorizacionAnalisisRef = autorizacionRef
	actualizada.ActualizadaEn = instante
	if analisis.Estado == EstadoAnalisisCargaLimpio {
		actualizada.Estado = EstadoCargaDocumentalAnalizadaLimpia
	} else {
		actualizada.Estado = EstadoCargaDocumentalRetenidaSeguridad
	}
	if err := actualizada.Validar(); err != nil {
		return CargaDocumental{}, err
	}
	return actualizada, nil
}

func (c CargaDocumental) Admitir(
	contenido ContenidoCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error) {
	instante = instante.UTC()
	if c.Validar() != nil || c.Estado != EstadoCargaDocumentalAnalizadaLimpia || contenido.Validar() != nil ||
		contenido.Zona != ZonaContenidoCargaAdmitida || contenido.Referencia == c.ContenidoCuarentena.Referencia ||
		contenido.MIME != c.ContenidoCuarentena.MIME || contenido.Tamano != c.ContenidoCuarentena.Tamano ||
		contenido.HuellaSHA256 != c.ContenidoCuarentena.HuellaSHA256 || !instante.After(c.ActualizadaEn) ||
		!contenido.RegistradoEn.UTC().After(c.Analisis.CompletadoEn.UTC()) || contenido.RegistradoEn.UTC().After(instante) {
		return CargaDocumental{}, ErrContenidoCargaNoCorresponde
	}
	contenido.RegistradoEn = contenido.RegistradoEn.UTC()
	actualizada := c.clonar()
	actualizada.Version++
	actualizada.ContenidoAdmitido = &contenido
	actualizada.AutorizacionPromocionRef = autorizacionRef
	actualizada.ActualizadaEn = instante
	actualizada.Estado = EstadoCargaDocumentalAdmitida
	if err := actualizada.Validar(); err != nil {
		return CargaDocumental{}, err
	}
	return actualizada, nil
}

func (c CargaDocumental) VigenteEn(instante time.Time) bool {
	return c.Validar() == nil && instante.UTC().Before(c.ExpiraEn) &&
		c.Estado != EstadoCargaDocumentalAbandonada && c.Estado != EstadoCargaDocumentalExpirada
}

func (c CargaDocumental) HuellaSHA256() (string, error) {
	if err := c.Validar(); err != nil {
		return "", err
	}
	canonico := c.clonar()
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", fmt.Errorf("%w: serializar", ErrCargaDocumentalInvalida)
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (c CargaDocumental) clonar() CargaDocumental {
	clon := c
	if c.ContenidoCuarentena != nil {
		contenido := *c.ContenidoCuarentena
		clon.ContenidoCuarentena = &contenido
	}
	if c.Analisis != nil {
		analisis := *c.Analisis
		clon.Analisis = &analisis
	}
	if c.ContenidoAdmitido != nil {
		contenido := *c.ContenidoAdmitido
		clon.ContenidoAdmitido = &contenido
	}
	return clon
}
