package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrDefinicionFlujoInvalida    = errors.New("vec: definicion de flujo invalida")
	ErrEstadoFlujoInvalido        = errors.New("vec: estado de flujo invalido")
	ErrTransicionFlujoInvalida    = errors.New("vec: transicion de flujo invalida")
	ErrGrafoFlujoInvalido         = errors.New("vec: grafo de flujo invalido")
	ErrDefinicionFlujoNoPublicada = errors.New("vec: definicion de flujo no publicada")
	ErrInstanciaFlujoInvalida     = errors.New("vec: instancia de flujo invalida")
	ErrDecisionReglaInvalida      = errors.New("vec: decision de regla invalida")
	ErrReglaFlujoDenegada         = errors.New("vec: regla de flujo no satisfecha")
	ErrAprobacionFlujoRequerida   = errors.New("vec: la transicion requiere aprobacion")
	ErrAprobacionFlujoInvalida    = errors.New("vec: aprobacion de flujo invalida")
)

const (
	maximoEstadosFlujo          = 512
	maximoTransicionesFlujo     = 4_096
	maximoOrigenesTransicion    = 512
	maximoAtributosFlujo        = 128
	maximoBytesFlujo            = 16 * 1024 * 1024
	vigenciaMaximaDecisionRegla = 15 * time.Minute
)

type EstadoFlujoConfigurable struct {
	Clave     string                    `json:"clave"`
	Catalogo  ReferenciaEntradaCatalogo `json:"catalogo"`
	Orden     int                       `json:"orden"`
	Terminal  bool                      `json:"terminal"`
	Atributos map[string]string         `json:"atributos,omitempty"`
}

func (e EstadoFlujoConfigurable) Validar() error {
	if !esClaveDocumentalCanonica(e.Clave) || e.Catalogo.Validar() != nil ||
		e.Clave != e.Catalogo.EntradaClave || e.Orden < 0 || len(e.Atributos) > maximoAtributosFlujo {
		return ErrEstadoFlujoInvalido
	}
	if !atributosFlujoValidos(e.Atributos) {
		return ErrEstadoFlujoInvalido
	}
	return nil
}

type TransicionFlujoConfigurable struct {
	Clave          string        `json:"clave"`
	Desde          []string      `json:"desde"`
	Hacia          string        `json:"hacia"`
	Accion         string        `json:"accion"`
	ReglaRef       string        `json:"regla_ref"`
	Prioridad      int           `json:"prioridad"`
	GarantiaMinima AuthAssurance `json:"garantia_minima"`
	// RequiereMotivo permite que la interfaz destaque una justificacion
	// reforzada. El dominio exige motivo en todas las transiciones por politica
	// general de trazabilidad administrativa.
	RequiereMotivo     bool              `json:"requiere_motivo"`
	RequiereAprobacion bool              `json:"requiere_aprobacion"`
	Automatica         bool              `json:"automatica"`
	PlazoRef           string            `json:"plazo_ref,omitempty"`
	Atributos          map[string]string `json:"atributos,omitempty"`
}

func (t TransicionFlujoConfigurable) Validar() error {
	if !esClaveDocumentalCanonica(t.Clave) || len(t.Desde) == 0 || len(t.Desde) > maximoOrigenesTransicion ||
		!esClaveDocumentalCanonica(t.Hacia) || !esClaveDocumentalCanonica(t.Accion) ||
		!referenciaDocumentalValida(t.ReglaRef) || t.Prioridad < 0 || !t.GarantiaMinima.Valida() ||
		(t.PlazoRef != "" && !referenciaDocumentalValida(t.PlazoRef)) || len(t.Atributos) > maximoAtributosFlujo ||
		!atributosFlujoValidos(t.Atributos) {
		return ErrTransicionFlujoInvalida
	}
	vistos := make(map[string]struct{}, len(t.Desde))
	for _, origen := range t.Desde {
		if !esClaveDocumentalCanonica(origen) || origen == t.Hacia {
			return ErrTransicionFlujoInvalida
		}
		if _, repetido := vistos[origen]; repetido {
			return ErrTransicionFlujoInvalida
		}
		vistos[origen] = struct{}{}
	}
	return nil
}

func (t TransicionFlujoConfigurable) AdmiteOrigen(estado string) bool {
	for _, origen := range t.Desde {
		if origen == estado {
			return true
		}
	}
	return false
}

type EstadoDefinicionFlujo string

const (
	EstadoDefinicionFlujoBorrador  EstadoDefinicionFlujo = "borrador"
	EstadoDefinicionFlujoPublicada EstadoDefinicionFlujo = "publicada"
	EstadoDefinicionFlujoRetirada  EstadoDefinicionFlujo = "retirada"
)

func (e EstadoDefinicionFlujo) Valido() bool {
	return e == EstadoDefinicionFlujoBorrador || e == EstadoDefinicionFlujoPublicada || e == EstadoDefinicionFlujoRetirada
}

const (
	AccionDefinicionFlujoBorradorCreada      = "vec.flujos.definicion.borrador.creada"
	AccionDefinicionFlujoBorradorActualizada = "vec.flujos.definicion.borrador.actualizada"
	AccionDefinicionFlujoPublicada           = "vec.flujos.definicion.publicada"
	AccionDefinicionFlujoRetirada            = "vec.flujos.definicion.retirada"
	AccionInstanciaFlujoIniciada             = "vec.flujos.instancia.iniciada"
	AccionInstanciaFlujoTransicionada        = "vec.flujos.instancia.transicionada"
	AccionDecisionReglaFlujoRegistrada       = "vec.flujos.regla.decision.registrada"
)

