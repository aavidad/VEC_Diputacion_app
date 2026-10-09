package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

// etiquetadorPeticionesRRHHDesarrollo pone nombre a las referencias que guarda
// una petición del centro (centro, contacto, categoría, motivo, documentos y
// personas) para la bandeja de RRHH. La petición solo conserva códigos; sin
// estos nombres RRHH veía «centro-520» o «per_…» y el motivo vacío.
//
// Solo nombra lo que aparece en las peticiones listadas: no es un directorio.
// Los nombres son de presentación; nunca vuelven como identidad ni permiso.
type etiquetadorPeticionesRRHHDesarrollo struct {
	proveedor *proveedorPeticionCentroDesarrollo
	catalogo  *catalogosAltaContratacionTemporalDesarrollo
}

type intervinienteRRHHDesarrollo struct {
	Nombre    string `json:"nombre"`
	Cargo     string `json:"cargo"`
	PuestoRef string `json:"puesto_ref"`
}

type etiquetasPeticionesRRHHDesarrollo struct {
	Centros        map[string]string                      `json:"centros"`
	Contactos      map[string]string                      `json:"contactos"`
	Categorias     map[string]string                      `json:"categorias"`
	Motivos        map[string]string                      `json:"motivos"`
	Documentos     map[string]string                      `json:"documentos"`
	Intervinientes map[string]intervinienteRRHHDesarrollo `json:"intervinientes"`
}

func (e *etiquetadorPeticionesRRHHDesarrollo) etiquetas(ctx context.Context, filas []ports.EntregaPeticionCentro) (etiquetasPeticionesRRHHDesarrollo, error) {
	r := etiquetasPeticionesRRHHDesarrollo{Centros: map[string]string{}, Contactos: map[string]string{}, Categorias: map[string]string{},
		Motivos: map[string]string{}, Documentos: map[string]string{}, Intervinientes: map[string]intervinienteRRHHDesarrollo{}}
	if e == nil || e.proveedor == nil || e.proveedor.catalogos == nil {
		return r, errCatalogosAltaContratacionTemporalDesarrolloNoDisponibles
	}
	organizacion, err := e.proveedor.catalogos.ObtenerCatalogo(ctx, personalports.IDCatalogoOrganizacion, e.proveedor.version)
	if err != nil {
		return r, err
	}
	nombres := make(map[string]string, len(organizacion.Entradas))
	for _, entrada := range organizacion.Entradas {
		nombres[entrada.Clave] = entrada.Etiqueta
	}
	categorias, motivos, documentos := map[string]string{}, map[string]string{}, map[string]string{}
	if e.catalogo != nil {
		for _, c := range e.catalogo.Categorias {
			categorias[c.Referencia] = c.Etiqueta
		}
		for _, m := range e.catalogo.Motivos {
			motivos[m.Clave] = m.Etiqueta
		}
		for _, d := range e.catalogo.Documentos {
			documentos[d.Referencia] = d.Etiqueta
		}
	}
	personas := make(map[string]*identidadPeticionCentroDesarrollo, len(e.proveedor.actores))
	for _, a := range e.proveedor.actores {
		personas[a.actor.ActorRef] = a
	}
	poner := func(destino map[string]string, clave, etiqueta string) {
		if clave != "" && etiqueta != "" {
			destino[clave] = etiqueta
		}
	}
	for _, fila := range filas {
		s := fila.Peticion.Solicitud
		poner(r.Centros, s.CentroRef, nombres[s.CentroRef])
		if s.ContactoRef == contactoAltaContratacionTemporalDesarrollo {
			poner(r.Contactos, s.ContactoRef, etiquetaContactoAltaContratacionTemporalDesarrollo)
		}
		poner(r.Categorias, s.CategoriaRef, categorias[s.CategoriaRef])
		poner(r.Motivos, string(s.MotivoClave), motivos[string(s.MotivoClave)])
		for _, d := range s.DocumentosAdjuntos {
			poner(r.Documentos, d, documentos[d])
		}
		for _, actor := range []string{fila.Peticion.Configuracion.Solicitante.ActorRef, fila.Peticion.Configuracion.Ratificador.ActorRef} {
			if p, ok := personas[actor]; ok && p.principal.DisplayName != "" {
				r.Intervinientes[actor] = intervinienteRRHHDesarrollo{Nombre: p.principal.DisplayName, Cargo: nombres[p.actor.PuestoRef], PuestoRef: p.actor.PuestoRef}
			}
		}
	}
	return r, nil
}
