package seguridad

import (
	"context"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAutoridadSellosAnotacionAdministrativaSellaDominiosSeparados(t *testing.T) {
	a1, _ := configuracionAsignacionPrueba(t, ports.DominioAmbitoIdempotenciaAnotacionAdministrativa, 2)
	a0, _ := configuracionAsignacionPrueba(t, ports.DominioAmbitoIdempotenciaAnotacionAdministrativa, 1)
	h1, _ := configuracionAsignacionPrueba(t, ports.DominioHuellaPeticionAnotacionAdministrativa, 2)
	h0, _ := configuracionAsignacionPrueba(t, ports.DominioHuellaPeticionAnotacionAdministrativa, 1)
	a, err := NuevaAutoridadSellosAnotacionAdministrativaHMAC(a1, []ConfiguracionSelladorHMAC{a0}, h1, []ConfiguracionSelladorHMAC{h0})
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: "12345678-1234-4abc-8def-1234567890ab", OrganizacionRef: "organizacion:dipgra:anotacion-001", ActorRef: "persona:tecnica:anotacion-001", PerfilRef: "perfil:tecnico:anotacion-001"}
	amb, err := a.SellarAmbitoAnotacionAdministrativa(context.Background(), s)
	if err != nil || amb.ValidarDominio(ports.DominioAmbitoIdempotenciaAnotacionAdministrativa) != nil {
		t.Fatalf("ambito: %v", err)
	}
	m := ports.MaterialAnotacionAdministrativa{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: "expediente:contratacion:anotacion-001", SolicitudPersonalRef: "solicitud:personal:anotacion-001", VersionEsperada: 7, ClaveIdempotencia: s.ClaveIdempotencia, ActorRef: s.ActorRef, PerfilRef: s.PerfilRef, Observaciones: "Anotacion sintetica."}
	huella, err := a.DerivarHuellaAnotacionAdministrativa(context.Background(), m)
	if err != nil || huella.ValidarDominio(ports.DominioHuellaPeticionAnotacionAdministrativa) != nil {
		t.Fatalf("huella: %v", err)
	}
	if amb == huella {
		t.Fatal("dominios indistinguibles")
	}
}
func TestAutoridadSellosAnotacionAdministrativaRechazaGeneracionesNoAlineadas(t *testing.T) {
	a, _ := configuracionAsignacionPrueba(t, ports.DominioAmbitoIdempotenciaAnotacionAdministrativa, 2)
	h, _ := configuracionAsignacionPrueba(t, ports.DominioHuellaPeticionAnotacionAdministrativa, 1)
	if x, err := NuevaAutoridadSellosAnotacionAdministrativaHMAC(a, nil, h, nil); x != nil || err == nil {
		t.Fatal("generaciones cruzadas aceptadas")
	}
}

func TestAutoridadSellosAnotacionAdministrativaCancelaTrasUltimoConector(t *testing.T) {
	for _, caso := range []struct {
		name   string
		huella bool
	}{{"ambito", false}, {"huella", true}} {
		t.Run(caso.name, func(t *testing.T) {
			ambito, sa := configuracionAsignacionPrueba(t, ports.DominioAmbitoIdempotenciaAnotacionAdministrativa, 1)
			huella, sh := configuracionAsignacionPrueba(t, ports.DominioHuellaPeticionAnotacionAdministrativa, 1)
			a, err := NuevaAutoridadSellosAnotacionAdministrativaHMAC(ambito, nil, huella, nil)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			if caso.huella {
				sh.cancelarTrasSellar = cancelar
			} else {
				sa.cancelarTrasSellar = cancelar
			}
			if caso.huella {
				m := ports.MaterialAnotacionAdministrativa{OrganizacionRef: "organizacion:dipgra:anotacion-001", ExpedienteRef: "expediente:contratacion:anotacion-001", SolicitudPersonalRef: "solicitud:personal:anotacion-001", VersionEsperada: 1, ClaveIdempotencia: "12345678-1234-4abc-8def-1234567890ab", ActorRef: "persona:tecnica:anotacion-001", PerfilRef: "perfil:tecnico:anotacion-001", Observaciones: "Anotacion sintetica."}
				if _, err := a.DerivarHuellaAnotacionAdministrativa(ctx, m); err != context.Canceled {
					t.Fatalf("error=%v", err)
				}
			} else {
				s := ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: "12345678-1234-4abc-8def-1234567890ab", OrganizacionRef: "organizacion:dipgra:anotacion-001", ActorRef: "persona:tecnica:anotacion-001", PerfilRef: "perfil:tecnico:anotacion-001"}
				if _, err := a.SellarAmbitoAnotacionAdministrativa(ctx, s); err != context.Canceled {
					t.Fatalf("error=%v", err)
				}
			}
		})
	}
}
