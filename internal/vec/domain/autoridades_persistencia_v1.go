package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"
)

var ErrEstadoPersistibleFuenteAutoridadInvalido = errors.New("vec: estado persistible de fuente de autoridad invalido")

const (
	esquemaEstadoPersistibleFuenteAutoridadV1 = "vec.fuente_autoridad.estado.v1"
	versionEstadoPersistibleFuenteAutoridadV1 = uint16(1)
)

// Los tipos persistibles V1 son una frontera de compatibilidad congelada. No
// deben sustituirse por alias ni contener tipos del modelo vivo: una evolucion
// del dominio exige un conversor nuevo y, si cambia el contrato almacenado, V2.
// Los instantes son textos RFC3339Nano canonicos y todas las listas se emiten,
// incluidas las vacias.
type estadoPersistibleFuenteAutoridadV1 struct {
	Esquema                      string                                  `json:"esquema"`
	FormatoVersion               uint16                                  `json:"formato_version"`
	ID                           string                                  `json:"id"`
	Version                      uint64                                  `json:"version"`
	Revision                     uint64                                  `json:"revision"`
	VersionAnterior              linajePersistibleAutoridadV1            `json:"version_anterior"`
	Contenido                    contenidoPersistibleAutoridadV1         `json:"contenido"`
	HuellaContenidoInicialSHA256 string                                  `json:"huella_contenido_inicial_sha256"`
	HuellaHistoriaInicialSHA256  string                                  `json:"huella_historia_inicial_sha256"`
	Estado                       string                                  `json:"estado"`
	CreadaPor                    string                                  `json:"creada_por"`
	CreadaEn                     string                                  `json:"creada_en"`
	MotivoCreacionCodigo         string                                  `json:"motivo_creacion_codigo"`
	EdicionesBorrador            []edicionBorradorPersistibleAutoridadV1 `json:"ediciones_borrador"`
	Transiciones                 []transicionPersistibleAutoridadV1      `json:"transiciones"`
}

type referenciaPersistibleAutoridadV1 struct {
	FuenteID              string `json:"fuente_id"`
	Version               uint64 `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
}

type linajePersistibleAutoridadV1 struct {
	Fuente               referenciaPersistibleAutoridadV1 `json:"fuente"`
	Revision             uint64                           `json:"revision"`
	Estado               string                           `json:"estado"`
	HuellaHistoriaSHA256 string                           `json:"huella_historia_sha256"`
	HuellaEstadoSHA256   string                           `json:"huella_estado_sha256"`
}

type contenidoPersistibleAutoridadV1 struct {
	MateriaClave string                           `json:"materia_clave"`
	Nombre       string                           `json:"nombre"`
	Ambitos      []ambitoPersistibleAutoridadV1   `json:"ambitos"`
	Documento    documentoPersistibleAutoridadV1  `json:"documento"`
	Preceptos    []preceptoPersistibleAutoridadV1 `json:"preceptos"`
	Vigencia     periodoPersistibleAutoridadV1    `json:"vigencia"`
	Efectos      periodoPersistibleAutoridadV1    `json:"efectos"`
	ConocidaEn   string                           `json:"conocida_en"`
}

type ambitoPersistibleAutoridadV1 struct {
	DimensionClave string   `json:"dimension_clave"`
	ValoresClave   []string `json:"valores_clave"`
}

type documentoPersistibleAutoridadV1 struct {
	DocumentoID           string `json:"documento_id"`
	DocumentoVersion      uint64 `json:"documento_version"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	PublicacionOficialRef string `json:"publicacion_oficial_ref"`
	ActoOrigenRef         string `json:"acto_origen_ref"`
	OrganoEmisorRef       string `json:"organo_emisor_ref"`
}

type preceptoPersistibleAutoridadV1 struct {
	Clave string `json:"clave"`
	Cita  string `json:"cita"`
}

type periodoPersistibleAutoridadV1 struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta"`
}

type edicionBorradorPersistibleAutoridadV1 struct {
	RevisionAnterior              uint64 `json:"revision_anterior"`
	RevisionNueva                 uint64 `json:"revision_nueva"`
	ActorRef                      string `json:"actor_ref"`
	MotivoCodigo                  string `json:"motivo_codigo"`
	RegistradaEn                  string `json:"registrada_en"`
	HuellaContenidoAnteriorSHA256 string `json:"huella_contenido_anterior_sha256"`
	HuellaContenidoNuevaSHA256    string `json:"huella_contenido_nueva_sha256"`
	HuellaHistoriaAnteriorSHA256  string `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256     string `json:"huella_historia_nueva_sha256"`
}

type transicionPersistibleAutoridadV1 struct {
	Secuencia                    uint64                          `json:"secuencia"`
	EstadoAnterior               string                          `json:"estado_anterior"`
	EstadoNuevo                  string                          `json:"estado_nuevo"`
	ActorRef                     string                          `json:"actor_ref"`
	MotivoCodigo                 string                          `json:"motivo_codigo"`
	SolicitudRef                 string                          `json:"solicitud_ref"`
	PreparadaEn                  string                          `json:"preparada_en"`
	ExpiraEn                     string                          `json:"expira_en"`
	RegistradaEn                 string                          `json:"registrada_en"`
	Evidencia                    evidenciaPersistibleAutoridadV1 `json:"evidencia"`
	HuellaHistoriaAnteriorSHA256 string                          `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256    string                          `json:"huella_historia_nueva_sha256"`
}

