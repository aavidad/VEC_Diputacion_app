package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var contenidoOriginalIngestaPrueba = strings.Repeat("A", 100)

type repositorioIngestaPrueba struct {
	ports.RepositorioIngestaMayan
	plan             ports.PlanIngestaMayan
	despachos        int
	resoluciones     int
	reservaRecibida  ports.ReservaRecursoDocumental
	despachoNuevo    bool
	ultimaResolucion ports.OrdenCambiarIngestaMayan
}

func (r *repositorioIngestaPrueba) ReservarRecurso(_ context.Context, o ports.OrdenReservarRecursoMayan) (ports.ReservaRecursoDocumental, error) {
	r.reservaRecibida = o.Reserva
	respuesta := o.Reserva
	respuesta.Estado = ports.EstadoRecursoReservado
	respuesta.Version = 1
	respuesta.CreadaEn = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	respuesta.ExpiraEn = respuesta.CreadaEn.Add(10 * time.Minute)
	return respuesta, nil
}

func (r *repositorioIngestaPrueba) ResolverIngesta(_ context.Context, o ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, error) {
	r.resoluciones++
	r.ultimaResolucion = o
	r.plan.Estado = o.Estado
	r.plan.Version++
	return r.plan, nil
}

func (r *repositorioIngestaPrueba) ConsultarIngesta(_ context.Context, _ ports.OrdenConsultarIngestaMayan) (ports.PlanIngestaMayan, error) {
	return r.plan, nil
}
func (r *repositorioIngestaPrueba) DespacharIngesta(_ context.Context, _ ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, bool, error) {
	r.despachos++
	if r.despachoNuevo {
		r.plan.Estado = ports.EstadoIngestaDespachada
		r.plan.Version++
		return r.plan, true, nil
	}
	return ports.PlanIngestaMayan{}, false, nil
}

type autoridadIngestaPrueba struct {
	t         *testing.T
	ahora     time.Time
	principal string
}

func (a autoridadIngestaPrueba) AutorizarIngestaMayan(_ context.Context, accion string, preimagen []byte, planRef, expedienteRef string) (ports.AutorizacionV3, error) {
	principal := a.principal
	if principal == "" {
		principal = "per:0001"
	}
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, accion, planRef, preimagen, a.ahora),
		Accion: accion, Finalidad: ports.FinalidadIngestaMayan, RecursoRef: planRef, AmbitoRef: expedienteRef,
		PrincipalID: principal, PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}, nil
}

func TestReservaRecursoFijaPersonaYContextoV2AntesDeS3(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: escenario.resultado.Contexto.PersonaRef}, Reloj: relojExternaPrueba{escenario.ahora}}
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	reserva, err := s.ReservarRecurso(context.Background(), ref("a"), ref("b"), strings.Repeat("c", 64),
		escenario.resultado, datos.VinculoAutenticacionActor)
	if err != nil || reserva.RecursoRef != ref("a") || repo.reservaRecibida.PersonaRef != escenario.resultado.Contexto.PersonaRef ||
		repo.reservaRecibida.RegistroContextoRef != escenario.resultado.RegistroContextoRef ||
		repo.reservaRecibida.HuellaContextoSHA256 != escenario.resultado.HuellaSHA256 {
		t.Fatalf("reserva sin titular V2: err=%v reserva=%+v", err, repo.reservaRecibida)
	}
}

type custodioIngestaPrueba struct {
	llamadas  int
	recibido  ports.DocumentoMayan
	respuesta ports.DocumentoMayan
}

func (c *custodioIngestaPrueba) IngerirOriginal(context.Context, ports.PlanIngestaMayan, io.Reader) (ports.DocumentoMayan, error) {
	c.llamadas++
	return ports.DocumentoMayan{}, errors.New("POST inesperado")
}
func (c *custodioIngestaPrueba) ReconciliarOriginal(_ context.Context, d ports.DocumentoMayan) (ports.DocumentoMayan, error) {
	c.recibido = d
	return c.respuesta, nil
}

