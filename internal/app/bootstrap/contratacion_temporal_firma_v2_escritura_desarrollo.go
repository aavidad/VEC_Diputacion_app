package bootstrap

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
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

// rolFirmaExternaRegistroCTDesarrollo es el rol que exigen por nombre las
// guardas de la vía externa (AD156/AD158/AD162/AD170, CT170/CT172 y, por la
// interior, AD177): version_rol_ref = rol:firma_externa_registro_ct_desarrollo:v1.
// Cambiarlo deniega todo registro de firma externa.
const rolFirmaExternaRegistroCTDesarrollo = "firma_externa_registro_ct_desarrollo"

// nuevaInstantaneaFirmaExternaV2CTDesarrollo define una sola versión de rol
// para el registrador RRHH. Su publicación y asignación tienen que ser
// aprobadas y cotejadas con la fuente central; esta plantilla no concede
// permisos por sí misma. Las operaciones de Documentos pertenecen al mismo
// actor y perfil que consulta y registra, con motivos y predicados distintos.
func nuevaInstantaneaFirmaExternaV2CTDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principalID, perfilRef, ahora,
		rolFirmaExternaRegistroCTDesarrollo, "Registro de firmas externas V2 de desarrollo",
		"firma-externa-v2-registrador-ct-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			{Accion: ports.AccionRegistrarFirmaExterna, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoFirmaExterna,
				Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh},
			{Accion: ports.AccionConsultarFirmasR5V2, ModuloID: ports.ModuloContratacion, TipoRecurso: ports.TipoRecursoConsultaFirmasR5,
				Finalidades: []string{ports.FinalidadFirmaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: ports.CamposConsultaFirmasR5V2()},
			{Accion: docports.AccionDescargar, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: tipoRecursoDescargaOriginalCT,
				Finalidades: []string{finalidadDescargaDocumento}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: []string{"contenido", "documento"}},
			{Accion: docports.AccionReservarOriginalFirmable, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: tipoRecursoOriginalFirmableCT,
				Finalidades: []string{docports.FinalidadOriginalFirmable}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: []string{"intento", "reserva"}},
			{Accion: docports.AccionConfirmarOriginalFirmable, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: tipoRecursoOriginalFirmableCT,
				Finalidades: []string{docports.FinalidadOriginalFirmable}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: []string{"documento", "recibo"}},
			{Accion: puertosvec.AccionNegocioEscribirOriginalFirmable, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: tipoRecursoOriginalFirmableCT,
				Finalidades: []string{docports.FinalidadOriginalFirmable}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: []string{"evidencia_almacen", "original_firmable.contenido"}},
			{Accion: docports.AccionCustodiarFirmado, ModuloID: moduloRecursoDocumentosCT, TipoRecurso: "documento_firmado",
				Finalidades: []string{docports.FinalidadCustodiarFirmado}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
				CamposPermitidos: []string{"documento_firmado.custodia", "evidencia_custodia"}},
		},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
}
