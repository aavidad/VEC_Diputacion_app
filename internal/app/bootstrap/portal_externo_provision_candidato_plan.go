package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Los errores son códigos de contrato; la CLI solo emite JSON técnico.
var ErrProvisionCandidatoExterno = errors.New("provision_candidato_externo_rechazada")

var referenciaProvisionExterna = regexp.MustCompile(`^[a-z]+_[A-Za-z0-9_-]{16,128}$`)

// ComponenteSnapshotContextoExterno conserva la fuente maestra acreditada;
// no deriva identidad ni procedencia del certificado o de un perfil interno.
type ComponenteSnapshotContextoExterno struct {
	Referencia              string `json:"referencia"`
	Version                 int64  `json:"version"`
	ProcedenciaRef          string `json:"procedencia_ref"`
	ProcedenciaVersion      int64  `json:"procedencia_version"`
	ProcedenciaHuellaSHA256 string `json:"procedencia_huella_sha256"`
	ProcedenciaAutoridad    string `json:"procedencia_autoridad"`
	Estado                  string `json:"estado"`
	VigenteDesde            string `json:"vigente_desde"`
	VigenteHasta            string `json:"vigente_hasta"`
}

type VinculoSnapshotCandidatoExterno struct {
	ComponenteSnapshotContextoExterno
	CandidatoRef string `json:"candidato_ref"`
}

// SnapshotContextoExterno es el contrato de datos cerrado CTX15. Usuarios
// puede reutilizarlo sin importar permisos ni credenciales de AUT16.
type SnapshotContextoExterno struct {
	ProvisionRef     string                            `json:"provision_ref"`
	Poblacion        string                            `json:"poblacion"`
	Estado           string                            `json:"estado"`
	Cuenta           ComponenteSnapshotContextoExterno `json:"cuenta"`
	Persona          ComponenteSnapshotContextoExterno `json:"persona"`
	Perfil           ComponenteSnapshotContextoExterno `json:"perfil"`
	Contexto         ComponenteSnapshotContextoExterno `json:"contexto"`
	VinculoCandidato *VinculoSnapshotCandidatoExterno  `json:"vinculo_candidato"`
}

func (SnapshotContextoExterno) String() string   { return "[SNAPSHOT EXTERNO PRIVADO]" }
func (SnapshotContextoExterno) GoString() string { return "[SNAPSHOT EXTERNO PRIVADO]" }

func (s SnapshotContextoExterno) validar() bool {
	prefijo := "pce_"
	if s.Poblacion == "usuarios" {
		prefijo = "pue_"
	}
	if (s.Poblacion != "candidato" && s.Poblacion != "usuarios") || !referenciaExterna(s.ProvisionRef, prefijo) ||
		(s.Estado != "activo" && s.Estado != "revocado") {
		return false
	}
	for i, c := range []ComponenteSnapshotContextoExterno{s.Cuenta, s.Persona, s.Perfil, s.Contexto} {
		if c.Estado != s.Estado || !componenteExternoValido(c, []string{"cta_", "per_", "prf_", "vca_"}[i]) {
			return false
		}
	}
	if s.Poblacion == "usuarios" {
		return s.VinculoCandidato == nil
	}
	return s.VinculoCandidato != nil && s.VinculoCandidato.Estado == s.Estado && componenteExternoValido(s.VinculoCandidato.ComponenteSnapshotContextoExterno, "vin_") && referenciaExterna(s.VinculoCandidato.CandidatoRef, "can_")
}

func componenteExternoValido(c ComponenteSnapshotContextoExterno, prefijo string) bool {
	desde, e1 := time.Parse("2006-01-02T15:04:05.000000Z", c.VigenteDesde)
	hasta, e2 := time.Parse("2006-01-02T15:04:05.000000Z", c.VigenteHasta)
	return referenciaExterna(c.Referencia, prefijo) && c.Version > 0 && c.ProcedenciaVersion > 0 &&
		referenciaExterna(c.ProcedenciaRef, "prc_") && huellaPreimagenMiBolsaPortalExterno.MatchString(c.ProcedenciaHuellaSHA256) &&
		c.ProcedenciaAutoridad == "autoridad_maestra_acreditada" && (c.Estado == "activo" || c.Estado == "revocado") &&
		e1 == nil && e2 == nil && desde.Format("2006-01-02T15:04:05.000000Z") == c.VigenteDesde && hasta.Format("2006-01-02T15:04:05.000000Z") == c.VigenteHasta && hasta.After(desde)
}

func referenciaExterna(ref, prefijo string) bool {
	return strings.HasPrefix(ref, prefijo) && referenciaProvisionExterna.MatchString(ref)
}

