package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/gobiernov3lector"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Pruebas de T1: B2 renueva la configuración publicada con la raíz compartida
// fija y falla cerrado ante revocación o rotación. El gobierno es un doble
// en memoria que imita el contrato de AD3-69; la función SQL real se ensaya
// en PostgreSQL 18.4 con su propio script.

type relojB2 struct {
	mu    sync.Mutex
	ahora time.Time
}

func (r *relojB2) Ahora() time.Time  { r.mu.Lock(); defer r.mu.Unlock(); return r.ahora }
func (r *relojB2) fijar(t time.Time) { r.mu.Lock(); r.ahora = t; r.mu.Unlock() }

type pdpRechazo struct{}

func (pdpRechazo) ExigirSolicitudLigadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errors.New("pdp de prueba")
}

// gobiernoB2Falso reproduce las comprobaciones de AD3-69 que importan aquí:
// lista cerrada de ocho audiencias, raíz compartida fija, puntero vigente,
// configuración anterior no revocada y clave HMAC vigente por audiencia.
type gobiernoB2Falso struct {
	mu         sync.Mutex
	raiz       raizGobiernoV3
	actual     gobiernov3lector.Publicacion
	revocadas  map[string]bool
	vigentes   map[string]string // audiencia -> clave_id vigente
	lecturas   int
	materiales [][]byte
}

func (g *gobiernoB2Falso) validar(material []byte, exigirActual bool) (gobiernov3lector.Publicacion, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.materiales = append(g.materiales, append([]byte(nil), material...))
	var m materialGobiernoV3
	d := json.NewDecoder(bytes.NewReader(material))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || m.Raiz != g.raiz || len(m.Claves) != 8 {
		return gobiernov3lector.Publicacion{}, errGobiernoV3
	}
	for i, c := range capacidadesPersonalB2(MaterialPersonalB2{}) {
		if m.Claves[i].AudienciaConsumo != c.audiencia || g.vigentes[c.audiencia] != m.Claves[i].ClaveID {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
	}
	if g.revocadas[m.Configuracion.Revision] || g.revocadas[g.actual.Revision] ||
		(exigirActual && m.Configuracion.Revision != g.actual.Revision) {
		return gobiernov3lector.Publicacion{}, errGobiernoV3
	}
	return g.actual, nil
}

func (g *gobiernoB2Falso) comprobar(_ context.Context, material []byte) (bool, error) {
	_, err := g.validar(material, true)
	return err == nil, err
}

func (g *gobiernoB2Falso) leer(_ context.Context, material []byte) ([]byte, error) {
	g.mu.Lock()
	g.lecturas++
	g.mu.Unlock()
	p, err := g.validar(material, false)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"revision": p.Revision, "secuencia": p.Secuencia,
		"huella_configuracion_sha256": p.HuellaSHA256, "publicada_en": p.PublicadaEn, "expira_en": p.ExpiraEn})
}

func (g *gobiernoB2Falso) publicar(p gobiernov3lector.Publicacion) {
	g.mu.Lock()
	g.actual = p
	g.mu.Unlock()
}
func (g *gobiernoB2Falso) revocar(ref string)       { g.mu.Lock(); g.revocadas[ref] = true; g.mu.Unlock() }
func (g *gobiernoB2Falso) rotarHMAC(aud, id string) { g.mu.Lock(); g.vigentes[aud] = id; g.mu.Unlock() }
func (g *gobiernoB2Falso) numeroLecturas() int      { g.mu.Lock(); defer g.mu.Unlock(); return g.lecturas }
func (g *gobiernoB2Falso) ultimoMaterial() materialGobiernoV3 {
	g.mu.Lock()
	defer g.mu.Unlock()
	var m materialGobiernoV3
	_ = json.Unmarshal(g.materiales[len(g.materiales)-1], &m)
	return m
}

