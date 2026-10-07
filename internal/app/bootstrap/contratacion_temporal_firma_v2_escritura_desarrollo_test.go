package bootstrap

import (
	"slices"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El perfil fijo de la firma externa concede exactamente registrar la firma
// externa (sin campos ni obligaciones, como AD170/AD177) y consultar las
// firmas R5 V2 con los campos de AD162, en la organización y sin unidad.
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
		tipo   string
		campos []string
	}{
		ports.AccionRegistrarFirmaExterna: {ports.TipoRecursoFirmaExterna, nil},
		ports.AccionConsultarFirmasR5V2:   {ports.TipoRecursoConsultaFirmasR5, ports.CamposConsultaFirmasR5V2()},
	}
	if len(i.VersionRol.Concesiones) != len(esperadas) {
		t.Fatal("concesiones de más o de menos")
	}
	for _, c := range i.VersionRol.Concesiones {
		e, ok := esperadas[c.Accion]
		if !ok || c.TipoRecurso != e.tipo || c.ModuloID != ports.ModuloContratacion || !slices.Equal(c.CamposPermitidos, e.campos) ||
			len(c.Obligaciones) != 0 || c.GarantiaMinima != dominiovec.AuthAssuranceHigh ||
			!slices.Equal(c.Finalidades, []string{ports.FinalidadFirmaDocumento}) {
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
