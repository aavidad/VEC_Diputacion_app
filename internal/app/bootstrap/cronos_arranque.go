package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
)

type configuracionCronosValidada struct {
	archivo configuracionCronosEmpleadoDesarrollo
	motivos cronoscomp.MotivosCronos
	canal   cronosdomain.AcreditacionCanalMarcaje
	zona    *time.Location
	cuentas map[string]cuentaRutasDietasDesarrollo
}

// prepararCronosAntesDeCT verifica sus dependencias privadas antes de que el
// gobierno común publique ninguna audiencia de Cronos. El selector apagado no
// lee ficheros ni abre conexiones.
func prepararCronosAntesDeCT(cfg config.Config, identidad *resolvedorIdentidadDesarrollo) (bool, error) {
	solicitado := cfg.CronosEmpleadoEnabled != "" && cfg.CronosEmpleadoEnabled != "false" ||
		cfg.CronosResolucionEnabled != "" && cfg.CronosResolucionEnabled != "false" ||
		cfg.CronosNotificacionesEnabled != "" && cfg.CronosNotificacionesEnabled != "false"
	activo, err := cfg.CronosEmpleadoDesarrolloActivo()
	if err != nil {
		return solicitado, marcarFalloComponenteArranque("cronos_selector", err)
	}
	resolucion, err := cfg.CronosResolucionDesarrolloActiva()
	if err != nil {
		return solicitado, marcarFalloComponenteArranque("cronos_selector", err)
	}
	notificaciones, err := cfg.CronosNotificacionesDesarrolloActivas()
	if err != nil {
		return solicitado, marcarFalloComponenteArranque("cronos_selector", err)
	}
	if !activo {
		return false, nil
	}
	if identidad == nil {
		return true, marcarFalloComponenteArranque("cronos_identidad", ErrComposicionCronosEmpleadoNoDisponible)
	}
	datos, err := cargarConfiguracionCronosValidada(cfg, identidad, resolucion, notificaciones)
	if err != nil {
		return true, marcarFalloComponenteArranque("cronos_material_privado", ErrComposicionCronosEmpleadoNoDisponible)
	}
	c := datos.archivo
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	entradas := entradasConexionCronos(c)
	var pools []*pgxpool.Pool
	defer func() {
		for _, pool := range pools {
			pool.Close()
		}
	}()
	usuarios := make(map[string]bool, len(entradas))
	for _, entrada := range entradas {
		pool, usuario, fallo := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if fallo != nil {
			return true, marcarFalloComponenteArranque("cronos_conexiones", ErrComposicionCronosEmpleadoNoDisponible)
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return true, marcarFalloComponenteArranque("cronos_conexiones", ErrComposicionCronosEmpleadoNoDisponible)
		}
		usuarios[usuario] = true
	}
	if err := preflightCronosEmpleado(ctx, pools[6], pools[7], resolucion, notificaciones); err != nil {
		return true, marcarFalloComponenteArranque("cronos_migraciones", err)
	}
	return true, nil
}

func configuracionCTSinCronos(cfg config.Config) config.Config {
	cfg.CronosEmpleadoEnabled = "false"
	cfg.CronosResolucionEnabled = "false"
	cfg.CronosNotificacionesEnabled = "false"
	return cfg
}

func cargarConfiguracionCronosValidada(cfg config.Config, identidad *resolvedorIdentidadDesarrollo, resolucion, notificaciones bool) (configuracionCronosValidada, error) {
	var vacia configuracionCronosValidada
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-empleado.json"), 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	defer borrarBytes(contenido)
	var c configuracionCronosEmpleadoDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	return validarConfiguracionCronos(c, identidad, resolucion, notificaciones)
}

