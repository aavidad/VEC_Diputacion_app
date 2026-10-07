package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type errorAuditadoGobiernoBaremoHTTPV3 struct {
	error
	confirmada bool
}

func (e errorAuditadoGobiernoBaremoHTTPV3) Unwrap() error { return e.error }

type claveIntentoGobiernoBaremoHTTPV3 struct{}
type intentoGobiernoBaremoHTTPV3 struct {
	mu          sync.Mutex
	correlacion vd.ReferenciaCorrelacionAutorizacionV2
	recursoRef  string
}

func contextoIntentoGobiernoBaremoHTTPV3(ctx context.Context, recurso string) (context.Context, error) {
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(context.WithoutCancel(ctx), seg.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return nil, err
	}
	return context.WithValue(ctx, claveIntentoGobiernoBaremoHTTPV3{}, &intentoGobiernoBaremoHTTPV3{correlacion: correlacion, recursoRef: recurso}), nil
}
func correlacionIntentoGobiernoBaremoHTTPV3(ctx context.Context) (vd.ReferenciaCorrelacionAutorizacionV2, error) {
	if intento, ok := ctx.Value(claveIntentoGobiernoBaremoHTTPV3{}).(*intentoGobiernoBaremoHTTPV3); ok && intento != nil {
		return intento.correlacion, nil
	}
	return vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
}
func vincularRecursoIntentoGobiernoBaremoHTTPV3(ctx context.Context, recurso string) {
	intento, ok := ctx.Value(claveIntentoGobiernoBaremoHTTPV3{}).(*intentoGobiernoBaremoHTTPV3)
	if !ok || intento == nil {
		return
	}
	prefijo := "reglas-baremo:"
	if strings.HasPrefix(recurso, "intencion-reglas-baremo:") {
		prefijo = "intencion-reglas-baremo:"
	}
	if !strings.HasPrefix(recurso, prefijo) || !shaHexGobiernoV3(strings.TrimPrefix(recurso, prefijo)) {
		return
	}
	intento.mu.Lock()
	intento.recursoRef = recurso
	intento.mu.Unlock()
}
func handlerIntentoGobiernoBaremoHTTPV3(h http.Handler, recurso string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := contextoIntentoGobiernoBaremoHTTPV3(r.Context(), recurso)
		if err != nil {
			(&bolsahttp.HandlerGobiernoReglasBaremoV3{}).ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// La composición recibe causas gobernadas y proceso del servidor. El cliente
// no elige identidad, perfil, canal ni motivo del intento observado.
type auditorGobiernoBaremoHTTPV3 struct {
	sesion                      *proveedorSesionConsultaRRHHDesarrollo
	registrador                 vp.RegistradorIntentosAuditoria
	proceso, recursoRef         string
	motivoDenegado, motivoError vd.ReferenciaEntradaCatalogo
}

func (a *auditorGobiernoBaremoHTTPV3) registrar(ctx context.Context, operativo contextoSeguridadComunDesarrollo, fallo error) error {
	if a == nil || dependenciaEsNulaContratacionTemporalDesarrollo(a.registrador) || fallo == nil || ctx == nil {
		return app.ErrGobiernoV3NoDisponible
	}
	f, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !a.sesion.sesionGobiernoReglasBaremoHTTPV3(ctx, f.ruta) {
		return app.ErrGobiernoV3NoDisponible
	}
	accion, finalidad := "", "consulta_gobierno_reglas_baremo"
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		if par.ruta == f.ruta {
			accion = par.accion
		}
	}
	if accion == "bolsa.reglas_baremo.borrador.crear" {
		finalidad = "gobierno_reglas_baremo"
	}
	resultado, motivo := vd.ResultadoIntentoAuditoriaError, a.motivoError
	if errors.Is(fallo, app.ErrGobiernoV3Prohibido) || errors.Is(fallo, app.ErrGobiernoV3NoAutenticado) {
		resultado, motivo = vd.ResultadoIntentoAuditoriaDenegado, a.motivoDenegado
	}
	intentoPeticion, ok := ctx.Value(claveIntentoGobiernoBaremoHTTPV3{}).(*intentoGobiernoBaremoHTTPV3)
	if !ok || intentoPeticion == nil {
		return app.ErrGobiernoV3NoDisponible
	}
	intentoPeticion.mu.Lock()
	recurso := intentoPeticion.recursoRef
	intentoPeticion.mu.Unlock()
	correlacion := intentoPeticion.correlacion
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return app.ErrGobiernoV3NoDisponible
	}
	intento, err := vp.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return app.ErrGobiernoV3NoDisponible
	}
	orden, err := vp.NuevaOrdenIntentoAuditoria(intento, operativo.Resultado, operativo.Vinculo, vd.DatosIntentoAuditoria{
		Accion: accion, ModuloID: "bolsa", RecursoRef: recurso, FinalidadRef: finalidad,
		Resultado: resultado, Motivo: motivo, Proceso: a.proceso,
		Canal: string(vd.SuperficieAutenticacionInternaCorporativaV1), CorrelacionRef: correlacionRef,
	})
	if err != nil {
		return app.ErrGobiernoV3NoDisponible
	}
	acuse, err := a.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return app.ErrGobiernoV3NoDisponible
	}
	return nil
}

