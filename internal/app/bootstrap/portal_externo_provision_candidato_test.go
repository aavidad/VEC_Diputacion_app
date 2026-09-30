package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/config"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func fuenteProvisionCandidatoPrueba() FuenteProvisionCandidatoExterno {
	componente := func(prefijo string) ComponenteSnapshotContextoExterno {
		return ComponenteSnapshotContextoExterno{
			Referencia: prefijo + "sintetico_1234567890123456", Version: 1, ProcedenciaRef: "prc_sintetica_1234567890123456", ProcedenciaVersion: 1,
			ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: "autoridad_maestra_acreditada", Estado: "activo",
			VigenteDesde: "2026-01-01T00:00:00.000000Z", VigenteHasta: "2027-01-01T00:00:00.000000Z"}
	}
	return FuenteProvisionCandidatoExterno{Version: 1, Snapshot: SnapshotContextoExterno{ProvisionRef: "pce_sintetica_1234567890123456", Poblacion: "candidato", Estado: "activo",
		Cuenta: componente("cta_"), Persona: componente("per_"), Perfil: componente("prf_"), Contexto: componente("vca_"),
		VinculoCandidato: &VinculoSnapshotCandidatoExterno{componente("vin_"), "can_sintetico_1234567890123456"}}}
}

func cfgProvisionCandidatoPrueba() config.Config {
	return config.Config{PortalProceso: config.ValorPortalProcesoInterno, ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
}

func TestProvisionCandidatoPlanDeterministaSinIdentidadEnResumen(t *testing.T) {
	f := fuenteProvisionCandidatoPrueba()
	cfg := cfgProvisionCandidatoPrueba()
	p, e := PrepararProvisionCandidatoExterno(cfg, f, "autorizacion", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	p2, e := PrepararProvisionCandidatoExterno(cfg, f, "autorizacion", time.Now().Add(time.Minute))
	if e != nil || !reflect.DeepEqual(p.resumen, p2.resumen) {
		t.Fatal("plan depende del reloj de ejecución")
	}
	resumen, _ := json.Marshal(p.Resumen())
	if strings.Contains(string(resumen), "sintetic") || strings.Contains(fmt.Sprintf("%+v %#v", p, p), f.Snapshot.Persona.Referencia) {
		t.Fatal("plan revela identidad")
	}
	f.Preimagen.VersionAsignacion = 1
	f.Preimagen.HuellaAsignacion = strings.Repeat("b", 64)
	cambiado, e := PrepararProvisionCandidatoExterno(cfg, f, "autorizacion", time.Now())
	if e != nil || cambiado.resumen.HuellaSHA256 == p.resumen.HuellaSHA256 || cambiado.resumen.PreimagenSHA256 == p.resumen.PreimagenSHA256 {
		t.Fatal("preimagen no ligada al plan")
	}
	for _, portal := range []string{"", config.ValorPortalProcesoExterno, "desconocido"} {
		cfg.PortalProceso = portal
		if _, e = PrepararProvisionCandidatoExterno(cfg, f, "autorizacion", time.Now()); e == nil {
			t.Fatal("plan fuera proceso interno")
		}
	}
}

func TestProvisionCandidatoRechazaAprobacionAntesDeConexion(t *testing.T) {
	p, e := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), fuenteProvisionCandidatoPrueba(), "autorizacion", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	for _, par := range [][2]string{{"", p.resumen.PreimagenSHA256}, {strings.Repeat("f", 64), p.resumen.PreimagenSHA256}, {p.resumen.HuellaSHA256, ""}, {p.resumen.HuellaSHA256, strings.Repeat("f", 64)}} {
		if _, e := EjecutarProvisionCandidatoExterno(context.Background(), cfgProvisionCandidatoPrueba(), "DSN no permitido", p, par[0], par[1]); e != ErrProvisionCandidatoExterno {
			t.Fatal("aprobación inválida no rechazada")
		}
	}
}

func TestProvisionCandidatoFuentePrivadaCerrada(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "fuente.json")
	f := fuenteProvisionCandidatoPrueba()
	b, _ := json.Marshal(f)
	if e := os.WriteFile(ruta, b, 0o600); e != nil {
		t.Fatal(e)
	}
	if _, e := CargarFuenteProvisionCandidatoExterno(ruta); e != nil {
		t.Fatal(e)
	}
	for _, contenido := range [][]byte{append(b, []byte(" {}")...), []byte(`{"version":1,"version":1}`), []byte(strings.Replace(string(b), `"version":1`, `"intruso":1`, 1))} {
		if e := os.WriteFile(ruta, contenido, 0o600); e != nil {
			t.Fatal(e)
		}
		if _, e := CargarFuenteProvisionCandidatoExterno(ruta); e == nil {
			t.Fatal("fuente ambigua aceptada")
		}
	}
	if e := os.WriteFile(ruta, b, 0o600); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(ruta, 0o644); e != nil {
		t.Fatal(e)
	}
	if _, e := CargarFuenteProvisionCandidatoExterno(ruta); e == nil {
		t.Fatal("fuente pública aceptada")
	}
	if e := os.Chmod(ruta, 0o600); e != nil {
		t.Fatal(e)
	}
	enlace := filepath.Join(filepath.Dir(ruta), "enlace.json")
	if e := os.Symlink(ruta, enlace); e != nil {
		t.Fatal(e)
	}
	if _, e := CargarFuenteProvisionCandidatoExterno(enlace); e == nil {
		t.Fatal("enlace aceptado")
	}
}

