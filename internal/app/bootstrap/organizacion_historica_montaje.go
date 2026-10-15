package bootstrap

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errMontajeOrganizacionHistoricaCombinada = errors.New("bootstrap: consulta de organizacion historica no disponible")

type configuracionOrganizacionHistoricaCombinada struct {
	Version                 int                            `json:"version"`
	DSNPersonal             string                         `json:"dsn_personal"`
	DSNFuenteAutorizacion   string                         `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion string                         `json:"dsn_registro_autorizacion"`
	DSNMotivos              string                         `json:"dsn_motivos"`
	MotivoDenegado          core.ReferenciaEntradaCatalogo `json:"motivo_denegado"`
	MotivoEntradaInvalida   core.ReferenciaEntradaCatalogo `json:"motivo_entrada_invalida"`
	MotivoNoDisponible      core.ReferenciaEntradaCatalogo `json:"motivo_no_disponible"`
}

type montajeOrganizacionHistoricaCombinada struct {
	manejador http.Handler
	fronteras catalogoFronterasComunDesarrollo
	fuente    *fuenteOrganizacionHistoricaCombinada
	barrera   *barreraPeticionesOrganizacionHistorica
	cerrar    func()
}

// Shutdown invoca callbacks mientras todavía puede haber GET activos.
// La barrera se toma antes de la autoridad exacta y drena antes del cierre.
type barreraPeticionesOrganizacionHistorica struct {
	mu       sync.Mutex
	cerrando bool
	activas  sync.WaitGroup
}

func (b *barreraPeticionesOrganizacionHistorica) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != httpapi.RutaOrganizacionHistoricaPersonal {
			siguiente.ServeHTTP(w, r)
			return
		}
		b.mu.Lock()
		if b.cerrando {
			b.mu.Unlock()
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b.activas.Add(1)
		b.mu.Unlock()
		defer b.activas.Done()
		siguiente.ServeHTTP(w, r)
	})
}

func (b *barreraPeticionesOrganizacionHistorica) detener() {
	b.mu.Lock()
	b.cerrando = true
	b.mu.Unlock()
	b.activas.Wait()
}

func cierreOrganizacionHistoricaCombinada(b *barreraPeticionesOrganizacionHistorica, cerrarRecursos func()) func() {
	var unaVez sync.Once
	return func() {
		unaVez.Do(func() {
			b.detener()
			cerrarRecursos()
		})
	}
}

func cargarConfiguracionOrganizacionHistoricaCombinada(dir string) (configuracionOrganizacionHistoricaCombinada, error) {
	var c configuracionOrganizacionHistoricaCombinada
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || dentroDeRepositorioGit(dir) {
		return c, errMontajeOrganizacionHistoricaCombinada
	}
	b, err := leerFicheroMaterialSeguro(filepath.Join(dir, "organizacion_historica_combinado.json"), 64<<10)
	if err != nil {
		return c, errMontajeOrganizacionHistoricaCombinada
	}
	defer borrarBytes(b)
	if validarClavesJSONUnicas(b) != nil {
		return c, errMontajeOrganizacionHistoricaCombinada
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || c.Version != 1 || c.DSNPersonal == "" ||
		c.DSNFuenteAutorizacion == "" || c.DSNRegistroAutorizacion == "" || c.DSNMotivos == "" ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoDenegado) || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoEntradaInvalida) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoNoDisponible) {
		return configuracionOrganizacionHistoricaCombinada{}, errMontajeOrganizacionHistoricaCombinada
	}
	return c, nil
}

func cotejarCapacidadOrganizacionHistoricaCombinada(m internactproveedores.MaterialOrganizacionHistorica, publicada CapacidadPublicadaOrganizacionHistoricaV3, dir, materialDesarrollo string, ahora time.Time) error {
	if publicada.Audiencia != personal.AudienciaConsultaOrganizacionHistorica ||
		(m.Capacidad.ClaveID != publicada.ClaveID || m.Capacidad.Version != publicada.Version || m.Capacidad.EmisorID != publicada.EmisorID ||
			m.Capacidad.RevisionGobierno != publicada.RevisionGobierno || m.Capacidad.HuellaGobierno != publicada.HuellaGobierno ||
			m.Capacidad.SHA256 != publicada.SHA256 || !m.Capacidad.Desde.Equal(publicada.Desde) || !m.Capacidad.Hasta.Equal(publicada.Hasta)) {
		return errMontajeOrganizacionHistoricaCombinada
	}
	derivada, err := DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(filepath.Join(materialDesarrollo, "idempotencia"), ahora)
	if err != nil {
		return errMontajeOrganizacionHistoricaCombinada
	}
	defer derivada.Borrar()
	if derivada.Audiencia != publicada.Audiencia || derivada.ClaveID != publicada.ClaveID || derivada.HuellaGobierno != publicada.HuellaGobierno || derivada.SHA256 != publicada.SHA256 ||
		derivada.EmisorID != publicada.EmisorID || !derivada.Desde.Equal(publicada.Desde) || !derivada.Hasta.Equal(publicada.Hasta) {
		return errMontajeOrganizacionHistoricaCombinada
	}
	clave, err := leerFicheroMaterialSeguro(filepath.Join(dir, m.Capacidad.Archivo), sha256.Size)
	if err != nil {
		return errMontajeOrganizacionHistoricaCombinada
	}
	defer borrarBytes(clave)
	esperada := derivada.CopiarSecreto()
	defer borrarBytes(esperada)
	huella := sha256.Sum256(clave)
	if len(clave) != sha256.Size || !hmac.Equal(clave, esperada) || hex.EncodeToString(huella[:]) != publicada.SHA256 {
		return errMontajeOrganizacionHistoricaCombinada
	}
	return nil
}

// La ruta de indisponibilidad pertenece sólo al combinado interno y no entra
// en el despachador público ni en el catálogo de capacidades.
func protegerOrganizacionHistoricaNoDisponible(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && r.URL.Path == httpapi.RutaOrganizacionHistoricaPersonal {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}

func (m *montajeOrganizacionHistoricaCombinada) proteger(a *autoridadConsultasContratacionTemporalDesarrollo, siguiente http.Handler) http.Handler {
	if m == nil || a == nil || m.fuente == nil || m.barrera == nil || siguiente == nil {
		return protegerOrganizacionHistoricaNoDisponible(siguiente)
	}
	return m.barrera.proteger(a.protegerConCatalogoSeguridadComun(siguiente, m.fronteras))
}

func capturarOrganizacionHistoricaCombinada(f *fuenteOrganizacionHistoricaCombinada, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil {
			capturado, err := f.vincular(r.Context())
			if capturado == nil {
				capturado = r.Context()
			}
			if err != nil {
				capturado = context.WithValue(capturado, claveFalloOrganizacionHistoricaCombinada{}, err)
			}
			r = r.WithContext(capturado)
		}
		h.ServeHTTP(w, r)
	})
}

func acreditarPoolOrganizacionHistoricaCombinada(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil || ctx == nil || ctx.Err() != nil {
		return errMontajeOrganizacionHistoricaCombinada
	}
	sonda, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(5*time.Second))
	defer cancelar()
	const sql = `SELECT pg_catalog.has_function_privilege(session_user,'vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') AND count(*)=9 AND NOT COALESCE(bool_or(pg_catalog.has_table_privilege(session_user,c.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')),true) FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_personal' AND c.relname=ANY($1::text[]) AND c.relkind IN ('r','p')`
	tablas := []string{"org_nodo_historia", "version_rpt_historia", "version_plantilla_historia", "puesto_tipo_historia", "dotacion_rpt_historia", "plaza_plantilla_historia", "puesto_rpt_historia", "vinculo_plaza_puesto_historia", "recibo_consulta_organizacion"}
	var permitido bool
	if pool.QueryRow(sonda, sql, tablas).Scan(&permitido) != nil || !permitido {
		return errMontajeOrganizacionHistoricaCombinada
	}
	return nil
}

func nuevoMontajeOrganizacionHistoricaCombinada(ctx context.Context, cfg config.Config, a *autoridadConsultasContratacionTemporalDesarrollo, registrador vecports.RegistradorAuditoriaFronteraRutaExacta) (_ *montajeOrganizacionHistoricaCombinada, err error) {
	activo, err := cfg.OrganizacionHistoricaConsultaDesarrolloActivo()
	if err != nil || !activo || a == nil || a.sesionOrganizacionHistorica == nil || a.proveedorMaterialOrganizacionHistorica == nil ||
		a.materialOrganizacionHistorica.ClaveID == "" || registrador == nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	if a.sesionOrganizacionHistorica.falloVinculoCorporativo != nil {
		return nil, errors.Join(errMontajeOrganizacionHistoricaCombinada, a.sesionOrganizacionHistorica.falloVinculoCorporativo)
	}
	if a.sesionOrganizacionHistorica.vinculoCorporativo == nil {
		return nil, errors.Join(errMontajeOrganizacionHistoricaCombinada, vecports.ErrVinculoCorporativoRRHHNoDisponible)
	}
	dir := cfg.OrganizacionHistoricaMaterialDir
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(dir)
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	defer m.Cerrar()
	c, err := cargarConfiguracionOrganizacionHistoricaCombinada(dir)
	if err != nil ||
		cotejarCapacidadOrganizacionHistoricaCombinada(m, a.materialOrganizacionHistorica, dir, cfg.DevelopmentMaterialDir, time.Now().UTC()) != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	pool, login, err := abrirPoolRutasDietas(ctx, c.DSNPersonal, "vec_personal_ejecutor")
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	pools := []*pgxpool.Pool{pool}
	logins := []string{login}
	cerrarPool := true
	defer func() {
		if cerrarPool {
			for _, abierto := range pools {
				abierto.Close()
			}
		}
	}()
	if acreditarPoolOrganizacionHistoricaCombinada(ctx, pool) != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	for _, entrada := range []struct{ dsn, rol string }{
		{c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"},
		{c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
	} {
		abierto, usuario, errorPool := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if errorPool != nil {
			return nil, errMontajeOrganizacionHistoricaCombinada
		}
		pools = append(pools, abierto)
		for _, previo := range logins {
			if usuario == previo {
				return nil, errMontajeOrganizacionHistoricaCombinada
			}
		}
		logins = append(logins, usuario)
	}
	registro, proceso, cerrarIntentos, err := AbrirRegistradorIntentosAuditoriaDesarrollo(ctx, cfg, pool, logins)
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	cerrarRegistro := true
	defer func() {
		if cerrarRegistro {
			cerrarIntentos()
		}
	}()
	destino, ok := registro.(internactproveedores.RegistradorIntentosOrganizacionHistorica)
	if !ok {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	intentos, err := internactproveedores.NuevoRegistroIntentosOrganizacionHistorica(internactproveedores.DependenciasIntentosOrganizacionHistorica{Registrador: destino,
		Configuracion: internactproveedores.ConfiguracionIntentosOrganizacionHistorica{Proceso: proceso, Canal: string(core.SuperficieAutenticacionInternaCorporativaV1), MotivoDenegado: c.MotivoDenegado, MotivoEntradaInvalida: c.MotivoEntradaInvalida, MotivoNoDisponible: c.MotivoNoDisponible}})
	if err != nil || intentos.PreflightIntentosOrganizacionHistorica(ctx) != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	perfiles := make([]string, 0, len(m.Contextos))
	vistos := map[string]bool{}
	for _, ambito := range m.Contextos {
		if !vistos[ambito.PerfilActivoRef] {
			vistos[ambito.PerfilActivoRef] = true
			perfiles = append(perfiles, ambito.PerfilActivoRef)
		}
	}
	sort.Strings(perfiles)
	fronteras, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{{Clave: "organizacion-historica-consulta", Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: httpapi.RutaOrganizacionHistoricaPersonal,
		PerfilesActivosRef: perfiles, ClavePolitica: "organizacion-historica-consulta", ClaveCapacidad: a.materialOrganizacionHistorica.ClaveID}})
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	sesion := *a.sesionOrganizacionHistorica
	sesion.fronteras = fronteras
	sesion.organizacionHistoricaPorCuenta = make(map[string]seleccionPerfilOrganizacionHistorica, len(m.Contextos))
	for cuenta, ambito := range m.Contextos {
		sesion.organizacionHistoricaPorCuenta[cuenta] = seleccionPerfilOrganizacionHistorica{perfil: ambito.PerfilActivoRef, version: ambito.PerfilVersion}
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(&sesion, a.sesionOrganizacionHistorica.reloj)
	if err != nil || sesion.vinculoCorporativo == nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	f := &fuenteOrganizacionHistoricaCombinada{sesion: seguridad, corporativo: sesion.vinculoCorporativo, reloj: sesion.reloj, contextos: m.Contextos, fronteras: fronteras}
	fuenteAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[1])
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[2])
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	validadorMotivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[3], m.CatalogoMotivos)
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuenteAutorizacion, registroAutorizacion, registroAutorizacion,
		validadorMotivos, sesion.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 5 * time.Second})
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, a.proveedorMaterialOrganizacionHistorica)
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	proveedor, err := internactproveedores.ConstruirOrganizacionHistoricaConEmisor(ctx, m, f, emisor, sesion.reloj)
	if err != nil {
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	repositorio, err := personalpg.NuevoRepositorioOrganizacionHistoricaPostgreSQL(pool)
	if err != nil {
		proveedor.Cerrar()
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	servicio, err := personalapp.NuevoServicioConsultaOrganizacionHistorica(proveedor, repositorio, intentos)
	if err != nil {
		proveedor.Cerrar()
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	consulta, err := internactproveedores.NuevaConsultaOrganizacionHistoricaConIntentos(servicio, f, intentos)
	if err != nil {
		proveedor.Cerrar()
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	h, err := httpapi.NewHandlerOrganizacionHistoricaPersonal(internactproveedores.AutoridadContextoOrganizacionHistorica{Fuente: f}, consulta,
		internactproveedores.AuditorDenegacionOrganizacionHistorica{Registrador: registrador})
	if err != nil {
		proveedor.Cerrar()
		return nil, errMontajeOrganizacionHistoricaCombinada
	}
	cerrarPool, cerrarRegistro = false, false
	barrera := &barreraPeticionesOrganizacionHistorica{}
	return &montajeOrganizacionHistoricaCombinada{manejador: capturarOrganizacionHistoricaCombinada(f, h), fronteras: fronteras, fuente: f, barrera: barrera,
		cerrar: cierreOrganizacionHistoricaCombinada(barrera, func() {
			proveedor.Cerrar()
			cerrarIntentos()
			for _, abierto := range pools {
				abierto.Close()
			}
		})}, nil
}
