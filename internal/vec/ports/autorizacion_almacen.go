package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	"vec-diputacion-granada/internal/vec/domain"
)

var (
	// ErrAutorizacionAlmacenInvalida expresa siempre una denegacion cerrada:
	// ausencia, ambiguedad, caducidad o cualquier dato no reconocido tienen el
	// mismo resultado y nunca se convierten en una concesion parcial.
	ErrAutorizacionAlmacenInvalida = errors.New("vec: autorizacion de almacen invalida")
	// ErrSerializacionContextoAlmacenProhibida evita que la capacidad o su
	// proyeccion interna crucen accidentalmente una frontera HTTP, de mensajes
	// o de persistencia generica.
	ErrSerializacionContextoAlmacenProhibida = almacencanonico.ErrSerializacionContextoAlmacenProhibida
)

const (
	EsquemaContextoOperacionAlmacenV1 = "vec.almacen.contexto-operacion.v1"

	// Acciones de negocio de lista positiva. Se duplican deliberadamente en
	// este puerto estable para no importar la capa de aplicacion ni el modulo
	// bolsa desde el nucleo VEC.
	AccionNegocioPrepararCargaDocumental     = "vec.documentos.carga.preparar"
	AccionNegocioConfirmarCargaDocumental    = "vec.documentos.carga.confirmar"
	AccionNegocioAnalizarCargaDocumental     = "vec.documentos.carga.analizar"
	AccionNegocioPromoverCargaDocumental     = "vec.documentos.carga.promover"
	AccionNegocioCustodiarDecisionBaremacion = "bolsa.decision.custodiar"
	AccionNegocioCustodiarDocumentoFirmado   = "bolsa.decision.firma.documento.custodiar"
	AccionNegocioRetenerDocumentoFirmado     = "bolsa.decision.firma.documento.retener"

	// Atributos que deben formar parte del RecursoAutorizable evaluado por el
	// PDP. Asi una decision no puede emplearse para acuñar capacidades sobre
	// otra operacion, carga, objeto o efecto despues de ser emitida.
	AtributoAlmacenOperacionRef        = "almacen_operacion_ref"
	AtributoAlmacenCargaRef            = "almacen_carga_ref"
	AtributoAlmacenClasificacion       = "almacen_clasificacion"
	AtributoAlmacenSujetoSeudonimoHMAC = "almacen_sujeto_seudonimo_hmac"
	AtributoAlmacenHuellaSolicitudHMAC = "almacen_huella_solicitud_hmac"
	AtributoAlmacenEfectoRef           = "almacen_efecto_ref"
	AtributoAlmacenObjetoRef           = "almacen_objeto_ref"
	AtributoAlmacenObjetoVersion       = "almacen_objeto_version"
	// Este atributo debe estar en el recurso antes de consultar al PDP.
	AtributoAlmacenHuellaManifiestoSHA256 = "almacen_manifiesto_generacion_sha256"
)

// PasoOperacionAlmacen identifica un paso previamente declarado por el
// nucleo. Aunque su representacion sea texto, ninguna fabrica acepta valores
// fuera del plan cerrado asociado a la accion de negocio.
type PasoOperacionAlmacen = almacencanonico.PasoOperacionAlmacen

const (
	PasoAlmacenPrepararCargaDirecta  = almacencanonico.PasoPrepararCargaDirecta
	PasoAlmacenAbandonarCargaDirecta = almacencanonico.PasoAbandonarCargaDirecta
	PasoAlmacenConfirmarCargaDirecta = almacencanonico.PasoConfirmarCargaDirecta
	PasoAlmacenLeerParaAnalisis      = almacencanonico.PasoLeerParaAnalisis
	PasoAlmacenAnalizarContenido     = almacencanonico.PasoAnalizarContenido
	PasoAlmacenPromover              = almacencanonico.PasoPromover
	PasoAlmacenCustodiarDecision     = almacencanonico.PasoCustodiarDecision
	PasoAlmacenCustodiarFirmado      = almacencanonico.PasoCustodiarFirmado
	PasoAlmacenRetenerFirmado        = almacencanonico.PasoRetenerFirmado
)

