package incorporacionejercicio

import (
	"context"
	"errors"
	"strings"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type contratoNominalPrueba func(context.Context, string, string) (ContratoPlanNominal, error)

func (f contratoNominalPrueba) LeerContratoPlanNominal(ctx context.Context, org, exp string) (ContratoPlanNominal, error) {
	return f(ctx, org, exp)
}

type intencionNominalPrueba func(context.Context, ContratoPlanNominal) (IntencionCTDurable, error)

func (f intencionNominalPrueba) LeerIntencionCTDurable(ctx context.Context, c ContratoPlanNominal) (IntencionCTDurable, error) {
	return f(ctx, c)
}

type ct124NominalPrueba func(context.Context, ContratoPlanNominal) (AntecedenteCT124, error)

func (f ct124NominalPrueba) LeerAntecedenteCT124(ctx context.Context, c ContratoPlanNominal) (AntecedenteCT124, error) {
	return f(ctx, c)
}

type bolsaNominalPrueba func(context.Context, ContratoPlanNominal) (PersonaSeleccionadaBolsa, error)

func (f bolsaNominalPrueba) LeerPersonaSeleccionada(ctx context.Context, c ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
	return f(ctx, c)
}

type rptNominalPrueba func(context.Context, ContratoPlanNominal) (PuestoRPTNominal, error)

func (f rptNominalPrueba) LeerPuestoRPT(ctx context.Context, c ContratoPlanNominal) (PuestoRPTNominal, error) {
	return f(ctx, c)
}

type personalNominalPrueba func(context.Context, SolicitudReservaPersonalB2) (ReservaPersonalB2, error)

func (f personalNominalPrueba) ReservarPersonalB2(ctx context.Context, s SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
	return f(ctx, s)
}

func contratoB2Prueba() ContratoPlanNominal {
	return ContratoPlanNominal{
		Protocolo: ProtocoloPersonalB2V1, OrganizacionRef: "organizacion:uno", ExpedienteRef: "expediente:uno",
		VersionExpediente: 7, ContratoVersion: 1, ContratoReciboRef: "recibo:contrato", ContratoSHA256: strings.Repeat("a", 64),
		SolicitudRef: "solicitud:personal", IntencionRef: "intencion:uno", IntencionReciboRef: "recibo:intencion",
		IntencionVersion: 1, AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion",
		LlamamientoRef: "llamamiento:uno", SeleccionRef: "seleccion:uno", VersionSeleccion: 2,
		SeleccionReciboRef: "recibo:bolsa", FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{
			Referencia: "rpt:uno", Version: 1, HuellaSHA256: strings.Repeat("b", 64),
		}, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo",
		PublicacionRPTReciboRef: "recibo:rpt", PuestoRef: "puesto:uno", PlazaRef: "plaza:uno",
		ReservaIdempotente: "11111111-1111-4111-8111-111111111111",
	}
}

func configuracionB2Prueba(c ContratoPlanNominal, reservas *int) ConfiguracionPlanesNominales {
	return ConfiguracionPlanesNominales{
		Contratos: contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) { return c, nil }),
		IntencionCT: intencionNominalPrueba(func(context.Context, ContratoPlanNominal) (IntencionCTDurable, error) {
			return IntencionCTDurable{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				ContratoReciboRef: c.ContratoReciboRef, IntencionRef: c.IntencionRef, ReciboRef: c.IntencionReciboRef,
				Version: c.IntencionVersion, ReservaIdempotente: c.ReservaIdempotente, Confirmada: true}, nil
		}),
		CT124: ct124NominalPrueba(func(context.Context, ContratoPlanNominal) (AntecedenteCT124, error) {
			return AntecedenteCT124{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				AceptacionRef: c.AceptacionRef, LlamamientoRef: c.LlamamientoRef, VersionExpediente: c.VersionExpediente,
				ReciboRef: "recibo:aceptacion"}, nil
		}),
		Bolsa: bolsaNominalPrueba(func(context.Context, ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
			return PersonaSeleccionadaBolsa{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				AceptacionRef: c.AceptacionRef, LlamamientoRef: c.LlamamientoRef, SeleccionRef: c.SeleccionRef,
				VersionSeleccion: c.VersionSeleccion, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3, ReciboRef: "recibo:bolsa"}, nil
		}),
		RPT: rptNominalPrueba(func(context.Context, ContratoPlanNominal) (PuestoRPTNominal, error) {
			return PuestoRPTNominal{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				Fuente: c.FuenteRPT, CategoriaRef: c.CategoriaRef, PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef,
				VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo", PublicacionReciboRef: "recibo:rpt", Prospectivo: true}, nil
		}),
		Personal: personalNominalPrueba(func(_ context.Context, s SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
			*reservas++
			return ReservaPersonalB2{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				SolicitudRef: c.SolicitudRef, IdempotenciaRef: c.ReservaIdempotente,
				PersonaRef: s.Persona.PersonaRef, PersonaVersion: s.Persona.PersonaVersion,
				ContratoReciboRef: c.ContratoReciboRef, ContratoSHA256: c.ContratoSHA256,
				FuenteRPT: c.FuenteRPT, PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef,
				ReservaRef: "reserva:uno", ReciboRef: "recibo:personal", VersionReserva: 1, EmpleadoRef: "emp_0123456789abcdefghijkl"}, nil
		}),
	}
}

