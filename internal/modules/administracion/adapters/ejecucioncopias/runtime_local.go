package ejecucioncopias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	aislamiento "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

var ErrRuntimeLocal = errors.New("copias_ejecucion_runtime_no_comprobable")

var sha256Runtime = regexp.MustCompile(`^[a-f0-9]{64}$`)

func huellaRuntime(s string) bool { return sha256Runtime.MatchString(s) }

// ConfiguracionRuntimeLocal procede de un fichero privado del operador, fuera
// del conjunto y de Git. Sus referencias nombran componentes ya autenticados;
// ninguna ruta del host o comando de la copia se convierte en instrucción.
type ConfiguracionRuntimeLocal struct {
	BinarioID          string            `json:"binario_id"`
	MaterialID         string            `json:"material_id"`
	CatalogoPersonalID string            `json:"catalogo_personal_id"`
	Perfil             string            `json:"perfil"`
	Puerto             uint16            `json:"puerto"`
	UsuarioPostgreSQL  string            `json:"usuario_postgresql"`
	Entorno            map[string]string `json:"entorno"`
}

type RuntimeLocal struct {
	Configuracion ConfiguracionRuntimeLocal
	configSHA     string
}

// CargarRuntimeLocal exige contenido regular, propio y 0600. El error nunca
// incorpora DSN, certificados, rutas privadas ni el contenido del fichero.
func CargarRuntimeLocal(ruta string) (*RuntimeLocal, error) {
	if !path.IsAbs(ruta) || strings.ContainsRune(ruta, 0) {
		return nil, ErrRuntimeLocal
	}
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- configuración local indicada al ejecutor; fstat y límites a continuación.
	if err != nil {
		return nil, ErrRuntimeLocal
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Size() < 2 || st.Size() > 1<<20 {
		return nil, ErrRuntimeLocal
	}
	sys, ok := st.Sys().(*syscall.Stat_t)
	if !ok || sys.Uid != uint32(os.Geteuid()) || sys.Nlink != 1 { // #nosec G115 -- UID Linux corresponde a uid_t de 32 bits.
		return nil, ErrRuntimeLocal
	}
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		clear(b)
		return nil, ErrRuntimeLocal
	}
	defer clear(b)
	var c ConfiguracionRuntimeLocal
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF {
		return nil, ErrRuntimeLocal
	}
	h := sha256.Sum256(b)
	return &RuntimeLocal{Configuracion: c, configSHA: hex.EncodeToString(h[:])}, nil
}

func componenteRuntime(e aislamiento.Entorno, id, tipo string) (aislamiento.Componente, bool) {
	for _, c := range e.Componentes {
		if c.ID == id && c.Tipo == tipo && strings.HasPrefix(c.RutaInterna, "/componentes/") && path.Clean(c.RutaInterna) == c.RutaInterna && huellaRuntime(c.SHA256) {
			return c, true
		}
	}
	return aislamiento.Componente{}, false
}

func artefactoRuntime(m copias.Manifiesto, id, tipo, sha string) bool {
	for _, a := range m.Componentes {
		if a.ID == id && a.Tipo == tipo && a.SHA256 == sha && a.TamanoBytes > 0 {
			return true
		}
	}
	return false
}

func entornoRuntime(c ConfiguracionRuntimeLocal, material, catalogo string) (map[string]string, bool) {
	if c.Puerto == 0 || c.Perfil != "sesion_sintetica" || c.BinarioID == "" || c.MaterialID == "" || c.CatalogoPersonalID == "" || c.BinarioID == c.MaterialID || c.BinarioID == c.CatalogoPersonalID || c.MaterialID == c.CatalogoPersonalID || !regexp.MustCompile(`^[a-z_][a-z0-9_]{1,62}$`).MatchString(c.UsuarioPostgreSQL) || len(c.Entorno) != 0 {
		return nil, false
	}
	// env -i en CS06 y red none impiden heredar transportes. El perfil de
	// sesión no abre conexiones funcionales a PostgreSQL ni despachadores.
	e := map[string]string{
		"VEC_EXECUTION_PROFILE":            "desarrollo",
		"VEC_AUTH_MODE":                    "desarrollo",
		"VEC_DEVELOPMENT_GUARD":            "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO",
		"VEC_DEVELOPMENT_MATERIAL_DIR":     material,
		"VEC_TLS_CERT_FILE":                path.Join(material, "tls/servidor.crt"),
		"VEC_TLS_KEY_FILE":                 path.Join(material, "tls/servidor.key"),
		"VEC_HTTP_ADDR":                    "127.0.0.1:" + strconv.Itoa(int(c.Puerto)),
		"VEC_HTTP_ALLOWED_CIDRS":           "127.0.0.1/32",
		"VEC_PORTAL_PROCESO":               "interno",
		"VEC_PERSONAL_CATALOG_PATH":        catalogo,
		"VEC_BOLSA_PUBLIC_SOURCE_PATH":     path.Join(material, "data/demo/convocatorias_publicas.demo.json"),
		"VEC_BOLSA_CATEGORIES_SOURCE_PATH": path.Join(material, "data/catalogos/categorias-profesionales/v1.demo.json"),
	}
	return e, true
}

