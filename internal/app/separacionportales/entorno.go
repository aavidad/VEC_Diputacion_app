package separacionportales

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PrefijoVariablesExterno reserva las variables del proceso externo. Una
// conexión o un secreto del portal externo se declara siempre con este
// prefijo; así cada proceso puede distinguir las suyas sin conocer las del
// otro.
const PrefijoVariablesExterno = "VEC_EXTERNO_"

// Entorno es la instantánea que se comprueba. Variables sigue el formato de
// os.Environ; DirectorioPersonal es el del usuario del sistema, donde el
// controlador de PostgreSQL busca credenciales por defecto.
// DirectorioMaterial es el material del propio proceso: en el externo toda
// ruta recibida debe estar dentro de él (salvo catálogos públicos).
type Entorno struct {
	Variables          []string
	DirectorioPersonal string
	DirectorioMaterial string
}

// variablesCredencialPostgreSQL son las que el controlador pgx lee por su
// cuenta del entorno. Con ellas un proceso podría abrir una conexión con
// credenciales que no figuran en su configuración. PGUSER también: una
// conexión sin usuario explícito tomaría el de otro portal.
var variablesCredencialPostgreSQL = map[string]struct{}{
	"PGPASSWORD": {}, "PGPASSFILE": {}, "PGSERVICE": {}, "PGSERVICEFILE": {}, "PGUSER": {},
	"PGSSLKEY": {}, "PGSSLCERT": {}, "PGSSLPASSWORD": {},
}

// ficherosCredencialPostgreSQL son los que pgx busca en el directorio
// personal cuando una conexión no trae contraseña o certificado propios.
var ficherosCredencialPostgreSQL = []string{
	".pgpass",
	".pg_service.conf",
	filepath.Join(".postgresql", "postgresql.key"),
}

// variablesSoloInterno señalan secretos o custodias con datos de personas que
// solo usa la parte interna. En el proceso externo se rechazan aunque no sean
// conexiones.
var variablesSoloInterno = map[string]struct{}{
	"VEC_BOLSA_IMPORTACION_CONVOCA_CUSTODIA_DIR": {},
	"VEC_FIRMA_VERIFICACION_TOKEN_FILE":          {},
	"VEC_FIRMA_VERIFICACION_KEY_FILE":            {},
	"VEC_CT_INCORPORACION_V2_FILE":               {},
	"VEC_FAKE_CREDENTIALS_FILE":                  {},
	"VEC_AUDITORIA_E2E_KEY":                      {},
}

// variablesSecretasExterno son los secretos que el proceso externo sí puede
// recibir sin prefijo propio: la clave TLS de su servidor, que el cargador
// exige además dentro de su propio directorio de material.
var variablesSecretasExterno = map[string]struct{}{
	"VEC_TLS_KEY_FILE": {},
}

// selectoresSoloExterno encienden capacidades del Área personal; el proceso
// interno no puede atenderlas.
var selectoresSoloExterno = map[string]struct{}{
	"VEC_BOLSA_PORTAL_CANDIDATO_ENABLED": {},
}

// ComprobarEntorno rechaza, antes de abrir nada, cualquier credencial o
// secreto del otro portal presente en el entorno del proceso. En el portal
// combinado no comprueba nada.
func ComprobarEntorno(p Portal, e Entorno) error {
	if !p.Separado() {
		if p != PortalCombinado {
			return ErrPortalDesconocido
		}
		return nil
	}
	variables := ordenarVariables(e.Variables)
	for _, v := range variables {
		if err := comprobarVariable(p, v.nombre, v.valor); err != nil {
			return err
		}
		if err := comprobarRutaPortal(p, v.nombre, v.valor, e.DirectorioMaterial); err != nil {
			return err
		}
	}
	return comprobarFicherosCredencialPersonales(e.DirectorioPersonal)
}

type variable struct{ nombre, valor string }