// PreimagenProvisionCandidatoExterno fija todos los CAS antes de preparar el
// plan. Una preimagen vacía solo representa el alta explícita de versión 0.
type PreimagenProvisionCandidatoExterno struct {
	RevisionControlRol int64  `json:"revision_control_rol"`
	HuellaControlRol   string `json:"huella_control_rol"`
	VersionAsignacion  int64  `json:"version_asignacion"`
	HuellaAsignacion   string `json:"huella_asignacion"`
	VersionContexto    int64  `json:"version_contexto"`
	HuellaContexto     string `json:"huella_contexto"`
	SecuenciaMotivos   int64  `json:"secuencia_motivos"`
}

func (p PreimagenProvisionCandidatoExterno) validar() bool {
	for _, v := range []struct {
		n int64
		h string
	}{{p.RevisionControlRol, p.HuellaControlRol}, {p.VersionAsignacion, p.HuellaAsignacion}, {p.VersionContexto, p.HuellaContexto}} {
		if v.n < 0 || v.n >= 1<<31 || (v.n == 0 && v.h != "") || (v.n > 0 && !huellaPreimagenMiBolsaPortalExterno.MatchString(v.h)) {
			return false
		}
	}
	return p.SecuenciaMotivos >= 0 && p.SecuenciaMotivos < 1<<62
}

type FuenteProvisionCandidatoExterno struct {
	Version   int                                 `json:"version"`
	Snapshot  SnapshotContextoExterno             `json:"snapshot"`
	Preimagen PreimagenProvisionCandidatoExterno  `json:"preimagen"`
	Identidad *IdentidadProvisionCandidatoExterno `json:"identidad,omitempty"`
}

func (FuenteProvisionCandidatoExterno) String() string   { return "[FUENTE EXTERNA PRIVADA]" }
func (FuenteProvisionCandidatoExterno) GoString() string { return "[FUENTE EXTERNA PRIVADA]" }

// LeerMaterialProvisionExterna aplica el lector privado existente y rechaza
// rutas de Git, enlaces de cualquier componente y ficheros accesibles a otros.
func LeerMaterialProvisionExterna(ruta string, limite int64) ([]byte, error) {
	if !filepath.IsAbs(ruta) || dentroDeRepositorioGit(ruta) {
		return nil, ErrProvisionCandidatoExterno
	}
	evaluada, err := filepath.EvalSymlinks(ruta)
	if err != nil || evaluada != filepath.Clean(ruta) {
		return nil, ErrProvisionCandidatoExterno
	}
	b, err := leerFicheroMaterialSeguro(ruta, limite)
	if err != nil {
		return nil, ErrProvisionCandidatoExterno
	}
	return b, nil
}

