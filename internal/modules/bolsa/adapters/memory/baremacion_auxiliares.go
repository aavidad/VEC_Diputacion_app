package memory

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"time"

	transaccionbolsa "vec-diputacion-granada/internal/modules/bolsa/internal/transaccion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func accionReserva(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionReservarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionReservarDecisionBaremacion, true
	default:
		return "", false
	}
}

func accionConfirmacion(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionConfirmarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionConfirmarDecisionBaremacion, true
	default:
		return "", false
	}
}

func contextosConfirmacionVigentes(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	accion puertosbolsa.AccionOperacionBaremacion,
	instante time.Time,
) bool {
	if solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.Agregado.ID, instante,
	) != nil {
		return false
	}
	if solicitud.Clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return solicitud.ContextoPrevalidacionArchivo.EsNulo()
	}
	return solicitud.Clase == puertosbolsa.ClaseCambioIncorporarDecision &&
		solicitud.ContextoPrevalidacionArchivo.ValidarVigentePara(
			puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
			puertosbolsa.ClaseRecursoBaremacion, solicitud.Agregado.ID, instante,
		) == nil
}

func accionAbandono(clase puertosbolsa.ClaseCambioBaremacion) (puertosbolsa.AccionOperacionBaremacion, bool) {
	switch clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		return puertosbolsa.AccionAbandonarAltaBaremacion, true
	case puertosbolsa.ClaseCambioIncorporarDecision:
		return puertosbolsa.AccionAbandonarDecisionBaremacion, true
	default:
		return "", false
	}
}

func confirmacionCorrespondeAReserva(
	confirmacion puertosbolsa.SolicitudConfirmarCambioBaremacion,
	reserva puertosbolsa.SolicitudReservarCambioBaremacion,
) bool {
	return mismoVinculoOperacion(confirmacion.Contexto, reserva.Contexto) && confirmacion.Clase == reserva.Clase &&
		confirmacion.Agregado.ID == reserva.BaremacionMeritoRef &&
		referenciasVersionOpcionalesIguales(confirmacion.VersionEsperada, reserva.VersionEsperada)
}

func solicitudesReservaIguales(
	a, b puertosbolsa.SolicitudReservarCambioBaremacion,
) bool {
	return proyeccionesAutorizacionIguales(a.Contexto, b.Contexto) && a.Clase == b.Clase && a.ClaveIdempotencia == b.ClaveIdempotencia &&
		a.BaremacionMeritoRef == b.BaremacionMeritoRef &&
		referenciasVersionOpcionalesIguales(a.VersionEsperada, b.VersionEsperada) &&
		cadenasConstantesIguales(a.HuellaSolicitudHMAC, b.HuellaSolicitudHMAC) &&
		a.SolicitadaEn.Equal(b.SolicitadaEn) && a.ExpiraEn.Equal(b.ExpiraEn)
}

func proyeccionesAutorizacionIguales(a, b puertosbolsa.ContextoOperacionBaremacion) bool {
	return a.CoincideExactamenteCon(b)
}

func mismoVinculoOperacion(a, b puertosbolsa.ContextoOperacionBaremacion) bool {
	pa, pb := a.Proyeccion(), b.Proyeccion()
	return a.MismoVinculoAutenticacionQue(b) && pa.RecursoRef == pb.RecursoRef &&
		pa.FinalidadClave == pb.FinalidadClave &&
		pa.CorrelacionRef == pb.CorrelacionRef
}

func huellaEfectoReserva(solicitud puertosbolsa.SolicitudReservarCambioBaremacion) (string, error) {
	return transaccionbolsa.HuellaEfectoReserva(solicitud)
}

func huellaEfectoConfirmacion(solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error) {
	return transaccionbolsa.HuellaEfectoConfirmacionV2(solicitud)
}

