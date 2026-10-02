package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/diagnostico"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func materialFirmaDesarrolloPrueba() ports.MaterialFirmaDocumento {
	return ports.MaterialFirmaDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:001",
		VersionExpediente: 7, Documento: "informe_definitivo", CatalogoRef: "vec.contratacion_temporal.circuito_firma:1",
		CatalogoHuella: strings.Repeat("c", 64), PasoRef: "vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1",
		PasoOrden: 1, Secuencia: 1, Resultado: ctdomain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta la fecha",
		ClaveIdempotencia: "clave-devolucion-00001"}
}

func TestPerfilFijoFirmaCustodiaDeclaradaAntesDePublicar(t *testing.T) {
	s, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for _, activa := range []bool{false, true} {
		i, err := instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(v.PrincipalID, v.PerfilActivoRef, s.reloj.Ahora(), activa)
		if err != nil {
			t.Fatal(err)
		}
		custodia := 0
		for _, c := range i.VersionRol.Concesiones {
			if c.Accion == docports.AccionCustodiarFirmado {
				custodia++
				if c.ModuloID != "documentos" || c.TipoRecurso != "documento_firmado" ||
					!slices.Equal(c.Finalidades, []string{docports.FinalidadCustodiarFirmado}) {
					t.Fatal("custodia con concesión divergente")
				}
			}
		}
		if activa && custodia != 1 || !activa && custodia != 0 || len(i.AsignacionPerfil.Ambitos) != 1 ||
			i.AsignacionPerfil.Ambitos[0].Clave != "organizacion_ref" ||
			!slices.Equal(i.AsignacionPerfil.Ambitos[0].Valores, []string{docports.OrganizacionRefV3}) {
			t.Fatal("la custodia amplió el ámbito o se activó sin configuración")
		}
	}
}

type sesionFirmaErrorPrueba struct{ err error }

func (s sesionFirmaErrorPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, s.err
}

type fuentePerfilFirmaCaidaPrueba struct {
	*autoridadAsignacionesContratacionTemporalDesarrolloPrueba
}

type fuentePerfilFirmaPDPIntermitentePrueba struct {
	*autoridadAsignacionesContratacionTemporalDesarrolloPrueba
	publicada  instantaneaPublicadaDesarrollo
	final      instantaneaPublicadaDesarrollo
	lecturas   int
	caidaFinal bool
}

func (f *fuentePerfilFirmaPDPIntermitentePrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	f.lecturas++
	if f.lecturas == 1 {
		return f.publicada, true, nil
	}
	if f.lecturas == 2 || f.caidaFinal {
		return instantaneaPublicadaDesarrollo{}, false, errors.New("fuente temporalmente caída")
	}
	return f.final, true, nil
}

type lectorFirmasNoEjecutadoPrueba struct{ llamadas int }

func (l *lectorFirmasNoEjecutadoPrueba) ConsultarFirmasAutorizadas(context.Context, ports.MaterialConsultaFirmasDocumento, ports.CapacidadConsultaFirmasDocumento) ([]ports.FirmaRegistrada, error) {
	l.llamadas++
	return nil, nil
}