type escenarioB2 struct {
	dia        time.Time
	reloj      *relojB2
	raiz       confianza.RaizPublicaAtestacionAutorizacionV3
	coord      raizGobiernoV3
	firmante   *firmanteV3
	base       *gobiernoV3Compartido
	gobierno   *gobiernoB2Falso
	pubA, pubB gobiernov3lector.Publicacion
	material   MaterialPersonalB2
}

func publicacionB2(t *testing.T, raiz confianza.RaizPublicaAtestacionAutorizacionV3, dia time.Time, secuencia uint64) gobiernov3lector.Publicacion {
	t.Helper()
	ref := "configuracion:prueba-b2:" + dia.Format("2006-01-02")
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(ref, secuencia, dia, dia.Add(24*time.Hour), raiz)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := config.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	return gobiernov3lector.Publicacion{Revision: ref, Secuencia: secuencia, HuellaSHA256: huella, PublicadaEn: dia, ExpiraEn: dia.Add(24 * time.Hour)}
}

func raizB2(t *testing.T, semilla byte, desde time.Time) (confianza.RaizPublicaAtestacionAutorizacionV3, ed25519.PrivateKey) {
	t.Helper()
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{semilla}, ed25519.SeedSize))
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:atestacion:prueba-b2:v3", 1,
		clave.Public().(ed25519.PublicKey), audienciaAtestacionCTInterna, confianza.EstadoClaveAtestacionAutorizacionV3Activa,
		desde.Add(-time.Hour), desde.Add(96*time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return raiz, clave
}

