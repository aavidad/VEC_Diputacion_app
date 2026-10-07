package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/almacen/ficheros"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	dochttp "vec-diputacion-granada/internal/vec/documentos/adapters/httpinterno"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Recorrido de extremo a extremo de 5.06 en PostgreSQL 18.4, sobre un clon de
// la principal con AD3-113, Documentos 000009 y CT145 instaladas
// (deploy/postgresql/contratacion_temporal/probar_custodia_firmado_pg18.sh):
//
//	POST firma (HTTP de CT) → verificación (cliente real del validador contra
//	su servidor de prueba) → custodia en Documentos (adaptador de la
//	composición, servicio, fábrica de concesiones, almacén de ficheros y
//	custodiar_firmado_v1) → registro con enlace (registrar_firma_documento_v2)
//	→ consulta HTTP del circuito con el documento → descarga HTTP de
//	Documentos → reintento de la misma firma sin duplicar nada.
//
// Solo son dobles el PDP (sin COSE real: las fachadas AD3 del clon los
// sustituye el guion) y el validador externo. El predicado del PDP de CT para
// la custodia sí es el real.

const organizacionE2E = organizacionAltaContratacionTemporalDesarrollo

type relojE2E struct{}

func (relojE2E) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

func dsnE2E(t *testing.T, variable string) string {
	t.Helper()
	dsn := os.Getenv(variable)
	if dsn == "" {
		t.Skip("sin clon PG18 del recorrido de custodia (" + variable + ")")
	}
	return dsn
}

// actorE2E es la identidad sintética de la petición: la misma en la firma de
// CT, en la custodia y en la descarga.
type actorE2E struct {
	persona, perfil, principal, perfilActivo string
	ahora                                    time.Time
}

func nuevoActorE2E(t *testing.T) actorE2E {
	t.Helper()
	// Las concesiones de prueba se registran un segundo después de su
	// instante: se emiten unos segundos antes para que ya estén vigentes.
	a := actorE2E{persona: "per_e2e5060123456789abcdef", perfil: "prf_e2e5060123456789abcdef", ahora: time.Now().UTC().Add(-5 * time.Second)}
	_, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(a.ahora, a.persona, a.perfil, dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	a.principal, a.perfilActivo = datos.PrincipalID, datos.PerfilActivoRef
	return a
}

func correlacionE2E(t *testing.T) string {
	c, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// materialE2E es un material V3 con la capacidad y la decisión en JSON: lo
// que cotejan las funciones SQL y las fachadas dobles del guion.
func materialE2E(t *testing.T, audiencia, accion, efectoRef, huellaEfecto string, decision map[string]any) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		t.Fatal(err)
	}
	decisionRef := "decision:e2e:" + hex.EncodeToString(aleatorio)
	decision["decision_ref"] = decisionRef
	capacidad, _ := json.Marshal(map[string]any{"audiencia_consumo": audiencia, "operacion": accion, "efecto_ref": efectoRef,
		"huella_efecto_sha256": huellaEfecto, "decision_ref": decisionRef, "relleno": strings.Repeat("r", puertosvec.TamanoMinimoCapacidadCanonicaV3)})
	cuerpo, _ := json.Marshal(decision)
	h := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, h, h, "contexto:e2e", h,
		accion, efectoRef, huellaEfecto, audiencia, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, _ := ed25519.GenerateKey(nil)
	spki, _ := x509.MarshalPKIXPublicKey(publica)
	m, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, cuerpo,
		[]byte("{}"), []byte("{}"), 1, 1, []byte("p"), []byte("s"), []byte("e"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// pdpE2E hace de PDP de CT: aplica el predicado real de la custodia y emite
// concesiones y materiales sintéticos para la identidad de la petición.
type pdpE2E struct {
	t     *testing.T
	actor actorE2E
	mu    sync.Mutex
	// denegadas cuenta las solicitudes que el predicado real rechazó.
	denegadas, concedidas int
}

func (p *pdpE2E) solicitarCustodiaV3(ctx context.Context, recurso dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: docports.AccionCustodiarFirmado, Finalidad: docports.FinalidadCustodiarFirmado,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: recurso}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !solicitudAutorizacionCustodiaFirmadoCTDesarrolloValida(ctx, datos) {
		p.denegadas++
		return solicitudCustodiaCTDesarrollo{}, errCustodiaFirmadoCTDenegada
	}
	c, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: p.actor.ahora, PersonaRef: p.actor.persona,
		PerfilRef: p.actor.perfil, Accion: docports.AccionCustodiarFirmado, Recurso: recurso, Finalidad: docports.FinalidadCustodiarFirmado,
		Campos: []string{"documento_firmado.custodia", "evidencia_custodia"}, DecisionRef: "dec_" + strings.Repeat("c", 32)})
	if err != nil {
		return solicitudCustodiaCTDesarrollo{}, err
	}
	p.concedidas++
	return solicitudCustodiaCTDesarrollo{solicitud: c.Solicitud, decision: c.Decision, confirmacion: c.Confirmacion,
		actor:       dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: p.actor.principal, PerfilActivoRef: p.actor.perfilActivo},
		correlacion: correlacionE2E(p.t)}, nil
}

