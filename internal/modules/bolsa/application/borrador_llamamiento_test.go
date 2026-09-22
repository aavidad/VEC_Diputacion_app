package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

var instanteBorradorLlamamientoPrueba = time.Date(2026, 9, 20, 10, 30, 0, 123456000, time.UTC)

func TestNuevoServicioBorradorLlamamientoFallaCerradoSinDependencias(t *testing.T) {
	if servicio, err := NuevoServicioBorradorLlamamiento(nil, nil, nil, nil); err == nil || servicio != nil {
		t.Fatalf("constructor acepto dependencias ausentes: %#v %v", servicio, err)
	}
}

func TestCrearBorradorLlamamientoComponeAutoridadYRecuperaIdempotencia(t *testing.T) {
	e := nuevoEscenarioBorradorLlamamiento(t)
	primero, err := e.servicio.Crear(context.Background(), e.crear)
	if err != nil || primero.ReintentoIdempotente || primero.Validar() != nil {
		t.Fatalf("crear borrador: recibo=%#v err=%v", primero, err)
	}
	segundo, err := e.servicio.Crear(context.Background(), e.crear)
	if err != nil || !segundo.ReintentoIdempotente || segundo.Referencia != primero.Referencia || segundo.HuellaComandoSHA256 != primero.HuellaComandoSHA256 {
		t.Fatalf("replay no recupero recibo original: %#v %v", segundo, err)
	}
	comando, datos := e.transaccion.ultimo, datosSolicitudBorradorPrueba(t, e.transaccion.ultimo.SolicitudAutorizacion)
	if e.transaccion.llamadas != 2 || e.autorizador.llamadas != 2 || datos.Accion != puertosbolsa.AccionCrearBorradorLlamamientoInterno || datos.Finalidad != puertosbolsa.FinalidadCrearBorradorLlamamientoInterno || datos.Recurso.Referencia != primero.Borrador.Referencia() || datos.Recurso.ModuloID != puertosbolsa.ModuloBorradorLlamamiento || datos.Recurso.Tipo != puertosbolsa.TipoRecursoBorradorLlamamiento || datos.Recurso.Ambitos["unidad_ref"] != e.contexto.UnidadRef || datos.Recurso.Ambitos["ambito_ref"] != e.contexto.AmbitoRef || comando.Material.ValidarEstructura() != nil {
		t.Fatalf("contrato de autoridad/material inesperado: %#v", datos)
	}
}

func TestCrearBorradorLlamamientoPropagaConflictoDeClave(t *testing.T) {
	e := nuevoEscenarioBorradorLlamamiento(t)
	e.transaccion.err = puertosbolsa.ErrClaveBorradorLlamamientoReutilizada
	if _, err := e.servicio.Crear(context.Background(), e.crear); !errors.Is(err, puertosbolsa.ErrClaveBorradorLlamamientoReutilizada) {
		t.Fatalf("conflicto de idempotencia perdido: %v", err)
	}
}

func TestConsultarBorradorLlamamientoDevuelveReciboDurableYDeniegaAjeno(t *testing.T) {
	e := nuevoEscenarioBorradorLlamamiento(t)
	creado, err := e.servicio.Crear(context.Background(), e.crear)
	if err != nil {
		t.Fatal(err)
	}
	e.lector.recibo = creado
	consulta := solicitudConsultarBorradorPrueba(t, e.resultado, e.vinculo, creado.Borrador.Referencia())
	if obtenido, err := e.servicio.Consultar(context.Background(), consulta); err != nil || obtenido != creado || e.lector.llamadas != 1 {
		t.Fatalf("consulta no devolvio recibo durable: %#v %v", obtenido, err)
	}
	e.lector.recibo = reciboBorradorPrueba(t, "per_aaaaaaaaaaaaaaaaaaaaaa", e.contexto, creado.HuellaComandoSHA256)
	if _, err := e.servicio.Consultar(context.Background(), consulta); !errors.Is(err, ErrServicioBorradorLlamamientoInvalido) {
		t.Fatalf("lectura ajena aceptada: %v", err)
	}
	e.lector.recibo = creado
	e.lector.mutar = func(r *puertosbolsa.ReciboBorradorLlamamiento) { r.HuellaComandoSHA256 = strings.Repeat("a", 64) }
	if _, err := e.servicio.Consultar(context.Background(), consulta); !errors.Is(err, ErrServicioBorradorLlamamientoInvalido) {
		t.Fatalf("recibo inconsistente aceptado: %v", err)
	}
}

