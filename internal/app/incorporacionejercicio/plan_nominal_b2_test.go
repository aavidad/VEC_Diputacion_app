package incorporacionejercicio

import (
	"context"
	"errors"
	"strings"
	"testing"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
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
		Protocolo: ProtocoloPersonalB2V1, ContratoRef: "planct:uno", OrganizacionRef: "organizacion:uno", ExpedienteRef: "expediente:uno",
		VersionExpediente: 7, ContratoVersion: 1, ContratoReciboRef: "recibo:contrato", ContratoSHA256: strings.Repeat("a", 64),
		SolicitudRef: "solicitud:personal", IntencionRef: "intencion:uno", IntencionReciboRef: "recibo:intencion",
		IntencionVersion: 1, AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion",
		LlamamientoRef: "llamamiento:uno", SeleccionRef: "seleccion:uno", VersionSeleccion: 2,
		SeleccionReciboRef: "recibo:bolsa", PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PersonaFuente: dom.FuentePlanPersonalB2{Ref: "fuente:bolsa", Version: 1, SHA256: strings.Repeat("d", 64)},
		FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{
			Referencia: "rpt:uno", Version: 1, HuellaSHA256: strings.Repeat("b", 64),
		}, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo",
		PuestoRef: "puesto:uno", PlazaRef: "plaza:uno",
		ReservaIdempotente: "11111111-1111-4111-8111-111111111111",
		SelectorBolsa:      dom.SelectorBolsaPlanB2{UnidadRef: "unidad:uno", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno", AceptacionOperacionRef: "bolsa:aceptacion", AceptacionRegistroSHA256: strings.Repeat("e", 64), AperturaOperacionRef: "bolsa:apertura", AperturaRegistroSHA256: strings.Repeat("f", 64), LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:uno"},
		DatosPersonal: DatosActosPersonalB2{
			OrganismoRef: "organismo:uno", UnidadRef: "unidad:uno",
			Regimen:   personal.EntradaCatalogoEmpleadoB2{Ref: "regimen:uno", Version: 1},
			Modalidad: personal.EntradaCatalogoEmpleadoB2{Ref: "modalidad:uno", Version: 1},
			Desde:     personal.FechaCivil("2026-10-01"), ClaseOcupacion: "temporal",
			VersionPlantillaRef: "plantilla:uno", VersionRPTRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 1,
			FuenteOrganizacionRef: "fuente:organizacion", FuenteOrganizacionSHA256: strings.Repeat("c", 64),
			CatalogoRPTID: "catalogo:rpt", ModuloRPTID: "personal", CategoriaID: "categoria:uno",
			CatalogoRPTVersion: 1, CatalogoRPTHuellaSHA256: strings.Repeat("b", 64),
			Procedencia: personal.ProcedenciaActoEmpleadoB2{ActoRef: "acto:preparacion", FuenteRef: "fuente:ct",
				FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64), IdempotenciaRef: "11111111-1111-4111-8111-111111111111"},
		},
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
				VersionSeleccion: c.VersionSeleccion, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3, ReciboRef: "recibo:bolsa",
				FuenteRef: "fuente:bolsa", FuenteVersion: 1, FuenteSHA256: strings.Repeat("d", 64)}, nil
		}),
		RPT: rptNominalPrueba(func(context.Context, ContratoPlanNominal) (PuestoRPTNominal, error) {
			return PuestoRPTNominal{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				Fuente: c.FuenteRPT, CategoriaRef: c.CategoriaRef, PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef,
				VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo", Prospectivo: true}, nil
		}),
		Personal: personalNominalPrueba(func(_ context.Context, s SolicitudReservaPersonalB2) (ReservaPersonalB2, error) {
			*reservas++
			return ReservaPersonalB2{OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
				SolicitudRef: c.SolicitudRef, IdempotenciaRef: c.ReservaIdempotente,
				PersonaRef: s.Persona.PersonaRef, PersonaVersion: s.Persona.PersonaVersion,
				ContratoReciboRef: c.ContratoReciboRef, ContratoSHA256: c.ContratoSHA256,
				FuenteRPT: c.FuenteRPT, PuestoRef: c.PuestoRef, PlazaRef: c.PlazaRef,
				ReservaRef: "reserva:uno", ReciboRef: "recibo:personal", VersionReserva: 1, EmpleadoRef: "emp_0123456789abcdefghijkl",
				EjercicioSintetico: c.EjercicioSintetico}, nil
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
	if !errors.Is(err, ct.ErrPreparacionIncorporacionPendiente) || p != (PlanNominal{}) {
		t.Fatalf("fixture B2 sigue exigiendo sus fuentes: %+v, %v", p, err)
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

func TestPlanesNominalesB2PersonaSelladaCambiaEntrePlanYConfirmacion(t *testing.T) {
	c := contratoB2Prueba()
	for _, caso := range []struct {
		nombre  string
		cambiar func(*PersonaSeleccionadaBolsa)
	}{
		{"persona", func(p *PersonaSeleccionadaBolsa) { p.PersonaRef = "per_zzzzzzzzzzzzzzzzzzzzzzzz" }},
		{"version", func(p *PersonaSeleccionadaBolsa) { p.PersonaVersion++ }},
		{"fuente", func(p *PersonaSeleccionadaBolsa) { p.FuenteRef = "fuente:otra" }},
		{"version fuente", func(p *PersonaSeleccionadaBolsa) { p.FuenteVersion++ }},
		{"huella fuente", func(p *PersonaSeleccionadaBolsa) { p.FuenteSHA256 = strings.Repeat("e", 64) }},
		{"recibo", func(p *PersonaSeleccionadaBolsa) { p.ReciboRef = "recibo:otro" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			reservas, rpt := 0, 0
			cfg := configuracionB2Prueba(c, &reservas)
			original := cfg.Bolsa
			cfg.Bolsa = bolsaNominalPrueba(func(ctx context.Context, c ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
				p, err := original.LeerPersonaSeleccionada(ctx, c)
				caso.cambiar(&p)
				return p, err
			})
			cfg.RPT = rptNominalPrueba(func(context.Context, ContratoPlanNominal) (PuestoRPTNominal, error) {
				rpt++
				return PuestoRPTNominal{}, nil
			})
			f, err := NuevosPlanesNominales(cfg)
			if err != nil {
				t.Fatal(err)
			}
			_, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
			if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || rpt != 0 || reservas != 0 {
				t.Fatalf("mutación admitida: %v, rpt=%d, reservas=%d", err, rpt, reservas)
			}
		})
	}
}

func TestPlanesNominalesB2ReleeBolsaTrasRPTAntesDeReservar(t *testing.T) {
	c := contratoB2Prueba()
	reservas, lecturas := 0, 0
	cfg := configuracionB2Prueba(c, &reservas)
	original := cfg.Bolsa
	cfg.Bolsa = bolsaNominalPrueba(func(ctx context.Context, c ContratoPlanNominal) (PersonaSeleccionadaBolsa, error) {
		lecturas++
		p, err := original.LeerPersonaSeleccionada(ctx, c)
		if lecturas == 2 {
			p.PersonaRef = "per_zzzzzzzzzzzzzzzzzzzzzzzz"
		}
		return p, err
	})
	f, err := NuevosPlanesNominales(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Preparar(context.Background(), c.OrganizacionRef, c.ExpedienteRef)
	if !errors.Is(err, ct.ErrConflictoIncorporacionAplicacion) || lecturas != 2 || reservas != 0 {
		t.Fatalf("mutación tras RPT admitida: %v, lecturas=%d, reservas=%d", err, lecturas, reservas)
	}
}
