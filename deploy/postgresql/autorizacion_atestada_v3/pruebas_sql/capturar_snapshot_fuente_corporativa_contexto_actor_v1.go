//go:build linux

// Capturador probatorio H0; no forma parte del producto.
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

const (
	maximoRutas       = 512
	maximoProfundidad = 64
	maximoFichero     = 1 << 20
	maximoTotal       = 256 << 20
)

type configuracion struct {
	raiz                string
	destino, manifiesto string
	rutas               []string
}

// ganchos coordina carreras deterministas; la ruta operativa lo deja vacío.
type ganchos struct {
	directorioAbierto func(string)
	archivoAbierto    func(string)
	lectura           func(string, int)
}

type metadatos struct {
	dispositivo          uint64
	inodo                uint64
	tipo                 os.FileMode
	tamano               int64
	modificado, cambiado syscall.Timespec
	enlaces              uint64
}

type directorioRetenido struct {
	ruta       string
	nombre     string
	padre      *os.Root
	raiz       *os.Root
	descriptor *os.File
	inicial    metadatos
}

type archivoRetenido struct {
	ruta       string
	nombre     string
	padre      *os.Root
	descriptor *os.File
	inicial    metadatos
	huella     [sha256.Size]byte
}

type fuenteRetenida struct {
	directorios []*directorioRetenido
	archivos    []*archivoRetenido
}

func main() {
	if err := ejecutar(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func ejecutar(argumentos []string) error {
	if len(argumentos) == 1 && argumentos[0] == "--autoprueba" {
		return ejecutarAutoprueba()
	}
	if len(argumentos) < 8 || argumentos[0] != "--raiz" || argumentos[2] != "--destino" ||
		argumentos[4] != "--manifiesto" || argumentos[6] != "--" {
		return errors.New("uso: --raiz . --destino RUTA --manifiesto RUTA -- FICHERO...")
	}
	if argumentos[1] != "." {
		return errors.New("--raiz debe ser literalmente .")
	}
	if argumentos[3] == "" || argumentos[5] == "" {
		return errors.New("--destino y --manifiesto son obligatorios")
	}
	return capturar(configuracion{
		raiz:       argumentos[1],
		destino:    argumentos[3],
		manifiesto: argumentos[5],
		rutas:      argumentos[7:],
	}, ganchos{})
}

func capturar(cfg configuracion, pruebas ganchos) (err error) {
	rutas, err := validarConfiguracion(cfg)
	if err != nil {
		return err
	}
	if err := os.Mkdir(cfg.destino, 0o700); err != nil {
		return fmt.Errorf("no se pudo crear el destino nuevo: %w", err)
	}
	exito := false
	defer func() {
		if !exito {
			_ = os.Remove(cfg.manifiesto)
			_ = os.RemoveAll(cfg.destino)
		}
	}()
	if err := os.Chmod(cfg.destino, 0o700); err != nil {
		return fmt.Errorf("no se pudo fijar el modo privado del destino: %w", err)
	}
	destino, err := os.OpenRoot(cfg.destino)
	if err != nil {
		return fmt.Errorf("no se pudo retener el destino: %w", err)
	}
	defer destino.Close()

	fuente, err := abrirFuenteRetenida()
	if err != nil {
		return err
	}
	defer fuente.cerrar()

	directorios := map[string]*directorioRetenido{".": fuente.directorios[0]}
	var total int64
	for _, ruta := range rutas {
		archivo, err := fuente.abrirArchivo(ruta, directorios, pruebas)
		if err != nil {
			return err
		}
		if archivo.inicial.tamano > maximoFichero || total > maximoTotal-archivo.inicial.tamano {
			return fmt.Errorf("%s: límite de bytes del snapshot excedido", ruta)
		}
		total += archivo.inicial.tamano
		if err := crearPadresDestino(destino, filepath.Dir(ruta)); err != nil {
			return fmt.Errorf("%s: no se pudo preparar el destino: %w", ruta, err)
		}
		huella, err := copiarDesdeDescriptor(destino, archivo, pruebas)
		if err != nil {
			return err
		}
		archivo.huella = huella
	}

	if err := fuente.acreditarInmutabilidad(); err != nil {
		return err
	}
	if err := fuente.cerrar(); err != nil {
		return fmt.Errorf("no se pudieron cerrar los descriptores fuente: %w", err)
	}
	if err := escribirManifiestoFinal(cfg.manifiesto, fuente.archivos); err != nil {
		return err
	}
	exito = true
	return nil
}

func validarConfiguracion(cfg configuracion) ([]string, error) {
	if cfg.raiz != "." {
		return nil, errors.New("la raíz fuente debe ser literalmente .")
	}
	if len(cfg.rutas) == 0 || len(cfg.rutas) > maximoRutas {
		return nil, fmt.Errorf("el inventario debe contener entre 1 y %d rutas", maximoRutas)
	}
	if !filepath.IsAbs(cfg.destino) || filepath.Clean(cfg.destino) != cfg.destino {
		return nil, errors.New("el destino debe ser una ruta absoluta limpia")
	}
	if !filepath.IsAbs(cfg.manifiesto) || filepath.Clean(cfg.manifiesto) != cfg.manifiesto {
		return nil, errors.New("el manifiesto debe ser una ruta absoluta limpia")
	}
	if _, err := os.Lstat(cfg.destino); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return nil, errors.New("el destino ya existe")
		}
		return nil, fmt.Errorf("no se pudo acreditar la ausencia del destino: %w", err)
	}
	if _, err := os.Lstat(cfg.manifiesto); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return nil, errors.New("el manifiesto ya existe")
		}
		return nil, fmt.Errorf("no se pudo acreditar la ausencia del manifiesto: %w", err)
	}
	cwd, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("no se pudo resolver el directorio retenido: %w", err)
	}
	if dentroDe(cfg.destino, cwd) || dentroDe(cfg.manifiesto, cwd) {
		return nil, errors.New("destino y manifiesto deben quedar fuera de la raíz fuente")
	}
	vistas := make(map[string]struct{}, len(cfg.rutas))
	rutas := append([]string(nil), cfg.rutas...)
	for _, ruta := range rutas {
		if err := validarRutaRelativa(ruta); err != nil {
			return nil, err
		}
		if _, duplicada := vistas[ruta]; duplicada {
			return nil, fmt.Errorf("%s: ruta duplicada", ruta)
		}
		vistas[ruta] = struct{}{}
	}
	sort.Strings(rutas)
	return rutas, nil
}

