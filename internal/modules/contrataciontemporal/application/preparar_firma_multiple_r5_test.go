package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func (d *dependenciasMultiplePrueba) ObtenerPerfilActivoOperadorFirmaV2(context.Context) (string, error) {
	return d.perfilOperador, nil
}

type circuitoMultiplePrueba struct{ alternativo bool }

func (c circuitoMultiplePrueba) CircuitoFirma(ctx context.Context) (domain.CircuitoFirma, error) {
	r, err := (circuitoFirmaPrueba{}).CircuitoFirma(ctx)
	r.CatalogoVersion = 2
	if c.alternativo {
		r.Documentos[0].Pasos[0].PerfilesAlternativos = []string{"perfil:ct:direccion"}
	}
	return r, err
}

// Este doble prueba coordinación e idempotencia. No verifica criptografía PDF.
type dependenciasMultiplePrueba struct {
	perfilOperador  string
	materiales      []ports.MaterialFirmaVerificadaV2
	lecturas        ports.LecturaFirmasR5V2
	pdf             map[string][]byte
	dictamen        docports.VerificacionFirmasDocumento
	alterarDictamen func(*docports.VerificacionFirmasDocumento)
	alterarLectura  func(*ports.LecturaFirmasR5V2)
	alterarCustodia bool
	verificaciones  int
	actor           string
	consultas       []ports.MaterialConsultaFirmasR5V2
}

func (d *dependenciasMultiplePrueba) VerificarFirmas(_ context.Context, s docports.SolicitudVerificacionFirma) (docports.VerificacionFirmasDocumento, error) {
	v := d.dictamen
	v.Firmas = append([]docports.FirmaPDFVerificada(nil), v.Firmas...)
	d.verificaciones++
	v.ComprobadoEn = v.ComprobadoEn.Add(time.Duration(d.verificaciones) * time.Second)
	if d.alterarDictamen != nil {
		d.alterarDictamen(&v)
	}
	return v, nil
}

func capacidadMultiplePrueba(tipos string, r vecdomain.RecursoAutorizable, accion, audiencia string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	hasta := instanteFirmaPrueba.Add(time.Minute)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:multiple:"+tipos, strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto:multiple", strings.Repeat("a", 64), accion, r.Referencia, h, audiencia, hasta.Add(-5*time.Second), hasta)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{11}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{7}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
}

