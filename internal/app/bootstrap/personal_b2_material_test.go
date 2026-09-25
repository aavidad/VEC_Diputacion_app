package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

// descriptoresPreviosPersonalB2Prueba reúne todos los descriptores que ya
// publicaba vec-server antes de B2, incluidos los dos declarados en línea.
func descriptoresPreviosPersonalB2Prueba() []descriptorMaterialConsumidorV3Desarrollo {
	previos := descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()
	previos = append(previos, descriptoresMaterialBorradorLlamamientoBolsaDesarrollo()...)
	previos = append(previos, descriptoresMaterialDietasDesarrollo()...)
	previos = append(previos, descriptoresMaterialCronosDesarrollo()...)
	return append(previos,
		descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo, Dominio: "vec.bolsa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:bolsa:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaMiBolsa, Dominio: "vec.bolsa.mi-bolsa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:bolsa-mi-bolsa:", ProveedorNominal: "proveedor-material-bolsa-mi-bolsa"},
	)
}

func TestDescriptoresPersonalB2CoincidenConElConsumidorInterno(t *testing.T) {
	// Mismo orden y mismas audiencias que las capacidades de personal_b2_v3.go.
	esperadas := [8][2]string{
		{"ficha", personal.AudienciaFichaEmpleadoB2},
		{"vacantes", personal.AudienciaVacantesB2},
		{"alta", personal.AudienciaAltaEmpleadoB2},
		{"hecho", personal.AudienciaHechoEmpleadoB2},
		{"catalogo_consultar", personal.AudienciaConsultarCatalogoEmpleadoB2},
		{"catalogo_publicar", personal.AudienciaPublicarCatalogoEmpleadoB2},
		{"catalogo_retirar", personal.AudienciaRetirarCatalogoEmpleadoB2},
		{"empleados", personal.AudienciaEmpleadosB2},
	}
	for i, d := range DescriptoresCapacidadPersonalB2V3Desarrollo() {
		if d.Capacidad != esperadas[i][0] || d.Audiencia != esperadas[i][1] || !strings.HasPrefix(d.Audiencia, "vec_personal.registro_empleado.") {
			t.Fatalf("descriptor %d divergente: %+v", i, d)
		}
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia B2 no publicable: %s", d.Audiencia)
		}
	}
}

func TestDescriptoresPersonalB2DominiosSeparadosSinColision(t *testing.T) {
	b2 := descriptoresMaterialPersonalB2Desarrollo()
	if len(b2) != 8 {
		t.Fatal("B2 exige ocho descriptores")
	}
	todos := append(descriptoresPreviosPersonalB2Prueba(), b2...)
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); err != nil {
		t.Fatalf("B2 colisiona con descriptores previos: %v", err)
	}
	for _, d := range b2 {
		if !strings.HasPrefix(d.Dominio, "vec.personal.registro-empleado.") || !strings.HasPrefix(d.Prefijo, "clave:capacidad:personal-b2-") || !strings.HasSuffix(d.Prefijo, ":") {
			t.Fatalf("dominio o prefijo B2 no exclusivo: %+v", d)
		}
		// Ningún prefijo puede ser prefijo textual de otro: el identificador
		// de clave derivado no debe poder confundirse entre consumidores.
		for _, otro := range todos {
			if otro.Audiencia != d.Audiencia && (strings.HasPrefix(otro.Prefijo, d.Prefijo) || strings.HasPrefix(d.Prefijo, otro.Prefijo) || otro.Dominio == d.Dominio) {
				t.Fatalf("prefijo o dominio solapado: %s / %s", d.Prefijo, otro.Prefijo)
			}
		}
	}
}