func TestConsultaFirmasDocumentoClasificaCaidaPosteriorDelPDP(t *testing.T) {
	for _, caso := range []struct {
		nombre                 string
		revocada, caida, comun bool
		esperado               error
		lecturas               int
	}{
		{"fuente_caida_tras_primera_lectura", false, true, false, ports.ErrRegistroFirmaDocumentoNoDisponible, 3},
		{"revocacion_confirmada_en_relectura", true, false, false, ports.ErrFirmaDocumentoDenegada, 3},
		{"asignacion_vigente_y_fuente_PDP_caida", false, false, false, ports.ErrRegistroFirmaDocumentoNoDisponible, 3},
		{"configuracion_autorizador_comun_caida", false, false, true, ports.ErrRegistroFirmaDocumentoNoDisponible, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
			ahora := s.reloj.Ahora()
			perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
				[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
				func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
					return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora)
				})
			if err != nil || s.registrarPerfilFijoCTDesarrollo(perfil) != nil {
				t.Fatal("perfil no compuesto", err)
			}
			perfil.contextoEsperadoRegistrado, perfil.sesionOperativa = perfil.contexto.Resultado, proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}
			s.motivoFirmaDocumento = motivoFirmaDocumentoCTDesarrollo()
			autoridad := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
			publicada := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
			final := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
			if caso.revocada {
				final.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
				final.instantanea.AsignacionPerfil.RevocadaEn, final.instantanea.AsignacionPerfil.RevocadaPor, final.instantanea.AsignacionPerfil.RevocacionRef = ahora, "revocador:prueba", "revocacion:prueba"
			}
			fuente := &fuentePerfilFirmaPDPIntermitentePrueba{autoridadAsignacionesContratacionTemporalDesarrolloPrueba: autoridad, publicada: publicada, final: final, caidaFinal: caso.caida}
			s.autoridadAsignaciones = fuente
			autorizador := base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)
			if caso.comun {
				autorizador = new(autorizadorComunDesarrollo)
			}
			lector := new(lectorFirmasNoEjecutadoPrueba)
			firma := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s, autorizador: autorizador,
				postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{proveedorMaterialConsultaFirmasDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)}}, lector: lector, reloj: s.reloj}
			ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaConsultaFirmaDocumento)
			_, err = (registroFirmasDocumentoNominal{firma}).ConsultarFirmas(ctx, organizacionAltaContratacionTemporalDesarrollo, "expediente:ct:uno")
			if !errors.Is(err, caso.esperado) || fuente.lecturas != caso.lecturas || lector.llamadas != 0 || autoridad.preparadas != 0 || autoridad.publicadas != 0 {
				t.Fatalf("clasificación=%v lecturas=%d SQL=%d preparaciones=%d publicaciones=%d", err, fuente.lecturas, lector.llamadas, autoridad.preparadas, autoridad.publicadas)
			}
		})
	}
}

func (fuentePerfilFirmaCaidaPrueba) leerAsignacionPublicada(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) {
	return instantaneaPublicadaDesarrollo{}, false, errors.New("fuente de asignaciones no disponible")
}

func TestConsultaFirmasDocumentoFuenteCaidaNoSeConfundeConRevocacion(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	fijo, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, perfil, ahora)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(fijo) != nil {
		t.Fatal("perfil no compuesto", err)
	}
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	publicada := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(fijo.plantilla)
	publicada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	publicada.AsignacionPerfil.RevocadaEn = ahora
	publicada.AsignacionPerfil.RevocadaPor = "revocador:prueba"
	publicada.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	a.asignaciones = map[string]instantaneaPublicadaDesarrollo{fijo.perfilRef(): {instantanea: publicada, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	f := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{
		soporte: s, autorizador: base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo),
		postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{proveedorMaterialConsultaFirmasDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)}}, reloj: s.reloj}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaConsultaFirmaDocumento)
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:uno"}
	if _, err := f.AutorizarConsultaFirmasDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("revocación: %v", err)
	}
	s.autoridadAsignaciones = fuentePerfilFirmaCaidaPrueba{a}
	if _, err := f.AutorizarConsultaFirmasDocumento(ctx, m); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("fuente caída: %v", err)
	}
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("un fallo de fuente publicó permisos")
	}
}