func TestConciliarConservaVersionConocidaYRechazaSustitucionRemota(t *testing.T) {
	p := ports.PlanIngestaMayan{Estado: ports.EstadoIngestaIndeterminada,
		Version: 3, DocumentoID: "15", ArchivoID: "20", VersionID: "30", HuellaSHA256: strings.Repeat("a", 64), Tamano: 10, MIME: "application/pdf"}
	d := ports.DocumentoMayan{DocumentoID: "15", ArchivoID: "20", VersionID: "31", HuellaSHA256: p.HuellaSHA256, Tamano: p.Tamano, MIME: p.MIME}
	if evidenciaMayanConservaIDs(p, d) {
		t.Fatal("la versión remota sustituyó el intento conocido")
	}
}

type cargasIngestaPrueba struct {
	ports.FuenteCargaAdmitidaCustodia
}

type fuenteCargaNoAutorizadaPrueba struct{ consultas int }

func (f *fuenteCargaNoAutorizadaPrueba) ConsultarAdmitida(context.Context, ports.SolicitudCargaAdmitidaCustodia) (ports.DatosCargaAdmitidaCustodia, error) {
	f.consultas++
	return ports.DatosCargaAdmitidaCustodia{}, ports.ErrCapacidadNoDisponible
}

type fuenteCargaVersionCambiadaPrueba struct {
	datos                ports.DatosCargaAdmitidaCustodia
	consultas, aperturas int
}

type fuenteCargaCierreFallidoPrueba struct {
	datos ports.DatosCargaAdmitidaCustodia
}

func (f fuenteCargaCierreFallidoPrueba) ConsultarAdmitida(context.Context, ports.SolicitudCargaAdmitidaCustodia) (ports.DatosCargaAdmitidaCustodia, error) {
	return f.datos, nil
}
func (f fuenteCargaCierreFallidoPrueba) AbrirOriginalExacto(context.Context, ports.SolicitudCargaAdmitidaCustodia, ports.DatosCargaAdmitidaCustodia) (ports.LecturaCargaAdmitidaCustodia, error) {
	return ports.LecturaCargaAdmitidaCustodia{Datos: f.datos, Contenido: lectorCierreFallido{Reader: strings.NewReader(contenidoOriginalIngestaPrueba)}}, nil
}

type lectorCierreFallido struct{ *strings.Reader }

func (lectorCierreFallido) Close() error { return errors.New("lectura S3 sin EOF verificado") }

type fuenteContenidoIngestaPrueba struct {
	datos     ports.DatosCargaAdmitidaCustodia
	contenido string
}

func (f fuenteContenidoIngestaPrueba) ConsultarAdmitida(context.Context, ports.SolicitudCargaAdmitidaCustodia) (ports.DatosCargaAdmitidaCustodia, error) {
	return f.datos, nil
}
func (f fuenteContenidoIngestaPrueba) AbrirOriginalExacto(context.Context, ports.SolicitudCargaAdmitidaCustodia, ports.DatosCargaAdmitidaCustodia) (ports.LecturaCargaAdmitidaCustodia, error) {
	return ports.LecturaCargaAdmitidaCustodia{Datos: f.datos, Contenido: io.NopCloser(strings.NewReader(f.contenido))}, nil
}

