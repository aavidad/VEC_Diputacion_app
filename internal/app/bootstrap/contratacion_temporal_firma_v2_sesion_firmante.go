package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

var errSesionFirmanteV2NoDisponible = errors.New("contratacion temporal: sesion del firmante no disponible")
var errSesionFirmanteV2Denegada = errors.New("contratacion temporal: sesion del firmante denegada")

const politicaSesionFirmanteV2 = "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa;retirada-firma-vec"

// El fichero privado habilita únicamente la identidad sintética de registro-vec.
// La cuenta y el perfil se seleccionan en AUT56, nunca en este fichero.
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

type autoridadSesionFirmanteV2 struct {
	resolvedor  *resolvedorIdentidadDesarrollo
	registro    httpseguridad.RegistroSesiones
	revalidador core.RevalidadorAutenticacionActorV1
	contextos   core.ResolutorContextoActorRegistradoV2
	reloj       vp.Reloj
	retirada    time.Time
	instancia   string
}

// La cápsula sólo existe dentro del proceso y queda ligada al objeto Request.
// Su selección llega de AUT56 en 3b; ni el cuerpo ni una cabecera la rellenan.
type capsulaSesionFirmanteV2 struct {
	autoridad   *autoridadSesionFirmanteV2
	peticion    *http.Request
	seleccion   ports.SeleccionFirmanteV2
	personaCA25 string
	huellaAUT56 string
	principal   core.Principal
	instante    time.Time
	consumida   atomic.Bool
}

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
	return &autoridadSesionFirmanteV2{resolvedor: resolvedor, registro: registro, revalidador: revalidador,
		contextos: contextos, reloj: reloj, retirada: retirada, instancia: instancia}, nil
}

// La ausencia deja la ruta sin componer; un manifiesto presente pero inválido
// impide arrancar. La raíz 4c-8 decide si publica registro-vec.
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

// Acreditar es llamada sólo tras la selección central AUT56 del paso. El
// certificado verificado debe ser el de esa selección; la persona CA25 debe
// coincidir con la selección y después con el contexto central registrado.
func (a *autoridadSesionFirmanteV2) acreditar(r *http.Request, seleccion ports.SeleccionFirmanteV2, personaCA25, huellaAUT56 string) (*capsulaSesionFirmanteV2, error) {
	if a == nil || r == nil || r.URL == nil || r.Context().Err() != nil || r.URL.Path != httpinterno.RutaRegistroFirmaVec ||
		r.URL.RawPath != "" || r.Method != http.MethodPost || a.resolvedor == nil || a.reloj == nil ||
		r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
		return nil, errSesionFirmanteV2Denegada
	}
	ahora := a.reloj.Ahora()
	if !ahora.Before(a.retirada) {
		return nil, errSesionFirmanteV2Denegada
	}
	principal, err := a.resolvedor.ResolveDemoIdentity(r.Context(), r)
	if err != nil || !principalSinteticoContratacionTemporalDesarrolloValido(principal) || r.TLS == nil ||
		len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) < 2 {
		return nil, errSesionFirmanteV2Denegada
	}
	cert := r.TLS.VerifiedChains[0][0]
	if cert == nil || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
		return nil, errSesionFirmanteV2Denegada
	}
	huella := sha256.Sum256(cert.Raw)
	if principal.Attributes["certificate_sha256"] != hex.EncodeToString(huella[:]) || huellaAUT56 != hex.EncodeToString(huella[:]) ||
		seleccion.PersonaRef == "" || seleccion.PersonaRef != personaCA25 || seleccion.CuentaRef == "" ||
		seleccion.PerfilActivoRef == "" || seleccion.RolID == "" || seleccion.RolID == rolFirmaExternaRegistroCTDesarrollo ||
		seleccion.VinculoCertificado.Referencia == "" ||
		seleccion.VinculoCertificado.Version == 0 || !huellaSHA256ValidaContratacionTemporalDesarrollo(seleccion.VinculoCertificado.HuellaSHA256) {
		return nil, errSesionFirmanteV2Denegada
	}
	return &capsulaSesionFirmanteV2{autoridad: a, peticion: r, seleccion: seleccion,
		personaCA25: personaCA25, huellaAUT56: huellaAUT56, principal: principal, instante: ahora}, nil
}

