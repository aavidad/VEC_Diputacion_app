package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Estas pruebas cotejan el contrato Go↔SQL (preimagen, proyección, cursor)
// contra la base desechable de probar_integracion_pg18.sh, cuyas fachadas AD3
// son recibos sintéticos. Sin VEC_DOCUMENTOS_PG18_DSN no se ejecutan. No
// acreditan COSE ni autorización real.
func dsnEnsayo(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("VEC_DOCUMENTOS_PG18_DSN")
	if dsn == "" {
		t.Skip("sin base PG18 desechable de Documentos")
	}
	return dsn
}

// materialSintetico lleva en la capacidad y la decisión los campos que cotejan
// las fachadas documentales; la estructura V3 solo es de forma.
func materialSintetico(t *testing.T, accion, recurso, finalidad, tipo string, campos []string, preimagen []byte, decisionRef string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	h := ports.HuellaEfectoV3(preimagen)
	capacidad, _ := json.Marshal(map[string]any{
		"audiencia_consumo": ports.AudienciaV3, "operacion": accion, "efecto_ref": recurso,
		"huella_efecto_sha256": h, "decision_ref": decisionRef, "relleno": strings.Repeat("r", 512),
	})
	decision, _ := json.Marshal(map[string]any{
		"accion": accion, "modulo_id": "documentos", "tipo_recurso": tipo, "finalidad": finalidad,
		"campos_permitidos": campos, "obligaciones": []string{}, "recurso_ref": recurso,
		"contexto_recurso_huella_sha256": h, "principal_id": "per:00000000-0000-4000-8000-0000000000a1",
		"perfil_activo_ref": "perfil:00000000-0000-4000-8000-0000000000a1",
		"correlacion_ref":   "corr:00000000-0000-4000-8000-0000000000a1", "decision_ref": decisionRef,
	})
	x := strings.Repeat("a", 64)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, x, x, "contexto:ensayo", x,
		accion, recurso, h, ports.AudienciaV3, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, _ := ed25519.GenerateKey(nil)
	spki, _ := x509.MarshalPKIXPublicKey(publica)
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, decision,
		[]byte("{}"), []byte("{}"), 1, 1, []byte("p"), []byte("s"), []byte("e"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func autorizacion(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion, finalidad, recurso, ambito string) ports.AutorizacionV3 {
	return ports.AutorizacionV3{Material: m, Accion: accion, Finalidad: finalidad, RecursoRef: recurso, AmbitoRef: ambito,
		PrincipalID:     "per:00000000-0000-4000-8000-0000000000a1",
		PerfilActivoRef: "perfil:00000000-0000-4000-8000-0000000000a1",
		CorrelacionRef:  "corr:00000000-0000-4000-8000-0000000000a1"}
}

func politicaEnsayo(t *testing.T, expediente, tipo string) vecports.ResultadoPoliticaConservacionDocumental {
	t.Helper()
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	s, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(ref("1"), ref("2"), tipo, expediente, ref("5"), 1,
		bytes.Repeat([]byte{0x6a}, 32), ref("6"), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	p, err := vecports.NuevaPoliticaConservacionDocumental(s, time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC),
		vecports.ProteccionPoliticaConservacionDocumentalOrdinaria, "", vecports.EstadoPoliticaConservacionDocumentalAprobada, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := vecports.NuevoResultadoPoliticaConservacionDocumental(p, s, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRepositorioPG18RegistraReferenciaExternaYLaListaConCustodia(t *testing.T) {
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsnEnsayo(t))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NuevoRepositorio(pool)
	if err != nil {
		t.Fatal(err)
	}
	expediente := "ref:" + strings.Repeat("a2", 32)
	tipo := "ref:" + strings.Repeat("a3", 32)
	alta := ports.AltaExternaPersistente{
		ID: "doc:00000000-0000-4000-8000-0000000000a4", ClaveIdempotencia: "idem:00000000-0000-4000-8000-0000000000a4",
		ModuloID: "dietas", ExpedienteRef: expediente, TipoRef: tipo, Version: 1,
		Custodia: domain.ReferenciaCustodiaExterna{CustodioID: "dietas", Referencia: "justificante:go:0001", HuellaSHA256: strings.Repeat("f", 64)},
		Politica: politicaEnsayo(t, expediente, tipo),
	}
	preimagen, err := alta.PreimagenExterna()
	if err != nil {
		t.Fatal(err)
	}
	alta.Autorizacion = autorizacion(materialSintetico(t, ports.AccionRegistrarExterno, alta.ID, "registrar_documento_externo",
		"documento_externo", []string{"documento", "recibo"}, preimagen, "decision:00000000-0000-4000-8000-0000000000a4"),
		ports.AccionRegistrarExterno, "registrar_documento_externo", alta.ID, expediente)
	d, err := repo.ConfirmarReferenciaExterna(ctx, alta)
	if err != nil || d.Custodia != domain.CustodiaExterna || d.CustodiaExternaRef != alta.Custodia || d.MIME != "" || d.Tamano != 0 || d.Descargable() {
		t.Fatalf("registro externo: %v %+v", err, d)
	}
	repetido, err := repo.ConfirmarReferenciaExterna(ctx, alta)
	if err != nil || repetido.NumeroVEC != d.NumeroVEC || !repetido.CreadoEn.Equal(d.CreadoEn) {
		t.Fatalf("replay externo: %v %+v", err, repetido)
	}
	// Misma clave con otra referencia: la fachada decide conflicto (23505),
	// que el adaptador traduce a ports.ErrConflicto, no a indisponibilidad.
	otra := alta
	otra.Custodia.Referencia = "justificante:go:0002"
	po, err := otra.PreimagenExterna()
	if err != nil {
		t.Fatal(err)
	}
	otra.Autorizacion = autorizacion(materialSintetico(t, ports.AccionRegistrarExterno, otra.ID, "registrar_documento_externo",
		"documento_externo", []string{"documento", "recibo"}, po, "decision:00000000-0000-4000-8000-0000000000a6"),
		ports.AccionRegistrarExterno, "registrar_documento_externo", otra.ID, expediente)
	if _, err := repo.ConfirmarReferenciaExterna(ctx, otra); !errors.Is(err, ports.ErrConflicto) {
		t.Fatalf("clave reutilizada: %v", err)
	}
	consulta := ports.ConsultaExpediente{ExpedienteRef: expediente, Limite: 10}
	pl, err := consulta.PreimagenListar()
	if err != nil {
		t.Fatal(err)
	}
	consulta.Autorizacion = autorizacion(materialSintetico(t, ports.AccionListar, expediente, "listar_documentos_expediente",
		"expediente_documental", []string{"items", "siguiente_cursor"}, pl, "decision:00000000-0000-4000-8000-0000000000a5"),
		ports.AccionListar, "listar_documentos_expediente", expediente, expediente)
	pagina, err := repo.ListarExpediente(ctx, consulta)
	if err != nil || len(pagina.Items) != 1 || pagina.Items[0].ID != alta.ID || pagina.Items[0].Custodia != domain.CustodiaExterna ||
		pagina.Items[0].CustodiaExternaRef != alta.Custodia || pagina.SiguienteCursor != "" {
		t.Fatalf("lista v2: %v %+v", err, pagina)
	}
}
