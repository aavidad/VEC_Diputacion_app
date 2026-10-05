package adminperfiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	is "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteIDsPruebaADMIN struct{ ids IdentificadoresFuenteADMIN }

func (f fuenteIDsPruebaADMIN) ResolverIdentificadoresADMIN(context.Context, ReferenciaFuenteIdentificadoresADMIN) (IdentificadoresFuenteADMIN, error) {
	return f.ids, nil
}

type seudIDsPruebaADMIN struct {
	resultado is.SeudonimosAlta
	recibidos is.IdentificadoresAlta
	llamadas  int
}

func (s *seudIDsPruebaADMIN) SeudonimizarAlta(_ context.Context, ids is.IdentificadoresAlta) (is.SeudonimosAlta, error) {
	s.llamadas++
	resultado := s.resultado
	if s.llamadas == 1 {
		s.recibidos = ids
	} else {
		resultado.CuentaIDHMAC = [32]byte{3}
	}
	return resultado, nil
}

func TestIdentificadoresADMINExigeTresHMACYCoordenadasOriginales(t *testing.T) {
	r := ReferenciaFuenteIdentificadoresADMIN{PersonaRef: "per_" + strings.Repeat("a", 32), CuentaRef: "cta_" + strings.Repeat("b", 32), CuentaOrdinariaRef: "cta_" + strings.Repeat("c", 32),
		CertificadoSHA256: strings.Repeat("d", 64), CASHA256: strings.Repeat("e", 64), EspacioIdentidad: "https://sintetico.example.invalid", EsquemaHMAC: is.EsquemaHMACSHA256V1,
		DominioHMACRef: "idh_" + strings.Repeat("f", 32), ClaveHMACID: "clave-sintetica-v1", ClaveHMACVersion: 1, FuenteRef: "fuente:ids:sintetica", FuenteSHA256: strings.Repeat("1", 64),
		SujetoHMAC: [32]byte{1}, CuentaHMAC: [32]byte{2}, CuentaOrdinariaHMAC: [32]byte{3}}
	ids := IdentificadoresFuenteADMIN{SujetoID: "sujeto-sintetico-original", CuentaID: "admin-sintetica-original", CuentaOrdinariaID: "ordinaria-sintetica-original",
		EspacioIdentidad: r.EspacioIdentidad, DominioHMACRef: r.DominioHMACRef, ClaveHMACID: r.ClaveHMACID, ClaveHMACVersion: 1, FuenteRef: r.FuenteRef, FuenteSHA256: r.FuenteSHA256}
	base := is.SeudonimosAlta{Esquema: r.EsquemaHMAC, EspacioIdentidad: r.EspacioIdentidad, DominioRef: r.DominioHMACRef, ClaveID: r.ClaveHMACID, ClaveVersion: 1,
		AsercionIDHMAC: [32]byte{4}, SesionIDHMAC: [32]byte{5}, SujetoIDHMAC: r.SujetoHMAC, CuentaIDHMAC: r.CuentaHMAC, CuentaOrdinariaIDHMAC: [32]byte{6}}
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*IdentificadoresFuenteADMIN, *is.SeudonimosAlta)
		permitido bool
	}{
		{"fuente original", func(*IdentificadoresFuenteADMIN, *is.SeudonimosAlta) {}, true},
		{"sujeto distinto", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.SujetoIDHMAC[0]++ }, false},
		{"cuenta distinta", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.CuentaIDHMAC[0]++ }, false},
		{"sujeto igual a cuenta", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.SujetoIDHMAC = s.CuentaIDHMAC }, false},
		{"otra generación", func(_ *IdentificadoresFuenteADMIN, s *is.SeudonimosAlta) { s.ClaveVersion++ }, false},
		{"otra procedencia", func(i *IdentificadoresFuenteADMIN, _ *is.SeudonimosAlta) { i.FuenteSHA256 = strings.Repeat("2", 64) }, false},
		{"sin preimagen", func(i *IdentificadoresFuenteADMIN, _ *is.SeudonimosAlta) { i.SujetoID = "" }, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			i, s := ids, base
			caso.cambiar(&i, &s)
			seud := &seudIDsPruebaADMIN{resultado: s}
			v, err := cotejarIdentificadoresFuenteADMIN(context.Background(), fuenteIDsPruebaADMIN{i}, seud, r)
			if (err == nil) != caso.permitido {
				t.Fatal("cotejo no coincide con autoridad esperada")
			}
			if caso.permitido && (seud.llamadas != 2 || v != ids || seud.recibidos.SujetoID != ids.SujetoID || seud.recibidos.CuentaID != ids.CuentaID || seud.recibidos.CuentaOrdinariaID != ids.CuentaOrdinariaID) {
				t.Fatal("se alteraron los identificadores originales")
			}
		})
	}
	t.Run("alias ordinario registrado distinto", func(t *testing.T) {
		otra := r
		otra.CuentaOrdinariaHMAC[0]++
		seud := &seudIDsPruebaADMIN{resultado: base}
		if _, err := cotejarIdentificadoresFuenteADMIN(context.Background(), fuenteIDsPruebaADMIN{ids}, seud, otra); err == nil || seud.llamadas != 2 {
			t.Fatal("se aceptó un alias ordinario ajeno al registrado")
		}
	})
}

