package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioOriginalPrueba struct {
	ports.Repositorio
	reservas  []ports.ReservaOriginalFirmable
	respuesta ports.IntentoOriginalFirmable
	err       error
}

func (r *repositorioOriginalPrueba) ReservarOriginalFirmable(_ context.Context, reserva ports.ReservaOriginalFirmable) (ports.IntentoOriginalFirmable, error) {
	r.reservas = append(r.reservas, reserva)
	if r.err != nil {
		return ports.IntentoOriginalFirmable{}, r.err
	}
	return r.respuesta, nil
}
func (r *repositorioOriginalPrueba) ConfirmarOriginalFirmable(context.Context, ports.ConfirmacionOriginalFirmable) (domain.Documento, error) {
	panic("confirmacion inesperada")
}

type almacenOriginalNoLlamado struct{ vecports.AlmacenObjetos }

type politicasOriginalPrueba struct {
	politicasExternaPrueba
	tipo string
}

func (p politicasOriginalPrueba) CustodiaOriginalCTReservada(tipo string) bool { return tipo == p.tipo }

func (almacenOriginalNoLlamado) Capacidades(context.Context) (vecports.CapacidadesAlmacenObjetos, error) {
	panic("almacen consultado")
}
func (almacenOriginalNoLlamado) Escribir(context.Context, vecports.SolicitudEscribirObjeto) (vecports.ResultadoOperacionObjeto, error) {
	panic("objeto escrito")
}

type autoridadOriginalPrueba struct {
	t        *testing.T
	ahora    time.Time
	llamadas int
}

func (a *autoridadOriginalPrueba) AutorizarReservaOriginal(_ context.Context, preimagen []byte, id, expediente string) (ports.AutorizacionV3, error) {
	a.llamadas++
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, ports.AccionReservarOriginalFirmable, id, preimagen, a.ahora),
		Accion: ports.AccionReservarOriginalFirmable, Finalidad: ports.FinalidadOriginalFirmable,
		RecursoRef: id, AmbitoRef: expediente, PrincipalID: "per:0001", PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}, nil
}
func (*autoridadOriginalPrueba) AutorizarConfirmacionOriginal(context.Context, []byte, string, string) (ports.AutorizacionV3, error) {
	panic("confirmacion autorizada antes del objeto")
}
func (*autoridadOriginalPrueba) ContextoEscrituraOriginal(context.Context, ports.ReservaOriginalFirmable, ports.IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error) {
	panic("concesion de almacen solicitada")
}

func escenarioOriginalPrueba(t *testing.T) (*Servicio, *repositorioOriginalPrueba, *autoridadOriginalPrueba, ports.OrdenCustodiarOriginalFirmable) {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	politica := politicaConservacionPrueba(t, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC), vecports.ProteccionPoliticaConservacionDocumentalOrdinaria)
	s := politica.Solicitud()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	in := ports.OrdenCustodiarOriginalFirmable{ID: ref("8"), ClaveIdempotencia: ref("9"), ModuloID: "contrataciontemporal",
		ExpedienteRef: s.ExpedienteRef(), TipoRef: s.TipoDocumentalRef(), Version: 1,
		MIME: "application/pdf", Contenido: []byte("%PDF-1.7\nensayo"), SolicitudPolitica: s}
	suma := sha256.Sum256(in.Contenido)
	repo := &repositorioOriginalPrueba{respuesta: ports.IntentoOriginalFirmable{ReservaRef: ref("a"), Estado: "confirmado",
		DocumentoID: in.ID, HuellaSHA256: hex.EncodeToString(suma[:]), Numero: 1, ClaveAlmacenRef: ref("b")}}
	autoridad := &autoridadOriginalPrueba{t: t, ahora: ahora}
	servicio := &Servicio{Repositorio: repo, Almacen: almacenOriginalNoLlamado{},
		Politicas: politicasOriginalPrueba{politicasExternaPrueba: politicasExternaPrueba{politica}, tipo: in.TipoRef},
		Reloj:     relojExternaPrueba{ahora}}
	return servicio, repo, autoridad, in
}