type evidenciaPersistibleAutoridadV1 struct {
	EvidenciaRef                string   `json:"evidencia_ref"`
	Accion                      string   `json:"accion"`
	FuenteID                    string   `json:"fuente_id"`
	FuenteVersion               uint64   `json:"fuente_version"`
	HuellaContenidoSHA256       string   `json:"huella_contenido_sha256"`
	ActoRef                     string   `json:"acto_ref"`
	DocumentoRef                string   `json:"documento_ref"`
	RepresentacionRef           string   `json:"representacion_ref"`
	HuellaDocumentoSHA256       string   `json:"huella_documento_sha256"`
	OrganoRef                   string   `json:"organo_ref"`
	FirmasRefs                  []string `json:"firmas_refs"`
	ComprobadorRef              string   `json:"comprobador_ref"`
	AtestacionRef               string   `json:"atestacion_ref"`
	HuellaAtestacionSHA256      string   `json:"huella_atestacion_sha256"`
	FirmaAtestacionRef          string   `json:"firma_atestacion_ref"`
	HuellaCompromisoSHA256      string   `json:"huella_compromiso_sha256"`
	HuellaMensajeAtestadoSHA256 string   `json:"huella_mensaje_atestado_sha256"`
	ActoOcurridoEn              string   `json:"acto_ocurrido_en"`
	ComprobadaEn                string   `json:"comprobada_en"`
}

// Los sobres firmados o usados como entrada de una huella son contratos V1
// congelados. No contienen tipos vivos del dominio: añadir un campo al modelo
// en el futuro no puede alterar bytes ya firmados.
type sobreContenidoPersistibleAutoridadV1 struct {
	Esquema         string                          `json:"esquema"`
	ID              string                          `json:"id"`
	Version         uint64                          `json:"version"`
	VersionAnterior linajePersistibleAutoridadV1    `json:"version_anterior"`
	Contenido       contenidoPersistibleAutoridadV1 `json:"contenido"`
}

type compromisoPersistibleAutoridadV1 struct {
	Esquema                    string                           `json:"esquema"`
	SolicitudRef               string                           `json:"solicitud_ref"`
	Fuente                     referenciaPersistibleAutoridadV1 `json:"fuente"`
	RevisionPrevia             uint64                           `json:"revision_previa"`
	Secuencia                  uint64                           `json:"secuencia"`
	EstadoAnterior             string                           `json:"estado_anterior"`
	EstadoNuevo                string                           `json:"estado_nuevo"`
	Accion                     string                           `json:"accion"`
	ActorRef                   string                           `json:"actor_ref"`
	MotivoCodigo               string                           `json:"motivo_codigo"`
	HuellaHistoriaPreviaSHA256 string                           `json:"huella_historia_previa_sha256"`
	PreparadaEn                string                           `json:"preparada_en"`
	ExpiraEn                   string                           `json:"expira_en"`
}

type mensajeAtestacionPersistibleAutoridadV1 struct {
	Esquema               string                           `json:"esquema"`
	Compromiso            compromisoPersistibleAutoridadV1 `json:"compromiso"`
	EvidenciaRef          string                           `json:"evidencia_ref"`
	ActoRef               string                           `json:"acto_ref"`
	DocumentoRef          string                           `json:"documento_ref"`
	RepresentacionRef     string                           `json:"representacion_ref"`
	HuellaDocumentoSHA256 string                           `json:"huella_documento_sha256"`
	OrganoRef             string                           `json:"organo_ref"`
	FirmasRefs            []string                         `json:"firmas_refs"`
	ComprobadorRef        string                           `json:"comprobador_ref"`
	ActoOcurridoEn        string                           `json:"acto_ocurrido_en"`
	ComprobadaEn          string                           `json:"comprobada_en"`
}

