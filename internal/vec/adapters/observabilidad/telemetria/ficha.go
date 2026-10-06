package telemetria

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// Ficha acompaña a una petición por su contexto y reúne lo que el registro de
// acceso escribe al terminar. La crea el middleware de acceso; los
// adaptadores solo anotan valores cerrados mediante las funciones del
// paquete, que no hacen nada si el contexto no lleva ficha.
type Ficha struct {
	inicio time.Time
	metodo string
	// rutaNormalizada se calcula al entrar, a partir del camino sin valores.
	rutaNormalizada string
	correlacion     string

	mu sync.Mutex
	// avisosEnCurso cuenta las líneas "en curso" ya escritas.
	avisosEnCurso int
	ruta          string
	incidencia    string
	componente    string
	etapaInc      string
	causa         string
	etapaFallo    string
}

type claveFicha struct{}

// FichaDe devuelve la ficha de la petición o nil.
func FichaDe(ctx context.Context) *Ficha {
	if ctx == nil {
		return nil
	}
	f, _ := ctx.Value(claveFicha{}).(*Ficha)
	return f
}

func conFicha(ctx context.Context, f *Ficha) context.Context {
	return context.WithValue(ctx, claveFicha{}, f)
}

// AnotarRuta fija la plantilla de la ruta (por ejemplo
// "/api/vec/bolsa/{bolsa}/participaciones"). Gana la primera anotación: el
// enrutador más interno termina antes que los externos.
func AnotarRuta(ctx context.Context, plantilla string) {
	f := FichaDe(ctx)
	if f == nil {
		return
	}
	plantilla = plantillaDePatron(plantilla)
	if plantilla == "" {
		return
	}
	f.mu.Lock()
	if f.ruta == "" {
		f.ruta = plantilla
	}
	f.mu.Unlock()
}

// AnotarFallo deja en el registro técnico la clase del error y la etapa en
// la que ocurrió. Nunca guarda el texto del error: solo la clase cerrada de
// ClasificarError. Se conserva el primer fallo, que suele ser la causa.
func AnotarFallo(ctx context.Context, etapa string, err error) {
	f := FichaDe(ctx)
	if f == nil || err == nil {
		return
	}
	etapa = identificadorSeguro(etapa, 40)
	if etapa == "" {
		etapa = "sin_etapa"
	}
	causa := ClasificarError(err)
	f.mu.Lock()
	if f.causa == "" {
		f.causa, f.etapaFallo = causa, etapa
	}
	f.mu.Unlock()
}

// anotarIncidencia guarda el código de incidencia que un adaptador declaró
// durante la petición. Gana la primera.
func (f *Ficha) anotarIncidencia(codigo, componente, etapa string) {
	f.mu.Lock()
	if f.incidencia == "" {
		f.incidencia, f.componente, f.etapaInc = codigo, componente, etapa
	}
	f.mu.Unlock()
}

// CapturarPatron envuelve un http.ServeMux (o cualquier manejador que fije
// Request.Pattern) y anota en la ficha el patrón con el que se atendió la
// petición. Sirve para que el registro use "/x/{ref}" en lugar del camino.
func CapturarPatron(siguiente http.Handler) http.Handler {
	if siguiente == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siguiente.ServeHTTP(w, r)
		if r.Pattern != "" {
			AnotarRuta(r.Context(), r.Pattern)
		}
	})
}

// ClasificarError reduce un error a una clase cerrada apta para el registro
// técnico. Nunca devuelve el texto del error, que puede contener rutas, DSN
// o datos de la petición.
func ClasificarError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "cancelada"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "plazo_vencido"
	}
	var conEstado interface{ SQLState() string }
	if errors.As(err, &conEstado) {
		if codigo := conEstado.SQLState(); esSQLState(codigo) {
			return "bd_" + codigo
		}
		return "bd"
	}
	var red net.Error
	if errors.As(err, &red) {
		if red.Timeout() {
			return "red_plazo"
		}
		return "red"
	}
	return "desconocida"
}

func esSQLState(codigo string) bool {
	if len(codigo) != 5 {
		return false
	}
	for i := 0; i < len(codigo); i++ {
		c := codigo[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return false
		}
	}
	return true
}

// identificadorSeguro admite solo [a-z0-9_.-] y acota la longitud; cualquier
// otro carácter invalida el valor entero.
func identificadorSeguro(valor string, maximo int) string {
	if valor == "" || len(valor) > maximo {
		return ""
	}
	for i := 0; i < len(valor); i++ {
		c := valor[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '.' && c != '-' {
			return ""
		}
	}
	return valor
}
