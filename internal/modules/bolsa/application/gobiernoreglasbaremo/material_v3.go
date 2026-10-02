package gobiernoreglasbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrGobiernoV3NoDisponible     = errors.New("gobierno_reglas_v3_no_disponible")
	ErrGobiernoV3NoAutenticado    = errors.New("gobierno_reglas_v3_no_autenticado")
	ErrGobiernoV3Prohibido        = errors.New("gobierno_reglas_v3_prohibido")
	ErrGobiernoV3PeticionInvalida = errors.New("gobierno_reglas_v3_peticion_invalida")
)

// La composición nominal debe instalar y acreditar esta audiencia; declararla
// aquí no registra concesiones ni abre la puerta SQL histórica V2.
const AudienciaGobiernoBorradorReglasV3 = "vec_bolsa_reglas_baremo.gobierno_borrador.v3"
const esquemaMaterialGobiernoV3 = "vec.bolsa.gobierno-borrador.material.v3"
const operacionAltaGobiernoV3 = "alta_borrador"
const tipoRecursoIntencionGobiernoV3 = "intencion_gobierno_reglas_baremo"
const prefijoRecursoIntencionGobiernoV3 = "intencion-reglas-baremo:"
const operacionConsultaGobiernoV3 = "consultar_exacta"
const operacionRecuperarGobiernoV3 = "recuperar_recibo"

// CredencialesGobiernoV3 se construye desde identidad resuelta por el servidor.
// Nunca se deserializa actor, perfil o vínculo desde una petición de negocio.
type CredencialesGobiernoV3 struct {
	bloqueoSerializacion
	actor   vd.ContextoActor
	vinculo vd.VinculoAutenticacionActorV2
}

func NuevasCredencialesGobiernoV3(actor vd.ContextoActor, vinculo vd.VinculoAutenticacionActorV2) (CredencialesGobiernoV3, error) {
	copia, err := actor.Clonar()
	c := CredencialesGobiernoV3{actor: copia, vinculo: vinculo}
	if err != nil || c.validar(time.Time{}) != nil {
		return CredencialesGobiernoV3{}, ErrGobiernoV3NoAutenticado
	}
	return c, nil
}
func (c CredencialesGobiernoV3) validar(ahora time.Time) error {
	d, err := c.vinculo.Datos()
	h, eh := c.actor.HuellaSHA256VinculadaV2()
	if err != nil || eh != nil || c.actor.Validar() != nil || d.CuentaPrivilegiada || d.Superficie != vd.SuperficieAutenticacionInternaCorporativaV1 ||
		d.PrincipalID != c.actor.PersonaRef || d.PerfilActivoRef != c.actor.PerfilActivoRef || d.CuentaRef != c.actor.Instantanea.CuentaRef || d.CuentaOrdinariaRef != c.actor.Instantanea.CuentaRef ||
		d.ContextoActorRef != c.actor.Instantanea.VinculoRef || d.ContextoActorVersion != c.actor.Instantanea.VinculoVersion || d.ContextoActorCuentaVersion != c.actor.Instantanea.CuentaVersion || d.ContextoActorHuellaSHA256 != h ||
		d.MetodoObservado != c.actor.Principal.AuthMethod || d.GarantiaObservada != c.actor.Principal.AuthAssurance {
		return ErrGobiernoV3NoAutenticado
	}
	if !ahora.IsZero() && (!c.actor.Instantanea.VigenteEn(ahora) || ahora.Before(d.SesionRevalidadaEn) || !ahora.Before(d.SesionValidaHasta)) {
		return ErrGobiernoV3NoAutenticado
	}
	return nil
}

// La clave identifica una intención; nunca constituye autorización.
type PeticionAltaBorradorV3 struct {
	bloqueoSerializacion
	Conjunto       reglas.ConjuntoReglasBaremo
	Motivo         reglas.MotivoCatalogadoReglasBaremo
	ClaveOperacion string
}

// HuellaSolicitudAltaBorradorV3 permite cotejar el negocio restaurado antes
// del commit sin duplicar su representación canónica en los adaptadores.
// La huella identifica la intención; no acredita identidad ni autorización.
func HuellaSolicitudAltaBorradorV3(persona string, peticion PeticionAltaBorradorV3) (string, error) {
	return huellaNegocioAltaV3(persona, peticion)
}

type PeticionConsultaExactaV3 struct {
	bloqueoSerializacion
	Selector ports.SelectorGobiernoReglasV3
	Motivo   vd.ReferenciaEntradaCatalogo
}