func (p *pdpE2E) materialCustodiaV3(_ context.Context, s solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := s.solicitud.Datos()
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huella, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return materialE2E(p.t, docports.AudienciaV3, docports.AccionCustodiarFirmado, datos.Recurso.Referencia, huella, map[string]any{
		"accion": docports.AccionCustodiarFirmado, "modulo_id": "documentos", "tipo_recurso": "documento_firmado",
		"finalidad": docports.FinalidadCustodiarFirmado, "campos_permitidos": []string{"documento_firmado.custodia", "evidencia_custodia"},
		"obligaciones": []string{}, "recurso_ref": datos.Recurso.Referencia, "contexto_recurso_huella_sha256": huella,
		"principal_id": p.actor.principal, "perfil_activo_ref": p.actor.perfilActivo, "correlacion_ref": s.correlacion,
	}), nil
}

// autorizadorFirmaE2E emite la V3 de la firma de CT (AD3-85 doble en SQL).
type autorizadorFirmaE2E struct {
	t     *testing.T
	actor actorE2E
}

func (a autorizadorFirmaE2E) AutorizarFirmaDocumento(_ context.Context, m ctports.MaterialFirmaDocumento) (ctports.CapacidadFirmaDocumento, error) {
	recurso, err := ctapplication.RecursoFirmaDocumento(m)
	if err != nil {
		return ctports.CapacidadFirmaDocumento{}, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return ctports.CapacidadFirmaDocumento{}, err
	}
	return ctports.TransportarMaterialFirmaDocumento(materialE2E(a.t, ctports.AudienciaFirmaDocumentoV3, ctports.AccionFirmarDocumento,
		recurso.Referencia, huella, map[string]any{
			"accion": ctports.AccionFirmarDocumento, "modulo_id": ctports.ModuloContratacion, "tipo_recurso": ctports.TipoRecursoFirmaDocumento,
			"finalidad": ctports.FinalidadFirmaDocumento, "recurso_ref": recurso.Referencia, "contexto_recurso_huella_sha256": huella,
			"principal_id": a.actor.principal, "perfil_activo_ref": a.actor.perfilActivo,
		})), nil
}

type circuitoE2E struct{}

func (circuitoE2E) CircuitoFirma(context.Context) (ctdomain.CircuitoFirma, error) {
	return ctdomain.CircuitoFirma{CatalogoRef: "vec.contratacion_temporal.circuito_firma:e2e", HuellaCatalogo: strings.Repeat("e", 64), Ejemplo: true,
		Documentos: []ctdomain.CircuitoFirmaDocumento{{Documento: "resolucion", Etiqueta: "Resolución", Pasos: []ctdomain.PasoCircuitoFirma{
			{Orden: 1, Cargo: "Órgano", PerfilRef: "perfil:ct:organo", Accion: "firma", Devolucion: ctdomain.DevolucionVuelveARedaccion,
				Referencia: "vec.contratacion_temporal.circuito_firma:e2e:resolucion.p1"}}}}}, nil
}

type canalE2E struct{}

func (canalE2E) ResolverOrganizacionFirmaDocumento(context.Context) (string, error) {
	return organizacionE2E, nil
}

// emisorLecturaE2E y autoridadDescargaE2E dan la concesión de lectura del
// almacén y la V3 de la descarga (fachada doble en SQL).
type emisorLecturaE2E struct{ actor actorE2E }

