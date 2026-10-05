package plannominal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Estos dobles prueban el consumidor del plan; no acreditan publicación,
// identidad, competencia ni enlaces Personal de una instalación.
type selectorDescriptorPrueba struct {
	seleccion SeleccionCentralDescriptorFirmaV2
	llamadas  int
	paso      ct.CompetenciaPasoFirmaV2
	fallo     error
	durante   func(ports.MaterialFirmaVerificadaV2)
}

func (s *selectorDescriptorPrueba) ResolverSeleccionDescriptorFirmaV2(_ context.Context, m ports.MaterialFirmaVerificadaV2, p ct.CompetenciaPasoFirmaV2) (SeleccionCentralDescriptorFirmaV2, error) {
	s.llamadas++
	s.paso = p
	if s.durante != nil {
		s.durante(m)
	}
	return s.seleccion, s.fallo
}

func descriptorPrueba(t *testing.T, orden int) (*FuenteDescriptorFirmaV2, ports.MaterialFirmaVerificadaV2, *selectorDescriptorPrueba, *publicacionPrueba, *consultaPlanPrueba) {
	t.Helper()
	c, en := catalogoPrueba()
	a := c.Entradas[0].Atributos
	h := strings.Repeat("a", 64)
	original, firmado := strings.Repeat("c", 64), strings.Repeat("d", 64)
	cert := strings.Repeat("b", 64)
	evidencia := []byte(`[{"orden":1}]`)
	if orden == 2 {
		evidencia = []byte(`[{"orden":1},{"orden":2}]`)
	}
	hash := sha256.Sum256(evidencia)
	m := ports.MaterialFirmaVerificadaV2{MaterialFirmaExterna: ports.MaterialFirmaExterna{
		Via: ports.ViaFirmaCertificadoVEC, OrganizacionRef: a["organizacion_ref"], ExpedienteRef: "expediente:prueba", VersionExpediente: 7,
		Documento: "resolucion", CatalogoRef: a["circuito_ref"], CatalogoHuella: a["circuito_sha256"], PasoRef: a["paso_ref"], PasoOrden: orden, Secuencia: orden,
		HistoriaHuella: h, OriginalRef: "original:prueba", OriginalVersion: 1, OriginalHuella: original, FirmadoHuella: firmado, CertificadoHuella: cert, FirmanteRef: "ref:" + cert,
		FirmantePrincipalRef: "per_firmante_sintetico", PerfilFirmanteRef: a["perfil_esperado_ref"], CargoFirmante: a["rol_id"], UnidadFirmanteRef: a["unidad_ref"], PerfilActivoFirmanteRef: "prf_firmante_sintetico",
		AsignacionFirmanteRef: "asignacion:prueba", AsignacionFirmanteVersion: 1, AsignacionFirmanteHuella: h, VersionRolFirmanteRef: "rol:central:v1", VersionRolFirmanteHuella: h,
		ControlVigenciaFirmanteRef: "rol:central:v1", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: h,
		AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z", PoliticaVerificacion: "politica:vec:firma:verificacion-autonoma:v2",
		RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", ClaveIdempotencia: "clave-prueba-firma-000001", DocumentoCustodiaRef: "custodia:prueba", DocumentoCustodiaVersion: ports.VersionDocumentoCustodiado},
		PerfilActivoOperadorRef: "prf_firmante_sintetico", CuentaFirmanteRef: "cuenta:prueba", VinculoCredencialFirmanteRef: "vinculo:prueba", VinculoCredencialFirmanteRevision: 1, VinculoCredencialFirmanteHuella: h,
		RolIDFirmante: a["rol_id"], CatalogoVersion: 1, EntradaDocumentoRef: "original:prueba", EntradaDocumentoVersion: 1, EntradaDocumentoLongitud: 100, EntradaDocumentoHuella: original,
		OrdenFirmaPDF: orden, ByteRange: [4]uint64{0, 120, 180, 20}, RevisionHuellaSHA256: firmado, ContenidoFirmadoHuellaSHA256: h, RevisionLongitud: 200,
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(hash[:]), ComprobadaEn: en}
	if orden == 2 {
		a["paso_orden"] = "2"
		m.FirmaAnteriorRef, m.ReciboAnteriorRef = "firma:anterior", "recibo:anterior"
		m.EntradaDocumentoRef, m.EntradaDocumentoVersion, m.EntradaDocumentoHuella = "custodia:anterior", 1, strings.Repeat("e", 64)
	}
	if err := m.Validar(); err != nil {
		t.Fatalf("material de prueba: %v", err)
	}
	consulta := &consultaPlanPrueba{c, en}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, CatalogoID: c.ID, ModuloID: c.ModuloID, Reloj: consulta})
	if err != nil {
		t.Fatal(err)
	}
	sha, _ := c.HuellaSHA256()
	pub := &publicacionPrueba{}
	plan, err := NuevaFuente(resolutor, pub, ct.VersionPlanFirmaV2{Referencia: c.ID, Version: 1, HuellaSHA256: sha})
	if err != nil {
		t.Fatal(err)
	}
	s := &selectorDescriptorPrueba{seleccion: SeleccionCentralDescriptorFirmaV2{
		Seleccion: ports.SeleccionConstructorFirmaV2{PerfilEsperadoRef: m.PerfilFirmanteRef, PerfilActivoRef: m.PerfilActivoFirmanteRef, RolID: m.RolIDFirmante, CargoRef: a["cargo_ref"], EnlaceEjercicioRef: "enlace:existente"},
		Recurso: vd.RecursoFirmaHistoricaV1{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef, ExpedienteRef: m.ExpedienteRef, DocumentoRef: m.OriginalRef, RecursoAutorizableRef: m.OriginalRef,
			ModuloID: ports.ModuloContratacion, TipoRecurso: a["tipo_recurso"], RecursoContextoSHA256: strings.Repeat("f", 64),
			Original: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.OriginalRef, Version: m.OriginalVersion, HuellaSHA256: m.OriginalHuella}, PDFRaizSHA256: m.OriginalHuella,
			Firmado: vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.DocumentoCustodiaRef, Version: m.DocumentoCustodiaVersion, HuellaSHA256: m.FirmadoHuella}, PDFFirmadoSHA256: m.FirmadoHuella, NumeroFirmas: uint64(orden)},
		EsquemaContexto: a["esquema_contexto"], Motivo: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos", CatalogoVersion: 1, CatalogoHuellaSHA256: h, EntradaClave: "firma"},
	}}
	if orden == 2 {
		s.seleccion.Recurso.EntradaRevision = &vd.ReferenciaHistoricaCompetenciaV1{Referencia: m.EntradaDocumentoRef, Version: m.EntradaDocumentoVersion, HuellaSHA256: m.EntradaDocumentoHuella}
	}
	f, err := NuevaFuenteDescriptorFirmaV2(plan, s)
	if err != nil {
		t.Fatal(err)
	}
	return f, m, s, pub, consulta
}