type historiaInicialPersistibleAutoridadV1 struct {
	Esquema                       string `json:"esquema"`
	ID                            string `json:"id"`
	Version                       uint64 `json:"version"`
	VersionAnteriorFuenteID       string `json:"version_anterior_fuente_id"`
	VersionAnteriorNumero         uint64 `json:"version_anterior_numero"`
	VersionAnteriorHuellaSHA256   string `json:"version_anterior_huella_sha256"`
	VersionAnteriorRevision       uint64 `json:"version_anterior_revision"`
	VersionAnteriorEstado         string `json:"version_anterior_estado"`
	VersionAnteriorHistoriaSHA256 string `json:"version_anterior_historia_sha256"`
	VersionAnteriorEstadoSHA256   string `json:"version_anterior_estado_sha256"`
	HuellaContenidoInicialSHA256  string `json:"huella_contenido_inicial_sha256"`
	ActorRef                      string `json:"actor_ref"`
	MotivoCodigo                  string `json:"motivo_codigo"`
	RegistradaEn                  string `json:"registrada_en"`
}

type historiaEdicionPersistibleAutoridadV1 struct {
	Esquema                       string `json:"esquema"`
	RevisionAnterior              uint64 `json:"revision_anterior"`
	RevisionNueva                 uint64 `json:"revision_nueva"`
	ActorRef                      string `json:"actor_ref"`
	MotivoCodigo                  string `json:"motivo_codigo"`
	RegistradaEn                  string `json:"registrada_en"`
	HuellaContenidoAnteriorSHA256 string `json:"huella_contenido_anterior_sha256"`
	HuellaContenidoNuevaSHA256    string `json:"huella_contenido_nueva_sha256"`
	HuellaHistoriaAnteriorSHA256  string `json:"huella_historia_anterior_sha256"`
}

type historiaTransicionPersistibleAutoridadV1 struct {
	Esquema                      string `json:"esquema"`
	HuellaHistoriaAnteriorSHA256 string `json:"huella_historia_anterior_sha256"`
	HuellaCompromisoSHA256       string `json:"huella_compromiso_sha256"`
	EvidenciaRef                 string `json:"evidencia_ref"`
	HuellaMensajeAtestadoSHA256  string `json:"huella_mensaje_atestado_sha256"`
	AtestacionRef                string `json:"atestacion_ref"`
	HuellaAtestacionSHA256       string `json:"huella_atestacion_sha256"`
	FirmaAtestacionRef           string `json:"firma_atestacion_ref"`
	RegistradaEn                 string `json:"registrada_en"`
}