func (e emisorLecturaE2E) SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (docautorizacion.DatosSeudonimosLectura, error) {
	return docautorizacion.DatosSeudonimosLectura{SujetoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		SolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64)}, nil
}

func (e emisorLecturaE2E) EmitirConcesionAlmacenV3(_ context.Context, s docautorizacion.SolicitudConcesionAlmacenV3) (docautorizacion.ConcesionAlmacenV3, error) {
	c, err := pruebas.NuevaConcesionV3Prueba(pruebas.DatosConcesionV3Prueba{Instante: e.actor.ahora, PersonaRef: e.actor.persona, PerfilRef: e.actor.perfil,
		Accion: s.Accion, Recurso: s.Recurso, Finalidad: s.Finalidad, Campos: []string{"contenido", "documento"}, DecisionRef: "dec_" + strings.Repeat("d", 32)})
	return docautorizacion.ConcesionAlmacenV3{Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion}, err
}

type autoridadDescargaE2E struct {
	t     *testing.T
	actor actorE2E
}

func (a autoridadDescargaE2E) autorizar(accion, finalidad, recursoRef, ambito string, preimagen []byte) (docports.AutorizacionV3, error) {
	recurso, err := docports.RecursoV3(accion, recursoRef, preimagen)
	if err != nil {
		return docports.AutorizacionV3{}, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	correlacion := correlacionE2E(a.t)
	tipo, _ := docports.TipoRecursoV3(accion)
	m := materialE2E(a.t, docports.AudienciaV3, accion, recursoRef, huella, map[string]any{
		"accion": accion, "modulo_id": "documentos", "tipo_recurso": tipo, "finalidad": finalidad,
		"campos_permitidos": []string{"contenido", "documento"}, "obligaciones": []string{}, "recurso_ref": recursoRef,
		"contexto_recurso_huella_sha256": huella, "principal_id": a.actor.principal, "perfil_activo_ref": a.actor.perfilActivo,
		"correlacion_ref": correlacion,
	})
	return docports.AutorizacionV3{Material: m, Accion: accion, Finalidad: finalidad, RecursoRef: recursoRef, AmbitoRef: ambito,
		PrincipalID: a.actor.principal, PerfilActivoRef: a.actor.perfilActivo, CorrelacionRef: correlacion}, nil
}

func (a autoridadDescargaE2E) ResolverConsultaExpediente(_ context.Context, c docports.ConsultaExpediente) (docports.AutorizacionV3, error) {
	p, err := c.PreimagenListar()
	if err != nil {
		return docports.AutorizacionV3{}, err
	}
	return a.autorizar(docports.AccionListar, "listar_documentos_expediente", c.ExpedienteRef, c.ExpedienteRef, p)
}

func (a autoridadDescargaE2E) ResolverDescargaOriginal(_ context.Context, c docports.ConsultaDocumento, expediente string) (docports.AutorizacionV3, error) {
	p, err := c.PreimagenDescargar()
	if err != nil {
		return docports.AutorizacionV3{}, err
	}
	return a.autorizar(docports.AccionDescargar, "descargar_documento_original", c.DocumentoID, expediente, p)
}

// pdfE2E es un PDF mínimo; el firmado añade la «firma» que el servidor de
// prueba del validador acredita (no verifica criptografía: es un doble).
func pdfE2E(texto string) []byte { return []byte("%PDF-1.7\n% " + texto + "\n%%EOF\n") }

func TestCustodiaFirmadoRecorridoPG18(t *testing.T) {
	admin := dsnE2E(t, "VEC_CUSTODIA_E2E_ADMIN_DSN")
	dsnCT := dsnE2E(t, "VEC_CUSTODIA_E2E_CT_DSN")
	dsnDoc := dsnE2E(t, "VEC_CUSTODIA_E2E_DOCUMENTOS_DSN")
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	abrir := func(dsn string) *pgxpool.Pool {
		p, err := pgxpool.New(ctx, dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	poolAdmin, poolCT, poolDoc := abrir(admin), abrir(dsnCT), abrir(dsnDoc)
	// Un expediente real del clon, de la organización de desarrollo, sin
	// firmas de la resolución.
	var expediente string
	var version uint64
	if err := poolAdmin.QueryRow(ctx, `SELECT a.expediente_ref, a.version::bigint FROM vec_contratacion_temporal.expediente_integral_actual a
		JOIN vec_contratacion_temporal.expediente_version_integral v USING (expediente_ref, version)
		WHERE v.agregado_json->>'organizacion_ref'=$1 AND NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.firma_documento_v1 f
		  WHERE f.expediente_ref=a.expediente_ref AND f.documento='resolucion') ORDER BY a.expediente_ref LIMIT 1`, organizacionE2E).Scan(&expediente, &version); err != nil {
		t.Fatalf("expediente del clon: %v", err)
	}
	actor := nuevoActorE2E(t)

	// Documentos: repositorio PG, catálogo provisional, almacén de ficheros.
	repoDoc, err := docpg.NuevoRepositorio(poolDoc)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := conservacion.NuevoCatalogoProvisional(relojE2E{})
	if err != nil {
		t.Fatal(err)
	}
	directorio := filepath.Join(t.TempDir(), "originales")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	almacen, err := ficheros.Nuevo(ficheros.Configuracion{ConectorID: "ficheros_e2e", Directorio: directorio, TamanoMaximo: 1 << 20}, relojE2E{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = almacen.Cerrar() })
	custodiaDoc, err := nuevaCustodiaDocumentosDesarrollo(&custodiaFirmadoConfigDesarrollo{Documentos: map[string]string{
		"resolucion": "contratacion_temporal.resolucion_firmada.v1"}}, repoDoc, almacen, catalogo, relojE2E{}, nuevoSeudonimizadorAlmacenDesarrollo([32]byte{7}))
	if err != nil {
		t.Fatal(err)
	}
	pdp := &pdpE2E{t: t, actor: actor}
	custodia, err := nuevaCustodiaFirmadoCTDesarrollo(pdp, custodiaDoc, relojE2E{})
	if err != nil {
		t.Fatal(err)
	}

	// Validador: cliente real contra su servidor de prueba.
	validador := servidorprueba.Nuevo(strings.Repeat("t", 40), false)
	defer validador.Close()
	verificador, err := validadorautofirma.Nuevo(validadorautofirma.Configuracion{URL: validador.URL,
		CAPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: validador.Certificate().Raw}),
		Token: []byte(strings.Repeat("t", 40)), Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	// Contratación temporal: registro PG (CT145), servicio y HTTP.
	registro, err := ctpostgres.NuevoRegistroFirmasDocumentoPostgreSQL(poolCT)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := ctapplication.NuevoServicioFirmaDocumento(circuitoE2E{}, registro, autorizadorFirmaE2E{t: t, actor: actor}, verificador)
	if err != nil {
		t.Fatal(err)
	}
	if err := servicio.ComponerCustodia(custodia, custodiaDoc.documentos); err != nil {
		t.Fatal(err)
	}
	manejador, err := cthttp.NuevoManejadorFirmaDocumento(canalE2E{}, servicio)
	if err != nil {
		t.Fatal(err)
	}
	pedir := func(ruta string, cuerpo any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(cuerpo)
		r := httptest.NewRequest(http.MethodPost, ruta, bytes.NewReader(b)).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		manejador.ServeHTTP(w, r)
		return w
	}

	// 1. Firma: se verifica, se custodia y se registra enlazada.
	original, firmado := pdfE2E("resolución de prueba"), pdfE2E("resolución de prueba · firma PAdES de prueba")
	firma := map[string]any{"expediente_ref": expediente, "version_expediente": version, "documento": "resolucion", "paso_orden": 1,
		"resultado": "firmado", "motivo_devolucion": "", "original_base64": base64.StdEncoding.EncodeToString(original),
		"firmado_base64": base64.StdEncoding.EncodeToString(firmado), "clave_idempotencia": "firma-e2e-" + hex.EncodeToString(actor.ahora.AppendFormat(nil, "150405.000000"))[:24]}
	w := pedir(cthttp.RutaFirmaDocumento, firma)
	var recibo struct {
		Data struct {
			ReciboRef    string `json:"recibo_ref"`
			YaRegistrada bool   `json:"ya_registrada"`
			Custodiado   struct {
				ExpedienteRef string `json:"expediente_ref"`
				DocumentoRef  string `json:"documento_ref"`
				Version       int    `json:"version"`
				Huella        string `json:"huella_sha256"`
			} `json:"documento_custodiado"`
		} `json:"data"`
	}
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &recibo) != nil || recibo.Data.Custodiado.DocumentoRef == "" {
		t.Fatalf("firma con custodia: %d %s (predicado denegó %d)", w.Code, w.Body, pdp.denegadas)
	}
	suma := sha256.Sum256(firmado)
	c := recibo.Data.Custodiado
	if c.Huella != hex.EncodeToString(suma[:]) || c.Version != 1 ||
		c.ExpedienteRef != ctports.ExpedienteDocumentalRef(organizacionE2E, expediente) ||
		c.DocumentoRef != ctports.DocumentoCustodiaRef(organizacionE2E, expediente, firma["clave_idempotencia"].(string)) ||
		pdp.concedidas != 2 || pdp.denegadas != 0 {
		t.Fatalf("documento custodiado inesperado: %+v (concedidas %d, denegadas %d)", c, pdp.concedidas, pdp.denegadas)
	}

	// 2. La consulta del circuito lo ofrece para descargar.
	w = pedir(cthttp.RutaConsultaFirmaDocumento, map[string]any{"expediente_ref": expediente})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"documento_ref":"`+c.DocumentoRef+`"`) ||
		!strings.Contains(w.Body.String(), `"estado":"firmado"`) {
		t.Fatalf("consulta del circuito: %d %s", w.Code, w.Body)
	}

	// 3. Descarga por la ruta de Documentos: los mismos bytes del firmado.
	lectura, err := docautorizacion.NuevaFabricaContextoLecturaOriginalV3(emisorLecturaE2E{actor: actor}, relojE2E{})
	if err != nil {
		t.Fatal(err)
	}
	servicioDoc := &docapp.Servicio{Repositorio: repoDoc, Almacen: almacen, Politicas: catalogo, Reloj: relojE2E{}, ContextosLectura: lectura}
	rutas, err := dochttp.NuevasRutas(dochttp.Configuracion{Servicio: servicioDoc, Autoridad: autoridadDescargaE2E{t: t, actor: actor}, DescargaDisponible: true})
	if err != nil {
		t.Fatal(err)
	}
	var descarga http.Handler
	for _, r := range rutas {
		if r.Ruta == dochttp.RutaDescargaOriginal {
			descarga = r.Manejador
		}
	}
	cuerpo, _ := json.Marshal(map[string]any{"expediente_ref": c.ExpedienteRef, "documento_ref": c.DocumentoRef, "version": c.Version})
	r := httptest.NewRequest(http.MethodPost, dochttp.RutaDescargaOriginal, bytes.NewReader(cuerpo)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	wd := httptest.NewRecorder()
	descarga.ServeHTTP(wd, r)
	if wd.Code != http.StatusOK || !bytes.Equal(wd.Body.Bytes(), firmado) || wd.Header().Get("Content-Type") != "application/pdf" ||
		wd.Header().Get("X-Content-SHA256") != c.Huella {
		t.Fatalf("descarga del firmado: %d %q %v", wd.Code, wd.Body.String(), wd.Header())
	}

	// 4. El reintento de la misma firma (respuesta perdida) devuelve el mismo
	// recibo y el mismo documento, sin nuevas filas de firma, enlace ni documento.
	w = pedir(cthttp.RutaFirmaDocumento, firma)
	var repetido = recibo
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &repetido) != nil || !repetido.Data.YaRegistrada ||
		repetido.Data.ReciboRef != recibo.Data.ReciboRef || repetido.Data.Custodiado != c {
		t.Fatalf("reintento: %d %s", w.Code, w.Body)
	}
	var firmas, enlaces, documentos int
	if err := poolAdmin.QueryRow(ctx, `SELECT (SELECT count(*) FROM vec_contratacion_temporal.firma_documento_v1 WHERE expediente_ref=$1 AND documento='resolucion'),
		(SELECT count(*) FROM vec_contratacion_temporal.firma_documento_custodia_v1 WHERE documento_ref=$2),
		(SELECT count(*) FROM vec_documentos.documento WHERE id=$2)`, expediente, c.DocumentoRef).Scan(&firmas, &enlaces, &documentos); err != nil {
		t.Fatal(err)
	}
	if firmas != 1 || enlaces != 1 || documentos != 1 {
		t.Fatalf("filas tras el reintento: firmas %d, enlaces %d, documentos %d", firmas, enlaces, documentos)
	}
}