func (a *autoridadSesionFirmanteV2) abrir(r *http.Request, capsula *capsulaSesionFirmanteV2) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error) {
	var vacio core.VinculoAutenticacionActorV2
	var sinContexto core.ResultadoContextoActorRegistradoV2
	if a == nil || r == nil || capsula == nil || capsula.autoridad != a || capsula.peticion != r ||
		!capsula.consumida.CompareAndSwap(false, true) || r.Context().Err() != nil ||
		!a.reloj.Ahora().Before(a.retirada) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	if r.TLS == nil || len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) < 2 ||
		r.TLS.VerifiedChains[0][0] == nil || r.URL == nil || r.URL.Path != httpinterno.RutaRegistroFirmaVec ||
		r.Method != http.MethodPost || capsula.principal.Attributes["certificate_sha256"] != capsula.huellaAUT56 {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	huellaActual := sha256.Sum256(r.TLS.VerifiedChains[0][0].Raw)
	if hex.EncodeToString(huellaActual[:]) != capsula.huellaAUT56 {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	seleccion := capsula.seleccion
	asercion, err := nonceRutasDietas()
	if err != nil {
		return vacio, sinContexto, errSesionFirmanteV2NoDisponible
	}
	sesion, err := nonceRutasDietas()
	if err != nil {
		return vacio, sinContexto, errSesionFirmanteV2NoDisponible
	}
	hasta := capsula.instante.Add(2 * time.Minute)
	if certificado := r.TLS.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond); certificado.Before(hasta) {
		hasta = certificado
	}
	if a.retirada.Before(hasta) {
		hasta = a.retirada
	}
	if !hasta.After(capsula.instante) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	huellaCanal := capsula.principal.Attributes["certificate_sha256"]
	alta := httpseguridad.AltaSesionAtomica{AsercionID: asercion, SesionID: sesion,
		SujetoID: capsula.principal.ID, CuentaID: "desarrollo:" + seleccion.CuentaRef,
		Superficie: httpseguridad.SuperficieInternaCorporativa, EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh,
		AutenticacionVerificadaEn: capsula.instante, SesionEmitidaEn: capsula.instante, AsercionExpiraEn: hasta,
		PoliticaGarantiaRef:          referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"),
		PoliticaGarantiaHuellaSHA256: huellaRutasDietas(politicaSesionFirmanteV2),
		AutenticacionHuellaSHA256: huellaRutasDietas(a.instancia + "|" + asercion + "|" + sesion + "|" + huellaCanal + "|" +
			seleccion.CuentaRef + "|" + seleccion.PerfilActivoRef + "|" + capsula.personaCA25 + "|" + r.URL.Path + "|" + r.Method),
	}
	confirmacion, err := a.registro.ConsumirAsercionYRegistrar(r.Context(), alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != seleccion.CuentaRef {
		return vacio, sinContexto, errSesionFirmanteV2NoDisponible
	}
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: a.revalidador, alta: alta,
		confirmacion: confirmacion, reloj: a.reloj, superficie: httpseguridad.SuperficieInternaCorporativa}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(r.Context(), revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef},
		a.contextos, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: seleccion.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh},
			PerfilActivoRef: seleccion.PerfilActivoRef}, a.reloj)
	if err != nil {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.CuentaRef != seleccion.CuentaRef || datos.PerfilActivoRef != seleccion.PerfilActivoRef ||
		datos.PrincipalID != seleccion.PersonaRef || datos.CuentaPrivilegiada ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		resultado.Contexto.PersonaRef != capsula.personaCA25 ||
		!resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() || !vinculo.VigenteEn(a.reloj.Ahora(), resultado) ||
		!a.reloj.Ahora().Before(a.retirada) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	return vinculo, resultado, nil
}