func TestConsultaFirmasDocumentoSesionCaidaYRevocada(t *testing.T) {
	for _, ruta := range []string{httpinterno.RutaConsultaFirmaDocumento, httpinterno.RutaFirmaDocumento} {
		for _, caso := range []struct {
			nombre           string
			origen, esperado error
		}{
			{"dependencia_caida", errors.New("detalle interno de la fuente"), ports.ErrConsultaRRHHNoDisponible},
			{"revocada", dominiovec.ErrAutorizacionDenegada, ErrSeguridadComunDesarrolloDenegada},
		} {
			t.Run(ruta+"/"+caso.nombre, func(t *testing.T) {
				e := nuevaSesionConsultaPrueba(t)
				canal := e.contexto().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
				canal.ruta, canal.metodo = ruta, http.MethodPost
				ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
				origen := &diagnostico.FalloConsultaRRHH{Etapa: diagnostico.EtapaSesionRevalidador, Causa: caso.origen}
				clasificada := e.p.errorSesionConsultaComunicacionesExpediente(ctx, origen)
				if !errors.Is(clasificada, caso.esperado) {
					t.Fatalf("clasificación %v, esperada %v", clasificada, caso.esperado)
				}
				e.soporte.sesionOperativa = sesionFirmaErrorPrueba{clasificada}
				canal.contextoOperacion = &contextoOperacionCTDesarrollo{}
				ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
				_, err := e.soporte.contextoOperativoDesarrollo(ctx)
				if caso.nombre == "dependencia_caida" && !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
					caso.nombre == "revocada" && !errors.Is(err, ports.ErrAutorizacionDenegada) {
					t.Fatalf("el contexto operativo perdió la clasificación: %v", err)
				}
			})
		}
	}
}

func TestConsultaFirmasDocumentoPredicadoNoCruzaExpediente(t *testing.T) {
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:uno"}
	recurso, err := consultafirmas.RecursoConsultaFirmasDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	d := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionConsultarFirmasDocumento,
		Finalidad: ports.FinalidadFirmaDocumento, ReferenciaMotivo: motivoConsultaFirmasDocumentoCTDesarrollo(), Recurso: recurso}
	ctx := context.WithValue(context.Background(), claveConsultaFirmasDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, d) {
		t.Fatal("lectura exacta denegada")
	}
	for _, modificar := range []func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Referencia = "expediente:ct:otro" },
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Accion = ports.AccionFirmarDocumento },
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Finalidad = "otra_finalidad" },
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": "organizacion:ajena"}
		},
		func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"material_sha256": strings.Repeat("0", 64)}
		},
	} {
		otro := d
		modificar(&otro)
		if solicitudAutorizacionConsultaFirmasDocumentoCTDesarrolloValida(ctx, otro) {
			t.Fatal("lectura divergente admitida")
		}
	}
}

func TestConsultaFirmasDocumentoPerfilFijoConsumeSinPublicar(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	fijo, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, perfil, ahora)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(fijo) != nil {
		t.Fatal("perfil de firma no compuesto", err)
	}
	if fijo.contexto.Resultado.Contexto.Instantanea.CuentaRef != s.contexto.Resultado.Contexto.Instantanea.CuentaRef ||
		fijo.contexto.Resultado.Contexto.PersonaRef != s.contexto.Resultado.Contexto.PersonaRef || fijo.perfilRef() == s.contexto.Resultado.Contexto.PerfilActivoRef {
		t.Fatal("el perfil de firma cambia la persona o reutiliza el perfil dinámico")
	}
	fijo.contextoEsperadoRegistrado = fijo.contexto.Resultado
	fijo.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: fijo.contexto}
	s.motivoFirmaDocumento = motivoFirmaDocumentoCTDesarrollo()
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	a.asignaciones = map[string]instantaneaPublicadaDesarrollo{fijo.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(fijo.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	for _, ruta := range []string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento} {
		for _, exp := range []string{"expediente:ct:uno", "expediente:ct:dos"} {
			ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
			m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: exp}
			r, _ := consultafirmas.RecursoConsultaFirmasDocumento(m)
			correlacion, _ := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
			d := dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: fijo.contexto.Vinculo,
				Accion: ports.AccionConsultarFirmasDocumento, ReferenciaMotivo: motivoConsultaFirmasDocumentoCTDesarrollo(), Recurso: r,
				Finalidad: ports.FinalidadFirmaDocumento, Correlacion: correlacion}
			ctx = context.WithValue(ctx, claveConsultaFirmasDocumentoCTDesarrollo{}, m)
			ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, d)
			solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo).ExigirSolicitudLigadaV3(ctx, solicitud, fijo.contexto.Resultado); err != nil {
				t.Fatalf("lectura nominal %s %s: %v", ruta, exp, err)
			}
			// Una revocación publicada no se reconstruye desde la plantilla.
			publicada := a.asignaciones[fijo.perfilRef()]
			publicada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			a.asignaciones[fijo.perfilRef()] = publicada
			if _, ok := s.instantaneaParaContexto(ctx, ruta); ok {
				t.Fatal("asignación revocada consumida")
			}
			a.asignaciones[fijo.perfilRef()] = instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(fijo.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
		}
	}
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("se publicaron permisos durante la consulta")
	}
}