// VinculosOperacionAlmacen contiene datos no autoritativos que el constructor
// coteja uno a uno con el RecursoAutorizable ya evaluado. No es una capacidad.
// ObjetoVinculado es obligatorio solo en planes de lectura, promocion o
// retencion; en el resto debe ser el valor cero.
type VinculosOperacionAlmacen struct {
	OperacionRef        string
	CargaRef            string
	Clasificacion       string
	SujetoSeudonimoHMAC string
	HuellaSolicitudHMAC string
	EfectoRef           string
	ObjetoVinculado     ReferenciaObjetoAlmacen
}

type pasoPlanOperacionAlmacen struct {
	referencia       PasoOperacionAlmacen
	accion           string
	huellaPasoSHA256 string
	pasoDocumental   *datosPasoGeneracionDocumental
}

type especificacionAutorizacionAlmacen struct {
	accionNegocio          string
	camposExactos          []string
	pasos                  []pasoPlanOperacionAlmacen
	requiereObjeto         bool
	huellaManifiestoSHA256 string
}

type datosContextoOperacionAlmacen struct {
	esquema                string
	operacionRef           string
	correlacionRef         string
	autorizacionRef        string
	finalidad              string
	clasificacion          string
	accionNegocio          string
	accionTecnica          string
	cargaRef               string
	sujetoSeudonimoHMAC    string
	recursoRef             string
	moduloID               string
	tipoRecurso            string
	huellaRecursoSHA256    string
	huellaSolicitudHMAC    string
	efectoRef              string
	huellaPlanEfectoSHA256 string
	huellaManifiestoSHA256 string
	huellaPasoSHA256       string
	pasoRef                PasoOperacionAlmacen
	objetoVinculado        ReferenciaObjetoAlmacen
	huellaDecisionSHA256   string
	verificadaEn           time.Time
	validaHasta            time.Time
	evidencia              EvidenciaUsoDecisionAutorizacion
	pasos                  []pasoPlanOperacionAlmacen
}

// ContextoOperacionAlmacen es una capacidad opaca e inmutable. Su valor cero
// y cualquier valor reconstruido por serializacion son invalidos.
type ContextoOperacionAlmacen struct {
	datos *datosContextoOperacionAlmacen
}

// ProyeccionContextoOperacionAlmacen es una copia defensiva para conectores
// dentro del proceso. No permite reconstruir la capacidad y tampoco se puede
// serializar mediante los codificadores habituales.
type ProyeccionContextoOperacionAlmacen = almacencanonico.ProyeccionContextoOperacionAlmacen

func NuevoContextoPrepararCargaDirectaAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionPrepararCargaDocumental())
}

func NuevoContextoConfirmarCargaDirectaAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionConfirmarCargaDocumental())
}

func NuevoContextoAnalizarCargaDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionAnalizarCargaDocumental())
}

func NuevoContextoPromoverCargaDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionPromoverCargaDocumental())
}

func NuevoContextoCustodiarDecisionBaremacionAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionCustodiarDecisionBaremacion())
}

func NuevoContextoCustodiarDocumentoFirmadoAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionCustodiarDocumentoFirmado())
}

func NuevoContextoRetenerDocumentoFirmadoAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn,
		especificacionRetenerDocumentoFirmado())
}