// Arrancar ejecuta el vec-server archivado en el aislamiento CS06. Este
// perfil comprueba sesión mTLS; el contraste de datos PostgreSQL es separado.
func (r *RuntimeLocal) Arrancar(ctx context.Context, e aislamiento.Entorno, m copias.Manifiesto) (ResultadoArranque, error) {
	fallo := func() (ResultadoArranque, error) { return ResultadoArranque{}, ErrRuntimeLocal }
	if r == nil || ctx == nil || ctx.Err() != nil || e.PostgreSQL == nil || e.Archivado == nil || !e.Aislamiento.RedSinSalida || !e.Aislamiento.VolumenesPropios || !e.Aislamiento.ConfiguracionOrigenExcluida || !e.Aislamiento.RuntimeInspeccionado || !huellaRuntime(r.configSHA) || len(m.Inventario.PostgreSQL.Bases) != 1 || len(m.Inventario.Modulos) != 1 || m.Inventario.Modulos[0].ID != "administracion" {
		return fallo()
	}
	c := r.Configuracion
	binario, ok := componenteRuntime(e, c.BinarioID, "binario")
	if !ok || !artefactoRuntime(m, binario.ID, binario.Tipo, binario.SHA256) {
		return fallo()
	}
	// El tar autenticado y el binario que se ejecutará son dos huellas
	// distintas. El lector CS06 comprueba los bytes del archivo extraído.
	if !huellaRuntime(binario.ContenidoSHA256) || binario.ContenidoBytes < 1 || binario.ContenidoBytes > 64<<20 {
		return fallo()
	}
	archivado, err := e.Archivado.LeerArchivado(ctx, binario.ID, 64<<20)
	if err != nil || int64(len(archivado)) != binario.ContenidoBytes {
		clear(archivado)
		return fallo()
	}
	binarioReal := sha256.Sum256(archivado)
	clear(archivado)
	if hex.EncodeToString(binarioReal[:]) != binario.ContenidoSHA256 {
		return fallo()
	}
	binarioEnRelease := false
	for _, a := range m.Inventario.Release.Binarios {
		if "fisica:"+a.ID == binario.ID && a.Tipo == binario.Tipo && a.SHA256 == binario.ContenidoSHA256 && a.TamanoBytes == binario.ContenidoBytes {
			binarioEnRelease = true
		}
	}
	if !binarioEnRelease {
		return fallo()
	}
	material, ok := componenteRuntime(e, c.MaterialID, "material")
	if !ok || !artefactoRuntime(m, material.ID, material.Tipo, material.SHA256) {
		return fallo()
	}
	materialEnRelease := false
	for _, a := range m.Inventario.Release.Componentes {
		if "fisica:"+a.ID == material.ID && a.Tipo == material.Tipo && a.SHA256 == material.ContenidoSHA256 && a.TamanoBytes == material.ContenidoBytes {
			materialEnRelease = true
		}
	}
	if !materialEnRelease || material.RutaMaterial == "" || !strings.HasPrefix(material.RutaMaterial, "/componentes/") || path.Clean(material.RutaMaterial) != material.RutaMaterial {
		return fallo()
	}
	catalogo, ok := componenteRuntime(e, c.CatalogoPersonalID, "catalogos")
	if !ok || !artefactoRuntime(m, catalogo.ID, catalogo.Tipo, catalogo.SHA256) || !huellaRuntime(catalogo.ContenidoSHA256) || catalogo.ContenidoBytes < 1 {
		return fallo()
	}
	catalogoEnRelease := false
	for _, a := range m.Inventario.Release.Componentes {
		if "fisica:"+a.ID == catalogo.ID && a.Tipo == catalogo.Tipo && a.SHA256 == catalogo.ContenidoSHA256 && a.TamanoBytes == catalogo.ContenidoBytes {
			catalogoEnRelease = true
		}
	}
	if !catalogoEnRelease {
		return fallo()
	}
	entorno, ok := entornoRuntime(c, material.RutaMaterial, catalogo.RutaInterna)
	if !ok {
		return fallo()
	}
	aislamientoAntes, err := e.PostgreSQL.ComprobarExclusion(ctx)
	if err != nil || !huellaRuntime(aislamientoAntes) {
		return fallo()
	}
	// CS06 arranca PostgreSQL con network=none y read-only por defecto. Esta
	// observación se hace contra el servidor efectivo antes de iniciar VEC.
	consulta := "SELECT current_setting('default_transaction_read_only'),current_setting('listen_addresses'),current_setting('archive_mode')"
	b, err := e.PostgreSQL.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", c.UsuarioPostgreSQL, "-d", "postgres", "-c", consulta}, nil, 1024)
	if err != nil || strings.TrimSpace(string(b)) != "on||off" {
		return fallo()
	}
	proceso, err := e.Archivado.IniciarArchivado(ctx, binario.ID, nil, entorno)
	if err != nil {
		return fallo()
	}
	defer func() {
		parada, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancelar()
		_ = proceso.Detener(parada)
	}()
	ca := path.Join(material.RutaMaterial, "ca/ca.crt")
	cert := path.Join(material.RutaMaterial, "mtls/cliente.crt")
	key := path.Join(material.RutaMaterial, "mtls/cliente.key")
	sonda := aislamiento.SondaHTTP{Puerto: c.Puerto, TLS: true, CA: ca, Certificado: cert, Clave: key, LimiteBytes: 1 << 16}
	var salud aislamiento.RespuestaHTTP
	var listo bool
	for intento := 0; intento < 60; intento++ {
		sonda.Ruta = "/livez"
		salud, err = proceso.SondarHTTP(ctx, sonda)
		if err == nil && salud.EstadoHTTP == 200 && salud.Bytes > 0 && huellaRuntime(salud.SHA256) {
			listo = true
			break
		}
		select {
		case <-ctx.Done():
			return fallo()
		case <-time.After(100 * time.Millisecond):
		}
	}
	if !listo {
		return fallo()
	}
	sonda.Ruta = "/api/vec/session"
	sesion, err := proceso.SondarHTTP(ctx, sonda)
	if err != nil || sesion.EstadoHTTP != 200 || sesion.Bytes < 16 || !huellaRuntime(sesion.SHA256) || !sesionAutorizadaRuntime(sesion.Contenido) {
		return fallo()
	}
	aislamientoDespues, err := e.PostgreSQL.ComprobarExclusion(ctx)
	if err != nil || aislamientoAntes != aislamientoDespues {
		return fallo()
	}
	// ConsultaDatosComprobada permanece false: este perfil sólo comprueba sesión.
	// Los cuatro sellos ligan bytes reales del binario, manifiesto, configuración
	// privada, sondas mTLS y restricción de red/PG observada.
	h := sha256.New()
	for _, s := range []string{binario.ContenidoSHA256, binario.SHA256, material.ContenidoSHA256, material.SHA256, catalogo.ContenidoSHA256, catalogo.SHA256, r.configSHA, m.InventarioSHA256, aislamientoAntes, salud.SHA256, sesion.SHA256} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}
	ref := hex.EncodeToString(h.Sum(nil))
	return ResultadoArranque{Ref: "arranque:" + ref, SaludRef: "salud:" + salud.SHA256, ConsultaRef: "consulta:" + sesion.SHA256, DespachosRef: "bloqueo:" + aislamientoAntes}, nil
}

func sesionAutorizadaRuntime(b []byte) bool {
	if len(b) < 16 || len(b) > 1<<16 {
		return false
	}
	var v struct {
		Principal struct {
			ID            string   `json:"id"`
			Permissions   []string `json:"permissions"`
			AuthAssurance string   `json:"auth_assurance"`
			AuthMethod    string   `json:"auth_method"`
		} `json:"principal"`
	}
	if json.Unmarshal(b, &v) != nil || v.Principal.ID == "" || v.Principal.AuthAssurance != "alto" || (v.Principal.AuthMethod != "dnie" && v.Principal.AuthMethod != "certificado") {
		return false
	}
	for _, p := range v.Principal.Permissions {
		if p == "vec.session.read" {
			return true
		}
	}
	return false
}
