package main

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// cargarDSNPrivado lee sólo el fichero indicado. No consulta servicios,
// certificados implícitos, pgpass ni variables de entorno para obtener datos.
func cargarDSNPrivado(ruta string) (string, error) {
	b, err := leerArchivoPrivadoPerfiles(ruta)
	if err != nil {
		return "", errConfiguracionPrivadaPerfiles
	}
	defer clear(b)
	var x struct {
		DSN string `json:"dsn"`
	}
	if decodificarConfiguracionPrivada(b, &x) != nil || len(x.DSN) == 0 || len(x.DSN) > 16<<10 || validarDSNPerfilesPrivado(x.DSN) != nil {
		return "", errConfiguracionPrivadaPerfiles
	}
	return desactivarPassfileDSNPerfiles(x.DSN), nil
}

func validarDSNPerfilesPrivado(dsn string) error {
	// Estos parámetros alteran identidad, destino o credenciales fuera del
	// fichero protegido. PGAPPNAME y PGTZ no son fuentes de autenticación.
	for _, k := range []string{"PGHOST", "PGHOSTADDR", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE", "PGSERVICE", "PGSERVICEFILE", "PGSSLMODE", "PGSSLROOTCERT", "PGSSLCERT", "PGSSLKEY", "PGSSLCRL", "PGSSLCRLDIR", "PGSSLNEGOTIATION", "PGSSLPASSWORD", "PGSSLSNI", "PGCHANNELBINDING", "PGREQUIREAUTH", "PGTARGETSESSIONATTRS", "PGOPTIONS"} {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return errConfiguracionPrivadaPerfiles
		}
	}
	campos, err := camposDSNPerfiles(dsn)
	if err != nil {
		return errConfiguracionPrivadaPerfiles
	}
	permitidos := map[string]bool{"host": true, "port": true, "dbname": true, "user": true, "password": true, "sslmode": true, "passfile": true, "sslrootcert": true, "sslcert": true, "sslkey": true, "sslcrl": true, "sslcrldir": true, "connect_timeout": true, "application_name": true, "target_session_attrs": true, "channel_binding": true, "require_auth": true, "sslnegotiation": true}
	for k := range campos {
		if !permitidos[k] {
			return errConfiguracionPrivadaPerfiles
		}
	}
	if _, ok := campos["password"]; !ok {
		return errConfiguracionPrivadaPerfiles
	}
	if p, ok := campos["passfile"]; ok && p != "/dev/null" {
		return errConfiguracionPrivadaPerfiles
	}
	for _, k := range []string{"host", "dbname", "user", "sslmode"} {
		if campos[k] == "" || strings.TrimSpace(campos[k]) != campos[k] {
			return errConfiguracionPrivadaPerfiles
		}
	}
	for _, k := range []string{"user", "dbname"} {
		for _, r := range campos[k] {
			if unicode.IsControl(r) {
				return errConfiguracionPrivadaPerfiles
			}
		}
	}
	hosts := strings.Split(campos["host"], ",")
	if len(hosts) > 8 {
		return errConfiguracionPrivadaPerfiles
	}
	todosUnix := true
	for _, host := range hosts {
		if host == "" || strings.TrimSpace(host) != host {
			return errConfiguracionPrivadaPerfiles
		}
		if strings.HasPrefix(host, "/") {
			if !filepath.IsAbs(host) || filepath.Clean(host) != host {
				return errConfiguracionPrivadaPerfiles
			}
			continue
		}
		todosUnix = false
		if strings.ContainsAny(host, "/\\ \t\r\n") || !hostDSNValido(host) {
			return errConfiguracionPrivadaPerfiles
		}
	}
	if todosUnix {
		if campos["sslmode"] != "disable" && campos["sslmode"] != "verify-full" {
			return errConfiguracionPrivadaPerfiles
		}
	} else if campos["sslmode"] != "verify-full" {
		return errConfiguracionPrivadaPerfiles
	}
	if p, ok := campos["port"]; ok {
		puertos := strings.Split(p, ",")
		if len(puertos) != 1 && len(puertos) != len(hosts) {
			return errConfiguracionPrivadaPerfiles
		}
		for _, s := range puertos {
			n, e := strconv.ParseUint(s, 10, 16)
			if e != nil || n == 0 {
				return errConfiguracionPrivadaPerfiles
			}
		}
	}
	if s, ok := campos["connect_timeout"]; ok {
		n, e := strconv.Atoi(s)
		if e != nil || n <= 0 || n > 60 {
			return errConfiguracionPrivadaPerfiles
		}
	}
	for _, k := range []string{"sslrootcert", "sslcert", "sslkey", "sslcrl", "sslcrldir"} {
		if v, ok := campos[k]; ok && v != "" && !(k == "sslrootcert" && v == "system") && !rutaPrivadaPerfilesValida(v) {
			return errConfiguracionPrivadaPerfiles
		}
	}
	if (campos["sslcert"] == "") != (campos["sslkey"] == "") {
		return errConfiguracionPrivadaPerfiles
	}
	return nil
}

