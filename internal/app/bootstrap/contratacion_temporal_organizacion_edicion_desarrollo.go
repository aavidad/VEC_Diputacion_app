package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"mime"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	personalcatalogos "vec-diputacion-granada/internal/modules/personal/adapters/catalogosvec"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const rutaCambiosOrganizacionContratacionTemporalDesarrollo = "/api/vec/contratacion-temporal/organizacion/cambios"
const finalidadOrganizacionDesarrollo = "gestionar_estructura_organizativa"

type claveMaterialOrganizacionDesarrollo struct{}

type proveedorOrganizacionDesarrollo struct {
	alta  *dependenciasAltaContratacionTemporalDesarrollo
	reloj relojContratacionTemporalDesarrollo
}

func nuevasRutasOrganizacionContratacionTemporalDesarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	editor := &manejadorEdicionOrganizacionDesarrollo{}
	if !cfg.PersonalOrganizacionPostgreSQL {
		consulta, err := nuevaRutaOrganizacionContratacionTemporalDesarrollo(cfg)
		return []vechttp.RutaExacta{consulta, {Ruta: rutaCambiosOrganizacionContratacionTemporalDesarrollo, Manejador: editor}}, err
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		alta.postgresql.gobierno == nil || alta.postgresql.proveedorMaterial == nil || alta.autorizador == nil {
		return nil, personalports.ErrCambioOrganizacionNoDisponible
	}
	proveedor := &proveedorOrganizacionDesarrollo{alta: alta, reloj: reloj}
	repositorio, err := personalpg.NuevoRepositorioOrganizacionPostgreSQL(alta.postgresql.ejecucion, proveedor, reloj)
	if err != nil {
		return nil, err
	}
	consulta, err := personalcatalogos.NuevaConsultaEstructuraOrganizativa(repositorio, personalports.IDCatalogoOrganizacion, cfg.PersonalOrganizacionVersion)
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	// La semilla y las migraciones se instalan explícitamente. No hay fallback
	// al fichero si falla el almacén seleccionado, ni importación al arrancar.
	if _, err := consulta.Obtener(ctx); err != nil {
		return nil, err
	}
	if err := configurarAutoridadOrganizacionDesarrollo(ctx, alta, reloj); err != nil {
		return nil, err
	}
	editor.proveedor, editor.repositorio, editor.version = proveedor, repositorio, cfg.PersonalOrganizacionVersion
	return []vechttp.RutaExacta{
		{Ruta: rutaOrganizacionContratacionTemporalDesarrollo, Manejador: &manejadorOrganizacionContratacionTemporalDesarrollo{consulta: consulta, edicionHabilitada: true}},
		{Ruta: rutaCambiosOrganizacionContratacionTemporalDesarrollo, Manejador: editor},
	}, nil
}

func (p *proveedorOrganizacionDesarrollo) ActorOrganizacion(ctx context.Context) (string, error) {
	if p == nil || p.alta == nil || p.alta.soporte == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return "", personalports.ErrCambioOrganizacionDenegado
	}
	c, ok := p.alta.soporte.capacidadValida(ctx)
	if !ok || c.ruta != rutaCambiosOrganizacionContratacionTemporalDesarrollo || c.certificadoVerificadoEn.IsZero() ||
		!p.reloj.Ahora().Before(c.certificadoValidoHasta) {
		return "", personalports.ErrCambioOrganizacionDenegado
	}
	v, err := p.alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return "", personalports.ErrCambioOrganizacionDenegado
	}
	return v.PrincipalID, nil
}

func (p *proveedorOrganizacionDesarrollo) AutorizarCambioOrganizacion(ctx context.Context, m personalports.MaterialCambioOrganizacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	actor, err := p.ActorOrganizacion(ctx)
	if err != nil || m.Validar() != nil || m.ActorID != actor || p.alta.postgresql.proveedorMaterial == nil {
		return vacio, personalports.ErrCambioOrganizacionDenegado
	}
	recurso, err := personalpg.RecursoCambioOrganizacion(m)
	if err != nil {
		return vacio, err
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	datos := vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: p.alta.soporte.contexto.Vinculo,
		ReferenciaMotivo:          motivoOrganizacionDesarrollo(), Accion: personalports.AccionCambioOrganizacion,
		Recurso: recurso, Finalidad: finalidadOrganizacionDesarrollo, Correlacion: correlacion,
	}
	ctx = context.WithValue(ctx, claveMaterialOrganizacionDesarrollo{}, m)
	if !solicitudAutorizacionOrganizacionDesarrolloValida(ctx, datos) {
		return vacio, personalports.ErrCambioOrganizacionDenegado
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacio, err
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := p.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, p.alta.soporte.contexto.Resultado)
	if err != nil {
		return vacio, err
	}
	return p.alta.postgresql.proveedorMaterial.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion,
		motivoOrganizacionDesarrollo(), p.alta.soporte.contexto.Resultado)
}

