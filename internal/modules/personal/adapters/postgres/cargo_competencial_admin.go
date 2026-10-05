package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"
	"unicode/utf8"

	efecto "vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Publicación de cargos competenciales desde vec-admin (Personal28 y AD166).
// Las coordenadas son las que la fachada y el núcleo exigen a la decisión V3;
// no son configurables.
const (
	AccionPublicarCargoCompetencial    = "personal.cargo_competencial.publicar"
	AudienciaPublicarCargoCompetencial = "vec_personal.cargo_competencial.publicar.v1"
	moduloCargoCompetencial            = "personal"
	tipoCargoCompetencial              = "cargo_competencial"
	finalidadCargoCompetencial         = "administrar_cargos_competenciales"
	esquemaPublicacionCargo            = "vec.personal.cargo-competencial.publicacion.v1"
	// MaximoMaterialPublicacionCargo es el límite de Personal28 (1..32768).
	MaximoMaterialPublicacionCargo = 32768

	publicarCargoCompetencialSQL = `SELECT vec_personal.publicar_cargo_competencial_v1($1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)::text`
)

var (
	errMaterialCargo = errors.New("personal: material de publicación de cargo no válido")

	clavesMaterialCargo = []string{"esquema", "operacion", "clave_idempotencia", "objeto_ref",
		"organizacion_ref", "unidad_ref", "version_esperada", "huella_esperada", "datos"}
	// Mismos formatos que exige Personal28.
	objetoCargo      = regexp.MustCompile(`^car_[A-Za-z0-9_-]{22,128}$`)
	objetoEnlace     = regexp.MustCompile(`^enc_[A-Za-z0-9_-]{22,128}$`)
	ambitoCargo      = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
	reciboCargo      = regexp.MustCompile(`^percar_[0-9a-f]{32}$`)
	auditoriaCargo   = regexp.MustCompile(`^aud_v3_[0-9a-f]{32}$`)
	huellaCargo      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	camposDecisionCa = []string{"cargo", "enlace", "huella_sha256", "recibo", "version"}
)

// ContratoPublicacionCargoCompetencial es el efecto nominal de vec-admin que
// publica un cargo competencial o su enlace con quien lo ocupa.
func ContratoPublicacionCargoCompetencial() efecto.Contrato {
	return efecto.Contrato{
		Audiencia: AudienciaPublicarCargoCompetencial, Modulo: moduloCargoCompetencial, Tipo: tipoCargoCompetencial,
		Finalidad: finalidadCargoCompetencial, Campos: append([]string(nil), camposDecisionCa...),
		Acciones:      []string{AccionPublicarCargoCompetencial},
		AccionIntento: AccionPublicarCargoCompetencial, PrefijoIntento: "solicitud_cargo_competencial",
		Recurso: RecursoPublicacionCargoCompetencial, Sentencia: publicarCargoCompetencialSQL,
		Recibo: reciboPublicacionCargoCompetencial,
	}
}