func TestPlanesNominalesB2ExigeFuentesYNoVuelveAlEjercicio(t *testing.T) {
	c := contratoB2Prueba()
	f, err := NuevosPlanesNominales(ConfiguracionPlanesNominales{
		Contratos: contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) { return c, nil }),
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) || p != (PlanNominal{}) {
		t.Fatalf("B2 sin fuentes debe quedar pendiente, obtenido %+v, %v", p, err)
	}
	c.EjercicioSintetico = true
	f, _ = NuevosPlanesNominales(ConfiguracionPlanesNominales{
		Contratos: contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) { return c, nil }),
	})
	p, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || p != (PlanNominal{}) {
		t.Fatalf("B2 no puede usar fuente sintetica: %+v, %v", p, err)
	}
	c.EjercicioSintetico = false
	c.Protocolo = "desconocido"
	f, _ = NuevosPlanesNominales(ConfiguracionPlanesNominales{
		Contratos: contratoNominalPrueba(func(context.Context, string, string) (ContratoPlanNominal, error) { return c, nil }),
	})
	p, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) || p != (PlanNominal{}) {
		t.Fatalf("protocolo desconocido no puede ir al ejercicio: %+v, %v", p, err)
	}
}

func TestPlanesNominalesB2LigaPersonaPuestoYReservaExactos(t *testing.T) {
	c := contratoB2Prueba()
	reservas := 0
	cfg := configuracionB2Prueba(c, &reservas)
	f, err := NuevosPlanesNominales(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if err != nil || p.Protocolo != ProtocoloPersonalB2V1 || p.Personal == nil || p.Ejercicio != nil || reservas != 1 ||
		p.Personal.Persona.PersonaRef != p.Personal.Reserva.PersonaRef {
		t.Fatalf("plan B2 no ligado: %+v, %v, reservas=%d", p, err, reservas)
	}
	// Un recibo Personal de otro contrato no puede convertirse en plan.
	cfg.Personal = personalNominalPrueba(func(context.Context, SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
		reservas++
		r := p.Personal.Reserva
		r.ContratoReciboRef = "recibo:otro"
		return r, nil
	})
	f, _ = NuevosPlanesNominales(cfg)
	p, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || p != (PlanNominal{}) {
		t.Fatalf("reserva ajena debe rechazarse: %+v, %v", p, err)
	}
	cfg.Personal = personalNominalPrueba(func(context.Context, SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
		r := ReservaPersonalB2{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
			SolicitudRef: c.SolicitudRef, IdempotenciaRef: c.ReservaIdempotente, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
			ContratoReciboRef: c.ContratoReciboRef, ContratoSHA256: c.ContratoSHA256, FuenteRPT: c.FuenteRPT,
			PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef, ReservaRef: "reserva:uno", ReciboRef: "recibo:personal",
			VersionReserva: 1, EmpleadoRef: "emp_0123456789abcdefghijkl", EjercicioSintetico: true}
		return r, nil
	})
	f, _ = NuevosPlanesNominales(cfg)
	p, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || p != (PlanNominal{}) {
		t.Fatalf("reserva sintetica no puede convertirse en B2: %+v, %v", p, err)
	}
}

func TestPlanesNominalesConservaDenegacionSinReservar(t *testing.T) {
	c := contratoB2Prueba()
	reservas := 0
	cfg := configuracionB2Prueba(c, &reservas)
	cfg.Bolsa = bolsaNominalPrueba(func(context.Context, ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
		return PersonaSeleccionadaBolsa{}, ct.ErrDenegadaIncorporacionAplicacion
	})
	f, _ := NuevosPlanesNominales(cfg)
	p, err := f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrDenegadaIncorporacionAplicacion) || p != (PlanNominal{}) || reservas != 0 {
		t.Fatalf("denegacion no debe revelar plan ni reservar: %+v, %v, reservas=%d", p, err, reservas)
	}
}