// PeticionRecuperarReciboV3 exige el selector exacto de un recibo conocido.
// Si se pierde la primera respuesta, se reintenta GuardarAltaBorrador con
// la misma clave y contenido; el repositorio devuelve el recibo original.
type PeticionRecuperarReciboV3 struct {
	bloqueoSerializacion
	Selector              ports.SelectorGobiernoReglasV3
	Motivo                vd.ReferenciaEntradaCatalogo
	ClaveOperacion        string
	HuellaSolicitudSHA256 string
}

// DTO de transporte explícito, separado de la autoridad de dominio/V3.
// Las cadenas SHA-256 comprometen los bytes canónicos Go, no jsonb::text.
type EstadoMaterialGobiernoV3 struct {
	Referencia            string `json:"referencia"`
	Version               uint64 `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	Revision              uint64 `json:"revision"`
	HuellaEstadoSHA256    string `json:"huella_estado_sha256"`
}

type MaterialGobiernoV3 struct {
	Esquema               string                    `json:"esquema"`
	Operacion             string                    `json:"operacion"`
	Accion                string                    `json:"accion"`
	ModuloID              string                    `json:"modulo_id"`
	TipoRecurso           string                    `json:"tipo_recurso"`
	Finalidad             string                    `json:"finalidad"`
	PersonaRef            string                    `json:"persona_ref"`
	PerfilRef             string                    `json:"perfil_ref"`
	ConvocatoriaRef       string                    `json:"convocatoria_ref"`
	ExpedienteRef         string                    `json:"expediente_ref"`
	Estado                EstadoMaterialGobiernoV3  `json:"estado"`
	EstadoEsperado        *EstadoMaterialGobiernoV3 `json:"estado_esperado"`
	VersionCanonica       []byte                    `json:"version_canonica"`
	ClaveOperacion        string                    `json:"clave_operacion"`
	HuellaSolicitudSHA256 string                    `json:"huella_solicitud_sha256"`
	MotivoCanonico        []byte                    `json:"motivo_canonico"`
	SolicitadaEn          string                    `json:"solicitada_en"`
}

type negocioAltaGobiernoV3 struct {
	Esquema          string `json:"esquema"`
	Operacion        string `json:"operacion"`
	PersonaRef       string `json:"persona_ref"`
	ConvocatoriaRef  string `json:"convocatoria_ref"`
	ExpedienteRef    string `json:"expediente_ref"`
	ConjuntoCanonico []byte `json:"conjunto_canonico"`
	MotivoCanonico   []byte `json:"motivo_canonico"`
	ClaveOperacion   string `json:"clave_operacion"`
}

func claveGobiernoV3Valida(k string) bool {
	// Identificador opaco de 128 bits: 32 dígitos hexadecimales en minúsculas.
	if len(k) != 32 {
		return false
	}
	for _, r := range k {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
func estadoMaterialV3(v reglas.VinculoEstadoReglasBaremo) EstadoMaterialGobiernoV3 {
	r := v.Contenido()
	return EstadoMaterialGobiernoV3{r.Referencia(), r.Version(), r.HuellaSHA256(), v.Revision(), v.HuellaEstadoSHA256()}
}
func motivoGobiernoV3(m reglas.MotivoCatalogadoReglasBaremo) (vd.ReferenciaEntradaCatalogo, error) {
	r := m.Catalogo()
	version := r.Version()
	if version == 0 || version > 1<<31-1 {
		return vd.ReferenciaEntradaCatalogo{}, ErrGobiernoV3PeticionInvalida
	}
	motivo := vd.ReferenciaEntradaCatalogo{CatalogoID: r.Referencia(), CatalogoVersion: int(version), CatalogoHuellaSHA256: r.HuellaSHA256(), EntradaClave: m.Clave()}
	if !vd.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vd.ReferenciaEntradaCatalogo{}, ErrGobiernoV3PeticionInvalida
	}
	return motivo, nil
}
func validarSelectorGobiernoV3(s ports.SelectorGobiernoReglasV3) error {
	id := s.Identidad
	_, err := reglas.NuevaIdentidadConjuntoReglasBaremo(id.Referencia(), id.Version(), id.ConvocatoriaRef(), id.ExpedienteRef())
	if err != nil || s.Estado.Validar() != nil || s.Estado.Revision() != 1 || s.Estado.Contenido().Referencia() != id.Referencia() || s.Estado.Contenido().Version() != id.Version() {
		return ErrGobiernoV3PeticionInvalida
	}
	return nil
}
func huellaNegocioAltaV3(persona string, p PeticionAltaBorradorV3) (string, error) {
	conjunto, err := p.Conjunto.RepresentacionCanonica()
	m, em := motivoGobiernoV3(p.Motivo)
	if err != nil || em != nil || !claveGobiernoV3Valida(p.ClaveOperacion) {
		return "", ErrGobiernoV3PeticionInvalida
	}
	motivo, err := vd.RepresentacionCanonicaMotivoAutorizacionV2(m)
	if err != nil {
		return "", ErrGobiernoV3PeticionInvalida
	}
	id := p.Conjunto.Identidad()
	b, err := json.Marshal(negocioAltaGobiernoV3{"vec.bolsa.gobierno-borrador.intencion.v3", operacionAltaGobiernoV3, persona, id.ConvocatoriaRef(), id.ExpedienteRef(), conjunto, motivo, p.ClaveOperacion})
	if err != nil {
		return "", ErrGobiernoV3PeticionInvalida
	}
	return shaGobiernoV3(b), nil
}
func prepararMaterialGobiernoV3(c CredencialesGobiernoV3, selector ports.SelectorGobiernoReglasV3, operacion string, motivo vd.ReferenciaEntradaCatalogo, canon []byte, clave, huella string, ahora time.Time) (MaterialGobiernoV3, ports.SolicitudMaterialGobiernoReglasV3, error) {
	if validarSelectorGobiernoV3(selector) != nil || !vd.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return MaterialGobiernoV3{}, ports.SolicitudMaterialGobiernoReglasV3{}, ErrGobiernoV3PeticionInvalida
	}
	accion, finalidad, campos := "bolsa.reglas_baremo.version.consultar", finalidadConsultaReglas, []string{"estado_reglas_baremo"}
	if operacion == operacionAltaGobiernoV3 {
		accion, finalidad, campos = "bolsa.reglas_baremo.borrador.crear", finalidadGobiernoReglas, append([]string{}, camposGobiernoReglas...)
	} else if operacion == operacionRecuperarGobiernoV3 {
		accion = "bolsa.reglas_baremo.recibo.consultar"
		campos = []string{"estado_reglas_baremo", "recibo"}
	} else if operacion != operacionConsultaGobiernoV3 {
		return MaterialGobiernoV3{}, ports.SolicitudMaterialGobiernoReglasV3{}, ErrOperacionInvalida
	}
	motivoCanon, err := vd.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		return MaterialGobiernoV3{}, ports.SolicitudMaterialGobiernoReglasV3{}, ErrGobiernoV3PeticionInvalida
	}
	id := selector.Identidad
	tipo, prefijo, ref := tipoRecursoReglasGobernadas, "reglas-baremo:", selector.Estado.HuellaEstadoSHA256()
	if operacion == operacionAltaGobiernoV3 {
		tipo, prefijo, ref = tipoRecursoIntencionGobiernoV3, prefijoRecursoIntencionGobiernoV3, huella
	}
	material := MaterialGobiernoV3{Esquema: esquemaMaterialGobiernoV3, Operacion: operacion, Accion: accion, ModuloID: moduloBolsaGobiernoReglas, TipoRecurso: tipo, Finalidad: finalidad, PersonaRef: c.actor.PersonaRef, PerfilRef: c.actor.PerfilActivoRef, ConvocatoriaRef: id.ConvocatoriaRef(), ExpedienteRef: id.ExpedienteRef(), Estado: estadoMaterialV3(selector.Estado), VersionCanonica: bytes.Clone(canon), ClaveOperacion: clave, HuellaSolicitudSHA256: huella, MotivoCanonico: motivoCanon, SolicitadaEn: ahora.Format("2006-01-02T15:04:05.000000Z")}
	datos, err := json.Marshal(material)
	if err != nil {
		return MaterialGobiernoV3{}, ports.SolicitudMaterialGobiernoReglasV3{}, ErrGobiernoV3PeticionInvalida
	}
	recurso := vd.RecursoAutorizable{Referencia: prefijo + ref, ModuloID: moduloBolsaGobiernoReglas, Tipo: tipo, Ambitos: map[string]string{ambitoConvocatoriaRef: id.ConvocatoriaRef(), ambitoExpedienteRef: id.ExpedienteRef()}, Atributos: map[string]string{"material_sha256": shaGobiernoV3(datos)}}
	if recurso.Validar() != nil {
		return MaterialGobiernoV3{}, ports.SolicitudMaterialGobiernoReglasV3{}, ErrGobiernoV3PeticionInvalida
	}
	return material, ports.SolicitudMaterialGobiernoReglasV3{Operacion: operacion, Accion: accion, Finalidad: finalidad, Campos: campos, Audiencia: AudienciaGobiernoBorradorReglasV3, Recurso: recurso, Motivo: motivo, MaterialCanonico: datos}, nil
}
func referenciaReciboGobiernoV3(s string) bool {
	return len(s) >= 3 && len(s) <= 256 && strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\r\n\t*")
}
