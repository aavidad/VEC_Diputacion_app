package domain

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const maximoDecisionesCoberturaGobernadas = 64

const (
	AccionDecidirCoberturaGobernada    ClaveCatalogo = "contratacion_temporal.cobertura.decidir"
	AccionRectificarCoberturaGobernada ClaveCatalogo = "contratacion_temporal.cobertura.rectificar"
)

type TipoDecisionCoberturaGobernada string

const (
	DecisionCoberturaInicial       TipoDecisionCoberturaGobernada = "inicial"
	DecisionCoberturaRectificacion TipoDecisionCoberturaGobernada = "rectificacion"
)

func (t TipoDecisionCoberturaGobernada) valido() bool {
	return t == DecisionCoberturaInicial ||
		t == DecisionCoberturaRectificacion
}

// MotivoGobernadoDecisionCobertura evita transportar explicaciones libres.
// La referencia fija catálogo, versión, huella y entrada; ClaveI18n presenta.
// Solo acredita autocoherencia e integridad: la aplicación debe reconsultar la
// publicación autoritativa y su vigencia antes de ordenar, y la persistencia
// durable debe repetir esa comprobación antes de producir cualquier efecto.
type MotivoGobernadoDecisionCobertura struct {
	ReferenciaCatalogo dominiovec.ReferenciaEntradaCatalogo `json:"referencia_catalogo"`
	ClaveI18n          ClaveCatalogo                        `json:"clave_i18n"`
}

func (m MotivoGobernadoDecisionCobertura) valido() bool {
	return m.ReferenciaCatalogo.Validar() == nil &&
		huellaValida(m.ReferenciaCatalogo.CatalogoHuellaSHA256) &&
		m.ClaveI18n.Valida()
}

func (m MotivoGobernadoDecisionCobertura) vacio() bool {
	return m == (MotivoGobernadoDecisionCobertura{})
}

// VinculoActuacionDecisionCobertura liga la decisión a la actuación exacta
// resuelta por el agregado. No contiene nombres ni otros datos identificativos.
type VinculoActuacionDecisionCobertura struct {
	Secuencia         uint64          `json:"secuencia"`
	VersionExpediente uint64          `json:"version_expediente"`
	AccionClave       ClaveCatalogo   `json:"accion_clave"`
	ActorRef          string          `json:"actor_ref"`
	UnidadRef         string          `json:"unidad_ref"`
	RealizadaEn       time.Time       `json:"realizada_en"`
	FaseOrigen        ClaveFase       `json:"fase_origen"`
	FaseDestino       ClaveFase       `json:"fase_destino"`
	EstadoOrigen      EstadoOperativo `json:"estado_origen"`
	EstadoDestino     EstadoOperativo `json:"estado_destino"`
	ReciboRef         string          `json:"recibo_ref"`
}

func (v VinculoActuacionDecisionCobertura) validar() error {
	if v.Secuencia < 2 || v.VersionExpediente < 2 ||
		v.Secuencia != v.VersionExpediente ||
		!v.AccionClave.Valida() || !referenciaValida(v.ActorRef) ||
		!referenciaValida(v.UnidadRef) ||
		!instanteCatalogoCoberturaValido(v.RealizadaEn) ||
		!v.FaseOrigen.Valida() || !v.FaseDestino.Valida() ||
		!v.EstadoOrigen.Valido() || !v.EstadoDestino.Valido() ||
		!referenciaValida(v.ReciboRef) {
		return ErrDatoInvalido
	}
	return nil
}

func (v VinculoActuacionDecisionCobertura) correspondeA(a Actuacion) bool {
	return v.validar() == nil &&
		v.Secuencia == a.Secuencia &&
		v.VersionExpediente == a.VersionExpediente &&
		v.AccionClave == a.AccionClave &&
		v.ActorRef == a.ActorRef &&
		v.UnidadRef == a.UnidadRef &&
		v.RealizadaEn.Equal(a.RealizadaEn) &&
		v.FaseOrigen == a.FaseOrigen &&
		v.FaseDestino == a.FaseDestino &&
		v.EstadoOrigen == a.EstadoOrigen &&
		v.EstadoDestino == a.EstadoDestino &&
		v.ReciboRef == a.ReciboRef
}

type CanonHuellaDecisionCoberturaGobernada struct {
	Dominio        string `json:"dominio"`
	VersionEsquema uint16 `json:"version_esquema"`
	Algoritmo      string `json:"algoritmo"`
}

func CanonHuellaDecisionCoberturaGobernadaV1() CanonHuellaDecisionCoberturaGobernada {
	return CanonHuellaDecisionCoberturaGobernada{
		Dominio:        "vec.dipgra.contratacion-temporal.decision-cobertura-gobernada",
		VersionEsquema: 1,
		Algoritmo:      "sha-256",
	}
}

