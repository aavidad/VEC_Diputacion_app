package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type documentoPreparado struct {
	Esquema       string                               `json:"esquema"`
	Estado        string                               `json:"estado"`
	AlcanceFuente string                               `json:"alcance_fuente"`
	Preparacion   ports.PreparacionDenominacionPersona `json:"preparacion"`
	// Estos bytes son json.Marshal(Sobre) original, conservados en base64.
	SobreCanonico []byte `json:"sobre_canonico"`
}
type diagnostico struct {
	Codigo      string `json:"codigo"`
	Mensaje     string `json:"mensaje,omitempty"`
	Reutilizada bool   `json:"reutilizada"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }
func ejecutar(args []string, salida, fallos io.Writer) int {
	f := flag.NewFlagSet("vec-preparar-denominacion-persona", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var cfg, nombreRuta, maestraRuta, destino, persona, procedencia, ambito, textos, idioma string
	var esperada uint64
	var sintetico bool
	f.StringVar(&cfg, "configuracion", "", "")
	f.StringVar(&nombreRuta, "nombre-fichero", "", "")
	f.StringVar(&maestraRuta, "maestra-fichero", "", "")
	f.StringVar(&destino, "salida", "", "")
	f.StringVar(&persona, "persona-ref", "", "")
	f.StringVar(&procedencia, "procedencia-ref", "", "")
	f.StringVar(&ambito, "ambito-ref", "", "")
	f.Uint64Var(&esperada, "version-esperada", 0, "")
	f.StringVar(&textos, "textos", "", "")
	f.StringVar(&idioma, "idioma", "", "")
	f.BoolVar(&sintetico, "sintetico", false, "")
	errParse := f.Parse(args)
	catalog, err := cargarTextos(textos, idioma)
	emitir := func(w io.Writer, codigo string, reutilizada bool, exit int) int {
		d := diagnostico{Codigo: codigo, Reutilizada: reutilizada}
		if catalog != nil {
			d.Mensaje = catalog.T(idioma, codigo)
		}
		b, e := json.Marshal(d)
		if e == nil {
			_, _ = w.Write(append(b, '\n'))
		}
		return exit
	}
	fallo := func(c string) int { return emitir(fallos, c, false, 1) }
	if err != nil {
		return fallo("denominacion.catalogo_no_disponible")
	}
	versionExplicita := false
	f.Visit(func(x *flag.Flag) {
		if x.Name == "version-esperada" {
			versionExplicita = true
		}
	})
	if errParse != nil || f.NArg() != 0 || !sintetico || !versionExplicita || !domain.ReferenciaPersonaDenominacionValida(persona) || len(persona) > 128 || esperada >= 1<<53-1 || !refProcedenciaCA32(procedencia) || !refOpaca(ambito) || !rutasSeparadas(cfg, nombreRuta, maestraRuta, destino, textos) {
		return fallo("denominacion.uso_invalido")
	}
	fuente, err := bootstrap.NuevaFuenteConfiguracionPrivadaDenominacionPersona(cfg)
	if err != nil {
		return fallo("denominacion.configuracion_no_disponible")
	}
	k, err := bootstrap.LeerArchivoPrivadoDenominacionPersona(maestraRuta, 32)
	if err != nil || len(k) != 32 {
		clear(k)
		return fallo("denominacion.material_no_disponible")
	}
	var maestra [32]byte
	copy(maestra[:], k)
	clear(k)
	defer clear(maestra[:])
	protector, indiceRef, err := bootstrap.NuevoProtectorDenominacionPersonaDesarrollo(maestra, fuente, time.Now)
	if err != nil {
		return fallo("denominacion.configuracion_no_disponible")
	}
	nombre, err := bootstrap.LeerArchivoPrivadoDenominacionPersona(nombreRuta, 4096)
	if err != nil {
		return fallo("denominacion.nombre_no_disponible")
	}
	defer clear(nombre)
	// Rutas diferentes pueden contener el mismo material: comparar los datos,
	// nunca la identidad del fichero o su directorio.
	if len(nombre) == len(maestra) && subtle.ConstantTimeCompare(nombre, maestra[:]) == 1 {
		return fallo("denominacion.nombre_no_disponible")
	}
	raiz, err := bootstrap.AbrirRaizPrivadaDenominacionPersona(destino)
	if err != nil {
		return fallo("denominacion.salida_no_disponible")
	}
	defer raiz.Close()
	base := filepath.Base(destino)
	// Un reintento consume sólo el artefacto original: no vuelve a crear nonce.
	if _, err := raiz.Lstat(base); err == nil {
		b, e := bootstrap.LeerEnRaizPrivadaDenominacionPersona(raiz, base, 65536)
		if e != nil {
			return fallo("denominacion.salida_no_disponible")
		}
		defer clear(b)
		var d documentoPreparado
		if decodificarDocumento(b, &d) != nil || d.Esquema != "vec.persona.denominacion.preparacion.v1" || d.Estado != "preparada_pendiente_cotejo_y_autorizacion" || d.AlcanceFuente != "sintetico_declarado" || d.Preparacion.PersonaRef != persona || d.Preparacion.VersionEsperada != esperada || d.Preparacion.ProcedenciaRef != procedencia || d.Preparacion.Sobre.PersonaRef != persona || d.Preparacion.Sobre.Version != esperada+1 || d.Preparacion.Sobre.Indice.AmbitoRef != ambito || d.Preparacion.Sobre.Indice.ClaveRef != indiceRef {
			return fallo("denominacion.preparacion_divergente")
		}
		original, e := json.Marshal(d.Preparacion.Sobre)
		if e != nil || !bytes.Equal(original, d.SobreCanonico) || huellaSobre(original) != d.Preparacion.SobreSHA256 {
			return fallo("denominacion.preparacion_divergente")
		}
		coincide := false
		e = protector.ConDenominacionDescifrada(context.Background(), d.Preparacion.Sobre, func(n domain.DenominacionPersona) error {
			return n.ConNombreMostrar(func(b []byte) error { coincide = bytes.Equal(b, nombre); return nil })
		})
		if e != nil || !coincide {
			return fallo("denominacion.preparacion_divergente")
		}
		return emitir(salida, "denominacion.preparada", true, 0)
	} else if !os.IsNotExist(err) {
		return fallo("denominacion.salida_no_disponible")
	}
	prep, err := protector.PrepararDenominacionPersona(context.Background(), persona, esperada, procedencia, ambito, indiceRef, nombre)
	if err != nil {
		return fallo("denominacion.nombre_no_disponible")
	}
	canon, err := json.Marshal(prep.Sobre)
	if err != nil || huellaSobre(canon) != prep.SobreSHA256 {
		return fallo("denominacion.preparacion_divergente")
	}
	d := documentoPreparado{Esquema: "vec.persona.denominacion.preparacion.v1", Estado: "preparada_pendiente_cotejo_y_autorizacion", AlcanceFuente: "sintetico_declarado", Preparacion: prep, SobreCanonico: canon}
	b, err := json.Marshal(d)
	if err != nil {
		return fallo("denominacion.salida_no_disponible")
	}
	// El artefacto contiene únicamente sobre, índice y coordenadas opacas.
	archivo, err := raiz.OpenFile(base, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return fallo("denominacion.salida_no_disponible")
	}
	cerrado := false
	defer func() {
		if !cerrado {
			_ = archivo.Close()
		}
	}()
	if err = archivo.Chmod(0600); err == nil {
		_, err = archivo.Write(append(b, '\n'))
	}
	if err == nil {
		err = archivo.Sync()
	}
	if e := archivo.Close(); err == nil {
		err = e
	}
	cerrado = true
	if err != nil {
		_ = raiz.Remove(base)
		return fallo("denominacion.salida_no_disponible")
	}
	return emitir(salida, "denominacion.preparada", false, 0)
}
func huellaSobre(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func refProcedenciaCA32(s string) bool {
	if !strings.HasPrefix(s, "prc_") || len(s) < 26 || len(s) > 128 {
		return false
	}
	for _, r := range s[4:] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
func refOpaca(s string) bool {
	if len(s) < 8 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
func rutasSeparadas(rutas ...string) bool {
	v := map[string]bool{}
	for _, r := range rutas {
		if !filepath.IsAbs(r) || filepath.Clean(r) != r || v[r] {
			return false
		}
		v[r] = true
	}
	return true
}
func decodificarDocumento(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func cargarTextos(ruta, idioma string) (*i18n.Catalog, error) {
	if idioma == "" || !filepath.IsAbs(ruta) {
		return nil, io.ErrUnexpectedEOF
	}
	raiz, e := os.OpenRoot(filepath.Dir(ruta))
	if e != nil {
		return nil, e
	}
	defer raiz.Close()
	f, e := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil || !i.Mode().IsRegular() || i.Size() < 1 || i.Size() > 65536 {
		return nil, io.ErrUnexpectedEOF
	}
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil || len(b) > 65536 {
		return nil, io.ErrUnexpectedEOF
	}
	var c struct {
		Idioma   string            `json:"idioma"`
		Mensajes map[string]string `json:"mensajes"`
	}
	if decodificarDocumento(b, &c) != nil || c.Idioma != idioma {
		return nil, io.ErrUnexpectedEOF
	}
	claves := []string{"denominacion.catalogo_no_disponible", "denominacion.uso_invalido", "denominacion.configuracion_no_disponible", "denominacion.material_no_disponible", "denominacion.nombre_no_disponible", "denominacion.salida_no_disponible", "denominacion.preparacion_divergente", "denominacion.preparada"}
	mensajes := make(map[string]string, len(claves))
	for _, k := range claves {
		mensaje := c.Mensajes[strings.ReplaceAll(k, ".", "_")]
		if mensaje == "" {
			return nil, io.ErrUnexpectedEOF
		}
		mensajes[k] = mensaje
	}
	return i18n.New(idioma, map[string]map[string]string{idioma: mensajes})
}
