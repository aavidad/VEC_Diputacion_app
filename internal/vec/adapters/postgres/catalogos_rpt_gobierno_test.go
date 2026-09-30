package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func borradorGobiernoRPTPrueba() ports.BorradorPropuestaGobiernoCategoriaRPT {
	return ports.BorradorPropuestaGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:rpt:001", ReciboRef: "recibo:rpt:propuesta:001",
		Contenido: domain.ContenidoGobiernoCategoriaRPT{
			Accion:     domain.AccionGobiernoCategoriaRPTDeshabilitar,
			CatalogoID: descriptorRPTPrueba.CatalogoID, ModuloID: descriptorRPTPrueba.ModuloID,
			Version: 2, CategoriaID: cadenaGobiernoRPTPrueba("categoria.uno"),
			RevisionEsperada: enteroGobiernoRPTPrueba(1),
			PreimagenesControl: map[string]domain.PreimagenControlGobiernoCategoriaRPT{
				"categoria.uno": {Version: 2, HuellaSHA256: strings.Repeat("a", 64), Revision: 1, Estado: "habilitada"}},
			MotivoRef: "motivo:prueba", FuenteRef: "fuente:prueba",
		},
	}
}

func cadenaGobiernoRPTPrueba(s string) *string { return &s }
func enteroGobiernoRPTPrueba(n int64) *int64   { return &n }

func TestPreparacionGobiernoRPTDerivaHuellasEnPostgreSQLSinEfecto(t *testing.T) {
	b := borradorGobiernoRPTPrueba()
	tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
	inicio := &iniciadorLecturaRPTPrueba{tx: tx}
	g, err := nuevoGestorGobiernoCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba)
	if err != nil {
		t.Fatal(err)
	}
	p, err := g.PrepararPropuestaGobiernoCategoriaRPT(t.Context(), b)
	if err != nil || p.Material.PropuestaRef != b.PropuestaRef || p.Material.ReciboRef != b.ReciboRef ||
		p.Material.Contenido.PreimagenesHuellaSHA256 != tx.huellaMaterial || p.Material.HuellaSHA256 != tx.huellaMaterial ||
		p.Autorizable.Recurso.Atributos["material_sha256"] != tx.huellaMaterial ||
		p.Autorizable.HuellaPropuesta != tx.huellaMaterial ||
		p.Autorizable.Accion != ports.AccionProponerGobiernoCategoriaRPT ||
		p.Autorizable.Finalidad != ports.FinalidadGobiernoCategoriaRPT ||
		p.Autorizable.Audiencia != ports.AudienciaGobiernoCategoriaRPT ||
		!reflect.DeepEqual(p.Autorizable.Recurso.Ambitos, map[string]string{"catalogo_id": b.Contenido.CatalogoID, "modulo_id": b.Contenido.ModuloID}) ||
		inicio.opciones.AccessMode != pgx.ReadOnly || tx.consultasCanon != 3 || tx.fachadas != 0 || !tx.confirmada {
		t.Fatalf("preparacion no deriva material: %+v %v tx=%+v", p, err, tx)
	}
	if b.Contenido.PreimagenesHuellaSHA256 != "" {
		t.Fatal("mutó el borrador del llamador")
	}
	var material map[string]json.RawMessage
	if json.Unmarshal([]byte(tx.materialCanonico), &material) != nil || len(material) != 4 {
		t.Fatalf("material final no contiene cuatro claves: %s", tx.materialCanonico)
	}

	// Una huella aportada por el llamador no pasa a ser autoridad de preparación.
	b.Contenido.PreimagenesHuellaSHA256 = strings.Repeat("a", 64)
	inicio.llamadas = 0
	if _, err := g.PrepararPropuestaGobiernoCategoriaRPT(t.Context(), b); !errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido) || inicio.llamadas != 0 {
		t.Fatalf("preimagen externa llegó a PostgreSQL: %v", err)
	}
}

func TestPreparacionAvanceGobiernoRPTFijaPerfilYRevision(t *testing.T) {
	m := ports.MaterialAvanceGobiernoCategoriaRPT{
		PropuestaRef: "propuesta:rpt:001", HuellaSHA256: strings.Repeat("a", 64),
		ReciboRef: "recibo:rpt:aprobacion:001", RevisionEsperada: 2,
		CatalogoID: descriptorRPTPrueba.CatalogoID, ModuloID: descriptorRPTPrueba.ModuloID,
	}
	tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
	inicio := &iniciadorLecturaRPTPrueba{tx: tx}
	g, _ := nuevoGestorGobiernoCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba)
	p, err := g.PrepararAprobacionGobiernoCategoriaRPT(t.Context(), m)
	if err != nil || p.Accion != ports.AccionAprobarGobiernoCategoriaRPT ||
		p.Recurso.Tipo != ports.TipoRecursoGobiernoCategoriaRPT || p.Recurso.Referencia != m.PropuestaRef ||
		p.Recurso.Atributos["material_sha256"] != tx.huellaMaterial || tx.consultasCanon != 1 || tx.fachadas != 0 || !tx.confirmada {
		t.Fatalf("preparacion de segunda aprobacion: %+v %v", p, err)
	}
	m.RevisionEsperada = 3
	tx.confirmada = false
	p, err = g.PrepararConfirmacionGobiernoCategoriaRPT(t.Context(), m)
	if err != nil || p.Accion != ports.AccionConfirmarGobiernoCategoriaRPT || !tx.confirmada {
		t.Fatalf("preparacion de confirmacion: %+v %v", p, err)
	}
	m.ModuloID = "bolsa"
	inicio.llamadas = 0
	if _, err := g.PrepararConfirmacionGobiernoCategoriaRPT(t.Context(), m); !errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido) || inicio.llamadas != 0 {
		t.Fatalf("descriptor ajeno: %v", err)
	}
}