func TestDescriptorPlanNominalConservaSeleccionYDosRevisiones(t *testing.T) {
	for _, orden := range []int{1, 2} {
		f, m, s, pub, _ := descriptorPrueba(t, orden)
		d, err := f.DescriptorFirmaV2(t.Context(), m)
		if err != nil || s.llamadas != 1 || pub.llamadas != 1 || d.Seleccion.CargoRef != "cargo:prueba" || d.Seleccion.EnlaceEjercicioRef != "enlace:existente" ||
			d.Accion != s.paso.Accion || d.Finalidad != s.paso.Finalidad || d.PasoOrden != uint64(orden) || d.FechaHistorica != nil {
			t.Fatalf("descriptor del paso %d: %+v, %v", orden, d, err)
		}
		canon, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
		if err != nil || firma.ValidarDescriptorFirmaVerificadaV2(m, canon) != nil {
			t.Fatalf("descriptor incompatible con consumidor V2: %v", err)
		}
		if orden == 2 {
			s.seleccion.Recurso.EntradaRevision.Referencia = "revision:posterior"
			if d.Recurso.EntradaRevision.Referencia != m.EntradaDocumentoRef {
				t.Fatal("el selector conserva un alias mutable del descriptor")
			}
		}
	}
}

func TestDescriptorPlanNominalDeniegaCrucesAntesDelSelector(t *testing.T) {
	mutaciones := map[string]func(*ports.MaterialFirmaVerificadaV2){
		"circuito":     func(m *ports.MaterialFirmaVerificadaV2) { m.CatalogoRef = "circuito:otro" },
		"version":      func(m *ports.MaterialFirmaVerificadaV2) { m.CatalogoVersion++ },
		"huella":       func(m *ports.MaterialFirmaVerificadaV2) { m.CatalogoHuella = strings.Repeat("b", 64) },
		"documento":    func(m *ports.MaterialFirmaVerificadaV2) { m.Documento = "informe_definitivo" },
		"paso":         func(m *ports.MaterialFirmaVerificadaV2) { m.PasoRef = "paso:otro" },
		"perfil":       func(m *ports.MaterialFirmaVerificadaV2) { m.PerfilFirmanteRef = "perfil:otro" },
		"organizacion": func(m *ports.MaterialFirmaVerificadaV2) { m.OrganizacionRef = "org:otra" },
		"unidad":       func(m *ports.MaterialFirmaVerificadaV2) { m.UnidadFirmanteRef = "unidad:otra" },
		"rol":          func(m *ports.MaterialFirmaVerificadaV2) { m.RolIDFirmante, m.CargoFirmante = "rol.otro", "rol.otro" },
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			f, m, s, _, _ := descriptorPrueba(t, 1)
			mutar(&m)
			if _, err := f.DescriptorFirmaV2(t.Context(), m); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || s.llamadas != 0 {
				t.Fatalf("cruce %s llegó al selector: %v", nombre, err)
			}
		})
	}
}