func huellaEfectoPrevalidacionArchivo(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (string, error) {
	return transaccionbolsa.HuellaEfectoPrevalidacionArchivoProbatorio(solicitud)
}

type usosAutorizacionConfirmacion struct {
	confirmacion         usoAutorizacionBaremacion
	prevalidacion        usoAutorizacionBaremacion
	incluyePrevalidacion bool
}

func nuevosUsosAutorizacionConfirmacion(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	instante time.Time,
	huellaConfirmacion string,
) (usosAutorizacionConfirmacion, error) {
	confirmacion, err := nuevoUsoAutorizacionBaremacion(
		solicitud.Contexto, instante, huellaConfirmacion,
	)
	if err != nil {
		return usosAutorizacionConfirmacion{}, err
	}
	if solicitud.Clase == puertosbolsa.ClaseCambioAltaBaremacion {
		return usosAutorizacionConfirmacion{confirmacion: confirmacion}, nil
	}
	huellaPrevalidacion, err := huellaEfectoPrevalidacionArchivo(solicitud)
	if err != nil {
		return usosAutorizacionConfirmacion{}, err
	}
	prevalidacion, err := nuevoUsoAutorizacionBaremacion(
		solicitud.ContextoPrevalidacionArchivo, instante, huellaPrevalidacion,
	)
	if err != nil || confirmacion.DecisionRef == prevalidacion.DecisionRef {
		return usosAutorizacionConfirmacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return usosAutorizacionConfirmacion{
		confirmacion: confirmacion, prevalidacion: prevalidacion, incluyePrevalidacion: true,
	}, nil
}

func nuevoUsoAutorizacionBaremacion(
	contexto puertosbolsa.ContextoOperacionBaremacion,
	instante time.Time,
	huellaEfecto string,
) (usoAutorizacionBaremacion, error) {
	evidencia, err := contexto.EvidenciaUsoAutorizacion()
	if err != nil || evidencia.ValidarEn(instante) != nil || !huellaSHA256MemoriaValida(huellaEfecto) {
		return usoAutorizacionBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	datos, err := evidencia.Datos()
	proyeccion := contexto.Proyeccion()
	if err != nil || datos.Decision.DecisionRef != proyeccion.AutorizacionRef ||
		!huellaSHA256MemoriaValida(datos.HuellaDecisionSHA256) {
		return usoAutorizacionBaremacion{}, puertosbolsa.ErrAutorizacionBaremacionInvalida
	}
	return usoAutorizacionBaremacion{
		DecisionRef:          datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		HuellaEfectoSHA256:   huellaEfecto,
	}, nil
}

func (r *RepositorioBaremaciones) comprobarUsoAutorizacionBloqueado(
	uso usoAutorizacionBaremacion,
) (bool, error) {
	if r == nil || r.usosAutorizacion == nil || uso.DecisionRef == "" ||
		!huellaSHA256MemoriaValida(uso.HuellaDecisionSHA256) || !huellaSHA256MemoriaValida(uso.HuellaEfectoSHA256) {
		return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	existente, consumida := r.usosAutorizacion[uso.DecisionRef]
	if !consumida {
		if len(r.usosAutorizacion) >= maximoUsosAutorizacionMemoria {
			return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		return false, nil
	}
	if !cadenasConstantesIguales(existente.HuellaDecisionSHA256, uso.HuellaDecisionSHA256) ||
		!cadenasConstantesIguales(existente.HuellaEfectoSHA256, uso.HuellaEfectoSHA256) {
		return true, puertosbolsa.ErrAutorizacionBaremacionReutilizada
	}
	return true, nil
}

func (r *RepositorioBaremaciones) comprobarUsosConfirmacionBloqueados(
	usos usosAutorizacionConfirmacion,
) (bool, error) {
	confirmacionConsumida, err := r.comprobarUsoAutorizacionBloqueado(usos.confirmacion)
	if err != nil {
		return false, err
	}
	if !usos.incluyePrevalidacion {
		return confirmacionConsumida, nil
	}
	prevalidacionConsumida, err := r.comprobarUsoAutorizacionBloqueado(usos.prevalidacion)
	if err != nil {
		return false, err
	}
	if confirmacionConsumida != prevalidacionConsumida {
		return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	if !confirmacionConsumida && len(r.usosAutorizacion)+2 > maximoUsosAutorizacionMemoria {
		return false, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	return confirmacionConsumida, nil
}

func huellaSHA256MemoriaValida(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	bytes, err := hex.DecodeString(valor)
	return err == nil && len(bytes) == sha256.Size
}

func huellaConfirmacion(s puertosbolsa.SolicitudConfirmarCambioBaremacion) (string, error) {
	return huellaEfectoConfirmacion(s)
}

func derivarEvidenciaTransaccion(
	solicitud puertosbolsa.SolicitudConfirmarCambioBaremacion,
	versionAnterior, versionNueva uint64,
	huellaAnterior, huellaNueva, huellaAuditoriaAnterior, huellaEventoAnterior string,
	secuenciaAuditoria, secuenciaEvento uint64,
	registradaEn time.Time,
) (registroAuditoriaBaremacion, eventoOutboxBaremacion, puertosbolsa.EvidenciaTransaccionBaremacion, error) {
	return transaccionbolsa.DerivarEvidencia(
		solicitud, versionAnterior, versionNueva, huellaAnterior, huellaNueva,
		huellaAuditoriaAnterior, huellaEventoAnterior, secuenciaAuditoria, secuenciaEvento, registradaEn,
	)
}

func huellaAuditoria(a registroAuditoriaBaremacion) string {
	return transaccionbolsa.HuellaAuditoria(a)
}

func huellaEvento(e eventoOutboxBaremacion) string {
	return transaccionbolsa.HuellaEvento(e)
}

func huellaCanonica(partes ...string) string {
	return transaccionbolsa.HuellaCanonica(partes...)
}

func generarTokenReserva() (puertosbolsa.TokenReservaBaremacion, error) {
	return transaccionbolsa.GenerarTokenReserva()
}

func claveAmbitoReserva(principalRef, claveIdempotencia string) string {
	return strconv.Itoa(len(principalRef)) + ":" + principalRef + claveIdempotencia
}

func referenciasVersionOpcionalesIguales(a, b *puertosbolsa.ReferenciaVersionBaremacion) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return referenciasVersionIguales(a, b)
}

func referenciasVersionIguales(a, b *puertosbolsa.ReferenciaVersionBaremacion) bool {
	return a != nil && b != nil && a.BaremacionMeritoRef == b.BaremacionMeritoRef && a.Numero == b.Numero &&
		cadenasConstantesIguales(a.HuellaEstadoSHA256, b.HuellaEstadoSHA256)
}

func huellaTokenReserva(token puertosbolsa.TokenReservaBaremacion) string {
	return transaccionbolsa.HuellaTokenReserva(token)
}

func cadenasConstantesIguales(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func validarContextoEjecucion(ctx context.Context) error {
	if ctx == nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	return ctx.Err()
}

func errorVerificacionConContexto(ctx context.Context, falloCerrado error) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	return falloCerrado
}

func interfazNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
