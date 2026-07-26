package postgres

import (
	"bytes"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const maximaProfundidadJSONDecisionCoberturaO404E = 8

var clavesReciboDecisionCoberturaO404E = []string{
	"esquema",
	"recibo_ref",
	"reserva_ref",
	"auditoria_ref",
	"correlacion_vec_ref",
	"decision_vec_ref",
	"decision_vec_huella_sha256",
	"codigo_probatorio_vec",
	"concedida_vec",
	"revision_cercado",
	"ambito_idempotencia_hmac",
	"huella_semantica_hmac",
	"confirmada_en",
	"aplicada",
	"denegada_vec",
	"decision_cobertura_ref",
	"decision_cobertura_huella_sha256",
	"version_resultante",
	"evento_ref",
	"actuacion_ref",
}

var clavesConsultaPrimariaDecisionCoberturaO404E = []string{
	"esquema",
	"organizacion_ref",
	"expediente_ref",
	"version_expediente",
	"reserva_ref",
	"recibo_ref",
	"correlacion_vec_ref",
	"decision_vec_ref",
	"revision_cercado",
	"huella_orden_sha256",
}

var clavesResultadoPrimarioDecisionCoberturaO404E = []string{
	"esquema",
	"encontrado",
	"consulta",
	"recibo",
	"observada_en_primario",
}

func decodificarObjetoJSONExactoDecisionCoberturaO404E(
	contenido []byte,
	claves []string,
	destino any,
) (map[string]json.RawMessage, error) {
	if err := validarJSONSinDuplicadosDecisionCoberturaO404E(
		contenido,
		maximaProfundidadJSONDecisionCoberturaO404E,
	); err != nil {
		return nil, err
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil ||
		!clavesJSONExactasDecisionCoberturaO404E(objeto, claves) {
		return nil, errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return nil, err
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return nil, errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return objeto, nil
}

func validarObjetoCrudoJSONExactoDecisionCoberturaO404E(
	contenido json.RawMessage,
	claves []string,
) error {
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil ||
		!clavesJSONExactasDecisionCoberturaO404E(objeto, claves) {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return nil
}

func clavesJSONExactasDecisionCoberturaO404E(
	objeto map[string]json.RawMessage,
	claves []string,
) bool {
	if objeto == nil || len(objeto) != len(claves) {
		return false
	}
	for _, clave := range claves {
		if _, existe := objeto[clave]; !existe {
			return false
		}
	}
	return true
}

func validarJSONSinDuplicadosDecisionCoberturaO404E(
	contenido []byte,
	maximaProfundidad int,
) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := consumirValorJSONDecisionCoberturaO404E(
		decodificador,
		0,
		maximaProfundidad,
	); err != nil {
		return err
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return nil
}

func consumirValorJSONDecisionCoberturaO404E(
	decodificador *json.Decoder,
	profundidad int,
	maximaProfundidad int,
) error {
	if profundidad > maximaProfundidad {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, contenedor := token.(json.Delim)
	if !contenedor {
		return nil
	}
	switch delimitador {
	case '{':
		vistas := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, errClave := decodificador.Token()
			clave, esTexto := tokenClave.(string)
			if errClave != nil || !esTexto {
				return errAdaptadorDecisionCoberturaO404ENoDisponible
			}
			if _, repetida := vistas[clave]; repetida {
				return errAdaptadorDecisionCoberturaO404ENoDisponible
			}
			vistas[clave] = struct{}{}
			if err := consumirValorJSONDecisionCoberturaO404E(
				decodificador,
				profundidad+1,
				maximaProfundidad,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return errAdaptadorDecisionCoberturaO404ENoDisponible
		}
	case '[':
		for decodificador.More() {
			if err := consumirValorJSONDecisionCoberturaO404E(
				decodificador,
				profundidad+1,
				maximaProfundidad,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return errAdaptadorDecisionCoberturaO404ENoDisponible
		}
	default:
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return nil
}

func validarReciboDecisionCoberturaO404E(
	d reciboDecisionCoberturaO404E,
) bool {
	if d.Esquema != esquemaReciboDecisionCoberturaO404E ||
		!domain.ReferenciaOpacaValida(d.ReciboRef) ||
		!domain.ReferenciaOpacaValida(d.ReservaRef) ||
		!domain.ReferenciaOpacaValida(d.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(d.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(d.DecisionVECRef) ||
		!huellaSHA256DecisionCoberturaO404EValida(
			d.DecisionVECHuellaSHA256,
		) ||
		!dominiovec.CodigoResultadoEvaluacionAutorizacionV3Valido(
			d.CodigoProbatorioVEC,
			d.ConcedidaVEC,
		) ||
		d.RevisionCercado == 0 ||
		d.RevisionCercado >
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura ||
		!puertosct.SelloHMACSHA256Valido(d.AmbitoIdempotenciaHMAC) ||
		!puertosct.SelloHMACSHA256Valido(d.HuellaSemanticaHMAC) ||
		!domain.InstanteUTCCanonico(d.ConfirmadaEn) ||
		d.Aplicada != d.ConcedidaVEC ||
		d.DenegadaVEC == d.ConcedidaVEC {
		return false
	}
	if d.Aplicada {
		return huellaSHA256DecisionCoberturaO404EValida(
			d.DecisionCoberturaHuella,
		) &&
			d.DecisionCoberturaRef ==
				"decision-cobertura:sha256:"+
					d.DecisionCoberturaHuella &&
			d.VersionResultante > 0 &&
			d.VersionResultante <=
				cobertura.MaximoEnteroSeguroOperacionDecisionCobertura &&
			domain.ReferenciaOpacaValida(d.EventoRef) &&
			domain.ReferenciaOpacaValida(d.ActuacionRef)
	}
	return d.DecisionCoberturaRef == "" &&
		d.DecisionCoberturaHuella == "" &&
		d.VersionResultante == 0 &&
		d.EventoRef == "" && d.ActuacionRef == ""
}

func validarConsultaPrimariaDecisionCoberturaO404E(
	d consultaPrimariaDecisionCoberturaO404E,
) bool {
	return d.Esquema == esquemaConsultaPrimariaDecisionCoberturaO404E &&
		domain.ReferenciaOpacaValida(d.OrganizacionRef) &&
		domain.ReferenciaOpacaValida(d.ExpedienteRef) &&
		d.VersionExpediente > 0 &&
		d.VersionExpediente <=
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura &&
		domain.ReferenciaOpacaValida(d.ReservaRef) &&
		domain.ReferenciaOpacaValida(d.ReciboRef) &&
		domain.ReferenciaOpacaValida(d.CorrelacionVECRef) &&
		domain.ReferenciaOpacaValida(d.DecisionVECRef) &&
		d.RevisionCercado > 0 &&
		d.RevisionCercado <=
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura &&
		huellaSHA256DecisionCoberturaO404EValida(d.HuellaOrdenSHA256)
}

func huellaSHA256DecisionCoberturaO404EValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	noNula := false
	for i := range valor {
		if (valor[i] < '0' || valor[i] > '9') &&
			(valor[i] < 'a' || valor[i] > 'f') {
			return false
		}
		noNula = noNula || valor[i] != '0'
	}
	return noNula
}
