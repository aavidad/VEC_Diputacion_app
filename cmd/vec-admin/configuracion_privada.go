package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var errConfiguracionPrivadaPerfiles = errors.New("configuracion_privada_admin_invalida")

const limiteConfiguracionPrivadaPerfiles = 256 << 10

type poolsPerfilesPrivados struct {
	FuenteAutorizacion   string `json:"fuente_autorizacion"`
	RegistroAutorizacion string `json:"registro_autorizacion"`
	Motivos              string `json:"motivos"`
	RegistroSesiones     string `json:"registro_sesiones"`
	RevalidacionSesiones string `json:"revalidacion_sesiones"`
	CuentasADMIN         string `json:"cuentas_admin"`
	ActosADMIN           string `json:"actos_admin"`
	AuditoriaFrontera    string `json:"auditoria_frontera"`
}

type firmantePerfilesPrivado struct {
	ClaveID               string `json:"clave_id"`
	Audiencia             string `json:"audiencia"`
	PrefijoEvidencia      string `json:"prefijo_evidencia"`
	ClavePrivadaArchivo   string `json:"clave_privada_archivo"`
	PublicaEsperadaBase64 string `json:"publica_esperada_base64"`
}

type identidadPerfilesPrivada struct {
	DirectorioMaterial     string `json:"directorio_material"`
	RutaConfiguracionHMAC  string `json:"ruta_configuracion_hmac"`
	EspacioIdentidad       string `json:"espacio_identidad"`
	DominioRef             string `json:"dominio_ref"`
	EspacioClave           string `json:"espacio_clave"`
	DominioHMAC            string `json:"dominio_hmac"`
	IncluirCuentaOrdinaria bool   `json:"incluir_cuenta_ordinaria"`
}

// ConfianzaJSON contiene metadatos públicos y rutas al material privado. La
// composición los convierte al contrato tipado y lee las claves por su fuente.
type configuracionPerfilesPrivada struct {
	Pools                    poolsPerfilesPrivados                       `json:"pools"`
	ConfianzaJSON            json.RawMessage                             `json:"confianza"`
	Firmante                 firmantePerfilesPrivado                     `json:"firmante"`
	Identidad                identidadPerfilesPrivada                    `json:"identidad"`
	MotivosLectura           map[string]domain.ReferenciaEntradaCatalogo `json:"motivos_lectura"`
	CatalogoMotivosID        string                                      `json:"catalogo_motivos_id"`
	VigenciaDecisionSegundos int                                         `json:"vigencia_decision_segundos"`
	TimeoutArranqueSegundos  int                                         `json:"timeout_arranque_segundos"`
	ActivosDirectorio        string                                      `json:"activos_directorio"`
}

func cargarConfiguracionPerfilesPrivada(ruta string) (configuracionPerfilesPrivada, error) {
	var cfg configuracionPerfilesPrivada
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return cfg, errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	if decodificarConfiguracionPrivada(b, &cfg) != nil || validarConfiguracionPerfilesPrivada(cfg) != nil {
		return configuracionPerfilesPrivada{}, errConfiguracionPrivadaPerfiles
	}
	return cfg, nil
}

