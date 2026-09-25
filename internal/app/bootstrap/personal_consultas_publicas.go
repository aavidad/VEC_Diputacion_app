package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"vec-diputacion-granada/config"
	personalorganizacion "vec-diputacion-granada/internal/modules/personal/adapters/organizacionpublica"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rptpublica"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// consultasPublicasPersonal reúne las tres consultas de Personal que no
// contienen personas: catálogo profesional gobernado, RPT publicada y
// estructura organizativa de referencia. Cada una lee su fuente versionada e
// inmovilizada por huella; si falta o no se valida, su ruta responde 503 en
// lugar de caer en la carcasa o en datos sustitutos.
type consultasPublicasPersonal struct {
	categorias http.Handler
	rpt        http.Handler
	estructura http.Handler
}

const plazoValidacionConsultasPublicasPersonal = 10 * time.Second

// nuevasConsultasPublicasPersonal compone las consultas con las fuentes ya
// configuradas en la raíz. No detiene el arranque: una fuente ausente o
// incompatible deja solo su ruta en «no disponible» y se anota en el registro
// sin rutas ni contenido.
func nuevasConsultasPublicasPersonal(
	cfg config.Config,
	categorias *personalapp.ServicioConsultaCategoriasProfesionales,
	registro io.Writer,
) consultasPublicasPersonal {
	anotar := func(consulta, motivo string) {
		if registro != nil {
			_, _ = fmt.Fprintf(registro, "personal: consulta publica %s no disponible (%s)\n", consulta, motivo)
		}
	}
	consultas := consultasPublicasPersonal{
		categorias: consultaPublicaPersonalNoDisponible("catalogo_categorias_profesionales_no_disponible"),
		rpt:        consultaPublicaPersonalNoDisponible("rpt_publica_no_disponible"),
		estructura: consultaPublicaPersonalNoDisponible("estructura_organizativa_publica_no_disponible"),
	}
	if manejador, err := vechttp.NewHandlerCategoriasProfesionalesPublicas(categorias); err == nil {
		consultas.categorias = manejador
	} else {
		anotar("categorias", "catalogo no compuesto")
	}
	if ruta := strings.TrimSpace(cfg.RPTCatalogoPath); ruta == "" {
		anotar("rpt", "fuente no configurada")
	} else if manejador, err := nuevoManejadorRPTPublicaPersonal(ruta); err != nil {
		anotar("rpt", "fuente no valida")
	} else {
		consultas.rpt = manejador
	}
	if ruta := strings.TrimSpace(cfg.PersonalOrganizacionSourcePath); ruta == "" {
		anotar("estructura", "fuente no configurada")
	} else if manejador, err := nuevoManejadorEstructuraPublicaPersonal(ruta); err != nil {
		anotar("estructura", "fuente no valida")
	} else {
		consultas.estructura = manejador
	}
	return consultas
}

func nuevoManejadorRPTPublicaPersonal(ruta string) (http.Handler, error) {
	fuente, err := personalrpt.NuevaFuente(ruta)
	if err != nil {
		return nil, err
	}
	servicio, err := personalapp.NuevoServicioConsultaRPTPublica(fuente)
	if err != nil {
		return nil, err
	}
	// La fuente se lee en cada consulta; se comprueba una vez al componer para
	// no ofrecer una ruta cuya huella ya no coincide.
	ctx, cancelar := context.WithTimeout(context.Background(), plazoValidacionConsultasPublicasPersonal)
	defer cancelar()
	catalogo, err := servicio.Listar(ctx)
	if err != nil {
		return nil, err
	}
	if err := catalogo.Validar(); err != nil {
		return nil, err
	}
	return vechttp.NewHandlerRPTPublica(servicio)
}

func nuevoManejadorEstructuraPublicaPersonal(ruta string) (http.Handler, error) {
	fuente, err := personalorganizacion.NuevaFuente(ruta)
	if err != nil {
		return nil, err
	}
	servicio, err := personalapp.NuevoServicioConsultaEstructuraOrganizativaPublica(fuente)
	if err != nil {
		return nil, err
	}
	return vechttp.NewHandlerEstructuraOrganizativaPublica(servicio)
}

func consultaPublicaPersonalNoDisponible(codigo string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": codigo})
	})
}

// componerRaizConPersonalPublico antepone las tres rutas exactas a la raíz.
// Solo la lectura del catálogo profesional se desvía de la carcasa: el resto
// de métodos sobre esa ruta siguen en la carcasa, que exige su permiso de
// gestión. Las otras dos rutas no existen en la carcasa y responden aquí a
// cualquier método.
func componerRaizConPersonalPublico(raiz http.Handler, consultas consultasPublicasPersonal) http.Handler {
	if raiz == nil || consultas.categorias == nil || consultas.rpt == nil || consultas.estructura == nil {
		return raiz
	}
	mux := http.NewServeMux()
	mux.Handle(vechttp.RutaRPTPublicaPersonal, consultas.rpt)
	mux.Handle(vechttp.RutaEstructuraOrganizativaPublicaPersonal, consultas.estructura)
	mux.Handle(vechttp.RutaCategoriasProfesionalesPersonal, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			consultas.categorias.ServeHTTP(w, r)
			return
		}
		raiz.ServeHTTP(w, r)
	}))
	mux.Handle("/", raiz)
	return mux
}
