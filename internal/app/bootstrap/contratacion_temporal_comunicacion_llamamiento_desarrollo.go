package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Catálogos sintéticos explícitos de esta composición. No son canales
// corporativos, política de plazos ni evidencia de entrega de una notificación.
const (
	canalComunicacionLlamamientoDesarrollo    = `{"esquema":"vec.ct.canal-comunicacion.desarrollo.v1","tipo":"registro_local","envio_externo":false}`
	politicaComunicacionLlamamientoDesarrollo = `{"esquema":"vec.ct.politica-comunicacion.desarrollo.v1","antecedente":"recibo_seleccion_confirmado","abre_plazo":false}`
)

type claveSolicitudComunicacionLlamamientoDesarrollo struct{}

type autorizacionComunicacionLlamamientoDesarrollo interface {
	AutorizarOperacion(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type proveedorComunicacionLlamamientoDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

var _ postgresct.ProveedorRegistroComunicacionLlamamiento = (*proveedorComunicacionLlamamientoDesarrollo)(nil)

type ejecutorComunicacionLlamamientoDesarrollo struct {
	directorioComunicaciones string
	soporte                  *soporteAltaContratacionTemporalDesarrollo
	lector                   ports.LectorExpedienteSeleccionLlamamiento
	servicio                 httpinterno.EjecutorComunicacionLlamamiento
}

var _ httpinterno.EjecutorComunicacionLlamamiento = (*ejecutorComunicacionLlamamientoDesarrollo)(nil)

// No abre conexiones, publica catálogos, aplica SQL ni registra rutas.
// La raíz conserva propiedad y cierre de los pools. El antecedente de Bolsa
// se acredita exclusivamente con el recibo confirmado que ya conserva CT.
func nuevoEjecutorComunicacionLlamamientoDesarrollo(
	poolCT *pgxpool.Pool,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo,
	reloj ports.Reloj,
	lector ports.LectorExpedienteSeleccionLlamamiento,
	directorioComunicaciones string,
) (*ejecutorComunicacionLlamamientoDesarrollo, error) {
	if poolCT == nil || alta == nil || alta.soporte == nil || material == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(alta.autorizador) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(lector) ||
		!rutaDirectorioComunicacionesValida(directorioComunicaciones) {
		return nil, application.ErrServicioComunicacionLlamamientoInvalido
	}
	proveedor := &proveedorComunicacionLlamamientoDesarrollo{
		soporte: alta.soporte, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: material, comunicacion: true},
	}
	transaccion, err := postgresct.NuevaTransaccionComunicacionLlamamientoPostgreSQL(poolCT, proveedor)
	if err != nil {
		return nil, application.ErrServicioComunicacionLlamamientoInvalido
	}
	servicio, err := application.NuevoServicioComunicacionLlamamiento(transaccion)
	if err != nil {
		return nil, err
	}
	return &ejecutorComunicacionLlamamientoDesarrollo{
		directorioComunicaciones: directorioComunicaciones,
		soporte:                  alta.soporte, lector: lector, servicio: servicio,
	}, nil
}

func (e *ejecutorComunicacionLlamamientoDesarrollo) Registrar(ctx context.Context, solicitud ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	vacio := ports.ComunicacionProbatoria{}
	if e == nil || contextoInterfazNulo(ctx) || e.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrServicioComunicacionLlamamientoInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if solicitud.Validar() != nil {
		return vacio, application.ErrSolicitudComunicacionLlamamientoInvalida
	}
	if solicitud.VersionEsperada != 1 {
		return vacio, application.ErrVersionComunicacionLlamamientoEnConflicto
	}
	capacidad, valida := e.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaRegistroComunicacionLlamamiento ||
		solicitud.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, application.ErrComunicacionLlamamientoDenegada
	}
	// Preparación interna: identidad/ámbito antes de leer, V3 antes de cualquier
	// efecto o replay. El lector verifica el agregado propio fiscalizado v6.
	expediente, err := e.lector.LeerExpedienteParaSeleccion(ctx, solicitud.OrganizacionRef, solicitud.ExpedienteRef, 6)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || expediente.Fiscalizado.Validar() != nil ||
		!expedienteComunicacionLlamamientoDesarrolloValido(expediente, solicitud) {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	// No asignar la clave HTTP de comunicación como si fuera la selección.
	// 000054 resuelve la clave real y la relación org/exp/llamamiento/recibo
	// después del consumo V3, en la misma transacción que guarda la comunicación.
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{},
		preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveSolicitudComunicacionLlamamientoDesarrollo{}, solicitud)
	return e.registrarConAviso(ctx, solicitud)
}