func (d *dependenciasMultiplePrueba) AutorizarConsultaFirmasR5V2(_ context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	d.consultas = append(d.consultas, m)
	r, e := RecursoConsultaFirmasR5V2(m)
	if e != nil {
		return ports.CapacidadConsultaFirmasR5V2{}, e
	}
	a, e := capacidadMultiplePrueba("lectura", r, ports.AccionConsultarFirmasR5V2, ports.AudienciaConsultaFirmasR5V2)
	return ports.TransportarMaterialConsultaFirmasR5V2(a), e
}
func (d *dependenciasMultiplePrueba) AutorizarFirmaVerificadaV2(_ context.Context, m ports.MaterialFirmaVerificadaV2) (ports.CapacidadFirmaVerificadaV2, error) {
	descriptor, e := CanonicoDescriptorFirmaVerificadaV2(m, descriptorMultiplePrueba(m))
	if e != nil {
		return ports.CapacidadFirmaVerificadaV2{}, e
	}
	r, e := RecursoFirmaVerificadaV2(m, descriptor)
	if e != nil {
		return ports.CapacidadFirmaVerificadaV2{}, e
	}
	accion, audiencia := ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	if m.Via == ports.ViaFirmaExternaPortafirmas {
		accion, audiencia = ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	}
	a, e := capacidadMultiplePrueba("firma", r, accion, audiencia)
	h := sha256.Sum256(descriptor)
	return ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(a, descriptor, hex.EncodeToString(h[:])), e
}
func (d *dependenciasMultiplePrueba) ConsultarFirmasAutorizadasV2(_ context.Context, m ports.MaterialConsultaFirmasR5V2, c ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	if e := ValidarCapacidadConsultaFirmasR5V2(c, m); e != nil {
		return ports.LecturaFirmasR5V2{}, e
	}
	l := d.lecturas
	l.Firmas = append([]ports.FirmaRegistrada(nil), l.Firmas...)
	l.RevisionesPDF = append([]ports.FirmaRegistradaRevisionPDFV2(nil), l.RevisionesPDF...)
	if d.alterarLectura != nil {
		d.alterarLectura(&l)
	}
	return l, nil
}
func (d *dependenciasMultiplePrueba) ObtenerPDFFirmaAnterior(_ context.Context, q ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	p := bytes.Clone(d.pdf[q.DocumentoRef])
	if d.alterarCustodia && len(p) > 0 {
		p[0] ^= 1
	}
	return ports.PDFFirmaAnterior{Solicitud: q, Contenido: p}, nil
}
func (d *dependenciasMultiplePrueba) RegistrarFirmaVerificadaV2(_ context.Context, m ports.MaterialFirmaVerificadaV2, c ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	if e := ValidarCapacidadFirmaVerificadaV2(c, m); e != nil {
		return ports.ReciboFirmaDocumento{}, e
	}
	h, _ := m.HuellaSHA256()
	replay := false
	for _, previa := range d.materiales {
		if previa.ClaveIdempotencia == m.ClaveIdempotencia {
			p, _ := previa.HuellaSHA256()
			if p != h {
				return ports.ReciboFirmaDocumento{}, ports.ErrClaveFirmaDocumentoUsada
			}
			replay = true
		}
	}
	r := ports.ReciboFirmaDocumento{FirmaRef: fmt.Sprintf("firma:multiple:%d", m.Secuencia), ReciboRef: fmt.Sprintf("recibo:multiple:%d", m.Secuencia), Secuencia: m.Secuencia, Resultado: domain.ResultadoFirmaFirmado, ExpedienteVersion: m.VersionExpediente, ActorRef: m.FirmantePrincipalRef, PerfilRef: m.PerfilFirmanteRef, RegistradaEn: instanteFirmaPrueba, SolicitudHuella: h, YaRegistrada: replay, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion}
	if d.actor != "" {
		r.ActorRef = d.actor
	}
	if !replay {
		d.materiales = append(d.materiales, m)
		f := ports.FirmaRegistrada{Via: m.Via, FirmaRef: r.FirmaRef, ReciboRef: r.ReciboRef, RegistradaEn: r.RegistradaEn, Documento: m.Documento, Secuencia: m.Secuencia, ExpedienteVersion: m.VersionExpediente, CatalogoRef: m.CatalogoRef, CatalogoHuella: m.CatalogoHuella, PasoRef: m.PasoRef, PasoOrden: m.PasoOrden, Resultado: r.Resultado, OriginalRef: m.OriginalRef, OriginalVersion: m.OriginalVersion, OriginalHuella: m.OriginalHuella, FirmadoHuella: m.FirmadoHuella, FirmantePrincipalAcreditado: true, HistoriaRevision: m.HistoriaRevision, HistoriaHuella: m.HistoriaHuella, ClaveIdempotencia: m.ClaveIdempotencia, DocumentoCustodiaRef: m.DocumentoCustodiaRef, DocumentoCustodiaVersion: m.DocumentoCustodiaVersion, ReferenciaPortafirmasDeclarada: m.ReferenciaPortafirmasDeclarada, FechaPortafirmasDeclarada: m.FechaPortafirmasDeclarada}
		d.lecturas.Firmas = append(d.lecturas.Firmas, f)
		tecnica := f
		tecnica.FirmanteRef, tecnica.CertificadoHuella = m.FirmanteRef, m.CertificadoHuella
		d.lecturas.RevisionesPDF = append(d.lecturas.RevisionesPDF, ports.FirmaRegistradaRevisionPDFV2{FirmaRegistrada: tecnica, FirmaAnteriorRef: m.FirmaAnteriorRef, ReciboAnteriorRef: m.ReciboAnteriorRef, OrdenFirmaPDF: m.OrdenFirmaPDF, ByteRange: m.ByteRange, RevisionHuellaSHA256: m.RevisionHuellaSHA256, ContenidoFirmadoHuellaSHA256: m.ContenidoFirmadoHuellaSHA256, RevisionLongitud: m.RevisionLongitud})
		d.lecturas.HistoriaRevision++
		d.lecturas.HistoriaHuella = h
	}
	return r, nil
}