func TestDescriptoresPersonalB2ColisionRechazada(t *testing.T) {
	previos := descriptoresPreviosPersonalB2Prueba()
	cronos := descriptoresMaterialCronosDesarrollo()[0]
	for i, d := range descriptoresMaterialPersonalB2Desarrollo() {
		for nombre, alterado := range map[string]descriptorMaterialConsumidorV3Desarrollo{
			"audiencia": {Audiencia: cronos.Audiencia, Dominio: d.Dominio, Prefijo: d.Prefijo, ProveedorNominal: d.ProveedorNominal},
			"dominio":   {Audiencia: d.Audiencia, Dominio: cronos.Dominio, Prefijo: d.Prefijo, ProveedorNominal: d.ProveedorNominal},
			"prefijo":   {Audiencia: d.Audiencia, Dominio: d.Dominio, Prefijo: cronos.Prefijo, ProveedorNominal: d.ProveedorNominal},
		} {
			b2 := descriptoresMaterialPersonalB2Desarrollo()
			b2[i] = alterado
			if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(append([]descriptorMaterialConsumidorV3Desarrollo(nil), previos...), b2...)); err == nil {
				t.Fatalf("colisión de %s admitida en %s", nombre, d.Audiencia)
			}
		}
	}
	doble := append(descriptoresMaterialPersonalB2Desarrollo(), descriptoresMaterialPersonalB2Desarrollo()[3])
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(doble); err == nil {
		t.Fatal("audiencia B2 duplicada admitida")
	}
}

// Huella de los descriptores previos tomada en f181a2bc: añadir B2 no puede
// alterar ni un byte de lo que ya derivaban CT, Bolsa, Dietas y Cronos.
const huellaDescriptoresPreviosPersonalB2Prueba = "fa714537fd41f5abc4eaad911d6270baf2c0077663eee114a93d58768e17b505"

func TestDescriptoresPreviosIntactosByteAByte(t *testing.T) {
	var b strings.Builder
	for _, d := range descriptoresPreviosPersonalB2Prueba() {
		fmt.Fprintf(&b, "%q|%q|%q|%q\n", d.Audiencia, d.Dominio, d.Prefijo, d.ProveedorNominal)
	}
	suma := sha256.Sum256([]byte(b.String()))
	if got := hex.EncodeToString(suma[:]); got != huellaDescriptoresPreviosPersonalB2Prueba {
		t.Fatalf("descriptores previos alterados: %s", got)
	}
}

// publicadorGobiernoPrueba emula la idempotencia del gobierno: una clave ya
// publicada conserva versión, revisión y orden; una nueva recibe el siguiente.
type publicadorGobiernoPrueba struct {
	versiones map[string]uint64
	siguiente uint64
	llamadas  int
	secretos  [][]byte
	fallarEn  int
}