func hostDSNValido(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if len(host) > 253 || strings.Contains(host, ":") {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

func camposDSNPerfiles(dsn string) (map[string]string, error) {
	campos := map[string]string{}
	agregar := func(k, v string) error {
		if k == "" {
			return errConfiguracionPrivadaPerfiles
		}
		if _, ok := campos[k]; ok {
			return errConfiguracionPrivadaPerfiles
		}
		campos[k] = v
		return nil
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil || u.Fragment != "" || u.Opaque != "" || u.User == nil || u.User.Username() == "" {
			return nil, errConfiguracionPrivadaPerfiles
		}
		campos["user"] = u.User.Username()
		if p, ok := u.User.Password(); ok {
			campos["password"] = p
		}
		if u.Host != "" {
			var hosts, ports []string
			for _, hp := range strings.Split(u.Host, ",") {
				h, p, e := net.SplitHostPort(hp)
				if e == nil {
					hosts = append(hosts, h)
					ports = append(ports, p)
				} else {
					if strings.ContainsAny(hp, "[]:") {
						return nil, errConfiguracionPrivadaPerfiles
					}
					hosts = append(hosts, hp)
					ports = append(ports, "")
				}
			}
			campos["host"] = strings.Join(hosts, ",")
			for _, p := range ports {
				if p != "" {
					for _, otro := range ports {
						if otro == "" {
							return nil, errConfiguracionPrivadaPerfiles
						}
					}
					campos["port"] = strings.Join(ports, ",")
					break
				}
			}
		}
		if u.Path != "" {
			if !strings.HasPrefix(u.Path, "/") || strings.Contains(u.Path[1:], "/") {
				return nil, errConfiguracionPrivadaPerfiles
			}
			campos["dbname"] = u.Path[1:]
		}
		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, errConfiguracionPrivadaPerfiles
		}
		for k, v := range q {
			if len(v) != 1 || agregar(k, v[0]) != nil {
				return nil, errConfiguracionPrivadaPerfiles
			}
		}
		return campos, nil
	}
	// La gramática libpq admite comillas simples y escapes. Las claves repetidas
	// se deniegan, en vez de adoptar silenciosamente su última aparición.
	for i := 0; i < len(dsn); {
		for i < len(dsn) && strings.ContainsRune(" \t\r\n", rune(dsn[i])) {
			i++
		}
		if i == len(dsn) {
			break
		}
		inicio := i
		for i < len(dsn) && dsn[i] != '=' && !strings.ContainsRune(" \t\r\n", rune(dsn[i])) {
			i++
		}
		clave := dsn[inicio:i]
		for i < len(dsn) && strings.ContainsRune(" \t", rune(dsn[i])) {
			i++
		}
		if i == len(dsn) || dsn[i] != '=' {
			return nil, errConfiguracionPrivadaPerfiles
		}
		i++
		for i < len(dsn) && strings.ContainsRune(" \t", rune(dsn[i])) {
			i++
		}
		var valor strings.Builder
		quoted := i < len(dsn) && dsn[i] == '\''
		if quoted {
			i++
		}
		cerrada := !quoted
		for i < len(dsn) {
			c := dsn[i]
			if c == '\\' {
				i++
				if i == len(dsn) {
					return nil, errConfiguracionPrivadaPerfiles
				}
				valor.WriteByte(dsn[i])
				i++
				continue
			}
			if quoted && c == '\'' {
				i++
				cerrada = true
				break
			}
			if !quoted && strings.ContainsRune(" \t\r\n", rune(c)) {
				break
			}
			valor.WriteByte(c)
			i++
		}
		if !cerrada || i < len(dsn) && !strings.ContainsRune(" \t\r\n", rune(dsn[i])) || agregar(clave, valor.String()) != nil {
			return nil, errConfiguracionPrivadaPerfiles
		}
	}
	return campos, nil
}

func desactivarPassfileDSNPerfiles(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, _ := url.Parse(dsn)
		q := u.Query()
		q.Set("passfile", "/dev/null")
		u.RawQuery = q.Encode()
		return u.String()
	}
	campos, _ := camposDSNPerfiles(dsn)
	if _, ok := campos["passfile"]; ok {
		return dsn
	}
	return dsn + " passfile=/dev/null"
}