func fixturePDFMultipleConsumidor(n int) ([]byte, docports.VerificacionFirmasDocumento) {
	original := []byte("%PDF-1.7 original vec sintetico")
	pdf := bytes.Clone(original)
	v := docports.VerificacionFirmasDocumento{Estado: docports.EstadoVerificacionValida, Motivo: docports.MotivoFirmaVerificada, Formato: "PAdES", VinculoOriginal: true, HuellaOriginalSHA256: huella(original), ComprobadoEn: instanteFirmaPrueba}
	zero := uint64(0)
	v.CambiosPosteriores = docports.CambioPosteriorFirmaPDF{Estado: "ninguno", BytesNoFirmados: &zero}
	for orden := 1; orden <= n; orden++ {
		pdf = append(pdf, []byte(" revision ")...)
		inicio := len(pdf)
		pdf = append(pdf, []byte("<firma>")...)
		fin := len(pdf)
		pdf = append(pdf, []byte(" trailer %%EOF")...)
		h := sha256.Sum256(append(bytes.Clone(pdf[:inicio]), pdf[fin:]...))
		cert := strings.Repeat(string(rune('g'-orden)), 64)
		v.Firmas = append(v.Firmas, docports.FirmaPDFVerificada{Orden: orden, ByteRange: [4]uint64{0, uint64(inicio), uint64(fin), uint64(len(pdf) - fin)}, RevisionHuellaSHA256: huella(pdf), ContenidoFirmadoHuellaSHA256: hex.EncodeToString(h[:]), RevisionLongitud: uint64(len(pdf)), CubreDocumentoCompletoHastaAqui: true, FirmanteRef: "ref:" + cert, CertificadoHuellaSHA256: cert, IntegridadEstado: "valida", CadenaEstado: "valida", CertificadoEstado: "vigente", RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", TipoFirma: "aprobacion", CambiosDesdeAnterior: docports.CambioFirmaPDF{Estado: "permitidos", Detalle: []string{"firma_anadida"}}})
	}
	v.HuellaFirmadoSHA256 = huella(pdf)
	return pdf, v
}

func prepararServicioMultipleConsumidor(t *testing.T) (*ServicioFirmaVec, *dependenciasMultiplePrueba, *competenciaExternaPrueba, *originalExternoPrueba) {
	t.Helper()
	s, _, o, competencia, _, _ := servicioFirmaVecPrueba(t)
	s.base.circuito = circuitoMultiplePrueba{}
	o.alterar = func(r *ports.OriginalFirmaAutorizado) { r.UnidadRef = "unidad:rrhh" }
	d := &dependenciasMultiplePrueba{perfilOperador: "perfil-activo:operador:001", pdf: map[string][]byte{}, lecturas: ports.LecturaFirmasR5V2{LecturaFirmasR5: ports.LecturaFirmasR5{HistoriaHuella: strings.Repeat("1", 64), HistoriaSeparacionAcreditada: true}}}
	if e := s.ComponerFirmaMultiple(d, d, d, d, d); e != nil {
		t.Fatal(e)
	}
	competencia.alterar = func(e *ports.EvidenciaCompetenciaFirmante) {
		e.CargoFirmante = "ct_cargo_jefatura_rrhh"
		e.RolIDFirmante = "ct_cargo_jefatura_rrhh"
		e.CuentaFirmanteRef = "cuenta:firmante:001"
		e.VinculoCredencialFirmanteRef = "vinculo:credencial:001"
		e.VinculoCredencialFirmanteRevision = 2
		e.VinculoCredencialFirmanteHuella = strings.Repeat("a", 64)
	}
	return s, d, competencia, o
}

func registrarPasoMultipleConsumidor(t *testing.T, s *ServicioFirmaVec, d *dependenciasMultiplePrueba, o *originalExternoPrueba, n int) (ResultadoFirmaVec, error) {
	t.Helper()
	sol := solicitudFirmaVecPrueba(o.contenido, fmt.Sprintf("multiple-clave-%d", n))
	sol.PasoOrden = n
	sol.PDFFirmado, d.dictamen = fixturePDFMultipleConsumidor(n)
	r, e := s.Firmar(context.Background(), sol)
	if e == nil {
		d.pdf[r.Custodiado.Ref] = bytes.Clone(sol.PDFFirmado)
	}
	return r, e
}

func TestConsumidorFirmaMultipleDosActosYReplay(t *testing.T) {
	s, d, competencia, o := prepararServicioMultipleConsumidor(t)
	primero, e := registrarPasoMultipleConsumidor(t, s, d, o, 1)
	if e != nil {
		t.Fatal(e)
	}
	segundo, e := registrarPasoMultipleConsumidor(t, s, d, o, 2)
	if e != nil {
		t.Fatal(e)
	}
	if primero.MaterialMultiple == nil || segundo.MaterialMultiple == nil || primero.Recibo.FirmaRef == segundo.Recibo.FirmaRef || segundo.MaterialMultiple.FirmaAnteriorRef != primero.Recibo.FirmaRef || segundo.MaterialMultiple.OriginalRef != primero.MaterialMultiple.OriginalRef || len(d.materiales) != 2 || len(competencia.vistas) != 2 {
		t.Fatal("cadena, actos o competencia incoherentes")
	}
	repetido, e := registrarPasoMultipleConsumidor(t, s, d, o, 2)
	if e != nil {
		t.Fatal(e)
	}
	if !repetido.Recibo.YaRegistrada || repetido.Recibo.SolicitudHuella != segundo.Recibo.SolicitudHuella || repetido.Recibo.RegistradaEn != segundo.Recibo.RegistradaEn || len(d.materiales) != 2 || repetido.MaterialMultiple.ComprobadaEn.Equal(segundo.MaterialMultiple.ComprobadaEn) {
		t.Fatal("replay alterado o duplicado")
	}
	primeroRepetido, e := registrarPasoMultipleConsumidor(t, s, d, o, 1)
	if e != nil || !primeroRepetido.Recibo.YaRegistrada || primeroRepetido.Recibo.SolicitudHuella != primero.Recibo.SolicitudHuella || len(d.materiales) != 2 {
		t.Fatalf("replay primer acto tras segundo: %+v %v", primeroRepetido, e)
	}
	if segundo.Material.Validar() == nil {
		t.Fatal("V2 se representa como material V1")
	}
	// En la vía VEC consulta quien firma: la unidad de su competencia (la del
	// paso) va en la consulta, y su recurso la lleva.
	if len(d.consultas) == 0 {
		t.Fatal("sin consultas previas")
	}
	for _, c := range d.consultas {
		r, err := RecursoConsultaFirmasR5V2(c)
		if c.UnidadRef == "" || c.UnidadRef != segundo.MaterialMultiple.UnidadFirmanteRef || err != nil || r.Ambitos["unidad_ref"] != c.UnidadRef {
			t.Fatalf("consulta VEC sin la unidad del paso: %+v", c)
		}
	}
}

func TestConsumidorFirmaMultipleRechazaAntecedenteAlterado(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*dependenciasMultiplePrueba, *originalExternoPrueba)
	}{
		{"certificado anterior", func(d *dependenciasMultiplePrueba, _ *originalExternoPrueba) {
			d.alterarDictamen = func(v *docports.VerificacionFirmasDocumento) {
				v.Firmas[0].CertificadoHuellaSHA256 = strings.Repeat("a", 64)
			}
		}},
		{"revision anterior", func(d *dependenciasMultiplePrueba, _ *originalExternoPrueba) {
			d.alterarLectura = func(l *ports.LecturaFirmasR5V2) { l.RevisionesPDF[0].ByteRange[1]++ }
		}},
		{"custodia anterior", func(d *dependenciasMultiplePrueba, _ *originalExternoPrueba) { d.alterarCustodia = true }},
		{"original distinto", func(_ *dependenciasMultiplePrueba, o *originalExternoPrueba) {
			o.contenido = []byte("%PDF-1.7 otro original")
		}},
		{"orden permutado", func(d *dependenciasMultiplePrueba, _ *originalExternoPrueba) {
			d.alterarDictamen = func(v *docports.VerificacionFirmasDocumento) { v.Firmas[0], v.Firmas[1] = v.Firmas[1], v.Firmas[0] }
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, d, _, o := prepararServicioMultipleConsumidor(t)
			if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 1); e != nil {
				t.Fatal(e)
			}
			c.alterar(d, o)
			if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 2); e == nil || len(d.materiales) != 1 {
				t.Fatalf("antecedente alterado admitido: %v", e)
			}
		})
	}
}

