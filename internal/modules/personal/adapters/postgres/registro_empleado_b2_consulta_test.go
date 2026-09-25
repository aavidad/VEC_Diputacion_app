package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenFichaB2Prueba(t *testing.T) ports.OrdenFichaEmpleadoB2 {
	t.Helper()
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, err := domain.NuevaFechaCivil("2026-09-20")
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	m, err := domain.NuevoMaterialFichaEmpleadoB2(domain.SolicitudFichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("a", 24), OrganismoRef: "organismo:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: instante}, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionFichaEmpleadoB2, m.Recurso().Referencia, h, domain.AudienciaFichaEmpleadoB2, instante, instante.Add(3*time.Second))
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
	return ports.OrdenFichaEmpleadoB2{Material: m, Autorizacion: a}
}

func respuestaFichaB2Prueba(t *testing.T, o ports.OrdenFichaEmpleadoB2) []byte {
	t.Helper()
	r := ports.ResultadoFichaEmpleadoB2{
		Ficha: domain.FichaEmpleadoB2{
			EmpleadoRef: o.Material.EmpleadoRef(), OrganismoRef: o.Material.OrganismoRef(), PersonaRef: o.Material.Actor().PersonaRef, Corte: o.Material.Corte(), Version: 1,
			Relaciones: []domain.RelacionRegistroEmpleadoB2{}, Ocupaciones: []domain.OcupacionEmpleadoB2{}, Situaciones: []domain.SituacionEmpleadoB2{}, Servicios: []domain.ServicioReconocidoB2{},
		},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:prueba", DecisionRef: o.Autorizacion.ResumenCapacidad().DecisionRef(), EfectoRef: o.Material.EmpleadoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: o.Material.Corte().ConocidoEn.Add(time.Microsecond)},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRegistroEmpleadoB2ConsultaFichaNominal(t *testing.T) {
	o := ordenFichaB2Prueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaFichaB2Prueba(t, o)}}}
	r, err := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ConsultarFichaRRHH(context.Background(), o)
	if err != nil || resultado.Ficha.EmpleadoRef != o.Material.EmpleadoRef() || tx.commits != 1 || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.q[1] != consultaFichaEmpleadoB2SQL {
		t.Fatal("consulta de ficha no confirmada", err)
	}
}

func TestRegistroEmpleadoB2NoEnviaAtestacionAjena(t *testing.T) {
	o := ordenFichaB2Prueba(t)
	o.Autorizacion = ordenP(t).Autorizacion
	p := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(p)
	_, err := r.ConsultarFichaRRHH(context.Background(), o)
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) || p.n != 0 {
		t.Fatal("atestación ajena llegó a PostgreSQL", err)
	}
}

func TestRegistroEmpleadoB2FichaDeniegaOrganismoAjenoAntesDeSQL(t *testing.T) {
	o := ordenFichaB2Prueba(t)
	materialAjeno, err := domain.NuevoMaterialFichaEmpleadoB2(domain.SolicitudFichaEmpleadoB2{
		EmpleadoRef: o.Material.EmpleadoRef(), OrganismoRef: "organismo:ajeno", Corte: o.Material.Corte(), Actor: o.Material.Actor(),
	})
	if err != nil {
		t.Fatal(err)
	}
	o.Material = materialAjeno
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(pool)
	_, err = r.ConsultarFichaRRHH(context.Background(), o)
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) || pool.n != 0 {
		t.Fatal("atestación de otro organismo llegó a SQL", err)
	}
}

func TestRegistroEmpleadoB2RevierteFichaSinFormaExacta(t *testing.T) {
	o := ordenFichaB2Prueba(t)
	base := respuestaFichaB2Prueba(t, o)
	casos := map[string][]byte{
		"coleccion_nula":    bytes.Replace(base, []byte(`"relaciones":[]`), []byte(`"relaciones":null`), 1),
		"clave_desconocida": bytes.Replace(base, []byte(`"ficha":{`), []byte(`"ficha":{"intruso":true,`), 1),
		"clave_duplicada":   bytes.Replace(base, []byte(`"ficha":{`), []byte(`"ficha":{"version":1,"version":1,`), 1),
		"evidencia_ajena":   bytes.Replace(base, []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_ajena"`), 1),
		"organismo_ajeno":   bytes.Replace(base, []byte(`"organismo_ref":"organismo:dipgra"`), []byte(`"organismo_ref":"organismo:ajeno"`), 1),
		"eficacia_omitida":  bytes.Replace(base, []byte(`"eficacia_administrativa":false,`), nil, 1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
			_, err := r.ConsultarFichaRRHH(context.Background(), o)
			if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta alterada confirmada", err)
			}
		})
	}
}