func validarConfiguracionCronos(c configuracionCronosEmpleadoDesarrollo, identidad *resolvedorIdentidadDesarrollo, resolucion, notificaciones bool) (configuracionCronosValidada, error) {
	var vacia configuracionCronosValidada
	if identidad == nil || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || len(c.Cuentas) == 0 || len(c.Cuentas) > 64 ||
		(c.ZonaHoraria != cronosdomain.ZonaSaldoPeninsula && c.ZonaHoraria != cronosdomain.ZonaSaldoCanarias) {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	canal, err := cronosdomain.NuevaAcreditacionCanalMarcaje(cronosdomain.DatosAcreditacionCanalMarcaje{
		PoliticaVersionRef: c.CanalRemoto.PoliticaVersionRef, CanalRef: c.CanalRemoto.CanalRef,
		OrigenRef: cronosdomain.OrigenMarcajeRemoto, CalidadRef: c.CanalRemoto.CalidadRef,
	})
	if err != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	zona, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	motivos := cronoscomp.MotivosCronos{Saldo: c.Motivos.Saldo, Marcaje: c.Motivos.Marcaje,
		Disponibilidad: c.Motivos.Disponibilidad, Recuperacion: c.Motivos.Recuperacion,
		Movimientos: c.Motivos.Movimientos, Correccion: c.Motivos.Correccion,
		Permisos: c.Motivos.Permisos, Permiso: c.Motivos.Permiso}
	if resolucion {
		motivos.Bandeja, motivos.Resolucion = c.Motivos.Bandeja, c.Motivos.Resolucion
		motivos.Avisos, motivos.ArchivoAviso = c.Motivos.Avisos, c.Motivos.ArchivoAviso
	}
	if notificaciones {
		motivos.Notificacion, motivos.Notificaciones = c.Motivos.Notificacion, c.Motivos.Notificaciones
		motivos.BandejaNotificaciones, motivos.AtencionNotificacion = c.Motivos.BandejaNotificaciones, c.Motivos.AtencionNotificacion
	}
	if !core.ReferenciaMotivoAutorizacionV2Valida(c.AuditoriaIntentos.MotivoDenegado) ||
		c.AuditoriaIntentos.MotivoDenegado.Referencia() == motivos.Saldo.Referencia() {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	comunes := []core.ReferenciaEntradaCatalogo{motivos.Marcaje, motivos.Disponibilidad, motivos.Recuperacion,
		motivos.Movimientos, motivos.Correccion, motivos.Permisos, motivos.Permiso}
	if resolucion {
		comunes = append(comunes, motivos.Bandeja, motivos.Resolucion, motivos.Avisos, motivos.ArchivoAviso)
	}
	if notificaciones {
		comunes = append(comunes, motivos.Notificacion, motivos.Notificaciones, motivos.BandejaNotificaciones, motivos.AtencionNotificacion)
	}
	for _, m := range comunes {
		if m.CatalogoID != motivos.Saldo.CatalogoID || m.Referencia() == c.AuditoriaIntentos.MotivoDenegado.Referencia() {
			return vacia, ErrComposicionCronosEmpleadoNoDisponible
		}
	}
	autorizador, err := cronoscomp.NuevoAutorizadorCronos(emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}, motivos)
	if err != nil || autorizador.ResolucionConfigurada() != resolucion || autorizador.NotificacionesConfiguradas() != notificaciones {
		return vacia, ErrComposicionCronosEmpleadoNoDisponible
	}
	cuentas := make(map[string]cuentaRutasDietasDesarrollo, len(c.Cuentas))
	for _, cuenta := range c.Cuentas {
		b, err := hex.DecodeString(cuenta.CertificadoSHA256)
		if err != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 ||
			cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return vacia, ErrComposicionCronosEmpleadoNoDisponible
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, ok := identidad.porHuella[digest]
		if !ok || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh {
			return vacia, ErrComposicionCronosEmpleadoNoDisponible
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	return configuracionCronosValidada{archivo: c, motivos: motivos, canal: canal, zona: zona, cuentas: cuentas}, nil
}

func entradasConexionCronos(c configuracionCronosEmpleadoDesarrollo) []struct{ dsn, rol string } {
	return []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"},
		{c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"},
		{c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"},
		{c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		{c.DSNCronos, "vec_cronos_v1_ejecutor"},
		{c.DSNCronosAuditor, "vec_cronos_v1_auditor"},
	}
}

// contextoPublicacionCronos conserva los valores de observabilidad del
// arranque, pero abre un plazo independiente del usado por CT. Se cancela al
// terminar la publicación opcional, tanto si confirma como si revierte.
func contextoPublicacionCronos(ctxCT context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctxCT), plazoarranque.Ampliar(15*time.Second))
}

// publicarMaterialCronosEnLote no confirma ninguna audiencia si falla una
// publicación. Emplea la transacción y el gobierno V3 de CT ya existentes.
func publicarMaterialCronosEnLote(ctx context.Context, gobierno *pgxpool.Pool, material materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo, seleccion seleccionMaterialCTDesarrollo) (materialCronosDesdeCTDesarrollo, error) {
	var vacio materialCronosDesdeCTDesarrollo
	if !seleccion.cronos {
		return vacio, nil
	}
	descriptores := descriptoresMaterialCronosDesarrollo()
	if seleccion.cronosResolucion {
		descriptores = append(descriptores, descriptoresMaterialCronosResolucionDesarrollo()...)
	}
	if seleccion.cronosAvisos {
		descriptores = append(descriptores, descriptoresMaterialCronosNotificacionesDesarrollo()...)
	}
	derivados := make([]materialAtestacionContratacionTemporalDesarrollo, 0, len(descriptores))
	defer func() {
		for i := range derivados {
			borrarBytes(derivados[i].claveHMAC)
		}
	}()
	for _, descriptor := range descriptores {
		if _, ok := catalogo.descriptorPara(descriptor.Audiencia); !ok {
			return vacio, ErrComposicionCronosEmpleadoNoDisponible
		}
		derivado, err := derivarMaterialConsumidorV3Desarrollo(material, descriptor)
		if err != nil {
			return vacio, err
		}
		derivados = append(derivados, derivado)
	}
	proveedores := make([]*proveedorMaterialAltaContratacionTemporalDesarrollo, len(derivados))
	err := ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range derivados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &derivados[i]); err != nil {
				return err
			}
			proveedor, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(derivados[i], reloj)
			if err != nil {
				return err
			}
			proveedores[i] = proveedor
		}
		return nil
	})
	if err != nil {
		return vacio, err
	}
	var base [8]*proveedorMaterialAltaContratacionTemporalDesarrollo
	copy(base[:], proveedores[:8])
	resultado := materialCronosDesdeProveedores(base)
	indice := 8
	if seleccion.cronosResolucion {
		var lote [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
		copy(lote[:], proveedores[indice:indice+4])
		resultado = resultado.conResolucion(lote)
		indice += 4
	}
	if seleccion.cronosAvisos {
		var lote [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
		copy(lote[:], proveedores[indice:indice+4])
		resultado = resultado.conNotificaciones(lote)
	}
	return resultado, nil
}

func registrarFalloCronosArranque(err error) {
	if err == nil {
		return
	}
	causa := "no_disponible"
	switch {
	case errors.Is(err, ErrCronosFalta000009):
		causa = "falta_cronos_000009"
	case errors.Is(err, ErrCronosFaltaAD358):
		causa = "falta_ad3_58"
	case errors.Is(err, ErrCronosFalta000010):
		causa = "falta_cronos_000010"
	}
	slog.Warn("Cronos no disponible", "componente", ComponenteFalloArranque(err), "causa", causa)
}

func componerRaizConCronosIndisponible(raiz http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && (r.URL.Path == "/api/interna/cronos" || len(r.URL.Path) >= len(prefijoRutasCronosEmpleado) && r.URL.Path[:len(prefijoRutasCronosEmpleado)] == prefijoRutasCronosEmpleado) {
			responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
			return
		}
		raiz.ServeHTTP(w, r)
	})
}