func validarRutaRelativa(ruta string) error {
	if ruta == "" || len(ruta) > 4096 || ruta == "." || !filepath.IsLocal(ruta) || filepath.Clean(ruta) != ruta {
		return fmt.Errorf("%q: ruta relativa no local o no limpia", ruta)
	}
	if strings.ContainsAny(ruta, "\x00\r\n\t") || strings.ContainsRune(ruta, '\\') {
		return fmt.Errorf("%q: ruta no representable en el manifiesto", ruta)
	}
	componentes := strings.Split(ruta, string(filepath.Separator))
	if len(componentes) > maximoProfundidad {
		return fmt.Errorf("%s: profundidad máxima excedida", ruta)
	}
	return nil
}

func dentroDe(ruta, raiz string) bool {
	relativa, err := filepath.Rel(raiz, ruta)
	return err == nil && (relativa == "." || filepath.IsLocal(relativa))
}

func abrirFuenteRetenida() (*fuenteRetenida, error) {
	r, err := os.OpenRoot(".")
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir la raíz fuente: %w", err)
	}
	descriptor, err := r.Open(".")
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("no se pudo retener la raíz fuente: %w", err)
	}
	lstat, err := r.Lstat(".")
	if err != nil {
		descriptor.Close()
		r.Close()
		return nil, fmt.Errorf("no se pudo acreditar la raíz fuente: %w", err)
	}
	fstat, err := descriptor.Stat()
	if err != nil {
		descriptor.Close()
		r.Close()
		return nil, fmt.Errorf("no se pudo consultar la raíz retenida: %w", err)
	}
	metaL, err := extraerMetadatos(lstat)
	if err != nil {
		descriptor.Close()
		r.Close()
		return nil, err
	}
	metaF, err := extraerMetadatos(fstat)
	if err != nil || metaL != metaF || metaL.tipo&os.ModeDir == 0 {
		descriptor.Close()
		r.Close()
		return nil, errors.New("la raíz abierta no coincide con su descriptor")
	}
	return &fuenteRetenida{directorios: []*directorioRetenido{{
		ruta: ".", nombre: ".", padre: r, raiz: r, descriptor: descriptor, inicial: metaF,
	}}}, nil
}

