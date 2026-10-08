package internagobierno

import (
	"errors"
	"testing"

	core "vec-diputacion-granada/internal/vec/domain"
)

func TestFuenteF1PeticionYContextoUnaResolucionPorPeticion(t *testing.T) {
	e := nuevoEntornoResolucionF1Prueba(t, personaF1Prueba, cuentaF1Prueba)
	consultasAntes := e.registro.consultas
	var primeraCorrelacion string
	for n := 1; n <= 2; n++ {
		p, c, err := e.fuente.ResolverPeticionYContexto(e.ctx)
		if err != nil || c.Resultado.Validar() != nil || c.Vinculo.ValidarPara(c.Resultado) != nil {
			t.Fatalf("peticion %d sin vinculacion F1: %v", n, err)
		}
		datos, err := c.Vinculo.Datos()
		if err != nil || datos.AutenticacionRef != p.Autenticacion.AutenticacionRef ||
			datos.SesionRef != p.Autenticacion.SesionRef || datos.PrincipalID != p.PreparacionCT.ActorRef ||
			datos.CuentaRef != p.Contexto.Cuenta.CuentaRef || datos.PerfilActivoRef != p.Contexto.PerfilActivoRef {
			t.Fatalf("peticion %d mezclo vinculo y seleccion nominal: %v", n, err)
		}
		if e.revalidador.llamadas != n || e.resolutor.llamadas != n || len(e.corporativo.solicitudes) != n ||
			e.registro.consultas-consultasAntes != 2*n {
			t.Fatalf("peticion %d duplico resolucion: autenticacion=%d F1=%d corporativo=%d sesion=%d", n,
				e.revalidador.llamadas, e.resolutor.llamadas, len(e.corporativo.solicitudes),
				e.registro.consultas-consultasAntes)
		}
		if n == 1 {
			primeraCorrelacion = p.PreparacionCT.CorrelacionRef
		} else if p.PreparacionCT.CorrelacionRef == primeraCorrelacion {
			t.Fatal("dos peticiones reutilizaron la correlacion")
		}
	}
}

func TestFuenteF1PeticionYContextoDeniegaCorporativoRetirado(t *testing.T) {
	e := nuevoEntornoResolucionF1Prueba(t, personaF1Prueba, cuentaF1Prueba)
	e.corporativo.vigente = false
	p, c, err := e.fuente.ResolverPeticionYContexto(e.ctx)
	if !errors.Is(err, ErrGobiernoInternoNoDisponible) || p.Autenticacion != (core.SolicitudRevalidacionAutenticacionActorV1{}) ||
		c.Resultado.Validar() == nil || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 ||
		len(e.corporativo.solicitudes) != 1 {
		t.Fatalf("corporativo retirado admitido o con resolucion extra: %v", err)
	}
}
