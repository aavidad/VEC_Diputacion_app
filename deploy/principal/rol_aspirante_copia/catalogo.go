package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	adminpg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	dominio "vec-diputacion-granada/internal/vec/domain"
)

const (
	esquemaPlan    = "vec.admin.catalogo-acciones.plan.v2"
	esquemaPaquete = "vec.admin.catalogo-acciones.paquete.v2"
)

// Misma forma que el paquete v2 del adaptador; ValidarPaqueteCatalogoAccionesV2
// vuelve a decodificarlo de forma estricta y coteja cada byte.
type fuentePaquete struct {
	ModuloID      string                                  `json:"modulo_id"`
	Referencia    string                                  `json:"referencia"`
	Version       int                                     `json:"version"`
	HuellaSHA256  string                                  `json:"huella_sha256"`
	EntradasCanon string                                  `json:"entradas_canon"`
	Entradas      []dominio.EntradaAccionAdministracionV1 `json:"entradas"`
}

type paquete struct {
	Esquema    string                                    `json:"esquema"`
	Referencia string                                    `json:"referencia"`
	Version    int                                       `json:"version"`
	Fuentes    []fuentePaquete                           `json:"fuentes"`
	Perfiles   []dominio.PerfilPublicadoAdministracionV1 `json:"perfiles"`
}

// perfilActual es la cabeza de un rol leída de la autoridad central: el
// documento normalizado por normalizar_rol_catalogo_acciones_admin_v1 y el
// documento de su control vigente, tal como los guarda PostgreSQL.
type perfilActual struct {
	Rol     json.RawMessage `json:"rol"`
	Control json.RawMessage `json:"control"`
}

type entradaCatalogo struct {
	PaqueteCanon     string            `json:"paquete_canon"`
	CatalogoCanon    string            `json:"catalogo_canon"`
	CatalogoSHA256   string            `json:"catalogo_sha256"`
	Fuentes          []json.RawMessage `json:"fuentes"`
	Perfiles         []perfilActual    `json:"perfiles"`
	OperacionRef     string            `json:"operacion_ref"`
	AprobacionRef    string            `json:"aprobacion_ref"`
	AprobacionSHA256 string            `json:"aprobacion_sha256"`
	Entorno          string            `json:"entorno"`
	PreparadoEn      time.Time         `json:"preparado_en"`
	CaducaEn         time.Time         `json:"caduca_en"`
}

// planAdmision tiene las 17 cadenas, en el orden que espera
// vec-catalogo-acciones-admitir y AUT58.
type planAdmision struct {
	Esquema          string `json:"esquema"`
	OperacionRef     string `json:"operacion_ref"`
	AprobacionRef    string `json:"aprobacion_ref"`
	AprobacionSHA256 string `json:"aprobacion_sha256"`
	PaqueteCanon     string `json:"paquete_canon"`
	PaqueteRef       string `json:"paquete_ref"`
	PaqueteVersion   string `json:"paquete_version"`
	PaqueteSHA256    string `json:"paquete_sha256"`
	CatalogoCanon    string `json:"catalogo_canon"`
	CatalogoRef      string `json:"catalogo_ref"`
	CatalogoVersion  string `json:"catalogo_version"`
	CatalogoSHA256   string `json:"catalogo_sha256"`
	EsperadoVersion  string `json:"esperado_version"`
	EsperadoSHA256   string `json:"esperado_sha256"`
	PreparadoEn      string `json:"preparado_en"`
	CaducaEn         string `json:"caduca_en"`
	Entorno          string `json:"entorno"`
}

type salidaCatalogo struct {
	Plan            string   `json:"plan"`
	PlanSHA256      string   `json:"plan_sha256"`
	PaqueteRef      string   `json:"paquete_ref"`
	PaqueteVersion  int      `json:"paquete_version"`
	PaqueteSHA256   string   `json:"paquete_sha256"`
	CatalogoRef     string   `json:"catalogo_ref"`
	CatalogoVersion int      `json:"catalogo_version"`
	CatalogoSHA256  string   `json:"catalogo_sha256"`
	Sustituidas     []string `json:"fuentes_sustituidas"`
	Entradas        int      `json:"entradas"`
	Perfiles        int      `json:"perfiles"`
}

func huellaBytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// leerCatalogoVigente decodifica la cabeza publicada y exige que sus bytes
// sean exactamente la serialización Go, como hace la herramienta de admisión.
func leerCatalogoVigente(canonCatalogo, canonPaquete, huellaCatalogo string) (dominio.CatalogoAccionesAdministracionV1, paquete, error) {
	var c dominio.CatalogoAccionesAdministracionV1
	var p paquete
	if err := decodificarExacto([]byte(canonCatalogo), &c); err != nil {
		return c, p, err
	}
	if b, err := json.Marshal(c); err != nil || !bytes.Equal(b, []byte(canonCatalogo)) ||
		huellaBytes(b) != huellaCatalogo || c.Validar() != nil {
		return c, p, errors.New("la cabeza del catálogo no es canónica o no coincide con su huella")
	}
	if err := decodificarExacto([]byte(canonPaquete), &p); err != nil {
		return c, p, err
	}
	if b, err := json.Marshal(p); err != nil || !bytes.Equal(b, []byte(canonPaquete)) || p.Esquema != esquemaPaquete {
		return c, p, errors.New("el paquete vigente no es canónico")
	}
	if adminpg.ValidarPaqueteCatalogoAccionesV2([]byte(canonPaquete), c, p.Referencia, p.Version,
		huellaBytes([]byte(canonPaquete))) != nil {
		return c, p, errors.New("el paquete vigente no respalda la cabeza del catálogo")
	}
	return c, p, nil
}

// mismoJSON compara el valor Go re-serializado con el documento de la base,
// para no proponer un perfil que AUT58 vería distinto.
func mismoJSON(valor any, documento json.RawMessage) bool {
	b, err := json.Marshal(valor)
	if err != nil {
		return false
	}
	var a, d any
	return json.Unmarshal(b, &a) == nil && json.Unmarshal(documento, &d) == nil && reflect.DeepEqual(a, d)
}

func perfilesActuales(entrada []perfilActual) (map[string]perfilActual, map[string]dominio.PerfilPublicadoAdministracionV1, error) {
	crudos := make(map[string]perfilActual, len(entrada))
	perfiles := make(map[string]dominio.PerfilPublicadoAdministracionV1, len(entrada))
	for _, pa := range entrada {
		var p dominio.PerfilPublicadoAdministracionV1
		if json.Unmarshal(pa.Rol, &p.Rol) != nil || json.Unmarshal(pa.Control, &p.ControlVigencia) != nil ||
			!mismoJSON(p.Rol, pa.Rol) || !mismoJSON(p.ControlVigencia, pa.Control) {
			return nil, nil, errors.New("un perfil de la base no se representa sin pérdida")
		}
		if _, repetido := perfiles[p.Rol.RolID]; repetido || p.Rol.RolID == "" {
			return nil, nil, errors.New("censo de perfiles con un rol repetido o vacío")
		}
		perfiles[p.Rol.RolID], crudos[p.Rol.RolID] = p, pa
	}
	return crudos, perfiles, nil
}