func TestOriginalS3SeVerificaCompletoAntesDePOSTAunqueCustodioNoLea(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p, datos := cargaAdmitidaIngestaPrueba(t, escenario.ahora, escenario.resultado.Contexto.PersonaRef)
	for _, caso := range []struct {
		nombre, contenido string
		valido            bool
	}{
		{"truncado", contenidoOriginalIngestaPrueba[:99], false},
		{"byte_extra", contenidoOriginalIngestaPrueba + "B", false},
		{"sha_distinto", strings.Repeat("B", 100), false},
		{"exacto", contenidoOriginalIngestaPrueba, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repositorioIngestaPrueba{plan: p, despachoNuevo: true}
			custodio := &custodioExitoIngestaPrueba{} // devuelve éxito sin leer
			s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: p.PersonaRef},
				Cargas: fuenteContenidoIngestaPrueba{datos, caso.contenido}, Custodio: custodio,
				Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{escenario.ahora}}
			_, err := s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datosActor.VinculoAutenticacionActor)
			if caso.valido {
				if err != nil || repo.despachos != 1 || custodio.llamadas != 1 || repo.ultimaResolucion.Estado != ports.EstadoIngestaConfirmada {
					t.Fatalf("original exacto no confirmado: err=%v despachos=%d POST=%d estado=%s", err, repo.despachos, custodio.llamadas, repo.ultimaResolucion.Estado)
				}
			} else if !errors.Is(err, ports.ErrSolicitudInvalida) || repo.despachos != 0 || custodio.llamadas != 0 {
				t.Fatalf("original inválido llegó a POST: err=%v despachos=%d POST=%d", err, repo.despachos, custodio.llamadas)
			}
		})
	}
}

func TestProveedorFallaTrasDespachoQuedaIndeterminadoYNoReenvia(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p, datos := cargaAdmitidaIngestaPrueba(t, escenario.ahora, escenario.resultado.Contexto.PersonaRef)
	repo := &repositorioIngestaPrueba{plan: p, despachoNuevo: true}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: p.PersonaRef},
		Cargas: fuenteContenidoIngestaPrueba{datos, contenidoOriginalIngestaPrueba}, Custodio: custodio,
		Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{escenario.ahora}}
	_, err = s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datosActor.VinculoAutenticacionActor)
	if !errors.Is(err, ports.ErrIngestaIndeterminada) || repo.ultimaResolucion.Estado != ports.EstadoIngestaIndeterminada || custodio.llamadas != 1 {
		t.Fatalf("POST incierto no conservado: err=%v estado=%s POST=%d", err, repo.ultimaResolucion.Estado, custodio.llamadas)
	}
	_, err = s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datosActor.VinculoAutenticacionActor)
	if !errors.Is(err, ports.ErrIngestaNoReenviable) || repo.despachos != 1 || custodio.llamadas != 1 {
		t.Fatalf("replay reenvió POST: err=%v despachos=%d POST=%d", err, repo.despachos, custodio.llamadas)
	}
}

type custodioExitoIngestaPrueba struct{ llamadas int }

func (c *custodioExitoIngestaPrueba) IngerirOriginal(_ context.Context, p ports.PlanIngestaMayan, _ io.Reader) (ports.DocumentoMayan, error) {
	c.llamadas++
	return ports.DocumentoMayan{DocumentoID: "15", ArchivoID: "20", VersionID: "30", HuellaSHA256: p.HuellaSHA256, Tamano: p.Tamano, MIME: p.MIME}, nil
}
func (*custodioExitoIngestaPrueba) ReconciliarOriginal(context.Context, ports.DocumentoMayan) (ports.DocumentoMayan, error) {
	panic("no conciliar sin petición")
}

func TestCierreS3FallidoAntesDelDespachoNoEnviaPOST(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p, datos := cargaAdmitidaIngestaPrueba(t, escenario.ahora, escenario.resultado.Contexto.PersonaRef)
	repo := &repositorioIngestaPrueba{plan: p, despachoNuevo: true}
	custodio := &custodioExitoIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: p.PersonaRef},
		Cargas: fuenteCargaCierreFallidoPrueba{datos}, Custodio: custodio, Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{escenario.ahora}}
	_, err = s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datosActor.VinculoAutenticacionActor)
	if !errors.Is(err, ports.ErrCapacidadNoDisponible) || repo.despachos != 0 || custodio.llamadas != 0 {
		t.Fatalf("cierre S3 permitió despacho: err=%v despachos=%d POST=%d", err, repo.despachos, custodio.llamadas)
	}
}

