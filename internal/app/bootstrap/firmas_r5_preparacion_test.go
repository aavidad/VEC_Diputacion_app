package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func peticionComprobadoresR5() (ports.SolicitudDisponibilidadFirmaR5, domain.CompetenciaPasoFirmaV2) {
	circuito := domain.VersionPlanFirmaV2{Referencia: "circuito_firma_001", Version: 1, HuellaSHA256: strings.Repeat("a", 64)}
	paso := domain.CompetenciaPasoFirmaV2{EntradaClave: "paso_firma_001", Circuito: circuito,
		Documento: "resolucion", PasoRef: "paso_001", PasoOrden: 1,
		PerfilEsperadoRef: "perfil_firmante_001", RolID: "rol_firmante_001", CargoRef: "cargo_firmante_001",
		OrganizacionRef: "organizacion_ct_001", UnidadRef: "unidad_ct_001", Accion: "firma_competencial",
		Finalidad: "firmar_documento", TipoRecurso: "documento_ct", EsquemaContexto: "esquema_contexto_001",
		MapeoVersion: 1, MapeoFuenteRef: "fuente_mapeo_001"}
	q := ports.SolicitudDisponibilidadFirmaR5{Preflight: ports.SolicitudPreflightFirmaR5{
		Canal:     ports.SolicitudConsultaCircuitoRRHH{OrganizacionRef: paso.OrganizacionRef, ExpedienteRef: "expediente:ct:001"},
		Documento: paso.Documento}, CatalogoRef: circuito.Referencia, CatalogoHuella: circuito.HuellaSHA256,
		PasoRef: paso.PasoRef, PasoOrden: int(paso.PasoOrden)}
	return q, paso
}

type repositorioCustodiaNoInvocadoR5 struct {
	docports.RepositorioCustodiaFirmado
}
type almacenNoInvocadoR5 struct{ vecports.AlmacenObjetos }

