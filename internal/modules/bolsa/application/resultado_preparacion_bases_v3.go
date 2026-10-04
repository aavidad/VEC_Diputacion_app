package application

import (
	"reflect"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

// El adaptador ejecuta esta validacion ANTES de COMMIT; la aplicacion vuelve
// a comprobarla antes de devolver material confirmado al consumidor.
func ValidarResultadoGuardarPreparacionBasesV3(o ports.OrdenGuardarPreparacionBasesV3, r ports.ResultadoPreparacionBasesV3) error {
	if ValidarMaterialGuardarPreparacionBasesV3(o) != nil || !accesoPreparacionV3Valido(r.Acceso, o.Autorizacion, o.Solicitud.Correlacion) {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	if r.Estado == "version_en_conflicto" || r.Estado == "clave_reutilizada" {
		if resultadoPreparacionSinMaterial(r) {
			return nil
		}
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	if r.Estado != "guardada" && r.Estado != "recuperada" {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	h, err := o.Solicitud.Material.HuellaSHA256()
	i, errIntencion := prep.HuellaIntencion(o.Solicitud.Esperada, o.Solicitud.Material, o.Solicitud.Ambito)
	exacta := prep.Esperada{PreparacionRef: o.Solicitud.Esperada.PreparacionRef, Revision: o.Solicitud.Esperada.Revision + 1, HuellaMaterialSHA256: h}
	if err != nil || errIntencion != nil || r.Version.Estado != exacta || r.Recibo.HuellaIntencionSHA256 != i ||
		!versionPreparacionV3Valida(r, o.Solicitud.Ambito) {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	return nil
}

func ValidarResultadoConsultarPreparacionBasesV3(o ports.OrdenConsultarPreparacionBasesV3, r ports.ResultadoPreparacionBasesV3) error {
	if ValidarMaterialConsultarPreparacionBasesV3(o) != nil || !accesoPreparacionV3Valido(r.Acceso, o.Autorizacion, o.Solicitud.Correlacion) {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	if r.Estado == "no_encontrada" {
		if resultadoPreparacionSinMaterial(r) {
			return nil
		}
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	if r.Estado != "obtenida" || !versionPreparacionV3Valida(r, o.Solicitud.Ambito) ||
		r.Version.Estado.PreparacionRef != o.Solicitud.Selector.Exacta.PreparacionRef {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	if o.Solicitud.Selector.Modo == "exacta" && r.Version.Estado != o.Solicitud.Selector.Exacta {
		return ports.ErrResultadoPreparacionBasesInvalido
	}
	return nil
}

func accesoPreparacionV3Valido(a ports.EvidenciaAccesoPreparacionBasesV3, e core.ExportacionMaterialConsumoAutorizacionAtestadaV3, c vec.ReferenciaCorrelacionAutorizacionV2) bool {
	x := e.ResumenCapacidad()
	correlacion, err := c.ValorCanonico()
	return err == nil && a.DecisionRef == x.DecisionRef() && a.CorrelacionRef == correlacion &&
		prep.HuellaValida(a.ConsumoHuellaSHA256) && prep.IdentificadorValido(a.ReciboRef) && prep.IdentificadorValido(a.AuditoriaRef) &&
		a.ReciboRef != a.AuditoriaRef && instanteCanonicoPreparacion(a.AccedidaEn) &&
		!a.AccedidaEn.Before(x.EmitidaEn()) && a.AccedidaEn.Before(x.ExpiraEn())
}

func versionPreparacionV3Valida(r ports.ResultadoPreparacionBasesV3, a bolsa.AmbitoOrganizativoConvocatoria) bool {
	if r.Version.Validar() != nil || r.Version.Ambito != a || !prep.HuellaValida(r.Recibo.HuellaIntencionSHA256) ||
		!instanteCanonicoPreparacion(r.Recibo.ConfirmadaEn) || r.Recibo.ConfirmadaEn.After(r.Acceso.AccedidaEn) {
		return false
	}
	vistos := map[string]bool{}
	for _, ref := range []string{r.Recibo.ReciboRef, r.Recibo.HistoriaRef, r.Recibo.AuditoriaRef, r.Recibo.EventoRef} {
		if !prep.IdentificadorValido(ref) || vistos[ref] {
			return false
		}
		vistos[ref] = true
	}
	if vistos[r.Acceso.ReciboRef] {
		return false
	}
	// El primer guardado de esta revision consume una sola autorizacion: su
	// auditoria real puede acreditar juntos acceso y efecto. Recuperar o leer
	// exige otra auditoria; nunca se fabrican referencias para distinguirlas.
	if vistos[r.Acceso.AuditoriaRef] && !(r.Estado == "guardada" && r.Acceso.AuditoriaRef == r.Recibo.AuditoriaRef) {
		return false
	}
	return true
}

func resultadoPreparacionSinMaterial(r ports.ResultadoPreparacionBasesV3) bool {
	return reflect.DeepEqual(r.Version, prep.Version{}) && r.Recibo == (ports.ReciboPreparacionBases{})
}

func clonarResultadoPreparacionBasesV3(r ports.ResultadoPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	m, err := r.Version.Material.Canonico()
	if err != nil {
		return ports.ResultadoPreparacionBasesV3{}, ports.ErrResultadoPreparacionBasesInvalido
	}
	r.Version.Material = m
	return r, nil
}
