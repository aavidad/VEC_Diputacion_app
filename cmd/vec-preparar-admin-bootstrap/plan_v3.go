package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/domain"
)

type opcionesBootstrapV3 struct {
	fuente, destino, textos, conexion, aprobacion, recibo string
	cotejar, aplicar                                      bool
}

type documentoBootstrapV3 struct {
	Plan             domain.PlanBootstrapAdministracionV3 `json:"plan"`
	HuellaPlanSHA256 string                               `json:"huella_plan_sha256"`
}

type textosBootstrapV3 struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}

type diagnosticoBootstrapV3 struct {
	Codigo             string `json:"codigo"`
	Mensaje            string `json:"mensaje"`
	Limite             string `json:"limite"`
	Preparado          bool   `json:"preparado"`
	Cotejado           bool   `json:"cotejado"`
	AprobacionCotejada bool   `json:"aprobacion_cotejada"`
	Aplicado           bool   `json:"aplicado"`
	HuellaPlanSHA256   string `json:"huella_plan_sha256,omitempty"`
}

func ejecutarBootstrapV3(b []byte, o opcionesBootstrapV3, salida, errores io.Writer) int {
	textos, idioma, err := cargarTextosBootstrapV3(o.textos)
	if err != nil {
		return codigoErrorCatalogoBootstrap(err)
	}
	fallo := func(codigo, clave string) int {
		if emitirDiagnosticoBootstrapV3(errores, textos, idioma, diagnosticoBootstrapV3{Codigo: codigo}, clave) != nil {
			return 2
		}
		return 1
	}
	// Este corte no compone un proveedor de aplicación. Ni conexión ni aprobación
	// abren una base o producen un recibo; la preparación se realiza por separado.
	if o.aplicar {
		return fallo("provision_no_confirmada", "admin_gobierno_plan_limite")
	}
	if o.conexion != "" || o.recibo != "" || o.aprobacion != "" && !o.cotejar ||
		o.aprobacion != "" && (o.aprobacion == o.fuente || o.aprobacion == o.destino || o.aprobacion == o.textos) {
		return fallo("uso_invalido", "admin_comprobar_entrada_invalida")
	}
	var plan domain.PlanBootstrapAdministracionV3
	if decodificarEstricto(b, &plan) != nil || plan.Validar() != nil {
		return fallo("fuente_invalida", "admin_plan_gobierno_perfil_invalido")
	}
	_, huella, err := plan.CanonicoYHuella()
	if err != nil {
		return fallo("plan_invalido", "admin_plan_gobierno_perfil_invalido")
	}
	documento, err := json.Marshal(documentoBootstrapV3{Plan: plan, HuellaPlanSHA256: huella})
	if err != nil || len(documento)+1 > limiteDocumento {
		return fallo("plan_invalido", "admin_plan_gobierno_perfil_invalido")
	}
	documento = append(documento, '\n')
	defer clear(documento)
	if o.cotejar {
		previo, err := leerPrivado(o.destino)
		if err != nil {
			return fallo("plan_ausente_o_inseguro", "admin_comprobar_entrada_invalida")
		}
		defer clear(previo)
		if !bytes.Equal(previo, documento) {
			return fallo("plan_divergente", "admin_plan_gobierno_perfil_invalido")
		}
	} else if crearOComparar(o.destino, documento) != nil {
		return fallo("plan_divergente_o_destino_inseguro", "admin_comprobar_entrada_invalida")
	}
	aprobacionCotejada := false
	if o.aprobacion != "" {
		aprobacion, err := leerPrivado(o.aprobacion)
		if err != nil {
			return fallo("aprobacion_insegura", "admin_comprobar_entrada_invalida")
		}
		defer clear(aprobacion)
		var aprobada aprobacionPrivada
		if decodificarEstricto(aprobacion, &aprobada) != nil || !domain.HuellaAdministracionPerfilesValida(aprobada.HuellaPlanSHA256) ||
			aprobada.HuellaPlanSHA256 != huella {
			return fallo("aprobacion_divergente", "admin_plan_gobierno_perfil_invalido")
		}
		aprobacionCotejada = true
	}
	d := diagnosticoBootstrapV3{Codigo: "plan_preparado", Preparado: true, Cotejado: o.cotejar,
		AprobacionCotejada: aprobacionCotejada, HuellaPlanSHA256: huella}
	if emitirDiagnosticoBootstrapV3(errores, textos, idioma, d, "admin_gobierno_plan_preparado") != nil {
		return 2
	}
	if _, err := fmt.Fprintln(salida, huella); err != nil {
		return fallo("plan_salida_fallida", "admin_comprobar_entrada_invalida")
	}
	return 0
}

func emitirDiagnosticoBootstrapV3(w io.Writer, textos *i18n.Catalog, idioma string, d diagnosticoBootstrapV3, clave string) error {
	d.Mensaje = textos.T(idioma, clave)
	d.Limite = textos.T(idioma, "admin_comprobar_limite")
	return json.NewEncoder(w).Encode(d)
}

func cargarTextosBootstrapV3(ruta string) (*i18n.Catalog, string, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || len(b) > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	defer clear(b)
	var datos textosBootstrapV3
	if decodificarEstricto(b, &datos) != nil {
		return nil, "", os.ErrInvalid
	}
	textos, err := i18n.New(datos.Idioma, map[string]map[string]string{datos.Idioma: datos.Mensajes})
	if err != nil {
		return nil, "", err
	}
	for _, clave := range []string{"admin_comprobar_entrada_invalida", "admin_comprobar_limite", "admin_plan_gobierno_perfil_invalido", "admin_gobierno_plan_preparado", "admin_gobierno_plan_limite"} {
		if mensaje, ok := textos.Message(datos.Idioma, clave); !ok || mensaje == "" {
			return nil, "", os.ErrInvalid
		}
	}
	return textos, datos.Idioma, nil
}

// El catálogo solo traduce diagnósticos. La versión del JSON decide el formato.
func fallarEntradaCLI(w io.Writer, rutaTextos, codigo string) int {
	if rutaTextos == "" {
		return fallar(w, codigo)
	}
	textos, idioma, err := cargarTextosBootstrapV3(rutaTextos)
	if err != nil {
		return codigoErrorCatalogoBootstrap(err)
	}
	codigoMensaje := "admin_comprobar_entrada_invalida"
	if codigo == "fuente_invalida" || codigo == "plan_invalido" || codigo == "plan_divergente" {
		codigoMensaje = "admin_plan_gobierno_perfil_invalido"
	}
	if emitirDiagnosticoBootstrapV3(w, textos, idioma, diagnosticoBootstrapV3{Codigo: codigo}, codigoMensaje) != nil {
		return 2
	}
	return 1
}

// Sin catálogo, el fallo se propaga como código de proceso. No se inventa un
// mensaje ni se vuelca una ruta o el contenido del archivo recibido.
func codigoErrorCatalogoBootstrap(fallo error) int {
	if fallo == nil {
		return 0
	}
	return 2
}
