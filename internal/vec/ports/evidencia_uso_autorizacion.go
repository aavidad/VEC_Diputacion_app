package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	// ErrEvidenciaUsoDecisionAutorizacionInvalida se devuelve siempre que no
	// pueda demostrarse de forma positiva que una evidencia representa una
	// decision reforzada, concedida, completa y vigente.
	ErrEvidenciaUsoDecisionAutorizacionInvalida = errors.New("vec: evidencia de uso de decision de autorizacion invalida")
	// ErrSerializacionEvidenciaUsoAutorizacionProhibida evita que la capacidad
	// opaca termine por accidente en codecs de transporte, trazas o HTTP.
	ErrSerializacionEvidenciaUsoAutorizacionProhibida = errors.New("vec: serializacion de evidencia de uso de autorizacion prohibida")
)

const (
	// EsquemaHuellaDecisionAutorizacionReforzadaV1 identifica tanto el dominio
	// criptografico como el formato canonico. Cambiar el significado o los
	// campos de la representacion exige publicar otro esquema.
	EsquemaHuellaDecisionAutorizacionReforzadaV1 = "vec.autorizacion.decision.reforzada.v1.autenticacion-actor"
	// EsquemaHuellaDecisionAutorizacionReforzadaV2 añade los compromisos
	// versionados de solicitud completa y motivo verificable por separado.
	EsquemaHuellaDecisionAutorizacionReforzadaV2 = "vec.autorizacion.decision.reforzada.v2.solicitud-ligada"
	formatoInstanteDecisionAutorizacionV1        = "2006-01-02T15:04:05.000000Z"
)

// DatosEvidenciaUsoDecisionAutorizacion es una proyeccion defensiva para el
// adaptador duradero que vaya a consumir la decision dentro de la misma
// transaccion que el efecto de negocio. No es una autorizacion serializable ni
// permite reconstruir EvidenciaUsoDecisionAutorizacion.
//
// Decision siempre se devuelve como copia profunda y en orden canonico. El
// adaptador debe volver a comprobar en su propia transaccion la decision
// registrada, su huella, la configuracion vigente y la identidad del efecto.
type DatosEvidenciaUsoDecisionAutorizacion struct {
	EsquemaHuella          string
	HuellaDecisionSHA256   string
	Decision               domain.DecisionAutorizacion
	VerificadaEn           time.Time
	representacionCanonica []byte
}

// RepresentacionCanonica devuelve una copia de los bytes exactos sobre los
// que se calculo HuellaDecisionSHA256. Es una salida deliberada para
// adaptadores duraderos: les permite cotejar la decision registrada sin
// reimplementar ni divergir del formato privado del nucleo.
//
// La proyeccion completa sigue bloqueando codecs y formateo; este metodo
// no convierte la evidencia opaca en una capacidad reconstruible.
func (d DatosEvidenciaUsoDecisionAutorizacion) RepresentacionCanonica() ([]byte, error) {
	if len(d.representacionCanonica) == 0 ||
		d.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!esHuellaSHA256EvidenciaUsoAutorizacion(d.HuellaDecisionSHA256) {
		return nil, errorEvidenciaUsoDecisionAutorizacion()
	}
	suma := sha256.Sum256(d.representacionCanonica)
	if hex.EncodeToString(suma[:]) != d.HuellaDecisionSHA256 {
		return nil, errorEvidenciaUsoDecisionAutorizacion()
	}
	return append([]byte(nil), d.representacionCanonica...), nil
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacion) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacion) String() string {
	return "[DATOS-EVIDENCIA-USO-AUTORIZACION-INTERNOS]"
}

func (d DatosEvidenciaUsoDecisionAutorizacion) GoString() string { return d.String() }

func (d DatosEvidenciaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}

func (d DatosEvidenciaUsoDecisionAutorizacion) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

type datosEvidenciaUsoDecisionAutorizacion struct {
	DatosEvidenciaUsoDecisionAutorizacion
}

