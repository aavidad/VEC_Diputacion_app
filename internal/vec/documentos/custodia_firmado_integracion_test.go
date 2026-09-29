package documentos_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/almacen/ficheros"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docpostgres "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Recorrido de la custodia del documento firmado con piezas reales: servicio,
// catálogo de conservación, fábrica de almacén V3 con concesiones V3
// evaluadas y registradas, y almacén de ficheros. Solo el repositorio SQL es
// un doble (lo prueba PG18 en adapters/postgres).

const (
	personaCustodia = "per_custodia0123456789abcd"
	perfilCustodia  = "prf_custodia0123456789abcd"
)

type relojCustodia struct{ t time.Time }

func (r relojCustodia) Ahora() time.Time { return r.t }

type emisorCustodia struct {
	ahora    time.Time
	persona  string
	accion   string
	llamadas int
}

func (e *emisorCustodia) SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	return docautorizacion.DatosSeudonimosLectura{
		SujetoHMAC:    "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		SolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
	}, nil
}

func (e *emisorCustodia) EmitirConcesionAlmacenV3(_ context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	e.llamadas++
	c, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{
		Instante: e.ahora, PersonaRef: e.persona, PerfilRef: perfilCustodia, Accion: s.Accion, AccionConcedida: e.accion,
		Recurso: s.Recurso, Finalidad: s.Finalidad, Campos: []string{"documento_firmado.custodia", "evidencia_custodia"},
		DecisionRef: "dec_custodia56789abcdef0123456789abcd",
	})
	return docautorizacion.ConcesionAlmacenV3{Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion}, err
}

type repositorioCustodia struct {
	persistente docports.CustodiaFirmadoPersistente
	llamadas    int
	err         error
}

func (r *repositorioCustodia) ConfirmarCustodiaFirmado(_ context.Context, c docports.CustodiaFirmadoPersistente) (domain.Documento, error) {
	r.llamadas++
	r.persistente = c
	if r.err != nil {
		return domain.Documento{}, r.err
	}
	p := c.Politica.Politica()
	s := p.Solicitud()
	return domain.Documento{
		ID: c.ID, NumeroVEC: "VEC-2026-1", ModuloID: c.ModuloID, ExpedienteRef: c.ExpedienteRef, TipoRef: c.TipoRef,
		Version: c.Version, MIME: docports.MIMEDocumentoFirmado, HuellaSHA256: c.HuellaSHA256, Tamano: c.Tamano,
		ObjetoRef: c.Objeto.Objeto.Objeto.Referencia, ObjetoVersion: c.Objeto.Objeto.Objeto.Version,
		PoliticaRef: s.PoliticaRef(), VersionPolitica: s.VersionPolitica(), HuellaPoliticaSHA256: hex.EncodeToString(s.HuellaPoliticaSHA256()),
		ConservacionHasta: p.ConservacionHasta(), Proteccion: string(p.Proteccion()), EstadoPolitica: docports.EstadoPolitica(p),
		EstadoFirma: domain.EstadoFirmaPendienteProveedor, CreadoEn: time.Now().UTC(), Custodia: domain.CustodiaVEC,
	}, nil
}

type escenarioCustodia struct {
	ahora    time.Time
	servicio *docapp.Servicio
	repo     *repositorioCustodia
	emisor   *emisorCustodia
	orden    docports.CustodiaFirmado
	// preimagen y material permiten pedir otra decisión V3 para la misma orden.
	preimagen []byte
}

// reautorizar sustituye la V3 por otra decisión de la misma orden, como una
// repetición de la petición tras perder la respuesta.
func (e *escenarioCustodia) reautorizar(t *testing.T, ahora time.Time, decisionRef string) {
	t.Helper()
	a := e.orden.Autorizacion
	a.Material = materialCustodia(t, e.orden.ID, e.preimagen, ahora, decisionRef, a.PrincipalID, a.PerfilActivoRef, a.CorrelacionRef)
	e.orden.Autorizacion = a
}

