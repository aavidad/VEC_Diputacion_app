package main

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
)

// composicionCopia deja a Go el cálculo de los documentos y huellas canónicos.
// La aplicación sigue siendo quien autoriza y publica las dos operaciones.
type composicionCopia struct {
	Catalogo              vd.CatalogoConfigurable `json:"catalogo"`
	CircuitoArchivo       string                  `json:"circuito_archivo"`
	CircuitoArchivoSHA256 string                  `json:"circuito_archivo_sha256"`
	Publicador            string                  `json:"publicador"`
	PerfilCreador         string                  `json:"perfil_creador"`
	PerfilPublicador      string                  `json:"perfil_publicador"`
	AutorizacionCrear     string                  `json:"autorizacion_crear"`
	AutorizacionPublicar  string                  `json:"autorizacion_publicar"`
	CorrelacionCrear      string                  `json:"correlacion_crear"`
	CorrelacionPublicar   string                  `json:"correlacion_publicar"`
	ClaveCrear            string                  `json:"clave_crear"`
	ClavePublicar         string                  `json:"clave_publicar"`
	AprobacionRef         string                  `json:"aprobacion_ref"`
	MotivoPublicacion     string                  `json:"motivo_publicacion"`
	PublicadoEn           time.Time               `json:"publicado_en"`
}

type materialCompuesto struct {
	Catalogo []byte
	Material []byte
	Resumen  resumen
}

func componerCopia(cfg composicionCopia) (materialCompuesto, materialCompuesto, error) {
	var cero materialCompuesto
	c := cfg.Catalogo
	if c.Estado != vd.EstadoCatalogoBorrador || c.Revision != 1 || c.ModuloID != "contratacion_temporal" ||
		c.Version != 1 || c.CreadoPor == cfg.Publicador ||
		cfg.PerfilCreador == "" || cfg.PerfilPublicador == "" || cfg.AutorizacionCrear == "" ||
		cfg.AutorizacionPublicar == "" || cfg.CorrelacionCrear == "" || cfg.CorrelacionPublicar == "" ||
		cfg.ClaveCrear == cfg.ClavePublicar || cfg.PublicadoEn.IsZero() {
		return cero, cero, errMaterial
	}
	circuitoRaw, err := leerEntradaPreparacion(cfg.CircuitoArchivo, 2<<20)
	if err != nil || jsonUnico(circuitoRaw) != nil || !shaValido.MatchString(cfg.CircuitoArchivoSHA256) || huella(circuitoRaw) != cfg.CircuitoArchivoSHA256 {
		return cero, cero, errMaterial
	}
	var fuente struct {
		Fuente struct {
			Demostracion *bool `json:"demostracion"`
		} `json:"fuente"`
		Catalogo vd.CatalogoConfigurable `json:"catalogo"`
	}
	if json.Unmarshal(circuitoRaw, &fuente) != nil || fuente.Fuente.Demostracion == nil || *fuente.Fuente.Demostracion ||
		fuente.Catalogo.ID != "vec.contratacion_temporal.circuito_firma" ||
		fuente.Catalogo.Version != 2 || fuente.Catalogo.Estado != vd.EstadoCatalogoPublicado {
		return cero, cero, errMaterial
	}
	circuitoSHA, err := fuente.Catalogo.HuellaSHA256()
	if err != nil {
		return cero, cero, err
	}
	canon, err := c.ClonarCanonico()
	if err != nil {
		return cero, cero, err
	}
	if fuente.Catalogo.PublicadoEn.After(canon.CreadoEn) {
		return cero, cero, errMaterial
	}
	for i := range canon.Entradas {
		entrada := &canon.Entradas[i]
		atributos := entrada.Atributos
		var original *vd.EntradaCatalogoConfigurable
		for j := range fuente.Catalogo.Entradas {
			if fuente.Catalogo.Entradas[j].Clave == entrada.Clave {
				original = &fuente.Catalogo.Entradas[j]
				break
			}
		}
		if atributos == nil || original == nil || atributos["documento"] != original.Atributos["documento"] ||
			atributos["paso_orden"] != original.Atributos["paso"] ||
			(atributos["perfil_esperado_ref"] != original.Atributos["perfil_ref"] &&
				(atributos["perfil_esperado_ref"] == "" || !slices.Contains(strings.Split(original.Atributos["perfiles_ref_alternativos"], ","), atributos["perfil_esperado_ref"]))) {
			return cero, cero, errMaterial
		}
		ref := vd.ReferenciaEntradaCatalogo{CatalogoID: fuente.Catalogo.ID, CatalogoVersion: fuente.Catalogo.Version,
			CatalogoHuellaSHA256: circuitoSHA, EntradaClave: entrada.Clave}
		for clave, valor := range map[string]string{"circuito_ref": fuente.Catalogo.Referencia(),
			"circuito_version": strconv.Itoa(fuente.Catalogo.Version), "circuito_sha256": circuitoSHA, "paso_ref": ref.Referencia()} {
			if anterior := atributos[clave]; anterior != "" && anterior != valor {
				return cero, cero, errMaterial
			}
			atributos[clave] = valor
		}
	}
	publicado, err := canon.Publicar(cfg.Publicador, cfg.AprobacionRef, cfg.MotivoPublicacion, cfg.PublicadoEn)
	if err != nil {
		return cero, cero, err
	}
	crear, err := componerOperacion(canon, "crear", 0, nil, cfg.ClaveCrear, cfg.PerfilCreador, cfg.AutorizacionCrear, cfg.CorrelacionCrear)
	if err != nil {
		return cero, cero, err
	}
	huellaAnterior := huella(crear.Catalogo)
	pub, err := componerOperacion(publicado, "publicar", canon.Revision, &huellaAnterior, cfg.ClavePublicar, cfg.PerfilPublicador, cfg.AutorizacionPublicar, cfg.CorrelacionPublicar)
	if err != nil {
		return cero, cero, err
	}
	return crear, pub, nil
}

