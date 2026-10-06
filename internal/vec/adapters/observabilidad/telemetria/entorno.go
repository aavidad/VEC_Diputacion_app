package telemetria

import (
	"strconv"
	"strings"
	"time"
)

// Variables de entorno que Sistemas puede fijar sin recompilar.
const (
	EnvUmbralLentaMS   = "VEC_TELEMETRIA_LENTA_MS"
	EnvUmbralEnCursoS  = "VEC_TELEMETRIA_EN_CURSO_S"
	maxUmbralLentaMS   = 600_000
	maxUmbralEnCursoS  = 3_600
	EnvUmbralConsultas = "VEC_TELEMETRIA_LENTA_CONSULTAS"
	maxUmbralConsultas = 100_000
)

// UmbralesDeEntorno lee los umbrales. Un valor ausente usa el
// predeterminado; uno presente pero no válido también, y valido es false
// para que el arranque deje constancia con un texto fijo.
func UmbralesDeEntorno(getenv func(string) string) (u Umbrales, valido bool) {
	valido = true
	if getenv == nil {
		return u, valido
	}
	if n, ok, presente := enteroAcotado(getenv(EnvUmbralLentaMS), maxUmbralLentaMS); presente {
		if ok {
			u.Lenta = time.Duration(n) * time.Millisecond
		} else {
			valido = false
		}
	}
	if n, ok, presente := enteroAcotado(getenv(EnvUmbralConsultas), maxUmbralConsultas); presente {
		if ok {
			u.Consultas = n
		} else {
			valido = false
		}
	}
	if n, ok, presente := enteroAcotado(getenv(EnvUmbralEnCursoS), maxUmbralEnCursoS); presente {
		if ok {
			u.EnCurso = time.Duration(n) * time.Second
		} else {
			valido = false
		}
	}
	return u, valido
}

func enteroAcotado(valor string, maximo int) (n int, ok, presente bool) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return 0, false, false
	}
	n, err := strconv.Atoi(valor)
	if err != nil || n < 1 || n > maximo {
		return 0, false, true
	}
	return n, true, true
}
