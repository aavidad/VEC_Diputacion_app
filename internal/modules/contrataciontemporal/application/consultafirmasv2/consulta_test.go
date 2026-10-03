package consultafirmasv2

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Dobles de contrato exclusivamente unitarios; no acreditan PDP, SQL ni E2E.
type fuentePrueba struct {
	c        Contexto
	err      error
	llamadas int
	despues  func()
}

func (f *fuentePrueba) ResolverContextoConsultaFirmasR5V2(context.Context) (Contexto, error) {
	f.llamadas++
	if f.despues != nil {
		f.despues()
	}
	return f.c, f.err
}

type autorizadorPrueba struct {
	t        *testing.T
	m        ports.MaterialConsultaFirmasR5V2
	err      error
	ajena    bool
	llamadas int
	despues  func()
}

func (a *autorizadorPrueba) AutorizarConsultaFirmasR5V2(_ context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	a.llamadas++
	a.m = m
	if a.despues != nil {
		a.despues()
	}
	if a.err != nil {
		return ports.CapacidadConsultaFirmasR5V2{}, a.err
	}
	if a.ajena {
		m.ExpedienteRef = "expediente:ajeno"
	}
	r, err := firmaautorizacionv2.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		a.t.Fatal(err)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		a.t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	res, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:consulta-v2", strings.Repeat("a", 64), strings.Repeat("b", 64),
		"contexto:consulta-v2", strings.Repeat("c", 64), ports.AccionConsultarFirmasR5V2, r.Referencia, h, ports.AudienciaConsultaFirmasR5V2, ahora, ahora.Add(5*time.Second))
	if err != nil {
		a.t.Fatal(err)
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{5}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		a.t.Fatal(err)
	}
	exportado, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{4}, vp.TamanoMinimoCapacidadCanonicaV3), res,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		a.t.Fatal(err)
	}
	return ports.TransportarMaterialConsultaFirmasR5V2(exportado), nil
}

type lectorPrueba struct {
	l        ports.LecturaFirmasR5V2
	err      error
	llamadas int
	despues  func()
}

