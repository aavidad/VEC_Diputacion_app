package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	altapersonal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lecturapersonal "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

// Ensayo PostgreSQL 18.4 de la corrección de escalas de AD3-69 (consenso B2
// R3–R5), dirigido por cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh.
// Publica con el publicador real de vec-server las cinco claves que lee CT y
// las ocho de B2, SIN tocar checkpoint_gobierno, y exige que
// comprobar/leer_*_interna_v2('ct'|'personal_b2') las acepten de inmediato con
// el LOGIN nominal de preflight. v1 (intacta) debe seguir rechazando CT: es el
// fallo que v2 corrige. Fase «publicar»: renovación de configuración (ayer →
// hoy), repetición, concurrencia, rollback y salto de versión. Fase «reinicio»
// (tras reiniciar PostgreSQL): republicación idempotente y negativos que
// alteran el gobierno (mínimos de configuración y raíz retrocedidos, clave
// revocada, clave sustituida). Solo afectan a CT; B2 queda intacto para T4.
//
// alta_ejercicio, lectura_incorporacion y confirmación de incorporación no
// tienen descriptor en vec-server (su material llega ya gobernado): el ensayo
// los deriva con descriptores propios de prueba y los publica con el mismo
// publicador. Cuadro y detalle usan los descriptores reales.
func TestGobiernoV2AceptaClavesRecienPublicadasSinCheckpointPostgreSQL18(t *testing.T) {
	fase := os.Getenv("VEC_AD369_FASE")
	if os.Getenv("VEC_T4_PG_DESECHABLE") != "si" || (fase != "publicar" && fase != "reinicio") {
		t.Skip("requiere PostgreSQL 18.4 desechable (cmd/vec-preparar-material-interno/probar_postgresql_pg18.sh)")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancelar()
	e := nuevoEnsayoEscalas(t, ctx)
	defer e.cerrar()

	if fase == "publicar" {
		ayer := e.publicarTodo(t, time.Now().Add(-24*time.Hour))
		checkpointAyer := e.checkpoint(t)
		hoy := e.publicarTodo(t, time.Now())
		e.exigirSinDesfaseResuelto(t)
		if hoy.config == ayer.config || hoy.config.Secuencia <= ayer.config.Secuencia {
			t.Fatal("la renovación no publicó una configuración nueva")
		}
		for i := range hoy.ct {
			if hoy.ct[i] != ayer.ct[i] {
				t.Fatal("la renovación de configuración cambió una clave CT")
			}
		}
		if e.checkpoint(t).Revision <= checkpointAyer.Revision {
			t.Fatal("el trigger real no avanzó el checkpoint al renovar")
		}
		e.exigirAcepta(t, hoy)
		// Renovación vista desde el material de ayer: la lectura devuelve la
		// configuración de hoy; la sonda rechaza la ya sustituida.
		for _, consumidor := range []string{"ct", "personal_b2"} {
			if r := e.leer(t, consumidor, ayer.json(t, consumidor)); r.Revision != hoy.config.Revision {
				t.Fatalf("%s: lectura desde la configuración previa: %+v", consumidor, r)
			}
			e.exigirRechazoSonda(t, consumidor, ayer.json(t, consumidor), "configuración sustituida")
		}
		// v1 intacta sigue comparando escalas: rechaza CT recién publicado.
		e.exigirRechazoV1(t, hoy.json(t, "ct"))

		huella := e.huellaGobierno(t)
		otra := e.publicarTodo(t, time.Now())
		if otra.config != hoy.config || otra.ct != hoy.ct || otra.b2 != hoy.b2 || e.huellaGobierno(t) != huella {
			t.Fatal("publicación repetida no idempotente")
		}
		e.concurrencia(t, hoy)
		if e.huellaGobierno(t) != huella {
			t.Fatal("la concurrencia añadió filas")
		}
		e.rollback(t, hoy, huella)
		e.saltoDeVersion(t, hoy)
		e.negativosSinEscritura(t, hoy)
		e.escribirMaterialCT(t, hoy)
		return
	}

	// Reinicio: vec-server vuelve a arrancar y republica sin cambios.
	huella := e.huellaGobierno(t)
	hoy := e.publicarTodo(t, time.Now())
	if e.huellaGobierno(t) != huella {
		t.Fatal("la republicación tras el reinicio añadió filas")
	}
	e.exigirSinDesfaseResuelto(t)
	e.exigirAcepta(t, hoy)
	e.exigirRechazoV1(t, hoy.json(t, "ct"))
	e.negativosConEscritura(t, hoy)
}

type coordenadaClave struct {
	Audiencia        string `json:"audiencia_consumo"`
	ClaveID          string `json:"clave_id"`
	Version          uint64 `json:"version"`
	RevisionGobierno uint64 `json:"revision_gobierno"`
	HuellaGobierno   string `json:"huella_gobierno_sha256"`
	HuellaSecreto    string `json:"huella_secreto_sha256"`
	EmisorID         string `json:"emisor_id"`
}

type coordenadaConfiguracion struct {
	Revision  string `json:"revision"`
	Secuencia uint64 `json:"secuencia"`
	Huella    string `json:"huella_configuracion_sha256"`
}

type coordenadaRaiz struct {
	ClaveID   string `json:"clave_id"`
	Version   uint64 `json:"version"`
	Huella    string `json:"huella_spki_sha256"`
	Audiencia string `json:"audiencia_despliegue"`
	Suite     string `json:"suite"`
}

type publicacionEscalas struct {
	ct     [5]coordenadaClave
	b2     [8]coordenadaClave
	config coordenadaConfiguracion
	raiz   coordenadaRaiz
}

func (p publicacionEscalas) json(t *testing.T, consumidor string) []byte {
	t.Helper()
	claves := p.ct[:]
	if consumidor == "personal_b2" {
		claves = p.b2[:]
	}
	b, err := json.Marshal(struct {
		Claves        []coordenadaClave       `json:"claves"`
		Configuracion coordenadaConfiguracion `json:"configuracion"`
		Raiz          coordenadaRaiz          `json:"raiz"`
	}{claves, p.config, p.raiz})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type lecturaConfiguracion struct {
	Revision  string `json:"revision"`
	Secuencia uint64 `json:"secuencia"`
}

type filaCheckpoint struct {
	Revision, SecuenciaMinima, RaizMinima int64
}

type ensayoEscalas struct {
	gobierno, preflight *pgxpool.Pool
	admin               *pgx.Conn
	directorio          string
}

func nuevoEnsayoEscalas(t *testing.T, ctx context.Context) *ensayoEscalas {
	t.Helper()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, os.Getenv("VEC_T4_PG_GOBIERNO_DSN"),
		"vec-ct-desarrollo-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		t.Fatalf("pool de gobierno de vec-server rechazado: %v", err)
	}
	preflight, err := pgxpool.New(ctx, os.Getenv("VEC_T4_PG_PREFLIGHT_DSN"))
	if err != nil {
		gobierno.Close()
		t.Fatal("pool de preflight no disponible")
	}
	admin, err := pgx.Connect(ctx, os.Getenv("VEC_T4_PG_ADMIN_DSN"))
	if err != nil {
		gobierno.Close()
		preflight.Close()
		t.Fatal("DBA de prueba no disponible")
	}
	return &ensayoEscalas{gobierno: gobierno, preflight: preflight, admin: admin, directorio: os.Getenv("VEC_T4_MATERIAL_IDEMPOTENCIA")}
}

func (e *ensayoEscalas) cerrar() {
	e.gobierno.Close()
	e.preflight.Close()
	_ = e.admin.Close(context.Background())
}

// descriptoresCTEscalas sigue el orden cerrado de v2('ct').
func descriptoresCTEscalas() [5]descriptorMaterialConsumidorV3Desarrollo {
	reales := descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()
	prueba := func(audiencia, nombre string) descriptorMaterialConsumidorV3Desarrollo {
		return descriptorMaterialConsumidorV3Desarrollo{Audiencia: audiencia, Dominio: "vec.ct.prueba-ad369." + nombre + ".capacidad-v3",
			Prefijo: "clave:capacidad:ct-prueba-ad369-" + nombre + ":", ProveedorNominal: proveedorMaterialContratacionTemporal}
	}
	return [5]descriptorMaterialConsumidorV3Desarrollo{
		prueba(altapersonal.AudienciaAltaEjercicio, "alta"),
		prueba(lecturapersonal.AudienciaV2, "lectura"),
		prueba(ctports.AudienciaConfirmacionIncorporacionV2, "incorporacion"),
		reales[0], reales[1],
	}
}

func (e *ensayoEscalas) material(t *testing.T, ahora time.Time) materialAtestacionContratacionTemporalDesarrollo {
	t.Helper()
	raiz := filepath.Dir(e.directorio)
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(raiz, filepath.Join(raiz, "idempotencia", "configuracion.json"))
	if err != nil {
		t.Fatal("material de idempotencia sintético rechazado")
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	if err != nil {
		t.Fatal(err)
	}
	defer derivador.borrar()
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func coordenada(d *materialAtestacionContratacionTemporalDesarrollo) coordenadaClave {
	return coordenadaClave{d.audienciaConsumo, d.claveHMACID, d.claveHMACVersion, d.claveHMACRevision,
		d.claveHMACHuella, d.claveHMACSecreto, d.emisorID}
}

// publicarTodo publica como vec-server: clave base CT, las cinco de CT (una
// transacción del publicador por clave) y las ocho de B2.
func (e *ensayoEscalas) publicarTodo(t *testing.T, ahora time.Time) publicacionEscalas {
	t.Helper()
	ctx := context.Background()
	m := e.material(t, ahora)
	defer m.borrarCopiasEfimeras()
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, e.gobierno, &m); err != nil {
		t.Fatalf("publicación de la clave base: %v", err)
	}
	var p publicacionEscalas
	for i, d := range descriptoresCTEscalas() {
		derivado, err := derivarMaterialConsumidorV3Desarrollo(m, d)
		if err != nil {
			t.Fatal(err)
		}
		err = publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, e.gobierno, &derivado)
		borrarBytes(derivado.claveHMAC)
		if err != nil {
			t.Fatalf("publicación CT %s: %v", d.Audiencia, err)
		}
		p.ct[i] = coordenada(&derivado)
		p.config = coordenadaConfiguracion{derivado.configuracionRef, derivado.configuracionOrden, derivado.configuracionHuella}
		p.raiz = coordenadaRaiz{derivado.claveID, derivado.claveVersion, derivado.spkiHuella,
			audienciaAtestacionContratacionTemporalDesarrollo, confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA}
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresMaterialAutorizacionContratacionTemporalDesarrollo(), descriptoresMaterialPersonalB2Desarrollo()...))
	if err != nil {
		t.Fatal(err)
	}
	publicadas, err := publicarMaterialPersonalB2Desarrollo(ctx, e.gobierno, m, catalogo)
	if err != nil {
		t.Fatalf("publicación B2: %v", err)
	}
	for i, c := range publicadas {
		p.b2[i] = coordenadaClave{c.Audiencia, c.ClaveID, c.Version, c.RevisionGobierno, c.HuellaGobierno, c.SHA256, c.EmisorID}
	}
	return p
}