func (f *fuenteCargaVersionCambiadaPrueba) ConsultarAdmitida(_ context.Context, _ ports.SolicitudCargaAdmitidaCustodia) (ports.DatosCargaAdmitidaCustodia, error) {
	f.consultas++
	return f.datos, nil
}
func (f *fuenteCargaVersionCambiadaPrueba) AbrirOriginalExacto(_ context.Context, _ ports.SolicitudCargaAdmitidaCustodia, _ ports.DatosCargaAdmitidaCustodia) (ports.LecturaCargaAdmitidaCustodia, error) {
	f.aperturas++
	otra := f.datos
	c := otra.Carga
	contenido := *c.ContenidoAdmitido
	contenido.Version = "version:alterada"
	c.ContenidoAdmitido = &contenido
	otra.Carga = c
	otra.HuellaCargaSHA256, _ = c.HuellaSHA256()
	return ports.LecturaCargaAdmitidaCustodia{Datos: otra, Contenido: io.NopCloser(strings.NewReader("contenido"))}, nil
}

func cargaAdmitidaIngestaPrueba(t *testing.T, ahora time.Time, persona string) (ports.PlanIngestaMayan, ports.DatosCargaAdmitidaCustodia) {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	suma := sha256.Sum256([]byte(contenidoOriginalIngestaPrueba))
	sha := hex.EncodeToString(suma[:])
	creada := ahora.Add(-5 * time.Minute)
	c, err := vecdomain.NuevaCargaDocumental(ref("1"), persona, ref("2"), "meritos", "evidencia", ref("3"), ref("4"),
		ports.FinalidadIngestaMayan, "privada", "application/pdf", 100, sha,
		"hmac-sha256:idempotencia_v1:"+strings.Repeat("c", 64),
		"hmac-sha256:solicitud_v1:"+strings.Repeat("d", 64), creada, ahora.Add(5*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	c, err = c.Preparar("hmac-sha256:sesion_v1:"+strings.Repeat("e", 64), ref("a"), creada.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	q := vecdomain.ContenidoCargaDocumental{ConectorID: "conector:s3", Referencia: ref("5"), Version: "version:1",
		Zona: vecdomain.ZonaContenidoCargaCuarentena, MIME: "application/pdf", Tamano: 100, HuellaSHA256: sha,
		EvidenciaRef: ref("6"), RegistradoEn: creada.Add(2 * time.Minute)}
	c, err = c.RegistrarCuarentena(q, ref("b"), creada.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	a := vecdomain.AnalisisCargaDocumental{ObjetoReferencia: q.Referencia, ObjetoVersion: q.Version, HuellaObjetoSHA256: sha,
		ConectorAnalizadorID: "antivirus:local", VersionConector: 1, Estado: vecdomain.EstadoAnalisisCargaLimpio,
		CodigoResultado: "limpio", EvidenciaRef: ref("7"), HuellaEvidenciaSHA256: strings.Repeat("b", 64),
		CompletadoEn: creada.Add(3 * time.Minute)}
	c, err = c.RegistrarAnalisis(a, ref("c"), a.CompletadoEn)
	if err != nil {
		t.Fatal(err)
	}
	p := ports.PlanIngestaMayan{PlanRef: "plan:123e4567-e89b-42d3-a456-426614174000", CargaID: c.ID,
		ExpedienteRef: c.RecursoRef, PersonaRef: persona, EvidenciaRef: ref("9"), HuellaSHA256: sha,
		Tamano: 100, MIME: "application/pdf", CargaVersion: 5, Estado: ports.EstadoIngestaPreparada, Version: 1}
	admitida := q
	admitida.Referencia = ref("8")
	admitida.Version = "version:2"
	admitida.Zona = vecdomain.ZonaContenidoCargaAdmitida
	admitida.EvidenciaRef = p.EvidenciaRef
	admitida.RegistradoEn = creada.Add(4 * time.Minute)
	c, err = c.Admitir(admitida, ref("d"), admitida.RegistradoEn)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return p, ports.DatosCargaAdmitidaCustodia{Carga: c, HuellaCargaSHA256: huella}
}

func TestCambioDeVersionS3AntesDelDespachoNoEnviaMayan(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p, datos := cargaAdmitidaIngestaPrueba(t, escenario.ahora, escenario.resultado.Contexto.PersonaRef)
	fuente := &fuenteCargaVersionCambiadaPrueba{datos: datos}
	repo := &repositorioIngestaPrueba{plan: p}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: p.PersonaRef},
		Cargas: fuente, Custodio: custodio, Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{escenario.ahora}}
	_, err = s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datosActor.VinculoAutenticacionActor)
	if !errors.Is(err, ports.ErrSolicitudInvalida) || fuente.consultas != 1 || fuente.aperturas != 1 ||
		repo.despachos != 0 || custodio.llamadas != 0 {
		t.Fatalf("cambio S3 permitió despacho: err=%v consultas=%d aperturas=%d despachos=%d POST=%d",
			err, fuente.consultas, fuente.aperturas, repo.despachos, custodio.llamadas)
	}
}
func (*fuenteCargaNoAutorizadaPrueba) AbrirOriginalExacto(context.Context, ports.SolicitudCargaAdmitidaCustodia, ports.DatosCargaAdmitidaCustodia) (ports.LecturaCargaAdmitidaCustodia, error) {
	panic("S3 abierto sin consulta V3")
}

func TestSinConsultaV3DeCargaNoDespachaNiEnviaMayan(t *testing.T) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	p := ports.PlanIngestaMayan{PlanRef: "plan:123e4567-e89b-42d3-a456-426614174000", CargaID: ref("1"),
		ExpedienteRef: ref("2"), PersonaRef: escenario.resultado.Contexto.PersonaRef, CargaVersion: 5,
		Estado: ports.EstadoIngestaPreparada, Version: 1}
	repo := &repositorioIngestaPrueba{plan: p}
	fuente := &fuenteCargaNoAutorizadaPrueba{}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: escenario.ahora, principal: p.PersonaRef},
		Cargas: fuente, Custodio: custodio, Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{escenario.ahora}}
	_, err = s.Ingerir(context.Background(), p.PlanRef, p.ExpedienteRef, escenario.resultado, datos.VinculoAutenticacionActor)
	if !errors.Is(err, ports.ErrCapacidadNoDisponible) || fuente.consultas != 1 || repo.despachos != 0 || custodio.llamadas != 0 {
		t.Fatalf("sin V3 consultó S3 o Mayan: err=%v consultas=%d despachos=%d POST=%d", err, fuente.consultas, repo.despachos, custodio.llamadas)
	}
}

func TestIngestaIndeterminadaNuncaReenviaPOST(t *testing.T) {
	ahora := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	repo := &repositorioIngestaPrueba{plan: ports.PlanIngestaMayan{PlanRef: ref("1"), ExpedienteRef: ref("2"), Estado: ports.EstadoIngestaIndeterminada, Version: 3}}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t: t, ahora: ahora}, Custodio: custodio, Cargas: cargasIngestaPrueba{}, Politicas: politicasExternaPrueba{}, Reloj: relojExternaPrueba{ahora}}
	_, err := s.Ingerir(context.Background(), repo.plan.PlanRef, repo.plan.ExpedienteRef, vecdomain.ResultadoContextoActorRegistradoV2{}, vecdomain.VinculoAutenticacionActorV2{})
	if !errors.Is(err, ports.ErrIngestaNoReenviable) || custodio.llamadas != 0 || repo.despachos != 0 {
		t.Fatalf("replay indeterminado: err=%v POST=%d despachos=%d", err, custodio.llamadas, repo.despachos)
	}
}