// NuevoContextoGeneracionDocumentalAlmacen es la unica fabrica que admite
// una accion de negocio configurada. La accion procede exclusivamente del
// PermisoGenerar de la plantilla publicada comprometida por el manifiesto;
// nunca se acepta como parametro libre.
func NuevoContextoGeneracionDocumentalAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	manifiesto ManifiestoGeneracionDocumental,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
) (ContextoOperacionAlmacen, error) {
	if manifiesto.validarEstructura() != nil || recurso.ModuloID != manifiesto.datos.moduloID {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	especificacion, err := manifiesto.especificacionAutorizacionAlmacen()
	if err != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return nuevoContextoOperacionAlmacen(decision, recurso, vinculos, verificadaEn, especificacion)
}

func nuevoContextoOperacionAlmacen(
	decision domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	vinculos VinculosOperacionAlmacen,
	verificadaEn time.Time,
	especificacion especificacionAutorizacionAlmacen,
) (ContextoOperacionAlmacen, error) {
	if !especificacion.valida() || !vinculos.validosPara(especificacion) ||
		decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida ||
		decision.Accion != especificacion.accionNegocio || len(decision.Obligaciones) != 0 ||
		!camposAutorizacionExactos(decision.CamposPermitidos, especificacion.camposExactos) ||
		contieneComodinContextoAlmacen(decision.DecisionRef, decision.Accion, decision.RecursoRef,
			decision.ModuloID, decision.TipoRecurso, decision.Finalidad, decision.CorrelacionRef) ||
		recurso.Validar() != nil || contieneComodinRecursoAlmacen(recurso) ||
		decision.RecursoRef != recurso.Referencia || decision.ModuloID != recurso.ModuloID ||
		decision.TipoRecurso != recurso.Tipo {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaRecurso != decision.ContextoRecursoHuellaSHA256 ||
		!recursoVinculaOperacionAlmacen(recurso, vinculos, especificacion) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	datosEvidencia, err := evidencia.Datos()
	if err != nil || datosEvidencia.Decision.DecisionRef != decision.DecisionRef ||
		datosEvidencia.Decision.Accion != especificacion.accionNegocio ||
		datosEvidencia.Decision.ContextoRecursoHuellaSHA256 != huellaRecurso {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	pasos := clonarPasosOperacionAlmacen(especificacion.pasos)
	huellaPlan := huellaPlanOperacionAlmacen(decision, datosEvidencia.HuellaDecisionSHA256,
		huellaRecurso, vinculos, especificacion)
	if !esSHA256Hexadecimal(huellaPlan) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	primerPaso := pasos[0]
	datos := &datosContextoOperacionAlmacen{
		esquema:      EsquemaContextoOperacionAlmacenV1,
		operacionRef: vinculos.OperacionRef, correlacionRef: decision.CorrelacionRef,
		autorizacionRef: decision.DecisionRef, finalidad: decision.Finalidad,
		clasificacion: vinculos.Clasificacion, accionNegocio: decision.Accion,
		accionTecnica: primerPaso.accion, cargaRef: vinculos.CargaRef,
		sujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		recursoRef:          recurso.Referencia, moduloID: recurso.ModuloID, tipoRecurso: recurso.Tipo,
		huellaRecursoSHA256: huellaRecurso, huellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		efectoRef: vinculos.EfectoRef, huellaPlanEfectoSHA256: huellaPlan,
		huellaManifiestoSHA256: especificacion.huellaManifiestoSHA256,
		huellaPasoSHA256:       primerPaso.huellaPasoSHA256, pasoRef: primerPaso.referencia,
		objetoVinculado:      vinculos.ObjetoVinculado,
		huellaDecisionSHA256: datosEvidencia.HuellaDecisionSHA256,
		verificadaEn:         verificadaEn, validaHasta: decision.ValidaHasta.UTC(), evidencia: evidencia,
		pasos: pasos,
	}
	contexto := ContextoOperacionAlmacen{datos: datos}
	if contexto.validarEstructura() != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return contexto, nil
}

func especificacionPrepararCargaDocumental() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioPrepararCargaDocumental,
		camposExactos: []string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"},
		pasos: []pasoPlanOperacionAlmacen{
			{referencia: PasoAlmacenPrepararCargaDirecta, accion: AccionAlmacenPrepararCargaDirecta},
			{referencia: PasoAlmacenAbandonarCargaDirecta, accion: AccionAlmacenAbandonarCargaDirecta},
		},
	}
}

func especificacionConfirmarCargaDocumental() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioConfirmarCargaDocumental,
		camposExactos: []string{"contenido_cuarentena", "estado"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenConfirmarCargaDirecta, accion: AccionAlmacenConfirmarCargaDirecta,
		}},
	}
}