func TestBorradorLlamamientoDeniegaCruceDeVinculoYSinAmbitoAntesDePuertos(t *testing.T) {
	e := nuevoEscenarioBorradorLlamamiento(t)
	_, vinculoAjeno, err := pruebas.NuevoContextoRegistradoYVinculoV2(instanteBorradorLlamamientoPrueba, "per_bbbbbbbbbbbbbbbbbbbbbb", "prf_bbbbbbbbbbbbbbbbbbbbbb", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	cruzada := e.crear
	cruzada.Vinculo = vinculoAjeno
	if _, err := e.servicio.Crear(context.Background(), cruzada); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || e.transaccion.llamadas != 0 || e.autorizador.llamadas != 0 {
		t.Fatalf("cruce alcanzo puertos: %v tx=%d auth=%d", err, e.transaccion.llamadas, e.autorizador.llamadas)
	}
	e.contextualizador.contexto = puertosbolsa.ContextoBorradorLlamamientoResuelto{}
	if _, err := e.servicio.Crear(context.Background(), e.crear); !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || e.transaccion.llamadas != 0 || e.autorizador.llamadas != 0 {
		t.Fatalf("contexto no verificable no quedo neutral: %v tx=%d auth=%d", err, e.transaccion.llamadas, e.autorizador.llamadas)
	}
}

func TestBorradorLlamamientoDeniegaMaterialDeOtroContexto(t *testing.T) {
	e := nuevoEscenarioBorradorLlamamiento(t)
	ajeno, _, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		instanteBorradorLlamamientoPrueba, "per_0123456789abcdefghijkl", "prf_bbbbbbbbbbbbbbbbbbbbbb",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	e.autorizador.resultadoMaterial = &ajeno
	if _, err := e.servicio.Crear(context.Background(), e.crear); !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || e.transaccion.llamadas != 0 {
		t.Fatalf("material de otro contexto no quedo neutral: %v tx=%d", err, e.transaccion.llamadas)
	}
}

func TestBorradorLlamamientoSoloPropagaDenegacionPositivaDeDependencias(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*escenarioBorradorLlamamiento)
		ejecutar func(*escenarioBorradorLlamamiento) error
	}{
		{
			nombre:   "resolutor deniega",
			preparar: func(e *escenarioBorradorLlamamiento) { e.contextualizador.err = dominiovec.ErrAutorizacionDenegada },
			ejecutar: func(e *escenarioBorradorLlamamiento) error {
				_, err := e.servicio.Crear(context.Background(), e.crear)
				return err
			},
		},
		{
			nombre:   "emisor deniega",
			preparar: func(e *escenarioBorradorLlamamiento) { e.autorizador.err = dominiovec.ErrAutorizacionDenegada },
			ejecutar: func(e *escenarioBorradorLlamamiento) error {
				_, err := e.servicio.Consultar(context.Background(), solicitudConsultarBorradorPrueba(t, e.resultado, e.vinculo, "borrador-llamamiento:alta:"+strings.Repeat("a", 64)))
				return err
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioBorradorLlamamiento(t)
			caso.preparar(e)
			err := caso.ejecutar(e)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) {
				t.Fatalf("la denegacion positiva no se conservo: %v", err)
			}
		})
	}
}

