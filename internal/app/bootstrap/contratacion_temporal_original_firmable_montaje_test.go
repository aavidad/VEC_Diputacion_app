package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	doc "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestOriginalR5SeleccionaPerfilesPorAccionSinCruce(t *testing.T) {
	e := nuevoEscenarioOriginalPDP(t)
	ctx, recurso := e.descarga(t, httpinterno.RutaOriginalFirmableCT)
	pedida, err := e.pdp.solicitarOriginalV3(ctx, doc.AccionDescargar, finalidadDescargaDocumento, recurso)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := pedida.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	conDoc := context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	if got := e.soporte.perfilFijoParaContexto(conDoc, httpinterno.RutaOriginalFirmableCT); got != e.soporte.perfilesOriginal.documentos {
		t.Fatal("la descarga no eligió la asignación Documentos")
	}
	datos.Accion = ct.AccionConsultarDetalleRRHH
	datos.Finalidad = ct.FinalidadConsultarDetalleRRHH
	datos.Recurso = core.RecursoAutorizable{ModuloID: ct.ModuloContratacion}
	conCT := context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	if got := e.soporte.perfilFijoParaContexto(conCT, httpinterno.RutaOriginalFirmableCT); got != e.soporte.perfilesOriginal.detalle {
		t.Fatal("la consulta no eligió la asignación CT")
	}
	datos.Accion = doc.AccionDescargar
	if got := e.soporte.perfilFijoParaContexto(context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos), httpinterno.RutaOriginalFirmableCT); got != nil {
		t.Fatal("la acción Documentos con recurso CT cruzó perfiles")
	}
	if e.soporte.perfilesOriginal.detalle.perfilRef() == e.soporte.perfilesOriginal.documentos.perfilRef() ||
		e.soporte.perfilesOriginal.detalle.contexto.Resultado.Contexto.PersonaRef != e.soporte.perfilesOriginal.documentos.contexto.Resultado.Contexto.PersonaRef ||
		len(e.soporte.perfilesOriginal.detalle.plantilla.AsignacionPerfil.Ambitos) != 3 ||
		len(e.soporte.perfilesOriginal.documentos.plantilla.AsignacionPerfil.Ambitos) != 1 {
		t.Fatal("perfiles nominales sin separación de ámbito y persona común")
	}
}

func TestOriginalR5RevocacionDelLectorNoAutorizaConsulta(t *testing.T) {
	e := nuevoEscenarioOriginalPDP(t)
	ctx, _ := e.descarga(t, httpinterno.RutaOriginalFirmableCT)
	lector := e.soporte.perfilesOriginal.detalle
	a := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	publicada := a.asignaciones[lector.perfilRef()]
	publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	ahora := e.soporte.reloj.Ahora()
	publicada.instantanea.AsignacionPerfil.RevocadaEn = ahora
	publicada.instantanea.AsignacionPerfil.RevocadaPor = "revocador:prueba"
	publicada.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	a.asignaciones[lector.perfilRef()] = publicada
	if _, err := leerAsignacionConsumiblePerfilNominalCT(ctx, e.soporte, lector); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("lector revocado quedó consumible: %v", err)
	}
}