func TestGobiernoRPTLigaSolicitudCompletaADecisionCanonica(t *testing.T) {
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	canon, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(escenario.decision)
	if err != nil {
		t.Fatal(err)
	}
	var proyeccion struct {
		DecisionRef string `json:"decision_ref"`
	}
	if json.Unmarshal(canon, &proyeccion) != nil || decisionCoincideSolicitudGobiernoRPT(escenario.solicitud, canon, proyeccion.DecisionRef) != nil {
		t.Fatal("la solicitud A no coincide con su decisión propia")
	}
	if err := decisionCoincideSolicitudGobiernoRPT(escenario.solicitud, canon, "dec_otra"); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
		t.Fatal("decisión distinta admitida")
	}
	d, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionRegistroContextoActorV3PostgreSQLPrueba{valor: "correlacion_22222222222222222222222222222222"})
	if err != nil {
		t.Fatal(err)
	}
	d.Correlacion = correlacion
	b, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := decisionCoincideSolicitudGobiernoRPT(b, canon, proyeccion.DecisionRef); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
		t.Fatal("solicitud B con correlación distinta usó exportación A")
	}
	d, _ = escenario.solicitud.Datos()
	d.ReferenciaMotivo.EntradaClave = "motivo_22222222222222222222222222222222"
	b, err = domain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	if err := decisionCoincideSolicitudGobiernoRPT(b, canon, proyeccion.DecisionRef); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
		t.Fatal("solicitud B con motivo distinto usó exportación A")
	}
	duplicada := bytes.Replace(canon, []byte(`"decision_ref":"`), []byte(`"decision_ref":"`+proyeccion.DecisionRef+`","decision_ref":"`), 1)
	if err := decisionCoincideSolicitudGobiernoRPT(escenario.solicitud, duplicada, proyeccion.DecisionRef); !errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado) {
		t.Fatal("decisión con clave duplicada aceptada")
	}
}

func autorizacionGobiernoRPTReciboPrueba(t *testing.T, accion, ref string) ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rpt:gobierno:nueva", h, h, "contexto:rpt:gobierno", h,
		accion, ref, h, ports.AudienciaGobiernoCategoriaRPT, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	a, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen,
		[]byte("d"), []byte("m"), []byte("c"), 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestGobiernoRPTReciboReplayConservaEfectoYExigeEvidenciaNueva(t *testing.T) {
	ref, huella, recibo := "propuesta:rpt:001", strings.Repeat("b", 64), "recibo:rpt:confirmacion:001"
	a := autorizacionGobiernoRPTReciboPrueba(t, ports.AccionConfirmarGobiernoCategoriaRPT, ref)
	z := a.ResumenCapacidad()
	gobierno := estadoGobiernoRPTWire{PropuestaRef: ref, HuellaSHA256: huella, Revision: 4,
		Estado: domain.EstadoGobiernoCategoriaRPTConfirmada, ReciboRef: recibo,
		Accion: domain.AccionGobiernoCategoriaRPTDeshabilitar, Version: 2, RevisionCategoria: 2}
	wire := reciboGobiernoRPTWire{DecisionRef: z.DecisionRef(), EfectoRef: ref,
		HuellaEfectoSHA256: z.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("c", 64),
		AuditoriaRef: "auditoria:rpt:001", ConsumidaEn: z.EmitidaEn().Add(time.Second),
		ConsumoNuevo: true, ReciboRef: recibo, Gobierno: jsonRPTPrueba(t, gobierno)}
	r, err := decodificarReciboGobiernoRPT(jsonRPTPrueba(t, wire), a, ref, huella, recibo, 4,
		domain.EstadoGobiernoCategoriaRPTConfirmada, ports.AccionConfirmarGobiernoCategoriaRPT)
	if err != nil || r.ReciboRef != recibo || r.Evidencia.DecisionRef != z.DecisionRef() ||
		!r.Evidencia.ConsumoNuevo || r.RevisionCategoria != 2 {
		t.Fatalf("recibo de replay: %+v %v", r, err)
	}
	w := wire
	w.ConsumoNuevo = false
	if _, err := decodificarReciboGobiernoRPT(jsonRPTPrueba(t, w), a, ref, huella, recibo, 4,
		domain.EstadoGobiernoCategoriaRPTConfirmada, ports.AccionConfirmarGobiernoCategoriaRPT); !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoConfiable) {
		t.Fatalf("consumo histórico aceptado: %v", err)
	}
	w = wire
	w.ReciboRef = "recibo:otro"
	if _, err := decodificarReciboGobiernoRPT(jsonRPTPrueba(t, w), a, ref, huella, recibo, 4,
		domain.EstadoGobiernoCategoriaRPTConfirmada, ports.AccionConfirmarGobiernoCategoriaRPT); !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoConfiable) {
		t.Fatalf("recibo cambiado aceptado: %v", err)
	}
}

func TestGobiernoRPTFalloFachadaNoDeclaraExito(t *testing.T) {
	for _, codigo := range []string{"42501", "22023", "23505", "40001", "55000", "XX000"} {
		err := errorGobiernoRPT(context.Background(), &pgconn.PgError{Code: codigo})
		if err == nil {
			t.Fatalf("SQLSTATE %s se volvió éxito", codigo)
		}
	}
}
