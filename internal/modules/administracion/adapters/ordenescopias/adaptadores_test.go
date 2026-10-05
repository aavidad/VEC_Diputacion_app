package ordenescopias_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	destino "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias"
	"vec-diputacion-granada/internal/modules/administracion/adapters/ordenescopias/sintetico"
	fs "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/ordenescopias"
	operation "vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	register "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

func fixture(t *testing.T) domain.Orden {
	t.Helper()
	b, err := os.ReadFile("../../../../../cmd/vec-copias-orden/testdata/orden.sintetica.json")
	if err != nil {
		t.Fatal(err)
	}
	var e struct {
		Orden domain.Datos `json:"orden"`
	}
	if json.Unmarshal(b, &e) != nil {
		t.Fatal("fixture")
	}
	o, err := domain.Nueva(e.Orden)
	if err != nil {
		t.Fatal(err)
	}
	return o
}
func TestJWEConOrdenExactaYDiarioTrasReabrir(t *testing.T) {
	ctx := context.Background()
	o := fixture(t)
	d, _ := o.Datos()
	e, err := sintetico.Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	// Dummy fixture key, never real KMS material. Production obtains a dedicated
	// order protector from the existing provider in trusted composition.
	p, err := destino.NuevoProtectorJWE([32]byte{1}, "clave:orden:sintetica", "version:uno", 8192)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := adapter.NuevoProtectorOrden(p)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	external := filepath.Join(root, "control")
	restored := filepath.Join(root, "restaurado")
	for _, dir := range []string{external, restored} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	config := fs.Config{Directorio: external, RaicesRestauradas: []string{restored}, LimiteListado: 25}
	journal, err := fs.Abrir(config)
	if err != nil {
		t.Fatal(err)
	}
	declaration := register.Declaracion{Actor: "actor:sintetico", Correlacion: "correlacion:sintetica"}
	_, err = journal.Reservar(ctx, declaration, operation.Solicitud{Operacion: d.Operacion, Clave: "clave:sintetica", SHA256: d.SolicitudSHA256, Conjunto: d.Conjunto, Destino: d.Destino, Politica: d.Politica})
	if err != nil {
		t.Fatal(err)
	}
	externalAdapter, err := adapter.NuevoRegistroExterno(journal, declaration)
	if err != nil {
		t.Fatal(err)
	}
	service, err := app.Nuevo(e, auth, e, externalAdapter)
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := service.Publicar(ctx, o, d.EmitidaEn)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Verificar(ctx, sobre, d.EmitidaEn); err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), sobre.Firma...)
	changed[len(changed)/2] ^= 1
	if auth.Verificar(ctx, o, changed) == nil {
		t.Fatal("tamper accepted")
	}
	altered := d
	altered.Fence++
	another, _ := domain.Nueva(altered)
	if auth.Verificar(ctx, another, sobre.Firma) == nil {
		t.Fatal("AAD substitution")
	}
	first, err := service.Aceptar(ctx, sobre, d.EmitidaEn)
	if err != nil || first.Replay {
		t.Fatal(err, first)
	}
	journal, err = fs.Abrir(config)
	if err != nil {
		t.Fatal(err)
	}
	externalAdapter, err = adapter.NuevoRegistroExterno(journal, declaration)
	if err != nil {
		t.Fatal(err)
	}
	service, err = app.Nuevo(e, auth, e, externalAdapter)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Aceptar(ctx, sobre, d.EmitidaEn)
	if err != nil || !second.Replay || second.Recibo != first.Recibo {
		t.Fatal(err, first, second)
	}
}
func TestPlanSinCircularidad(t *testing.T) {
	o := fixture(t)
	plan, _ := o.PlanSHA256()
	full, _ := o.SHA256()
	d, _ := o.Datos()
	d.ConsumoV3 = strings.Repeat("9", 64)
	d.Outbox = "outbox:otro"
	d.DecisionSHA256 = d.PreimagenSHA256
	updated, err := domain.Nueva(d)
	if err != nil {
		t.Fatal(err)
	}
	otherPlan, _ := updated.PlanSHA256()
	otherFull, _ := updated.SHA256()
	if plan != otherPlan || full == otherFull {
		t.Fatal("receipt commitment circular or omitted")
	}
}
