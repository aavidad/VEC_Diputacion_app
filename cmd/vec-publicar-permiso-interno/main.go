// Command vec-publicar-permiso-interno publishes the temporary CT read role
// for an already registered F1 profile. It is an explicit administrative step.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	var s bootstrap.SolicitudPermisoInternoCT
	flag.StringVar(&s.CuentaRef, "cuenta", "", "referencia de cuenta F1")
	flag.StringVar(&s.PrincipalRef, "persona", "", "referencia de persona F1")
	flag.StringVar(&s.PerfilRef, "perfil", "", "referencia de perfil F1 propio")
	flag.StringVar(&s.OrganizacionRef, "organizacion", "", "ámbito CT de organización autorizado")
	flag.StringVar(&s.OrganizacionCorporativaRef, "organizacion-corporativa", "", "organización org_ del vínculo corporativo F1")
	flag.StringVar(&s.PoliticaRef, "politica", "", "referencia de política temporal de Identidad")
	flag.StringVar(&s.PoliticaHuellaSHA256, "politica-huella", "", "huella SHA256 de la política temporal")
	flag.Parse()
	dsn := os.Getenv("VEC_PERMISO_INTERNO_DSN")
	if dsn == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "permiso interno CT: configuración administrativa incompleta")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "permiso interno CT: conexión no disponible")
		os.Exit(1)
	}
	defer pool.Close()
	if err = bootstrap.PublicarPermisoInternoCT(ctx, pool, s); err != nil {
		fmt.Fprintln(os.Stderr, "permiso interno CT: publicación denegada")
		os.Exit(1)
	}
	fmt.Println("permiso interno CT: publicación exacta verificada")
}
