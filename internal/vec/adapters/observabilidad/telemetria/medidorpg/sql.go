package medidorpg

import "strings"

const (
	// maxEscaneoSQL acota lo que se lee del texto de la consulta: el nombre
	// de la operación siempre aparece al principio.
	maxEscaneoSQL        = 2048
	maxIdentificador     = 63
	operacionDesconocida = "sql"
)

// palabrasPrevias introducen el objeto principal de una consulta.
var palabrasPrevias = map[string]bool{
	"from": true, "join": true, "into": true, "update": true, "call": true, "table": true,
}

// ordenesConocidas se usan como operación si no aparece ningún objeto
// (BEGIN, COMMIT, SET…).
var ordenesConocidas = map[string]bool{
	"select": true, "insert": true, "update": true, "delete": true, "with": true, "begin": true,
	"commit": true, "rollback": true, "set": true, "reset": true, "savepoint": true, "release": true,
	"call": true, "values": true, "show": true, "discard": true, "listen": true, "unlisten": true,
	"notify": true, "lock": true, "declare": true, "fetch": true, "close": true, "copy": true,
	"analyze": true, "explain": true, "prepare": true, "execute": true, "deallocate": true,
	"start": true, "end": true, "abort": true, "merge": true, "truncate": true,
}

// operacionSQL obtiene un nombre estable y sin datos para una consulta:
// la primera función cualificada que se invoca (esquema.funcion) o el
// objeto que sigue a FROM, JOIN, INTO, UPDATE, CALL o TABLE. Si no hay
// ninguno, la orden (begin, commit, set…). Se saltan literales, nombres
// entre comillas y comentarios, de modo que ningún valor llega al registro.
func operacionSQL(sql string) string {
	if len(sql) > maxEscaneoSQL {
		sql = sql[:maxEscaneoSQL]
	}
	var primera string
	esperaObjeto := false
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case c == '\'' || c == '"':
			i = saltarCitado(sql, i+1, c)
			esperaObjeto = false
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			fin := strings.Index(sql[i+2:], "*/")
			if fin < 0 {
				i = len(sql)
			} else {
				i += fin + 4
			}
		case c == '$' && i+1 < len(sql) && (sql[i+1] == '$' || esLetra(sql[i+1])):
			// Cadena con dólares ($$…$$ o $etiqueta$…$etiqueta$).
			i = saltarDolares(sql, i)
			esperaObjeto = false
		case esLetra(c):
			nombre, fin := leerNombre(sql, i)
			i = fin
			if nombre == "" {
				continue
			}
			if primera == "" {
				primera = nombre
			}
			if esperaObjeto && !palabrasPrevias[nombre] {
				return nombre
			}
			if strings.Contains(nombre, ".") && i < len(sql) && sql[i] == '(' {
				return nombre
			}
			esperaObjeto = palabrasPrevias[nombre]
		default:
			// Cualquier signo (paréntesis, coma, $1…) corta la espera del
			// objeto; los espacios no.
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
				esperaObjeto = false
			}
			i++
		}
	}
	if ordenesConocidas[primera] {
		return primera
	}
	return operacionDesconocida
}

// leerNombre lee un identificador, opcionalmente cualificado (a.b o a.b.c),
// en minúsculas. Un identificador demasiado largo invalida el nombre.
func leerNombre(sql string, i int) (string, int) {
	inicio := i
	for i < len(sql) && (esLetra(sql[i]) || esDigito(sql[i]) || sql[i] == '.' || sql[i] == '$') {
		i++
	}
	partes := strings.Split(strings.Trim(strings.ToLower(sql[inicio:i]), "."), ".")
	if len(partes) > 3 {
		return "", i
	}
	for j, p := range partes {
		partes[j] = identificadorSQL(p)
		if partes[j] == "" {
			return "", i
		}
	}
	return strings.Join(partes, "."), i
}

func saltarCitado(sql string, i int, cierre byte) int {
	for i < len(sql) {
		if sql[i] == cierre {
			if i+1 < len(sql) && sql[i+1] == cierre {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return i
}

func saltarDolares(sql string, i int) int {
	fin := strings.IndexByte(sql[i+1:], '$')
	if fin < 0 {
		return len(sql)
	}
	etiqueta := sql[i : i+fin+2]
	resto := strings.Index(sql[i+len(etiqueta):], etiqueta)
	if resto < 0 {
		return len(sql)
	}
	return i + len(etiqueta) + resto + len(etiqueta)
}

// identificadorSQL admite solo [a-z0-9_] empezando por letra o guion bajo.
func identificadorSQL(v string) string {
	if v == "" || len(v) > maxIdentificador || esDigito(v[0]) {
		return ""
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if (c < 'a' || c > 'z') && !esDigito(c) && c != '_' {
			return ""
		}
	}
	return v
}

func nombreCualificado(partes []string) string {
	limpias := make([]string, 0, len(partes))
	for _, p := range partes {
		if l := identificadorSQL(strings.ToLower(p)); l != "" {
			limpias = append(limpias, l)
		}
	}
	if len(limpias) == 0 || len(limpias) != len(partes) {
		return "tabla"
	}
	return strings.Join(limpias, ".")
}

func esLetra(c byte) bool  { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' }
func esDigito(c byte) bool { return c >= '0' && c <= '9' }
