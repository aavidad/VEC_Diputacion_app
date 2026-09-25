package bootstrap

import (
	"strings"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// accionesPropiasPortalDesarrollo reúne las acciones propias del candidato
// que se componen con el portal: las cuatro de AD3-84 y la confirmación del
// contacto de AD3-86. Cada una tiene su audiencia y su proveedor de material.
func accionesPropiasPortalDesarrollo() [][2]string {
	return append(puertosbolsa.AccionesPortalCandidato(), puertosbolsa.AccionesContactoPropio()...)
}

// descriptoresMaterialContactoPropioDesarrollo declara la audiencia de
// AD3-86 en el catálogo común de material; solo se publica con el portal.
func descriptoresMaterialContactoPropioDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, 1)
	for _, par := range puertosbolsa.AccionesContactoPropio() {
		nombre := strings.ReplaceAll(strings.TrimPrefix(par[0], "bolsa.participaciones_propias."), "_", "-")
		descriptores = append(descriptores, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        par[1],
			Dominio:          "vec.bolsa.mi-bolsa." + strings.ReplaceAll(nombre, "-", "_") + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:bolsa-mi-bolsa-" + nombre + ":",
			ProveedorNominal: "proveedor-material-bolsa-mi-bolsa-" + nombre,
		})
	}
	return descriptores
}

// concesionesContactoPropioDesarrollo concede la confirmación del contacto
// propio sobre 'mi-bolsa:<candidato>': sin campos ni obligaciones, con la
// misma garantía que la consulta.
func concesionesContactoPropioDesarrollo() []dominiovec.ConcesionRol {
	concesiones := make([]dominiovec.ConcesionRol, 0, 1)
	for _, par := range puertosbolsa.AccionesContactoPropio() {
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: par[0], ModuloID: puertosbolsa.ModuloMiBolsa, TipoRecurso: puertosbolsa.TipoRecursoMiBolsa,
			Finalidades: []string{puertosbolsa.FinalidadPortalCandidato}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		})
	}
	return concesiones
}
