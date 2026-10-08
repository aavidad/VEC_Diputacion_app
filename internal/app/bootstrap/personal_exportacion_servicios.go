package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/web"
)

type configuracionExportacionServiciosPersonal struct {
	Motivo   core.ReferenciaEntradaCatalogo                                `json:"motivo"`
	Intentos personalcomp.ConfiguracionIntentosExportacionServiciosPropios `json:"intentos"`
}

func descriptorMaterialExportacionServiciosPersonal() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        personaldomain.AudienciaExportacionServiciosPropios,
		Dominio:          "vec.personal.exportacion-servicios.capacidad-v3",
		Prefijo:          "clave:capacidad:personal-exportacion-servicios:",
		ProveedorNominal: "proveedor-material-personal-exportacion-servicios",
	}
}

// La presencia de configuración privada selecciona material, nunca permisos.
// Un material inválido no activa silenciosamente otra audiencia.
func exportacionServiciosPersonalSolicitada(cfg config.Config) (bool, error) {
	if !personalEmpleadoSolicitado(cfg.PersonalEmpleadoEnabled) {
		return false, nil
	}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "personal-empleado.json"), 256<<10)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, errPersonalEmpleadoEn()
	}
	defer borrarBytes(b)
	if validarClavesJSONUnicas(b) != nil {
		return false, errPersonalEmpleadoEn()
	}
	var c configuracionPersonalEmpleadoDesarrollo
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Version != 2 {
		return false, errPersonalEmpleadoEn()
	}
	return c.ExportacionServicios != nil, nil
}

func componerExportacionServiciosPersonal(ctx context.Context, pool *pgxpool.Pool, identidad seguridadPersonalEmpleadoDesarrollo, emisor emisorMaterialDietasDesarrollo, c configuracionExportacionServiciosPersonal, destino vecports.RegistradorIntentosAuditoria, proceso string, limite time.Duration) (http.Handler, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || identidad.autoridad == nil || dependenciaDietasNula(emisor) || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Intentos.Proceso != proceso || c.Intentos.Canal != string(core.SuperficieAutenticacionInternaCorporativaV1) {
		return nil, errPersonalEmpleadoEn()
	}
	d, ok := destino.(personalcomp.RegistradorIntentosExportacionServiciosPropios)
	if !ok {
		return nil, errPersonalEmpleadoEn()
	}
	registro, err := personalcomp.NuevoRegistroIntentosExportacionServiciosPropios(d, c.Intentos)
	if err != nil || registro.VerificarRegistroExportacionServiciosPropios(ctx) != nil || personalpg.PreflightEjecutorExportacionServiciosPropios(ctx, pool) != nil {
		return nil, errPersonalEmpleadoEn()
	}
	catalogos, err := web.CatalogosExportacionServicios()
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	formatos, err := personalcomp.NuevoProveedorFormatosExportacionServiciosPropios(catalogos)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	proveedor, err := personalcomp.NuevoProveedorAutorizacionExportacionServiciosPropios(identidad, emisor, c.Motivo)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	repo, err := personalpg.NuevoRepositorioExportacionServiciosPropiosPostgreSQL(pool)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	servicio, err := personalapp.NuevoServicioExportacionServiciosPropios(proveedor, repo, formatos, registro)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	handler, err := personalhttp.NuevoManejadorExportacionServiciosPropios(identidad, servicio, registro)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	return capturarPeticionPersonal(identidad, handler, limite), nil
}

func capturarPeticionPersonal(identidad seguridadPersonalEmpleadoDesarrollo, siguiente http.Handler, limite time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || siguiente == nil {
			responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
		ctx := r.Context()
		var err error
		if _, ok := vecports.CorrelacionIncidenciasPeticion(ctx); !ok {
			ctx, err = vecports.ConCorrelacionIncidenciasPeticion(ctx)
		}
		if err == nil {
			ctx, err = personalcomp.PrepararContextoIntentoFichaPropia(ctx, identidad, limite)
		}
		if err != nil {
			responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}
