package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenVinculoCRN11Prueba(t *testing.T) ports.OrdenVinculoPropioCRN11 {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	actor := base.Material.Actor()
	m, err := domain.NuevoMaterialVinculoPropioCRN11(domain.SolicitudVinculoPropioCRN11{Actor: actor, EmpleadoRef: base.Material.EmpleadoRef()})
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_crn11", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionVinculoPropioCRN11, m.EmpleadoRef(), h, domain.AudienciaVinculoPropioCRN11, actor.ResueltoEn, actor.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenVinculoPropioCRN11{Material: m, Autorizacion: a}
}
func respuestaVinculoCRN11Prueba(t *testing.T, o ports.OrdenVinculoPropioCRN11) []byte {
	t.Helper()
	m := o.Material
	x := o.Autorizacion.ResumenCapacidad()
	b, err := json.Marshal(ports.ResultadoVinculoPropioCRN11{Vinculo: domain.VinculoHistoricoCRN11{PersonaRef: m.Actor().PersonaRef, EmpleadoRef: m.EmpleadoRef(), VinculoRef: m.VinculoRef(), FuenteRef: "prc_" + strings.Repeat("f", 24), Version: m.VinculoVersion()}, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:personal:crn11:1", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:personal:1", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestVinculoCRN11PostgresConsultaNominalYMinima(t *testing.T) {
	o := ordenVinculoCRN11Prueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaVinculoCRN11Prueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, err := nuevoRepositorioVinculoCRN11(pool)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ConsultarVinculoPropioCRN11(context.Background(), o)
	if err != nil || resultado.Vinculo.FuenteRef != "prc_"+strings.Repeat("f", 24) || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || tx.commits != 1 || tx.rollbacks != 0 || tx.q[1] != consultaVinculoPropioCRN11SQL || len(tx.a[0]) != 11 || tx.a[0][0] != string(o.Material.Canonico()) {
		t.Fatal("lectura nominal no confirmada", err)
	}
}
func TestVinculoCRN11PostgresNoConsumePermisoFichaORRHH(t *testing.T) {
	o := ordenVinculoCRN11Prueba(t)
	o.Autorizacion = ordenFichaPropiaPrueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioVinculoCRN11(pool)
	if _, err := r.ConsultarVinculoPropioCRN11(context.Background(), o); !errors.Is(err, domain.ErrVinculoCRN11Invalido) || pool.n != 0 {
		t.Fatal("permiso prestado llegó a SQL", err)
	}
}
func TestVinculoCRN11PostgresRevierteSinHistoriaExacta(t *testing.T) {
	o := ordenVinculoCRN11Prueba(t)
	base := respuestaVinculoCRN11Prueba(t, o)
	casos := map[string][]byte{
		"otra_persona":    bytes.Replace(base, []byte(o.Material.Actor().PersonaRef), []byte("per_"+strings.Repeat("x", 24)), 1),
		"otra_proyeccion": bytes.Replace(base, []byte(o.Material.VinculoRef()), []byte("pep_"+strings.Repeat("x", 24)), 1),
		"otra_version":    bytes.Replace(base, []byte(`"version":1`), []byte(`"version":2`), 1),
		"sin_fuente":      bytes.Replace(base, []byte(`"fuente_ref":"prc_`+strings.Repeat("f", 24)+`",`), nil, 1),
		"fuente_nula":     bytes.Replace(base, []byte(`"fuente_ref":"prc_`+strings.Repeat("f", 24)+`"`), []byte(`"fuente_ref":null`), 1),
		"amplia_ficha":    bytes.Replace(base, []byte(`"vinculo":{`), []byte(`"vinculo":{"relaciones":[],`), 1),
		"duplicada":       bytes.Replace(base, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1),
		"sin_auditoria":   bytes.Replace(base, []byte(`"auditoria_ref":"auditoria:personal:1",`), nil, 1),
		"decision_ajena":  bytes.Replace(base, []byte(`"decision_ref":"dec_crn11"`), []byte(`"decision_ref":"dec_ajena"`), 1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioVinculoCRN11(&poolP{tx: tx})
			resultado, err := r.ConsultarVinculoPropioCRN11(context.Background(), o)
			if !errors.Is(err, domain.ErrVinculoCRN11NoDisponible) || resultado != (ports.ResultadoVinculoPropioCRN11{}) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta divergente confirmada", err)
			}
		})
	}
}
func TestVinculoCRN11PostgresCierraRevocacionAusenciaYCommitFallido(t *testing.T) {
	for _, caso := range []struct {
		nombre, codigo string
		commit         bool
		esperado       error
	}{
		{"revocada", "42501", false, domain.ErrVinculoCRN11Denegado}, {"sin_funcion", "42883", false, domain.ErrVinculoCRN11NoDisponible}, {"ambiguo", "P0002", false, domain.ErrVinculoCRN11Denegado}, {"commit", "40001", true, domain.ErrVinculoCRN11NoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			o := ordenVinculoCRN11Prueba(t)
			tx := &txP{fila: filaP{vals: []any{respuestaVinculoCRN11Prueba(t, o)}}}
			pgerr := &pgconn.PgError{Code: caso.codigo, Message: "detalle privado"}
			if caso.commit {
				tx.errC = pgerr
			} else {
				tx.errQ = pgerr
			}
			r, _ := nuevoRepositorioVinculoCRN11(&poolP{tx: tx})
			resultado, err := r.ConsultarVinculoPropioCRN11(context.Background(), o)
			if !errors.Is(err, caso.esperado) || resultado != (ports.ResultadoVinculoPropioCRN11{}) || tx.rollbacks != 1 || strings.Contains(err.Error(), "privado") {
				t.Fatal("fallo no cerrado", err)
			}
		})
	}
}
