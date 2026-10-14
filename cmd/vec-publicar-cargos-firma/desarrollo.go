package main

import (
	"regexp"
	"strings"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	doc "vec-diputacion-granada/internal/vec/documentos/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vec "vec-diputacion-granada/internal/vec/ports"
)

var reRolDesarrollo = regexp.MustCompile(`^ct_cargo_(tecnico|jefatura|diputacion|secretaria)_(firma|consulta|documentos|circuito)_dev$`)

func garantiaModo(desarrollo bool) vd.AuthAssurance {
	if desarrollo {
		return vd.AuthAssuranceSubstantial
	}
	return vd.AuthAssuranceHigh
}

// La variante sólo sucede roles del mismo circuito de desarrollo con CAS.
// Los roles altos y sus recibos anteriores siguen perteneciendo al contrato v1.
func rolDesarrolloValido(c cargoEntrada) bool {
	if !reRolDesarrollo.MatchString(c.RolID) || c.Version < 1 {
		return false
	}
	clase := strings.Split(c.RolID, "_")[3]
	if len(c.OperacionesV2) == 0 {
		return false
	}
	for _, op := range c.OperacionesV2 {
		switch clase {
		case "firma":
			if op != "consultar_r5" && op != "firmar_vec" {
				return false
			}
		case "consulta":
			if op != "consultar_detalle" && op != "consultar_cuadro" && op != "descargar_borrador" {
				return false
			}
		case "circuito":
			if op != "consultar_circuito" {
				return false
			}
		case "documentos":
			if op != "descargar_original" && op != "custodiar_firmado" &&
				!(strings.Contains(c.RolID, "_tecnico_") && (op == "reservar_original" || op == "confirmar_original" || op == "escribir_original")) {
				return false
			}
		default:
			return false
		}
	}
	return clase == "firma" && len(c.Competencias) > 0 || clase != "firma" && len(c.Competencias) == 0
}

func concesionModo(op string, desarrollo bool) (vd.ConcesionRol, bool) {
	if !desarrollo {
		return concesionV2(op)
	}
	if c, ok := concesionV2(op); ok {
		c.GarantiaMinima = vd.AuthAssuranceSubstantial
		return c, true
	}
	crear := func(accion, modulo, tipo, finalidad string, campos ...string) (vd.ConcesionRol, bool) {
		return vd.ConcesionRol{Accion: accion, ModuloID: modulo, TipoRecurso: tipo,
			Finalidades: []string{finalidad}, GarantiaMinima: vd.AuthAssuranceSubstantial, CamposPermitidos: campos}, true
	}
	switch op {
	case "consultar_detalle":
		return crear(ct.AccionConsultarDetalleRRHH, ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadConsultarDetalleRRHH)
	case "consultar_cuadro":
		return crear(ct.AccionConsultarCuadroRRHH, ct.ModuloContratacion, ct.TipoRecursoCuadroRRHH, ct.FinalidadConsultarCuadroRRHH)
	case "descargar_borrador":
		return crear(ct.AccionDescargarBorradorRRHH, ct.ModuloContratacion, ct.TipoRecursoExpediente, ct.FinalidadDescargarBorradorRRHH)
	case "consultar_circuito":
		return vd.ConcesionRol{Accion: "contratacion_temporal.circuito.consultar", ModuloID: ct.ModuloContratacion, TipoRecurso: "expediente_circuito_rrhh", Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: vd.AuthAssuranceSubstantial, CamposPermitidos: []string{"circuito"}, Obligaciones: []string{"auditar"}}, true
	case "descargar_original":
		return crear(doc.AccionDescargar, "documentos", "documento_original", "descargar_documento_original", "contenido", "documento")
	case "custodiar_firmado":
		return crear(doc.AccionCustodiarFirmado, "documentos", "documento_firmado", doc.FinalidadCustodiarFirmado, "documento_firmado.custodia", "evidencia_custodia")
	case "reservar_original":
		return crear(doc.AccionReservarOriginalFirmable, "documentos", "documento_original_firmable", doc.FinalidadOriginalFirmable, "intento", "reserva")
	case "confirmar_original":
		return crear(doc.AccionConfirmarOriginalFirmable, "documentos", "documento_original_firmable", doc.FinalidadOriginalFirmable, "documento", "recibo")
	case "escribir_original":
		return crear(vec.AccionNegocioEscribirOriginalFirmable, "documentos", "documento_original_firmable", doc.FinalidadOriginalFirmable, "evidencia_almacen", "original_firmable.contenido")
	}
	return vd.ConcesionRol{}, false
}
