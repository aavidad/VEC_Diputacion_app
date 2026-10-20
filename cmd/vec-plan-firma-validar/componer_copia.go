package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
)

//go:embed plantilla_plan_copia.json
var plantillaPlanCopia []byte

type configuracionPlanCopia struct {
	Version             int    `json:"version"`
	CopiaRef            string `json:"copia_ref"`
	CircuitoArchivo     string `json:"circuito_archivo"`
	CircuitoSHA         string `json:"circuito_archivo_sha256"`
	EntradasArchivo     string `json:"entradas_archivo"`
	EntradasSHA         string `json:"entradas_archivo_sha256"`
	FuenteRef           string `json:"fuente_ref"`
	CreadoPor           string `json:"creado_por"`
	PerfilCreador       string `json:"creado_perfil_ref"`
	PublicadoPor        string `json:"publicado_por"`
	PerfilPublicador    string `json:"publicado_perfil_ref"`
	AprobacionRef       string `json:"aprobacion_ref"`
	PreparacionRef      string `json:"preparacion_ref"`
	ClaveCrear          string `json:"clave_operacion_crear"`
	ClavePublicar       string `json:"clave_operacion_publicar"`
	CorrelacionCrear    string `json:"correlacion_crear"`
	CorrelacionPublicar string `json:"correlacion_publicar"`
}

func ejecutarComposicionCopia(args []string, salida io.Writer) int {
	if len(args) != 4 || args[0] != "--config" || args[2] != "--salida" || !filepath.IsAbs(args[3]) {
		return rechazar(errMaterial, "argumentos")
	}
	b, err := leerEntradaPreparacion(args[1], 16<<10)
	var cfg configuracionPlanCopia
	if err != nil || jsonUnico(b) != nil || decodificarEstricto(b, &cfg) != nil {
		return rechazar(errors.Join(errMaterial, err), "entrada")
	}
	var recibidas, esperadas map[string]json.RawMessage
	plantilla, err := json.Marshal(configuracionPlanCopia{})
	if err != nil || json.Unmarshal(plantilla, &esperadas) != nil || json.Unmarshal(b, &recibidas) != nil || len(recibidas) != len(esperadas) {
		return rechazar(errMaterial, "entrada")
	}
	for clave := range recibidas {
		if _, ok := esperadas[clave]; !ok {
			return rechazar(errMaterial, "entrada")
		}
	}
	crear, publicar, err := componerPlanCopia(cfg, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		return rechazar(err, "validacion")
	}
	if err := crearDirectorioPlan(args[3]); err != nil {
		return rechazar(err, "salida")
	}
	for _, bloque := range []struct {
		nombre string
		datos  []byte
	}{
		{"crear-catalogo.json", crear.Catalogo}, {"crear-material.json", crear.Material},
		{"crear-material.sha256", []byte(crear.Resumen.SHA256 + "\n")},
		{"publicar-catalogo.json", publicar.Catalogo}, {"publicar-material.json", publicar.Material},
		{"publicar-material.sha256", []byte(publicar.Resumen.SHA256 + "\n")},
	} {
		if err := conservarPreparacion(filepath.Join(args[3], bloque.nombre), bloque.datos, resumen{}, io.Discard); err != nil {
			return rechazar(err, "salida")
		}
	}
	if err := json.NewEncoder(salida).Encode(map[string]string{"estado": "preparado_sin_autorizacion",
		"crear_sha256": crear.Resumen.SHA256, "publicar_sha256": publicar.Resumen.SHA256}); err != nil {
		return rechazar(err, "salida")
	}
	return 0
}

func crearDirectorioPlan(ruta string) (err error) {
	padre, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, padre.Close()) }()
	info, err := padre.Stat(".")
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return errors.Join(errMaterial, err)
	}
	return padre.Mkdir(filepath.Base(ruta), 0700)
}

func componerPlanCopia(cfg configuracionPlanCopia, ahora time.Time) (materialCompuesto, materialCompuesto, error) {
	var cero materialCompuesto
	if cfg.Version != 1 || !clavePlan.MatchString(cfg.CopiaRef) || !strings.HasPrefix(cfg.PreparacionRef, "preparacion:") ||
		!shaValido.MatchString(cfg.EntradasSHA) || ahora.IsZero() {
		return cero, cero, errMaterial
	}
	b, err := leerEntradaPreparacion(cfg.EntradasArchivo, 2<<20)
	if err != nil || jsonUnico(b) != nil || huella(b) != cfg.EntradasSHA {
		return cero, cero, errors.Join(errMaterial, err)
	}
	var entradas []vd.EntradaCatalogoConfigurable
	if decodificarEstricto(b, &entradas) != nil {
		return cero, cero, errMaterial
	}
	var datos struct {
		Catalogo          vd.CatalogoConfigurable `json:"catalogo"`
		MotivoPublicacion string                  `json:"motivo_publicacion"`
		Pasos             []string                `json:"pasos"`
	}
	if json.Unmarshal(plantillaPlanCopia, &datos) != nil || len(entradas) != len(datos.Pasos) {
		return cero, cero, errMaterial
	}
	for i, e := range entradas {
		if e.Clave != datos.Pasos[i] {
			return cero, cero, errMaterial
		}
	}
	c := datos.Catalogo
	c.Entradas, c.FuenteRef, c.CreadoPor, c.CreadoEn = entradas, cfg.FuenteRef, cfg.CreadoPor, ahora
	return componerCopia(composicionCopia{Catalogo: c, CircuitoArchivo: cfg.CircuitoArchivo, CircuitoArchivoSHA256: cfg.CircuitoSHA,
		Publicador: cfg.PublicadoPor, PerfilCreador: cfg.PerfilCreador, PerfilPublicador: cfg.PerfilPublicador,
		AutorizacionCrear: cfg.PreparacionRef + ":crear", AutorizacionPublicar: cfg.PreparacionRef + ":publicar",
		CorrelacionCrear: cfg.CorrelacionCrear, CorrelacionPublicar: cfg.CorrelacionPublicar,
		ClaveCrear: cfg.ClaveCrear, ClavePublicar: cfg.ClavePublicar, AprobacionRef: cfg.AprobacionRef,
		MotivoPublicacion: datos.MotivoPublicacion, PublicadoEn: ahora})
}