// ordenarVariables da un orden estable al primer rechazo, para que el mensaje
// no dependa del orden del entorno.
func ordenarVariables(entorno []string) []variable {
	resultado := make([]variable, 0, len(entorno))
	for _, linea := range entorno {
		nombre, valor, _ := strings.Cut(linea, "=")
		resultado = append(resultado, variable{nombre: nombre, valor: valor})
	}
	sort.SliceStable(resultado, func(i, j int) bool { return resultado[i].nombre < resultado[j].nombre })
	return resultado
}

// comprobarVariable ignora las variables vacías: no aportan credencial ni
// secreto alguno.
func comprobarVariable(p Portal, nombre, valor string) error {
	if strings.TrimSpace(valor) == "" {
		return nil
	}
	if _, ok := variablesCredencialPostgreSQL[nombre]; ok {
		return rechazo("credencial de PostgreSQL implicita en el entorno", nombre)
	}
	externa := strings.HasPrefix(nombre, PrefijoVariablesExterno)
	if esConexion(nombre, valor) {
		if p == PortalExterno && !externa {
			return rechazo("conexion de base de datos ajena al portal externo", nombre)
		}
		if p == PortalInterno && externa {
			return rechazo("conexion de base de datos del portal externo", nombre)
		}
		return nil
	}
	if p == PortalInterno {
		if externa {
			return rechazo("variable del portal externo", nombre)
		}
		if _, ok := selectoresSoloExterno[nombre]; ok && valor != "false" {
			return rechazo("capacidad del portal externo", nombre)
		}
		return nil
	}
	if externa {
		return nil
	}
	if _, ok := variablesSoloInterno[nombre]; ok {
		return rechazo("secreto o custodia del portal interno", nombre)
	}
	if _, ok := variablesSecretasExterno[nombre]; !ok && pareceSecreto(nombre) {
		return rechazo("secreto sin clasificar para el portal externo", nombre)
	}
	return nil
}

// comprobarRutaPortal solo restringe rutas en el proceso externo; las
// variables vacías no apuntan a nada.
func comprobarRutaPortal(p Portal, nombre, valor, material string) error {
	if p != PortalExterno || strings.TrimSpace(valor) == "" {
		return nil
	}
	return comprobarRutaExterno(nombre, valor, material)
}

// esConexion reconoce una conexión por su nombre o por su valor, para que un
// nombre inesperado no oculte una cadena de conexión.
func esConexion(nombre, valor string) bool {
	if strings.HasPrefix(nombre, "VEC_") && (strings.Contains(nombre, "DATABASE_URL") ||
		strings.HasSuffix(nombre, "_DSN") || strings.Contains(nombre, "_DSN_")) {
		return true
	}
	v := strings.ToLower(strings.TrimSpace(valor))
	if strings.HasPrefix(v, "postgres://") || strings.HasPrefix(v, "postgresql://") {
		return true
	}
	// Forma clave=valor: basta con dos claves típicas de una conexión.
	claves := 0
	for _, clave := range []string{"host=", "user=", "dbname=", "password=", "sslkey=", "service="} {
		if strings.Contains(v, clave) {
			claves++
		}
	}
	return claves >= 2
}

// pareceSecreto solo mira variables de VEC: las ajenas no las lee el proceso
// (las de PostgreSQL se tratan aparte).
func pareceSecreto(nombre string) bool {
	if !strings.HasPrefix(nombre, "VEC_") {
		return false
	}
	for _, marca := range []string{"_KEY", "_TOKEN", "_SECRET", "_PASSWORD", "_CREDENTIALS", "CUSTODIA"} {
		if strings.Contains(nombre, marca) {
			return true
		}
	}
	return false
}

func comprobarFicherosCredencialPersonales(directorio string) error {
	if directorio == "" {
		return nil
	}
	for _, relativa := range ficherosCredencialPostgreSQL {
		_, err := os.Lstat(filepath.Join(directorio, relativa))
		if err == nil {
			return rechazo("fichero de credenciales de PostgreSQL en el directorio personal", "~/"+filepath.ToSlash(relativa))
		}
		if !errors.Is(err, os.ErrNotExist) {
			return rechazo("no se puede comprobar el directorio personal", "~/"+filepath.ToSlash(relativa))
		}
	}
	return nil
}