func serializarEstadoPersistibleFuenteAutoridadV1(f FuenteAutoridadVersionada) ([]byte, error) {
	resultado, err := json.Marshal(convertirAEstadoPersistibleAutoridadV1(f))
	if err != nil || len(resultado) == 0 || len(resultado) > maximoBytesEstadoAutoridad {
		return nil, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	return resultado, nil
}

func serializarContenidoPersistibleAutoridadV1(c ContenidoFuenteAutoridad) ([]byte, error) {
	resultado, err := json.Marshal(convertirAContenidoPersistibleAutoridadV1(c))
	if err != nil || len(resultado) == 0 || len(resultado) > maximoBytesContenidoAutoridad {
		return nil, ErrFuenteAutoridadInvalida
	}
	return resultado, nil
}

func serializarSobreContenidoPersistibleAutoridadV1(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	contenido ContenidoFuenteAutoridad,
) ([]byte, error) {
	resultado, err := json.Marshal(sobreContenidoPersistibleAutoridadV1{
		Esquema: "vec.fuente_autoridad.contenido.v1", ID: id, Version: uint64(version),
		VersionAnterior: convertirALinajePersistibleAutoridadV1(versionAnterior),
		Contenido:       convertirAContenidoPersistibleAutoridadV1(contenido),
	})
	if err != nil || len(resultado) == 0 || len(resultado) > maximoBytesSobreContenido {
		return nil, ErrFuenteAutoridadInvalida
	}
	return resultado, nil
}

func convertirACompromisoPersistibleAutoridadV1(
	c CompromisoTransicionFuenteAutoridadV1,
) compromisoPersistibleAutoridadV1 {
	return compromisoPersistibleAutoridadV1{
		Esquema: c.Esquema, SolicitudRef: c.SolicitudRef,
		Fuente:         convertirAReferenciaPersistibleAutoridadV1(c.Fuente),
		RevisionPrevia: c.RevisionPrevia, Secuencia: c.Secuencia,
		EstadoAnterior: string(c.EstadoAnterior), EstadoNuevo: string(c.EstadoNuevo),
		Accion: string(c.Accion), ActorRef: c.ActorRef, MotivoCodigo: string(c.MotivoCodigo),
		HuellaHistoriaPreviaSHA256: c.HuellaHistoriaPreviaSHA256,
		PreparadaEn:                textoInstantePersistibleAutoridadV1(c.PreparadaEn),
		ExpiraEn:                   textoInstantePersistibleAutoridadV1(c.ExpiraEn),
	}
}

func serializarCompromisoPersistibleAutoridadV1(c CompromisoTransicionFuenteAutoridadV1) ([]byte, error) {
	resultado, err := json.Marshal(convertirACompromisoPersistibleAutoridadV1(c))
	if err != nil || len(resultado) == 0 || len(resultado) > maximoBytesSobreContenido {
		return nil, ErrTransicionAutoridadInvalida
	}
	return resultado, nil
}

func serializarMensajeAtestacionPersistibleAutoridadV1(
	m MensajeAtestacionActoFuenteAutoridadV1,
) ([]byte, error) {
	firmas := append([]string(nil), m.FirmasRefs...)
	resultado, err := json.Marshal(mensajeAtestacionPersistibleAutoridadV1{
		Esquema: m.Esquema, Compromiso: convertirACompromisoPersistibleAutoridadV1(m.Compromiso),
		EvidenciaRef: m.EvidenciaRef, ActoRef: m.ActoRef, DocumentoRef: m.DocumentoRef,
		RepresentacionRef: m.RepresentacionRef, HuellaDocumentoSHA256: m.HuellaDocumentoSHA256,
		OrganoRef: m.OrganoRef, FirmasRefs: firmas, ComprobadorRef: m.ComprobadorRef,
		ActoOcurridoEn: textoInstantePersistibleAutoridadV1(m.ActoOcurridoEn),
		ComprobadaEn:   textoInstantePersistibleAutoridadV1(m.ComprobadaEn),
	})
	if err != nil || len(resultado) == 0 || len(resultado) > maximoBytesSobreContenido {
		return nil, ErrEvidenciaActoAutoridadInvalida
	}
	return resultado, nil
}

// RehidratarSolicitudTransicionFuenteAutoridadV1 permite reanudar de forma
// segura una operación de Portafirmas tras un callback o reinicio. Solo acepta
// exactamente los bytes canónicos que produjo BytesCanonicos.
func RehidratarSolicitudTransicionFuenteAutoridadV1(
	datos []byte,
) (SolicitudTransicionFuenteAutoridadV1, error) {
	if len(datos) == 0 || len(datos) > maximoBytesSobreContenido || validarEstructuraJSONAutoridadV1(datos) != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	var persistible compromisoPersistibleAutoridadV1
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&persistible); err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return SolicitudTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	compromiso, err := convertirDesdeCompromisoPersistibleAutoridadV1(persistible)
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	solicitud, err := nuevaSolicitudTransicionFuenteAutoridadV1(compromiso)
	if err != nil || !bytes.Equal(datos, solicitud.bytesCanonicos) {
		return SolicitudTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	return solicitud, nil
}

func convertirDesdeCompromisoPersistibleAutoridadV1(
	c compromisoPersistibleAutoridadV1,
) (CompromisoTransicionFuenteAutoridadV1, error) {
	preparadaEn, err := interpretarInstantePersistibleAutoridadV1(c.PreparadaEn, false)
	if err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	expiraEn, err := interpretarInstantePersistibleAutoridadV1(c.ExpiraEn, false)
	if err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	compromiso := CompromisoTransicionFuenteAutoridadV1{
		Esquema: c.Esquema, SolicitudRef: c.SolicitudRef,
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: c.Fuente.FuenteID, Version: c.Fuente.Version,
			HuellaContenidoSHA256: c.Fuente.HuellaContenidoSHA256,
		},
		RevisionPrevia: c.RevisionPrevia, Secuencia: c.Secuencia,
		EstadoAnterior: EstadoFuenteAutoridad(c.EstadoAnterior), EstadoNuevo: EstadoFuenteAutoridad(c.EstadoNuevo),
		Accion: AccionActoFuenteAutoridad(c.Accion), ActorRef: c.ActorRef,
		MotivoCodigo:               CodigoMotivoFuenteAutoridad(c.MotivoCodigo),
		HuellaHistoriaPreviaSHA256: c.HuellaHistoriaPreviaSHA256,
		PreparadaEn:                preparadaEn, ExpiraEn: expiraEn,
	}
	if err := compromiso.Validar(); err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	return compromiso, nil
}