func TestRegistroEmpleadoB2ExigeSnapshotDeCatalogoPublicado(t *testing.T) {
	o := ordenFichaB2Prueba(t)
	var respuesta ports.ResultadoFichaEmpleadoB2
	if err := json.Unmarshal(respuestaFichaB2Prueba(t, o), &respuesta); err != nil {
		t.Fatal(err)
	}
	desde, _ := domain.NuevaFechaCivil("2026-09-01")
	entrada := func(tipo, ref string) *domain.SnapshotEntradaCatalogoEmpleadoB2 {
		return &domain.SnapshotEntradaCatalogoEmpleadoB2{
			OrganismoRef: o.Material.OrganismoRef(), Tipo: tipo, Ref: ref, Version: 1, Revision: 1,
			Denominacion: "Entrada sintética", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: desde, Estado: "publicada",
		}
	}
	respuesta.Ficha.Relaciones = []domain.RelacionRegistroEmpleadoB2{{
		RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "uni:prueba", OrganismoRef: o.Material.OrganismoRef(),
		RegimenRef: "reg:funcionario", ModalidadRef: "mod:interino", Estado: "vigente",
		CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{
			Regimen: entrada("regimen", "reg:funcionario"), Modalidad: entrada("modalidad", "mod:interino"),
		},
		Traza: domain.TrazaEmpleadoB2{Desde: desde, RegistradaEn: o.Material.Corte().ConocidoEn.Add(-time.Hour), Version: 1, ActoRef: "acto:prueba", FuenteRef: "fuente:prueba", FuenteVersion: 1},
	}}
	valido, _ := json.Marshal(respuesta)
	if _, err := decodificarFichaEmpleadoB2(valido, o); err != nil {
		t.Fatal("snapshot publicado válido rechazado", err)
	}
	var bruto map[string]json.RawMessage
	if err := json.Unmarshal(valido, &bruto); err != nil {
		t.Fatal(err)
	}
	var ficha map[string]json.RawMessage
	if err := json.Unmarshal(bruto["ficha"], &ficha); err != nil {
		t.Fatal(err)
	}
	var relaciones []map[string]json.RawMessage
	if err := json.Unmarshal(ficha["relaciones"], &relaciones); err != nil {
		t.Fatal(err)
	}
	delete(relaciones[0], "catalogo_snapshot")
	ficha["relaciones"], _ = json.Marshal(relaciones)
	bruto["ficha"], _ = json.Marshal(ficha)
	sinSnapshot, _ := json.Marshal(bruto)
	if _, err := decodificarFichaEmpleadoB2(sinSnapshot, o); err == nil {
		t.Fatal("relación sin snapshot aceptada")
	}
	respuesta.Ficha.Relaciones[0].CatalogoSnapshot.Regimen.Estado = "retirada"
	retirada, _ := json.Marshal(respuesta)
	if _, err := decodificarFichaEmpleadoB2(retirada, o); err == nil {
		t.Fatal("snapshot retirado aceptado")
	}
}

func ordenVacantesB2Prueba(t *testing.T) ports.OrdenVacantesB2 {
	t.Helper()
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, _ := domain.NuevaFechaCivil("2026-09-20")
	instante := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	m, err := domain.NuevoMaterialVacantesB2(domain.SolicitudVacantesB2{OrganismoRef: "org:dipgra", Corte: domain.CorteEmpleadoB2{VigenteEn: fecha, ConocidoEn: instante}, Limite: 10, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	h, _ := m.HuellaSHA256()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionVacantesB2, m.Recurso().Referencia, h, domain.AudienciaVacantesB2, instante, instante.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenVacantesB2{Material: m, Autorizacion: a}
}

func TestRegistroEmpleadoB2ConsultaVacantesConCobertura(t *testing.T) {
	o := ordenVacantesB2Prueba(t)
	m := o.Material
	a := o.Autorizacion
	instante := m.Corte().ConocidoEn
	resultado := ports.ResultadoVacantesB2{
		Pagina:    domain.PaginaVacantesB2{OrganismoRef: m.OrganismoRef(), Corte: m.Corte(), Limite: m.Limite(), Cobertura: "completa", Vacantes: []domain.VacantePlazaB2{}},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:prueba", DecisionRef: a.ResumenCapacidad().DecisionRef(), EfectoRef: m.OrganismoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: instante.Add(time.Microsecond)},
	}
	bruto, _ := json.Marshal(resultado)
	tx := &txP{fila: filaP{vals: []any{bruto}}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	respuesta, err := r.ListarVacantesRRHH(context.Background(), o)
	if err != nil || respuesta.Pagina.Cobertura != "completa" || tx.commits != 1 || tx.q[1] != consultaVacantesB2SQL {
		t.Fatal("vacantes acreditadas no confirmadas", err)
	}
	alterada := bytes.Replace(bruto, []byte(`"cobertura":"completa"`), []byte(`"cobertura":"parcial"`), 1)
	tx2 := &txP{fila: filaP{vals: []any{alterada}}}
	r2, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx2})
	_, err = r2.ListarVacantesRRHH(context.Background(), o)
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) || tx2.commits != 0 || tx2.rollbacks != 1 {
		t.Fatal("cobertura parcial confirmada", err)
	}
}

func TestRegistroEmpleadoB2NoEncontradoNoFiltraDetalle(t *testing.T) {
	tx := &txP{errQ: &pgconn.PgError{Code: "P7404", Message: "empleado privado"}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	_, err := r.ConsultarFichaRRHH(context.Background(), ordenFichaB2Prueba(t))
	if !errors.Is(err, domain.ErrRegistroEmpleadoB2NoEncontrado) || strings.Contains(err.Error(), "privado") || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("ausencia nominal incorrecta", err)
	}
}

func TestRegistroEmpleadoB2CoberturaSQLDenegada(t *testing.T) {
	tx := &txP{errQ: &pgconn.PgError{Code: "P7401", Message: "detalle privado de cobertura"}}
	r, _ := nuevoRepositorioRegistroEmpleadoB2PostgreSQL(&poolP{tx: tx})
	_, err := r.ListarVacantesRRHH(context.Background(), ordenVacantesB2Prueba(t))
	if !errors.Is(err, domain.ErrCoberturaVacantesB2NoAcreditada) || strings.Contains(err.Error(), "privado") || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("cobertura no acreditada mal traducida", err)
	}
}