func TestPredicadoFirmaDocumentoLigadoAlMaterial(t *testing.T) {
	m := materialFirmaDesarrolloPrueba()
	recurso, err := ctapplication.RecursoFirmaDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionFirmarDocumento, Finalidad: ports.FinalidadFirmaDocumento,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: recurso}
	ctx := context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, datos) {
		t.Fatal("la solicitud exacta no se admite")
	}
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.Background(), datos) {
		t.Fatal("sin material ligado se admite")
	}
	otro := m
	otro.MotivoDevolucion = "Otro motivo"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, otro), datos) {
		t.Fatal("un material distinto reutiliza la solicitud")
	}
	cambiada := datos
	cambiada.Accion = "contratacion_temporal.seguimiento.cerrar"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, cambiada) {
		t.Fatal("otra acción admitida")
	}
	ajena := m
	ajena.OrganizacionRef = "organizacion:ajena"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, ajena), datos) {
		t.Fatal("organización ajena admitida")
	}
}

func TestFirmaDocumentoDesarrolloRechazaCertificadoDistintoDelCanal(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(perfil) != nil {
		t.Fatal("perfil de firma no compuesto", err)
	}
	perfil.contextoEsperadoRegistrado, perfil.sesionOperativa = perfil.contexto.Resultado, proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}
	m := materialFirmaDesarrolloPrueba()
	m.Resultado, m.MotivoDevolucion = ctdomain.ResultadoFirmaFirmado, ""
	m.OriginalHuella, m.FirmadoHuella = strings.Repeat("a", 64), strings.Repeat("b", 64)
	m.CertificadoHuella, m.FirmanteRef = strings.Repeat("c", 64), "ref:"+strings.Repeat("c", 64)
	m.PoliticaVerificacion, m.RevocacionEstado, m.SelloTiempoEstado = ports.PoliticaVerificacionFirma, "vigente", "no_presente"
	if m.Validar() != nil || m.CertificadoHuella == s.certificadoSHA256 {
		t.Fatal("material de certificado ajeno inválido")
	}
	f := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s,
		autorizador: base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo),
		postgresql:  dependenciasPostgreSQLContratacionTemporalDesarrollo{proveedorMaterialFirmaDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)}}, reloj: s.reloj}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
	if _, err := f.AutorizarFirmaDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("certificado ajeno admitido: %v", err)
	}
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("el certificado ajeno alcanzó la autorización V3")
	}
}

type relojFirmaAvanzaPrueba struct {
	instantes []time.Time
	lecturas  int
}

func (r *relojFirmaAvanzaPrueba) Ahora() time.Time {
	i := r.lecturas
	r.lecturas++
	return r.instantes[i]
}

