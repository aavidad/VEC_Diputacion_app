package bootstrap

import (
	"encoding/json"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const maximoEnteroJSONComprobacion = 9007199254740991

func ambitoOrdenOrigenConstituidoCT(orden json.RawMessage, unidad string) (string, error) {
	var alcance struct {
		AmbitoRef string `json:"ambito_ref"`
		UnidadRef string `json:"unidad_ref"`
	}
	if json.Unmarshal(orden, &alcance) != nil ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(alcance.AmbitoRef) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(alcance.UnidadRef) ||
		alcance.UnidadRef != unidad {
		return "", ports.ErrRespuestaBolsaNoConfiable
	}
	return alcance.AmbitoRef, nil
}

// Completa únicamente el error de la posición pendiente de la lectura
// auditada. La escritura de Bolsa volverá a validar este contexto en su POST.
func enriquecerErrorCriteriosPendientesCT(
	pendiente *ports.ErrorCriteriosOfrecimientoPendientesRRHH,
	origen origenConstituidoVerificadoCT, d preparacionLlamamientoDesarrollo,
) error {
	if pendiente == nil || pendiente.Comprobacion.Contexto != nil ||
		origen.registro.CriteriosOfrecimiento == nil ||
		origen.registro.BolsaRef != pendiente.Comprobacion.BolsaRef ||
		origen.registro.BolsaRef != origen.bolsa.BolsaRef ||
		origen.registro.CategoriaRef != d.categoria ||
		origen.registro.VersionBolsa != origen.bolsa.Version ||
		d.expediente.Fiscalizado.OrganizacionRef == "" ||
		d.expediente.Fiscalizado.Referencia == "" || d.necesidad == "" || d.unidad == "" ||
		origen.registro.VersionBolsa > maximoEnteroJSONComprobacion ||
		origen.registro.VersionInstantanea > maximoEnteroJSONComprobacion ||
		d.expediente.Fiscalizado.Version > maximoEnteroJSONComprobacion ||
		pendiente.Comprobacion.OrdenActa > maximoEnteroJSONComprobacion {
		return ports.ErrRespuestaBolsaNoConfiable
	}
	// La orden ya fue cotejada por SQL y ligada al acuse, pero este alcance
	// solo hace falta cuando el corte contiene una comprobación pendiente.
	ambitoRef, err := ambitoOrdenOrigenConstituidoCT(origen.ordenPermiso, d.unidad)
	if err != nil {
		return err
	}
	var participacion string
	for _, entrada := range origen.registro.Entradas {
		if entrada.OrdenActa == pendiente.Comprobacion.OrdenActa {
			if participacion != "" || entrada.Razon != "pendiente_comprobacion_rrhh" ||
				entrada.Elegible || entrada.OrdenVigente != nil {
				return ports.ErrRespuestaBolsaNoConfiable
			}
			participacion = entrada.ParticipacionRef
		}
	}
	criterios := origen.registro.CriteriosOfrecimiento
	contexto := ports.ContextoComprobacionPendiente{
		AmbitoRef: ambitoRef, BolsaRef: origen.registro.BolsaRef,
		CategoriaRef: d.categoria, CausaOfrecida: criterios.CausaOfrecida,
		ExpedienteRef:               d.expediente.Fiscalizado.Referencia,
		HuellaPoliticaOfertasSHA256: criterios.HuellaPoliticaOfertasSHA256,
		ModalidadOfrecida:           criterios.ModalidadOfrecida, NecesidadRef: d.necesidad,
		OrdenActa:        int64(pendiente.Comprobacion.OrdenActa),
		OrganizacionRef:  d.expediente.Fiscalizado.OrganizacionRef,
		ParticipacionRef: participacion, UnidadRef: d.unidad,
		VersionBolsa:           int64(origen.registro.VersionBolsa),
		VersionExpediente:      int64(d.expediente.Fiscalizado.Version),
		VersionInstantanea:     int64(origen.registro.VersionInstantanea),
		VersionPoliticaOfertas: criterios.VersionPoliticaOfertas,
	}
	if !contexto.ValidoPara(pendiente.Comprobacion) ||
		origen.instantanea.Version != origen.registro.VersionInstantanea {
		return ports.ErrRespuestaBolsaNoConfiable
	}
	pendiente.Comprobacion.Contexto = &contexto
	return pendiente
}
