package postgres

import (
	"encoding/json"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaEfectoAltaV2 = "vec.contratacion-temporal.efecto-alta.v2"
	esquemaSellosAltaV1 = "vec.contratacion-temporal.sellos-hmac.v1"
)

type efectoAltaV2 struct {
	Esquema         string            `json:"esquema"`
	ReservaRef      string            `json:"reserva_ref"`
	ExpedienteRef   string            `json:"expediente_ref"`
	NumeroVisible   string            `json:"numero_visible"`
	ReciboRef       string            `json:"recibo_ref"`
	OrganizacionRef string            `json:"organizacion_ref"`
	ActorRef        string            `json:"actor_ref"`
	PerfilRef       string            `json:"perfil_ref"`
	Version         uint64            `json:"version"`
	Flujo           flujoEfectoAltaV2 `json:"flujo"`
	FaseActual      string            `json:"fase_actual"`
	EstadoActual    string            `json:"estado_actual"`
	Solicitud       solicitudEfectoV2 `json:"solicitud"`
	CreadoEn        string            `json:"creado_en"`
	ActualizadoEn   string            `json:"actualizado_en"`
	Actuacion       actuacionEfectoV2 `json:"actuacion"`
}

type flujoEfectoAltaV2 struct {
	DefinicionRef string `json:"definicion_ref"`
	Version       uint64 `json:"version"`
	HuellaSHA256  string `json:"huella_sha256"`
}

type solicitudEfectoV2 struct {
	CentroRef          string          `json:"centro_ref"`
	ContactoRef        string          `json:"contacto_ref"`
	CategoriaRef       string          `json:"categoria_ref"`
	GrupoSubgrupo      string          `json:"grupo_subgrupo"`
	MotivoClave        string          `json:"motivo_clave"`
	Detalle            string          `json:"detalle"`
	Periodo            periodoEfectoV2 `json:"periodo"`
	RC                 rcEfectoV2      `json:"rc"`
	DocumentosAdjuntos []string        `json:"documentos_adjuntos"`
	Observaciones      string          `json:"observaciones"`
}

type periodoEfectoV2 struct {
	Inicio string `json:"inicio"`
	Fin    string `json:"fin"`
}

type rcEfectoV2 struct {
	Existe       bool            `json:"existe"`
	Numero       string          `json:"numero"`
	Fecha        string          `json:"fecha"`
	Importe      importeEfectoV2 `json:"importe"`
	DocumentoRef string          `json:"documento_ref"`
}

type importeEfectoV2 struct {
	Centimos int64  `json:"centimos"`
	Moneda   string `json:"moneda"`
}

type actuacionEfectoV2 struct {
	Secuencia         uint64   `json:"secuencia"`
	VersionExpediente uint64   `json:"version_expediente"`
	AccionClave       string   `json:"accion_clave"`
	ActorRef          string   `json:"actor_ref"`
	UnidadRef         string   `json:"unidad_ref"`
	ReciboRef         string   `json:"recibo_ref"`
	RealizadaEn       string   `json:"realizada_en"`
	FaseOrigen        string   `json:"fase_origen"`
	FaseDestino       string   `json:"fase_destino"`
	EstadoOrigen      string   `json:"estado_origen"`
	EstadoDestino     string   `json:"estado_destino"`
	Observaciones     string   `json:"observaciones"`
	DocumentosRef     []string `json:"documentos_ref"`
}

type sellosAltaV1 struct {
	Esquema   string        `json:"esquema"`
	Activo    parSellosV1   `json:"activo"`
	Retenidos []parSellosV1 `json:"retenidos"`
}

type parSellosV1 struct {
	Generacion uint32 `json:"generacion"`
	AmbitoHMAC string `json:"ambito_hmac"`
	HuellaHMAC string `json:"huella_hmac"`
}

type parametrosConfirmacionAlta struct {
	capacidad      []byte
	decision       []byte
	motivo         []byte
	contexto       []byte
	personaVersion string
	perfilVersion  string
	payload        []byte
	cose           []byte
	evidencia      []byte
	spki           []byte
	alta           []byte
	sellos         []byte
}

// ProyectorEfectoAltaV2 construye una sola vez el canon que se compromete en
// la solicitud V3 y que después cruza sin remarshalizar la frontera SQL.
type ProyectorEfectoAltaV2 struct{}

func NuevoProyectorEfectoAltaV2() *ProyectorEfectoAltaV2 {
	return &ProyectorEfectoAltaV2{}
}