func (e *ensayoEscalas) checkpoint(t *testing.T) filaCheckpoint {
	t.Helper()
	var c filaCheckpoint
	if err := e.admin.QueryRow(context.Background(), `SELECT revision::bigint, configuracion_secuencia_minima::bigint,
	    raiz_version_minima::bigint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id`).
		Scan(&c.Revision, &c.SecuenciaMinima, &c.RaizMinima); err != nil {
		t.Fatal(err)
	}
	return c
}

// exigirSinDesfaseResuelto comprueba que el ensayo reproduce el caso real: el
// contador del checkpoint queda por debajo de las revisiones publicadas.
func (e *ensayoEscalas) exigirSinDesfaseResuelto(t *testing.T) {
	t.Helper()
	var maxima int64
	if err := e.admin.QueryRow(context.Background(), `SELECT max(revision_gobierno)::bigint
	    FROM vec_autorizacion_atestada_v3.clave_capacidad_version`).Scan(&maxima); err != nil {
		t.Fatal(err)
	}
	if c := e.checkpoint(t); c.Revision >= maxima {
		t.Fatalf("el checkpoint (%d) ya alcanza las claves (%d): no reproduce el desfase", c.Revision, maxima)
	}
}

func (e *ensayoEscalas) huellaGobierno(t *testing.T) string {
	t.Helper()
	var s string
	if err := e.admin.QueryRow(context.Background(), `SELECT concat_ws(',',
	  (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
	  (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
	  (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_confianza_version),
	  (SELECT count(*) FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual),
	  (SELECT count(*) FROM vec_autorizacion_atestada_v3.raiz_confianza_version),
	  (SELECT revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno))`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (e *ensayoEscalas) sonda(consumidor string, material []byte) (bool, error) {
	var ok bool
	err := e.preflight.QueryRow(context.Background(),
		`SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2($1,$2::jsonb)`, consumidor, material).Scan(&ok)
	return ok, err
}

func (e *ensayoEscalas) lectura(consumidor string, material []byte) (lecturaConfiguracion, error) {
	var r lecturaConfiguracion
	var b []byte
	err := e.preflight.QueryRow(context.Background(),
		`SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v2($1,$2::jsonb)`, consumidor, material).Scan(&b)
	if err == nil {
		err = json.Unmarshal(b, &r)
	}
	return r, err
}

func (e *ensayoEscalas) leer(t *testing.T, consumidor string, material []byte) lecturaConfiguracion {
	t.Helper()
	r, err := e.lectura(consumidor, material)
	if err != nil {
		t.Fatalf("%s: lectura v2 rechazada: %v", consumidor, err)
	}
	return r
}

func (e *ensayoEscalas) exigirAcepta(t *testing.T, p publicacionEscalas) {
	t.Helper()
	for _, consumidor := range []string{"ct", "personal_b2"} {
		if ok, err := e.sonda(consumidor, p.json(t, consumidor)); err != nil || !ok {
			t.Fatalf("%s: sonda v2 rechazó claves recién publicadas: %v", consumidor, err)
		}
		if r := e.leer(t, consumidor, p.json(t, consumidor)); r.Revision != p.config.Revision || r.Secuencia != p.config.Secuencia {
			t.Fatalf("%s: lectura v2 %+v", consumidor, r)
		}
	}
}

func rechazo42501(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "42501"
}

func (e *ensayoEscalas) exigirRechazoSonda(t *testing.T, consumidor string, material []byte, caso string) {
	t.Helper()
	if _, err := e.sonda(consumidor, material); !rechazo42501(err) {
		t.Fatalf("%s: sonda v2 no rechazó con 42501 (%s): %v", caso, consumidor, err)
	}
}

func (e *ensayoEscalas) exigirRechazo(t *testing.T, consumidor string, material []byte, caso string) {
	t.Helper()
	e.exigirRechazoSonda(t, consumidor, material, caso)
	if _, err := e.lectura(consumidor, material); !rechazo42501(err) {
		t.Fatalf("%s: lectura v2 no rechazó con 42501 (%s): %v", caso, consumidor, err)
	}
}

func (e *ensayoEscalas) exigirRechazoV1(t *testing.T, material []byte) {
	t.Helper()
	for _, sql := range []string{
		`SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1($1::jsonb)::text`,
		`SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v1($1::jsonb)::text`,
	} {
		var s string
		if err := e.preflight.QueryRow(context.Background(), sql, material).Scan(&s); !rechazo42501(err) {
			t.Fatalf("v1 aceptó CT con el checkpoint por debajo (comparación entre escalas intacta esperada): %v", err)
		}
	}
}

// concurrencia: cuatro arranques de vec-server republican a la vez mientras
// el preflight lee; el publicador serializa y nadie ve un estado mezclado.
func (e *ensayoEscalas) concurrencia(t *testing.T, p publicacionEscalas) {
	t.Helper()
	var wg sync.WaitGroup
	fallos := make(chan string, 16)
	materiales := make([]materialAtestacionContratacionTemporalDesarrollo, 4)
	for i := range materiales {
		materiales[i] = e.material(t, time.Now())
	}
	defer func() {
		for i := range materiales {
			materiales[i].borrarCopiasEfimeras()
		}
	}()
	jsonCT, jsonB2 := p.json(t, "ct"), p.json(t, "personal_b2")
	for i := range materiales {
		wg.Add(2)
		go func(m materialAtestacionContratacionTemporalDesarrollo) {
			defer wg.Done()
			ctx := context.Background()
			for _, d := range descriptoresCTEscalas() {
				derivado, err := derivarMaterialConsumidorV3Desarrollo(m, d)
				if err != nil {
					fallos <- "derivación"
					return
				}
				err = publicarGobiernoAtestacionContratacionTemporalDesarrollo(ctx, e.gobierno, &derivado)
				borrarBytes(derivado.claveHMAC)
				if err != nil || coordenada(&derivado) == (coordenadaClave{}) {
					fallos <- "publicación concurrente: " + d.Audiencia
					return
				}
			}
		}(materiales[i])
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				for consumidor, material := range map[string][]byte{"ct": jsonCT, "personal_b2": jsonB2} {
					if ok, err := e.sonda(consumidor, material); err != nil || !ok {
						fallos <- "sonda concurrente " + consumidor
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(fallos)
	for f := range fallos {
		t.Fatal(f)
	}
	e.exigirAcepta(t, p)
}

// rollback: una publicación nueva (otra audiencia) revertida no deja filas ni
// mueve el checkpoint; CT y B2 siguen aceptados.
func (e *ensayoEscalas) rollback(t *testing.T, p publicacionEscalas, huella string) {
	t.Helper()
	m := e.material(t, time.Now())
	defer m.borrarCopiasEfimeras()
	d := descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()[2]
	derivado, err := derivarMaterialConsumidorV3Desarrollo(m, d)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(derivado.claveHMAC)
	provocado := errors.New("rollback provocado")
	err = ejecutarTransaccionGobiernoCTDesarrollo(context.Background(), e.gobierno, func(tx pgx.Tx) error {
		if err := publicarGobiernoAtestacionCTEnTxDesarrollo(context.Background(), tx, &derivado); err != nil {
			return err
		}
		return provocado
	})
	if err == nil {
		t.Fatal("la transacción provocada terminó sin error")
	}
	if e.huellaGobierno(t) != huella {
		t.Fatal("el rollback dejó rastro en el gobierno")
	}
	var n int
	if err := e.admin.QueryRow(context.Background(), `SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version
	    WHERE audiencia_consumo=$1`, ctapplication.AudienciaDespachoCorreoLlamamientoV3).Scan(&n); err != nil || n != 0 {
		t.Fatal("clave revertida presente")
	}
	e.exigirAcepta(t, p)
}

// saltoDeVersion: una clave nueva de otra audiencia toma versión y revisión
// max+1, muy por encima del contador; las claves CT/B2 vigentes no cambian y
// siguen aceptadas, y la nueva queda por encima del checkpoint.
func (e *ensayoEscalas) saltoDeVersion(t *testing.T, p publicacionEscalas) {
	t.Helper()
	m := e.material(t, time.Now())
	defer m.borrarCopiasEfimeras()
	derivado, err := derivarMaterialConsumidorV3Desarrollo(m, descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()[2])
	if err != nil {
		t.Fatal(err)
	}
	err = publicarGobiernoAtestacionContratacionTemporalDesarrollo(context.Background(), e.gobierno, &derivado)
	borrarBytes(derivado.claveHMAC)
	if err != nil {
		t.Fatal(err)
	}
	maxima := uint64(0)
	for _, c := range append(p.ct[:], p.b2[:]...) {
		maxima = max(maxima, c.RevisionGobierno)
	}
	if derivado.claveHMACRevision <= maxima || int64(derivado.claveHMACRevision) <= e.checkpoint(t).Revision {
		t.Fatalf("sin salto: revisión %d, máxima previa %d", derivado.claveHMACRevision, maxima)
	}
	e.exigirSinDesfaseResuelto(t)
	e.exigirAcepta(t, p)
}

func alterar(t *testing.T, p publicacionEscalas, consumidor string, cambiar func(*publicacionEscalas)) []byte {
	t.Helper()
	copia := p
	cambiar(&copia)
	return copia.json(t, consumidor)
}

// negativosSinEscritura: revisión o huella falsificadas, consumidor cruzado,
// LOGIN incorrecto y ausencia de acceso directo a tablas y secretos.
func (e *ensayoEscalas) negativosSinEscritura(t *testing.T, p publicacionEscalas) {
	t.Helper()
	e.exigirRechazo(t, "ct", alterar(t, p, "ct", func(q *publicacionEscalas) { q.ct[3].RevisionGobierno++ }), "revisión falsificada")
	e.exigirRechazo(t, "personal_b2", alterar(t, p, "personal_b2", func(q *publicacionEscalas) { q.b2[4].RevisionGobierno-- }), "revisión falsificada")
	e.exigirRechazo(t, "ct", alterar(t, p, "ct", func(q *publicacionEscalas) { q.ct[0].HuellaGobierno = strings.Repeat("d", 64) }), "huella de gobierno falsa")
	e.exigirRechazo(t, "ct", alterar(t, p, "ct", func(q *publicacionEscalas) { q.config.Secuencia-- }), "secuencia de configuración falsa")
	e.exigirRechazo(t, "ct", alterar(t, p, "ct", func(q *publicacionEscalas) { q.raiz.Version++ }), "raíz inexistente")
	e.exigirRechazo(t, "personal_b2", p.json(t, "ct"), "material CT como B2")
	e.exigirRechazo(t, "ct", p.json(t, "personal_b2"), "material B2 como CT")
	ctx := context.Background()
	for _, sql := range []string{
		`SELECT secreto_hmac FROM vec_autorizacion_atestada_v3.clave_capacidad_version LIMIT 1`,
		`SELECT revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno`,
	} {
		if _, err := e.preflight.Exec(ctx, sql); err == nil {
			t.Fatalf("preflight con acceso directo: %s", sql)
		}
	}
	// Otros LOGIN: el de gobierno no ejecuta v2; el superusuario queda
	// rechazado por la comprobación de LOGIN nominal.
	for _, con := range []interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}{e.gobierno, e.admin} {
		var ok bool
		err := con.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct',$1::jsonb)`, p.json(t, "ct")).Scan(&ok)
		if !rechazo42501(err) {
			t.Fatalf("LOGIN incorrecto aceptado: %v", err)
		}
	}
}

// negativosConEscritura alteran el gobierno de forma persistente y solo
// afectan a CT; los mínimos se restituyen al terminar cada caso.
func (e *ensayoEscalas) negativosConEscritura(t *testing.T, p publicacionEscalas) {
	t.Helper()
	ctx := context.Background()
	c := e.checkpoint(t)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := e.admin.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=$1 WHERE control_id`, c.SecuenciaMinima+1)
	e.exigirRechazo(t, "ct", p.json(t, "ct"), "configuración retrocedida")
	e.exigirRechazo(t, "personal_b2", p.json(t, "personal_b2"), "configuración retrocedida")
	exec(`UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET configuracion_secuencia_minima=$1, raiz_version_minima=$2 WHERE control_id`,
		c.SecuenciaMinima, c.RaizMinima+1)
	e.exigirRechazo(t, "ct", p.json(t, "ct"), "raíz retrocedida")
	e.exigirRechazo(t, "personal_b2", p.json(t, "personal_b2"), "raíz retrocedida")
	exec(`UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET raiz_version_minima=$1 WHERE control_id`, c.RaizMinima)
	e.exigirAcepta(t, p)

	// Clave sustituida: nuevo puntero para la audiencia cuadro. El material
	// anterior queda rechazado; el que apunta a la clave nueva, aceptado.
	const idSustituta = "clave:capacidad:ct-cuadro:prueba-ad369"
	exec(`WITH s AS (SELECT gen_random_bytes(32) AS b),
	       o AS (SELECT max(orden) + 1 AS n FROM vec_autorizacion_atestada_v3.puntero_clave_emision)
	  INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
	   (clave_id, version, revision_gobierno, huella_gobierno_sha256, secreto_hmac, huella_secreto_sha256,
	    emisor_id, audiencia_consumo, valida_desde, valida_hasta, acto_ref)
	  SELECT $3, o.n, o.n, repeat('5', 64), s.b, encode(sha256(s.b), 'hex'),
	         $1, $2, clock_timestamp() - interval '1 day', clock_timestamp() + interval '1 day',
	         'acto:ct:desarrollo:clave-capacidad:prueba-ad369' FROM s, o`, p.ct[3].EmisorID, p.ct[3].Audiencia, idSustituta)
	exec(`INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision (orden, clave_id, version, establecida_en, acto_ref)
	  SELECT version, clave_id, version, clock_timestamp() - interval '1 second', 'acto:ct:desarrollo:puntero-clave:prueba-ad369'
	    FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id = $1`, idSustituta)
	e.exigirRechazo(t, "ct", p.json(t, "ct"), "clave sustituida")
	nueva := p
	var version, revision int64
	if err := e.admin.QueryRow(ctx, `SELECT version::bigint, revision_gobierno::bigint, huella_gobierno_sha256, huella_secreto_sha256
	    FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE clave_id = $1`, idSustituta).
		Scan(&version, &revision, &nueva.ct[3].HuellaGobierno, &nueva.ct[3].HuellaSecreto); err != nil {
		t.Fatal(err)
	}
	nueva.ct[3].ClaveID, nueva.ct[3].Version, nueva.ct[3].RevisionGobierno =
		idSustituta, uint64(version), uint64(revision)
	e.exigirSinDesfaseResuelto(t)
	e.exigirAcepta(t, nueva)

	// Clave revocada: CT detalle. B2 no se ve afectado.
	exec(`INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
	  (clave_id, version, revocada_en, motivo_catalogado_ref, acto_ref)
	  VALUES ($1, $2, clock_timestamp() - interval '1 second', 'motivo:prueba:ad369', 'acto:ct:desarrollo:prueba-ad369:revocacion')`,
		nueva.ct[4].ClaveID, int64(nueva.ct[4].Version))
	e.exigirRechazo(t, "ct", nueva.json(t, "ct"), "clave revocada")
	if ok, err := e.sonda("personal_b2", p.json(t, "personal_b2")); err != nil || !ok {
		t.Fatalf("la revocación CT afectó a B2: %v", err)
	}
}

// escribirMaterialCT deja las coordenadas públicas (sin secretos) de CT para
// la prueba de internactproveedores, que ejecuta la consulta exacta de
// vec-interno contra este mismo gobierno.
func (e *ensayoEscalas) escribirMaterialCT(t *testing.T, p publicacionEscalas) {
	t.Helper()
	ruta := os.Getenv("VEC_AD369_MATERIAL_CT")
	if ruta == "" {
		return
	}
	var r lecturaConfiguracion
	r.Revision, r.Secuencia = p.config.Revision, p.config.Secuencia
	salida, err := json.Marshal(struct {
		Material json.RawMessage      `json:"material"`
		Esperada lecturaConfiguracion `json:"esperada"`
	}{p.json(t, "ct"), r})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, salida, 0o600); err != nil {
		t.Fatal(err)
	}
}
