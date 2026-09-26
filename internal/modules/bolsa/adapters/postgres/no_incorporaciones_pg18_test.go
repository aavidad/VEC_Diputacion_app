package postgres

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type catalogoBajaPG18Prueba struct{}

func (catalogoBajaPG18Prueba) ResolverSancion(_ context.Context, clave string, _ time.Time) (ports.ResolucionCatalogoSancion, error) {
	return ports.ResolucionCatalogoSancion{Consecuencia: ports.ConsecuenciaSancion{Clave: clave, Etiqueta: "Baja por no incorporarse tras aceptar el llamamiento",
		Efecto: "excluir", ReglaRef: "vec.bolsa.reglas:3:" + clave, Huella: strings.Repeat("c", 64)},
		Recurso: ports.PlazoSancion{UltimoDia: "2026-10-20", ReglaRef: "vec.bolsa.reglas:3:b24.consecuencias", Huella: strings.Repeat("d", 64)}}, nil
}

// Prueba de contrato Go↔SQL de la bandeja de Bolsa 000042. Solo dentro del
// ensayo desechable (probar_ct124_no_incorporacion_pg18.sh con
// VEC_CT124_GO=1), tras la prueba de CT que publica la no incorporación.
func TestBandejaNoIncorporacionesPostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_B42_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de la no incorporación")
	}
	contenido, err := os.ReadFile(os.Getenv("VEC_CT124NI_EVENTO"))
	if err != nil {
		t.Fatal(err)
	}
	var publicado struct {
		Contenido string    `json:"contenido"`
		Huella    string    `json:"huella"`
		Posicion  int64     `json:"posicion"`
		Creada    time.Time `json:"creada"`
	}
	if err := json.Unmarshal(contenido, &publicado); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	buzon, err := NuevoBuzonNoIncorporacionesPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	s, err := application.NuevoServicioRecepcionNoIncorporaciones(buzon, catalogoBajaPG18Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	if _, hay, err := s.Cursor(ctx); err != nil || hay {
		t.Fatalf("bandeja vacía: %v %v", hay, err)
	}
	r, err := s.Recibir(ctx, []byte(publicado.Contenido), publicado.Huella, publicado.Creada, publicado.Posicion)
	if err != nil || r.Reutilizado || r.Estado != "aplicada" || r.ParticipacionRef == "" {
		t.Fatalf("baja: %+v %v", r, err)
	}
	r2, err := s.Recibir(ctx, []byte(publicado.Contenido), publicado.Huella, publicado.Creada, publicado.Posicion)
	if err != nil || !r2.Reutilizado || r2.Estado != "aplicada" || r2.ParticipacionRef != r.ParticipacionRef {
		t.Fatalf("reentrega: %+v %v", r2, err)
	}
	cursor, hay, err := s.Cursor(ctx)
	if err != nil || !hay || cursor.Posicion != publicado.Posicion {
		t.Fatalf("cursor: %+v %v %v", cursor, hay, err)
	}
}
