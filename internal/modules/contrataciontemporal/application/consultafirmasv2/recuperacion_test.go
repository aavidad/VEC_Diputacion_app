package consultafirmasv2

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func lecturaRecuperacionPrueba(t *testing.T) ports.LecturaRecuperacionFirmasV2 {
	t.Helper()
	base := lecturaPrueba()
	f, revision := base.Firmas[0], base.RevisionesPDF[0]
	f.CertificadoHuella = revision.CertificadoHuella
	base.Firmas[0] = f
	ref := func(referencia, huella string) vecdomain.ReferenciaHistoricaCompetenciaV1 {
		return vecdomain.ReferenciaHistoricaCompetenciaV1{Referencia: referencia, Version: 1, HuellaSHA256: huella}
	}
	persona := "per_1234567890abcdef1234567890abcdef"
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hasta := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	huella := strings.Repeat("a", 64)
	canon := vecdomain.CanonCompetenciaFirmanteHistoricaV1{
		Esquema: vecdomain.EsquemaCanonCompetenciaFirmanteHistoricaV1,
		Identidad: vecdomain.IdentidadFirmanteHistoricaV1{
			CertificadoDERSHA256: f.CertificadoHuella, PersonaRef: persona,
			Persona: ref(persona, huella), Cuenta: ref("cuenta:1", huella),
			VinculoCuentaPersona: ref("vinculo-cuenta-persona:1", huella), VinculoCertificado: ref("vinculo:1", huella),
			CuentaPersonaCuentaRef: "cuenta:1", CuentaPersonaPersonaRef: persona,
			VinculoCuentaRef: "cuenta:1", VinculoPersonaRef: persona, VinculoDERSHA256: f.CertificadoHuella,
		},
		Competencia: vecdomain.AsignacionFirmanteHistoricaV1{
			Asignacion: ref("asignacion:1", huella), Rol: ref("rol:ct_direccion_rrhh:v1", huella),
			ControlRol: ref("control:1", huella), RolID: "ct_direccion_rrhh", PersonaRef: persona,
			PerfilEsperadoRef: "perfil:ct:firma", PerfilActivoRef: "prf_1234567890abcdef1234567890abcdef",
			ModuloID: ports.ModuloContratacion, TipoRecurso: "documento", RecursoRef: f.OriginalRef,
			AmbitoOrganizacionRef: "organizacion:central", AmbitoUnidadRef: "unidad:prueba",
			AsignacionRolRef: "rol:ct_direccion_rrhh:v1", ControlRolRef: "rol:ct_direccion_rrhh:v1",
			VigenteDesde: desde, VigenteHasta: hasta,
		},
		Personal: vecdomain.FuentePersonalFirmanteHistoricaV1{
			Cargo: ref("cargo:1", huella), EnlaceOcupante: ref("ocupante:1", huella),
			OcupantePersonaRef: persona, CargoRefEnlace: "cargo:1",
			CargoVigenteDesde: desde, CargoVigenteHasta: hasta,
			EnlaceVigenteDesde: desde, EnlaceVigenteHasta: hasta,
		},
		Recurso: vecdomain.RecursoFirmaHistoricaV1{
			OrganizacionRef: "organizacion:central", UnidadRef: "unidad:prueba", ExpedienteRef: "expediente:prueba",
			DocumentoRef: f.OriginalRef, RecursoAutorizableRef: f.OriginalRef, ModuloID: ports.ModuloContratacion,
			TipoRecurso: "documento", RecursoContextoSHA256: strings.Repeat("1", 64),
			Original: ref(f.OriginalRef, f.OriginalHuella), PDFRaizSHA256: f.OriginalHuella,
			Firmado: ref(f.DocumentoCustodiaRef, f.FirmadoHuella), PDFFirmadoSHA256: f.FirmadoHuella,
			NumeroFirmas: 1,
		},
		RelacionCT: vecdomain.RelacionCTFirmanteHistoricaV1{
			ExpedienteRef: "expediente:prueba", UnidadRef: "unidad:prueba", OrigenRef: "ct050:1", OrigenVersion: 4,
			PruebaSnapshotSHA256: strings.Repeat("d", 64), EventoRef: "evento:1",
			EventoHuellaSHA256: strings.Repeat("e", 64), ConfirmadaEn: desde,
		},
		Accion: ports.AccionRegistrarFirmaVec, Finalidad: ports.FinalidadFirmaDocumento,
		Motivo: vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivo_firma", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("f", 64), EntradaClave: "competencia"},
		Circuito: ref(f.CatalogoRef, f.CatalogoHuella), PasoRef: f.PasoRef, PasoOrden: 1,
		FechaHistorica: f.RegistradaEn,
	}
	canon.Recurso.Original.Version = f.OriginalVersion
	canon.Recurso.Firmado.Version = f.DocumentoCustodiaVersion
	b, err := canon.Canonico()
	if err != nil || len(b) < 512 {
		t.Fatalf("fixture nominal: %v, %d bytes", err, len(b))
	}
	h := sha256.Sum256(b)
	return ports.LecturaRecuperacionFirmasV2{LecturaFirmasR5V2: base,
		Recuperaciones: []ports.RecuperacionFirmaV2{{FirmaRef: revision.FirmaRef, MaterialRootSHA256: strings.Repeat("2", 64),
			CanonNominal: string(b), CanonNominalSHA256: hex.EncodeToString(h[:]),
			CanonNominalRef: "evidencia:competencia-firmante-ct:" + strings.Repeat("3", 64)}}}
}