type DefinicionFlujo struct {
	ID                              string                        `json:"id"`
	Version                         int                           `json:"version"`
	Revision                        int                           `json:"revision"`
	VersionAnteriorRef              string                        `json:"version_anterior_ref,omitempty"`
	ModuloID                        string                        `json:"modulo_id"`
	TipoEntidad                     string                        `json:"tipo_entidad"`
	Nombre                          string                        `json:"nombre"`
	Descripcion                     string                        `json:"descripcion,omitempty"`
	FuenteRef                       string                        `json:"fuente_ref"`
	MotivoCreacion                  string                        `json:"motivo_creacion"`
	EstadoInicial                   string                        `json:"estado_inicial,omitempty"`
	AccionInicio                    string                        `json:"accion_inicio"`
	GarantiaInicio                  AuthAssurance                 `json:"garantia_inicio"`
	PermiteFinalizacionTrasRetirada bool                          `json:"permite_finalizacion_tras_retirada"`
	Estados                         []EstadoFlujoConfigurable     `json:"estados"`
	Transiciones                    []TransicionFlujoConfigurable `json:"transiciones"`
	Estado                          EstadoDefinicionFlujo         `json:"estado"`
	CreadaPor                       string                        `json:"creada_por"`
	CreadaEn                        time.Time                     `json:"creada_en"`
	UltimaModificacionPor           string                        `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn            time.Time                     `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion              string                        `json:"motivo_modificacion,omitempty"`
	PublicadaPor                    string                        `json:"publicada_por,omitempty"`
	PublicadaEn                     time.Time                     `json:"publicada_en,omitempty"`
	AprobacionRef                   string                        `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion               string                        `json:"motivo_publicacion,omitempty"`
	RetiradaPor                     string                        `json:"retirada_por,omitempty"`
	RetiradaEn                      time.Time                     `json:"retirada_en,omitempty"`
	RetiradaAprobacionRef           string                        `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada                  string                        `json:"motivo_retirada,omitempty"`
}

func (d DefinicionFlujo) Referencia() string {
	return d.ID + ":" + strconv.Itoa(d.Version)
}