// No existe resolución local equivalente a aceptación/renuncia/expiración.
// Se deniega sin lecturas ni efectos. No se utiliza el permiso de registro
// para fingir autorización de una acción diferente.
func (e *ejecutorComunicacionLlamamientoDesarrollo) Resolver(ctx context.Context, solicitud ports.SolicitudResolverLlamamiento) (ports.ResultadoResolucionLlamamiento, error) {
	if contextoInterfazNulo(ctx) || e == nil || e.soporte == nil {
		return ports.ResultadoResolucionLlamamiento{}, application.ErrServicioComunicacionLlamamientoInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoResolucionLlamamiento{}, err
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoResolucionLlamamiento{}, application.ErrSolicitudComunicacionLlamamientoInvalida
	}
	return ports.ResultadoResolucionLlamamiento{}, application.ErrComunicacionLlamamientoDenegada
}

func expedienteComunicacionLlamamientoDesarrolloValido(expediente ports.ExpedienteParaSeleccion, s ports.SolicitudRegistrarComunicacionLlamamiento) bool {
	e := expediente.Fiscalizado
	return e.Referencia == s.ExpedienteRef && e.OrganizacionRef == s.OrganizacionRef &&
		e.OrganizacionRef == organizacionAltaContratacionTemporalDesarrollo && e.Version == 6 &&
		expediente.VersionActual >= 6 && expediente.VersionActual <= ports.MaximoEnteroSeguroIntegracionBolsa &&
		e.FaseActual == domain.FaseFiscalizacion && e.EstadoActual == domain.EstadoEnCurso &&
		e.Fiscalizacion != nil && (e.Fiscalizacion.Resultado == domain.FiscalizacionFavorable ||
		e.Fiscalizacion.Resultado == domain.FiscalizacionFavorableConObservaciones)
}

func (p *proveedorComunicacionLlamamientoDesarrollo) PrepararRegistroComunicacion(ctx context.Context, solicitud ports.SolicitudRegistrarComunicacionLlamamiento) (postgresct.MaterialRegistroComunicacionLlamamiento, error) {
	vacio := postgresct.MaterialRegistroComunicacionLlamamiento{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) || solicitud.Validar() != nil ||
		solicitud.VersionEsperada != 1 {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	capacidad, valida := p.soporte.capacidadValida(ctx)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	ligada, ligadaValida := ctx.Value(claveSolicitudComunicacionLlamamientoDesarrollo{}).(ports.SolicitudRegistrarComunicacionLlamamiento)
	if !valida || capacidad.ruta != httpinterno.RutaRegistroComunicacionLlamamiento ||
		!preparada || !ligadaValida || ligada != solicitud ||
		!expedienteComunicacionLlamamientoDesarrolloValido(preparacion.expediente, solicitud) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return postgresct.MaterialRegistroComunicacionLlamamiento{
		Solicitud: solicitud,
		Canal:     referenciaCatalogoComunicacionDesarrollo("canal:ct:desarrollo:registro-local:v1", canalComunicacionLlamamientoDesarrollo),
		Politica:  referenciaCatalogoComunicacionDesarrollo("politica:ct:desarrollo:registro-local:v1", politicaComunicacionLlamamientoDesarrollo),
	}, nil
}

func (p *proveedorComunicacionLlamamientoDesarrollo) AutorizarRegistroComunicacion(ctx context.Context, material postgresct.MaterialRegistroComunicacionLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	esperado, err := p.PrepararRegistroComunicacion(ctx, material.Solicitud)
	if err != nil {
		return vacio, err
	}
	if esperado != material || dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	recurso, err := postgresct.RecursoRegistroComunicacionLlamamiento(material)
	if err != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	// Sin caché ni excepción de replay: cada invocación requiere decisión fresca
	// y material V3; solo PostgreSQL decide después si existe el mismo efecto.
	autorizacion, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionRegistroComunicacionLlamamiento, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	if autorizacion.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	resumen := autorizacion.ResumenCapacidad()
	ahora := p.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) || ahora.Before(resumen.EmitidaEn()) ||
		!ahora.Before(resumen.ExpiraEn()) || resumen.ExpiraEn().Sub(resumen.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionComunicacionLlamamientoDenegada
	}
	return autorizacion, nil
}