func componerOperacion(c vd.CatalogoConfigurable, op string, revision int, anterior *string, clave, perfil, autorizacion, correlacion string) (materialCompuesto, error) {
	var cero materialCompuesto
	if !claveOperacion.MatchString(clave) {
		return cero, errMaterial
	}
	catalogo, err := json.Marshal(c)
	if err != nil {
		return cero, err
	}
	accion := vd.AccionCatalogoBorradorCreado
	fecha, regla, motivo := c.CreadoEn, c.FuenteRef, c.MotivoCreacion
	if op == "publicar" {
		accion, fecha, regla, motivo = vd.AccionCatalogoPublicado, c.PublicadoEn, c.AprobacionRef, c.MotivoPublicacion
	}
	antes := ""
	if anterior != nil {
		antes = *anterior
	}
	sha := huella(catalogo)
	a := vd.AuditEntry{ActorID: c.CreadoPor, ActorProfile: perfil, AuthorizationRef: autorizacion,
		Purpose: "gestionar_contratacion_temporal", Action: accion, ModuleID: c.ModuloID,
		SubjectRef: c.Referencia(), ObjectVersion: c.Version, RuleRef: regla, Reason: motivo,
		Result: "correcto", BeforeHash: antes, AfterHash: sha, CorrelationRef: correlacion,
		OccurredAt: fecha, Metadata: map[string]string{"catalogo_id": c.ID, "catalogo_version": strconv.Itoa(c.Version),
			"estado": string(c.Estado), "revision": strconv.Itoa(c.Revision)}}
	if op == "publicar" {
		a.ActorID = c.PublicadoPor
	}
	e := vd.Event{Type: accion, ModuleID: c.ModuloID, SubjectRef: c.Referencia(), ActorID: a.ActorID,
		OccurredAt: fecha, Payload: map[string]string{"catalogo_id": c.ID, "catalogo_version": strconv.Itoa(c.Version),
			"catalogo_revision": strconv.Itoa(c.Revision), "estado": string(c.Estado), "huella_sha256": sha}}
	traza, err := json.Marshal(a)
	if err != nil {
		return cero, err
	}
	evento, err := json.Marshal(e)
	if err != nil {
		return cero, err
	}
	config, err := json.Marshal(configuracionPreparacion{Operacion: op, RevisionEsperada: revision, HuellaEsperada: anterior, ClaveOperacion: clave})
	if err != nil {
		return cero, err
	}
	material, r, err := prepararMaterial(config, catalogo, traza, evento)
	if err != nil {
		return cero, fmt.Errorf("componer %s: %w", op, err)
	}
	return materialCompuesto{Catalogo: catalogo, Material: material, Resumen: r}, nil
}
