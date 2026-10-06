package bootstrap

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Firma V2 de escritura (4c-5): material de las dos audiencias de registro y
// perfil fijo de la vía externa.
//
// La vía VEC la registra quien firma, con su asignación de cargo real (AUT53
// ya le concede firma_vec.registrar y la consulta R5): no tiene perfil fijo.
// La vía externa la registra RRHH con este perfil fijo, sólo de organización
// (opción C del 06/10; la unidad de RRHH está en dudas.md, 148). En la misma
// petición RRHH consulta las firmas R5 antes de registrar, así que el perfil
// lleva las dos concesiones.
const clavePerfilFijoFirmaExternaV2CTDesarrollo = "firma_externa_v2"

// descriptoresMaterialFirmaV2EscrituraCTDesarrollo son los consumidores V3 de
// las decisiones interior y exterior de la firma V2 (AD162/AD170/AD177). Cada
// audiencia tiene su dominio y su prefijo de clave de capacidad.
func descriptoresMaterialFirmaV2EscrituraCTDesarrollo() (vec, externa descriptorMaterialConsumidorV3Desarrollo) {
	return descriptorMaterialConsumidorV3Desarrollo{
			Audiencia: ports.AudienciaFirmaVecV2, Dominio: "vec.ct.firma-v2.vec.capacidad-v3",
			Prefijo: "clave:capacidad:ct-firma-v2-vec:", ProveedorNominal: proveedorMaterialContratacionTemporal,
		}, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia: ports.AudienciaFirmaExternaV2, Dominio: "vec.ct.firma-v2.externa.capacidad-v3",
			Prefijo: "clave:capacidad:ct-firma-v2-externa:", ProveedorNominal: proveedorMaterialContratacionTemporal,
		}
}

// nuevaInstantaneaFirmaExternaV2CTDesarrollo: registrar la firma externa
// (interior y exterior del plan usan la misma acción, sin campos ni
// obligaciones, como exigen AD170 y AD177) y consultar las firmas R5 V2 con
// los campos exactos de AD162.
func nuevaInstantaneaFirmaExternaV2CTDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		"firma_externa_v2_registrador_ct_desarrollo", "Registro de firmas externas V2 de desarrollo",
		"firma-externa-v2-registrador-ct-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			{Accion: ports.AccionRegistrarFirmaExterna, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaExterna,
				Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh},
			{Accion: ports.AccionConsultarFirmasR5V2, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoConsultaFirmasR5,
				Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: ports.CamposConsultaFirmasR5V2()},
		},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}