func (c CanonHuellaDecisionCoberturaGobernada) valido() bool {
	return c == CanonHuellaDecisionCoberturaGobernadaV1()
}

// PublicacionDecisionCoberturaGobernada es estado transportable y minimizado.
// Su restauración verifica canon y huella, pero no afirma persistencia durable.
type PublicacionDecisionCoberturaGobernada struct {
	Referencia                        string                                `json:"referencia"`
	HuellaSHA256                      string                                `json:"huella_sha256"`
	Canon                             CanonHuellaDecisionCoberturaGobernada `json:"canon"`
	Tipo                              TipoDecisionCoberturaGobernada        `json:"tipo"`
	OrganizacionRef                   string                                `json:"organizacion_ref"`
	ExpedienteRef                     string                                `json:"expediente_ref"`
	VersionExpedienteOrigen           uint64                                `json:"version_expediente_origen"`
	VersionExpediente                 uint64                                `json:"version_expediente"`
	ActorRef                          string                                `json:"actor_ref"`
	PerfilRef                         string                                `json:"perfil_ref"`
	PropuestaRef                      string                                `json:"propuesta_ref"`
	PropuestaHuellaSHA256             string                                `json:"propuesta_huella_sha256"`
	PreparacionEvidenciasRef          string                                `json:"preparacion_evidencias_ref"`
	PreparacionEvidenciasHuellaSHA256 string                                `json:"preparacion_evidencias_huella_sha256"`
	AnalisisRef                       string                                `json:"analisis_ref"`
	AnalisisHuellaSHA256              string                                `json:"analisis_huella_sha256"`
	Catalogo                          IdentidadCatalogoViasCobertura        `json:"catalogo"`
	Politica                          IdentidadPoliticaDecisionCobertura    `json:"politica"`
	ViaElegida                        ClaveCatalogo                         `json:"via_elegida"`
	ViaRecomendada                    ClaveCatalogo                         `json:"via_recomendada"`
	Motivo                            MotivoGobernadoDecisionCobertura      `json:"motivo,omitempty"`
	PredecesoraRef                    string                                `json:"predecesora_ref,omitempty"`
	PredecesoraHuellaSHA256           string                                `json:"predecesora_huella_sha256,omitempty"`
	DecididaEn                        time.Time                             `json:"decidida_en"`
	Actuacion                         VinculoActuacionDecisionCobertura     `json:"actuacion"`
}

type DecisionCoberturaGobernada struct {
	publicacion PublicacionDecisionCoberturaGobernada
}

type DatosAdoptarDecisionCobertura struct {
	PerfilRef  string
	ViaElegida ClaveCatalogo
	Motivo     MotivoGobernadoDecisionCobertura
}

type DatosRectificarDecisionCobertura struct {
	PerfilRef               string
	ViaElegida              ClaveCatalogo
	Motivo                  MotivoGobernadoDecisionCobertura
	PredecesoraRef          string
	PredecesoraHuellaSHA256 string
}

type datosCrearDecisionCobertura struct {
	Tipo        TipoDecisionCoberturaGobernada
	Expediente  Expediente
	PerfilRef   string
	ViaElegida  ClaveCatalogo
	Motivo      MotivoGobernadoDecisionCobertura
	Predecesora *PublicacionDecisionCoberturaGobernada
	Propuesta   PropuestaDecisionCobertura
	Actuacion   DatosActuacion
}