type filaProvisionPrueba struct {
	valores []any
	err     error
}

func (f filaProvisionPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return fmt.Errorf("cardinalidad")
	}
	for i, v := range f.valores {
		reflect.ValueOf(destinos[i]).Elem().Set(reflect.ValueOf(v))
	}
	return nil
}

type consultaProvisionPrueba struct {
	pgx.Tx
	llamadas  int
	respuesta func(int, string, []any) pgx.Row
	escritura func(string, []any) (pgconn.CommandTag, error)
}

func (q *consultaProvisionPrueba) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return q.escritura(sql, args)
}

func (q *consultaProvisionPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	q.llamadas++
	return q.respuesta(q.llamadas, sql, args)
}

func TestProvisionCandidatoContextoCASYRecibo(t *testing.T) {
	s := fuenteProvisionCandidatoPrueba().Snapshot
	h := strings.Repeat("c", 64)
	q := &consultaProvisionPrueba{respuesta: func(n int, sql string, args []any) pgx.Row {
		switch n {
		case 1:
			if !strings.Contains(sql, "preimagen_snapshot") {
				t.Fatal(sql)
			}
			return filaProvisionPrueba{err: pgx.ErrNoRows}
		case 2:
			if args[1] != int64(1) || !strings.Contains(sql, "huella_snapshot") {
				t.Fatal("versión no aprobada")
			}
			return filaProvisionPrueba{valores: []any{h}}
		case 3:
			if args[1] != int64(0) || args[2] != nil || args[3] != h || !strings.Contains(sql, "publicar_snapshot") {
				t.Fatal("CAS perdido")
			}
			return filaProvisionPrueba{valores: []any{s.ProvisionRef, int64(1), h}}
		default:
			t.Fatal("consulta inesperada")
			return nil
		}
	}}
	if e := PublicarSnapshotContextoExterno(context.Background(), q, s, 0, "", h); e != nil || q.llamadas != 3 {
		t.Fatal(e)
	}
	q = &consultaProvisionPrueba{respuesta: func(_ int, _ string, _ []any) pgx.Row {
		return filaProvisionPrueba{valores: []any{int64(2), strings.Repeat("d", 64)}}
	}}
	if _, e := HuellaSnapshotContextoExterno(context.Background(), q, s, 1, h); e == nil || q.llamadas != 1 {
		t.Fatal("CAS divergente escribió o preparó hash")
	}
	usuarios := s
	usuarios.Poblacion = "usuarios"
	usuarios.ProvisionRef = "pue_sintetica_1234567890123456"
	usuarios.VinculoCandidato = nil
	if !usuarios.validar() {
		t.Fatal("helper no admite Usuarios sin vínculo candidato")
	}
	if _, e := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), FuenteProvisionCandidatoExterno{Version: 1, Snapshot: usuarios}, "autorizacion", time.Now()); e == nil {
		t.Fatal("AUT16 acepta Usuarios")
	}
}

