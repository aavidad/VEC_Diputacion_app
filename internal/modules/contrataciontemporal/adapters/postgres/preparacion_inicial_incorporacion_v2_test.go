package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestLeerPreparacionInicialIncorporacionV2Nominal(t *testing.T) {
	_, f, now := fixtureRaicesV2(t)
	var esperada dom.PublicacionDefinicionSeguimiento
	var inicial dom.EstadoPersistidoSeguimiento
	registroV2Exigir(t, json.Unmarshal(f.publicacion, &esperada))
	registroV2Exigir(t, json.Unmarshal(f.estado, &inicial))
	for _, version := range []string{"7", "8", "9007199254740991"} {
		t.Run(version, func(t *testing.T) {
			m := dobleRaicesV2(t, f)
			m.filas[0].versionExp = version
			m.destruir = true
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			p, s, v, e := l.LeerPreparacionInicial(context.Background(), f.org, f.exp, f.rel)
			want, err := decimalRaicesV2(version)
			registroV2Exigir(t, err)
			if e != nil || v != want || !reflect.DeepEqual(p, esperada) || !reflect.DeepEqual(s, inicial) || s.Version != 0 || !m.confirmado {
				t.Fatalf("preparación inicial alterada: versión=%d error=%v", v, e)
			}
			if !reflect.DeepEqual(m.pasos, []string{"begin", "settings", "query", "next", "scan", "next", "commit"}) {
				t.Fatalf("secuencia inesperada: %v", m.pasos)
			}
		})
	}
}

func TestLeerPreparacionInicialIncorporacionV2Rechazos(t *testing.T) {
	o, f, now := fixtureRaicesV2(t)
	for _, caso := range []string{"ausente", "ambigua", "canon", "raiz", "publicacion", "estado_v1", "version", "triple", "entrada", "ctx_nil", "begin", "tx_nil", "settings", "query", "rows_nil", "scan", "rows_err", "commit"} {
		t.Run(caso, func(t *testing.T) {
			m := dobleRaicesV2(t, f)
			m.fallo = caso
			org := f.org
			ctx := context.Background()
			switch caso {
			case "ausente":
				m.filas = nil
			case "ambigua":
				m.filas = append(m.filas, f.clonar())
			case "canon":
				m.filas[0].canon[0] ^= 1
			case "raiz":
				m.filas[0].raiz[0] ^= 1
			case "publicacion":
				m.filas[0].publicacion = []byte("null")
			case "estado_v1":
				w, _ := wireTransporte(t, o, now)
				m.filas[0].estado = serialTransporte(t, w.Historia.Posterior)
			case "version":
				m.filas[0].versionExp = "08"
			case "triple":
				m.filas[0].exp = registroV2Ref("otro")
			case "entrada":
				org = ""
			case "ctx_nil":
				ctx = nil
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			p, s, v, e := l.LeerPreparacionInicial(ctx, org, f.exp, f.rel)
			if !errors.Is(e, ErrResolucionRaizIncorporacionV2) || v != 0 || !reflect.DeepEqual(p, dom.PublicacionDefinicionSeguimiento{}) || !reflect.DeepEqual(s, dom.EstadoPersistidoSeguimiento{}) || m.confirmado {
				t.Fatalf("rechazo no cerrado: %v", e)
			}
			exigirErrorRaizV2(t, "", e, m)
			if (caso == "entrada" || caso == "ctx_nil") && len(m.pasos) != 0 {
				t.Fatal("entrada inválida accede al pool")
			}
		})
	}
}

func TestLeerPreparacionInicialIncorporacionV2Cancelacion(t *testing.T) {
	_, f, now := fixtureRaicesV2(t)
	for _, paso := range []string{"antes", "begin", "settings", "query", "next", "scan", "commit"} {
		t.Run(paso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := dobleRaicesV2(t, f)
			m.hook = func(p string) {
				if p == paso {
					cancel()
				}
			}
			if paso == "antes" {
				cancel()
			}
			l, e := nuevoResolverRaizIncorporacionV2(m, relojLecturaHistoriaV2(func() time.Time { return now }))
			registroV2Exigir(t, e)
			p, s, v, e := l.LeerPreparacionInicial(ctx, f.org, f.exp, f.rel)
			if !errors.Is(e, context.Canceled) || v != 0 || !reflect.DeepEqual(p, dom.PublicacionDefinicionSeguimiento{}) || !reflect.DeepEqual(s, dom.EstadoPersistidoSeguimiento{}) {
				t.Fatalf("cancelación no cerrada: %v", e)
			}
			exigirErrorRaizV2(t, "", e, m)
			if m.confirmado != (paso == "commit") {
				t.Fatal("commit o rollback indebido")
			}
		})
	}
}
