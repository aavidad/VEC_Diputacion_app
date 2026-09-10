package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"reflect"
	"sync"
	"testing"
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

type raizRegistroTXPrueba func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (string, error)

func (f raizRegistroTXPrueba) ResolverSeguimientoIncorporacionV2(c context.Context, o ct.OrdenConfirmacionIncorporacionV2) (string, error) {
	return f(c, o)
}

type historiaRegistroTXPrueba func(context.Context, hist.Selector) (hist.Restauracion, error)

func (f historiaRegistroTXPrueba) Restaurar(c context.Context, s hist.Selector) (hist.Restauracion, error) {
	return f(c, s)
}

type acreditadorRegistroTXPrueba func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error)

func (f acreditadorRegistroTXPrueba) AcreditarRegistroPersonal(c context.Context, o ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
	return f(c, o)
}

type relojRegistroTXPrueba func() time.Time

func (f relojRegistroTXPrueba) Ahora() time.Time { return f() }

type mockRegistroTX struct {
	pgx.Tx
	t              *testing.T
	o              ct.OrdenConfirmacionIncorporacionV2
	now            time.Time
	wire           salidaTransporteRegistroV2
	decisionL      string
	pasos          []string
	fail           string
	hook           func(string)
	mutar          func(*salidaTransporteRegistroV2)
	params         []any
	rollbackSeguro bool
}

func (m *mockRegistroTX) paso(s string) {
	m.pasos = append(m.pasos, s)
	if m.hook != nil {
		m.hook(s)
	}
}
func (m *mockRegistroTX) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	m.paso("begin")
	if o != (pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}) {
		m.t.Fatal("opciones SQL")
	}
	if m.fail == "begin" {
		return m, errors.New("privado")
	}
	return m, nil
}
func (m *mockRegistroTX) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	m.paso("exec")
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		m.t.Fatal("settings")
	}
	if m.fail == "exec" {
		return pgconn.CommandTag{}, errors.New("privado")
	}
	return pgconn.CommandTag{}, nil
}
func (m *mockRegistroTX) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	m.paso("query")
	if q != consultaRegistroIncorporacionTXV2 || len(args) != 23 {
		m.t.Fatal("firma23")
	}
	if !reflect.DeepEqual(args, m.params) {
		m.t.Fatal("piezas23 distintas; no mostrar material")
	}
	return filaRegistroTX{m}
}

type filaRegistroTX struct{ m *mockRegistroTX }