func TestBorradorLlamamientoNormalizaFalloOCancelacionDeDependencias(t *testing.T) {
	fallos := []error{errors.New("caida del proveedor"), context.Canceled, context.DeadlineExceeded, puertosbolsa.ErrSolicitudBorradorLlamamientoInvalida}
	for _, fallo := range fallos {
		t.Run(fallo.Error(), func(t *testing.T) {
			e := nuevoEscenarioBorradorLlamamiento(t)
			e.contextualizador.err = fallo
			if _, err := e.servicio.Crear(context.Background(), e.crear); !erroresDependenciaBorradorLlamamientoNormalizados(err, fallo) || e.autorizador.llamadas != 0 || e.transaccion.llamadas != 0 {
				t.Fatalf("fallo del resolutor no quedo neutral: %v auth=%d tx=%d", err, e.autorizador.llamadas, e.transaccion.llamadas)
			}

			e = nuevoEscenarioBorradorLlamamiento(t)
			e.autorizador.err = fallo
			if _, err := e.servicio.Consultar(context.Background(), solicitudConsultarBorradorPrueba(t, e.resultado, e.vinculo, "borrador-llamamiento:alta:"+strings.Repeat("a", 64))); !erroresDependenciaBorradorLlamamientoNormalizados(err, fallo) || e.lector.llamadas != 0 {
				t.Fatalf("fallo del emisor no quedo neutral: %v lecturas=%d", err, e.lector.llamadas)
			}

			e = nuevoEscenarioBorradorLlamamiento(t)
			e.autorizador.errorExportador = fallo
			if _, err := e.servicio.Crear(context.Background(), e.crear); !erroresDependenciaBorradorLlamamientoNormalizados(err, fallo) || e.transaccion.llamadas != 0 {
				t.Fatalf("fallo del exportador no quedo neutral: %v tx=%d", err, e.transaccion.llamadas)
			}
		})
	}
}

func erroresDependenciaBorradorLlamamientoNormalizados(err, original error) bool {
	return errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) &&
		!errors.Is(err, dominiovec.ErrAutorizacionDenegada) &&
		!errors.Is(err, original)
}

type escenarioBorradorLlamamiento struct {
	servicio         *ServicioBorradorLlamamiento
	crear            puertosbolsa.SolicitudCrearBorradorLlamamiento
	resultado        dominiovec.ResultadoContextoActorRegistradoV2
	vinculo          dominiovec.VinculoAutenticacionActorV2
	contexto         puertosbolsa.ContextoBorradorLlamamientoResuelto
	contextualizador *contextualizadorBorradorPrueba
	autorizador      *autorizadorBorradorPrueba
	transaccion      *transaccionBorradorPrueba
	lector           *lectorBorradorPrueba
}

func nuevoEscenarioBorradorLlamamiento(t *testing.T) *escenarioBorradorLlamamiento {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(instanteBorradorLlamamientoPrueba, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	contexto := puertosbolsa.ContextoBorradorLlamamientoResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}
	c, a, tx, l := &contextualizadorBorradorPrueba{contexto: contexto}, &autorizadorBorradorPrueba{t: t, instante: instanteBorradorLlamamientoPrueba}, &transaccionBorradorPrueba{}, &lectorBorradorPrueba{}
	servicio, err := NuevoServicioBorradorLlamamiento(c, a, tx, l)
	if err != nil {
		t.Fatal(err)
	}
	return &escenarioBorradorLlamamiento{servicio, puertosbolsa.SolicitudCrearBorradorLlamamiento{Vinculo: vinculo, ResultadoContexto: resultado, ClaveIdempotencia: "clave-prueba-0001", Contenido: dominiobolsa.ContenidoBorradorLlamamiento{Resumen: "Preparar borrador interno"}, Correlacion: correlacionBorradorPrueba(t), Motivo: motivoBorradorPrueba()}, resultado, vinculo, contexto, c, a, tx, l}
}

type contextualizadorBorradorPrueba struct {
	contexto puertosbolsa.ContextoBorradorLlamamientoResuelto
	llamadas int
	err      error
}

func (c *contextualizadorBorradorPrueba) ResolverContextoBorradorLlamamiento(context.Context, dominiovec.ContextoActor) (puertosbolsa.ContextoBorradorLlamamientoResuelto, error) {
	c.llamadas++
	return c.contexto, c.err
}

type transaccionBorradorPrueba struct {
	llamadas int
	ultimo   puertosbolsa.ComandoCrearBorradorLlamamiento
	recibo   puertosbolsa.ReciboBorradorLlamamiento
	err      error
}