func (d DefinicionFlujo) Validar() error {
	if !esClaveDocumentalCanonica(d.ID) || d.Version < 1 || d.Revision < 1 ||
		!esClaveDocumentalCanonica(d.ModuloID) || !esClaveDocumentalCanonica(d.TipoEntidad) ||
		!textoAcotadoCatalogo(d.Nombre, maximoCaracteresEtiqueta, true) ||
		!textoAcotadoCatalogo(d.Descripcion, maximoCaracteresDescripcion, false) ||
		!textoAcotadoCatalogo(d.FuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(d.MotivoCreacion, maximoCaracteresDescripcion, true) ||
		!esClaveDocumentalCanonica(d.AccionInicio) || !d.GarantiaInicio.Valida() ||
		len(d.Estados) > maximoEstadosFlujo || len(d.Transiciones) > maximoTransicionesFlujo ||
		!d.Estado.Valido() || !referenciaDocumentalValida(d.CreadaPor) || d.CreadaEn.IsZero() {
		return ErrDefinicionFlujoInvalida
	}
	if (d.Version == 1 && d.VersionAnteriorRef != "") ||
		(d.Version > 1 && d.VersionAnteriorRef != d.ID+":"+strconv.Itoa(d.Version-1)) {
		return ErrDefinicionFlujoInvalida
	}
	if d.Revision == 1 {
		if d.UltimaModificacionPor != "" || !d.UltimaModificacionEn.IsZero() || d.MotivoModificacion != "" {
			return ErrDefinicionFlujoInvalida
		}
	} else if !referenciaDocumentalValida(d.UltimaModificacionPor) || d.UltimaModificacionEn.IsZero() ||
		d.UltimaModificacionEn.Before(d.CreadaEn) ||
		!textoAcotadoCatalogo(d.MotivoModificacion, maximoCaracteresDescripcion, true) {
		return ErrDefinicionFlujoInvalida
	}
	if err := validarElementosFlujo(d); err != nil {
		return err
	}
	switch d.Estado {
	case EstadoDefinicionFlujoBorrador:
		if datosPublicacionFlujoPresentes(d) || datosRetiradaFlujoPresentes(d) {
			return ErrDefinicionFlujoInvalida
		}
	case EstadoDefinicionFlujoPublicada:
		if !datosPublicacionFlujoValidos(d) || datosRetiradaFlujoPresentes(d) || validarGrafoFlujo(d) != nil {
			return ErrDefinicionFlujoInvalida
		}
	case EstadoDefinicionFlujoRetirada:
		if !datosPublicacionFlujoValidos(d) || !datosRetiradaFlujoValidos(d) || validarGrafoFlujo(d) != nil {
			return ErrDefinicionFlujoInvalida
		}
	}
	return nil
}

func (d DefinicionFlujo) ClonarCanonico() (DefinicionFlujo, error) {
	canonico := d
	canonico.CreadaEn = fechaCatalogoUTC(d.CreadaEn)
	canonico.UltimaModificacionEn = fechaCatalogoUTC(d.UltimaModificacionEn)
	canonico.PublicadaEn = fechaCatalogoUTC(d.PublicadaEn)
	canonico.RetiradaEn = fechaCatalogoUTC(d.RetiradaEn)
	canonico.Estados = make([]EstadoFlujoConfigurable, len(d.Estados))
	for indice, estado := range d.Estados {
		canonico.Estados[indice] = estado
		canonico.Estados[indice].Atributos = clonarAtributosCatalogo(estado.Atributos)
	}
	sort.Slice(canonico.Estados, func(i, j int) bool {
		if canonico.Estados[i].Orden != canonico.Estados[j].Orden {
			return canonico.Estados[i].Orden < canonico.Estados[j].Orden
		}
		return canonico.Estados[i].Clave < canonico.Estados[j].Clave
	})
	canonico.Transiciones = make([]TransicionFlujoConfigurable, len(d.Transiciones))
	for indice, transicion := range d.Transiciones {
		canonico.Transiciones[indice] = transicion
		canonico.Transiciones[indice].Desde = append([]string(nil), transicion.Desde...)
		sort.Strings(canonico.Transiciones[indice].Desde)
		canonico.Transiciones[indice].Atributos = clonarAtributosCatalogo(transicion.Atributos)
	}
	sort.Slice(canonico.Transiciones, func(i, j int) bool {
		if canonico.Transiciones[i].Prioridad != canonico.Transiciones[j].Prioridad {
			return canonico.Transiciones[i].Prioridad < canonico.Transiciones[j].Prioridad
		}
		return canonico.Transiciones[i].Clave < canonico.Transiciones[j].Clave
	})
	if err := canonico.Validar(); err != nil {
		return DefinicionFlujo{}, err
	}
	return canonico, nil
}

func (d DefinicionFlujo) HuellaSHA256() (string, error) {
	canonico, err := d.ClonarCanonico()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrDefinicionFlujoInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// HuellaContenidoSHA256 identifica la semantica inmutable de esta version del
// flujo. No cambia al publicarla o retirarla; HuellaSHA256 conserva, en cambio,
// la evidencia de la instantanea completa de gobierno.
func (d DefinicionFlujo) HuellaContenidoSHA256() (string, error) {
	canonico, err := d.ClonarCanonico()
	if err != nil {
		return "", err
	}
	canonico.Estado = ""
	canonico.CreadaPor = ""
	canonico.CreadaEn = time.Time{}
	canonico.MotivoCreacion = ""
	canonico.UltimaModificacionPor = ""
	canonico.UltimaModificacionEn = time.Time{}
	canonico.MotivoModificacion = ""
	canonico.PublicadaPor = ""
	canonico.PublicadaEn = time.Time{}
	canonico.AprobacionRef = ""
	canonico.MotivoPublicacion = ""
	canonico.RetiradaPor = ""
	canonico.RetiradaEn = time.Time{}
	canonico.RetiradaAprobacionRef = ""
	canonico.MotivoRetirada = ""
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrDefinicionFlujoInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (d DefinicionFlujo) Publicar(actorID, aprobacionRef, motivo string, instante time.Time) (DefinicionFlujo, error) {
	if err := d.Validar(); err != nil {
		return DefinicionFlujo{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if d.Estado != EstadoDefinicionFlujoBorrador || validarGrafoFlujo(d) != nil ||
		!referenciaDocumentalValida(actorID) || actorID == d.CreadaPor || actorID == d.UltimaModificacionPor ||
		!textoAcotadoCatalogo(aprobacionRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() ||
		instante.Before(ultimaFechaGobiernoFlujo(d)) {
		return DefinicionFlujo{}, ErrTransicionFlujoInvalida
	}
	publicada := d
	publicada.Estado = EstadoDefinicionFlujoPublicada
	publicada.PublicadaPor = actorID
	publicada.PublicadaEn = instante.UTC()
	publicada.AprobacionRef = strings.TrimSpace(aprobacionRef)
	publicada.MotivoPublicacion = strings.TrimSpace(motivo)
	return publicada.ClonarCanonico()
}

func (d DefinicionFlujo) Retirar(actorID, aprobacionRef, motivo string, instante time.Time) (DefinicionFlujo, error) {
	if err := d.Validar(); err != nil {
		return DefinicionFlujo{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if d.Estado != EstadoDefinicionFlujoPublicada || !referenciaDocumentalValida(actorID) || actorID == d.PublicadaPor ||
		!textoAcotadoCatalogo(aprobacionRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() || instante.Before(d.PublicadaEn) {
		return DefinicionFlujo{}, ErrTransicionFlujoInvalida
	}
	retirada := d
	retirada.Estado = EstadoDefinicionFlujoRetirada
	retirada.RetiradaPor = actorID
	retirada.RetiradaEn = instante.UTC()
	retirada.RetiradaAprobacionRef = strings.TrimSpace(aprobacionRef)
	retirada.MotivoRetirada = strings.TrimSpace(motivo)
	return retirada.ClonarCanonico()
}

func (d DefinicionFlujo) NuevaVersion(version int, creadorID, fuenteRef, motivo string, instante time.Time) (DefinicionFlujo, error) {
	if err := d.Validar(); err != nil {
		return DefinicionFlujo{}, err
	}
	creadorID = strings.TrimSpace(creadorID)
	if d.Estado == EstadoDefinicionFlujoBorrador || version != d.Version+1 ||
		!referenciaDocumentalValida(creadorID) || !textoAcotadoCatalogo(fuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || instante.IsZero() ||
		instante.Before(ultimaFechaGobiernoFlujo(d)) {
		return DefinicionFlujo{}, ErrTransicionFlujoInvalida
	}
	nueva := DefinicionFlujo{
		ID:                              d.ID,
		Version:                         version,
		Revision:                        1,
		VersionAnteriorRef:              d.Referencia(),
		ModuloID:                        d.ModuloID,
		TipoEntidad:                     d.TipoEntidad,
		Nombre:                          d.Nombre,
		Descripcion:                     d.Descripcion,
		FuenteRef:                       strings.TrimSpace(fuenteRef),
		MotivoCreacion:                  strings.TrimSpace(motivo),
		EstadoInicial:                   d.EstadoInicial,
		AccionInicio:                    d.AccionInicio,
		GarantiaInicio:                  d.GarantiaInicio,
		PermiteFinalizacionTrasRetirada: d.PermiteFinalizacionTrasRetirada,
		Estados:                         d.Estados,
		Transiciones:                    d.Transiciones,
		Estado:                          EstadoDefinicionFlujoBorrador,
		CreadaPor:                       creadorID,
		CreadaEn:                        instante.UTC(),
	}
	return nueva.ClonarCanonico()
}

type ConfiguracionBorradorFlujo struct {
	Nombre                          string
	Descripcion                     string
	FuenteRef                       string
	EstadoInicial                   string
	AccionInicio                    string
	GarantiaInicio                  AuthAssurance
	PermiteFinalizacionTrasRetirada bool
	Estados                         []EstadoFlujoConfigurable
	Transiciones                    []TransicionFlujoConfigurable
}

func (d DefinicionFlujo) ActualizarBorrador(
	revisionEsperada int,
	actorID, motivo string,
	configuracion ConfiguracionBorradorFlujo,
	instante time.Time,
) (DefinicionFlujo, error) {
	if err := d.Validar(); err != nil {
		return DefinicionFlujo{}, err
	}
	actorID = strings.TrimSpace(actorID)
	if d.Estado != EstadoDefinicionFlujoBorrador || revisionEsperada != d.Revision ||
		!referenciaDocumentalValida(actorID) || !textoAcotadoCatalogo(configuracion.Nombre, maximoCaracteresEtiqueta, true) ||
		!textoAcotadoCatalogo(configuracion.Descripcion, maximoCaracteresDescripcion, false) ||
		!textoAcotadoCatalogo(configuracion.FuenteRef, maximoCaracteresReferenciaDocumental, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) ||
		(configuracion.EstadoInicial != "" && !esClaveDocumentalCanonica(configuracion.EstadoInicial)) ||
		!esClaveDocumentalCanonica(configuracion.AccionInicio) || !configuracion.GarantiaInicio.Valida() ||
		instante.IsZero() || instante.Before(d.CreadaEn) ||
		(!d.UltimaModificacionEn.IsZero() && instante.Before(d.UltimaModificacionEn)) {
		return DefinicionFlujo{}, ErrTransicionFlujoInvalida
	}
	actualizada := d
	actualizada.Revision++
	actualizada.Nombre = strings.TrimSpace(configuracion.Nombre)
	actualizada.Descripcion = strings.TrimSpace(configuracion.Descripcion)
	actualizada.FuenteRef = strings.TrimSpace(configuracion.FuenteRef)
	actualizada.EstadoInicial = strings.TrimSpace(configuracion.EstadoInicial)
	actualizada.AccionInicio = strings.TrimSpace(configuracion.AccionInicio)
	actualizada.GarantiaInicio = configuracion.GarantiaInicio
	actualizada.PermiteFinalizacionTrasRetirada = configuracion.PermiteFinalizacionTrasRetirada
	actualizada.Estados = append([]EstadoFlujoConfigurable(nil), configuracion.Estados...)
	actualizada.Transiciones = append([]TransicionFlujoConfigurable(nil), configuracion.Transiciones...)
	actualizada.UltimaModificacionPor = actorID
	actualizada.UltimaModificacionEn = instante.UTC()
	actualizada.MotivoModificacion = strings.TrimSpace(motivo)
	return actualizada.ClonarCanonico()
}

func (d DefinicionFlujo) ObtenerTransicion(clave, estadoActual string) (TransicionFlujoConfigurable, error) {
	if err := d.Validar(); err != nil {
		return TransicionFlujoConfigurable{}, err
	}
	if d.Estado != EstadoDefinicionFlujoPublicada &&
		!(d.Estado == EstadoDefinicionFlujoRetirada && d.PermiteFinalizacionTrasRetirada) {
		return TransicionFlujoConfigurable{}, ErrDefinicionFlujoNoPublicada
	}
	clave, estadoActual = strings.TrimSpace(clave), strings.TrimSpace(estadoActual)
	if !esClaveDocumentalCanonica(clave) || !esClaveDocumentalCanonica(estadoActual) {
		return TransicionFlujoConfigurable{}, ErrTransicionFlujoInvalida
	}
	for _, transicion := range d.Transiciones {
		if transicion.Clave == clave && transicion.AdmiteOrigen(estadoActual) {
			transicion.Desde = append([]string(nil), transicion.Desde...)
			transicion.Atributos = clonarAtributosCatalogo(transicion.Atributos)
			return transicion, nil
		}
	}
	return TransicionFlujoConfigurable{}, ErrTransicionFlujoInvalida
}

type DecisionReglaFlujo struct {
	DecisionRef                     string    `json:"decision_ref"`
	Concedida                       bool      `json:"concedida"`
	Codigo                          string    `json:"codigo"`
	DefinicionRef                   string    `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string    `json:"definicion_contenido_huella_sha256"`
	InstanciaRef                    string    `json:"instancia_ref"`
	InstanciaRevision               int       `json:"instancia_revision"`
	EstadoOrigen                    string    `json:"estado_origen"`
	TransicionClave                 string    `json:"transicion_clave"`
	ReglaRef                        string    `json:"regla_ref"`
	ActorID                         string    `json:"actor_id"`
	Finalidad                       string    `json:"finalidad"`
	CorrelacionRef                  string    `json:"correlacion_ref"`
	EntradaHuellaHMAC               string    `json:"entrada_huella_hmac"`
	ResultadoHuellaSHA256           string    `json:"resultado_huella_sha256"`
	EvaluadaEn                      time.Time `json:"evaluada_en"`
	ValidaHasta                     time.Time `json:"valida_hasta"`
}

func (d DecisionReglaFlujo) Validar() error {
	if !referenciaDocumentalValida(d.DecisionRef) || !textoAcotadoCatalogo(d.Codigo, maximoCaracteresEtiqueta, true) ||
		!referenciaDocumentalValida(d.DefinicionRef) || !esSHA256(d.DefinicionContenidoHuellaSHA256) ||
		!referenciaDocumentalValida(d.InstanciaRef) || d.InstanciaRevision < 1 ||
		!esClaveDocumentalCanonica(d.EstadoOrigen) || !esClaveDocumentalCanonica(d.TransicionClave) ||
		!referenciaDocumentalValida(d.ReglaRef) || !referenciaDocumentalValida(d.ActorID) ||
		!textoAcotadoCatalogo(d.Finalidad, maximoCaracteresDescripcion, true) ||
		!referenciaDocumentalValida(d.CorrelacionRef) || !esHuellaHMACSHA256(d.EntradaHuellaHMAC) ||
		!esSHA256(d.ResultadoHuellaSHA256) || d.EvaluadaEn.IsZero() || d.ValidaHasta.IsZero() ||
		!d.ValidaHasta.After(d.EvaluadaEn) || d.ValidaHasta.Sub(d.EvaluadaEn) > vigenciaMaximaDecisionRegla {
		return ErrDecisionReglaInvalida
	}
	return nil
}

func (d DecisionReglaFlujo) VigenteEn(instante time.Time) bool {
	return !instante.UTC().Before(d.EvaluadaEn.UTC()) && instante.UTC().Before(d.ValidaHasta.UTC())
}

func (d DecisionReglaFlujo) HuellaSHA256() (string, error) {
	if err := d.Validar(); err != nil {
		return "", err
	}
	canonica := d
	canonica.EvaluadaEn = d.EvaluadaEn.UTC()
	canonica.ValidaHasta = d.ValidaHasta.UTC()
	contenido, err := json.Marshal(canonica)
	if err != nil {
		return "", ErrDecisionReglaInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// EvidenciaAprobacionFlujo fija una aprobacion independiente a la revision y
// estado exactos que fueron revisados. Un simple identificador recibido del
// cliente nunca basta para satisfacer el doble control.
type EvidenciaAprobacionFlujo struct {
	AprobacionRef                   string        `json:"aprobacion_ref"`
	AprobadorID                     string        `json:"aprobador_id"`
	PerfilAprobadorRef              string        `json:"perfil_aprobador_ref"`
	Garantia                        AuthAssurance `json:"garantia"`
	SolicitanteID                   string        `json:"solicitante_id"`
	DefinicionRef                   string        `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string        `json:"definicion_contenido_huella_sha256"`
	InstanciaRef                    string        `json:"instancia_ref"`
	InstanciaRevision               int           `json:"instancia_revision"`
	EstadoOrigen                    string        `json:"estado_origen"`
	TransicionClave                 string        `json:"transicion_clave"`
	DecisionReglaRef                string        `json:"decision_regla_ref"`
	Motivo                          string        `json:"motivo"`
	EvidenciaHuellaSHA256           string        `json:"evidencia_huella_sha256"`
	AprobadaEn                      time.Time     `json:"aprobada_en"`
	ValidaHasta                     time.Time     `json:"valida_hasta"`
}

func (e EvidenciaAprobacionFlujo) Validar() error {
	if !referenciaDocumentalValida(e.AprobacionRef) || !referenciaDocumentalValida(e.AprobadorID) ||
		!referenciaDocumentalValida(e.PerfilAprobadorRef) || !e.Garantia.Valida() ||
		!referenciaDocumentalValida(e.SolicitanteID) || e.AprobadorID == e.SolicitanteID ||
		!referenciaDocumentalValida(e.DefinicionRef) || !esSHA256(e.DefinicionContenidoHuellaSHA256) ||
		!referenciaDocumentalValida(e.InstanciaRef) || e.InstanciaRevision < 1 ||
		!esClaveDocumentalCanonica(e.EstadoOrigen) || !esClaveDocumentalCanonica(e.TransicionClave) ||
		!referenciaDocumentalValida(e.DecisionReglaRef) ||
		!textoAcotadoCatalogo(e.Motivo, maximoCaracteresDescripcion, true) || !esSHA256(e.EvidenciaHuellaSHA256) ||
		e.AprobadaEn.IsZero() || e.ValidaHasta.IsZero() || !e.ValidaHasta.After(e.AprobadaEn) ||
		e.ValidaHasta.Sub(e.AprobadaEn) > vigenciaMaximaDecisionRegla {
		return ErrAprobacionFlujoInvalida
	}
	return nil
}

func (e EvidenciaAprobacionFlujo) VigenteEn(instante time.Time) bool {
	return !instante.UTC().Before(e.AprobadaEn.UTC()) && instante.UTC().Before(e.ValidaHasta.UTC())
}

func (e EvidenciaAprobacionFlujo) HuellaSHA256() (string, error) {
	if err := e.Validar(); err != nil {
		return "", err
	}
	canonica := e
	canonica.AprobadaEn = e.AprobadaEn.UTC()
	canonica.ValidaHasta = e.ValidaHasta.UTC()
	contenido, err := json.Marshal(canonica)
	if err != nil {
		return "", ErrAprobacionFlujoInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

type InstanciaFlujo struct {
	ID                              string    `json:"id"`
	TipoEntidad                     string    `json:"tipo_entidad"`
	EntidadRef                      string    `json:"entidad_ref"`
	DefinicionRef                   string    `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string    `json:"definicion_contenido_huella_sha256"`
	EstadoActual                    string    `json:"estado_actual"`
	Revision                        int       `json:"revision"`
	CreadaPor                       string    `json:"creada_por"`
	CreadaEn                        time.Time `json:"creada_en"`
	UltimaTransicionClave           string    `json:"ultima_transicion_clave,omitempty"`
	UltimaDecisionReglaRef          string    `json:"ultima_decision_regla_ref,omitempty"`
	UltimaAutorizacionRef           string    `json:"ultima_autorizacion_ref,omitempty"`
	UltimaAprobacionRef             string    `json:"ultima_aprobacion_ref,omitempty"`
	UltimaCorrelacionRef            string    `json:"ultima_correlacion_ref,omitempty"`
	UltimoMotivo                    string    `json:"ultimo_motivo,omitempty"`
	ActualizadaPor                  string    `json:"actualizada_por,omitempty"`
	ActualizadaEn                   time.Time `json:"actualizada_en,omitempty"`
}

func (i InstanciaFlujo) Validar() error {
	if !referenciaDocumentalValida(i.ID) || !esClaveDocumentalCanonica(i.TipoEntidad) ||
		!referenciaDocumentalValida(i.EntidadRef) || !referenciaDocumentalValida(i.DefinicionRef) ||
		!esSHA256(i.DefinicionContenidoHuellaSHA256) || !esClaveDocumentalCanonica(i.EstadoActual) || i.Revision < 1 ||
		!referenciaDocumentalValida(i.CreadaPor) || i.CreadaEn.IsZero() {
		return ErrInstanciaFlujoInvalida
	}
	if i.Revision == 1 {
		if i.UltimaTransicionClave != "" || i.UltimaDecisionReglaRef != "" || i.UltimaAutorizacionRef != "" ||
			i.UltimaAprobacionRef != "" || i.UltimaCorrelacionRef != "" || i.UltimoMotivo != "" ||
			i.ActualizadaPor != "" || !i.ActualizadaEn.IsZero() {
			return ErrInstanciaFlujoInvalida
		}
	} else if !esClaveDocumentalCanonica(i.UltimaTransicionClave) ||
		!referenciaDocumentalValida(i.UltimaDecisionReglaRef) || !referenciaDocumentalValida(i.UltimaAutorizacionRef) ||
		(i.UltimaAprobacionRef != "" && !referenciaDocumentalValida(i.UltimaAprobacionRef)) ||
		!referenciaDocumentalValida(i.UltimaCorrelacionRef) || !textoAcotadoCatalogo(i.UltimoMotivo, maximoCaracteresDescripcion, true) ||
		!referenciaDocumentalValida(i.ActualizadaPor) || i.ActualizadaEn.IsZero() || i.ActualizadaEn.Before(i.CreadaEn) {
		return ErrInstanciaFlujoInvalida
	}
	return nil
}

func (i InstanciaFlujo) HuellaSHA256() (string, error) {
	if err := i.Validar(); err != nil {
		return "", err
	}
	canonico := i
	canonico.CreadaEn = i.CreadaEn.UTC()
	canonico.ActualizadaEn = fechaCatalogoUTC(i.ActualizadaEn)
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrInstanciaFlujoInvalida
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func IniciarInstanciaFlujo(definicion DefinicionFlujo, id, entidadRef, actorID string, instante time.Time) (InstanciaFlujo, error) {
	if err := definicion.Validar(); err != nil {
		return InstanciaFlujo{}, err
	}
	if definicion.Estado != EstadoDefinicionFlujoPublicada || !referenciaDocumentalValida(id) ||
		!referenciaDocumentalValida(entidadRef) || !referenciaDocumentalValida(actorID) || instante.IsZero() ||
		instante.Before(definicion.PublicadaEn) {
		return InstanciaFlujo{}, ErrInstanciaFlujoInvalida
	}
	huella, err := definicion.HuellaContenidoSHA256()
	if err != nil {
		return InstanciaFlujo{}, err
	}
	instancia := InstanciaFlujo{
		ID:                              strings.TrimSpace(id),
		TipoEntidad:                     definicion.TipoEntidad,
		EntidadRef:                      strings.TrimSpace(entidadRef),
		DefinicionRef:                   definicion.Referencia(),
		DefinicionContenidoHuellaSHA256: huella,
		EstadoActual:                    definicion.EstadoInicial,
		Revision:                        1,
		CreadaPor:                       strings.TrimSpace(actorID),
		CreadaEn:                        instante.UTC(),
	}
	if err := instancia.Validar(); err != nil {
		return InstanciaFlujo{}, err
	}
	return instancia, nil
}

type CambioEstadoFlujo struct {
	InstanciaRef      string `json:"instancia_ref"`
	RevisionAnterior  int    `json:"revision_anterior"`
	RevisionPosterior int    `json:"revision_posterior"`
	EstadoAnterior    string `json:"estado_anterior"`
	EstadoPosterior   string `json:"estado_posterior"`
	TransicionClave   string `json:"transicion_clave"`
	DecisionReglaRef  string `json:"decision_regla_ref"`
	AutorizacionRef   string `json:"autorizacion_ref"`
	HuellaAnterior    string `json:"huella_anterior"`
	HuellaPosterior   string `json:"huella_posterior"`
}

func (i InstanciaFlujo) AplicarTransicion(
	definicion DefinicionFlujo,
	transicionClave string,
	decisionRegla DecisionReglaFlujo,
	autorizacionRef, aprobacionRef, actorID, finalidad, motivo, correlacionRef string,
	instante time.Time,
) (InstanciaFlujo, CambioEstadoFlujo, error) {
	if err := i.Validar(); err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	if err := definicion.Validar(); err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	huellaDefinicion, err := definicion.HuellaContenidoSHA256()
	definicionUtilizable := definicion.Estado == EstadoDefinicionFlujoPublicada ||
		(definicion.Estado == EstadoDefinicionFlujoRetirada && definicion.PermiteFinalizacionTrasRetirada)
	if err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	if !definicionUtilizable {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrDefinicionFlujoNoPublicada
	}
	if i.DefinicionRef != definicion.Referencia() ||
		i.DefinicionContenidoHuellaSHA256 != huellaDefinicion || i.TipoEntidad != definicion.TipoEntidad {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrInstanciaFlujoInvalida
	}
	if i.Revision == 1 && i.EstadoActual != definicion.EstadoInicial {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrInstanciaFlujoInvalida
	}
	transicion, err := definicion.ObtenerTransicion(strings.TrimSpace(transicionClave), i.EstadoActual)
	if err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	if err := decisionRegla.Validar(); err != nil || instante.IsZero() || !decisionRegla.VigenteEn(instante) ||
		decisionRegla.EvaluadaEn.Before(i.CreadaEn) || decisionRegla.DefinicionRef != definicion.Referencia() ||
		decisionRegla.DefinicionContenidoHuellaSHA256 != huellaDefinicion ||
		decisionRegla.InstanciaRef != i.ID || decisionRegla.InstanciaRevision != i.Revision ||
		decisionRegla.EstadoOrigen != i.EstadoActual || decisionRegla.TransicionClave != transicion.Clave ||
		decisionRegla.ReglaRef != transicion.ReglaRef {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrDecisionReglaInvalida
	}
	if !decisionRegla.Concedida {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrReglaFlujoDenegada
	}
	autorizacionRef, aprobacionRef, actorID = strings.TrimSpace(autorizacionRef), strings.TrimSpace(aprobacionRef), strings.TrimSpace(actorID)
	finalidad, motivo, correlacionRef = strings.TrimSpace(finalidad), strings.TrimSpace(motivo), strings.TrimSpace(correlacionRef)
	if !referenciaDocumentalValida(autorizacionRef) || !referenciaDocumentalValida(actorID) ||
		!textoAcotadoCatalogo(finalidad, maximoCaracteresDescripcion, true) ||
		!textoAcotadoCatalogo(motivo, maximoCaracteresDescripcion, true) || !referenciaDocumentalValida(correlacionRef) ||
		(aprobacionRef != "" && !referenciaDocumentalValida(aprobacionRef)) ||
		instante.Before(i.CreadaEn) || (!i.ActualizadaEn.IsZero() && instante.Before(i.ActualizadaEn)) {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrInstanciaFlujoInvalida
	}
	if transicion.RequiereAprobacion && aprobacionRef == "" {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrAprobacionFlujoRequerida
	}
	if decisionRegla.ActorID != actorID || decisionRegla.Finalidad != finalidad || decisionRegla.CorrelacionRef != correlacionRef {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, ErrDecisionReglaInvalida
	}
	huellaAnterior, err := i.HuellaSHA256()
	if err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	actualizada := i
	actualizada.EstadoActual = transicion.Hacia
	actualizada.Revision++
	actualizada.UltimaTransicionClave = transicion.Clave
	actualizada.UltimaDecisionReglaRef = decisionRegla.DecisionRef
	actualizada.UltimaAutorizacionRef = autorizacionRef
	actualizada.UltimaAprobacionRef = aprobacionRef
	actualizada.UltimaCorrelacionRef = correlacionRef
	actualizada.UltimoMotivo = motivo
	actualizada.ActualizadaPor = actorID
	actualizada.ActualizadaEn = instante.UTC()
	if err := actualizada.Validar(); err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	huellaPosterior, err := actualizada.HuellaSHA256()
	if err != nil {
		return InstanciaFlujo{}, CambioEstadoFlujo{}, err
	}
	cambio := CambioEstadoFlujo{
		InstanciaRef:      i.ID,
		RevisionAnterior:  i.Revision,
		RevisionPosterior: actualizada.Revision,
		EstadoAnterior:    i.EstadoActual,
		EstadoPosterior:   actualizada.EstadoActual,
		TransicionClave:   transicion.Clave,
		DecisionReglaRef:  decisionRegla.DecisionRef,
		AutorizacionRef:   autorizacionRef,
		HuellaAnterior:    huellaAnterior,
		HuellaPosterior:   huellaPosterior,
	}
	return actualizada, cambio, nil
}

func validarElementosFlujo(d DefinicionFlujo) error {
	estados := make(map[string]EstadoFlujoConfigurable, len(d.Estados))
	totalBytes := len(d.ID) + len(d.VersionAnteriorRef) + len(d.ModuloID) + len(d.TipoEntidad) + len(d.Nombre) +
		len(d.Descripcion) + len(d.FuenteRef) + len(d.MotivoCreacion) + len(d.EstadoInicial) + len(d.AccionInicio) +
		len(d.CreadaPor) + len(d.UltimaModificacionPor) + len(d.MotivoModificacion) + len(d.PublicadaPor) +
		len(d.AprobacionRef) + len(d.MotivoPublicacion) + len(d.RetiradaPor) + len(d.RetiradaAprobacionRef) +
		len(d.MotivoRetirada)
	for _, estado := range d.Estados {
		if err := estado.Validar(); err != nil {
			return err
		}
		if _, repetido := estados[estado.Clave]; repetido {
			return ErrEstadoFlujoInvalido
		}
		estados[estado.Clave] = estado
		totalBytes += len(estado.Clave) + len(estado.Catalogo.Referencia()) + len(estado.Catalogo.CatalogoHuellaSHA256) +
			bytesAtributosFlujo(estado.Atributos)
	}
	if len(d.Estados) == 0 {
		if d.EstadoInicial != "" || len(d.Transiciones) != 0 {
			return ErrGrafoFlujoInvalido
		}
	} else {
		if _, existe := estados[d.EstadoInicial]; !existe {
			return ErrGrafoFlujoInvalido
		}
	}
	transiciones := make(map[string]struct{}, len(d.Transiciones))
	rutas := make(map[string]struct{})
	for _, transicion := range d.Transiciones {
		if err := transicion.Validar(); err != nil {
			return err
		}
		if _, repetida := transiciones[transicion.Clave]; repetida {
			return ErrTransicionFlujoInvalida
		}
		transiciones[transicion.Clave] = struct{}{}
		if _, existe := estados[transicion.Hacia]; !existe {
			return ErrTransicionFlujoInvalida
		}
		for _, origen := range transicion.Desde {
			if _, existe := estados[origen]; !existe {
				return ErrTransicionFlujoInvalida
			}
			claveRuta := origen + "\x00" + transicion.Accion + "\x00" + strconv.Itoa(transicion.Prioridad)
			if _, repetida := rutas[claveRuta]; repetida {
				return ErrTransicionFlujoInvalida
			}
			rutas[claveRuta] = struct{}{}
		}
		totalBytes += len(transicion.Clave) + len(transicion.Hacia) + len(transicion.Accion) + len(transicion.ReglaRef) +
			len(transicion.PlazoRef) + bytesAtributosFlujo(transicion.Atributos)
		for _, origen := range transicion.Desde {
			totalBytes += len(origen)
		}
	}
	if totalBytes > maximoBytesFlujo {
		return ErrDefinicionFlujoInvalida
	}
	return nil
}

func validarGrafoFlujo(d DefinicionFlujo) error {
	if len(d.Estados) < 2 || len(d.Transiciones) == 0 || d.EstadoInicial == "" {
		return ErrGrafoFlujoInvalido
	}
	estados := make(map[string]EstadoFlujoConfigurable, len(d.Estados))
	salidas := make(map[string][]string)
	entradas := make(map[string][]string)
	for _, estado := range d.Estados {
		estados[estado.Clave] = estado
	}
	for _, transicion := range d.Transiciones {
		for _, origen := range transicion.Desde {
			salidas[origen] = append(salidas[origen], transicion.Hacia)
			entradas[transicion.Hacia] = append(entradas[transicion.Hacia], origen)
		}
	}
	terminales := 0
	for clave, estado := range estados {
		if estado.Terminal {
			terminales++
			if len(salidas[clave]) != 0 {
				return ErrGrafoFlujoInvalido
			}
		} else if len(salidas[clave]) == 0 {
			return ErrGrafoFlujoInvalido
		}
	}
	if terminales == 0 || estados[d.EstadoInicial].Terminal {
		return ErrGrafoFlujoInvalido
	}
	visitados := map[string]struct{}{d.EstadoInicial: {}}
	cola := []string{d.EstadoInicial}
	for len(cola) > 0 {
		actual := cola[0]
		cola = cola[1:]
		for _, siguiente := range salidas[actual] {
			if _, visto := visitados[siguiente]; visto {
				continue
			}
			visitados[siguiente] = struct{}{}
			cola = append(cola, siguiente)
		}
	}
	if len(visitados) != len(estados) {
		return ErrGrafoFlujoInvalido
	}
	// Todo estado debe disponer de algun camino hacia un estado terminal. La
	// mera existencia de una salida no evita ciclos cerrados que inmovilicen un
	// expediente administrativo.
	alcanzanTerminal := make(map[string]struct{}, terminales)
	cola = cola[:0]
	for clave, estado := range estados {
		if estado.Terminal {
			alcanzanTerminal[clave] = struct{}{}
			cola = append(cola, clave)
		}
	}
	for len(cola) > 0 {
		actual := cola[0]
		cola = cola[1:]
		for _, anterior := range entradas[actual] {
			if _, visto := alcanzanTerminal[anterior]; visto {
				continue
			}
			alcanzanTerminal[anterior] = struct{}{}
			cola = append(cola, anterior)
		}
	}
	if len(alcanzanTerminal) != len(estados) {
		return ErrGrafoFlujoInvalido
	}
	return nil
}

func datosPublicacionFlujoPresentes(d DefinicionFlujo) bool {
	return d.PublicadaPor != "" || !d.PublicadaEn.IsZero() || d.AprobacionRef != "" || d.MotivoPublicacion != ""
}

func datosRetiradaFlujoPresentes(d DefinicionFlujo) bool {
	return d.RetiradaPor != "" || !d.RetiradaEn.IsZero() || d.RetiradaAprobacionRef != "" || d.MotivoRetirada != ""
}

func datosPublicacionFlujoValidos(d DefinicionFlujo) bool {
	return referenciaDocumentalValida(d.PublicadaPor) && !d.PublicadaEn.IsZero() && !d.PublicadaEn.Before(d.CreadaEn) &&
		(d.UltimaModificacionEn.IsZero() || !d.PublicadaEn.Before(d.UltimaModificacionEn)) &&
		textoAcotadoCatalogo(d.AprobacionRef, maximoCaracteresReferenciaDocumental, true) &&
		textoAcotadoCatalogo(d.MotivoPublicacion, maximoCaracteresDescripcion, true)
}

func ultimaFechaGobiernoFlujo(d DefinicionFlujo) time.Time {
	ultima := d.CreadaEn
	for _, candidata := range []time.Time{d.UltimaModificacionEn, d.PublicadaEn, d.RetiradaEn} {
		if candidata.After(ultima) {
			ultima = candidata
		}
	}
	return ultima
}

func datosRetiradaFlujoValidos(d DefinicionFlujo) bool {
	return referenciaDocumentalValida(d.RetiradaPor) && !d.RetiradaEn.IsZero() && !d.RetiradaEn.Before(d.PublicadaEn) &&
		textoAcotadoCatalogo(d.RetiradaAprobacionRef, maximoCaracteresReferenciaDocumental, true) &&
		textoAcotadoCatalogo(d.MotivoRetirada, maximoCaracteresDescripcion, true)
}

func atributosFlujoValidos(atributos map[string]string) bool {
	for clave, valor := range atributos {
		if !esClaveDocumentalCanonica(clave) || !textoAcotadoCatalogo(valor, maximoCaracteresAtributo, true) {
			return false
		}
	}
	return true
}

func bytesAtributosFlujo(atributos map[string]string) int {
	total := 0
	for clave, valor := range atributos {
		total += len(clave) + len(valor)
	}
	return total
}