func especificacionAnalizarCargaDocumental() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioAnalizarCargaDocumental,
		camposExactos: []string{"analisis_seguridad", "estado"},
		pasos: []pasoPlanOperacionAlmacen{
			{referencia: PasoAlmacenLeerParaAnalisis, accion: AccionAlmacenLeer},
			{referencia: PasoAlmacenAnalizarContenido, accion: AccionAlmacenAnalizarContenido},
		},
		requiereObjeto: true,
	}
}

func especificacionPromoverCargaDocumental() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioPromoverCargaDocumental,
		camposExactos: []string{"contenido_admitido", "estado"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenPromover, accion: AccionAlmacenPromover,
		}},
		requiereObjeto: true,
	}
}

func especificacionCustodiarDecisionBaremacion() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioCustodiarDecisionBaremacion,
		camposExactos: []string{"documento_custodiado", "evidencia_custodia"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenCustodiarDecision, accion: AccionAlmacenEscribir,
		}},
	}
}

func especificacionCustodiarDocumentoFirmado() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioCustodiarDocumentoFirmado,
		camposExactos: []string{"documento_firmado.custodia", "evidencia_custodia"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenCustodiarFirmado, accion: AccionAlmacenEscribir,
		}},
	}
}

func especificacionRetenerDocumentoFirmado() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioRetenerDocumentoFirmado,
		camposExactos: []string{"documento_firmado.retencion", "evidencia_retencion"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenRetenerFirmado, accion: AccionAlmacenAplicarRetencion,
		}},
		requiereObjeto: true,
	}
}

func (e especificacionAutorizacionAlmacen) valida() bool {
	if e.accionNegocio == "" || contieneComodinContextoAlmacen(e.accionNegocio) || len(e.pasos) == 0 {
		return false
	}
	esDocumental := e.huellaManifiestoSHA256 != ""
	if esDocumental && !esSHA256Hexadecimal(e.huellaManifiestoSHA256) {
		return false
	}
	vistos := make(map[PasoOperacionAlmacen]struct{}, len(e.pasos))
	for _, paso := range e.pasos {
		if paso.referencia == "" || !accionOperacionAlmacenValida(paso.accion) ||
			contieneComodinContextoAlmacen(string(paso.referencia), paso.accion) {
			return false
		}
		if esDocumental {
			if paso.accion != AccionAlmacenEscribir || !esSHA256Hexadecimal(paso.huellaPasoSHA256) ||
				paso.pasoDocumental == nil || paso.pasoDocumental.validar() != nil ||
				paso.pasoDocumental.pasoRef != paso.referencia ||
				paso.pasoDocumental.huellaPasoSHA256 != paso.huellaPasoSHA256 {
				return false
			}
		} else if paso.huellaPasoSHA256 != "" || paso.pasoDocumental != nil {
			return false
		}
		if _, repetido := vistos[paso.referencia]; repetido {
			return false
		}
		vistos[paso.referencia] = struct{}{}
	}
	return camposAutorizacionExactos(e.camposExactos, e.camposExactos)
}

