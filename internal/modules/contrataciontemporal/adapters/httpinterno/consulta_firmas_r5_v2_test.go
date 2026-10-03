package httpinterno

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
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
		"contexto:consulta-v2", strings.Repeat("c", 64), ct.AccionConsultarFirmasR5V2, r.Referencia, h, ct.AudienciaConsultaFirmasR5V2, ahora, ahora.Add(time.Minute))
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
}

func (r *registroConsultaV2Prueba) RegistrarFirmaVerificadaV2(context.Context, ct.MaterialFirmaVerificadaV2, ct.CapacidadFirmaVerificadaV2) (ct.ReciboFirmaDocumento, error) {
	r.escrituras++
	return ct.ReciboFirmaDocumento{}, errors.New("prohibido registrar")
}
func (r *registroConsultaV2Prueba) ConsultarFirmasAutorizadasV2(context.Context, ct.MaterialConsultaFirmasR5V2, ct.CapacidadConsultaFirmasR5V2) (ct.LecturaFirmasR5V2, error) {
	r.llamadas++
	r.cerrado = true
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
