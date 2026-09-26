package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type relojExternaPrueba struct{ instante time.Time }

func (r relojExternaPrueba) Ahora() time.Time { return r.instante }

type politicasExternaPrueba struct {
	politica vecports.PoliticaConservacionDocumental
}

func (p politicasExternaPrueba) BuscarPoliticasConservacionDocumental(context.Context, vecports.SolicitudPoliticaConservacionDocumental) ([]vecports.PoliticaConservacionDocumental, error) {
	return []vecports.PoliticaConservacionDocumental{p.politica}, nil
}

// repositorioExternaPrueba devuelve lo que confirmaria la fachada SQL y
// registra lo que recibe; las demas operaciones no deben alcanzarse.
type repositorioExternaPrueba struct {
	recibido  ports.AltaExternaPersistente
	llamadas  int
	respuesta func(ports.AltaExternaPersistente) domain.Documento
}

func (r *repositorioExternaPrueba) ConfirmarAlta(context.Context, ports.AltaPersistente) (domain.Documento, error) {
	return domain.Documento{}, errors.New("no esperado")
}
func (r *repositorioExternaPrueba) ListarExpediente(context.Context, ports.ConsultaExpediente) (ports.PaginaDocumentos, error) {
	return ports.PaginaDocumentos{}, errors.New("no esperado")
}
func (r *repositorioExternaPrueba) Obtener(context.Context, ports.ConsultaDocumento) (domain.Documento, error) {
	return domain.Documento{}, errors.New("no esperado")
}
func (r *repositorioExternaPrueba) ConfirmarPreparacion(context.Context, ports.PreparacionNotificacion) (domain.NotificacionPreparada, error) {
	return domain.NotificacionPreparada{}, errors.New("no esperado")
}
func (r *repositorioExternaPrueba) ConfirmarReferenciaExterna(_ context.Context, a ports.AltaExternaPersistente) (domain.Documento, error) {
	r.llamadas++
	r.recibido = a
	return r.respuesta(a), nil
}

func documentoConfirmadoPrueba(a ports.AltaExternaPersistente) domain.Documento {
	p := a.Politica.Politica()
	s := p.Solicitud()
	return domain.Documento{
		ID: a.ID, NumeroVEC: "VEC-2026-7", ModuloID: a.ModuloID, ExpedienteRef: a.ExpedienteRef,
		TipoRef: a.TipoRef, Version: a.Version, MIME: a.MIME, Tamano: a.Tamano,
		HuellaSHA256: a.Custodia.HuellaSHA256, PoliticaRef: s.PoliticaRef(), VersionPolitica: s.VersionPolitica(),
		HuellaPoliticaSHA256: hex.EncodeToString(s.HuellaPoliticaSHA256()), ConservacionHasta: p.ConservacionHasta(),
		Proteccion: string(p.Proteccion()), EstadoPolitica: ports.EstadoPolitica(p), EstadoFirma: domain.EstadoFirmaPendienteProveedor,
		CreadoEn: time.Date(2026, 9, 25, 12, 0, 1, 0, time.UTC), Custodia: domain.CustodiaExterna,
		CustodiaExternaRef: a.Custodia,
	}
}

