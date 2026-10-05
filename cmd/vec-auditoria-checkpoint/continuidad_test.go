package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "vec-diputacion-granada/config/auditoriacheckpoint"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestCLIContinuidadConAnclaConservadaYLimites(t *testing.T) {
	d := t.TempDir()
	b, err := os.ReadFile("testdata/config.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.AuditoriaCheckpointOffline
	if decodificar(b, &cfg) != nil {
		t.Fatal("config")
	}
	var m, ts [32]byte
	if _, err := rand.Read(m[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(ts[:]); err != nil {
		t.Fatal(err)
	}
	defer clear(m[:])
	defer clear(ts[:])
	p, err := bootstrap.NuevoProveedorCheckpointDesarrollo(m, ts, cfg.Politica)
	if err != nil {
		t.Fatal(err)
	}
	defer p.CerrarCheckpoint()
	der, err := p.PublicaCheckpointDER()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("1", 64)
	firmar := func(primera, ultima uint64, anterior, cabeza string) domain.ReciboCheckpointDesarrollo {
		r, e := application.EmitirCheckpointDesarrollo(context.Background(), domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Politica: cfg.Politica, Cobertura: domain.CoberturaCheckpoint{
			CadenaID: "cadena:sintetica", PrimeraSecuencia: primera, UltimaSecuencia: ultima, AnteriorSHA256: anterior, CabezaSHA256: cabeza, Registros: ultima - primera + 1}}, p, p, 4)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	a := firmar(1, 2, strings.Repeat("0", 64), h)
	s := firmar(3, 4, h, strings.Repeat("2", 64))
	ab, _ := json.Marshal(a)
	sb, _ := json.Marshal(s)
	lote := []byte(`{"esquema":"` + domain.EsquemaContinuidadCheckpointDesarrollo + `","recibos":[` + string(sb) + `]}`)
	guardar := func(nombre string, contenido []byte) string {
		ruta := filepath.Join(d, nombre)
		if e := os.WriteFile(ruta, contenido, 0600); e != nil {
			t.Fatal(e)
		}
		return ruta
	}
	ancla, entrada, spki := guardar("ancla.json", ab), guardar("lote.json", lote), guardar("publica.der", der)
	cfg.MaxBytes = int64(len(ab) + len(lote))
	cfg.MaxRegistros = 4
	cb, _ := json.Marshal(cfg)
	conf := guardar("config.json", cb)
	args := []string{"-operacion", "verificar-continuidad", "-config", conf, "-ancla", ancla, "-entrada", entrada, "-spki", spki, "-pin-spki-sha256", p.PinCheckpoint(), "-max-recibos", "1"}
	var out, log bytes.Buffer
	if run(args, &out, &log) != 0 {
		t.Fatal(out.String())
	}
	var r application.ResultadoContinuidadCheckpoint
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Continuidad != "verificada" || r.Firma != "verificada_con_pin_externo" || r.RegistrosTotal != 4 || r.RecibosVerificados != 2 || r.IntegridadCadena != "no_evaluada" || r.FirmaLegal || r.TiempoIndependiente {
		t.Fatal("resultado", out.String())
	}
	if !strings.Contains(log.String(), `"resultado":"correcto"`) || strings.Contains(out.String(), "cadena:sintetica") || strings.Contains(log.String(), ancla) {
		t.Fatal("metadata o emisor")
	}
	for _, extra := range [][]string{{"-cadena", ""}, {"-kms-master", ""}, {"-tsa-secret", ""}, {"-salida", ""}, {"-max-recibos", "0"}, {"-max-recibos", "257"}, {"-ancla", ""}, {"-pin-spki-sha256", strings.Repeat("0", 64)}, {"-operacion", "verificar"}} {
		out.Reset()
		if run(append(append([]string(nil), args...), extra...), &out, &log) == 0 {
			t.Fatal("argumentos incompatibles admitidos")
		}
	}
	for _, cambiar := range []func(){func() { cfg.MaxBytes-- }, func() { cfg.MaxBytes++; cfg.MaxRegistros = 3 }} {
		cambiar()
		cb, _ := json.Marshal(cfg)
		guardar("config.json", cb)
		out.Reset()
		if run(args, &out, &log) == 0 {
			t.Fatal("presupuesto acumulado admitido")
		}
	}
	for _, bruto := range [][]byte{bytes.Replace(ab, []byte(`"primera_secuencia":1`), []byte(`"primera_secuencia":null`), 1), bytes.Replace(ab, []byte(`,"registros":2`), nil, 1), bytes.Replace(ab, []byte(`"firma_base64":`), []byte(`"SPKI":"otra","firma_base64":`), 1)} {
		if _, e := decodificarReciboContinuidad(bruto); e == nil {
			t.Fatal("recibo incompleto, null o desconocido admitido")
		}
	}
	for _, bruto := range []string{`{"esquema":"` + domain.EsquemaContinuidadCheckpointDesarrollo + `","recibos":null}`, `{"esquema":"uno","esquema":"dos","recibos":[]}`, `{"esquema":"` + domain.EsquemaContinuidadCheckpointDesarrollo + `","recibos":[]}`, string(bytes.Replace(lote, []byte(`"recibos"`), []byte(`"RECIBOS"`), 1)), string(bytes.Replace(lote, []byte(`]}`), []byte(`,`+string(sb)+`]}`), 1))} {
		if _, e := decodificarLoteContinuidad([]byte(bruto), 1); e == nil {
			t.Fatal("lote fuera de contrato admitido")
		}
	}
}