func crearDecisionCoberturaGobernada(
	datos datosCrearDecisionCobertura,
) (DecisionCoberturaGobernada, error) {
	publicacionPropuesta := datos.Propuesta.Publicacion()
	vinculo, err := datos.Propuesta.VinculoParaDecision(
		datos.ViaElegida,
		datos.Actuacion.RealizadaEn,
	)
	if err != nil ||
		!datos.Tipo.valido() ||
		!referenciaValida(datos.PerfilRef) ||
		datos.Expediente.Analisis == nil ||
		datos.Actuacion.Observaciones != "" ||
		len(datos.Actuacion.DocumentosRef) != 0 ||
		publicacionPropuesta.OrganizacionRef != datos.Expediente.OrganizacionRef ||
		publicacionPropuesta.ExpedienteRef != datos.Expediente.Referencia ||
		publicacionPropuesta.VersionExpediente != datos.Expediente.Version ||
		publicacionPropuesta.CategoriaRef != datos.Expediente.Analisis.CategoriaRef ||
		publicacionPropuesta.Periodo != datos.Expediente.Analisis.Periodo ||
		accionEsperadaDecision(datos.Tipo) != datos.Actuacion.AccionClave ||
		(datos.ViaElegida != publicacionPropuesta.ViaPropuesta &&
			!datos.Motivo.valido()) {
		return DecisionCoberturaGobernada{}, ErrTransicionInvalida
	}
	if datos.Tipo == DecisionCoberturaInicial {
		if datos.Predecesora != nil ||
			(datos.ViaElegida == publicacionPropuesta.ViaPropuesta &&
				!datos.Motivo.vacio()) {
			return DecisionCoberturaGobernada{}, ErrTransicionInvalida
		}
	} else if datos.Predecesora == nil || !datos.Motivo.valido() {
		return DecisionCoberturaGobernada{}, ErrTransicionInvalida
	}
	publicacion := PublicacionDecisionCoberturaGobernada{
		Canon: CanonHuellaDecisionCoberturaGobernadaV1(), Tipo: datos.Tipo,
		OrganizacionRef:         datos.Expediente.OrganizacionRef,
		ExpedienteRef:           datos.Expediente.Referencia,
		VersionExpedienteOrigen: datos.Expediente.Version,
		VersionExpediente:       datos.Expediente.Version + 1,
		ActorRef:                datos.Actuacion.ActorRef, PerfilRef: datos.PerfilRef,
		PropuestaRef:                      vinculo.PropuestaRef,
		PropuestaHuellaSHA256:             vinculo.PropuestaHuella,
		PreparacionEvidenciasRef:          vinculo.PreparacionEvidenciasRef,
		PreparacionEvidenciasHuellaSHA256: vinculo.PreparacionEvidenciasHuellaSHA256,
		AnalisisRef:                       vinculo.AnalisisRef,
		AnalisisHuellaSHA256:              vinculo.AnalisisHuellaSHA256,
		Catalogo:                          vinculo.Catalogo, Politica: vinculo.Politica,
		ViaElegida:     datos.ViaElegida,
		ViaRecomendada: publicacionPropuesta.ViaPropuesta,
		Motivo:         datos.Motivo, DecididaEn: datos.Actuacion.RealizadaEn,
		Actuacion: VinculoActuacionDecisionCobertura{
			Secuencia:         uint64(len(datos.Expediente.Actuaciones) + 1),
			VersionExpediente: datos.Expediente.Version + 1,
			AccionClave:       datos.Actuacion.AccionClave,
			ActorRef:          datos.Actuacion.ActorRef,
			UnidadRef:         datos.Actuacion.UnidadRef,
			RealizadaEn:       datos.Actuacion.RealizadaEn,
			FaseOrigen:        datos.Expediente.FaseActual,
			FaseDestino:       datos.Actuacion.FaseDestino,
			EstadoOrigen:      datos.Expediente.EstadoActual,
			EstadoDestino:     datos.Actuacion.EstadoDestino,
			ReciboRef:         datos.Actuacion.ReciboRef,
		},
	}
	if datos.Predecesora != nil {
		publicacion.PredecesoraRef = datos.Predecesora.Referencia
		publicacion.PredecesoraHuellaSHA256 = datos.Predecesora.HuellaSHA256
	}
	publicacion.HuellaSHA256, err = calcularHuellaDecisionCobertura(publicacion)
	if err != nil {
		return DecisionCoberturaGobernada{}, ErrTransicionInvalida
	}
	publicacion.Referencia = referenciaDecisionCobertura(publicacion.HuellaSHA256)
	return DecisionCoberturaGobernada{publicacion: publicacion}, nil
}

func accionEsperadaDecision(
	tipo TipoDecisionCoberturaGobernada,
) ClaveCatalogo {
	if tipo == DecisionCoberturaInicial {
		return AccionDecidirCoberturaGobernada
	}
	if tipo == DecisionCoberturaRectificacion {
		return AccionRectificarCoberturaGobernada
	}
	return ""
}

func RestaurarDecisionCoberturaGobernada(
	publicacion PublicacionDecisionCoberturaGobernada,
) (DecisionCoberturaGobernada, error) {
	huella, err := calcularHuellaDecisionCobertura(publicacion)
	if err != nil || huella != publicacion.HuellaSHA256 ||
		publicacion.Referencia != referenciaDecisionCobertura(huella) {
		return DecisionCoberturaGobernada{}, ErrDatoInvalido
	}
	return DecisionCoberturaGobernada{publicacion: publicacion}, nil
}

func (d DecisionCoberturaGobernada) Publicacion() PublicacionDecisionCoberturaGobernada {
	return d.publicacion
}

func referenciaDecisionCobertura(huella string) string {
	return "decision-cobertura:sha256:" + huella
}
