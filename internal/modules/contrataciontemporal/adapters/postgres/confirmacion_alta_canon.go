package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaEfectoAltaV2      = "vec.contratacion-temporal.efecto-alta.v2"
	esquemaSellosAltaV1      = "vec.contratacion-temporal.sellos-hmac.v1"
	audienciaConfirmarAltaV1 = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

type efectoAltaCanonico struct {
	Esquema         string                `json:"esquema"`
	ReservaRef      string                `json:"reserva_ref"`
	ExpedienteRef   string                `json:"expediente_ref"`
	NumeroVisible   string                `json:"numero_visible"`
	ReciboRef       string                `json:"recibo_ref"`
	OrganizacionRef string                `json:"organizacion_ref"`
	ActorRef        string                `json:"actor_ref"`
	PerfilRef       string                `json:"perfil_ref"`
	Version         uint64                `json:"version"`
	Flujo           flujoAltaCanonico     `json:"flujo"`
	FaseActual      string                `json:"fase_actual"`
	EstadoActual    string                `json:"estado_actual"`
	Solicitud       solicitudAltaCanonica `json:"solicitud"`
	CreadoEn        string                `json:"creado_en"`
	ActualizadoEn   string                `json:"actualizado_en"`
	Actuacion       actuacionAltaCanonica `json:"actuacion"`
}

type flujoAltaCanonico struct {
	DefinicionRef string `json:"definicion_ref"`
	Version       uint64 `json:"version"`
	HuellaSHA256  string `json:"huella_sha256"`
}

type solicitudAltaCanonica struct {
	CentroRef          string              `json:"centro_ref"`
	ContactoRef        string              `json:"contacto_ref"`
	CategoriaRef       string              `json:"categoria_ref"`
	GrupoSubgrupo      string              `json:"grupo_subgrupo"`
	MotivoClave        string              `json:"motivo_clave"`
	Detalle            string              `json:"detalle"`
	Periodo            periodoAltaCanonico `json:"periodo"`
	RC                 rcAltaCanonica      `json:"rc"`
	DocumentosAdjuntos []string            `json:"documentos_adjuntos"`
	Observaciones      string              `json:"observaciones"`
}

type periodoAltaCanonico struct {
	Inicio string `json:"inicio"`
	Fin    string `json:"fin"`
}

type rcAltaCanonica struct {
	Existe       bool                `json:"existe"`
	Numero       string              `json:"numero"`
	Fecha        string              `json:"fecha"`
	Importe      importeAltaCanonico `json:"importe"`
	DocumentoRef string              `json:"documento_ref"`
}

type importeAltaCanonico struct {
	Centimos int64  `json:"centimos"`
	Moneda   string `json:"moneda"`
}

