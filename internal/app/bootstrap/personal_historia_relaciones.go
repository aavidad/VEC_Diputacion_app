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
)

type configuracionHistoriaRelacionesPersonal struct {
	Motivo   core.ReferenciaEntradaCatalogo                             `json:"motivo"`
	Intentos personalcomp.ConfiguracionIntentosHistoriaRelacionesPropia `json:"intentos"`
}

func descriptorMaterialHistoriaRelacionesPersonal() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        personaldomain.AudienciaHistoriaRelacionesPropia,
		Dominio:          "vec.personal.historia-relaciones.capacidad-v3",
		Prefijo:          "clave:capacidad:personal-historia-relaciones:",
		ProveedorNominal: "proveedor-material-personal-historia-relaciones",
	}
}

// La presencia de configuración privada selecciona material, nunca permisos.
// Un material inválido no activa silenciosamente otra audiencia.
func historiaRelacionesPersonalSolicitada(cfg config.Config) (bool, error) {
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
	return c.HistoriaRelaciones != nil, nil
}

func componerHistoriaRelacionesPersonal(ctx context.Context, pool *pgxpool.Pool, identidad seguridadPersonalEmpleadoDesarrollo, emisor emisorMaterialDietasDesarrollo, c configuracionHistoriaRelacionesPersonal, destino vecports.RegistradorIntentosAuditoria, proceso string, limite time.Duration) (http.Handler, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || identidad.autoridad == nil || dependenciaDietasNula(emisor) || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Intentos.Proceso != proceso || c.Intentos.Canal != string(core.SuperficieAutenticacionInternaCorporativaV1) {
		return nil, errPersonalEmpleadoEn()
	}
	d, ok := destino.(personalcomp.RegistradorIntentosHistoriaRelacionesPropia)
	if !ok {
		return nil, errPersonalEmpleadoEn()
	}
	registro, err := personalcomp.NuevoRegistroIntentosHistoriaRelacionesPropia(d, c.Intentos)
	if err != nil || registro.VerificarRegistroHistoriaRelacionesPropia(ctx) != nil || personalpg.PreflightEjecutorHistoriaRelacionesPropia(ctx, pool) != nil {
		return nil, errPersonalEmpleadoEn()
	}
	proveedor, err := personalcomp.NuevoProveedorAutorizacionHistoriaRelacionesPropia(identidad, emisor, c.Motivo)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	repo, err := personalpg.NuevoRepositorioHistoriaRelacionesPropiaPostgreSQL(pool)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	servicio, err := personalapp.NuevoServicioHistoriaRelacionesPropia(proveedor, repo, registro)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	handler, err := personalhttp.NuevoManejadorHistoriaRelacionesPropia(identidad, servicio, registro, time.Now)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	return capturarPeticionPersonal(identidad, handler, limite), nil
}