func (p *publicadorGobiernoPrueba) publicar(m *materialAtestacionContratacionTemporalDesarrollo) error {
	p.llamadas++
	if p.fallarEn > 0 && p.llamadas == p.fallarEn {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if len(m.claveHMAC) == 0 || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(m.audienciaConsumo) {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	p.secretos = append(p.secretos, m.claveHMAC)
	version, ok := p.versiones[m.claveHMACID]
	if !ok {
		p.siguiente++
		version = p.siguiente
		p.versiones[m.claveHMACID] = version
	}
	m.claveHMACVersion, m.claveHMACRevision, m.claveHMACOrden = version, version, version
	return nil
}

func catalogoPersonalB2Prueba(t *testing.T) catalogoMaterialAutorizacionComunDesarrollo {
	t.Helper()
	c, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresPreviosPersonalB2Prueba(), descriptoresMaterialPersonalB2Desarrollo()...))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPublicacionPersonalB2IdempotenteYConservaMaterialPrevio(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
	claveBase := append([]byte(nil), m.claveHMAC...)
	idBase, audienciaBase, huellaBase := m.claveHMACID, m.audienciaConsumo, m.claveHMACHuella
	catalogo := catalogoPersonalB2Prueba(t)
	gobierno := &publicadorGobiernoPrueba{versiones: map[string]uint64{}, siguiente: 40}
	primera, err := publicarMaterialPersonalB2ConDesarrollo(m, catalogo, gobierno.publicar)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := publicarMaterialPersonalB2ConDesarrollo(m, catalogo, gobierno.publicar)
	if err != nil || primera != segunda || gobierno.siguiente != 48 || len(gobierno.versiones) != 8 {
		t.Fatalf("publicación repetida no idempotente: %v", err)
	}
	for _, s := range gobierno.secretos {
		if !bytes.Equal(s, make([]byte, len(s))) {
			t.Fatal("secreto derivado no borrado tras publicar")
		}
	}
	if !bytes.Equal(m.claveHMAC, claveBase) || m.claveHMACID != idBase || m.audienciaConsumo != audienciaBase || m.claveHMACHuella != huellaBase {
		t.Fatal("B2 alteró el material CT de partida")
	}
	vistos := map[string]bool{}
	for i, p := range primera {
		if p.Version != uint64(41+i) || p.OrdenPuntero != p.Version || p.Desde != m.validaDesde || p.Hasta != m.validaHasta || p.EmisorID != m.emisorID {
			t.Fatalf("coordenadas B2 incoherentes: %+v", p)
		}
		for _, v := range []string{p.ClaveID, p.SHA256, p.HuellaGobierno} {
			if vistos[v] || v == "" {
				t.Fatalf("dominios B2 no separados: %s", v)
			}
			vistos[v] = true
		}
		if p.ClaveID != p.Prefijo+strings.TrimPrefix(idBase, "clave:capacidad:ct:") {
			t.Fatalf("identificador B2 no deriva del prefijo: %s", p.ClaveID)
		}
	}
	// La interfaz para la herramienta de composición usa la misma derivación.
	claves, err := DerivarClavesPersonalB2V3Desarrollo(claveBase, idBase, m.emisorID, m.validaDesde, m.validaHasta)
	if err != nil {
		t.Fatal(err)
	}
	for i := range claves {
		c := &claves[i]
		secreto := c.CopiarSecreto()
		suma := sha256.Sum256(secreto)
		if c.ClaveID != primera[i].ClaveID || c.SHA256 != primera[i].SHA256 || c.HuellaGobierno != primera[i].HuellaGobierno || hex.EncodeToString(suma[:]) != c.SHA256 {
			t.Fatalf("derivación de composición divergente en %s", c.Capacidad)
		}
		cadena := hex.EncodeToString(secreto)
		j, _ := json.Marshal(c)
		for _, formato := range []string{fmt.Sprint(*c), fmt.Sprintf("%#v", *c), fmt.Sprintf("%v", c), string(j)} {
			if strings.Contains(formato, cadena) || strings.Contains(strings.ToLower(formato), "secreto") {
				t.Fatalf("formato expone el secreto: %s", formato)
			}
		}
		c.Borrar()
		if c.CopiarSecreto() != nil {
			t.Fatal("Borrar no retiró el secreto")
		}
		borrarBytes(secreto)
	}
}

func TestPublicacionPersonalB2FallaCerrada(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC))
	gobierno := &publicadorGobiernoPrueba{versiones: map[string]uint64{}, fallarEn: 5}
	if p, err := publicarMaterialPersonalB2ConDesarrollo(m, catalogoPersonalB2Prueba(t), gobierno.publicar); !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) || p != ([8]CapacidadPublicadaPersonalB2V3{}) {
		t.Fatal("fallo de publicación no propagado")
	}
	sinB2, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresPreviosPersonalB2Prueba())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publicarMaterialPersonalB2ConDesarrollo(m, sinB2, (&publicadorGobiernoPrueba{versiones: map[string]uint64{}}).publicar); err == nil {
		t.Fatal("publicó B2 sin descriptores en el catálogo")
	}
	if _, err := publicarMaterialPersonalB2ConDesarrollo(m, catalogoPersonalB2Prueba(t), nil); err == nil {
		t.Fatal("publicó sin publicador")
	}
	if _, err := DerivarClavesPersonalB2V3Desarrollo(make([]byte, 16), m.claveHMACID, m.emisorID, m.validaDesde, m.validaHasta); err == nil {
		t.Fatal("derivó desde una clave base corta")
	}
	if _, err := DerivarClavesPersonalB2V3Desarrollo(m.claveHMAC, "clave:capacidad:bolsa:desarrollo:v1", m.emisorID, m.validaDesde, m.validaHasta); err == nil {
		t.Fatal("derivó desde una clave que no es la base CT")
	}
}