func (f *fuenteRetenida) abrirArchivo(ruta string, conocidos map[string]*directorioRetenido, pruebas ganchos) (*archivoRetenido, error) {
	componentes := strings.Split(ruta, string(filepath.Separator))
	actual := conocidos["."]
	acumulada := ""
	for _, nombre := range componentes[:len(componentes)-1] {
		if acumulada == "" {
			acumulada = nombre
		} else {
			acumulada = filepath.Join(acumulada, nombre)
		}
		if existente, ok := conocidos[acumulada]; ok {
			actual = existente
			continue
		}
		directorio, err := abrirDirectorio(actual.raiz, acumulada, nombre)
		if err != nil {
			return nil, err
		}
		f.directorios = append(f.directorios, directorio)
		conocidos[acumulada] = directorio
		actual = directorio
		if pruebas.directorioAbierto != nil {
			pruebas.directorioAbierto(acumulada)
		}
	}
	nombre := componentes[len(componentes)-1]
	inicialL, err := actual.raiz.Lstat(nombre)
	if err != nil {
		return nil, fmt.Errorf("%s: no se pudo acreditar el fichero: %w", ruta, err)
	}
	metaL, err := extraerMetadatos(inicialL)
	if err != nil || metaL.tipo != 0 {
		return nil, fmt.Errorf("%s: se exige un fichero regular no simbólico", ruta)
	}
	descriptor, err := actual.raiz.Open(nombre)
	if err != nil {
		return nil, fmt.Errorf("%s: no se pudo abrir el fichero: %w", ruta, err)
	}
	inicialF, err := descriptor.Stat()
	if err != nil {
		descriptor.Close()
		return nil, fmt.Errorf("%s: no se pudo consultar el descriptor: %w", ruta, err)
	}
	metaF, err := extraerMetadatos(inicialF)
	if err != nil || metaL != metaF || metaF.tipo != 0 || metaF.enlaces != 1 {
		descriptor.Close()
		return nil, fmt.Errorf("%s: descriptor distinto, no regular o enlazado", ruta)
	}
	archivo := &archivoRetenido{ruta: ruta, nombre: nombre, padre: actual.raiz, descriptor: descriptor, inicial: metaF}
	f.archivos = append(f.archivos, archivo)
	if pruebas.archivoAbierto != nil {
		pruebas.archivoAbierto(ruta)
	}
	return archivo, nil
}

func abrirDirectorio(padre *os.Root, ruta, nombre string) (*directorioRetenido, error) {
	inicialL, err := padre.Lstat(nombre)
	if err != nil {
		return nil, fmt.Errorf("%s: no se pudo acreditar el directorio: %w", ruta, err)
	}
	metaL, err := extraerMetadatos(inicialL)
	if err != nil || metaL.tipo&os.ModeDir == 0 || metaL.tipo&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: se exige un directorio real no simbólico", ruta)
	}
	raiz, err := padre.OpenRoot(nombre)
	if err != nil {
		return nil, fmt.Errorf("%s: no se pudo abrir el directorio: %w", ruta, err)
	}
	descriptor, err := raiz.Open(".")
	if err != nil {
		raiz.Close()
		return nil, fmt.Errorf("%s: no se pudo retener el directorio: %w", ruta, err)
	}
	inicialF, err := descriptor.Stat()
	if err != nil {
		descriptor.Close()
		raiz.Close()
		return nil, fmt.Errorf("%s: no se pudo consultar el directorio: %w", ruta, err)
	}
	metaF, err := extraerMetadatos(inicialF)
	if err != nil || metaL != metaF || metaF.tipo&os.ModeDir == 0 {
		descriptor.Close()
		raiz.Close()
		return nil, fmt.Errorf("%s: el directorio abierto no coincide", ruta)
	}
	return &directorioRetenido{ruta: ruta, nombre: nombre, padre: padre, raiz: raiz, descriptor: descriptor, inicial: metaF}, nil
}