func (v VinculosOperacionAlmacen) validosPara(especificacion especificacionAutorizacionAlmacen) bool {
	if !referenciaOpacaAlmacenValida(v.OperacionRef, 512) ||
		!referenciaOpacaAlmacenValida(v.CargaRef, 512) ||
		!referenciaOpacaAlmacenValida(v.Clasificacion, 256) ||
		!hmacSHA256PuertoValido(v.SujetoSeudonimoHMAC) ||
		!hmacSHA256PuertoValido(v.HuellaSolicitudHMAC) ||
		!referenciaOpacaAlmacenValida(v.EfectoRef, 512) ||
		contieneComodinContextoAlmacen(v.OperacionRef, v.CargaRef, v.Clasificacion,
			v.SujetoSeudonimoHMAC, v.HuellaSolicitudHMAC, v.EfectoRef) {
		return false
	}
	if especificacion.requiereObjeto {
		return v.ObjetoVinculado.Validar() == nil &&
			!contieneComodinContextoAlmacen(v.ObjetoVinculado.Referencia, v.ObjetoVinculado.Version)
	}
	return v.ObjetoVinculado == (ReferenciaObjetoAlmacen{})
}

func recursoVinculaOperacionAlmacen(
	recurso domain.RecursoAutorizable,
	v VinculosOperacionAlmacen,
	especificacion especificacionAutorizacionAlmacen,
) bool {
	atributos := recurso.Atributos
	if atributos[AtributoAlmacenOperacionRef] != v.OperacionRef ||
		atributos[AtributoAlmacenCargaRef] != v.CargaRef ||
		atributos[AtributoAlmacenClasificacion] != v.Clasificacion ||
		atributos[AtributoAlmacenSujetoSeudonimoHMAC] != v.SujetoSeudonimoHMAC ||
		atributos[AtributoAlmacenHuellaSolicitudHMAC] != v.HuellaSolicitudHMAC ||
		atributos[AtributoAlmacenEfectoRef] != v.EfectoRef {
		return false
	}
	huellaManifiesto, existeManifiesto := atributos[AtributoAlmacenHuellaManifiestoSHA256]
	if especificacion.huellaManifiestoSHA256 == "" {
		if existeManifiesto {
			return false
		}
	} else if !existeManifiesto || huellaManifiesto != especificacion.huellaManifiestoSHA256 {
		return false
	}
	objetoRef, existeRef := atributos[AtributoAlmacenObjetoRef]
	objetoVersion, existeVersion := atributos[AtributoAlmacenObjetoVersion]
	if especificacion.requiereObjeto {
		return existeRef && existeVersion && objetoRef == v.ObjetoVinculado.Referencia &&
			objetoVersion == v.ObjetoVinculado.Version
	}
	return !existeRef && !existeVersion
}