func TestRecuperacionCruzaCanonHistoricoConFirmaYRevision(t *testing.T) {
	l := lecturaRecuperacionPrueba(t)
	m := materialRecuperacionPrueba()
	r, err := proyectarRecuperacion(m, l)
	if err != nil || len(r.Recuperaciones) != 1 || r.Recuperaciones[0].CanonNominal != l.Recuperaciones[0].CanonNominal {
		t.Fatalf("canon original perdido: %v", err)
	}
	for nombre, alterar := range map[string]func(*ports.LecturaRecuperacionFirmasV2){
		"root": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones[0].MaterialRootSHA256 = "invalida" },
		"huella": func(x *ports.LecturaRecuperacionFirmasV2) {
			x.Recuperaciones[0].CanonNominalSHA256 = strings.Repeat("0", 64)
		},
		"ausente": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones = nil },
		"duplicada": func(x *ports.LecturaRecuperacionFirmasV2) {
			x.Recuperaciones = append(x.Recuperaciones, x.Recuperaciones[0])
		},
		"firma ajena": func(x *ports.LecturaRecuperacionFirmasV2) { x.Recuperaciones[0].FirmaRef = "firma:ajena" },
		"documento": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, func(c *vecdomain.CanonCompetenciaFirmanteHistoricaV1) {
				c.Recurso.DocumentoRef, c.Recurso.RecursoAutorizableRef, c.Competencia.RecursoRef = "ref:ajeno", "ref:ajeno", "ref:ajeno"
			})
		},
		"pdf": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, func(c *vecdomain.CanonCompetenciaFirmanteHistoricaV1) {
				c.Recurso.Original.HuellaSHA256, c.Recurso.PDFRaizSHA256 = strings.Repeat("4", 64), strings.Repeat("4", 64)
			})
		},
		"paso": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, func(c *vecdomain.CanonCompetenciaFirmanteHistoricaV1) { c.PasoOrden = 2 })
		},
		"fecha": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, func(c *vecdomain.CanonCompetenciaFirmanteHistoricaV1) {
				c.FechaHistorica = c.FechaHistorica.Add(time.Second)
			})
		},
		"certificado": func(x *ports.LecturaRecuperacionFirmasV2) {
			mutarCanonPrueba(t, x, func(c *vecdomain.CanonCompetenciaFirmanteHistoricaV1) {
				c.Identidad.CertificadoDERSHA256, c.Identidad.VinculoDERSHA256 = strings.Repeat("4", 64), strings.Repeat("4", 64)
			})
		},
		"esquema": func(x *ports.LecturaRecuperacionFirmasV2) {
			v := &x.Recuperaciones[0]
			v.CanonNominal = strings.Replace(v.CanonNominal, vecdomain.EsquemaCanonCompetenciaFirmanteHistoricaV1, "otro", 1)
			h := sha256.Sum256([]byte(v.CanonNominal))
			v.CanonNominalSHA256 = hex.EncodeToString(h[:])
		},
		"clave duplicada": func(x *ports.LecturaRecuperacionFirmasV2) {
			v := &x.Recuperaciones[0]
			v.CanonNominal = strings.Replace(v.CanonNominal, `"paso_ref":`, `"paso_ref":"paso:ajeno","paso_ref":`, 1)
			h := sha256.Sum256([]byte(v.CanonNominal))
			v.CanonNominalSHA256 = hex.EncodeToString(h[:])
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			x := lecturaRecuperacionPrueba(t)
			alterar(&x)
			r, err := proyectarRecuperacion(m, x)
			if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || len(r.Firmas) != 0 || len(r.Recuperaciones) != 0 {
				t.Fatalf("material incoherente expuesto: %v", err)
			}
		})
	}
}

