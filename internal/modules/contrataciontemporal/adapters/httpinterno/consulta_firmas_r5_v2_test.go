package httpinterno

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	cd "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Fuentes y capacidades de contrato unitarias, sin infraestructura productiva.
type fuenteConsultaV2Prueba struct {
	llamadas int
	err      error
}

func (f *fuenteConsultaV2Prueba) ResolverContextoConsultaFirmasR5V2(context.Context) (consultafirmasv2.Contexto, error) {
	f.llamadas++
	return consultafirmasv2.Contexto{OrganizacionRef: "organizacion:central", FirmantePrincipalCandidatoRef: "per_candidato_nominal"}, f.err
}

type autorizaConsultaV2Prueba struct {
	t        *testing.T
	llamadas int
	err      error
}

func (a *autorizaConsultaV2Prueba) AutorizarConsultaFirmasR5V2(_ context.Context, m ct.MaterialConsultaFirmasR5V2) (ct.CapacidadConsultaFirmasR5V2, error) {
	a.llamadas++
	if a.err != nil {
		return ct.CapacidadConsultaFirmasR5V2{}, a.err
	}
	r, e := firmaautorizacionv2.RecursoConsultaFirmasR5V2(m)
	if e != nil {
		a.t.Fatal(e)
	}
	h, e := r.HuellaContextoAutorizacionSHA256()
	if e != nil {
		a.t.Fatal(e)
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	res, e := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:consulta-v2", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"contexto:consulta-v2", strings.Repeat("c", 64), ct.AccionConsultarFirmasR5V2, r.Referencia, h, ct.AudienciaConsultaFirmasR5V2, ahora, ahora.Add(5*time.Second))
	if e != nil {
		a.t.Fatal(e)
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{5}, ed25519.SeedSize))
	raiz, e := x509.MarshalPKIXPublicKey(clave.Public())
	if e != nil {
		a.t.Fatal(e)
	}
	exportado, e := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{4}, vp.TamanoMinimoCapacidadCanonicaV3), res,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if e != nil {
		a.t.Fatal(e)
	}
	return ct.TransportarMaterialConsultaFirmasR5V2(exportado), nil
}

type registroConsultaV2Prueba struct {
	err                  error
	llamadas, escrituras int
	cerrado              bool
	proyeccionInvalida   bool
	lectura              *ct.LecturaFirmasR5V2
	cancelar             context.CancelFunc
}

func (r *registroConsultaV2Prueba) RegistrarFirmaVerificadaV2(context.Context, ct.MaterialFirmaVerificadaV2, ct.CapacidadFirmaVerificadaV2) (ct.ReciboFirmaDocumento, error) {
	r.escrituras++
	return ct.ReciboFirmaDocumento{}, errors.New("prohibido registrar")
}
func (r *registroConsultaV2Prueba) ConsultarFirmasAutorizadasV2(context.Context, ct.MaterialConsultaFirmasR5V2, ct.CapacidadConsultaFirmasR5V2) (ct.LecturaFirmasR5V2, error) {
	r.llamadas++
	r.cerrado = true
	if r.cancelar != nil {
		r.cancelar()
	}
	if r.lectura != nil {
		return *r.lectura, r.err
	}
	if r.proyeccionInvalida {
		return ct.LecturaFirmasR5V2{}, nil
	}
	return ct.LecturaFirmasR5V2{LecturaFirmasR5: ct.LecturaFirmasR5{HistoriaRevision: 0, HistoriaHuella: strings.Repeat("a", 64)}}, r.err
}

type fabricaConsultaV2Prueba struct {
	t                      *testing.T
	registro               *registroConsultaV2Prueba
	fallo, requiereCerrado bool
	llamadas               int
	recursoSolicitado      string
}

