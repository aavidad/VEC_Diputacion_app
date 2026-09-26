package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestServicioCargaDocumentalRespuestaRemotaAmbiguaRecuperaLaMismaIntencion(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	errorRemoto := errors.New("detalle interno sensible del almacen")
	e.almacen.errorConfirmacion = errorRemoto
	orden := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	if _, err = e.servicio.Confirmar(context.Background(), orden); !errors.Is(err, ports.ErrConfirmacionCargaDirectaNoDisponible) ||
		!errors.Is(err, ports.ErrConfirmacionCargaDocumentalPendiente) {
		t.Fatalf("primer intento: %v", err)
	}
	if errors.Is(err, errorRemoto) || strings.Contains(err.Error(), errorRemoto.Error()) {
		t.Fatalf("se filtro el error remoto: %v", err)
	}
	if e.seguridad.siguienteConsumo != 1 || e.almacen.intentosConfirmar != 1 ||
		e.almacen.confirmaciones != 0 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatalf("la respuesta ambigua no quedo pendiente: consumos=%d intentos=%d confirmaciones=%d estado=%s",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar, e.almacen.confirmaciones,
			e.repositorio.carga.Estado)
	}
	e.almacen.errorConfirmacion = nil
	recuperada, err := e.servicio.Confirmar(context.Background(), orden)
	if err != nil || recuperada.Estado != domain.EstadoCargaDocumentalCuarentena {
		t.Fatalf("la intencion pendiente no se recupero: %v", err)
	}
	if e.seguridad.siguienteConsumo != 1 || e.almacen.intentosConfirmar != 2 || e.almacen.confirmaciones != 1 {
		t.Fatalf("el reintento no confirmo la misma intencion: consumos=%d intentos=%d confirmaciones=%d",
			e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar, e.almacen.confirmaciones)
	}
}

func TestServicioCargaDocumentalRecuperaFalloEntreAlmacenYCommit(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	e.repositorio.fallarConfirmacionUnaVez = true
	orden := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"),
		Recurso: e.recurso, CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion,
		Recibo: preparada.Recibo, Finalidad: preparada.Carga.Finalidad,
		Motivo: "Confirmar aportacion", CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	if _, err := e.servicio.Confirmar(context.Background(), orden); !errors.Is(err, ports.ErrConfirmacionCargaDocumentalPendiente) {
		t.Fatalf("fallo entre pasos no quedo pendiente: %v", err)
	}
	if e.almacen.confirmaciones != 1 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalPreparada {
		t.Fatal("el primer intento no dejo el objeto en cuarentena pendiente de asociar")
	}
	objetoOriginal := e.almacen.objetoCuarentena.Objeto
	recuperada, err := e.servicio.Confirmar(context.Background(), orden)
	if err != nil || recuperada.Estado != domain.EstadoCargaDocumentalCuarentena ||
		recuperada.ContenidoCuarentena == nil || recuperada.ContenidoCuarentena.Referencia != objetoOriginal.Referencia ||
		recuperada.ContenidoCuarentena.Version != objetoOriginal.Version {
		t.Fatalf("reintento no enlazo el mismo objeto: %v", err)
	}
	if e.seguridad.siguienteConsumo != 1 || e.almacen.confirmaciones != 1 ||
		e.almacen.intentosConfirmar != 2 || len(e.repositorio.confirmaciones) != 2 {
		t.Fatalf("se duplico el consumo, objeto o historia: consumos=%d objetos=%d intentos=%d transiciones=%d",
			e.seguridad.siguienteConsumo, e.almacen.confirmaciones, e.almacen.intentosConfirmar,
			len(e.repositorio.confirmaciones))
	}
}

func TestServicioCargaDocumentalRecuperaRespuestaPerdidaTrasCommit(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	e.repositorio.fallarRespuestaTrasCommit = true
	recuperada, err := e.servicio.Confirmar(context.Background(), OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"),
		Recurso: e.recurso, CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion,
		Recibo: preparada.Recibo, Finalidad: preparada.Carga.Finalidad,
		Motivo: "Confirmar aportacion", CorrelacionRef: preparada.Carga.CorrelacionRef,
	})
	if err != nil || recuperada.Estado != domain.EstadoCargaDocumentalCuarentena ||
		e.almacen.confirmaciones != 1 || len(e.repositorio.confirmaciones) != 2 {
		t.Fatalf("la respuesta perdida no se recupero desde el commit: %v", err)
	}
}

func TestServicioCargaDocumentalConfirmacionConcurrenteConsumeUnaSolaVez(t *testing.T) {
	e := nuevoEntornoServicioCarga(t)
	preparada, err := e.servicio.Preparar(context.Background(), e.ordenPreparar())
	if err != nil {
		t.Fatal(err)
	}
	orden := OrdenConfirmarCargaDocumental{
		Principal: e.externo, PerfilActivo: perfilAutorizacionPrueba("externo:0123456789abcdef"), Recurso: e.recurso,
		CargaID: preparada.Carga.ID, SesionRef: e.almacen.sesion, Recibo: preparada.Recibo,
		Finalidad: preparada.Carga.Finalidad, Motivo: "Confirmar aportacion",
		CorrelacionRef: preparada.Carga.CorrelacionRef,
	}
	type resultadoConfirmacion struct {
		carga domain.CargaDocumental
		err   error
	}
	inicio := make(chan struct{})
	resultados := make(chan resultadoConfirmacion, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-inicio
			carga, errConfirmacion := e.servicio.Confirmar(context.Background(), orden)
			resultados <- resultadoConfirmacion{carga: carga, err: errConfirmacion}
		}()
	}
	close(inicio)
	exitos := 0
	for i := 0; i < 2; i++ {
		resultado := <-resultados
		if resultado.err == nil {
			exitos++
			if resultado.carga.Estado != domain.EstadoCargaDocumentalCuarentena {
				t.Fatalf("estado confirmado=%s", resultado.carga.Estado)
			}
			continue
		}
		if !errors.Is(resultado.err, ports.ErrConfirmacionCargaDocumentalPendiente) &&
			!errors.Is(resultado.err, ports.ErrManifiestoPreparacionNoEncontrado) {
			t.Fatalf("error concurrente inesperado: %v", resultado.err)
		}
	}
	if (exitos != 1 && exitos != 2) || e.seguridad.siguienteConsumo != 1 ||
		e.almacen.confirmaciones != 1 || e.repositorio.carga.Estado != domain.EstadoCargaDocumentalCuarentena {
		t.Fatalf("confirmacion concurrente no fue unica: exitos=%d consumos=%d intentos=%d confirmaciones=%d estado=%s",
			exitos, e.seguridad.siguienteConsumo, e.almacen.intentosConfirmar,
			e.almacen.confirmaciones, e.repositorio.carga.Estado)
	}
}
