package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Los pools ya acreditados por la raíz suministran fuente, registro y motivos.
// S2 aporta descriptores nominales al mismo catálogo/PDP de CT y Bolsa;
// no reutiliza la política del soporte CT ni publica permisos en petición.
func (m *MontajePreparacionBasesV3) autorizacionesPostgreSQL(fuente, registro, motivos *pgxpool.Pool) ([]descriptorAutorizacionComunDesarrollo, error) {
	if m == nil || fuente == nil || registro == nil || motivos == nil {
		return nil, errMontajePreparacionBasesV3
	}
	f, err := pgvec.NuevoAlmacenAutorizacion(fuente)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	r, err := pgvec.NuevoAlmacenAutorizacion(registro)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	v, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivos, m.configuracion.MotivoGuardar.CatalogoID)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	for _, motivo := range []core.ReferenciaEntradaCatalogo{m.configuracion.MotivoIntentoDenegado, m.configuracion.MotivoIntentoError} {
		if v.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, m.reloj.Ahora()) != nil {
			return nil, errMontajePreparacionBasesV3
		}
	}
	p, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(f, r, r, v)
	if err != nil {
		return nil, errMontajePreparacionBasesV3
	}
	return m.Autorizaciones(p)
}

func (m *MontajePreparacionBasesV3) rutasDesdeRaiz(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo,
	identidad *proveedorSesionConsultaRRHHDesarrollo, fronteras catalogoFronterasComunDesarrollo, autoridades ...*pgxpool.Pool,
) ([]vechttp.RutaExacta, func(), error) {
	if m == nil || alta == nil || identidad == nil || !identidad.fronteras.mismaInstancia(fronteras) {
		return nil, nil, errMontajePreparacionBasesV3
	}
	comun, ok := alta.autorizador.(*autorizadorAnalisisContratacionTemporalDesarrollo)
	if !ok || comun == nil || !comun.instalado {
		return nil, nil, errMontajePreparacionBasesV3
	}
	auditar, err := NuevaAuditoriaFronteraPreparacionBasesV3(alta.postgresql.registradorAuditoriaFrontera)
	if err != nil {
		return nil, nil, err
	}
	var reservados []string
	pools := append([]*pgxpool.Pool{alta.postgresql.ejecucion, alta.postgresql.gobierno,
		alta.postgresql.registroAutorizacion, alta.postgresql.confirmador,
		alta.postgresql.auditoriaFrontera, alta.postgresql.bolsa, alta.postgresql.calculadorPoliticaOfertas}, autoridades...)
	for _, pool := range pools {
		if pool != nil {
			reservados = append(reservados, pool.Config().ConnConfig.User)
		}
	}
	registrador, proceso, cerrarIntentos, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctx,
		config.Config{DevelopmentMaterialDir: m.directorio}, alta.postgresql.gobierno, reservados)
	if err != nil {
		return nil, nil, err
	}
	rutas, cerrar, err := m.Componer(ctx, DependenciasMontajePreparacionBasesV3{SesionBase: identidad, Fronteras: fronteras,
		PDP: comun, Gobierno: alta.postgresql.gobierno,
		MaterialGuardar: alta.postgresql.materialPreparacionBases[0], MaterialConsultar: alta.postgresql.materialPreparacionBases[1],
		RegistrarRechazoFrontera: auditar, RegistradorIntentos: registrador, ProcesoIntentos: proceso, LoginsReservados: reservados})
	if err != nil {
		cerrarIntentos()
		return nil, nil, err
	}
	return rutas, func() { cerrar(); cerrarIntentos() }, nil
}
