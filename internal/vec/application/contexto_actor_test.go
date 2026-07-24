package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteContextoActorPrueba struct {
	mu           sync.Mutex
	instantaneas []domain.InstantaneaContextoActor
	error        error
	llamadas     int
	ultima       domain.SolicitudContextoActor
}

func (f *fuenteContextoActorPrueba) BuscarInstantaneasContextoActor(
	_ context.Context,
	solicitud domain.SolicitudContextoActor,
) ([]domain.InstantaneaContextoActor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.llamadas++
	f.ultima = solicitud
	return f.instantaneas, f.error
}

func (f *fuenteContextoActorPrueba) numeroLlamadas() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.llamadas
}

type relojContextoActorPrueba struct{ ahora time.Time }

func (r *relojContextoActorPrueba) Ahora() time.Time { return r.ahora }

type relojSecuenciaContextoActorPrueba struct {
	mu        sync.Mutex
	instantes []time.Time
	llamadas  int
}

func (r *relojSecuenciaContextoActorPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	indice := r.llamadas
	r.llamadas++
	if len(r.instantes) == 0 {
		return time.Time{}
	}
	if indice >= len(r.instantes) {
		indice = len(r.instantes) - 1
	}
	return r.instantes[indice]
}

type resolutorRegistroContextoActorV2Prueba struct {
	mu        sync.Mutex
	resultado ports.ConfirmacionRegistroContextoActorV2
	error     error
	llamadas  int
	solicitud ports.SolicitudResolucionRegistroContextoActorV2
}

func (r *resolutorRegistroContextoActorV2Prueba) ResolverYRegistrarContextoActorV2(
	_ context.Context,
	solicitud ports.SolicitudResolucionRegistroContextoActorV2,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadas++
	r.solicitud = solicitud
	return r.resultado, r.error
}

func (r *resolutorRegistroContextoActorV2Prueba) observacion() (int, ports.SolicitudResolucionRegistroContextoActorV2) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas, r.solicitud
}

type generadorOperacionContextoActorV2Prueba struct {
	mu       sync.Mutex
	ref      string
	error    error
	llamadas int
}

func (g *generadorOperacionContextoActorV2Prueba) NuevaReferenciaOperacionContextoActorV2(
	_ context.Context,
) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.llamadas++
	return g.ref, g.error
}

func nuevoGeneradorOperacionContextoActorV2Prueba() *generadorOperacionContextoActorV2Prueba {
	return &generadorOperacionContextoActorV2Prueba{
		ref: referenciaServicioContextoActorPrueba("oca_", "o"),
	}
}