func TestDescriptorPlanNominalDeniegaSeleccionCentralInexacta(t *testing.T) {
	mutaciones := map[string]func(*SeleccionCentralDescriptorFirmaV2){
		"perfil esperado": func(s *SeleccionCentralDescriptorFirmaV2) { s.Seleccion.PerfilEsperadoRef = "perfil:otro" },
		"perfil activo":   func(s *SeleccionCentralDescriptorFirmaV2) { s.Seleccion.PerfilActivoRef = "prf_otro" },
		"rol":             func(s *SeleccionCentralDescriptorFirmaV2) { s.Seleccion.RolID = "rol.otro" },
		"cargo":           func(s *SeleccionCentralDescriptorFirmaV2) { s.Seleccion.CargoRef = "cargo:otro" },
		"enlace":          func(s *SeleccionCentralDescriptorFirmaV2) { s.Seleccion.EnlaceEjercicioRef = "" },
		"tipo":            func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.TipoRecurso = "tipo:otro" },
		"esquema":         func(s *SeleccionCentralDescriptorFirmaV2) { s.EsquemaContexto = "contexto:otro" },
		"expediente":      func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.ExpedienteRef = "expediente:otro" },
		"unidad":          func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.UnidadRef = "unidad:otra" },
		"original":        func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.DocumentoRef = "original:otro" },
		"revision":        func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.Firmado.HuellaSHA256 = strings.Repeat("e", 64) },
		"contexto":        func(s *SeleccionCentralDescriptorFirmaV2) { s.Recurso.RecursoContextoSHA256 = "" },
		"motivo":          func(s *SeleccionCentralDescriptorFirmaV2) { s.Motivo = vd.ReferenciaEntradaCatalogo{} },
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			f, m, s, _, _ := descriptorPrueba(t, 1)
			mutar(&s.seleccion)
			if _, err := f.DescriptorFirmaV2(t.Context(), m); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
				t.Fatalf("selección %s aceptada: %v", nombre, err)
			}
		})
	}
}

func TestDescriptorPlanNominalReleePublicacionYProtegeMaterial(t *testing.T) {
	f, m, s, pub, _ := descriptorPrueba(t, 1)
	s.durante = func(m ports.MaterialFirmaVerificadaV2) { m.EvidenciaFirmasCanonica[0] = 'X' }
	if _, err := f.DescriptorFirmaV2(t.Context(), m); err != nil || m.Validar() != nil {
		t.Fatalf("el selector cambió el material: %v", err)
	}
	pub.err = errors.New("publicacion retirada")
	if _, err := f.DescriptorFirmaV2(t.Context(), m); !errors.Is(err, ports.ErrCompetenciaFirmanteNoDisponible) || s.llamadas != 1 || pub.llamadas != 2 {
		t.Fatalf("reutilizó publicación anterior: %v", err)
	}
}

func TestDescriptorPlanNominalDependenciasYCancelacion(t *testing.T) {
	f, m, s, _, _ := descriptorPrueba(t, 1)
	for _, plan := range []*Fuente{nil, {}} {
		if d, err := NuevaFuenteDescriptorFirmaV2(plan, s); d != nil || err == nil {
			t.Fatal("aceptó fuente sin publicación")
		}
	}
	var selector *selectorDescriptorPrueba
	if d, err := NuevaFuenteDescriptorFirmaV2(f.plan, selector); d != nil || err == nil {
		t.Fatal("aceptó selector nulo tipado")
	}
	if _, err := f.DescriptorFirmaV2(nil, m); err == nil {
		t.Fatal("aceptó contexto nulo")
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	if _, err := f.DescriptorFirmaV2(ctx, m); !errors.Is(err, context.Canceled) || s.llamadas != 0 {
		t.Fatalf("cancelación inicial: %v", err)
	}
	ctx, cancelar = context.WithCancel(t.Context())
	s.durante = func(ports.MaterialFirmaVerificadaV2) { cancelar() }
	if _, err := f.DescriptorFirmaV2(ctx, m); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación del selector: %v", err)
	}
	var nula *FuenteDescriptorFirmaV2
	if _, err := nula.DescriptorFirmaV2(t.Context(), m); err == nil {
		t.Fatal("aceptó receptor nulo")
	}
}