func TestIS16RechazaAcuseAjenoIncompletoODuplicado(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	corr := "correlacion_" + strings.Repeat("a", 32)
	a := acuseIS16{Referencia: "aud_v3_ap2_" + strings.Repeat("b", 32), Secuencia: 1, Huella: strings.Repeat("c", 64), Correlacion: corr, RegistradaEn: ahora}
	bruto := []byte(`{"estado":"denegado","datos":null}`)
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*acuseIS16)
		permitido bool
	}{
		{"denegación confirmada", func(*acuseIS16) {}, true},
		{"otra correlación", func(a *acuseIS16) { a.Correlacion = "correlacion_" + strings.Repeat("d", 32) }, false},
		{"otro evento", func(a *acuseIS16) { a.Referencia = "aud_v3_ap2_" + strings.Repeat("d", 32) }, false},
		{"sin secuencia", func(a *acuseIS16) { a.Secuencia = 0 }, false},
		{"sin huella", func(a *acuseIS16) { a.Huella = "" }, false},
		{"reloj SQL un segundo adelantado", func(a *acuseIS16) { a.RegistradaEn = ahora.Add(time.Second) }, true},
		{"en el límite de tolerancia", func(a *acuseIS16) { a.RegistradaEn = ahora.Add(toleranciaRelojAcuseIS16) }, true},
		{"fecha futura", func(a *acuseIS16) { a.RegistradaEn = ahora.Add(toleranciaRelojAcuseIS16 + time.Microsecond) }, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			v := a
			caso.cambiar(&v)
			b, _ := json.Marshal(v)
			_, decision, err := leerResultadoIS16(bruto, b, "evento_"+strings.Repeat("b", 32), corr, ahora)
			if (err == nil) != caso.permitido || (caso.permitido && decision == nil) {
				t.Fatal("se aceptó un acuse no acreditado")
			}
		})
	}
	b, _ := json.Marshal(a)
	for _, v := range [][]byte{[]byte(`{"estado":"permitido","estado":"denegado","datos":null}`), []byte(`{"estado":"denegado","datos":null,"extra":1}`)} {
		if _, _, err := leerResultadoIS16(v, b, "evento_"+strings.Repeat("b", 32), corr, ahora); err == nil {
			t.Fatal("se aceptó resultado ambiguo")
		}
	}
}

type filaIS16Prueba struct {
	resultado, acuse []byte
	err              error
}

func (f filaIS16Prueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*[]byte) = f.resultado
	*destinos[1].(*[]byte) = f.acuse
	return nil
}

type txIS16Prueba struct {
	pgx.Tx
	pool *poolIS16Prueba
	sub  bool
}

func (tx *txIS16Prueba) Begin(context.Context) (pgx.Tx, error) {
	if tx.pool.falloSubBegin != nil {
		return nil, tx.pool.falloSubBegin
	}
	return &txIS16Prueba{pool: tx.pool, sub: true}, nil
}
func (tx *txIS16Prueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, tx.pool.falloSet
}
func (tx *txIS16Prueba) QueryRow(_ context.Context, consulta string, args ...any) pgx.Row {
	p := tx.pool
	var evento, corr string
	if tx.sub {
		p.lecturas++
		evento = args[9].(string)
		corr = args[10].(string)
		if strings.Contains(consulta, "vincular_sesion_admin_perfiles_v1") {
			evento = args[13].(string)
			corr = args[14].(string)
		}
	} else {
		if !strings.Contains(consulta, "rechazar_fuente_cuenta_admin_v1") || !p.revertida || len(args) != 3 {
			panic("error sin revertir lectura positiva")
		}
		p.errores++
		evento = args[0].(string)
		corr = args[1].(string)
		p.accionRechazo = args[2].(string)
	}
	a, _ := json.Marshal(acuseIS16{Referencia: "aud_v3_ap2_" + evento[7:], Secuencia: 1, Huella: strings.Repeat("f", 64), Correlacion: corr, RegistradaEn: p.ahora})
	bruto := p.cuenta
	if !tx.sub {
		bruto = []byte(`{"estado":"error","datos":null}`)
	}
	return filaIS16Prueba{resultado: bruto, acuse: a, err: p.falloQuery}
}
func (tx *txIS16Prueba) Rollback(context.Context) error {
	if tx.sub {
		tx.pool.revertida = true
	}
	return nil
}
func (tx *txIS16Prueba) Commit(context.Context) error {
	if tx.sub {
		tx.pool.subconfirmadas++
		return tx.pool.falloSubCommit
	} else {
		tx.pool.confirmadas++
		return tx.pool.falloCommit
	}
}

