package application

import (
	"context"
	"reflect"
	"strings"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioAutorizacionRutasDietas struct {
	autorizador vecports.AutorizadorSolicitudLigadaV3
	proveedor   dietasports.ProveedorMaterialAccesoRutas
	consumidor  dietasports.ConsumidorAccesoRutasDietas
}

func NuevoServicioAutorizacionRutasDietas(a vecports.AutorizadorSolicitudLigadaV3, p dietasports.ProveedorMaterialAccesoRutas, c dietasports.ConsumidorAccesoRutasDietas) (*ServicioAutorizacionRutasDietas, error) {
	if nuloRutas(a) || nuloRutas(p) || nuloRutas(c) {
		return nil, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	return &ServicioAutorizacionRutasDietas{a, p, c}, nil
}
func (s *ServicioAutorizacionRutasDietas) AutorizarYConsumirAccesoRutas(ctx context.Context, e dietasports.SolicitudAccesoRutasDietas) (dietasports.ReciboAccesoRutasDietas, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || nuloRutas(s.autorizador) || nuloRutas(s.proveedor) || nuloRutas(s.consumidor) || !solicitudRutasValida(e) {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	q, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: e.Vinculo, ReferenciaMotivo: e.ReferenciaMotivo, Accion: e.Accion, Recurso: e.Recurso, Finalidad: e.Finalidad, Correlacion: e.Correlacion})
	if err != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	d, c, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, q, e.ResultadoContexto)
	if err != nil || ctx.Err() != nil || d.ValidarPara(q) != nil || c.Validar() != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(q, d, e.ReferenciaMotivo, e.ResultadoContexto)
	if err != nil || c.ValidarPara(orden) != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	m, err := s.proveedor.ProveerMaterialAccesoRutas(ctx, q, d, c, e.ResultadoContexto)
	if err != nil || ctx.Err() != nil {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	if !vecports.MaterialAtestadoLigadoV3(q, d, c, e.ResultadoContexto, e.ReferenciaMotivo, m, e.Audiencia) {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	if !materialRutasValido(m, e) {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasDenegado
	}
	r, err := s.consumidor.ConsumirAccesoRutasDietas(ctx, dietasports.OrdenConsumoAccesoRutasDietas{Solicitud: q, Material: m})
	if err != nil {
		return dietasports.ReciboAccesoRutasDietas{}, err
	}
	if ctx.Err() != nil || !reciboRutasValido(r, m, e) {
		return dietasports.ReciboAccesoRutasDietas{}, dietasports.ErrAccesoRutasDietasNoDisponible
	}
	return r, nil
}
func solicitudRutasValida(e dietasports.SolicitudAccesoRutasDietas) bool {
	if e.ResultadoContexto.Validar() != nil || e.Vinculo.ValidarPara(e.ResultadoContexto) != nil || e.Recurso.Validar() != nil || e.Audiencia != dietasports.AudienciaAccesoRutasDietas || e.Finalidad != dietasports.FinalidadConsultarItinerarioDietas {
		return false
	}
	d, err := e.Vinculo.Datos()
	if err != nil || d.CuentaPrivilegiada || d.Superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1 {
		return false
	}
	tipoOK := (e.Accion == dietasports.AccionConsultarCatalogoRutasDietas && e.Recurso.ModuloID == "dietas" && e.Recurso.Tipo == dietasports.TipoCatalogoRutasDietas) || (e.Accion == dietasports.AccionSolicitarCalculoRutasDietas && e.Recurso.ModuloID == "dietas" && e.Recurso.Tipo == dietasports.TipoCalculoRutasDietas)
	a := e.Recurso.Atributos
	corr, err := e.Correlacion.ValorCanonico()
	rutaOK := (e.Accion == dietasports.AccionConsultarCatalogoRutasDietas && a["ruta"] == "/api/vec/dietas/route-catalog" && a["metodo"] == "GET") || (e.Accion == dietasports.AccionSolicitarCalculoRutasDietas && a["ruta"] == "/api/vec/dietas/road-route" && a["metodo"] == "POST")
	return tipoOK && err == nil && len(e.Recurso.Ambitos) == 1 && len(a) == 7 && e.Recurso.Ambitos["ambito_ref"] != "" && seguro(a["version"]) && huellaHex(a["huella_sha256"]) && seguro(a["canal"]) && seguro(a["instancia"]) && rutaOK && a["correlacion_ref"] == corr
}
func materialRutasValido(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e dietasports.SolicitudAccesoRutasDietas) bool {
	h, err := e.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || m.ValidarEstructura() != nil {
		return false
	}
	r := m.ResumenCapacidad()
	return r.Operacion() == e.Accion && r.EfectoRef() == e.Recurso.Referencia && r.EfectoHuellaSHA256() == h && r.AudienciaConsumo() == e.Audiencia
}
func reciboRutasValido(r dietasports.ReciboAccesoRutasDietas, m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, e dietasports.SolicitudAccesoRutasDietas) bool {
	h, err := e.Recurso.HuellaContextoAutorizacionSHA256()
	z := m.ResumenCapacidad()
	return err == nil && r.DecisionRef == z.DecisionRef() && r.EfectoRef == e.Recurso.Referencia && r.HuellaEfectoSHA256 == h && huellaHex(r.ConsumoHuellaSHA256) && seguro(r.AuditoriaRef) && !r.ConsumidaEn.IsZero() && r.ConsumidaEn.Location().String() == "UTC" && !r.ConsumidaEn.Before(z.EmitidaEn()) && r.ConsumidaEn.Before(z.ExpiraEn()) && r.ConsumoNuevo
}
func seguro(v string) bool { return v != "" && len(v) <= 512 && v == strings.TrimSpace(v) }
func huellaHex(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func nuloRutas(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Chan || r.Kind() == reflect.Func || r.Kind() == reflect.Interface || r.Kind() == reflect.Map || r.Kind() == reflect.Pointer || r.Kind() == reflect.Slice) && r.IsNil()
}