func (f *fabricaConsultaV2Prueba) CrearOrdenIntentoFirma(ctx context.Context, intento, accion, recurso string, resultado vd.ResultadoIntentoAuditoria) (vp.OrdenIntentoAuditoria, error) {
	f.llamadas++
	f.recursoSolicitado = recurso
	if f.fallo {
		return vp.OrdenIntentoAuditoria{}, errors.New("fuente nominal ausente")
	}
	if f.requiereCerrado && !f.registro.cerrado {
		f.t.Fatal("audita lector antes de terminar transacción")
	}
	if recurso == "" {
		recurso = "recurso:gobernado:consulta-v2"
	} // Sólo la fuente sintética de este test.
	ref, e := vp.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if e != nil {
		return vp.OrdenIntentoAuditoria{}, e
	}
	correlacion, e := ref.ValorCanonico()
	if e != nil {
		return vp.OrdenIntentoAuditoria{}, e
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	r, v, e := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", vd.AuthMethodCertificate, vd.AuthAssuranceHigh)
	if e != nil {
		f.t.Fatal(e)
	}
	return vp.NuevaOrdenIntentoAuditoria(intento, r, v, vd.DatosIntentoAuditoria{Accion: accion, ModuloID: ct.ModuloContratacion,
		RecursoRef: recurso, FinalidadRef: "contratacion_temporal", Resultado: resultado, Proceso: "vec-server", Canal: "interna_corporativa", CorrelacionRef: correlacion,
		Motivo: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_auditoria", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "acceso_denegado"}})
}

type intentosConsultaV2Prueba struct {
	t                 *testing.T
	fallo, acuseAjeno bool
	llamadas          int
}

func (a *intentosConsultaV2Prueba) AppendIntentoAuditoria(_ context.Context, o vp.OrdenIntentoAuditoria) (vp.AcuseIntentoAuditoria, error) {
	a.llamadas++
	if a.fallo {
		return vp.AcuseIntentoAuditoria{}, errors.New("commit auditoria no confirmado")
	}
	d, e := o.Datos()
	if e != nil {
		a.t.Fatal(e)
	}
	correlacion := d.Datos.CorrelacionRef
	if a.acuseAjeno {
		correlacion = "correlacion_22222222222222222222222222222222"
	}
	return vp.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_11111111111111111111111111111111", Secuencia: 1, HuellaSHA256: strings.Repeat("b", 64),
		CorrelacionRef: correlacion, RegistradaEn: time.Date(2026, 10, 3, 10, 0, 1, 0, time.UTC)}, nil
}
func manejadorConsultaV2Prueba(t *testing.T) (http.Handler, *fuenteConsultaV2Prueba, *autorizaConsultaV2Prueba, *registroConsultaV2Prueba, *intentosConsultaV2Prueba, *fabricaConsultaV2Prueba) {
	f := &fuenteConsultaV2Prueba{}
	a := &autorizaConsultaV2Prueba{t: t}
	r := &registroConsultaV2Prueba{}
	i := &intentosConsultaV2Prueba{t: t}
	fab := &fabricaConsultaV2Prueba{t: t, registro: r}
	h, e := NuevoManejadorConsultaFirmasR5V2(f, a, r, i, fab)
	if e != nil {
		t.Fatal(e)
	}
	return h, f, a, r, i, fab
}
func peticionConsultaV2(t *testing.T, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, RutaConsultaFirmasR5V2, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	ctx, e := vp.ConCorrelacionIncidenciasPeticion(r.Context())
	if e != nil {
		t.Fatal(e)
	}
	return r.WithContext(ctx)
}
func cuerpoConsultaV2() string {
	return `{"expediente_ref":"expediente:prueba","version_expediente":7,"documento":"informe_definitivo","paso_orden":1,"clave_idempotencia":"clave-prueba-000001","catalogo_huella":"` + strings.Repeat("a", 64) + `","via":"certificado_vec"}`
}
func TestConsultaV2HTTPPositivaNoEscribeNiInfiereEncontrado(t *testing.T) {
	h, f, a, r, i, fab := manejadorConsultaV2Prueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConsultaV2(t, cuerpoConsultaV2()))
	if w.Code != 200 || f.llamadas != 1 || a.llamadas != 1 || r.llamadas != 1 || r.escrituras != 0 || i.llamadas != 0 || fab.llamadas != 0 {
		t.Fatalf("composición inválida: %d %s", w.Code, w.Body.String())
	}
	var datos map[string]any
	if json.Unmarshal(w.Body.Bytes(), &datos) != nil {
		t.Fatal("no JSON")
	}
	d := datos["data"].(map[string]any)
	if d["esquema"] != EsquemaConsultaFirmasR5V2 || d["recuperacion"] != "parcial" || d["firma_eficaz"] != false || len(d["firmas"].([]any)) != 0 {
		t.Fatal("colección vacía o límites sustituidos")
	}
	for _, prohibido := range []string{"encontrado", "FirmanteRef", "CertificadoHuella", "clave_idempotencia", "actor", "perfil", "EvidenciaFirmasCanonica"} {
		if strings.Contains(w.Body.String(), prohibido) {
			t.Fatalf("campo no minimizado: %s", prohibido)
		}
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Fatal("cookie")
	}
}
func TestConsultaV2HTTPCuerpoCerradoAuditadoAntesFuente(t *testing.T) {
	for _, extra := range []string{`,"organizacion_ref":"organizacion:cliente"`, `,"firmante_principal_candidato_ref":"per_cliente"`, `,"perfil_ref":"perfil:cliente"`, `,"encontrado":true`, `,"documento":"resolucion"`} {
		h, f, a, r, i, fab := manejadorConsultaV2Prueba(t)
		w := httptest.NewRecorder()
		cuerpo := strings.TrimSuffix(cuerpoConsultaV2(), "}") + extra + "}"
		h.ServeHTTP(w, peticionConsultaV2(t, cuerpo))
		if w.Code != 400 || f.llamadas != 0 || a.llamadas != 0 || r.llamadas != 0 || i.llamadas != 1 || fab.recursoSolicitado != "" {
			t.Fatalf("campos extra admitidos/no auditados: %d %s", w.Code, w.Body.String())
		}
	}
}
func TestConsultaV2HTTPAuditoriaFallidaCierraAntesFuente(t *testing.T) {
	for _, modo := range []string{"escritor", "acuse", "fuente"} {
		h, f, _, _, i, fab := manejadorConsultaV2Prueba(t)
		switch modo {
		case "escritor":
			i.fallo = true
		case "acuse":
			i.acuseAjeno = true
		case "fuente":
			fab.fallo = true
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionConsultaV2(t, `{"actor":"no-admitido"}`))
		if w.Code != 503 || f.llamadas != 0 || strings.Contains(w.Body.String(), "no-admitido") {
			t.Fatalf("fallo auditoria no cerrado: %d", w.Code)
		}
	}
}
func TestConsultaV2HTTPDenegacionFuenteExigeAcuseYNoLector(t *testing.T) {
	for _, falloAuditoria := range []bool{false, true} {
		h, f, a, r, i, fab := manejadorConsultaV2Prueba(t)
		f.err = ct.ErrFirmaDocumentoDenegada
		i.fallo = falloAuditoria
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionConsultaV2(t, cuerpoConsultaV2()))
		esperado := 404
		if falloAuditoria {
			esperado = 503
		}
		if w.Code != esperado || a.llamadas != 0 || r.llamadas != 0 || i.llamadas != 1 || fab.recursoSolicitado != "expediente:prueba" {
			t.Fatalf("denegación sin acuse: %d", w.Code)
		}
	}
}
func TestConsultaV2HTTPFalloLectorAuditaUnaVezDespuesTX(t *testing.T) {
	h, _, _, r, i, fab := manejadorConsultaV2Prueba(t)
	r.err = ct.ErrFirmaDocumentoDenegada
	fab.requiereCerrado = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConsultaV2(t, cuerpoConsultaV2()))
	if w.Code != 404 || !r.cerrado || i.llamadas != 1 || fab.llamadas != 1 || r.escrituras != 0 {
		t.Fatalf("auditoría duplicada o antesTX: %d", w.Code)
	}
}
func TestConsultaV2HTTPConstructorRechazaAuditoriaTypedNil(t *testing.T) {
	var i *intentosConsultaV2Prueba
	var fab *fabricaConsultaV2Prueba
	var f *fuenteConsultaV2Prueba
	var a *autorizaConsultaV2Prueba
	var r *registroConsultaV2Prueba
	if _, e := NuevoManejadorConsultaFirmasR5V2(f, a, r, i, fab); !errors.Is(e, ct.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatal("acepta autoridad nula")
	}
	h, _, _, _, _, _ := manejadorConsultaV2Prueba(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, RutaConsultaFirmasR5V2, strings.NewReader(`{}`)))
	if w.Code != 503 {
		t.Fatal("sin contexto nominal no cierra")
	}
}

