package bootstrap

import (
	"testing"
)

func TestIncorporacionV2ReferenciasOriginales(t *testing.T) {
	r := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref: "per_956e44e7abde00cefad2999ecff7e0fe", PerfilV3Ref: "prf_259f07adc2bfd87bc2fa4e6e5bd9a790",
		OrganizacionRef: "organizacion:desarrollo:dipgra", UnidadRef: "unidad:desarrollo:rrhh", ActorRef: "per_956e44e7abde00cefad2999ecff7e0fe",
	}
	if !r.valida() {
		t.Fatal("rechaza las referencias conservadas del expediente")
	}
	for _, modificar := range []func(*ReferenciasCTIncorporacionDesarrollo){
		func(r *ReferenciasCTIncorporacionDesarrollo) { r.UnidadRef = "unidad:" },
		func(r *ReferenciasCTIncorporacionDesarrollo) { r.ActorRef = "actor:rrhh" },
		func(r *ReferenciasCTIncorporacionDesarrollo) { r.OrganizacionRef = "unidad:rrhh" },
	} {
		c := r
		modificar(&c)
		if c.valida() {
			t.Fatal("acepta otra gramática de referencias")
		}
	}
}