func (p *ProyectorEfectoAltaV2) ProyectarEfectoAlta(
	solicitud ports.SolicitudProyectarEfectoAlta,
) (ports.ProyeccionEfectoAlta, error) {
	if p == nil || solicitud.Validar() != nil {
		return ports.ProyeccionEfectoAlta{},
			ports.ErrProyeccionEfectoAltaInvalida
	}
	contenido, err := canonEfectoAltaV2(solicitud)
	if err != nil {
		return ports.ProyeccionEfectoAlta{},
			ports.ErrProyeccionEfectoAltaInvalida
	}
	proyeccion, err := ports.NuevaProyeccionEfectoAlta(contenido)
	if err != nil {
		return ports.ProyeccionEfectoAlta{},
			ports.ErrProyeccionEfectoAltaInvalida
	}
	return proyeccion, nil
}

func construirParametrosConfirmacionAlta(
	orden ports.OrdenConfirmarAlta,
	material ports.MaterialConfirmacionAlta,
) (parametrosConfirmacionAlta, error) {
	evidenciaOrden, err := orden.Datos()
	datosSolicitud, errSolicitud := evidenciaOrden.SolicitudAutorizacionV3.Datos()
	contexto, errContexto := evidenciaOrden.ResultadoContextoV2.Clonar()
	datosMaterial, errMaterial := material.Datos()
	if err != nil || errSolicitud != nil || errContexto != nil ||
		errMaterial != nil {
		return parametrosConfirmacionAlta{}, ports.ErrOrdenAltaInvalida
	}
	decision, errDecision :=
		dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(
			evidenciaOrden.DecisionAutorizacionV3,
		)
	motivo, errMotivo := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(
		datosSolicitud.ReferenciaMotivo,
	)
	alta, huellaAlta, errAlta := evidenciaOrden.ProyeccionEfecto.Datos()
	sellos, errSellos := canonSellosAltaV1(evidenciaOrden)
	capacidad, errCapacidad :=
		datosMaterial.CapacidadVECAD3.ExportacionCanonicaParaConsumidor()
	if errDecision != nil || errMotivo != nil || errAlta != nil ||
		errSellos != nil || errCapacidad != nil ||
		len(capacidad) < 512 || len(capacidad) > 32*1024 ||
		len(decision) == 0 || len(decision) > 512*1024 ||
		len(motivo) == 0 || len(motivo) > 64*1024 ||
		len(contexto.RepresentacionCanonica) == 0 ||
		len(contexto.RepresentacionCanonica) > 256*1024 ||
		len(datosMaterial.PayloadVECAD3) == 0 ||
		len(datosMaterial.PayloadVECAD3) > 1024*1024 ||
		len(datosMaterial.SobreCOSESign1) == 0 ||
		len(datosMaterial.SobreCOSESign1) > 1024*1024 ||
		len(datosMaterial.EvidenciaVerificacion) == 0 ||
		len(datosMaterial.EvidenciaVerificacion) > 256*1024 ||
		len(datosMaterial.RaizPublicaSPKI) != 44 ||
		len(alta) < 256 || len(alta) > 32*1024 ||
		len(sellos) < 256 || len(sellos) > 8*1024 ||
		len(datosSolicitud.Recurso.Atributos) != 5 ||
		datosSolicitud.Recurso.Atributos[ports.AtributoHuellaEfectoAltaSHA256] !=
			huellaAlta {
		return parametrosConfirmacionAlta{},
			ports.ErrMaterialConfirmacionAltaInvalido
	}
	resultado := parametrosConfirmacionAlta{
		capacidad: append([]byte(nil), capacidad...),
		decision:  append([]byte(nil), decision...),
		motivo:    append([]byte(nil), motivo...),
		contexto:  append([]byte(nil), contexto.RepresentacionCanonica...),
		personaVersion: strconv.FormatUint(
			contexto.Contexto.Instantanea.PersonaVersion, 10,
		),
		perfilVersion: strconv.FormatUint(
			contexto.Contexto.Instantanea.PerfilVersion, 10,
		),
		payload: append([]byte(nil), datosMaterial.PayloadVECAD3...),
		cose:    append([]byte(nil), datosMaterial.SobreCOSESign1...),
		evidencia: append(
			[]byte(nil), datosMaterial.EvidenciaVerificacion...,
		),
		spki:   append([]byte(nil), datosMaterial.RaizPublicaSPKI...),
		alta:   append([]byte(nil), alta...),
		sellos: append([]byte(nil), sellos...),
	}
	return resultado, nil
}