func confirmacionRegistroContextoActorV2Prueba(
	t *testing.T,
	actor domain.ContextoActor,
	operacionRef string,
) ports.ConfirmacionRegistroContextoActorV2 {
	t.Helper()
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatalf("representar contexto: %v", err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatalf("calcular huella contexto: %v", err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:     referenciaServicioContextoActorPrueba("prc_", "p"),
		ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("1", 64),
		ProcedenciaAutoridad: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: actor.Instantanea.CuentaRef, Version: actor.Instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: actor.PersonaRef, Version: actor.Instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: actor.PerfilActivoRef, Version: actor.Instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: actor.Instantanea.VinculoRef, Version: actor.Instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0, len(actor.Instantanea.Vinculos)),
	}
	for _, vinculo := range actor.Instantanea.Vinculos {
		manifiesto.Vinculos = append(manifiesto.Vinculos, domain.ProcedenciaVinculoReferenciaContextoActorV1{
			VinculoRef: vinculo.VinculoRef, Version: vinculo.Version,
			Tipo: vinculo.Tipo, Referencia: vinculo.Referencia,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		})
	}
	representacionManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatalf("representar manifiesto: %v", err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(representacionManifiesto)
	if err != nil {
		t.Fatalf("huella manifiesto: %v", err)
	}
	return ports.ConfirmacionRegistroContextoActorV2{
		OperacionRef:        operacionRef,
		RegistroContextoRef: referenciaServicioContextoActorPrueba("rca_", "r"),
		Contexto:            actor, RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     representacionManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
}

type contextoActorPruebaMutador struct {
	solicitud   domain.SolicitudContextoActor
	instantanea domain.InstantaneaContextoActor
	instante    time.Time
}

func (m contextoActorPruebaMutador) crear(t *testing.T) domain.ContextoActor {
	t.Helper()
	actor, err := domain.NuevoContextoActor(m.solicitud.Cuenta, m.instantanea, m.instante)
	if err != nil {
		t.Fatalf("crear variante de contexto: %v", err)
	}
	return actor
}

func contextoActorServicioPrueba(
	t *testing.T,
	instante time.Time,
	solicitud domain.SolicitudContextoActor,
) domain.ContextoActor {
	t.Helper()
	actor, err := domain.NuevoContextoActor(
		solicitud.Cuenta,
		instantaneaServicioContextoActorPrueba(instante, solicitud),
		instante,
	)
	if err != nil {
		t.Fatalf("crear contexto actor: %v", err)
	}
	return actor
}

func TestServicioContextoActorResuelveUnaPersonaSinHeredarAutoridad(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	instantanea := instantaneaServicioContextoActorPrueba(instante, solicitud)
	fuente := &fuenteContextoActorPrueba{instantaneas: []domain.InstantaneaContextoActor{instantanea}}
	servicio := nuevoServicioContextoActorPrueba(t, fuente, instante)

	contextoActor, err := servicio.Resolver(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if contextoActor.Principal.ID != instantanea.PersonaRef ||
		contextoActor.PerfilActivoRef != solicitud.PerfilActivoRef ||
		contextoActor.Principal.AuthMethod != solicitud.Cuenta.Metodo ||
		contextoActor.Principal.AuthAssurance != solicitud.Cuenta.Garantia {
		t.Fatalf("contexto distinto: %#v", contextoActor)
	}
	if len(contextoActor.Principal.Roles) != 0 || len(contextoActor.Principal.Permissions) != 0 ||
		len(contextoActor.Principal.Attributes) != 0 {
		t.Fatalf("se heredaron claims como autoridad: %#v", contextoActor.Principal)
	}
	if fuente.ultima.Cuenta.CuentaRef != solicitud.Cuenta.CuentaRef ||
		fuente.ultima.PerfilActivoRef != solicitud.PerfilActivoRef {
		t.Fatalf("la fuente no recibio la seleccion exacta: %#v", fuente.ultima)
	}
	candidatos, _ := contextoActor.Referencias(domain.TipoReferenciaContextoActorCandidato)
	empleados, _ := contextoActor.Referencias(domain.TipoReferenciaContextoActorEmpleado)
	if len(candidatos) != 1 || len(empleados) != 1 {
		t.Fatalf("faltan referencias de modulo: candidatos=%#v empleados=%#v", candidatos, empleados)
	}
}

func TestServicioContextoActorDeniegaCeroDosAjenoRevocadoONoVigente(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	base := instantaneaServicioContextoActorPrueba(instante, solicitud)
	casos := []struct {
		nombre       string
		instantaneas func() []domain.InstantaneaContextoActor
	}{
		{"cero coincidencias", func() []domain.InstantaneaContextoActor { return nil }},
		{"dos coincidencias de la misma persona", func() []domain.InstantaneaContextoActor { return []domain.InstantaneaContextoActor{base, base} }},
		{"cuenta ajena", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.CuentaRef = referenciaServicioContextoActorPrueba("cta_", "x")
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"perfil ajeno", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.PerfilActivoRef = referenciaServicioContextoActorPrueba("prf_", "x")
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"perfil revocado", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.Estado = domain.EstadoVinculoContextoActorRevocado
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"perfil futuro", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.VigenteDesde = instante.Add(time.Minute)
			valor.VigenteHasta = instante.Add(time.Hour)
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"perfil caducado", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.VigenteDesde = instante.Add(-time.Hour)
			valor.VigenteHasta = instante
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"vinculo de modulo revocado", func() []domain.InstantaneaContextoActor {
			valor, _ := base.ClonarCanonica()
			valor.Vinculos[0].Estado = domain.EstadoVinculoContextoActorRevocado
			return []domain.InstantaneaContextoActor{valor}
		}},
		{"instantanea no canonica", func() []domain.InstantaneaContextoActor {
			valor := base
			valor.PersonaRef += "*"
			return []domain.InstantaneaContextoActor{valor}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteContextoActorPrueba{instantaneas: caso.instantaneas()}
			servicio := nuevoServicioContextoActorPrueba(t, fuente, instante)
			resultado, err := servicio.Resolver(context.Background(), solicitud)
			if !errors.Is(err, domain.ErrContextoActorNoResuelto) || resultado.Validar() == nil {
				t.Fatalf("se resolvio contexto ambiguo o no vigente: resultado=%#v error=%v", resultado, err)
			}
		})
	}
}

func TestServicioContextoActorNoInfiereNiNormalizaPerfil(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	base := solicitudServicioContextoActorPrueba()
	casos := []struct {
		nombre string
		mutar  func(*domain.SolicitudContextoActor)
	}{
		{"perfil ausente", func(s *domain.SolicitudContextoActor) { s.PerfilActivoRef = "" }},
		{"perfil con espacio", func(s *domain.SolicitudContextoActor) { s.PerfilActivoRef = " " + s.PerfilActivoRef }},
		{"perfil con comodin", func(s *domain.SolicitudContextoActor) { s.PerfilActivoRef += "*" }},
		{"cuenta como DNI", func(s *domain.SolicitudContextoActor) { s.Cuenta.CuentaRef = "12345678Z" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fuente := &fuenteContextoActorPrueba{instantaneas: []domain.InstantaneaContextoActor{
				instantaneaServicioContextoActorPrueba(instante, base),
			}}
			servicio := nuevoServicioContextoActorPrueba(t, fuente, instante)
			solicitud := base
			caso.mutar(&solicitud)
			if _, err := servicio.Resolver(context.Background(), solicitud); !errors.Is(err, domain.ErrContextoActorNoResuelto) {
				t.Fatalf("error no cerrado: %v", err)
			}
			if fuente.numeroLlamadas() != 0 {
				t.Fatal("una entrada invalida llego a la dependencia")
			}
		})
	}
}

func TestServicioContextoActorSaneaFalloDeDependencia(t *testing.T) {
	const detalleSensible = "dsn=postgres://secreto@interno"
	instante := instanteServicioContextoActorPrueba()
	fuente := &fuenteContextoActorPrueba{error: errors.New(detalleSensible)}
	servicio := nuevoServicioContextoActorPrueba(t, fuente, instante)

	_, err := servicio.Resolver(context.Background(), solicitudServicioContextoActorPrueba())
	if !errors.Is(err, domain.ErrContextoActorNoResuelto) ||
		!errors.Is(err, ports.ErrFuenteContextoActorNoDisponible) {
		t.Fatalf("fallo no traducido: %v", err)
	}
	if strings.Contains(err.Error(), detalleSensible) {
		t.Fatalf("se filtro el error de la dependencia: %v", err)
	}
}

func TestServicioContextoActorHaceCopiasDefensivas(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	base := instantaneaServicioContextoActorPrueba(instante, solicitud)
	fuente := &fuenteContextoActorPrueba{instantaneas: []domain.InstantaneaContextoActor{base}}
	servicio := nuevoServicioContextoActorPrueba(t, fuente, instante)

	primero, err := servicio.Resolver(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primera resolucion: %v", err)
	}
	primero.Instantanea.Vinculos[0].Referencia = referenciaServicioContextoActorPrueba("can_", "z")
	segundo, err := servicio.Resolver(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("segunda resolucion: %v", err)
	}
	referencias, _ := segundo.Referencias(domain.TipoReferenciaContextoActorCandidato)
	if len(referencias) != 1 || referencias[0] != referenciaServicioContextoActorPrueba("can_", "c") {
		t.Fatalf("el llamador altero la fuente: %#v", referencias)
	}

	// Una mutacion posterior de la respuesta defectuosa de una fuente tampoco
	// cambia el contexto ya emitido.
	fuente.mu.Lock()
	fuente.instantaneas[0].Vinculos[0].Referencia = referenciaServicioContextoActorPrueba("can_", "x")
	fuente.mu.Unlock()
	originales, _ := segundo.Referencias(domain.TipoReferenciaContextoActorCandidato)
	if originales[0] != referenciaServicioContextoActorPrueba("can_", "c") {
		t.Fatal("el contexto compartio memoria con la fuente")
	}
}

func TestNuevoServicioContextoActorRechazaDependenciasNulasTipadas(t *testing.T) {
	var fuenteNula *fuenteContextoActorPrueba
	var relojNulo *relojContextoActorPrueba
	validaFuente := &fuenteContextoActorPrueba{}
	validoReloj := &relojContextoActorPrueba{ahora: instanteServicioContextoActorPrueba()}
	casos := []struct {
		nombre string
		fuente ports.FuenteContextoActor
		reloj  ports.Reloj
	}{
		{"fuente nil", nil, validoReloj},
		{"fuente nil tipada", fuenteNula, validoReloj},
		{"reloj nil", validaFuente, nil},
		{"reloj nil tipado", validaFuente, relojNulo},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if servicio, err := NuevoServicioContextoActor(caso.fuente, caso.reloj); servicio != nil ||
				!errors.Is(err, domain.ErrContextoActorInvalido) {
				t.Fatalf("dependencia nula aceptada: servicio=%#v error=%v", servicio, err)
			}
		})
	}
}

func TestServicioContextoActorDeniegaRelojNuloYContextoCancelado(t *testing.T) {
	instante := instanteServicioContextoActorPrueba()
	solicitud := solicitudServicioContextoActorPrueba()
	fuente := &fuenteContextoActorPrueba{instantaneas: []domain.InstantaneaContextoActor{
		instantaneaServicioContextoActorPrueba(instante, solicitud),
	}}
	servicio, err := NuevoServicioContextoActor(fuente, &relojContextoActorPrueba{})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	if _, err = servicio.Resolver(context.Background(), solicitud); !errors.Is(err, domain.ErrContextoActorNoResuelto) {
		t.Fatalf("reloj nulo no denegado: %v", err)
	}

	servicio = nuevoServicioContextoActorPrueba(t, fuente, instante)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = servicio.Resolver(ctx, solicitud); !errors.Is(err, context.Canceled) ||
		!errors.Is(err, domain.ErrContextoActorNoResuelto) {
		t.Fatalf("cancelacion no preservada y cerrada: %v", err)
	}
}

func nuevoServicioContextoActorPrueba(
	t *testing.T,
	fuente ports.FuenteContextoActor,
	instante time.Time,
) *ServicioContextoActor {
	t.Helper()
	servicio, err := NuevoServicioContextoActor(fuente, &relojContextoActorPrueba{ahora: instante})
	if err != nil {
		t.Fatalf("crear servicio: %v", err)
	}
	return servicio
}

func solicitudServicioContextoActorPrueba() domain.SolicitudContextoActor {
	return domain.SolicitudContextoActor{
		Cuenta: domain.CuentaAutenticadaContextoActor{
			CuentaRef: referenciaServicioContextoActorPrueba("cta_", "a"),
			Metodo:    domain.AuthMethodCertificate,
			Garantia:  domain.AuthAssuranceHigh,
		},
		PerfilActivoRef: referenciaServicioContextoActorPrueba("prf_", "p"),
	}
}

func instantaneaServicioContextoActorPrueba(
	instante time.Time,
	solicitud domain.SolicitudContextoActor,
) domain.InstantaneaContextoActor {
	return domain.InstantaneaContextoActor{
		VinculoRef:      referenciaServicioContextoActorPrueba("vca_", "v"),
		VinculoVersion:  1,
		CuentaRef:       solicitud.Cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      referenciaServicioContextoActorPrueba("per_", "r"),
		PersonaVersion:  2,
		PerfilActivoRef: solicitud.PerfilActivoRef,
		PerfilVersion:   3,
		Estado:          domain.EstadoVinculoContextoActorActivo,
		VigenteDesde:    instante.Add(-time.Hour),
		VigenteHasta:    instante.Add(time.Hour),
		Vinculos: []domain.VinculoReferenciaContextoActor{
			{
				VinculoRef:   referenciaServicioContextoActorPrueba("vin_", "c"),
				Version:      1,
				Tipo:         domain.TipoReferenciaContextoActorCandidato,
				Referencia:   referenciaServicioContextoActorPrueba("can_", "c"),
				Estado:       domain.EstadoVinculoContextoActorActivo,
				VigenteDesde: instante.Add(-time.Hour),
				VigenteHasta: instante.Add(time.Hour),
			},
			{
				VinculoRef:   referenciaServicioContextoActorPrueba("vin_", "e"),
				Version:      1,
				Tipo:         domain.TipoReferenciaContextoActorEmpleado,
				Referencia:   referenciaServicioContextoActorPrueba("emp_", "e"),
				Estado:       domain.EstadoVinculoContextoActorActivo,
				VigenteDesde: instante.Add(-time.Hour),
				VigenteHasta: instante.Add(time.Hour),
			},
		},
	}
}

func instanteServicioContextoActorPrueba() time.Time {
	return time.Date(2026, time.July, 15, 11, 0, 0, 0, time.UTC)
}

func referenciaServicioContextoActorPrueba(prefijo, caracter string) string {
	return prefijo + strings.Repeat(caracter, 24)
}
