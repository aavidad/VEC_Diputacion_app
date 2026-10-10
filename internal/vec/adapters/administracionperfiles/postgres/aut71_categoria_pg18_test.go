package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
)

// TestResolverRolAdministrableRealDevuelveCategoriaPostgreSQL18 lee el
// catálogo administrable con la función SQL real (AUT24 sustituida por AUT71),
// sin el doble de contrato_v2_test.go. Antes de AUT71 el resolver no traía
// categoria_admin y el adaptador devolvía «no disponible» (503 en B1).
//
// Sólo corre contra una base desechable: el runner aporta un LOGIN miembro de
// uno de los grupos con EXECUTE sobre el resolver y las versiones de rol que
// debe resolver. No escribe nada.
func TestResolverRolAdministrableRealDevuelveCategoriaPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUT71_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable con AUT71")
	}
	dsn, admin := os.Getenv("VEC_AUT71_PG_DSN"), os.Getenv("VEC_AUT71_ROL_ADMIN")
	if dsn == "" || admin == "" {
		t.Fatal("faltan VEC_AUT71_PG_DSN o VEC_AUT71_ROL_ADMIN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión: %v", err)
	}
	defer pool.Close()
	a := &Autoridad{pool: pool, emisor: &emisorFalso{}, reloj: relojFijo(time.Now())}
	rol, err := a.ResolverRolAdministrable(ctx, admin)
	if err != nil {
		t.Fatalf("ADMIN %s no resuelve con el catálogo real: %v", admin, err)
	}
	if rol.Clase != domain.ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "aplicacion" {
		t.Fatalf("ADMIN %s: clase=%q categoria=%q", admin, rol.Clase, rol.CategoriaAdmin)
	}
	if sistemas := os.Getenv("VEC_AUT71_ROL_SISTEMAS"); sistemas != "" {
		rol, err := a.ResolverRolAdministrable(ctx, sistemas)
		if err != nil || rol.Clase != domain.ClaseControlPerfilAdministrador || rol.CategoriaAdmin != "sistemas" {
			t.Fatalf("Sistemas %s: %+v %v", sistemas, rol, err)
		}
	}
	if ordinario := os.Getenv("VEC_AUT71_ROL_ORDINARIO"); ordinario != "" {
		rol, err := a.ResolverRolAdministrable(ctx, ordinario)
		if err != nil || rol.Clase != domain.ClaseControlPerfilOrdinario || rol.CategoriaAdmin != "" {
			t.Fatalf("ordinario %s: %+v %v", ordinario, rol, err)
		}
	}
}
