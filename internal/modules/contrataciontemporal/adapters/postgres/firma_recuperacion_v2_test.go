package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func capacidadRecuperacionFirmaV2Prueba(t *testing.T, m ports.MaterialConsultaFirmasR5V2, campos []string) ports.CapacidadRecuperacionFirmasV2 {
	t.Helper()
	c := capacidadConsultaFirmaV2Prueba(t, m)
	x := c.ExportarMaterialParaConsumidor()
	s := x.ResumenCapacidad()
	r, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(s.DecisionRef(), s.DecisionHuellaSHA256(), s.MotivoHuellaSHA256(),
		s.ContextoRef(), s.ContextoHuellaSHA256(), ports.AccionRecuperarFirmasR5V2, s.EfectoRef(), s.EfectoHuellaSHA256(),
		ports.AudienciaRecuperacionFirmasR5V2, s.EmitidaEn(), s.ExpiraEn())
	if err != nil {
		t.Fatal(err)
	}
	y, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(x.CapacidadCanonica(), r, x.DecisionCanonica(), x.MotivoCanonico(),
		x.ContextoActorCanonico(), x.PersonaVersion(), x.PerfilVersion(), x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	return ports.TransportarMaterialRecuperacionFirmasV2(y, campos, nil)
}

func fixtureRecuperacionSQL175(t *testing.T) (ports.MaterialConsultaFirmasR5V2, respuestaRecuperacionSQL175) {
	t.Helper()
	m, base := fixtureConsultaFirmaV2()
	custodia, version := "custodia:ct175:firmado", uint64(1)
	base.Firmas[0].DocumentoCustodia, base.Firmas[0].VersionCustodia = &custodia, &version
	base.RevisionesPDF[0].firmaExternaSQL170 = base.Firmas[0]
	f := base.Firmas[0]
	r := base.RevisionesPDF[0]
	ref := func(s, h string) vd.ReferenciaHistoricaCompetenciaV1 {
		return vd.ReferenciaHistoricaCompetenciaV1{Referencia: s, Version: 1, HuellaSHA256: h}
	}
	persona := "per_1234567890abcdef1234567890abcdef"
	antes := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	despues := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	c := vd.CanonCompetenciaFirmanteHistoricaV1{Esquema: vd.EsquemaCanonCompetenciaFirmanteHistoricaV1,
		Identidad: vd.IdentidadFirmanteHistoricaV1{CertificadoDERSHA256: r.CertificadoHuella, PersonaRef: persona,
			Persona: ref(persona, h), Cuenta: ref("cuenta:1", h), VinculoCuentaPersona: ref("vinculo-cuenta-persona:1", h),
			VinculoCertificado: ref("vinculo:1", h), CuentaPersonaCuentaRef: "cuenta:1", CuentaPersonaPersonaRef: persona,
			VinculoCuentaRef: "cuenta:1", VinculoPersonaRef: persona, VinculoDERSHA256: r.CertificadoHuella},
		Competencia: vd.AsignacionFirmanteHistoricaV1{Asignacion: ref("asignacion:1", h), Rol: ref("rol:ct_direccion_rrhh:v1", h),
			ControlRol: ref("control:1", h), RolID: "ct_direccion_rrhh", PersonaRef: persona, PerfilEsperadoRef: "perfil:ct:firma",
			PerfilActivoRef: "prf_1234567890abcdef1234567890abcdef", ModuloID: ports.ModuloContratacion, TipoRecurso: "documento",
			RecursoRef: *f.OriginalRef, AmbitoOrganizacionRef: m.OrganizacionRef, AmbitoUnidadRef: "unidad:prueba",
			AsignacionRolRef: "rol:ct_direccion_rrhh:v1", ControlRolRef: "rol:ct_direccion_rrhh:v1", VigenteDesde: antes, VigenteHasta: despues},
		Personal: vd.FuentePersonalFirmanteHistoricaV1{Cargo: ref("cargo:1", h), EnlaceOcupante: ref("ocupante:1", h),
			OcupantePersonaRef: persona, CargoRefEnlace: "cargo:1", CargoVigenteDesde: antes, CargoVigenteHasta: despues,
			EnlaceVigenteDesde: antes, EnlaceVigenteHasta: despues},
		Recurso: vd.RecursoFirmaHistoricaV1{OrganizacionRef: m.OrganizacionRef, UnidadRef: "unidad:prueba", ExpedienteRef: m.ExpedienteRef,
			DocumentoRef: *f.OriginalRef, RecursoAutorizableRef: *f.OriginalRef, ModuloID: ports.ModuloContratacion,
			TipoRecurso: "documento", RecursoContextoSHA256: strings.Repeat("1", 64),
			Original: ref(*f.OriginalRef, *f.OriginalHuella), PDFRaizSHA256: *f.OriginalHuella,
			Firmado: ref(custodia, *f.FirmadoHuella), PDFFirmadoSHA256: *f.FirmadoHuella, NumeroFirmas: 1},
		RelacionCT: vd.RelacionCTFirmanteHistoricaV1{ExpedienteRef: m.ExpedienteRef, UnidadRef: "unidad:prueba",
			OrigenRef: "ct050:1", OrigenVersion: 4, PruebaSnapshotSHA256: strings.Repeat("d", 64), EventoRef: "evento:1",
			EventoHuellaSHA256: strings.Repeat("e", 64), ConfirmadaEn: antes},
		Accion: ports.AccionRegistrarFirmaVec, Finalidad: ports.FinalidadFirmaDocumento,
		Motivo:   vd.ReferenciaEntradaCatalogo{CatalogoID: "motivo_firma", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("f", 64), EntradaClave: "competencia"},
		Circuito: ref(f.CatalogoRef, f.CatalogoHuella), PasoRef: f.PasoRef, PasoOrden: 1, FechaHistorica: f.RegistradaEn}
	c.Recurso.Original.Version = *f.OriginalVersion
	canon, err := c.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(canon)
	rec := recuperacionFirmaSQL175{FirmaRef: f.FirmaRef, MaterialRootSHA256: strings.Repeat("2", 64), CanonNominal: string(canon),
		CanonNominalSHA256: hex.EncodeToString(suma[:]), CanonNominalRef: "evidencia:competencia-firmante-ct:" + strings.Repeat("3", 64)}
	return m, respuestaRecuperacionSQL175{respuestaFirmasR5SQL172: base, Recuperaciones: []recuperacionFirmaSQL175{rec}}
}

func TestRecuperacionPGConfirmaCanonOriginalSoloTrasCommit(t *testing.T) {
	m, w := fixtureRecuperacionSQL175(t)
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := m.Canonico()
	tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: b, sql: recuperarFirmasSQL175, argumentos: 11}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
	l, err := r.RecuperarFirmasAutorizadasV2(context.Background(), m, capacidadRecuperacionFirmaV2Prueba(t, m, ports.CamposRecuperacionFirmasV2()))
	if err != nil || tx.commits != 1 || tx.rollbacks != 0 || len(l.Recuperaciones) != 1 || l.Recuperaciones[0].CanonNominal != w.Recuperaciones[0].CanonNominal ||
		l.Firmas[0].CertificadoHuella != w.RevisionesPDF[0].CertificadoHuella {
		t.Fatalf("historia original no confirmada: %v", err)
	}
}

