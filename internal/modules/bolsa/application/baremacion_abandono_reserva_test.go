package application

import (
	"context"
	"errors"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestFinalizarFirmaAbandonaReservaSiFallaAntesDeConfirmar(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.autorizador.obligacionEn = puertosbolsa.AccionConfirmarDecisionBaremacion

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "abandono-autorizacion"),
	)
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("no se conservo la causa previa al COMMIT: %v", err)
	}
	if entorno.repositorio.reservas != 1 || entorno.repositorio.confirmaciones != 0 ||
		entorno.repositorio.abandonos != 1 || entorno.repositorio.intentosAbandono != 1 ||
		entorno.repositorio.abandono == nil {
		t.Fatalf("efectos reserva=%d confirmacion=%d abandono=%d",
			entorno.repositorio.reservas, entorno.repositorio.confirmaciones, entorno.repositorio.abandonos)
	}
	abandono := *entorno.repositorio.abandono
	if abandono.Validar() != nil || abandono.Clase != puertosbolsa.ClaseCambioIncorporarDecision ||
		abandono.BaremacionMeritoRef != preparada.decision.decision.revision.contenido.BaremacionMeritoRef ||
		!tokensReservaBaremacionCoinciden(abandono.Token, entorno.repositorio.token) {
		t.Fatal("el abandono no quedo ligado a la reserva exacta")
	}
	acciones := entorno.autorizador.acciones()
	esperadas := []puertosbolsa.AccionOperacionBaremacion{
		puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
		puertosbolsa.AccionConfirmarDecisionBaremacion,
		puertosbolsa.AccionAbandonarDecisionBaremacion,
	}
	if len(acciones) < len(esperadas) ||
		!accionesBaremacionIguales(acciones[len(acciones)-len(esperadas):], esperadas) ||
		entorno.autorizador.referenciasRepetidas() {
		t.Fatalf("autorizaciones de cierre no exclusivas: %v", acciones)
	}
}

func TestFinalizarFirmaAbandonaReservaConContextoDeEntradaCancelado(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	ctx, cancelar := context.WithCancel(context.Background())
	entorno.repositorio.alReservar = cancelar

	_, err := entorno.servicio.FinalizarFirma(
		ctx, ordenFinalizarBaremacionPrueba(preparada, "abandono-cancelacion"),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("no se conservo la cancelacion original: %v", err)
	}
	if entorno.repositorio.reservas != 1 || entorno.repositorio.confirmaciones != 0 ||
		entorno.repositorio.abandonos != 1 || entorno.repositorio.intentosAbandono != 1 {
		t.Fatalf("efectos reserva=%d confirmacion=%d abandono=%d",
			entorno.repositorio.reservas, entorno.repositorio.confirmaciones, entorno.repositorio.abandonos)
	}
}

func TestFinalizarFirmaAcreditaFalloDelAbandonoSinPerderCausa(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.servicio.selladorSolicitud = selladorCentinelaPorFinalidadBaremacionPrueba{
		finalidad: []byte(puertosbolsa.FinalidadSelloConfirmacionBaremacionV2),
	}
	errorAbandono := errors.New("fallo interno de abandono simulado")
	entorno.repositorio.erroresAbandono = []error{errorAbandono, errorAbandono}

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "abandono-fallido"),
	)
	if !errors.Is(err, ErrResultadoBaremacionNoConfiable) ||
		!errors.Is(err, ErrAbandonoReservaBaremacionNoAcreditado) {
		t.Fatalf("clasificacion incompleta del fallo: %v", err)
	}
	if err.Error() != mensajeDocumentoFirmadoHuerfano || entorno.repositorio.abandonos != 0 ||
		entorno.repositorio.intentosAbandono != 2 || entorno.repositorio.confirmaciones != 0 {
		t.Fatalf("error no expurgado o efectos inesperados: %v", err)
	}
}

func TestFinalizarFirmaReintentaAbandonoConLaMismaAutorizacion(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	entorno.servicio.selladorSolicitud = selladorCentinelaPorFinalidadBaremacionPrueba{
		finalidad: []byte(puertosbolsa.FinalidadSelloConfirmacionBaremacionV2),
	}
	entorno.repositorio.erroresAbandono = []error{errors.New("respuesta de abandono perdida")}

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "abandono-reintentado"),
	)
	if !errors.Is(err, ErrResultadoBaremacionNoConfiable) ||
		errors.Is(err, ErrAbandonoReservaBaremacionNoAcreditado) {
		t.Fatalf("resultado del reintento inesperado: %v", err)
	}
	solicitudes := entorno.repositorio.solicitudesAbandono
	if entorno.repositorio.intentosAbandono != 2 || entorno.repositorio.abandonos != 1 ||
		len(solicitudes) != 2 || entorno.repositorio.reserva != nil {
		t.Fatalf("reintento no idempotente: intentos=%d efectos=%d solicitudes=%d",
			entorno.repositorio.intentosAbandono, entorno.repositorio.abandonos, len(solicitudes))
	}
	primera, segunda := solicitudes[0], solicitudes[1]
	if primera.Contexto.Proyeccion().AutorizacionRef != segunda.Contexto.Proyeccion().AutorizacionRef ||
		!tokensReservaBaremacionCoinciden(primera.Token, segunda.Token) || primera.Clase != segunda.Clase ||
		primera.BaremacionMeritoRef != segunda.BaremacionMeritoRef {
		t.Fatal("el reintento amplio o sustituyo la capacidad de abandono")
	}
}

func TestFinalizarFirmaNuncaAbandonaTrasInvocarConfirmacion(t *testing.T) {
	entorno := nuevoEntornoBaremacionPrueba(t)
	preparada := prepararFirmaBaremacionPrueba(t, entorno)
	errorRespuesta := errors.New("respuesta perdida despues del efecto simulado")
	entorno.repositorio.errorConfirmar = errorRespuesta

	_, err := entorno.servicio.FinalizarFirma(
		context.Background(), ordenFinalizarBaremacionPrueba(preparada, "confirmacion-ambigua"),
	)
	if errors.Is(err, errorRespuesta) {
		t.Fatalf("se propago una causa tecnica posterior al efecto: %v", err)
	}
	if !errors.Is(err, puertosbolsa.ErrResultadoTransaccionalBaremacionIndeterminado) ||
		!errors.Is(err, puertosbolsa.ErrReconciliacionTransaccionalBaremacionRequerida) ||
		errors.Is(err, puertosbolsa.ErrTransaccionBaremacionNoAplicada) {
		t.Fatalf("respuesta perdida sin clasificacion fail-closed: %v", err)
	}
	if entorno.repositorio.reservas != 1 || entorno.repositorio.confirmaciones != 1 ||
		entorno.repositorio.intentosAbandono != 0 || entorno.repositorio.abandonos != 0 ||
		entorno.repositorio.version.Referencia.Numero != 2 {
		t.Fatalf("se intento revertir un COMMIT potencial: version=%d reservas=%d confirmaciones=%d abandonos=%d",
			entorno.repositorio.version.Referencia.Numero, entorno.repositorio.reservas,
			entorno.repositorio.confirmaciones, entorno.repositorio.abandonos)
	}
}
