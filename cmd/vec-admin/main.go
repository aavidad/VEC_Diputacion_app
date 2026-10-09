package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/shared/telemetria"
)

func main() {
	inicioComposicion := time.Now()
	inicioFase := inicioComposicion
	registrarArranque := func(fase, resultado, causa string) {
		telemetria.RegistrarArranque(os.Stderr, telemetria.EventoArranque{
			Servicio: "vec-admin", Superficie: "administracion",
			Entorno: telemetria.Entorno(os.Getenv("VEC_ENTORNO"), os.Getenv("VEC_ADMIN_ENTORNO")),
			Fase:    fase, Resultado: resultado, Causa: causa, Duracion: time.Since(inicioFase),
		})
	}
	fallarConfiguracion := func(etapa string, causa error) {
		registrarArranque("configuracion", "fallida", "configuracion")
		if causa == nil {
			log.Fatal(errorArranque(etapa))
		}
		log.Fatalf("%s: %s", errorArranque(etapa), telemetria.MensajeErrorArranque(causa))
	}
	retirada, err := time.Parse(time.RFC3339, os.Getenv("VEC_ADMIN_RETIRADA_EN"))
	if err != nil {
		fallarConfiguracion("retirada", err)
	}
	configServidor := administracion.Configuracion{
		Entorno:             os.Getenv("VEC_ADMIN_ENTORNO"),
		Escucha:             os.Getenv("VEC_ADMIN_ESCUCHA"),
		Host:                os.Getenv("VEC_ADMIN_HOST"),
		Audiencia:           os.Getenv("VEC_ADMIN_AUDIENCIA"),
		EmisorIdentidad:     os.Getenv("VEC_ADMIN_EMISOR_IDENTIDAD"),
		CertificadoServidor: os.Getenv("VEC_ADMIN_TLS_CERT_FILE"),
		ClaveServidor:       os.Getenv("VEC_ADMIN_TLS_KEY_FILE"),
		CAAdministracion:    os.Getenv("VEC_ADMIN_CA_FILE"),
		CRLAdministracion:   os.Getenv("VEC_ADMIN_CRL_FILE"),
		RedesPermitidas:     strings.Split(os.Getenv("VEC_ADMIN_REDES_PERMITIDAS"), ","),
		RetiradaEn:          retirada,
	}
	privada, err := cargarConfiguracionPerfilesPrivada(os.Getenv("VEC_ADMIN_PERFILES_CONFIG_FILE"))
	if err != nil {
		fallarConfiguracion("perfiles_config", err)
	}
	runtime, err := cargarConfiguracionRuntimeADMIN(os.Getenv("VEC_ADMIN_RUNTIME_CONFIG_FILE"), privada)
	if err != nil {
		fallarConfiguracion("runtime_config", err)
	}
	var servidor *http.Server
	var cerrar func()
	if rutaUsuarios := os.Getenv("VEC_ADMIN_USUARIOS_CONFIG_FILE"); rutaUsuarios != "" {
		// El modo lo fija el archivo privado cerrado; una configuración inválida
		// nunca cae al arranque heredado de perfiles.
		usuarios, errorConfig := cargarConfiguracionUsuariosMetadatosPrivada(rutaUsuarios, privada)
		if errorConfig != nil {
			fallarConfiguracion("usuarios_config", errorConfig)
		}
		// El lote sólo existe junto a las lecturas de usuarios y con su propio
		// archivo privado; sin él, el proceso no abre ninguna escritura.
		var lote *configuracionLotePrivada
		if rutaLote := os.Getenv("VEC_ADMIN_LOTE_CONFIG_FILE"); rutaLote != "" {
			c, errorLote := cargarConfiguracionLotePrivada(rutaLote, privada, usuarios, runtime)
			if errorLote != nil {
				fallarConfiguracion("lote_config", errorLote)
			}
			lote = &c
		}
		// El gobierno del plan de firma sólo existe junto a las lecturas de
		// usuarios y con su propio archivo privado.
		var plan *configuracionPlanFirmaPrivada
		if rutaPlan := os.Getenv("VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE"); rutaPlan != "" {
			c, errorPlan := cargarConfiguracionPlanFirmaPrivada(rutaPlan, lote, privada, usuarios, runtime)
			if errorPlan != nil {
				fallarConfiguracion("plan_firma_config", errorPlan)
			}
			plan = &c
		}
		// Los efectos nominales (cargos competenciales, certificados
		// nominales), igual: sólo con las lecturas de usuarios y cada uno con
		// su propio archivo privado.
		var efectos []efectoConfigurado
		for _, e := range efectosADMIN() {
			ruta := os.Getenv(e.variable)
			if ruta == "" {
				continue
			}
			otros := make([]configuracionEfectoPrivada, 0, len(efectos))
			for _, o := range efectos {
				otros = append(otros, o.cfg)
			}
			c, errorEfecto := cargarConfiguracionEfectoPrivada(ruta, e.audiencia, otros, lote, plan, privada, usuarios, runtime)
			if errorEfecto != nil {
				fallarConfiguracion(e.nombre+"_config", errorEfecto)
			}
			efectos = append(efectos, efectoConfigurado{efectoADMIN: e, cfg: c})
		}
		var gobierno *configuracionGobiernoRolesPrivada
		if rutaGobierno := os.Getenv("VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE"); rutaGobierno != "" {
			c, errorGobierno := cargarConfiguracionGobiernoRolesPrivada(rutaGobierno, privada, usuarios, runtime, lote, plan, efectos)
			if errorGobierno != nil {
				fallarConfiguracion("gobierno_roles_config", errorGobierno)
			}
			gobierno = &c
		}
		var inscripcion *configuracionGobiernoInscripcionPrivada
		if rutaInscripcion := os.Getenv("VEC_ADMIN_INSCRIPCION_GOBIERNO_CONFIG_FILE"); rutaInscripcion != "" {
			c, errorInscripcion := cargarConfiguracionGobiernoInscripcionPrivada(rutaInscripcion, privada, usuarios, runtime, lote, plan, efectos, gobierno)
			if errorInscripcion != nil {
				fallarConfiguracion("inscripcion_gobierno_config", errorInscripcion)
			}
			inscripcion = &c
		}
		servidor, cerrar, err = componerProcesoUsuariosMetadatosADMINConGobierno(configServidor, privada, usuarios, runtime, lote, plan, efectos, gobierno, inscripcion, fuenteCatalogoGobiernoOficial)
	} else if os.Getenv("VEC_ADMIN_LOTE_CONFIG_FILE") != "" || os.Getenv("VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE") != "" ||
		os.Getenv("VEC_ADMIN_CARGOS_CONFIG_FILE") != "" || os.Getenv("VEC_ADMIN_CERTIFICADOS_CONFIG_FILE") != "" ||
		os.Getenv("VEC_ADMIN_GOBIERNO_ROLES_CONFIG_FILE") != "" ||
		os.Getenv("VEC_ADMIN_INSCRIPCION_GOBIERNO_CONFIG_FILE") != "" {
		fallarConfiguracion("lote_sin_usuarios", nil)
	} else {
		servidor, cerrar, err = componerProcesoADMINConRuntime(configServidor, privada, runtime)
	}
	if err != nil {
		if cerrar != nil {
			cerrar()
		}
		causa := telemetria.ClaseErrorArranque(err)
		if errors.Is(err, administracion.ErrConfiguracion) {
			causa = "configuracion"
		}
		registrarArranque("composicion", "fallida", causa)
		log.Fatalf("%s: %s", errorArranque(etapaComposicionADMIN(err)), telemetria.MensajeErrorArranque(err))
	}
	defer cerrar()
	telemetria.Montar(servidor, telemetria.Opciones{
		Destino: os.Stderr, Servicio: "vec-admin", Superficie: "administracion",
		Entorno: telemetria.Entorno(os.Getenv("VEC_ENTORNO"), configServidor.Entorno), Lenta: telemetria.UmbralLenta(os.Getenv),
		Consultas: telemetria.UmbralConsultas(os.Getenv), Diagnostico: os.Getenv("VEC_DIAGNOSTICO_ESCUCHA"),
	})
	registrarArranque("composicion", "preparada", "")
	inicioFase = time.Now()
	if err := servidor.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		cerrar()
		registrarArranque("escucha", "fallida", telemetria.ClaseErrorArranque(err))
		log.Fatalf("%s: %s", errorArranque("escucha"), telemetria.MensajeErrorArranque(err))
	}
}

