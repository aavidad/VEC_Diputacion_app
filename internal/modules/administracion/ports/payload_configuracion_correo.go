package ports

import (
	"encoding/json"
	"errors"

	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// PayloadNegocioConfiguracionCorreo es la única preimagen admitida por la
// preparación, la autorización V3 y ADM1. No contiene secreto claro.
func PayloadNegocioConfiguracionCorreo(p PreparacionConfiguracionCorreo, auditoria vecdomain.AuditEntry) ([]byte, error) {
	if p.Entrada.SecretoNuevo != nil || p.Entrada.Validar() != nil || auditoria.ActorID == "" || auditoria.Action == "" || auditoria.ModuleID == "" || auditoria.SubjectRef == "" || auditoria.Result == "" || auditoria.OccurredAt.IsZero() {
		return nil, errors.New("preparacion correo invalida")
	}
	type cfg struct {
		Host      string                              `json:"host"`
		Puerto    uint16                              `json:"puerto"`
		Nombre    string                              `json:"server_name"`
		CA        string                              `json:"referencia_ca"`
		Remitente string                              `json:"remitente_fijo"`
		Usuario   string                              `json:"usuario"`
		TLS       admindomain.ModoTLSCorreo           `json:"modo_tls"`
		Auth      admindomain.ModoAutenticacionCorreo `json:"modo_autenticacion"`
		Tiempo    int64                               `json:"tiempo_maximo_ms"`
		Version   uint64                              `json:"version_esperada"`
	}
	type sobre struct {
		Version uint64 `json:"version"`
		Clave   string `json:"clave_ref"`
		Nonce   []byte `json:"nonce"`
		Cifrado []byte `json:"cifrado"`
		Huella  string `json:"huella_aad_sha256"`
	}
	var s *sobre
	if p.Sustituir {
		if p.SobreNuevo.Version != p.Entrada.VersionEsperada+1 || p.SobreNuevo.ClaveRef == "" || len(p.SobreNuevo.Nonce) == 0 || len(p.SobreNuevo.Cifrado) == 0 || len(p.HuellaAADSHA256) != 64 {
			return nil, errors.New("sobre correo invalido")
		}
		s = &sobre{p.SobreNuevo.Version, p.SobreNuevo.ClaveRef, append([]byte(nil), p.SobreNuevo.Nonce...), append([]byte(nil), p.SobreNuevo.Cifrado...), p.HuellaAADSHA256}
	}
	v := p.Entrada.VistaConfiguracionCorreo
	return json.Marshal(struct {
		Esquema   string               `json:"esquema"`
		Config    cfg                  `json:"configuracion"`
		Sobre     *sobre               `json:"sobre_secreto"`
		Auditoria vecdomain.AuditEntry `json:"auditoria"`
	}{"vec.administracion.configuracion-correo.persistencia.v1", cfg{v.Host, v.Puerto, v.NombreServidor, v.ReferenciaCA, v.RemitenteFijo, v.Usuario, v.ModoTLS, v.ModoAutenticacion, v.TiempoMaximoMillis, p.Entrada.VersionEsperada}, s, auditoria})
}
