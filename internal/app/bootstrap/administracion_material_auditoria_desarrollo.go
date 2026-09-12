package bootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/config"
	registroaccesos "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	seguridaddocumentos "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const esquemaMaterialAuditoriaAdministracionDesarrolloV1 = "vec.administracion.auditoria-hmac.v1"

// archivoMaterialAuditoriaAdministracionDesarrollo es deliberadamente mínimo:
// la referencia identifica la clave sin revelarla y la clave sólo existe en el
// fichero privado preparado fuera de Git.
type archivoMaterialAuditoriaAdministracionDesarrollo struct {
	Esquema     string `json:"esquema"`
	ClaveRef    string `json:"clave_ref"`
	ClaveBase64 string `json:"clave_base64"`
}

// nuevoSeudonimizadorAuditoriaAdministracionDesarrollo carga la clave propia
// de T13. No genera material ni admite la degradación a claves de sesión,
// idempotencia o KMS.
func nuevoSeudonimizadorAuditoriaAdministracionDesarrollo(
	cfg config.Config,
) (vecports.SeudonimizadorSujetoAlmacen, error) {
	fallo := ErrConfiguracionCorreoAdministracionNoDisponible
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
		return nil, fallo
	}
	base := filepath.Join(cfg.DevelopmentMaterialDir, "administracion")
	if dentroDeRepositorioGit(base) || validarArbolMaterialDesarrollo(base) != nil {
		return nil, fallo
	}
	antes, err := os.Lstat(base)
	if err != nil || !antes.IsDir() {
		return nil, fallo
	}
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return nil, fallo
	}
	defer raiz.Close()
	despues, err := raiz.Stat(".")
	if err != nil || !os.SameFile(antes, despues) {
		return nil, fallo
	}

	contenido, err := leerArchivoIncorporacionV2(raiz, "auditoria-hmac.json", 4<<10)
	if err != nil {
		return nil, fallo
	}
	defer borrarBytes(contenido)
	if validarClavesJSONUnicas(contenido) != nil {
		return nil, fallo
	}
	var archivo archivoMaterialAuditoriaAdministracionDesarrollo
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if decodificador.Decode(&archivo) != nil ||
		!errors.Is(decodificador.Decode(new(any)), io.EOF) ||
		archivo.Esquema != esquemaMaterialAuditoriaAdministracionDesarrolloV1 ||
		!referenciaClaveAuditoriaAdministracionValida(archivo.ClaveRef) {
		return nil, fallo
	}
	clave, err := base64.StdEncoding.Strict().DecodeString(archivo.ClaveBase64)
	if err != nil || len(clave) != sha256.Size || bytes.Equal(clave, make([]byte, sha256.Size)) {
		borrarBytes(clave)
		return nil, fallo
	}
	defer borrarBytes(clave)
	sellador, err := seguridaddocumentos.NuevoSelladorHMAC(archivo.ClaveRef, clave)
	if err != nil {
		return nil, fallo
	}
	return sellador, nil
}

// referenciaClaveAuditoriaAdministracionValida reutiliza la gramática exacta
// de T13 sin permitir prefijos de los otros dominios de material sensible.
func referenciaClaveAuditoriaAdministracionValida(referencia string) bool {
	if referencia == "" || referencia != strings.TrimSpace(referencia) ||
		!strings.HasPrefix(referencia, "administracion_auditoria_t13_") ||
		strings.TrimPrefix(referencia, "administracion_auditoria_t13_") == "" {
		return false
	}
	return registroaccesos.ActorSeudonimizadoValido(
		"hmac-sha256:" + referencia + ":" + strings.Repeat("a", sha256.Size*2),
	)
}