func (c ContextoOperacionAlmacen) Proyeccion() (ProyeccionContextoOperacionAlmacen, error) {
	if c.validarEstructura() != nil {
		return ProyeccionContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	d := c.datos
	return ProyeccionContextoOperacionAlmacen{
		Esquema: d.esquema, OperacionRef: d.operacionRef, CorrelacionRef: d.correlacionRef,
		AutorizacionRef: d.autorizacionRef, Finalidad: d.finalidad, Clasificacion: d.clasificacion,
		AccionNegocio: d.accionNegocio, AccionTecnica: d.accionTecnica, CargaRef: d.cargaRef,
		SujetoSeudonimoHMAC: d.sujetoSeudonimoHMAC, RecursoRef: d.recursoRef, ModuloID: d.moduloID,
		TipoRecurso: d.tipoRecurso, HuellaRecursoSHA256: d.huellaRecursoSHA256,
		HuellaSolicitudHMAC: d.huellaSolicitudHMAC, EfectoRef: d.efectoRef,
		HuellaPlanEfectoSHA256: d.huellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: d.huellaManifiestoSHA256,
		HuellaPasoSHA256:       d.huellaPasoSHA256, PasoRef: d.pasoRef,
		ObjetoVinculado: d.objetoVinculado, HuellaDecisionSHA256: d.huellaDecisionSHA256,
		VerificadaEn: d.verificadaEn, ValidaHasta: d.validaHasta,
	}, nil
}

// EvidenciaAutorizacion devuelve deliberadamente la capacidad que el
// adaptador duradero debe revalidar y consumir de forma unica en la misma
// transaccion que DecisionRef -> (EfectoRef, HuellaPlanEfectoSHA256).
func (c ContextoOperacionAlmacen) EvidenciaAutorizacion() (EvidenciaUsoDecisionAutorizacion, error) {
	if c.validarEstructura() != nil {
		return EvidenciaUsoDecisionAutorizacion{}, errorAutorizacionAlmacen()
	}
	return c.datos.evidencia, nil
}

func (c ContextoOperacionAlmacen) ValidarEn(instante time.Time) error {
	if c.validarEstructura() != nil || instante.IsZero() || c.datos.evidencia.ValidarEn(instante) != nil ||
		!instante.UTC().Before(c.datos.validaHasta) {
		return errorAutorizacionAlmacen()
	}
	return nil
}

func (c ContextoOperacionAlmacen) ValidarParaEn(accionTecnica string, instante time.Time) error {
	if c.ValidarEn(instante) != nil || c.datos.accionTecnica != accionTecnica ||
		!accionOperacionAlmacenValida(accionTecnica) || contieneComodinContextoAlmacen(accionTecnica) {
		return errorAutorizacionAlmacen()
	}
	return nil
}

// DerivarPaso solo selecciona un paso del plan comprometido al emitir la
// capacidad. No acepta una accion tecnica ni permite ampliar el plan.
func (c ContextoOperacionAlmacen) DerivarPaso(pasoRef PasoOperacionAlmacen) (ContextoOperacionAlmacen, error) {
	if c.validarEstructura() != nil || pasoRef == "" || contieneComodinContextoAlmacen(string(pasoRef)) {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	var seleccionado pasoPlanOperacionAlmacen
	encontrado := false
	for _, paso := range c.datos.pasos {
		if paso.referencia == pasoRef {
			seleccionado = paso
			encontrado = true
			break
		}
	}
	if !encontrado {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	copia := *c.datos
	copia.accionTecnica = seleccionado.accion
	copia.pasoRef = seleccionado.referencia
	copia.huellaPasoSHA256 = seleccionado.huellaPasoSHA256
	copia.pasos = clonarPasosOperacionAlmacen(c.datos.pasos)
	resultado := ContextoOperacionAlmacen{datos: &copia}
	if resultado.validarEstructura() != nil {
		return ContextoOperacionAlmacen{}, errorAutorizacionAlmacen()
	}
	return resultado, nil
}

func (c ContextoOperacionAlmacen) coincideObjeto(objeto ReferenciaObjetoAlmacen) bool {
	return c.validarEstructura() == nil && c.datos.objetoVinculado == objeto
}

func (c ContextoOperacionAlmacen) validarParaPaso(accionTecnica string) error {
	if c.validarEstructura() != nil || c.datos.accionTecnica != accionTecnica ||
		!accionOperacionAlmacenValida(accionTecnica) {
		return errorAutorizacionAlmacen()
	}
	return nil
}

func (c ContextoOperacionAlmacen) validarEstructura() error {
	if c.datos == nil {
		return ErrAutorizacionAlmacenInvalida
	}
	d := c.datos
	if d.esquema != EsquemaContextoOperacionAlmacenV1 ||
		!referenciaOpacaAlmacenValida(d.operacionRef, 512) ||
		!referenciaOpacaAlmacenValida(d.correlacionRef, 512) ||
		!referenciaOpacaAlmacenValida(d.autorizacionRef, 512) ||
		!referenciaOpacaAlmacenValida(d.finalidad, 1024) ||
		!referenciaOpacaAlmacenValida(d.clasificacion, 256) ||
		!referenciaOpacaAlmacenValida(d.accionNegocio, 256) ||
		!accionOperacionAlmacenValida(d.accionTecnica) ||
		!referenciaOpacaAlmacenValida(d.cargaRef, 512) ||
		!hmacSHA256PuertoValido(d.sujetoSeudonimoHMAC) ||
		!referenciaOpacaAlmacenValida(d.recursoRef, 512) ||
		!referenciaOpacaAlmacenValida(d.moduloID, 128) ||
		!referenciaOpacaAlmacenValida(d.tipoRecurso, 128) ||
		!esSHA256Hexadecimal(d.huellaRecursoSHA256) ||
		!hmacSHA256PuertoValido(d.huellaSolicitudHMAC) ||
		!referenciaOpacaAlmacenValida(d.efectoRef, 512) ||
		!esSHA256Hexadecimal(d.huellaPlanEfectoSHA256) || d.pasoRef == "" ||
		!esSHA256Hexadecimal(d.huellaDecisionSHA256) || d.verificadaEn.IsZero() ||
		d.validaHasta.IsZero() || !d.validaHasta.After(d.verificadaEn) ||
		contieneComodinContextoAlmacen(d.operacionRef, d.correlacionRef, d.autorizacionRef,
			d.finalidad, d.clasificacion, d.accionNegocio, d.accionTecnica, d.cargaRef,
			d.sujetoSeudonimoHMAC, d.recursoRef, d.moduloID, d.tipoRecurso,
			d.huellaSolicitudHMAC, d.efectoRef, string(d.pasoRef)) || len(d.pasos) == 0 {
		return ErrAutorizacionAlmacenInvalida
	}
	if (d.huellaManifiestoSHA256 == "") != (d.huellaPasoSHA256 == "") {
		return ErrAutorizacionAlmacenInvalida
	}
	if d.huellaManifiestoSHA256 != "" &&
		(!esSHA256Hexadecimal(d.huellaManifiestoSHA256) || !esSHA256Hexadecimal(d.huellaPasoSHA256)) {
		return ErrAutorizacionAlmacenInvalida
	}
	pasoValido := false
	for _, paso := range d.pasos {
		if paso.referencia == d.pasoRef && paso.accion == d.accionTecnica &&
			paso.huellaPasoSHA256 == d.huellaPasoSHA256 {
			pasoValido = true
		}
	}
	if !pasoValido || d.evidencia.ValidarEn(d.verificadaEn) != nil {
		return ErrAutorizacionAlmacenInvalida
	}
	datosEvidencia, err := d.evidencia.Datos()
	if err != nil || datosEvidencia.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		datosEvidencia.HuellaDecisionSHA256 != d.huellaDecisionSHA256 ||
		datosEvidencia.Decision.DecisionRef != d.autorizacionRef ||
		datosEvidencia.Decision.Accion != d.accionNegocio ||
		datosEvidencia.Decision.RecursoRef != d.recursoRef ||
		datosEvidencia.Decision.ModuloID != d.moduloID ||
		datosEvidencia.Decision.TipoRecurso != d.tipoRecurso ||
		datosEvidencia.Decision.ContextoRecursoHuellaSHA256 != d.huellaRecursoSHA256 ||
		datosEvidencia.Decision.Finalidad != d.finalidad ||
		datosEvidencia.Decision.CorrelacionRef != d.correlacionRef ||
		!datosEvidencia.Decision.ValidaHasta.Equal(d.validaHasta) {
		return ErrAutorizacionAlmacenInvalida
	}
	especificacion := especificacionAutorizacionAlmacen{
		accionNegocio: d.accionNegocio, camposExactos: datosEvidencia.Decision.CamposPermitidos,
		pasos: d.pasos, requiereObjeto: d.objetoVinculado != (ReferenciaObjetoAlmacen{}),
		huellaManifiestoSHA256: d.huellaManifiestoSHA256,
	}
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: d.operacionRef, CargaRef: d.cargaRef, Clasificacion: d.clasificacion,
		SujetoSeudonimoHMAC: d.sujetoSeudonimoHMAC, HuellaSolicitudHMAC: d.huellaSolicitudHMAC,
		EfectoRef: d.efectoRef, ObjetoVinculado: d.objetoVinculado,
	}
	if !especificacion.valida() || !vinculos.validosPara(especificacion) ||
		huellaPlanOperacionAlmacen(datosEvidencia.Decision, d.huellaDecisionSHA256,
			d.huellaRecursoSHA256, vinculos, especificacion) != d.huellaPlanEfectoSHA256 {
		return ErrAutorizacionAlmacenInvalida
	}
	return nil
}

func camposAutorizacionExactos(recibidos, esperados []string) bool {
	if len(recibidos) != len(esperados) {
		return false
	}
	conjunto := make(map[string]struct{}, len(recibidos))
	for _, campo := range recibidos {
		if !referenciaOpacaAlmacenValida(campo, 256) || contieneComodinContextoAlmacen(campo) {
			return false
		}
		if _, repetido := conjunto[campo]; repetido {
			return false
		}
		conjunto[campo] = struct{}{}
	}
	for _, campo := range esperados {
		if _, existe := conjunto[campo]; !existe {
			return false
		}
	}
	return true
}

func contieneComodinRecursoAlmacen(recurso domain.RecursoAutorizable) bool {
	for clave, valor := range recurso.Ambitos {
		if contieneComodinContextoAlmacen(clave, valor) {
			return true
		}
	}
	for clave, valor := range recurso.Atributos {
		if contieneComodinContextoAlmacen(clave, valor) {
			return true
		}
	}
	return false
}

func contieneComodinContextoAlmacen(valores ...string) bool {
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}

func huellaPlanOperacionAlmacen(
	decision domain.DecisionAutorizacion,
	huellaDecision, huellaRecurso string,
	v VinculosOperacionAlmacen,
	e especificacionAutorizacionAlmacen,
) string {
	valores := []string{
		EsquemaContextoOperacionAlmacenV1, decision.DecisionRef, huellaDecision,
		decision.Accion, decision.RecursoRef, huellaRecurso, decision.Finalidad,
		decision.CorrelacionRef, v.OperacionRef, v.CargaRef, v.Clasificacion,
		v.SujetoSeudonimoHMAC, v.HuellaSolicitudHMAC, v.EfectoRef,
		v.ObjetoVinculado.Referencia, v.ObjetoVinculado.Version,
	}
	if e.huellaManifiestoSHA256 != "" {
		valores = append(valores, e.huellaManifiestoSHA256)
	}
	for _, paso := range e.pasos {
		valores = append(valores, string(paso.referencia), paso.accion)
		if paso.huellaPasoSHA256 != "" {
			valores = append(valores, paso.huellaPasoSHA256)
		}
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

func clonarPasosOperacionAlmacen(pasos []pasoPlanOperacionAlmacen) []pasoPlanOperacionAlmacen {
	resultado := append([]pasoPlanOperacionAlmacen(nil), pasos...)
	for indice := range resultado {
		if resultado[indice].pasoDocumental != nil {
			copia := *resultado[indice].pasoDocumental
			resultado[indice].pasoDocumental = &copia
		}
	}
	return resultado
}

func errorAutorizacionAlmacen() error {
	return errors.Join(ErrSolicitudAlmacenInvalida, ErrAutorizacionAlmacenInvalida)
}

func (ContextoOperacionAlmacen) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionContextoAlmacenProhibida
}

func (*ContextoOperacionAlmacen) UnmarshalJSON([]byte) error {
	return ErrSerializacionContextoAlmacenProhibida
}

func (ContextoOperacionAlmacen) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionContextoAlmacenProhibida
}

func (*ContextoOperacionAlmacen) UnmarshalText([]byte) error {
	return ErrSerializacionContextoAlmacenProhibida
}

func (ContextoOperacionAlmacen) String() string {
	return "[CONTEXTO-OPERACION-ALMACEN-OPACO]"
}

func (c ContextoOperacionAlmacen) GoString() string { return c.String() }

func (c ContextoOperacionAlmacen) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (c ContextoOperacionAlmacen) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
