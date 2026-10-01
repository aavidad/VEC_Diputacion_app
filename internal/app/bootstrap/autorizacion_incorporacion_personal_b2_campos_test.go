package bootstrap

import (
	"slices"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestIncorporacionB2PerfilesRPTOtorganSoloCamposNominales(t *testing.T) {
	alta, consultas, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	soporte := alta.soporte
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref:  vinculo.PrincipalID,
		PerfilV3Ref:     vinculo.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		UnidadRef:       "unidad:desarrollo:rrhh",
		ActorRef:        vinculo.PrincipalID,
	}
	detalle, err := nuevoPerfilNominalIncorporacion(soporte, refs, claveIncorporacionDetalle, nil, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	perfiles := &perfilesNominalesIncorporacion{soporte: soporte, consultas: consultas, detalle: detalle}
	config := configuracionB2PuraPrueba().PersonalB2
	if err := extenderPerfilesNominalesB2(perfiles, refs, config, soporte.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	if len(perfiles.b2) != len(operacionesIncorporacionB2()) || perfiles.b2[ct.AccionConsultarDetalleRRHH] != detalle {
		t.Fatal("B2 perdió una operación o el perfil propio de detalle")
	}

	lectura := perfiles.b2["vec.catalogos.categorias.consultar_historica"]
	uso := perfiles.b2["vec.catalogos.categorias.consultar_uso"]
	if lectura == nil || uso == nil || lectura == uso || lectura == detalle || uso == detalle {
		t.Fatal("lectura RPT y usos RPT no tienen perfiles separados")
	}
	type contratoRPT struct {
		campos          []string
		tipo, finalidad string
	}
	casos := []struct {
		perfil   *perfilFijoCTDesarrollo
		acciones map[string]contratoRPT
		ambitos  []core.AmbitoPerfil
	}{
		{lectura, map[string]contratoRPT{
			"vec.catalogos.categorias.consultar_historica": {campos: []string{"control_actual", "entrada", "publicacion"}, tipo: "catalogo_configurable", finalidad: "consultar_categorias_rpt"},
		}, []core.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{config.CatalogoRPTID}}, {Clave: "modulo_id", Valores: []string{config.ModuloRPTID}}}},
		{uso, map[string]contratoRPT{
			"vec.catalogos.categorias.consultar_uso": {campos: []string{"uso"}, tipo: "uso_categoria", finalidad: "consultar_categorias_rpt"},
			"vec.catalogos.categorias.reservar_uso":  {campos: []string{"recibo", "uso"}, tipo: "uso_categoria", finalidad: "vincular_categoria_a_operacion"},
			"vec.catalogos.categorias.confirmar_uso": {campos: []string{"recibo", "uso"}, tipo: "uso_categoria", finalidad: "vincular_categoria_a_operacion"},
		}, []core.AmbitoPerfil{{Clave: "catalogo_id", Valores: []string{config.CatalogoRPTID}}, {Clave: "modulo_id", Valores: []string{config.ModuloRPTID}}, {Clave: "consumidor", Valores: []string{"personal"}}}},
	}
	for _, caso := range casos {
		concesiones := caso.perfil.plantilla.VersionRol.Concesiones
		if len(concesiones) != len(caso.acciones) || !slices.EqualFunc(caso.perfil.plantilla.AsignacionPerfil.Ambitos, caso.ambitos, func(a, b core.AmbitoPerfil) bool {
			return a.Clave == b.Clave && slices.Equal(a.Valores, b.Valores)
		}) {
			t.Fatal("perfil RPT amplió acciones o ámbitos")
		}
		for _, concesion := range concesiones {
			contrato, existe := caso.acciones[concesion.Accion]
			if !existe || !slices.Equal(concesion.CamposPermitidos, contrato.campos) || concesion.ModuloID != config.ModuloRPTID ||
				!slices.Equal(concesion.Finalidades, []string{contrato.finalidad}) || concesion.TipoRecurso != contrato.tipo ||
				concesion.GarantiaMinima != core.AuthAssuranceHigh {
				t.Fatalf("concesión RPT sin campos o autoridad exactos: %s", concesion.Accion)
			}
		}
	}
	for accion, perfil := range perfiles.b2 {
		_, enLectura := casos[0].acciones[accion]
		_, enUso := casos[1].acciones[accion]
		if perfil == lectura && !enLectura || perfil == uso && !enUso || enLectura && perfil != lectura || enUso && perfil != uso {
			t.Fatalf("acción %s tomó un perfil RPT ajeno", accion)
		}
	}
}