// materialExternaPrueba solo tiene forma valida: su resumen liga la operacion
// y la huella exacta de la preimagen. No es una autorizacion; la fachada SQL
// real la rechazaria sin COSE.
func materialExternaPrueba(t *testing.T, accion, efecto string, preimagen []byte, ahora time.Time) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := strings.Repeat("a", 64)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:externa:prueba", h, h, "contexto:prueba", h, accion,
		efecto, ports.HuellaEfectoV3(preimagen), ports.AudienciaV3, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("c"), vecports.TamanoMinimoCapacidadCanonicaV3), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("p"), []byte("s"), []byte("e"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func escenarioExterna(t *testing.T) (*Servicio, *repositorioExternaPrueba, ports.AltaExterna) {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	politica := politicaConservacionPrueba(t, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC), vecports.ProteccionPoliticaConservacionDocumentalOrdinaria)
	s := politica.Solicitud()
	in := ports.AltaExterna{
		ID: ref("8"), ClaveIdempotencia: ref("9"), ModuloID: "dietas",
		ExpedienteRef: s.ExpedienteRef(), TipoRef: s.TipoDocumentalRef(), Version: 1,
		Custodia: domain.ReferenciaCustodiaExterna{CustodioID: "dietas.justificantes",
			Referencia: "justificante:0001", HuellaSHA256: strings.Repeat("e", 64)},
		SolicitudPolitica: s,
	}
	resultado, err := vecports.NuevoResultadoPoliticaConservacionDocumental(politica, s, ahora)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := (ports.AltaExternaPersistente{ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia,
		ModuloID: in.ModuloID, ExpedienteRef: in.ExpedienteRef, TipoRef: in.TipoRef, Version: in.Version,
		Custodia: in.Custodia, Politica: resultado}).PreimagenExterna()
	if err != nil {
		t.Fatal(err)
	}
	in.Autorizacion = ports.AutorizacionV3{
		Material: materialExternaPrueba(t, ports.AccionRegistrarExterno, in.ID, preimagen, ahora),
		Accion:   ports.AccionRegistrarExterno, Finalidad: "registrar_documento_externo",
		RecursoRef: in.ID, AmbitoRef: in.ExpedienteRef, PrincipalID: "per:0001",
		PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001",
	}
	repo := &repositorioExternaPrueba{respuesta: documentoConfirmadoPrueba}
	return &Servicio{Repositorio: repo, Politicas: politicasExternaPrueba{politica}, Reloj: relojExternaPrueba{ahora}}, repo, in
}

func TestRegistrarExternoSinAlmacenLigaPreimagenYDevuelveCustodiaExterna(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	d, err := servicio.RegistrarExterno(context.Background(), in)
	if err != nil || repo.llamadas != 1 {
		t.Fatalf("registro externo: %v, llamadas=%d", err, repo.llamadas)
	}
	if d.Custodia != domain.CustodiaExterna || d.Descargable() || d.CustodiaExternaRef != in.Custodia ||
		repo.recibido.Custodia != in.Custodia || repo.recibido.Politica.Validar() != nil {
		t.Fatalf("documento externo discordante: %+v", d)
	}
	preimagen, _ := repo.recibido.PreimagenExterna()
	if !bytes.Contains(preimagen, []byte(`"custodia_ref":"justificante:0001"`)) || bytes.Contains(preimagen, []byte("contenido")) {
		t.Fatalf("preimagen externa inesperada: %s", preimagen)
	}
}

func TestRegistrarExternoDeniegaAntesDelRepositorio(t *testing.T) {
	casos := map[string]func(*ports.AltaExterna){
		"huella ausente":         func(a *ports.AltaExterna) { a.Custodia.HuellaSHA256 = "" },
		"referencia con ruta":    func(a *ports.AltaExterna) { a.Custodia.Referencia = "../x" },
		"custodio vacio":         func(a *ports.AltaExterna) { a.Custodia.CustodioID = "" },
		"mime con parametro":     func(a *ports.AltaExterna) { a.MIME = "application/pdf; x=1" },
		"tamano negativo":        func(a *ports.AltaExterna) { a.Tamano = -1 },
		"otro expediente":        func(a *ports.AltaExterna) { a.ExpedienteRef = "ref:" + strings.Repeat("f", 64) },
		"accion de alta":         func(a *ports.AltaExterna) { a.Autorizacion.Accion = ports.AccionAlta },
		"finalidad ajena":        func(a *ports.AltaExterna) { a.Autorizacion.Finalidad = "alta_documento_generado" },
		"recurso ajeno":          func(a *ports.AltaExterna) { a.Autorizacion.RecursoRef = "ref:" + strings.Repeat("7", 64) },
		"huella V3 de otra cosa": func(a *ports.AltaExterna) { a.Custodia.Referencia = "justificante:0002" },
		"identificador personal": func(a *ports.AltaExterna) { a.ID = "DNI:12345678Z" },
	}
	for nombre, alterar := range casos {
		servicio, repo, in := escenarioExterna(t)
		alterar(&in)
		if _, err := servicio.RegistrarExterno(context.Background(), in); !errors.Is(err, ports.ErrSolicitudInvalida) || repo.llamadas != 0 {
			t.Errorf("%s: err=%v llamadas=%d", nombre, err, repo.llamadas)
		}
	}
}

