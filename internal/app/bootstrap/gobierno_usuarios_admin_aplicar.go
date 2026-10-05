package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfirmacionGobiernoUsuariosAdmin sólo se entrega después de COMMIT. Un
// fallo/COMMIT indeterminado no autoriza repetir con plan o material nuevos.
type ConfirmacionGobiernoUsuariosAdmin struct {
	Estado           string          `json:"estado"`
	Codigo           string          `json:"codigo"`
	Recibo           json.RawMessage `json:"recibo"`
	AuditoriaIntento json.RawMessage `json:"auditoria_intento"`
}

func AplicarGobiernoUsuariosAdmin(ctx context.Context, pool *pgxpool.Pool, planCanonico, shaAprobado string, material *MaterialUsuariosAdmin) (ConfirmacionGobiernoUsuariosAdmin, error) {
	var vacio ConfirmacionGobiernoUsuariosAdmin
	if ctx == nil || ctx.Err() != nil || pool == nil || material == nil || len(planCanonico) > 16384 || len(shaAprobado) != 64 {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	if validarPlanGobiernoUsuariosAdmin(planCanonico, shaAprobado, material) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	var secreto bytes.Buffer
	defer func() { borrarBytes(secreto.Bytes()); secreto.Reset() }()
	if material.EscribirMaterialPrivado(&secreto) != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	// El conjunto del material elige la función: AD188 para usuarios, AD198
	// para un conjunto de capacidades. Ambas devuelven el mismo acuse.
	funcion := `SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1($1::text,$2::text,$3::text)`
	material.mu.RLock()
	if material.conjunto != 0 {
		funcion = `SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1($1::text,$2::text,$3::text)`
	}
	material.mu.RUnlock()
	var raw []byte
	if err = tx.QueryRow(ctx, funcion, planCanonico, shaAprobado, secreto.String()).Scan(&raw); err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	r, err := validarAcuseGobiernoUsuariosAdmin(raw, planCanonico, shaAprobado, secreto.Bytes())
	if err != nil {
		return vacio, ErrGobiernoUsuariosAdmin
	}
	if tx.Commit(ctx) != nil {
		// Tras enviar COMMIT no se sabe si la base lo aplicó: el llamante
		// recupera con el mismo plan, nunca con plan o material nuevos.
		return vacio, ErrCommitGobiernoUsuariosIndeterminado
	}
	return r, nil
}

var shaGobiernoUsuarios = regexp.MustCompile(`^[0-9a-f]{64}$`)
var operacionGobiernoUsuarios = regexp.MustCompile(`^gcu_[A-Za-z0-9_-]{22,124}$`)
var operacionGobiernoCapacidades = regexp.MustCompile(`^gca_[A-Za-z0-9_-]{22,124}$`)
var refAuditGobiernoUsuarios = regexp.MustCompile(`^aud_v3_gu_[0-9a-f]{32}$`)
var refIntentoGobiernoUsuarios = regexp.MustCompile(`^aud_v3_gui_[0-9a-f]{32}$`)
var correlacionGobiernoUsuarios = regexp.MustCompile(`^correlacion_[0-9a-f]{32}$`)

type planGobiernoUsuariosAdmin struct {
	Version         uint64    `json:"version"`
	OperacionRef    string    `json:"operacion_ref"`
	PreparadoEn     time.Time `json:"preparado_en"`
	CaducaEn        time.Time `json:"caduca_en"`
	PreimagenSHA256 string    `json:"preimagen_sha256"`
	// ConjuntoVersion sólo existe en el plan 2 (AD198).
	ConjuntoVersion uint64 `json:"conjunto_version,omitempty"`
	Configuracion   struct {
		Revision    string    `json:"revision"`
		Secuencia   uint64    `json:"secuencia"`
		Huella      string    `json:"huella_sha256"`
		PublicadaEn time.Time `json:"publicada_en"`
		ExpiraEn    time.Time `json:"expira_en"`
	} `json:"configuracion"`
	Ordenes []uint64 `json:"clave_ordenes"`
}

func decodificarGobiernoUsuarios(b []byte, destino any) error {
	if validarClavesJSONUnicas(b) != nil {
		return ErrGobiernoUsuariosAdmin
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return ErrGobiernoUsuariosAdmin
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}
func validarPlanGobiernoUsuariosAdmin(plan, sha string, m *MaterialUsuariosAdmin) error {
	if !shaGobiernoUsuarios.MatchString(sha) || m == nil {
		return ErrGobiernoUsuariosAdmin
	}
	h := sha256.Sum256([]byte(plan))
	if hex.EncodeToString(h[:]) != sha {
		return ErrGobiernoUsuariosAdmin
	}
	var p planGobiernoUsuariosAdmin
	m.mu.RLock()
	conjuntoMaterial := m.conjunto
	m.mu.RUnlock()
	conjunto, ok := AudienciasConjuntoCapacidadesAdmin(conjuntoMaterial)
	if !ok || decodificarGobiernoUsuarios([]byte(plan), &p) != nil || !shaGobiernoUsuarios.MatchString(p.PreimagenSHA256) ||
		p.ConjuntoVersion != conjuntoMaterial || len(p.Ordenes) != len(conjunto) || p.Ordenes[0] == 0 {
		return ErrGobiernoUsuariosAdmin
	}
	// Plan 1 (AD188): conjunto 0 y gcu_. Plan 2 (AD198): conjunto propio y gca_.
	if conjuntoMaterial == 0 && (p.Version != 1 || !operacionGobiernoUsuarios.MatchString(p.OperacionRef)) ||
		conjuntoMaterial != 0 && (p.Version != 2 || !operacionGobiernoCapacidades.MatchString(p.OperacionRef)) {
		return ErrGobiernoUsuariosAdmin
	}
	for i := 1; i < len(p.Ordenes); i++ {
		if p.Ordenes[i] <= p.Ordenes[i-1] {
			return ErrGobiernoUsuariosAdmin
		}
	}
	c, _, err := m.Configuracion()
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	g := c.Gobierno
	if p.Configuracion.Revision != g.Revision || p.Configuracion.Secuencia != g.Secuencia || p.Configuracion.Huella != g.HuellaSHA256 || !p.Configuracion.PublicadaEn.Equal(g.PublicadaEn) || !p.Configuracion.ExpiraEn.Equal(g.ExpiraEn) {
		return ErrGobiernoUsuariosAdmin
	}
	return nil
}

type acuseAuditoriaGobiernoUsuarios struct {
	AuditoriaRef     string    `json:"auditoria_ref"`
	Secuencia        uint64    `json:"secuencia"`
	Huella           string    `json:"huella_sha256"`
	Correlacion      string    `json:"correlacion_ref"`
	RegistradaEn     time.Time `json:"registrada_en"`
	SolicitudSHA256  string    `json:"solicitud_sha256,omitempty"`
	PlanSHA256       string    `json:"plan_sha256,omitempty"`
	PreimagenSHA256  string    `json:"preimagen_sha256,omitempty"`
	ClavesSHA256     string    `json:"claves_sha256,omitempty"`
	MaterialSHA256   string    `json:"material_sha256,omitempty"`
	ConfiguracionRef string    `json:"configuracion_ref,omitempty"`
	Replay           *bool     `json:"replay,omitempty"`
}

func (a acuseAuditoriaGobiernoUsuarios) valido(intento bool) bool {
	_, offset := a.RegistradaEn.Zone()
	ref := refAuditGobiernoUsuarios.MatchString(a.AuditoriaRef)
	if intento {
		ref = refIntentoGobiernoUsuarios.MatchString(a.AuditoriaRef)
	}
	return ref && a.Secuencia > 0 && a.Secuencia <= 9007199254740991 && shaGobiernoUsuarios.MatchString(a.Huella) && correlacionGobiernoUsuarios.MatchString(a.Correlacion) && !a.RegistradaEn.IsZero() && a.RegistradaEn.Year() >= 1 && a.RegistradaEn.Year() <= 9999 && offset == 0 && a.RegistradaEn.Nanosecond()%1000 == 0
}
func validarAcuseGobiernoUsuariosAdmin(raw []byte, plan, sha string, material []byte) (ConfirmacionGobiernoUsuariosAdmin, error) {
	var r ConfirmacionGobiernoUsuariosAdmin
	if !shaGobiernoUsuarios.MatchString(sha) {
		return r, ErrGobiernoUsuariosAdmin
	}
	if decodificarGobiernoUsuarios(raw, &r) != nil {
		return r, ErrGobiernoUsuariosAdmin
	}
	var i acuseAuditoriaGobiernoUsuarios
	if decodificarGobiernoUsuarios(r.AuditoriaIntento, &i) != nil || !i.valido(true) || i.SolicitudSHA256 != sha || i.PlanSHA256 != "" || i.PreimagenSHA256 != "" || i.ClavesSHA256 != "" || i.MaterialSHA256 != "" || i.ConfiguracionRef != "" || i.Replay != nil {
		return ConfirmacionGobiernoUsuariosAdmin{}, ErrGobiernoUsuariosAdmin
	}
	switch r.Estado {
	case "permitido":
		if r.Codigo != "gobierno_usuarios_registrado" && r.Codigo != "gobierno_usuarios_replay" {
			return ConfirmacionGobiernoUsuariosAdmin{}, ErrGobiernoUsuariosAdmin
		}
		var a acuseAuditoriaGobiernoUsuarios
		var p planGobiernoUsuariosAdmin
		h := sha256.Sum256(material)
		if decodificarGobiernoUsuarios(r.Recibo, &a) != nil || !a.valido(false) || decodificarGobiernoUsuarios([]byte(plan), &p) != nil || a.PlanSHA256 != sha || a.PreimagenSHA256 != p.PreimagenSHA256 || a.ConfiguracionRef != p.Configuracion.Revision || a.MaterialSHA256 != hex.EncodeToString(h[:]) || !shaGobiernoUsuarios.MatchString(a.ClavesSHA256) || a.Replay == nil || (*a.Replay != (r.Codigo == "gobierno_usuarios_replay")) || a.SolicitudSHA256 != "" || !strings.HasSuffix(a.AuditoriaRef, sha[:32]) {
			return ConfirmacionGobiernoUsuariosAdmin{}, ErrGobiernoUsuariosAdmin
		}
	case "denegado", "error":
		if r.Codigo != "gobierno_usuarios_"+r.Estado || !bytes.Equal(bytes.TrimSpace(r.Recibo), []byte("null")) {
			return ConfirmacionGobiernoUsuariosAdmin{}, ErrGobiernoUsuariosAdmin
		}
	default:
		return ConfirmacionGobiernoUsuariosAdmin{}, ErrGobiernoUsuariosAdmin
	}
	return r, nil
}