func extraerMetadatos(info os.FileInfo) (metadatos, error) {
	estado, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return metadatos{}, errors.New("el sistema no expone stat Linux")
	}
	return metadatos{
		dispositivo: uint64(estado.Dev), inodo: estado.Ino,
		tipo: info.Mode() & os.ModeType, tamano: info.Size(),
		modificado: estado.Mtim, cambiado: estado.Ctim,
		enlaces: estado.Nlink,
	}, nil
}

func crearPadresDestino(destino *os.Root, directorio string) error {
	if directorio == "." {
		return nil
	}
	actual := ""
	for _, nombre := range strings.Split(directorio, string(filepath.Separator)) {
		actual = filepath.Join(actual, nombre)
		info, err := destino.Lstat(actual)
		if errors.Is(err, os.ErrNotExist) {
			if err := destino.Mkdir(actual, 0o700); err != nil {
				return err
			}
			if err := destino.Chmod(actual, 0o700); err != nil {
				return err
			}
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("un padre del destino no es un directorio real")
		}
	}
	return nil
}

func copiarDesdeDescriptor(destino *os.Root, archivo *archivoRetenido, pruebas ganchos) ([sha256.Size]byte, error) {
	salida, err := destino.OpenFile(archivo.ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%s: no se pudo crear la copia exclusiva: %w", archivo.ruta, err)
	}
	correcto := false
	defer func() {
		if !correcto {
			salida.Close()
			_ = destino.Remove(archivo.ruta)
		}
	}()
	if err := salida.Chmod(0o600); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%s: no se pudo fijar el modo privado: %w", archivo.ruta, err)
	}
	hash := sha256.New()
	buffer := make([]byte, 32<<10)
	restantes := archivo.inicial.tamano
	for restantes > 0 {
		n, errLectura := archivo.descriptor.Read(buffer[:min(int64(len(buffer)), restantes)])
		if n > 0 {
			if pruebas.lectura != nil {
				pruebas.lectura(archivo.ruta, n)
			}
			_, _ = hash.Write(buffer[:n])
			cantidad, err := salida.Write(buffer[:n])
			restantes -= int64(cantidad)
			if err != nil || cantidad != n {
				return [sha256.Size]byte{}, fmt.Errorf("%s: escritura incompleta del snapshot", archivo.ruta)
			}
		}
		if errors.Is(errLectura, io.EOF) {
			break
		}
		if errLectura != nil || n == 0 {
			return [sha256.Size]byte{}, fmt.Errorf("%s: lectura fallida o sin progreso: %v", archivo.ruta, errLectura)
		}
	}
	if restantes != 0 {
		return [sha256.Size]byte{}, fmt.Errorf("%s: tamaño leído distinto del acreditado", archivo.ruta)
	}
	n, err := archivo.descriptor.Read(buffer[:1])
	if pruebas.lectura != nil {
		pruebas.lectura(archivo.ruta, n)
	}
	if n != 0 || !errors.Is(err, io.EOF) {
		return [sha256.Size]byte{}, fmt.Errorf("%s: crecimiento o fin inexacto tras el presupuesto", archivo.ruta)
	}
	if err := salida.Sync(); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%s: no se pudo sincronizar la copia: %w", archivo.ruta, err)
	}
	if err := salida.Close(); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%s: no se pudo cerrar la copia: %w", archivo.ruta, err)
	}
	correcto = true
	var huella [sha256.Size]byte
	copy(huella[:], hash.Sum(nil))
	return huella, nil
}

