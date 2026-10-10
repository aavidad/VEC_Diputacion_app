package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
)

// El conjunto 0 conserva byte a byte los descriptores de AD188.
func TestConjuntoCeroConservaDescriptoresAD188(t *testing.T) {
	usuarios, ok := AudienciasConjuntoCapacidadesAdmin(0)
	ahora := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	d := descriptoresClavesUsuariosAdmin(usuarios, 7, 4, 9, ahora, time.Hour)
	if !ok || len(d) != 2 ||
		d[0].Audiencia != administracion.AudienciaUsuariosListarV3 || d[0].Dominio != "vec.admin.desarrollo.usuarios.listar.capacidad-v3.s7" ||
		d[0].PrefijoClave != "clave:capacidad:admin:usuarios:listar:s7:" || d[0].EmisorID != "emisor:admin:usuarios:desarrollo:v1" ||
		d[1].Audiencia != administracion.AudienciaUsuariosConsultarV3 || d[1].Dominio != "vec.admin.desarrollo.usuarios.consultar.capacidad-v3.s7" ||
		d[1].PrefijoClave != "clave:capacidad:admin:usuarios:consultar:s7:" || d[1].Version != 6 || d[1].RevisionGobierno != 11 {
		t.Fatalf("descriptores del conjunto 0 distintos de AD188: %+v", d)
	}
	if _, ok := AudienciasConjuntoCapacidadesAdmin(6); ok {
		t.Fatal("conjunto desconocido aceptado")
	}
}

// El conjunto 1 añade la clave del lote, en tercer lugar y con su segmento,
// con el mismo formato que exige AD198.
func TestConjuntoUnoDerivaClaveDelLote(t *testing.T) {
	formatoAD198 := regexp.MustCompile(`^clave:capacidad:admin:[a-z0-9:._-]{1,160}$`)
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	conjunto, ok := AudienciasConjuntoCapacidadesAdmin(1)
	if !ok || len(conjunto) != 3 {
		t.Fatal("conjunto 1 incompleto")
	}
	cfg.ConjuntoVersion = 1
	cfg.Entradas = descriptoresClavesUsuariosAdmin(conjunto, 20261006, 4, 9, reloj.Ahora(), time.Hour)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	conf, _, err := m.Configuracion()
	if err != nil || len(conf.EntradasCapacidad) != 3 {
		t.Fatal("material del conjunto 1 incompleto")
	}
	vistas := map[string]bool{}
	for i, e := range conf.EntradasCapacidad {
		if e.Audiencia != conjunto[i].Audiencia || !formatoAD198.MatchString(e.ClaveID) ||
			!strings.HasPrefix(e.ClaveID, "clave:capacidad:admin:"+conjunto[i].Segmento+":") || vistas[string(e.Material)] {
			t.Fatalf("clave %d fuera del conjunto o repetida: %s", i, e.ClaveID)
		}
		vistas[string(e.Material)] = true
	}
	if conf.EntradasCapacidad[2].Audiencia != administracion.AudienciaLoteOrdinarioV3 || conf.EntradasCapacidad[2].EmisorID != "emisor:admin:perfiles:desarrollo:v1" {
		t.Fatal("la tercera clave no es la del lote")
	}
}

// Un material de un conjunto no admite las entradas de otro ni otro orden.
func TestConjuntoRechazaEntradasAjenasODesordenadas(t *testing.T) {
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	uno, _ := AudienciasConjuntoCapacidadesAdmin(1)
	cero, _ := AudienciasConjuntoCapacidadesAdmin(0)
	for nombre, caso := range map[string]struct {
		conjunto uint64
		entradas []DescriptorClaveUsuariosAdmin
	}{
		"cero_con_tres": {0, descriptoresClavesUsuariosAdmin(uno, 9, 4, 9, reloj.Ahora(), time.Hour)},
		"uno_con_dos":   {1, descriptoresClavesUsuariosAdmin(cero, 9, 4, 9, reloj.Ahora(), time.Hour)},
		"uno_desordenado": {1, func() []DescriptorClaveUsuariosAdmin {
			d := descriptoresClavesUsuariosAdmin(uno, 9, 4, 9, reloj.Ahora(), time.Hour)
			d[0], d[2] = d[2], d[0]
			return d
		}()},
		"conjunto_inexistente": {5, descriptoresClavesUsuariosAdmin(uno, 9, 4, 9, reloj.Ahora(), time.Hour)},
	} {
		c := cfg
		c.ConjuntoVersion, c.Entradas = caso.conjunto, caso.entradas
		if m, err := PrepararMaterialUsuariosAdmin(context.Background(), c, reloj); err == nil {
			m.Cerrar()
			t.Fatalf("%s aceptado", nombre)
		}
	}
}