func TestConsumidorFirmaMultipleNoPermitePasoDosSinPrimero(t *testing.T) {
	s, d, _, o := prepararServicioMultipleConsumidor(t)
	if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 2); !errors.Is(e, ErrPasoFirmaNoPendiente) || len(d.materiales) != 0 {
		t.Fatal(e)
	}
}

type fuenteCompetenciaMultipleFunc func(context.Context, ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error)

func (f fuenteCompetenciaMultipleFunc) AcreditarCompetenciaFirmante(c context.Context, q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	return f(c, q)
}

func TestConsumidorFirmaMultipleCompetenciaUnicaEntreAlternativas(t *testing.T) {
	for _, ambiguo := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguo), func(t *testing.T) {
			s, d, comp, o := prepararServicioMultipleConsumidor(t)
			s.base.circuito = circuitoMultiplePrueba{alternativo: true}
			s.competencia = fuenteCompetenciaMultipleFunc(func(c context.Context, q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
				if !ambiguo && q.PerfilFirmanteRef != "perfil:ct:direccion" {
					return ports.EvidenciaCompetenciaFirmante{}, ports.ErrCompetenciaFirmanteNoAcreditada
				}
				e, err := comp.AcreditarCompetenciaFirmante(c, q)
				e.CargoFirmante = "ct_cargo_direccion_rrhh"
				e.RolIDFirmante = "ct_cargo_direccion_rrhh"
				return e, err
			})
			r, e := registrarPasoMultipleConsumidor(t, s, d, o, 1)
			if ambiguo {
				if !errors.Is(e, ports.ErrCompetenciaFirmanteNoAcreditada) || len(d.materiales) != 0 {
					t.Fatal(e)
				}
			} else if e != nil || r.MaterialMultiple.PerfilFirmanteRef != "perfil:ct:direccion" || r.MaterialMultiple.CargoFirmante != "ct_cargo_direccion_rrhh" {
				t.Fatalf("alternativa o cargo real perdido: %+v %v", r, e)
			}
		})
	}
}