func TestOriginalFirmableReservaAntesDeTocarAlmacenYReplayConfirmado(t *testing.T) {
	s, repo, autoridad, in := escenarioOriginalPrueba(t)
	got, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad)
	if err != nil || got != repo.respuesta || len(repo.reservas) != 1 || autoridad.llamadas != 1 {
		t.Fatalf("replay confirmado: intento=%+v err=%v reservas=%d autorizaciones=%d", got, err, len(repo.reservas), autoridad.llamadas)
	}
	if repo.reservas[0].HuellaSHA256 != got.HuellaSHA256 || repo.reservas[0].Tamano != int64(len(in.Contenido)) {
		t.Fatal("la reserva no fijo bytes exactos")
	}
	fallo := errors.New("reserva no disponible")
	repo.err = fallo
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, fallo) {
		t.Fatalf("fallo de reserva: %v", err)
	}
	// Las llamadas al almacén lanzan panic: el fallo SQL no debe producir objeto.
}

func TestOriginalFirmableNoAdmitePDFAlteradoNiTipoDistintoConReplay(t *testing.T) {
	s, repo, autoridad, in := escenarioOriginalPrueba(t)
	in.Contenido = []byte("%PDF-1.7\notros bytes")
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("replay de bytes distintos: %v", err)
	}
	in.Contenido = []byte("%PDF-1.7\nensayo")
	in.MIME = "text/plain"
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("tipo no PDF: %v", err)
	}
	if len(repo.reservas) != 1 || autoridad.llamadas != 1 {
		t.Fatalf("la segunda solicitud llego a SQL: reservas=%d autorizaciones=%d", len(repo.reservas), autoridad.llamadas)
	}
}

func TestAltaGenericaDeniegaTipoOriginalCTAntesDelObjeto(t *testing.T) {
	s, _, _, in := escenarioOriginalPrueba(t)
	if !s.tipoReservadoOriginalCT(in.TipoRef) {
		t.Fatal("la marca original_ct del catalogo no se reconocio")
	}
	_, err := s.AltaGenerado(context.Background(), ports.AltaGenerado{
		ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia, ModuloID: in.ModuloID,
		ExpedienteRef: in.ExpedienteRef, TipoRef: in.TipoRef, Version: in.Version,
		MIME: in.MIME, Contenido: in.Contenido, SolicitudPolitica: in.SolicitudPolitica,
	})
	if !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("alta generica del original reservado: %v", err)
	}
	// El almacén del escenario hace panic al escribir o consultar capacidades.
	if s.tipoReservadoOriginalCT("ref:" + strings.Repeat("7", 64)) {
		t.Fatal("otro tipo quedo reservado implicitamente")
	}
}

type autoridadRecuperacionOriginal struct {
	t              *testing.T
	base           escenarioOrdenAutorizacionV3Prueba
	intentos       []ports.IntentoOriginalFirmable
	confirmaciones int
}

func (a *autoridadRecuperacionOriginal) AutorizarReservaOriginal(_ context.Context, preimagen []byte, id, expediente string) (ports.AutorizacionV3, error) {
	return a.autorizacion(preimagen, id, expediente, ports.AccionReservarOriginalFirmable), nil
}

func (a *autoridadRecuperacionOriginal) AutorizarConfirmacionOriginal(_ context.Context, preimagen []byte, id, expediente string) (ports.AutorizacionV3, error) {
	a.confirmaciones++
	return a.autorizacion(preimagen, id, expediente, ports.AccionConfirmarOriginalFirmable), nil
}

func (a *autoridadRecuperacionOriginal) autorizacion(preimagen []byte, id, expediente, accion string) ports.AutorizacionV3 {
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, accion, id, preimagen, a.base.ahora.Add(2*time.Second)),
		Accion: accion, Finalidad: ports.FinalidadOriginalFirmable, RecursoRef: id,
		AmbitoRef: expediente, PrincipalID: "per:0001", PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}
}

