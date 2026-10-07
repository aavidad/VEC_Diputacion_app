package ports

// La escritura del original firmable tiene concesión propia de almacén.
// Reservar o confirmar en Documentos no concede acceso al objeto.
const AccionNegocioEscribirOriginalFirmable = "documentos.original_firmable.almacen.escribir"

const PasoAlmacenEscribirOriginalFirmable PasoOperacionAlmacen = "01_escribir_original_firmable"

func especificacionEscribirOriginalFirmable() especificacionAutorizacionAlmacen {
	return especificacionAutorizacionAlmacen{
		accionNegocio: AccionNegocioEscribirOriginalFirmable,
		camposExactos: []string{"original_firmable.contenido", "evidencia_almacen"},
		pasos: []pasoPlanOperacionAlmacen{{
			referencia: PasoAlmacenEscribirOriginalFirmable, accion: AccionAlmacenEscribir,
		}},
	}
}