// EstadoPersistibleV1 devuelve el unico JSON aceptado para la version V1. La
// validacion no serializa el agregado completo; este metodo lo hace una sola
// vez despues de obtener una copia defensiva canonica.
func (f FuenteAutoridadVersionada) EstadoPersistibleV1() ([]byte, error) {
	canonica, _, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return nil, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	resultado, err := serializarEstadoPersistibleFuenteAutoridadV1(canonica)
	if err != nil {
		return nil, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	return append([]byte(nil), resultado...), nil
}

// RehidratarFuenteAutoridadV1 solo acepta la representacion byte a byte
// canonica de V1. Rechaza extensiones, campos repetidos, espacios, ordenes de
// listas no canonicos y datos que el agregado vivo no pueda validar.
func RehidratarFuenteAutoridadV1(datos []byte) (FuenteAutoridadVersionada, error) {
	if len(datos) == 0 || len(datos) > maximoBytesEstadoAutoridad || validarEstructuraJSONAutoridadV1(datos) != nil {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}

	var estado estadoPersistibleFuenteAutoridadV1
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&estado); err != nil {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	if estado.Esquema != esquemaEstadoPersistibleFuenteAutoridadV1 ||
		estado.FormatoVersion != versionEstadoPersistibleFuenteAutoridadV1 ||
		!listasPersistiblesAutoridadV1Presentes(estado) {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}

	fuente, err := convertirDesdeEstadoPersistibleAutoridadV1(estado)
	if err != nil {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	canonica, _, err := fuente.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	bytesCanonicos, err := serializarEstadoPersistibleFuenteAutoridadV1(canonica)
	if err != nil || !bytes.Equal(datos, bytesCanonicos) {
		return FuenteAutoridadVersionada{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	return canonica, nil
}

func convertirAEstadoPersistibleAutoridadV1(f FuenteAutoridadVersionada) estadoPersistibleFuenteAutoridadV1 {
	ediciones := make([]edicionBorradorPersistibleAutoridadV1, len(f.EdicionesBorrador))
	for indice, edicion := range f.EdicionesBorrador {
		ediciones[indice] = edicionBorradorPersistibleAutoridadV1{
			RevisionAnterior: edicion.RevisionAnterior, RevisionNueva: edicion.RevisionNueva,
			ActorRef: edicion.ActorRef, MotivoCodigo: string(edicion.MotivoCodigo),
			RegistradaEn:                  textoInstantePersistibleAutoridadV1(edicion.RegistradaEn),
			HuellaContenidoAnteriorSHA256: edicion.HuellaContenidoAnteriorSHA256,
			HuellaContenidoNuevaSHA256:    edicion.HuellaContenidoNuevaSHA256,
			HuellaHistoriaAnteriorSHA256:  edicion.HuellaHistoriaAnteriorSHA256,
			HuellaHistoriaNuevaSHA256:     edicion.HuellaHistoriaNuevaSHA256,
		}
	}
	transiciones := make([]transicionPersistibleAutoridadV1, len(f.Transiciones))
	for indice, transicion := range f.Transiciones {
		transiciones[indice] = transicionPersistibleAutoridadV1{
			Secuencia: transicion.Secuencia, EstadoAnterior: string(transicion.EstadoAnterior),
			EstadoNuevo: string(transicion.EstadoNuevo), ActorRef: transicion.ActorRef,
			MotivoCodigo:                 string(transicion.MotivoCodigo),
			SolicitudRef:                 transicion.SolicitudRef,
			PreparadaEn:                  textoInstantePersistibleAutoridadV1(transicion.PreparadaEn),
			ExpiraEn:                     textoInstantePersistibleAutoridadV1(transicion.ExpiraEn),
			RegistradaEn:                 textoInstantePersistibleAutoridadV1(transicion.RegistradaEn),
			Evidencia:                    convertirAEvidenciaPersistibleAutoridadV1(transicion.Evidencia),
			HuellaHistoriaAnteriorSHA256: transicion.HuellaHistoriaAnteriorSHA256,
			HuellaHistoriaNuevaSHA256:    transicion.HuellaHistoriaNuevaSHA256,
		}
	}
	return estadoPersistibleFuenteAutoridadV1{
		Esquema:        esquemaEstadoPersistibleFuenteAutoridadV1,
		FormatoVersion: versionEstadoPersistibleFuenteAutoridadV1,
		ID:             f.ID, Version: uint64(f.Version), Revision: f.Revision,
		VersionAnterior:              convertirALinajePersistibleAutoridadV1(f.VersionAnterior),
		Contenido:                    convertirAContenidoPersistibleAutoridadV1(f.Contenido),
		HuellaContenidoInicialSHA256: f.HuellaContenidoInicialSHA256,
		HuellaHistoriaInicialSHA256:  f.HuellaHistoriaInicialSHA256,
		Estado:                       string(f.Estado), CreadaPor: f.CreadaPor,
		CreadaEn:             textoInstantePersistibleAutoridadV1(f.CreadaEn),
		MotivoCreacionCodigo: string(f.MotivoCreacionCodigo),
		EdicionesBorrador:    ediciones, Transiciones: transiciones,
	}
}

func convertirAReferenciaPersistibleAutoridadV1(r ReferenciaFuenteAutoridad) referenciaPersistibleAutoridadV1 {
	return referenciaPersistibleAutoridadV1{
		FuenteID: r.FuenteID, Version: uint64(r.Version), HuellaContenidoSHA256: r.HuellaContenidoSHA256,
	}
}

func convertirALinajePersistibleAutoridadV1(r ReferenciaLinajeFuenteAutoridad) linajePersistibleAutoridadV1 {
	return linajePersistibleAutoridadV1{
		Fuente: convertirAReferenciaPersistibleAutoridadV1(r.Fuente), Revision: r.Revision,
		Estado: string(r.Estado), HuellaHistoriaSHA256: r.HuellaHistoriaSHA256,
		HuellaEstadoSHA256: r.HuellaEstadoSHA256,
	}
}

func convertirAContenidoPersistibleAutoridadV1(c ContenidoFuenteAutoridad) contenidoPersistibleAutoridadV1 {
	ambitos := make([]ambitoPersistibleAutoridadV1, len(c.Ambitos))
	for indice, ambito := range c.Ambitos {
		valores := make([]string, len(ambito.ValoresClave))
		copy(valores, ambito.ValoresClave)
		ambitos[indice] = ambitoPersistibleAutoridadV1{DimensionClave: ambito.DimensionClave, ValoresClave: valores}
	}
	preceptos := make([]preceptoPersistibleAutoridadV1, len(c.Preceptos))
	for indice, precepto := range c.Preceptos {
		preceptos[indice] = preceptoPersistibleAutoridadV1{Clave: precepto.Clave, Cita: precepto.Cita}
	}
	return contenidoPersistibleAutoridadV1{
		MateriaClave: c.MateriaClave, Nombre: c.Nombre, Ambitos: ambitos,
		Documento: documentoPersistibleAutoridadV1{
			DocumentoID: c.Documento.DocumentoID, DocumentoVersion: uint64(c.Documento.DocumentoVersion),
			RepresentacionRef:     c.Documento.RepresentacionRef,
			HuellaContenidoSHA256: c.Documento.HuellaContenidoSHA256,
			PublicacionOficialRef: c.Documento.PublicacionOficialRef,
			ActoOrigenRef:         c.Documento.ActoOrigenRef, OrganoEmisorRef: c.Documento.OrganoEmisorRef,
		},
		Preceptos:  preceptos,
		Vigencia:   convertirAPeriodoPersistibleAutoridadV1(c.Vigencia),
		Efectos:    convertirAPeriodoPersistibleAutoridadV1(c.Efectos),
		ConocidaEn: textoInstantePersistibleAutoridadV1(c.ConocidaEn),
	}
}

func convertirAPeriodoPersistibleAutoridadV1(p PeriodoFuenteAutoridad) periodoPersistibleAutoridadV1 {
	return periodoPersistibleAutoridadV1{
		Desde: textoInstantePersistibleAutoridadV1(p.Desde), Hasta: textoInstantePersistibleAutoridadV1(p.Hasta),
	}
}

func convertirAEvidenciaPersistibleAutoridadV1(e EvidenciaActoFuenteAutoridad) evidenciaPersistibleAutoridadV1 {
	firmas := make([]string, len(e.FirmasRefs))
	copy(firmas, e.FirmasRefs)
	return evidenciaPersistibleAutoridadV1{
		EvidenciaRef: e.EvidenciaRef, Accion: string(e.Accion), FuenteID: e.FuenteID,
		FuenteVersion: uint64(e.FuenteVersion), HuellaContenidoSHA256: e.HuellaContenidoSHA256,
		ActoRef: e.ActoRef, DocumentoRef: e.DocumentoRef, RepresentacionRef: e.RepresentacionRef,
		HuellaDocumentoSHA256: e.HuellaDocumentoSHA256, OrganoRef: e.OrganoRef, FirmasRefs: firmas,
		ComprobadorRef: e.ComprobadorRef, AtestacionRef: e.AtestacionRef,
		HuellaAtestacionSHA256: e.HuellaAtestacionSHA256, FirmaAtestacionRef: e.FirmaAtestacionRef,
		HuellaCompromisoSHA256:      e.HuellaCompromisoSHA256,
		HuellaMensajeAtestadoSHA256: e.HuellaMensajeAtestadoSHA256,
		ActoOcurridoEn:              textoInstantePersistibleAutoridadV1(e.ActoOcurridoEn),
		ComprobadaEn:                textoInstantePersistibleAutoridadV1(e.ComprobadaEn),
	}
}

func convertirDesdeEstadoPersistibleAutoridadV1(e estadoPersistibleFuenteAutoridadV1) (FuenteAutoridadVersionada, error) {
	creadaEn, err := interpretarInstantePersistibleAutoridadV1(e.CreadaEn, false)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	contenido, err := convertirDesdeContenidoPersistibleAutoridadV1(e.Contenido)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	ediciones := make([]EdicionBorradorFuenteAutoridad, len(e.EdicionesBorrador))
	for indice, edicion := range e.EdicionesBorrador {
		registradaEn, err := interpretarInstantePersistibleAutoridadV1(edicion.RegistradaEn, false)
		if err != nil {
			return FuenteAutoridadVersionada{}, err
		}
		ediciones[indice] = EdicionBorradorFuenteAutoridad{
			RevisionAnterior: edicion.RevisionAnterior, RevisionNueva: edicion.RevisionNueva,
			ActorRef: edicion.ActorRef, MotivoCodigo: CodigoMotivoFuenteAutoridad(edicion.MotivoCodigo),
			RegistradaEn:                  registradaEn,
			HuellaContenidoAnteriorSHA256: edicion.HuellaContenidoAnteriorSHA256,
			HuellaContenidoNuevaSHA256:    edicion.HuellaContenidoNuevaSHA256,
			HuellaHistoriaAnteriorSHA256:  edicion.HuellaHistoriaAnteriorSHA256,
			HuellaHistoriaNuevaSHA256:     edicion.HuellaHistoriaNuevaSHA256,
		}
	}
	transiciones := make([]TransicionFuenteAutoridad, len(e.Transiciones))
	for indice, transicion := range e.Transiciones {
		preparadaEn, err := interpretarInstantePersistibleAutoridadV1(transicion.PreparadaEn, false)
		if err != nil {
			return FuenteAutoridadVersionada{}, err
		}
		expiraEn, err := interpretarInstantePersistibleAutoridadV1(transicion.ExpiraEn, false)
		if err != nil {
			return FuenteAutoridadVersionada{}, err
		}
		registradaEn, err := interpretarInstantePersistibleAutoridadV1(transicion.RegistradaEn, false)
		if err != nil {
			return FuenteAutoridadVersionada{}, err
		}
		evidencia, err := convertirDesdeEvidenciaPersistibleAutoridadV1(transicion.Evidencia)
		if err != nil {
			return FuenteAutoridadVersionada{}, err
		}
		transiciones[indice] = TransicionFuenteAutoridad{
			Secuencia:      transicion.Secuencia,
			EstadoAnterior: EstadoFuenteAutoridad(transicion.EstadoAnterior),
			EstadoNuevo:    EstadoFuenteAutoridad(transicion.EstadoNuevo), ActorRef: transicion.ActorRef,
			MotivoCodigo: CodigoMotivoFuenteAutoridad(transicion.MotivoCodigo),
			SolicitudRef: transicion.SolicitudRef, PreparadaEn: preparadaEn, ExpiraEn: expiraEn,
			RegistradaEn: registradaEn,
			Evidencia:    evidencia, HuellaHistoriaAnteriorSHA256: transicion.HuellaHistoriaAnteriorSHA256,
			HuellaHistoriaNuevaSHA256: transicion.HuellaHistoriaNuevaSHA256,
		}
	}
	return FuenteAutoridadVersionada{
		ID: e.ID, Version: e.Version, Revision: e.Revision,
		VersionAnterior: ReferenciaLinajeFuenteAutoridad{
			Fuente: ReferenciaFuenteAutoridad{
				FuenteID: e.VersionAnterior.Fuente.FuenteID, Version: e.VersionAnterior.Fuente.Version,
				HuellaContenidoSHA256: e.VersionAnterior.Fuente.HuellaContenidoSHA256,
			},
			Revision: e.VersionAnterior.Revision, Estado: EstadoFuenteAutoridad(e.VersionAnterior.Estado),
			HuellaHistoriaSHA256: e.VersionAnterior.HuellaHistoriaSHA256,
			HuellaEstadoSHA256:   e.VersionAnterior.HuellaEstadoSHA256,
		},
		Contenido: contenido, HuellaContenidoInicialSHA256: e.HuellaContenidoInicialSHA256,
		HuellaHistoriaInicialSHA256: e.HuellaHistoriaInicialSHA256,
		Estado:                      EstadoFuenteAutoridad(e.Estado), CreadaPor: e.CreadaPor, CreadaEn: creadaEn,
		MotivoCreacionCodigo: CodigoMotivoFuenteAutoridad(e.MotivoCreacionCodigo),
		EdicionesBorrador:    ediciones, Transiciones: transiciones,
	}, nil
}

func convertirDesdeContenidoPersistibleAutoridadV1(c contenidoPersistibleAutoridadV1) (ContenidoFuenteAutoridad, error) {
	conocidaEn, err := interpretarInstantePersistibleAutoridadV1(c.ConocidaEn, false)
	if err != nil {
		return ContenidoFuenteAutoridad{}, err
	}
	vigencia, err := convertirDesdePeriodoPersistibleAutoridadV1(c.Vigencia)
	if err != nil {
		return ContenidoFuenteAutoridad{}, err
	}
	efectos, err := convertirDesdePeriodoPersistibleAutoridadV1(c.Efectos)
	if err != nil {
		return ContenidoFuenteAutoridad{}, err
	}
	ambitos := make([]AmbitoFuenteAutoridad, len(c.Ambitos))
	for indice, ambito := range c.Ambitos {
		valores := make([]string, len(ambito.ValoresClave))
		copy(valores, ambito.ValoresClave)
		ambitos[indice] = AmbitoFuenteAutoridad{DimensionClave: ambito.DimensionClave, ValoresClave: valores}
	}
	preceptos := make([]PreceptoFuenteAutoridad, len(c.Preceptos))
	for indice, precepto := range c.Preceptos {
		preceptos[indice] = PreceptoFuenteAutoridad{Clave: precepto.Clave, Cita: precepto.Cita}
	}
	return ContenidoFuenteAutoridad{
		MateriaClave: c.MateriaClave, Nombre: c.Nombre, Ambitos: ambitos,
		Documento: DocumentoFuenteAutoridad{
			DocumentoID: c.Documento.DocumentoID, DocumentoVersion: c.Documento.DocumentoVersion,
			RepresentacionRef:     c.Documento.RepresentacionRef,
			HuellaContenidoSHA256: c.Documento.HuellaContenidoSHA256,
			PublicacionOficialRef: c.Documento.PublicacionOficialRef,
			ActoOrigenRef:         c.Documento.ActoOrigenRef, OrganoEmisorRef: c.Documento.OrganoEmisorRef,
		},
		Preceptos: preceptos, Vigencia: vigencia, Efectos: efectos, ConocidaEn: conocidaEn,
	}, nil
}

func convertirDesdePeriodoPersistibleAutoridadV1(p periodoPersistibleAutoridadV1) (PeriodoFuenteAutoridad, error) {
	desde, err := interpretarInstantePersistibleAutoridadV1(p.Desde, false)
	if err != nil {
		return PeriodoFuenteAutoridad{}, err
	}
	hasta, err := interpretarInstantePersistibleAutoridadV1(p.Hasta, true)
	if err != nil {
		return PeriodoFuenteAutoridad{}, err
	}
	return PeriodoFuenteAutoridad{Desde: desde, Hasta: hasta}, nil
}

func convertirDesdeEvidenciaPersistibleAutoridadV1(e evidenciaPersistibleAutoridadV1) (EvidenciaActoFuenteAutoridad, error) {
	actoOcurridoEn, err := interpretarInstantePersistibleAutoridadV1(e.ActoOcurridoEn, false)
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	comprobadaEn, err := interpretarInstantePersistibleAutoridadV1(e.ComprobadaEn, false)
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	firmas := make([]string, len(e.FirmasRefs))
	copy(firmas, e.FirmasRefs)
	return EvidenciaActoFuenteAutoridad{
		EvidenciaRef: e.EvidenciaRef, Accion: AccionActoFuenteAutoridad(e.Accion), FuenteID: e.FuenteID,
		FuenteVersion: e.FuenteVersion, HuellaContenidoSHA256: e.HuellaContenidoSHA256,
		ActoRef: e.ActoRef, DocumentoRef: e.DocumentoRef, RepresentacionRef: e.RepresentacionRef,
		HuellaDocumentoSHA256: e.HuellaDocumentoSHA256, OrganoRef: e.OrganoRef, FirmasRefs: firmas,
		ComprobadorRef: e.ComprobadorRef, AtestacionRef: e.AtestacionRef,
		HuellaAtestacionSHA256: e.HuellaAtestacionSHA256, FirmaAtestacionRef: e.FirmaAtestacionRef,
		HuellaCompromisoSHA256:      e.HuellaCompromisoSHA256,
		HuellaMensajeAtestadoSHA256: e.HuellaMensajeAtestadoSHA256,
		ActoOcurridoEn:              actoOcurridoEn, ComprobadaEn: comprobadaEn,
	}, nil
}

func listasPersistiblesAutoridadV1Presentes(e estadoPersistibleFuenteAutoridadV1) bool {
	if e.EdicionesBorrador == nil || e.Transiciones == nil || e.Contenido.Ambitos == nil || e.Contenido.Preceptos == nil {
		return false
	}
	for _, ambito := range e.Contenido.Ambitos {
		if ambito.ValoresClave == nil {
			return false
		}
	}
	for _, transicion := range e.Transiciones {
		if transicion.Evidencia.FirmasRefs == nil {
			return false
		}
	}
	return true
}

func textoInstantePersistibleAutoridadV1(instante time.Time) string {
	if instante.IsZero() {
		return ""
	}
	return instante.UTC().Format(time.RFC3339Nano)
}

func interpretarInstantePersistibleAutoridadV1(texto string, opcional bool) (time.Time, error) {
	if texto == "" {
		if opcional {
			return time.Time{}, nil
		}
		return time.Time{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	instante, err := time.Parse(time.RFC3339Nano, texto)
	if err != nil || texto != instante.UTC().Format(time.RFC3339Nano) || !instanteFuenteAutoridadCanonico(instante) {
		return time.Time{}, ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	return instante, nil
}
