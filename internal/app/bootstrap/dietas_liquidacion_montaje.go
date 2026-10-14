package bootstrap

import (
	"net/http"
	"strings"
	"time"

	dietascomp "vec-diputacion-granada/internal/modules/dietas/adapters/composicion"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type dependenciasLiquidacionEconomica struct {
	config *configuracionLiquidacionDietas
	fuente vecports.FuenteAutorizacion
	motivo vecdomain.ReferenciaEntradaCatalogo
}

type relojLiquidacionEconomica struct{ origen vecports.Reloj }

func (r relojLiquidacionEconomica) AhoraUTC() time.Time {
	return r.origen.Ahora().UTC().Truncate(time.Microsecond)
}

func componerLiquidacionEconomica(d *dependenciasLiquidacionEconomica, identidad *identidadPersonalDietas,
	resolver *dietascomp.ResolutorIdentidadEfectivaCircuito, circuito *dietasapp.ServicioCircuitoComision,
	repo *dietaspg.RepositorioBorradorComisionPostgreSQL, reloj vecports.Reloj) http.Handler {
	if d == nil || d.config == nil || d.fuente == nil || d.motivo.Validar() != nil || identidad == nil || resolver == nil || circuito == nil || repo == nil || reloj == nil {
		return nil
	}
	servicio, err := dietasapp.NuevoServicioLiquidacionEconomica(repo, relojLiquidacionEconomica{origen: reloj})
	if err != nil {
		return nil
	}
	preparador := &preparadorLiquidacionEconomica{identidad: resolver, circuito: circuito, catalogo: repo}
	captura := &capturadorLecturaLiquidacion{identidad: identidad, fuente: d.fuente}
	manejador, err := dietashttp.NuevoManejadorLiquidacionEconomica(preparador, resolver, servicio, captura, repo)
	if err != nil {
		return nil
	}
	documento := componerDocumentoLiquidacion(captura, repo)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil {
			if _, _, ok := dietashttp.RutaDocumentoLiquidacion(r.URL.Path); ok {
				if documento != nil {
					documento.ServeHTTP(w, r)
					return
				}
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		manejador.ServeHTTP(w, r)
	})
}

func esRutaLiquidacionEconomica(ruta string) bool {
	return strings.Contains(ruta, "/liquidaciones") &&
		(strings.HasPrefix(ruta, dietashttp.RutaCircuito+"/") || strings.HasPrefix(ruta, dietashttp.RutaBorradores+"/"))
}

func metodoLiquidacionEconomicaValido(ruta, metodo string) bool {
	var resto string
	var circuito bool
	if s, ok := strings.CutPrefix(ruta, dietashttp.RutaCircuito+"/"); ok {
		resto, circuito = s, true
	} else if s, ok := strings.CutPrefix(ruta, dietashttp.RutaBorradores+"/"); ok {
		resto = s
	} else {
		return false
	}
	partes := strings.Split(resto, "/")
	if len(partes) < 2 || !referenciaRutaComisionDietas.MatchString(partes[0]) || partes[1] != "liquidaciones" {
		return false
	}
	if len(partes) == 2 {
		return circuito && metodo == http.MethodPost
	}
	if len(partes) == 4 && partes[3] == "documento" && partes[2] != "propuesta" {
		return metodo == http.MethodGet && (partes[2] == "actual" || versionRutaLiquidacionValida(partes[2]))
	}
	if len(partes) != 3 || metodo != http.MethodGet {
		return false
	}
	if partes[2] == "propuesta" {
		return circuito
	}
	if partes[2] == "actual" {
		return true
	}
	return versionRutaLiquidacionValida(partes[2])
}

func versionRutaLiquidacionValida(s string) bool {
	if len(s) < 1 || len(s) > 18 || s[0] == '0' {
		return false
	}
	var version uint64
	for _, digito := range s {
		if digito < '0' || digito > '9' {
			return false
		}
		version = version*10 + uint64(digito-'0')
	}
	return version >= 2 && version <= 999999999999999999
}