func (l *lectorPrueba) ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	l.llamadas++
	if l.despues != nil {
		l.despues()
	}
	return l.l, l.err
}
func solicitudPrueba() Solicitud {
	return Solicitud{"expediente:prueba", 7, "informe_definitivo", 1, "clave-firma-prueba-001", strings.Repeat("a", 64), ports.ViaFirmaCertificadoVEC}
}
func lecturaPrueba() ports.LecturaFirmasR5V2 {
	f := ports.FirmaRegistrada{Via: ports.ViaFirmaCertificadoVEC, FirmaRef: "firma:primera", ReciboRef: "recibo:primero", Documento: "informe_definitivo",
		Secuencia: 1, ExpedienteVersion: 7, CatalogoRef: "catalogo:firmas:v2", CatalogoHuella: strings.Repeat("a", 64), PasoRef: "paso:primero", PasoOrden: 1,
		Resultado: domain.ResultadoFirmaFirmado, OriginalRef: "ref:" + strings.Repeat("b", 64), OriginalVersion: 7, OriginalHuella: strings.Repeat("b", 64),
		FirmadoHuella: strings.Repeat("c", 64), DocumentoCustodiaRef: "ref:" + strings.Repeat("c", 64), DocumentoCustodiaVersion: 1,
		RegistradaEn: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), ClaveIdempotencia: "clave-firma-no-exponer", ActorRef: "actor:no-exponer", PerfilRef: "perfil:no-exponer"}
	r := ports.FirmaRegistradaRevisionPDFV2{FirmaRegistrada: f, EntradaDocumentoRef: f.OriginalRef, EntradaDocumentoVersion: f.OriginalVersion,
		EntradaDocumentoHuella: f.OriginalHuella, EntradaDocumentoLongitud: 100, OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 100, 200, 100}, RevisionLongitud: 300,
		RevisionHuellaSHA256: f.FirmadoHuella, ContenidoFirmadoHuellaSHA256: strings.Repeat("d", 64), EvidenciaFirmasCanonica: json.RawMessage(`[{"privado":"evidencia-no-exponer"}]`)}
	h := sha256.Sum256(r.EvidenciaFirmasCanonica)
	r.EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(h[:])
	r.CertificadoHuella = strings.Repeat("e", 64)
	r.FirmanteRef = "ref:" + r.CertificadoHuella
	return ports.LecturaFirmasR5V2{LecturaFirmasR5: ports.LecturaFirmasR5{Firmas: []ports.FirmaRegistrada{f}, HistoriaRevision: 1, HistoriaHuella: strings.Repeat("f", 64)}, RevisionesPDF: []ports.FirmaRegistradaRevisionPDFV2{r}}
}
func servicioPrueba(t *testing.T) (*Servicio, *fuentePrueba, *autorizadorPrueba, *lectorPrueba) {
	f := &fuentePrueba{c: Contexto{"organizacion:central", "per_candidato_nominal"}}
	a := &autorizadorPrueba{t: t}
	l := &lectorPrueba{l: lecturaPrueba()}
	s, e := Nuevo(f, a, l)
	if e != nil {
		t.Fatal(e)
	}
	return s, f, a, l
}
func TestConsultaLigaContextoCentralYMinimiza(t *testing.T) {
	s, _, a, l := servicioPrueba(t)
	r, e := s.Consultar(context.Background(), solicitudPrueba())
	if e != nil {
		t.Fatal(e)
	}
	if a.m.OrganizacionRef != "organizacion:central" || a.m.FirmantePrincipalCandidatoRef != "per_candidato_nominal" || a.m.UnidadRef != "" || l.llamadas != 1 {
		t.Fatal("material no procede del contexto")
	}
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	for _, privado := range []string{"actor:no-exponer", "perfil:no-exponer", "clave-firma-no-exponer", "evidencia-no-exponer", strings.Repeat("e", 64)} {
		if bytes.Contains(b, []byte(privado)) {
			t.Fatalf("proyección no minimizada: %s", privado)
		}
	}
	if len(r.Firmas) != 1 || r.Firmas[0].RevisionPDF.EvidenciaFirmasSHA256 != l.l.RevisionesPDF[0].EvidenciaFirmasHuellaSHA256 {
		t.Fatal("metadatos persistidos perdidos")
	}
}
func TestConsultaRechazaCapacidadAjenaYMaterialInvalido(t *testing.T) {
	s, f, a, l := servicioPrueba(t)
	a.ajena = true
	if _, e := s.Consultar(context.Background(), solicitudPrueba()); !errors.Is(e, ports.ErrFirmaDocumentoDenegada) || l.llamadas != 0 {
		t.Fatalf("capacidad ajena consumida: %v", e)
	}
	a.ajena = false
	f.c.FirmantePrincipalCandidatoRef = "actor:cliente"
	if _, e := s.Consultar(context.Background(), solicitudPrueba()); !errors.Is(e, ports.ErrFirmaDocumentoDenegada) || a.llamadas != 1 {
		t.Fatalf("fuente inválida autoriza: %v", e)
	}
}
func TestConsultaNoRetornaDatosTrasCancelar(t *testing.T) {
	for _, etapa := range []string{"antes", "fuente", "autorizador", "lector"} {
		t.Run(etapa, func(t *testing.T) {
			s, f, a, l := servicioPrueba(t)
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			switch etapa {
			case "antes":
				cancelar()
			case "fuente":
				f.despues = cancelar
			case "autorizador":
				a.despues = cancelar
			case "lector":
				l.despues = cancelar
			}
			r, e := s.Consultar(ctx, solicitudPrueba())
			if !errors.Is(e, context.Canceled) || len(r.Firmas) != 0 || r.ExpedienteRef != "" {
				t.Fatalf("cancelación expone datos: %v", e)
			}
		})
	}
}
func TestConsultaConstructorDeniegaTypedNil(t *testing.T) {
	_, f, a, l := servicioPrueba(t)
	var fn *fuentePrueba
	var an *autorizadorPrueba
	var ln *lectorPrueba
	for _, deps := range []struct {
		f FuenteContexto
		a ports.AutorizadorConsultaFirmasR5V2
		l Lector
	}{{nil, a, l}, {fn, a, l}, {f, an, l}, {f, a, ln}} {
		if _, e := Nuevo(deps.f, deps.a, deps.l); !errors.Is(e, ports.ErrRegistroFirmaDocumentoNoDisponible) {
			t.Fatal("acepta dependencia nula")
		}
	}
}
func TestConsultaRechazaRevisionDuplicadaOManipulada(t *testing.T) {
	for _, alterar := range []func(*ports.LecturaFirmasR5V2){
		func(l *ports.LecturaFirmasR5V2) { l.RevisionesPDF = append(l.RevisionesPDF, l.RevisionesPDF[0]) },
		func(l *ports.LecturaFirmasR5V2) {
			l.RevisionesPDF[0].EvidenciaFirmasHuellaSHA256 = strings.Repeat("0", 64)
		},
		func(l *ports.LecturaFirmasR5V2) { l.RevisionesPDF[0].ReciboRef = "recibo:ajeno" },
		func(l *ports.LecturaFirmasR5V2) { l.RevisionesPDF[0].ByteRange[3] = 101 },
		func(l *ports.LecturaFirmasR5V2) { l.Firmas[0].Documento = "resolucion" },
	} {
		s, _, _, l := servicioPrueba(t)
		alterar(&l.l)
		r, e := s.Consultar(context.Background(), solicitudPrueba())
		if !errors.Is(e, ports.ErrResultadoFirmaDocumentoInvalido) || len(r.Firmas) != 0 {
			t.Fatalf("datos alterados admitidos: %v", e)
		}
	}
}
func TestConsultaColeccionVaciaNoInfiereAusencia(t *testing.T) {
	s, _, _, l := servicioPrueba(t)
	l.l.Firmas = nil
	l.l.RevisionesPDF = nil
	r, e := s.Consultar(context.Background(), solicitudPrueba())
	if e != nil || r.Firmas == nil || len(r.Firmas) != 0 || r.HistoriaHuella != l.l.HistoriaHuella {
		t.Fatalf("lectura vacía reinterpretada: %v", e)
	}
}