func (f *fuenteRetenida) acreditarInmutabilidad() error {
	for _, archivo := range f.archivos {
		porRuta, err := archivo.padre.Lstat(archivo.nombre)
		if err != nil {
			return fmt.Errorf("%s: la ruta final desapareció: %w", archivo.ruta, err)
		}
		porDescriptor, err := archivo.descriptor.Stat()
		if err != nil {
			return fmt.Errorf("%s: no se pudo reacreditar el descriptor: %w", archivo.ruta, err)
		}
		metaRuta, errRuta := extraerMetadatos(porRuta)
		metaDescriptor, errDescriptor := extraerMetadatos(porDescriptor)
		if errRuta != nil || errDescriptor != nil || archivo.inicial != metaRuta || archivo.inicial != metaDescriptor {
			return fmt.Errorf("%s: el fichero fue sustituido o mutado", archivo.ruta)
		}
	}
	for i := len(f.directorios) - 1; i >= 0; i-- {
		directorio := f.directorios[i]
		porRuta, err := directorio.padre.Lstat(directorio.nombre)
		if err != nil {
			return fmt.Errorf("%s: el directorio final desapareció: %w", directorio.ruta, err)
		}
		porDescriptor, err := directorio.descriptor.Stat()
		if err != nil {
			return fmt.Errorf("%s: no se pudo reacreditar el directorio: %w", directorio.ruta, err)
		}
		metaRuta, errRuta := extraerMetadatos(porRuta)
		metaDescriptor, errDescriptor := extraerMetadatos(porDescriptor)
		if errRuta != nil || errDescriptor != nil || directorio.inicial != metaRuta || directorio.inicial != metaDescriptor {
			return fmt.Errorf("%s: el componente de directorio fue sustituido o mutado", directorio.ruta)
		}
	}
	return nil
}

func (f *fuenteRetenida) cerrar() error {
	var resultado error
	for _, archivo := range f.archivos {
		if archivo.descriptor != nil {
			resultado = errors.Join(resultado, archivo.descriptor.Close())
			archivo.descriptor = nil
		}
	}
	for i := len(f.directorios) - 1; i >= 0; i-- {
		directorio := f.directorios[i]
		if directorio.descriptor != nil {
			resultado = errors.Join(resultado, directorio.descriptor.Close())
			directorio.descriptor = nil
		}
		if directorio.raiz != nil && (i != 0 || directorio.raiz != f.directorios[0].padre) {
			resultado = errors.Join(resultado, directorio.raiz.Close())
			directorio.raiz = nil
		}
	}
	if len(f.directorios) > 0 && f.directorios[0].padre != nil {
		resultado = errors.Join(resultado, f.directorios[0].padre.Close())
		f.directorios[0].padre = nil
		f.directorios[0].raiz = nil
	}
	return resultado
}

func escribirManifiestoFinal(ruta string, archivos []*archivoRetenido) error {
	padre := filepath.Dir(ruta)
	temporal, err := os.CreateTemp(padre, ".manifiesto-f0-parcial-*")
	if err != nil {
		return fmt.Errorf("no se pudo crear el manifiesto parcial: %w", err)
	}
	temporalRuta := temporal.Name()
	publicado := false
	defer func() {
		temporal.Close()
		_ = os.Remove(temporalRuta)
		if !publicado {
			_ = os.Remove(ruta)
		}
	}()
	if err := temporal.Chmod(0o600); err != nil {
		return fmt.Errorf("no se pudo fijar el modo del manifiesto: %w", err)
	}
	for _, archivo := range archivos {
		if _, err := fmt.Fprintf(temporal, "%s\t%x\n", archivo.ruta, archivo.huella); err != nil {
			return fmt.Errorf("no se pudo escribir el manifiesto: %w", err)
		}
	}
	if err := temporal.Sync(); err != nil {
		return fmt.Errorf("no se pudo sincronizar el manifiesto: %w", err)
	}
	if err := temporal.Close(); err != nil {
		return fmt.Errorf("no se pudo cerrar el manifiesto: %w", err)
	}
	if _, err := os.Lstat(ruta); !errors.Is(err, os.ErrNotExist) {
		return errors.New("el manifiesto dejó de ser exclusivo")
	}
	if err := os.Link(temporalRuta, ruta); err != nil {
		return fmt.Errorf("no se pudo publicar el manifiesto exclusivo: %w", err)
	}
	if err := os.Remove(temporalRuta); err != nil {
		_ = os.Remove(ruta)
		return fmt.Errorf("no se pudo retirar el manifiesto parcial: %w", err)
	}
	info, err := os.Lstat(ruta)
	if err != nil {
		return fmt.Errorf("no se pudo reacreditar el manifiesto: %w", err)
	}
	meta, errMeta := extraerMetadatos(info)
	if errMeta != nil || meta.tipo != 0 || meta.enlaces != 1 || info.Mode().Perm() != 0o600 {
		_ = os.Remove(ruta)
		return errors.New("el manifiesto publicado no conserva forma y modo exactos")
	}
	publicado = true
	return nil
}

