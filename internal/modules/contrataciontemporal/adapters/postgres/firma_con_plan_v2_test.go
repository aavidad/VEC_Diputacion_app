package postgres

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

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func exportacionFirmaPlanPrueba(t *testing.T, m ports.MaterialFirmaVerificadaV2, recurso vd.RecursoAutorizable, decisionRef string) vp.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision, err := json.Marshal(map[string]string{
		"decision_ref": decisionRef, "principal_id": m.FirmantePrincipalRef, "perfil_activo_ref": m.PerfilActivoOperadorRef,
		"version_rol_ref": m.VersionRolFirmanteRef, "accion": ports.AccionRegistrarFirmaVec,
		"recurso_ref": m.RecursoRef(), "modulo_id": ports.ModuloContratacion,
		"tipo_recurso": ports.TipoRecursoFirmaVec, "finalidad": ports.FinalidadFirmaDocumento,
		"contexto_recurso_huella_sha256": h,
	})
	if err != nil {
		t.Fatal(err)
	}
	dh := sha256.Sum256(decision)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, hex.EncodeToString(dh[:]),
		strings.Repeat("a", 64), "contexto:plan-prueba", strings.Repeat("b", 64),
		ports.AccionRegistrarFirmaVec, m.RecursoRef(), h, ports.AudienciaFirmaVecV2,
		m.ComprobadaEn, m.ComprobadaEn.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte{'x'}, vp.TamanoMinimoCapacidadCanonicaV3), resumen, decision, []byte("{}"),
		[]byte("{}"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

func capacidadFirmaConPlanPrueba(t *testing.T, m ports.MaterialFirmaVerificadaV2) ports.CapacidadFirmaConPlanV2 {
	t.Helper()
	_, d := fixtureRegistroFirmaV2(t)
	descriptor, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d)
	if err != nil {
		t.Fatal(err)
	}
	recursoI, err := firma.RecursoFirmaVerificadaV2(m, descriptor)
	if err != nil {
		t.Fatal(err)
	}
	interior := exportacionFirmaPlanPrueba(t, m, recursoI, "decision:firma:interior")
	descSHA := sha256.Sum256(descriptor)
	capInterior := ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(interior, descriptor, hex.EncodeToString(descSHA[:]))
	decSHA := sha256.Sum256(interior.DecisionCanonica())
	plan := vd.ReferenciaEntradaCatalogo{CatalogoID: "plan.firma", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "entrada.uno"}
	envoltorio, err := firma.CanonicoPlanAutorizadoFirmaV2(m, ports.DescriptorPlanFijadoFirmaV2{Descriptor: d, Plan: plan}, hex.EncodeToString(decSHA[:]))
	if err != nil {
		t.Fatal(err)
	}
	recursoE, err := firma.RecursoPlanAutorizadoFirmaV2(m, plan, hex.EncodeToString(decSHA[:]), envoltorio, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef})
	if err != nil {
		t.Fatal(err)
	}
	exterior := exportacionFirmaPlanPrueba(t, m, recursoE, "decision:firma:exterior")
	return ports.TransportarFirmaConPlanV2(capInterior, exterior, plan, envoltorio, hex.EncodeToString(decSHA[:]))
}

func TestFirmaConPlanPGUnaFachadaYReciboOriginal(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(map[bool]string{false: "nueva", true: "replay"}[replay], func(t *testing.T) {
			m, _ := fixtureRegistroFirmaV2(t)
			c := capacidadFirmaConPlanPrueba(t, m)
			canon, _ := m.Canonico()
			h, _ := m.HuellaSHA256()
			ref, version := m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion
			w := reciboFirmaSQLV2{reciboFirmaSQL118: reciboFirmaSQL118{FirmaRef: "firma:original", ReciboRef: "recibo:original", Secuencia: m.Secuencia,
				Resultado: "firmado", ExpedienteVersion: m.VersionExpediente, ActorRef: m.FirmantePrincipalRef,
				PerfilRef: m.PerfilActivoOperadorRef, RegistradaEn: m.ComprobadaEn,
				SolicitudHuella: h, YaRegistrada: replay, DocumentoCustodia: &ref, VersionCustodia: &version},
				CompetenciaEvidenciaRef:          "evidencia:competencia-firmante-ct:" + strings.Repeat("d", 64),
				CompetenciaEvidenciaHuellaSHA256: strings.Repeat("e", 64)}
			contenido, _ := json.Marshal(w)
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido, sql: registrarFirmaConPlanSQL176, argumentos: 23}
			tx.inspeccionar = func(args []any) {
				interior, exterior, _, env, _ := c.ExportarParaConsumidor()
				if args[1] != m.ComprobadaEn || !bytes.Equal(args[2].([]byte), env) ||
					!bytes.Equal(args[4].([]byte), interior.ExportarMaterialParaConsumidor().DecisionCanonica()) ||
					!bytes.Equal(args[14].([]byte), exterior.DecisionCanonica()) {
					t.Fatal("una fachada recibió material o decisión cruzados")
				}
			}
			pool := &poolFirmaV2Prueba{tx: tx}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: pool}
			got, err := r.RegistrarFirmaConPlanV2(context.Background(), m, c)
			if err != nil || got.ReciboRef != w.ReciboRef || got.YaRegistrada != replay ||
				tx.commits != 1 || tx.rollbacks != 0 || tx.consultas != 1 || pool.inicios != 1 {
				t.Fatalf("fachada/recibo inválidos: %v", err)
			}
		})
	}
}

