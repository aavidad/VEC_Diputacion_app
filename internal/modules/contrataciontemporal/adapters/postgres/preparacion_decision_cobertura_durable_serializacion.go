package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

const (
	esquemaPreparacionDecisionCoberturaDurable = "" +
		"vec.contratacion-temporal.preparar-decision-cobertura.v1"
	maximoBytesCargaDecisionCoberturaDurable = 512 * 1024
)

type operacionPrepararDecisionCoberturaDurableV1 struct {
	Esquema                   string   `json:"esquema"`
	OrganizacionRef           string   `json:"organizacion_ref"`
	ExpedienteRef             string   `json:"expediente_ref"`
	VersionExpediente         uint64   `json:"version_expediente"`
	AmbitoActivoHMAC          string   `json:"ambito_activo_hmac"`
	HuellaSemanticaActivaHMAC string   `json:"huella_semantica_activa_hmac"`
	TokenPropietarioSHA256    string   `json:"token_propietario_sha256"`
	AmbitosConsultaHMAC       []string `json:"ambitos_consulta_hmac"`
}

type parPersistidoDecisionCoberturaDurableV1 struct {
	AmbitoHMAC          string `json:"ambito_hmac"`
	HuellaSemanticaHMAC string `json:"huella_semantica_hmac"`
}

