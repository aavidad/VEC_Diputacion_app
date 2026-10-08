package postgres

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const esquemaEfectoAltaV3 = "vec.contratacion-temporal.efecto-alta.v3"

// La declaración explícita conserva el orden de cada campo de v2. La necesidad
// se añade exclusivamente al final de solicitud cuando existe una instantánea.
type efectoAltaCanonicoV3 struct {
	Esquema         string                  `json:"esquema"`
	ReservaRef      string                  `json:"reserva_ref"`
	ExpedienteRef   string                  `json:"expediente_ref"`
	NumeroVisible   string                  `json:"numero_visible"`
	ReciboRef       string                  `json:"recibo_ref"`
	OrganizacionRef string                  `json:"organizacion_ref"`
	ActorRef        string                  `json:"actor_ref"`
	PerfilRef       string                  `json:"perfil_ref"`
	Version         uint64                  `json:"version"`
	Flujo           flujoAltaCanonico       `json:"flujo"`
	FaseActual      string                  `json:"fase_actual"`
	EstadoActual    string                  `json:"estado_actual"`
	Solicitud       solicitudAltaCanonicaV3 `json:"solicitud"`
	CreadoEn        string                  `json:"creado_en"`
	ActualizadoEn   string                  `json:"actualizado_en"`
	Actuacion       actuacionAltaCanonica   `json:"actuacion"`
}

type solicitudAltaCanonicaV3 struct {
	CentroRef          string                `json:"centro_ref"`
	ContactoRef        string                `json:"contacto_ref"`
	CategoriaRef       string                `json:"categoria_ref"`
	GrupoSubgrupo      string                `json:"grupo_subgrupo"`
	MotivoClave        string                `json:"motivo_clave"`
	Detalle            string                `json:"detalle"`
	Periodo            periodoAltaCanonico   `json:"periodo"`
	RC                 rcAltaCanonica        `json:"rc"`
	DocumentosAdjuntos []string              `json:"documentos_adjuntos"`
	Observaciones      string                `json:"observaciones"`
	Necesidad          necesidadAltaCanonica `json:"necesidad"`
}

type necesidadAltaCanonica struct {
	Esquema              string               `json:"esquema"`
	CatalogoRef          string               `json:"catalogo_ref"`
	CatalogoVersion      uint64               `json:"catalogo_version"`
	CatalogoHuellaSHA256 string               `json:"catalogo_huella_sha256"`
	CausaClave           domain.ClaveCatalogo `json:"causa_clave"`
	Periodo              periodoAltaCanonico  `json:"periodo"`
	JornadaMinutos       uint16               `json:"jornada_minutos"`
	Campos               map[string]string    `json:"campos"`
	CatalogoInstantanea  []byte               `json:"catalogo_instantanea"`
}

func construirEfectoAltaCanonicoV3(
	expediente domain.Expediente,
	candidatura ports.DatosCandidaturaAlta,
) efectoAltaCanonicoV3 {
	v2 := construirEfectoAltaCanonico(expediente, candidatura)
	n := expediente.Solicitud.Necesidad
	campos := n.Campos
	if campos == nil {
		campos = map[string]string{}
	}
	return efectoAltaCanonicoV3{
		Esquema:    esquemaEfectoAltaV3,
		ReservaRef: v2.ReservaRef, ExpedienteRef: v2.ExpedienteRef,
		NumeroVisible: v2.NumeroVisible, ReciboRef: v2.ReciboRef,
		OrganizacionRef: v2.OrganizacionRef, ActorRef: v2.ActorRef,
		PerfilRef: v2.PerfilRef, Version: v2.Version, Flujo: v2.Flujo,
		FaseActual: v2.FaseActual, EstadoActual: v2.EstadoActual,
		Solicitud: solicitudAltaCanonicaV3{
			CentroRef: v2.Solicitud.CentroRef, ContactoRef: v2.Solicitud.ContactoRef,
			CategoriaRef: v2.Solicitud.CategoriaRef, GrupoSubgrupo: v2.Solicitud.GrupoSubgrupo,
			MotivoClave: v2.Solicitud.MotivoClave, Detalle: v2.Solicitud.Detalle,
			Periodo: v2.Solicitud.Periodo, RC: v2.Solicitud.RC,
			DocumentosAdjuntos: v2.Solicitud.DocumentosAdjuntos,
			Observaciones:      v2.Solicitud.Observaciones,
			Necesidad: necesidadAltaCanonica{
				Esquema: n.Esquema, CatalogoRef: n.CatalogoRef,
				CatalogoVersion:      n.CatalogoVersion,
				CatalogoHuellaSHA256: n.CatalogoHuellaSHA256,
				CausaClave:           n.CausaClave, Periodo: canonPeriodoAlta(n.Periodo),
				JornadaMinutos: n.JornadaMinutos, Campos: campos,
				CatalogoInstantanea: n.CatalogoInstantanea,
			},
		},
		CreadoEn: v2.CreadoEn, ActualizadoEn: v2.ActualizadoEn,
		Actuacion: v2.Actuacion,
	}
}