type poolIS16Prueba struct {
	ahora                                                                        time.Time
	cuenta                                                                       []byte
	lecturas, errores, confirmadas, subconfirmadas                               int
	revertida                                                                    bool
	accionRechazo                                                                string
	inicios                                                                      int
	falloBegin, falloSubBegin, falloSet, falloQuery, falloSubCommit, falloCommit error
}

func (p *poolIS16Prueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	if p.falloBegin != nil {
		return nil, p.falloBegin
	}
	return &txIS16Prueba{pool: p}, nil
}
func (*poolIS16Prueba) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("consulta fuera de transacción")
}

func TestIS16FalloFuenteRevierteLecturaYConfirmaErrorComun(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1", CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), AutenticacionVerificadaEn: ahora, RevocacionVerificadaEn: ahora, CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Minute)}
	datos, _ := json.Marshal(cuentaSQLIS16{PersonaRef: "per_" + strings.Repeat("a", 32), CuentaRef: "cta_" + strings.Repeat("b", 32), CuentaOrdinariaRef: "cta_" + strings.Repeat("c", 32), EspacioIdentidad: "https://sintetico.example.invalid", EsquemaHMAC: is.EsquemaHMACSHA256V1, ClaveHMACVersion: 1,
		FuenteSHA256: strings.Repeat("1", 64), SujetoHMAC: strings.Repeat("2", 64), CuentaHMAC: strings.Repeat("3", 64), CuentaOrdinariaHMAC: strings.Repeat("4", 64)})
	bruto, _ := json.Marshal(struct {
		Estado string          `json:"estado"`
		Datos  json.RawMessage `json:"datos"`
	}{"permitido", datos})
	pool := &poolIS16Prueba{ahora: ahora, cuenta: bruto}
	p := &PostgreSQL{pool: pool, reloj: relojPrueba{ahora}, identificadores: fuenteIDsPruebaADMIN{}, seudonimizador: &seudIDsPruebaADMIN{}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	c, err := p.ResolverCuentaADMIN(ctx, o)
	if !errors.Is(err, api.ErrConfiguracionIncompleta) || c != (CuentaADMIN{}) || pool.lecturas != 1 || pool.errores != 1 || pool.confirmadas != 1 || pool.subconfirmadas != 0 || !pool.revertida ||
		pool.accionRechazo != "resolver_cuenta_admin" {
		t.Fatal("se devolvió éxito, quedó lectura positiva o no se confirmó el error común")
	}
}

func TestIS16VinculoNoCotejadoRevierteYConfirmaErrorComun(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	o := ObservacionADMIN{Entorno: "desarrollo", Host: "admin.example.invalid", Audiencia: "vec.admin.perfiles.v1", CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), AutenticacionVerificadaEn: ahora, RevocacionVerificadaEn: ahora, CRLVigenteHasta: ahora.Add(time.Minute), CertificadoVigenteHasta: ahora.Add(time.Minute)}
	c := CuentaADMIN{materialCuentaSQL: `{"sintetica":true}`,
		SujetoID: "admin-persona-v1:per_ABCDEFGHIJKLMNOPQRSTUV", CuentaID: "admin-cuenta-v1:cta_abcdefghijklmnopqrstuv", CuentaOrdinariaID: "admin-cuenta-v1:cta_zyxwvutsrqponmlkjihgfe",
		PersonaRef: "per_ABCDEFGHIJKLMNOPQRSTUV", CuentaRef: "cta_abcdefghijklmnopqrstuv", CuentaOrdinariaRef: "cta_zyxwvutsrqponmlkjihgfe", PerfilActivoRef: "prf_ABCDEFGHIJKLMNOPQRSTUV",
		RolID: "administracion_perfiles", VinculoRef: "vca_ABCDEFGHIJKLMNOPQRSTUV", VinculoVersion: 1, SeleccionRevision: 1,
		PoliticaGarantiaRef: "pga_ABCDEFGHIJKLMNOPQRSTUV", PoliticaGarantiaHuellaSHA256: strings.Repeat("a", 64), GarantiaObservada: domain.AuthAssuranceHigh, VigenteHasta: ahora.Add(time.Minute)}
	refs := ReferenciasSesionADMIN{AutenticacionRef: "aut_" + strings.Repeat("a", 22), SesionRef: "ses_" + strings.Repeat("a", 22)}
	// Vínculo bien formado, pero con otra referencia que la pedida por Go.
	ref := "vis_" + strings.Repeat("e", 32)
	ajeno, _ := json.Marshal(vinculoSQLIS16{Referencia: ref, Version: 1, Huella: strings.Repeat("d", 64), Autenticacion: refs.AutenticacionRef, Sesion: refs.SesionRef,
		Persona: c.PersonaRef, Cuenta: c.CuentaRef, Ordinaria: c.CuentaOrdinariaRef, Perfil: c.PerfilActivoRef, Certificado: o.CertificadoSHA256, CA: o.CASHA256,
		VinculoCertificado: c.VinculoRef, VinculoCertificadoVersion: 1, Politica: c.PoliticaGarantiaRef, PoliticaSHA: c.PoliticaGarantiaHuellaSHA256, Seleccion: 1,
		Control: "cse_" + strings.Repeat("a", 22), ControlRevision: 1, ControlSHA: strings.Repeat("a", 64), Vinculada: ahora, Hasta: ahora.Add(time.Minute),
		Fuente: "vinculo_sesion_admin:" + ref[4:], FuenteSHA: strings.Repeat("d", 64)})
	if _, err := decodificarVinculoIS16(ajeno); err != nil {
		t.Fatal("el vínculo sintético debe ser válido por sí solo")
	}
	for nombre, datos := range map[string][]byte{"vínculo ajeno": ajeno, "vínculo ilegible": []byte(`{"referencia":"vis_x"}`)} {
		t.Run(nombre, func(t *testing.T) {
			bruto, _ := json.Marshal(struct {
				Estado string          `json:"estado"`
				Datos  json.RawMessage `json:"datos"`
			}{"permitido", datos})
			pool := &poolIS16Prueba{ahora: ahora, cuenta: bruto}
			p := &PostgreSQL{pool: pool, reloj: relojPrueba{ahora}, identificadores: fuenteIDsPruebaADMIN{}, seudonimizador: &seudIDsPruebaADMIN{}}
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			v, err := p.VincularSesionADMINConAcuse(ctx, o, c, refs)
			if !errors.Is(err, api.ErrConfiguracionIncompleta) || v != (VinculoSesionADMIN{}) || pool.lecturas != 1 || pool.errores != 1 ||
				pool.confirmadas != 1 || pool.subconfirmadas != 0 || !pool.revertida || pool.accionRechazo != "vincular_sesion_admin" {
				t.Fatal("vínculo no cotejado devuelto, no revertido o sin error común confirmado")
			}
		})
	}
}