func TestRecuperacionPGDeniegaCapacidad44YMaterialAjenoAntesDeBegin(t *testing.T) {
	m, _ := fixtureRecuperacionSQL175(t)
	tx := &txFirmaV2Prueba{t: t}
	p := &poolFirmaV2Prueba{tx: tx}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: p}
	if _, err := r.RecuperarFirmasAutorizadasV2(context.Background(), m, capacidadRecuperacionFirmaV2Prueba(t, m, ports.CamposConsultaFirmasR5V2())); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatal("44 campos habilitan recuperación")
	}
	ajeno := m
	ajeno.Via = ports.ViaFirmaExternaPortafirmas
	if _, err := r.RecuperarFirmasAutorizadasV2(context.Background(), ajeno, capacidadRecuperacionFirmaV2Prueba(t, m, ports.CamposRecuperacionFirmasV2())); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatal("capacidad cruzada habilita recuperación")
	}
}

func TestRecuperacionPGRevierteCommitInciertoYCanonManipulado(t *testing.T) {
	for _, caso := range []string{"commit", "canon", "campo", "modulo"} {
		t.Run(caso, func(t *testing.T) {
			m, w := fixtureRecuperacionSQL175(t)
			if caso == "canon" {
				w.Recuperaciones[0].CanonNominalSHA256 = strings.Repeat("0", 64)
			}
			if caso == "modulo" {
				original, err := vd.RecuperarCanonCompetenciaFirmanteHistoricaV1([]byte(w.Recuperaciones[0].CanonNominal),
					w.Recuperaciones[0].CanonNominalSHA256)
				if err != nil {
					t.Fatal(err)
				}
				original.Recurso.ModuloID = "modulo_ajeno"
				original.Competencia.ModuloID = "modulo_ajeno"
				canon, err := original.Canonico()
				if err != nil {
					t.Fatal(err)
				}
				huella := sha256.Sum256(canon)
				w.Recuperaciones[0].CanonNominal = string(canon)
				w.Recuperaciones[0].CanonNominalSHA256 = hex.EncodeToString(huella[:])
			}
			b, err := json.Marshal(w)
			if err != nil {
				t.Fatal(err)
			}
			if caso == "campo" {
				b = bytes.Replace(b, []byte(`"Recuperaciones":`), []byte(`"Ajeno":null,"Recuperaciones":`), 1)
			}
			canon, _ := m.Canonico()
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: b, sql: recuperarFirmasSQL175, argumentos: 11}
			if caso == "commit" {
				tx.falloCommit = errors.New("commit incierto")
			}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
			l, err := r.RecuperarFirmasAutorizadasV2(context.Background(), m, capacidadRecuperacionFirmaV2Prueba(t, m, ports.CamposRecuperacionFirmasV2()))
			if err == nil || len(l.Recuperaciones) != 0 || tx.rollbacks != 1 || (caso != "commit" && tx.commits != 0) {
				t.Fatalf("lectura incorrecta tras fallo: %v", err)
			}
			if caso == "canon" && !errors.Is(err, vd.ErrCanonCompetenciaFirmanteHistoricaV1Invalido) {
				t.Fatal("se perdió la causa del validador común")
			}
		})
	}
}