func nuevaOperacionPrepararDecisionCoberturaDurableV1(
	solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
) (operacionPrepararDecisionCoberturaDurableV1, error) {
	datos, errDatos := solicitud.Datos()
	ambitos, errAmbitos := solicitud.AmbitosIdempotencia()
	datosAmbitos, errColeccion := ambitos.Datos()
	if errDatos != nil || errAmbitos != nil || errColeccion != nil {
		return operacionPrepararDecisionCoberturaDurableV1{},
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	valores := make([]string, 0, len(datosAmbitos.Retenidos)+1)
	valores = append(valores, datosAmbitos.Activo.Valor)
	for _, retenido := range datosAmbitos.Retenidos {
		valores = append(valores, retenido.Valor)
	}
	if len(valores) == 0 ||
		valores[0] != datos.AmbitoIdempotenciaHMAC {
		return operacionPrepararDecisionCoberturaDurableV1{},
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return operacionPrepararDecisionCoberturaDurableV1{
		Esquema:                   esquemaPreparacionDecisionCoberturaDurable,
		OrganizacionRef:           datos.OrganizacionRef,
		ExpedienteRef:             datos.ExpedienteRef,
		VersionExpediente:         datos.VersionExpediente,
		AmbitoActivoHMAC:          datos.AmbitoIdempotenciaHMAC,
		HuellaSemanticaActivaHMAC: datos.HuellaSemanticaHMAC,
		TokenPropietarioSHA256:    datos.TokenPropietarioSHA256,
		AmbitosConsultaHMAC:       valores,
	}, nil
}

func codificarAmbitosConsultaDecisionCoberturaDurable(
	solicitud cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) ([]byte, error) {
	ambitos, err := solicitud.AmbitosIdempotencia()
	if err != nil {
		return nil, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datos, err := ambitos.Datos()
	if err != nil {
		return nil, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	valores := make([]string, 0, len(datos.Retenidos)+1)
	valores = append(valores, datos.Activo.Valor)
	for _, retenido := range datos.Retenidos {
		valores = append(valores, retenido.Valor)
	}
	return json.Marshal(valores)
}

type cargaPropietariaDecisionCoberturaDurableV1 struct {
	ReservaRef              string          `json:"reserva_ref"`
	ReciboRef               string          `json:"recibo_ref"`
	ActuacionRef            string          `json:"actuacion_ref"`
	AuditoriaRef            string          `json:"auditoria_ref"`
	EventoRef               string          `json:"evento_ref"`
	CorrelacionVECRef       string          `json:"correlacion_vec_ref"`
	DecisionVECRef          string          `json:"decision_vec_ref"`
	AnalisisRef             string          `json:"analisis_ref"`
	AnalisisHuellaSHA256    string          `json:"analisis_huella_sha256"`
	TokenPropietarioSHA256  string          `json:"token_propietario_sha256"`
	AmbitoIdempotenciaHMAC  string          `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC     string          `json:"huella_semantica_hmac"`
	AgregadoAnterior        json.RawMessage `json:"agregado_anterior"`
	RevisionCercadoAnterior uint64          `json:"revision_cercado_anterior"`
	RevisionCercado         uint64          `json:"revision_cercado"`
	ObservadaEnDB           time.Time       `json:"observada_en"`
	PropiedadHasta          time.Time       `json:"propiedad_hasta"`
}

type cargaTerminalDecisionCoberturaDurableV1 struct {
	ReservaTerminal reservaTerminalDecisionCoberturaDurableV1 `json:"reserva_terminal"`
	Recibo          reciboDecisionCoberturaDurableV1          `json:"recibo"`
}

type reservaTerminalDecisionCoberturaDurableV1 struct {
	OrganizacionRef        string    `json:"organizacion_ref"`
	ExpedienteRef          string    `json:"expediente_ref"`
	VersionExpediente      uint64    `json:"version_expediente"`
	ReservaRef             string    `json:"reserva_ref"`
	ReciboRef              string    `json:"recibo_ref"`
	ActuacionRef           string    `json:"actuacion_ref"`
	AuditoriaRef           string    `json:"auditoria_ref"`
	EventoRef              string    `json:"evento_ref"`
	CorrelacionVECRef      string    `json:"correlacion_vec_ref"`
	DecisionVECRef         string    `json:"decision_vec_ref"`
	AmbitoIdempotenciaHMAC string    `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC    string    `json:"huella_semantica_hmac"`
	RevisionCercado        uint64    `json:"revision_cercado"`
	ObservadaEnDB          time.Time `json:"observada_en"`
}

type reciboDecisionCoberturaDurableV1 struct {
	ReciboRef                     string    `json:"recibo_ref"`
	ReservaRef                    string    `json:"reserva_ref"`
	AuditoriaRef                  string    `json:"auditoria_ref"`
	CorrelacionVECRef             string    `json:"correlacion_vec_ref"`
	DecisionVECRef                string    `json:"decision_vec_ref"`
	DecisionVECHuellaSHA256       string    `json:"decision_vec_huella_sha256"`
	CodigoProbatorioVEC           string    `json:"codigo_probatorio_vec"`
	ConcedidaVEC                  bool      `json:"concedida_vec"`
	RevisionCercado               uint64    `json:"revision_cercado"`
	AmbitoIdempotenciaHMAC        string    `json:"ambito_idempotencia_hmac"`
	HuellaSemanticaHMAC           string    `json:"huella_semantica_hmac"`
	ConfirmadaEn                  time.Time `json:"confirmada_en"`
	Aplicada                      bool      `json:"aplicada"`
	DenegadaVEC                   bool      `json:"denegada_vec"`
	DecisionCoberturaRef          string    `json:"decision_cobertura_ref"`
	DecisionCoberturaHuellaSHA256 string    `json:"decision_cobertura_huella_sha256"`
	VersionResultante             uint64    `json:"version_resultante"`
	EventoRef                     string    `json:"evento_ref"`
	ActuacionRef                  string    `json:"actuacion_ref"`
}

func restaurarPreparacionPropietariaDecisionCoberturaDurable(
	solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
	operacion operacionPrepararDecisionCoberturaDurableV1,
	contenido string,
) (cobertura.PreparacionOperacionDecisionCobertura, error) {
	var carga cargaPropietariaDecisionCoberturaDurableV1
	if decodificarCargaDecisionCoberturaDurable(
		[]byte(contenido),
		&carga,
	) != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	expediente, err := decodificarExpedienteAnalisisDurableO3(
		carga.AgregadoAnterior,
		operacion.OrganizacionRef,
		operacion.ExpedienteRef,
		operacion.VersionExpediente,
		carga.AnalisisHuellaSHA256,
	)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	preparacion, err :=
		cobertura.NuevaPreparacionOperacionDecisionCoberturaPropietaria(
			solicitud,
			cobertura.DatosReservaPropietariaOperacionDecisionCobertura{
				ReservaRef:              carga.ReservaRef,
				ReciboRef:               carga.ReciboRef,
				ActuacionRef:            carga.ActuacionRef,
				AuditoriaRef:            carga.AuditoriaRef,
				EventoRef:               carga.EventoRef,
				CorrelacionVECRef:       carga.CorrelacionVECRef,
				DecisionVECRef:          carga.DecisionVECRef,
				AnalisisRef:             carga.AnalisisRef,
				AnalisisHuellaSHA256:    carga.AnalisisHuellaSHA256,
				TokenPropietarioSHA256:  carga.TokenPropietarioSHA256,
				AmbitoIdempotenciaHMAC:  carga.AmbitoIdempotenciaHMAC,
				HuellaSemanticaHMAC:     carga.HuellaSemanticaHMAC,
				AgregadoAnterior:        &expediente,
				RevisionCercadoAnterior: carga.RevisionCercadoAnterior,
				RevisionCercado:         carga.RevisionCercado,
				ObservadaEnDB:           carga.ObservadaEnDB,
				PropiedadHasta:          carga.PropiedadHasta,
			},
		)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	return preparacion, nil
}

func restaurarPreparacionTerminalDecisionCoberturaDurable(
	solicitud cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	contenido string,
) (cobertura.PreparacionOperacionDecisionCobertura, error) {
	var carga cargaTerminalDecisionCoberturaDurableV1
	if decodificarCargaDecisionCoberturaDurable(
		[]byte(contenido),
		&carga,
	) != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	d := carga.ReservaTerminal
	reserva, err :=
		cobertura.RehidratarReservaTerminalOperacionDecisionCobertura(
			solicitud,
			cobertura.DatosReservaTerminalOperacionDecisionCobertura{
				OrganizacionRef:   d.OrganizacionRef,
				ExpedienteRef:     d.ExpedienteRef,
				VersionExpediente: d.VersionExpediente,
				ReservaRef:        d.ReservaRef, ReciboRef: d.ReciboRef,
				ActuacionRef: d.ActuacionRef,
				AuditoriaRef: d.AuditoriaRef, EventoRef: d.EventoRef,
				CorrelacionVECRef:      d.CorrelacionVECRef,
				DecisionVECRef:         d.DecisionVECRef,
				AmbitoIdempotenciaHMAC: d.AmbitoIdempotenciaHMAC,
				HuellaSemanticaHMAC:    d.HuellaSemanticaHMAC,
				RevisionCercado:        d.RevisionCercado,
				ObservadaEnDB:          d.ObservadaEnDB,
			},
		)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	r := carga.Recibo
	recibo := cobertura.ReciboOperacionDecisionCobertura{
		ReciboRef: r.ReciboRef, ReservaRef: r.ReservaRef,
		AuditoriaRef:            r.AuditoriaRef,
		CorrelacionVECRef:       r.CorrelacionVECRef,
		DecisionVECRef:          r.DecisionVECRef,
		DecisionVECHuellaSHA256: r.DecisionVECHuellaSHA256,
		CodigoProbatorioVEC:     r.CodigoProbatorioVEC,
		ConcedidaVEC:            r.ConcedidaVEC,
		RevisionCercado:         r.RevisionCercado,
		AmbitoIdempotenciaHMAC:  r.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     r.HuellaSemanticaHMAC,
		ConfirmadaEn:            r.ConfirmadaEn,
	}
	switch {
	case r.Aplicada && !r.DenegadaVEC:
		recibo.Aplicada =
			&cobertura.ResultadoAplicadoOperacionDecisionCobertura{
				DecisionCoberturaRef:    r.DecisionCoberturaRef,
				DecisionCoberturaHuella: r.DecisionCoberturaHuellaSHA256,
				VersionResultante:       r.VersionResultante,
				EventoRef:               r.EventoRef,
				ActuacionRef:            r.ActuacionRef,
			}
	case r.DenegadaVEC && !r.Aplicada:
		recibo.DenegadaVEC =
			&cobertura.ResultadoDenegadoVECOperacionDecisionCobertura{}
	default:
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	preparacion, err :=
		cobertura.NuevaPreparacionOperacionDecisionCoberturaConfirmada(
			solicitud,
			reserva,
			recibo,
		)
	if err != nil {
		return cobertura.PreparacionOperacionDecisionCobertura{},
			errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	return preparacion, nil
}

func decodificarCargaDecisionCoberturaDurable(
	contenido []byte,
	destino any,
) error {
	if len(contenido) < 2 ||
		len(contenido) > maximoBytesCargaDecisionCoberturaDurable ||
		destino == nil {
		return errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return errPersistenciaDecisionCoberturaDurableNoDisponible
	}
	return nil
}