func (a *autoridadRecuperacionOriginal) ContextoEscrituraOriginal(_ context.Context, reserva ports.ReservaOriginalFirmable, intento ports.IntentoOriginalFirmable) (vecports.ContextoOperacionAlmacen, error) {
	a.intentos = append(a.intentos, intento)
	datosBase, err := a.base.solicitud.Datos()
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	vinculos := vecports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:original:" + intento.ReservaRef, CargaRef: intento.ClaveAlmacenRef,
		Clasificacion: "datos_personales_alta", SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64), EfectoRef: reserva.ID,
	}
	atributos := map[string]string{
		vecports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		vecports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		vecports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		vecports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		vecports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		vecports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: datosBase.VinculoAutenticacionActor, ReferenciaMotivo: datosBase.ReferenciaMotivo,
		Accion: vecports.AccionNegocioEscribirOriginalFirmable,
		Recurso: core.RecursoAutorizable{Referencia: reserva.ID, ModuloID: "documentos", Tipo: "original_firmable",
			Ambitos: map[string]string{"unidad": "seleccion"}, Atributos: atributos},
		Finalidad: "gestion_documental", Correlacion: datosBase.Correlacion,
	})
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	instantanea := a.base.instantnea
	instantanea.VersionRol.Concesiones = []core.ConcesionRol{{
		Accion: vecports.AccionNegocioEscribirOriginalFirmable, ModuloID: "documentos", TipoRecurso: "original_firmable",
		Finalidades: []string{"gestion_documental"}, GarantiaMinima: core.AuthAssuranceSubstantial,
		CamposPermitidos: []string{"original_firmable.contenido", "evidencia_almacen"},
	}}
	decisionRef := "dec_" + strings.Repeat("a", 31) + string(rune('0'+intento.Numero))
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, decisionRef,
		a.base.ahora, a.base.ahora.Add(90*time.Second))
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	orden, err := vecports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, datosBase.ReferenciaMotivo, a.base.resultado)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	confirmacion, err := vecports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), &registroConcesionLigadaV3Prueba{registradaEn: a.base.ahora.Add(time.Second)}, orden)
	if err != nil {
		return vecports.ContextoOperacionAlmacen{}, err
	}
	return vecports.NuevoContextoEscribirOriginalFirmableAlmacenV3(solicitud, decision, confirmacion, vinculos, a.base.ahora.Add(2*time.Second))
}

type repositorioRecuperacionOriginal struct {
	ports.Repositorio
	intentos       []ports.IntentoOriginalFirmable
	reservas       []ports.ReservaOriginalFirmable
	confirmaciones []ports.ConfirmacionOriginalFirmable
	confirmadas    int
	fallo          error
}

func (r *repositorioRecuperacionOriginal) ReservarOriginalFirmable(_ context.Context, reserva ports.ReservaOriginalFirmable) (ports.IntentoOriginalFirmable, error) {
	r.reservas = append(r.reservas, reserva)
	intento := r.intentos[len(r.reservas)-1]
	intento.DocumentoID = reserva.ID
	intento.HuellaSHA256 = reserva.HuellaSHA256
	return intento, nil
}

func (r *repositorioRecuperacionOriginal) ConfirmarOriginalFirmable(_ context.Context, c ports.ConfirmacionOriginalFirmable) (domain.Documento, error) {
	r.confirmaciones = append(r.confirmaciones, c)
	actual := r.intentos[len(r.reservas)-1]
	if c.Intento.ReservaRef != actual.ReservaRef || c.Intento.ClaveAlmacenRef != actual.ClaveAlmacenRef ||
		c.Objeto.ClaveAlmacenRef != actual.ClaveAlmacenRef {
		return domain.Documento{}, ports.ErrSolicitudInvalida
	}
	if r.fallo != nil {
		fallo := r.fallo
		r.fallo = nil
		return domain.Documento{}, fallo
	}
	reserva := r.reservas[len(r.reservas)-1]
	r.confirmadas++
	p := reserva.Politica.Politica()
	s := p.Solicitud()
	return domain.Documento{ID: reserva.ID, NumeroVEC: "VEC-2026-7", ModuloID: reserva.ModuloID,
		ExpedienteRef: reserva.ExpedienteRef, TipoRef: reserva.TipoRef, Version: reserva.Version,
		MIME: reserva.MIME, HuellaSHA256: reserva.HuellaSHA256, Tamano: reserva.Tamano,
		ObjetoRef: c.Objeto.ObjetoRef, ObjetoVersion: c.Objeto.ObjetoVersion,
		PoliticaRef: s.PoliticaRef(), VersionPolitica: s.VersionPolitica(),
		HuellaPoliticaSHA256: hex.EncodeToString(s.HuellaPoliticaSHA256()),
		ConservacionHasta:    p.ConservacionHasta(), Proteccion: string(p.Proteccion()),
		EstadoPolitica: ports.EstadoPolitica(p), EstadoFirma: domain.EstadoFirmaPendienteProveedor,
		CreadoEn: time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC), Custodia: domain.CustodiaVEC}, nil
}

