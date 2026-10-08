package bootstrap

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const politicaSesionFirmanteV2Sintetica = "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa;retirada-firma-vec"

// Sólo el adaptador opt-in de desarrollo interpreta este fichero. La fuente
// corporativa inyecta su evidencia ya autenticada y registrada por otro canal.
type configuracionSesionFirmanteV2 struct {
	Version                  int    `json:"version"`
	Autoridad                string `json:"autoridad"`
	DSNRegistroIdentidad     string `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string `json:"dsn_revalidacion_identidad"`
	DSNContexto              string `json:"dsn_contexto"`
	RetiradaEn               string `json:"retirada_en"`
}

func (configuracionSesionFirmanteV2) String() string   { return "[CONFIGURACION PRIVADA FIRMA VEC]" }
func (configuracionSesionFirmanteV2) GoString() string { return "[CONFIGURACION PRIVADA FIRMA VEC]" }

func decodificarConfiguracionSesionFirmanteV2(contenido []byte) (configuracionSesionFirmanteV2, time.Time, error) {
	var c configuracionSesionFirmanteV2
	if validarClavesJSONUnicas(contenido) != nil {
		return c, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		c.DSNRegistroIdentidad == "" || c.DSNRevalidacionIdentidad == "" || c.DSNContexto == "" {
		return configuracionSesionFirmanteV2{}, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	retirada, err := time.Parse(time.RFC3339, c.RetiradaEn)
	if err != nil || retirada.Location() != time.UTC || retirada.Format(time.RFC3339) != c.RetiradaEn {
		return configuracionSesionFirmanteV2{}, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	return c, retirada, nil
}

type fuenteSinteticaSesionFirmanteV2 struct {
	resolvedor  *resolvedorIdentidadDesarrollo
	registro    httpseguridad.RegistroSesiones
	revalidador core.RevalidadorAutenticacionActorV1
	contextos   core.ResolutorContextoActorRegistradoV2
	reloj       vp.Reloj
	retirada    time.Time
	instancia   string
}

var _ ports.FuenteSesionFirmanteV2 = (*fuenteSinteticaSesionFirmanteV2)(nil)

func nuevaAutoridadSesionFirmanteV2ConPuertos(resolvedor *resolvedorIdentidadDesarrollo,
	registro httpseguridad.RegistroSesiones, revalidador core.RevalidadorAutenticacionActorV1,
	contextos core.ResolutorContextoActorRegistradoV2, reloj vp.Reloj, retirada time.Time, instancia string,
) (*autoridadSesionFirmanteV2, error) {
	if resolvedor == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(revalidador) || dependenciaEsNulaContratacionTemporalDesarrollo(contextos) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) || retirada.IsZero() || !retirada.After(reloj.Ahora()) ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(instancia) {
		return nil, errSesionFirmanteV2NoDisponible
	}
	f := &fuenteSinteticaSesionFirmanteV2{resolvedor: resolvedor, registro: registro, revalidador: revalidador,
		contextos: contextos, reloj: reloj, retirada: retirada, instancia: instancia}
	return nuevaAutoridadSesionFirmanteV2ConFuente(f, reloj, f.preacreditar)
}

func (f *fuenteSinteticaSesionFirmanteV2) preacreditar(r *http.Request) error {
	if f == nil || r == nil || f.resolvedor == nil || f.reloj == nil || !f.reloj.Ahora().Before(f.retirada) {
		return errSesionFirmanteV2Denegada
	}
	p, err := f.resolvedor.ResolveDemoIdentity(r.Context(), r)
	if err != nil || !principalSinteticoContratacionTemporalDesarrolloValido(p) {
		return errSesionFirmanteV2Denegada
	}
	return nil
}

func (f *fuenteSinteticaSesionFirmanteV2) AbrirSesionFirmanteV2(ctx context.Context,
	q ports.SolicitudSesionFirmanteV2,
) (ports.SesionFirmanteV2, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || f.resolvedor == nil || f.reloj == nil ||
		!f.reloj.Ahora().Before(f.retirada) || !huellaSHA256ValidaContratacionTemporalDesarrollo(q.CertificadoCanalSHA256) ||
		q.CuentaEsperadaRef == "" || q.PerfilEsperadoRef == "" || q.PersonaEsperadaRef == "" {
		return nil, errSesionFirmanteV2Denegada
	}
	b, err := hex.DecodeString(q.CertificadoCanalSHA256)
	if err != nil || len(b) != 32 {
		return nil, errSesionFirmanteV2Denegada
	}
	var digest [32]byte
	copy(digest[:], b)
	principal, ok := f.resolvedor.porHuella[digest]
	if !ok || !principalSinteticoContratacionTemporalDesarrolloValido(principal) ||
		principal.Attributes["certificate_sha256"] != q.CertificadoCanalSHA256 {
		return nil, errSesionFirmanteV2Denegada
	}
	ahora := f.reloj.Ahora()
	if q.CertificadoVerificadoEn.IsZero() || q.CertificadoVerificadoEn.After(ahora) ||
		ahora.Sub(q.CertificadoVerificadoEn) > 2*time.Minute {
		return nil, errSesionFirmanteV2Denegada
	}
	hasta := ahora.Add(2 * time.Minute)
	if q.CertificadoTLSValidoHasta.Before(hasta) {
		hasta = q.CertificadoTLSValidoHasta
	}
	if f.retirada.Before(hasta) {
		hasta = f.retirada
	}
	if !hasta.After(ahora) {
		return nil, errSesionFirmanteV2Denegada
	}
	asercion, err := nonceRutasDietas()
	if err != nil {
		return nil, errSesionFirmanteV2NoDisponible
	}
	sesion, err := nonceRutasDietas()
	if err != nil {
		return nil, errSesionFirmanteV2NoDisponible
	}
	alta := httpseguridad.AltaSesionAtomica{AsercionID: asercion, SesionID: sesion,
		SujetoID: principal.ID, CuentaID: "desarrollo:" + q.CuentaEsperadaRef,
		Superficie: httpseguridad.SuperficieInternaCorporativa, EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh,
		AutenticacionVerificadaEn: q.CertificadoVerificadoEn, SesionEmitidaEn: ahora, AsercionExpiraEn: hasta,
		PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"),
		PoliticaGarantiaHuellaSHA256: huellaRutasDietas(politicaSesionFirmanteV2Sintetica),
		AutenticacionHuellaSHA256: huellaRutasDietas(f.instancia + "|" + asercion + "|" + sesion + "|" + q.CertificadoCanalSHA256 + "|" +
			q.CuentaEsperadaRef + "|" + q.PerfilEsperadoRef + "|" + q.PersonaEsperadaRef + "|" + httpinterno.RutaRegistroFirmaVec),
	}
	confirmacion, err := f.registro.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != q.CuentaEsperadaRef {
		return nil, errSesionFirmanteV2NoDisponible
	}
	return &sesionSinteticaFirmanteV2{fuente: f, solicitud: q, alta: alta, confirmacion: confirmacion}, nil
}

type sesionSinteticaFirmanteV2 struct {
	fuente       *fuenteSinteticaSesionFirmanteV2
	solicitud    ports.SolicitudSesionFirmanteV2
	alta         httpseguridad.AltaSesionAtomica
	confirmacion httpseguridad.ConfirmacionAltaSesion
}

func (s *sesionSinteticaFirmanteV2) RevalidarSesionFirmanteV2(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	var cero ports.EvidenciaSesionFirmanteV2
	if s == nil || s.fuente == nil || ctx == nil || ctx.Err() != nil ||
		!s.fuente.reloj.Ahora().Before(s.fuente.retirada) {
		return cero, errSesionFirmanteV2Denegada
	}
	f := s.fuente
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: f.revalidador,
		alta: s.alta, confirmacion: s.confirmacion, reloj: f.reloj, superficie: httpseguridad.SuperficieInternaCorporativa}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: s.confirmacion.AutenticacionRef,
			SesionRef: s.confirmacion.SesionRef}, f.contextos,
		core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: s.solicitud.CuentaEsperadaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh},
			PerfilActivoRef: s.solicitud.PerfilEsperadoRef}, f.reloj)
	if err != nil {
		return cero, errSesionFirmanteV2Denegada
	}
	hasta := s.solicitud.CertificadoTLSValidoHasta
	if f.retirada.Before(hasta) {
		hasta = f.retirada
	}
	return ports.EvidenciaSesionFirmanteV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: s.solicitud.CertificadoCanalSHA256, CertificadoValidoHasta: hasta}, nil
}

// Sólo este constructor selecciona el adaptador sintético. Ausencia: ruta no
// compuesta; presencia inválida o sin doble llave: arranque rechazado.
func nuevaAutoridadSesionFirmanteV2(cfg config.Config, resolvedor httpapi.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
) (*autoridadSesionFirmanteV2, func(), error) {
	vacio := func() {}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "firma-vec.json")
	if _, err := os.Lstat(ruta); errors.Is(err, os.ErrNotExist) {
		return nil, vacio, nil
	} else if err != nil || !cfg.DevelopmentEnabledByDoubleKey() || derivador == nil || !derivador.valido() {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	contenido, err := leerFicheroMaterialSeguro(ruta, 16<<10)
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	defer borrarBytes(contenido)
	c, retirada, err := decodificarConfiguracionSesionFirmanteV2(contenido)
	if err != nil || !retirada.After(time.Now().UTC()) {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	entradas := []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, rolRegistroIdentidadConsultasDesarrollo},
		{c.DSNRevalidacionIdentidad, rolRevalidacionIdentidadConsultasDesarrollo},
		{c.DSNContexto, rolContextoActorConsultasDesarrollo},
	}
	var pools []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, entrada := range entradas {
		pool, usuario, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil || usuarios[usuario] {
			if pool != nil {
				pool.Close()
			}
			return nil, vacio, errSesionFirmanteV2NoDisponible
		}
		usuarios[usuario] = true
		pools = append(pools, pool)
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1],
		&seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	reloj := relojRutasDietas{}
	servicio, err := servicioContextoActorDietas(resolutor, reloj)
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	instancia, err := nonceRutasDietas()
	if err != nil {
		return nil, vacio, errSesionFirmanteV2NoDisponible
	}
	autoridad, err := nuevaAutoridadSesionFirmanteV2ConPuertos(identidad, registro, revalidador, contextos, reloj, retirada, instancia)
	if err != nil {
		return nil, vacio, err
	}
	completa = true
	return autoridad, cerrar, nil
}
