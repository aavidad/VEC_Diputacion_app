package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/adapters/fuentesintetica"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	autoridadPeticionLlamamientoDesarrollo         = "autoridad:ct:llamamiento:desarrollo"
	autoridadRespuestaLlamamientoDesarrollo        = "autoridad:bolsa:llamamiento:desarrollo"
	clavePeticionLlamamientoDesarrollo             = "vec.contratacion-temporal.integracion-bolsa-peticion/v1"
	claveRespuestaLlamamientoDesarrollo            = "vec.contratacion-temporal.integracion-bolsa-respuesta/v1"
	claveSeleccionLlamamientoDesarrollo            = "vec.contratacion-temporal.seleccion/v1"
	posicionesFuenteLlamamientoDesarrollo   uint32 = 3
)

// Las referencias de intención NO son identificadores de decisiones VEC.
// La raíz autoriza cada petición antes de instalar la preparación privada;
// cada escritura Bolsa solicita y consume además su autorización mutante V3.
func prepararReferenciasLlamamientoDesarrollo(expediente ports.ExpedienteParaSeleccion, clave string) (preparacionLlamamientoDesarrollo, error) {
	e := expediente.Fiscalizado
	if !ports.ClaveIdempotenciaValida(clave) || e.Validar() != nil ||
		e.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		e.Version != 6 || expediente.VersionActual < 6 ||
		expediente.VersionActual > ports.MaximoEnteroSeguroIntegracionBolsa ||
		e.FaseActual != domain.FaseFiscalizacion || e.EstadoActual != domain.EstadoEnCurso ||
		e.Fiscalizacion == nil || e.Fiscalizacion.Resultado == domain.FiscalizacionDesfavorable ||
		e.Analisis == nil || e.Asignacion == nil {
		return preparacionLlamamientoDesarrollo{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	expediente.Fiscalizado = e.Clonar()
	necesidad := referenciaPuenteLlamamientoDesarrollo("necesidad", e.OrganizacionRef, e.Referencia, "6")
	p := preparacionLlamamientoDesarrollo{
		expediente: expediente, clave: clave, necesidad: necesidad,
		categoria: e.Analisis.CategoriaRef, unidad: e.Asignacion.UnidadRef,
		operacionOrden:     referenciaPuenteLlamamientoDesarrollo("operacion-orden", necesidad, clave),
		operacionPropuesta: referenciaPuenteLlamamientoDesarrollo("operacion-propuesta", necesidad, clave),
	}
	if !puertosbolsa.ReferenciaOpacaLlamamientoValida(p.categoria) ||
		!puertosbolsa.ReferenciaOpacaLlamamientoValida(p.unidad) {
		return preparacionLlamamientoDesarrollo{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return p, nil
}

type puenteBolsaLlamamientoDesarrollo struct {
	alta            *dependenciasAltaContratacionTemporalDesarrollo
	repositorio     puertosbolsa.RepositorioLlamamientoDesarrollo
	autorizador     puertosbolsa.AutorizadorLlamamientoDesarrollo
	reloj           ports.Reloj
	privadaFuente   ed25519.PrivateKey
	seleccion       selladorPuenteLlamamientoDesarrollo
	emisorPeticion  *ports.EmisorContextoPeticionIntegracionBolsa
	emisorRespuesta *ports.EmisorEvidenciaIntegracionBolsa
	verificador     *ports.VerificadorEvidenciaIntegracionBolsa
	autenticador    *ports.AutenticadorContextoPeticionIntegracionBolsa
}

var (
	_ ports.PreparadorSeleccionLlamamiento = (*puenteBolsaLlamamientoDesarrollo)(nil)
	_ ports.ConsultaDisponibilidadBolsa    = (*puenteBolsaLlamamientoDesarrollo)(nil)
	_ ports.PreparadorOrdenBolsa           = (*puenteBolsaLlamamientoDesarrollo)(nil)
	_ ports.GestorLlamamientosBolsa        = (*puenteBolsaLlamamientoDesarrollo)(nil)
)

// No abre conexiones, publica autoridades ni ejecuta SQL. El secreto es
// persistente y exclusivo de composición; nunca se deriva de datos públicos.
func nuevoPuenteBolsaLlamamientoDesarrollo(poolBolsa *pgxpool.Pool,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	proveedorMaterialBolsa *proveedorMaterialAltaContratacionTemporalDesarrollo,
	reloj ports.Reloj, secretoPuente []byte,
) (*puenteBolsaLlamamientoDesarrollo, error) {
	if poolBolsa == nil || alta == nil || alta.soporte == nil || alta.autorizador == nil ||
		proveedorMaterialBolsa == nil || reloj == nil || len(secretoPuente) < 32 {
		return nil, ports.ErrIntegracionBolsaNoDisponible
	}
	repo, err := postgresbolsa.NuevoRepositorioIntegracionLlamamientosDesarrollo(poolBolsa)
	if err != nil {
		return nil, err
	}
	p := &puenteBolsaLlamamientoDesarrollo{
		alta: alta, repositorio: repo, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: proveedorMaterialBolsa},
	}
	if err := p.configurarFirmas(secretoPuente); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) Verificador() *ports.VerificadorEvidenciaIntegracionBolsa {
	return p.verificador
}
func (p *puenteBolsaLlamamientoDesarrollo) AutenticadorContexto() *ports.AutenticadorContextoPeticionIntegracionBolsa {
	return p.autenticador
}

// Emisor y verificador son objetos distintos: el consumidor no recibe SellarDatos.
type selladorPuenteLlamamientoDesarrollo struct {
	referencia string
	secreto    []byte
}
type verificadorPuenteLlamamientoDesarrollo struct {
	referencia string
	secreto    []byte
}

func macPuenteLlamamientoDesarrollo(secreto []byte, dominio string, material []byte) []byte {
	m := hmac.New(sha256.New, secreto)
	_, _ = m.Write([]byte("vec.ct.bolsa.desarrollo.v1\n" + dominio + "\n"))
	_, _ = m.Write(material)
	return m.Sum(nil)
}
func (s selladorPuenteLlamamientoDesarrollo) SellarDatos(ctx context.Context, material []byte) (string, error) {
	if ctx == nil || ctx.Err() != nil || len(s.secreto) != 32 {
		return "", ports.ErrEvidenciaBolsaNoAutenticada
	}
	return "hmac-sha256:" + s.referencia + ":" +
		hex.EncodeToString(macPuenteLlamamientoDesarrollo(s.secreto, s.referencia, material)), nil
}
func (v verificadorPuenteLlamamientoDesarrollo) VerificarDatos(ctx context.Context, referencia string, material []byte, sello string) error {
	if ctx == nil || ctx.Err() != nil || referencia != v.referencia || len(v.secreto) != 32 {
		return ports.ErrEvidenciaBolsaNoAutenticada
	}
	esperado := "hmac-sha256:" + referencia + ":" +
		hex.EncodeToString(macPuenteLlamamientoDesarrollo(v.secreto, referencia, material))
	if !hmac.Equal([]byte(esperado), []byte(sello)) {
		return ports.ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}
func (p *puenteBolsaLlamamientoDesarrollo) configurarFirmas(secreto []byte) error {
	if p == nil || len(secreto) < 32 {
		return ports.ErrEvidenciaBolsaNoAutenticada
	}
	derivar := func(dominio string) []byte {
		return macPuenteLlamamientoDesarrollo(secreto, "derivacion:"+dominio, nil)
	}
	semilla := derivar("fuente-ed25519")
	p.privadaFuente = ed25519.NewKeyFromSeed(semilla)
	clear(semilla)
	peticion := selladorPuenteLlamamientoDesarrollo{clavePeticionLlamamientoDesarrollo, derivar(clavePeticionLlamamientoDesarrollo)}
	respuesta := selladorPuenteLlamamientoDesarrollo{claveRespuestaLlamamientoDesarrollo, derivar(claveRespuestaLlamamientoDesarrollo)}
	p.seleccion = selladorPuenteLlamamientoDesarrollo{claveSeleccionLlamamientoDesarrollo, derivar(claveSeleccionLlamamientoDesarrollo)}
	var err error
	p.emisorPeticion, err = ports.NuevoEmisorContextoPeticionIntegracionBolsa(autoridadPeticionLlamamientoDesarrollo, peticion.referencia, peticion)
	if err != nil {
		return err
	}
	p.emisorRespuesta, err = ports.NuevoEmisorEvidenciaIntegracionBolsa(autoridadRespuestaLlamamientoDesarrollo, respuesta.referencia, respuesta)
	if err != nil {
		return err
	}
	p.verificador, err = ports.NuevoVerificadorEvidenciaIntegracionBolsa(autoridadRespuestaLlamamientoDesarrollo, respuesta.referencia, nil,
		verificadorPuenteLlamamientoDesarrollo{respuesta.referencia, respuesta.secreto})
	if err != nil {
		return err
	}
	p.autenticador, err = ports.NuevoAutenticadorContextoPeticionIntegracionBolsa(autoridadPeticionLlamamientoDesarrollo, peticion.referencia, nil,
		verificadorPuenteLlamamientoDesarrollo{peticion.referencia, peticion.secreto})
	return err
}

func huellaPuenteLlamamientoDesarrollo(material []byte) string {
	h := sha256.Sum256(material)
	return hex.EncodeToString(h[:])
}
func referenciaPuenteLlamamientoDesarrollo(prefijo string, partes ...string) string {
	b, _ := json.Marshal(append([]string{"vec.ct.bolsa.desarrollo.v1", prefijo}, partes...))
	return puertosbolsa.ReferenciaDesdeHuellaLlamamientoDesarrollo(prefijo, huellaPuenteLlamamientoDesarrollo(b))
}
func referenciaVersionadaPuenteLlamamientoDesarrollo(ref string, version uint64, huella string) ports.ReferenciaVersionadaIntegracionBolsa {
	return ports.ReferenciaVersionadaIntegracionBolsa{Referencia: ref, Version: version, HuellaSHA256: huella}
}
func referenciaCatalogoPuenteLlamamientoDesarrollo(clave string) ports.ReferenciaVersionadaIntegracionBolsa {
	return referenciaVersionadaPuenteLlamamientoDesarrollo(clave, 1, huellaPuenteLlamamientoDesarrollo([]byte("catalogo-sintetico-desarrollo-v1\n"+clave)))
}

func (p *puenteBolsaLlamamientoDesarrollo) preparacion(ctx context.Context, clave string) (preparacionLlamamientoDesarrollo, error) {
	vacio := preparacionLlamamientoDesarrollo{}
	if ctx == nil || ctx.Err() != nil || p == nil || p.alta == nil || p.alta.soporte == nil {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	capacidad, valida := p.alta.soporte.capacidadValida(ctx)
	recibida, ok := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !valida || capacidad.ruta != httpinterno.RutaSeleccionLlamamiento || !ok || (clave != "" && recibida.clave != clave) {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	canonica, err := prepararReferenciasLlamamientoDesarrollo(recibida.expediente, recibida.clave)
	if err != nil || canonica.necesidad != recibida.necesidad || canonica.categoria != recibida.categoria ||
		canonica.unidad != recibida.unidad || canonica.operacionOrden != recibida.operacionOrden ||
		canonica.operacionPropuesta != recibida.operacionPropuesta {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return canonica, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) contexto(ctx context.Context, preparacion preparacionLlamamientoDesarrollo,
	operacion, accion string, recurso ports.ReferenciaVersionadaIntegracionBolsa,
) (ports.ContextoPeticionIntegracionBolsa, error) {
	v, err := p.alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return ports.ContextoPeticionIntegracionBolsa{}, err
	}
	e := preparacion.expediente.Fiscalizado
	// Excluye la sesión/decisión efímera: compromete el vínculo/perfil y la
	// intención. La autorización fresca de la raíz no se sustituye por este MAC.
	material, err := json.Marshal([]string{v.ContextoActorRef, strconv.FormatUint(v.ContextoActorVersion, 10),
		v.PerfilActivoRef, e.OrganizacionRef, e.Referencia, "6", preparacion.clave})
	if err != nil {
		return ports.ContextoPeticionIntegracionBolsa{}, err
	}
	h := huellaPuenteLlamamientoDesarrollo(material)
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	d := ports.DatosContextoPeticionIntegracionBolsa{
		OperacionRef: operacion, OrganizacionRef: e.OrganizacionRef, ExpedienteRef: e.Referencia,
		VersionExpediente: 6, CorrelacionRef: referenciaPuenteLlamamientoDesarrollo("correlacion", preparacion.necesidad, preparacion.clave),
		ContratoVersion: 1, AutoridadSolicitante: autoridadPeticionLlamamientoDesarrollo,
		Autorizacion: referenciaVersionadaPuenteLlamamientoDesarrollo(puertosbolsa.ReferenciaDesdeHuellaLlamamientoDesarrollo("intencion-certificada", h), 1, h),
		Accion:       referenciaCatalogoPuenteLlamamientoDesarrollo(accion), Recurso: recurso,
		Finalidad:    referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.FinalidadIntegracionLlamamientoDesarrollo),
		SolicitadaEn: ahora, ValidaHasta: ahora.Add(5 * time.Minute),
	}
	return p.emisorPeticion.Emitir(ctx, d, ahora)
}

// Fuente cerrada de tres participaciones sintéticas: orden suministrado, NO
// baremación. Primera ocupada y las otras disponibles. Las cinco reglas son
// datos firmados de desarrollo, no una interpretación de normativa.
func (p *puenteBolsaLlamamientoDesarrollo) fuente(preparacion preparacionLlamamientoDesarrollo) (*fuentesintetica.FuenteLlamamientos, fuentesintetica.DocumentoFuenteLlamamientos, error) {
	vacio := fuentesintetica.DocumentoFuenteLlamamientos{}
	e := preparacion.expediente.Fiscalizado
	if e.Analisis == nil || e.Asignacion == nil || e.Fiscalizacion == nil || len(p.privadaFuente) != ed25519.PrivateKeySize {
		return nil, vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora())
	if !vigente || e.Fiscalizacion.FiscalizadaEn.Before(desde) || !e.Fiscalizacion.FiscalizadaEn.Before(hasta) {
		return nil, vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	ref := func(tipo string) string { return referenciaPuenteLlamamientoDesarrollo(tipo, preparacion.necesidad) }
	h := func(s string) string { return huellaPuenteLlamamientoDesarrollo([]byte("fuente-sintetica-v1\n" + s)) }
	listado, _ := json.Marshal([]string{"orden-sintetico-no-baremo", "ocupado", "disponible", "disponible"})
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: ref("bolsa-sintetica"), Version: 1, ProcesoRef: ref("proceso-sintetico"),
		CategoriaRef: preparacion.categoria, ListadoDefinitivoRef: ref("listado-sintetico"),
		VersionListado: 1, HuellaListadoSHA256: huellaPuenteLlamamientoDesarrollo(listado),
		ResolucionConstitucionRef: ref("constitucion-sintetica"), HuellaResolucionSHA256: h("constitucion-sintetica"),
		ConstituidaEn: desde, VigenteDesde: desde, VigenteHasta: &hasta,
	})
	if err != nil {
		return nil, vacio, err
	}
	bolsaHash, _ := bolsa.HuellaCanonicaSHA256()
	// Fin de CT es fecha civil inclusiva; Bolsa usa extremo temporal exclusivo.
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: preparacion.necesidad, Version: 6, BolsaRef: bolsa.BolsaRef, VersionBolsa: bolsa.Version,
		HuellaBolsaSHA256: bolsaHash, CategoriaRef: preparacion.categoria, PuestoRef: ref("puesto-sintetico"),
		UnidadRef: preparacion.unidad, TipoCoberturaRef: string(e.Analisis.ModalidadClave), NumeroPuestos: 1,
		InicioPrevisto: e.Analisis.Periodo.Inicio, FinPrevisto: e.Analisis.Periodo.Fin.AddDate(0, 0, 1),
		CreadaEn: e.Fiscalizacion.FiscalizadaEn,
	})
	if err != nil {
		return nil, vacio, err
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:bolsa:fuente-sintetica:desarrollo", Clave: "fuente_sintetica_desarrollo", Version: 1,
		HuellaSHA256: h("disponible-elegible-resto-no-elegible"), PublicadaEn: desde, VigenteDesde: desde, VigenteHasta: &hasta,
	})
	if err != nil {
		return nil, vacio, err
	}
	d := fuentesintetica.DocumentoFuenteLlamamientos{
		Esquema: fuentesintetica.EsquemaFuenteLlamamientos, OrigenRef: ref("fuente-sintetica"), Version: 1,
		VigenteDesde: desde, VigenteHasta: hasta,
		Datos: puertosbolsa.DatosAutoritativosLlamamiento{Bolsa: bolsa, Necesidad: necesidad, Politica: politica},
	}
	for _, estado := range []string{"disponible", "ocupado", "no_disponible", "excluido", "renuncia_pendiente"} {
		resultado := dominiobolsa.ResultadoNoElegible
		if estado == "disponible" {
			resultado = dominiobolsa.ResultadoElegible
		}
		d.Reglas = append(d.Reglas, fuentesintetica.ReglaEstado{
			EstadoClave: estado, EstadoVersion: 1, HuellaEstadoSHA256: h(estado), Resultado: resultado,
			Motivo: dominiobolsa.MotivoEvaluacionLlamamiento{Clave: "fuente_sintetica", ReglaRef: "regla:sintetica:" + estado, VersionRegla: 1, HuellaReglaSHA256: h("regla:" + estado)},
		})
	}
	for i := uint32(0); i < posicionesFuenteLlamamientoDesarrollo; i++ {
		regla := d.Reglas[0]
		if i == 0 {
			regla = d.Reglas[1]
		}
		n := strconv.FormatUint(uint64(i+1), 10)
		participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
			ParticipacionRef: ref("participacion-sintetica-" + n), BolsaRef: bolsa.BolsaRef,
			SujetoRef: ref("sujeto-sintetico-" + n), Version: 1, AltaEn: desde,
			Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
				Secuencia: 1, EstadoClave: regla.EstadoClave, EstadoVersion: 1, HuellaEstadoSHA256: regla.HuellaEstadoSHA256,
				CausaClave: "fixture_sintetica", CausaVersion: 1, HuellaCausaSHA256: h("fixture_sintetica"),
				DecisionRef: ref("decision-sintetica-" + n), HuellaDecisionSHA256: h("decision-sintetica:" + n),
				Desde: desde,
			}},
		})
		if err != nil {
			return nil, vacio, err
		}
		d.Datos.Entradas = append(d.Datos.Entradas, dominiobolsa.EntradaOrdenBolsa{Orden: uint64(i + 1), Participacion: participacion})
	}
	canonico, err := json.Marshal(d)
	if err != nil {
		return nil, vacio, err
	}
	firma := ed25519.Sign(p.privadaFuente, fuentesintetica.MaterialFirmaFuenteLlamamientos(canonico))
	f, err := fuentesintetica.NuevaFuenteLlamamientos(canonico, firma, p.privadaFuente.Public().(ed25519.PublicKey), p.reloj)
	return f, d, err
}
func (p *puenteBolsaLlamamientoDesarrollo) servicio(d preparacionLlamamientoDesarrollo) (*appbolsa.ServicioIntegracionLlamamientosDesarrollo, fuentesintetica.DocumentoFuenteLlamamientos, error) {
	f, fuente, err := p.fuente(d)
	if err != nil {
		return nil, fuente, err
	}
	s, err := appbolsa.NuevoServicioIntegracionLlamamientosDesarrollo(f, p.repositorio, p.autorizador, p.reloj)
	return s, fuente, err
}
func referenciasFuentePuenteLlamamientoDesarrollo(f fuentesintetica.DocumentoFuenteLlamamientos) (necesidad, bolsa, politica ports.ReferenciaVersionadaIntegracionBolsa) {
	n, _ := f.Datos.Necesidad.HuellaCanonicaSHA256()
	b, _ := f.Datos.Bolsa.HuellaCanonicaSHA256()
	return referenciaVersionadaPuenteLlamamientoDesarrollo(f.Datos.Necesidad.NecesidadRef, f.Datos.Necesidad.Version, n),
		referenciaVersionadaPuenteLlamamientoDesarrollo(f.Datos.Bolsa.BolsaRef, f.Datos.Bolsa.Version, b),
		referenciaVersionadaPuenteLlamamientoDesarrollo(f.Datos.Politica.PoliticaRef, f.Datos.Politica.Version, f.Datos.Politica.HuellaSHA256)
}