type almacenRecuperacionOriginal struct {
	vecports.AlmacenObjetos
	t          *testing.T
	escrituras []vecports.SolicitudEscribirObjeto
	bytes      [][]byte
	resultados []vecports.ResultadoOperacionObjeto
}

func (a *almacenRecuperacionOriginal) Capacidades(context.Context) (vecports.CapacidadesAlmacenObjetos, error) {
	return vecports.CapacidadesAlmacenObjetos{ConectorID: "almacen_s3_corporativo", EscrituraEnFlujo: true,
		ReferenciasOpacas: true, IntegridadSHA256: true, Retencion: true, TamanoMaximoObjeto: 1 << 20}, nil
}

func (a *almacenRecuperacionOriginal) Escribir(_ context.Context, s vecports.SolicitudEscribirObjeto) (vecports.ResultadoOperacionObjeto, error) {
	if err := s.Validar(); err != nil {
		a.t.Fatalf("solicitud de escritura: %v", err)
	}
	b, err := io.ReadAll(s.Contenido)
	if err != nil {
		a.t.Fatal(err)
	}
	a.escrituras = append(a.escrituras, s)
	a.bytes = append(a.bytes, b)
	proyeccion, err := s.Contexto.Proyeccion()
	if err != nil {
		a.t.Fatal(err)
	}
	ref := vecports.ReferenciaObjetoAlmacen{Referencia: "objeto:original:ensayo:" + s.ClaveIdempotencia, Version: "v1"}
	instante := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	e := vecports.EvidenciaOperacionAlmacen{Referencia: "evidencia:original:" + s.ClaveIdempotencia,
		ConectorID: "almacen_s3_corporativo", EsquemaContexto: proyeccion.Esquema,
		AccionNegocio: proyeccion.AccionNegocio, Accion: proyeccion.AccionTecnica,
		EfectoRef: proyeccion.EfectoRef, HuellaPlanEfectoSHA256: proyeccion.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: proyeccion.HuellaManifiestoSHA256, HuellaPasoSHA256: proyeccion.HuellaPasoSHA256,
		PasoRef: proyeccion.PasoRef, HuellaDecisionSHA256: proyeccion.HuellaDecisionSHA256,
		Objeto: ref, OperacionRef: proyeccion.OperacionRef, CorrelacionRef: proyeccion.CorrelacionRef,
		AutorizacionRef: proyeccion.AutorizacionRef, Finalidad: proyeccion.Finalidad,
		Clasificacion: proyeccion.Clasificacion, RealizadaEn: instante, CargaRef: proyeccion.CargaRef,
		SujetoSeudonimoHMAC: proyeccion.SujetoSeudonimoHMAC, RecursoRef: proyeccion.RecursoRef,
		ModuloID: proyeccion.ModuloID, HuellaSolicitudHMAC: proyeccion.HuellaSolicitudHMAC}
	obj := vecports.ObjetoAlmacenado{Objeto: ref, ConectorID: e.ConectorID, Zona: s.Zona, MIME: s.MIME,
		Tamano: s.Tamano, HuellaSHA256: s.HuellaSHA256, EvidenciaCreacionRef: e.Referencia,
		AlmacenadoEn: instante, RetenidoHasta: time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)}
	resultado := vecports.ResultadoOperacionObjeto{Objeto: obj, Evidencia: e}
	a.resultados = append(a.resultados, resultado)
	return resultado, nil
}

