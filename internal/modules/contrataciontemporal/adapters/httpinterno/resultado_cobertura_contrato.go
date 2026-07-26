package httpinterno

import (
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const EsquemaResultadoConsultaCobertura = "" +
	"vec.contratacion-temporal.resultado-consulta-cobertura.v1"

type resultadoCoberturaEntradaJSON struct {
	ExpedienteRef     string `json:"expediente_ref"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

func resultadoCoberturaDesdePeticion(
	w http.ResponseWriter,
	r *http.Request,
) (resultadoCoberturaEntradaJSON, error) {
	var entrada resultadoCoberturaEntradaJSON
	if err := decodificarCobertura(w, r, &entrada); err != nil {
		return resultadoCoberturaEntradaJSON{}, err
	}
	if !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) ||
		!ports.ClaveIdempotenciaValida(entrada.ClaveIdempotencia) {
		return resultadoCoberturaEntradaJSON{},
			errContenidoCoberturaNoValido
	}
	return entrada, nil
}

type envoltorioResultadoConsultaCobertura struct {
	Data resultadoConsultaCoberturaJSON `json:"data"`
}

type resultadoConsultaCoberturaJSON struct {
	Esquema string               `json:"esquema"`
	Estado  string               `json:"estado"`
	Recibo  *reciboCoberturaJSON `json:"recibo,omitempty"`
}

func proyectarResultadoConsultaCobertura(
	entrada application.DatosConsultaResultadoCoberturaParaAdaptador,
) (resultadoConsultaCoberturaJSON, int, bool) {
	salida := resultadoConsultaCoberturaJSON{
		Esquema: EsquemaResultadoConsultaCobertura,
		Estado:  string(entrada.Estado),
	}
	switch entrada.Estado {
	case application.ResultadoCoberturaNoObservable:
		if entrada.Recibo != nil {
			return resultadoConsultaCoberturaJSON{}, 0, false
		}
		return salida, http.StatusAccepted, true
	case application.ResultadoCoberturaConfirmado:
		if entrada.Recibo == nil {
			return resultadoConsultaCoberturaJSON{}, 0, false
		}
		recibo, valida := proyectarDatosReciboCobertura(*entrada.Recibo)
		if !valida {
			return resultadoConsultaCoberturaJSON{}, 0, false
		}
		salida.Recibo = &recibo
		return salida, http.StatusOK, true
	default:
		return resultadoConsultaCoberturaJSON{}, 0, false
	}
}

func proyectarDatosReciboCobertura(
	datos application.DatosReciboDecisionCoberturaParaAdaptador,
) (reciboCoberturaJSON, bool) {
	if !domain.ReferenciaOpacaValida(datos.ReciboRef) ||
		!domain.InstanteUTCCanonico(datos.ConfirmadaEn) {
		return reciboCoberturaJSON{}, false
	}
	salida := reciboCoberturaJSON{
		Esquema:      "vec.contratacion-temporal.recibo-cobertura.v1",
		ReciboRef:    datos.ReciboRef,
		ConfirmadaEn: datos.ConfirmadaEn.UTC().Format(time.RFC3339Nano),
	}
	switch datos.Estado {
	case "aplicada":
		if !domain.ReferenciaOpacaValida(datos.DecisionCoberturaRef) ||
			datos.VersionResultante == 0 ||
			datos.VersionResultante >
				cobertura.MaximoEnteroSeguroOperacionDecisionCobertura {
			return reciboCoberturaJSON{}, false
		}
		salida.Estado = "aplicada"
		salida.DecisionCoberturaRef = datos.DecisionCoberturaRef
		salida.VersionResultante = datos.VersionResultante
		return salida, true
	case "denegada":
		if datos.DecisionCoberturaRef != "" || datos.VersionResultante != 0 {
			return reciboCoberturaJSON{}, false
		}
		salida.Estado = "denegada"
		return salida, true
	default:
		return reciboCoberturaJSON{}, false
	}
}