func ejecutarAutoprueba() error {
	pruebas := []struct {
		nombre string
		fn     func() error
	}{
		{"normal", probarNormal},
		{"enlace_simbólico", func() error { return probarEnlace(true) }},
		{"enlace_duro", func() error { return probarEnlace(false) }},
		{"renombre_reemplazo", probarRenombre},
		{"mutación_mismo_inode", probarMutacion},
		{"componente_directorio", probarComponenteDirectorio},
	}
	for _, prueba := range pruebas {
		if err := prueba.fn(); err != nil {
			return fmt.Errorf("autoprueba %s: %w", prueba.nombre, err)
		}
	}
	fmt.Println("autoprueba=ok casos=6")
	return nil
}

type casoTemporal struct {
	base, fuente, destino, manifiesto, cwd string
}

func nuevoCaso() (*casoTemporal, error) {
	base, err := os.MkdirTemp("", "capturador-f0-*")
	if err != nil {
		return nil, err
	}
	caso := &casoTemporal{base: base, fuente: filepath.Join(base, "fuente"), destino: filepath.Join(base, "snapshot"), manifiesto: filepath.Join(base, "manifiesto.sha256")}
	if err := os.Mkdir(caso.fuente, 0o700); err != nil {
		os.RemoveAll(base)
		return nil, err
	}
	caso.cwd, err = os.Getwd()
	if err != nil {
		os.RemoveAll(base)
		return nil, err
	}
	if err := os.Chdir(caso.fuente); err != nil {
		os.RemoveAll(base)
		return nil, err
	}
	return caso, nil
}

func (c *casoTemporal) cerrar() {
	_ = os.Chdir(c.cwd)
	_ = os.RemoveAll(c.base)
}

func (c *casoTemporal) cfg(rutas ...string) configuracion {
	return configuracion{raiz: ".", destino: c.destino, manifiesto: c.manifiesto, rutas: rutas}
}

func (c *casoTemporal) exigirFalloSinManifiesto(err error) error {
	if err == nil {
		return errors.New("la captura adversa fue aceptada")
	}
	if _, fallo := os.Lstat(c.manifiesto); !errors.Is(fallo, os.ErrNotExist) {
		return errors.New("un fallo dejó un manifiesto consumible")
	}
	if _, fallo := os.Lstat(c.destino); !errors.Is(fallo, os.ErrNotExist) {
		return errors.New("un fallo dejó un destino parcial consumible")
	}
	return nil
}

func conCaso(prueba func(*casoTemporal) error) error {
	caso, err := nuevoCaso()
	if err != nil {
		return err
	}
	defer caso.cerrar()
	return prueba(caso)
}

func ejecutarCarrera(caso *casoTemporal, rutas []string, gancho func(func()) ganchos, mutar func() error) error {
	listo, continuar, cancelar := make(chan struct{}), make(chan struct{}), make(chan struct{})
	errores := make(chan error, 1)
	go func() {
		select {
		case <-listo:
			errores <- mutar()
			close(continuar)
		case <-cancelar:
			errores <- nil
		}
	}()
	sincronizar := func() { close(listo); <-continuar }
	errCaptura := capturar(caso.cfg(rutas...), gancho(sincronizar))
	close(cancelar)
	if errMutacion := <-errores; errMutacion != nil {
		return errMutacion
	}
	return caso.exigirFalloSinManifiesto(errCaptura)
}