func referenciaCatalogoComunicacionDesarrollo(referencia, contenido string) ports.ReferenciaGobernadaComunicacionLlamamiento {
	huella := sha256.Sum256([]byte(contenido))
	return ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: referencia, Version: 1, HuellaSHA256: hex.EncodeToString(huella[:]),
	}
}

// registrarConAviso no reintenta la operación SQL. Si el commit existe pero el
// fichero falla, el cliente recibe error y debe repetir expresamente la misma
// UUID; el servicio reacreditará V3 y devolverá su recibo durable.
func (e *ejecutorComunicacionLlamamientoDesarrollo) registrarConAviso(ctx context.Context, solicitud ports.SolicitudRegistrarComunicacionLlamamiento) (ports.ComunicacionProbatoria, error) {
	recibo, err := e.servicio.Registrar(ctx, solicitud)
	if err != nil {
		return ports.ComunicacionProbatoria{}, err
	}
	if recibo.ValidarPara(solicitud) != nil || !recibo.EsRegistroLocal() {
		return ports.ComunicacionProbatoria{}, application.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	if err := escribirAvisoComunicacionDesarrollo(ctx, e.directorioComunicaciones, solicitud, recibo); err != nil {
		if ctx.Err() != nil {
			return ports.ComunicacionProbatoria{}, ctx.Err()
		}
		return ports.ComunicacionProbatoria{}, application.ErrComunicacionLlamamientoNoDisponible
	}
	return recibo, nil
}

type avisoComunicacionDesarrollo struct {
	Esquema            string    `json:"esquema"`
	Texto              string    `json:"texto"`
	ClaveIdempotencia  string    `json:"clave_idempotencia"`
	OrganizacionRef    string    `json:"organizacion_ref"`
	ExpedienteRef      string    `json:"expediente_ref"`
	LlamamientoRef     string    `json:"llamamiento_ref"`
	ReciboSeleccionRef string    `json:"recibo_seleccion_ref"`
	ComunicacionRef    string    `json:"comunicacion_ref"`
	IntencionEnvioRef  string    `json:"intencion_envio_ref"`
	ReciboRef          string    `json:"recibo_ref"`
	AuditoriaRef       string    `json:"auditoria_ref"`
	RegistradaEn       time.Time `json:"registrada_en"`
}

func rutaDirectorioComunicacionesValida(directorio string) bool {
	return filepath.IsAbs(directorio) && filepath.Clean(directorio) == directorio &&
		filepath.Dir(directorio) != directorio
}

func escribirAvisoComunicacionDesarrollo(ctx context.Context, directorio string, s ports.SolicitudRegistrarComunicacionLlamamiento, r ports.ComunicacionProbatoria) error {
	if contextoInterfazNulo(ctx) || !rutaDirectorioComunicacionesValida(directorio) ||
		r.ValidarPara(s) != nil || !r.EsRegistroLocal() {
		return application.ErrComunicacionLlamamientoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	contenido, err := json.Marshal(avisoComunicacionDesarrollo{
		Esquema:           "vec.ct.aviso-desarrollo.v1",
		Texto:             "Aviso de desarrollo, no enviado, no abre plazo",
		ClaveIdempotencia: s.ClaveIdempotencia, OrganizacionRef: s.OrganizacionRef,
		ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef,
		ReciboSeleccionRef: s.PruebaEntregaRef, ComunicacionRef: r.ComunicacionRef,
		IntencionEnvioRef: r.IntencionEnvioRef, ReciboRef: r.ReciboRef,
		AuditoriaRef: r.AuditoriaRef, RegistradaEn: r.RegistradaEn,
	})
	if err != nil {
		return err
	}
	raiz, dir, err := abrirDirectorioAvisosDesarrollo(directorio)
	if err != nil {
		return err
	}
	defer raiz.Close()
	defer dir.Close()
	huella := sha256.Sum256([]byte(r.IntencionEnvioRef))
	nombre := "aviso-" + hex.EncodeToString(huella[:]) + ".json"
	if existe, err := comprobarAvisoComunicacionDesarrollo(raiz, nombre, contenido); err != nil {
		return err
	} else if existe {
		if err := dir.Sync(); err != nil {
			return err
		}
		return ctx.Err()
	}
	temporal := ".aviso-" + rand.Text() + ".tmp"
	fichero, err := raiz.OpenFile(temporal, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer fichero.Close()
	defer raiz.Remove(temporal)
	if n, err := fichero.Write(contenido); err != nil {
		return err
	} else if n != len(contenido) {
		return io.ErrShortWrite
	}
	if err := fichero.Sync(); err != nil {
		return err
	}
	if err := fichero.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := raiz.Link(temporal, nombre); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	// Link publica sin reemplazar. Desde aquí nunca se retira el final,
	// tampoco si el contexto se cancela o una sincronización posterior falla.
	if err := raiz.Remove(temporal); err != nil {
		return err
	}
	if existe, err := comprobarAvisoComunicacionDesarrollo(raiz, nombre, contenido); err != nil {
		return err
	} else if !existe {
		return application.ErrComunicacionLlamamientoNoDisponible
	}
	if err := dir.Sync(); err != nil {
		return err
	}
	return ctx.Err()
}

// Directorios abiertos por descriptor y comparados con Lstat: un cambio por
// enlace entre inspección y apertura no redirige la publicación a otra raíz.
func abrirDirectorioAvisosDesarrollo(directorio string) (*os.Root, *os.File, error) {
	fallo := application.ErrComunicacionLlamamientoNoDisponible
	padreRuta := filepath.Dir(directorio)
	padreInfo, err := os.Lstat(padreRuta)
	if err != nil || !padreInfo.IsDir() || padreInfo.Mode().Perm() != 0700 {
		return nil, nil, fallo
	}
	padre, err := os.OpenRoot(padreRuta)
	if err != nil {
		return nil, nil, fallo
	}
	defer padre.Close()
	padreFichero, err := padre.Open(".")
	if err != nil {
		return nil, nil, fallo
	}
	defer padreFichero.Close()
	abierto, err := padreFichero.Stat()
	if err != nil || !os.SameFile(padreInfo, abierto) {
		return nil, nil, fallo
	}
	nombre := filepath.Base(directorio)
	if err := padre.Mkdir(nombre, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, nil, fallo
	}
	info, err := padre.Lstat(nombre)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, nil, fallo
	}
	raiz, err := padre.OpenRoot(nombre)
	if err != nil {
		return nil, nil, fallo
	}
	dir, err := raiz.Open(".")
	if err != nil {
		raiz.Close()
		return nil, nil, fallo
	}
	actual, err := dir.Stat()
	if err != nil || !os.SameFile(info, actual) || padreFichero.Sync() != nil {
		dir.Close()
		raiz.Close()
		return nil, nil, fallo
	}
	return raiz, dir, nil
}

func comprobarAvisoComunicacionDesarrollo(raiz *os.Root, nombre string, esperado []byte) (bool, error) {
	info, err := raiz.Lstat(nombre)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	fallo := application.ErrComunicacionLlamamientoNoDisponible
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() != int64(len(esperado)) {
		return true, fallo
	}
	fichero, err := raiz.OpenFile(nombre, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return true, fallo
	}
	defer fichero.Close()
	actual, err := fichero.Stat()
	if err != nil || !os.SameFile(info, actual) || !actual.Mode().IsRegular() ||
		actual.Mode().Perm() != 0600 || actual.Size() != int64(len(esperado)) {
		return true, fallo
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, int64(len(esperado))+1))
	if err != nil || !bytes.Equal(contenido, esperado) {
		return true, fallo
	}
	return true, fichero.Sync()
}
