package bootstrap

import (
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Una única versión de rol debe cubrir las operaciones anidadas de Documentos,
// la consulta y el registro; la fuente publicada sigue siendo la autoridad.
func TestFirmaExternaV2InstantaneaConcedeLoQueExigeElNucleo(t *testing.T) {
	i, err := nuevaInstantaneaFirmaExternaV2CTDesarrollo("per_firma_externa_prueba", "prf_firma_externa_prueba", time.Now().UTC().Truncate(time.Microsecond))
	if err != nil || i.Validar() != nil {
		t.Fatal(err)
	}
	// Las guardas SQL de la vía externa exigen este rol por su referencia.
	if i.AsignacionPerfil.VersionRolRef != "rol:firma_externa_registro_ct_desarrollo:v1" || i.VersionRol.Referencia() != i.AsignacionPerfil.VersionRolRef {
		t.Fatalf("rol distinto del que exige el núcleo: %q", i.AsignacionPerfil.VersionRolRef)
	}
	esperadas := map[string]struct {
		modulo    string
		tipo      string
		finalidad string
		campos    []string
	}{
		ports.AccionRegistrarFirmaExterna:                {ports.ModuloContratacion, ports.TipoRecursoFirmaExterna, ports.FinalidadFirmaDocumento, nil},
		ports.AccionConsultarFirmasR5V2:                  {ports.ModuloContratacion, ports.TipoRecursoConsultaFirmasR5, ports.FinalidadFirmaDocumento, ports.CamposConsultaFirmasR5V2()},
		docports.AccionDescargar:                         {moduloRecursoDocumentosCT, tipoRecursoDescargaOriginalCT, finalidadDescargaDocumento, []string{"contenido", "documento"}},
		docports.AccionReservarOriginalFirmable:          {moduloRecursoDocumentosCT, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable, []string{"intento", "reserva"}},
		docports.AccionConfirmarOriginalFirmable:         {moduloRecursoDocumentosCT, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable, []string{"documento", "recibo"}},
		puertosvec.AccionNegocioEscribirOriginalFirmable: {moduloRecursoDocumentosCT, tipoRecursoOriginalFirmableCT, docports.FinalidadOriginalFirmable, []string{"evidencia_almacen", "original_firmable.contenido"}},
		docports.AccionCustodiarFirmado:                  {moduloRecursoDocumentosCT, "documento_firmado", docports.FinalidadCustodiarFirmado, []string{"documento_firmado.custodia", "evidencia_custodia"}},
	}
	if len(i.VersionRol.Concesiones) != len(esperadas) {
		t.Fatal("concesiones de más o de menos")
	}
	for _, c := range i.VersionRol.Concesiones {
		e, ok := esperadas[c.Accion]
		if !ok || c.TipoRecurso != e.tipo || c.ModuloID != e.modulo || !slices.Equal(c.CamposPermitidos, e.campos) ||
			len(c.Obligaciones) != 0 || c.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
			!slices.Equal(c.Finalidades, []string{e.finalidad}) {
			t.Fatalf("concesión %q distinta de la que exige el núcleo: %+v", c.Accion, c)
		}
	}
	a := i.AsignacionPerfil.Ambitos
	if len(a) != 1 || a[0].Clave != "organizacion_ref" || !slices.Equal(a[0].Valores, []string{organizacionAltaContratacionTemporalDesarrollo}) {
		t.Fatalf("ámbitos distintos de sólo organización: %+v", a)
	}
}

// Las dos audiencias de escritura tienen descriptor propio y conviven en el
// catálogo con las de consulta y recuperación R5 V2 (sin repetir audiencia,
// dominio ni prefijo).
func TestFirmaV2EscrituraDescriptoresPropios(t *testing.T) {
	vec, externa := descriptoresMaterialFirmaV2EscrituraCTDesarrollo()
	consulta, recuperacion := descriptoresMaterialFirmasR5V2CTDesarrollo()
	if vec.Audiencia != ports.AudienciaFirmaVecV2 || externa.Audiencia != ports.AudienciaFirmaExternaV2 {
		t.Fatal("audiencias distintas de las de AD162")
	}
	c, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{vec, externa, consulta, recuperacion})
	if err != nil {
		t.Fatalf("descriptores repetidos: %v", err)
	}
	for _, d := range []descriptorMaterialConsumidorV3Desarrollo{vec, externa} {
		if got, ok := c.descriptorPara(d.Audiencia); !ok || got != d {
			t.Fatalf("descriptor de %s no encontrado", d.Audiencia)
		}
	}
}
