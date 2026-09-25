package main

import _ "time/tzdata"

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "rellenar-vinculos-bolsa" {
		opciones := flag.NewFlagSet("rellenar-vinculos-bolsa", flag.ExitOnError)
		huella := opciones.String("huella", "", "SHA-256 del fichero importado")
		categoria := opciones.String("categoria", "", "categoría importada")
		aplicar := opciones.Bool("aplicar", false, "confirmar; por defecto se ensaya con ROLLBACK")
		if err := opciones.Parse(os.Args[2:]); err != nil || opciones.NArg() != 0 {
			log.Fatal("argumentos de relleno invalidos")
		}
		r, err := bootstrap.EjecutarRellenoVinculosBolsa(context.Background(), config.Load(), *huella, *categoria, *aplicar)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("relleno_vinculos aplicar=%t nuevos=%d existentes=%d", *aplicar, r.Nuevos, r.Existentes)
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "comprobar-dietas" {
		os.Exit(ejecutarComprobacionDietas(context.Background(), os.Args[2:], os.Stdout, os.Stderr, config.Load(), bootstrap.ComprobarArranqueDietasSoloLectura))
	}
	if len(os.Args) > 1 && os.Args[1] == "publicar-proyeccion-publica" {
		if err := ejecutarPublicacionProyeccionPublica(context.Background(), os.Args[2:], os.Stdout, config.Load(), publicarProyeccionPublicaPostgreSQL); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "constituir-bolsa" {
		a, err := leerArgumentosConstituirBolsa(os.Args[2:], os.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		cfg := config.Load()
		r, err := bootstrap.EjecutarConstitucionBolsa(context.Background(), cfg, bootstrap.SolicitudConstitucionBolsa{Fichero: a.fichero, Categoria: a.categoria})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("bolsa=%s version=%d instantanea=%s acta=%s reutilizada=%t confirmada_en=%s vinculos_nuevos=%d vinculos_existentes=%d sustituye_a=%s", r.BolsaRef, r.VersionBolsa, r.InstantaneaRef, r.ActaRef, r.Reutilizada, r.ConfirmadaEn.Format("2006-01-02T15:04:05Z07:00"), r.Vinculos.Nuevos, r.Vinculos.Existentes, describirSustituidas(r.SustituyeA))
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "importar-convoca" {
		a, err := leerArgumentosImportarConvoca(os.Args[2:], os.Stderr)
		if err != nil {
			log.Fatal(err)
		}
		cfg := config.Load()
		r, err := bootstrap.EjecutarImportacionConvoca(context.Background(), cfg, bootstrap.SolicitudImportacionConvoca{Fichero: a.fichero, Categoria: a.categoria, BolsaRef: a.bolsaRef})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("acta=%s huella=%s leidas=%d aceptadas=%d rechazadas=%d", r.Acta.ActaRef, r.Acta.HuellaFicheroSHA256, r.Acta.FilasLeidas, r.Acta.FilasAceptadas, r.Acta.FilasRechazadas)
		if r.Acta.FilasRechazadas > 0 && !a.admitirRechazos {
			os.Exit(2)
		}
		return
	}
	cfg := config.Load()
	srv, err := bootstrap.NewHTTPServerWithConfig(cfg)
	if err != nil {
		log.Fatalf("bootstrap server: %v", err)
	}

	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			log.Fatal("serve TLS: VEC_TLS_CERT_FILE and VEC_TLS_KEY_FILE must be configured together")
		}
		log.Printf("vec server listening with TLS on %s", srv.Addr)
		err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		log.Printf("vec server listening on %s", srv.Addr)
		err = srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}
