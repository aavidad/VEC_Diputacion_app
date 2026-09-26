package application

// Fixtures sintéticas copiadas del entorno V3 de «Mi bolsa»: PDP y cadena
// criptográfica reales; gobierno y registro son dobles en memoria.
import (
	"context"
	"strings"
	"testing"
	"time"
	app "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"

	sel "vec-diputacion-granada/internal/modules/seleccion/ports"
)

const claveMotivoAutorizacionV2Prueba = "motivo_11111111111111111111111111111111"
const referenciaCorrelacionAutorizacionV2Prueba = "correlacion_11111111111111111111111111111111"

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
		// Persona ajena a la Diputación: ningún vínculo de candidato ni de
		// empleado.
		Vinculos: []domain.VinculoReferenciaContextoActor{},
	}
}

func referenciaServicioContextoActorPrueba(prefijo, caracter string) string {
	return prefijo + strings.Repeat(caracter, 24)
}

type fuenteAutorizacionServicioPrueba struct {
	instantanea  domain.InstantaneaAutorizacion
	err          error
	invocaciones int
	despues      func()
}

func (f *fuenteAutorizacionServicioPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context,
	_, _ string,
) (domain.InstantaneaAutorizacion, error) {
	f.invocaciones++
	if f.despues != nil {
		f.despues()
	}
	return f.instantanea, f.err
}

type relojAutorizacionServicioPrueba struct{ ahora time.Time }

func (r *relojAutorizacionServicioPrueba) Ahora() time.Time { return r.ahora }

type generadorAutorizacionServicioPrueba struct {
	referencia   string
	invocaciones int
}

func (g *generadorAutorizacionServicioPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	g.invocaciones++
	return g.referencia, nil
}

func instantaneaAutorizacionServicioPrueba(t *testing.T) domain.InstantaneaAutorizacion {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	concesiones := []domain.ConcesionRol{}
	for _, par := range sel.AccionesPropias() {
		concesiones = append(concesiones, domain.ConcesionRol{Accion: par[0], ModuloID: sel.ModuloSeleccion, TipoRecurso: sel.TipoRecursoSolicitudesPropias,
			Finalidades: []string{sel.FinalidadSolicitudesPropias}, GarantiaMinima: domain.AuthAssuranceHigh})
	}
	for _, par := range sel.AccionesRRHH() {
		concesiones = append(concesiones, domain.ConcesionRol{Accion: par[0], ModuloID: sel.ModuloSeleccion, TipoRecurso: sel.TipoRecursoSolicitudes,
			Finalidades: []string{sel.FinalidadConsultaSolicitudes}, GarantiaMinima: domain.AuthAssuranceHigh})
	}
	version := domain.VersionRol{
		RolID: "seleccion_prueba", Version: 1, Nombre: "Selección de prueba", Estado: domain.EstadoVersionRolPublicada,
		Concesiones:  concesiones,
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	asignacion := domain.AsignacionPerfil{
		AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PrincipalID: "per_0123456789abcdefghijkl",
		VersionRolRef: version.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos:      []domain.AmbitoPerfil{{Clave: "ambito_ref", Valores: []string{"seleccion"}}},
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatalf("huella de catalogo vacio: %v", err)
	}
	return domain.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
}

type generadorCorrelacionAplicacionPrueba struct{ valor string }

func (g generadorCorrelacionAplicacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func referenciaCorrelacionAplicacionPrueba(
	valor string,
) domain.ReferenciaCorrelacionAutorizacionV2 {
	referencia, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionAplicacionPrueba{valor: valor},
	)
	if err != nil {
		panic("fixture de correlacion nominal V2 invalido: " + err.Error())
	}
	return referencia
}

type validadorMotivoAutorizacionV2Prueba struct {
	referencia domain.ReferenciaEntradaCatalogo
}

func (v *validadorMotivoAutorizacionV2Prueba) ValidarReferenciaMotivoAutorizacionV2(
	_ context.Context,
	referencia domain.ReferenciaEntradaCatalogo,
	_ time.Time,
) error {
	if v == nil || referencia != v.referencia {
		return domain.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func nuevoValidadorMotivoAutorizacionV2Prueba() *validadorMotivoAutorizacionV2Prueba {
	return &validadorMotivoAutorizacionV2Prueba{
		referencia: referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba),
	}
}

func referenciaMotivoAutorizacionV2Prueba(clave string) domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		EntradaClave:         clave,
	}
}

type revalidadorVinculoAplicacionAdversarial struct {
	resultado    domain.AutenticacionRevalidadaV1
	err          error
	invocaciones int
	despues      func()
}

func (r *revalidadorVinculoAplicacionAdversarial) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	r.invocaciones++
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

type resolutorContextoAutorizacionV3Prueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorContextoAutorizacionV3Prueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type registroConcesionesAutorizacionV3Prueba struct {
	err            error
	invocaciones   int
	orden          ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3
	cancelar       context.CancelFunc
	decisionNoVive bool
	registradaEn   *time.Time
	devolverCero   bool
}

func (r *registroConcesionesAutorizacionV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.invocaciones++
	r.orden = orden
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	desde, _, err := datos.Decision.VentanaValidez()
	if err != nil {
		return time.Time{}, err
	}
	r.decisionNoVive = !datos.Decision.VigenteEn(desde)
	if r.err != nil {
		return time.Time{}, r.err
	}
	if r.devolverCero {
		return time.Time{}, nil
	}
	if r.registradaEn != nil {
		return *r.registradaEn, nil
	}
	if r.cancelar != nil {
		r.cancelar()
	}
	return desde, nil
}

