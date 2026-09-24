package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioConsultaOrganizacionHistorica struct {
	autorizador ports.ProveedorAutorizacionConsultaOrganizacionHistorica
	repositorio ports.RepositorioOrganizacionHistorica
}

func NuevoServicioConsultaOrganizacionHistorica(a ports.ProveedorAutorizacionConsultaOrganizacionHistorica, r ports.RepositorioOrganizacionHistorica) (*ServicioConsultaOrganizacionHistorica, error) {
	if dependenciaOrganizacionHistoricaNula(a) || dependenciaOrganizacionHistoricaNula(r) {
		return nil, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return &ServicioConsultaOrganizacionHistorica{autorizador: a, repositorio: r}, nil
}

func (s *ServicioConsultaOrganizacionHistorica) Consultar(ctx context.Context, solicitud domain.SolicitudConsultaOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio ports.ResultadoConsultaOrganizacionHistorica
	if ctx == nil || s == nil || dependenciaOrganizacionHistoricaNula(s.autorizador) || dependenciaOrganizacionHistoricaNula(s.repositorio) {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material, err := domain.NuevoMaterialConsultaOrganizacionHistorica(solicitud)
	if err != nil {
		return vacio, domain.ErrConsultaOrganizacionHistoricaInvalida
	}
	autorizacion, err := s.autorizador.AutorizarConsultaOrganizacionHistorica(ctx, material)
	if err != nil {
		return vacio, errorConsultaOrganizacionOpaco(ctx, err)
	}
	if !autorizacionOrganizacionHistoricaValida(material, autorizacion) {
		return vacio, domain.ErrConsultaOrganizacionHistoricaDenegada
	}
	resultado, err := s.repositorio.ConsultarOrganizacionHistorica(ctx, ports.OrdenConsultaOrganizacionHistorica{Material: material, Autorizacion: autorizacion})
	if err != nil {
		return vacio, errorConsultaOrganizacionOpaco(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !resultadoOrganizacionHistoricaValido(material, autorizacion, resultado) {
		return vacio, domain.ErrOrganizacionHistoricaNoDisponible
	}
	return resultado, nil
}

func autorizacionOrganizacionHistoricaValida(m domain.MaterialConsultaOrganizacionHistorica, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) bool {
	s := m.Solicitud()
	r := m.Recurso()
	h, err := m.HuellaSHA256()
	x := a.ResumenCapacidad()
	return err == nil && a.ValidarEstructura() == nil &&
		a.PersonaVersion() == s.Actor.Instantanea.PersonaVersion && a.PerfilVersion() == s.Actor.Instantanea.PerfilVersion &&
		x.Operacion() == domain.AccionConsultaOrganizacionHistorica && x.AudienciaConsumo() == domain.AudienciaConsultaOrganizacionHistorica &&
		x.EfectoRef() == r.Referencia && x.EfectoHuellaSHA256() == h
}

func resultadoOrganizacionHistoricaValido(m domain.MaterialConsultaOrganizacionHistorica, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, r ports.ResultadoConsultaOrganizacionHistorica) bool {
	s := m.Solicitud().Selector
	p := r.Pagina
	e := r.Evidencia
	x := a.ResumenCapacidad()
	if !selectorResultadoValido(s, p.Selector) || !estadoCoberturaValido(p.Cobertura.Unidades) || !estadoCoberturaValido(p.Cobertura.PuestosTipo) ||
		!estadoCoberturaValido(p.Cobertura.Dotaciones) || !estadoCoberturaValido(p.Cobertura.Plazas) ||
		!estadoCoberturaValido(p.Cobertura.PuestosIndividuales) || !estadoCoberturaValido(p.Cobertura.Vinculos) ||
		!referenciaOpcionalValida(p.VersionRPTRef) || !referenciaOpcionalValida(p.VersionPlantillaRef) ||
		(s.VersionRPTRef != "" && p.VersionRPTRef != "" && p.VersionRPTRef != s.VersionRPTRef) ||
		(s.VersionPlantillaRef != "" && p.VersionPlantillaRef != "" && p.VersionPlantillaRef != s.VersionPlantillaRef) ||
		(p.VersionRPTRef == "" && (len(p.PuestosTipo)+len(p.Dotaciones)+len(p.PuestosIndividuales) != 0 || p.Cobertura.PuestosTipo != "sin_datos" || p.Cobertura.Dotaciones != "sin_datos" || p.Cobertura.PuestosIndividuales != "sin_datos")) ||
		(p.VersionPlantillaRef == "" && (len(p.Plazas) != 0 || p.Cobertura.Plazas != "sin_datos")) ||
		!cursorResultadoValido(p.CursorSiguiente) || p.CursorSiguiente == s.Cursor && p.CursorSiguiente != "" ||
		len(p.Unidades)+len(p.PuestosTipo)+len(p.Dotaciones)+len(p.Plazas)+len(p.PuestosIndividuales)+len(p.Vinculos) > s.Limite ||
		!referenciaResultadoValida(e.ReciboRef) || !referenciaResultadoValida(e.AuditoriaRef) ||
		!huellaResultadoValida(e.ConsumoHuellaSHA256) || e.DecisionRef != x.DecisionRef() || e.EfectoRef != x.EfectoRef() ||
		!instanteResultadoValido(e.ConsultadaEn) || e.ConsultadaEn.Before(x.EmitidaEn()) || !e.ConsultadaEn.Before(x.ExpiraEn()) {
		return false
	}
	if p.Cobertura.Unidades == "sin_datos" && len(p.Unidades) != 0 ||
		p.Cobertura.PuestosTipo == "sin_datos" && len(p.PuestosTipo) != 0 ||
		p.Cobertura.Dotaciones == "sin_datos" && len(p.Dotaciones) != 0 ||
		p.Cobertura.Plazas == "sin_datos" && len(p.Plazas) != 0 ||
		p.Cobertura.PuestosIndividuales == "sin_datos" && len(p.PuestosIndividuales) != 0 ||
		p.Cobertura.Vinculos == "sin_datos" && len(p.Vinculos) != 0 {
		return false
	}
	ids := map[string]struct{}{}
	validarTraza := func(t domain.TrazaOrganizacionHistorica) bool {
		if t.ValidarEn(s) != nil {
			return false
		}
		if _, existe := ids[t.ID]; existe {
			return false
		}
		ids[t.ID] = struct{}{}
		return true
	}
	for _, v := range p.Unidades {
		if !validarTraza(v.Traza) || v.CatalogoID != ports.IDCatalogoOrganizacion || v.CatalogoVersion < 1 || v.CatalogoRevision < 1 ||
			!referenciaResultadoValida(v.ClaveCatalogo) || !referenciaOpcionalValida(v.PadreID) || !tipoUnidadValido(v.Tipo) || !textoResultadoValido(v.Etiqueta) {
			return false
		}
	}
	for _, v := range p.PuestosTipo {
		if !validarTraza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !codigoResultadoValido(v.CodigoFuente) ||
			!referenciaResultadoValida(v.UnidadID) || !textoResultadoValido(v.Denominacion) || !referenciaResultadoValida(v.ClasificacionRef) {
			return false
		}
	}
	for _, v := range p.Dotaciones {
		if !validarTraza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !referenciaResultadoValida(v.PuestoTipoID) || v.Cantidad < 1 || v.Cantidad > 100000 {
			return false
		}
	}
	for _, v := range p.Plazas {
		if !validarTraza(v.Traza) || v.VersionPlantillaRef != p.VersionPlantillaRef || !codigoResultadoValido(v.CodigoFuente) ||
			!referenciaResultadoValida(v.ClasificacionRef) || !referenciaResultadoValida(v.UnidadID) || !estadoPlazaValido(v.EstadoEstructural) {
			return false
		}
	}
	for _, v := range p.PuestosIndividuales {
		if !validarTraza(v.Traza) || v.VersionRPTRef != p.VersionRPTRef || !codigoResultadoValido(v.CodigoFuente) ||
			!referenciaResultadoValida(v.PuestoTipoID) || !referenciaResultadoValida(v.UnidadID) || !estadoPuestoValido(v.EstadoEstructural) {
			return false
		}
	}
	for _, v := range p.Vinculos {
		if !validarTraza(v.Traza) || !referenciaResultadoValida(v.PlazaID) || !referenciaResultadoValida(v.PuestoID) {
			return false
		}
	}
	return true
}

func estadoCoberturaValido(v string) bool {
	return v == "completa" || v == "parcial" || v == "sin_datos"
}
func tipoUnidadValido(v string) bool {
	return v == "delegacion" || v == "centro" || v == "puesto_responsabilidad"
}
func estadoPlazaValido(v string) bool  { return v == "vigente" || v == "amortizada" }
func estadoPuestoValido(v string) bool { return v == "vigente" || v == "suprimido" }
func referenciaResultadoValida(v string) bool {
	return v != "" && len(v) <= 160 && v[0] >= 'a' && v[0] <= 'z' && referenciaOpcionalValida(v)
}
func referenciaOpcionalValida(v string) bool {
	if v == "" {
		return true
	}
	if len(v) < 3 || len(v) > 160 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for _, c := range v {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}
func codigoResultadoValido(v string) bool {
	if v == "" || len(v) > 128 || v[0] == ' ' || v[len(v)-1] == ' ' {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func textoResultadoValido(v string) bool {
	if v == "" || len([]rune(v)) > 512 || v[0] == ' ' || v[len(v)-1] == ' ' {
		return false
	}
	for _, c := range v {
		if c < 32 || c == 127 {
			return false
		}
	}
	return true
}
func cursorResultadoValido(v string) bool {
	if len(v) > 256 {
		return false
	}
	for _, c := range v {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
func huellaResultadoValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
func instanteResultadoValido(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}
func selectorResultadoValido(esperado, recibido domain.SelectorOrganizacionHistorica) bool {
	return recibido.Validar() == nil && esperado.OrganismoRef == recibido.OrganismoRef &&
		esperado.UnidadClave == recibido.UnidadClave && esperado.VigenteEn == recibido.VigenteEn &&
		esperado.ConocidoEn.Equal(recibido.ConocidoEn) && esperado.VersionRPTRef == recibido.VersionRPTRef &&
		esperado.VersionPlantillaRef == recibido.VersionPlantillaRef && esperado.Limite == recibido.Limite && esperado.Cursor == recibido.Cursor
}
func dependenciaOrganizacionHistoricaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Pointer || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice) && r.IsNil()
}
func errorConsultaOrganizacionOpaco(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, domain.ErrConsultaOrganizacionHistoricaDenegada) {
		return domain.ErrConsultaOrganizacionHistoricaDenegada
	}
	return domain.ErrOrganizacionHistoricaNoDisponible
}