type actuacionAltaCanonica struct {
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

type sellosAltaCanonicos struct {
	Esquema   string                 `json:"esquema"`
	Activo    parSelloAltaCanonico   `json:"activo"`
	Retenidos []parSelloAltaCanonico `json:"retenidos"`
}

type parSelloAltaCanonico struct {
	Generacion uint32 `json:"generacion"`
	AmbitoHMAC string `json:"ambito_hmac"`
	HuellaHMAC string `json:"huella_hmac"`
}

func canonConfirmacionAlta(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
) ([]byte, []byte, string, error) {
	candidatura, err := evidencia.Candidatura.Datos()
	if err != nil || evidencia.Expediente.Validar() != nil ||
		len(evidencia.Expediente.Actuaciones) != 1 {
		return nil, nil, "", ports.ErrOrdenAltaInvalida
	}
	efecto := construirEfectoAltaCanonico(evidencia.Expediente, candidatura)
	alta, err := json.Marshal(efecto)
	if err != nil || len(alta) < 256 || len(alta) > 32*1024 {
		return nil, nil, "", ports.ErrOrdenAltaInvalida
	}
	sellos, err := construirSellosAltaCanonicos(
		evidencia.AmbitosIdempotenciaHMAC,
		evidencia.HuellasPeticionHMAC,
	)
	if err != nil {
		borrarBytes(alta)
		return nil, nil, "", err
	}
	sellosJSON, err := json.Marshal(sellos)
	if err != nil || len(sellosJSON) < 256 || len(sellosJSON) > 8*1024 {
		borrarBytes(alta)
		borrarBytes(sellosJSON)
		return nil, nil, "", ports.ErrOrdenAltaInvalida
	}
	huella := sha256.Sum256(alta)
	return alta, sellosJSON, hex.EncodeToString(huella[:]), nil
}

func construirEfectoAltaCanonico(
	expediente domain.Expediente,
	candidatura ports.DatosCandidaturaAlta,
) efectoAltaCanonico {
	solicitud := expediente.Solicitud
	actuacion := expediente.Actuaciones[0]
	rc := solicitud.RC
	fechaRC := ""
	monedaRC := "EUR"
	centimosRC := int64(0)
	if rc.Existe {
		fechaRC = formatoFechaCivil(rc.Fecha)
		monedaRC = rc.Importe.Moneda
		centimosRC = rc.Importe.Centimos
	}
	documentosSolicitud := append([]string{}, solicitud.DocumentosAdjuntos...)
	documentosActuacion := append([]string{}, actuacion.DocumentosRef...)
	return efectoAltaCanonico{
		Esquema: esquemaEfectoAltaV2, ReservaRef: candidatura.ReservaRef,
		ExpedienteRef: expediente.Referencia, NumeroVisible: expediente.NumeroVisible,
		ReciboRef: actuacion.ReciboRef, OrganizacionRef: expediente.OrganizacionRef,
		ActorRef: candidatura.ActorRef, PerfilRef: candidatura.PerfilRef,
		Version: expediente.Version,
		Flujo: flujoAltaCanonico{DefinicionRef: expediente.Flujo.DefinicionRef,
			Version: expediente.Flujo.Version, HuellaSHA256: expediente.Flujo.HuellaSHA256},
		FaseActual: string(expediente.FaseActual), EstadoActual: string(expediente.EstadoActual),
		Solicitud: solicitudAltaCanonica{
			CentroRef: solicitud.CentroRef, ContactoRef: solicitud.ContactoRef,
			CategoriaRef: solicitud.CategoriaRef, GrupoSubgrupo: solicitud.GrupoSubgrupo,
			MotivoClave: string(solicitud.MotivoClave), Detalle: solicitud.Detalle,
			Periodo: periodoAltaCanonico{Inicio: formatoFechaCivil(solicitud.Periodo.Inicio),
				Fin: formatoFechaCivil(solicitud.Periodo.Fin)},
			RC: rcAltaCanonica{Existe: rc.Existe, Numero: rc.Numero,
				Fecha: fechaRC,
				Importe: importeAltaCanonico{Centimos: centimosRC,
					Moneda: monedaRC}, DocumentoRef: rc.DocumentoRef},
			DocumentosAdjuntos: documentosSolicitud, Observaciones: solicitud.Observaciones,
		},
		CreadoEn:      formatoInstanteMicro(expediente.CreadoEn),
		ActualizadoEn: formatoInstanteMicro(expediente.ActualizadoEn),
		Actuacion: actuacionAltaCanonica{
			Secuencia: actuacion.Secuencia, VersionExpediente: actuacion.VersionExpediente,
			AccionClave: string(actuacion.AccionClave), ActorRef: actuacion.ActorRef,
			UnidadRef: actuacion.UnidadRef, ReciboRef: actuacion.ReciboRef,
			RealizadaEn: formatoInstanteMicro(actuacion.RealizadaEn),
			FaseOrigen:  string(actuacion.FaseOrigen), FaseDestino: string(actuacion.FaseDestino),
			EstadoOrigen: string(actuacion.EstadoOrigen), EstadoDestino: string(actuacion.EstadoDestino),
			Observaciones: actuacion.Observaciones, DocumentosRef: documentosActuacion,
		},
	}
}

func construirSellosAltaCanonicos(
	ambitos ports.ColeccionSellosHMAC,
	huellas ports.ColeccionSellosHMAC,
) (sellosAltaCanonicos, error) {
	datosAmbitos, errAmbitos := ambitos.Datos()
	datosHuellas, errHuellas := huellas.Datos()
	if errAmbitos != nil || errHuellas != nil ||
		datosAmbitos.Activo.Generacion != datosHuellas.Activo.Generacion ||
		len(datosAmbitos.Retenidos) != len(datosHuellas.Retenidos) {
		return sellosAltaCanonicos{}, ports.ErrOrdenAltaInvalida
	}
	sellos := sellosAltaCanonicos{Esquema: esquemaSellosAltaV1,
		Activo: parSelloAltaCanonico{Generacion: datosAmbitos.Activo.Generacion,
			AmbitoHMAC: datosAmbitos.Activo.Valor, HuellaHMAC: datosHuellas.Activo.Valor},
		Retenidos: make([]parSelloAltaCanonico, len(datosAmbitos.Retenidos))}
	for indice := range datosAmbitos.Retenidos {
		if datosAmbitos.Retenidos[indice].Generacion != datosHuellas.Retenidos[indice].Generacion {
			return sellosAltaCanonicos{}, ports.ErrOrdenAltaInvalida
		}
		sellos.Retenidos[indice] = parSelloAltaCanonico{
			Generacion: datosAmbitos.Retenidos[indice].Generacion,
			AmbitoHMAC: datosAmbitos.Retenidos[indice].Valor,
			HuellaHMAC: datosHuellas.Retenidos[indice].Valor,
		}
	}
	return sellos, nil
}

func formatoInstanteMicro(instante time.Time) string {
	return instante.UTC().Format("2006-01-02T15:04:05.000000Z")
}

func formatoFechaCivil(instante time.Time) string {
	return instante.UTC().Format("2006-01-02")
}

func huellaReciboAlta(recibo ports.ReciboAlta) string {
	calculador := sha256.New()
	for _, valor := range []string{
		recibo.ExpedienteRef, recibo.NumeroVisible,
		strconv.FormatUint(recibo.Version, 10), recibo.ReciboRef,
		recibo.AuditoriaRef, recibo.EventoRef,
		formatoInstanteMicro(recibo.ConfirmadaEn),
	} {
		_, _ = calculador.Write([]byte(strconv.Itoa(len([]byte(valor)))))
		_, _ = calculador.Write([]byte(":"))
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte("\n"))
	}
	return hex.EncodeToString(calculador.Sum(nil))
}