// materialCustodia es la V3 de la custodia ligada a la preimagen exacta que
// calculará el servicio (misma política y mismo reloj). La capacidad y la
// decisión llevan, en JSON, los campos que cotejan las fachadas AD3
// sintéticas de probar_integracion_pg18.sh (sin COSE real).
func materialCustodia(t *testing.T, id string, preimagen []byte, ahora time.Time, decisionRef, principal, perfil, correlacion string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h := strings.Repeat("a", 64)
	efecto := docports.HuellaEfectoV3(preimagen)
	capacidad, err := json.Marshal(map[string]any{
		"audiencia_consumo": docports.AudienciaV3, "operacion": docports.AccionCustodiarFirmado, "efecto_ref": id,
		"huella_efecto_sha256": efecto, "decision_ref": decisionRef, "relleno": strings.Repeat("r", vecports.TamanoMinimoCapacidadCanonicaV3),
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := json.Marshal(map[string]any{
		"accion": docports.AccionCustodiarFirmado, "modulo_id": "documentos", "tipo_recurso": "documento_firmado",
		"finalidad": docports.FinalidadCustodiarFirmado, "campos_permitidos": []string{"documento_firmado.custodia", "evidencia_custodia"},
		"obligaciones": []string{}, "recurso_ref": id, "contexto_recurso_huella_sha256": efecto,
		"principal_id": principal, "perfil_activo_ref": perfil, "correlacion_ref": correlacion, "decision_ref": decisionRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, h, h,
		"contexto:prueba", h, docports.AccionCustodiarFirmado, id, efecto, docports.AudienciaV3,
		ahora, ahora.Add(5*time.Second))
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
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen,
		decision, []byte("{}"), []byte("{}"), 1, 1, []byte("p"), []byte("s"), []byte("e"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func nuevoEscenarioCustodia(t *testing.T) escenarioCustodia {
	t.Helper()
	return nuevoEscenarioCustodiaEn(t, time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC), "1")
}

// nuevoEscenarioCustodiaEn fija el instante de la autorización y un sufijo que
// distingue identificador, clave, operación de firma y decisión.
func nuevoEscenarioCustodiaEn(t *testing.T, ahora time.Time, sufijo string) escenarioCustodia {
	t.Helper()
	reloj := relojCustodia{ahora.Add(time.Second)}
	catalogo, err := conservacion.NuevoCatalogoProvisional(reloj)
	if err != nil {
		t.Fatal(err)
	}
	// El almacén de ficheros sin retención al escribir es el de la principal
	// (catálogo de conservación provisional).
	directorio := filepath.Join(t.TempDir(), "originales")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	almacen, err := ficheros.Nuevo(ficheros.Configuracion{ConectorID: "ficheros_ensayo", Directorio: directorio, TamanoMaximo: 1 << 20}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = almacen.Cerrar() })
	emisor := &emisorCustodia{ahora: ahora, persona: personaCustodia}
	fabrica, err := docautorizacion.NuevaFabricaContextoCustodiaFirmadoV3(emisor, reloj)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioCustodia{}
	servicio := &docapp.Servicio{Repositorio: nil, Almacen: almacen, Politicas: catalogo, Reloj: reloj,
		RepositorioCustodia: repo, ContextosCustodia: fabrica}
	tipo, err := catalogo.TipoDocumentalRef("contratacion_temporal.resolucion_firmada.v1")
	if err != nil {
		t.Fatal(err)
	}
	expediente := "ref:" + strings.Repeat("e", 64)
	solicitud, err := catalogo.SolicitudPara("contratacion_temporal.resolucion_firmada.v1", expediente)
	if err != nil {
		t.Fatal(err)
	}
	pdf := []byte("%PDF-1.7 resolucion firmada de prueba")
	suma := sha256.Sum256(pdf)
	orden := docports.CustodiaFirmado{
		ID: refPrueba("id", sufijo), ClaveIdempotencia: refPrueba("clave", sufijo),
		ModuloID: "contratacion_temporal", ExpedienteRef: expediente, TipoRef: tipo, Version: 1, Contenido: pdf,
		HuellaOriginalSHA256: strings.Repeat("0", 63) + "9", FirmaOperacionRef: refPrueba("firma", sufijo),
		SolicitudPolitica: solicitud,
	}
	politica, err := vecports.NuevoResultadoPoliticaConservacionDocumental(politicaDe(t, catalogo, solicitud), solicitud, reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	preimagen, err := (docports.CustodiaFirmadoPersistente{ID: orden.ID, ClaveIdempotencia: orden.ClaveIdempotencia,
		ModuloID: orden.ModuloID, ExpedienteRef: orden.ExpedienteRef, TipoRef: orden.TipoRef, Version: orden.Version,
		HuellaSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(pdf)), HuellaOriginalSHA256: orden.HuellaOriginalSHA256,
		FirmaOperacionRef: orden.FirmaOperacionRef, Politica: politica}).PreimagenCustodia()
	if err != nil {
		t.Fatal(err)
	}
	_, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, personaCustodia, perfilCustodia,
		vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	orden.Autorizacion = docports.AutorizacionV3{
		Material: materialCustodia(t, orden.ID, preimagen, ahora, refPrueba("decision", sufijo),
			datos.PrincipalID, datos.PerfilActivoRef, "correlacion:custodia:0001"),
		Accion: docports.AccionCustodiarFirmado, Finalidad: docports.FinalidadCustodiarFirmado,
		RecursoRef: orden.ID, AmbitoRef: expediente,
		PrincipalID: datos.PrincipalID, PerfilActivoRef: datos.PerfilActivoRef, CorrelacionRef: "correlacion:custodia:0001",
	}
	return escenarioCustodia{ahora: ahora, servicio: servicio, repo: repo, emisor: emisor, orden: orden, preimagen: preimagen}
}

func refPrueba(uso, sufijo string) string {
	s := sha256.Sum256([]byte(uso + "\x00" + sufijo))
	return "ref:" + hex.EncodeToString(s[:])
}

func politicaDe(t *testing.T, c *conservacion.Catalogo, s vecports.SolicitudPoliticaConservacionDocumental) vecports.PoliticaConservacionDocumental {
	t.Helper()
	p, err := c.BuscarPoliticasConservacionDocumental(context.Background(), s)
	if err != nil || len(p) != 1 {
		t.Fatalf("política: %v %d", err, len(p))
	}
	return p[0]
}

func TestCustodiaFirmadoEscribeYConfirmaConConcesionRegistrada(t *testing.T) {
	e := nuevoEscenarioCustodia(t)
	documento, err := e.servicio.CustodiarFirmado(context.Background(), e.orden)
	if err != nil {
		t.Fatalf("custodia: %v", err)
	}
	if documento.Custodia != domain.CustodiaVEC || documento.EstadoFirma != domain.EstadoFirmaPendienteProveedor ||
		e.repo.llamadas != 1 || e.emisor.llamadas != 1 ||
		e.repo.persistente.FirmaOperacionRef != e.orden.FirmaOperacionRef ||
		e.repo.persistente.HuellaOriginalSHA256 != e.orden.HuellaOriginalSHA256 {
		t.Fatalf("custodia inesperada: %+v", documento)
	}
	// El objeto escrito es el PDF exacto.
	if e.repo.persistente.Objeto.Objeto.HuellaSHA256 != documento.HuellaSHA256 || e.repo.persistente.Objeto.Objeto.Tamano != int64(len(e.orden.Contenido)) {
		t.Fatal("el objeto escrito no es el PDF firmado")
	}
}

func TestCustodiaFirmadoNoEscribeSinConcesionValida(t *testing.T) {
	casos := map[string]func(*escenarioCustodia){
		"concesión de almacén denegada": func(e *escenarioCustodia) { e.emisor.accion = docports.AccionListar },
		"otro actor en la concesión":    func(e *escenarioCustodia) { e.emisor.persona = "per_otro000123456789abcdef" },
		"autorización de otra acción": func(e *escenarioCustodia) {
			e.orden.Autorizacion.Accion = docports.AccionAlta
			e.orden.Autorizacion.Finalidad = "alta_documento_generado"
		},
		"autorización de otra preimagen": func(e *escenarioCustodia) { e.orden.HuellaOriginalSHA256 = strings.Repeat("8", 64) },
		"tipo no reservado": func(e *escenarioCustodia) {
			e.orden.TipoRef = "ref:" + strings.Repeat("4", 64)
		},
		"no es un PDF": func(e *escenarioCustodia) { e.orden.Contenido = []byte("<html>no</html>") },
		"firmado = original": func(e *escenarioCustodia) {
			s := sha256.Sum256(e.orden.Contenido)
			e.orden.HuellaOriginalSHA256 = hex.EncodeToString(s[:])
		},
	}
	for nombre, alterar := range casos {
		e := nuevoEscenarioCustodia(t)
		alterar(&e)
		if _, err := e.servicio.CustodiarFirmado(context.Background(), e.orden); err == nil || e.repo.llamadas != 0 {
			t.Errorf("%s: se custodió (%v, confirmaciones=%d)", nombre, err, e.repo.llamadas)
		}
	}
}

// Una repetición con otra decisión V3 (respuesta perdida) debe poder
// escribir: el almacén liga su clave a la concesión, y con la clave del
// documento rechazaría el reintento para siempre.
func TestCustodiaFirmadoRecuperaConOtraDecision(t *testing.T) {
	e := nuevoEscenarioCustodia(t)
	if _, err := e.servicio.CustodiarFirmado(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	primero := e.repo.persistente.Objeto.Objeto.Objeto
	e.reautorizar(t, e.ahora, refPrueba("decision-repetida", "1"))
	if _, err := e.servicio.CustodiarFirmado(context.Background(), e.orden); err != nil || e.repo.llamadas != 2 {
		t.Fatalf("recuperación con otra decisión: %v (%d)", err, e.repo.llamadas)
	}
	if e.repo.persistente.Objeto.Objeto.Objeto == primero {
		t.Fatal("otra decisión debe escribir otro objeto, no reutilizar la concesión anterior")
	}
}

func TestCustodiaFirmadoPropagaElFalloDeLaConfirmacion(t *testing.T) {
	e := nuevoEscenarioCustodia(t)
	e.repo.err = errors.New("sql caído")
	if _, err := e.servicio.CustodiarFirmado(context.Background(), e.orden); err == nil {
		t.Fatal("un fallo de la confirmación no puede darse por custodia")
	}
}

// repositorioGenericoNoAlcanzable deja disponible el servicio genérico.
type repositorioGenericoNoAlcanzable struct{ docports.Repositorio }

// politicasEspia cuenta las resoluciones de política: el rechazo del tipo
// reservado debe llegar antes. Conserva la reserva del catálogo real.
type politicasEspia struct {
	*conservacion.Catalogo
	busquedas int
}

func (p *politicasEspia) BuscarPoliticasConservacionDocumental(ctx context.Context, s vecports.SolicitudPoliticaConservacionDocumental) ([]vecports.PoliticaConservacionDocumental, error) {
	p.busquedas++
	return p.Catalogo.BuscarPoliticasConservacionDocumental(ctx, s)
}

func escenarioConEspia(t *testing.T) (escenarioCustodia, *politicasEspia) {
	t.Helper()
	e := nuevoEscenarioCustodia(t)
	espia := &politicasEspia{Catalogo: e.servicio.Politicas.(*conservacion.Catalogo)}
	e.servicio.Politicas = espia
	e.servicio.Repositorio = repositorioGenericoNoAlcanzable{}
	return e, espia
}

func TestAltaGenericaRechazaElTipoReservado(t *testing.T) {
	e, espia := escenarioConEspia(t)
	autorizacion := e.orden.Autorizacion
	autorizacion.Accion, autorizacion.Finalidad = docports.AccionAlta, "alta_documento_generado"
	_, err := e.servicio.AltaGenerado(context.Background(), docports.AltaGenerado{
		ID: e.orden.ID, ClaveIdempotencia: e.orden.ClaveIdempotencia, ModuloID: e.orden.ModuloID,
		ExpedienteRef: e.orden.ExpedienteRef, TipoRef: e.orden.TipoRef, Version: 1, MIME: "application/pdf",
		Contenido: e.orden.Contenido, SolicitudPolitica: e.orden.SolicitudPolitica, Autorizacion: autorizacion,
	})
	if !errors.Is(err, docports.ErrSolicitudInvalida) || espia.busquedas != 0 {
		t.Fatalf("el alta genérica del tipo reservado debe rechazarse antes de resolver la política: %v (%d)", err, espia.busquedas)
	}
}

type autorizadorEspia struct{ llamadas int }

func (a *autorizadorEspia) AutorizarRegistroExterno(context.Context, []byte, string, string) (docports.AutorizacionV3, error) {
	a.llamadas++
	return docports.AutorizacionV3{}, errors.New("no debe pedirse")
}

func TestRegistroExternoRechazaElTipoReservado(t *testing.T) {
	e, espia := escenarioConEspia(t)
	autorizador := &autorizadorEspia{}
	_, err := e.servicio.RegistrarExternoAutorizado(context.Background(), docports.AltaExterna{
		ID: e.orden.ID, ClaveIdempotencia: e.orden.ClaveIdempotencia, ModuloID: e.orden.ModuloID,
		ExpedienteRef: e.orden.ExpedienteRef, TipoRef: e.orden.TipoRef, Version: 1,
		Custodia:          domain.ReferenciaCustodiaExterna{CustodioID: "registro_entrada", Referencia: "justificante:reservado", HuellaSHA256: strings.Repeat("f", 64)},
		SolicitudPolitica: e.orden.SolicitudPolitica,
	}, autorizador)
	if !errors.Is(err, docports.ErrSolicitudInvalida) || autorizador.llamadas != 0 || espia.busquedas != 0 {
		t.Fatalf("el registro externo del tipo reservado debe rechazarse antes de autorizar: %v (%d, %d)", err, autorizador.llamadas, espia.busquedas)
	}
}

// TestCustodiaFirmadoPG18 recorre servicio, concesión de almacén V3, almacén
// de ficheros y vec_documentos.custodiar_firmado_v1 real sobre la base
// desechable de probar_integracion_pg18.sh (fachadas AD3 sintéticas). Sin
// VEC_DOCUMENTOS_PG18_DSN no se ejecuta.
func TestCustodiaFirmadoPG18(t *testing.T) {
	dsn := os.Getenv("VEC_DOCUMENTOS_PG18_DSN")
	if dsn == "" {
		t.Skip("sin base PG18 desechable de Documentos")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := docpostgres.NuevoRepositorio(pool)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	sufijo := ahora.Format(time.RFC3339Nano)
	e := nuevoEscenarioCustodiaEn(t, ahora, sufijo)
	e.servicio.RepositorioCustodia = repo
	d, err := e.servicio.CustodiarFirmado(ctx, e.orden)
	if err != nil || d.Custodia != domain.CustodiaVEC || d.ObjetoVersion == "" || !d.Descargable() {
		t.Fatalf("custodia en PG18: %v %+v", err, d)
	}
	// Repetición con otra decisión V3 (respuesta perdida): el almacén escribe
	// otro objeto con otra clave y SQL devuelve el documento original.
	e.reautorizar(t, ahora, refPrueba("decision-repetida", sufijo))
	r, err := e.servicio.CustodiarFirmado(ctx, e.orden)
	if err != nil || r.ID != d.ID || r.NumeroVEC != d.NumeroVEC || !r.CreadoEn.Equal(d.CreadoEn) ||
		r.ObjetoRef != d.ObjetoRef || r.ObjetoVersion != d.ObjetoVersion {
		t.Fatalf("repetición en PG18: %v %+v", err, r)
	}
	// La misma clave con otro PDF firmado es conflicto y no crea otro documento.
	otro := nuevoEscenarioCustodiaEn(t, ahora, sufijo)
	otro.servicio.RepositorioCustodia = repo
	otro.orden.Contenido = append(append([]byte(nil), e.orden.Contenido...), " otra firma"...)
	suma := sha256.Sum256(otro.orden.Contenido)
	p := docports.CustodiaFirmadoPersistente{ID: otro.orden.ID, ClaveIdempotencia: otro.orden.ClaveIdempotencia,
		ModuloID: otro.orden.ModuloID, ExpedienteRef: otro.orden.ExpedienteRef, TipoRef: otro.orden.TipoRef,
		Version: otro.orden.Version, HuellaSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(otro.orden.Contenido)),
		HuellaOriginalSHA256: otro.orden.HuellaOriginalSHA256, FirmaOperacionRef: otro.orden.FirmaOperacionRef,
		Politica: politicaResuelta(t, otro)}
	if otro.preimagen, err = p.PreimagenCustodia(); err != nil {
		t.Fatal(err)
	}
	otro.reautorizar(t, ahora, refPrueba("decision-otro", sufijo))
	if _, err := otro.servicio.CustodiarFirmado(ctx, otro.orden); !errors.Is(err, docports.ErrConflicto) {
		t.Fatalf("otro PDF con la misma clave: %v", err)
	}
}

func politicaResuelta(t *testing.T, e escenarioCustodia) vecports.ResultadoPoliticaConservacionDocumental {
	t.Helper()
	catalogo, ok := e.servicio.Politicas.(*conservacion.Catalogo)
	if !ok {
		t.Fatal("catálogo de conservación esperado")
	}
	r, err := vecports.NuevoResultadoPoliticaConservacionDocumental(politicaDe(t, catalogo, e.orden.SolicitudPolitica),
		e.orden.SolicitudPolitica, e.servicio.Reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	return r
}