func CargarFuenteProvisionCandidatoExterno(ruta string) (FuenteProvisionCandidatoExterno, error) {
	var f FuenteProvisionCandidatoExterno
	b, err := LeerMaterialProvisionExterna(ruta, 64<<10)
	if err != nil {
		return f, err
	}
	defer clear(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var sobra any
	if validarClavesJSONUnicas(b) != nil || d.Decode(&f) != nil || !errors.Is(d.Decode(&sobra), io.EOF) ||
		f.Version != 1 || !f.Snapshot.validar() || !f.Preimagen.validar() {
		return FuenteProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	return f, nil
}

type ResumenProvisionCandidatoExterno struct {
	Estado          string `json:"estado"`
	Fase            string `json:"fase"`
	HuellaSHA256    string `json:"huella_sha256"`
	PreimagenSHA256 string `json:"preimagen_sha256"`
}

// PlanProvisionCandidatoExterno no expone documentos. Su resumen es seguro
// para stdout; no incluye cuentas, personas, rutas, certificados ni DSN.
type PlanProvisionCandidatoExterno struct {
	resumen        ResumenProvisionCandidatoExterno
	preimagen      PreimagenProvisionCandidatoExterno
	snapshot       []byte
	huellaContexto string
	rol            documentosRolMiBolsaPortalExterno
	asignacion     documentoAsignacionMiBolsaPortalExterno
	semilla        core.InstantaneaAutorizacion
	motivos        []core.ReferenciaEntradaCatalogo
	desde          time.Time
	identidad      *IdentidadProvisionCandidatoExterno
}

func (PlanProvisionCandidatoExterno) String() string                              { return "[PLAN EXTERNO PRIVADO]" }
func (PlanProvisionCandidatoExterno) GoString() string                            { return "[PLAN EXTERNO PRIVADO]" }
func (p PlanProvisionCandidatoExterno) Resumen() ResumenProvisionCandidatoExterno { return p.resumen }

func procesoInternoProvisionExterna(cfg config.Config) bool {
	p, e := separacionportales.Parsear(cfg.PortalProceso)
	return e == nil && p == separacionportales.PortalInterno && cfg.DevelopmentEnabledByDoubleKey()
}

// PrepararProvisionCandidatoExterno es puro: no conecta ni publica nada.
// Contexto se completa después con su hash canónico calculado por CTX15.
func PrepararProvisionCandidatoExterno(cfg config.Config, f FuenteProvisionCandidatoExterno, fase string, ahora time.Time) (PlanProvisionCandidatoExterno, error) {
	var p PlanProvisionCandidatoExterno
	if !procesoInternoProvisionExterna(cfg) || f.Version != 1 || !f.Snapshot.validar() ||
		(f.Snapshot.Poblacion != "candidato" && !(fase == "identidad" && f.Snapshot.Poblacion == "usuarios")) ||
		!f.Preimagen.validar() || f.Snapshot.Estado != "activo" || (fase != "contexto" && fase != "autorizacion" && fase != "motivos" && fase != "identidad") {
		return p, ErrProvisionCandidatoExterno
	}
	s := f.Snapshot
	if s.Poblacion == "usuarios" {
		if f.Identidad == nil || !f.Identidad.validaPara(s.Cuenta.Referencia) {
			return p, ErrProvisionCandidatoExterno
		}
		var err error
		p.snapshot, err = json.Marshal(s)
		if err != nil {
			return p, ErrProvisionCandidatoExterno
		}
		p.preimagen = f.Preimagen
		copia := *f.Identidad
		p.identidad = &copia
		p.resumen = ResumenProvisionCandidatoExterno{Fase: fase, Estado: "preparado", PreimagenSHA256: huellaAusenciaIdentidadExterna(copia.CuentaRef)}
		p.actualizarHuella()
		return p, nil
	}
	identidad := &identidadCandidatoBolsaDesarrollo{cuentaRef: s.Cuenta.Referencia, personaRef: s.Persona.Referencia, perfilRef: s.Perfil.Referencia, candidatoRef: s.VinculoCandidato.CandidatoRef}
	semilla, err := semillaMiBolsaPortalExterno(identidad, ahora, "areaPersonal.miBolsa.rolPortal")
	if err != nil {
		return p, ErrProvisionCandidatoExterno
	}
	p.rol, err = prepararRolMiBolsaPortalExterno(semilla, f.Preimagen.RevisionControlRol, f.Preimagen.HuellaControlRol)
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	p.asignacion, err = prepararAsignacionMiBolsaPortalExterno(semilla, semilla.VersionRol, f.Preimagen.VersionAsignacion, f.Preimagen.HuellaAsignacion)
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	p.snapshot, err = json.Marshal(s)
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	p.semilla, p.preimagen, p.desde = semilla, f.Preimagen, semilla.VersionRol.PublicadaEn
	p.motivos = []core.ReferenciaEntradaCatalogo{motivoMiBolsaDesarrollo(), motivoHistorialMiBolsaDesarrollo(), motivoPortalMiBolsaDesarrollo()}
	p.resumen.Fase = fase
	p.resumen.Estado = "preparado"
	p.resumen.PreimagenSHA256 = huellaJSONProvisionExterna(f.Preimagen)
	if fase == "identidad" {
		if f.Identidad == nil || !f.Identidad.validaPara(s.Cuenta.Referencia) {
			return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
		}
		copia := *f.Identidad
		p.identidad = &copia
		p.resumen.PreimagenSHA256 = huellaAusenciaIdentidadExterna(copia.CuentaRef)
	}
	p.actualizarHuella()
	return p, nil
}

func huellaJSONProvisionExterna(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (p *PlanProvisionCandidatoExterno) actualizarHuella() {
	p.resumen.HuellaSHA256 = huellaJSONProvisionExterna(struct {
		Contrato                 string
		Fase                     string
		Preimagen                PreimagenProvisionCandidatoExterno
		Snapshot                 json.RawMessage
		HuellaContexto           string
		Rol, Control, Asignacion []byte
		Motivos                  []core.ReferenciaEntradaCatalogo
		Desde                    time.Time
		Identidad                *IdentidadProvisionCandidatoExterno
	}{"provision_candidato_externo_v1", p.resumen.Fase, p.preimagen, p.snapshot, p.huellaContexto, p.rol.RolDocumento, p.rol.ControlDocumento, p.asignacion.Documento, p.motivos, p.desde, p.identidad})
}