type registroDenegacionesAutorizacionV3Prueba struct {
	err          error
	invocaciones int
	orden        ports.OrdenRegistroDenegacionAutorizacionLigadaV3
}

func (r *registroDenegacionesAutorizacionV3Prueba) RegistrarDenegacionAutorizacionLigadaV3(
	_ context.Context,
	orden ports.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	r.invocaciones++
	r.orden = orden
	if _, err := orden.Datos(); err != nil {
		return err
	}
	return r.err
}

type entornoAutorizacionSolicitudV3Prueba struct {
	autenticacion domain.AutenticacionRevalidadaV1
	ahora         time.Time
	solicitud     domain.SolicitudAutorizacionLigadaV3
	resultado     domain.ResultadoContextoActorRegistradoV2
	instantanea   domain.InstantaneaAutorizacion
	fuente        *fuenteAutorizacionServicioPrueba
	concesiones   *registroConcesionesAutorizacionV3Prueba
	denegaciones  *registroDenegacionesAutorizacionV3Prueba
	motivos       *validadorMotivoAutorizacionV2Prueba
	servicio      *app.ServicioAutorizacionSolicitudLigadaV3
}

func nuevoEntornoAutorizacionSolicitudV3Prueba(t *testing.T, superficie domain.SuperficieAutenticacionActorV1, opciones ...opcionContexto) *entornoAutorizacionSolicitudV3Prueba {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	solicitudContexto := solicitudServicioContextoActorPrueba()
	instantaneaActor := instantaneaServicioContextoActorPrueba(ahora.Add(-2*time.Minute), solicitudContexto)
	for _, opcion := range opciones {
		opcion(&instantaneaActor)
	}
	actor, errActor := domain.NuevoContextoActor(solicitudContexto.Cuenta, instantaneaActor, ahora.Add(-2*time.Minute))
	if errActor != nil {
		t.Fatal(errActor)
	}
	recibo := confirmacionRegistroContextoActorV2Prueba(
		t, actor, referenciaServicioContextoActorPrueba("oca_", "o"),
	)
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: recibo.RegistroContextoRef, Contexto: recibo.Contexto,
		RepresentacionCanonica: append([]byte(nil), recibo.RepresentacionCanonica...),
		HuellaSHA256:           recibo.HuellaSHA256,
		ManifiestoProcedenciaCanonico: append(
			[]byte(nil), recibo.ManifiestoProcedenciaCanonico...,
		),
		ManifiestoProcedenciaHuellaSHA256: recibo.ManifiestoProcedenciaHuellaSHA256,
		AutoridadEfectiva:                 recibo.AutoridadEfectiva,
		ResueltoEnAutoritativo:            recibo.ResueltoEnAutoritativo,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             referenciaServicioContextoActorPrueba("aut_", "a"),
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  referenciaServicioContextoActorPrueba("ase_", "s"),
		SesionRef:                    referenciaServicioContextoActorPrueba("ses_", "e"),
		ControlSesionRef:             referenciaServicioContextoActorPrueba("cse_", "c"),
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    solicitudContexto.Cuenta.CuentaRef,
		CuentaOrdinariaRef:           solicitudContexto.Cuenta.CuentaRef,
		Superficie:                   superficie,
		MetodoObservado:              solicitudContexto.Cuenta.Metodo,
		GarantiaObservada:            solicitudContexto.Cuenta.Garantia,
		PoliticaGarantiaRef:          referenciaServicioContextoActorPrueba("pga_", "g"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
	}
	revalidador := &revalidadorVinculoAplicacionAdversarial{resultado: autenticacion}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidador,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorContextoAutorizacionV3Prueba{resultado: resultado}, solicitudContexto,
		&relojAutorizacionServicioPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear vinculo V2: %v", err)
	}
	referenciaMotivo := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: referenciaMotivo,
			Accion: sel.AccionConsultarPropias,
			Recurso: domain.RecursoAutorizable{
				Referencia: sel.PrefijoRecursoPropio + actor.PersonaRef, ModuloID: sel.ModuloSeleccion, Tipo: sel.TipoRecursoSolicitudesPropias,
				Ambitos: map[string]string{"ambito_ref": "seleccion"},
			},
			Finalidad: sel.FinalidadSolicitudesPropias,
			Correlacion: referenciaCorrelacionAplicacionPrueba(
				referenciaCorrelacionAutorizacionV2Prueba,
			),
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud V3: %v", err)
	}
	instantanea := instantaneaAutorizacionServicioPrueba(t)
	instantanea.AsignacionPerfil.PrincipalID = actor.Principal.ID
	instantanea.AsignacionPerfil.PerfilActivoRef = actor.PerfilActivoRef
	fuente := &fuenteAutorizacionServicioPrueba{instantanea: instantanea}
	concesiones := &registroConcesionesAutorizacionV3Prueba{}
	denegaciones := &registroDenegacionesAutorizacionV3Prueba{}
	motivos := &validadorMotivoAutorizacionV2Prueba{referencia: referenciaMotivo}
	servicio, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, concesiones, denegaciones, motivos,
		&relojAutorizacionServicioPrueba{ahora: ahora},
		&generadorAutorizacionServicioPrueba{referencia: "dec_0123456789abcdef0123456789abcdef"},
		app.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear servicio V3: %v", err)
	}
	return &entornoAutorizacionSolicitudV3Prueba{
		ahora: ahora, solicitud: solicitud, resultado: resultado, instantanea: instantanea, autenticacion: autenticacion,
		fuente: fuente, concesiones: concesiones, denegaciones: denegaciones,
		motivos: motivos, servicio: servicio,
	}
}

type opcionContexto func(*domain.InstantaneaContextoActor)
