package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func planGobiernoUsuariosPrueba(t *testing.T, m *MaterialUsuariosAdmin) (string, string) {
	t.Helper()
	c, _, err := m.Configuracion()
	if err != nil {
		t.Fatal(err)
	}
	p := planGobiernoUsuariosAdmin{Version: 1, OperacionRef: "gcu_" + strings.Repeat("x", 22), PreparadoEn: c.Gobierno.PublicadaEn, CaducaEn: c.Gobierno.ExpiraEn, PreimagenSHA256: strings.Repeat("a", 64), Ordenes: []uint64{101, 102}}
	p.Configuracion.Revision = c.Gobierno.Revision
	p.Configuracion.Secuencia = c.Gobierno.Secuencia
	p.Configuracion.Huella = c.Gobierno.HuellaSHA256
	p.Configuracion.PublicadaEn = c.Gobierno.PublicadaEn
	p.Configuracion.ExpiraEn = c.Gobierno.ExpiraEn
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	return string(b), hex.EncodeToString(h[:])
}
func TestGobiernoUsuariosPlanLigadoAlGobiernoPreparado(t *testing.T) {
	cfg, r := configuracionGobiernoUsuariosPrueba(t)
	m, err := PrepararMaterialUsuariosAdmin(t.Context(), cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	plan, sha := planGobiernoUsuariosPrueba(t, m)
	if validarPlanGobiernoUsuariosAdmin(plan, sha, m) != nil {
		t.Fatal("plan original rechazado")
	}
	var p planGobiernoUsuariosAdmin
	_ = json.Unmarshal([]byte(plan), &p)
	p.Configuracion.Huella = strings.Repeat("f", 64)
	b, _ := json.Marshal(p)
	h := sha256.Sum256(b)
	if validarPlanGobiernoUsuariosAdmin(string(b), hex.EncodeToString(h[:]), m) == nil {
		t.Fatal("SHA aprobado no liga otro gobierno al material preparado")
	}
}
func TestGobiernoUsuariosAcuseCompletoAntesCommit(t *testing.T) {
	cfg, r := configuracionGobiernoUsuariosPrueba(t)
	m, err := PrepararMaterialUsuariosAdmin(t.Context(), cfg, r)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	plan, sha := planGobiernoUsuariosPrueba(t, m)
	var p planGobiernoUsuariosAdmin
	_ = json.Unmarshal([]byte(plan), &p)
	material := []byte("material privado de prueba")
	mh := sha256.Sum256(material)
	replay := false
	base := acuseAuditoriaGobiernoUsuarios{Secuencia: 6251, Huella: strings.Repeat("b", 64), Correlacion: "correlacion_" + strings.Repeat("1", 32), RegistradaEn: time.Date(2026, 10, 4, 8, 31, 0, 123000, time.UTC)}
	i := base
	i.AuditoriaRef = "aud_v3_gui_" + strings.Repeat("2", 32)
	i.SolicitudSHA256 = sha
	a := base
	a.AuditoriaRef = "aud_v3_gu_" + sha[:32]
	a.PlanSHA256 = sha
	a.PreimagenSHA256 = p.PreimagenSHA256
	a.ClavesSHA256 = strings.Repeat("c", 64)
	a.MaterialSHA256 = hex.EncodeToString(mh[:])
	a.ConfiguracionRef = p.Configuracion.Revision
	a.Replay = &replay
	ar, _ := json.Marshal(a)
	ir, _ := json.Marshal(i)
	valid := ConfirmacionGobiernoUsuariosAdmin{Estado: "permitido", Codigo: "gobierno_usuarios_registrado", Recibo: ar, AuditoriaIntento: ir}
	raw, _ := json.Marshal(valid)
	if _, err := validarAcuseGobiernoUsuariosAdmin(raw, plan, sha, material); err != nil {
		t.Fatal("acuse original completo rechazado")
	}
	for _, caso := range []string{"intento_vacio", "recibo_vacio", "plan", "material", "fecha", "denegado_con_recibo", "sha_corto"} {
		t.Run(caso, func(t *testing.T) {
			x := valid
			sh := sha
			switch caso {
			case "intento_vacio":
				x.AuditoriaIntento = json.RawMessage(`{}`)
			case "recibo_vacio":
				x.Recibo = json.RawMessage(`{}`)
			case "plan":
				b := a
				b.PlanSHA256 = strings.Repeat("d", 64)
				x.Recibo, _ = json.Marshal(b)
			case "material":
				b := a
				b.MaterialSHA256 = strings.Repeat("d", 64)
				x.Recibo, _ = json.Marshal(b)
			case "fecha":
				b := a
				b.RegistradaEn = time.Time{}
				x.Recibo, _ = json.Marshal(b)
			case "denegado_con_recibo":
				x.Estado = "denegado"
				x.Codigo = "gobierno_usuarios_denegado"
			case "sha_corto":
				sh = "x"
			}
			raw, _ := json.Marshal(x)
			if _, err := validarAcuseGobiernoUsuariosAdmin(raw, plan, sh, material); err == nil {
				t.Fatal("acuse no ligado aceptado")
			}
		})
	}
}
