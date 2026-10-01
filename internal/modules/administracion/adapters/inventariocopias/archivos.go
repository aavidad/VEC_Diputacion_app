package inventariocopias

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
)

var errArchivo = errors.New("inventario_archivo_no_comprobable")

// leerHuella solo lee archivos regulares dentro de la raíz autorizada. El tamaño
// previsto limita la lectura; los errores nunca conservan rutas ni contenido.
func leerHuella(raiz *os.Root, ruta string, tamanoEsperado int64) (string, int64, error) {
	if !rutaValida(ruta) || tamanoEsperado < 0 || tamanoEsperado == int64(^uint64(0)>>1) {
		return "", 0, errArchivo
	}
	info, err := raiz.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() {
		return "", 0, errArchivo
	}
	// NONBLOCK evita quedar esperando si un fichero se cambia por un FIFO entre
	// Lstat y OpenFile. Root mantiene confinados también los enlaces simbólicos.
	archivo, err := raiz.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", 0, errArchivo
	}
	defer archivo.Close()
	info, err = archivo.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", 0, errArchivo
	}
	if info.Size() != tamanoEsperado {
		return "", info.Size(), nil
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(archivo, tamanoEsperado+1))
	if err != nil {
		return "", 0, errArchivo
	}
	if n != tamanoEsperado {
		return "", n, nil
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func rutaValida(ruta string) bool {
	return ruta != "." && fs.ValidPath(ruta) && !strings.ContainsAny(ruta, "\\\x00")
}
