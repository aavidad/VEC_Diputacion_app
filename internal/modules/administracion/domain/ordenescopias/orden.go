// Package ordenescopias defines exact commitments. Structural validity is never
// authorization, a SQL commit, cryptographic verification or platform execution.
package ordenescopias

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

var ErrOrden = errors.New("orden_invalida")
var ref = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9:_-]{0,127}$`)

const Esquema = "vec.administracion.orden-copias.v1"

type Datos struct {
	SolicitudSHA256   string    `json:"solicitud_sha256"`
	Orden             string    `json:"orden"`
	Operacion         string    `json:"operacion"`
	Accion            string    `json:"accion"`
	Conjunto          string    `json:"conjunto"`
	ManifiestoSHA256  string    `json:"manifiesto_sha256"`
	PreimagenSHA256   string    `json:"preimagen_sha256"`
	Destino           string    `json:"destino"`
	ProponentePersona string    `json:"proponente_persona"`
	AprobadorPersona  string    `json:"aprobador_persona"`
	Politica          string    `json:"politica"`
	PoliticaSHA256    string    `json:"politica_sha256"`
	DecisionV3        string    `json:"decision_v3"`
	DecisionSHA256    string    `json:"decision_sha256"`
	ConsumoV3         string    `json:"consumo_v3"`
	Auditoria         string    `json:"auditoria"`
	Outbox            string    `json:"outbox"`
	Epoca             string    `json:"epoca"`
	Fence             uint64    `json:"fence"`
	VersionCAS        uint64    `json:"version_cas"`
	EmitidaEn         time.Time `json:"emitida_en"`
	CaducaEn          time.Time `json:"caduca_en"`
}

// Orden is an immutable structural commitment, deliberately not a capability.
type Orden struct{ datos *Datos }

func Nueva(d Datos) (Orden, error) {
	for _, s := range []string{d.Orden, d.Operacion, d.Conjunto, d.Destino, d.ProponentePersona, d.AprobadorPersona, d.Politica, d.DecisionV3, d.Auditoria, d.Outbox, d.Epoca} {
		if !ref.MatchString(s) {
			return Orden{}, ErrOrden
		}
	}
	for _, s := range []string{d.SolicitudSHA256, d.ManifiestoSHA256, d.PreimagenSHA256, d.PoliticaSHA256, d.DecisionSHA256, d.ConsumoV3} {
		if !huella(s) {
			return Orden{}, ErrOrden
		}
	}
	switch d.Accion {
	case "capturar_conjunto", "restaurar_conjunto", "configurar_copias", "aplicar_retencion":
	default:
		return Orden{}, ErrOrden
	}
	if d.ProponentePersona == d.AprobadorPersona || d.Fence == 0 || d.VersionCAS == 0 || !instante(d.EmitidaEn) || !instante(d.CaducaEn) || !d.EmitidaEn.Before(d.CaducaEn) {
		return Orden{}, ErrOrden
	}
	return Orden{datos: &d}, nil
}
func (o Orden) Datos() (Datos, error) {
	if o.datos == nil {
		return Datos{}, ErrOrden
	}
	return *o.datos, nil
}
func (o Orden) ValidarEn(t time.Time) error {
	d, err := o.Datos()
	if err != nil || !instante(t) || t.Before(d.EmitidaEn) || !t.Before(d.CaducaEn) {
		return ErrOrden
	}
	return nil
}
func (o Orden) Bytes() ([]byte, error) {
	d, err := o.Datos()
	if err != nil {
		return nil, err
	}
	// Closed ordered DTO and a versioned domain separator; no maps or free text.
	return json.Marshal(struct {
		Esquema string `json:"esquema"`
		Datos   Datos  `json:"datos"`
	}{Esquema, d})
}
func (o Orden) SHA256() (string, error) {
	b, e := o.Bytes()
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}
func huella(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func instante(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Year() >= 1 && t.Year() <= 9999 && t.Nanosecond()%1000 == 0
}

type datosPlan struct {
	SolicitudSHA256   string    `json:"solicitud_sha256"`
	Orden             string    `json:"orden"`
	Operacion         string    `json:"operacion"`
	Accion            string    `json:"accion"`
	Conjunto          string    `json:"conjunto"`
	ManifiestoSHA256  string    `json:"manifiesto_sha256"`
	PreimagenSHA256   string    `json:"preimagen_sha256"`
	Destino           string    `json:"destino"`
	ProponentePersona string    `json:"proponente_persona"`
	AprobadorPersona  string    `json:"aprobador_persona"`
	Politica          string    `json:"politica"`
	PoliticaSHA256    string    `json:"politica_sha256"`
	Epoca             string    `json:"epoca"`
	Fence             uint64    `json:"fence"`
	VersionCAS        uint64    `json:"version_cas"`
	EmitidaEn         time.Time `json:"emitida_en"`
	CaducaEn          time.Time `json:"caduca_en"`
}

// PlanBytes excludes authority receipts. The PDP commits this exact plan before
// consumption; hashing the final order into its own decision would be circular.
// The final authenticated order additionally binds the decision and receipts.
func (o Orden) PlanBytes() ([]byte, error) {
	d, err := o.Datos()
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Esquema string    `json:"esquema"`
		Datos   datosPlan `json:"datos"`
	}{
		"vec.administracion.plan-orden-copias.v1", datosPlan{
			SolicitudSHA256: d.SolicitudSHA256, Orden: d.Orden, Operacion: d.Operacion, Accion: d.Accion,
			Conjunto: d.Conjunto, ManifiestoSHA256: d.ManifiestoSHA256, PreimagenSHA256: d.PreimagenSHA256,
			Destino: d.Destino, ProponentePersona: d.ProponentePersona, AprobadorPersona: d.AprobadorPersona,
			Politica: d.Politica, PoliticaSHA256: d.PoliticaSHA256, Epoca: d.Epoca, Fence: d.Fence, VersionCAS: d.VersionCAS,
			EmitidaEn: d.EmitidaEn, CaducaEn: d.CaducaEn}})
}
func (o Orden) PlanSHA256() (string, error) {
	b, e := o.PlanBytes()
	if e != nil {
		return "", e
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}