func motivoOrganizacionDesarrollo() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_organizacion_preparacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("organizacion-preparacion-desarrollo-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "organizacion-preparacion")}
}

func solicitudAutorizacionOrganizacionDesarrolloValida(ctx context.Context, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	if contextoInterfazNulo(ctx) || d.Accion != personalports.AccionCambioOrganizacion ||
		d.Finalidad != finalidadOrganizacionDesarrollo || d.ReferenciaMotivo != motivoOrganizacionDesarrollo() {
		return false
	}
	m, ok := ctx.Value(claveMaterialOrganizacionDesarrollo{}).(personalports.MaterialCambioOrganizacion)
	if !ok || m.Validar() != nil {
		return false
	}
	v, err := d.VinculoAutenticacionActor.Datos()
	if err != nil || v.PrincipalID != m.ActorID {
		return false
	}
	esperado, err := personalpg.RecursoCambioOrganizacion(m)
	r := d.Recurso
	return err == nil && r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo &&
		maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
}

func configurarAutoridadOrganizacionDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) error {
	v, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return personalports.ErrCambioOrganizacionDenegado
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, reloj.Ahora(),
		"organizacion_preparacion", "Preparación de organización", "organizacion-preparacion-desarrollo",
		[]vecdomain.ConcesionRol{{Accion: personalports.AccionCambioOrganizacion, ModuloID: "personal", TipoRecurso: personalports.TipoCambioOrganizacion,
			Finalidades: []string{finalidadOrganizacionDesarrollo}, GarantiaMinima: vecdomain.AuthAssuranceHigh}},
		[]vecdomain.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vecdomain.ReferenciaEntradaCatalogo{motivoOrganizacionDesarrollo()}, desde); err != nil {
		return err
	}
	alta.soporte.mu.Lock()
	alta.soporte.instantaneaOrganizacion = instantanea
	alta.soporte.mu.Unlock()
	return nil
}

type manejadorEdicionOrganizacionDesarrollo struct {
	proveedor   personalpg.ProveedorCambioOrganizacion
	repositorio personalports.RepositorioCambiosOrganizacion
	version     int
}

func (m *manejadorEdicionOrganizacionDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	prepararCabecerasCatalogosAltaContratacionTemporalDesarrollo(w)
	fallo := func(estado int, codigo string) {
		responderErrorCatalogosAltaContratacionTemporalDesarrollo(w, r, estado, codigo)
	}
	if r == nil || r.URL == nil || r.URL.Path != rutaCambiosOrganizacionContratacionTemporalDesarrollo || r.URL.RawQuery != "" ||
		len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || cabeceraCatalogosAltaContratacionTemporalDesarrolloProhibida(r.Header) {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		fallo(http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if m == nil || m.proveedor == nil || m.repositorio == nil {
		fallo(http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if _, err := m.proveedor.ActorOrganizacion(r.Context()); err != nil {
		fallo(http.StatusUnauthorized, "operacion_denegada")
		return
	}
	tipo, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || (len(parametros) > 0 && (len(parametros) != 1 || !strings.EqualFold(parametros["charset"], "utf-8"))) ||
		r.Header.Get("Content-Encoding") != "" || r.ContentLength > 16*1024 || r.Body == nil {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16*1024))
	if err != nil || validarClavesJSONUnicas(b) != nil {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	var s personalports.SolicitudCambioOrganizacion
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&s) != nil || s.Validar() != nil || s.CatalogoVersion != m.version {
		fallo(http.StatusBadRequest, "solicitud_invalida")
		return
	}
	recibo, err := m.repositorio.GuardarCambio(r.Context(), s)
	if err != nil {
		switch {
		case errors.Is(err, personalports.ErrClaveCambioOrganizacionUsada), errors.Is(err, personalports.ErrRevisionCambioOrganizacionEnConflicto), errors.Is(err, personaldomain.ErrRevisionOrganizacionEnConflicto):
			fallo(http.StatusConflict, "organizacion_en_conflicto")
		case errors.Is(err, personalports.ErrSolicitudCambioOrganizacionInvalida), errors.Is(err, personaldomain.ErrCambioOrganizacionInvalido):
			fallo(http.StatusBadRequest, "solicitud_invalida")
		case errors.Is(err, personalports.ErrCambioOrganizacionDenegado), errors.Is(err, vecdomain.ErrAutorizacionDenegada):
			fallo(http.StatusForbidden, "operacion_denegada")
		default:
			fallo(http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		Data personalports.ReciboCambioOrganizacion `json:"data"`
	}{recibo})
}