func validarConfiguracionPerfilesPrivada(c configuracionPerfilesPrivada) error {
	rutas := []string{c.Pools.FuenteAutorizacion, c.Pools.RegistroAutorizacion, c.Pools.Motivos, c.Pools.RegistroSesiones, c.Pools.RevalidacionSesiones, c.Pools.CuentasADMIN, c.Pools.AuditoriaFrontera}
	if c.Pools.ActosADMIN != "" {
		rutas = append(rutas, c.Pools.ActosADMIN)
	}
	vistas := map[string]bool{}
	for _, p := range rutas {
		if !rutaPrivadaPerfilesValida(p) || vistas[p] {
			return errConfiguracionPrivadaPerfiles
		}
		vistas[p] = true
	}
	if c.VigenciaDecisionSegundos <= 0 || int64(c.VigenciaDecisionSegundos) > math.MaxInt64/int64(time.Second) || c.TimeoutArranqueSegundos <= 0 || c.TimeoutArranqueSegundos > 60 || !textoConfiguracionPerfiles(c.CatalogoMotivosID) || len(c.MotivosLectura) == 0 || len(c.MotivosLectura) > 32 {
		return errConfiguracionPrivadaPerfiles
	}
	for accion, m := range c.MotivosLectura {
		if !textoConfiguracionPerfiles(accion) || m.Validar() != nil || m.CatalogoID != c.CatalogoMotivosID {
			return errConfiguracionPrivadaPerfiles
		}
	}
	for _, s := range []string{c.Firmante.ClaveID, c.Firmante.Audiencia, c.Firmante.PrefijoEvidencia, c.Firmante.PublicaEsperadaBase64, c.Identidad.EspacioIdentidad, c.Identidad.DominioRef, c.Identidad.EspacioClave, c.Identidad.DominioHMAC} {
		if !textoConfiguracionPerfiles(s) {
			return errConfiguracionPrivadaPerfiles
		}
	}
	if !rutaPrivadaPerfilesValida(c.Firmante.ClavePrivadaArchivo) || !rutaPrivadaPerfilesValida(c.Identidad.RutaConfiguracionHMAC) || validarDirectorioPrivadoPerfiles(c.Identidad.DirectorioMaterial) != nil || validarDirectorioPrivadoPerfiles(c.ActivosDirectorio) != nil {
		return errConfiguracionPrivadaPerfiles
	}
	publica, err := base64.StdEncoding.Strict().DecodeString(c.Firmante.PublicaEsperadaBase64)
	if err != nil || len(publica) != 32 {
		return errConfiguracionPrivadaPerfiles
	}
	trust, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil || trust.Raiz.PublicaBase64 != c.Firmante.PublicaEsperadaBase64 || trust.Raiz.ClaveID != c.Firmante.ClaveID || trust.Raiz.Audiencia != c.Firmante.Audiencia {
		return errConfiguracionPrivadaPerfiles
	}
	// Las claves privadas sólo pueden llegar a la composición por archivos. El
	// formato exacto de los metadatos lo valida el constructor V3 central.
	if contieneClavePrivadaInline(c.ConfianzaJSON) {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

func contieneClavePrivadaInline(b []byte) bool {
	var v any
	if json.Unmarshal(b, &v) != nil {
		return true
	}
	var recorrer func(any) bool
	recorrer = func(x any) bool {
		switch y := x.(type) {
		case nil:
			return true
		case map[string]any:
			for k, v := range y {
				n := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(k))
				switch n {
				case "material", "materialbase64", "materialbytes", "semilla", "semillabase64", "claveprivada", "claveprivadabase64", "privatekey", "hmackey", "clavehmac", "secret":
					return true
				}
				if recorrer(v) {
					return true
				}
			}
		case []any:
			for _, v := range y {
				if recorrer(v) {
					return true
				}
			}
		}
		return false
	}
	return recorrer(v)
}

func textoConfiguracionPerfiles(s string) bool {
	if s == "" || len(s) > 512 || strings.TrimSpace(s) != s || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

func rutaPrivadaPerfilesValida(p string) bool {
	return filepath.IsAbs(p) && filepath.Clean(p) == p && !strings.ContainsRune(p, 0)
}
func archivoPropioPerfiles(i os.FileInfo) bool {
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && int64(s.Uid) == int64(os.Getuid())
}
func fueraGitPerfiles(p string) bool {
	for a := p; ; a = filepath.Dir(a) {
		if _, err := os.Lstat(filepath.Join(a, ".git")); err == nil || !errors.Is(err, os.ErrNotExist) {
			return false
		}
		if a == "/" {
			return true
		}
	}
}
func validarDirectorioPrivadoPerfiles(p string) error {
	if !rutaPrivadaPerfilesValida(p) {
		return errConfiguracionPrivadaPerfiles
	}
	resuelta, err := filepath.EvalSymlinks(p)
	if err != nil || resuelta != p {
		return errConfiguracionPrivadaPerfiles
	}
	i, err := os.Lstat(p)
	if err != nil || !i.IsDir() || i.Mode().Perm() != 0700 || !archivoPropioPerfiles(i) || !fueraGitPerfiles(p) {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

func leerArchivoPrivadoPerfiles(ruta string) ([]byte, error) {
	if !rutaPrivadaPerfilesValida(ruta) {
		return nil, errConfiguracionPrivadaPerfiles
	}
	padre := filepath.Dir(ruta)
	if validarDirectorioPrivadoPerfiles(padre) != nil {
		return nil, errConfiguracionPrivadaPerfiles
	}
	previo, err := os.Lstat(padre)
	if err != nil {
		return nil, errConfiguracionPrivadaPerfiles
	}
	raiz, err := os.OpenRoot(padre)
	if err != nil {
		return nil, errConfiguracionPrivadaPerfiles
	}
	defer raiz.Close()
	actual, err := raiz.Stat(".")
	if err != nil || !os.SameFile(previo, actual) || actual.Mode().Perm() != 0700 || !archivoPropioPerfiles(actual) {
		return nil, errConfiguracionPrivadaPerfiles
	}
	i, err := raiz.Lstat(filepath.Base(ruta))
	if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 || !archivoPropioPerfiles(i) || i.Size() <= 0 || i.Size() > limiteConfiguracionPrivadaPerfiles {
		return nil, errConfiguracionPrivadaPerfiles
	}
	stat, ok := i.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return nil, errConfiguracionPrivadaPerfiles
	}
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, errConfiguracionPrivadaPerfiles
	}
	defer f.Close()
	abierto, err := f.Stat()
	if err != nil || !os.SameFile(i, abierto) {
		return nil, errConfiguracionPrivadaPerfiles
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteConfiguracionPrivadaPerfiles+1))
	if err != nil || int64(len(b)) != i.Size() {
		clear(b)
		return nil, errConfiguracionPrivadaPerfiles
	}
	despues, err := f.Stat()
	if err != nil || !os.SameFile(abierto, despues) || despues.Size() != abierto.Size() || !despues.ModTime().Equal(abierto.ModTime()) || despues.Mode().Perm() != 0600 || !archivoPropioPerfiles(despues) {
		clear(b)
		return nil, errConfiguracionPrivadaPerfiles
	}
	ultimo, ok := despues.Sys().(*syscall.Stat_t)
	if !ok || ultimo.Nlink != 1 {
		clear(b)
		return nil, errConfiguracionPrivadaPerfiles
	}
	return b, nil
}

func decodificarConfiguracionPrivada(b []byte, destino any) error {
	if len(b) == 0 || len(b) > limiteConfiguracionPrivadaPerfiles || clavesUnicasConfiguracionPerfiles(b) != nil || clavesExactasConfiguracionPerfiles(b, reflect.TypeOf(destino).Elem()) != nil {
		return errConfiguracionPrivadaPerfiles
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return errConfiguracionPrivadaPerfiles
	}
	if d.Decode(new(any)) != io.EOF {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}
func clavesExactasConfiguracionPerfiles(b []byte, t reflect.Type) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return errConfiguracionPrivadaPerfiles
	}
	if t == reflect.TypeOf(json.RawMessage{}) {
		var x map[string]json.RawMessage
		if json.Unmarshal(b, &x) != nil || x == nil {
			return errConfiguracionPrivadaPerfiles
		}
		return nil
	}
	if t == reflect.TypeOf(time.Time{}) {
		var fecha time.Time
		if json.Unmarshal(b, &fecha) != nil {
			return errConfiguracionPrivadaPerfiles
		}
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		var x map[string]json.RawMessage
		if json.Unmarshal(b, &x) != nil || len(x) != t.NumField() {
			return errConfiguracionPrivadaPerfiles
		}
		for i := range t.NumField() {
			f := t.Field(i)
			v, ok := x[f.Tag.Get("json")]
			if !ok || clavesExactasConfiguracionPerfiles(v, f.Type) != nil {
				return errConfiguracionPrivadaPerfiles
			}
		}
	case reflect.Slice:
		var x []json.RawMessage
		if json.Unmarshal(b, &x) != nil || x == nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, v := range x {
			if clavesExactasConfiguracionPerfiles(v, t.Elem()) != nil {
				return errConfiguracionPrivadaPerfiles
			}
		}
	case reflect.Map:
		var x map[string]json.RawMessage
		if json.Unmarshal(b, &x) != nil || x == nil {
			return errConfiguracionPrivadaPerfiles
		}
		for _, v := range x {
			if clavesExactasConfiguracionPerfiles(v, t.Elem()) != nil {
				return errConfiguracionPrivadaPerfiles
			}
		}
	}
	return nil
}
func clavesUnicasConfiguracionPerfiles(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var recorrer func(int) error
	recorrer = func(nivel int) error {
		if nivel > 32 {
			return errConfiguracionPrivadaPerfiles
		}
		t, err := d.Token()
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		switch t {
		case json.Delim('{'):
			vistas := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				s, ok := k.(string)
				if err != nil || !ok || vistas[s] {
					return errConfiguracionPrivadaPerfiles
				}
				vistas[s] = true
				if recorrer(nivel+1) != nil {
					return errConfiguracionPrivadaPerfiles
				}
			}
			_, err = d.Token()
		case json.Delim('['):
			for d.More() {
				if recorrer(nivel+1) != nil {
					return errConfiguracionPrivadaPerfiles
				}
			}
			_, err = d.Token()
		}
		if err != nil {
			return errConfiguracionPrivadaPerfiles
		}
		return nil
	}
	if recorrer(0) != nil {
		return errConfiguracionPrivadaPerfiles
	}
	if _, err := d.Token(); err != io.EOF {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}
