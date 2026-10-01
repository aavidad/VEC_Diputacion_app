package ejecucioncopias

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

func aprobacionVigente(a puertos.Aprobacion, p puertos.Propuesta) bool {
	return a.ProponentePersonaRef != "" && a.AprobadorPersonaRef != "" && a.ProponentePersonaRef != a.AprobadorPersonaRef && a.HuellaPropuesta == p.HuellaPropuesta && a.PreimagenSHA256 == p.PreimagenSHA256 && a.DecisionRef != "" && time.Now().UTC().Before(a.Vence)
}

func (s *Servicio) aprobar(ctx context.Context, p puertos.Propuesta) error {
	a, err := s.d.Autorizador.Aprobar(ctx, p)
	if err != nil {
		return err
	}
	if !aprobacionVigente(a, p) {
		return denegar("doble_control_no_vigente")
	}
	return nil
}

func (s *Servicio) Restaurar(ctx context.Context, p puertos.Propuesta) (resultado Recibo, retErr error) {
	if s == nil || s.d.Plataforma == nil {
		return Recibo{}, denegar("plataforma_restauracion_ausente")
	}
	if p.OperacionRef == "" || p.ActorRef == "" || p.OrigenRef == "" || p.DestinoRef == "" || p.MotivoRef == "" || p.ConjuntoRef == "" || p.ConjuntoPreviaRef == "" || p.ConjuntoPreviaRef == p.ConjuntoRef || p.PoliticaRef == "" || p.HuellaPropuesta == "" || p.PreimagenSHA256 == "" {
		return Recibo{}, denegar("propuesta_incompleta")
	}
	if !huellaValida(p.HuellaPropuesta) || !huellaValida(p.PreimagenSHA256) {
		return Recibo{}, denegar("huellas_propuesta_invalidas")
	}
	if err := s.autorizar(ctx, p.Peticion, "restaurar"); err != nil {
		return Recibo{}, err
	}
	if err := s.aprobar(ctx, p); err != nil {
		return Recibo{}, err
	}
	l, err := s.lectura(ctx, p.DestinoRef)
	if err != nil {
		return Recibo{}, err
	}
	if l.Politica.Ref != p.PoliticaRef {
		return Recibo{}, denegar("politica_distinta")
	}
	preimagen := l.PreimagenSHA256
	if !huellaValida(preimagen) {
		return Recibo{}, denegar("preimagen_actual_no_comprobable")
	}
	if p.PreimagenSHA256 != preimagen {
		return Recibo{}, denegar("preimagen_diferente")
	}
	objetivo, err := s.d.Destino.Recuperar(ctx, p.ConjuntoRef)
	if err != nil {
		return Recibo{}, err
	}
	if err := validarConjunto(objetivo, "valida"); err != nil {
		return Recibo{}, err
	}
	if err := s.copiaRestaurable(ctx, objetivo); err != nil {
		return Recibo{}, err
	}
	if objetivo.Manifiesto.PoliticaRef != l.Politica.Ref || !mismoContenido(objetivo.Origen, objetivo.Manifiesto.Verificacion.Fisica) || !mismoContenido(objetivo.Origen, objetivo.Manifiesto.Verificacion.Logica) {
		return Recibo{}, denegar("copia_sin_verificacion_coherente")
	}
	if copias.CompararVersiones(objetivo.Manifiesto, l.Observado, l.Politica, copias.ConjuntoCompleto).Estado != copias.Compatible {
		return Recibo{}, denegar("conjunto_incompatible")
	}
	op, err := s.d.Registro.ReservarRestauracion(ctx, p)
	if err != nil {
		return Recibo{}, err
	}
	if op.Ref != p.OperacionRef || op.VersionRef == "" || op.ConjuntoRef != objetivo.Ref || op.ConjuntoPreviaPlaneadaRef != p.ConjuntoPreviaRef || op.PoliticaRef != p.PoliticaRef || op.PreimagenSHA256 != preimagen || op.HuellaPropuesta != p.HuellaPropuesta {
		return Recibo{}, denegar("reserva_restauracion_no_vinculada")
	}
	if op.Estado != "solicitada" {
		return Recibo{}, denegar("restauracion_existente_requiere_conciliacion")
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, "exclusion_solicitada", preimagen); err != nil {
		return Recibo{}, err
	}
	x, err := s.d.Ventana.AbrirRestauracion(ctx, p.Peticion)
	if err != nil {
		return Recibo{}, err
	}
	// Antes del CAS no existe efecto destructivo: cerrar al fallar. Después del
	// CAS se conserva mantenimiento, incluso si el proceso muere.
	efecto := false
	mantenimientoConfirmado := false
	defer func() {
		if efecto {
			if !mantenimientoConfirmado {
				if err := x.ConservarMantenimiento(context.WithoutCancel(ctx)); err != nil {
					resultado = Recibo{}
					retErr = errors.Join(retErr, ErrConciliacion)
				}
			}
		} else {
			if err := x.Cerrar(context.WithoutCancel(ctx)); err != nil {
				resultado = Recibo{}
				retErr = errors.Join(retErr, err)
			}
		}
	}()
	if err := s.autorizar(ctx, p.Peticion, "restaurar"); err != nil {
		return Recibo{}, err
	}
	if err := s.aprobar(ctx, p); err != nil {
		return Recibo{}, err
	}
	actual, err := x.PreimagenActual(ctx)
	if err != nil {
		return Recibo{}, err
	}
	if actual != preimagen {
		return Recibo{}, denegar("preimagen_bajo_exclusion_diferente")
	}
	peticionPrevia := p.Peticion
	peticionPrevia.ConjuntoRef = p.ConjuntoPreviaRef
	previa, err := x.CapturarPrevia(ctx, peticionPrevia, l)
	if err != nil {
		return Recibo{}, err
	}
	if previa.Manifiesto.ConjuntoRef != p.ConjuntoPreviaRef {
		return Recibo{}, denegar("copia_previa_sin_identidad_propia")
	}
	if err := validarCaptura(previa, l, peticionPrevia); err != nil {
		return Recibo{}, err
	}
	previo, err := s.sellarYEnsayar(ctx, previa)
	if err != nil {
		return Recibo{}, err
	}
	if previo.Ref == objetivo.Ref {
		return Recibo{}, denegar("copia_previa_igual_objetivo")
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, "copia_previa_verificada", previo.Ref); err != nil {
		return Recibo{}, err
	}
	// Recuperar de nuevo verifica el índice y todos los componentes tras la
	// captura previa, antes de tocar la plataforma principal.
	objetivo, err = s.d.Destino.Recuperar(ctx, p.ConjuntoRef)
	if err != nil {
		return Recibo{}, err
	}
	if err := validarConjunto(objetivo, "valida"); err != nil {
		return Recibo{}, err
	}
	if err := s.copiaRestaurable(ctx, objetivo); err != nil {
		return Recibo{}, err
	}
	preparado, err := s.d.Plataforma.Preparar(ctx, objetivo)
	if err != nil {
		return Recibo{}, err
	}
	if preparado.ConjuntoRef != objetivo.Ref || preparado.PlanRef == "" {
		return Recibo{}, denegar("plan_no_vinculado")
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, "plan_preparado", preparado.PlanRef); err != nil {
		return Recibo{}, err
	}
	// El control actual y la preimagen se comprueban inmediatamente antes del
	// CAS. La plataforma recibirá el identificador durable del plan y no rutas.
	if err := s.autorizar(ctx, p.Peticion, "restaurar"); err != nil {
		return Recibo{}, err
	}
	if err := s.aprobar(ctx, p); err != nil {
		return Recibo{}, err
	}
	lecturaFinal, err := s.lectura(ctx, p.DestinoRef)
	if err != nil {
		return Recibo{}, err
	}
	if lecturaFinal.Politica.Ref != l.Politica.Ref || lecturaFinal.PreimagenSHA256 != preimagen || copias.HuellaInventario(lecturaFinal.Observado) != copias.HuellaInventario(l.Observado) || copias.CompararVersiones(objetivo.Manifiesto, lecturaFinal.Observado, lecturaFinal.Politica, copias.ConjuntoCompleto).Estado != copias.Compatible {
		return Recibo{}, denegar("politica_o_conjunto_cambiado")
	}
	objetivoFinal, err := s.d.Destino.Recuperar(ctx, objetivo.Ref)
	if err != nil {
		return Recibo{}, err
	}
	if objetivoFinal.ManifiestoSHA256 != objetivo.ManifiestoSHA256 || objetivoFinal.IndiceAutenticadoRef != objetivo.IndiceAutenticadoRef || !reflect.DeepEqual(objetivoFinal.Manifiesto, objetivo.Manifiesto) || !reflect.DeepEqual(objetivoFinal.Origen, objetivo.Origen) {
		return Recibo{}, denegar("conjunto_cambiado_antes_del_efecto")
	}
	if err := s.copiaRestaurable(ctx, objetivoFinal); err != nil {
		return Recibo{}, err
	}
	actual, err = x.PreimagenActual(ctx)
	if err != nil {
		return Recibo{}, err
	}
	if actual != preimagen {
		return Recibo{}, denegar("preimagen_final_diferente")
	}
	actualOp, err := s.d.Registro.Leer(ctx, op.Ref)
	if err != nil {
		return Recibo{}, err
	}
	if actualOp.CopiaPreviaRef != previo.Ref || actualOp.ConjuntoPreviaPlaneadaRef != p.ConjuntoPreviaRef || actualOp.PlanRef != preparado.PlanRef || actualOp.ConjuntoRef != objetivo.Ref || actualOp.PoliticaRef != p.PoliticaRef || actualOp.PreimagenSHA256 != preimagen || actualOp.HuellaPropuesta != p.HuellaPropuesta || actualOp.VersionRef == "" {
		return Recibo{}, denegar("diario_incompleto")
	}
	cas, err := s.d.Registro.CAS(ctx, op.Ref, actualOp.VersionRef, preimagen, "sustitucion_iniciada")
	if err != nil {
		return Recibo{}, err
	}
	if cas.Estado != "sustitucion_iniciada" || cas.Ref != op.Ref {
		return Recibo{}, denegar("cas_no_confirmado")
	}
	efecto = true
	if err := s.d.Plataforma.Sustituir(ctx, preparado, cas.Ref); err != nil {
		_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "pendiente_conciliacion", "sustitucion_interrumpida")
		return Recibo{}, fmt.Errorf("sustitucion_interrumpida: %w", ErrConciliacion)
	}
	instalado, err := s.d.Plataforma.IdentificarInstalado(ctx, p.DestinoRef)
	if err != nil || instalado != objetivo.Ref {
		_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "pendiente_conciliacion", "instalacion_no_identificada")
		return Recibo{}, ErrConciliacion
	}
	arranqueRef, err := s.d.Plataforma.ArrancarAislado(ctx, preparado)
	if err != nil || arranqueRef == "" {
		_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "reversion_iniciada", previo.Ref)
		if er := s.d.Plataforma.Revertir(context.WithoutCancel(ctx), previo, op.Ref); er != nil {
			_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "pendiente_conciliacion", "reversion_interrumpida")
			return Recibo{}, ErrConciliacion
		}
		if anterior, er := s.d.Plataforma.IdentificarInstalado(context.WithoutCancel(ctx), p.DestinoRef); er != nil || anterior != previo.Ref {
			_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "pendiente_conciliacion", "reversion_no_identificada")
			return Recibo{}, ErrConciliacion
		}
		_ = s.d.Registro.Anotar(context.WithoutCancel(ctx), op.Ref, "revertida", previo.Ref)
		return Recibo{}, denegar("arranque_del_conjunto_fallido")
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, "instalado_pendiente_conciliacion", objetivo.Ref); err != nil {
		return Recibo{}, ErrConciliacion
	}
	if err := x.ConservarMantenimiento(ctx); err != nil {
		return Recibo{}, ErrConciliacion
	}
	mantenimientoConfirmado = true
	return Recibo{op.Ref, objetivo.Ref, "instalado_pendiente_conciliacion", objetivo.IndiceAutenticadoRef}, nil
}

