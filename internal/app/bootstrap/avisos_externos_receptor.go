package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	usuarioscanonico "vec-diputacion-granada/internal/modules/usuarios/canonico"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const envOutboxAvisosExternos = "VEC_EXTERNO_AVISOS_BOLSA_DATABASE_URL"
const envInboxAvisosExternos = "VEC_EXTERNO_AVISOS_USUARIOS_DATABASE_URL"
const envAvisosExternos = "VEC_EXTERNO_AVISOS_ENABLED"

type catalogoAvisosExternos struct {
	bolsa dominiobolsa.CatalogoCorreoLlamamiento
}

func (c catalogoAvisosExternos) AdmitePlantillaAvisoExterno(ctx context.Context, ref, version, tipo string) bool {
	if ctx == nil || ctx.Err() != nil || ref != version || tipo != usuariosports.TipoAvisoLlamamientoExternoV1 {
		return false
	}
	_, ok := c.bolsa.PlantillaAdmitida(version)
	return ok
}

type transporteAvisosExternos struct {
	smtp                    enviadorCorreoLlamamientoDesarrollo
	asunto, cuerpo, dominio string
	ahora                   func() time.Time
}

func (t *transporteAvisosExternos) EnviarAvisoExterno(ctx context.Context, m usuariosports.MensajeAvisoExterno) bool {
	if t == nil || ctx == nil || t.smtp == nil || m.Destino == "" || !strings.HasPrefix(m.EnvioRef, "aviso_recibo:") {
		return false
	}
	resultado := t.smtp.Enviar(ctx, smtpct.Mensaje{Destino: m.Destino, Asunto: t.asunto, Cuerpo: t.cuerpo,
		MessageID: "<" + strings.ReplaceAll(m.EnvioRef, ":", "-") + "@" + t.dominio + ">", FechaOrigen: t.ahora().UTC()})
	return resultado.Estado == smtpct.AceptadoPorRelay
}
func nuevoTransporteAvisosExternos(cfg config.Config, c configuracionAvisosExternos) (*transporteAvisosExternos, error) {
	u, err := url.Parse(c.URLPersonal)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errAvisosExternos
	}
	dominio := dominioRemitente(cfg.SMTPFrom)
	if dominio == "" {
		return nil, errAvisosExternos
	}
	f, err := os.Open(filepath.Join("web", "static", "textos", c.Idioma, "avisos-externos.json"))
	if err != nil {
		return nil, errAvisosExternos
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 16<<10+1))
	if err != nil || len(raw) > 16<<10 {
		return nil, errAvisosExternos
	}
	var textos map[string]string
	if json.Unmarshal(raw, &textos) != nil {
		return nil, errAvisosExternos
	}
	asunto, cuerpo := textos["usuarios.avisos_externos.asunto"], textos["usuarios.avisos_externos.cuerpo"]
	if strings.TrimSpace(asunto) == "" || !strings.Contains(cuerpo, "{portal}") || strings.ContainsAny(asunto, "\r\n") {
		return nil, errAvisosExternos
	}
	smtp, err := nuevoEnviadorCorreoLlamamientoDesarrollo(cfg)
	if err != nil || smtp == nil {
		return nil, errAvisosExternos
	}
	return &transporteAvisosExternos{smtp: smtp, asunto: asunto, cuerpo: strings.ReplaceAll(cuerpo, "{portal}", c.URLPersonal), dominio: dominio, ahora: time.Now}, nil
}

func abrirPoolOutboxAvisosExternos(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errAvisosExternos
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User != "vec_externo_avisos_bolsa" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil || len(c.ConnConfig.Fallbacks) != 0 {
		return nil, errAvisosExternos
	}
	c.MaxConns = 2
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"search_path": "pg_catalog", "timezone": "UTC", "statement_timeout": "10s", "lock_timeout": "2s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, errAvisosExternos
	}
	var nominal bool
	if err := pool.QueryRow(ctx, `SELECT session_user=current_user AND session_user='vec_externo_avisos_bolsa'
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_stat_ssl WHERE pid=pg_backend_pid() AND ssl)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin AND r.rolinherit
   AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
   AND (r.rolvaliduntil IS NULL OR r.rolvaliduntil>clock_timestamp())
   AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members WHERE member=r.oid)
   AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
     WHERE m.member=r.oid AND g.rolname='vec_bolsa_avisos_externos_consumidor' AND m.inherit_option AND NOT(m.set_option OR m.admin_option)
     AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
     AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid)))
 AND COALESCE(pg_catalog.has_function_privilege(session_user,to_regprocedure('vec_bolsa_llamamientos.tirar_avisos_externos_v1(integer)'),'EXECUTE'),false)
 AND COALESCE(pg_catalog.has_function_privilege(session_user,to_regprocedure('vec_bolsa_llamamientos.confirmar_aceptacion_aviso_externo_v1(text,text,text,text)'),'EXECUTE'),false)
 AND COALESCE(pg_catalog.has_function_privilege(session_user,to_regprocedure('vec_bolsa_llamamientos.registrar_resultado_aviso_externo_v1(text,text,text,text,text)'),'EXECUTE'),false)`).Scan(&nominal); err != nil || !nominal {
		pool.Close()
		return nil, errAvisosExternos
	}
	return pool, nil
}