func TestRegistrarExternoRechazaConfirmacionDiscordante(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	repo.respuesta = func(a ports.AltaExternaPersistente) domain.Documento {
		d := documentoConfirmadoPrueba(a)
		d.CustodiaExternaRef.Referencia = "justificante:otro"
		return d
	}
	if _, err := servicio.RegistrarExterno(context.Background(), in); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("confirmacion con otra referencia aceptada: %v", err)
	}
}

// Una repetición con concesión nueva recalcula la conservación (ahora +
// plazo) y la fachada devuelve el recibo original, con la fecha anterior: se
// acepta. Una fecha posterior a la resuelta no procede de esa política.
func TestRegistrarExternoAceptaReciboOriginalConConservacionAnterior(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	repo.respuesta = func(a ports.AltaExternaPersistente) domain.Documento {
		d := documentoConfirmadoPrueba(a)
		d.ConservacionHasta = d.ConservacionHasta.Add(-7 * time.Second)
		return d
	}
	if _, err := servicio.RegistrarExterno(context.Background(), in); err != nil {
		t.Fatalf("recibo original de una repetición rechazado: %v", err)
	}
	servicio, repo, in = escenarioExterna(t)
	repo.respuesta = func(a ports.AltaExternaPersistente) domain.Documento {
		d := documentoConfirmadoPrueba(a)
		d.ConservacionHasta = d.ConservacionHasta.Add(time.Second)
		return d
	}
	if _, err := servicio.RegistrarExterno(context.Background(), in); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("conservación posterior a la resuelta aceptada: %v", err)
	}
}

// almacenNoAlcanzable falla si la descarga llega a pedir bytes al almacen.
type almacenNoAlcanzable struct{ vecports.AlmacenObjetos }

type repositorioObtenerExterno struct {
	repositorioExternaPrueba
	documento domain.Documento
}

func (r *repositorioObtenerExterno) Obtener(context.Context, ports.ConsultaDocumento) (domain.Documento, error) {
	return r.documento, nil
}

func TestDescargaNoSirveDocumentosDeCustodiaExterna(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	if _, err := servicio.RegistrarExterno(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	ahora := servicio.Reloj.Ahora()
	consulta := ports.ConsultaDocumento{DocumentoID: in.ID, Version: in.Version}
	preimagen, err := consulta.PreimagenDescargar()
	if err != nil {
		t.Fatal(err)
	}
	consulta.Autorizacion = ports.AutorizacionV3{
		Material: materialExternaPrueba(t, ports.AccionDescargar, in.ID, preimagen, ahora),
		Accion:   ports.AccionDescargar, Finalidad: "descargar_documento_original",
		RecursoRef: in.ID, AmbitoRef: in.ExpedienteRef, PrincipalID: "per:0001",
		PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001",
	}
	servicio.Repositorio = &repositorioObtenerExterno{documento: documentoConfirmadoPrueba(repo.recibido)}
	servicio.Almacen = almacenNoAlcanzable{}
	servicio.ContextosLectura = fabricaNoAlcanzable{t: t}
	if _, err := servicio.DescargarOriginal(context.Background(), consulta); !errors.Is(err, ports.ErrOriginalNoDisponible) {
		t.Fatalf("descarga de custodia externa: %v", err)
	}
	// Sin autoridad de lectura del almacén no se llega a consumir V3.
	servicio.ContextosLectura = nil
	servicio.Repositorio = &repositorioExternaPrueba{}
	if _, err := servicio.DescargarOriginal(context.Background(), consulta); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("descarga sin autoridad de lectura: %v", err)
	}
}

// fabricaNoAlcanzable falla la prueba si se pide un contexto de lectura.
type fabricaNoAlcanzable struct{ t *testing.T }

func (f fabricaNoAlcanzable) ContextoLecturaOriginal(context.Context, domain.Documento, ports.AutorizacionV3) (vecports.ContextoOperacionAlmacen, error) {
	f.t.Fatal("no debe pedirse contexto de lectura de un original externo")
	return vecports.ContextoOperacionAlmacen{}, nil
}

