package bootstrap

import (
	"testing"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestReincorporacionPerfilFijoSoloAdmitePreimagenOperativaPropia(t *testing.T) {
	s, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	i := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea)
	i.AsignacionPerfil.Ambitos = []vecdomain.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
	}
	p := &perfilFijoCTDesarrollo{plantilla: i}
	base := instantaneaPublicadaDesarrollo{instantanea: i, actoAsignacion: actoAsignacionReincorporacionTitularDesarrollo,
		actualizadaPor: i.AsignacionPerfil.EmitidaPor, actoControl: actoControlRolReincorporacionTitularDesarrollo}
	admitida := preimagenPropiaReincorporacionTitular(p)
	for _, caso := range []struct {
		nombre string
		cambio func(*instantaneaPublicadaDesarrollo)
		valida bool
	}{
		{"inicial", func(*instantaneaPublicadaDesarrollo) {}, true},
		{"lectura histórica", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Ambitos = append(p.instantanea.AsignacionPerfil.Ambitos,
				vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:previo"}})
		}, true},
		{"escritura histórica", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Ambitos = append(p.instantanea.AsignacionPerfil.Ambitos,
				vecdomain.AmbitoPerfil{Clave: "expediente_ref", Valores: []string{"expediente:previo"}},
				vecdomain.AmbitoPerfil{Clave: "fase_previa", Valores: []string{"nombramiento"}},
				vecdomain.AmbitoPerfil{Clave: "estado_previo", Valores: []string{"en_curso"}})
		}, true},
		{"acto ajeno", func(p *instantaneaPublicadaDesarrollo) { p.actoAsignacion = "acto:administrativo:restriccion" }, false},
		{"control ajeno", func(p *instantaneaPublicadaDesarrollo) { p.actoControl = "acto:administrativo:control" }, false},
		{"otra organización", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "organizacion:ajena"
		}, false},
		{"revocada", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
			p.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
		}, false},
		{"rol retirado", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
		}, false},
		{"concesión retirada", func(p *instantaneaPublicadaDesarrollo) { p.instantanea.VersionRol.Concesiones = nil }, false},
		{"dimensión ajena", func(p *instantaneaPublicadaDesarrollo) {
			p.instantanea.AsignacionPerfil.Ambitos = append(p.instantanea.AsignacionPerfil.Ambitos,
				vecdomain.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:otra"}})
		}, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			publicada := base
			publicada.instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(base.instantanea)
			caso.cambio(&publicada)
			if admitida(publicada, s.reloj.Ahora()) != caso.valida {
				t.Fatalf("preimagen %s: admisión inesperada", caso.nombre)
			}
		})
	}
}