func TestFirmaDocumentoRevalidaCertificadoTrasEsperaPDP(t *testing.T) {
	inicio := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	vence := inicio.Add(time.Second)
	reloj := &relojFirmaAvanzaPrueba{instantes: []time.Time{inicio, vence}}
	firma := firmaDocumentoCTDesarrollo{reloj: reloj}
	canal := capacidadConsultaContratacionTemporalDesarrollo{
		certificadoVerificadoEn: inicio.Add(-time.Second), certificadoValidoHasta: vence,
	}
	if ahora, vigente := firma.certificadoVigenteFirmaDocumentoCTDesarrollo(canal); !vigente || !ahora.Equal(inicio) {
		t.Fatal("certificado vigente rechazado antes del PDP")
	}
	if ahora, vigente := firma.certificadoVigenteFirmaDocumentoCTDesarrollo(canal); vigente || !ahora.Equal(vence) {
		t.Fatal("certificado caducado aceptado tras el PDP")
	}
	if reloj.lecturas != 2 {
		t.Fatalf("se esperaban dos lecturas del reloj: %d", reloj.lecturas)
	}
}

func TestFirmaDocumentoCapacidadEmitidaDurantePDPUsaHoraFinal(t *testing.T) {
	inicio := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	final := inicio.Add(500 * time.Millisecond)
	emitida := inicio.Add(250 * time.Millisecond)
	expira := inicio.Add(time.Second)
	reloj := &relojFirmaAvanzaPrueba{instantes: []time.Time{inicio, final}}
	firma := firmaDocumentoCTDesarrollo{reloj: reloj}
	canal := capacidadConsultaContratacionTemporalDesarrollo{
		certificadoVerificadoEn: inicio.Add(-time.Second), certificadoValidoHasta: expira,
	}
	if _, vigente := firma.certificadoVigenteFirmaDocumentoCTDesarrollo(canal); !vigente {
		t.Fatal("certificado vigente rechazado antes del PDP")
	}
	if capacidadFirmaDocumentoCTDesarrolloVigenteEn(emitida, expira, inicio) {
		t.Fatal("la hora inicial admitió una capacidad aún no emitida")
	}
	ahoraFinal, vigente := firma.certificadoVigenteFirmaDocumentoCTDesarrollo(canal)
	if !vigente || !capacidadFirmaDocumentoCTDesarrolloVigenteEn(emitida, expira, ahoraFinal) || reloj.lecturas != 2 {
		t.Fatal("la hora final rechazó certificado y capacidad vigentes")
	}
}

func TestConsultaFirmasDocumentoSinRelojDeniega(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	firma := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{
		soporte: s, autorizador: base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo),
		postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{
			proveedorMaterialConsultaFirmasDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)},
	}}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaConsultaFirmaDocumento)
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:uno"}
	if _, err := firma.AutorizarConsultaFirmasDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("consulta sin reloj admitida: %v", err)
	}
}

