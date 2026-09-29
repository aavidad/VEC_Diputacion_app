package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type validadorMotivoPreparacionVigentePrueba struct{ disponible bool }

func (v validadorMotivoPreparacionVigentePrueba) ValidarReferenciaMotivoAutorizacionV2(
	_ context.Context, motivo dominiovec.ReferenciaEntradaCatalogo, _ time.Time,
) error {
	if !v.disponible || motivo != motivoAutorizacionPreparacionVigenteCoberturaDesarrollo() {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

type autoridadPreparacionVigenteRevocablePrueba struct {
	preparadas   int
	publicadas   int
	retirada     bool
	roles        []string
	asignaciones []string
}

type registroPreparacionVigenteRevocablePrueba struct {
	base      registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba
	autoridad *autoridadPreparacionVigenteRevocablePrueba
}

func (r *registroPreparacionVigenteRevocablePrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if r.autoridad.retirada {
		return time.Time{}, puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	return r.base.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (r *registroPreparacionVigenteRevocablePrueba) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if r.autoridad.retirada {
		return puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	return r.base.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

func (a *autoridadPreparacionVigenteRevocablePrueba) PrepararInstantanea(
	_ context.Context, i dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	a.preparadas++
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(i), nil
}

func (a *autoridadPreparacionVigenteRevocablePrueba) PublicarInstantanea(
	_ context.Context, i dominiovec.InstantaneaAutorizacion,
) error {
	if a.retirada {
		return puertosvec.ErrInstantaneaAutorizacionObsoleta
	}
	a.publicadas++
	a.roles = append(a.roles, i.VersionRol.Referencia())
	a.asignaciones = append(a.asignaciones, i.AsignacionPerfil.Referencia())
	return nil
}

func escenarioPreparacionVigenteAutorizacionPrueba(t *testing.T) (
	*soporteAltaContratacionTemporalDesarrollo,
	*autorizadorConsultaPreparacionVigenteDesarrollo,
	*autorizadorConsultasCoberturaDesarrollo,
	*politicaPreparacionVigenteCoberturaDesarrollo,
	*autoridadPreparacionVigenteRevocablePrueba,
	dominiovec.Principal,
) {
	t.Helper()
	soporte, propuesta, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v3, err := instantaneaCoberturaConPreparacionVigenteDesarrollo(soporte.instantaneaCobertura)
	if err != nil {
		t.Fatal(err)
	}
	soporte.instantaneaCobertura = v3
	autoridad := &autoridadPreparacionVigenteRevocablePrueba{}
	soporte.autoridadAsignaciones = autoridad
	publicada, err := autoridad.PrepararInstantanea(context.Background(), v3)
	if err != nil || autoridad.PublicarInstantanea(context.Background(), publicada) != nil {
		t.Fatalf("publicación gobernada de prueba: %v", err)
	}
	soporte.registroDecisionesAnalisis = &registroPreparacionVigenteRevocablePrueba{autoridad: autoridad}
	politica, err := nuevaPoliticaPreparacionVigenteCoberturaDesarrollo(
		soporte, validadorMotivoPreparacionVigentePrueba{disponible: true}, publicada,
	)
	if err != nil {
		t.Fatal(err)
	}
	p := politica.fuente.(*politicaPreparacionVigenteCoberturaDesarrollo)
	generador := seguridadvec.GeneradorReferenciasCriptograficas{}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		politica.fuente, politica.registroConcesiones, politica.registroDenegaciones,
		politica.validadorMotivos, soporte.reloj, generador,
		aplicacionvec.ConfiguracionServicioAutorizacion{},
	)
	if err != nil {
		t.Fatal(err)
	}
	autorizador, err := nuevoAutorizadorConsultaPreparacionVigenteDesarrollo(
		soporte, servicio, generador, motivoAutorizacionPreparacionVigenteCoberturaDesarrollo(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return soporte, autorizador, propuesta, p, autoridad, principal
}

func solicitudContextoPreparacionVigentePrueba(t *testing.T, soporte *soporteAltaContratacionTemporalDesarrollo) ports.SolicitudResolverContextoAutorizacionAltaV3 {
	t.Helper()
	vinculo, err := soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
	}
}

func TestPreparacionVigenteCoberturaAccionPropiaYRegistroV3(t *testing.T) {
	soporte, autorizador, _, politica, autoridad, principal := escenarioPreparacionVigenteAutorizacionPrueba(t)
	if politica.publicada.VersionRol.Validar() != nil || politica.publicada.VersionRol.Version != 3 ||
		len(politica.publicada.VersionRol.Concesiones) != 5 ||
		politica.publicada.VersionRol.Concesiones[4].Accion != accionPreparacionVigenteCoberturaDesarrollo ||
		politica.publicada.VersionRol.Concesiones[4].TipoRecurso != tipoPreparacionVigenteCoberturaDesarrollo ||
		len(politica.publicada.VersionRol.Concesiones[4].CamposPermitidos) != 12 ||
		len(politica.publicada.VersionRol.Concesiones[4].Obligaciones) != 1 ||
		politica.publicada.VersionRol.Concesiones[4].Obligaciones[0] != "registrar_acceso" {
		t.Fatalf("concesion GET no independiente: %+v", politica.publicada.VersionRol.Concesiones)
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, rutaPreparacionVigenteCoberturaDesarrollo)
	solicitudContexto := solicitudContextoPreparacionVigentePrueba(t, soporte)
	solicitud, err := autorizador.nuevaSolicitud(ctx, soporte.contexto, organizacionAltaContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := solicitud.Datos()
	if err != nil || datos.Accion != accionPreparacionVigenteCoberturaDesarrollo ||
		datos.Finalidad != finalidadPreparacionVigenteCoberturaDesarrollo ||
		datos.Recurso.Referencia != recursoPreparacionVigenteCoberturaDesarrollo ||
		datos.Recurso.Tipo != tipoPreparacionVigenteCoberturaDesarrollo ||
		len(datos.Recurso.Ambitos) != 2 || len(datos.Recurso.Atributos) != 0 {
		t.Fatalf("solicitud GET no fija: %+v, %v", datos, err)
	}
	if err := autorizador.AutorizarConsultaPreparacionCoberturaVigente(
		ctx, solicitudContexto, soporte.contexto,
		organizacionAltaContratacionTemporalDesarrollo, soporte.reloj.Ahora(),
	); err != nil {
		t.Fatalf("GET con concesion propia: %v", err)
	}
	registro := soporte.registroDecisionesAnalisis.(*registroPreparacionVigenteRevocablePrueba)
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 || registro.base.concesiones != 1 || registro.base.huella == "" ||
		len(soporte.concesiones) != 0 {
		t.Fatalf("faltó publicación y registro V3: autoridad=%+v registro=%+v", autoridad, registro)
	}
}

func TestPreparacionVigenteCoberturaVersionaSinMutarRolPublicado(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v2 := soporte.instantaneaCobertura
	v3, err := instantaneaCoberturaConPreparacionVigenteDesarrollo(v2)
	if err != nil || v3.VersionRol.Version != 3 || v2.VersionRol.Version != 2 ||
		len(v3.VersionRol.Concesiones) != 5 || len(v2.VersionRol.Concesiones) != 4 ||
		v2.VersionRol.Referencia() == v3.VersionRol.Referencia() {
		t.Fatalf("versionado no aditivo: v2=%+v v3=%+v err=%v", v2.VersionRol, v3.VersionRol, err)
	}
	repetida, err := instantaneaCoberturaConPreparacionVigenteDesarrollo(v3)
	if err != nil || repetida.VersionRol.Referencia() != v3.VersionRol.Referencia() ||
		len(repetida.VersionRol.Concesiones) != 5 {
		t.Fatalf("rearranque no idempotente: %+v, %v", repetida.VersionRol, err)
	}
	repetida.VersionRol.Concesiones[4].CamposPermitidos[0] = "alterado"
	if v3.VersionRol.Concesiones[4].CamposPermitidos[0] != camposPreparacionVigenteCoberturaDesarrollo[0] {
		t.Fatal("v3 compartió campos con su copia")
	}
}

func TestPreparacionVigenteCoberturaComparteRolV3ConPropuestaYRevoca(t *testing.T) {
	soporte, get, propuesta, politica, autoridad, principal := escenarioPreparacionVigenteAutorizacionPrueba(t)
	if politica.publicada.VersionRol.Referencia() != soporte.instantaneaCobertura.VersionRol.Referencia() ||
		politica.publicada.AsignacionPerfil.Referencia() != soporte.instantaneaCobertura.AsignacionPerfil.Referencia() {
		t.Fatal("GET y propuesta no comparten la semilla de rol y asignacion v3")
	}
	solicitudContexto := solicitudContextoPreparacionVigentePrueba(t, soporte)
	getCtx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, rutaPreparacionVigenteCoberturaDesarrollo)
	propuestaCtx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaPropuestaCobertura)
	analisis, err := cobertura.NuevaSolicitudInstantaneaAnalisisDurableO3(
		organizacionAltaContratacionTemporalDesarrollo, "expediente_temporal_desarrollo_0001", 2,
	)
	if err != nil {
		t.Fatal(err)
	}
	operaciones := []func() error{
		func() error {
			return get.AutorizarConsultaPreparacionCoberturaVigente(getCtx,
				solicitudContexto, soporte.contexto, organizacionAltaContratacionTemporalDesarrollo, soporte.reloj.Ahora())
		},
		func() error {
			return propuesta.AutorizarPresentacionPropuestaCobertura(propuestaCtx,
				solicitudContexto, soporte.contexto, analisis, soporte.reloj.Ahora())
		},
		func() error {
			return get.AutorizarConsultaPreparacionCoberturaVigente(getCtx,
				solicitudContexto, soporte.contexto, organizacionAltaContratacionTemporalDesarrollo, soporte.reloj.Ahora())
		},
	}
	for indice, operacion := range operaciones {
		if err := operacion(); err != nil {
			t.Fatalf("operacion %d GET/propuesta/GET: %v", indice, err)
		}
	}
	registro := soporte.registroDecisionesAnalisis.(*registroPreparacionVigenteRevocablePrueba)
	if autoridad.publicadas != 2 || registro.base.concesiones != 3 || len(autoridad.roles) != 2 ||
		len(autoridad.asignaciones) != 2 {
		t.Fatalf("recorrido incompleto: autoridad=%+v registro=%+v", autoridad, registro)
	}
	for indice := 1; indice < 2; indice++ {
		if autoridad.roles[indice] != autoridad.roles[0] ||
			autoridad.asignaciones[indice] != autoridad.asignaciones[0] {
			t.Fatalf("flapping de autoridad en %d: %+v", indice, autoridad)
		}
	}
	autoridad.retirada = true
	for indice, operacion := range operaciones[:2] {
		if err := operacion(); err == nil {
			t.Fatalf("operacion %d autorizada tras revocacion", indice)
		}
	}
	if registro.base.concesiones != 3 || autoridad.publicadas != 2 {
		t.Fatalf("revocacion produjo efecto: autoridad=%+v registro=%+v", autoridad, registro)
	}
}

func TestPreparacionVigenteCoberturaCancelacionesNoRetienenSolicitudes(t *testing.T) {
	soporte, autorizador, _, politica, autoridad, principal := escenarioPreparacionVigenteAutorizacionPrueba(t)
	for indice := 0; indice < 1030; indice++ {
		ctx, cancelar := context.WithCancel(contextoRutaCoberturaDesarrolloPrueba(
			soporte, principal, rutaPreparacionVigenteCoberturaDesarrollo,
		))
		solicitud, err := autorizador.nuevaSolicitud(ctx, soporte.contexto,
			organizacionAltaContratacionTemporalDesarrollo)
		if err != nil {
			t.Fatal(err)
		}
		ctx = context.WithValue(ctx, claveSolicitudPreparacionVigenteCoberturaDesarrollo{}, solicitud)
		vinculo, err := soporte.contexto.Vinculo.Datos()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := politica.ObtenerInstantaneaAutorizacion(ctx, vinculo.PrincipalID, vinculo.PerfilActivoRef); err != nil {
			t.Fatalf("consulta %d agotó la fuente: %v", indice, err)
		}
		cancelar()
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, rutaPreparacionVigenteCoberturaDesarrollo)
	if err := autorizador.AutorizarConsultaPreparacionCoberturaVigente(ctx,
		solicitudContextoPreparacionVigentePrueba(t, soporte), soporte.contexto,
		organizacionAltaContratacionTemporalDesarrollo, soporte.reloj.Ahora()); err != nil {
		t.Fatalf("fuente no recuperó tras cancelaciones: %v", err)
	}
	if autoridad.preparadas != 1 || autoridad.publicadas != 1 {
		t.Fatalf("GET reescribió gobierno durante lectura: %+v", autoridad)
	}
}

func TestPreparacionVigenteCoberturaCierraRutaPerfilRevocacionYRegistro(t *testing.T) {
	for _, caso := range []struct {
		nombre       string
		preparar     func(*soporteAltaContratacionTemporalDesarrollo, *politicaPreparacionVigenteCoberturaDesarrollo, *autoridadPreparacionVigenteRevocablePrueba)
		ruta         string
		organizacion string
		esperado     error
	}{
		{"ruta propuesta", nil, httpinterno.RutaPropuestaCobertura, organizacionAltaContratacionTemporalDesarrollo, application.ErrPresentacionPropuestaCoberturaDenegada},
		{"organizacion ajena", nil, rutaPreparacionVigenteCoberturaDesarrollo, "organizacion:ajena", application.ErrPresentacionPropuestaCoberturaDenegada},
		{"perfil ajeno", func(s *soporteAltaContratacionTemporalDesarrollo, _ *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			s.instantaneaCobertura.AsignacionPerfil.PerfilActivoRef = "perfil:ajeno"
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPresentacionPropuestaCoberturaDenegada},
		{"revocada", func(_ *soporteAltaContratacionTemporalDesarrollo, _ *politicaPreparacionVigenteCoberturaDesarrollo, a *autoridadPreparacionVigenteRevocablePrueba) {
			a.retirada = true
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPresentacionPropuestaCoberturaDenegada},
		{"registro caido", func(s *soporteAltaContratacionTemporalDesarrollo, _ *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			s.registroDecisionesAnalisis.(*registroPreparacionVigenteRevocablePrueba).base.errConcesion =
				puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPresentacionPropuestaCoberturaNoDisponible},
		{"motivo sin publicar", func(_ *soporteAltaContratacionTemporalDesarrollo, p *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			p.validador = validadorMotivoPreparacionVigentePrueba{disponible: false}
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPresentacionPropuestaCoberturaDenegada},
		{"campo ausente", func(_ *soporteAltaContratacionTemporalDesarrollo, p *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			p.publicada.VersionRol.Concesiones[4].CamposPermitidos = p.publicada.VersionRol.Concesiones[4].CamposPermitidos[:11]
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
		{"campo futuro", func(_ *soporteAltaContratacionTemporalDesarrollo, p *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			p.publicada.VersionRol.Concesiones[4].CamposPermitidos = append(p.publicada.VersionRol.Concesiones[4].CamposPermitidos, "catalogo.vias.campo_futuro")
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
		{"obligacion futura", func(_ *soporteAltaContratacionTemporalDesarrollo, p *politicaPreparacionVigenteCoberturaDesarrollo, _ *autoridadPreparacionVigenteRevocablePrueba) {
			p.publicada.VersionRol.Concesiones[4].Obligaciones = []string{"registrar_acceso", "obligacion_futura"}
		}, rutaPreparacionVigenteCoberturaDesarrollo, organizacionAltaContratacionTemporalDesarrollo, application.ErrPreparacionCatalogoCoberturaNoDisponiblePerfil},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			soporte, autorizador, _, politica, autoridad, principal := escenarioPreparacionVigenteAutorizacionPrueba(t)
			if caso.preparar != nil {
				caso.preparar(soporte, politica, autoridad)
			}
			ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, caso.ruta)
			err := autorizador.AutorizarConsultaPreparacionCoberturaVigente(
				ctx, solicitudContextoPreparacionVigentePrueba(t, soporte), soporte.contexto,
				caso.organizacion, soporte.reloj.Ahora(),
			)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("error=%v, esperado=%v", err, caso.esperado)
			}
			registro := soporte.registroDecisionesAnalisis.(*registroPreparacionVigenteRevocablePrueba)
			if caso.nombre == "revocada" || caso.nombre == "registro caido" {
				if registro.base.concesiones != 0 {
					t.Fatalf("registro tras fallo: %+v", registro)
				}
			}
		})
	}
}

func TestPreparacionVigenteCoberturaNoPrestaMotivoDePropuesta(t *testing.T) {
	soporte, base, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	if _, err := nuevoAutorizadorConsultaPreparacionVigenteDesarrollo(
		soporte, nil, seguridadvec.GeneradorReferenciasCriptograficas{},
		motivoAutorizacionPreparacionVigenteCoberturaDesarrollo(),
	); !errors.Is(err, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible) {
		t.Fatalf("autorizador nulo aceptado: %v", err)
	}
	if _, err := nuevoAutorizadorConsultaPreparacionVigenteDesarrollo(
		soporte, base.autorizador, seguridadvec.GeneradorReferenciasCriptograficas{},
		soporte.motivoPropuestaCobertura,
	); !errors.Is(err, errAutorizacionPreparacionVigenteCoberturaDesarrolloNoDisponible) {
		t.Fatalf("motivo de propuesta prestado: %v", err)
	}
}
