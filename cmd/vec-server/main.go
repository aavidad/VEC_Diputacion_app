package main

import _ "time/tzdata"

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/shared/telemetria"
	"vec-diputacion-granada/internal/vec/domain"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == subcomandoComprobarSeparacion {
		if err := ejecutarComprobacionSeparacion(os.Args[2:], os.Stdout); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := comprobarSubcomandoEnPortal(os.Args, os.Getenv(config.EnvPortalProceso)); err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == subcomandoExportarSeudonimosPortalExterno {
		if err := ejecutarExportacionSeudonimos(os.Stdout, config.Load(), bootstrap.ExportarSeudonimosPortalExterno); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == subcomandoPrepararPortalExterno {
		if err := ejecutarPreparacionPortalExterno(context.Background(), os.Args[2:], os.Stdout, config.Load(), bootstrap.PrepararMaterialPortalExterno); err != nil {
			log.Fatal(err)
		}
		return
	}
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
		log.Printf("bolsa=%s version=%d instantanea=%s acta=%s reutilizada=%t confirmada_en=%s vinculos_nuevos=%d vinculos_existentes=%d sustituye_a=%s pendientes_revision=%s", r.BolsaRef, r.VersionBolsa, r.InstantaneaRef, r.ActaRef, r.Reutilizada, r.ConfirmadaEn.Format("2006-01-02T15:04:05Z07:00"), r.Vinculos.Nuevos, r.Vinculos.Existentes, describirSustituidas(r.SustituyeA), describirPendientesRevision(r.PendientesRevision))
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
	inicioComposicion := time.Now()
	inicioFase := inicioComposicion
	cfg := config.Load()
	registrarArranque := func(fase, resultado, causa string) {
		telemetria.RegistrarArranque(os.Stderr, telemetria.EventoArranque{
			Servicio: "vec-server", Superficie: superficieServidor(cfg), Entorno: entornoSupervision(),
			Fase: fase, Resultado: resultado, Causa: causa, Duracion: time.Since(inicioFase),
		})
	}
	// Paciencia de las comprobaciones de arranque con CPU escasa: solo amplía
	// plazos mientras se compone el servidor; un valor no válido no arranca.
	plazo, err := plazoarranque.Analizar(os.Getenv(envArranquePlazoPreflight))
	if err != nil {
		registrarFalloArranque(os.Stdout, domain.ComponenteIncidenciaComposicion, domain.EtapaIncidenciaConfiguracion)
		registrarArranque("configuracion", "fallida", "configuracion")
		log.Fatalf("bootstrap server: etapa=configuracion causa=configuracion variable=%s", envArranquePlazoPreflight)
	}
	if err := plazoarranque.Fijar(plazo); err != nil {
		registrarFalloArranque(os.Stdout, domain.ComponenteIncidenciaComposicion, domain.EtapaIncidenciaConfiguracion)
		registrarArranque("configuracion", "fallida", "configuracion")
		log.Fatalf("bootstrap server: etapa=configuracion causa=configuracion variable=%s", envArranquePlazoPreflight)
	}
	if plazo > 0 {
		log.Printf("arranque: plazo mínimo de las comprobaciones previas %s", plazo)
	}
	emisor, cerrarEmisor := crearEmisorServidor(os.Stdout, os.Stderr)
	srv, err := bootstrap.NuevoServidorHTTPSupervisado(cfg, emisor)
	// Compuesto (o fallido) el servidor, los plazos vuelven a ser los declarados.
	plazoarranque.Terminar()
	if err != nil {
		cerrarEmisor()
		registrarFalloArranque(os.Stdout, domain.ComponenteIncidenciaComposicion, domain.EtapaIncidenciaComposicion)
		causa := telemetria.ClaseErrorArranque(err)
		registrarArranque("composicion", "fallida", causa)
		log.Fatalf("bootstrap server: etapa=composicion causa=%s", causa)
	}
	// Registro de acceso técnico: una línea JSON por petición en stderr. Va
	// dentro de la supervisión, que aporta la correlación.
	telemetria.Montar(srv, telemetria.Opciones{
		Destino: os.Stderr, Servicio: "vec-server", Superficie: superficieServidor(cfg),
		Entorno: entornoSupervision(), Lenta: telemetria.UmbralLenta(os.Getenv),
		Consultas: telemetria.UmbralConsultas(os.Getenv), Diagnostico: os.Getenv("VEC_DIAGNOSTICO_ESCUCHA"),
	})
	cerrarSupervision := componerSupervisionServidor(srv, emisor, cerrarEmisor, os.Stderr)

	if cfg.TLSCertFile != "" || cfg.TLSKeyFile != "" {
		if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
			cerrarSupervision()
			registrarFalloArranque(os.Stdout, domain.ComponenteIncidenciaServidor, domain.EtapaIncidenciaConfiguracion)
			registrarArranque("configuracion", "fallida", "configuracion")
			log.Fatal("serve TLS: VEC_TLS_CERT_FILE and VEC_TLS_KEY_FILE must be configured together")
		}
		registrarArranque("composicion", "preparada", "")
		log.Printf("vec server listening with TLS on %s", srv.Addr)
		inicioFase = time.Now()
		err = srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
	} else {
		registrarArranque("composicion", "preparada", "")
		log.Printf("vec server listening on %s", srv.Addr)
		inicioFase = time.Now()
		err = srv.ListenAndServe()
	}
	cerrarSupervision()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		registrarFalloArranque(os.Stdout, domain.ComponenteIncidenciaServidor, domain.EtapaIncidenciaEscucha)
		causa := telemetria.ClaseErrorArranque(err)
		registrarArranque("escucha", "fallida", causa)
		log.Fatalf("serve: etapa=escucha causa=%s", causa)
	}
}

// envArranquePlazoPreflight fija, en segundos (1-600), el plazo mínimo de las
// comprobaciones previas del arranque. Vacía: los plazos declarados.
const envArranquePlazoPreflight = "VEC_ARRANQUE_PLAZO_PREFLIGHT"

// superficieServidor nombra el portal que atiende el proceso en el registro
// de acceso: "interno", "externo" o "integrada" si sirve ambos.
func superficieServidor(cfg config.Config) string {
	if cfg.PortalProceso == config.ValorPortalProcesoInterno || cfg.PortalProceso == config.ValorPortalProcesoExterno {
		return cfg.PortalProceso
	}
	return "integrada"
}