func (f filaRegistroTX) Scan(dest ...any) error {
	m := f.m
	m.paso("scan")
	if m.fail == "scan" {
		return errors.New("privado")
	}
	w := m.wire
	w.Consumos.DecisionLecturaRef = m.decisionL
	if !w.Recuperado {
		w.Recibo.ConsumosOriginales = w.Consumos
		w.Historia.Recibo = w.Recibo
	}
	if m.mutar != nil {
		m.mutar(&w)
	}
	*dest[0].(*[]byte) = serialTransporte(m.t, w)
	return nil
}
func (m *mockRegistroTX) Commit(context.Context) error {
	m.paso("commit")
	if m.fail == "commit" {
		return errors.New("privado")
	}
	return nil
}
func (m *mockRegistroTX) Rollback(c context.Context) error {
	m.paso("rollback")
	d, ok := c.Deadline()
	m.rollbackSeguro = ok && c.Err() == nil && time.Until(d) > 0 && time.Until(d) <= 2*time.Second
	return nil
}
func setupRegistroTX(t *testing.T) (*mockRegistroTX, *TransaccionRegistroIncorporacionV2PostgreSQL, ct.AcreditacionPersonalIncorporacion) {
	t.Helper()
	o, _, now := fixtureTransporte(t)
	w, _ := wireTransporte(t, o, now)
	m := &mockRegistroTX{t: t, o: o, now: now, wire: w}
	reloj := relojRegistroTXPrueba(func() time.Time { return m.now })
	d, e := o.Material().Datos()
	registroV2Exigir(t, e)
	ac, e := ct.AcreditarPersonalParaIncorporacion(context.Background(), o, acreditadorRegistroTXPrueba(func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
		return d.Personal, nil
	}), reloj)
	registroV2Exigir(t, e)
	provider := proveedorTransporteV2(func(_ context.Context, mat lector.MaterialV2) (lector.AutorizacionV2, error) {
		m.paso("provider")
		if m.fail == "provider" {
			return lector.AutorizacionV2{}, errors.New("privado")
		}
		cx, e := mat.Contexto()
		registroV2Exigir(t, e)
		snapshot := instantaneaLectorV2(t, cx, mat.Selector().OrganizacionRef, mat.UnidadRef(), m.now)
		au := permisoFijoV2(t, solicitudLectorV2(t, mat, nil), cx, snapshot, m.now.Add(-time.Microsecond), "dec_99999999999999999999999999999999", lector.AudienciaV2)
		m.decisionL = au.Exportacion.ResumenCapacidad().DecisionRef()
		// Expectativa SQL independiente de OrdenL opaca: piezas derivadas de la
		// exportación nominal realmente emitida y del material CT original.
		material, e := m.o.Material().MaterialCanonico()
		registroV2Exigir(t, e)
		evidence, e := ct.CapturarEvidenciaOrdenOriginalIncorporacionV2(context.Background(), m.o)
		registroV2Exigir(t, e)
		pc, e := exportacionParametrosRegistroV2(m.o.Exportacion(), true)
		registroV2Exigir(t, e)
		pl, e := exportacionParametrosRegistroV2(au.Exportacion, false)
		registroV2Exigir(t, e)
		m.params = append([]any{material, m.wire.Recibo.SeguimientoRef}, pc...)
		m.params = append(m.params, pl...)
		m.params = append(m.params, serialTransporte(t, evidence))
		return au, nil
	})
	raiz := raizRegistroTXPrueba(func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (string, error) {
		m.paso("raiz")
		return w.Recibo.SeguimientoRef, nil
	})
	restaurador := historiaRegistroTXPrueba(func(context.Context, hist.Selector) (hist.Restauracion, error) {
		t.Fatal("original no requiere restaurar")
		return hist.Restauracion{}, ct.ErrRegistroIncorporacionV2
	})
	a, e := nuevaTransaccionRegistroIncorporacionV2PostgreSQL(m, provider, raiz, restaurador, reloj)
	registroV2Exigir(t, e)
	return m, a, ac
}
func TestRegistroIncorporacionV2TXNominal(t *testing.T) {
	m, a, ac := setupRegistroTX(t)
	r, e := a.RegistrarORecuperarIncorporacion(context.Background(), m.o, ac)
	if e != nil || r.ValidarPara(m.o, m.now) != nil || r.Recuperado || !reflect.DeepEqual(m.pasos, []string{"raiz", "provider", "begin", "exec", "query", "scan", "commit"}) {
		t.Fatal("nominal no commit", e)
	}
	r.Recibo.MaterialOriginalCanonico[0] ^= 1
	if r.Historia.ReciboOriginal().ValidarPara(m.o) != nil {
		t.Fatal("copias compartidas")
	}
}
func TestRegistroIncorporacionV2TXRechazos(t *testing.T) {
	for _, caso := range []string{"previa", "provider", "begin", "exec", "scan", "commit", "raiz", "lector", "cancel_provider", "cancel_scan", "cancel_ultimo_reloj", "caduca", "retroceso"} {
		t.Run(caso, func(t *testing.T) {
			m, a, ac := setupRegistroTX(t)
			m.fail = caso
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "previa":
				ac = ct.AcreditacionPersonalIncorporacion{}
			case "raiz":
				m.mutar = func(w *salidaTransporteRegistroV2) {
					w.Recibo.SeguimientoRef = registroV2Ref("otra")
					w.Historia.Recibo = w.Recibo
				}
			case "lector":
				m.mutar = func(w *salidaTransporteRegistroV2) {
					w.Consumos.DecisionLecturaRef = "dec_77777777777777777777777777777777"
					w.Recibo.ConsumosOriginales = w.Consumos
					w.Historia.Recibo = w.Recibo
				}
			case "cancel_provider":
				m.hook = func(p string) {
					if p == "provider" {
						cancel()
					}
				}
			case "cancel_scan":
				m.hook = func(p string) {
					if p == "scan" {
						cancel()
					}
				}
			case "cancel_ultimo_reloj":
				a.reloj = relojRegistroTXPrueba(func() time.Time {
					if len(m.pasos) > 0 && m.pasos[len(m.pasos)-1] == "commit" {
						cancel()
					}
					return m.now
				})
			case "caduca":
				m.hook = func(p string) {
					if p == "scan" {
						m.now = m.now.Add(time.Minute)
					}
				}
			case "retroceso":
				m.hook = func(p string) {
					if p == "scan" {
						m.now = m.now.Add(-time.Microsecond)
					}
				}
			}
			r, e := a.RegistrarORecuperarIncorporacion(ctx, m.o, ac)
			if e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) {
				t.Fatal("fallo entrega resultado")
			}
			if caso == "cancel_provider" || caso == "cancel_scan" || caso == "cancel_ultimo_reloj" {
				if e != context.Canceled {
					t.Fatal("cancelación no prioritaria")
				}
			}
			count := func(s string) int {
				n := 0
				for _, p := range m.pasos {
					if p == s {
						n++
					}
				}
				return n
			}
			if caso == "previa" && len(m.pasos) != 0 {
				t.Fatal("sin acreditación accedió dependencias")
			}
			if caso == "provider" || caso == "cancel_provider" {
				if count("begin") != 0 {
					t.Fatal("permiso inválido abrió TX")
				}
			}
			if count("begin") > 1 || count("query") > 1 || count("commit") > 1 {
				t.Fatal("retry")
			}
			if caso == "cancel_ultimo_reloj" {
				if count("commit") != 1 || count("rollback") != 0 {
					t.Fatal("commit confirmado revertido")
				}
			} else if count("begin") == 1 && !m.rollbackSeguro {
				t.Fatal("rollback ausente")
			}
		})
	}
}
func TestRegistroIncorporacionV2TXOperacionesIndependientes(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		m, a, ac := setupRegistroTX(t)
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := a.RegistrarORecuperarIncorporacion(context.Background(), m.o, ac)
			if e != nil || r.ValidarPara(m.o, m.now) != nil {
				t.Error("operación concurrente")
			}
		}()
	}
	wg.Wait()
}