func TestIS16InterbloqueoYSerializacionSonConflicto(t *testing.T) {
	for _, codigo := range []string{"40001", "40P01"} {
		if !errors.Is(errorConsultaIS16(&pgconn.PgError{Code: codigo, Message: "detalle sintético"}), api.ErrConflictoEstado) {
			t.Fatalf("%s no se trata como conflicto reintentable", codigo)
		}
	}
	for _, codigo := range []string{"42501", "57014", "XX000"} {
		if !errors.Is(errorConsultaIS16(&pgconn.PgError{Code: codigo}), api.ErrConfiguracionIncompleta) {
			t.Fatalf("%s no queda como configuración incompleta", codigo)
		}
	}
}

func TestADMINNoSerializaIdentificadoresNiMaterialSQL(t *testing.T) {
	c := CuentaADMIN{SujetoID: "ORIGINAL-PRIVADO", CuentaID: "cuenta-privada", materialCuentaSQL: "MATERIAL-PRIVADO"}
	ids := IdentificadoresFuenteADMIN{SujetoID: c.SujetoID}
	for _, v := range []any{c, ids} {
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "PRIVADO") || strings.Contains(fmt.Sprintf("%+v %#v", v, v), "ORIGINAL-PRIVADO") {
			t.Fatal("identificadores expuestos")
		}
	}
}