// El receptor acepta primero en Usuarios, intenta el despacho con su reserva
// exclusiva y confirma a Bolsa sólo el recibo de aceptación. Una interrupción
// deja el evento recuperable; repetir el lote no vuelve a despachar la reserva.
func procesarLoteAvisosExternos(ctx context.Context, r bolsaports.RepositorioAvisosExternos, s *usuariosapp.ServicioAvisosExternos, c configuracionAvisosExternos) error {
	eventos, err := r.Extraer(ctx, c.Lote)
	if err != nil {
		return err
	}
	for _, pendiente := range eventos {
		raw, err := json.Marshal(pendiente.Evento)
		if err != nil {
			return errAvisosExternos
		}
		var e usuariosports.EventoAvisoExterno
		if json.Unmarshal(raw, &e) != nil {
			return errAvisosExternos
		}
		huella, err := usuarioscanonico.HuellaAvisoExterno(e)
		if err != nil || huella != pendiente.Huella {
			return errAvisosExternos
		}
		recibo, err := s.Aceptar(ctx, e)
		if err != nil {
			return err
		}
		despacho, err := s.Despachar(ctx, recibo.ReciboRef)
		if err != nil {
			return err
		}
		estado := despacho.Estado
		if estado == "reservado" {
			estado = "reservado_incierto"
		}
		if err = r.RegistrarResultadoDespacho(ctx, e.ProductorRef, e.EventoRef, recibo.Huella, recibo.ReciboRef, estado); err != nil {
			return err
		}
		if err = r.ConfirmarAceptacion(ctx, e.ProductorRef, e.EventoRef, recibo.Huella, recibo.ReciboRef); err != nil {
			return err
		}
	}
	return nil
}

// componerBuzonAvisosExternos recibe exclusivamente el protector externo ya
// compuesto. Sus dos pools tienen autoridades nominales distintas; ningún
// secreto ni conexión del proceso interno llega a este trabajador.
func componerBuzonAvisosExternos(ctx context.Context, cfg config.Config, cripto usuariosports.ProtectorDireccionCorreo, preflight *pgxpool.Pool, incidencias vecports.EmisorIncidenciasTecnicas) (func(), error) {
	vacio := func() {}
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envAvisosExternos)
	if err != nil || !activo {
		return vacio, err
	}
	if cfg.PortalProceso != string(separacionportales.PortalExterno) || ctx == nil || cripto == nil || preflight == nil || incidencias == nil {
		return vacio, errAvisosExternos
	}
	c, err := leerConfiguracionAvisosExternos(cfg, "usuarios/avisos-externos.json")
	if err != nil {
		return vacio, err
	}
	intervalo, err := c.intervaloValido()
	if err != nil {
		return vacio, err
	}
	catalogo, err := cargarCatalogoCorreoLlamamientoBolsa()
	if err != nil {
		return vacio, errAvisosExternos
	}
	transporte, err := nuevoTransporteAvisosExternos(cfg, c)
	if err != nil {
		return vacio, err
	}
	fuente, err := abrirPoolOutboxAvisosExternos(ctx, os.Getenv(envOutboxAvisosExternos))
	if err != nil {
		return vacio, err
	}
	cerrarFuente := func() { fuente.Close() }
	dsnInbox := os.Getenv(envInboxAvisosExternos)
	conexionInbox, err := pgxpool.ParseConfig(dsnInbox)
	if err != nil || dsnInbox == "" || conexionInbox.ConnConfig.User != "vec_externo_avisos_usuarios" {
		cerrarFuente()
		return vacio, errAvisosExternos
	}
	receptor, _, err := abrirPoolUsuariosPreferencias(ctx, dsnInbox, rolEjecutorPreferencias(string(core.SuperficieAutenticacionExternaPersonalV1)))
	if err != nil {
		cerrarFuente()
		return vacio, errAvisosExternos
	}
	cerrarPools := func() { receptor.Close(); cerrarFuente() }
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, preflight)
	if err != nil || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, fuente, topologia) != nil || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, receptor, topologia) != nil {
		cerrarPools()
		return vacio, errAvisosExternos
	}
	outbox, err := bolsapg.NuevoRepositorioAvisosExternosPostgreSQL(fuente)
	if err != nil {
		cerrarPools()
		return vacio, errAvisosExternos
	}
	inbox, err := usuariospg.NuevoRegistroAvisosExternosPostgreSQL(ctx, receptor)
	if err != nil {
		cerrarPools()
		return vacio, errAvisosExternos
	}
	servicio, err := usuariosapp.NuevoServicioAvisosExternos(inbox, cripto, transporte, catalogoAvisosExternos{catalogo}, c.ProductorRef)
	if err != nil {
		cerrarPools()
		return vacio, errAvisosExternos
	}
	trabajo, cancelar := context.WithCancel(context.Background())
	terminado := make(chan struct{})
	go func() {
		defer close(terminado)
		timer := time.NewTicker(intervalo)
		defer timer.Stop()
		for {
			select {
			case <-trabajo.Done():
				return
			case <-timer.C:
				lote, cancel := context.WithTimeout(trabajo, intervalo)
				if err := procesarLoteAvisosExternos(lote, outbox, servicio, c); err != nil {
					incidencias.Emitir(core.SolicitudIncidenciaTecnica{Codigo: core.IncidenciaPostgresNoDisponible, Componente: core.ComponenteIncidenciaPostgreSQL, Etapa: core.EtapaIncidenciaConsulta})
				}
				cancel()
			}
		}
	}()
	return func() { cancelar(); <-terminado; cerrarPools() }, nil
}
