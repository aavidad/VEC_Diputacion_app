package bootstrap

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

func configuracionOrganizacionHistoricaPrueba() config.Config {
	return config.Config{
		OrganizacionHistoricaGobiernoEnabled: "true",
		ExecutionProfile:                     config.ExecutionProfileDevelopment,
		AuthMode:                             config.AuthModeDevelopment,
		DevelopmentGuard:                     config.DevelopmentGuardAcknowledgement,
	}
}

func TestOrganizacionHistoricaSeleccionOpcionalConservaCTYB2(t *testing.T) {
	previos := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTCompletaDesarrollo())
	copia := append([]descriptorMaterialConsumidorV3Desarrollo(nil), previos...)
	cfg := configuracionOrganizacionHistoricaPrueba()
	for _, valor := range []string{"", "false", "TRUE"} {
		cfg.OrganizacionHistoricaGobiernoEnabled = valor
		catalogo, activo, err := seleccionarMaterialOrganizacionHistorica(cfg, previos)
		if activo || len(catalogo.porAudiencia) != 0 || (valor == "TRUE" && !errors.Is(err, config.ErrConfiguracionOrganizacionHistoricaGobiernoSelector)) || (valor != "TRUE" && err != nil) {
			t.Fatalf("%q: selección inesperada %v %v", valor, activo, err)
		}
		if !reflect.DeepEqual(previos, copia) {
			t.Fatal("fallo OH alteró selección CT/B2")
		}
	}
	cfg.OrganizacionHistoricaGobiernoEnabled = "true"
	catalogo, activo, err := seleccionarMaterialOrganizacionHistorica(cfg, previos)
	if err != nil || !activo || len(catalogo.porAudiencia) != len(previos)+1 || !reflect.DeepEqual(previos, copia) {
		t.Fatal("extensión OH modifica capacidades previas", err)
	}
	for _, d := range previos {
		if conservado, ok := catalogo.descriptorPara(d.Audiencia); !ok || conservado != d {
			t.Fatalf("descriptor previo alterado: %s", d.Audiencia)
		}
	}
}

func TestOrganizacionHistoricaDescriptorNominalSeparadoDeB2(t *testing.T) {
	d := descriptorMaterialOrganizacionHistorica()
	if d.Audiencia != personal.AudienciaConsultaOrganizacionHistorica || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
		t.Fatal("audiencia OH no nominal o no publicable")
	}
	previos := descriptoresMaterialSeleccionadosCTDesarrollo(seleccionMaterialCTCompletaDesarrollo())
	for _, otro := range previos {
		if otro.Audiencia == d.Audiencia || otro.Dominio == d.Dominio || strings.HasPrefix(otro.Prefijo, d.Prefijo) || strings.HasPrefix(d.Prefijo, otro.Prefijo) {
			t.Fatalf("descriptor OH colisiona: %+v", otro)
		}
	}
	catalogo, activo, err := seleccionarMaterialOrganizacionHistorica(configuracionOrganizacionHistoricaPrueba(), nil)
	if err != nil || !activo || len(catalogo.porAudiencia) != 1 {
		t.Fatal("OH exige otras capacidades", err)
	}
	if _, activo, err := seleccionarMaterialOrganizacionHistorica(configuracionOrganizacionHistoricaPrueba(), []descriptorMaterialConsumidorV3Desarrollo{d}); activo || err == nil {
		t.Fatal("OH admite colisión con catálogo previo")
	}
}