func TestConsultaDosRevisionesConservaCadenaPersistida(t *testing.T) {
	s, _, _, l := servicioPrueba(t)
	previa := l.l.RevisionesPDF[0]
	f := previa.FirmaRegistrada
	f.FirmaRef = "firma:segunda"
	f.ReciboRef = "recibo:segundo"
	f.PasoRef = "paso:segundo"
	f.PasoOrden = 2
	f.Secuencia = 2
	f.FirmadoHuella = strings.Repeat("7", 64)
	f.DocumentoCustodiaRef = "ref:" + f.FirmadoHuella
	f.DocumentoCustodiaVersion = 2
	segunda := previa
	segunda.FirmaRegistrada = f
	segunda.FirmaAnteriorRef = previa.FirmaRef
	segunda.ReciboAnteriorRef = previa.ReciboRef
	segunda.OrdenFirmaPDF = 2
	segunda.EntradaDocumentoRef = previa.DocumentoCustodiaRef
	segunda.EntradaDocumentoVersion = previa.DocumentoCustodiaVersion
	segunda.EntradaDocumentoHuella = previa.FirmadoHuella
	segunda.EntradaDocumentoLongitud = previa.RevisionLongitud
	segunda.RevisionLongitud = 500
	segunda.ByteRange = [4]uint64{0, 300, 400, 100}
	segunda.RevisionHuellaSHA256 = f.FirmadoHuella
	segunda.EvidenciaFirmasCanonica = json.RawMessage(`[{"privado":"primera"},{"privado":"segunda"}]`)
	h := sha256.Sum256(segunda.EvidenciaFirmasCanonica)
	segunda.EvidenciaFirmasHuellaSHA256 = hex.EncodeToString(h[:])
	l.l.Firmas = append(l.l.Firmas, f)
	l.l.RevisionesPDF = append(l.l.RevisionesPDF, segunda)
	q := solicitudPrueba()
	q.PasoOrden = 2
	r, e := s.Consultar(context.Background(), q)
	if e != nil || len(r.Firmas) != 2 || r.Firmas[1].RevisionPDF.FirmaAnteriorRef != previa.FirmaRef || r.Firmas[1].RevisionPDF.Entrada.SHA256 != previa.FirmadoHuella {
		t.Fatalf("cadena persistida perdida: %v", e)
	}
	l.l.RevisionesPDF[1].ReciboAnteriorRef = "recibo:ajeno"
	if _, e = s.Consultar(context.Background(), q); !errors.Is(e, ports.ErrResultadoFirmaDocumentoInvalido) {
		t.Fatal("antecedente ajeno admitido")
	}
}