// El catálogo de ejemplo conserva perfiles de cargo opacos, pero el único
// perfil publicado por la composición antigua es el genérico de firma. Su
// asignación vigente no le convierte en Jefatura, Diputación ni Secretaría.
func TestFirmaDocumentoDesarrolloSinPerfilNominalDeniegaPaso(t *testing.T) {
	s, base, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoFirmaCTDesarrollo,
		[]string{httpinterno.RutaFirmaDocumento, httpinterno.RutaConsultaFirmaDocumento},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaPerfilFijoFirmaDocumentoCTDesarrollo(actor, ref, ahora)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(perfil) != nil {
		t.Fatal("perfil de firma no compuesto", err)
	}
	perfil.contextoEsperadoRegistrado = perfil.contexto.Resultado
	perfil.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}
	a := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	publicada := instantaneaPublicadaDesarrollo{instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla),
		actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}
	a.asignaciones = map[string]instantaneaPublicadaDesarrollo{perfil.perfilRef(): publicada}
	compuestas, err := nuevasReglasEjemploDesarrollo(configuracionCircuitoFirmaPrueba(rutaCircuitoFirmaCTEjemploPrueba), nil, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	circuito, err := compuestas.circuitoFirmaCT.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	f := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: s,
		autorizador: base.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo),
		postgresql: dependenciasPostgreSQLContratacionTemporalDesarrollo{
			proveedorMaterialFirmaDocumento: new(proveedorMaterialAltaContratacionTemporalDesarrollo)}},
		circuito: compuestas.circuitoFirmaCT, reloj: s.reloj}
	m := materialFirmaDesarrolloPrueba()
	m.CatalogoRef = circuito.CatalogoID + ":1"
	m.CatalogoHuella = circuito.HuellaCatalogo
	m.PasoRef = circuito.Documentos[0].Pasos[0].Referencia
	if m.Validar() != nil || perfil.perfilRef() == circuito.Documentos[0].Pasos[0].PerfilRef {
		t.Fatal("el escenario no separa perfil genérico y paso del catálogo")
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaFirmaDocumento)
	if _, err := f.AutorizarFirmaDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("perfil genérico admitido: %v", err)
	}
	if a.preparadas != 0 || a.publicadas != 0 {
		t.Fatal("la escritura publicó una asignación para superar el perfil genérico")
	}
	canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	canal.certificadoValidoHasta = ahora
	ctxCaducado := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, canal)
	if _, err := f.AutorizarFirmaDocumento(ctxCaducado, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("certificado caducado admitido: %v", err)
	}

	// La revocación se lee de la fuente central y no revive por la
	// plantilla mantenida en memoria.
	revocada := publicada
	revocada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	revocada.instantanea.AsignacionPerfil.RevocadaEn = ahora
	revocada.instantanea.AsignacionPerfil.RevocadaPor = "revocador:prueba"
	revocada.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	a.asignaciones[perfil.perfilRef()] = revocada
	if _, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil); estado != perfilFijoConsumoDenegado {
		t.Fatalf("asignación revocada consumida: %v", estado)
	}
	if _, err := f.AutorizarFirmaDocumento(ctx, m); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("perfil revocado admitido: %v", err)
	}
	a.asignaciones[perfil.perfilRef()] = publicada
	unidadAjena := publicada
	unidadAjena.instantanea.AsignacionPerfil.Ambitos = append(
		unidadAjena.instantanea.AsignacionPerfil.Ambitos,
		dominiovec.AmbitoPerfil{Clave: "unidad_ref", Valores: []string{"unidad:ajena"}},
	)
	a.asignaciones[perfil.perfilRef()] = unidadAjena
	if _, estado := s.consumirPerfilFijoCTDesarrolloConEstado(ctx, perfil); estado != perfilFijoConsumoDenegado {
		t.Fatalf("ámbito de unidad ajena consumido: %v", estado)
	}
	a.asignaciones[perfil.perfilRef()] = publicada

	ajena := m
	ajena.OrganizacionRef = "organizacion:ajena"
	if _, err := f.AutorizarFirmaDocumento(ctx, ajena); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("organización ajena admitida: %v", err)
	}
	cambiado := m
	cambiado.CatalogoHuella = strings.Repeat("0", 64)
	if perfilPasoFirmaDocumentoCTDesarrolloCoincide(cambiado, circuito, circuito.Documentos[0].Pasos[0].PerfilRef) {
		t.Fatal("material de otro catálogo aceptado por el paso")
	}
	if _, err := f.AutorizarFirmaDocumento(ctx, cambiado); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) {
		t.Fatalf("catálogo cambiado admitido: %v", err)
	}
	pasoAjeno := m
	pasoAjeno.PasoRef = circuito.Documentos[0].Pasos[1].Referencia
	if perfilPasoFirmaDocumentoCTDesarrolloCoincide(pasoAjeno, circuito, circuito.Documentos[0].Pasos[0].PerfilRef) {
		t.Fatal("referencia de otro paso aceptada")
	}
}