func TestConsultaV2HTTPProyeccionInvalidaAuditaErrorTrasLectura(t *testing.T) {
	h, _, _, r, i, fab := manejadorConsultaV2Prueba(t)
	r.proyeccionInvalida = true
	fab.requiereCerrado = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConsultaV2(t, cuerpoConsultaV2()))
	if w.Code != 502 || !r.cerrado || i.llamadas != 1 || fab.llamadas != 1 || r.escrituras != 0 {
		t.Fatalf("resultado inválido sin auditoría: %d", w.Code)
	}
}

func TestConsultaV2HTTPOverflowAuditaAntesDeResponder(t *testing.T) {
	h, _, _, r, i, fab := manejadorConsultaV2Prueba(t)
	fab.requiereCerrado = true
	l := ct.LecturaFirmasR5V2{LecturaFirmasR5: ct.LecturaFirmasR5{HistoriaRevision: 1, HistoriaHuella: strings.Repeat("f", 64)}}
	evidencia := json.RawMessage(`[{}]`)
	hash := sha256.Sum256(evidencia)
	ref := func(prefijo string, n int) string { return prefijo + strings.Repeat("x", 135) + fmt.Sprint(n) }
	for n := 0; n < consultafirmasv2.MaximoFilas; n++ {
		f := ct.FirmaRegistrada{Via: ct.ViaFirmaCertificadoVEC, FirmaRef: ref("firma:", n), ReciboRef: ref("recibo:", n), PasoRef: ref("paso:", n),
			Documento: "informe_definitivo", Secuencia: n + 1, PasoOrden: 1, ExpedienteVersion: 7, CatalogoRef: ref("catalogo:", n), CatalogoHuella: strings.Repeat("a", 64),
			Resultado: cd.ResultadoFirmaFirmado, RegistradaEn: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), OriginalRef: ref("original:", n), OriginalVersion: 7,
			OriginalHuella: strings.Repeat("b", 64), FirmadoHuella: strings.Repeat("c", 64), DocumentoCustodiaRef: ref("custodia:", n), DocumentoCustodiaVersion: 1}
		v := ct.FirmaRegistradaRevisionPDFV2{FirmaRegistrada: f, EntradaDocumentoRef: f.OriginalRef, EntradaDocumentoVersion: 7,
			EntradaDocumentoHuella: f.OriginalHuella, EntradaDocumentoLongitud: 100, OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 100, 200, 100},
			RevisionLongitud: 300, RevisionHuellaSHA256: f.FirmadoHuella, ContenidoFirmadoHuellaSHA256: strings.Repeat("d", 64),
			EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(hash[:])}
		l.Firmas = append(l.Firmas, f)
		l.RevisionesPDF = append(l.RevisionesPDF, v)
	}
	r.lectura = &l
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionConsultaV2(t, cuerpoConsultaV2()))
	if w.Code != 502 || i.llamadas != 1 || fab.llamadas != 1 || !r.cerrado || r.escrituras != 0 {
		t.Fatalf("desbordamiento no auditado: HTTP%d bytes%d append%d", w.Code, w.Body.Len(), i.llamadas)
	}
	if strings.Contains(w.Body.String(), "firma:xxxxx") {
		t.Fatal("desbordamiento expone metadatos")
	}
}

func TestConsultaV2HTTPCancelacionNoOcultaAuditoriaFallida(t *testing.T) {
	h, _, _, r, i, fab := manejadorConsultaV2Prueba(t)
	r.err = ct.ErrFirmaDocumentoDenegada
	i.fallo = true
	fab.requiereCerrado = true
	peticion := peticionConsultaV2(t, cuerpoConsultaV2())
	ctx, cancelar := context.WithCancel(peticion.Context())
	defer cancelar()
	r.cancelar = cancelar
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticion.WithContext(ctx))
	if w.Code != http.StatusServiceUnavailable || !r.cerrado || i.llamadas != 1 || fab.llamadas != 1 || r.escrituras != 0 {
		t.Fatalf("cancelación oculta auditoría fallida: HTTP%d append%d fabrica%d", w.Code, i.llamadas, fab.llamadas)
	}
	var respuesta map[string]any
	if json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta["data"] != nil {
		t.Fatal("fallo expone datos")
	}
}