func mutarCanonPrueba(t *testing.T, l *ports.LecturaRecuperacionFirmasV2, cambiar func(*vecdomain.CanonCompetenciaFirmanteHistoricaV1)) {
	t.Helper()
	v := &l.Recuperaciones[0]
	c, err := vecdomain.RecuperarCanonCompetenciaFirmanteHistoricaV1([]byte(v.CanonNominal), v.CanonNominalSHA256)
	if err != nil {
		t.Fatal(err)
	}
	cambiar(&c)
	b, err := c.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	v.CanonNominal = string(b)
	h := sha256.Sum256(b)
	v.CanonNominalSHA256 = hex.EncodeToString(h[:])
}

func materialRecuperacionPrueba() ports.MaterialConsultaFirmasR5V2 {
	q := solicitudPrueba()
	return ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: "organizacion:central", ExpedienteRef: q.ExpedienteRef, VersionExpediente: q.VersionExpediente,
		Documento: q.Documento, FirmantePrincipalCandidatoRef: "per_candidato_nominal", PasoOrden: q.PasoOrden,
		ClaveIdempotencia: q.ClaveIdempotencia, CatalogoHuella: q.CatalogoHuella}, Via: q.Via}
}

type autorizadorRecuperacionPrueba struct {
	t        *testing.T
	ajena    bool
	consulta bool
	campos44 bool
	llamado  int
}

func (a *autorizadorRecuperacionPrueba) AutorizarRecuperacionFirmasV2(_ context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadRecuperacionFirmasV2, error) {
	a.llamado++
	if a.ajena {
		m.ExpedienteRef = "expediente:ajeno"
	}
	base := &autorizadorPrueba{t: a.t}
	c, err := base.AutorizarConsultaFirmasR5V2(context.Background(), m)
	if err != nil {
		return ports.CapacidadRecuperacionFirmasV2{}, err
	}
	x := c.ExportarMaterialParaConsumidor()
	if a.consulta {
		return ports.TransportarMaterialRecuperacionFirmasV2(x, ports.CamposRecuperacionFirmasV2(), nil), nil
	}
	s := x.ResumenCapacidad()
	res, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(s.DecisionRef(), s.DecisionHuellaSHA256(),
		s.MotivoHuellaSHA256(), s.ContextoRef(), s.ContextoHuellaSHA256(), ports.AccionRecuperarFirmasR5V2,
		s.EfectoRef(), s.EfectoHuellaSHA256(), ports.AudienciaRecuperacionFirmasR5V2, s.EmitidaEn(), s.ExpiraEn())
	if err != nil {
		a.t.Fatal(err)
	}
	y, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), res,
		x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), x.PersonaVersion(), x.PerfilVersion(),
		x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	if err != nil {
		a.t.Fatal(err)
	}
	campos := ports.CamposRecuperacionFirmasV2()
	if a.campos44 {
		campos = ports.CamposConsultaFirmasR5V2()
	}
	return ports.TransportarMaterialRecuperacionFirmasV2(y, campos, nil), nil
}

type lectorRecuperacionPrueba struct {
	l       ports.LecturaRecuperacionFirmasV2
	llamado int
	despues func()
}

func (l *lectorRecuperacionPrueba) RecuperarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadRecuperacionFirmasV2) (ports.LecturaRecuperacionFirmasV2, error) {
	l.llamado++
	if l.despues != nil {
		l.despues()
	}
	return l.l, nil
}

func TestServicioRecuperacionExigeCapacidadPropiaYNoExponeTrasCancelar(t *testing.T) {
	a := &autorizadorRecuperacionPrueba{t: t, ajena: true}
	l := &lectorRecuperacionPrueba{l: lecturaRecuperacionPrueba(t)}
	s, err := NuevaRecuperacion(&fuentePrueba{c: Contexto{"organizacion:central", "per_candidato_nominal"}}, a, l)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("capacidad ajena consumida: %v", err)
	}
	a.ajena = false
	a.consulta = true
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("acción de consulta antigua consumida: %v", err)
	}
	a.consulta = false
	a.campos44 = true
	if _, err := s.Recuperar(context.Background(), solicitudPrueba()); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || l.llamado != 0 {
		t.Fatalf("concesión de 44 campos consumida: %v", err)
	}
	a.campos44 = false
	ctx, cancelar := context.WithCancel(context.Background())
	l.despues = cancelar
	r, err := s.Recuperar(ctx, solicitudPrueba())
	if !errors.Is(err, context.Canceled) || !CubiertoPorLector(err) || len(r.Recuperaciones) != 0 {
		t.Fatalf("cancelación expone datos: %v", err)
	}
}
