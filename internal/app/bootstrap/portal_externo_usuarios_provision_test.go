package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func solicitudProvisionUsuariosPrueba(t *testing.T) SolicitudProvisionUsuariosExterno {
	return SolicitudProvisionUsuariosExterno{CuentaRef: "cta_usuarios_sintetica", AprobacionRef: "aprobacion:sintetica",
		Instantanea: instantaneaPerfilUsuariosExternoPrueba(t)}
}

func TestProvisionUsuariosExternoHuellaLigaCuentaAprobacionYCAS(t *testing.T) {
	s := solicitudProvisionUsuariosPrueba(t)
	h, err := s.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	for _, cambiar := range []func(*SolicitudProvisionUsuariosExterno){
		func(s *SolicitudProvisionUsuariosExterno) { s.CuentaRef = "cta_otra_sintetica" },
		func(s *SolicitudProvisionUsuariosExterno) { s.AprobacionRef = "aprobacion:otra" },
		func(s *SolicitudProvisionUsuariosExterno) {
			s.Instantanea.AsignacionPerfil.PerfilActivoRef = "prf_otro_sintetico"
		},
		func(s *SolicitudProvisionUsuariosExterno) {
			s.VersionAsignacionEsperada = 1
			s.HuellaAsignacionEsperada = strings.Repeat("a", 64)
			s.Instantanea.AsignacionPerfil.Version = 2
		},
		func(s *SolicitudProvisionUsuariosExterno) {
			s.RevisionControlEsperada = 1
			s.HuellaControlEsperada = strings.Repeat("b", 64)
			s.Instantanea.ControlVigenciaVersionRol.Revision = 2
		},
	} {
		otra := s
		cambiar(&otra)
		nueva, err := otra.HuellaSHA256()
		if err != nil || nueva == h {
			t.Fatalf("material distinto no quedó ligado: %v", err)
		}
	}
	p, err := s.preparar()
	if err != nil {
		t.Fatal(err)
	}
	hRol, _ := s.Instantanea.VersionRol.HuellaSHA256()
	hAsignacion, _ := s.Instantanea.AsignacionPerfil.HuellaSHA256()
	if p.HuellaRol != hRol || p.HuellaAsignacion != hAsignacion {
		t.Fatal("se cambió el canon del dominio")
	}
}

func TestProvisionUsuariosExternoDecoderAcotado(t *testing.T) {
	s := solicitudProvisionUsuariosPrueba(t)
	b, _ := json.Marshal(s)
	for nombre, b := range map[string][]byte{
		"concatenado":       append(append([]byte(nil), b...), b...),
		"campo duplicado":   append([]byte(`{"cuenta_ref":"cta_distinta",`), b[1:]...),
		"campo desconocido": append([]byte(`{"extra":true,`), b[1:]...),
		"demasiado grande":  bytes.Repeat([]byte(" "), 262145),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := LeerSolicitudProvisionUsuariosExterno(bytes.NewReader(b)); err == nil {
				t.Fatal("entrada inválida admitida")
			}
		})
	}
	leida, err := LeerSolicitudProvisionUsuariosExterno(bytes.NewReader(b))
	if err != nil || !reflect.DeepEqual(s, leida) {
		t.Fatalf("plan válido rechazado: %v", err)
	}
}

type filaProvisionUsuariosPrueba struct {
	valores []any
	err     error
}

func (f filaProvisionUsuariosPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas incompatibles")
	}
	for i, d := range destinos {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(f.valores[i]))
	}
	return nil
}

type txProvisionUsuariosPrueba struct {
	filas     []filaProvisionUsuariosPrueba
	consultas []string
	args      [][]any
	commit    int
	errCommit error
}

func (tx *txProvisionUsuariosPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	tx.consultas = append(tx.consultas, q)
	tx.args = append(tx.args, args)
	i := len(tx.consultas) - 1
	if i >= len(tx.filas) {
		return filaProvisionUsuariosPrueba{err: errors.New("consulta inesperada")}
	}
	return tx.filas[i]
}
func (tx *txProvisionUsuariosPrueba) Commit(context.Context) error { tx.commit++; return tx.errCommit }

func TestProvisionUsuariosExternoConfirmaSoloDosRespuestasExactas(t *testing.T) {
	s := solicitudProvisionUsuariosPrueba(t)
	p, _ := s.preparar()
	h, _ := s.HuellaSHA256()
	for _, caso := range []string{"exito", "fallo rol", "rol divergente", "fallo asignacion", "asignacion divergente", "fallo commit"} {
		t.Run(caso, func(t *testing.T) {
			tx := &txProvisionUsuariosPrueba{filas: []filaProvisionUsuariosPrueba{
				{valores: []any{s.Instantanea.VersionRol.Referencia(), 1, "1", p.HuellaRol, p.HuellaControl}},
				{valores: []any{s.Instantanea.AsignacionPerfil.Referencia(), 1, p.HuellaAsignacion}},
			}}
			switch caso {
			case "fallo rol":
				tx.filas[0].err = errors.New("private database details")
			case "rol divergente":
				tx.filas[0].valores[3] = strings.Repeat("0", 64)
			case "fallo asignacion":
				tx.filas[1].err = errors.New("private database details")
			case "asignacion divergente":
				tx.filas[1].valores[0] = "asignacion:ajena:v1"
			case "fallo commit":
				tx.errCommit = errors.New("private database details")
			}
			r, err := publicarPerfilUsuariosEnTransaccion(context.Background(), tx, s, h)
			if caso == "exito" {
				if err != nil || tx.commit != 1 || r.HuellaPlan != h || len(tx.consultas) != 2 {
					t.Fatalf("publicación exacta falló: %v", err)
				}
				if tx.args[0][5] != nil || tx.args[1][3] != nil || tx.args[1][6] != s.CuentaRef {
					t.Fatal("CAS inicial o cuenta cambiados")
				}
			} else {
				if err != errPerfilUsuariosExterno || r != (ReciboProvisionUsuariosExterno{}) {
					t.Fatalf("fallo filtrado incorrectamente: %v", err)
				}
				if caso != "fallo commit" && tx.commit != 0 {
					t.Fatal("commit antes del cotejo completo")
				}
			}
		})
	}
}