func TestOriginalFirmableRecuperaConfirmacionFallidaConNuevoIntento(t *testing.T) {
	s, _, _, in := escenarioOriginalPrueba(t)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	repo := &repositorioRecuperacionOriginal{intentos: []ports.IntentoOriginalFirmable{
		{ReservaRef: ref("a"), Estado: "pendiente", ClaveAlmacenRef: ref("b"), Numero: 1},
		{ReservaRef: ref("a"), Estado: "pendiente", ClaveAlmacenRef: ref("d"), Numero: 2},
	}}
	fallo := errors.New("confirmacion SQL interrumpida")
	repo.fallo = fallo
	almacen := &almacenRecuperacionOriginal{t: t}
	autoridad := &autoridadRecuperacionOriginal{t: t, base: nuevoEscenarioOrdenAutorizacionV3Prueba(t)}
	s.Repositorio = repo
	s.Almacen = almacen
	if _, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad); !errors.Is(err, fallo) {
		t.Fatalf("primera confirmacion debio fallar: %v", err)
	}
	if len(almacen.escrituras) != 1 || len(repo.confirmaciones) != 1 {
		t.Fatalf("primer intento: escrituras=%d confirmaciones=%d", len(almacen.escrituras), len(repo.confirmaciones))
	}
	primera := repo.confirmaciones[0]
	if primera.Objeto.ClaveAlmacenRef != ref("b") || primera.Intento.Numero != 1 {
		t.Fatal("primer recibo no pertenece a su reserva")
	}
	got, err := s.CustodiarOriginalFirmable(context.Background(), in, autoridad)
	if err != nil {
		t.Fatalf("recuperacion: %v", err)
	}
	if got.Estado != "confirmado" || got.ReservaRef != ref("a") || got.ClaveAlmacenRef != ref("d") || got.Numero != 2 ||
		got.HuellaSHA256 != repo.reservas[0].HuellaSHA256 || len(repo.reservas) != 2 || len(almacen.escrituras) != 2 ||
		len(repo.confirmaciones) != 2 || repo.confirmadas != 1 || autoridad.confirmaciones != 2 {
		t.Fatalf("recuperacion incoherente: intento=%+v reservas=%d escrituras=%d confirmaciones=%d", got, len(repo.reservas), len(almacen.escrituras), len(repo.confirmaciones))
	}
	segunda := repo.confirmaciones[1]
	proyeccion1, err := almacen.escrituras[0].Contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	proyeccion2, err := almacen.escrituras[1].Contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	if almacen.escrituras[0].ClaveIdempotencia != ref("b") || almacen.escrituras[1].ClaveIdempotencia != ref("d") ||
		proyeccion1.CargaRef != ref("b") || proyeccion2.CargaRef != ref("d") ||
		proyeccion1.AutorizacionRef == proyeccion2.AutorizacionRef {
		t.Fatal("la recuperacion no uso otra clave y concesion V3")
	}
	if segunda.Objeto.ClaveAlmacenRef != ref("d") || segunda.Objeto.ReciboObjetoRef != almacen.resultados[1].Evidencia.OperacionRef ||
		segunda.Objeto.HuellaSHA256 != got.HuellaSHA256 || !bytes.Equal(almacen.bytes[0], in.Contenido) || !bytes.Equal(almacen.bytes[1], in.Contenido) {
		t.Fatalf("segunda confirmacion: clave=%q recibo=%q huella=%q, bytes=%q/%q", segunda.Objeto.ClaveAlmacenRef, segunda.Objeto.ReciboObjetoRef, segunda.Objeto.HuellaSHA256, almacen.bytes[0], almacen.bytes[1])
	}
	recibo, err := json.Marshal(almacen.resultados[1].Evidencia)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecibo := sha256.Sum256(recibo)
	if segunda.Objeto.ReciboObjetoHuellaSHA256 != hex.EncodeToString(huellaRecibo[:]) {
		t.Fatal("huella del recibo de la segunda escritura incorrecta")
	}
	claveAnterior := segunda
	claveAnterior.Objeto.ClaveAlmacenRef = primera.Objeto.ClaveAlmacenRef
	if _, err := PreimagenConfirmacionOriginalFirmable(claveAnterior); err == nil {
		t.Fatal("la confirmacion admitio la clave del intento anterior")
	}
	if _, err := repo.ConfirmarOriginalFirmable(context.Background(), primera); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("se acepto la clave del intento anterior: %v", err)
	}
}