// Conciliar observa el estado real tras reinicio. Nunca repite Sustituir. El
// desbloqueo de despachos requiere una operación de autoridad separada.
func (s *Servicio) Conciliar(ctx context.Context, p puertos.Propuesta) (Recibo, error) {
	if s == nil || s.d.Plataforma == nil {
		return Recibo{}, denegar("plataforma_restauracion_ausente")
	}
	if err := s.autorizar(ctx, p.Peticion, "restaurar"); err != nil {
		return Recibo{}, err
	}
	op, err := s.d.Registro.Leer(ctx, p.OperacionRef)
	if err != nil {
		return Recibo{}, err
	}
	if op.Ref != p.OperacionRef || op.ConjuntoRef != p.ConjuntoRef || op.ConjuntoPreviaPlaneadaRef != p.ConjuntoPreviaRef || op.CopiaPreviaRef != p.ConjuntoPreviaRef || op.PoliticaRef != p.PoliticaRef || op.HuellaPropuesta != p.HuellaPropuesta || op.PreimagenSHA256 != p.PreimagenSHA256 || op.PlanRef == "" {
		return Recibo{}, denegar("diario_no_conciliable")
	}
	if op.Estado != "pendiente_conciliacion" && op.Estado != "instalado_pendiente_conciliacion" && op.Estado != "sustitucion_iniciada" {
		return Recibo{}, denegar("estado_no_conciliable")
	}
	instalado, err := s.d.Plataforma.IdentificarInstalado(ctx, p.DestinoRef)
	if err != nil {
		return Recibo{}, ErrConciliacion
	}
	if instalado != op.ConjuntoRef && instalado != op.CopiaPreviaRef {
		return Recibo{}, ErrConciliacion
	}
	c, err := s.d.Destino.Recuperar(ctx, instalado)
	if err != nil {
		return Recibo{}, ErrConciliacion
	}
	if err := validarConjunto(c, "valida"); err != nil {
		return Recibo{}, ErrConciliacion
	}
	estado := "revertida"
	if instalado == op.ConjuntoRef {
		estado = "instalado_pendiente_conciliacion"
	}
	if err := s.d.Registro.Anotar(ctx, op.Ref, estado, instalado); err != nil {
		return Recibo{}, ErrConciliacion
	}
	return Recibo{op.Ref, instalado, estado, c.IndiceAutenticadoRef}, nil
}