func canonEfectoAltaV2(
	s ports.SolicitudProyectarEfectoAlta,
) ([]byte, error) {
	if s.Validar() != nil {
		return nil, ports.ErrOrdenAltaInvalida
	}
	x := s.Expediente
	a := x.Actuaciones[0]
	rc := x.Solicitud.RC
	fechaRC := ""
	moneda, centimos := "EUR", int64(0)
	if rc.Existe {
		fechaRC = fechaCivilAlta(rc.Fecha)
		moneda, centimos = rc.Importe.Moneda, rc.Importe.Centimos
	}
	documento := efectoAltaV2{
		Esquema:         esquemaEfectoAltaV2,
		ReservaRef:      s.Candidatura.ReservaRef,
		ExpedienteRef:   x.Referencia,
		NumeroVisible:   x.NumeroVisible,
		ReciboRef:       a.ReciboRef,
		OrganizacionRef: x.OrganizacionRef,
		ActorRef:        a.ActorRef,
		PerfilRef:       s.Candidatura.PerfilRef,
		Version:         x.Version,
		Flujo: flujoEfectoAltaV2{
			DefinicionRef: x.Flujo.DefinicionRef,
			Version:       x.Flujo.Version,
			HuellaSHA256:  x.Flujo.HuellaSHA256,
		},
		FaseActual:   string(x.FaseActual),
		EstadoActual: string(x.EstadoActual),
		Solicitud: solicitudEfectoV2{
			CentroRef:     x.Solicitud.CentroRef,
			ContactoRef:   x.Solicitud.ContactoRef,
			CategoriaRef:  x.Solicitud.CategoriaRef,
			GrupoSubgrupo: x.Solicitud.GrupoSubgrupo,
			MotivoClave:   string(x.Solicitud.MotivoClave),
			Detalle:       x.Solicitud.Detalle,
			Periodo: periodoEfectoV2{
				Inicio: fechaCivilAlta(x.Solicitud.Periodo.Inicio),
				Fin:    fechaCivilAlta(x.Solicitud.Periodo.Fin),
			},
			RC: rcEfectoV2{
				Existe: rc.Existe, Numero: rc.Numero, Fecha: fechaRC,
				Importe:      importeEfectoV2{Centimos: centimos, Moneda: moneda},
				DocumentoRef: rc.DocumentoRef,
			},
			DocumentosAdjuntos: copiaListaAlta(x.Solicitud.DocumentosAdjuntos),
			Observaciones:      x.Solicitud.Observaciones,
		},
		CreadoEn:      instanteAlta(x.CreadoEn),
		ActualizadoEn: instanteAlta(x.ActualizadoEn),
		Actuacion: actuacionEfectoV2{
			Secuencia: a.Secuencia, VersionExpediente: a.VersionExpediente,
			AccionClave: string(a.AccionClave), ActorRef: a.ActorRef,
			UnidadRef: a.UnidadRef, ReciboRef: a.ReciboRef,
			RealizadaEn: instanteAlta(a.RealizadaEn),
			FaseOrigen:  string(a.FaseOrigen), FaseDestino: string(a.FaseDestino),
			EstadoOrigen:  string(a.EstadoOrigen),
			EstadoDestino: string(a.EstadoDestino),
			Observaciones: a.Observaciones,
			DocumentosRef: copiaListaAlta(a.DocumentosRef),
		},
	}
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) < 256 || len(contenido) > 32*1024 {
		return nil, ports.ErrOrdenAltaInvalida
	}
	return contenido, nil
}

func canonSellosAltaV1(
	e ports.EvidenciaOrdenConfirmarAlta,
) ([]byte, error) {
	ambitos, errAmbitos := e.AmbitosIdempotenciaHMAC.Datos()
	huellas, errHuellas := e.HuellasPeticionHMAC.Datos()
	if errAmbitos != nil || errHuellas != nil ||
		ambitos.Activo.Generacion != huellas.Activo.Generacion ||
		len(ambitos.Retenidos) != len(huellas.Retenidos) {
		return nil, ports.ErrOrdenAltaInvalida
	}
	documento := sellosAltaV1{
		Esquema: esquemaSellosAltaV1,
		Activo: parSellosV1{
			Generacion: ambitos.Activo.Generacion,
			AmbitoHMAC: ambitos.Activo.Valor,
			HuellaHMAC: huellas.Activo.Valor,
		},
		Retenidos: make([]parSellosV1, len(ambitos.Retenidos)),
	}
	for i := range ambitos.Retenidos {
		if ambitos.Retenidos[i].Generacion != huellas.Retenidos[i].Generacion {
			return nil, ports.ErrOrdenAltaInvalida
		}
		documento.Retenidos[i] = parSellosV1{
			Generacion: ambitos.Retenidos[i].Generacion,
			AmbitoHMAC: ambitos.Retenidos[i].Valor,
			HuellaHMAC: huellas.Retenidos[i].Valor,
		}
	}
	contenido, err := json.Marshal(documento)
	if err != nil || len(contenido) < 256 || len(contenido) > 8192 {
		return nil, ports.ErrOrdenAltaInvalida
	}
	return contenido, nil
}

func instanteAlta(valor time.Time) string {
	return valor.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func fechaCivilAlta(valor time.Time) string {
	return valor.UTC().Format("2006-01-02")
}

func copiaListaAlta(valores []string) []string {
	return append([]string{}, valores...)
}

var _ ports.ProyectorEfectoAlta = (*ProyectorEfectoAltaV2)(nil)