func TestPreparacionR5CustodiaLeePoliticaDelObjeto(t *testing.T) {
	politicas, err := conservacion.NuevoCatalogoProvisional(relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	documentos := &custodiaDocumentosDesarrollo{repositorio: &repositorioCustodiaNoInvocadoR5{},
		almacen: &almacenNoInvocadoR5{}, politicas: politicas, reloj: relojContratacionTemporalDesarrollo{},
		documentos: map[string]string{"resolucion": "contratacion_temporal.resolucion_firmada.v1"}}
	c, err := nuevoComprobadorCustodiaPreparadaR5(documentos, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	q, paso := peticionComprobadoresR5()
	e, err := c.ComprobarCustodiaPreparadaR5(context.Background(), q, paso)
	if err != nil || e.Version != 1 || e.Referencia == "" || !domain.HuellaSHA256FirmaValida(e.HuellaSHA256) || !e.VigenteHasta.After(time.Now()) {
		t.Fatalf("política real no acreditada: %+v %v", e, err)
	}
	delete(documentos.documentos, q.Preflight.Documento)
	if _, err := c.ComprobarCustodiaPreparadaR5(context.Background(), q, paso); !errors.Is(err, ports.ErrPreparacionExternaR5NoAcreditada) {
		t.Fatal("mapeo retirado aún preparó custodia")
	}
}

type filaCatalogoRegistroR5Prueba struct{ valores []any }

func (f filaCatalogoRegistroR5Prueba) Scan(dest ...any) error {
	if len(dest) != len(f.valores) {
		return errors.New("fila incompleta")
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(f.valores[i]))
	}
	return nil
}

type lectorCatalogoRegistroR5Prueba struct {
	valores  []any
	consulta string
}

func (l *lectorCatalogoRegistroR5Prueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	l.consulta = sql
	return filaCatalogoRegistroR5Prueba{valores: l.valores}
}

func TestPreparacionR5RegistroCompruebaDosFachadasYLogin(t *testing.T) {
	valores := []any{"vec_ct_ejecutor_prueba", "vec_ct_ejecutor_prueba", true, true, true,
		true, true, true, true, "registrar_firma_con_plan_v4", "seleccionar_firmante_plan_ct_v1",
		"CREATE FUNCTION registrar_firma_con_plan_v4() RETURNS void", "CREATE FUNCTION seleccionar_firmante_plan_ct_v1() RETURNS void"}
	lector := &lectorCatalogoRegistroR5Prueba{valores: valores}
	c, err := nuevoComprobadorRegistroConPlanR5(lector, "vec_ct_ejecutor_prueba", relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	q, paso := peticionComprobadoresR5()
	e, err := c.ComprobarRegistroConPlanR5(context.Background(), q, paso)
	if err != nil || e.Version != 4 || !domain.HuellaSHA256FirmaValida(e.HuellaSHA256) ||
		!strings.Contains(lector.consulta, "pg_catalog.to_regprocedure") ||
		!strings.Contains(lector.consulta, "seleccionar_firmante_plan_ct_v1") {
		t.Fatalf("contrato CT185/AUT56 no acreditado: %+v %v", e, err)
	}
	lector.valores = append([]any(nil), valores...)
	lector.valores[8] = false
	if _, err := c.ComprobarRegistroConPlanR5(context.Background(), q, paso); !errors.Is(err, ports.ErrPreparacionExternaR5NoAcreditada) {
		t.Fatal("sin EXECUTE AUT56 se anunció registro")
	}
	lector.valores = append([]any(nil), valores...)
	lector.valores[1] = "otro_login"
	if _, err := c.ComprobarRegistroConPlanR5(context.Background(), q, paso); !errors.Is(err, ports.ErrPreparacionExternaR5NoAcreditada) {
		t.Fatal("SET ROLE ajeno se anunció como LOGIN CT")
	}
}

func TestPreparacionR5VerificadorExigeDescriptorYMaterialExacto(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Second)
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{SerialNumber: big.NewInt(1),
		Subject: pkix.Name{CommonName: "R5 test CA"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign},
		&x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "R5 test CA"},
			NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), IsCA: true,
			BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	token := []byte(strings.Repeat("t", 40))
	dir := t.TempDir()
	guardar := func(nombre string, b []byte) string {
		ruta := filepath.Join(dir, nombre)
		if err := os.WriteFile(ruta, b, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	caRuta, tokenRuta := guardar("ca.pem", ca), guardar("token", token)
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement, DocumentosEnabled: "true",
		FirmaVerificacionEnabled: "true", FirmaVerificacionURL: "https://localhost",
		FirmaVerificacionCAFile: caRuta, FirmaVerificacionTokenFile: tokenRuta, FirmaVerificacionTimeout: "5s"}
	desc := DescriptorConfiguracionVerificadorR5{Referencia: "configuracion:grxfirma:r5:001", Version: 3,
		HuellaSHA256: huellaMaterialVerificadorR5([]byte("true"), []byte(cfg.FirmaVerificacionURL),
			nil, []byte(cfg.FirmaVerificacionTimeout), ca, token, nil, nil), VigenteHasta: ahora.Add(2 * time.Hour)}
	if _, _, err := nuevosVerificadorYComprobadorPreparacionR5(cfg, DescriptorConfiguracionVerificadorR5{}, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("descriptor ausente aceptado")
	}
	apagada := cfg
	apagada.FirmaVerificacionEnabled = "false"
	if cliente, comprobador, err := nuevosVerificadorYComprobadorPreparacionR5(apagada, desc, relojContratacionTemporalDesarrollo{}); err == nil || cliente != nil || comprobador != nil {
		t.Fatal("cliente apagado aceptado")
	}
	divergente := desc
	divergente.HuellaSHA256 = strings.Repeat("a", 64)
	if _, _, err := nuevosVerificadorYComprobadorPreparacionR5(cfg, divergente, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("material divergente aceptado")
	}
	caducado := desc
	caducado.VigenteHasta = ahora.Add(-time.Second)
	if _, _, err := nuevosVerificadorYComprobadorPreparacionR5(cfg, caducado, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("descriptor vencido aceptado")
	}
	cliente, c, err := nuevosVerificadorYComprobadorPreparacionR5(cfg, desc, relojContratacionTemporalDesarrollo{})
	if err != nil || cliente == nil || c == nil || c.cliente != cliente {
		t.Fatal(err)
	}
	q, paso := peticionComprobadoresR5()
	e, err := c.ComprobarConfiguracionVerificadorR5(context.Background(), q, paso)
	if err != nil || e.HuellaSHA256 != desc.HuellaSHA256 || e.Version != desc.Version || e.VigenteHasta.After(ahora.Add(time.Hour)) {
		t.Fatalf("configuración no acreditada: %+v %v", e, err)
	}
	guardar("token", []byte(strings.Repeat("u", 40)))
	if _, err := c.ComprobarConfiguracionVerificadorR5(context.Background(), q, paso); !errors.Is(err, ports.ErrPreparacionExternaR5NoAcreditada) {
		t.Fatal("rotación de material no cerró la vía")
	}
}