func TestConsumidorFirmaMultipleReplayNoAceptaOtroPDF(t *testing.T) {
	s, d, _, o := prepararServicioMultipleConsumidor(t)
	if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 1); e != nil {
		t.Fatal(e)
	}
	sol := solicitudFirmaVecPrueba(o.contenido, "multiple-clave-1")
	sol.PDFFirmado, d.dictamen = fixturePDFMultipleConsumidor(1)
	sol.PDFFirmado[len(sol.PDFFirmado)-1] ^= 1
	if _, e := s.Firmar(context.Background(), sol); e == nil || len(d.materiales) != 1 {
		t.Fatalf("replay alterado: %v", e)
	}
}

func TestConsumidorFirmaMultipleExternaDosPasosRegistraRRHHOtroFirmante(t *testing.T) {
	vec, d, comp, o := prepararServicioMultipleConsumidor(t)
	s, e := NuevoServicioFirmaExterna(vec.base, &registroExternoPrueba{}, &autorizadorConsultaFirmasR5Prueba{}, &autorizadorExternoPrueba{}, comp)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ComponerFirmaMultiple(d, d, d, d, d); e != nil {
		t.Fatal(e)
	}
	d.actor = "per_rrhh_registrador_001"
	for n := 1; n <= 2; n++ {
		vs := solicitudFirmaVecPrueba(o.contenido, fmt.Sprintf("externa-multiple-%d", n))
		sol := solicitudMultipleDesdeVec(vs).SolicitudFirmaExterna
		sol.PasoOrden = n
		sol.ReferenciaPortafirmasDeclarada = "portafirmas:declaracion"
		sol.FechaPortafirmasDeclarada = "2026-10-02T10:00:00Z"
		sol.PDFFirmado, d.dictamen = fixturePDFMultipleConsumidor(n)
		r, e := s.Registrar(context.Background(), sol)
		if e != nil {
			t.Fatal(e)
		}
		if r.MaterialMultiple == nil || r.MaterialMultiple.PoliticaVerificacion != PoliticaVerificacionFirmaMultipleV2 || r.Recibo.ActorRef == r.MaterialMultiple.FirmantePrincipalRef {
			t.Fatal("identidad del registrador confundida")
		}
		d.pdf[r.Custodiado.Ref] = bytes.Clone(sol.PDFFirmado)
	}
	if len(d.materiales) != 2 {
		t.Fatal("duplicación del acto")
	}
	// En la vía externa consulta RRHH: sólo organización.
	if len(d.consultas) == 0 {
		t.Fatal("sin consultas previas")
	}
	for _, c := range d.consultas {
		if c.Via != ports.ViaFirmaExternaPortafirmas || c.UnidadRef != "" {
			t.Fatalf("consulta externa con unidad: %+v", c)
		}
	}
}