// El estado provisional viaja en la preimagen autorizada y en el documento:
// una autorización emitida para la política aprobada no sirve para registrar
// con la provisional, y el registro confirmado declara el estado.
func TestRegistrarExternoConPoliticaProvisionalLaDeclaraYLaLigaALaPreimagen(t *testing.T) {
	servicio, repo, in := escenarioExterna(t)
	aprobada := servicio.Politicas.(politicasExternaPrueba).politica
	provisional := politicaConservacionEstadoPrueba(t, aprobada.ConservacionHasta(),
		vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, vecports.EstadoPoliticaConservacionDocumentalProvisional)
	servicio.Politicas = politicasExternaPrueba{provisional}
	if _, err := servicio.RegistrarExterno(context.Background(), in); !errors.Is(err, ports.ErrSolicitudInvalida) || repo.llamadas != 0 {
		t.Fatalf("autorizacion de la politica aprobada usada con la provisional: %v llamadas=%d", err, repo.llamadas)
	}
	ahora := servicio.Reloj.Ahora()
	resultado, err := vecports.NuevoResultadoPoliticaConservacionDocumental(provisional, in.SolicitudPolitica, ahora)
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := (ports.AltaExternaPersistente{ID: in.ID, ClaveIdempotencia: in.ClaveIdempotencia,
		ModuloID: in.ModuloID, ExpedienteRef: in.ExpedienteRef, TipoRef: in.TipoRef, Version: in.Version,
		Custodia: in.Custodia, Politica: resultado}).PreimagenExterna()
	if err != nil || !strings.Contains(string(preimagen), `"estado_politica":"provisional"`) {
		t.Fatalf("preimagen sin estado provisional: %v %s", err, preimagen)
	}
	in.Autorizacion.Material = materialExternaPrueba(t, ports.AccionRegistrarExterno, in.ID, preimagen, ahora)
	d, err := servicio.RegistrarExterno(context.Background(), in)
	if err != nil || d.EstadoPolitica != domain.EstadoPoliticaProvisional || repo.llamadas != 1 {
		t.Fatalf("registro provisional: %v %+v", err, d)
	}
	// Un repositorio que confirmara otro estado no se acepta.
	repo.respuesta = func(a ports.AltaExternaPersistente) domain.Documento {
		d := documentoConfirmadoPrueba(a)
		d.EstadoPolitica = domain.EstadoPoliticaAprobada
		return d
	}
	if _, err := servicio.RegistrarExterno(context.Background(), in); !errors.Is(err, ports.ErrCapacidadNoDisponible) {
		t.Fatalf("estado confirmado distinto aceptado: %v", err)
	}
}

type fabricaLecturaFallida struct{ causa error }

func (f fabricaLecturaFallida) ContextoLecturaOriginal(context.Context, domain.Documento, ports.AutorizacionV3) (vecports.ContextoOperacionAlmacen, error) {
	return vecports.ContextoOperacionAlmacen{}, f.causa
}

func TestDescargaConservaCausaDeFabricaSinPerderCategoria(t *testing.T) {
	servicio, repo, alta := escenarioExterna(t)
	if _, err := servicio.RegistrarExterno(context.Background(), alta); err != nil {
		t.Fatal(err)
	}
	ahora := servicio.Reloj.Ahora()
	consulta := ports.ConsultaDocumento{DocumentoID: alta.ID, Version: alta.Version}
	preimagen, err := consulta.PreimagenDescargar()
	if err != nil {
		t.Fatal(err)
	}
	consulta.Autorizacion = ports.AutorizacionV3{
		Material: materialExternaPrueba(t, ports.AccionDescargar, alta.ID, preimagen, ahora),
		Accion:   ports.AccionDescargar, Finalidad: "descargar_documento_original",
		RecursoRef: alta.ID, AmbitoRef: alta.ExpedienteRef, PrincipalID: "per:0001",
		PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001",
	}
	documento := documentoConfirmadoPrueba(repo.recibido)
	documento.Custodia = domain.CustodiaVEC
	documento.CustodiaExternaRef = domain.ReferenciaCustodiaExterna{}
	documento.MIME, documento.Tamano = "application/pdf", 6
	documento.ObjetoRef, documento.ObjetoVersion = "obj:123", "version:1"
	servicio.Repositorio = &repositorioObtenerExterno{documento: documento}
	servicio.Almacen = almacenNoAlcanzable{}
	causa := errors.New("origen de lectura reservado")
	servicio.ContextosLectura = fabricaLecturaFallida{causa: causa}
	_, err = servicio.DescargarOriginal(context.Background(), consulta)
	if !errors.Is(err, ports.ErrCapacidadNoDisponible) || !errors.Is(err, causa) {
		t.Fatalf("categoria o causa perdida: %v", err)
	}
}
