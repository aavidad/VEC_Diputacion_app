package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	dominio "vec-diputacion-granada/internal/vec/domain"
	vec "vec-diputacion-granada/internal/vec/ports"
)

// Doble de transporte, sin firma ni autorización reales: solo verifica los
// límites de la transacción y que los bytes originales no se sustituyen.
type poolLecturaMiBolsaPrueba struct {
	t             *testing.T
	preparar      func(int) *txLecturaMiBolsaPrueba
	transacciones []*txLecturaMiBolsaPrueba
}

func (p *poolLecturaMiBolsaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite {
		p.t.Fatal("aislamiento o modo alterados")
	}
	if n := len(p.transacciones); n > 0 && p.transacciones[n-1].reversiones != 1 {
		p.t.Fatal("se abrió otro intento sin cerrar el anterior")
	}
	tx := p.preparar(len(p.transacciones))
	p.transacciones = append(p.transacciones, tx)
	return tx, nil
}

type txLecturaMiBolsaPrueba struct {
	transaccionPanelPostgreSQLPrueba
	errorConfiguracion  error
	filas               []filaPanelPostgreSQLPrueba
	consultas           []string
	argumentosConsultas [][]any
	alRevertir          func()
}

func (tx *txLecturaMiBolsaPrueba) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	etiqueta, _ := tx.transaccionPanelPostgreSQLPrueba.Exec(ctx, sql, args...)
	return etiqueta, tx.errorConfiguracion
}

func (tx *txLecturaMiBolsaPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx.transaccionPanelPostgreSQLPrueba.QueryRow(ctx, sql, args...)
	tx.consultas = append(tx.consultas, sql)
	tx.argumentosConsultas = append(tx.argumentosConsultas, tx.argumentos)
	return tx.filas[len(tx.consultas)-1]
}

func (tx *txLecturaMiBolsaPrueba) Rollback(ctx context.Context) error {
	if ctx.Err() != nil {
		panic("rollback con contexto cancelado")
	}
	if _, ok := ctx.Deadline(); !ok {
		panic("rollback sin límite")
	}
	if tx.alRevertir != nil {
		tx.alRevertir()
	}
	return tx.transaccionPanelPostgreSQLPrueba.Rollback(ctx)
}