func (t *transaccionBorradorPrueba) CrearBorradorLlamamiento(_ context.Context, c puertosbolsa.ComandoCrearBorradorLlamamiento) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	t.llamadas++
	t.ultimo = c
	if t.err != nil {
		return puertosbolsa.ReciboBorradorLlamamiento{}, t.err
	}
	if t.recibo.Referencia == "" {
		t.recibo = reciboDesdeComandoBorradorPrueba(c, false)
		return t.recibo, nil
	}
	r := t.recibo
	r.ReintentoIdempotente = true
	return r, nil
}

type lectorBorradorPrueba struct {
	llamadas int
	recibo   puertosbolsa.ReciboBorradorLlamamiento
	mutar    func(*puertosbolsa.ReciboBorradorLlamamiento)
}

func (l *lectorBorradorPrueba) ObtenerBorradorLlamamiento(_ context.Context, _ string, _ string, _ string, _ string, _ dominiovec.SolicitudAutorizacionLigadaV3, _ dominiovec.DecisionAutorizacionLigadaV3, _ puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, _ puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (puertosbolsa.ReciboBorradorLlamamiento, error) {
	l.llamadas++
	r := l.recibo
	if l.mutar != nil {
		l.mutar(&r)
	}
	return r, nil
}
func reciboDesdeComandoBorradorPrueba(c puertosbolsa.ComandoCrearBorradorLlamamiento, replay bool) puertosbolsa.ReciboBorradorLlamamiento {
	return puertosbolsa.ReciboBorradorLlamamiento{Referencia: "recibo:" + strings.Repeat("a", 64), Borrador: c.Borrador, HuellaComandoSHA256: c.HuellaComandoSHA256, ReintentoIdempotente: replay, RegistradoEn: instanteBorradorLlamamientoPrueba}
}
func reciboBorradorPrueba(t *testing.T, propietario string, contexto puertosbolsa.ContextoBorradorLlamamientoResuelto, huella string) puertosbolsa.ReciboBorradorLlamamiento {
	t.Helper()
	b, err := dominiobolsa.NuevoBorradorLlamamiento("borrador-llamamiento:alta:"+huella, propietario, contexto.UnidadRef, contexto.AmbitoRef, dominiobolsa.ContenidoBorradorLlamamiento{Resumen: "Preparar borrador interno"})
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.ReciboBorradorLlamamiento{Referencia: "recibo:" + strings.Repeat("a", 64), Borrador: b, HuellaComandoSHA256: huella, RegistradoEn: instanteBorradorLlamamientoPrueba}
}

type generadorCorrelacionBorradorPrueba struct{}

func (generadorCorrelacionBorradorPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}
func correlacionBorradorPrueba(t *testing.T) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	r, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionBorradorPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func motivoBorradorPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}
}
func solicitudConsultarBorradorPrueba(t *testing.T, resultado dominiovec.ResultadoContextoActorRegistradoV2, vinculo dominiovec.VinculoAutenticacionActorV2, referencia string) puertosbolsa.SolicitudConsultarBorradorLlamamiento {
	return puertosbolsa.SolicitudConsultarBorradorLlamamiento{Vinculo: vinculo, ResultadoContexto: resultado, BorradorRef: referencia, Correlacion: correlacionBorradorPrueba(t), Motivo: motivoBorradorPrueba()}
}
func datosSolicitudBorradorPrueba(t *testing.T, s dominiovec.SolicitudAutorizacionLigadaV3) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
	t.Helper()
	d, err := s.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

type registroConcesionBorradorPrueba struct{ instante time.Time }

func (r registroConcesionBorradorPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.instante, nil
}

type exportadorBorradorPrueba struct {
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	err      error
}

func (e exportadorBorradorPrueba) ExportarMaterialParaConsumidor() (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.material, e.err
}
func (exportadorBorradorPrueba) String() string         { return "[EXPORTADOR-BORRADOR-PRUEBA]" }
func (e exportadorBorradorPrueba) LogValue() slog.Value { return slog.StringValue(e.String()) }

type autorizadorBorradorPrueba struct {
	t                 *testing.T
	instante          time.Time
	llamadas          int
	resultadoMaterial *dominiovec.ResultadoContextoActorRegistradoV2
	err               error
	errorExportador   error
}