// El plan 2 (AD198) exige el conjunto del material, gca_ y una orden por
// clave; el plan 1 no sirve para un material de conjunto ni al revés.
func TestPlanDosLigadoAlConjuntoDelMaterial(t *testing.T) {
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	uno, _ := AudienciasConjuntoCapacidadesAdmin(1)
	cfg.ConjuntoVersion = 1
	cfg.Entradas = descriptoresClavesUsuariosAdmin(uno, 9, 4, 9, reloj.Ahora(), time.Hour)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	base, _ := planGobiernoUsuariosPrueba(t, m)
	firmar := func(mutar func(*planGobiernoUsuariosAdmin)) (string, string) {
		var p planGobiernoUsuariosAdmin
		if err := json.Unmarshal([]byte(base), &p); err != nil {
			t.Fatal(err)
		}
		p.Version, p.OperacionRef, p.ConjuntoVersion, p.Ordenes = 2, "gca_"+strings.Repeat("x", 22), 1, []uint64{101, 102, 103}
		mutar(&p)
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		return string(b), hex.EncodeToString(h[:])
	}
	if plan, sha := firmar(func(*planGobiernoUsuariosAdmin) {}); validarPlanGobiernoUsuariosAdmin(plan, sha, m) != nil {
		t.Fatal("plan 2 válido rechazado")
	}
	for nombre, mutar := range map[string]func(*planGobiernoUsuariosAdmin){
		"version_1":      func(p *planGobiernoUsuariosAdmin) { p.Version = 1 },
		"prefijo_gcu":    func(p *planGobiernoUsuariosAdmin) { p.OperacionRef = "gcu_" + strings.Repeat("x", 22) },
		"otro_conjunto":  func(p *planGobiernoUsuariosAdmin) { p.ConjuntoVersion = 2 },
		"sin_conjunto":   func(p *planGobiernoUsuariosAdmin) { p.ConjuntoVersion = 0 },
		"dos_ordenes":    func(p *planGobiernoUsuariosAdmin) { p.Ordenes = p.Ordenes[:2] },
		"orden_repetida": func(p *planGobiernoUsuariosAdmin) { p.Ordenes[2] = p.Ordenes[1] },
	} {
		if plan, sha := firmar(mutar); validarPlanGobiernoUsuariosAdmin(plan, sha, m) == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
}

// El conjunto 2 (AD202) conserva el 1 en el mismo orden y añade, en cuarto
// lugar, la audiencia del gobierno del plan nominal de firma con su tramo.
func TestConjuntoDosAnadeGobiernoPlanFirma(t *testing.T) {
	uno, _ := AudienciasConjuntoCapacidadesAdmin(1)
	dos, ok := AudienciasConjuntoCapacidadesAdmin(2)
	if administracion.AudienciaGobiernoPlanFirmaV3 != plannominal.AudienciaGobiernoPlanFirma {
		t.Fatal("la audiencia de vec-admin no es la del gobierno del plan")
	}
	if !ok || len(dos) != 4 || dos[3].Audiencia != plannominal.AudienciaGobiernoPlanFirma || dos[3].Segmento != "catalogos:plan-firma" ||
		!regexp.MustCompile(`^emisor:admin:[a-z0-9:._-]{1,120}$`).MatchString(dos[3].EmisorID) {
		t.Fatalf("conjunto 2 distinto de AD202: %+v", dos)
	}
	for i := range uno {
		if dos[i] != uno[i] {
			t.Fatalf("el conjunto 2 cambia la audiencia %d del conjunto 1", i)
		}
	}
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	cfg.ConjuntoVersion = 2
	cfg.Entradas = descriptoresClavesUsuariosAdmin(dos, 20261006, 4, 9, reloj.Ahora(), time.Hour)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	conf, _, err := m.Configuracion()
	if err != nil || len(conf.EntradasCapacidad) != 4 || conf.EntradasCapacidad[3].Audiencia != plannominal.AudienciaGobiernoPlanFirma ||
		!strings.HasPrefix(conf.EntradasCapacidad[3].ClaveID, "clave:capacidad:admin:catalogos:plan-firma:") {
		t.Fatal("material del conjunto 2 sin la clave del gobierno del plan")
	}
}

// El conjunto 3 (AD204) conserva el 2 en el mismo orden y añade, en quinto
// lugar, la audiencia de la publicación de cargos competenciales.
func TestConjuntoTresAnadeCargosCompetenciales(t *testing.T) {
	dos, _ := AudienciasConjuntoCapacidadesAdmin(2)
	tres, ok := AudienciasConjuntoCapacidadesAdmin(3)
	// La audiencia es la que fijan AD166 y Personal28 en SQL.
	if administracion.AudienciaCargoCompetencialV3 != "vec_personal.cargo_competencial.publicar.v1" {
		t.Fatal("la audiencia de vec-admin no es la de los cargos competenciales")
	}
	if !ok || len(tres) != 5 || tres[4].Audiencia != administracion.AudienciaCargoCompetencialV3 || tres[4].Segmento != "personal:cargo-competencial" ||
		!regexp.MustCompile(`^emisor:admin:[a-z0-9:._-]{1,120}$`).MatchString(tres[4].EmisorID) {
		t.Fatalf("conjunto 3 distinto de AD204: %+v", tres)
	}
	for i := range dos {
		if tres[i] != dos[i] {
			t.Fatalf("el conjunto 3 cambia la audiencia %d del conjunto 2", i)
		}
	}
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	cfg.ConjuntoVersion = 3
	cfg.Entradas = descriptoresClavesUsuariosAdmin(tres, 20261006, 4, 9, reloj.Ahora(), time.Hour)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	conf, _, err := m.Configuracion()
	if err != nil || len(conf.EntradasCapacidad) != 5 || conf.EntradasCapacidad[4].Audiencia != administracion.AudienciaCargoCompetencialV3 ||
		!strings.HasPrefix(conf.EntradasCapacidad[4].ClaveID, "clave:capacidad:admin:personal:cargo-competencial:") {
		t.Fatal("material del conjunto 3 sin la clave de cargos competenciales")
	}
}

// El conjunto 4 (AD205) conserva el 3 y añade la audiencia de certificados
// nominales, con su tramo.
func TestConjuntoCuatroAnadeCertificadosNominales(t *testing.T) {
	tres, _ := AudienciasConjuntoCapacidadesAdmin(3)
	cuatro, ok := AudienciasConjuntoCapacidadesAdmin(4)
	if administracion.AudienciaCertificadoNominalV3 != "vec_contexto_actor.certificado_nominal.publicar.v1" {
		t.Fatal("la audiencia de vec-admin no es la de AD165")
	}
	if !ok || len(cuatro) != 6 || cuatro[5].Audiencia != administracion.AudienciaCertificadoNominalV3 || cuatro[5].Segmento != "certificados:nominal" ||
		!regexp.MustCompile(`^emisor:admin:[a-z0-9:._-]{1,120}$`).MatchString(cuatro[5].EmisorID) {
		t.Fatalf("conjunto 4 distinto de AD205: %+v", cuatro)
	}
	for i := range tres {
		if cuatro[i] != tres[i] {
			t.Fatalf("el conjunto 4 cambia la audiencia %d del conjunto 3", i)
		}
	}
}