func (p *puenteBolsaLlamamientoDesarrollo) PrepararConsultaDisponibilidad(ctx context.Context, clave string) (ports.SolicitudDisponibilidadBolsa, error) {
	d, err := p.preparacion(ctx, clave)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	_, f, err := p.fuente(d)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	n, _, _ := referenciasFuentePuenteLlamamientoDesarrollo(f)
	c, err := p.contexto(ctx, d, referenciaPuenteLlamamientoDesarrollo("operacion-consulta", d.necesidad, clave), "bolsa.disponibilidad.consultar", n)
	if err != nil {
		return ports.SolicitudDisponibilidadBolsa{}, err
	}
	return ports.SolicitudDisponibilidadBolsa{Contexto: c, Necesidad: n, CategoriaRef: d.categoria, MaximoResultados: posicionesFuenteLlamamientoDesarrollo}, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) PrepararOrdenCompleto(ctx context.Context, clave string, resultado ports.ResultadoDisponibilidadBolsa) (ports.ComandoPrepararOrdenBolsa, error) {
	vacio := ports.ComandoPrepararOrdenBolsa{}
	d, err := p.preparacion(ctx, clave)
	if err != nil || d.expediente.VersionActual != 6 {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	_, f, err := p.fuente(d)
	if err != nil {
		return vacio, err
	}
	n, b, politica := referenciasFuentePuenteLlamamientoDesarrollo(f)
	if resultado.Necesidad != n || resultado.Bolsa != b || resultado.CategoriaRef != d.categoria ||
		resultado.CantidadDisponible != 2 || !resultado.CantidadExacta || !resultado.Disponible {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	c, err := p.contexto(ctx, d, d.operacionOrden, puertosbolsa.AccionPrepararOrdenDesarrollo, b)
	if err != nil {
		return vacio, err
	}
	return ports.ComandoPrepararOrdenBolsa{Contexto: c, Necesidad: n, Bolsa: b, Politica: politica, MaximoPosiciones: posicionesFuenteLlamamientoDesarrollo}, nil
}

func (p *puenteBolsaLlamamientoDesarrollo) PrepararContextoLlamamiento(ctx context.Context, clave string, recibo ports.ReciboOrdenBolsa) (ports.ContextoPeticionIntegracionBolsa, error) {
	d, err := p.preparacion(ctx, clave)
	if err != nil || d.expediente.VersionActual != 6 || recibo.OperacionRef != d.operacionOrden ||
		!recibo.OrdenGenerada || !recibo.OrdenCompleta || recibo.TotalPosiciones != posicionesFuenteLlamamientoDesarrollo ||
		recibo.AccionLlamamiento != referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.AccionAbrirLlamamientoDesarrollo) {
		return ports.ContextoPeticionIntegracionBolsa{}, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return p.contexto(ctx, d, d.operacionPropuesta, puertosbolsa.AccionAbrirLlamamientoDesarrollo, recibo.Orden)
}

// Autentica el contexto recibido y su ligadura con la preparación privada,
// antes de consultar/escribir. No admite como autoridad un DTO HTTP nominal.
func (p *puenteBolsaLlamamientoDesarrollo) verificarPeticion(ctx context.Context, d preparacionLlamamientoDesarrollo,
	c ports.ContextoPeticionIntegracionBolsa, operacion, accion string,
	recurso ports.ReferenciaVersionadaIntegracionBolsa,
) (ports.DatosContextoPeticionIntegracionBolsa, error) {
	vacio := ports.DatosContextoPeticionIntegracionBolsa{}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	registro, err := c.Registro()
	if err != nil {
		return vacio, err
	}
	if _, err := p.autenticador.Reautenticar(ctx, registro, ahora); err != nil {
		return vacio, err
	}
	esperado, err := p.contexto(ctx, d, operacion, accion, recurso)
	if err != nil {
		return vacio, err
	}
	// El reloj puede avanzar durante Emitir: aquí solo se comparan las
	// coordenadas estables de un contexto recién creado por este emisor.
	registroEsperado, err := esperado.Registro()
	if err != nil {
		return vacio, err
	}
	e := registroEsperado.Datos
	r, err := c.DatosEn(ahora)
	if err != nil || r.OperacionRef != operacion || r.OrganizacionRef != e.OrganizacionRef ||
		r.ExpedienteRef != e.ExpedienteRef || r.VersionExpediente != 6 || r.CorrelacionRef != e.CorrelacionRef ||
		r.Autorizacion != e.Autorizacion || r.Accion != e.Accion || r.Recurso != recurso || r.Finalidad != e.Finalidad {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	return r, nil
}
func (p *puenteBolsaLlamamientoDesarrollo) procedencia(c ports.DatosContextoPeticionIntegracionBolsa, f fuentesintetica.DocumentoFuenteLlamamientos, tipo string, ahora time.Time) ports.ProcedenciaIntegracionBolsa {
	b, _ := json.Marshal(f)
	ref := referenciaPuenteLlamamientoDesarrollo("respuesta-"+tipo, c.OperacionRef, ahora.Format(time.RFC3339Nano))
	return ports.ProcedenciaIntegracionBolsa{
		AutoridadRef: autoridadRespuestaLlamamientoDesarrollo, RespuestaRef: ref, ContratoVersion: 1,
		Fuente: referenciaVersionadaPuenteLlamamientoDesarrollo(f.OrigenRef, f.Version, huellaPuenteLlamamientoDesarrollo(b)),
		Evidencia: ports.EvidenciaNominalIntegracionBolsa{
			EvidenciaRef: referenciaPuenteLlamamientoDesarrollo("evidencia", ref),
			EmitidaEn:    ahora, ValidaHasta: c.ValidaHasta, RetenerHasta: f.VigenteHasta.AddDate(1, 0, 0),
		},
	}
}

func (p *puenteBolsaLlamamientoDesarrollo) ConsultarDisponibilidad(ctx context.Context, solicitud ports.SolicitudDisponibilidadBolsa) (ports.ResultadoDisponibilidadBolsa, error) {
	vacio := ports.ResultadoDisponibilidadBolsa{}
	d, err := p.preparacion(ctx, "")
	if err != nil || d.expediente.VersionActual != 6 {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	s, f, err := p.servicio(d)
	if err != nil {
		return vacio, err
	}
	n, b, _ := referenciasFuentePuenteLlamamientoDesarrollo(f)
	c, err := p.verificarPeticion(ctx, d, solicitud.Contexto, referenciaPuenteLlamamientoDesarrollo("operacion-consulta", d.necesidad, d.clave), "bolsa.disponibilidad.consultar", n)
	if err != nil || solicitud.Necesidad != n || solicitud.CategoriaRef != d.categoria ||
		solicitud.MaximoResultados != posicionesFuenteLlamamientoDesarrollo {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	resultado, err := s.ConsultarDisponibilidad(ctx, d.necesidad, solicitud.MaximoResultados)
	if err != nil {
		return vacio, err
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	material, _ := json.Marshal(resultado)
	r := ports.ResultadoDisponibilidadBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		VersionExpediente: 6, CorrelacionRef: c.CorrelacionRef, Necesidad: n, CategoriaRef: d.categoria,
		Resultado:       referenciaVersionadaPuenteLlamamientoDesarrollo(referenciaPuenteLlamamientoDesarrollo("resultado-disponibilidad", c.OperacionRef), 1, huellaPuenteLlamamientoDesarrollo(material)),
		BolsaEncontrada: true, Bolsa: b, Disponible: resultado.CantidadDisponible > 0,
		CantidadDisponible: resultado.CantidadDisponible, CantidadExacta: resultado.CantidadExacta,
		Procedencia: p.procedencia(c, f, "disponibilidad", ahora),
	}
	return p.emisorRespuesta.FirmarDisponibilidad(ctx, solicitud, r, ahora)
}

func (p *puenteBolsaLlamamientoDesarrollo) PrepararOrden(ctx context.Context, comando ports.ComandoPrepararOrdenBolsa) (ports.ReciboOrdenBolsa, error) {
	vacio := ports.ReciboOrdenBolsa{}
	d, err := p.preparacion(ctx, "")
	if err != nil || d.expediente.VersionActual != 6 {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	s, f, err := p.servicio(d)
	if err != nil {
		return vacio, err
	}
	n, b, politica := referenciasFuentePuenteLlamamientoDesarrollo(f)
	c, err := p.verificarPeticion(ctx, d, comando.Contexto, d.operacionOrden, puertosbolsa.AccionPrepararOrdenDesarrollo, b)
	if err != nil || comando.Necesidad != n || comando.Bolsa != b || comando.Politica != politica ||
		comando.MaximoPosiciones != posicionesFuenteLlamamientoDesarrollo {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	r, err := s.PrepararOrden(ctx, puertosbolsa.PeticionLlamamientoDesarrollo{OperacionRef: d.operacionOrden, NecesidadRef: d.necesidad, MaximoPosiciones: comando.MaximoPosiciones})
	if err != nil {
		return vacio, err
	}
	return p.firmarOrdenPersistida(ctx, comando, c, f, r)
}

func (p *puenteBolsaLlamamientoDesarrollo) firmarOrdenPersistida(ctx context.Context, comando ports.ComandoPrepararOrdenBolsa,
	c ports.DatosContextoPeticionIntegracionBolsa, f fuentesintetica.DocumentoFuenteLlamamientos, r puertosbolsa.ReciboLlamamientoDesarrollo,
) (ports.ReciboOrdenBolsa, error) {
	canonico, err := r.Registro.Canonico()
	if err != nil || r.Registro.Tipo != "orden" || r.Registro.OperacionRef != c.OperacionRef {
		return ports.ReciboOrdenBolsa{}, ports.ErrRespuestaBolsaNoConfiable
	}
	i := r.Registro.Instantanea
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	recibo := ports.ReciboOrdenBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		VersionExpediente: 6, CorrelacionRef: c.CorrelacionRef, Necesidad: comando.Necesidad, Bolsa: comando.Bolsa, Politica: comando.Politica,
		Resultado:     referenciaVersionadaPuenteLlamamientoDesarrollo(r.ReciboRef, 1, huellaPuenteLlamamientoDesarrollo(canonico)),
		OrdenGenerada: true, OrdenCompleta: true,
		// El servicio ya consumió autorización fresca, también al leer la
		// misma fila. Se atesta la recuperación sin simular otro COMMIT.
		OrdenRecuperada:   r.ConfirmadaEn.Before(c.SolicitadaEn),
		Orden:             referenciaVersionadaPuenteLlamamientoDesarrollo(i.InstantaneaRef, i.Version, i.HuellaContenidoSHA256),
		AccionLlamamiento: referenciaCatalogoPuenteLlamamientoDesarrollo(puertosbolsa.AccionAbrirLlamamientoDesarrollo),
		TotalPosiciones:   uint32(len(i.Entradas)), ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef,
		ConfirmadaEn: r.ConfirmadaEn, Procedencia: p.procedencia(c, f, "orden", ahora),
	}
	return p.emisorRespuesta.FirmarOrden(ctx, comando, recibo, ahora)
}

func (p *puenteBolsaLlamamientoDesarrollo) SolicitarLlamamiento(ctx context.Context, comando ports.ComandoSolicitarLlamamientoBolsa) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	vacio := ports.ReciboSolicitudLlamamientoBolsa{}
	d, err := p.preparacion(ctx, "")
	if err != nil || d.expediente.VersionActual != 6 {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	s, f, err := p.servicio(d)
	if err != nil {
		return vacio, err
	}
	n, b, politica := referenciasFuentePuenteLlamamientoDesarrollo(f)
	datos, err := comando.DatosEn(p.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil {
		return vacio, err
	}
	c, err := p.verificarPeticion(ctx, d, datos.Contexto, d.operacionPropuesta, puertosbolsa.AccionAbrirLlamamientoDesarrollo, datos.Orden)
	if err != nil || datos.Necesidad != n || datos.Bolsa != b || datos.Politica != politica ||
		datos.TotalPosicionesOrden != posicionesFuenteLlamamientoDesarrollo || datos.MaximaPosicionEvaluable != posicionesFuenteLlamamientoDesarrollo {
		return vacio, ports.ErrPeticionIntegracionBolsaInvalida
	}
	// La referencia de orden sale del registro Bolsa, nunca se recrea en CT.
	orden, existe, err := p.repositorio.BuscarOperacion(ctx, d.operacionOrden)
	if err != nil || !existe || orden.Tipo != "orden" ||
		datos.Orden != referenciaVersionadaPuenteLlamamientoDesarrollo(orden.Instantanea.InstantaneaRef, orden.Instantanea.Version, orden.Instantanea.HuellaContenidoSHA256) {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	r, err := s.SolicitarLlamamiento(ctx, puertosbolsa.PeticionLlamamientoDesarrollo{
		OperacionRef: d.operacionPropuesta, OrdenOperacionRef: d.operacionOrden, NecesidadRef: d.necesidad, MaximoPosiciones: datos.MaximaPosicionEvaluable,
	})
	if err != nil {
		return vacio, err
	}
	return p.firmarLlamamientoPersistido(ctx, comando, c, f, r)
}

func (p *puenteBolsaLlamamientoDesarrollo) firmarLlamamientoPersistido(ctx context.Context, comando ports.ComandoSolicitarLlamamientoBolsa,
	c ports.DatosContextoPeticionIntegracionBolsa, f fuentesintetica.DocumentoFuenteLlamamientos, r puertosbolsa.ReciboLlamamientoDesarrollo,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	vacio := ports.ReciboSolicitudLlamamientoBolsa{}
	canonico, err := r.Registro.Canonico()
	if err != nil || r.Registro.Tipo != "propuesta" || r.Registro.OperacionRef != c.OperacionRef ||
		r.Registro.Propuesta == nil || r.Registro.Llamamiento == nil || r.Registro.EstadoLlamamiento != dominiobolsa.EstadoLlamamientoAbierto ||
		r.ConfirmadaEn.Before(c.SolicitadaEn) {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	propuesta := r.Registro.Propuesta
	llamamiento := r.Registro.Llamamiento
	if llamamiento.PropuestaRef != propuesta.PropuestaRef || llamamiento.NecesidadRef != propuesta.NecesidadRef {
		return vacio, ports.ErrRespuestaBolsaNoConfiable
	}
	material, _ := json.Marshal([]string{c.OrganizacionRef, c.ExpedienteRef, llamamiento.LlamamientoRef, propuesta.ParticipacionSeleccionadaRef, propuesta.SujetoSeleccionadoRef})
	sello, err := p.seleccion.SellarDatos(ctx, material)
	clear(material)
	if err != nil {
		return vacio, err
	}
	seudonimo, err := ports.NuevoSeudonimoSeleccionBolsa(sello)
	if err != nil {
		return vacio, err
	}
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := comando.DatosEn(ahora)
	if err != nil {
		return vacio, err
	}
	recibo := ports.ReciboSolicitudLlamamientoBolsa{
		OperacionRef: c.OperacionRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: c.ExpedienteRef,
		VersionExpediente: 6, CorrelacionRef: c.CorrelacionRef, Necesidad: datos.Necesidad, Bolsa: datos.Bolsa, Orden: datos.Orden, Politica: datos.Politica,
		Resultado:         referenciaVersionadaPuenteLlamamientoDesarrollo(r.ReciboRef, 1, huellaPuenteLlamamientoDesarrollo(canonico)),
		PropuestaGenerada: true,
		Propuesta:         referenciaVersionadaPuenteLlamamientoDesarrollo(propuesta.PropuestaRef, 1, propuesta.HuellaContenidoSHA256),
		AccionEvento:      referenciaCatalogoPuenteLlamamientoDesarrollo("bolsa.llamamiento.evento.registrar"),
		LlamamientoRef:    llamamiento.LlamamientoRef, SeleccionRef: seudonimo,
		RetencionSeleccion: referenciaCatalogoPuenteLlamamientoDesarrollo("retencion:seleccion:sintetica:desarrollo"),
		OrdenSeleccionado:  uint32(propuesta.OrdenSeleccionado), ReciboRef: r.ReciboRef, AuditoriaRef: r.AuditoriaRef, EventoRef: r.EventoRef,
		ConfirmadaEn: r.ConfirmadaEn, Procedencia: p.procedencia(c, f, "llamamiento", ahora),
	}
	return p.emisorRespuesta.FirmarLlamamiento(ctx, comando, recibo, ahora)
}