// materialB2Prueba escribe ocho claves HMAC sintéticas y el inventario v4.
func materialB2Prueba(t *testing.T, desde time.Time) MaterialPersonalB2 {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	referencia := map[string]any{"catalogo_id": "motivos.b2", "catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("a", 64), "entrada_clave": "motivo_0123456789abcdef0123456789abcdef"}
	motivos := map[string]any{}
	capacidades := map[string]any{}
	for i, nombre := range []string{"ficha", "vacantes", "alta", "hecho", "catalogo_consultar", "catalogo_publicar", "catalogo_retirar", "empleados"} {
		secreto := bytes.Repeat([]byte{byte(0x41 + i)}, 32)
		if err := os.WriteFile(filepath.Join(dir, nombre+".key"), secreto, 0600); err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(secreto)
		motivos[nombre] = referencia
		capacidades[nombre] = map[string]any{"clave_id": "clave:capacidad:personal-b2-" + nombre + ":1", "version": 1,
			"archivo": nombre + ".key", "sha256": hex.EncodeToString(suma[:]), "emisor_id": "emisor:personal-b2:prueba",
			"desde": desde.Add(-time.Hour), "hasta": desde.Add(96 * time.Hour), "revision_gobierno": 1,
			"huella_gobierno": strings.Repeat("b", 64)}
	}
	b, err := json.Marshal(map[string]any{"version": versionMaterialPersonalB2, "catalogo_motivos": "motivos.b2",
		"motivos": motivos, "v3": map[string]any{"capacidades": capacidades}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "personal_b2_v3.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	m, err := CargarMaterialPersonalB2(dir)
	if err != nil {
		t.Fatalf("inventario v4 sintético rechazado: %v", err)
	}
	t.Cleanup(func() { _ = m.Cerrar() })
	return m
}

func nuevoEscenarioB2(t *testing.T) *escenarioB2 {
	t.Helper()
	dia := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	raiz, clave := raizB2(t, 7, dia)
	coord, err := coordenadasRaizV3("clave:atestacion:prueba-b2:v3", 1, audienciaAtestacionCTInterna, clave.Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	e := &escenarioB2{dia: dia, reloj: &relojB2{ahora: dia.Add(time.Hour)}, raiz: raiz, coord: coord,
		pubA: publicacionB2(t, raiz, dia, 20260925), pubB: publicacionB2(t, raiz, dia.Add(24*time.Hour), 20260926)}
	e.firmante = &firmanteV3{claveID: coord.ClaveID, audiencia: coord.AudienciaDespliegue, privada: append(ed25519.PrivateKey(nil), clave...), reloj: e.reloj}
	e.gobierno = &gobiernoB2Falso{raiz: coord, actual: e.pubA, revocadas: map[string]bool{}, vigentes: map[string]string{}}
	e.material = materialB2Prueba(t, dia)
	for _, c := range capacidadesPersonalB2(e.material) {
		e.gobierno.vigentes[c.audiencia] = c.material.ClaveID
	}
	e.base = e.baseCT(t, e.pubA)
	return e
}

// baseCT simula el lector CT ya construido: misma raíz, gobierno vivo.
func (e *escenarioB2) baseCT(t *testing.T, anterior gobiernov3lector.Publicacion) *gobiernoV3Compartido {
	t.Helper()
	lector, err := gobiernov3lector.Nuevo(anterior, e.raiz, e.reloj, func(context.Context, gobiernov3lector.Publicacion) (gobiernov3lector.Publicacion, error) {
		e.gobierno.mu.Lock()
		defer e.gobierno.mu.Unlock()
		if e.gobierno.revocadas[e.gobierno.actual.Revision] {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
		return e.gobierno.actual, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return &gobiernoV3Compartido{raiz: e.raiz, coord: e.coord, lector: lector}
}

func (e *escenarioB2) dependencias() dependenciasPersonalB2 {
	return dependenciasPersonalB2{pdp: pdpRechazo{}, catalogo: "motivos.b2", gobierno: e.base, firmante: e.firmante,
		comprobar: e.gobierno.comprobar, leer: e.gobierno.leer, fuente: &internagobierno.FuenteF1{}, reloj: e.reloj}
}

func (e *escenarioB2) construir(t *testing.T) *ProveedorAutorizacionPersonalB2 {
	t.Helper()
	p, err := construirPersonalB2(context.Background(), e.material, e.dependencias())
	if err != nil {
		t.Fatalf("B2 con raíz compartida rechazado: %v", err)
	}
	t.Cleanup(p.Cerrar)
	return p
}

// emitir distingue la denegación del gobierno (ErrProveedoresCTNoDisponibles,
// sin llegar al PDP) de un fallo posterior de la cadena nominal.
func emitir(p *ProveedorAutorizacionPersonalB2, i int) error {
	_, _, _, err := p.emisores[i].EmitirMaterialAutorizacionAtestadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{})
	return err
}

func gobiernoDenego(err error) bool { return errors.Is(err, ErrProveedoresCTNoDisponibles) }

func TestPersonalB2UsaRaizCompartidaYOchoAudienciasEnOrden(t *testing.T) {
	e := nuevoEscenarioB2(t)
	p := e.construir(t)
	if p.firmante.base != e.firmante || p.firmante.base.audiencia != audienciaAtestacionCTInterna || p.firmante.base.claveID != e.coord.ClaveID {
		t.Fatal("B2 no firma con la raíz y audiencia compartidas")
	}
	m := e.gobierno.ultimoMaterial()
	if m.Raiz != e.coord || m.Raiz.AudienciaDespliegue != "vec:desarrollo:contratacion-temporal:atestacion:v3" {
		t.Fatalf("raíz enviada distinta de la compartida: %+v", m.Raiz)
	}
	esperadas := []string{personal.AudienciaFichaEmpleadoB2, personal.AudienciaVacantesB2, personal.AudienciaAltaEmpleadoB2,
		personal.AudienciaHechoEmpleadoB2, personal.AudienciaConsultarCatalogoEmpleadoB2, personal.AudienciaPublicarCatalogoEmpleadoB2,
		personal.AudienciaRetirarCatalogoEmpleadoB2, personal.AudienciaEmpleadosB2}
	if len(m.Claves) != len(esperadas) {
		t.Fatalf("claves B2 enviadas: %d", len(m.Claves))
	}
	for i, a := range esperadas {
		if m.Claves[i].AudienciaConsumo != a {
			t.Fatalf("orden de audiencias B2 alterado en %d: %s", i, m.Claves[i].AudienciaConsumo)
		}
	}
	for _, sql := range []string{sqlComprobarMaterialB2, sqlLeerConfiguracionB2} {
		if !strings.Contains(sql, "_interna_v2('personal_b2',$1::jsonb)") {
			t.Fatalf("B2 no usa el consumidor cerrado v2: %s", sql)
		}
	}
	if !strings.Contains(sqlLeerConfiguracionCT, "leer_configuracion_interna_v1($1::jsonb)") {
		t.Fatal("CT dejó de usar la lectura v1")
	}
}

func TestPersonalB2RenuevaConfiguracionConRaizConservada(t *testing.T) {
	e := nuevoEscenarioB2(t)
	p := e.construir(t)
	antes := e.gobierno.numeroLecturas()
	if err := emitir(p, 0); gobiernoDenego(err) {
		t.Fatalf("emisión con configuración vigente denegada por gobierno: %v", err)
	}
	if e.gobierno.numeroLecturas() != antes+1 {
		t.Fatal("la emisión no tomó instantánea del gobierno")
	}
	// Medianoche UTC: vec-server publica la configuración del día siguiente
	// con la misma raíz. B2 la adopta sin reinicio ni material nuevo.
	e.gobierno.publicar(e.pubB)
	e.reloj.fijar(e.pubB.PublicadaEn.Add(time.Minute))
	if err := emitir(p, 7); gobiernoDenego(err) {
		t.Fatalf("renovación diaria rechazada: %v", err)
	}
	if m := e.gobierno.ultimoMaterial(); m.Configuracion.Revision != e.pubA.Revision || m.Raiz != e.coord {
		t.Fatalf("la renovación no partió de la publicación anterior con la raíz fija: %+v", m.Configuracion)
	}
	if err := emitir(p, 3); gobiernoDenego(err) {
		t.Fatalf("segunda emisión tras renovar: %v", err)
	}
	if m := e.gobierno.ultimoMaterial(); m.Configuracion.Revision != e.pubB.Revision {
		t.Fatalf("el lector no avanzó a la configuración renovada: %s", m.Configuracion.Revision)
	}
	// Retroceso a la publicación anterior: denegado.
	e.gobierno.publicar(e.pubA)
	if err := emitir(p, 0); !gobiernoDenego(err) {
		t.Fatalf("retroceso de configuración admitido: %v", err)
	}
}

func TestPersonalB2InventarioPrevioSigueValidoTrasRenovar(t *testing.T) {
	e := nuevoEscenarioB2(t)
	// El inventario CT y el B2 se compusieron el día A; B2 arranca el día B,
	// ya renovado. El inventario B2 no fija configuración: no caduca.
	e.gobierno.publicar(e.pubB)
	e.reloj.fijar(e.pubB.PublicadaEn.Add(2 * time.Hour))
	e.base = e.baseCT(t, e.pubA)
	p := e.construir(t)
	if err := emitir(p, 1); gobiernoDenego(err) {
		t.Fatalf("inventario previo rechazado tras renovación: %v", err)
	}
	if m := e.gobierno.ultimoMaterial(); m.Configuracion.Revision != e.pubB.Revision {
		t.Fatalf("B2 no partió de la publicación vigente: %s", m.Configuracion.Revision)
	}
}

func TestPersonalB2RevocacionCortaLaSiguienteEmision(t *testing.T) {
	e := nuevoEscenarioB2(t)
	p := e.construir(t)
	if err := emitir(p, 0); gobiernoDenego(err) {
		t.Fatalf("emisión previa: %v", err)
	}
	// Revocación antes del vencimiento: la siguiente emisión ya no sale.
	e.gobierno.revocar(e.pubA.Revision)
	for i := range p.emisores {
		if err := emitir(p, i); !gobiernoDenego(err) {
			t.Fatalf("capacidad %d emitida tras revocar la configuración: %v", i, err)
		}
	}
}

func TestPersonalB2RotacionIncompatibleSeDeniega(t *testing.T) {
	t.Run("raiz", func(t *testing.T) {
		e := nuevoEscenarioB2(t)
		p := e.construir(t)
		otra, _ := raizB2(t, 9, e.dia)
		// Configuración nueva ligada a otra raíz: su huella no cuadra con la fija.
		e.gobierno.publicar(publicacionB2(t, otra, e.dia.Add(24*time.Hour), 20260926))
		e.reloj.fijar(e.pubB.PublicadaEn.Add(time.Minute))
		if err := emitir(p, 0); !gobiernoDenego(err) {
			t.Fatalf("rotación de raíz adoptada automáticamente: %v", err)
		}
	})
	t.Run("hmac", func(t *testing.T) {
		e := nuevoEscenarioB2(t)
		p := e.construir(t)
		e.gobierno.rotarHMAC(personal.AudienciaAltaEmpleadoB2, "clave:capacidad:personal-b2-alta:2")
		if err := emitir(p, 2); !gobiernoDenego(err) {
			t.Fatalf("rotación HMAC admitida sin material nuevo: %v", err)
		}
	})
	t.Run("arranque", func(t *testing.T) {
		e := nuevoEscenarioB2(t)
		e.gobierno.rotarHMAC(personal.AudienciaEmpleadosB2, "clave:capacidad:personal-b2-empleados:2")
		if _, err := construirPersonalB2(context.Background(), e.material, e.dependencias()); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
			t.Fatalf("material HMAC superado arrancó: %v", err)
		}
	})
}

func TestPersonalB2AusenteNoAfectaCT(t *testing.T) {
	e := nuevoEscenarioB2(t)
	privada := append([]byte(nil), e.firmante.privada...)
	for nombre, cambiar := range map[string]func(*dependenciasPersonalB2){
		"sonda": func(d *dependenciasPersonalB2) {
			d.comprobar = func(context.Context, []byte) (bool, error) { return false, nil }
		},
		"catalogo": func(d *dependenciasPersonalB2) { d.catalogo = "otro.catalogo" },
		"sin_raiz": func(d *dependenciasPersonalB2) { d.gobierno = nil },
		"audiencia": func(d *dependenciasPersonalB2) {
			g := *d.gobierno
			g.coord.AudienciaDespliegue = "vec:interno:personal:registro-empleado:atestacion:v3"
			d.gobierno = &g
		},
	} {
		d := e.dependencias()
		cambiar(&d)
		if _, err := construirPersonalB2(context.Background(), e.material, d); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
			t.Fatalf("%s: B2 arrancó: %v", nombre, err)
		}
	}
	p := e.construir(t)
	p.Cerrar()
	if !bytes.Equal(e.firmante.privada, privada) {
		t.Fatal("cerrar B2 borró la clave del firmante CT")
	}
	if _, err := e.base.lector.Instantanea(context.Background()); err != nil {
		t.Fatalf("CT perdió su gobierno por B2: %v", err)
	}
	if p.firmante != nil || p.emisores != [8]emisorMaterialV3{} {
		t.Fatal("B2 cerrado conserva firmante o emisores")
	}
}

func TestPersonalB2RechazaInventarioConRaizPropia(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b := []byte(`{"version":4,"catalogo_motivos":"motivos.b2","motivos":{},"v3":{"audiencia":"vec:interno:personal:registro-empleado:atestacion:v3","clave_archivo":"raiz.key","capacidades":{}}}`)
	if err := os.WriteFile(filepath.Join(dir, "personal_b2_v3.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CargarMaterialPersonalB2(dir); !errors.Is(err, ErrPersonalB2V3NoDisponible) {
		t.Fatalf("inventario con raíz B2 propia admitido: %v", err)
	}
}