func materialLecturaMiBolsaPrueba(t *testing.T, instante time.Time) vec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	huella := strings.Repeat("a", 64)
	resumen, err := vec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", huella, huella,
		"contexto:prueba", huella, bolsa.AccionConsultarMiBolsa, "lectura:prueba", huella,
		bolsa.AudienciaMiBolsa, instante, instante.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{0x71}, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	m, err := vec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{1}, 512), resumen,
		[]byte("decision"), []byte("motivo"), []byte("contexto"), 1, 2, []byte("payload"), []byte("cose"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type casoLecturaMiBolsaPrueba struct {
	nombre       string
	ejecutar     func(context.Context, *ConsultaMiBolsaPostgreSQL) (any, error)
	respuesta    func(string) []byte
	vacio        any
	noDisponible error
	denegado     error
}

func casosLecturaMiBolsaPrueba(t *testing.T) []casoLecturaMiBolsaPrueba {
	t.Helper()
	instante := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	material := materialLecturaMiBolsaPrueba(t, instante)
	return []casoLecturaMiBolsaPrueba{
		{nombre: "participaciones", vacio: bolsa.InstantaneaMiBolsa{}, noDisponible: bolsa.ErrMaterialMiBolsaNoDisponible,
			denegado: bolsa.ErrConsultaMiBolsaInvalida,
			ejecutar: func(ctx context.Context, r *ConsultaMiBolsaPostgreSQL) (any, error) {
				return r.ConsultarMiBolsa(ctx, bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_sintetico", ConsultadaEn: instante, Material: material})
			},
			respuesta: func(marca string) []byte {
				return []byte(fmt.Sprintf(`{"consultada_en":"2026-09-30T10:00:00Z","participaciones":[{"bolsa":"bolsa:%s","categoria":"categoria:auxiliar","version":1,"orden_inicial":1,"total_instantanea":1,"estado_bolsa":"vigente","vigente_desde":"2026-09-01T00:00:00Z"}]}`, marca))
			}},
		{nombre: "historial", vacio: bolsa.PaginaHistorialMiBolsa{}, noDisponible: bolsa.ErrHistorialMiBolsaNoDisponible, denegado: dominio.ErrAutorizacionDenegada,
			ejecutar: func(ctx context.Context, r *ConsultaMiBolsaPostgreSQL) (any, error) {
				return r.ConsultarHistorialMiBolsa(ctx, bolsa.SolicitudConsultaHistorialMiBolsa{CandidatoRef: "can_sintetico", ConsultadaEn: instante, Pagina: 2, Material: material})
			},
			respuesta: func(marca string) []byte {
				return []byte(fmt.Sprintf(`{"consultada_en":"2026-09-30T10:00:00Z","pagina":2,"tamano":20,"hay_mas":false,"items":[{"clase":"llamamiento","bolsa":"bolsa:%s","categoria":"categoria:auxiliar","ocurrido_en":"2026-09-29T10:00:00Z","canal":"correo","resultado":"enviado"}]}`, marca))
			}},
	}
}

func TestLecturasMiBolsaReintentanTransaccionCompleta(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		for _, etapa := range []string{"configuracion", "consulta", "commit"} {
			for _, codigo := range []string{"40001", "40P01"} {
				t.Run(caso.nombre+"/"+etapa+"/"+codigo, func(t *testing.T) {
					pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(n int) *txLecturaMiBolsaPrueba {
						tx := &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{contenido: caso.respuesta("confirmada")}}}
						if n == 0 {
							aborto := &pgconn.PgError{Code: codigo}
							tx.filas[0].contenido = caso.respuesta("abortada")
							switch etapa {
							case "configuracion":
								tx.errorConfiguracion = aborto
							case "consulta":
								tx.filas[0].error = aborto
							case "commit":
								tx.errorCommit = aborto
							}
						}
						return tx
					}}
					obtenido, err := caso.ejecutar(context.Background(), &ConsultaMiBolsaPostgreSQL{pool: pool})
					if err != nil || len(pool.transacciones) != 2 {
						t.Fatalf("resultado/reintentos: %v, %d", err, len(pool.transacciones))
					}
					primero, segundo := pool.transacciones[0], pool.transacciones[1]
					if primero == segundo || primero.reversiones != 1 || segundo.reversiones != 1 || segundo.confirmaciones != 1 || segundo.configuraciones != 1 {
						t.Fatal("transacciones no se cerraron y confirmaron por separado")
					}
					if etapa != "configuracion" && (!reflect.DeepEqual(primero.consultas, segundo.consultas) || !reflect.DeepEqual(primero.argumentosConsultas, segundo.argumentosConsultas)) {
						t.Fatal("operación o material sustituidos durante el reintento")
					}
					switch r := obtenido.(type) {
					case bolsa.InstantaneaMiBolsa:
						if len(r.Participaciones) != 1 || r.Participaciones[0].Bolsa != "bolsa:confirmada" {
							t.Fatal("resultado abortado o duplicado")
						}
					case bolsa.PaginaHistorialMiBolsa:
						if len(r.Items) != 1 || r.Items[0].Bolsa != "bolsa:confirmada" || r.Pagina != 2 {
							t.Fatal("historial abortado o duplicado")
						}
					}
				})
			}
		}
	}
}

func TestLecturasMiBolsaNoRepitenFalloDefinitivoNiCommitIncierto(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		for _, fallo := range []error{&pgconn.PgError{Code: "42501"}, &pgconn.PgError{Code: "55P03"}, &pgconn.PgError{Code: "23505"}, &pgconn.PgError{Code: "22023"}, &pgconn.PgError{Code: "57014"}, io.ErrUnexpectedEOF, pgx.ErrTxCommitRollback} {
			for _, etapa := range []string{"consulta", "commit"} {
				t.Run(caso.nombre+"/"+etapa+"/"+fallo.Error(), func(t *testing.T) {
					pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(int) *txLecturaMiBolsaPrueba {
						tx := &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{contenido: caso.respuesta("incierta")}}}
						if etapa == "consulta" {
							tx.filas[0].error = fallo
						} else {
							tx.errorCommit = fallo
						}
						return tx
					}}
					obtenido, err := caso.ejecutar(context.Background(), &ConsultaMiBolsaPostgreSQL{pool: pool})
					if err == nil || !reflect.DeepEqual(obtenido, caso.vacio) || len(pool.transacciones) != 1 || pool.transacciones[0].reversiones != 1 {
						t.Fatal("se repitió un fallo definitivo o se publicó un resultado sin commit")
					}
					var pg *pgconn.PgError
					if errors.As(err, &pg) {
						t.Fatal("se expuso error de PostgreSQL")
					}
				})
			}
		}
	}
}