func probarNormal() error {
	return conCaso(func(caso *casoTemporal) error {
		if err := os.Mkdir("dir", 0o700); err != nil {
			return err
		}
		if err := errors.Join(
			os.WriteFile("dir/a.sql", []byte("SELECT 1;\n"), 0o600),
			os.WriteFile("b.sql", []byte("SELECT 2;\n"), 0o600)); err != nil {
			return err
		}
		if err := capturar(caso.cfg("dir/a.sql", "b.sql"), ganchos{}); err != nil {
			return err
		}
		contenido, err := os.ReadFile(caso.manifiesto)
		if err != nil {
			return err
		}
		hB, hA := sha256.Sum256([]byte("SELECT 2;\n")), sha256.Sum256([]byte("SELECT 1;\n"))
		if string(contenido) != fmt.Sprintf("b.sql\t%x\ndir/a.sql\t%x\n", hB, hA) {
			return errors.New("manifiesto no ordenado o inexacto")
		}
		infoDestino, errDestino := os.Stat(caso.destino)
		infoManifiesto, errManifiesto := os.Stat(caso.manifiesto)
		if errDestino != nil || errManifiesto != nil || infoDestino.Mode().Perm() != 0o700 || infoManifiesto.Mode().Perm() != 0o600 {
			return errors.New("destino o manifiesto sin modo privado exacto")
		}
		copia, err := os.ReadFile(filepath.Join(caso.destino, "dir/a.sql"))
		if err != nil || string(copia) != "SELECT 1;\n" {
			return errors.New("copia normal distinta")
		}
		return nil
	})
}

func probarEnlace(simbolico bool) error {
	return conCaso(func(caso *casoTemporal) error {
		if err := os.WriteFile("real.sql", []byte("SELECT 1;\n"), 0o600); err != nil {
			return err
		}
		ruta := "duro.sql"
		crear := os.Link
		if simbolico {
			ruta, crear = "enlace.sql", os.Symlink
		}
		if err := crear("real.sql", ruta); err != nil {
			return err
		}
		if !simbolico {
			ruta = "real.sql"
		}
		return caso.exigirFalloSinManifiesto(capturar(caso.cfg(ruta), ganchos{}))
	})
}

func probarRenombre() error {
	return conCaso(func(caso *casoTemporal) error {
		if err := os.WriteFile("dato.sql", []byte("ORIGINAL\n"), 0o600); err != nil {
			return err
		}
		return ejecutarCarrera(caso, []string{"dato.sql"},
			func(s func()) ganchos { return ganchos{archivoAbierto: func(string) { s() }} },
			func() error {
				if err := os.Rename("dato.sql", "dato.anterior"); err != nil {
					return err
				}
				return os.WriteFile("dato.sql", []byte("REEMPLAZO\n"), 0o600)
			})
	})
}

func probarMutacion() error {
	return conCaso(func(caso *casoTemporal) error {
		const tamanoInicial = 128 << 10
		if err := os.WriteFile("dato.sql", bytes.Repeat([]byte{'A'}, tamanoInicial), 0o600); err != nil {
			return err
		}
		leidos := 0
		if err := ejecutarCarrera(caso, []string{"dato.sql"},
			func(s func()) ganchos {
				return ganchos{lectura: func(_ string, n int) {
					leidos += n
					if leidos == n {
						s()
					}
				}}
			},
			func() error {
				f, err := os.OpenFile("dato.sql", os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					return err
				}
				for i := 0; i < 64 && err == nil; i++ {
					_, err = f.Write(bytes.Repeat([]byte{'B'}, 4096))
				}
				return errors.Join(err, f.Close())
			}); err != nil {
			return err
		}
		if leidos != tamanoInicial+1 {
			return fmt.Errorf("se leyeron %d bytes; se esperaban %d", leidos, tamanoInicial+1)
		}
		return nil
	})
}

func probarComponenteDirectorio() error {
	return conCaso(func(caso *casoTemporal) error {
		if err := os.Mkdir("dir", 0o700); err != nil {
			return err
		}
		if err := os.WriteFile("dir/dato.sql", []byte("ORIGINAL\n"), 0o600); err != nil {
			return err
		}
		return ejecutarCarrera(caso, []string{"dir/dato.sql"},
			func(s func()) ganchos {
				return ganchos{directorioAbierto: func(ruta string) {
					if ruta == "dir" {
						s()
					}
				}}
			},
			func() error {
				if err := os.Rename("dir", "dir.anterior"); err != nil {
					return err
				}
				if err := os.Mkdir("dir", 0o700); err != nil {
					return err
				}
				return os.WriteFile("dir/dato.sql", []byte("REEMPLAZO\n"), 0o600)
			})
	})
}