func TestFirmaConPlanPGCommitInciertoNoEntregaRecibo(t *testing.T) {
	m, _ := fixtureRegistroFirmaV2(t)
	c := capacidadFirmaConPlanPrueba(t, m)
	canon, _ := m.Canonico()
	h, _ := m.HuellaSHA256()
	ref, version := m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion
	contenido, _ := json.Marshal(reciboFirmaSQLV2{reciboFirmaSQL118: reciboFirmaSQL118{FirmaRef: "firma:original", ReciboRef: "recibo:original",
		Secuencia: m.Secuencia, Resultado: "firmado", ExpedienteVersion: m.VersionExpediente,
		ActorRef: m.FirmantePrincipalRef, PerfilRef: m.PerfilActivoOperadorRef,
		RegistradaEn: m.ComprobadaEn, SolicitudHuella: h, DocumentoCustodia: &ref, VersionCustodia: &version},
		CompetenciaEvidenciaRef:          "evidencia:competencia-firmante-ct:" + strings.Repeat("d", 64),
		CompetenciaEvidenciaHuellaSHA256: strings.Repeat("e", 64)})
	tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido, sql: registrarFirmaConPlanSQL176,
		argumentos: 23, falloCommit: errors.New("commit incierto")}
	r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
	got, err := r.RegistrarFirmaConPlanV2(context.Background(), m, c)
	if !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) || got.ReciboRef != "" ||
		tx.commits != 1 || tx.rollbacks != 1 || tx.consultas != 1 {
		t.Fatal("commit incierto entregó recibo o reintentó")
	}
}

func TestFirmaConPlanPGRechazaRespuestaIncompletaOAjenaAntesCommit(t *testing.T) {
	for _, caso := range []string{"evidencia_ausente", "campo_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			m, _ := fixtureRegistroFirmaV2(t)
			c := capacidadFirmaConPlanPrueba(t, m)
			canon, _ := m.Canonico()
			h, _ := m.HuellaSHA256()
			ref, version := m.DocumentoCustodiaRef, m.DocumentoCustodiaVersion
			base := reciboFirmaSQLV2{reciboFirmaSQL118: reciboFirmaSQL118{FirmaRef: "firma:original", ReciboRef: "recibo:original",
				Secuencia: m.Secuencia, Resultado: "firmado", ExpedienteVersion: m.VersionExpediente,
				ActorRef: m.FirmantePrincipalRef, PerfilRef: m.PerfilActivoOperadorRef,
				RegistradaEn: m.ComprobadaEn, SolicitudHuella: h, DocumentoCustodia: &ref, VersionCustodia: &version},
				CompetenciaEvidenciaRef:          "evidencia:competencia-firmante-ct:" + strings.Repeat("d", 64),
				CompetenciaEvidenciaHuellaSHA256: strings.Repeat("e", 64)}
			if caso == "evidencia_ausente" {
				base.CompetenciaEvidenciaRef = ""
			}
			contenido, _ := json.Marshal(base)
			if caso == "campo_ajeno" {
				contenido = bytes.Replace(contenido, []byte(`"CompetenciaEvidenciaRef":`),
					[]byte(`"PlanInventado":null,"CompetenciaEvidenciaRef":`), 1)
			}
			tx := &txFirmaV2Prueba{t: t, canonico: string(canon), contenido: contenido,
				sql: registrarFirmaConPlanSQL176, argumentos: 23}
			r := &RegistroFirmasVerificadasPostgreSQL{pool: &poolFirmaV2Prueba{tx: tx}}
			got, err := r.RegistrarFirmaConPlanV2(context.Background(), m, c)
			if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) ||
				got.ReciboRef != "" || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatalf("respuesta %s no cerró antes de COMMIT: %v", caso, err)
			}
		})
	}
}