// errorArranque conserva ErrConfiguracion y añade la etapa que falló. Es un
// código fijo del programa: nunca lleva rutas, valores de configuración,
// mensajes de PostgreSQL ni secretos, sólo dice al operador dónde mirar.
func errorArranque(etapa string) error {
	return fmt.Errorf("%w: etapa=%s", administracion.ErrConfiguracion, etapa)
}

// etapaComposicionADMIN conserva la etapa fija de los compositores antiguos.
// El texto completo del error nunca sale al registro: podría contener rutas.
func etapaComposicionADMIN(err error) string {
	if !errors.Is(err, administracion.ErrConfiguracion) {
		return "composicion"
	}
	const marcador = ": etapa="
	texto := err.Error()
	indice := strings.LastIndex(texto, marcador)
	if indice < 0 {
		return "composicion"
	}
	etapa := texto[indice+len(marcador):]
	if !etapaComposicionADMINPermitida(etapa) {
		return "composicion"
	}
	return etapa
}

// Lista positiva de las etapas emitidas por los dos compositores ADMIN y sus
// piezas de lote, plan de firma y efectos nominales. Los índices de pool van
// de 0 a 14 porque esa es la capacidad máxima del montaje actual.
func etapaComposicionADMINPermitida(etapa string) bool {
	switch etapa {
	case "emisor_identidad", "configuracion", "lote_configuracion", "plan_firma_configuracion", "gobierno_roles_config", "gobierno_roles_configuracion", "gobierno_roles_fuente",
		"gobierno_roles_pool", "gobierno_roles_confianza_metadatos", "gobierno_roles_confianza_material", "gobierno_roles_servicio",
		"inscripcion_gobierno_config", "inscripcion_gobierno_configuracion", "inscripcion_gobierno_fuente",
		"inscripcion_gobierno_pool", "inscripcion_gobierno_confianza_metadatos", "inscripcion_gobierno_confianza_material", "inscripcion_gobierno_servicio",
		"firmante_publica", "firmante", "confianza_metadatos", "confianza_material", "confianza_cadena",
		"emisor_usuarios", "auditoria_intentos", "auditoria_nominal", "frontera_tecnica", "auditor_compuesto",
		"lector_usuarios", "lecturas_usuarios", "selector", "seudonimos", "identificadores", "servidor",
		"auditor_frontera", "lote_confianza_metadatos", "lote_confianza_material", "lote_confianza_cadena",
		"lote_emisor", "lote_ambitos", "lote_autoridad", "lote_servicio", "plan_firma_pool",
		"plan_firma_confianza_metadatos", "plan_firma_confianza_material", "plan_firma_confianza_cadena",
		"plan_firma_emisor", "plan_firma_autoridad", "plan_firma_servicio":
		return true
	}
	for _, nombre := range []string{"cargos", "certificados"} {
		for _, sufijo := range []string{"_configuracion", "_pool", "_zona_horaria", "_confianza_metadatos",
			"_confianza_material", "_confianza_cadena", "_emisor", "_ejecutor", "_servicio"} {
			if etapa == nombre+sufijo {
				return true
			}
		}
	}
	for i := 0; i < 17; i++ {
		prefijo := "pool_" + strconv.Itoa(i)
		for _, sufijo := range []string{"_dsn", "_config", "_abrir", "_login"} {
			if etapa == prefijo+sufijo {
				return true
			}
		}
	}
	for _, i := range []int{0, 1, 2, 6, 7, 8, 9, 11} {
		if etapa == "pool_"+strconv.Itoa(i)+"_grupo" {
			return true
		}
	}
	return false
}