func TestLecturasMiBolsaCancelacionYAgotamiento(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		for _, cancelar := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/cancelar=%t", caso.nombre, cancelar), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(int) *txLecturaMiBolsaPrueba {
					tx := &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{error: &pgconn.PgError{Code: "40001"}}}}
					if cancelar {
						tx.alRevertir = cancel
					}
					return tx
				}}
				obtenido, err := caso.ejecutar(ctx, &ConsultaMiBolsaPostgreSQL{pool: pool})
				esperado, intentos := caso.noDisponible, postgresql.IntentosMaximosCarreraSerializable
				if cancelar {
					esperado, intentos = context.Canceled, 1
				}
				if !errors.Is(err, esperado) || !reflect.DeepEqual(obtenido, caso.vacio) || len(pool.transacciones) != intentos {
					t.Fatalf("cierre: %v, intentos=%d", err, len(pool.transacciones))
				}
				for _, tx := range pool.transacciones {
					if tx.reversiones != 1 || tx.confirmaciones != 0 {
						t.Fatal("transacción abortada sin cerrar")
					}
				}
			})
		}
	}
}

func TestLecturasMiBolsaRevalidanDenegacionTrasCarrera(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		t.Run(caso.nombre, func(t *testing.T) {
			pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(n int) *txLecturaMiBolsaPrueba {
				codigo := "42501"
				if n == 0 {
					codigo = "40001"
				}
				return &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{error: &pgconn.PgError{Code: codigo}}}}
			}}
			obtenido, err := caso.ejecutar(context.Background(), &ConsultaMiBolsaPostgreSQL{pool: pool})
			if !errors.Is(err, caso.denegado) || !reflect.DeepEqual(obtenido, caso.vacio) || len(pool.transacciones) != 2 {
				t.Fatal("denegación viva no detuvo la recuperación")
			}
			if !reflect.DeepEqual(pool.transacciones[0].argumentosConsultas, pool.transacciones[1].argumentosConsultas) {
				t.Fatal("se reemplazó la concesión")
			}
			for _, tx := range pool.transacciones {
				if tx.confirmaciones != 0 || tx.reversiones != 1 {
					t.Fatal("se confirmó la lectura revocada")
				}
			}
		})
	}
}

func TestMiBolsaReintentaTambienLecturasSecundarias(t *testing.T) {
	instante := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	for _, variante := range []string{"portal", "contacto", "ofertas"} {
		for _, codigo := range []string{"40001", "40P01"} {
			t.Run(variante+"/"+codigo, func(t *testing.T) {
				s := bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_sintetico", ConsultadaEn: instante, Material: materialLecturaMiBolsaPrueba(t, instante)}
				switch variante {
				case "portal":
					s.ResultadosEfectivos = []string{"enviado"}
				case "contacto":
					s.LeerContacto = true
				case "ofertas":
					s.LeerOfertas = true
				}
				pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(n int) *txLecturaMiBolsaPrueba {
					tx := &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{
						{contenido: []byte(`{"consultada_en":"2026-09-30T10:00:00Z","participaciones":[]}`)},
						{contenido: []byte(`[]`)},
					}}
					if n == 0 {
						tx.filas[1].error = &pgconn.PgError{Code: codigo}
					}
					return tx
				}}
				_, err := (&ConsultaMiBolsaPostgreSQL{pool: pool}).ConsultarMiBolsa(context.Background(), s)
				if err != nil || len(pool.transacciones) != 2 {
					t.Fatalf("no recuperó lectura secundaria: %v", err)
				}
				a, b := pool.transacciones[0], pool.transacciones[1]
				if !reflect.DeepEqual(a.consultas, b.consultas) || !reflect.DeepEqual(a.argumentosConsultas, b.argumentosConsultas) || a.confirmaciones != 0 || b.confirmaciones != 1 {
					t.Fatal("no repitió autorización y todas las lecturas")
				}
			})
		}

	}
}