func TestPlanesNominalesB2ExigeIntencionDurableExactaAntesDeReservar(t *testing.T) {
	c := contratoB2Prueba()
	reservas := 0
	cfg := configuracionB2Prueba(c, &reservas)
	for _, caso := range []struct {
		nombre string
		fuente intencionNominalPrueba
	}{
		{"ausente", func(context.Context, ContratoPlanNominal) (IntencionCTDurable, error) {
			return IntencionCTDurable{}, ct.ErrPreparacionIncorporacionPendiente
		}},
		{"cruzada", func(context.Context, ContratoPlanNominal) (IntencionCTDurable, error) {
			return IntencionCTDurable{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: "expediente:otro",
				ContratoReciboRef: c.ContratoReciboRef, IntencionRef: c.IntencionRef, ReciboRef: c.IntencionReciboRef,
				Version: c.IntencionVersion, ReservaIdempotente: c.ReservaIdempotente, Confirmada: true}, nil
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cfg.IntencionCT = caso.fuente
			f, err := NuevosPlanesNominales(cfg)
			if err != nil {
				t.Fatal(err)
			}
			p, err := f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
			if err == nil || p != (PlanNominal{}) || reservas != 0 {
				t.Fatalf("sin intencion exacta no hay plan ni reserva: %+v, %v, reservas=%d", p, err, reservas)
			}
		})
	}
}

func TestPlanesNominalesB2NoReservaAnteAntecedenteOPersonaIncongruentes(t *testing.T) {
	c := contratoB2Prueba()
	reservas := 0
	cfg := configuracionB2Prueba(c, &reservas)
	cfg.CT124 = ct124NominalPrueba(func(context.Context, ContratoPlanNominal) (AntecedenteCT124, error) {
		return AntecedenteCT124{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
			AceptacionRef: c.AceptacionRef, LlamamientoRef: c.LlamamientoRef,
			VersionExpediente: c.VersionExpediente, ReciboRef: "recibo:aceptacion", NoIncorporacion: true}, nil
	})
	f, _ := NuevosPlanesNominales(cfg)
	_, err := f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || reservas != 0 {
		t.Fatalf("CT124 contradictorio no debe reservar: %v, reservas=%d", err, reservas)
	}
	cfg = configuracionB2Prueba(c, &reservas)
	cfg.Bolsa = bolsaNominalPrueba(func(context.Context, ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
		return PersonaSeleccionadaBolsa{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
			AceptacionRef: c.AceptacionRef, LlamamientoRef: c.LlamamientoRef, SeleccionRef: "seleccion:otra",
			VersionSeleccion: c.VersionSeleccion, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, ReciboRef: "recibo:bolsa"}, nil
	})
	f, _ = NuevosPlanesNominales(cfg)
	_, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || reservas != 0 {
		t.Fatalf("Bolsa incongruente no debe reservar: %v, reservas=%d", err, reservas)
	}
}