type revalidadorOrdenAutorizacionV3Prueba struct {
	resultado core.AutenticacionRevalidadaV1
}

func (r revalidadorOrdenAutorizacionV3Prueba) RevalidarAutenticacionActorV1(
	context.Context,
	core.SolicitudRevalidacionAutenticacionActorV1,
) (core.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorOrdenAutorizacionV3Prueba struct {
	resultado core.ResultadoContextoActorRegistradoV2
}

func (r resolutorOrdenAutorizacionV3Prueba) ResolverContextoActorRegistradoV2(
	context.Context,
	core.SolicitudContextoActor,
) (core.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojOrdenAutorizacionV3Prueba struct{ ahora time.Time }

func (r relojOrdenAutorizacionV3Prueba) Ahora() time.Time { return r.ahora }

type registroConcesionLigadaV3Prueba struct {
	registradaEn time.Time
	err          error
	invocaciones int
	cancelar     context.CancelFunc
}

func (r *registroConcesionLigadaV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	_ vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.invocaciones++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.registradaEn, r.err
}

type generadorCorrelacionOrdenAutorizacionV3Prueba struct{ valor string }

func (g generadorCorrelacionOrdenAutorizacionV3Prueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioOrdenAutorizacionV3Prueba struct {
	ahora      time.Time
	solicitud  core.SolicitudAutorizacionLigadaV3
	decision   core.DecisionAutorizacionLigadaV3
	motivo     core.ReferenciaEntradaCatalogo
	resultado  core.ResultadoContextoActorRegistradoV2
	instantnea core.InstantaneaAutorizacion
}

func nuevoEscenarioOrdenAutorizacionV3Prueba(t *testing.T) escenarioOrdenAutorizacionV3Prueba {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 11, 59, 58, 0, time.UTC)
	cuenta := core.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: core.AuthMethodCertificate,
		Garantia: core.AuthAssuranceHigh,
	}
	instantaneaActor := core.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       core.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := core.NuevoContextoActor(cuenta, instantaneaActor, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := core.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := core.ManifiestoProcedenciaContextoActorV1{
		Esquema:           core.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: core.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: core.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: core.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: core.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]core.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := core.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := core.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      core.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute), SesionRevalidadaEn: ahora.Add(-3 * time.Minute),
		SesionValidaHasta: ahora.Add(20 * time.Minute),
	}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de contexto: %v", err)
	}
	if err := autenticacion.Validar(); err != nil {
		t.Fatalf("autenticacion: %v", err)
	}
	vinculo, err := core.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidadorOrdenAutorizacionV3Prueba{autenticacion},
		core.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorOrdenAutorizacionV3Prueba{resultado},
		core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef},
		relojOrdenAutorizacionV3Prueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := core.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionOrdenAutorizacionV3Prueba{
			valor: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(
		core.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
			Accion: "bolsa.expediente.leer",
			Recurso: core.RecursoAutorizable{
				Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
				Ambitos: map[string]string{"unidad": "seleccion"},
			},
			Finalidad: "gestion_bolsa", Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	version := core.VersionRol{
		RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico de bolsa",
		Estado: core.EstadoVersionRolPublicada,
		Concesiones: []core.ConcesionRol{{
			Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente",
			Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: core.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := core.InstantaneaAutorizacion{
		AsignacionPerfil: core.AsignacionPerfil{
			AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
			PrincipalID: instantaneaActor.PersonaRef, VersionRolRef: version.Referencia(),
			Estado:       core.EstadoAsignacionPerfilActiva,
			Ambitos:      []core.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         core.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef",
		ahora, ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioOrdenAutorizacionV3Prueba{
		ahora: ahora, solicitud: solicitud, decision: decision, motivo: motivo,
		resultado: resultado, instantnea: instantanea,
	}
}