func (a *autorizadorBorradorPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.t.Helper()
	a.llamadas++
	if a.err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, a.err
	}
	datos := datosSolicitudBorradorPrueba(a.t, solicitud)
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		a.t.Fatal(err)
	}
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(datos.Recurso.Ambitos))
	for clave, valor := range datos.Recurso.Ambitos {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{valor}})
	}
	rol := dominiovec.VersionRol{RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico bolsa", Estado: dominiovec.EstadoVersionRolPublicada, Concesiones: []dominiovec.ConcesionRol{{Accion: datos.Accion, ModuloID: datos.Recurso.ModuloID, TipoRecurso: datos.Recurso.Tipo, Finalidades: []string{datos.Finalidad}, GarantiaMinima: dominiovec.AuthAssuranceSubstantial}}, PublicadaPor: "seguridad", PublicadaEn: a.instante.Add(-time.Hour)}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		a.t.Fatal(err)
	}
	instantanea := dominiovec.InstantaneaAutorizacion{AsignacionPerfil: dominiovec.AsignacionPerfil{AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: vinculo.PerfilActivoRef, PrincipalID: vinculo.PrincipalID, VersionRolRef: rol.Referencia(), Estado: dominiovec.EstadoAsignacionPerfilActiva, Ambitos: ambitos, VigenteDesde: a.instante.Add(-time.Hour), VigenteHasta: a.instante.Add(time.Hour), EmitidaPor: "seguridad", EmitidaEn: a.instante.Add(-time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "seguridad", ActualizadoEn: a.instante.Add(-time.Hour)}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo}
	evidencia, err := dominiovec.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, "decision:borrador:prueba", a.instante, a.instante.Add(time.Minute))
	if err != nil {
		a.t.Fatal(err)
	}
	decision, err := dominiovec.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		a.t.Fatal(err)
	}
	orden, err := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, datos.ReferenciaMotivo, resultado)
	if err != nil {
		a.t.Fatal(err)
	}
	confirmacion, err := puertosvec.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Background(), registroConcesionBorradorPrueba{a.instante}, orden)
	if err != nil {
		a.t.Fatal(err)
	}
	resultadoMaterial := resultado
	if a.resultadoMaterial != nil {
		resultadoMaterial = *a.resultadoMaterial
	}
	return decision, confirmacion, exportadorBorradorPrueba{material: materialBorradorPrueba(a.t, decision, resultadoMaterial, datos, a.instante), err: a.errorExportador}, nil
}
func materialBorradorPrueba(t *testing.T, decision dominiovec.DecisionAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2, datos dominiovec.DatosSolicitudAutorizacionLigadaV3, instante time.Time) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	dh, err := dominiovec.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	mh, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	rh, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	audiencia := puertosbolsa.AudienciaCrearBorradorLlamamientoInterno
	if datos.Accion == puertosbolsa.AccionConsultarBorradorLlamamientoInterno {
		audiencia = puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno
	} else if datos.Accion == puertosbolsa.AccionCambiarSituacionParticipacion {
		audiencia = puertosbolsa.AudienciaCambiarSituacionParticipacion
	} else if datos.Accion == puertosbolsa.AccionRegistrarDatosContactoParticipacion {
		audiencia = puertosbolsa.AudienciaRegistrarDatosContactoParticipacion
	} else if datos.Accion == puertosbolsa.AccionRegistrarContactoParticipacion {
		audiencia = puertosbolsa.AudienciaRegistrarContactoParticipacion
	} else if datos.Accion == puertosbolsa.AccionConsultarContactoParticipacion {
		audiencia = puertosbolsa.AudienciaConsultarContactoParticipacion
	}
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:borrador:prueba", dh, mh, resultado.RegistroContextoRef, resultado.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh, audiencia, instante, instante.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	dc, err := dominiovec.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	mc, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(privada.Public())
	if err != nil {
		t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{'b'}, puertosvec.TamanoMinimoCapacidadCanonicaV3), resumen, dc, mc, resultado.RepresentacionCanonica, resultado.Contexto.Instantanea.PersonaVersion, resultado.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return material
}
