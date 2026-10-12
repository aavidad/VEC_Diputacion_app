package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	informe "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	ficheros "vec-diputacion-granada/internal/vec/adapters/almacen/ficheros"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	pdf "vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	doc "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestOriginalR5MontaConServicioDocumentosV1SinCambiarHistoria(t *testing.T) {
	e := nuevoEscenarioOriginalPDP(t)
	v1, err := conservacion.NuevoCatalogoProvisional(e.soporte.reloj)
	if err != nil {
		t.Fatal(err)
	}
	repo := &docpg.Repositorio{}
	objetos := &ficheros.Almacen{}
	servicio := &docapp.Servicio{Repositorio: repo, Almacen: objetos, Politicas: v1, Reloj: e.soporte.reloj}
	documentos := &autoridadDocumentosDesarrollo{servicioOriginal: servicio}
	clave := [32]byte{1}
	seudonimos := nuevoSeudonimizadorAlmacenDesarrollo(clave)
	defer seudonimos.borrar()
	consultas := dependenciasConsultasRRHHDesarrollo{detalle: &ctapp.ServicioConsultaDetalleRRHH{}, originalPropuesta: &ctapp.ServicioConsultaDetalleRRHH{}}
	render := informe.RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}}
	montaje, err := componerRutaOriginalFirmableCTDesarrollo(e.pdp.alta, consultas, render, documentos, seudonimos, e.soporte.reloj)
	if err != nil || len(montaje.rutas) != 1 || montaje.rutas[0].Ruta != httpinterno.RutaOriginalFirmableCT || montaje.rutas[0].Manejador == nil {
		t.Fatalf("ruta original sin servicio compartido: rutas=%d error=%v", len(montaje.rutas), err)
	}
	if montaje.servicio == nil || montaje.tipos == nil || montaje.documentos == nil || montaje.documentos == servicio ||
		montaje.documentos.Politicas == v1 || montaje.documentos.Repositorio != repo || montaje.documentos.Almacen != objetos {
		t.Fatal("original sin su copia V2 o sin recursos Documentos compartidos")
	}
	if servicio.Politicas != v1 || servicio.Repositorio != repo || servicio.Almacen != objetos {
		t.Fatal("el montaje reescribió los recursos o la política V1 de Documentos")
	}
	if _, err := v1.TipoDocumentalRef("contratacion_temporal.borrador.resolucion.v1"); !errors.Is(err, conservacion.ErrTipoNoCatalogado) {
		t.Fatalf("el catálogo histórico adquirió un tipo reservado V2: %v", err)
	}
	if rutas, err := nuevaRutaOriginalFirmableCTDesarrollo(e.pdp.alta, consultas, render, documentos, seudonimos, e.soporte.reloj); err == nil || len(rutas) != 0 {
		t.Fatal("se publicó ruta sin gobierno PostgreSQL para el motivo Original")
	}
}

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