func prepararCatalogo(e entradaCatalogo) (salidaCatalogo, error) {
	var vacia salidaCatalogo
	if !strings.HasPrefix(e.OperacionRef, "caa_") || len(e.OperacionRef) < 26 || e.AprobacionRef == "" ||
		len(e.AprobacionSHA256) != 64 || e.Entorno != "desarrollo" || len(e.Fuentes) == 0 ||
		!e.CaducaEn.After(e.PreparadoEn) || e.CaducaEn.Sub(e.PreparadoEn) > 24*time.Hour {
		return vacia, errors.New("parámetros del plan fuera de contrato")
	}
	anterior, p, err := leerCatalogoVigente(e.CatalogoCanon, e.PaqueteCanon, e.CatalogoSHA256)
	if err != nil {
		return vacia, err
	}
	var sustituidas []string
	for _, crudo := range e.Fuentes {
		var nueva fuentePaquete
		if err := decodificarExacto(crudo, &nueva); err != nil {
			return vacia, err
		}
		indice := -1
		for i, f := range p.Fuentes {
			if f.ModuloID == nueva.ModuloID && f.Referencia == nueva.Referencia {
				indice = i
			}
		}
		// Solo se sustituye una fuente ya publicada por una versión posterior:
		// un catálogo no admite dos entradas con la misma acción.
		if indice < 0 || nueva.Version <= p.Fuentes[indice].Version {
			return vacia, fmt.Errorf("la fuente %s v%d no sustituye a ninguna anterior", nueva.Referencia, nueva.Version)
		}
		p.Fuentes[indice] = nueva
		sustituidas = append(sustituidas, nueva.Referencia+":v"+strconv.Itoa(nueva.Version))
	}
	_, actuales, err := perfilesActuales(e.Perfiles)
	if err != nil {
		return vacia, err
	}
	if len(actuales) != len(p.Perfiles) {
		return vacia, errors.New("el censo actual de roles difiere del publicado; un rol nuevo exige decidir su tipo")
	}
	for i, perfil := range p.Perfiles {
		actual, ok := actuales[perfil.Rol.RolID]
		if !ok {
			return vacia, fmt.Errorf("el rol %s ya no está en el censo actual", perfil.Rol.RolID)
		}
		// El tipo (fijo o administrable) se conserva de la publicación anterior.
		actual.TipoPerfil = perfil.TipoPerfil
		p.Perfiles[i] = actual
	}
	p.Version++
	canonPaquete, err := json.Marshal(p)
	if err != nil {
		return vacia, err
	}
	nuevo := dominio.CatalogoAccionesAdministracionV1{
		Referencia: anterior.Referencia, Version: anterior.Version + 1,
		FuenteRef: p.Referencia, FuenteVersion: p.Version, FuenteHuellaSHA256: huellaBytes(canonPaquete),
		VigenteDesde: anterior.VigenteDesde, VigenteHasta: anterior.VigenteHasta, Perfiles: p.Perfiles,
	}
	for _, f := range p.Fuentes {
		nuevo.Entradas = append(nuevo.Entradas, f.Entradas...)
	}
	canonCatalogo, err := json.Marshal(nuevo)
	if err != nil || nuevo.Validar() != nil {
		return vacia, errors.New("el catálogo nuevo no es válido")
	}
	if err := adminpg.ValidarPaqueteCatalogoAccionesV2(canonPaquete, nuevo, p.Referencia, p.Version,
		nuevo.FuenteHuellaSHA256); err != nil {
		return vacia, errors.New("el paquete nuevo no respalda el catálogo")
	}
	instante := func(t time.Time) string { return t.UTC().Truncate(time.Second).Format(time.RFC3339) }
	plan := planAdmision{
		Esquema: esquemaPlan, OperacionRef: e.OperacionRef,
		AprobacionRef: e.AprobacionRef, AprobacionSHA256: e.AprobacionSHA256,
		PaqueteCanon: string(canonPaquete), PaqueteRef: p.Referencia,
		PaqueteVersion: strconv.Itoa(p.Version), PaqueteSHA256: nuevo.FuenteHuellaSHA256,
		CatalogoCanon: string(canonCatalogo), CatalogoRef: nuevo.Referencia,
		CatalogoVersion: strconv.Itoa(nuevo.Version), CatalogoSHA256: huellaBytes(canonCatalogo),
		EsperadoVersion: strconv.Itoa(anterior.Version), EsperadoSHA256: e.CatalogoSHA256,
		PreparadoEn: instante(e.PreparadoEn), CaducaEn: instante(e.CaducaEn), Entorno: e.Entorno,
	}
	bytesPlan, err := json.Marshal(plan)
	if err != nil {
		return vacia, err
	}
	return salidaCatalogo{
		Plan: string(bytesPlan), PlanSHA256: huellaBytes(bytesPlan),
		PaqueteRef: p.Referencia, PaqueteVersion: p.Version, PaqueteSHA256: nuevo.FuenteHuellaSHA256,
		CatalogoRef: nuevo.Referencia, CatalogoVersion: nuevo.Version, CatalogoSHA256: plan.CatalogoSHA256,
		Sustituidas: sustituidas, Entradas: len(nuevo.Entradas), Perfiles: len(nuevo.Perfiles),
	}, nil
}