// EvidenciaUsoDecisionAutorizacion es una capacidad opaca e inmutable dentro
// del proceso. Su valor cero no es valido y sus campos no pueden rellenarse
// con un literal desde otro paquete.
//
// Esta evidencia solo fija la decision que debe consumirse. No sustituye el
// consumo atomico con el efecto de negocio ni acredita que una transaccion de
// base de datos haya llegado a COMMIT. Su huella no es una firma ni una
// atestacion del PDP: por si sola tampoco demuestra que el autorizador no haya
// sido suplantado. Esa procedencia exige cableado interno confiable y, cuando
// corresponda, una atestacion separada verificable por el adaptador duradero.
type EvidenciaUsoDecisionAutorizacion struct {
	datos *datosEvidenciaUsoDecisionAutorizacion
}

// NuevaEvidenciaUsoDecisionAutorizacion crea una evidencia exclusivamente a
// partir de una decision reforzada, concedida y vigente en verificadaEn. El
// instante procede del reloj confiable del servidor y debe estar en UTC con
// precision de microsegundo, igual que la persistencia PostgreSQL prevista.
//
// Mientras no exista una prueba tipada del cumplimiento de obligaciones, una
// decision que las contenga se deniega. Una futura ampliacion debe incorporarlas
// de forma positiva; nunca puede ignorarlas silenciosamente.
func NuevaEvidenciaUsoDecisionAutorizacion(
	decision domain.DecisionAutorizacion,
	verificadaEn time.Time,
) (EvidenciaUsoDecisionAutorizacion, error) {
	if decision.ValidarEvidenciaInstantanea() != nil || decision.TieneSolicitudLigadaV2() || !decision.Concedida ||
		!instanteEvidenciaUsoAutorizacionCanonico(verificadaEn) ||
		!decision.VigenteEn(verificadaEn) || contieneComodinDecisionAutorizacion(decision) ||
		len(decision.Obligaciones) != 0 {
		return EvidenciaUsoDecisionAutorizacion{}, errorEvidenciaUsoDecisionAutorizacion()
	}

	decisionCanonica := clonarDecisionAutorizacionCanonica(decision)
	if decisionCanonica.ValidarEvidenciaInstantanea() != nil || decisionCanonica.TieneSolicitudLigadaV2() ||
		!decisionCanonica.VigenteEn(verificadaEn) {
		return EvidenciaUsoDecisionAutorizacion{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	representacionCanonica, err := serializarDecisionAutorizacionReforzadaV1(decisionCanonica)
	if err != nil {
		return EvidenciaUsoDecisionAutorizacion{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	suma := sha256.Sum256(representacionCanonica)
	huella := hex.EncodeToString(suma[:])

	evidencia := EvidenciaUsoDecisionAutorizacion{
		datos: &datosEvidenciaUsoDecisionAutorizacion{
			DatosEvidenciaUsoDecisionAutorizacion: DatosEvidenciaUsoDecisionAutorizacion{
				EsquemaHuella:          EsquemaHuellaDecisionAutorizacionReforzadaV1,
				HuellaDecisionSHA256:   huella,
				Decision:               decisionCanonica,
				VerificadaEn:           verificadaEn,
				representacionCanonica: append([]byte(nil), representacionCanonica...),
			},
		},
	}
	if evidencia.validarEstructura() != nil {
		return EvidenciaUsoDecisionAutorizacion{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	return evidencia, nil
}

// Datos devuelve una copia profunda de la proyeccion necesaria para el
// adaptador. La comprobacion estructural no implica vigencia actual: antes de
// un efecto debe usarse ValidarEn y, en produccion, revalidarse y consumirse la
// decision dentro de la transaccion que confirma dicho efecto.
func (e EvidenciaUsoDecisionAutorizacion) Datos() (DatosEvidenciaUsoDecisionAutorizacion, error) {
	if e.validarEstructura() != nil {
		return DatosEvidenciaUsoDecisionAutorizacion{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	resultado := e.datos.DatosEvidenciaUsoDecisionAutorizacion
	resultado.Decision = clonarDecisionAutorizacionCanonica(resultado.Decision)
	resultado.representacionCanonica = append([]byte(nil), resultado.representacionCanonica...)
	return resultado, nil
}

// ValidarEn vuelve a comprobar alcance temporal usando un instante efectivo
// del servidor. Se rechaza tambien un reloj anterior a la primera verificacion:
// un retroceso temporal nunca recupera una capacidad ya emitida.
func (e EvidenciaUsoDecisionAutorizacion) ValidarEn(instante time.Time) error {
	if e.validarEstructura() != nil || instante.IsZero() {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	instante = instante.UTC()
	if instante.Before(e.datos.VerificadaEn) || !e.datos.Decision.VigenteEn(instante) {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	return nil
}

func (e EvidenciaUsoDecisionAutorizacion) validarEstructura() error {
	if e.datos == nil ||
		e.datos.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV1 ||
		!esHuellaSHA256EvidenciaUsoAutorizacion(e.datos.HuellaDecisionSHA256) ||
		!instanteEvidenciaUsoAutorizacionCanonico(e.datos.VerificadaEn) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	decision := e.datos.Decision
	if decision.ValidarEvidenciaInstantanea() != nil || decision.TieneSolicitudLigadaV2() || !decision.Concedida ||
		!decision.VigenteEn(e.datos.VerificadaEn) || contieneComodinDecisionAutorizacion(decision) ||
		len(decision.Obligaciones) != 0 {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	huella, err := huellaDecisionAutorizacionReforzadaV1(decision)
	representacion, errRepresentacion := serializarDecisionAutorizacionReforzadaV1(decision)
	if err != nil || errRepresentacion != nil || huella != e.datos.HuellaDecisionSHA256 ||
		!bytes.Equal(representacion, e.datos.representacionCanonica) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return nil
}

// La capacidad opaca no se serializa. Solo Datos permite una extraccion
// deliberada, defensiva y tipada dentro de un adaptador de salida.
func (EvidenciaUsoDecisionAutorizacion) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacion) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacion) String() string {
	return "[EVIDENCIA-USO-AUTORIZACION-INTERNA]"
}

func (e EvidenciaUsoDecisionAutorizacion) GoString() string { return e.String() }

func (e EvidenciaUsoDecisionAutorizacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}

func (e EvidenciaUsoDecisionAutorizacion) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

type politicaDecisionAutorizacionCanonicaV1 struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type vinculoAutenticacionActorCanonicoV1 struct {
	BloqueVersion                uint16 `json:"bloque_version"`
	AutenticacionRef             string `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string `json:"autenticacion_huella_sha256"`
	AsercionRef                  string `json:"asercion_ref"`
	SesionRef                    string `json:"sesion_ref"`
	ControlSesionRef             string `json:"control_sesion_ref"`
	ControlSesionRevision        uint64 `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string `json:"control_sesion_huella_sha256"`
	CuentaRef                    string `json:"cuenta_ref"`
	CuentaOrdinariaRef           string `json:"cuenta_ordinaria_ref"`
	PrincipalID                  string `json:"principal_id"`
	PerfilActivoRef              string `json:"perfil_activo_ref"`
	CuentaPrivilegiada           bool   `json:"cuenta_privilegiada"`
	Superficie                   string `json:"superficie"`
	MetodoObservado              string `json:"metodo_observado"`
	GarantiaObservada            string `json:"garantia_observada"`
	PoliticaGarantiaRef          string `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    string `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              string `json:"sesion_emitida_en"`
	SesionValidaHasta            string `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           string `json:"sesion_revalidada_en"`
	ContextoActorRef             string `json:"contexto_actor_ref"`
	ContextoActorVersion         uint64 `json:"contexto_actor_version"`
	ContextoActorHuellaSHA256    string `json:"contexto_actor_huella_sha256"`
}

// decisionAutorizacionCanonicaV1 evita depender del orden de mapas, del orden
// fisico de conjuntos o de la representacion variable de time.Time. Todos los
// campos de DecisionAutorizacion reforzada forman parte del compromiso.
type decisionAutorizacionCanonicaV1 struct {
	Esquema                               string                                   `json:"esquema"`
	DecisionRef                           string                                   `json:"decision_ref"`
	Concedida                             bool                                     `json:"concedida"`
	Codigo                                string                                   `json:"codigo"`
	PrincipalID                           string                                   `json:"principal_id"`
	PerfilActivoRef                       string                                   `json:"perfil_activo_ref"`
	Accion                                string                                   `json:"accion"`
	RecursoRef                            string                                   `json:"recurso_ref"`
	ModuloID                              string                                   `json:"modulo_id"`
	TipoRecurso                           string                                   `json:"tipo_recurso"`
	ContextoRecursoHuellaSHA256           string                                   `json:"contexto_recurso_huella_sha256"`
	Finalidad                             string                                   `json:"finalidad"`
	CorrelacionRef                        string                                   `json:"correlacion_ref"`
	VinculoAutenticacionActor             vinculoAutenticacionActorCanonicoV1      `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                                   `json:"asignacion_ref"`
	AsignacionHuellaSHA256                string                                   `json:"asignacion_huella_sha256"`
	VersionRolRef                         string                                   `json:"version_rol_ref"`
	VersionRolHuellaSHA256                string                                   `json:"version_rol_huella_sha256"`
	ControlVigenciaVersionRolRef          string                                   `json:"control_vigencia_version_rol_ref"`
	ControlVigenciaVersionRolRevision     uint64                                   `json:"control_vigencia_version_rol_revision"`
	ControlVigenciaVersionRolHuellaSHA256 string                                   `json:"control_vigencia_version_rol_huella_sha256"`
	RevisionCatalogoPoliticas             uint64                                   `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256         string                                   `json:"catalogo_politicas_huella_sha256"`
	PoliticasEvaluadas                    []politicaDecisionAutorizacionCanonicaV1 `json:"politicas_evaluadas"`
	PoliticasAplicables                   []politicaDecisionAutorizacionCanonicaV1 `json:"politicas_aplicables"`
	GarantiaMinima                        domain.AuthAssurance                     `json:"garantia_minima"`
	CamposPermitidos                      []string                                 `json:"campos_permitidos"`
	Obligaciones                          []string                                 `json:"obligaciones"`
	EmitidaEn                             string                                   `json:"emitida_en"`
	ValidaHasta                           string                                   `json:"valida_hasta"`
}

func huellaDecisionAutorizacionReforzadaV1(decision domain.DecisionAutorizacion) (string, error) {
	contenido, err := RepresentacionCanonicaDecisionAutorizacionReforzadaV1(decision)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

// RepresentacionCanonicaDecisionAutorizacionReforzadaV1 devuelve el unico
// perfil JSON comprometido por la huella de una decision reforzada. Esta
// proyeccion estrecha existe para que los adaptadores duraderos no repliquen el
// orden de conjuntos ni el formato UTC de microsegundo fijo.
//
// A diferencia de NuevaEvidenciaUsoDecisionAutorizacion, no acredita que las
// obligaciones hayan sido cumplidas ni convierte la decision en una capacidad
// consumible. Por ello admite decisiones validas con obligaciones y solo debe
// usarse para persistencia, cotejo o firma de la representacion.
func RepresentacionCanonicaDecisionAutorizacionReforzadaV1(
	decision domain.DecisionAutorizacion,
) ([]byte, error) {
	return serializarDecisionAutorizacionReforzadaV1(decision)
}

func serializarDecisionAutorizacionReforzadaV1(decision domain.DecisionAutorizacion) ([]byte, error) {
	if decision.ValidarEvidenciaInstantanea() != nil || decision.TieneSolicitudLigadaV2() ||
		contieneComodinDecisionAutorizacion(decision) {
		return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	politicasEvaluadas, err := politicasDecisionAutorizacionCanonicas(
		decision.PoliticasEvaluadasRefs,
		decision.PoliticasEvaluadasHuellasSHA256,
	)
	if err != nil {
		return nil, err
	}
	politicasAplicables, err := politicasDecisionAutorizacionCanonicas(
		decision.PoliticasRefs,
		decision.PoliticasHuellasSHA256,
	)
	if err != nil {
		return nil, err
	}
	campos := append([]string{}, decision.CamposPermitidos...)
	obligaciones := append([]string{}, decision.Obligaciones...)
	vinculo, err := vinculoAutenticacionActorDecisionCanonicoV1(decision.VinculoAutenticacionActor)
	if err != nil {
		return nil, err
	}
	sort.Strings(campos)
	sort.Strings(obligaciones)
	canonica := decisionAutorizacionCanonicaV1{
		Esquema:     EsquemaHuellaDecisionAutorizacionReforzadaV1,
		DecisionRef: decision.DecisionRef, Concedida: decision.Concedida, Codigo: decision.Codigo,
		PrincipalID: decision.PrincipalID, PerfilActivoRef: decision.PerfilActivoRef,
		Accion: decision.Accion, RecursoRef: decision.RecursoRef, ModuloID: decision.ModuloID,
		TipoRecurso: decision.TipoRecurso, ContextoRecursoHuellaSHA256: decision.ContextoRecursoHuellaSHA256,
		Finalidad: decision.Finalidad, CorrelacionRef: decision.CorrelacionRef,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             decision.AsignacionRef, AsignacionHuellaSHA256: decision.AsignacionHuellaSHA256,
		VersionRolRef: decision.VersionRolRef, VersionRolHuellaSHA256: decision.VersionRolHuellaSHA256,
		ControlVigenciaVersionRolRef:          decision.ControlVigenciaVersionRolRef,
		ControlVigenciaVersionRolRevision:     decision.ControlVigenciaVersionRolRevision,
		ControlVigenciaVersionRolHuellaSHA256: decision.ControlVigenciaVersionRolHuellaSHA256,
		RevisionCatalogoPoliticas:             decision.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:         decision.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadas:                    politicasEvaluadas, PoliticasAplicables: politicasAplicables,
		GarantiaMinima: decision.GarantiaMinima, CamposPermitidos: campos, Obligaciones: obligaciones,
		EmitidaEn:   decision.EmitidaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		ValidaHasta: decision.ValidaHasta.UTC().Format(formatoInstanteDecisionAutorizacionV1),
	}
	contenido, err := json.Marshal(canonica)
	if err != nil {
		return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return contenido, nil
}

func vinculoAutenticacionActorDecisionCanonicoV1(
	vinculo domain.VinculoAutenticacionActorV1,
) (vinculoAutenticacionActorCanonicoV1, error) {
	datos, err := vinculo.Datos()
	if err != nil {
		return vinculoAutenticacionActorCanonicoV1{}, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return vinculoAutenticacionActorCanonicoV1{
		BloqueVersion:             datos.BloqueVersion,
		AutenticacionRef:          datos.AutenticacionRef,
		AutenticacionHuellaSHA256: datos.AutenticacionHuellaSHA256,
		AsercionRef:               datos.AsercionRef, SesionRef: datos.SesionRef,
		ControlSesionRef: datos.ControlSesionRef, ControlSesionRevision: datos.ControlSesionRevision,
		ControlSesionHuellaSHA256: datos.ControlSesionHuellaSHA256,
		CuentaRef:                 datos.CuentaRef, CuentaOrdinariaRef: datos.CuentaOrdinariaRef,
		PrincipalID: datos.PrincipalID, PerfilActivoRef: datos.PerfilActivoRef,
		CuentaPrivilegiada: datos.CuentaPrivilegiada, Superficie: string(datos.Superficie),
		MetodoObservado: string(datos.MetodoObservado), GarantiaObservada: string(datos.GarantiaObservada),
		PoliticaGarantiaRef:          datos.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: datos.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    datos.AutenticacionVerificadaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		SesionEmitidaEn:              datos.SesionEmitidaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		SesionValidaHasta:            datos.SesionValidaHasta.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		SesionRevalidadaEn:           datos.SesionRevalidadaEn.UTC().Format(formatoInstanteDecisionAutorizacionV1),
		ContextoActorRef:             datos.ContextoActorRef, ContextoActorVersion: datos.ContextoActorVersion,
		ContextoActorHuellaSHA256: datos.ContextoActorHuellaSHA256,
	}, nil
}

func politicasDecisionAutorizacionCanonicas(
	referencias []string,
	huellas map[string]string,
) ([]politicaDecisionAutorizacionCanonicaV1, error) {
	ordenadas := append([]string{}, referencias...)
	sort.Strings(ordenadas)
	resultado := make([]politicaDecisionAutorizacionCanonicaV1, 0, len(ordenadas))
	for _, referencia := range ordenadas {
		huella, existe := huellas[referencia]
		if !existe || !esHuellaSHA256EvidenciaUsoAutorizacion(huella) {
			return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
		}
		resultado = append(resultado, politicaDecisionAutorizacionCanonicaV1{
			Referencia: referencia, HuellaSHA256: huella,
		})
	}
	if len(huellas) != len(resultado) {
		return nil, ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return resultado, nil
}

func clonarDecisionAutorizacionCanonica(decision domain.DecisionAutorizacion) domain.DecisionAutorizacion {
	copia := decision
	copia.PoliticasEvaluadasRefs = append([]string{}, decision.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string{}, decision.PoliticasRefs...)
	copia.CamposPermitidos = append([]string{}, decision.CamposPermitidos...)
	copia.Obligaciones = append([]string{}, decision.Obligaciones...)
	sort.Strings(copia.PoliticasEvaluadasRefs)
	sort.Strings(copia.PoliticasRefs)
	sort.Strings(copia.CamposPermitidos)
	sort.Strings(copia.Obligaciones)
	copia.PoliticasEvaluadasHuellasSHA256 = clonarMapaHuellaDecisionAutorizacion(decision.PoliticasEvaluadasHuellasSHA256)
	copia.PoliticasHuellasSHA256 = clonarMapaHuellaDecisionAutorizacion(decision.PoliticasHuellasSHA256)
	return copia
}

func clonarMapaHuellaDecisionAutorizacion(origen map[string]string) map[string]string {
	copia := make(map[string]string, len(origen))
	for referencia, huella := range origen {
		copia[referencia] = huella
	}
	return copia
}

func contieneComodinDecisionAutorizacion(decision domain.DecisionAutorizacion) bool {
	valores := []string{
		decision.DecisionRef, decision.Codigo, decision.PrincipalID, decision.PerfilActivoRef,
		decision.Accion, decision.RecursoRef, decision.ModuloID, decision.TipoRecurso,
		decision.ContextoRecursoHuellaSHA256, decision.Finalidad, decision.CorrelacionRef,
		decision.AsignacionRef, decision.AsignacionHuellaSHA256, decision.VersionRolRef,
		decision.VersionRolHuellaSHA256, decision.ControlVigenciaVersionRolRef,
		decision.ControlVigenciaVersionRolHuellaSHA256, decision.CatalogoPoliticasHuellaSHA256,
		string(decision.GarantiaMinima),
	}
	if vinculo, err := decision.VinculoAutenticacionActor.Datos(); err == nil {
		valores = append(valores,
			vinculo.AutenticacionRef, vinculo.AutenticacionHuellaSHA256,
			vinculo.AsercionRef, vinculo.SesionRef, vinculo.ControlSesionRef,
			vinculo.ControlSesionHuellaSHA256, vinculo.CuentaRef, vinculo.CuentaOrdinariaRef,
			vinculo.PrincipalID, vinculo.PerfilActivoRef,
			string(vinculo.Superficie), string(vinculo.MetodoObservado), string(vinculo.GarantiaObservada),
			vinculo.PoliticaGarantiaRef, vinculo.PoliticaGarantiaHuellaSHA256,
			vinculo.ContextoActorRef, vinculo.ContextoActorHuellaSHA256,
		)
	} else {
		return true
	}
	valores = append(valores, decision.PoliticasEvaluadasRefs...)
	valores = append(valores, decision.PoliticasRefs...)
	valores = append(valores, decision.CamposPermitidos...)
	valores = append(valores, decision.Obligaciones...)
	for referencia, huella := range decision.PoliticasEvaluadasHuellasSHA256 {
		valores = append(valores, referencia, huella)
	}
	for referencia, huella := range decision.PoliticasHuellasSHA256 {
		valores = append(valores, referencia, huella)
	}
	for _, valor := range valores {
		if strings.ContainsRune(valor, '*') {
			return true
		}
	}
	return false
}

func instanteEvidenciaUsoAutorizacionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func esHuellaSHA256EvidenciaUsoAutorizacion(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func errorEvidenciaUsoDecisionAutorizacion() error {
	return errors.Join(
		domain.ErrAutorizacionDenegada,
		domain.ErrDecisionAutorizacionInvalida,
		ErrEvidenciaUsoDecisionAutorizacionInvalida,
	)
}