func TestProvisionCandidatoAUT16ConservaCanonYRechazaReciboAjeno(t *testing.T) {
	p, e := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), fuenteProvisionCandidatoPrueba(), "autorizacion", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	q := &consultaProvisionPrueba{respuesta: func(n int, sql string, args []any) pgx.Row {
		if n == 1 {
			if !strings.Contains(sql, "publicar_rol_candidato") || !reflect.DeepEqual(args[0], p.rol.RolDocumento) {
				t.Fatal("canon de rol alterado")
			}
			return filaProvisionPrueba{valores: []any{p.semilla.VersionRol.Referencia(), int64(1), int64(1), p.rol.RolHuellaSHA256, p.rol.ControlHuellaSHA256}}
		}
		if !strings.Contains(sql, "publicar_asignacion_candidato") || args[2] != int64(0) || !reflect.DeepEqual(args[0], p.asignacion.Documento) {
			t.Fatal("asignación fuera de canon")
		}
		return filaProvisionPrueba{valores: []any{"asignacion:ajena:v1", int64(1), p.asignacion.HuellaSHA256}}
	}}
	if publicarAutorizacionCandidatoExterno(context.Background(), q, p) == nil || q.llamadas != 2 {
		t.Fatal("recibo ajeno confirmado")
	}
}

func TestProvisionCandidatoIdentidadCanonID7YAprobacionAtomica(t *testing.T) {
	vector := IdentidadProvisionCandidatoExterno{CuentaRef: "cta_sintetica_1234567890123456", Esquema: "hmac_sha256_v1", DominioRef: "idh_sintetico_1234567890123456", ClaveID: "vec.identidad.desarrollo.externo.g1", ClaveVersion: 1, CuentaHMAC: strings.Repeat("a", 64), SujetoHMAC: strings.Repeat("b", 64)}
	if huellaMaterialIdentidadExterna(vector) != "53db413c45d71916c6952906f6fec3cc08f1241168bdb4821f183f7a5181abd3" ||
		huellaAusenciaIdentidadExterna(vector.CuentaRef) != "01222f36b642a750e8c23980fb46cf185357a53b53bc10c18ad0700f3fbe9446" {
		t.Fatal("canon ID7 divergente")
	}
	f := fuenteProvisionCandidatoPrueba()
	i := vector
	i.CuentaRef = f.Snapshot.Cuenta.Referencia
	i.DominioRef = dominioIdentidadSesionDesarrollo
	i.Esquema = postgresidentidad.EsquemaHMACSHA256V1
	f.Identidad = &i
	p, e := PrepararProvisionCandidatoExterno(cfgProvisionCandidatoPrueba(), f, "identidad", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	preimagen := p.resumen.PreimagenSHA256
	i.CuentaHMAC = strings.Repeat("c", 64)
	if p.identidad.CuentaHMAC == i.CuentaHMAC {
		t.Fatal("plan comparte material mutable")
	}
	orden := 0
	q := &consultaProvisionPrueba{escritura: func(sql string, args []any) (pgconn.CommandTag, error) {
		orden++
		if orden != 1 || !strings.Contains(sql, "INSERT INTO vec_identidad_externa_v1.aprobacion_alta") || args[2] != preimagen {
			t.Fatal("aprobación no ligada a ausencia")
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}, respuesta: func(_ int, sql string, args []any) pgx.Row {
		orden++
		if orden != 2 || !strings.Contains(sql, "confirmar_alta_v1") || args[10] != preimagen || args[11] != int64(0) {
			t.Fatal("confirmación anterior a aprobación o CAS distinto")
		}
		return filaProvisionPrueba{valores: []any{f.Snapshot.Cuenta.Referencia}}
	}}
	if e := publicarIdentidadCandidatoExterno(context.Background(), q, p); e != nil || orden != 2 {
		t.Fatal(e)
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", *p.identidad, *p.identidad), strings.Repeat("a", 64)) {
		t.Fatal("alias en registro")
	}
}