// Conserva la identidad registrada que ya acreditó la frontera. Un fallo o
// una revocación posterior no obliga a emitir otra identidad para auditarlo.
func (a *auditorGobiernoBaremoHTTPV3) contextoHistorico(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if a == nil || ctx == nil {
		return vacio, app.ErrGobiernoV3NoDisponible
	}
	f, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !a.sesion.sesionGobiernoReglasBaremoHTTPV3(ctx, f.ruta) {
		return vacio, app.ErrGobiernoV3NoDisponible
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok || capacidad.contextoOperacion == nil {
		return vacio, app.ErrGobiernoV3NoDisponible
	}
	holder := capacidad.contextoOperacion
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.soporte != a.sesion.soporte || holder.contexto.Vinculo.ValidarPara(holder.contexto.Resultado) != nil {
		return vacio, app.ErrGobiernoV3NoDisponible
	}
	copia, err := holder.contexto.Resultado.Clonar()
	if err != nil {
		return vacio, app.ErrGobiernoV3NoDisponible
	}
	return contextoSeguridadComunDesarrollo{Resultado: copia, Vinculo: holder.contexto.Vinculo}, nil
}

func auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion *proveedorSesionConsultaRRHHDesarrollo, frontera vp.RegistradorAuditoriaFronteraRutaExacta, auditor *auditorGobiernoBaremoHTTPV3) func(context.Context, error, *contextoSeguridadComunDesarrollo) error {
	auditarSesion := auditorRechazoSesionGobiernoReglasBaremoHTTPV3(sesion, frontera)
	return func(ctx context.Context, fallo error, operativo *contextoSeguridadComunDesarrollo) error {
		f, ok := fronteraSeguridadComunDesdeContexto(ctx)
		if !ok || !sesion.sesionGobiernoReglasBaremoHTTPV3(ctx, f.ruta) {
			return app.ErrGobiernoV3NoDisponible
		}
		if operativo == nil {
			historico, err := auditor.contextoHistorico(ctx)
			if err == nil {
				return auditor.registrar(ctx, historico, fallo)
			}
			return auditarSesion(ctx, f.ruta, fallo)
		}
		capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		if !ok || capacidad.contextoOperacion == nil {
			return app.ErrGobiernoV3NoDisponible
		}
		holder := capacidad.contextoOperacion
		holder.mu.Lock()
		esperado, err := holder.contexto.Resultado.Clonar()
		confiable := err == nil && holder.soporte == sesion.soporte && holder.err == nil &&
			reflect.DeepEqual(esperado, operativo.Resultado) &&
			holder.contexto.Vinculo.ValidarPara(operativo.Resultado) == nil &&
			reflect.DeepEqual(holder.contexto.Vinculo, operativo.Vinculo)
		holder.mu.Unlock()
		if !confiable {
			return app.ErrGobiernoV3NoDisponible
		}
		return auditor.registrar(ctx, *operativo, fallo)
	}
}

type operadorAuditadoGobiernoBaremoHTTPV3 struct {
	proveedor *ProveedorGobiernoReglasBaremoV3
	operador  bolsahttp.OperadorGobiernoReglasBaremoV3
	auditor   *auditorGobiernoBaremoHTTPV3
}

func (o *operadorAuditadoGobiernoBaremoHTTPV3) cerrarIntento(ctx context.Context, operativo contextoSeguridadComunDesarrollo, err error) error {
	if err == nil {
		return nil
	}
	var auditado errorAuditadoGobiernoBaremoHTTPV3
	if errors.As(err, &auditado) {
		return err
	}
	if o.auditor.registrar(ctx, operativo, err) != nil {
		return errorAuditadoGobiernoBaremoHTTPV3{app.ErrGobiernoV3NoDisponible, false}
	}
	return errorAuditadoGobiernoBaremoHTTPV3{err, true}
}

func (o *operadorAuditadoGobiernoBaremoHTTPV3) GuardarAltaBorrador(ctx context.Context, c app.CredencialesGobiernoV3, p app.PeticionAltaBorradorV3) (bp.ResultadoAltaBorradorReglasV3, error) {
	operativo, err := o.proveedor.contexto(ctx, "alta_borrador")
	if err != nil {
		return bp.ResultadoAltaBorradorReglasV3{}, err
	}
	resultado, err := o.operador.GuardarAltaBorrador(ctx, c, p)
	err = o.cerrarIntento(ctx, operativo, err)
	if err != nil {
		return bp.ResultadoAltaBorradorReglasV3{}, err
	}
	return resultado, nil
}
func (o *operadorAuditadoGobiernoBaremoHTTPV3) ConsultarExacta(ctx context.Context, c app.CredencialesGobiernoV3, p app.PeticionConsultaExactaV3) (bp.ResultadoConsultaGobiernoReglasV3, error) {
	operativo, err := o.proveedor.contexto(ctx, "consultar_exacta")
	if err != nil {
		return bp.ResultadoConsultaGobiernoReglasV3{}, err
	}
	resultado, err := o.operador.ConsultarExacta(ctx, c, p)
	err = o.cerrarIntento(ctx, operativo, err)
	if err != nil {
		return bp.ResultadoConsultaGobiernoReglasV3{}, err
	}
	return resultado, nil
}
func (o *operadorAuditadoGobiernoBaremoHTTPV3) RecuperarRecibo(ctx context.Context, c app.CredencialesGobiernoV3, p app.PeticionRecuperarReciboV3) (bp.ResultadoRecuperacionGobiernoReglasV3, error) {
	operativo, err := o.proveedor.contexto(ctx, "recuperar_recibo")
	if err != nil {
		return bp.ResultadoRecuperacionGobiernoReglasV3{}, err
	}
	resultado, err := o.operador.RecuperarRecibo(ctx, c, p)
	err = o.cerrarIntento(ctx, operativo, err)
	if err != nil {
		return bp.ResultadoRecuperacionGobiernoReglasV3{}, err
	}
	return resultado, nil
}