// RecursoPublicacionCargoCompetencial deriva del material exacto el recurso
// que Personal28 exige a la decisión: referencia = objeto_ref, ámbitos
// organización y unidad del material y atributos SHA-256 del material y
// operación. La asignación del administrador debe cubrir esos ámbitos; eso lo
// comprueban el emisor y el PDP. No autoriza nada: la fachada lo recalcula.
func RecursoPublicacionCargoCompetencial(material []byte, _ vd.AsignacionPerfil) (string, vd.RecursoAutorizable, error) {
	var cero vd.RecursoAutorizable
	if len(material) < 2 || len(material) > MaximoMaterialPublicacionCargo || !utf8.Valid(material) ||
		bytes.Contains(material, []byte(`\u0000`)) || !json.Valid(material) {
		return "", cero, errMaterialCargo
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(material, &m) != nil || len(m) != len(clavesMaterialCargo) {
		return "", cero, errMaterialCargo
	}
	for _, clave := range clavesMaterialCargo {
		if _, ok := m[clave]; !ok {
			return "", cero, errMaterialCargo
		}
	}
	var esquema, operacion, objeto, organizacion, unidad string
	if cadena(m["esquema"], &esquema) != nil || esquema != esquemaPublicacionCargo ||
		cadena(m["operacion"], &operacion) != nil || cadena(m["objeto_ref"], &objeto) != nil ||
		cadena(m["organizacion_ref"], &organizacion) != nil || !ambitoCargo.MatchString(organizacion) ||
		cadena(m["unidad_ref"], &unidad) != nil || !ambitoCargo.MatchString(unidad) {
		return "", cero, errMaterialCargo
	}
	switch {
	case operacion == "cargo" && objetoCargo.MatchString(objeto):
	case operacion == "enlace" && objetoEnlace.MatchString(objeto):
	default:
		return "", cero, errMaterialCargo
	}
	suma := sha256.Sum256(material)
	return AccionPublicarCargoCompetencial, vd.RecursoAutorizable{
		Referencia: objeto, ModuloID: moduloCargoCompetencial, Tipo: tipoCargoCompetencial,
		Ambitos:   map[string]string{"organizacion_ref": organizacion, "unidad_ref": unidad},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(suma[:]), "operacion": operacion},
	}, nil
}

func cadena(b json.RawMessage, destino *string) error {
	if len(b) == 0 || b[0] != '"' {
		return errMaterialCargo
	}
	return json.Unmarshal(b, destino)
}

// ReciboPublicacionCargoCompetencial es lo que ve el administrador: el recibo
// de la publicación (en un reintento, el original) y la auditoría del efecto.
type ReciboPublicacionCargoCompetencial struct {
	ReciboRef    string    `json:"recibo_ref"`
	ObjetoRef    string    `json:"objeto_ref"`
	Version      int64     `json:"version"`
	HuellaSHA256 string    `json:"huella_sha256,omitempty"`
	RegistradaEn time.Time `json:"registrada_en"`
	AuditoriaRef string    `json:"auditoria_ref"`
	EstadoReplay string    `json:"estado_replay"`
}

// reciboPublicacionCargoCompetencial lee la respuesta exacta de Personal28:
// «nueva» lleva la huella del dato y su auditoría es la de este acceso;
// «recuperada» devuelve el recibo original y la auditoría del acceso actual.
func reciboPublicacionCargoCompetencial(bruto []byte, _ string, recurso vd.RecursoAutorizable) (efecto.Recibo, error) {
	var r struct {
		ReciboPublicacionCargoCompetencial
		AccesoActual *string `json:"acceso_actual_auditoria_ref"`
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return efecto.Recibo{}, errMaterialCargo
	}
	p := r.ReciboPublicacionCargoCompetencial
	if !reciboCargo.MatchString(p.ReciboRef) || p.ObjetoRef != recurso.Referencia || p.Version < 1 ||
		p.RegistradaEn.IsZero() || !auditoriaCargo.MatchString(p.AuditoriaRef) {
		return efecto.Recibo{}, errMaterialCargo
	}
	consumo := p.AuditoriaRef
	switch p.EstadoReplay {
	case "nueva":
		if !huellaCargo.MatchString(p.HuellaSHA256) || r.AccesoActual != nil {
			return efecto.Recibo{}, errMaterialCargo
		}
	case "recuperada":
		if p.HuellaSHA256 != "" || r.AccesoActual == nil || !auditoriaCargo.MatchString(*r.AccesoActual) || *r.AccesoActual == p.AuditoriaRef {
			return efecto.Recibo{}, errMaterialCargo
		}
		consumo = *r.AccesoActual
	default:
		return efecto.Recibo{}, errMaterialCargo
	}
	cuerpo, err := json.Marshal(p)
	if err != nil {
		return efecto.Recibo{}, errMaterialCargo
	}
	return efecto.Recibo{Cuerpo: cuerpo, ConsumoAuditoriaRef: consumo}, nil
}
