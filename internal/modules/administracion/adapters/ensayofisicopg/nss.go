package ensayofisicopg

import (
	"fmt"
	"os"
	"path/filepath"
)

// prepararNSS no lee passwd/group del host. La identidad corresponde al UID/GID
// sin privilegios que usa Docker, con HOME en el tmpfs privado del ensayo.
func prepararNSS(raiz string) error {
	uid, gid := os.Getuid(), os.Getgid()
	if uid <= 0 || gid <= 0 {
		return errRuntime
	}
	root, err := os.OpenRoot(raiz)
	if err != nil {
		return errRuntime
	}
	defer root.Close()
	passwd := fmt.Sprintf("vecensayo:x:%d:%d:VEC:/tmp:/usr/sbin/nologin\n", uid, gid)
	group := fmt.Sprintf("vecensayo:x:%d:\n", gid)
	for n, contenido := range map[string]string{"passwd": passwd, "group": group} {
		f, err := root.OpenFile(n, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
		if err != nil {
			return errRuntime
		}
		_, escribir := f.WriteString(contenido)
		cerrar := f.Close()
		if escribir != nil || cerrar != nil {
			return errRuntime
		}
	}
	return nil
}
func montajesNSS(raiz string) []string {
	return []string{"-v", filepath.Join(raiz, "passwd") + ":/etc/passwd:ro", "-v", filepath.Join(raiz, "group") + ":/etc/group:ro"}
}
