package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type catalogoEtiquetasRRHHPrueba struct{ err error }

func (c catalogoEtiquetasRRHHPrueba) ObtenerCatalogo(context.Context, string, int) (vecdomain.CatalogoConfigurable, error) {
	return vecdomain.CatalogoConfigurable{Revision: 1, Entradas: []vecdomain.EntradaCatalogoConfigurable{
		{Clave: "centro-520", Etiqueta: "TRANSFORMACIÓN DIGITAL", Atributos: map[string]string{"tipo": "centro"}},
		{Clave: "rpt-520-735", Etiqueta: "Jefatura de Servicio", Atributos: map[string]string{"tipo": "puesto_responsabilidad", "adscripcion_clave": "centro-520"}},
		{Clave: "rpt-520-001", Etiqueta: "Subdirección de Informática", Atributos: map[string]string{"tipo": "puesto_responsabilidad", "adscripcion_clave": "centro-520"}},
		{Clave: "centro-999", Etiqueta: "OTRO CENTRO", Atributos: map[string]string{"tipo": "centro"}},
	}}, c.err
}

func (catalogoEtiquetasRRHHPrueba) ListarVersionesCatalogo(context.Context, string) ([]vecdomain.CatalogoConfigurable, error) {
	return nil, nil
}

func TestEtiquetasBandejaRRHHNombraLasReferenciasDeLaPeticion(t *testing.T) {
	persona := func(id, nombre, actor, puesto string) *identidadPeticionCentroDesarrollo {
		return &identidadPeticionCentroDesarrollo{principal: vecdomain.Principal{ID: id, DisplayName: nombre},
			actor: domain.ActorPeticionCentro{ActorRef: actor, CentroRef: "centro-520", PuestoRef: puesto}}
	}
	proveedor := &proveedorPeticionCentroDesarrollo{catalogos: catalogoEtiquetasRRHHPrueba{}, actores: map[string]*identidadPeticionCentroDesarrollo{
		"sol": persona("sol", "Lucía Fernández Castillo", "per_sol", "rpt-520-735"),
		"rat": persona("rat", "Manuel García Robles", "per_rat", "rpt-520-001"),
		"aje": persona("aje", "Persona ajena", "per_aje", "rpt-520-001"),
	}}
	catalogo := &catalogosAltaContratacionTemporalDesarrollo{
		Categorias: []categoriaCatalogosAltaContratacionTemporalDesarrollo{{Referencia: "categoria:rpt:analista-programador", Etiqueta: "ANALISTA-PROGRAMADOR"}},
		Motivos:    []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{Clave: "acumulacion_tareas", Etiqueta: "Acumulación de tareas"}},
		Documentos: []opcionReferenciaCatalogosAltaContratacionTemporalDesarrollo{{Referencia: "doc:memoria", Etiqueta: "Memoria justificativa"}},
	}
	fila := ports.EntregaPeticionCentro{Peticion: domain.DatosPeticionCentro{
		Configuracion: domain.ConfiguracionPeticionCentro{Solicitante: domain.ActorPeticionCentro{ActorRef: "per_sol"}, Ratificador: domain.ActorPeticionCentro{ActorRef: "per_rat"}},
		Solicitud: domain.SolicitudCentro{CentroRef: "centro-520", ContactoRef: contactoAltaContratacionTemporalDesarrollo,
			CategoriaRef: "categoria:rpt:analista-programador", MotivoClave: "acumulacion_tareas", DocumentosAdjuntos: []string{"doc:memoria"}},
	}}
	e := &etiquetadorPeticionesRRHHDesarrollo{proveedor: proveedor, catalogo: catalogo}
	got, err := e.etiquetas(context.Background(), []ports.EntregaPeticionCentro{fila})
	if err != nil {
		t.Fatal(err)
	}
	if got.Centros["centro-520"] != "TRANSFORMACIÓN DIGITAL" || got.Categorias["categoria:rpt:analista-programador"] != "ANALISTA-PROGRAMADOR" ||
		got.Motivos["acumulacion_tareas"] != "Acumulación de tareas" || got.Documentos["doc:memoria"] != "Memoria justificativa" ||
		got.Contactos[contactoAltaContratacionTemporalDesarrollo] != etiquetaContactoAltaContratacionTemporalDesarrollo {
		t.Fatalf("etiquetas: %+v", got)
	}
	if got.Intervinientes["per_sol"] != (intervinienteRRHHDesarrollo{Nombre: "Lucía Fernández Castillo", Cargo: "Jefatura de Servicio", PuestoRef: "rpt-520-735"}) ||
		got.Intervinientes["per_rat"].Nombre != "Manuel García Robles" || got.Intervinientes["per_rat"].Cargo != "Subdirección de Informática" {
		t.Fatalf("intervinientes: %+v", got.Intervinientes)
	}
	// Solo lo que aparece en las peticiones: ni otras personas ni otros centros.
	if _, ok := got.Intervinientes["per_aje"]; ok || len(got.Centros) != 1 || len(got.Intervinientes) != 2 {
		t.Fatalf("nombra más de lo listado: %+v", got)
	}

	if _, err := (&etiquetadorPeticionesRRHHDesarrollo{proveedor: &proveedorPeticionCentroDesarrollo{catalogos: catalogoEtiquetasRRHHPrueba{err: errors.New("caído")}}}).
		etiquetas(context.Background(), nil); err == nil {
		t.Fatal("un catálogo caído no debe dar etiquetas")
	}
	if _, err := (*etiquetadorPeticionesRRHHDesarrollo)(nil).etiquetas(context.Background(), nil); err == nil {
		t.Fatal("sin etiquetador no hay etiquetas")
	}
}