func TestRegistroIncorporacionV2TXReplayPropietario(t *testing.T) {
	for _, cruce := range []bool{false, true} {
		m, a, _ := setupRegistroTX(t)
		original := m.o
		_, historico := wireTransporte(t, original, m.now)
		d, e := original.Material().Datos()
		registroV2Exigir(t, e)
		m.now = m.now.Add(24 * time.Hour)
		d.Contexto = registroV2_contextoAutorizacionAltaV3Prueba(t, m.now)
		v, e := d.Contexto.Vinculo.Datos()
		registroV2Exigir(t, e)
		d.SolicitudContexto = ct.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef}
		m.o = ordenCTTransporte(t, d, m.now)
		ac, e := ct.AcreditarPersonalParaIncorporacion(context.Background(), m.o, acreditadorRegistroTXPrueba(func(context.Context, ct.OrdenConfirmacionIncorporacionV2) (ct.RegistroPersonalEjercicio, error) {
			return d.Personal, nil
		}), a.reloj)
		registroV2Exigir(t, e)
		m.wire.Recuperado = true
		m.wire.Consumos = registroV2Consumos(t, m.o, m.now)
		llamadas := 0
		a.restaurador = historiaRegistroTXPrueba(func(_ context.Context, s hist.Selector) (hist.Restauracion, error) {
			llamadas++
			m.paso("historia")
			want := hist.Selector{ReciboRef: historico.Recibo.Transicion.ReciboRef, MaterialSHA256: historico.Recibo.MaterialOriginalSHA256, IntencionSHA256: historico.Recibo.IntencionSHA256}
			if s != want {
				t.Fatal("selector histórico no exacto")
			}
			if cruce {
				return hist.Restauracion{OrdenOriginal: m.o}, nil
			}
			return hist.Restauracion{OrdenOriginal: original, Historia: historico.Historia}, nil
		})
		r, e := a.RegistrarORecuperarIncorporacion(context.Background(), m.o, ac)
		if llamadas != 1 {
			t.Fatal("replay sin lector histórico")
		}
		if cruce {
			if e == nil || !reflect.DeepEqual(r, ct.ResultadoRegistroIncorporacionV2{}) || !m.rollbackSeguro {
				t.Fatal("historia ajena aceptada")
			}
			continue
		}
		if e != nil || !r.Recuperado || !igualRegistroTX(r.Recibo, historico.Recibo) || original.ValidarEn(m.now) == nil {
			t.Fatal("replay no conservó original", e)
		}
	}
}

type poolRegistroTXFuncion func(context.Context, pgx.TxOptions) (pgx.Tx, error)

func (f poolRegistroTXFuncion) BeginTx(c context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	return f(c, o)
}

type claveOperacionRegistroTX struct{}

func TestRegistroIncorporacionV2TXInstanciaCompartida(t *testing.T) {
	m1, a1, ac1 := setupRegistroTX(t)
	m2, a2, ac2 := setupRegistroTX(t)
	ms := []*mockRegistroTX{m1, m2}
	as := []*TransaccionRegistroIncorporacionV2PostgreSQL{a1, a2}
	acs := []ct.AcreditacionPersonalIncorporacion{ac1, ac2}
	indice := func(c context.Context) int { return c.Value(claveOperacionRegistroTX{}).(int) }
	a, e := nuevaTransaccionRegistroIncorporacionV2PostgreSQL(poolRegistroTXFuncion(func(c context.Context, o pgx.TxOptions) (pgx.Tx, error) { return ms[indice(c)].BeginTx(c, o) }), proveedorTransporteV2(func(c context.Context, m lector.MaterialV2) (lector.AutorizacionV2, error) {
		return as[indice(c)].proveedor.AutorizarLecturaIncorporacionV2(c, m)
	}), raizRegistroTXPrueba(func(c context.Context, o ct.OrdenConfirmacionIncorporacionV2) (string, error) {
		return as[indice(c)].raices.ResolverSeguimientoIncorporacionV2(c, o)
	}), a1.restaurador, a1.reloj)
	registroV2Exigir(t, e)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), claveOperacionRegistroTX{}, n)
			r, e := a.RegistrarORecuperarIncorporacion(ctx, ms[n].o, acs[n])
			if e != nil || r.ValidarPara(ms[n].o, ms[n].now) != nil {
				t.Error("estado cruzado entre operaciones")
			}
		}(i)
	}
	wg.Wait()
}