func TestLecturasMiBolsaDescartanJSONInvalidoSinReintento(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		t.Run(caso.nombre, func(t *testing.T) {
			pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(int) *txLecturaMiBolsaPrueba {
				return &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{contenido: []byte(`{"consultada_en":`)}}}
			}}
			obtenido, err := caso.ejecutar(context.Background(), &ConsultaMiBolsaPostgreSQL{pool: pool})
			if err == nil || !reflect.DeepEqual(obtenido, caso.vacio) || len(pool.transacciones) != 1 || pool.transacciones[0].confirmaciones != 0 || pool.transacciones[0].reversiones != 1 {
				t.Fatal("JSON inválido confirmado o repetido")
			}
		})
	}
}

func TestMiBolsaLecturasSecundariasConservanDenegacionSanitizada(t *testing.T) {
	instante := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	for _, variante := range []string{"portal", "contacto", "ofertas"} {
		t.Run(variante, func(t *testing.T) {
			s := bolsa.SolicitudConsultaMiBolsa{CandidatoRef: "can_sintetico", ConsultadaEn: instante, Material: materialLecturaMiBolsaPrueba(t, instante)}
			switch variante {
			case "portal":
				s.ResultadosEfectivos = []string{"enviado"}
			case "contacto":
				s.LeerContacto = true
			case "ofertas":
				s.LeerOfertas = true
			}
			pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(int) *txLecturaMiBolsaPrueba {
				return &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{
					{contenido: []byte(`{"consultada_en":"2026-09-30T10:00:00Z","participaciones":[]}`)},
					{error: &pgconn.PgError{Code: "42501", Message: "contenido privado"}},
				}}
			}}
			resultado, err := (&ConsultaMiBolsaPostgreSQL{pool: pool}).ConsultarMiBolsa(context.Background(), s)
			if !errors.Is(err, dominio.ErrAutorizacionDenegada) || !errors.Is(err, bolsa.ErrMaterialMiBolsaNoDisponible) || len(pool.transacciones) != 1 || !reflect.DeepEqual(resultado, bolsa.InstantaneaMiBolsa{}) {
				t.Fatal("denegación alterada o lectura repetida")
			}
			if strings.Contains(err.Error(), "contenido privado") || postgresql.EsCarreraSerializable(err) {
				t.Fatal("causa privada o marca de carrera expuesta")
			}
		})
	}
}

// La política común consulta Done al entrar en la espera. Este contexto
// cancela en ese punto para distinguirlo de una cancelación antes del bucle.
type contextoCanceladoAlEsperarMiBolsa struct {
	context.Context
	cancelar context.CancelFunc
	esperas  int
}

func (c *contextoCanceladoAlEsperarMiBolsa) Done() <-chan struct{} {
	c.esperas++
	c.cancelar()
	return c.Context.Done()
}

func TestLecturasMiBolsaCancelacionDuranteEspera(t *testing.T) {
	for _, caso := range casosLecturaMiBolsaPrueba(t) {
		t.Run(caso.nombre, func(t *testing.T) {
			base, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			ctx := &contextoCanceladoAlEsperarMiBolsa{Context: base, cancelar: cancelar}
			pool := &poolLecturaMiBolsaPrueba{t: t, preparar: func(int) *txLecturaMiBolsaPrueba {
				return &txLecturaMiBolsaPrueba{filas: []filaPanelPostgreSQLPrueba{{error: &pgconn.PgError{Code: "40001"}}}}
			}}
			resultado, err := caso.ejecutar(ctx, &ConsultaMiBolsaPostgreSQL{pool: pool})
			if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(resultado, caso.vacio) || ctx.esperas != 1 || len(pool.transacciones) != 1 || pool.transacciones[0].reversiones != 1 {
				t.Fatal("la espera cancelada inició otra transacción o devolvió datos")
			}
		})
	}
}