func TestFirmaDocumentoCTApagadaNoCompone(t *testing.T) {
	f, err := nuevaFirmaDocumentoCTDesarrollo(config.Config{}, nil, relojContratacionTemporalDesarrollo{}, nil)
	if f != nil || err != nil {
		t.Fatalf("selector apagado: %v %v", f, err)
	}
	var nula *firmaDocumentoCTDesarrollo
	if rutas, err := nula.rutas(config.Config{}, nil); rutas != nil || err != nil {
		t.Fatal("sin firma no hay rutas")
	}
	if _, err := nula.ResolverOrganizacionFirmaDocumento(context.Background()); err == nil {
		t.Fatal("canal sin composición admitido")
	}
	if _, err := nula.AutorizarFirmaDocumento(context.Background(), materialFirmaDesarrolloPrueba()); err == nil {
		t.Fatal("autorización sin composición")
	}
	if _, err := nuevaFirmaDocumentoCTDesarrollo(config.Config{CTFirmaRegistroEnabled: "si"}, nil, relojContratacionTemporalDesarrollo{}, nil); err == nil {
		t.Fatal("selector inválido admitido")
	}
}

func TestFirmaDocumentoCTRutasYConsumidor(t *testing.T) {
	if !rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaFirmaDocumento) || !rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaConsultaFirmaDocumento) ||
		rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaFirmaDocumento+"/") {
		t.Fatal("rutas de firma")
	}
	if !rutaMutacionDurableContratacionTemporalDesarrollo(httpinterno.RutaFirmaDocumento) ||
		rutaMutacionDurableContratacionTemporalDesarrollo(httpinterno.RutaConsultaFirmaDocumento) ||
		!rutaContextoAutorizacionContratacionTemporalDesarrollo(httpinterno.RutaFirmaDocumento) {
		t.Fatal("la escritura debe ser mutación durable y la consulta no")
	}
	d := descriptorMaterialFirmaDocumentoCTDesarrollo()
	if d.Audiencia != ports.AudienciaFirmaDocumentoV3 || !slices.Contains(audienciasConsumoGobiernoCTDesarrollo(), d.Audiencia) {
		t.Fatal("la audiencia de firma no está en la lista única")
	}
	if dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivoFirmaDocumentoCTDesarrollo()) == false {
		t.Fatal("motivo de firma inválido")
	}
}

type servicioFirmaConsultaCerradaPrueba struct{ consultas int }

func (*servicioFirmaConsultaCerradaPrueba) Firmar(context.Context, ctapplication.SolicitudFirmaDocumento) (ctapplication.ResultadoFirmaDocumento, error) {
	return ctapplication.ResultadoFirmaDocumento{}, ports.ErrFirmaDocumentoDenegada
}

func (s *servicioFirmaConsultaCerradaPrueba) Consultar(context.Context, string, string) (ctapplication.EstadoFirmasExpediente, error) {
	s.consultas++
	return ctapplication.EstadoFirmasExpediente{}, nil
}

func (*servicioFirmaConsultaCerradaPrueba) VerificacionDisponible() bool { return false }

func TestFirmaDocumentoCTConsultaCerradaAntesDeLeerRegistro(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	firma := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte}}
	registro := &servicioFirmaConsultaCerradaPrueba{}
	h, err := httpinterno.NuevoManejadorFirmaDocumento(firma, registro)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, httpinterno.RutaConsultaFirmaDocumento,
		strings.NewReader(`{"expediente_ref":"expediente:ct:001"}`))
	peticion.Header.Set("Content-Type", "application/json")
	peticion = peticion.WithContext(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaConsultaFirmaDocumento))
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || !strings.Contains(respuesta.Body.String(), `"codigo":"acceso_denegado"`) || registro.consultas != 0 {
		t.Fatalf("consulta sin autoridad nominal: estado=%d cuerpo=%s lecturas=%d", respuesta.Code, respuesta.Body, registro.consultas)
	}
	organizacion, err := firma.ResolverOrganizacionFirmaDocumento(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaFirmaDocumento))
	if err != nil || organizacion != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("registro de firma nominal cerrado por error: organizacion=%q error=%v", organizacion, err)
	}
}