func TestOrganizacionHistoricaPublicaMismoGobiernoYBorraClave(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC))
	defer m.borrarCopiasEfimeras()
	base := append([]byte(nil), m.claveHMAC...)
	catalogo, _, err := seleccionarMaterialOrganizacionHistorica(configuracionOrganizacionHistoricaPrueba(), nil)
	if err != nil {
		t.Fatal(err)
	}
	publicador := &publicadorGobiernoPrueba{versiones: map[string]uint64{}, siguiente: 40}
	primera, err := publicarMaterialOrganizacionHistoricaCon(m, catalogo, publicador.publicar)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := publicarMaterialOrganizacionHistoricaCon(m, catalogo, publicador.publicar)
	if err != nil || primera != segunda || publicador.siguiente != 41 || len(publicador.versiones) != 1 {
		t.Fatal("OH no conserva coordenadas en replay simulado", err)
	}
	if primera.Audiencia != personal.AudienciaConsultaOrganizacionHistorica || primera.ClaveID == m.claveHMACID || primera.SHA256 == m.claveHMACSecreto || primera.Version != 41 || primera.RevisionGobierno != 41 || primera.OrdenPuntero != 41 || primera.EmisorID != m.emisorID || primera.Desde != m.validaDesde || primera.Hasta != m.validaHasta {
		t.Fatalf("coordenadas OH incoherentes: %+v", primera)
	}
	for _, secreto := range publicador.secretos {
		if !bytes.Equal(secreto, make([]byte, len(secreto))) {
			t.Fatal("clave OH retenida al salir")
		}
	}
	if !bytes.Equal(m.claveHMAC, base) {
		t.Fatal("OH altera clave CT")
	}
	for _, d := range descriptoresMaterialPersonalB2Desarrollo() {
		b2, err := derivarMaterialConsumidorV3Desarrollo(m, d)
		if err != nil {
			t.Fatal(err)
		}
		igual := b2.claveHMACSecreto == primera.SHA256 || b2.claveHMACID == primera.ClaveID
		borrarBytes(b2.claveHMAC)
		if igual {
			t.Fatal("OH comparte clave B2")
		}
	}
}

func TestOrganizacionHistoricaFalloPublicacionBorraClaveSinResultado(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC))
	defer m.borrarCopiasEfimeras()
	base := append([]byte(nil), m.claveHMAC...)
	catalogo, _, err := seleccionarMaterialOrganizacionHistorica(configuracionOrganizacionHistoricaPrueba(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var retenida []byte
	esperado := errors.New("fallo sintético")
	publicada, err := publicarMaterialOrganizacionHistoricaCon(m, catalogo, func(d *materialAtestacionContratacionTemporalDesarrollo) error {
		if d.audienciaConsumo != personal.AudienciaConsultaOrganizacionHistorica {
			t.Fatal("publica audiencia ajena")
		}
		retenida = d.claveHMAC
		return esperado
	})
	if !errors.Is(err, esperado) || publicada != (CapacidadPublicadaOrganizacionHistoricaV3{}) || len(retenida) == 0 || !bytes.Equal(retenida, make([]byte, len(retenida))) || !bytes.Equal(m.claveHMAC, base) {
		t.Fatal("fallo OH deja clave o resultado y modifica CT", err)
	}
}

func TestOrganizacionHistoricaRechazaDescriptorAjenoAntesPublicar(t *testing.T) {
	d := descriptorMaterialOrganizacionHistorica()
	for _, alterar := range []func(*descriptorMaterialConsumidorV3Desarrollo){
		func(d *descriptorMaterialConsumidorV3Desarrollo) { d.Audiencia = personal.AudienciaFichaEmpleadoB2 },
		func(d *descriptorMaterialConsumidorV3Desarrollo) { d.Dominio = "dominio:ajeno" },
		func(d *descriptorMaterialConsumidorV3Desarrollo) { d.Prefijo = "clave:ajena:" },
		func(d *descriptorMaterialConsumidorV3Desarrollo) { d.ProveedorNominal = "proveedor:ajeno" },
	} {
		alterado := d
		alterar(&alterado)
		catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{alterado})
		if err != nil {
			t.Fatal(err)
		}
		publicada, err := publicarMaterialOrganizacionHistoricaCon(materialAtestacionContratacionTemporalDesarrollo{}, catalogo, func(*materialAtestacionContratacionTemporalDesarrollo) error {
			t.Fatal("publica descriptor ajeno")
			return nil
		})
		if !errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente) || publicada != (CapacidadPublicadaOrganizacionHistoricaV3{}) {
			t.Fatal("descriptor ajeno admitido", err)
		}
	}
}