func TestConsumidorFirmaMultipleConservaSeparacionGlobal(t *testing.T) {
	for _, desconocida := range []bool{false, true} {
		t.Run(fmt.Sprint(desconocida), func(t *testing.T) {
			s, d, _, o := prepararServicioMultipleConsumidor(t)
			if desconocida {
				d.lecturas.HistoriaSeparacionAcreditada = false
			} else {
				d.lecturas.CoincideFirmanteEnOtroPaso = true
			}
			if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 1); !errors.Is(e, ports.ErrMismaPersonaEnOtroPasoR5) || len(d.materiales) != 0 {
				t.Fatalf("separación no acreditada admitida: %v", e)
			}
		})
	}
}

func TestConsumidorFirmaMultipleRechazaCompetenciaAjenaOCaducada(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ports.EvidenciaCompetenciaFirmante)
	}{
		{"perfil ajeno", func(e *ports.EvidenciaCompetenciaFirmante) { e.PerfilFirmanteRef = "perfil:ct:otro" }},
		{"vigencia agotada", func(e *ports.EvidenciaCompetenciaFirmante) { e.Vigente = false }},
		{"binding ausente", func(e *ports.EvidenciaCompetenciaFirmante) { e.VinculoCredencialFirmanteRef = "" }},
		{"rol no acreditado", func(e *ports.EvidenciaCompetenciaFirmante) { e.RolIDFirmante = "" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, d, comp, o := prepararServicioMultipleConsumidor(t)
			previa := comp.alterar
			comp.alterar = func(e *ports.EvidenciaCompetenciaFirmante) { previa(e); c.alterar(e) }
			if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 1); !errors.Is(e, ports.ErrCompetenciaFirmanteNoAcreditada) || len(d.materiales) != 0 {
				t.Fatalf("competencia alterada admitida: %v", e)
			}
		})
	}
}

func TestConsumidorFirmaMultipleNoInfierePerfilOperacionalDelCargo(t *testing.T) {
	s, d, _, o := prepararServicioMultipleConsumidor(t)
	d.perfilOperador = ""
	if _, e := registrarPasoMultipleConsumidor(t, s, d, o, 1); !errors.Is(e, ports.ErrFirmaDocumentoDenegada) || len(d.materiales) != 0 {
		t.Fatalf("perfil operacional ausente admitido: %v", e)
	}
}
